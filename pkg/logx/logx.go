package logx

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var lg *zap.SugaredLogger

func Init() {
	lvl := strings.ToLower(os.Getenv("LOG_LEVEL"))
	level := zapcore.InfoLevel

	switch lvl {
	case "debug":
		level = zapcore.DebugLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(level)
	cfg.Encoding = "json"
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	z, _ := cfg.Build()
	lg = z.Sugar()

}

func L() *zap.SugaredLogger {
	if lg == nil {
		Init()
	}
	return lg
}

func Sync() { _ = L().Sync() }
