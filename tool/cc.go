package tool

import (
	"context"
	"sync"
	"sync/atomic"
)

type CtxCloser struct {
	ctx    context.Context
	cl     context.CancelFunc
	closer sync.Once

	isClosed atomic.Bool

	lock        sync.Mutex
	closeFnList []func()
}

func NewCtxCloser(ctx context.Context) *CtxCloser {
	ctx2, cl := context.WithCancel(ctx)
	return &CtxCloser{
		ctx:    ctx2,
		cl:     cl,
		closer: sync.Once{},
		lock:   sync.Mutex{},
	}
}

func (cc *CtxCloser) AddCloseFn(fn func()) {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	cc.closeFnList = append(cc.closeFnList, fn)
}
func (cc *CtxCloser) Close() {
	cc.closer.Do(func() {
		cc.isClosed.Store(true)
		cc.cl()
		cc.lock.Lock()
		defer cc.lock.Unlock()
		for _, one := range cc.closeFnList {
			one()
		}
	})
}
func (cc *CtxCloser) Chan() <-chan struct{} {
	return cc.ctx.Done()
}

func (cc *CtxCloser) CheckDone() bool {
	select {
	case <-cc.ctx.Done():
		cc.Close()
	default:
	}
	return cc.isClosed.Load()
}
