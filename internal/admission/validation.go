package admission

import (
	"context"
	"github.com/kyma-project/warden/pkg"
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

func (w *ValidationWebhook) handle(_ context.Context, req admission.Request) admission.Response {
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

	if pod.Annotations[pkg.PodValidationRejectAnnotation] != pkg.ValidationReject {
		return admission.Allowed("nothing to do")
	}

	return admission.Denied("Pod images validation failed")
}

func (w *ValidationWebhook) InjectDecoder(decoder *admission.Decoder) error {
	w.decoder = decoder
	return nil
}
