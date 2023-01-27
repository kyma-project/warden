package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("Load test config from absolute path", func(t *testing.T) {
		wd, err := os.Getwd()
		require.NoError(t, err)
		path := filepath.Join(wd, "testData", "config.yaml")

		cfg, err := Load(path)
		require.NoError(t, err)
		require.Empty(t, cfg.Notary.AllowedRegistries)
		require.NotEmpty(t, cfg.Notary.URL)
	})

	t.Run("Load test config from relative path", func(t *testing.T) {
		path := filepath.Join(".", "testData", "config.yaml")

		cfg, err := Load(path)
		require.NoError(t, err)
		require.Empty(t, cfg.Notary.AllowedRegistries)
		require.NotEmpty(t, cfg.Notary.URL)
	})

	t.Run("Load test config error", func(t *testing.T) {
		path := filepath.Join("this", "path", "doesnot.exist")

		cfg, err := Load(path)
		require.Error(t, err)
		require.Nil(t, cfg)
	})
}
