package webhook

type Config struct {
	SystemNamespace string `envconfig:"default=default"`
	ServiceName     string `envconfig:"default=warden-webhook"`
	SecretName      string `envconfig:"default=warden-webhook"`
	Port            int    `envconfig:"default=8443"`
	ConfigPath      string `envconfig:"default=/appdata/config.yaml"`
}
