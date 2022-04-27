package main

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// logger, _ := zap.NewProduction(zap.AddCaller())
	// defer logger.Sync()

	// logger.Info("hello world")

	c := zap.NewProductionConfig()
	ec := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "l", //"level",
		NameKey:        "logger",
		CallerKey:      "c", //"caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.RFC3339TimeEncoder, //zapcore.EpochTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	c.OutputPaths = []string{"product.log", "stderr"}
	//c.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	c.EncoderConfig = ec
	logger1, _ := c.Build()
	defer logger1.Sync()
	logger1.Info("hello world")

}

//{"l":"info","ts":"2022-04-27T15:49:26+08:00","c":"zaplog/product.go:34","msg":"hello world"}
