package LoaderGenerater

import (
	"context"
	"github.com/TonyXMH/LoaderGenerater/lib"
	"time"
)

type myGenerator struct {
	timeoutNS   time.Duration        //响应超时时间
	lps         uint32               //每秒负载量loader per second
	durationNS  time.Duration        //负载持续时间
	resultCh    *chan lib.CallResult //调用的结果chan
	concurrency uint32               //载荷并发量 timeoutNS/(1e9/lps)
	stopSign    chan struct{}        //停止信号的传递通道
	ctx         context.Context      //上下文
	ctxFunc     context.CancelFunc   //取消函数
	status      uint32               //状态
	caller      lib.Caller           //调用器
	callCount int64//调用计数

}
