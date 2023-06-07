package cfc

import (
	"github.com/peakedshout/go-CFC/client"
	"github.com/peakedshout/go-CFC/server"
	"net"
)

type Listener struct {
	box *client.DeviceBox
}

func (ln *Listener) Accept() (conn net.Conn, err error) {
	return ln.box.ListenSubBoxOnce()
}
func (ln *Listener) Close() error {
	return ln.box.Close()
}
func (ln *Listener) Addr() net.Addr {
	return nil
}

func Listen(lname string, proxyAddr string, key string) (net.Listener, error) {
	box, err := client.LinkProxyServer(lname, proxyAddr, key)
	if err != nil {
		return nil, err
	}
	return &Listener{box: box}, nil
}

type Dialer struct {
	box *client.DeviceBox
}

func (dl *Dialer) Close() error {
	return dl.box.Close()
}

func (dl *Dialer) Call(rname string) (net.Conn, error) {
	return dl.box.GetSubBox(rname)
}

func (dl *Dialer) CallUP2P(rname string) (net.Conn, error) {
	return dl.box.GetSubBoxByUP2P(rname)
}

func Dial(lname string, proxyAddr string, key string) (*Dialer, error) {
	box, err := client.LinkProxyServer(lname, proxyAddr, key)
	if err != nil {
		return nil, err
	}
	return &Dialer{box: box}, nil
}

func Proxy(proxyAddr string, key string) *server.ProxyServer {
	return server.NewProxyServer(proxyAddr, key)
}
