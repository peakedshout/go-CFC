package main

import (
	"flag"
	"github.com/peakedshout/go-CFC/client"
	"github.com/peakedshout/go-CFC/tool"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
)

func main() {
	path := flag.String("c", "./config.json", `Default configuration file location. If not specified, the default is "./config.json".`)
	flag.Parse()
	config := tool.GetCFCHookConfig(*path)
	if len(config.Tcp.Server) == 0 && len(config.Tcp.Client) == 0 {
		panic("meaningless")
	}
	for _, one := range config.Tcp.Server {
		info := one
		go func() {
			c, err := client.LinkLongConn(info.Name, config.Ct.IP, config.Ct.Port, config.Ct.Key)
			if err != nil {
				panic(err)
			}
			err = c.ListenSubConn(func(sub *client.SubConnContext) {
				defer sub.Close()
				conn, err := net.Dial("tcp", info.IP+info.Port)
				if err != nil {
					log.Println(err)
					return
				}
				defer conn.Close()
				go func() {
					_, err = io.Copy(sub.GetConn(), conn)
					if err != nil {
						log.Println(err)
						return
					}
				}()
				_, err = io.Copy(conn, sub.GetConn())
				if err != nil {
					log.Println(err)
					return
				}
			})
			if err != nil {
				log.Println(err)
			}
		}()
	}
	if len(config.Tcp.Client) != 0 {
		c, err := client.LinkLongConn(config.Ct.Name, config.Ct.IP, config.Ct.Port, config.Ct.Key)
		if err != nil {
			panic(err)
		}
		for _, one := range config.Tcp.Client {
			info := one
			go func() {
				ln, err := net.Listen("tcp", info.IP+info.Port)
				if err != nil {
					panic(err)
				}
				defer ln.Close()
				log.Println("client Listen:", info.IP+info.Port)
				for {
					conn, err := ln.Accept()
					if err != nil {
						log.Println(err)
						return
					}
					go handler(conn, c, info.Name)
				}
			}()
		}
	}
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan
}
func handler(conn net.Conn, c *client.ClientContext, name string) {
	sub, err := c.GetSubConn(name)
	if err != nil {
		log.Println(err)
		return
	}
	defer sub.Close()
	defer conn.Close()
	go func() {
		_, err = io.Copy(conn, sub.GetConn())
		if err != nil {
			log.Println(err)
			return
		}
	}()
	_, err = io.Copy(sub.GetConn(), conn)
	if err != nil {
		log.Println(err)
		return
	}
}
