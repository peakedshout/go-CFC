package server

import (
	"bufio"
	"bytes"
	"github.com/peakedshout/go-CFC/loger"
	"github.com/peakedshout/go-CFC/tool"
	"net"
	"sync"
	"time"
)

func (ps *ProxyServer) listenUP2P() {
	ln, err := net.ListenPacket("udp", ps.addr.String())
	if err != nil {
		loger.SetLogError("up2p Listen :" + err.Error())
	}
	ps.uP2PLn = ln
	go func() {
		for {
			buf := make([]byte, tool.BufferSize)
			n, addr, err := ps.uP2PLn.ReadFrom(buf)
			if err != nil {
				loger.SetLogWarn("up2p Accept :", err)
				return
			}
			ps.handleUP2P(buf[:n], addr)
		}
	}()
}

func (ps *ProxyServer) handleUP2P(b []byte, addr net.Addr) {
	reader := bufio.NewReader(bytes.NewReader(b))
	cMsg, err := ps.key.ReadCMsg(reader, nil, nil)
	if err != nil {
		loger.SetLogInfo("up2p", err)
		return
	}
	err = cMsg.CheckConnMsgHeaderAndCode(tool.P2PUdpO, 200)
	if err != nil {
		loger.SetLogInfo("up2p", err)
		return
	}
	var info tool.OdjUP2PReq
	err = cMsg.Unmarshal(&info)
	if err != nil {
		loger.SetLogInfo("up2p", err)
		return
	}
	pc, ok := ps.getProxyClient(info.SrcName)
	if !ok {
		return
	}
	up, ok := ps.getUP2P(info.Id)
	if ok {
		if up.handle(info.Addr, addr.(*net.UDPAddr), pc) {
			ps.delUP2P(info.Id)
		}
	}
}

func (ps *ProxyServer) newAndSetUP2P() string {
	id := tool.NewId(2)
	up2p := newUP2P(id)
	ps.setUP2P(id, up2p)
	return id
}

type proxyUP2P struct {
	id         string
	expireTime time.Time

	lock sync.Mutex

	step statusStep

	c1 *ProxyClient
	c2 *ProxyClient

	c1IAddr *net.UDPAddr
	c2IAddr *net.UDPAddr
	c1PAddr *net.UDPAddr
	c2PAddr *net.UDPAddr
}

func newUP2P(id string) *proxyUP2P {
	return &proxyUP2P{
		id:         id,
		expireTime: time.Now().Add(30 * time.Second),
		lock:       sync.Mutex{},
		step:       0,
	}
}

func (up2p *proxyUP2P) handle(iAddr, pAddr *net.UDPAddr, pc *ProxyClient) bool {
	up2p.lock.Lock()
	defer up2p.lock.Unlock()
	switch up2p.step {
	case 0:
		up2p.c1PAddr = pAddr
		up2p.c1IAddr = iAddr
		up2p.c1 = pc
		up2p.step++
		return false
	case 1:
		if pAddr.String() == up2p.c1PAddr.String() {
			return false
		}
		up2p.c2PAddr = pAddr
		up2p.c2IAddr = iAddr
		up2p.c2 = pc
		up2p.step++

		resp1 := &tool.SubInfo{
			LocalName:           up2p.c1.name,
			RemoteName:          up2p.c2.name,
			ULocalIntranetAddr:  up2p.c1IAddr,
			URemoteIntranetAddr: up2p.c2IAddr,
			ULocalPublicAddr:    up2p.c1PAddr,
			URemotePublicAddr:   up2p.c2PAddr,
		}
		resp2 := &tool.SubInfo{
			LocalName:           up2p.c2.name,
			RemoteName:          up2p.c1.name,
			ULocalIntranetAddr:  up2p.c2IAddr,
			URemoteIntranetAddr: up2p.c1IAddr,
			ULocalPublicAddr:    up2p.c2PAddr,
			URemotePublicAddr:   up2p.c1PAddr,
		}

		up2p.c1.writeCMsgAndCheck(tool.P2PUdpA1, up2p.id, 200, tool.OdjUP2PResp{Addr: up2p.c2PAddr, Info: resp1})
		up2p.c2.writeCMsgAndCheck(tool.P2PUdpA1, up2p.id, 200, tool.OdjUP2PResp{Addr: up2p.c1PAddr, Info: resp2})
		return true
	default:
		return true
	}
}

func (ps *ProxyServer) delUP2P(tid string) {
	ps.proxyUP2PMap.Delete(tid)
}
func (ps *ProxyServer) delExpireUP2P() {
	t := time.Now()
	ps.rangeUP2P(func(key string, value *proxyUP2P) {
		if t.Sub(value.expireTime) > 0 {
			ps.proxyUP2PMap.Delete(key)
		}
	})
}
func (ps *ProxyServer) getUP2P(tid string) (*proxyUP2P, bool) {
	odj, ok := ps.proxyUP2PMap.Load(tid)
	if !ok {
		return nil, false
	}
	task, ok := odj.(*proxyUP2P)
	if !ok {
		return nil, false
	}
	return task, true
}
func (ps *ProxyServer) setUP2P(tid string, ptr *proxyUP2P) {
	ps.proxyUP2PMap.Store(tid, ptr)
}
func (ps *ProxyServer) rangeUP2P(fn func(key string, value *proxyUP2P)) {
	ps.proxyUP2PMap.Range(func(key, value any) bool {
		odj, ok := value.(*proxyUP2P)
		if !ok {
			ps.proxyUP2PMap.Delete(key)
		} else {
			fn(key.(string), odj)
		}
		return true
	})
}
