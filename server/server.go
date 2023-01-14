package server

import (
	"bufio"
	"github.com/peakedshout/go-CFC/tool"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type ServerContext struct {
	ip   string
	port string

	key       tool.Key
	clientMap sync.Map //k name v *Client

	taskRoomMap    sync.Map //k id v *taskRoom
	taskRoomDelMap sync.Map //k time v task id
}
type ClientInfo struct {
	s *ServerContext

	id   string
	name string

	fastConn    atomic.Bool
	fastOdjChan chan [][]byte
	step        *int64
	stop        chan uint8

	conn      net.Conn
	writeChan chan [][]byte
	ping      tool.Ping

	parent string
	subMap sync.Map
}

func NewServer(ip, port, key string) {
	if len(key) != 32 {
		panic("key is not 32byte")
	}
	s := &ServerContext{
		ip:             ip,
		port:           port,
		key:            tool.NewKey(key),
		clientMap:      sync.Map{},
		taskRoomMap:    sync.Map{},
		taskRoomDelMap: sync.Map{},
	}
	go func() {
		t := time.NewTicker(1 * time.Minute)
		defer t.Stop()
		select {
		case <-t.C:
			s.taskRoomDelMap.Range(func(key, value any) bool {
				if time.Now().Sub(key.(time.Time)) > 0 {
					s.taskRoomMap.Delete(value)
					s.taskRoomDelMap.Delete(key)
				}
				return true
			})
			//case svc.Stop:

		}
	}()

	ln, err := net.Listen("tcp", s.ip+s.port)
	if err != nil {
		panic(err)
	}
	defer ln.Close()
	for true {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			return
		}
		go s.tcpHandler(conn)
	}
}

