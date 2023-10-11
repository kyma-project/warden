package certs

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testSecretName    = "test-secret"
	testNamespaceName = "test-namespace"
	testServiceName   = "test-service"
)

func Test_serviceAltNames(t *testing.T) {
	type args struct {
		serviceName string
		namespace   string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "service AltNames are generated correctly",
			args: args{serviceName: "test-service", namespace: "test-namespace"},
			// not using consts here to make it as readable as possible.
			want: []string{
				"test-service.test-namespace.svc",
				"test-service",
				"test-service.test-namespace",
				"test-service.test-namespace.svc.cluster.local",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serviceAltNames(tt.args.serviceName, tt.args.namespace)
			require.ElementsMatch(t, got, tt.want)
		})
	}
}

func TestEnsureWebhookSecret(t *testing.T) {
	ctx := context.Background()
	cert, key, err := generateWebhookCertificates(testServiceName, testNamespaceName)
	require.NoError(t, err)
	fakeLogger := zap.NewNop().Sugar()

	t.Run("can ensure the secret if it doesn't exist", func(t *testing.T) {
		client := fake.NewClientBuilder().Build()

		err := EnsureWebhookSecret(ctx, client, testSecretName, testNamespaceName, testServiceName, "", false, fakeLogger)
		require.NoError(t, err)

		secret := &corev1.Secret{}
		err = client.Get(ctx, types.NamespacedName{Name: testSecretName, Namespace: testNamespaceName}, secret)

		require.NoError(t, err)
		require.NotNil(t, secret)
		require.Equal(t, testSecretName, secret.Name)
		require.Equal(t, testNamespaceName, secret.Namespace)
		require.Contains(t, secret.Data, KeyFile)
		require.Contains(t, secret.Data, CertFile)
	})

	t.Run("can ensure the secret is updated if it exists", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      testSecretName,
				Namespace: testNamespaceName,
				Labels: map[string]string{
					"dont-remove-me": "true",
				},
			},
		}

		client := fake.NewClientBuilder().
			WithObjects(secret).
			Build()

		err := EnsureWebhookSecret(ctx, client, testSecretName, testNamespaceName, testServiceName, "", false, fakeLogger)
		require.NoError(t, err)

		updatedSecret := &corev1.Secret{}
		err = client.Get(ctx, types.NamespacedName{Name: testSecretName, Namespace: testNamespaceName}, updatedSecret)

		require.NoError(t, err)
		require.NotNil(t, secret)
		require.Equal(t, testSecretName, updatedSecret.Name)
		require.Equal(t, testNamespaceName, updatedSecret.Namespace)
		require.Contains(t, updatedSecret.Data, KeyFile)
		require.Contains(t, updatedSecret.Data, CertFile)
		require.Contains(t, updatedSecret.Labels, "dont-remove-me")
	})

	t.Run("can ensure the secret is updated if it's missing a value", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      testSecretName,
				Namespace: testNamespaceName,
				Labels: map[string]string{
					"dont-remove-me": "true",
				},
			},
			Data: map[string][]byte{
				KeyFile: key,
			},
		}

		client := fake.NewClientBuilder().
			WithObjects(secret).
			Build()

		err := EnsureWebhookSecret(ctx, client, testSecretName, testNamespaceName, testServiceName, "", false, fakeLogger)
		require.NoError(t, err)

		updatedSecret := &corev1.Secret{}
		err = client.Get(ctx, types.NamespacedName{Name: testSecretName, Namespace: testNamespaceName}, updatedSecret)

		require.NoError(t, err)
		require.NotNil(t, secret)
		require.Equal(t, testSecretName, updatedSecret.Name)
		require.Equal(t, testNamespaceName, updatedSecret.Namespace)
		// make sure the test is updated
		require.NotEqual(t, secret.ResourceVersion, updatedSecret.ResourceVersion)
		require.Contains(t, updatedSecret.Data, KeyFile)
		require.Contains(t, updatedSecret.Data, CertFile)
		require.Contains(t, updatedSecret.Labels, "dont-remove-me")
	})

	t.Run("doesn't update the secret if it's ok", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      testSecretName,
				Namespace: testNamespaceName,
				Labels: map[string]string{
					"dont-remove-me": "true",
				},
			},
			Data: map[string][]byte{
				KeyFile:  key,
				CertFile: cert,
			},
		}

		client := fake.NewClientBuilder().
			WithObjects(secret).
			Build()

		err := EnsureWebhookSecret(ctx, client, testSecretName, testNamespaceName, testServiceName, "", false, fakeLogger)
		require.NoError(t, err)

		updatedSecret := &corev1.Secret{}
		err = client.Get(ctx, types.NamespacedName{Name: testSecretName, Namespace: testNamespaceName}, updatedSecret)

		require.NoError(t, err)
		require.NotNil(t, secret)
		require.Equal(t, testSecretName, updatedSecret.Name)
		require.Equal(t, testNamespaceName, updatedSecret.Namespace)
		// make sure it's not updated
		require.Equal(t, secret.ResourceVersion, updatedSecret.ResourceVersion)
		require.Contains(t, updatedSecret.Data, KeyFile)
		require.Contains(t, updatedSecret.Data, CertFile)
		require.Equal(t, key, updatedSecret.Data[KeyFile])
		require.Equal(t, cert, updatedSecret.Data[CertFile])
		require.Contains(t, updatedSecret.Labels, "dont-remove-me")
	})

	t.Run("should update if the cert will expire in 10 days", func(t *testing.T) {
		tenDaysCert, err := generateShortLivedCertWithKey(key, testServiceName, 10*24*time.Hour)
		require.NoError(t, err)

		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      testSecretName,
				Namespace: testNamespaceName,
				Labels: map[string]string{
					"dont-remove-me": "true",
				},
			},
			Data: map[string][]byte{
				KeyFile:  key,
				CertFile: tenDaysCert,
			},
		}

		client := fake.NewClientBuilder().
			WithObjects(secret).
			Build()

		err = EnsureWebhookSecret(ctx, client, testSecretName, testNamespaceName, testServiceName, "", false, fakeLogger)
		require.NoError(t, err)

		updatedSecret := &corev1.Secret{}
		err = client.Get(ctx, types.NamespacedName{Name: testSecretName, Namespace: testNamespaceName}, updatedSecret)

		require.NoError(t, err)
		require.NotNil(t, secret)
		require.Equal(t, testSecretName, updatedSecret.Name)
		require.Equal(t, testNamespaceName, updatedSecret.Namespace)
		require.Contains(t, updatedSecret.Data, KeyFile)
		require.Contains(t, updatedSecret.Data, CertFile)
		// make sure it's updated, not overridden.
		require.NotEqual(t, secret.ResourceVersion, updatedSecret.ResourceVersion)
		require.NotEqual(t, key, updatedSecret.Data[KeyFile])
		require.NotEqual(t, cert, updatedSecret.Data[CertFile])
		require.Contains(t, updatedSecret.Labels, "dont-remove-me")
	})

	t.Run("should not update if the cert will expire in more than 10 days", func(t *testing.T) {
		elevenDaysCert, err := generateShortLivedCertWithKey(key, testServiceName, 11*24*time.Hour)
		require.NoError(t, err)

		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      testSecretName,
				Namespace: testNamespaceName,
				Labels: map[string]string{
					"dont-remove-me": "true",
				},
			},
			Data: map[string][]byte{
				KeyFile:  key,
				CertFile: elevenDaysCert,
			},
		}

		client := fake.NewClientBuilder().
			WithObjects(secret).
			Build()

		err = EnsureWebhookSecret(ctx, client, testSecretName, testNamespaceName, testServiceName, "", false, fakeLogger)
		require.NoError(t, err)

		updatedSecret := &corev1.Secret{}
		err = client.Get(ctx, types.NamespacedName{Name: testSecretName, Namespace: testNamespaceName}, updatedSecret)

		require.NoError(t, err)
		require.NotNil(t, secret)
		require.Equal(t, testSecretName, updatedSecret.Name)
		require.Equal(t, testNamespaceName, updatedSecret.Namespace)
		require.Contains(t, updatedSecret.Data, KeyFile)
		require.Contains(t, updatedSecret.Data, CertFile)
		// make sure it's NOT updated, not overridden.
		require.Equal(t, secret.ResourceVersion, updatedSecret.ResourceVersion)
		require.Equal(t, key, updatedSecret.Data[KeyFile])
		require.Equal(t, elevenDaysCert, updatedSecret.Data[CertFile])
		require.Contains(t, updatedSecret.Labels, "dont-remove-me")
	})
}

