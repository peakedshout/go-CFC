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
	if len(key) != 32 {
		loger.SetLogFatal(tool.ErrKeyIsNot32Bytes)
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		loger.SetLogFatal("ResolveTCPAddr :", err)
	}

	ps := &ProxyServer{
		addr:               tcpAddr,
		ln:                 nil,
		stop:               make(chan uint8, 1),
		key:                tool.NewKey(key),
		proxyClientMap:     sync.Map{},
		proxyClientMapLock: sync.Mutex{},
		proxyTaskRoomMap:   sync.Map{},
	}
	go func() {
		t := time.NewTimer(1 * time.Minute)
		defer t.Stop()
		select {
		case <-t.C:
			ps.delExpireTaskRoom()
		case <-ps.stop:
			ps.stop <- 1
			return
		}
	}()

	ln, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		loger.SetLogError("ListenTCP :" + err.Error())
	}
	ps.ln = ln
	go func() {
		defer ps.ln.Close()
		for {
			conn, err := ln.AcceptTCP()
			if err != nil {
				loger.SetLogWarn("AcceptTCP :", err)
				return
			}
			go ps.tcpHandler(conn)
		}
	}()
	loger.SetLogMust(loger.SprintColor(5, 37, 37, "~~~ Start Proxy Server ~~~"))
	return ps
}
func (ps *ProxyServer) tcpHandler(conn *net.TCPConn) {
	i := int64(0)
	pc := &ProxyClient{
		s:            ps,
		id:           tool.NewId(1),
		name:         "",
		disable:      atomic.Bool{},
		fastOdj:      nil,
		fastConn:     atomic.Bool{},
		fastOdjChan:  nil,
		step:         &i,
		stop:         make(chan uint8, 1),
		closerOnce:   sync.Once{},
		conn:         conn,
		writeChan:    make(chan [][]byte, 1),
		ping:         tool.Ping{},
		networkSpeed: tool.NewNetworkSpeedTicker(),
		parent:       nil,
		subMap:       sync.Map{},
		subMapLock:   sync.Mutex{},
	}
	defer pc.close()
	pc.SetDeadline(10 * time.Second)
	go func() {
		for {
			select {
			case b := <-pc.writeChan:
				for _, one := range b {
					n, err := pc.conn.Write(one)
					if err != nil {
						pc.SetInfoLog(err)
						return
					}
					pc.networkSpeed.Download.Set(n)
				}
			case <-pc.stop:
				pc.stop <- 1
				return
			}
		}
	}()
	reader := bufio.NewReaderSize(conn, tool.BufferSize)
	for {
		cMsg, err := ps.key.ReadCMsg(reader, &pc.fastConn, pc.networkSpeed.Upload)
		if err != nil {
			loger.SetLogInfo(err)
			break
		}
		ps.cMsgHandler(pc, cMsg)
	}
	for {
		if pc.fastConn.Load() {
			var b [4 * 1024]byte
			n, err := reader.Read(b[:])
			if err != nil {
				pc.SetInfoLog(err)
				pc.close()
				if pc.fastOdj != nil {
					pc.fastOdj.close()
				}
				return
			}
			pc.writerFast(b[:n])
		} else {
			pc.close()
			break
		}
	}
}
func (ps *ProxyServer) cMsgHandler(pc *ProxyClient, cMsg tool.ConnMsg) {
	pc.SetInfoLog(cMsg)
	switch atomic.LoadInt64(pc.step) {
	case ProxyInitialization:
		ps.cMsgProxyInitialization(pc, cMsg)
	case ProxyRegister:
		ps.cMsgProxyRegister(pc, cMsg)
	case ProxyBusiness:
		ps.cMsgProxyBusiness(pc, cMsg)
	}
}
func (ps *ProxyServer) cMsgProxyInitialization(pc *ProxyClient, cMsg tool.ConnMsg) {
	switch cMsg.Header {
	case tool.HandshakeCheckStepQ1:
		atomic.AddInt64(pc.step, 1)
		pc.writerCMsg(tool.HandshakeCheckStepA1, cMsg.Id, 200, nil)
	case tool.TaskQ:
		var info tool.OdjSub
		err := tool.UnmarshalV2(cMsg.Data, &info)
		if err != nil {
			pc.close()
			pc.SetInfoLog(err)
			return
		}
		if info.DstKey == "" {
			pc.close()
			pc.SetInfoLog(err)
			return
		}
		if info.SrcName != "" {
			parent, ok := ps.getProxyClient(info.SrcName)
			if !ok {
				pc.close()
				pc.SetInfoLog(tool.ErrHandleCMsgMissProxyClient, info.SrcName)
				return
			}
			parent.setProxySubClient(pc.id, pc)
			pc.parent = parent
		}
		ps.joinTaskRoom(info.DstKey, pc)
	}
}
func (ps *ProxyServer) cMsgProxyRegister(pc *ProxyClient, cMsg tool.ConnMsg) {
	switch cMsg.Header {
	case tool.HandshakeCheckStepQ2:
		var info tool.OdjClientInfo
		err := tool.UnmarshalV2(cMsg.Data, &info)
		if err != nil {
			pc.close()
			pc.SetInfoLog(err)
			return
		}
		if info.Name == "" {
			pc.writerCMsg(tool.HandshakeCheckStepA2, cMsg.Id, 400, nil)
			pc.close()
			pc.SetInfoLog(tool.ErrHandleCMsgProxyClientNameIsNil)
			return
		}
		pc.name = info.Name
		ps.setProxyClient(pc.name, pc)
		pc.writerCMsg(tool.HandshakeCheckStepA2, cMsg.Id, 200, nil)
		atomic.AddInt64(pc.step, 1)
	}
}
func (ps *ProxyServer) cMsgProxyBusiness(pc *ProxyClient, cMsg tool.ConnMsg) {
	switch cMsg.Header {
	case tool.PingMsg:
		var info tool.Ping
		err := tool.UnmarshalV2(cMsg.Data, &info)
		if err != nil {
			pc.close()
			pc.SetInfoLog(err)
			return
		}
		pc.ping = info
		pc.SetDeadline(60 * time.Second)
		pc.writerCMsg(tool.PongMsg, cMsg.Id, 200, nil)
	case tool.SOpenQ:
		var info tool.OdjMsg
		err := tool.UnmarshalV2(cMsg.Data, &info)
		if err != nil {
			pc.writerCMsg(tool.SOpenA, cMsg.Id, 400, tool.OdjMsg{Msg: "bad req :" + err.Error()})
			pc.SetInfoLog(err)
			return
		}
		odj, ok := ps.getProxyClient(info.Msg)
		if !ok {
			err = tool.ErrHandleCMsgMissProxyClient
			pc.writerCMsg(tool.SOpenA, cMsg.Id, 400, tool.OdjMsg{Msg: "bad req :" + err.Error()})
			pc.SetInfoLog(err, info.Msg)
			return
		}
		tid := ps.newTaskRoom()
		odj.writerCMsg(tool.SOpenA, "", 200, tool.OdjMsg{Msg: tid})
		pc.writerCMsg(tool.SOpenA, cMsg.Id, 200, tool.OdjMsg{Msg: tid})
	case tool.DelayQ:
		var info tool.OdjIdList
		err := tool.UnmarshalV2(cMsg.Data, &info)
		if err != nil {
			pc.writerCMsg(tool.DelayA, cMsg.Id, 400, tool.OdjMsg{Msg: "bad req :" + err.Error()})
			pc.SetInfoLog(err)
			return
		}
		pingList := ps.getProxyClientDelay(info.IdList...)
		pc.writerCMsg(tool.DelayA, cMsg.Id, 200, pingList)
	case tool.SpeedQ0:
		pc.writerCMsg(tool.SpeedA0, cMsg.Id, 200, ps.getAllProxyClientNetworkSpeed())
	case tool.SpeedQ1:
		var info tool.OdjIdList
		err := tool.UnmarshalV2(cMsg.Data, &info)
		if err != nil {
			pc.writerCMsg(tool.SpeedA1, cMsg.Id, 400, tool.OdjMsg{Msg: "bad req :" + err.Error()})
			pc.SetInfoLog(err)
			return
		}
		pc.writerCMsg(tool.SpeedA1, cMsg.Id, 200, ps.getProxyClientNetworkSpeed(info.IdList...))
	case tool.SpeedQ2:
		pc.writerCMsg(tool.SpeedA2, cMsg.Id, 200, pc.getAllNetworkSpeed())
	case tool.SpeedQ3:
		pc.writerCMsg(tool.SpeedA3, cMsg.Id, 200, pc.getNetworkSpeed())
	}
}

func (ps *ProxyServer) Close() {
	ps.ln.Close()
	ps.stop <- 1
	ps.proxyClientMapLock.Lock()
	defer ps.proxyClientMapLock.Unlock()
	ps.rangeProxyClient(func(key string, value *ProxyClient) {
		value.close()
	})
	loger.SetLogMust(loger.SprintColor(5, 37, 37, "~~~ Closed Proxy Server ~~~"))
}

func (ps *ProxyServer) Wait() {
	<-ps.stop
	loger.SetLogMust(loger.SprintColor(5, 37, 37, "~~~ EndWait Proxy Server ~~~"))
	ps.stop <- 1
}
