package main

import (
	"fmt"
	"math"
	"runtime"
	"time"
)

func main() {
	fmt.Printf("%v,%s\n", 10*time.Millisecond, 10*time.Millisecond)
	var a = [5]int{1}
	fmt.Println(a)
	ballast := make([]byte, 1024*1024*1024) //1G

	<-time.After(time.Duration(math.MaxInt64))
	runtime.KeepAlive(ballast)

}
