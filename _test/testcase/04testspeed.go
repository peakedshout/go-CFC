package testcase

import (
	"fmt"
	"github.com/peakedshout/go-CFC/client"
	"github.com/peakedshout/go-CFC/loger"
	"time"
)

func Test04() {
	loger.SetLoggerLevel(loger.LogLevelError)
	testPrint("04 _test speed")
	defer testPrint("04 _test speed")
	ctx := newServer()
	defer ctx.closeAll()
	time.Sleep(1 * time.Second)
	ctx.new2Link()
	var sub1 *client.SubBox
	var sub2 *client.SubBox
	go func() {
		ctx.listen(func(sub *client.SubBox) {
			defer sub.Close()
			sub1 = sub
			var b [4 * 1024]byte
			for {
				_, _ = sub.Read(b[:])
				//errCheck(err)
			}
		})
		sub, err := ctx.dial()
		errCheck(err)
		sub2 = sub
		for {
			_, err = sub.Write([]byte("test123"))
			if err != nil {
				loger.SetLogWarn(err)
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	time.Sleep(2 * time.Second)
	for i := 0; i < 30; i++ {
		fmt.Println(ctx.box1.GetAllNetworkSpeedView())
		fmt.Println(ctx.box2.GetAllNetworkSpeedView())
		fmt.Println(sub2.GetNetworkSpeedView())
		fmt.Println(sub1.GetNetworkSpeedView())
		time.Sleep(100 * time.Millisecond)
	}
}
