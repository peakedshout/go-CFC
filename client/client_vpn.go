package client

import (
	"github.com/peakedshout/go-CFC/loger"
	"github.com/peakedshout/go-CFC/tool"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type LinkConnReq struct {
	CopyConn  net.Conn
	ConnType  string
	ConnAddr  string
	ProxyAddr string
	ProxyKey  string
}

type LinkClient struct {
	stop    chan error
	disable atomic.Bool

	proxyAddr   string
	proxyKey    tool.Key
	proxySwitch bool
	proxySpeed  tool.NetworkSpeedTicker

	connAddr string
	connType string

	copyConn net.Conn

	linkConn net.Conn
	linkRw   *LinkClientRW
	wLock    sync.Mutex

	taskCbCtx *tool.TaskCbContext
	closer    sync.Once
}

func LinkOtherConn(req LinkConnReq) (*LinkClient, error) {
	vc := LinkClient{
		stop:        make(chan error, 1),
		disable:     atomic.Bool{},
		proxyAddr:   "",
		proxyKey:    tool.Key{},
		proxySwitch: false,
		proxySpeed:  tool.NetworkSpeedTicker{},
		connAddr:    req.ConnAddr,
		connType:    req.ConnType,
		copyConn:    req.CopyConn,
		linkConn:    nil,
		wLock:       sync.Mutex{},
		taskCbCtx:   nil,
		closer:      sync.Once{},
	}
	if req.ProxyAddr != "" {
		if len(req.ProxyKey) != 32 {
			loger.SetLogError(tool.ErrKeyIsNot32Bytes)
		}
		vc.proxyKey = tool.NewKey(req.ProxyKey)
		vc.proxyAddr = req.ProxyAddr
		vc.proxySwitch = true
		vc.proxySpeed = tool.NewNetworkSpeedTicker()
		err := vc.proxyConn()
		if err != nil {
			return nil, err
		}
	} else {
		err := vc.directConn()
		if err != nil {
			return nil, err
		}
	}
	return &vc, nil
}

func (lc *LinkClient) Close(err error) error {
	if lc.disable.Load() {
		return tool.ErrLinkClientIsClosed
	}
	lc.closer.Do(func() {
		lc.disable.Store(true)
		lc.taskCbCtx.Close()
		lc.copyConn.Close()
		lc.linkConn.Close()
		lc.stop <- err
	})
	return nil
}
func (lc *LinkClient) Wait() error {
	return <-lc.stop
}
func (lc *LinkClient) LocalAddr() net.Addr {
	return lc.copyConn.LocalAddr()
}
func (lc *LinkClient) RemoteAddr() net.Addr {
	return lc.linkConn.RemoteAddr()
}
func (lc *LinkClient) SetDeadline(t time.Time) error {
	err := lc.linkConn.SetDeadline(t)
	if err != nil {
		return err
	}
	return lc.copyConn.SetDeadline(t)
}
func (lc *LinkClient) SetReadDeadline(t time.Time) error {
	err := lc.linkConn.SetReadDeadline(t)
	if err != nil {
		return err
	}
	return lc.copyConn.SetReadDeadline(t)
}
func (lc *LinkClient) SetWriteDeadline(t time.Time) error {
	err := lc.linkConn.SetWriteDeadline(t)
	if err != nil {
		return err
	}
	return lc.copyConn.SetWriteDeadline(t)
}

func (lc *LinkClient) proxyConn() error {
	if lc.connType != tool.LinkConnTypeTCP && lc.connType != tool.LinkConnTypeUDP {
		loger.SetLogError()
	}
	raddr, err := net.ResolveTCPAddr("", lc.proxyAddr)
	if err != nil {
		return err
	}
	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		return err
	}
	lc.linkConn = conn
	lc.linkRw = &LinkClientRW{lc: lc}
	lc.taskCbCtx = tool.NewTaskContext(lc.linkRw, lc.proxyKey)
	lc.taskCbCtx.SetWriteLock(&lc.wLock)
	lc.handleCMsg()
	err = lc.handshakeCheck()
	if err != nil {
		return err
	}
	go func() {
		defer lc.copyConn.Close()
		for {
			buf := make([]byte, tool.BufferSize)
			n, err := lc.copyConn.Read(buf)
			if err != nil {
				return
			}
			err = lc.writeFromCopyConn(buf[:n])
			if err != nil {
				return
			}
		}
	}()
	return nil
}

