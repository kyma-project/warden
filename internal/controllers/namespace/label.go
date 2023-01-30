package namespace

import (
	"context"

	warden "github.com/kyma-project/warden/pkg"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type patch func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error

func labelWithValidationPendin(ctx context.Context, patch patch, pod *corev1.Pod) error {
	// make a deep copy and initialize labels if needed
	podCopy := pod.DeepCopy()
	if podCopy.Labels == nil {
		podCopy.Labels = make(map[string]string, 1)
	}
	// add validation lable and apply patch
	podCopy.Labels[warden.NamespaceValidationLabel] = warden.NamespaceValidationEnabled
	return patch(ctx, podCopy, client.MergeFrom(pod))
}
