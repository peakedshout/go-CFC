package testcase

import (
	"github.com/peakedshout/go-CFC/loger"
	"time"
)

func Test07() {
	loger.SetLoggerLevel(loger.LogLevelInfo)
	testPrint("07 _test linkAnonymity")
	defer testPrint("07 _test linkAnonymity")
	ctx := newServer()
	defer ctx.closeAll()
	time.Sleep(1 * time.Second)
	ctx.new2LinkAnonymity()
}
