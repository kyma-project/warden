package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type notary struct {
	URL               string `yaml:"URL"`
	AllowedRegistries string `yaml:"allowedRegistries"`
}

type config struct {
	Notary notary `yaml:"notary"`
}

func Load(path string) (*config, error) {
	var config config

	sanitizedPath := filepath.Clean(path)
	yamlFile, err := os.ReadFile(sanitizedPath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(yamlFile, &config)
	return &config, err
}
