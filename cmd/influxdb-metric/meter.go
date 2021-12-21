package main

import (
	"log"
	"os"
	"time"

	"github.com/rcrowley/go-metrics"
)

func main() {

	m := metrics.NewMeter()
	metrics.Register("quux", m)
	m.Mark(1)

	go metrics.Log(metrics.DefaultRegistry,
		1*time.Second,
		log.New(os.Stdout, "metrics: ", log.Lmicroseconds))

	var j int64
	j = 1
	time.Sleep(time.Millisecond * 500)
	for true {
		time.Sleep(time.Second * 1)
		//j++
		m.Mark(j)
	}

}
