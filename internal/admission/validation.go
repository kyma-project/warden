package admission

import (
	"context"
	"fmt"
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
	return admission.Allowed("nothing to do")
}

func (w *ValidationWebhook) InjectDecoder(decoder *admission.Decoder) error {
	w.decoder = decoder
	return nil
}
