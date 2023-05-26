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

type statusStep uint8

type ProxyClient struct {
	ps *ProxyServer

	id      string //session id
	name    string //login name
	disable atomic.Bool

	reader    *bufio.Reader
	key       tool.Key
	writeLock sync.Mutex

	rawConn net.Conn

	linkConn   net.Conn
	linkSwitch atomic.Bool
	linkType   string
	linkBox    *tool.LinkBox

	ping         tool.Ping
	networkSpeed tool.NetworkSpeedTicker

	step       statusStep
	closerOnce sync.Once

	parent *ProxyClient

	subMap     sync.Map //map[string]*ProxyClient
	subMapLock sync.Mutex
}

func (pc *ProxyClient) SetDeadline(t time.Time) error {
	return pc.rawConn.SetDeadline(t)
}
func (pc *ProxyClient) SetReadDeadline(t time.Time) error {
	return pc.rawConn.SetReadDeadline(t)
}
func (pc *ProxyClient) SetWriteDeadline(t time.Time) error {
	return pc.rawConn.SetWriteDeadline(t)
}
func (pc *ProxyClient) LocalAddr() net.Addr {
	return pc.rawConn.LocalAddr()
}
func (pc *ProxyClient) RemoteAddr() net.Addr {
	return pc.rawConn.RemoteAddr()
}

func (pc *ProxyClient) Write(b []byte) (n int, err error) {
	pc.writeLock.Lock()
	defer pc.writeLock.Unlock()
	if pc.disable.Load() {
		return 0, tool.ErrProxyClientIsClosed
	}
	n, err = pc.rawConn.Write(b)
	if err != nil {
		return 0, err
	}
	pc.networkSpeed.Download.Set(n)
	return n, nil
}

func (pc *ProxyClient) Read(b []byte) (n int, err error) {
	n, err = pc.reader.Read(b)
	if err != nil {
		return 0, nil
	}
	pc.networkSpeed.Upload.Set(n)
	return n, err
}

func (pc *ProxyClient) Close() error {
	if pc.disable.Load() {
		return tool.ErrProxyClientIsClosed
	}
	pc.close()
	return nil
}

func (pc *ProxyClient) Writes(b [][]byte) error {
	pc.writeLock.Lock()
	defer pc.writeLock.Unlock()
	if pc.disable.Load() {
		return tool.ErrProxyClientIsClosed
	}
	for _, one := range b {
		n, err := pc.rawConn.Write(one)
		if err != nil {
			return err
		}
		pc.networkSpeed.Download.Set(n)
	}
	return nil
}

func (pc *ProxyClient) writeCMsg(header, id string, code int, data interface{}) error {
	return pc.Writes(pc.key.SetMsg(header, id, code, data))
}

func (pc *ProxyClient) readCMsg() (cMsg tool.ConnMsg, err error) {
	return pc.key.ReadCMsg(pc.reader, &pc.linkSwitch, pc.networkSpeed.Upload)
}

func (pc *ProxyClient) writeCMsgAndCheck(header, id string, code int, data interface{}) {
	err := pc.writeCMsg(header, id, code, data)
	if err != nil {
		pc.SetInfoLog(err)
		pc.close()
	}
}

func (pc *ProxyClient) initLinkConn(conn net.Conn, linkType string, rf, wf tool.LinkBoxRWFn) {
	pc.linkConn = conn
	pc.linkType = linkType
	pc.linkSwitch.Store(true)
	lb := tool.NewLinkBox(pc.linkConn, tool.BufferSize, rf, wf)
	pc.linkBox = lb
	go pc.readLinkConn()
}
func (pc *ProxyClient) writeLinkConn() error {
	return pc.linkBox.WriteLinkBoxFromReader(pc.reader)
}
func (pc *ProxyClient) readLinkConn() {
	defer pc.linkConn.Close()
	for {
		err := pc.linkBox.ReadLinkBoxToWriter(pc.rawConn, &pc.writeLock)
		if err != nil {
			pc.SetInfoLog("linkConn:", err)
			return
		}
	}
}

func (pc *ProxyClient) close() {
	pc.closerOnce.Do(func() {
		pc.disable.Store(true)
		pc.rawConn.Close()
		if pc.linkConn != nil {
			pc.linkConn.Close()
		}
		//pc.stop <- 1
		pc.subMapLock.Lock()
		defer pc.subMapLock.Unlock()
		pc.rangeProxySubClient(func(key string, value *ProxyClient) {
			value.close()
			pc.delProxySubClient(key)
		})
		if pc.name != "" {
			pc.ps.delProxyClient(pc.name)
		}

		pc.SetInfoLog("is closed")
	})
}

func (pc *ProxyClient) checkErrAndSend400ErrCMsg(header string, id string, err error, needClose bool) bool {
	if err == nil {
		return false
	}
	pc.writeCMsgAndCheck(header, id, 400, tool.NewErrMsg("bad req : ", err))
	if needClose {
		pc.close()
	}
	pc.SetInfoLog(err)
	return true
}

func (pc *ProxyClient) SetInfoLog(a ...any) {
	loger.SetLogInfo("[", "id:", pc.id, "name:", pc.name, "]", loger.SprintConn(pc.rawConn, a...))
}

func (pc *ProxyClient) delProxySubClient(key string) {
	pc.subMap.Delete(key)
}
func (pc *ProxyClient) getProxySubClient(key string) (*ProxyClient, bool) {
	odj, ok := pc.subMap.Load(key)
	if !ok {
		return nil, false
	}
	sub, ok := odj.(*ProxyClient)
	if !ok {
		return nil, false
	}
	return sub, true
}
func (pc *ProxyClient) setProxySubClient(key string, sub *ProxyClient) {
	pc.subMapLock.Lock()
	defer pc.subMapLock.Unlock()
	if pc.disable.Load() {
		sub.close()
		return
	} else {
		pc.subMap.Store(key, sub)
	}
}
func (pc *ProxyClient) rangeProxySubClient(fn func(key string, value *ProxyClient)) {
	pc.subMap.Range(func(key, value any) bool {
		odj, ok := value.(*ProxyClient)
		if !ok {
			pc.subMap.Delete(key)
		} else {
			fn(key.(string), odj)
		}
		return true
	})
}

func (pc *ProxyClient) getNetworkSpeed() tool.NetworkSpeedView {
	return pc.networkSpeed.ToView()
}
func (pc *ProxyClient) getAllNetworkSpeed() tool.NetworkSpeedView {
	var list []tool.NetworkSpeedView
	pc.rangeProxySubClient(func(key string, value *ProxyClient) {
		list = append(list, value.getAllNetworkSpeed())
	})
	list = append(list, pc.getNetworkSpeed())
	return tool.CountAllNetworkSpeedView(list...)
}
