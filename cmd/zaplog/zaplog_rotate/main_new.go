package main

import (
	"fmt"
	"path/filepath"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// 配置日志切割
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder, // 日期时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 创建一个文件切割器
	logPath := "./" // 更改为您的日志文件路径

	logFile, err := rotatelogs.New(
		logPath+"app3-%Y%m%d.log", // 文件名格式
		rotatelogs.WithLinkName(logPath+"app3.log"),
		rotatelogs.WithClock(rotatelogs.Local), //rotatelogs.Local, rotatelogs.UTC
		//rotatelogs.WithRotationCount(),
		//rotatelogs.WithMaxAge(24*time.Hour), // 保留最近2天的日志
		//WithRotationTime 跟"app3-%Y%m%d.log" 名称, 以名称定义为主。即WithRotationTime(1*time.Hour) 无效
		rotatelogs.WithRotationTime(1*time.Hour), //每小时切割一次,默认是一天一次86400
		rotatelogs.WithMaxAge(-1),
		rotatelogs.WithRotationCount(7), //如果使用了WithRotationCount，要明确设置WithMaxAge(-1)。
	)

	if err != nil {
		panic(err)
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig), // 使用JSON编码器
		zapcore.AddSync(logFile),              // 文件切割器
		zap.NewAtomicLevelAt(zap.DebugLevel),  // 最低日志级别
	)

	// 创建Logger
	logger := zap.New(core)

	// 记录日志
	// for i := 0; i < 3; i++ {
	// 	logger.Info("This is a log message.")

	// }

	for {
		logger.Info("This is a log message.")
		time.Sleep(time.Minute)
	}
	// 关闭Logger
	defer logger.Sync()
}

func filenameParse() {
	dir, filename := filepath.Split("./logdir/app.x.log")
	fmt.Printf("dir:%s, file:%s\n", dir, filename)
	suffix := filepath.Ext(filename)
	fmt.Printf("suffix:%s\n", suffix)
	var fileprefix string
	if suffix == "" {
		suffix = ".log"
		fileprefix = filename
	} else {
		//fileprefix, err := strings.TrimSuffix(filenameall, filesuffix)
		fileprefix = filename[0 : len(filename)-len(suffix)]
	}
	fmt.Printf("fileprefix:%s\n", fileprefix)
	logxx := dir + fileprefix + "_%Y%m%d" + suffix
	fmt.Printf("logxx:%s\n", logxx)
	/*
		dir:./logdir/, file:app.x.log
		suffix:.log
		fileprefix:app.x
		logxx:./logdir/app.x_%Y%m%d.log
	*/
	return
}
