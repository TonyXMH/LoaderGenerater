package LoaderGenerater

import (
	"context"
	"errors"
	"fmt"
	"github.com/TonyXMH/LoaderGenerater/lib"
	"math"
	"sync/atomic"
	"time"
)

type myGenerator struct {
	timeoutNS   time.Duration        //响应超时时间
	lps         uint32               //每秒负载量loader per second
	durationNS  time.Duration        //负载持续时间
	resultCh    chan *lib.CallResult //调用的结果chan
	concurrency uint32               //载荷并发量 timeoutNS/(1e9/lps)
	stopSign    chan struct{}        //停止信号的传递通道
	ctx         context.Context      //上下文
	ctxFunc     context.CancelFunc   //取消函数
	status      uint32               //状态
	caller      lib.Caller           //调用器
	callCount   int64                //调用计数
	tickets     lib.GoTickets        //票池
}

func NewGenerator(caller lib.Caller, timeoutNS time.Duration, lps uint32, durationNS time.Duration, resultCh chan *lib.CallResult) (lib.Generator, error) {
	gen := &myGenerator{
		caller:     caller,
		timeoutNS:  timeoutNS,
		lps:        lps,
		durationNS: durationNS,
		resultCh:   resultCh,
		status:     lib.StatusOriginal,
	}
	if err := gen.init(); err != nil {
		return nil, err
	}
	return gen, nil
}
func (g *myGenerator) init() (err error) {
	total64 := int64(g.timeoutNS) / int64(1e9/g.lps)
	if total64 > math.MaxInt32 {
		total64 = math.MaxInt32
	}
	g.concurrency = uint32(total64)
	g.tickets, err = lib.NewGoTickets(g.concurrency)
	return
}
func (g *myGenerator) Start() bool {

	var throttle <-chan time.Time
	if g.lps > 0 {
		throttle = time.Tick(time.Duration(1e9 / g.lps))
	}

	g.ctx, g.ctxFunc = context.WithTimeout(context.Background(), g.durationNS)

	g.callCount = 0
	atomic.StoreUint32(&g.status, lib.StatusStarted)

	go g.genLoad(throttle)
	return true
}

func (g *myGenerator) genLoad(throttle <-chan time.Time) {
	for {
		select {
		case <-g.ctx.Done():
			g.prepareToStop(g.ctx.Err())
			return
		default:
		}
		g.asyncCall()
		if g.lps > 0 {
			select {
			case <-throttle:
			case <-g.ctx.Done():
				g.prepareToStop(g.ctx.Err())
				return
			}
		}
	}
}

func (g *myGenerator) prepareToStop(err error) {
	fmt.Printf("Prepare to stop load generator cause %s\n", err)
	atomic.CompareAndSwapUint32(&g.status, lib.StatusStarted, lib.StatusStopping)
	close(g.resultCh)
	atomic.StoreUint32(&g.status, lib.StatusStopped)
}

func (g *myGenerator) asyncCall() {
	g.tickets.Take()
	go func() {
		defer func() {
			if p := recover(); p != nil { //崩溃处理
				err, ok := p.(error)
				var errMsg string
				if ok {
					errMsg = fmt.Sprintf("Async Call Panic (error:%s)", err)
				} else {
					errMsg = fmt.Sprintf("Async Call Panic (clue:%#v)", p)
				}
				fmt.Println(errMsg)
				result := &lib.CallResult{
					ID:   -1,
					Code: lib.RetFatalCall,
					Msg:  errMsg,
				}
				g.sendResult(result)
			}
			g.tickets.Return()
		}()
		rawReq := g.caller.BuildReq()
		//0表示未调用，1表示调用完成，2表示调用超时
		const (
			NotCall = iota
			CallCompeted
			CallTimeout
		)
		var callStatus uint32
		timer := time.AfterFunc(g.timeoutNS, func() {
			if !atomic.CompareAndSwapUint32(&callStatus, NotCall, CallTimeout) {
				return
			}
			result := &lib.CallResult{
				ID:     rawReq.ID,
				Req:    rawReq,
				Resp:   lib.RawResp{},
				Code:   lib.RetCodeWarningCallTimeout, //状态设置为超时
				Msg:    fmt.Sprintf("Timeout! (expected time:%+v)", g.timeoutNS),
				Elapse: g.timeoutNS,
			}
			g.sendResult(result)
		})

		rawResp := g.callOne(&rawReq)
		if !atomic.CompareAndSwapUint32(&callStatus, NotCall, CallCompeted) {
			return
		}
		timer.Stop()
		var result *lib.CallResult
		if rawResp.Err != nil {
			result = &lib.CallResult{
				ID:     rawReq.ID,
				Req:    rawReq,
				Code:   lib.RetErrorCall,
				Msg:    rawResp.Err.Error(),
				Elapse: rawResp.Elapse,
			}
		} else {
			result = g.caller.CheckResp(rawReq, *rawResp)
			result.Elapse = rawResp.Elapse
		}
		g.sendResult(result)
	}()

}

func (g *myGenerator) sendResult(result *lib.CallResult) bool {
	if atomic.LoadUint32(&g.status) != lib.StatusStarted {
		fmt.Println("gen status ", g.status, " can not send result")
		return false
	}
	select {
	case g.resultCh <- result:
		return true
	default:
		fmt.Println("ResultCh is full")
		return false
	}
}

func (g *myGenerator) callOne(rawReq *lib.RawReq) *lib.RawResp {
	atomic.AddInt64(&g.callCount, 1)
	if rawReq == nil {
		return &lib.RawResp{
			ID:  -1,
			Err: errors.New("Invalid raw request"),
		}
	}
	start := time.Now().UnixNano()
	resp, err := g.caller.Call(rawReq.Req, g.timeoutNS)
	end := time.Now().UnixNano()
	elapsedTime := time.Duration(end - start)
	var rawResp *lib.RawResp
	if err != nil {
		errMsg := fmt.Sprintf("sync call error:%s", err)
		rawResp = &lib.RawResp{
			ID: rawReq.ID,
			Err:    errors.New(errMsg),
			Elapse: elapsedTime,
		}
	}else{
		rawResp=&lib.RawResp{
			ID:rawReq.ID,
			Resp:resp,
			Elapse:elapsedTime,
		}
	}
	return rawResp
}

func (g *myGenerator) Stop() bool {
	if !atomic.CompareAndSwapUint32(&g.status,lib.StatusStarted,lib.StatusStopping){
		return false
	}
	g.ctxFunc()
	for{
		if atomic.LoadUint32(&g.status) == lib.StatusStopped{
			break
		}
		time.Sleep(time.Millisecond)
	}
	return true
}

func (g *myGenerator) Status() uint32 {
	return g.status
}

func (g *myGenerator) CallCount() int64 {
	return g.callCount
}
