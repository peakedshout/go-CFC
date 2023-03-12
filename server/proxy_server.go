package server

import (
	"github.com/peakedshout/go-CFC/tool"
	"net"
	"sync"
)

type ProxyServer struct {
	addr *net.TCPAddr
	ln   *net.TCPListener

	stop chan uint8

	key tool.Key

	//clientMap     map[string]*ProxyClient
	//clientMapLock sync.Mutex

	proxyClientMap     sync.Map //map[string]*ProxyClient
	proxyClientMapLock sync.Mutex

	//taskRoomMap     map[string]*proxyTaskRoom
	//taskRoomMapLock sync.Mutex

	proxyTaskRoomMap sync.Map //map[string]*proxyTaskRoom
}

func (ps *ProxyServer) delProxyClient(key string) {
	ps.proxyClientMap.Delete(key)
}
func (ps *ProxyServer) getProxyClient(key string) (*ProxyClient, bool) {
	if key == "" {
		return nil, false
	}
	odj, ok := ps.proxyClientMap.Load(key)
	if !ok {
		return nil, false
	}
	pc, ok := odj.(*ProxyClient)
	if !ok {
		return nil, false
	}
	return pc, true
}
func (ps *ProxyServer) setProxyClient(key string, pc *ProxyClient) {
	ps.proxyClientMapLock.Lock()
	defer ps.proxyClientMapLock.Unlock()
	obj, ok := ps.getProxyClient(key)
	if ok {
		obj.close()
	}
	ps.proxyClientMap.Store(key, pc)
}
func (ps *ProxyServer) rangeProxyClient(fn func(key string, value *ProxyClient)) {
	ps.proxyClientMap.Range(func(key, value any) bool {
		odj, ok := value.(*ProxyClient)
		if !ok {
			ps.proxyClientMap.Delete(key)
		} else {
			fn(key.(string), odj)
		}
		return true
	})
}

func (ps *ProxyServer) getProxyClientDelay(key ...string) (resp []tool.OdjPing) {
	if len(key) == 0 {
		ps.rangeProxyClient(func(key string, value *ProxyClient) {
			resp = append(resp, tool.OdjPing{
				Name:   value.name,
				Ping:   value.ping,
				Active: true,
			})
		})
	} else {
		for _, one := range key {
			pc, ok := ps.getProxyClient(one)
			if ok {
				resp = append(resp, tool.OdjPing{
					Name:   pc.name,
					Ping:   pc.ping,
					Active: true,
				})
			} else {
				resp = append(resp, tool.OdjPing{
					Name:   one,
					Ping:   tool.Ping{},
					Active: false,
				})
			}
		}
	}
	return resp
}

func (ps *ProxyServer) getProxyClientNetworkSpeed(key ...string) (resp []tool.NetworkSpeedView) {
	if len(key) == 0 {
		ps.rangeProxyClient(func(key string, value *ProxyClient) {
			resp = append(resp, value.getAllNetworkSpeed())
		})
	} else {
		for _, one := range key {
			odj, ok := ps.getProxyClient(one)
			if !ok {
				resp = append(resp, tool.CountAllNetworkSpeedView())
			} else {
				resp = append(resp, odj.getAllNetworkSpeed())
			}
		}
	}
	return resp
}

func (ps *ProxyServer) getAllProxyClientNetworkSpeed() tool.NetworkSpeedView {
	var list []tool.NetworkSpeedView
	ps.rangeProxyClient(func(key string, value *ProxyClient) {
		list = append(list, value.getAllNetworkSpeed())
	})
	return tool.CountAllNetworkSpeedView(list...)
}
