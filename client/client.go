package client

import (
	"github.com/peakedshout/go-CFC/loger"
	"github.com/peakedshout/go-CFC/tool"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

func newBox(name, addr, key string) *DeviceBox {
	if name == "" {
		loger.SetLogError(tool.ErrNameIsNil)
	}
	if len(key) != 32 {
		loger.SetLogError(tool.ErrKeyIsNot32Bytes)
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		loger.SetLogError("ResolveTCPAddr :", err)
	}
	box := &DeviceBox{
		name:          name,
		addr:          tcpAddr,
		conn:          nil,
		writeLock:     sync.Mutex{},
		stop:          make(chan uint8, 1),
		ping:          tool.Ping{},
		networkSpeed:  tool.NewNetworkSpeedTicker(),
		key:           tool.NewKey(key),
		taskCbCtx:     nil,
		subMap:        sync.Map{},
		subListen:     nil,
		subListenStop: nil,
		closerOnce:    sync.Once{},
	}
	return box
}
func LinkProxyServer(name, addr, key string) (*DeviceBox, error) {
	box := newBox(name, addr, key)
	conn, err := net.DialTCP("tcp", nil, box.addr)
	if err != nil {
		loger.SetLogWarn(err)
		return nil, err
	}
	box.conn = conn
	box.taskCbCtx = tool.NewTaskContext(box, box.key)
	box.listenCMsg()
	box.asyncWaitSendAndPing()
	err = box.handshakeCheck()
	if err != nil {
		box.Close()
		return nil, err
	}
	loger.SetLogInfo(loger.SprintColor(5, 37, 37, "~~~ Start Proxy Box ~~~"))
	return box, nil
}

func (box *DeviceBox) GetSubBox(name string) (*SubBox, error) {
	var info tool.OdjSubOpenResp
	err := box.taskCbCtx.NewTaskCbCMsg(tool.SOpenQ, 200, tool.OdjSubOpenReq{
		Type:    tool.SubOpenTypeDefault,
		OdjName: name,
	}).WaitCb(10*time.Second, func(cMsg tool.ConnMsg) error {
		err1 := cMsg.CheckConnMsgHeaderAndCode(tool.SOpenA, 200)
		if err1 != nil {
			box.SetInfoLog(err1)
			return err1
		}
		err1 = cMsg.Unmarshal(&info)
		if err1 != nil {
			box.SetInfoLog(err1)
			return err1
		}
		return nil
	})
	if err != nil {
		box.SetWarnLog(err)
		return nil, err
	}
	conn, err := net.DialTCP("tcp", nil, box.addr)
	if err != nil {
		box.SetWarnLog(err)
		return nil, err
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
		return nil, err
	}
	sub.SetDeadlineDuration(0)
	box.setSubBox(sub.id, sub)
	return sub, nil
}

func (box *DeviceBox) ListenSubBox(fn func(sub *SubBox)) error {
	if box.subListen != nil || box.subListenStop != nil {
		err := tool.ErrBoxComplexListen
		loger.SetLogError(err)
	}
	box.subListen = make(chan *SubBox, 100)
	box.subListenStop = make(chan error, 1)

	//box.SetInfoLog("~~~ Start Listening SubBox ~~~")
	loger.SetLogMust(loger.SprintColor(5, 37, 37, "~~~ Start Listening SubBox ~~~"))
	for {
		select {
		case sub := <-box.subListen:
			go fn(sub)
		case err := <-box.subListenStop:
			err = tool.ErrAppend(tool.ErrBoxStopListen, err)
			box.SetWarnLog(err)
			return err
		}
	}
}

func (box *DeviceBox) GetOtherDelayPing(name ...string) ([]tool.OdjPing, error) {
	var resp []tool.OdjPing
	err := box.taskCbCtx.NewTaskCbCMsg(tool.DelayQ, 200, tool.OdjIdList{IdList: name}).WaitCb(10*time.Second, func(cMsg tool.ConnMsg) error {
		if cMsg.Header != tool.DelayA {
			err := tool.ErrReqBadAny(tool.ErrReqUnexpectedHeader)
			box.SetInfoLog(err)
			return err
		}
		if cMsg.Code != 200 {
			err := tool.ErrReqBadAny(cMsg.Code, cMsg.Data)
			box.SetInfoLog(err)
			return err
		}
		var info []tool.OdjPing
		err := cMsg.Unmarshal(&info)
		if err != nil {
			box.SetInfoLog(err)
			return err
		}
		resp = info
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (box *DeviceBox) Close() {
	box.closerOnce.Do(func() {
		box.conn.Close()
		box.stop <- 1
		box.subMapLock.Lock()
		defer box.subMapLock.Unlock()
		box.disable.Store(true)
		if box.subListenStop != nil {
			box.subListenStop <- tool.ErrBoxIsClosed
		}
		box.rangeProxySubClient(func(key string, value *SubBox) {
			value.Close()
		})
		loger.SetLogMust(loger.SprintColor(5, 37, 37, "~~~ Closed Proxy Box ~~~"))
	})
}

func (box *DeviceBox) Wait() {
	<-box.stop
	loger.SetLogMust(loger.SprintColor(5, 37, 37, "~~~ EndWait Proxy Box ~~~"))
	box.stop <- 1
}
