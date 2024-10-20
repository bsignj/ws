package log

import (
	"time"

	"go.uber.org/zap"
)

var ZLogger *zap.Logger

func NewLogger(logLevel, outputPath, errOutputPath string) *zap.Logger {
	zapLogLevel := zap.NewAtomicLevelAt(zap.DebugLevel)

	switch logLevel {
	case "DEBUG":
		zapLogLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "INFO":
		zapLogLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "WARN":
		zapLogLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "ERROR":
		zapLogLevel = zap.NewAtomicLevelAt(zap.ErrorLevel)
	case "PANIC":
		zapLogLevel = zap.NewAtomicLevelAt(zap.PanicLevel)
	case "FATAL":
		zapLogLevel = zap.NewAtomicLevelAt(zap.FatalLevel)
	}

	if outputPath == "file" {
		outputPath = time.Now().Format("2006-01-02") + ".log"
	}

	var cfg = zap.Config{
		Level:       zapLogLevel,
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{outputPath},
		ErrorOutputPaths: []string{errOutputPath},
	}
	zapLogger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	ZLogger = zapLogger
	return ZLogger
}
