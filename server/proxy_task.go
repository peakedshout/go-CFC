package server

import (
	"github.com/peakedshout/go-CFC/tool"
	"time"
)

type proxyTaskRoom struct {
	id   string
	join chan uint8

	expireTime time.Time

	c1 *ProxyClient
	c2 *ProxyClient

	i1 tool.OdjSubReq
	i2 tool.OdjSubReq
}

func (ps *ProxyServer) joinTaskRoom(info tool.OdjSubReq, pc *ProxyClient) {
	task, ok := ps.getTaskRoom(info.DstKey)
	if !ok {
		pc.SetInfoLog(tool.ErrHandleCMsgMissProxyTaskRoom)
		return
	}
	select {
	case <-task.join:
		task.c1 = pc
		task.i1 = info
	default:
		ps.delTaskRoom(info.DstKey)
		task.c2 = pc
		task.i2 = info
		task.c1.fastOdjChan = task.c2.writeChan
		task.c2.fastOdjChan = task.c1.writeChan
		task.c1.fastConn.Store(true)
		task.c2.fastConn.Store(true)
		task.c1.fastOdj = task.c2
		task.c2.fastOdj = task.c1

		info1 := tool.SubInfo{
			LocalName:          task.i1.SrcName,
			RemoteName:         task.i2.SrcName,
			LocalIntranetAddr:  task.i1.Addr,
			RemoteIntranetAddr: task.i2.Addr,
			LocalPublicAddr:    tool.MustResolveTCPAddr(task.c1.conn.RemoteAddr()),
			RemotePublicAddr:   tool.MustResolveTCPAddr(task.c2.conn.RemoteAddr()),
		}
		info2 := tool.SubInfo{
			LocalName:          task.i2.SrcName,
			RemoteName:         task.i1.SrcName,
			LocalIntranetAddr:  task.i2.Addr,
			RemoteIntranetAddr: task.i1.Addr,
			LocalPublicAddr:    tool.MustResolveTCPAddr(task.c2.conn.RemoteAddr()),
			RemotePublicAddr:   tool.MustResolveTCPAddr(task.c1.conn.RemoteAddr()),
		}

		task.c1.writerCMsg(tool.TaskA, task.i1.Id, 200, info1)
		task.c2.writerCMsg(tool.TaskA, task.i2.Id, 200, info2)
		task.c1.SetDeadline(0)
		task.c2.SetDeadline(0)

	}
}
func (ps *ProxyServer) newTaskRoom() string {
	t := &proxyTaskRoom{
		id:         tool.NewId(1),
		join:       make(chan uint8, 1),
		expireTime: time.Now().Add(30 * time.Second),
		c1:         nil,
		c2:         nil,
	}
	t.join <- 1
	ps.setTaskRoom(t.id, t)
	return t.id
}

func (ps *ProxyServer) delTaskRoom(tid string) {
	ps.proxyTaskRoomMap.Delete(tid)
}
func (ps *ProxyServer) delExpireTaskRoom() {
	t := time.Now()
	ps.rangeTaskRoom(func(key string, value *proxyTaskRoom) {
		if t.Sub(value.expireTime) > 0 {
			ps.proxyTaskRoomMap.Delete(key)
		}
	})
}
func (ps *ProxyServer) getTaskRoom(tid string) (*proxyTaskRoom, bool) {
	odj, ok := ps.proxyTaskRoomMap.Load(tid)
	if !ok {
		return nil, false
	}
	task, ok := odj.(*proxyTaskRoom)
	if !ok {
		return nil, false
	}
	return task, true
}
func (ps *ProxyServer) setTaskRoom(tid string, ptr *proxyTaskRoom) {
	ps.proxyTaskRoomMap.Store(tid, ptr)
}
func (ps *ProxyServer) rangeTaskRoom(fn func(key string, value *proxyTaskRoom)) {
	ps.proxyTaskRoomMap.Range(func(key, value any) bool {
		odj, ok := value.(*proxyTaskRoom)
		if !ok {
			ps.proxyTaskRoomMap.Delete(key)
		} else {
			fn(key.(string), odj)
		}
		return true
	})
}
