package controllers

import (
	"context"
	"github.com/kyma-project/warden/internal/validate"
	"github.com/kyma-project/warden/internal/validate/mocks"
	"github.com/kyma-project/warden/pkg"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

const validImage = "valid"

func TestName(t *testing.T) {

	testEnv, k8sClient := Setup(t)
	defer TearDown(t, testEnv)

	imageValidator := mocks.NewImageValidatorService(t)
	imageValidator.On("Validate", validImage).Return(nil)

	podValidator := validate.NewPodValidator(imageValidator)

	validatableNs := "warden-enabled"
	ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: validatableNs,
		Labels: map[string]string{
			pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled,
		}},
	}
	require.NoError(t, k8sClient.Create(context.TODO(), &ns))

	validpod := "validpod"
	pod := v1.Pod{ObjectMeta: metav1.ObjectMeta{
		Namespace: validatableNs,
		Name:      validpod,
	},
		Spec: v1.PodSpec{Containers: []v1.Container{{Image: validImage, Name: "container"}}},
	}
	require.NoError(t, k8sClient.Create(context.TODO(), &pod))

	ctrl := PodReconciler{
		Client:    k8sClient,
		Scheme:    scheme.Scheme,
		Validator: podValidator,
	}

	t.Run("Success", func(t *testing.T) {
		//GIVEN
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: validatableNs,
				Name:      validpod,
			},
		}
		//WHEN
		_, err := ctrl.Reconcile(context.TODO(), req)
		//THEN
		require.NoError(t, err)
		key := ctrlclient.ObjectKeyFromObject(&pod)

		finalPod := v1.Pod{}
		require.NoError(t, k8sClient.Get(context.TODO(), key, &finalPod))

		labeValue, ok := finalPod.Labels[pkg.PodValidationLabel]
		require.True(t, ok)
		require.Equal(t, pkg.ValidationStatusSuccess, labeValue)
	})
}
