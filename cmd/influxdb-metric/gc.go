package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

func main() {
	s := debug.GCStats{}
	debug.ReadGCStats(&s)
	fmt.Printf("gcNum:%d\n", s.NumGC)

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	fmt.Printf("Alloc:%d HeapInuse:%d HeapIdle:%d HeapReleased:%d\n", ms.Alloc, ms.HeapInuse, ms.HeapIdle, ms.HeapReleased)

	debug.ReadGCStats(&s)
	fmt.Printf("after ReadMemStats gcNum:%d\n", s.NumGC)

	runtime.GC()
	debug.ReadGCStats(&s)
	fmt.Printf("after runtime.GC() gcNum:%d\n", s.NumGC)
}

/*
gcNum:0
Alloc:94992 HeapInuse:401408 HeapIdle:66478080 HeapReleased:66445312
after ReadMemStats gcNum:0
after runtime.GC() gcNum:1
*/
