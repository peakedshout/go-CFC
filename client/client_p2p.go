package client

import (
	"context"
	"github.com/peakedshout/go-CFC/control"
	"github.com/peakedshout/go-CFC/loger"
	"github.com/peakedshout/go-CFC/tool"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

func (box *DeviceBox) GetSubBoxByP2P(name string) (*SubBox, error) {
	var info tool.OdjSubOpenResp
	err := box.taskCbCtx.NewTaskCbCMsg(tool.SOpenQ, 200, tool.OdjSubOpenReq{
		Type:    tool.SubOpenTypeTCPP2P,
		OdjName: name,
	}).WaitCb(10*time.Second, func(cMsg tool.ConnMsg) error {
		err1 := cMsg.CheckConnMsgHeaderAndCode(tool.SOpenA, 200)
		if err1 != nil {
			box.SetInfoLog(err1)
			return err1
		}
		err1 = cMsg.Unmarshal(&info)
		if err1 != nil {
			box.SetInfoLog(err1)
			return err1
		}
		return nil
	})
	if err != nil {
		box.SetWarnLog(err)
		return nil, err
	}
	rconn, err := newDialer(nil, 0).Dial("tcp", box.addr.String())
	if err != nil {
		box.SetWarnLog(err)
		return nil, err
	}
	conn := rconn.(*net.TCPConn)
	sub := &SubBox{
		id:           tool.NewId(1),
		addr:         nil,
		key:          box.key,
		conn:         conn,
		root:         box,
		parent:       nil,
		networkSpeed: tool.NewNetworkSpeedTicker(),
		writerLock:   sync.Mutex{},
		disable:      atomic.Bool{},
		subMap:       sync.Map{},
		subMapLock:   sync.Mutex{},
		closerOnce:   sync.Once{},
	}
	defer conn.Close()
	err = sub.fastHandshake(info.Tid)
	if err != nil {
		err = tool.ErrOpenSubBoxBadAny(err)
		sub.Close()
		box.SetWarnLog(err)
		return nil, err
	}
	sub.SetDeadlineDuration(0)

	var ln net.Listener
	go func() {
		for i := 0; i < 3; i++ {
			_, err := newDialer(conn.LocalAddr(), 3*time.Second).Dial(sub.GetRemotePublicAddr().Network(), sub.GetRemotePublicAddr().String())
			if err != nil {
				loger.SetLogMust(err)
				//return
			}
			time.Sleep(500 * time.Millisecond)
		}
		time.Sleep(15 * time.Second)
		ln.Close()
	}()
	ln, err = newListenConfig().Listen(context.Background(), rconn.LocalAddr().Network(), rconn.LocalAddr().String())
	if err != nil {
		sub.Close()
		box.SetWarnLog(err)
		return nil, err
	}
	defer ln.Close()
	lconn, err := ln.Accept()
	if err != nil {
		sub.Close()
		box.SetWarnLog(err)
		return nil, err
	}
	sub.conn = lconn.(*net.TCPConn)

	box.setSubBox(sub.id, sub)
	return sub, nil
}

func newDialer(lAddr net.Addr, timeout time.Duration) *net.Dialer {
	var t time.Time
	if timeout != 0 {
		t = time.Now().Add(timeout)
	}
	dialer := &net.Dialer{
		//Timeout:   timeout,
		Deadline:  t,
		LocalAddr: lAddr,
		Control:   control.NetControl,
	}
	return dialer
}
func newListenConfig() *net.ListenConfig {
	return &net.ListenConfig{
		Control:   control.NetControl,
		KeepAlive: 0,
	}
}
