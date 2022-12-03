package client

import (
	"bufio"
	"errors"
	"github.com/peakedshout/go-CFC/tool"
	"net"
	"sync"
	"time"
)

type ClientContext struct {
	name string

	ip   string
	port string

	conn      net.Conn
	writeChan chan []byte
	stop      chan uint8
	ping      tool.Ping

	key tool.Key

	taskMap   sync.Map
	subMap    sync.Map
	subListen chan *SubConnContext
}

func newClient(name, ip, port, key string) *ClientContext {
	if len(key) != 32 {
		panic("key is not 32byte")
	}
	c := &ClientContext{
		name:      name,
		ip:        ip,
		port:      port,
		conn:      nil,
		writeChan: make(chan []byte, 100),
		stop:      make(chan uint8, 10),
		ping:      tool.Ping{},
		key:       tool.NewKey(key),
		taskMap:   sync.Map{},
		subMap:    sync.Map{},
		subListen: make(chan *SubConnContext, 100),
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
			for {
				select {
				case b := <-c.writeChan:
					_, err := conn.Write(b)
					if err != nil {
						tool.Println(conn, err)
						return
					}
				case <-c.stop:
					c.conn.Close()
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
			_, err = conn.Write(c.key.SetMsg(tool.TaskQ, "", 200, tool.OdjSub{
				SrcName: c.name,
				DstKey:  info.Msg,
			}))
			if err != nil {
				tool.Println(conn, err)
				return
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
				id:   tool.NewId(1),
				conn: conn,
				f:    c,
			}
			c.subMap.Store(s.id, s)
			time.Sleep(500 * time.Millisecond)
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

func (c *ClientContext) sendAndWait(b []byte, header string, code int) (cMsg tool.ConnMsg, err error) {
	_, err = c.conn.Write(b)
	if err != nil {
		return
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
}

func (c *ClientContext) GetSubConn(name string) (*SubConnContext, error) {
	tid := tool.NewId(1)
	tk := time.NewTicker(100 * time.Second)

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
		_, err = conn.Write(c.key.SetMsg(tool.TaskQ, "", 200, tool.OdjSub{
			SrcName: c.name,
			DstKey:  k,
		}))
		if err != nil {
			conn.Close()
			return nil, err
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
		s := &SubConnContext{
			id:   tool.NewId(1),
			conn: conn,
			f:    c,
		}
		c.subMap.Store(s.id, s)
		time.Sleep(500 * time.Millisecond)
		return s, nil
	}
	//return nil, errors.New("err")
}

func (c *ClientContext) ListenSubConn(fn func(sub *SubConnContext)) {
	for scc := range c.subListen {
		go fn(scc)
	}
}

type SubConnContext struct {
	id   string
	conn net.Conn
	f    *ClientContext
}

func (s *SubConnContext) GetConn() net.Conn {
	return s.conn
}

func (s *SubConnContext) Close() {
	s.conn.Close()
	s.f.subMap.Delete(s.id)
}

func (c *ClientContext) sendAndCallBack(timeout time.Duration, send tool.ConnMsg, fn func(cMsg tool.ConnMsg)) (timeOut <-chan time.Time) {
	tid := tool.NewId(1)
	tk := time.NewTicker(timeout * time.Second)
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
		Data:   name,
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
