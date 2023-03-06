package admission

import (
	"context"
	"github.com/kyma-project/warden/internal/helpers"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	ValidationPath = "/validation/pods"
)

type ValidationWebhook struct {
	decoder    *admission.Decoder
	baseLogger *zap.SugaredLogger
}

func NewValidationWebhook(logger *zap.SugaredLogger) *ValidationWebhook {
	return &ValidationWebhook{
		baseLogger: logger,
	}
}

func (w *ValidationWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	return HandleWithLogger(w.baseLogger,
		HandlerWithTimeMeasure(w.handle))(ctx, req)
}

func (w *ValidationWebhook) handle(ctx context.Context, req admission.Request) admission.Response {
	logger := helpers.LoggerFromCtx(ctx)
	if req.Operation == admissionv1.Delete {
		return admission.Allowed("")
	}

	if req.Kind.Kind != PodType {
		return admission.Errored(http.StatusBadRequest,
			errors.Errorf("Invalid request kind: %s, expected: %s", req.Kind.Kind, PodType))
	}

	pod := &corev1.Pod{}
	if err := w.decoder.Decode(req, pod); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if pod.Annotations == nil {
		return admission.Allowed("nothing to do")
	}

	if pod.Annotations[PodValidationRejectAnnotation] != ValidationReject {
		return admission.Allowed("nothing to do")
	}

	logger.Info("Pod images validation failed")
	return admission.Denied("Pod images validation failed")
}

func (w *ValidationWebhook) InjectDecoder(decoder *admission.Decoder) error {
	w.decoder = decoder
	return nil
}
