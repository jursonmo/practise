package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/zero-contrib/logx/zapx"
	"go.uber.org/zap"
)

func main() {
	var c logx.LogConf
	file, err := os.Open("./logxConf.json")
	if err != nil {
		panic(err)
	}
	de := json.NewDecoder(file)
	de.Decode(&c)
	fmt.Printf("%#v\n", c)

	if err := logx.SetUp(c); err != nil {
		panic(err)
	}

	writer, err := zapx.NewZapWriter(zap.WithCaller(false)) //zap.WithCaller(false) 禁用caller, 避免日志里打印两个caller
	if err != nil {
		panic(err)
	}
	logx.SetWriter(writer)

	logx.Info("info")
	logx.Infof("infof:%v", 123)
	logx.Infow("infof", []logx.LogField{{"key1", "value1"}, {"key2", "value2"}}...)
}

/*

logx.LogConf{ServiceName:"xxxServiceName", Mode:"file", Encoding:"", TimeFormat:"", Path:"logd", Level:"info", MaxContentLength:0x0, Compress:false, Stat:false, KeepDays:3, StackCooldownMillis:0, MaxBackups:0, MaxSize:0, Rotation:""}
{"level":"info","ts":1704453618.741898,"msg":"info","caller":"conf/logxConf.go:33"}
{"level":"info","ts":1704453618.741963,"msg":"infof:123","caller":"conf/logxConf.go:34"}
{"level":"info","ts":1704453618.741975,"msg":"infof","key1":"value1","key2":"value2","caller":"conf/logxConf.go:35"}
*/
