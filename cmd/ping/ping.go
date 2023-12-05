package main

import (
	"fmt"
	"time"

	"github.com/go-ping/ping"
)

func main() {
	ip := "192.168.4.254"
	fmt.Printf("checkip :%s\n", ip)
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		panic(err)

	}
	pinger.Count = 5
	pinger.Timeout = time.Second * 1
	start := time.Now()
	err = pinger.Run() // Blocks until finished.
	if err != nil {
		panic(err)
	}
	stats := pinger.Statistics()
	if stats.PacketsRecv > 0 {
	}
	fmt.Printf(" ping %s, fail, cost:%v\n", ip, time.Since(start).Seconds())
}
