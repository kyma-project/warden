package admission

import (
	"context"
	"fmt"
	"github.com/kyma-project/warden/internal/helpers"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"time"
)

type Handler func(ctx context.Context, req admission.Request) admission.Response

func HandleWithLogger(baseLogger *zap.SugaredLogger, handler Handler) Handler {
	return func(ctx context.Context, req admission.Request) admission.Response {
		loggerWithReqId := baseLogger.With("req-id", req.UID).
			With("resource-name", fmt.Sprintf("%s/%s", req.Namespace, req.Name))
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

func HandleWithTimeout(timeout time.Duration, handler Handler) Handler {
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
				helpers.LoggerFromCtx(ctx).Infof("request exceeded desired timeout: %s", timeout.String())
				return admission.Errored(http.StatusRequestTimeout, errors.Wrapf(err, "request exceeded desired timeout: %s", timeout.String()))
			}
		}
		return resp
	}
}
