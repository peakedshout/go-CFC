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
	id string

	subType SubType

	key tool.Key

	conn         net.Conn
	root         *DeviceBox
	parent       *SubBox
	networkSpeed tool.NetworkSpeedTicker
	addr         *tool.SubInfo

	writerLock sync.Mutex

	disable atomic.Bool

	subMap     sync.Map
	subMapLock sync.Mutex

	closerOnce sync.Once
}

func (sub *SubBox) fastHandshake(tid string) error {
	sub.SetDeadlineDuration(10 * time.Second)
	tc := tool.NewTaskContext(sub, sub.root.key)

	go tc.ReadCMsg()
	defer tc.Close()
	id := tool.NewId(1)
	err := tc.NewTaskCbCMsgNeedId(tool.TaskQ, id, 200, tool.OdjSubReq{
		Id:      id,
		SrcName: sub.root.name,
		DstKey:  tid,
		Addr:    tool.MustResolveTCPAddr(sub.conn.LocalAddr()),
	}).WaitCb(10*time.Second, func(cMsg tool.ConnMsg) error {
		defer tc.Close()
		err1 := cMsg.CheckConnMsgHeaderAndCode(tool.TaskA, 200)
		if err1 != nil {
			sub.SetInfoLog(err1)
			return err1
		}
		var info tool.SubInfo
		err1 = cMsg.Unmarshal(&info)
		if err1 != nil {
			sub.SetInfoLog(err1)
			return err1
		}
		if info.LocalName == "" || info.RemoteName == "" ||
			info.LocalIntranetAddr == nil || info.RemoteIntranetAddr == nil ||
			info.LocalPublicAddr == nil || info.RemotePublicAddr == nil {
			err1 = tool.ErrOpenSubUnexpectedOdj
			sub.SetInfoLog(err1)
			return err1
		}
		sub.addr = &info
		return nil
	})
	return err
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
	sub.closerOnce.Do(func() {
		sub.subMapLock.Lock()
		defer sub.subMapLock.Unlock()
		sub.disable.Store(true)
		var errList []error
		sub.rangeProxySubClient(func(key string, value *SubBox) {
			errList = append(errList, value.Close())
		})
		err = tool.ErrAppend(err, errList...)
		sub.root.delSubBox(sub.id)
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
	if sub.addr == nil {
		return ""
	}
	return sub.addr.LocalName
}

func (sub *SubBox) GetRemoteName() string {
	if sub.addr == nil {
		return ""
	}
	return sub.addr.RemoteName
}

func (sub *SubBox) GetLocalIntranetAddr() net.Addr {
	if sub.addr == nil {
		return nil
	}
	if sub.addr.LocalIntranetAddr != nil {
		return sub.addr.LocalIntranetAddr
	}
	if sub.addr.ULocalIntranetAddr != nil {
		return sub.addr.ULocalIntranetAddr
	}
	return nil
}
func (sub *SubBox) GetRemoteIntranetAddr() net.Addr {
	if sub.addr == nil {
		return nil
	}
	if sub.addr.RemoteIntranetAddr != nil {
		return sub.addr.RemoteIntranetAddr
	}
	if sub.addr.URemoteIntranetAddr != nil {
		return sub.addr.URemoteIntranetAddr
	}
	return nil
}
func (sub *SubBox) GetLocalPublicAddr() net.Addr {
	if sub.addr == nil {
		return nil
	}
	if sub.addr.LocalPublicAddr != nil {
		return sub.addr.LocalPublicAddr
	}
	if sub.addr.ULocalPublicAddr != nil {
		return sub.addr.ULocalPublicAddr
	}
	return nil
}
func (sub *SubBox) GetRemotePublicAddr() net.Addr {
	if sub.addr == nil {
		return nil
	}
	if sub.addr.RemotePublicAddr != nil {
		return sub.addr.RemotePublicAddr
	}
	if sub.addr.URemotePublicAddr != nil {
		return sub.addr.URemotePublicAddr
	}
	return nil
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

func (sub *SubBox) SetDebugLog(a ...any) {
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

func (sub *SubBox) ReadCMsgCb(fn func(cMsg tool.ConnMsg) (bool, error)) error {
	reader := bufio.NewReaderSize(sub, tool.BufferSize)
	for {
		cMsg, err := sub.key.ReadCMsg(reader, nil, nil)
		if err != nil {
			return err
		}
		r, err := fn(cMsg)
		if err != nil {
			return err
		}
		if !r {
			return nil
		}
	}
}

func (sub *SubBox) WriteCMsg(header string, id string, code int, data interface{}) error {
	sub.writerLock.Lock()
	defer sub.writerLock.Unlock()
	for _, one := range sub.key.SetMsg(header, id, code, data) {
		_, err := sub.Write(one)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sub *SubBox) WriteQueueBytes(b [][]byte) error {
	sub.writerLock.Lock()
	defer sub.writerLock.Unlock()
	for _, one := range b {
		_, err := sub.Write(one)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sub *SubBox) Type() string {
	return sub.subType.String()
}

func (sub *SubBox) Id() string {
	return sub.id
}

func (sub *SubBox) GetRawKey() string {
	return sub.key.GetRawKey()
}
