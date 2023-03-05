package testcase

import (
	"github.com/peakedshout/go-CFC/client"
	"github.com/peakedshout/go-CFC/loger"
	"time"
)

func Test03() {
	loger.SetLoggerLevel(loger.LogLevelError)
	testPrint("03 test sub")
	defer testPrint("03 test sub")
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
	sub, err := ctx.dial()
	errCheck(err)
	test := "test123"
	_, err = sub.Write([]byte(test))
	errCheck(err)
	time.Sleep(1 * time.Second)
	eqCheck(test, a2)
}
