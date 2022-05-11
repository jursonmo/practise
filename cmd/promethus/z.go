package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// 创建一个自定义的注册表
	registry := prometheus.NewRegistry()
	// 可选: 添加 process 和 Go 运行时指标到我们自定义的注册表中
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	registry.MustRegister(prometheus.NewGoCollector())

	// 创建一个简单呃 gauge 指标。
	temp := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "home_temperature_celsius",
		Help: "The current temperature in degrees Celsius.",
	})

	// 使用我们自定义的注册表注册 gauge
	registry.MustRegister(temp)

	// 设置 gague 的值为 39
	temp.Set(39)

	// 暴露自定义指标
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{Registry: registry}))
	http.ListenAndServe(":8080", nil)
}
