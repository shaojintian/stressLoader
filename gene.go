package stressLoader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"stressLoader/lib"
)

type myGenerator struct {
	caller		lib.Caller		  	// caller
	timeoutNS   time.Duration     	//   unit:ns
	lps         uint32			 	// load per second
	concurrency uint32				// concurrency numbers
	tickets     lib.GoTickets   	// goroutine  tickets pool
	ctx         context.Context 	// context
	cancelFunc  context.CancelFunc  // cancel func
	callCount   int64    			// count numbers
	status  	uint32				//status
	resultCh    chan *lib.CallResult	// call result chan
	durationNS  time.Duration


}

// param for NewGenerator

type ParamSet struct {
	caller  	lib.Caller
	timeoutNS 	time.Duration
	lps			uint32
	durationNS  time.Duration
	resultCh    chan *lib.CallResult
}

// func for check  errs
func ( pset *ParamSet)  Check() error{
	var errMsgs  []string
	if pset.lps == 0{
		errMsgs = append(errMsgs, "Invalid lpad per second!")
	}
	if pset.durationNS == 0{
		errMsgs = append(errMsgs,"Invalid durationNs")
	}
	if pset.caller == nil{
		errMsgs = append(errMsgs, "Invalid caller")

	}
	if pset.resultCh == nil{
		errMsgs = append(errMsgs, "Invalid resultCh")

	}
	if pset.timeoutNS == 0{
		errMsgs = append(errMsgs,"Invalid timeoutNs")
	}

	return  nil

}
// func  a NewGenerator

func  NewGenerator(pset ParamSet) (lib.Generator,error){

	logger.Infoln("Generate a new  Generator")

	if err := pset.Check(); err != nil{
		return nil , err
	}

	gen := &myGenerator{
		caller:		pset.caller,
		timeoutNS:  pset.timeoutNS,
		lps:		pset.lps,
		durationNS: pset.durationNS,
		status: 	lib.STATUS_ORIGINAL,
		resultCh:   pset.resultCh,

	}

	if err := gen.init() ; err != nil{
		return nil , err
	}
	return gen ,nil

}

// init generator
func (gen *myGenerator) init() error{
	var buf bytes.Buffer
	buf.WriteString("Initializing myGenerator...\n")
	// concurrency = timeout / time_interval
	var total32 = int32(gen.timeoutNS) / int32(1e9/gen.lps)+1
	if total32 > math.MaxInt32{
		total32 = math.MaxInt32
	}
	gen.concurrency = uint32(total32)
	tickets,err := lib.NewGoTickets(gen.concurrency)
	if err != nil {
		return  err
	}
	gen.tickets = tickets
	buf.WriteString(fmt.Sprintf("Done  concurrency =%d\n",gen.concurrency))
	logger.Infoln(buf.String())

	return nil
}

// callOne  to call  the  tested  software


func (gen *myGenerator) callOne(rawReq *lib.RawReq) *lib.RawResp {
	//concurrency secure
	atomic.AddInt64(&gen.callCount ,1)
	if rawReq == nil{
		return &lib.RawResp{ID:-1,Err:errors.New("no raw request")}
	}
	start := time.Now().UnixNano()
	resp , err := gen.caller.Call(rawReq.Req , gen.timeoutNS)
	end := time.Now().UnixNano()
	elapsedTime := time.Duration(end-start)
	if err != nil{
		errMsg := fmt.Sprintf("Sync RawReq : %d caller Call :error",rawReq.ID)
		return &lib.RawResp{
				ID:rawReq.ID,
				Err:errors.New(errMsg),
				Elapse:elapsedTime,
							}
	}

	var rawResp lib.RawResp

	rawResp = lib.RawResp{
		ID:rawReq.ID,
		Resp:resp,
		Elapse :elapsedTime,
	}

	return &rawResp

}

// async call  callee

