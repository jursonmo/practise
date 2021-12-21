package main

import (
	"expvar"
	"net/http"
	"os"
	"runtime"
	rtpprof "runtime/pprof"
	"time"

	"github.com/shirou/gopsutil/process"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/vars", exp)
	http.ListenAndServe(":8080", mux)
}

var h = expvar.Handler()
var cpuNum = expvar.NewInt("cpuNum")
var threadNum = expvar.NewInt("threadNum")
var grNum = expvar.NewInt("goroutineNum")
var threadProfile = rtpprof.Lookup("threadcreate")

var p, _ = process.NewProcess(int32(os.Getpid()))
var memPercent = expvar.NewInt("memPercent")
var cpuPercent = expvar.NewInt("cpuPercent")

func exp(w http.ResponseWriter, req *http.Request) {
	cpuNum.Set(int64(runtime.NumCPU()))
	threadNum.Set(int64(threadProfile.Count()))
	grNum.Set(int64(runtime.NumGoroutine()))

	mp, _ := p.MemoryPercent()
	memPercent.Set(int64(mp))
	cp, _ := p.Percent(time.Second)
	cpuPercent.Set(int64(cp))

	h.ServeHTTP(w, req)
}
