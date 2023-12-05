package main

import (
	"fmt"
	"time"
)

func main() {
	i := 0
	next := time.Now()
	t := time.NewTimer(time.Second)
	for {
		select {
		case tt := <-t.C:
			d := next.Sub(time.Now())
			fmt.Printf("i:%d, tt:%v, next exec after d:%v\n", i, tt, d)
			t.Reset(d) //d 是一个负数(-1秒左右)，结果是，timer 马上继续被调度执行。
			i++
			if i == 3 {
				return
			}
		}
	}
}

func ResetToNext(t *time.Timer, next time.Time) {
	// n := next.Sub(time.Now())
	// if n > time.Duration(0) {
	// 	t.Reset(n)
	// 	return
	// }
	// t.Reset(time.Nanosecond)
	t.Reset(next.Sub(time.Now()))
}

/*go run reset.go
i:0, tt:2023-12-05 14:53:33.599408 +0800 CST m=+1.005299245, next exec after d:-1.005275831s
i:1, tt:2023-12-05 14:53:33.600098 +0800 CST m=+1.005989385, next exec after d:-1.005909319s
i:2, tt:2023-12-05 14:53:33.600126 +0800 CST m=+1.006017215, next exec after d:-1.005934448s
*/
