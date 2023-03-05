package testcase

import (
	"fmt"
	"github.com/peakedshout/go-CFC/client"
	"github.com/peakedshout/go-CFC/server"
	"log"
)

type cfcTestCtx struct {
	ps *server.ProxyServer

	box1Name string
	box2Name string
	box1     *client.DeviceBox
	box2     *client.DeviceBox
}

func eqCheck[T string](a1, a2 T) {
	if a1 != a2 {
		panic(fmt.Sprintf("eq check err : %v != %v", a1, a2))
	}
}

func errCheck(err error) {
	if err != nil {
		panic(err)
	}
}
func newServer() *cfcTestCtx {
	ps := server.NewProxyServer(":9999", "6a647c0bf889419c84e461486f83d776")
	return &cfcTestCtx{
		ps:       ps,
		box1Name: "box1",
		box2Name: "box2",
		box1:     nil,
		box2:     nil,
	}
}
func (ctc *cfcTestCtx) new2Link() {
	box1, err := client.LinkProxyServer(ctc.box1Name, "127.0.0.1:9999", "6a647c0bf889419c84e461486f83d776")
	errCheck(err)
	box2, err := client.LinkProxyServer(ctc.box2Name, "127.0.0.1:9999", "6a647c0bf889419c84e461486f83d776")
	errCheck(err)
	ctc.box1 = box1
	ctc.box2 = box2
}

func (ctc *cfcTestCtx) closeAll() {
	if ctc.box1 != nil {
		ctc.box1.Close()
	}
	if ctc.box2 != nil {
		ctc.box2.Close()
	}
	if ctc.ps != nil {
		ctc.ps.Close()
	}
}

func (ctc *cfcTestCtx) listen(fn func(sub *client.SubBox)) chan<- error {
	ch := make(chan error)
	go func() {
		err := ctc.box1.ListenSubBox(func(sub *client.SubBox) {
			fn(sub)
		})
		ch <- err
	}()
	return ch
}

func (ctc *cfcTestCtx) dial() (*client.SubBox, error) {
	return ctc.box2.GetSubBox(ctc.box1Name)
}

var testCount int

func testPrint(str string) {
	switch testCount % 2 {
	case 0:
		log.Println("cfc test : ", str, "----- start	------------------------------")
	case 1:
		log.Println("cfc test : ", str, "----- end		------------------------------")
	}
	testCount++
}
