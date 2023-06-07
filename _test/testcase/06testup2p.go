package testcase

import (
	"github.com/peakedshout/go-CFC/client"
	"github.com/peakedshout/go-CFC/loger"
	"time"
)

func Test06() {
	loger.SetLoggerLevel(loger.LogLevelError)
	testPrint("06 _test up2p")
	defer testPrint("06 _test up2p")
	ctx := newServer()
	defer ctx.closeAll()
	time.Sleep(1 * time.Second)
	ctx.new2Link()

	a2 := ""
	ctx.listen(func(sub *client.SubBox) {
		defer sub.Close()
		var b [4 * 1024]byte
		n, err := sub.Read(b[:])
		errCheck(err)
		a2 = string(b[:n])
	})
	sub, err := ctx.dialByUP2P()
	errCheck(err)
	test := "test123"
	_, err = sub.Write([]byte(test))
	errCheck(err)
	time.Sleep(1 * time.Second)
	eqCheck(test, a2)
}
