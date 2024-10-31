package helpers

import (
	"context"
	"encoding/json"

	k8sconfig "github.com/docker/cli/cli/config/configfile"
	cliType "github.com/docker/cli/cli/config/types"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetRemotePullCredentials(ctx context.Context, client k8sclient.Client, pod *corev1.Pod) (map[string]cliType.AuthConfig, error) {
	remoteSecrets := make(map[string]cliType.AuthConfig)
	for _, imagePullSecret := range pod.Spec.ImagePullSecrets {
		secret := &corev1.Secret{}
		dockerConfig := []byte{}
		if err := client.Get(ctx, k8sclient.ObjectKey{Namespace: pod.Namespace, Name: imagePullSecret.Name}, secret); err != nil {
			continue
		}
		if dc, ok := secret.Data[".dockerconfigjson"]; ok {
			dockerConfig = dc
		} else if dc, ok := secret.Data["config.json"]; ok {
			dockerConfig = dc
		} else {
			return nil, errors.New("no dockerconfigjson or config.json found in secret")
		}

		var config k8sconfig.ConfigFile
		if err := json.Unmarshal(dockerConfig, &config); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal dockerconfigjson")
		}
		for authRepo, auth := range config.AuthConfigs {
			remoteSecrets[authRepo] = auth
		}
	}
	// TODO-cred
	return remoteSecrets, nil
}
