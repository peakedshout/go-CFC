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

	err := tool.ReRun(c.Setting.ReLinkTime, func() {
		server.NewProxyServer(c.ProxyServerHost.ProxyServerAddr, c.ProxyServerHost.LinkProxyKey).Wait()
	})
	if err != nil {
		loger.SetLogError(err)
	}
}
