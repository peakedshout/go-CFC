package server

import (
	"bufio"
	"github.com/peakedshout/go-CFC/loger"
	"github.com/peakedshout/go-CFC/tool"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

func NewProxyServer(addr, key string) *ProxyServer {
	config := &Config{
		RawKey:           key,
		LnAddr:           addr,
		lnAddr:           nil,
		HandleWaitTime:   0,
		PingWaitTime:     0,
		CGTaskTime:       0,
		SwitchVPNProxy:   false,
		SwitchLinkClient: true,
	}
	return NewProxyServer2(config)
}

func NewProxyServer2(config *Config) *ProxyServer {
	config.check()
	ps := &ProxyServer{
		addr:               config.lnAddr,
		ln:                 nil,
		stop:               make(chan uint8, 1),
		key:                tool.NewKey(config.RawKey),
		proxyClientMap:     sync.Map{},
		proxyClientMapLock: sync.Mutex{},
		proxyTaskRoomMap:   sync.Map{},
		proxyUP2PMap:       sync.Map{},
		config:             config,
		CloseWaiter:        tool.NewCloseWaiter(),
	}
	ps.setCloseFn()

	if ps.config.SwitchUdpP2P {
		ps.listenUP2P()
	}

	ps.gcTask()

	ps.listenTcp()

	loger.SetLogMust(loger.SprintColor(5, 37, 37, "~~~ Start Proxy Server ~~~"))
	return ps
}

func (ps *ProxyServer) gcTask() {
	go func() {
		t := time.NewTimer(ps.config.CGTaskTime)
		defer t.Stop()
		select {
		case <-t.C:
			ps.delExpireTaskRoom()
			if ps.config.SwitchUdpP2P {
				ps.delExpireUP2P()
			}
		case <-ps.stop:
			ps.stop <- 1
			return
		}
	}()
}

func (ps *ProxyServer) listenTcp() {
	ln, err := net.Listen(ps.addr.Network(), ps.addr.String())
	if err != nil {
		loger.SetLogError("Listen :" + err.Error())
	}
	ps.ln = ln
	go func() {
		defer ps.ln.Close()
		for {
			conn, err := ln.Accept()
			if err != nil {
				loger.SetLogWarn("Accept :", err)
				return
			}
			go ps.tcpHandler(conn)
		}
	}()
}

func (ps *ProxyServer) tcpHandler(conn net.Conn) {
	pc := &ProxyClient{
		ps:           ps,
		id:           tool.NewId(1),
		name:         "",
		disable:      atomic.Bool{},
		reader:       bufio.NewReaderSize(conn, tool.BufferSize),
		key:          ps.key,
		writeLock:    sync.Mutex{},
		rawConn:      conn,
		linkConn:     nil,
		linkSwitch:   atomic.Bool{},
		linkType:     "",
		linkBox:      nil,
		ping:         tool.Ping{},
		networkSpeed: tool.NewNetworkSpeedTicker(),
		step:         0,
		closerOnce:   sync.Once{},
		parent:       nil,
		subMap:       sync.Map{},
		subMapLock:   sync.Mutex{},
	}
	defer pc.close()
	pc.SetDeadline(time.Now().Add(ps.config.PingWaitTime))

	for {
		if pc.linkSwitch.Load() {
			err := pc.writeLinkConn()
			if err != nil {
				pc.SetInfoLog("linkConn:", err)
				return
			}
		} else {
			cMsg, err := pc.readCMsg()
			if err != nil {
				if err == tool.ErrReadCSkipToFastConn {
					continue
				}
				pc.SetInfoLog(err)
				break
			}
			pc.cMsgHandler(cMsg)
		}
	}
}

func (pc *ProxyClient) cMsgHandler(cMsg tool.ConnMsg) {
	pc.SetInfoLog(cMsg)
	switch pc.step {
	case ProxyInitialization:
		pc.cMsgProxyInitialization(cMsg)
	case ProxyRegister:
		pc.cMsgProxyRegister(cMsg)
	case ProxyBusiness:
		pc.cMsgProxyBusiness(cMsg)
	}
}

func (pc *ProxyClient) cMsgProxyInitialization(cMsg tool.ConnMsg) {
	switch cMsg.Header {
	case tool.HandshakeCheckStepQ1:
		pc.step += 1
		pc.writeCMsgAndCheck(tool.HandshakeCheckStepA1, cMsg.Id, 200, nil)
	case tool.TaskQ:
		if !pc.ps.config.SwitchLinkClient {
			err := tool.ErrMethodIsRefused
			pc.writeCMsgAndCheck(tool.TaskA, cMsg.Id, 401, tool.NewErrMsg("bad req : ", err))
			pc.close()
			pc.SetInfoLog(err)
			return
		}
		var info tool.OdjSubReq
		err := cMsg.Unmarshal(&info)
		if pc.checkErrAndSend400ErrCMsg(tool.TaskA, cMsg.Id, err, true) {
			return
		}
		if info.DstKey == "" {
			err = tool.ErrSubDstKeyIsNil
			pc.checkErrAndSend400ErrCMsg(tool.TaskA, cMsg.Id, err, true)
			return
		}
		if info.Addr == nil {
			err = tool.ErrSubLocalAddrIsNil
			pc.checkErrAndSend400ErrCMsg(tool.TaskA, cMsg.Id, err, true)
			return
		}
		if info.SrcName != "" {
			parent, ok := pc.ps.getProxyClient(info.SrcName)
			if !ok {
				err = tool.ErrHandleCMsgMissProxyClient
				pc.checkErrAndSend400ErrCMsg(tool.TaskA, cMsg.Id, err, true)
				return
			}
			parent.setProxySubClient(pc.id, pc)
			pc.parent = parent
		}
		pc.ps.joinTaskRoom(info, pc)
	case tool.ConnVPNQ1:
		if !pc.ps.config.SwitchVPNProxy {
			err := tool.ErrMethodIsRefused
			pc.writeCMsgAndCheck(tool.ConnVPNA1, cMsg.Id, 401, tool.NewErrMsg("bad req : ", err))
			pc.close()
			pc.SetInfoLog(err)
			return
		}
		var info tool.OdjVPNLinkAddr
		err := cMsg.Unmarshal(&info)
		if pc.checkErrAndSend400ErrCMsg(tool.ConnVPNA1, cMsg.Id, err, true) {
			return
		}
		err = pc.linkVPNConn(&info)
		if pc.checkErrAndSend400ErrCMsg(tool.ConnVPNA1, cMsg.Id, err, true) {
			return
		}
		pc.writeCMsgAndCheck(tool.ConnVPNA1, cMsg.Id, 200, info.AddrInfo)
	}
}

func (pc *ProxyClient) cMsgProxyRegister(cMsg tool.ConnMsg) {
	switch cMsg.Header {
	case tool.HandshakeCheckStepQ2:
		var info tool.OdjClientInfo
		err := cMsg.Unmarshal(&info)
		if pc.checkErrAndSend400ErrCMsg(tool.HandshakeCheckStepA2, cMsg.Id, err, true) {
			return
		}
		if info.Name != "" {
			pc.name = info.Name
		} else if info.Anonymity && pc.ps.config.SwitchAnonymity {
			info.Name = tool.NewId(2)
			pc.name = info.Name
		} else {
			err = tool.ErrHandleCMsgProxyClientNameIsNil
			pc.checkErrAndSend400ErrCMsg(tool.HandshakeCheckStepA2, cMsg.Id, err, true)
			return
		}
		pc.ps.setProxyClient(pc.name, pc)
		pc.writeCMsgAndCheck(tool.HandshakeCheckStepA2, cMsg.Id, 200, info)
		pc.step += 1
	}
}

func (pc *ProxyClient) cMsgProxyBusiness(cMsg tool.ConnMsg) {
	switch cMsg.Header {
	case tool.PingMsg:
		var info tool.Ping
		err := cMsg.Unmarshal(&info)
		if pc.checkErrAndSend400ErrCMsg(tool.PingMsg, cMsg.Id, err, true) {
			return
		}
		pc.ping = info
		pc.SetDeadline(time.Now().Add(pc.ps.config.PingWaitTime))
		pc.writeCMsgAndCheck(tool.PongMsg, cMsg.Id, 200, nil)
	case tool.SOpenQ:
		if !pc.ps.config.SwitchLinkClient {
			err := tool.ErrMethodIsRefused
			pc.writeCMsgAndCheck(tool.SOpenA, cMsg.Id, 401, tool.NewErrMsg("bad req : ", err))
			pc.SetInfoLog(err)
			return
		}
		var info tool.OdjSubOpenReq
		err := cMsg.Unmarshal(&info)
		if pc.checkErrAndSend400ErrCMsg(tool.SOpenA, cMsg.Id, err, false) {
			return
		}
		odj, ok := pc.ps.getProxyClient(info.OdjName)
		if !ok {
			err = tool.ErrHandleCMsgMissProxyClient
			pc.checkErrAndSend400ErrCMsg(tool.SOpenA, cMsg.Id, err, false)
			return
		}
		tid := pc.ps.newTaskRoom()
		odj.writeCMsgAndCheck(tool.SOpenA, "", 200, tool.OdjSubOpenResp{
			Tid:  tid,
			Type: info.Type,
		})
		pc.writeCMsgAndCheck(tool.SOpenA, cMsg.Id, 200, tool.OdjSubOpenResp{
			Tid:  tid,
			Type: info.Type,
		})
	case tool.DelayQ:
		var info tool.OdjIdList
		err := cMsg.Unmarshal(&info)
		if pc.checkErrAndSend400ErrCMsg(tool.DelayA, cMsg.Id, err, false) {
			return
		}
		pingList := pc.ps.getProxyClientDelay(info.IdList...)
		pc.writeCMsgAndCheck(tool.DelayA, cMsg.Id, 200, pingList)
	case tool.SpeedQ0:
		pc.writeCMsgAndCheck(tool.SpeedA0, cMsg.Id, 200, pc.ps.getAllProxyClientNetworkSpeed())
	case tool.SpeedQ1:
		var info tool.OdjIdList
		err := cMsg.Unmarshal(&info)
		if pc.checkErrAndSend400ErrCMsg(tool.SpeedA1, cMsg.Id, err, false) {
			return
		}
		pc.writeCMsgAndCheck(tool.SpeedA1, cMsg.Id, 200, pc.ps.getProxyClientNetworkSpeed(info.IdList...))
	case tool.SpeedQ2:
		pc.writeCMsgAndCheck(tool.SpeedA2, cMsg.Id, 200, pc.getAllNetworkSpeed())
	case tool.SpeedQ3:
		pc.writeCMsgAndCheck(tool.SpeedA3, cMsg.Id, 200, pc.getNetworkSpeed())
	case tool.P2PUdpQ1:
		if !pc.ps.config.SwitchUdpP2P {
			err := tool.ErrMethodIsRefused
			pc.writeCMsgAndCheck(tool.P2PUdpQ1, cMsg.Id, 401, tool.NewErrMsg("bad req : ", err))
			pc.SetInfoLog(err)
			return
		}
		var info tool.OdjUP2PKName
		err := cMsg.Unmarshal(&info)
		if pc.checkErrAndSend400ErrCMsg(tool.P2PUdpQ1, cMsg.Id, err, false) {
			return
		}
		c, ok := pc.ps.getProxyClient(info.Name)
		if !ok {
			if pc.checkErrAndSend400ErrCMsg(tool.P2PUdpQ1, cMsg.Id, tool.ErrHandleCMsgMissProxyClient, false) {
				return
			}
		}

		id := pc.ps.newAndSetUP2P()
		var resp tool.OdjUP2PKId
		resp.Id = id
		pc.writeCMsgAndCheck(tool.P2PUdpQ1, cMsg.Id, 200, resp)
		c.writeCMsgAndCheck(tool.P2PUdpQ1, "", 200, resp)
	}
}

func (ps *ProxyServer) setCloseFn() {
	ps.CloseWaiter.AddCloseFn(func() {
		if ps.ln != nil {
			ps.ln.Close()
		}
		if ps.uP2PLn != nil {
			ps.uP2PLn.Close()
		}
		ps.stop <- 1
		ps.proxyClientMapLock.Lock()
		defer ps.proxyClientMapLock.Unlock()
		ps.rangeProxyClient(func(key string, value *ProxyClient) {
			value.close()
		})
		loger.SetLogMust(loger.SprintColor(5, 37, 37, "~~~ Closed Proxy Server ~~~"))
	})
}

//func (ps *ProxyServer) Close() {
//	ps.ln.Close()
//	ps.stop <- 1
//	ps.proxyClientMapLock.Lock()
//	defer ps.proxyClientMapLock.Unlock()
//	ps.rangeProxyClient(func(key string, value *ProxyClient) {
//		value.close()
//	})
//	loger.SetLogMust(loger.SprintColor(5, 37, 37, "~~~ Closed Proxy Server ~~~"))
//}
//
//func (ps *ProxyServer) Wait() {
//	<-ps.stop
//	loger.SetLogMust(loger.SprintColor(5, 37, 37, "~~~ EndWait Proxy Server ~~~"))
//	ps.stop <- 1
//}
