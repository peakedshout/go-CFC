package client

import (
	"github.com/peakedshout/go-CFC/tool"
	"time"
)

type taskCb struct {
	box  *DeviceBox
	id   string
	fn   func()
	cb   func(cMsg tool.ConnMsg) error
	wait chan error
}

func (box *DeviceBox) delTask(key string) {
	box.taskFnMap.Delete(key)
}
func (box *DeviceBox) getTask(key string) (*taskCb, bool) {
	odj, ok := box.taskFnMap.LoadAndDelete(key)
	if !ok {
		return nil, false
	}
	fn, ok := odj.(*taskCb)
	if !ok {
		return nil, false
	}
	return fn, true
}
func (box *DeviceBox) getTaskAndRun(key string, cMsg tool.ConnMsg) {
	task, ok := box.getTask(key)
	if ok {
		task.wait <- task.cb(cMsg)
	}
}

//	func (box *DeviceBox) setTask(key string, fn func(cMsg tool.ConnMsg)) *taskCb {
//		task := &taskCb{
//			cb:   fn,
//			wait: make(chan uint8, 1),
//		}
//		box.taskFnMap.Store(key, task)
//		return task
//	}
//
//	func (box *DeviceBox) setTaskAndWait(key string, fn func(cMsg tool.ConnMsg), timeout time.Duration) error {
//		task := box.setTask(key, fn)
//		select {
//		case <-time.After(timeout):
//			return tool.ErrTimeout
//		case <-task.wait:
//			return nil
//		}
//	}

func (box *DeviceBox) newTaskCb(header string, code int, data interface{}) *taskCb {
	id := tool.NewId(1)
	return &taskCb{
		box: box,
		id:  id,
		fn: func() {
			box.writerCMsg(header, id, code, data)
		},
		cb:   nil,
		wait: make(chan error, 1),
	}
}
func (task *taskCb) waitCb(timeout time.Duration, cb func(cMsg tool.ConnMsg) error) error {
	if task.box.disable.Load() {
		return tool.ErrIsDisable
	}
	task.cb = cb
	task.box.taskFnMap.Store(task.id, task)
	task.fn()
	select {
	case <-time.After(timeout):
		return tool.ErrTimeout
	case err := <-task.wait:
		return err
	}
}
func (task *taskCb) nowaitCb(cb func(cMsg tool.ConnMsg) error) {
	if task.box.disable.Load() {
		return
	}
	task.cb = cb
	task.box.taskFnMap.Store(task.id, task)
	task.fn()
}