func (lc *LinkClient) handleCMsg() {
	lc.taskCbCtx.SetNoCb(func(cMsg tool.ConnMsg) {
		err := cMsg.CheckConnMsgHeaderAndCode(tool.ConnVPNA2, 200)
		if err != nil {
			loger.SetLogInfo("vpn:", err)
			lc.Close(err)
			return
		}
		var b []byte
		err = cMsg.Unmarshal(&b)
		if err != nil {
			loger.SetLogInfo("vpn:", err)
			lc.Close(err)
			return
		}
		_, err = lc.copyConn.Write(b)
		if err != nil {
			loger.SetLogInfo("vpn:", err)
			lc.Close(err)
			return
		}
	})
	go func() {
		defer lc.Close(nil)
		err := lc.taskCbCtx.ReadCMsg()
		if err != nil {
			loger.SetLogInfo("vpn:", err)
			lc.Close(err)
			return
		}
	}()
}

func (lc *LinkClient) handshakeCheck() error {
	req := tool.OdjVPNLinkAddr{
		ConnType: lc.connType,
		Addr:     lc.connAddr,
	}
	err := lc.taskCbCtx.NewTaskCbCMsg(tool.ConnVPNQ1, 200, req).WaitCb(3*time.Second, func(cMsg tool.ConnMsg) error {
		err1 := cMsg.CheckConnMsgHeaderAndCode(tool.ConnVPNA1, 200)
		if err1 != nil {
			return err1
		}
		return nil
	})
	return err
}

func (lc *LinkClient) directConn() error {
	switch lc.connType {
	case tool.LinkConnTypeTCP:
		raddr, err := net.ResolveTCPAddr("", lc.connAddr)
		if err != nil {
			return err
		}
		conn, err := net.DialTCP("tcp", nil, raddr)
		if err != nil {
			return err
		}
		lc.linkConn = conn
	case tool.LinkConnTypeUDP:
		raddr, err := net.ResolveUDPAddr("", lc.connAddr)
		if err != nil {
			return err
		}
		conn, err := net.DialUDP("tcp", nil, raddr)
		if err != nil {
			return err
		}
		lc.linkConn = conn
	default:
		loger.SetLogError(tool.ErrUnexpectedLinkConnType)
	}
	go io.Copy(lc.linkConn, lc.copyConn)
	go func() {
		_, err := io.Copy(lc.copyConn, lc.linkConn)
		lc.Close(err)
	}()
	return nil
}

func (lc *LinkClient) writeFromCopyConn(b []byte) error {
	lc.wLock.Lock()
	defer lc.wLock.Unlock()

	bs := lc.proxyKey.SetMsg(tool.ConnVPNQ2, "", 200, b)
	for _, one := range bs {
		n, err := lc.linkConn.Write(one)
		if err != nil {
			return err
		}
		lc.proxySpeed.Upload.Set(n)
	}
	return nil
}

func (lc *LinkClient) WriteToLinkConn(b []byte) error {
	if lc.proxySwitch {
		return lc.writeFromCopyConn(b)
	} else {
		_, err := lc.linkConn.Write(b)
		return err
	}
}

type LinkClientRW struct {
	lc *LinkClient
}

func (lcrw *LinkClientRW) Read(b []byte) (n int, err error) {
	n, err = lcrw.lc.linkConn.Read(b)
	if err != nil {
		return 0, err
	}
	lcrw.lc.proxySpeed.Download.Set(n)
	return n, nil
}
func (lcrw *LinkClientRW) Write(b []byte) (n int, err error) {
	n, err = lcrw.lc.linkConn.Write(b)
	if err != nil {
		return 0, err
	}
	lcrw.lc.proxySpeed.Upload.Set(n)
	return n, nil
}
