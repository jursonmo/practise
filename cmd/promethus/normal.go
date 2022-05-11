package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// 创建一个没有任何 label 标签的 gauge 指标
	temp := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "home_temperature_celsius",
		Help: "The current temperature in degrees Celsius.",
	})

	// 在默认的注册表中注册该指标
	prometheus.MustRegister(temp)

	// 设置 gauge 的值为 39
	temp.Set(39)

	// 暴露指标
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":8080", nil)
}
