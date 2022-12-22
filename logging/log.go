package logging

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LoggerKey struct{}

func NewContext(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, LoggerKey{}, logger)
}

func FromContext(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(LoggerKey{}).(*zap.Logger); ok {
		return logger
	}
	return New(zap.DebugLevel, "", false)
}

func New(level zapcore.LevelEnabler, logFileName string, json bool) *zap.Logger {
	var encoder zapcore.Encoder
	if json {
		encoder = zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig())
	} else {
		encoder = zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	}

	consoleSyncer := zapcore.Lock(os.Stdout)
	var cores []zapcore.Core
	cores = append(cores, zapcore.NewCore(encoder, consoleSyncer, level))

	if logFileName != "" {
		fileLogger := &lumberjack.Logger{
			Filename: logFileName,
			MaxSize:  500,
			MaxAge:   28,
			Compress: true,
		}
		wr := fileLogger
		fs := zapcore.AddSync(wr)
		cores = append(cores, zapcore.NewCore(encoder, fs, zap.DebugLevel))
	}

	return zap.New(zapcore.NewTee(cores...))
}
