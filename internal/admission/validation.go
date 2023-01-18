package admission

import (
	"context"
	"github.com/kyma-project/warden/pkg"
	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	ValidationPath = "/validation/pods"
)

type ValidationWebhook struct {
	decoder *admission.Decoder
}

func NewValidationWebhook() *ValidationWebhook {
	return &ValidationWebhook{}
}

func (w *ValidationWebhook) Handle(_ context.Context, req admission.Request) admission.Response {
	if req.Operation == admissionv1.Delete {
		return admission.Allowed("")
	}

	if req.Kind.Kind != corev1.ResourcePods.String() {
		return admission.Errored(http.StatusBadRequest,
			errors.Errorf("Invalid request kind :%s, expected: %s", req.Resource.Resource, corev1.ResourcePods.String()))
	}

	pod := &corev1.Pod{}
	if err := w.decoder.Decode(req, pod); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if pod.Labels == nil {
		return admission.Allowed("nothing to do")
	}

	if pod.Labels[pkg.PodValidationLabel] != pkg.ValidationStatusReject {
		return admission.Allowed("nothing to do")

	}

	return admission.Denied("Pod images validation failed")
}

func (w *ValidationWebhook) InjectDecoder(decoder *admission.Decoder) error {
	w.decoder = decoder
	return nil
}
