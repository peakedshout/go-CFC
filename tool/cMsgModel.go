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
		return ErrCheckUnexpectedHeader
	}
	if cMsg.Code != code {
		return ErrCheckBadAny(cMsg.Code, cMsg.Data)
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

type VpnInfo struct {
	LinkConnType      string
	LinkRemoteAddrTcp *net.TCPAddr
	LinkLocalAddrTcp  *net.TCPAddr
	LinkRemoteAddrUdp *net.UDPAddr
	LinkLocalAddrUdp  *net.UDPAddr
	CopyConnType      string
	CopyRemoteAddrTcp *net.TCPAddr
	CopyLocalAddrTcp  *net.TCPAddr
	CopyRemoteAddrUdp *net.UDPAddr
	CopyLocalAddrUdp  *net.UDPAddr
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
	CopyConnTypeTCP = "CopyConnTypeTCP"
	CopyConnTypeUDP = "LinkConnTypeUDP"
)

type OdjVPNLinkAddr struct {
	ConnType string
	Addr     string

	AddrInfo *VpnInfo
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
