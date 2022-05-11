package main

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

//https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Histogram
func main() {
	temps := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "pond_temperature_celsius",
		Help:    "The temperature of the frog pond.", // Sorry, we can't measure how badly it smells.
		Buckets: prometheus.LinearBuckets(20, 5, 5),  // 5 buckets, each 5 centigrade wide.
	})
	fmt.Printf("buckets:%v\n", prometheus.LinearBuckets(20, 5, 5))
	// Simulate some observations.
	// for i := 0; i < 1000; i++ {
	// 	temps.Observe(30 + math.Floor(120*math.Sin(float64(i)*0.1))/10)
	// }
	for i := 0; i < 20; i++ {
		temps.Observe(float64(30 + i))
	}
	// Just for demonstration, let's check the state of the histogram by
	// (ab)using its Write method (which is usually only used by Prometheus
	// internally).
	metric := &dto.Metric{}
	temps.Write(metric)
	fmt.Println(proto.MarshalTextString(metric))

}

/*
buckets:[20 25 30 35 40]
histogram: <
  sample_count: 20
  sample_sum: 790
  bucket: <
    cumulative_count: 0
    upper_bound: 20  //上限值
  >
  bucket: <
    cumulative_count: 0
    upper_bound: 25
  >
  bucket: <
    cumulative_count: 1
    upper_bound: 30
  >
  bucket: <
    cumulative_count: 6
    upper_bound: 35
  >
  bucket: <
    cumulative_count: 11
    upper_bound: 40
  >
>
*/
