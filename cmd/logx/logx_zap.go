package main

import (
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/zero-contrib/logx/zapx"
	"go.uber.org/zap"
)

func main() {

	writer, err := zapx.NewZapWriter()
	logx.Must(err)
	logx.SetWriter(writer)

	logx.Infow("infow foo",
		logx.Field("url", "http://localhost:8080/hello"),
		logx.Field("attempt", 3),
		logx.Field("backoff", time.Second),
	)

	logx.WithDuration(1100*time.Microsecond).Infow("infow withduration",
		logx.Field("url", "localhost:8080/hello"),
		logx.Field("attempt", 3),
		logx.Field("backoff", time.Second),
	)

	logx.Infof("xxxxxx:%d", 123)
	//{"level":"info","ts":1704423738.4924748,"caller":"logx/logx2.go:28","msg":"xxxxxx:123","caller":"logx/logx2.go:28"}
	//出现两个caller, 前面caller是zap自动加的， 最后这个caller 是logx 加的，

	writer2, _ := zapx.NewZapWriter(zap.WithCaller(false)) //zap.WithCaller(false) 禁用caller
	logx.SetWriter(writer2)
	logx.Infof("yyyyyyy:%d", 123) //{"level":"info","ts":1704427234.035352,"msg":"yyyyyyy:123","caller":"logx/logx_zap.go:35"} 只剩下logx的caller了。
}

/*
{"level":"info","ts":1704423738.492178,"caller":"logx/logx2.go:16","msg":"infow foo","url":"http://localhost:8080/hello","attempt":3,"backoff":"1s","caller":"logx/logx2.go:16"}
{"level":"info","ts":1704423738.492447,"caller":"logx/logx2.go:22","msg":"infow withduration","duration":"1.1ms","url":"localhost:8080/hello","attempt":3,"backoff":"1s","caller":"logx/logx2.go:22"}
{"level":"info","ts":1704423738.4924748,"caller":"logx/logx2.go:28","msg":"xxxxxx:123","caller":"logx/logx2.go:28"}

{"level":"info","ts":1704427234.035352,"msg":"yyyyyyy:123","caller":"logx/logx_zap.go:35"}
*/
