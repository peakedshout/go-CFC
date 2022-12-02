package main

import (
	"bufio"
	"go-CFC/client"
	"go-CFC/server"
	"log"
	"time"
)

func main() {
	test1()
}
func test1() {
	go server.NewServer("127.0.0.1", ":9999", "6a647c0bf889419c84e461486f83d776")
	time.Sleep(1 * time.Second)
	c, err := client.LinkLongConn("test1", "127.0.0.1", ":9999", "6a647c0bf889419c84e461486f83d776")
	if err != nil {
		panic(err)
	}
	go func() {
		c1, err := client.LinkLongConn("test2", "127.0.0.1", ":9999", "6a647c0bf889419c84e461486f83d776")
		if err != nil {
			panic(err)
		}
		c1.ListenSubConn(func(sub *client.SubConnContext) {
			reader := bufio.NewReader(sub.GetConn())
			for {
				var b [1024]byte
				_, err := reader.Read(b[:])
				if err != nil {
					panic(err)
				}
				//tool.Println(sub.conn, "ln:", string(b[:n]))
			}
		})
	}()
	time.Sleep(2 * time.Second)
	for {
		p, err := c.GetOtherDelayPing()
		if err != nil {
			panic(err)
		}
		log.Println(p)
		time.Sleep(1 * time.Second)
	}
}
