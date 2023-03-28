package main

import (
	"flag"
	"github.com/peakedshout/go-CFC/_hook-tcp/config"
	"github.com/peakedshout/go-CFC/client"
	"github.com/peakedshout/go-CFC/loger"
	"github.com/peakedshout/go-CFC/tool"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

func main() {
	runClient()
}
func runClient() {
	p := flag.String("c", "./config.json", "config file path , default is ./config.json")
	flag.Parse()
	c := config.ReadConfig(*p)
	if c.Setting.LogLevel == 0 {
		c.Setting.LogLevel = loger.LogLevelWarn
	}
	loger.SetLoggerLevel(c.Setting.LogLevel)
	loger.SetLoggerStack(c.Setting.LogStack)

	var td time.Duration
	if c.Setting.ReLinkTime != "" {
		var err error
		td, err = time.ParseDuration(c.Setting.ReLinkTime)
		if err != nil {
			loger.SetLogError(err)
		}
	}
	bc := &boxContext{
		proxyHost:  c.ProxyServerHost,
		proxyTcp:   c.ProxyDeviceBox.ProxyTcp,
		reLinkTime: td,
	}
	bc.handleProxyTcp()

	if !bc.hasWork {
		loger.SetLogError("This is meaningless : no work to handle")
	}
	ch := make(chan os.Signal)
	<-ch
}

type boxContext struct {
	proxyHost config.ProxyServerHostConfig
	proxyTcp  config.ProxyTcp

	reLinkTime time.Duration

	hasWork bool
}

func (bc *boxContext) handleProxyTcp() {
	if len(bc.proxyTcp.Server) == 0 && len(bc.proxyTcp.Client) == 0 {
		return
	}
	bc.hasWork = true
	bc.handleProxyTcpServer()
	bc.handleProxyTcpClient()
}

func (bc *boxContext) handleProxyTcpServer() {
	if len(bc.proxyTcp.Server) == 0 {
		return
	}
	for _, one := range bc.proxyTcp.Server {
		info := one
		tAddr, err1 := net.ResolveTCPAddr("tcp", info.ServerDialAddr)
		if err1 != nil {
			loger.SetLogError(err1)
		}
		go reLink(bc.reLinkTime, func() {
			box, err := client.LinkProxyServer(info.ListenProxyName, bc.proxyHost.ProxyServerAddr, bc.proxyHost.LinkProxyKey)
			if err != nil {
				loger.SetLogWarn(err)
				return
			}
			err = box.ListenSubBox(func(sub *client.SubBox) {
				defer sub.Close()
				if info.ProxyCryptoKey != "" {

				} else {
					conn, err := net.DialTCP("tcp", nil, tAddr)
					if err != nil {
						loger.SetLogWarn(err)
						return
					}
					go func() {
						loger.SetLogInfo(io.Copy(conn, sub))
					}()
					loger.SetLogInfo(io.Copy(sub, conn))
				}
			})
			if err != nil {
				loger.SetLogWarn(err)
			}
		})
	}
}
func (bc *boxContext) handleProxyTcpClient() {
	if len(bc.proxyTcp.Client) == 0 {
		return
	}
	var box *client.DeviceBox
	var lock sync.Mutex
	var id string = tool.NewId(2)
	go reLink(bc.reLinkTime, func() {
		lock.Lock()
		var err error
		box, err = client.LinkProxyServer(id, bc.proxyHost.ProxyServerAddr, bc.proxyHost.LinkProxyKey)
		if err != nil {
			loger.SetLogWarn(err)
			lock.Unlock()
			return
		}
		defer box.Close()
		lock.Unlock()
		box.Wait()
	})

	fn := func(name string) (*client.SubBox, error) {
		lock.Lock()
		defer lock.Unlock()
		if box == nil {
			return nil, tool.ErrBoxIsNil
		}
		return box.GetSubBox(name)
	}
	fn2 := func(info config.ProxyTcpClientConfig, conn *net.TCPConn) {
		defer conn.Close()
		if info.ProxyCryptoKey != "" {

		} else {
			sub, err := fn(info.DialProxyName)
			if err != nil {
				loger.SetLogWarn(err)
				return
			}
			go func() {
				loger.SetLogInfo(io.Copy(conn, sub))
			}()
			loger.SetLogInfo(io.Copy(sub, conn))
		}
	}

	for _, one := range bc.proxyTcp.Client {
		info := one
		tAddr, err1 := net.ResolveTCPAddr("tcp", info.ClientListenAddr)
		if err1 != nil {
			loger.SetLogError(err1)
		}
		go func() {
			ln, err := net.ListenTCP("tcp", tAddr)
			if err != nil {
				loger.SetLogError(err1)
			}
			loger.SetLogMust(loger.SprintColor(5, 37, 37, "~~~ Listen ", info.DialProxyName, " : ", tAddr.String(), " ~~~"))
			defer ln.Close()
			for {
				conn, err := ln.AcceptTCP()
				if err != nil {
					loger.SetLogInfo(err)
				}
				go fn2(info, conn)
			}
		}()
	}
}

func reLink(td time.Duration, fn func()) {
	if td == 0 {
		fn()
	} else {
		if td < time.Second {
			td = time.Second
		}
		for {
			fn()
			time.Sleep(td)
		}
	}
}
