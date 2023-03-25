package tool

import (
	"bufio"
	"errors"
	"github.com/peakedshout/go-CFC/loger"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

type TaskCbContext struct {
	taskMap sync.Map
	key     Key
	rw      io.ReadWriter
	wLock   *sync.Mutex
	disable atomic.Bool
	noCbFn  func(cMsg ConnMsg)
}
type TaskCb struct {
	ctx     *TaskCbContext
	id      string
	fn      func() error
	cb      func(cMsg ConnMsg) error
	wait    chan error
	keep    atomic.Bool
	disable atomic.Bool
	stop    chan error
}

func NewTaskContext(rw io.ReadWriter, key Key) *TaskCbContext {
	return &TaskCbContext{
		taskMap: sync.Map{},
		key:     key,
		rw:      rw,
		wLock:   &sync.Mutex{},
		disable: atomic.Bool{},
		noCbFn:  nil,
	}
}

func (tc *TaskCbContext) Close() {
	tc.disable.Store(true)
}

func (tc *TaskCbContext) ReadCMsg() error {
	reader := bufio.NewReaderSize(tc.rw, BufferSize)
	for !tc.disable.Load() {
		cMsg, err := tc.key.ReadCMsg(reader, &tc.disable, nil)
		if err != nil {
			return err
		}
		tc.getTaskAndRun(cMsg)
	}
	return ErrIsDisable
}
func (tc *TaskCbContext) WriteCMsg(header string, id string, code int, data interface{}) error {
	tc.wLock.Lock()
	defer tc.wLock.Unlock()
	if tc.disable.Load() {
		return ErrIsDisable
	}
	for _, one := range tc.key.SetMsg(header, id, code, data) {
		_, err := tc.rw.Write(one)
		if err != nil {
			return err
		}
	}
	return nil
}

func (tc *TaskCbContext) SetNoCb(fn func(cMsg ConnMsg)) {
	tc.noCbFn = fn
}

func (tc *TaskCbContext) SetWriteLock(lock *sync.Mutex) {
	tc.wLock = lock
}

func (tc *TaskCbContext) getTask(key string) (*TaskCb, bool) {
	odj, ok := tc.taskMap.Load(key)
	if !ok {
		return nil, false
	}
	fn, ok := odj.(*TaskCb)
	if !ok {
		return nil, false
	}
	if !fn.keep.Load() {
		tc.taskMap.Delete(key)
	}
	return fn, true
}
func (tc *TaskCbContext) getTaskAndRun(cMsg ConnMsg) {
	task, ok := tc.getTask(cMsg.Id)
	if ok {
		if task.disable.Load() {
			return
		}
		task.wait <- task.cb(cMsg)
	} else {
		if tc.noCbFn != nil {
			tc.noCbFn(cMsg)
		}
	}
}
func (tc *TaskCbContext) NewTaskCb(id string, fn func() error) *TaskCb {
	return &TaskCb{
		ctx:     tc,
		id:      id,
		fn:      fn,
		cb:      nil,
		wait:    make(chan error, 1),
		keep:    atomic.Bool{},
		disable: atomic.Bool{},
		stop:    make(chan error, 1),
	}
}
func (tc *TaskCbContext) NewTaskCbCMsg(header string, code int, data interface{}) *TaskCb {
	id := NewId(1)
	return &TaskCb{
		ctx: tc,
		id:  id,
		fn: func() error {
			return tc.WriteCMsg(header, id, code, data)
		},
		cb:      nil,
		wait:    make(chan error, 1),
		keep:    atomic.Bool{},
		disable: atomic.Bool{},
		stop:    make(chan error, 1),
	}
}
func (tc *TaskCbContext) NewTaskCbCMsgNeedId(header string, id string, code int, data interface{}) *TaskCb {
	if id == "" {
		loger.SetLogError(TaskCbIdIsNeed)
	}
	return &TaskCb{
		ctx: tc,
		id:  id,
		fn: func() error {
			return tc.WriteCMsg(header, id, code, data)
		},
		cb:      nil,
		wait:    make(chan error, 1),
		keep:    atomic.Bool{},
		disable: atomic.Bool{},
		stop:    make(chan error, 1),
	}
}

func (task *TaskCb) WaitCb(timeout time.Duration, cb func(cMsg ConnMsg) error) (err error) {
	defer func() {
		if err != nil {
			task.OverTask(nil)
		}
	}()
	if task.ctx.disable.Load() {
		return ErrIsDisable
	}
	task.cb = cb
	task.ctx.taskMap.Store(task.id, task)
	if task.fn != nil {
		err = task.fn()
		if err != nil {
			return err
		}
	}
	select {
	case <-time.After(timeout):
		return ErrTimeout
	case err = <-task.wait:
		return err
	case <-task.stop:
		return nil
	}
}
func (task *TaskCb) NowaitCb(cb func(cMsg ConnMsg) error) error {
	if task.ctx.disable.Load() {
		return ErrIsDisable
	}
	task.cb = cb
	task.ctx.taskMap.Store(task.id, task)
	if task.fn != nil {
		err := task.fn()
		if err != nil {
			return err
		}
	}
	return nil
}
func (task *TaskCb) KeepWaitCb(timeout time.Duration, cb func(cMsg ConnMsg) error) (err error) {
	defer func() {
		if err != nil {
			task.OverTask(nil)
		}
	}()
	if task.ctx.disable.Load() {
		return ErrIsDisable
	}
	task.cb = cb
	task.ctx.taskMap.Store(task.id, task)
	if task.fn != nil {
		err = task.fn()
		if err != nil {
			return err
		}
	}
	tk := time.NewTicker(timeout)
	defer tk.Stop()
	for {
		select {
		case <-tk.C:
			return ErrTimeout
		case err = <-task.wait:
			tk.Reset(timeout)
			if err != nil {
				return err
			}
		case err = <-task.stop:
			return err
		}
	}
}

func (task *TaskCb) SetKeep(keep bool) *TaskCb {
	task.keep.Store(keep)
	return task
}
func (task *TaskCb) OverTask(err error) {
	select {
	case task.stop <- err:
	default:
	}
	task.disable.Store(true)
	task.ctx.taskMap.Delete(task.id)
}

var TaskSkipErr = errors.New("task skip err")

var TaskCbIdIsNeed = errors.New("task cb id is nil")
