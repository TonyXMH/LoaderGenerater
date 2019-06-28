package lib

import "time"
//载荷器的状态
const (
	StatusOriginal = 0
	StatusStarting = 1
	StatusStarted  = 2
	StatusStopping = 3
	StatusStopped  = 4
)
//
const (
	RetCodeSuccess            RetCode = 0
	RetCodeWarningCallTimeout         = 1001
	RetErrorCall                      = 2001
	RetErrorResponse                  = 2002
	RetErrorCalee                     = 2003//被测软件的内部错误
	RetFatalCall                      = 3001
)

type RawReq struct {
	ID  int
	Req []byte
}

type RawResp struct {
	ID     int
	Resp   []byte
	Err    error
	Elapse time.Duration //处理耗时
}

type RetCode int

type CallResult struct {
	ID     int
	Req    RawReq
	Resp   RawResp
	Code   RetCode       //处理状态
	Msg    string        //Code的描述
	Elapse time.Duration //处理耗时单位ns
}

type Generator interface {
	Start() bool
	Stop() bool
	Status() uint32
	CallCount() int64
}
