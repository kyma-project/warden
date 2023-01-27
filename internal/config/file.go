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

type config struct {
	Notary  notary        `yaml:"notary"`
	Timeout time.Duration `yaml:"timeout"`
}

func Load(path string) (*config, error) {
	var config config

	sanitizedPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(sanitizedPath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(yamlFile, &config)
	return &config, err
}
