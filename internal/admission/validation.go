package admission

import (
	"context"
	"fmt"
	"github.com/kyma-project/warden/pkg"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	ValidationPath = "/validation/pods"
)

type ValidationWebhook struct {
	client  ctrlclient.Client
	decoder *admission.Decoder
}

func NewValidationWebhook(client ctrlclient.Client) *ValidationWebhook {
	return &ValidationWebhook{
		client: client,
	}
}

func (w *ValidationWebhook) Handle(_ context.Context, req admission.Request) admission.Response {
	fmt.Println(req.Name, req.Kind.String())
	if req.Resource.Resource != corev1.ResourcePods.String() {
		return admission.Errored(http.StatusBadRequest,
			errors.Errorf("Invalid request kind :%s, expected: %s", req.Resource.Resource, corev1.ResourcePods.String()))
	}

	pod := &corev1.Pod{}
	if err := w.decoder.Decode(req, pod); err != nil {
		admission.Errored(http.StatusInternalServerError, err)
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
