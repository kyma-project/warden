package admission

import (
	"context"
	"github.com/kyma-project/warden/internal/helpers"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"time"
)

const finalizerKey = "finalizer"

type Handler func(ctx context.Context, req admission.Request) admission.Response

func HandleWithLogger(baseLogger *zap.SugaredLogger, handler Handler) Handler {
	return func(ctx context.Context, req admission.Request) admission.Response {
		loggerWithReqId := baseLogger.With("req-id", req.UID).
			With("namespace", req.Namespace).
			With("name", req.Name)
		ctxLogger := helpers.LoggerToContext(ctx, loggerWithReqId)

		resp := handler(ctxLogger, req)
		return resp
	}
}

func HandlerWithTimeMeasure(handler Handler) Handler {
	return func(ctx context.Context, req admission.Request) admission.Response {
		logger := helpers.LoggerFromCtx(ctx)
		logger.Debug("request handling started")
		startTime := time.Now()
		defer func(startTime time.Time) {
			helpers.LogEndTime(ctx, "request handling finished", startTime)
		}(startTime)

		resp := handler(ctx, req)
		return resp
	}
}

type TimeoutHandler func(ctx context.Context, err error, req admission.Request) admission.Response

func HandleWithTimeout(timeout time.Duration, handler Handler, timeoutHandler TimeoutHandler) Handler {
	return func(ctx context.Context, req admission.Request) admission.Response {
		ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		var resp admission.Response
		done := make(chan bool, 1)
		go func() {
			defer close(done)
			resp = handler(ctxTimeout, req)
			done <- true
		}()

		select {
		case <-done:
		case <-ctxTimeout.Done():
			if err := ctxTimeout.Err(); err != nil {
				return timeoutHandler(ctxTimeout, err, req)
			}
		}
		return resp
	}
}
