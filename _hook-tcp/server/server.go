package main

import (
	"flag"
	"github.com/peakedshout/go-CFC/_hook-tcp/config"
	"github.com/peakedshout/go-CFC/loger"
	"github.com/peakedshout/go-CFC/server"
	"time"
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

	var td time.Duration
	if c.Setting.ReLinkTime != "" {
		var err error
		td, err = time.ParseDuration(c.Setting.ReLinkTime)
		if err != nil {
			loger.SetLogError(err)
		}
	}
	if td == 0 {
		server.NewProxyServer(c.ProxyServerHost.ProxyServerAddr, c.ProxyServerHost.LinkProxyKey).Wait()
	} else {
		if td < time.Second {
			td = time.Second
		}
		for {
			server.NewProxyServer(c.ProxyServerHost.ProxyServerAddr, c.ProxyServerHost.LinkProxyKey).Wait()
			time.Sleep(td)
		}
	}
}
