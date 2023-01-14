package client

import (
	"bufio"
	"errors"
	"github.com/peakedshout/go-CFC/tool"
	"log"
	"net"
	"sync"
	"time"
)

type ClientContext struct {
	name string

	ip   string
	port string

	conn      net.Conn
	writeChan chan [][]byte
	stop      chan uint8
	ping      tool.Ping

	key tool.Key

	taskMap       sync.Map
	subMap        sync.Map
	subListen     chan *SubConnContext
	subListenStop chan uint8
}

func newClient(name, ip, port, key string) *ClientContext {
	if name == "" {
		panic("name is nil")
	}
	if len(key) != 32 {
		panic("key is not 32byte")
	}
	c := &ClientContext{
		name:      name,
		ip:        ip,
		port:      port,
		conn:      nil,
		writeChan: make(chan [][]byte, 100),
		stop:      make(chan uint8, 10),
		ping:      tool.Ping{},
		key:       tool.NewKey(key),
		taskMap:   sync.Map{},
		subMap:    sync.Map{},
	}
	return c
}
func LinkLongConn(name, ip, port, key string) (*ClientContext, error) {
	if name == "" {
		return nil, errors.New("name is nil")
	}
	c := newClient(name, ip, port, key)
	conn, err := net.Dial("tcp", ip+port)
	if err != nil {
		return nil, err
	}
	c.conn = conn
	_, err = c.sendAndWait(c.key.SetMsg(tool.HandshakeCheckStepQ1, "", 200, nil), tool.HandshakeCheckStepA1, 200)
	if err != nil {
		return nil, err
	}
	_, err = c.sendAndWait(c.key.SetMsg(tool.HandshakeCheckStepQ2, "", 200, tool.OdjClientInfo{Name: c.name}), tool.HandshakeCheckStepA2, 200)
	if err != nil {
		return nil, err
	}
	go func() {
		var t0 time.Time
		go func() {
			t := time.NewTicker(5 * time.Second)
			defer t.Stop()
			for {
				select {
				case b := <-c.writeChan:
					for _, one := range b {
						_, err := conn.Write(one)
						if err != nil {
							tool.Println(conn, err)
							return
						}
					}
				case <-c.stop:
					c.conn.Close()
					c.subMap.Range(func(key, value any) bool {
						value.(*SubConnContext).Close()
						return true
					})
				case <-t.C:
					t0 = time.Now()
					c.writeChan <- c.key.SetMsg(tool.PingMsg, "", 200, c.ping)
				}
			}
		}()
		reader := bufio.NewReader(conn)
		for {
			cMsg, err := c.key.GetMsg(reader)
			if err != nil {
				tool.Println(conn, err)
				c.Close()
				return
			}
			c.cMsgHandler(cMsg, t0)
		}
	}()
	return c, nil
}
func (c *ClientContext) cMsgHandler(msg tool.ConnMsg, tp time.Time) {
	switch msg.Header {
	case tool.PongMsg:
		c.ping.Ping = time.Now().Sub(tp)
	case tool.SOpenA:
		if msg.Id != "" {
			fn, ok := c.taskMap.Load(msg.Id)
			if !ok {
				return
			}
			fn.(func(cMsg tool.ConnMsg))(msg)
		} else {
			var info tool.OdjMsg
			err := tool.UnmarshalV2(msg.Data, &info)
			if err != nil {
				return
			}
			conn, err := net.Dial("tcp", c.ip+c.port)
			if err != nil {
				tool.Println(conn, err)
				return
			}
			for _, one := range c.key.SetMsg(tool.TaskQ, "", 200, tool.OdjSub{
				SrcName: c.name,
				DstKey:  info.Msg,
			}) {
				_, err = conn.Write(one)
				if err != nil {
					tool.Println(conn, err)
					return
				}
			}
			reader := bufio.NewReader(conn)
			cMsg, err := c.key.GetMsg(reader)
			if cMsg.Header != tool.TaskA {
				tool.Println(conn, "join is bad")
				conn.Close()
				return
			}
			if cMsg.Code != 200 {
				tool.Println(conn, "task is bad")
				conn.Close()
				return
			}
			s := &SubConnContext{
				id:         tool.NewId(1),
				localName:  c.name,
				remoteName: cMsg.Id,
				conn:       conn,
				f:          c,
				key:        c.key,
			}
			c.subMap.Store(s.id, s)
			//time.Sleep(500 * time.Millisecond)
			c.subListen <- s
		}
	case tool.DelayA:
		if msg.Id != "" {
			fn, ok := c.taskMap.Load(msg.Id)
			if !ok {
				return
			}
			fn.(func(cMsg tool.ConnMsg))(msg)
		}
	}
}

