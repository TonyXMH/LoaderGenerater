package lib

import "time"

type Caller interface {
	BuildReq() RawReq                                         //构建请求
	Call(req []byte, timeoutNS time.Duration) ([]byte, error) //调用请求
	CheckResp(rawReq RawReq, rawResp RawResp) *CallResult     //检查响应
}
