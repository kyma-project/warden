package helpers

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"time"
)

func LoggerFromCtx(ctx context.Context) *zap.SugaredLogger {
	return (ctx.Value("log")).(*zap.SugaredLogger)
}

func LogStartTime(ctx context.Context, message string) time.Time {
	logger := LoggerFromCtx(ctx)
	logger.Debugf("%s start", message)
	startTime := time.Now()
	return startTime
}

func LogEndTime(ctx context.Context, message string, startTime time.Time) {
	logger := LoggerFromCtx(ctx)
	duration := time.Now().Sub(startTime)
	logger.Debugw(fmt.Sprintf("%s end", message), "exec-time", duration)
}
