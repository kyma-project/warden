package defaulting

import (
	"context"
	"fmt"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	WebhookPath = "/defaulting/pods"
)

type DefaultingWebHook struct {
	client  ctrlclient.Client
	decoder *admission.Decoder
}

func NewWebhook(client ctrlclient.Client) *DefaultingWebHook {
	return &DefaultingWebHook{
		client: client,
	}
}

func (w *DefaultingWebHook) Handle(_ context.Context, req admission.Request) admission.Response {
	fmt.Println(req.Name, req.Kind.String())
	return admission.Allowed("nothing to do")
}

func (w *DefaultingWebHook) InjectDecoder(decoder *admission.Decoder) error {
	w.decoder = decoder
	return nil
}
