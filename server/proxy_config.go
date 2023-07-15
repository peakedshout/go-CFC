package server

import (
	"github.com/peakedshout/go-CFC/loger"
	"github.com/peakedshout/go-CFC/tool"
	"net"
	"time"
)

type Config struct {
	RawKey string
	LnAddr string

	lnAddr net.Addr

	HandleWaitTime time.Duration //new conn wait handle time, if cant known set, please set zero.
	PingWaitTime   time.Duration //new client wait ping time, if cant known set, please set zero.
	CGTaskTime     time.Duration //gc task room time, if cant known set, please set zero.

	SwitchVPNProxy   bool
	SwitchLinkClient bool
	SwitchUdpP2P     bool
	SwitchAnonymity  bool
}

func (config *Config) check() {
	if len(config.RawKey) != 32 {
		loger.SetLogError(tool.ErrKeyIsNot32Bytes)
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", config.LnAddr)
	if err != nil {
		loger.SetLogError("ResolveTCPAddr :", err)
	}
	config.lnAddr = tcpAddr

	if config.HandleWaitTime == 0 {
		config.HandleWaitTime = 10 * time.Second
	}
	if config.PingWaitTime == 0 {
		config.PingWaitTime = 30 * time.Second
	}
	if config.CGTaskTime == 0 {
		config.CGTaskTime = 60 * time.Second
	}
}