func (s *ServerContext) tcpHandler(conn net.Conn) {
	i := int64(0)
	c := &ClientInfo{
		s:        s,
		id:       tool.NewId(1),
		name:     "",
		fastConn: atomic.Bool{},
		//fastOdjChan: make(chan []byte, 100),
		step:      &i,
		stop:      make(chan uint8, 10),
		conn:      conn,
		writeChan: make(chan [][]byte, 100),
		ping:      tool.Ping{},
		parent:    "",
		subMap:    sync.Map{},
	}
	defer c.close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	//s.ClientMap.Store(c.Id, c)
	go func() {
		for true {
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
				return
			}
		}
	}()
	reader := bufio.NewReaderSize(conn, tool.BufferSize)
	for true {
		cMsg, err := s.key.GetMsg(reader)
		if err != nil {
			tool.Println(conn, err)
			break
		}
		s.cMsgHandler(c, cMsg)
		if c.fastConn.Load() {
			break
		}
	}
	for {
		if c.fastConn.Load() {
			var b [1024]byte
			n, err := reader.Read(b[:])
			if err != nil {
				tool.Println(conn, err)
				c.close()
				return
			}
			c.writerFast(b[:n])
		} else {
			c.close()
			break
		}
	}
}
func (s *ServerContext) cMsgHandler(c *ClientInfo, msg tool.ConnMsg) {
	switch atomic.LoadInt64(c.step) {
	case 0:
		if msg.Header == tool.HandshakeCheckStepQ1 {
			atomic.AddInt64(c.step, 1)
			c.writerData(s.key.SetMsg(tool.HandshakeCheckStepA1, "", 200, nil))
			tool.Println(c.conn, msg)
		}
		if msg.Header == tool.TaskQ {
			var info tool.OdjSub
			err := tool.UnmarshalV2(msg.Data, &info)
			if err != nil {
				c.close()
				return
			}
			if info.DstKey == "" {
				c.close()
				return
			}
			if info.SrcName != "" {
				v, ok := s.clientMap.Load(info.SrcName)
				if !ok {
					c.close()
					return
				}
				parent := v.(*ClientInfo)
				parent.subMap.Store(c.id, c)
				c.parent = parent.name
			}
			s.JoinTaskRoom(info.DstKey, c)
			c.conn.SetDeadline(time.Time{})
		}
	case 1:
		if msg.Header == tool.HandshakeCheckStepQ2 {
			var info tool.OdjClientInfo
			tool.UnmarshalV2(msg.Data, &info)
			atomic.AddInt64(c.step, 1)
			if info.Name == "" {
				c.writerData(s.key.SetMsg(tool.HandshakeCheckStepA2, "", 400, nil))
				c.close()
				return
			}
			c.name = info.Name
			s.clientMap.Store(c.name, c)
			c.writerData(s.key.SetMsg(tool.HandshakeCheckStepA2, "", 200, nil))
			tool.Println(c.conn, msg)
		}
	case 2:
		if msg.Header == tool.PingMsg {
			var info tool.Ping
			err := tool.UnmarshalV2(msg.Data, &info)
			if err != nil {
				c.close()
			}
			c.ping = info
			c.SetDeadline(time.Now().Add(60 * time.Second))
			c.writerData(s.key.SetMsg(tool.PongMsg, "", 200, nil))
			//tool.Println(c.conn, info)
		}
		if msg.Header == tool.SOpenQ {
			var info tool.OdjMsg
			err := tool.UnmarshalV2(msg.Data, &info)
			if err != nil {
				c.writerData(s.key.SetMsg(tool.SOpenA, msg.Id, 400, tool.OdjMsg{Msg: "bad req"}))
				return
			}
			odj, ok := s.clientMap.Load(info.Msg)
			if !ok {
				c.writerData(s.key.SetMsg(tool.SOpenA, msg.Id, 400, tool.OdjMsg{Msg: "nobody"}))
				return
			}
			tid := s.NewTaskRoom()
			odj.(*ClientInfo).writerData(s.key.SetMsg(tool.SOpenA, "", 200, tool.OdjMsg{Msg: tid}))
			c.writerData(s.key.SetMsg(tool.SOpenA, msg.Id, 200, tool.OdjMsg{Msg: tid}))
		}
		if msg.Header == tool.DelayQ {
			var info tool.OdjIdList
			err := tool.UnmarshalV2(msg.Data, &info)
			if err != nil {
				c.writerData(s.key.SetMsg(tool.DelayA, msg.Id, 400, tool.OdjMsg{Msg: "bad req"}))
				return
			}
			var resp []tool.OdjPing
			if len(info.IdList) == 0 {
				s.clientMap.Range(func(key, value any) bool {
					resp = append(resp, tool.OdjPing{
						Name:   key.(string),
						Ping:   value.(*ClientInfo).ping,
						Active: true,
					})
					return true
				})
			} else {
				for _, one := range info.IdList {
					odj, ok := s.clientMap.Load(one)
					if !ok {
						resp = append(resp, tool.OdjPing{
							Name:   one,
							Ping:   tool.Ping{},
							Active: false,
						})
					} else {
						resp = append(resp, tool.OdjPing{
							Name:   one,
							Ping:   odj.(*ClientInfo).ping,
							Active: true,
						})
					}
				}
			}
			c.writerData(s.key.SetMsg(tool.DelayA, msg.Id, 200, resp))
		}
	}
}
func (c *ClientInfo) writerData(b [][]byte) {
	c.writeChan <- b
}
func (c *ClientInfo) writerFast(b []byte) {
	c.fastOdjChan <- [][]byte{b}
}
func (c *ClientInfo) close() {
	c.subMap.Range(func(key, value any) bool {
		value.(*ClientInfo).close()
		c.subMap.Delete(key)
		return true
	})
	if c.name != "" {
		c.s.clientMap.Delete(c.name)
	}
	c.stop <- 1
}
func (c *ClientInfo) closeSub() {

}
func (c *ClientInfo) SetDeadline(t time.Time) {
	if err := c.conn.SetDeadline(t); err != nil {
		c.close()
	}
}

type taskRoom struct {
	id   string
	join chan uint8
	//c2 chan uint8
	c1 *ClientInfo
	c2 *ClientInfo
}

func (s *ServerContext) NewTaskRoom() string {
	t := &taskRoom{
		id:   tool.NewId(1),
		join: make(chan uint8, 1),
		c1:   nil,
		c2:   nil,
	}
	t.join <- 1
	s.taskRoomMap.Store(t.id, t)
	s.taskRoomDelMap.Store(time.Now().Add(30*time.Second), t.id)
	return t.id
}
func (s *ServerContext) JoinTaskRoom(id string, c *ClientInfo) {
	task, ok := s.taskRoomMap.Load(id)
	if !ok {
		c.close()
		return
	}
	taskR := task.(*taskRoom)
	select {
	case <-taskR.join:
		taskR.c1 = c
	default:
		s.taskRoomMap.Delete(id)
		taskR.c2 = c
		taskR.c1.fastOdjChan = taskR.c2.writeChan
		taskR.c2.fastOdjChan = taskR.c1.writeChan
		taskR.c1.fastConn.Store(true)
		taskR.c2.fastConn.Store(true)
		taskR.c1.writerData(s.key.SetMsg(tool.TaskA, taskR.c2.parent, 200, nil))
		taskR.c2.writerData(s.key.SetMsg(tool.TaskA, taskR.c1.parent, 200, nil))
	}
}
