package testcase

import (
	"github.com/peakedshout/go-CFC/loger"
	"time"
)

func Test01() {
	loger.SetLoggerLevel(loger.LogLevelInfo)
	testPrint("01 test link")
	defer testPrint("01 test link")
	ctx := newServer()
	defer ctx.closeAll()
	time.Sleep(1 * time.Second)
	ctx.new2Link()
}