func (c *ClientContext) sendAndWait(b [][]byte, header string, code int) (cMsg tool.ConnMsg, err error) {
	for _, one := range b {
		_, err = c.conn.Write(one)
		if err != nil {
			return
		}
	}
	reader := bufio.NewReader(c.conn)
	cMsg, err = c.key.GetMsg(reader)
	if cMsg.Header != header || cMsg.Code != code {
		err = errors.New("handshake is bad")
	}
	return
}
func (c *ClientContext) Close() {
	c.stop <- 1
	if c.subListenStop != nil {
		c.subListenStop <- 1
	}
}

func (c *ClientContext) GetSubConn(name string) (*SubConnContext, error) {
	tid := tool.NewId(1)
	tk := time.NewTicker(100 * time.Second)
	defer tk.Stop()

	run := make(chan string)
	stop := make(chan error)
	c.taskMap.Store(tid, func(cMsg tool.ConnMsg) {
		c.taskMap.Delete(tid)
		var info tool.OdjMsg
		err := tool.UnmarshalV2(cMsg.Data, &info)
		if err != nil || info.Msg == "" {
			stop <- errors.New("bad")
			return
		}
		if cMsg.Code != 200 {
			stop <- errors.New("bad:" + info.Msg)
			return
		}
		run <- info.Msg
	})
	c.writeChan <- c.key.SetMsg(tool.SOpenQ, tid, 200, tool.OdjMsg{Msg: name})
	select {
	case <-tk.C:
		return nil, errors.New("timeout")
	case err := <-stop:
		return nil, err
	case k := <-run:
		conn, err := net.Dial("tcp", c.ip+c.port)
		if err != nil {
			return nil, err
		}
		for _, one := range c.key.SetMsg(tool.TaskQ, "", 200, tool.OdjSub{
			SrcName: c.name,
			DstKey:  k,
		}) {
			_, err = conn.Write(one)
			if err != nil {
				conn.Close()
				return nil, err
			}
		}
		reader := bufio.NewReader(conn)
		cMsg, err := c.key.GetMsg(reader)
		if cMsg.Header != tool.TaskA {
			conn.Close()
			return nil, errors.New("join is bad")
		}
		if cMsg.Code != 200 {
			conn.Close()
			return nil, errors.New("task is bad")
		}
		if name != cMsg.Id {
			conn.Close()
			return nil, errors.New("odj is bad")
		}
		s := &SubConnContext{
			id:         tool.NewId(1),
			localName:  c.name,
			remoteName: cMsg.Id,
			conn:       conn,
			f:          c,
			key:        c.key,
		}
		c.subMap.Store(s.id, s)
		//time.Sleep(500 * time.Millisecond)
		return s, nil
	}
	//return nil, errors.New("err")
}

func (c *ClientContext) ListenSubConn(fn func(sub *SubConnContext)) error {
	return c.listenSubConn(fn, 0)
}

func (c *ClientContext) ListenSubConnWriter(fn func(sub *SubConnContext)) error {
	return c.listenSubConn(fn, 1)
}
func (c *ClientContext) ListenSubConnReadWriter(fn func(sub *SubConnContext)) error {
	return c.listenSubConn(fn, 2)
}

