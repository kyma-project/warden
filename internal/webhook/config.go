package webhook

type Config struct {
	SystemNamespace string `envconfig:"default=default"`
	ServiceName     string `envconfig:"default=warden-admission"`
	SecretName      string `envconfig:"default=warden-admission-cert"`
	Port            int    `envconfig:"default=8443"`
	ConfigPath      string `envconfig:"default=/appdata/config.yaml"`
}
