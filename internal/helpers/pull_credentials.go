package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	k8sconfig "github.com/docker/cli/cli/config/configfile"
	cliType "github.com/docker/cli/cli/config/types"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetRemotePullCredentials(ctx context.Context, reader k8sclient.Reader, pod *corev1.Pod) (map[string]cliType.AuthConfig, error) {
	remoteSecrets := make(map[string]cliType.AuthConfig)
	for _, imagePullSecret := range pod.Spec.ImagePullSecrets {
		secret := &corev1.Secret{}
		var dockerConfig []byte
		if err := reader.Get(ctx, k8sclient.ObjectKey{Namespace: pod.Namespace, Name: imagePullSecret.Name}, secret); err != nil {
			if k8sclient.IgnoreNotFound(err) != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("can't get %s/%s", pod.Namespace, imagePullSecret.Name)) //"failed to get secret")
			}
			return remoteSecrets, nil
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
			// remove any protocol from authRepo string, and trailing slash
			authRepoFragments := strings.Split(authRepo, "://")
			repoURL := authRepoFragments[len(authRepoFragments)-1]
			repoURL = strings.TrimRight(repoURL, "/")

			// technically, you could get a slice of authConfigs for each repoURL
			remoteSecrets[repoURL] = auth
		}
	}
	return remoteSecrets, nil
}
