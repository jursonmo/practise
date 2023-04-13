package main

import (
	"fmt"
	"time"
)

//GOOS=linux go build main.go
//timedatectl set-ntp false 关门ntp 同步时间
//date -s "2023-4-13 17:00:00"  设置系统时间
func main() {
	start := time.Now()
	for {
		time.Sleep(time.Second * 2)
		fmt.Printf("since:%v, time.Now:%v \n", time.Since(start), time.Now())
	}
}

//结论是， 系统时间改变，不影响time.Since(start)， 影响time.Now()
