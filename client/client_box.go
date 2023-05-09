package client

import (
	"context"
	"github.com/peakedshout/go-CFC/loger"
	"github.com/peakedshout/go-CFC/tool"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type DeviceBox struct {
	name      string
	disable   atomic.Bool
	handshake atomic.Bool

	addr *net.TCPAddr

	conn      *net.TCPConn
	writeLock sync.Mutex

	stop         chan uint8
	ping         tool.Ping
	networkSpeed tool.NetworkSpeedTicker

	key tool.Key

	taskCbCtx *tool.TaskCbContext

	subMap     sync.Map
	subMapLock sync.Mutex

	ListenLock    sync.Mutex
	isListen      atomic.Bool
	subListen     chan *SubBox
	subListenStop chan error

	closerOnce sync.Once
}

func (box *DeviceBox) listenCMsg() {
	box.taskCbCtx.SetNoCb(box.cMsgHandler)
	go func() {
		defer box.Close()
		err := box.taskCbCtx.ReadCMsg()
		if err != nil {
			box.SetWarnLog(err)
		}
	}()
}
func (box *DeviceBox) cMsgHandler(cMsg tool.ConnMsg) {
	box.SetInfoLog(cMsg)
	switch cMsg.Header {
	case tool.SOpenA:
		box.listenSub(cMsg)
	}
}

func (box *DeviceBox) listenSub(cMsg tool.ConnMsg) {
	if cMsg.Id != "" {
		return
	}
	if box.subListen == nil || box.subListenStop == nil {
		return
	}
	var info tool.OdjSubOpenResp
	err := cMsg.Unmarshal(&info)
	if err != nil {
		box.SetWarnLog(err)
		return
	}

	var conn *net.TCPConn
	switch info.Type {
	case tool.SubOpenTypeTCPP2P:

		rconn, err := newDialer(nil, 0).Dial("tcp", box.addr.String())
		if err != nil {
			box.SetWarnLog(err)
			return
		}
		conn = rconn.(*net.TCPConn)
		defer conn.Close()
	case tool.SubOpenTypeDefault:
		conn, err = net.DialTCP("tcp", nil, box.addr)
		if err != nil {
			box.SetWarnLog(err)
			return
		}
	default:
		box.SetWarnLog(tool.ErrUnexpectedSubOpenType)
		return
	}
	sub := &SubBox{
		id:           tool.NewId(1),
		addr:         nil,
		key:          box.key,
		conn:         conn,
		root:         box,
		parent:       nil,
		networkSpeed: tool.NewNetworkSpeedTicker(),
		writerLock:   sync.Mutex{},
		stop:         make(chan uint8, 1),
		disable:      atomic.Bool{},
		subMap:       sync.Map{},
		subMapLock:   sync.Mutex{},
		closerOnce:   sync.Once{},
	}
	err = sub.fastHandshake(info.Tid)
	if err != nil {
		err = tool.ErrOpenSubBoxBadAny(err)
		sub.Close()
		box.SetWarnLog(err)
		return
	}
	sub.SetDeadlineDuration(0)

	switch info.Type {
	case tool.SubOpenTypeTCPP2P:
		ln, err := newListenConfig().Listen(context.Background(), conn.LocalAddr().Network(), conn.LocalAddr().String())
		if err != nil {
			sub.Close()
			loger.SetLogMust(err)
			return
		}
		time.Sleep(2 * time.Second)
		//ln.Close()
		var lconn *net.TCPConn
		for i := 0; i < 3; i++ {
			//fmt.Println("wdwad", sub.GetRemotePublicAddr().Network(), sub.GetRemotePublicAddr().String())
			pconn, err := newDialer(conn.LocalAddr(), 3*time.Second).Dial(sub.GetRemotePublicAddr().Network(), sub.GetRemotePublicAddr().String())
			if err != nil {
				loger.SetLogMust(err)
				time.Sleep(500 * time.Millisecond)
				continue
			}
			lconn = pconn.(*net.TCPConn)
			break
		}
		if lconn == nil {
			box.SetWarnLog(tool.ErrOpenSubBoxBadAny("p2p open bad"))
			return
		}
		sub.conn = lconn
		ln.Close()
	case tool.SubOpenTypeDefault:
	default:
		box.SetWarnLog(tool.ErrUnexpectedSubOpenType)
		return
	}

	select {
	case box.subListen <- sub:
		box.setSubBox(sub.id, sub)
	case err := <-box.subListenStop:
		box.subListenStop <- err
		sub.Close()
	}
}

func (box *DeviceBox) handshakeCheck() error {
	err := box.taskCbCtx.NewTaskCbCMsg(tool.HandshakeCheckStepQ1, 200, nil).WaitCb(10*time.Second, func(cMsg tool.ConnMsg) error {
		if cMsg.Header != tool.HandshakeCheckStepA1 || cMsg.Code != 200 {
			return tool.ErrHandshakeIsDad
		}
		return nil
	})
	if err != nil {
		box.SetWarnLog(err)
		return err
	}
	err = box.taskCbCtx.NewTaskCbCMsg(tool.HandshakeCheckStepQ2, 200, tool.OdjClientInfo{Name: box.name}).WaitCb(10*time.Second, func(cMsg tool.ConnMsg) error {
		if cMsg.Header != tool.HandshakeCheckStepA2 || cMsg.Code != 200 {
			return tool.ErrHandshakeIsDad
		}
		return nil
	})
	if err != nil {
		box.SetWarnLog(err)
		return err
	}
	box.handshake.Store(true)
	return nil
}

func (box *DeviceBox) asyncWaitSendAndPing() {
	go func() {
		defer box.Close()
		var t time.Time
		tk := time.NewTicker(5 * time.Second)
		defer tk.Stop()
		for {
			select {
			case <-box.stop:
				box.stop <- 1
				return
			case <-tk.C:
				if !box.handshake.Load() {
					continue
				}
				t = time.Now()
				box.taskCbCtx.NewTaskCbCMsg(tool.PingMsg, 200, box.ping).NowaitCb(func(cMsg tool.ConnMsg) error {
					if cMsg.Header == tool.PongMsg && cMsg.Code == 200 {
						box.ping.Ping = time.Now().Sub(t)
					}
					return nil
				})
			}
		}
	}()
}

func (box *DeviceBox) Write(b []byte) (int, error) {
	box.writeLock.Lock()
	defer box.writeLock.Unlock()
	if box.disable.Load() {
		return 0, tool.ErrIsDisable
	}
	n, err := box.conn.Write(b)
	if err != nil {
		return n, err
	}
	box.networkSpeed.Upload.Set(n)
	return n, nil
}
func (box *DeviceBox) Read(b []byte) (int, error) {
	if box.disable.Load() {
		return 0, tool.ErrIsDisable
	}
	n, err := box.conn.Read(b)
	if err != nil {
		return n, err
	}
	box.networkSpeed.Download.Set(n)
	return n, nil
}

func (box *DeviceBox) SetDeadline(timeout time.Duration) bool {
	if box.conn == nil {
		return false
	}
	if timeout == 0 {
		if box.conn.SetDeadline(time.Time{}) != nil {
			return true
		}
	} else {
		if box.conn.SetDeadline(time.Now().Add(timeout)) != nil {
			return true
		}
	}
	return false
}

func (box *DeviceBox) SetInfoLog(a ...any) {
	loger.SetLogInfo("[ name:", box.name, "]", loger.SprintConn(box.conn, a...))
}

func (box *DeviceBox) SetWarnLog(a ...any) {
	loger.SetLogWarn("[ name:", box.name, "]", loger.SprintConn(box.conn, a...))
}

func (box *DeviceBox) delSubBox(key string) {
	box.subMap.Delete(key)
}
func (box *DeviceBox) getSubBox(key string) (*SubBox, bool) {
	odj, ok := box.subMap.Load(key)
	if !ok {
		return nil, false
	}
	sub, ok := odj.(*SubBox)
	if !ok {
		return nil, false
	}
	return sub, true
}
func (box *DeviceBox) setSubBox(key string, sub *SubBox) {
	box.subMapLock.Lock()
	defer box.subMapLock.Unlock()
	if box.disable.Load() {
		sub.Close()
		return
	} else {
		box.subMap.Store(key, sub)
	}
}
func (box *DeviceBox) rangeProxySubClient(fn func(key string, value *SubBox)) {
	box.subMap.Range(func(key, value any) bool {
		odj, ok := value.(*SubBox)
		if !ok {
			box.subMap.Delete(key)
		} else {
			fn(key.(string), odj)
		}
		return true
	})
}

func (box *DeviceBox) GetNetworkSpeedView() tool.NetworkSpeedView {
	return box.networkSpeed.ToView()
}
func (box *DeviceBox) GetAllNetworkSpeedView() tool.NetworkSpeedView {
	var list []tool.NetworkSpeedView
	box.rangeProxySubClient(func(key string, value *SubBox) {
		list = append(list, value.GetAllNetworkSpeedView())
	})
	list = append(list, box.GetNetworkSpeedView())
	return tool.CountAllNetworkSpeedView(list...)
}
