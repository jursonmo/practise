package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/debug"
	"time"
)

func printMemStats() {
	t := time.NewTicker(time.Second)
	s := runtime.MemStats{}
	for {
		select {
		case <-t.C:
			runtime.ReadMemStats(&s)
			fmt.Printf("gc %d last@%v, next_heap_size@%vMB\n", s.NumGC, time.Unix(int64(time.Duration(s.LastGC).Seconds()), 0), s.NextGC/(1<<20))
		}
	}
}
func printGCStats() {
	t := time.NewTicker(time.Second)
	s := debug.GCStats{}
	for {
		select {
		case <-t.C:
			debug.ReadGCStats(&s)
			fmt.Printf("gc %d last@%v, PauseTotal %v, Pause:%+v, PauseEnd:%+v\n", s.NumGC, s.LastGC, s.PauseTotal, s.Pause, s.PauseEnd)
			data, _ := json.MarshalIndent(&s, "", "\t")
			fmt.Printf("%s\n", string(data))
		}
	}
}

func main() {
	go printGCStats()
	//go printMemStats()
	go alloc()
	for {

		select {}
	}
}

var data []byte

func alloc() {
	i := 0
	for {
		time.Sleep(2 * time.Second)
		i++
		fmt.Printf("%d , alloc\n", i)
		data = make([]byte, 10000*i)
	}
}

/*
.......
25 , alloc
gc 0 last@1970-01-01 08:00:00 +0800 CST, PauseTotal 0s, Pause:[]
gc 0 last@1970-01-01 08:00:00 +0800 CST, PauseTotal 0s, Pause:[]
gc 0 last@1970-01-01 08:00:00 +0800 CST, PauseTotal 0s, Pause:[]
26 , alloc
gc 1 last@2021-09-01 17:15:08.783839 +0800 CST, PauseTotal 13.782µs, Pause:[13.782µs]
gc 1 last@2021-09-01 17:15:08.783839 +0800 CST, PauseTotal 13.782µs, Pause:[13.782µs]
gc 1 last@2021-09-01 17:15:08.783839 +0800 CST, PauseTotal 13.782µs, Pause:[13.782µs]
27 , alloc
......
44 , alloc
gc 2 last@2021-09-01 17:15:41.810728 +0800 CST, PauseTotal 24.809µs, Pause:[11.027µs 13.782µs]
gc 2 last@2021-09-01 17:15:41.810728 +0800 CST, PauseTotal 24.809µs, Pause:[11.027µs 13.782µs]
gc 2 last@2021-09-01 17:15:41.810728 +0800 CST, PauseTotal 24.809µs, Pause:[11.027µs 13.782µs]
45 , alloc
gc 3 last@2021-09-01 17:16:05.834971 +0800 CST, PauseTotal 48.651µs, Pause:[23.842µs 11.027µs 13.782µs]
gc 3 last@2021-09-01 17:16:05.834971 +0800 CST, PauseTotal 48.651µs, Pause:[23.842µs 11.027µs 13.782µs]
gc 3 last@2021-09-01 17:16:05.834971 +0800 CST, PauseTotal 48.651µs, Pause:[23.842µs 11.027µs 13.782µs]
46 , alloc
........
gc 3 last@2021-09-01 17:16:05.834971 +0800 CST, PauseTotal 48.651µs, Pause:[23.842µs 11.027µs 13.782µs]
gc 3 last@2021-09-01 17:16:05.834971 +0800 CST, PauseTotal 48.651µs, Pause:[23.842µs 11.027µs 13.782µs]
gc 3 last@2021-09-01 17:16:05.834971 +0800 CST, PauseTotal 48.651µs, Pause:[23.842µs 11.027µs 13.782µs]
52 , alloc
gc 4 last@2021-09-01 17:16:26.857147 +0800 CST, PauseTotal 61.613µs, Pause:[12.962µs 23.842µs 11.027µs 13.782µs]
*/
