package main

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/zero-contrib/logx/zapx"
	"go.uber.org/zap"
)

// go-zero/core/logx/readme-cn.md 查看使用说明
func main() {
	var xx logx.Logger
	_ = xx
	logx.Info("first")

	ctx := context.Background() //ctx 可以携带traceID， SpanID， 或者多个LogField, 通过WithFields(ctx, fields ...LogField)添加
	ctx = logx.WithFields(ctx, []logx.LogField{{Key: "k1", Value: "v1"}, {Key: "k2", Value: "v2"}}...)

	logc.Info(ctx, "info message")
	logc.Errorf(ctx, "error message: %d", 123)
	logc.Debugw(ctx, "info filed", logc.Field("key", "value"))
	logc.Slowv(ctx, "object")

	log2 := logx.WithContext(ctx)
	log2.WithFields([]logx.LogField{{Key: "logName", Value: "log2"}}...)
	log2.Info("info message")
	log2.Errorf("error message: %d", 123)

	exec()

	//----------
	flog := NewFieldLogx(nil, []logx.LogField{{Key: "module", Value: "udpx"}})
	_ = flog
	flog.Infof("flog start with %d", 123)

	writer2, _ := zapx.NewZapWriter(zap.WithCaller(false)) //zap.WithCaller(false) 禁用caller, 避免日志里打印两个caller
	logx.SetWriter(writer2)
	flog2 := NewFieldLogx(logx.WithContext(context.Background()), []logx.LogField{{Key: "module", Value: "udpx"}})
	flog2.Infof("flog2 with zap start with %d", 123)

}
func exec() error {
	logx.WithCallerSkip(1).Info("exec info")
	return nil
}

/*
{"@timestamp":"2024-01-05T18:39:22.434+08:00","caller":"logx/logx.go:17","content":"first","level":"info"}
{"@timestamp":"2024-01-05T18:39:22.434+08:00","caller":"logx/logx.go:22","content":"info message","k1":"v1","k2":"v2","level":"info"}
{"@timestamp":"2024-01-05T18:39:22.434+08:00","caller":"logx/logx.go:23","content":"error message: 123","k1":"v1","k2":"v2","level":"error"}
{"@timestamp":"2024-01-05T18:39:22.434+08:00","caller":"logx/logx.go:24","content":"info filed","k1":"v1","k2":"v2","key":"value","level":"debug"}
{"@timestamp":"2024-01-05T18:39:22.434+08:00","caller":"logx/logx.go:25","content":"object","k1":"v1","k2":"v2","level":"slow"}
{"@timestamp":"2024-01-05T18:39:22.434+08:00","caller":"logx/logx.go:29","content":"info message","k1":"v1","k2":"v2","level":"info","logName":"log2"}
{"@timestamp":"2024-01-05T18:39:22.434+08:00","caller":"logx/logx.go:30","content":"error message: 123","k1":"v1","k2":"v2","level":"error","logName":"log2"}
{"@timestamp":"2024-01-05T18:39:22.434+08:00","caller":"logx/logx.go:32","content":"exec info","level":"info"}
{"@timestamp":"2024-01-05T18:39:22.434+08:00","caller":"logx/logx.go:95","content":"flog start with 123","level":"info","module":"udpx"}
{"level":"info","ts":1704451162.434636,"msg":"flog2 with zap start with 123","module":"udpx","caller":"logx/logx.go:95"}
*/

// udpx 定义的日志库Logger, 把mvnet logx 实现这个Logger， 并且传递给udpx 用于打印
type Logger interface {
	Fatalf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Debugf(format string, v ...interface{})
}
type FieldLogx struct {
	logger logx.Logger
}

func NewFieldLogx(logger logx.Logger, fields []logx.LogField) Logger {
	var l logx.Logger
	if logger != nil {
		l = logger.WithFields(fields...)
	} else {
		l = logx.WithContext(nil).WithFields(fields...)
	}
	return &FieldLogx{logger: l}
}

func (l *FieldLogx) Fatalf(format string, v ...interface{}) {
	//TODO
	panic(fmt.Sprintf(format, v...))
}
func (l *FieldLogx) Errorf(format string, v ...interface{}) {
	l.logger.Errorf(format, v...)
}
func (l *FieldLogx) Warnf(format string, v ...interface{}) {
	l.logger.Infof(format, v...)
}
func (l *FieldLogx) Infof(format string, v ...interface{}) {
	l.logger.Infof(format, v...)
}
func (l *FieldLogx) Debugf(format string, v ...interface{}) {
	l.logger.Debugf(format, v...)
}

//-----------------------------

type FieldLog struct {
	fields []logx.LogField
}

func (l *FieldLog) Fatalf(format string, v ...interface{}) {
	//TODO
	panic(fmt.Sprintf(format, v...))
}
func (l *FieldLog) Errorf(format string, v ...interface{}) {
	//这里有性能问题，再判断日志级别前就fmt.Sprintf(format, v...)
	logx.Errorw(fmt.Sprintf(format, v...), l.fields...)
}
func (l *FieldLog) Warnf(format string, v ...interface{}) {
	//这里有性能问题，再判断日志级别前就fmt.Sprintf(format, v...)
	logx.Infow(fmt.Sprintf(format, v...), l.fields...)
}
func (l *FieldLog) Infof(format string, v ...interface{}) {
	logx.Infow(fmt.Sprintf(format, v...), l.fields...)
}
func (l *FieldLog) Debugf(format string, v ...interface{}) {
	logx.Debugw(fmt.Sprintf(format, v...), l.fields...)
}

// ------------------------
type CtxLog struct {
	ctx context.Context
}

func (l *CtxLog) Fatalf(format string, v ...interface{}) {
	//TODO
	panic(fmt.Sprintf(format, v...))
}
func (l *CtxLog) Errorf(format string, v ...interface{}) {
	logc.Errorf(l.ctx, format, v...)
}
func (l *CtxLog) Warnf(format string, v ...interface{}) {
	logc.Infof(l.ctx, format, v...)
}
func (l *CtxLog) Infof(format string, v ...interface{}) {
	logc.Infof(l.ctx, format, v...)
}
func (l *CtxLog) Debugf(format string, v ...interface{}) {
	logc.Debugf(l.ctx, format, v...)
}

// ----------------
