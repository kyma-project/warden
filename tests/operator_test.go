//go:build integration

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
	TrustedImageName   = "europe-docker.pkg.dev/kyma-project/prod/function-controller:v20230428-1ea34f8e"
)

//TODO: as unit tests:
//pending
//different image names
//mock image validator?
//skip some images from list

//TODO: as integration test?
//notary svc is not available while validating untrusted image, so the controller mark it later

//TODO: update unscanned namespace to scanned

func Test_SimplePodWithImage_ShouldBeCreated(t *testing.T) {
	tc := th.NewTestContext(t, "warden-simple").Initialize()
	defer tc.Destroy()

	container := corev1.Container{Name: "test-container", Image: "nginx"}
	pod := tc.Pod().WithContainer(container).Build()
	err := tc.Create(pod)

	require.NoError(t, err)
	defer tc.Delete(pod)
}

func Test_PodInsideVerifiedNamespaceWithUntrustedImage_ShouldBeRejected(t *testing.T) {
	tc := th.NewTestContext(t, "warden-verified-namespace-untrusted-image").
		ValidationEnabled(true).
		Initialize()
	defer tc.Destroy()

	container := corev1.Container{Name: "test-container", Image: UntrustedImageName}
	pod := tc.Pod().WithContainer(container).Build()
	err := tc.Create(pod)
	require.Error(t, err)
	require.ErrorContains(t, err, "Pod images validation failed")
}

func Test_PodInsideVerifiedNamespaceWithTrustedImage_ShouldBeCreatedWithValidationLabel(t *testing.T) {
	tc := th.NewTestContext(t, "warden-verified-namespace-trusted-image").
		ValidationEnabled(true).
		Initialize()
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

func Test_PodInsideNotVerifiedNamespaceWithTrustedImage_ShouldBeCreatedWithoutValidationLabel(t *testing.T) {
	tc := th.NewTestContext(t, "warden-not-verified-namespace").
		ValidationEnabled(false).
		Initialize()
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

func Test_UpdateVerifiedPodWithUntrustedImage_ShouldBeRejected(t *testing.T) {
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
	require.Error(t, err)
	require.ErrorContains(t, err, "Pod images validation failed")
}
