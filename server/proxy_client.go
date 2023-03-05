package server

import (
	"github.com/peakedshout/go-CFC/loger"
	"github.com/peakedshout/go-CFC/tool"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type ProxyClient struct {
	s       *ProxyServer
	id      string
	name    string
	disable atomic.Bool

	fastOdj     *ProxyClient
	fastConn    atomic.Bool
	fastOdjChan chan [][]byte
	step        *int64
	stop        chan uint8
	closerOnce  sync.Once

	conn         *net.TCPConn
	writeChan    chan [][]byte
	ping         tool.Ping
	networkSpeed tool.NetworkSpeedTicker

	parent *ProxyClient

	//subMap     map[string]*ProxyClient
	subMap     sync.Map
	subMapLock sync.Mutex
}

func (pc *ProxyClient) close() {
	pc.closerOnce.Do(func() {
		pc.disable.Store(true)
		pc.conn.Close()
		pc.stop <- 1
		pc.subMapLock.Lock()
		defer pc.subMapLock.Unlock()
		pc.rangeProxySubClient(func(key string, value *ProxyClient) {
			value.close()
			pc.delProxySubClient(key)
		})
		if pc.name != "" {
			pc.s.delProxyClient(pc.name)
		}
		pc.s.delProxyClient(pc.name)
		pc.SetInfoLog("is closed")
	})
}

func (pc *ProxyClient) SetDeadline(timeout time.Duration) bool {
	if pc.conn == nil {
		return false
	}
	if timeout == 0 {
		if pc.conn.SetDeadline(time.Time{}) != nil {
			return true
		}
	} else {
		if pc.conn.SetDeadline(time.Now().Add(timeout)) != nil {
			return true
		}
	}
	return false
}
func (pc *ProxyClient) writerCMsg(header, id string, code int, data interface{}) {
	select {
	case pc.writeChan <- pc.s.key.SetMsg(header, id, code, data):
	case <-pc.stop:
		pc.stop <- 1
	}
}
func (pc *ProxyClient) SetInfoLog(a ...any) {
	loger.SetLogInfo("[", "id:", pc.id, "name:", pc.name, "]", loger.SprintConn(pc.conn, a...))
}

func (pc *ProxyClient) writerFast(b []byte) {
	select {
	case pc.fastOdjChan <- [][]byte{b}:
	case <-pc.stop:
		pc.stop <- 1
	}
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
