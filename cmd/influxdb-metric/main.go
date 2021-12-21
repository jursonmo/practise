package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/google/gops/agent"
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"

	//"github.com/rcrowley/go-metrics/exp"
	influxdb "github.com/vrischmann/go-metrics-influxdb"
)

func main() {

	go func() {
		gnum := metrics.NewGauge()
		metrics.Register("goroutinenum", gnum)
		gctime := metrics.NewGauge()
		metrics.Register("gctime", gctime)
		gcCounter := metrics.NewCounter()
		metrics.Register("gc", gcCounter)

		go influxdb.InfluxDBWithTags(metrics.DefaultRegistry,
			5*time.Second,
			"http://127.0.0.1:8086",
			"gostatus", // 这时数据库名称，手动创建
			"measure1", //measurement 会自动创建，但是数据库不会自动创建
			"",         //userame
			"",         //password
			map[string]string{"dev": "shenzhen_ec"},
			true,
		)
		for {
			time.Sleep(time.Second)
			gnum.Update(int64(runtime.NumGoroutine() + rand.Intn(3)))
			gctime.Update(int64(5 + rand.Intn(5)))

			if gcCounter.Count() > 30 {
				//gcCounter.Inc(int64(0))
			} else {
				gcCounter.Inc(int64(5))
			}

		}
	}()

	go func() {
		r := metrics.NewRegistry()
		gnum := metrics.NewGauge()
		r.Register("goroutinenum", gnum)

		gcNum := metrics.NewGauge()
		r.Register("gcNum", gcNum)
		gcPause := metrics.NewGauge()
		r.Register("gcPause", gcPause)

		go influxdb.InfluxDBWithTags(r,
			5*time.Second,
			"http://127.0.0.1:8086",
			"gostatus",
			"measure1",
			"", //userame
			"", //password
			map[string]string{"dev": "shanghai-ec"},
			true,
		)
		s := debug.GCStats{}
		memStats := &runtime.MemStats{}
		gcNumOld := int64(0)
		data := make([]int, 128)
		for {
			time.Sleep(time.Second)
			gnum.Update(int64(runtime.NumGoroutine() + rand.Intn(6)))
			debug.ReadGCStats(&s)
			if gcNumOld < s.NumGC && len(s.Pause) > 0 {
				//i := s.NumGC % int64(len(s.Pause))
				//pause := s.Pause[int(i)]
				pause := s.Pause[0]
				gcPause.Update(pause.Microseconds())
				fmt.Printf("pause.Microseconds()=%d\n", pause.Microseconds())
				gcNum.Update(s.NumGC)
				gcNumOld = s.NumGC

				//发生gc 后再读取memstat, 没必要定期读取memstat, 它会stop the world
				//或者gc 时间超过某个阈值后，再定期	读取memstat
				runtime.ReadMemStats(memStats)
				fmt.Println(memStats)
			} else {
				data = make([]int, len(data)*2)
			}
			fmt.Printf("data len :%d\n", len(data))
			fmt.Printf("gc %d last@%v, PauseTotal %v, Pause:%+v, PauseEnd:%+v\n", s.NumGC, s.LastGC, s.PauseTotal, s.Pause, s.PauseEnd)
		}
	}()

	//是把
	er := metrics.NewRegistry()
	age := metrics.NewGauge()
	er.Register("age", age)
	age.Update(18)
	//这是把age metrics 加到 expvar, 这样就可以通过 "http://localhost:1818/debug/metrics" 看到expvar 默认的信息外，还看到age
	exp.Exp(er)
	http.ListenAndServe(":1818", nil) //http://localhost:1818/debug/metrics  会显示出来expvar 默认的信息外，还有goroutinenum: x, 信息，即metrics里注册的信息就加入了expvar

	// go influxdb.InfluxDB(metrics.DefaultRegistry,
	// 	time.Second,
	// 	"http://127.0.0.1:8086",
	// 	"gostatus",
	// 	"goroutine-measure",
	// 	"", //userame
	// 	"", //password
	// 	true,
	// )

	if err := agent.Listen(agent.Options{Addr: "0.0.0.0:7878", ShutdownCleanup: true}); err != nil {
		log.Fatal(err)
	}
	for {
		time.Sleep(time.Second * 100)
	}
}
