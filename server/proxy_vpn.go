package server

import (
	"bufio"
	"github.com/peakedshout/go-CFC/tool"
	"net"
)

func (pc *ProxyClient) linkVPNConn(info *tool.OdjVPNLinkAddr) error {
	switch info.ConnType {
	case tool.LinkConnTypeTCP:
		raddr, err := net.ResolveTCPAddr("", info.Addr)
		if err != nil {
			pc.SetInfoLog("vpn:", info.ConnType, info.Addr, err)
			return err
		}
		conn, err := net.DialTCP("tcp", nil, raddr)
		if err != nil {
			pc.SetInfoLog("vpn:", info.ConnType, info.Addr, err)
			return err
		}
		info.AddrInfo.LinkConnType = tool.LinkConnTypeTCP
		info.AddrInfo.LinkRemoteAddrTcp = conn.RemoteAddr().(*net.TCPAddr)
		info.AddrInfo.LinkLocalAddrTcp = conn.LocalAddr().(*net.TCPAddr)
		pc.initVPNSettings(conn)
		return nil
	case tool.LinkConnTypeUDP:
		raddr, err := net.ResolveUDPAddr("", info.Addr)
		if err != nil {
			pc.SetInfoLog("vpn:", info.ConnType, info.Addr, err)
			return err
		}
		conn, err := net.DialUDP("tcp", nil, raddr)
		if err != nil {
			pc.SetInfoLog("vpn:", info.ConnType, info.Addr, err)
			return err
		}
		info.AddrInfo.LinkConnType = tool.LinkConnTypeUDP
		info.AddrInfo.LinkRemoteAddrUdp = conn.RemoteAddr().(*net.UDPAddr)
		info.AddrInfo.LinkLocalAddrUdp = conn.LocalAddr().(*net.UDPAddr)
		pc.initVPNSettings(conn)
		return nil
	default:
		return tool.ErrUnexpectedLinkConnType
	}
}

func (pc *ProxyClient) initVPNSettings(conn net.Conn) {
	pc.initLinkConn(conn, LinkTypeVPN, func(reader *bufio.Reader) ([][]byte, error) {
		cMsg, err := pc.key.ReadCMsg(pc.reader, nil, pc.networkSpeed.Upload)
		if err != nil {
			return nil, err
		}
		var b []byte
		err = cMsg.Unmarshal(&b)
		if err != nil {
			return nil, err
		}
		err = cMsg.CheckConnMsgHeaderAndCode(tool.ConnVPNQ2, 200)
		if err != nil {
			return nil, err
		}
		return [][]byte{b}, nil
	}, func(reader *bufio.Reader) ([][]byte, error) {
		buf := make([]byte, reader.Size())
		n, err := reader.Read(buf)
		if err != nil {
			return nil, err
		}
		return pc.key.SetMsg(tool.ConnVPNA2, "", 200, buf[:n]), nil
	})
}
