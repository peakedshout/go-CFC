package tool

import (
	"time"
)

type ConnMsg struct {
	Header string
	Code   int
	Data   interface{}
	Id     string
}

func (cMsg *ConnMsg) CheckConnMsgHeaderAndCode(header string, code int) error {
	if cMsg.Header != header {
		return ErrReqUnexpectedHeader
	}
	if cMsg.Code != code {
		return ErrReqBadAny(cMsg.Code, cMsg.Data)
	}
	return nil
}

type Ping struct {
	Ping time.Duration
}

type OdjClientInfo struct {
	Name string
}

type OdjAddr struct {
	Id   string
	Addr string
}

type OdjMsg struct {
	Msg string
}

type OdjInfo struct {
	Id       string
	User     string
	Password string
}

type OdjSub struct {
	SrcName string
	DstKey  string
}

type OdjIdList struct {
	IdList []string
}
type OdjPing struct {
	Name   string
	Ping   Ping
	Active bool
}

type OdjP2P struct {
	Addr   string
	Status string
}
