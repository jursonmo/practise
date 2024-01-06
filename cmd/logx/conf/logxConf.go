package main

import (
	"encoding/json"
	"os"

	"github.com/zeromicro/go-zero/core/logx"
	"go.uber.org/zap"
)

func parseJSONConfig(config *logx.LogConf, path string) error {
	file, err := os.Open(path) // For read access.
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(config)
}

func main() {
	var c logx.LogConf
	// file, err := os.Open("./logxConf.json")
	// if err != nil {
	// 	panic(err)
	// }
	// de := json.NewDecoder(file)
	// de.Decode(&c)
	// fmt.Printf("%#v\n", c)
	err := parseJSONConfig(&c, "./logxConf.json")
	if err != nil {
		panic(err)
	}

	if err := logx.SetUp(c); err != nil {
		panic(err)
	}
	//如果使用了zap Writer, 就无法实现配置里的日志切割等功能，
	//日志切割的功能是在fw:= newFileWriter(c)里实现，实现完后logx.SetWriter(fw)
	//也就是如果这里设置了logx.SetWriter(zapwriter)，只能在zapwriter里实现日志切割等功能
	//zapwriter, err := zapx.NewZapWriter(zap.WithCaller(false)) //zap.WithCaller(false) 禁用caller, 避免日志里打印两个caller
	zapwriter, err := NewZapWriter(c.Path+".log", zap.WithCaller(false))
	if err != nil {
		panic(err)
	}
	logx.SetWriter(zapwriter)

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
