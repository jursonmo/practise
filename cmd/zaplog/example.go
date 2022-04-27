package main

import (
	"time"

	"go.uber.org/zap"
)

func main() {
	logger := zap.NewExample()
	defer logger.Sync()

	logger.Info("tracked some metrics",
		zap.Namespace("metrics"),
		zap.Int("counter", 1),
	)

	logger.Info("failed to fetch URL",
		zap.String("url", "baidu.com"),
		zap.Int("attempt", 3),
		zap.Duration("backoff", time.Second),
	)

	logger2 := logger.With(
		zap.Namespace("metrics"),
		zap.Int("counter", 1),
	)
	logger2.Info("tracked some metrics")
	logger2.Sugar().Infof("test format:%s", "test")

}

/*
{"level":"info","msg":"tracked some metrics","metrics":{"counter":1}}
{"level":"info","msg":"failed to fetch URL","url":"baidu.com","attempt":3,"backoff":"1s"}
{"level":"info","msg":"tracked some metrics","metrics":{"counter":1}}
{"level":"info","msg":"test format:test","metrics":{"counter":1}}
*/
