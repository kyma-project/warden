package certs

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_saveToFile(t *testing.T) {
	t.Run("save cert files", func(t *testing.T) {
		namespace := "default"
		secretName := "test-secret"
		certData := []byte("test cert data")
		keyData := []byte("test key data")
		secret := fixTestCertSecret(secretName, namespace, certData, keyData)
		client := fake.NewClientBuilder().
			WithObjects(secret).
			Build()

		certDir := path.Join(t.TempDir(), "k8s-cert", "webhook")

		err := saveToFile(
			context.Background(),
			client,
			secretName,
			namespace,
			certDir,
			zap.NewNop().Sugar(),
		)

		require.NoError(t, err)

		expectedCertFile, err := os.ReadFile(path.Join(certDir, CertFile))
		require.NoError(t, err)
		require.Equal(t, certData, expectedCertFile)

		expectedKeyFile, err := os.ReadFile(path.Join(certDir, KeyFile))
		require.NoError(t, err)
		require.Equal(t, keyData, expectedKeyFile)
	})

	t.Run("failed to get secret", func(t *testing.T) {
		client := fake.NewClientBuilder().Build()
		certDir := path.Join(t.TempDir(), "k8s-cert", "webhook")

		err := saveToFile(
			context.Background(),
			client,
			"test-secret",
			"default",
			certDir,
			zap.NewNop().Sugar(),
		)

		require.Error(t, err)
	})
}

func fixTestCertSecret(name, namespace string, certData, keyData []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			KeyFile:  keyData,
			CertFile: certData,
		},
	}
}
