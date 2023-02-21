package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/kyma-project/warden/internal/controllers/test_suite"
	"github.com/kyma-project/warden/internal/test_helpers"
	"github.com/kyma-project/warden/internal/validate"
	"github.com/kyma-project/warden/internal/validate/mocks"
	"github.com/kyma-project/warden/pkg"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	validImage   = "valid"
	invalidImage = "invalid"
)

func Test_PodReconcile(t *testing.T) {
	testEnv, k8sClient := test_suite.Setup(t)
	defer test_suite.TearDown(t, testEnv)

	imageValidator := mocks.NewImageValidatorService(t)
	imageValidator.On("Validate", mock.Anything, validImage).Return(nil).Maybe()
	imageValidator.On("Validate", mock.Anything, invalidImage).Return(errors.New("")).Maybe()

	podValidator := validate.NewPodValidator(imageValidator)

	validatableNs := "warden-enabled"
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: validatableNs,
		Labels: map[string]string{
			pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled,
		}},
	}
	require.NoError(t, k8sClient.Create(context.TODO(), &ns))

	testLogger := test_helpers.NewTestZapLogger(t)
	ctrl := NewPodReconciler(k8sClient, scheme.Scheme, podValidator, PodReconcilerConfig{
		RequeueAfter: time.Minute * 60,
	}, testLogger.Sugar())

	testCases := []struct {
		name          string
		pod           corev1.Pod
		expectedLabel string
	}{
		{
			name: "Success",
			pod: corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Namespace: validatableNs,
				Name:      "valid-pod"},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Image: validImage, Name: "container"}}},
			},
			expectedLabel: pkg.ValidationStatusSuccess,
		},
		{
			name: "Image is not valid and have pending label",
			pod: corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Namespace: validatableNs,
				Name:      "invalid-pod-pending",
				Labels:    map[string]string{pkg.PodValidationLabel: pkg.ValidationStatusPending},
			},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Image: invalidImage, Name: "container"}}},
			},
			expectedLabel: pkg.ValidationStatusFailed,
		},
		{
			name: "Image is not valid",
			pod: corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Namespace: validatableNs,
				Name:      "invalid-pod"},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Image: invalidImage, Name: "container"}}},
			},
			expectedLabel: pkg.ValidationStatusFailed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//GIVEN
			require.NoError(t, k8sClient.Create(context.TODO(), &tc.pod))
			req := reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: validatableNs,
				Name:      tc.pod.Name},
			}
			//WHEN
			_, err := ctrl.Reconcile(context.TODO(), req)
			//THEN
			require.NoError(t, err)
			key := ctrlclient.ObjectKeyFromObject(&tc.pod)

			finalPod := corev1.Pod{}
			require.NoError(t, k8sClient.Get(context.TODO(), key, &finalPod))

			labeValue, found := finalPod.Labels[pkg.PodValidationLabel]
			require.True(t, found)
			require.Equal(t, tc.expectedLabel, labeValue)
		})
	}
}

func Test_areImagesChanged(t *testing.T) {
	type args struct {
		oldPod *corev1.Pod
		newPod *corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "the same images in different order",
			args: args{
				oldPod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "old", Labels: map[string]string{"ala": "makota"}},
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{Image: "image-alpha", Name: "container-alpha"},
							{Image: "image-bravo", Name: "container-bravo"},
						},
						Containers: []corev1.Container{
							{Image: "image-oscar", Name: "container-oscar"},
							{Image: "image-papa", Name: "container-papa"},
						},
					},
				},
				newPod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "new"},
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{Image: "image-oscar", Name: "container-juliett"},
							{Image: "image-bravo", Name: "container-yankee"},
						},
						Containers: []corev1.Container{
							{Image: "image-alpha", Name: "container-zulu"},
							{Image: "image-papa", Name: "container-india"},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "no images",
			args: args{
				oldPod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "old", Labels: map[string]string{"ala": "makota"}},
				},
				newPod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "new"},
				},
			},
			want: false,
		},
		{
			name: "added image",
			args: args{
				oldPod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "old", Labels: map[string]string{"ala": "makota"}},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Image: "image-mike", Name: "container-mike"},
						},
					},
				},
				newPod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "new"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Image: "image-lima", Name: "container-lima"},
							{Image: "image-mike", Name: "container-mike"},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "removed image",
			args: args{
				oldPod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "old", Labels: map[string]string{"ala": "makota"}},
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{Image: "image-quebec"},
							{Image: "image-romeo"},
							{Image: "image-sierra"},
						},
					},
				},
				newPod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "new"},
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{Image: "image-quebec"},
							{Image: "image-sierra"},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "changed image",
			args: args{
				oldPod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "old", Labels: map[string]string{"ala": "makota"}},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Image: "image-foxtrot:1", Name: "container-foxtrot"},
						},
					},
				},
				newPod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "new"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Image: "image-foxtrot:2", Name: "container-foxtrot"},
						},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := areImagesChanged(tt.args.oldPod, tt.args.newPod)
			require.Equal(t, tt.want, got)
		})
	}
}
