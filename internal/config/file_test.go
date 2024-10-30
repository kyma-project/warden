package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testURL                             = "https://signing-dev.repositories.cloud.sap"
	testAllowedRegistries               = "test1,\ntest2,\ntest3"
	testPredefinedUserAllowedRegistries = "user1,\nuser2"
)

func TestLoad(t *testing.T) {
	t.Run("Load test config from absolute path", func(t *testing.T) {
		wd, err := os.Getwd()
		require.NoError(t, err)
		path := filepath.Join(wd, "testData", "config.yaml")

		cfg, err := Load(path)
		require.NoError(t, err)
		require.Equal(t, testAllowedRegistries, cfg.Notary.AllowedRegistries)
		require.Equal(t, testPredefinedUserAllowedRegistries, cfg.Notary.PredefinedUserAllowedRegistries)
		require.Equal(t, testURL, cfg.Notary.URL)
	})

	t.Run("Load test config from relative path", func(t *testing.T) {
		path := filepath.Join(".", "testData", "config.yaml")

		cfg, err := Load(path)
		require.NoError(t, err)
		require.Equal(t, testAllowedRegistries, cfg.Notary.AllowedRegistries)
		require.Equal(t, testPredefinedUserAllowedRegistries, cfg.Notary.PredefinedUserAllowedRegistries)
		require.Equal(t, testURL, cfg.Notary.URL)
	})

	t.Run("Path does not exist error", func(t *testing.T) {
		path := filepath.Join("this", "path", "doesnot.exist")

		cfg, err := Load(path)
		require.Error(t, err)
		require.Nil(t, cfg)
	})

	t.Run("Empty path error", func(t *testing.T) {
		cfg, err := Load("")
		require.Error(t, err)
		require.Nil(t, cfg)
	})
}
