//go:build integration

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/warden/pkg"
	"github.com/kyma-project/warden/tests/helpers"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNamespaceWithLabel_AfterPodCreation(t *testing.T) {
	//GIVEN
	ctx := context.TODO()
	k8sClient, err := ctrlclient.NewWithWatch(ctrl.GetConfigOrDie(), ctrlclient.Options{})
	require.NoError(t, err)
	tc := helpers.NewTestContext(t, "namespace-labeled-later")
	ns := tc.Namespace().WithValidation(false).Build()
	require.NoError(t, k8sClient.Create(ctx, ns))
	defer k8sClient.Delete(ctx, ns)

	container := corev1.Container{Name: "test-container", Image: TrustedImageName}
	trustedPod := tc.PodWithName("valid").WithContainer(container).Build()
	require.NoError(t, k8sClient.Create(ctx, trustedPod))

	untrustedContainer := corev1.Container{Name: "test-container", Image: UntrustedImageName}
	untrustedPod := tc.PodWithName("invalid").WithContainer(untrustedContainer).Build()
	require.NoError(t, k8sClient.Create(ctx, untrustedPod))

	//WHEN
	ns = tc.Namespace().WithName(ns.ObjectMeta.Name).WithValidation(true).Build()
	require.NoError(t, k8sClient.Update(ctx, ns))

	//THEN
	dynamicCli, err := dynamic.NewForConfig(ctrl.GetConfigOrDie())
	require.NoError(t, err)
	podCli := dynamicCli.Resource(corev1.SchemeGroupVersion.WithResource(corev1.ResourcePods.String()))

	t.Parallel()
	t.Run("Invalid image has failed label", func(t *testing.T) {
		watchPod(t, podCli, untrustedPod.ObjectMeta, pkg.ValidationStatusFailed, time.Second*15)
	})

	t.Run("Valid image has success label", func(t *testing.T) {
		watchPod(t, podCli, trustedPod.ObjectMeta, pkg.ValidationStatusSuccess, time.Second*15)
	})
	fmt.Println("")
}

func watchPod(t *testing.T, podCli dynamic.NamespaceableResourceInterface, podMeta metav1.ObjectMeta, expectedLabelValue string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()

	watcher, err := podCli.Namespace(podMeta.Namespace).Watch(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	defer watcher.Stop()
	for {
		select {
		case <-ctx.Done():
			require.NoError(t, ctx.Err())
		case result, ok := <-watcher.ResultChan():
			require.True(t, ok)
			ready := isLabelApplied(t, result, podMeta.Name, expectedLabelValue)
			if ready {
				return
			}
		}
	}
}

func isLabelApplied(t *testing.T, event watch.Event, podName, expectedValue string) bool {
	pod := decodePod(t, event.Object)
	if pod.Name != podName {
		return false
	}
	podLabelValue, ok := pod.Labels[pkg.PodValidationLabel]
	if !ok {
		return false
	}
	t.Logf("Name: %s, label value: %s", pod.Name, podLabelValue)

	if podLabelValue != expectedValue {
		t.Logf("Pod: %s has different expectedValue on validation label, expected: %s, got: %s", pod.Name, expectedValue, podLabelValue)
		return false
	}
	t.Logf("Pod: %s has expected label: %s", pod.Name, expectedValue)
	return true
}

func decodePod(t *testing.T, object runtime.Object) corev1.Pod {
	u, ok := object.(runtime.Unstructured)
	require.True(t, ok)
	pod := corev1.Pod{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), &pod)
	require.NoError(t, err)
	return pod
}
