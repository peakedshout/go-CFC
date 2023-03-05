package client

import (
	"bufio"
	"github.com/peakedshout/go-CFC/loger"
	"github.com/peakedshout/go-CFC/tool"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type SubBox struct {
	id         string
	localName  string
	remoteName string

	key tool.Key

	conn         *net.TCPConn
	root         *DeviceBox
	parent       *SubBox
	networkSpeed tool.NetworkSpeedTicker

	writerChan chan []byte
	stop       chan uint8

	disable atomic.Bool

	subMap     sync.Map
	subMapLock sync.Mutex

	closerOnce sync.Once
}

func (sub *SubBox) FastHandshake(tid string) error {
	sub.SetDeadlineDuration(10 * time.Second)
	for _, one := range sub.root.key.SetMsg(tool.TaskQ, "", 200, tool.OdjSub{
		SrcName: sub.root.name,
		DstKey:  tid,
	}) {
		_, err := sub.Write(one)
		if err != nil {
			return err
		}
	}
	reader := bufio.NewReaderSize(sub, tool.BufferSize)
	cMsg, err := sub.root.key.ReadCMsg(reader, nil, nil)
	if err != nil {
		err = tool.ErrReqBadAny(err)
		sub.SetInfoLog(err)
		return err
	}
	if cMsg.Header != tool.TaskA {
		err = tool.ErrReqBadAny(tool.ErrReqUnexpectedHeader)
		sub.SetInfoLog(err)
		return err
	}
	if cMsg.Code != 200 {
		err = tool.ErrReqBadAny(cMsg.Code, cMsg.Data)
		sub.SetInfoLog(err)
		return err
	}
	if sub.remoteName != "" && sub.remoteName != cMsg.Id {
		err = tool.ErrOpenSubUnexpectedOdj
		sub.SetInfoLog(err)
		return err
	}
	return nil
}

func (sub *SubBox) Read(b []byte) (int, error) {
	if sub.conn == nil {
		return 0, tool.ErrConnIsNil
	}
	if sub.disable.Load() {
		return 0, tool.ErrSubIsDisable
	}
	n, err := sub.conn.Read(b)
	if err != nil {
		return n, err
	}
	sub.networkSpeed.Download.Set(n)
	return n, err
}

func (sub *SubBox) Write(b []byte) (int, error) {
	if sub.conn == nil {
		return 0, tool.ErrConnIsNil
	}
	if sub.disable.Load() {
		return 0, tool.ErrSubIsDisable
	}
	n, err := sub.conn.Write(b)
	if err != nil {
		return n, err
	}
	sub.networkSpeed.Upload.Set(n)
	return n, err
}

func (sub *SubBox) Close() error {
	if sub.conn == nil {
		return tool.ErrConnIsNil
	}
	err := sub.conn.Close()
	//var err error = nil
	sub.SetWarnLog("d42343423423")
	sub.closerOnce.Do(func() {
		sub.stop <- 1
		sub.subMapLock.Lock()
		defer sub.subMapLock.Unlock()
		sub.disable.Store(true)
		var errList []error
		sub.rangeProxySubClient(func(key string, value *SubBox) {
			errList = append(errList, value.Close())
		})
		err = tool.ErrAppend(err, errList...)
		sub.SetInfoLog("is closed")
	})
	return err
}
func (sub *SubBox) LocalAddr() net.Addr {
	if sub.conn == nil {
		return nil
	}
	return sub.conn.LocalAddr()
}
func (sub *SubBox) RemoteAddr() net.Addr {
	if sub.conn == nil {
		return nil
	}
	return sub.conn.RemoteAddr()
}
func (sub *SubBox) SetDeadline(t time.Time) error {
	if sub.conn == nil {
		return tool.ErrConnIsNil
	}
	return sub.conn.SetDeadline(t)
}
func (sub *SubBox) SetReadDeadline(t time.Time) error {
	if sub.conn == nil {
		return tool.ErrConnIsNil
	}
	return sub.conn.SetReadDeadline(t)
}
func (sub *SubBox) SetWriteDeadline(t time.Time) error {
	if sub.conn == nil {
		return tool.ErrConnIsNil
	}
	return sub.conn.SetWriteDeadline(t)
}

func (sub *SubBox) GetLocalName() string {
	return sub.localName
}

func (sub *SubBox) GetRemoteName() string {
	return sub.remoteName
}

func (sub *SubBox) delSubBox(key string) {
	sub.subMap.Delete(key)
}
func (sub *SubBox) getSubBox(key string) (*SubBox, bool) {
	odj, ok := sub.subMap.Load(key)
	if !ok {
		return nil, false
	}
	sub1, ok := odj.(*SubBox)
	if !ok {
		return nil, false
	}
	return sub1, true
}
func (sub *SubBox) setProxySubClient(key string, sub1 *SubBox) {
	sub.subMapLock.Lock()
	defer sub.subMapLock.Unlock()
	if sub.disable.Load() {
		sub.Close()
		return
	} else {
		sub.subMap.Store(key, sub1)
	}
}
func (sub *SubBox) rangeProxySubClient(fn func(key string, value *SubBox)) {
	sub.subMap.Range(func(key, value any) bool {
		odj, ok := value.(*SubBox)
		if !ok {
			sub.subMap.Delete(key)
		} else {
			fn(key.(string), odj)
		}
		return true
	})
}

func (sub *SubBox) SetInfoLog(a ...any) {
	loger.SetLogInfo("[ name:", sub.root.name, "sid:", sub.id, "]", loger.SprintConn(sub.conn, a...))
}

func (sub *SubBox) SetWarnLog(a ...any) {
	loger.SetLogWarn("[ name:", sub.root.name, "sid:", sub.id, "]", loger.SprintConn(sub.conn, a...))
}

func (sub *SubBox) SetDeadlineDuration(timeout time.Duration) bool {
	if sub.conn == nil {
		return false
	}
	if timeout == 0 {
		if sub.SetDeadline(time.Time{}) != nil {
			return true
		}
	} else {
		if sub.SetDeadline(time.Now().Add(timeout)) != nil {
			return true
		}
	}
	return false
}

func (sub *SubBox) NewKey(key string) tool.Key {
	sub.key = tool.NewKey(key)
	return sub.key
}

func (sub *SubBox) GetNetworkSpeedView() tool.NetworkSpeedView {
	return sub.networkSpeed.ToView()
}

func (sub *SubBox) GetAllNetworkSpeedView() tool.NetworkSpeedView {
	var list []tool.NetworkSpeedView
	sub.rangeProxySubClient(func(key string, value *SubBox) {
		list = append(list, value.GetAllNetworkSpeedView())
	})
	list = append(list, sub.GetNetworkSpeedView())
	return tool.CountAllNetworkSpeedView(list...)
}

//func (sub *SubBox) writerCMsg(header, id string, code int, data interface{}) {
//	select {
//	case pc.writeChan <- pc.s.key.SetMsg(header, id, code, data):
//	case <-pc.stop:
//		pc.stop <- 1
//	}
//}
