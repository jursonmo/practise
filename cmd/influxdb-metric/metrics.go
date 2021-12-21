package main

import (
	metrics "github.com/tevjef/go-runtime-metrics"
)

func main() {
	err := metrics.RunCollector(metrics.DefaultConfig)

	if err != nil {
		// handle error
	}
}
