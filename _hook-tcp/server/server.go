package main

import (
	"flag"
	"github.com/peakedshout/go-CFC/_hook-tcp/config"
	"github.com/peakedshout/go-CFC/loger"
	"github.com/peakedshout/go-CFC/server"
	"github.com/peakedshout/go-CFC/tool"
)

func main() {
	runServer()
}

func runServer() {
	p := flag.String("c", "./config.json", "config file path , default is ./config.json")
	flag.Parse()
	c := config.ReadConfig(*p)
	if c.Setting.LogLevel == 0 {
		c.Setting.LogLevel = loger.LogLevelWarn
	}
	loger.SetLoggerLevel(c.Setting.LogLevel)
	loger.SetLoggerStack(c.Setting.LogStack)

	sc := &server.Config{
		RawKey:           c.ProxyServerHost.LinkProxyKey,
		LnAddr:           c.ProxyServerHost.ProxyServerAddr,
		HandleWaitTime:   0,
		PingWaitTime:     0,
		CGTaskTime:       0,
		SwitchVPNProxy:   c.ProxyServerHost.SwitchVPNProxy,
		SwitchLinkClient: c.ProxyServerHost.SwitchLinkClient,
		SwitchUdpP2P:     c.ProxyServerHost.SwitchUdpP2P,
	}

	err := tool.ReRun(c.Setting.ReLinkTime, func() bool {
		server.NewProxyServer2(sc).Wait()
		return true
	})
	if err != nil {
		loger.SetLogError(err)
	}
}
