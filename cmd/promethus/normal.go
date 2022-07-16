package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	lables := make(map[string]string)
	lables["dc"] = "dc1"
	lables["hostname"] = "mygoapp-1-host"
	lables["service"] = "mygoapp-1-service"
	// 创建一个没有任何 label 标签的 gauge 指标
	temp := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "home_temperature_celsius",
		Help:        "The current temperature in degrees Celsius.",
		ConstLabels: prometheus.Labels(lables),
	})

	// 在默认的注册表中注册该指标
	prometheus.MustRegister(temp)

	// 设置 gauge 的值为 39
	go func() {
		for {
			n := float64(30 + rand.Intn(10))
			temp.Set(n)
			time.Sleep(time.Second)
			fmt.Println(n)
		}
	}()
	// 暴露指标
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":1313", nil)
}

/*
go_threads 6
# HELP home_temperature_celsius The current temperature in degrees Celsius.
# TYPE home_temperature_celsius gauge
home_temperature_celsius{dc="dc1",hostname="mygoapp-1-host",service="mygoapp-1-service"} 34
# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 9.27
*/
