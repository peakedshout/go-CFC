package tool

import "sync"

type CloseWaiter struct {
	stop   chan error
	closer sync.Once

	lock        sync.Mutex
	closeFnList []func()
}

func NewCloseWaiter() *CloseWaiter {
	return &CloseWaiter{
		stop:   make(chan error, 1),
		closer: sync.Once{},
	}
}
func (cw *CloseWaiter) AddCloseFn(fn func()) {
	cw.lock.Lock()
	defer cw.lock.Unlock()
	cw.closeFnList = append(cw.closeFnList, fn)
}
func (cw *CloseWaiter) Close(err error) {
	cw.closer.Do(func() {
		cw.lock.Lock()
		defer cw.lock.Unlock()
		for _, one := range cw.closeFnList {
			one()
		}
		cw.stop <- err
	})
}
func (cw *CloseWaiter) Wait() error {
	return <-cw.stop
}
