package test_suite

import (
	"github.com/stretchr/testify/require"
	"testing"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	//+kubebuilder:scaffold:imports
)

func Setup(t *testing.T) (*envtest.Environment, client.Client) {
	var cfg *rest.Config
	var k8sClient client.Client
	var testEnv *envtest.Environment
	t.Log("bootstrapping test environment")
	testEnv = &envtest.Environment{}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	require.NoError(t, err)
	require.NotNil(t, k8sClient)
	return testEnv, k8sClient
}

func TearDown(t *testing.T, testEnv *envtest.Environment) {
	t.Log("tearing down the test environment")
	err := testEnv.Stop()
	require.NoError(t, err)
}
