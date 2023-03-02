package controllers

import (
	"context"
	"github.com/stretchr/testify/assert"
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
	validImage       = "valid"
	invalidImage     = "invalid"
	unavailableImage = "unavailable"
)

func Test_PodReconcile(t *testing.T) {
	testEnv, k8sClient := test_suite.Setup(t)
	defer test_suite.TearDown(t, testEnv)

	imageValidator := mocks.NewImageValidatorService(t)
	imageValidator.On("Validate", mock.Anything, validImage).Return(nil).Maybe()
	imageValidator.On("Validate", mock.Anything, invalidImage).Return(errors.New("")).Maybe()
	imageValidator.On("Validate", mock.Anything, unavailableImage).Return(pkg.NewUnknownResultErr(errors.New(""))).Maybe()

	podValidator := validate.NewPodValidator(imageValidator)

	validatableNs := "warden-enabled"
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: validatableNs,
		Labels: map[string]string{
			pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled,
		}},
	}
	require.NoError(t, k8sClient.Create(context.TODO(), &ns))

	requeueireTime := 60 * time.Minute
	testLogger := test_helpers.NewTestZapLogger(t)
	ctrl := NewPodReconciler(k8sClient, scheme.Scheme, podValidator, PodReconcilerConfig{
		RequeueAfter: requeueireTime,
	}, testLogger.Sugar())

	testCases := []struct {
		name           string
		pod            corev1.Pod
		expectedLabel  string
		expectedResult reconcile.Result
	}{
		{
			name: "Success",
			pod: corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Namespace: validatableNs,
				Name:      "valid-pod"},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Image: validImage, Name: "container"}}},
			},
			expectedLabel:  pkg.ValidationStatusSuccess,
			expectedResult: reconcile.Result{},
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
			expectedLabel:  pkg.ValidationStatusFailed,
			expectedResult: reconcile.Result{},
		},
		{
			name: "Image is not valid",
			pod: corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Namespace: validatableNs,
				Name:      "invalid-pod"},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Image: invalidImage, Name: "container"}}},
			},
			expectedLabel:  pkg.ValidationStatusFailed,
			expectedResult: reconcile.Result{}},
		{
			name: "Image has label pending and unknown error occurred",
			pod: corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Namespace: validatableNs,
				Name:      "unavailable-pod"},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Image: unavailableImage, Name: "container"}}}},
			expectedLabel:  pkg.ValidationStatusPending,
			expectedResult: reconcile.Result{RequeueAfter: requeueireTime},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//GIVEN
			require.NoError(t, k8sClient.Create(context.TODO(), &tc.pod))
			defer k8sClient.Delete(context.TODO(), &tc.pod)
			req := reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: validatableNs,
				Name:      tc.pod.Name},
			}
			//WHEN
			res, err := ctrl.Reconcile(context.TODO(), req)
			//THEN
			require.NoError(t, err)
			assert.Equal(t, tc.expectedResult, res)

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
	type podImages struct {
		Containers     []corev1.Container
		InitContainers []corev1.Container
	}
	type args struct {
		oldPod podImages
		newPod podImages
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "the same images in different order",
			args: args{
				oldPod: podImages{InitContainers: []corev1.Container{
					{Image: "image-alpha", Name: "container-alpha"},
					{Image: "image-bravo", Name: "container-bravo"},
				},
					Containers: []corev1.Container{
						{Image: "image-oscar", Name: "container-oscar"},
						{Image: "image-papa", Name: "container-papa"},
					},
				},
				newPod: podImages{
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
			want: false,
		},
		{
			name: "no images",
			args: args{
				oldPod: podImages{},
				newPod: podImages{},
			},
			want: false,
		},
		{
			name: "added image",
			args: args{
				oldPod: podImages{
					Containers: []corev1.Container{
						{Image: "image-mike", Name: "container-mike"},
					},
				},
				newPod: podImages{
					Containers: []corev1.Container{
						{Image: "image-lima", Name: "container-lima"},
						{Image: "image-mike", Name: "container-mike"},
					},
				},
			},
			want: true,
		},
		{
			name: "removed image",
			args: args{
				oldPod: podImages{
					InitContainers: []corev1.Container{
						{Image: "image-quebec"},
						{Image: "image-romeo"},
						{Image: "image-sierra"},
					},
				},
				newPod: podImages{
					InitContainers: []corev1.Container{
						{Image: "image-quebec"},
						{Image: "image-sierra"},
					},
				},
			},
			want: true,
		},
		{
			name: "changed image",
			args: args{
				oldPod: podImages{
					Containers: []corev1.Container{
						{Image: "image-foxtrot:1", Name: "container-foxtrot"},
					},
				},
				newPod: podImages{
					Containers: []corev1.Container{
						{Image: "image-foxtrot:2", Name: "container-foxtrot"},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//GIVEN
			oldPod := fixPod(tt.args.oldPod.InitContainers, tt.args.oldPod.Containers)
			newPod := fixPod(tt.args.newPod.InitContainers, tt.args.newPod.Containers)
			//WHEN
			got := areImagesChanged(oldPod, newPod)

			//THEN
			require.Equal(t, tt.want, got)
		})
	}
}

func fixPod(initContainers []corev1.Container, containers []corev1.Container) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "new"},
		Spec: corev1.PodSpec{
			InitContainers: initContainers,
			Containers:     containers,
		},
	}
}
