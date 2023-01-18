/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

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
	//logf.SetLogger(zap.New(zap.WriteTo(), zap.UseDevMode(true)))

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
