package lib

import "time"

// req , request
// resp  ,  response

// caller interface
type Caller interface {
	// conduct request
	BuildReq()  RawReq
	// call
	Call(req []byte , timeoutNS time.Duration) ([]byte , error)
	// check resp
	CheckResp (rawReq RawReq , rawResp RawResp ) *CallResult
}

