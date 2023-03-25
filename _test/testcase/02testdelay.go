package testcase

import (
	"fmt"
	"github.com/peakedshout/go-CFC/loger"
	"time"
)

func Test02() {
	loger.SetLoggerLevel(loger.LogLevelError)
	testPrint("02 _test delay")
	defer testPrint("02 _test delay")
	ctx := newServer()
	defer ctx.closeAll()
	time.Sleep(1 * time.Second)
	ctx.new2Link()
	for i := 0; i < 5; i++ {
		fmt.Println(ctx.box1.GetOtherDelayPing())
		time.Sleep(1 * time.Second)
	}
}
