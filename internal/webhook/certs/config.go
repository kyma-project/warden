package certs

type WebhookConfig struct {
	CABundel         []byte
	ServiceName      string
	ServiceNamespace string
}
