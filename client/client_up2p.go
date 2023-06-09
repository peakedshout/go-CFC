package client

import (
	"context"
	"github.com/peakedshout/go-CFC/control"
	"github.com/peakedshout/go-CFC/tool"
	"github.com/xtaci/kcp-go"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// GetSubBoxByUP2P udp to kcp
func (box *DeviceBox) GetSubBoxByUP2P(name string) (*SubBox, error) {
	var info tool.OdjUP2PKId
	err := box.taskCbCtx.NewTaskCbCMsg(tool.P2PUdpQ1, 200, tool.OdjUP2PKName{Name: name}).WaitCb(10*time.Second, func(cMsg tool.ConnMsg) error {
		err1 := cMsg.CheckConnMsgHeaderAndCode(tool.P2PUdpQ1, 200)
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
	conn, si, err := box.handleUP2P(info.Id, true)
	if err != nil {
		box.SetWarnLog(err)
		return nil, err
	}

	sub := &SubBox{
		id:           tool.NewId(1),
		subType:      SubTypeUP2P,
		addr:         si,
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
	box.setSubBox(sub.id, sub)
	return sub, nil
}

func (box *DeviceBox) listenUP2P(cMsg tool.ConnMsg) {
	var info tool.OdjUP2PKId
	err := cMsg.Unmarshal(&info)
	if err != nil {
		box.SetWarnLog(err)
		return
	}
	go func() {
		conn, si, err := box.handleUP2P(info.Id, false)
		if err != nil {
			box.SetWarnLog(err)
			return
		}

		sub := &SubBox{
			id:           tool.NewId(1),
			addr:         si,
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

		select {
		case box.subListen <- sub:
			box.setSubBox(sub.id, sub)
		case err := <-box.subListenStop:
			box.subListenStop <- err
			sub.Close()
		}
	}()
}

func (box *DeviceBox) handleUP2P(kid string, IsClient bool) (net.Conn, *tool.SubInfo, error) {
	pc, err := newUP2PLn("")
	if err != nil {
		return nil, nil, err
	}
	defer pc.Close()
	stop := make(chan struct{})
	go func() {
		bs := box.key.SetMsg(tool.P2PUdpO, "", 200, tool.OdjUP2PReq{
			Id:      kid,
			SrcName: box.name,
			Addr:    pc.LocalAddr().(*net.UDPAddr),
		})
		for i := 0; i < 3; i++ {
			select {
			case <-stop:
				break
			case <-time.After(1 * time.Second):
				for _, one := range bs {
					_, err := pc.WriteTo(one, &net.UDPAddr{
						IP:   box.addr.IP,
						Port: box.addr.Port,
						Zone: box.addr.Zone,
					})
					if err != nil {
						box.SetInfoLog(err)
						return
					}
				}
			}
		}
	}()

	var info tool.OdjUP2PResp
	err = box.taskCbCtx.NewTaskCb(kid, nil).WaitCb(10*time.Second, func(cMsg tool.ConnMsg) error {
		err1 := cMsg.CheckConnMsgHeaderAndCode(tool.P2PUdpA1, 200)
		if err1 != nil {
			box.SetInfoLog(err1)
			return err1
		}
		err1 = cMsg.Unmarshal(&info)
		if err1 != nil {
			box.SetInfoLog(err1)
			return err1
		}
		stop <- struct{}{}
		return nil
	})

	if err != nil {
		box.SetWarnLog(err)
		return nil, nil, err
	}

	err = handshakeUP2P(pc, info.Addr, kid)
	if err != nil {
		box.SetWarnLog(err)
		return nil, nil, err
	}

	time.Sleep(100 * time.Millisecond)

	pc2, err := newUP2PLn(pc.LocalAddr().String())
	if err != nil {
		return nil, nil, err
	}
	pc2.SetDeadline(time.Now().Add(10 * time.Second))

	if IsClient {
		kconn, err := kcp.NewConn2(info.Addr, nil, 0, 0, pc2)
		if err != nil {
			box.SetWarnLog(err)
			return nil, nil, err
		}
		pc2.SetDeadline(time.Time{})
		kcpConnSetParam(kconn)
		return kconn, info.Info, nil
	} else {
		ln, err := kcp.ServeConn(nil, 0, 0, pc2)
		if err != nil {
			box.SetWarnLog(err)
			return nil, nil, err
		}
		kcpListenSetParam(ln)
		for {
			kconn, err := ln.AcceptKCP()
			if err != nil {
				box.SetWarnLog(err)
				return nil, nil, err
			}
			if kconn.RemoteAddr().String() != info.Addr.String() {
				kconn.Close()
				continue
			}
			pc2.SetDeadline(time.Time{})
			kcpConnSetParam(kconn)
			return kconn, info.Info, nil
		}
	}
}

func handshakeUP2P(conn net.PacketConn, addr net.Addr, kid string) error {
	defer conn.Close()
	stop := make(chan struct{})
	timeout := true
	go func() {
		for timeout {
			buf := make([]byte, 4096)
			n, xaddr, err := conn.ReadFrom(buf)
			if err != nil {
				return
			}
			if xaddr.String() != addr.String() {
				continue
			}
			if kid != string(buf[:n]) {
				continue
			}
			timeout = false
			stop <- struct{}{}
			break
		}
	}()
	for i := 0; i < 3; i++ {
		select {
		case <-stop:
			break
		case <-time.After(1 * time.Second):
			_, err := conn.WriteTo([]byte(kid), addr)
			if err != nil {
				return err
			}
		}
	}
	if timeout {
		return tool.ErrTimeout
	} else {
		return nil
	}
}

func newUP2PLn(addr string) (net.PacketConn, error) {
	lc := net.ListenConfig{Control: control.NetControl}
	return lc.ListenPacket(context.Background(), "udp", addr)
}

func kcpConnSetParam(conn *kcp.UDPSession) {
	conn.SetStreamMode(true)
	conn.SetWindowSize(8192, 8192)
	conn.SetReadBuffer(1 * 1024 * 1024)
	conn.SetWriteBuffer(1 * 1024 * 1024)
	conn.SetNoDelay(0, 100, 1, 1)
	conn.SetMtu(1024)
	conn.SetACKNoDelay(false)
}

func kcpListenSetParam(listener *kcp.Listener) {
	listener.SetReadBuffer(1 * 1024 * 1024)
	listener.SetWriteBuffer(1 * 1024 * 1024)
	listener.SetDSCP(46)
}
