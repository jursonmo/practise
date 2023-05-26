package pprof

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
)

type Config struct {
	Enable  bool   `toml:"enable"`
	Address string `toml:"address"`
}

func Run(cfg *Config) {
	if cfg == nil || !cfg.Enable {
		fmt.Println("pprof config says not to run pprof")
		return
	}

	mux := new(http.ServeMux)
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/allocs", pprof.Handler("allocs").ServeHTTP)
	mux.HandleFunc("/debug/block", pprof.Handler("block").ServeHTTP)
	mux.HandleFunc("/debug/goroutine", pprof.Handler("goroutine").ServeHTTP)
	mux.HandleFunc("/debug/heap", pprof.Handler("heap").ServeHTTP)
	mux.HandleFunc("/debug/mutex", pprof.Handler("mutex").ServeHTTP)
	mux.HandleFunc("/debug/threadcreate", pprof.Handler("threadcreate").ServeHTTP)

	mux.HandleFunc("/debug/pprof/memory", func(w http.ResponseWriter, r *http.Request) {
		m := &runtime.MemStats{}
		runtime.ReadMemStats(m)

		mss, _ := json.MarshalIndent(m, "", "  ")
		w.Write(mss)
	})

	go func() {
		if err := http.ListenAndServe(cfg.Address, mux); err != nil {
			fmt.Printf("Pprof listen and serve error: %v\n", err)
		}
	}()
}