func (c *ClientContext) listenSubConn(fn func(sub *SubConnContext), rw uint8) error {
	if c.subListen != nil || c.subListenStop != nil {
		return errors.New("only one client can listen")
	}
	c.subListen = make(chan *SubConnContext, 100)
	c.subListenStop = make(chan uint8, 10)

	log.Println("~~~ Start Listening SubConn ~~~")
	for {
		select {
		case scc := <-c.subListen:
			if rw == 1 || rw == 2 {
				scc.writerChan = make(chan []byte, 100)
				scc.writerStop = make(chan uint8, 10)
				go func() {
					for {
						select {
						case b := <-scc.writerChan:
							_, err := scc.conn.Write(b)
							if err != nil {
								tool.Println(scc.conn, err)
								return
							}
						case <-scc.writerStop:
							return
						}
					}
				}()
			}
			if rw == 2 {
				scc.reader = bufio.NewReaderSize(scc.conn, tool.BufferSize)
			}
			go fn(scc)
		case <-c.subListenStop:
			return errors.New("the cfc-hook-server connection is disconnected")
		}
	}
}

type SubConnContext struct {
	id         string
	localName  string
	remoteName string
	conn       net.Conn
	f          *ClientContext
	writerChan chan []byte
	writerStop chan uint8
	reader     *bufio.Reader
	key        tool.Key
}

func (s *SubConnContext) GetLocalName() string {
	return s.localName
}

func (s *SubConnContext) GetRemoteName() string {
	return s.remoteName
}

func (s *SubConnContext) GetConn() net.Conn {
	return s.conn
}

func (s *SubConnContext) Close() {
	s.conn.Close()
	s.f.subMap.Delete(s.id)
	if s.writerStop != nil {
		s.writerStop <- 1
	}
}

// Write,
// This is the case where conn writes to the data race, it will be written in FIFO order. If you are using ListenSubConn/GetSubConn, please do not use this method. It does not support (and does not create objects)
func (s *SubConnContext) Write(b []byte) {
	s.writerChan <- b
}

// Read,
// This is according to the CFC custom protocol to read content, if you want to use this protocol communication, then you write content should also follow the protocol, the callback return false will stop reading, If you don't have ListenSubConnReadWriter/NewSubConnReader please don't use it (because did not create objects)
func (s *SubConnContext) Read(fn func(cMsg tool.ConnMsg) bool) error {
	for {
		cMsg, err := s.key.GetMsg(s.reader)
		if err != nil {
			return err
		}
		if !fn(cMsg) {
			return nil
		}
	}
}

// NewSubConnReader new reader
func (s *SubConnContext) NewSubConnReader() {
	s.reader = bufio.NewReaderSize(s.conn, tool.BufferSize)
}

func (s *SubConnContext) NewKey(key string) tool.Key {
	s.key = tool.NewKey(key)
	return s.key
}

func (c *ClientContext) sendAndCallBack(timeout time.Duration, send tool.ConnMsg, fn func(cMsg tool.ConnMsg)) (timeOut <-chan time.Time) {
	tid := tool.NewId(1)
	tk := time.NewTicker(timeout * time.Second)
	defer tk.Stop()
	//run := make(chan any)
	//stop := make(chan error)
	c.taskMap.Store(tid, func(cMsg tool.ConnMsg) {
		c.taskMap.Delete(tid)
		fn(cMsg)
	})
	send.Id = tid
	c.writeChan <- c.key.Encode(send)
	return tk.C
}

func (c *ClientContext) GetOtherDelayPing(name ...string) ([]tool.OdjPing, error) {
	stop := make(chan error)
	run := make(chan []tool.OdjPing)
	tk := c.sendAndCallBack(10, tool.ConnMsg{
		Header: tool.DelayQ,
		Code:   200,
		Data:   tool.OdjIdList{IdList: name},
	}, func(cMsg tool.ConnMsg) {
		if cMsg.Code != 200 {
			var info tool.OdjMsg
			err := tool.UnmarshalV2(cMsg.Data, &info)
			if err != nil {
				stop <- err
				return
			}
			stop <- errors.New("bad:" + info.Msg)
			return
		}
		var info []tool.OdjPing
		err := tool.UnmarshalV2(cMsg.Data, &info)
		if err != nil {
			stop <- err
			return
		}
		run <- info
	})
	select {
	case <-tk:
		return nil, errors.New("timeout")
	case err := <-stop:
		return nil, err
	case k := <-run:
		return k, nil
	}
}
