package log

import (
	"os"
	"time"

	golog "log"

	"github.com/alert666/alertmanager-agent/base/conf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger() {
	var (
		encoder  zapcore.Encoder
		writer   zapcore.WriteSyncer
		logLevel zapcore.Level
	)
	timeZone := conf.GetTimeZone()
	cst, err := time.LoadLocation(timeZone)
	if err != nil {
		golog.Printf("failed to load location %s: %v, use local time instead", timeZone, err)
		cst = time.Local
	}
	// 修改全局时区
	time.Local = cst

	logLevelStr := conf.GetLogLevel()
	logEncoder := conf.GetLogEncoder()

	config := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		EncodeDuration: zapcore.SecondsDurationEncoder,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	switch logEncoder {
	case "json":
		encoder = zapcore.NewJSONEncoder(config)
	case "console":
		config.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(config)
	default:
		config.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(config)
	}

	writer = zapcore.AddSync(os.Stdout)

	switch logLevelStr {
	case "debug":
		logLevel = zap.DebugLevel
	case "info":
		logLevel = zap.InfoLevel
	case "warn":
		logLevel = zap.WarnLevel
	case "error":
		logLevel = zap.ErrorLevel
	default:
		logLevel = zap.InfoLevel
	}

	core := zapcore.NewCore(encoder, writer, logLevel)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.FatalLevel))
	zap.ReplaceGlobals(logger)
	zap.L().Info("log initialization successful", zap.String("level", logLevelStr))
}
