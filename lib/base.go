package lib

import "time"

// all status of stress Loader
const (
	// original
	STATUS_ORIGINAL  uint32 = 0
	// starting
	STATUS_STARTING  uint32 = 1
	// started
	STATUS_STARTED  uint32 = 2
	// stopping
	STATUS_STOPPING  uint32 = 3
	//stopped
	STATUS_STOPPED  uint32  =4

)




// represent call result
type CallResult struct {
	ID    int64        //ID
	Req   RawReq       //raw request
	Resp  RawResp     //raw response
	Code  RetCode     //return code
	Msg   string     // result details
	Elapse time.Duration  //time cost


}

//raw request
type RawReq struct {
	ID  int64
	Req []byte
}


//raw  response
type RawResp struct {
	ID  	int64
	Resp 	[]byte
	Err  	error
	Elapse 	time.Duration
}

// return code
type RetCode int


//return code  redefined

const  (
	RET_CODE_SUCCESS    			RetCode = 0       //success
	RET_CODE_WARNING_CALL_TIMEOUT 	RetCode =1001	  //timeout
	RET_CODE_ERROR_CALL				RetCode =2001	  //error call
	RET_CODE_ERROR_RESPONSE			RetCode =2002     //error_response
	RET_CODE_ERROR_CALLEE			RetCode = 2003	  //called software error
	RET_CODE_FATAL_ERROR			RetCode = 3001    // fatal error in call


)

//return    specific  explanation  of  "Call" func

func GetRetCodePlain(code RetCode)  string{
	var codePlain string
	switch code {
	case RET_CODE_SUCCESS:
		codePlain = "Success"
	case RET_CODE_WARNING_CALL_TIMEOUT:
		codePlain = "Call Timeout Warning"
	case RET_CODE_ERROR_CALL:
		codePlain = "Call Error"
	case RET_CODE_ERROR_RESPONSE:
		codePlain = "Response Error"
	case RET_CODE_ERROR_CALLEE:
		codePlain = "Callee Error"
	case RET_CODE_FATAL_ERROR:
		codePlain = "Call Fatal Error"
	default:
		codePlain = "Unknown result code"
	}

	return  codePlain
}

// Generator interface

type Generator interface {
	//start generator
	Start() 	bool
	//stop
	Stop()		bool
	//status
	Status()    uint32
	//call count
	CallCount()  int64
}


