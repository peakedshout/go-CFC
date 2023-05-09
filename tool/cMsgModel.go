package tool

import (
	"fmt"
	"net"
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

func (cMsg *ConnMsg) Unmarshal(out any) error {
	return UnmarshalV2(cMsg.Data, out)
}

func (cMsg *ConnMsg) MustUnmarshal(out any) {
	MustUnmarshalV2(cMsg.Data, out)
}

type Ping struct {
	Ping time.Duration
}

type OdjErrMsg struct {
	ErrMsg string
}

func NewErrMsg(pre string, err error) OdjErrMsg {
	str := fmt.Sprintf(pre+" %v", err)
	return OdjErrMsg{ErrMsg: str}
}

type OdjClientInfo struct {
	Name string
}

type OdjMsg struct {
	Msg string
}

type OdjSubOpenReq struct {
	Type    string
	OdjName string
}
type OdjSubOpenResp struct {
	Tid  string
	Type string
}

type OdjSubReq struct {
	Id      string
	SrcName string
	DstKey  string
	Addr    *net.TCPAddr
}

type SubInfo struct {
	LocalName  string
	RemoteName string

	LocalIntranetAddr  *net.TCPAddr
	RemoteIntranetAddr *net.TCPAddr

	LocalPublicAddr  *net.TCPAddr
	RemotePublicAddr *net.TCPAddr
}

type OdjIdList struct {
	IdList []string
}
type OdjPing struct {
	Name   string
	Ping   Ping
	Active bool
}

const (
	LinkConnTypeTCP = "LinkConnTypeTCP"
	LinkConnTypeUDP = "LinkConnTypeUDP"
)

type OdjVPNLinkAddr struct {
	ConnType string
	Addr     string
}

//type OdjHttpVPNLinkReq struct {
//	Id      string
//	SrcName string
//	Key     string
//	Data    []byte
//}

//type OdjHttpVPNOpenReq struct {
//	Addr string
//}
//type OdjHttpVPNOpenResp struct {
//	Key string
//}

//type OdjHttpVPNData struct {
//	Data []byte
//}
//type OdjHttpVPNReq struct {
//	Addr string
//}
