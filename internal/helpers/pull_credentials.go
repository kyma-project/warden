package helpers

import (
	registryType "github.com/docker/docker/api/types/registry"
	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetRemotePullCredentials(client k8sclient.Client, pod *corev1.Pod) map[string]registryType.AuthConfig {
	remoteSecrets := make(map[string]registryType.AuthConfig)
	// TODO-cred
	return remoteSecrets
}
