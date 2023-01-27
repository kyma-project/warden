package webhook

import "time"

type Config struct {
	SystemNamespace string        `envconfig:"default=default"`
	ServiceName     string        `envconfig:"default=warden-admission"`
	SecretName      string        `envconfig:"default=warden-admission-cert"`
	Port            int           `envconfig:"default=8443"`
	ConfigPath      string        `envconfig:"default=./hack/config.yaml"`
	NotaryTimeout   time.Duration `envconfig:"default=30s"`
	Timeout         time.Duration `envconfig:"default=2s"`
}
