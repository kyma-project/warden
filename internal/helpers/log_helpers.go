package helpers

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"time"
)

type logkey string

const keyLog logkey = "log"

func LoggerFromCtx(ctx context.Context) *zap.SugaredLogger {
	logger, ok := ctx.Value(keyLog).(*zap.SugaredLogger)
	if logger == nil || !ok {
		tmpLogger := zap.NewExample().Sugar().With("WARNING", "uninitialized logger from context")
		tmpLogger.Warn("couldn't find logger in context")
		return tmpLogger
	}
	return logger
}

func LogStartTime(ctx context.Context, message string) time.Time {
	logger := LoggerFromCtx(ctx)
	logger.Debugf("%s start", message)
	startTime := time.Now()
	return startTime
}

func LogEndTime(ctx context.Context, message string, startTime time.Time) {
	logger := LoggerFromCtx(ctx)
	duration := time.Since(startTime)
	logger.Debugw(fmt.Sprintf("%s end", message), "exec-time", duration)
}
