package main

import (
	"time"

	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
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
		logPath+"app2-%Y%m%d.log", // 文件名格式
		rotatelogs.WithLinkName(logPath+"app2.log"),
		rotatelogs.WithClock(rotatelogs.Local), //rotatelogs.Local, rotatelogs.UTC
		rotatelogs.WithMaxAge(24*time.Hour),    // 保留最近2天的日志
		// rotatelogs.WithRotationTime(24*time.Hour), // 每24小时切割一次
	)

	// logfileName := "app.log"
	// var cstSh, _ = time.LoadLocation("Asia/Shanghai") //上海
	// fileSuffix := time.Now().In(cstSh).Format("2006-01-02") + logfileName
	// fmt.Println(fileSuffix)

	// logFile, err := rotatelogs.New(
	// 	logPath+"-"+fileSuffix,
	// 	rotatelogs.WithLinkName(logPath+logfileName), // 生成软链，指向最新日志文件
	// 	rotatelogs.WithRotationCount(2),              // 文件最大保存份数
	// 	rotatelogs.WithRotationTime(time.Hour*24),    // 日志切割时间间隔
	// )

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
