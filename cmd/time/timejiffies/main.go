package main

import (
	"fmt"
	"time"
)

func main() {
	start := time.Now()
	for {
		time.Sleep(time.Second)
		fmt.Printf("时间戳（秒）：%v;\n", time.Now().Unix())
		fmt.Printf("时间戳（纳秒）：%v;\n", time.Now().UnixNano())
		fmt.Printf("运行（秒）：%d\n", time.Since(start)/time.Second)
	}
}
