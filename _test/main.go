package main

import (
	"github.com/peakedshout/go-CFC/_test/testcase"
	"github.com/peakedshout/go-CFC/loger"
	"time"
)

func main() {
	loger.SetLoggerStack(false)
	testcase.Test01()
	time.Sleep(1 * time.Second)
	testcase.Test02()
	time.Sleep(1 * time.Second)
	testcase.Test03()
	time.Sleep(1 * time.Second)
	testcase.Test04()
	//time.Sleep(1 * time.Second)
	//testcase.Test05()
	time.Sleep(1 * time.Second)
	testcase.Test06()
}
