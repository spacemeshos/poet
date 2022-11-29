package logging

import (
	"context"

	"github.com/spacemeshos/smutil/log"
)

type LoggerKey struct{}

func NewContext(ctx context.Context, logger log.Log) context.Context {
	return context.WithValue(ctx, LoggerKey{}, logger)
}

func FromContext(ctx context.Context) log.Log {
	if logger, ok := ctx.Value(LoggerKey{}).(log.Log); ok {
		return logger
	}
	return log.AppLog
}