func (gen *myGenerator) asyncCall(){
	gen.tickets.Take()
	go func() {
		defer func() {
			if p:=recover(); p!=nil{
				var errMsg string
				err , ok := p.(error)
				if ok{
					errMsg = fmt.Sprintf("Async call panic ! err :%s",err)
				}else{
					errMsg = fmt.Sprintf("Async call panic! clue:%s",p)
				}
				logger.Errorln(errMsg)

				result :=&lib.CallResult{
					ID :-1,
					Code : lib.RET_CODE_FATAL_ERROR,
					Msg:errMsg,

				}
				gen.sendResult(result)
			}
			gen.tickets.Return()
		}()

		rawReq := gen.caller.BuildReq()
		// call status  : 0:not call 1:call finished 2:call timeout
		var callStatus  uint32
		// if call timeout
		timer := time.AfterFunc(gen.timeoutNS,func(){
			if !atomic.CompareAndSwapUint32(&callStatus,0,2){
				return
			}
			result := &lib.CallResult{
				ID:		rawReq.ID,
				Req:	rawReq,
				Code:   lib.RET_CODE_WARNING_CALL_TIMEOUT,
				Msg:    fmt.Sprintf("Timeout! expected time < %s ",gen.timeoutNS),
				Elapse: gen.timeoutNS,

			}

			gen.sendResult(result)
		})
		// callOne
		rawResp := gen.callOne(&rawReq)
		if !atomic.CompareAndSwapUint32(&callStatus,0,1){
			//logger.Errorln("Change CallStatus to 'called finished(1)' error !" )
			return
		}
		timer.Stop()
		var result *lib.CallResult
		if rawResp.Err != nil {
			result = &lib.CallResult{
				ID:     rawResp.ID,
				Req:    rawReq,
				Code:   lib.RET_CODE_ERROR_CALL,
				Msg:    rawResp.Err.Error(),
				Elapse: rawResp.Elapse}
		} else {
			result = gen.caller.CheckResp(rawReq, *rawResp)
			result.Elapse = rawResp.Elapse
		}
		gen.sendResult(result)

	}()



}
//sendResult :  send  async res
func (gen *myGenerator) sendResult(result *lib.CallResult) bool{
	if atomic.LoadUint32(&gen.status) != lib.STATUS_STARTED {
		gen.printIgnoredResult(result, "stopped load generator")
		return false
	}
	select {
	case gen.resultCh <- result:
		return true
	default:
		gen.printIgnoredResult(result, "full result channel")
		return false
	}
}

// printIgnoredResult 打印被忽略的结果。
func (gen *myGenerator) printIgnoredResult(result *lib.CallResult, cause string) {
	resultMsg := fmt.Sprintf(
		"ID=%d, Code=%d, Msg=%s, Elapse=%v",
		result.ID, result.Code, result.Msg, result.Elapse)
	logger.Warnf("Ignored result: %s. (cause: %s)\n", resultMsg, cause)
}

// prepareStop 用于为停止载荷发生器做准备。
func (gen *myGenerator) prepareToStop(ctxError error) {
	logger.Infof("Prepare to stop load generator (cause: %s)...", ctxError)
	atomic.CompareAndSwapUint32(
		&gen.status, lib.STATUS_STARTED, lib.STATUS_STOPPING)
	logger.Infof("Closing result channel...")
	close(gen.resultCh)
	atomic.StoreUint32(&gen.status, lib.STATUS_STOPPED)
}

// genLoad 会产生载荷并向承受方发送。
func (gen *myGenerator) genLoad(throttle <-chan time.Time) {
	for {
		select {
		case <-gen.ctx.Done():
			gen.prepareToStop(gen.ctx.Err())
			return
		default:
		}
		gen.asyncCall()
		if gen.lps > 0 {
			select {
			case <-throttle:
			case <-gen.ctx.Done():
				gen.prepareToStop(gen.ctx.Err())
				return
			}
		}
	}
}

// Start 会启动载荷发生器。
func (gen *myGenerator) Start() bool {
	logger.Infoln("Starting load generator...")
	// 检查是否具备可启动的状态，顺便设置状态为正在启动
	if !atomic.CompareAndSwapUint32(
		&gen.status, lib.STATUS_ORIGINAL, lib.STATUS_STARTING) {
		if !atomic.CompareAndSwapUint32(
			&gen.status, lib.STATUS_STOPPED, lib.STATUS_STARTING) {
			return false
		}
	}

	// 设定节流阀。
	var throttle <-chan time.Time
	if gen.lps > 0 {
		interval := time.Duration(1e9 / gen.lps)
		logger.Infof("Setting throttle (%v)...", interval)
		throttle = time.Tick(interval)
	}

	// 初始化上下文和取消函数。
	gen.ctx, gen.cancelFunc = context.WithTimeout(
		context.Background(), gen.durationNS)

	// 初始化调用计数。
	gen.callCount = 0

	// 设置状态为已启动。
	atomic.StoreUint32(&gen.status, lib.STATUS_STARTED)

	go func() {
		// 生成并发送载荷。
		logger.Infoln("Generating loads...")
		gen.genLoad(throttle)
		logger.Infof("Stopped. (call count: %d)", gen.callCount)
	}()
	return true
}

func (gen *myGenerator) Stop() bool {
	if !atomic.CompareAndSwapUint32(
		&gen.status, lib.STATUS_STARTED, lib.STATUS_STOPPING) {
		return false
	}
	gen.cancelFunc()
	for {
		if atomic.LoadUint32(&gen.status) == lib.STATUS_STOPPED {
			break
		}
		time.Sleep(time.Microsecond)
	}
	return true
}

func (gen *myGenerator) Status() uint32 {
	return atomic.LoadUint32(&gen.status)
}

func (gen *myGenerator) CallCount() int64 {
	return atomic.LoadInt64(&gen.callCount)
}




