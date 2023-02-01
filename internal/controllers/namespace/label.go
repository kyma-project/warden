package namespace

import (
	"context"

	warden "github.com/kyma-project/warden/pkg"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type patch func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error

func labelWithValidationPending(ctx context.Context, pod *corev1.Pod, patch patch) error {
	// if validation label is already set do not patch the pod
	value, found := pod.Labels[warden.PodValidationLabel]
	if found && value == warden.ValidationStatusPending {
		return nil
	}

	// make a deep copy and initialize labels if needed
	podCopy := pod.DeepCopy()
	if podCopy.Labels == nil {
		podCopy.Labels = make(map[string]string, 1)
	}
	// add validation label and apply patch
	podCopy.Labels[warden.PodValidationLabel] = warden.ValidationStatusPending
	return patch(ctx, podCopy, client.MergeFrom(pod))
}
