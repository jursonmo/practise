package main

import (
	"encoding/json"

	"go.uber.org/zap"
)

// logger 就是根据配置来Build()--> zap.NewCore -->logger 生成出来的，
// zap.NewProduction() 最终也是生成 配置再Build: NewProductionConfig().Build(options...)  --> NewCore --> 生成logger

func main() {
	rawJSON := []byte(`{
	  "level":"debug",
	  "encoding":"json",
	  "outputPaths": ["stdout", "server.log"],
	  "errorOutputPaths": ["stderr"],
	  "initialFields":{"name":"dj"},
	  "encoderConfig": {
		"messageKey": "message",
		"levelKey": "level",
		"levelEncoder": "lowercase"
	  }
	}`)

	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}
	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	logger.Info("server start work successfully!")
}

//{"level":"info","message":"server start work successfully!","name":"dj"}
