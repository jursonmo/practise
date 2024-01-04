package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/go-ping/ping"
)

// 在root 用户下, 在非default namespace下运行程序， pinger.Run() 可能返回 socket: permission denied， 就需要把pinger.SetPrivileged(true)。
// 如果在root 用户下最好默认设置为true，不管在哪个net namespace 运行，可以保证不会返回错误
//如果在非root用下,只能默认设置为false.

/*
如果在非root用下，pinger.SetPrivileged(true)会返回错误：listen ip4:icmp : socket: operation not permitted

ubuntu@gw:~$ ./ping -dst=192.168.10.5 -p=true
ping :192.168.10.5
source :
panic: listen ip4:icmp : socket: operation not permitted
*/

/*
https://github.com/go-ping/ping/blob/master/README.md#linux
This library attempts to send an "unprivileged" ping via UDP. On Linux, this must be enabled with the following sysctl command:
README.md 说在linux , 默认是通过udp 来ping, 但是通过抓包看，是icmp, 不管是否设置 pinger.SetPrivileged()
*/

var (
	dst = flag.String("dst", "", "ping dst ip")
	src = flag.String("src", "", "ping dst ip")
	p   = flag.Bool("p", false, "false udp? true mean's icmp")
)

func main() {
	flag.Parse()
	fmt.Printf("ping :%s\n", *dst)
	pinger, err := ping.NewPinger(*dst)
	if err != nil {
		panic(err)

	}
	pinger.Source = *src
	fmt.Printf("source :%s\n", *src)
	pinger.Count = 5
	pinger.Timeout = time.Second * 1
	pinger.Interval = time.Millisecond * 100
	pinger.SetPrivileged(*p)
	start := time.Now()
	err = pinger.Run() // Blocks until finished.
	if err != nil {
		panic(err)
	}

	stats := pinger.Statistics()
	if stats.PacketsRecv > 0 {
		fmt.Printf(" ping %s, count:%d, PacketsRecv:%d, cost:%v\n", *dst, pinger.Count, stats.PacketsRecv, time.Since(start).Seconds())
	} else {
		fmt.Printf(" ping %s, fail, cost:%v\n", *dst, time.Since(start).Seconds())
	}
}
