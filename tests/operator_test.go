package tests

import (
	"github.com/kyma-project/warden/pkg"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"testing"
	th "warden.kyma-project.io/tests/helpers"
)

const (
	UntrustedImageName = "nginx:latest"
	TrustedImageName   = "eu.gcr.io/kyma-project/function-controller:PR-16481"
)

//TODO: now these tests based only on warden-operator - should be modified after add warden-admission

//TODO: as unit tests:
//pending
//different image names
//how to mock image validator?

func Test_SimplePodWithImage_ShouldBeCreated(t *testing.T) {
	tc := th.NewTestContext(t, "warden").Initialize()
	defer tc.Destroy()

	container := corev1.Container{Name: "test-container", Image: "nginx"}
	pod := tc.Pod().WithContainer(container).Build()
	err := tc.Create(pod)

	require.NoError(t, err)
	defer tc.Delete(pod)
}

// invalid image in scanned namespace
func Test_PodInsideVerifiedNamespaceWithUntrustedImage_ShouldBeCreatedWithValidationLabel(t *testing.T) {
	tc := th.NewTestContext(t, "warden").ValidationEnabled(true).Initialize()
	defer tc.Destroy()

	container := corev1.Container{Name: "test-container", Image: UntrustedImageName}
	pod := tc.Pod().WithContainer(container).Build()
	err := tc.Create(pod)
	require.NoError(t, err)
	defer tc.Delete(pod)

	var existingPod corev1.Pod
	tc.GetPodWhenReady(pod, &existingPod)
	require.Contains(t, existingPod.ObjectMeta.Labels, pkg.PodValidationLabel)
	require.Equal(t, pkg.ValidationStatusFailed, existingPod.ObjectMeta.Labels[pkg.PodValidationLabel])
}

// valid image in scanned namespace
func Test_PodInsideVerifiedNamespaceWithTrustedImage_ShouldBeCreatedWithValidationLabel(t *testing.T) {
	tc := th.NewTestContext(t, "warden").ValidationEnabled(true).Initialize()
	defer tc.Destroy()

	container := corev1.Container{Name: "test-container", Image: TrustedImageName}
	pod := tc.Pod().WithContainer(container).Build()
	err := tc.Create(pod)
	require.NoError(t, err)
	defer tc.Delete(pod)

	var existingPod corev1.Pod
	tc.GetPodWhenReady(pod, &existingPod)
	require.Contains(t, existingPod.ObjectMeta.Labels, pkg.PodValidationLabel)
	require.Equal(t, pkg.ValidationStatusSuccess, existingPod.ObjectMeta.Labels[pkg.PodValidationLabel])
}

// invalid image in unscanned namespace
func Test_PodInsideNotVerifiedNamespaceWithUntrustedImage_ShouldBeCreatedWithoutValidationLabel(t *testing.T) {
	tc := th.NewTestContext(t, "warden").ValidationEnabled(false).Initialize()
	defer tc.Destroy()

	container := corev1.Container{Name: "test-container", Image: UntrustedImageName}
	pod := tc.Pod().WithContainer(container).Build()
	err := tc.Create(pod)
	require.NoError(t, err)
	defer tc.Delete(pod)

	var existingPod corev1.Pod
	tc.GetPodWhenReady(pod, &existingPod)
	require.NotContains(t, existingPod.ObjectMeta.Labels, pkg.PodValidationLabel)
}

// valid image in unscanned namespace
func Test_PodInsideNotVerifiedNamespaceWithTrustedImage_ShouldBeCreatedWithoutValidationLabel(t *testing.T) {
	tc := th.NewTestContext(t, "warden").ValidationEnabled(false).Initialize()
	defer tc.Destroy()

	container := corev1.Container{Name: "test-container", Image: TrustedImageName}
	pod := tc.Pod().WithContainer(container).Build()
	err := tc.Create(pod)
	require.NoError(t, err)
	defer tc.Delete(pod)

	var existingPod corev1.Pod
	tc.GetPodWhenReady(pod, &existingPod)
	require.NotContains(t, existingPod.ObjectMeta.Labels, pkg.PodValidationLabel)
}

// valid image in scanned namespace and update it with invalid/valid
func Test_PodInsideVerifiedNamespaceWithTrustedImage_ShouldBeUpdatedWithProperValidationLabel(t *testing.T) {
	tc := th.NewTestContext(t, "warden").ValidationEnabled(true).Initialize()
	defer tc.Destroy()

	container := corev1.Container{Name: "test-container", Image: TrustedImageName}
	pod := tc.Pod().WithContainer(container).Build()
	err := tc.Create(pod)
	require.NoError(t, err)
	defer tc.Delete(pod)

	var existingPod corev1.Pod
	tc.GetPodWhenReady(pod, &existingPod)
	require.Contains(t, existingPod.ObjectMeta.Labels, pkg.PodValidationLabel)
	require.Equal(t, pkg.ValidationStatusSuccess, existingPod.ObjectMeta.Labels[pkg.PodValidationLabel])

	pod = &existingPod
	pod.Spec.Containers[0].Image = UntrustedImageName
	err = tc.Update(pod)
	require.NoError(t, err)

	tc.GetPodWhenCondition(pod, &existingPod, func(p *corev1.Pod) bool {
		if p.ObjectMeta.Labels == nil {
			return false
		}
		v, err := p.ObjectMeta.Labels[pkg.PodValidationLabel]
		if err {
			return false
		}
		return v == pkg.ValidationStatusFailed
	})
	require.Contains(t, existingPod.ObjectMeta.Labels, pkg.PodValidationLabel)
	require.Equal(t, pkg.ValidationStatusFailed, existingPod.ObjectMeta.Labels[pkg.PodValidationLabel])
}