func generateShortLivedCertWithKey(keyBytes []byte, host string, age time.Duration) ([]byte, error) {
	pemKey, _ := pem.Decode(keyBytes)
	key, err := x509.ParsePKCS1PrivateKey(pemKey.Bytes)
	if err != nil {
		return nil, err
	}
	t := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(age),
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &t, &t, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	buf := bytes.Buffer{}
	if err := pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func Test_buildOwnerRefs(t *testing.T) {
	t.Run("build owner reference for deployment", func(t *testing.T) {
		namespace := "default"
		deploy := &appsv1.Deployment{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-deploy",
				Namespace: namespace,
				UID:       "72b54330-6695-11ee-8c99-0242ac120002",
			},
		}

		client := fake.NewClientBuilder().WithObjects(deploy).Build()

		ownerRefs, err := buildOwnerRefs(
			context.Background(),
			client,
			namespace,
			deploy.GetName(),
			true,
		)
		require.NoError(t, err)
		require.Equal(t, []metav1.OwnerReference{
			{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       deploy.GetName(),
				UID:        deploy.GetUID(),
			},
		}, ownerRefs)
	})

	t.Run("skip building", func(t *testing.T) {
		ownerRefs, err := buildOwnerRefs(
			context.Background(),
			nil,
			"default",
			"test-deploy",
			false,
		)
		require.NoError(t, err)
		require.Equal(t, []metav1.OwnerReference{}, ownerRefs)
	})

	t.Run("failed to get deploy", func(t *testing.T) {
		client := fake.NewClientBuilder().Build()

		ownerRefs, err := buildOwnerRefs(
			context.Background(),
			client,
			"test-namespace",
			"test-deploy",
			true,
		)

		require.Error(t, err)

		var expectedRefs []metav1.OwnerReference
		require.Equal(t, expectedRefs, ownerRefs)
	})
}
