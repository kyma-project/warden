package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type notary struct {
	URL               string        `yaml:"URL"`
	Timeout           time.Duration `yaml:"timeout"`
	AllowedRegistries string        `yaml:"allowedRegistries"`
}

type admission struct {
	SystemNamespace string        `yaml:"systemNamespace"`
	ServiceName     string        `yaml:"serviceName"`
	SecretName      string        `yaml:"secretName"`
	Timeout         time.Duration `yaml:"timeout"`
	Port            int           `yaml:"port"`
}

type operator struct {
	MetricsBindAddress     string `yaml:"metricsBindAddress"`
	HealthProbeBindAddress string `yaml:"healthProbeBindAddress"`
	LeaderElect            bool   `yaml:"leaderElect"`
}

type config struct {
	Notary    notary    `yaml:"notary"`
	Admission admission `yaml:"admission"`
	Operator  operator  `yaml:"operator"`
}

func Load(path string) (*config, error) {
	config := defaultConfig()

	sanitizedPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(sanitizedPath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(yamlFile, config)
	return config, err
}

func defaultConfig() *config {
	return &config{
		Notary: notary{
			URL:     "https://signing-dev.repositories.cloud.sap",
			Timeout: time.Second * 30,
		},
		Admission: admission{
			SystemNamespace: "default",
			ServiceName:     "warden-admission",
			SecretName:      "warden-admission-cert",
			Port:            8443,
			Timeout:         time.Second * 2,
		},
		Operator: operator{
			MetricsBindAddress:     ":8080",
			HealthProbeBindAddress: ":8081",
			LeaderElect:            false,
		},
	}
}
