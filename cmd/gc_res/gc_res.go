package main

import (
	"flag"
	"fmt"
	"net/http"
	"runtime"
	"runtime/debug"
)

var mem []byte
var addr = flag.String("addr", ":8080", "listen addr")

func main() {
	flag.Parse()
	http.HandleFunc("/alloc", allocMemory)
	http.HandleFunc("/free", freeMemory)
	fmt.Printf("listen:%s\n", *addr)
	http.ListenAndServe(*addr, nil)
}
func allocMemory(w http.ResponseWriter, req *http.Request) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	//ms.Alloc same as ms.HeapAlloc
	before := fmt.Sprintf("HeapSys:%d Alloc:%d HeapAlloc:%d NextGC:%d HeapObjects:%d HeapInuse:%d HeapIdle:%d HeapReleased:%d, goroutineNum:%d\n",
		ms.HeapSys, ms.Alloc, ms.HeapAlloc, ms.NextGC, ms.HeapObjects, ms.HeapInuse, ms.HeapIdle, ms.HeapReleased, runtime.NumGoroutine())
	mem = make([]byte, 20000*1000)
	runtime.ReadMemStats(&ms)
	done := fmt.Sprintf("HeapSys:%d Alloc:%d HeapAlloc:%d NextGC:%d HeapObjects:%d HeapInuse:%d HeapIdle:%d HeapReleased:%d, goroutineNum:%d\n",
		ms.HeapSys, ms.Alloc, ms.HeapAlloc, ms.NextGC, ms.HeapObjects, ms.HeapInuse, ms.HeapIdle, ms.HeapReleased, runtime.NumGoroutine())
	w.Write([]byte(before + "allocMemory done:\n" + done))
}

func freeMemory(w http.ResponseWriter, req *http.Request) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	//ms.Alloc same as ms.HeapAlloc
	before := fmt.Sprintf("HeapSys:%d Alloc:%d HeapAlloc:%d NextGC:%d HeapObjects:%d HeapInuse:%d HeapIdle:%d HeapReleased:%d, goroutineNum:%d\n",
		ms.HeapSys, ms.Alloc, ms.HeapAlloc, ms.NextGC, ms.HeapObjects, ms.HeapInuse, ms.HeapIdle, ms.HeapReleased, runtime.NumGoroutine())
	mem = nil
	runtime.GC()
	debug.FreeOSMemory()
	runtime.ReadMemStats(&ms)
	done := fmt.Sprintf("HeapSys:%d Alloc:%d HeapAlloc:%d NextGC:%d HeapObjects:%d HeapInuse:%d HeapIdle:%d HeapReleased:%d, goroutineNum:%d\n",
		ms.HeapSys, ms.Alloc, ms.HeapAlloc, ms.NextGC, ms.HeapObjects, ms.HeapInuse, ms.HeapIdle, ms.HeapReleased, runtime.NumGoroutine())
	w.Write([]byte(before + "gc and freeOSMemory done:\n" + done))
}
