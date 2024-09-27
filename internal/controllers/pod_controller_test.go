package controllers

import (
	"context"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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

	requeueTime := 60 * time.Minute
	testLogger := test_helpers.NewTestZapLogger(t)
	ctrl := NewPodReconciler(k8sClient, scheme.Scheme, podValidator, nil, PodReconcilerConfig{
		RequeueAfter: requeueTime,
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
			expectedResult: reconcile.Result{RequeueAfter: requeueTime},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//GIVEN
			require.NoError(t, k8sClient.Create(context.TODO(), &tc.pod))
			defer deletePod(t, k8sClient, &tc.pod)
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

			labelValue, found := finalPod.Labels[pkg.PodValidationLabel]
			require.True(t, found)
			require.Equal(t, tc.expectedLabel, labelValue)
		})
	}
}

func Test_PodReconcileForSystemOrUserValidation(t *testing.T) {
	testEnv, k8sClient := test_suite.Setup(t)
	defer test_suite.TearDown(t, testEnv)

	requeueTime := 60 * time.Minute
	testLogger := test_helpers.NewTestZapLogger(t)

	testCases := []struct {
		name      string
		namespace corev1.Namespace
	}{
		{
			name: "Reconcile pod with system (enabled) validator",
			namespace: corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name:   "warden-enabled",
				Labels: map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled}}},
		},
		{
			name: "Reconcile pod with system (system) validator",
			namespace: corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name:   "warden-system",
				Labels: map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationSystem}}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//GIVEN
			// system validator should be called
			systemImageValidator := mocks.NewImageValidatorService(t)
			systemImageValidator.On("Validate", mock.Anything, mock.Anything).
				Return(nil).Once()
			systemPodValidator := validate.NewPodValidator(systemImageValidator)

			// user validator (factory) should not be called
			userValidatorFactory := mocks.NewValidatorSvcFactory(t)
			userValidatorFactory.AssertNotCalled(t, "NewValidatorSvc")
			defer userValidatorFactory.AssertExpectations(t)

			require.NoError(t, k8sClient.Create(context.TODO(), &tc.namespace))
			defer deleteNamespace(t, k8sClient, &tc.namespace)

			pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Namespace: tc.namespace.GetName(),
				Name:      "valid-pod"},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Image: "image", Name: "container"}}},
			}
			require.NoError(t, k8sClient.Create(context.TODO(), &pod))
			defer deletePod(t, k8sClient, &pod)
			req := reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: pod.GetNamespace(),
				Name:      pod.GetName()},
			}

			ctrl := NewPodReconciler(k8sClient, scheme.Scheme, systemPodValidator, userValidatorFactory,
				PodReconcilerConfig{RequeueAfter: requeueTime}, testLogger.Sugar())

			//WHEN
			res, err := ctrl.Reconcile(context.TODO(), req)

			//THEN
			require.NoError(t, err)
			assert.Equal(t, reconcile.Result{}, res)

			key := ctrlclient.ObjectKeyFromObject(&pod)

			finalPod := corev1.Pod{}
			require.NoError(t, k8sClient.Get(context.TODO(), key, &finalPod))

			labelValue, found := finalPod.Labels[pkg.PodValidationLabel]
			require.True(t, found)
			require.Equal(t, pkg.ValidationStatusSuccess, labelValue)
		})
	}

	t.Run("Reconcile pod with user validator", func(t *testing.T) {
		//GIVEN
		// system validator should not be called
		systemImageValidator := mocks.NewImageValidatorService(t)
		systemImageValidator.AssertNotCalled(t, "Validate")
		systemPodValidator := validate.NewPodValidator(systemImageValidator)

		// user validator should be called
		userValidator := mocks.NewPodValidator(t)
		userValidator.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
			Return(validate.ValidationResult{Status: validate.Valid}, nil).Once()
		defer userValidator.AssertExpectations(t)

		userValidatorFactory := mocks.NewValidatorSvcFactory(t)
		userValidatorFactory.On("NewValidatorSvc", mock.Anything, mock.Anything, mock.Anything).
			Return(userValidator).Once()
		defer userValidatorFactory.AssertExpectations(t)

		ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name:        "warden-user",
			Labels:      map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationUser},
			Annotations: map[string]string{pkg.NamespaceNotaryURLAnnotation: "notary"},
		}}
		require.NoError(t, k8sClient.Create(context.TODO(), &ns))
		defer deleteNamespace(t, k8sClient, &ns)

		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{
			Namespace: ns.GetName(),
			Name:      "valid-pod"},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Image: "image", Name: "container"}}},
		}
		require.NoError(t, k8sClient.Create(context.TODO(), &pod))
		defer deletePod(t, k8sClient, &pod)
		req := reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: pod.GetNamespace(),
			Name:      pod.GetName()},
		}

		ctrl := NewPodReconciler(k8sClient, scheme.Scheme, systemPodValidator, userValidatorFactory,
			PodReconcilerConfig{RequeueAfter: requeueTime}, testLogger.Sugar())

		//WHEN
		res, err := ctrl.Reconcile(context.TODO(), req)

		//THEN
		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, res)

		key := ctrlclient.ObjectKeyFromObject(&pod)

		finalPod := corev1.Pod{}
		require.NoError(t, k8sClient.Get(context.TODO(), key, &finalPod))

		labelValue, found := finalPod.Labels[pkg.PodValidationLabel]
		require.True(t, found)
		require.Equal(t, pkg.ValidationStatusSuccess, labelValue)
	})
}

func deletePod(t *testing.T, k8sClient ctrlclient.Client, pod *corev1.Pod) {
	require.NoError(t, k8sClient.Delete(context.TODO(), pod))
}

func deleteNamespace(t *testing.T, k8sClient ctrlclient.Client, ns *corev1.Namespace) {
	require.NoError(t, k8sClient.Delete(context.TODO(), ns))
}

type MockK8sClient struct {
	ctrlclient.Client
	called bool
}

func (c *MockK8sClient) Patch(ctx context.Context, obj ctrlclient.Object, patch ctrlclient.Patch, opts ...ctrlclient.PatchOption) error {
	c.called = true
	return errors.New("Error occurred")
}

func (c *MockK8sClient) assertCalled(t *testing.T) {
	require.True(t, c.called)
}

func TestReconcile_K8sOperationFails(t *testing.T) {
	imageValidator := mocks.NewImageValidatorService(t)
	imageValidator.On("Validate", mock.Anything, validImage).Return(nil).Maybe()
	podValidator := validate.NewPodValidator(imageValidator)

	validatableNs := "warden-enabled"
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: validatableNs,
		Labels: map[string]string{
			pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled,
		}},
	}
	requeueTime := 60 * time.Minute
	testLogger := test_helpers.NewTestZapLogger(t)

	t.Run("Image is valid, patching failed, should requeue", func(t *testing.T) {
		builder := fake.ClientBuilder{}
		k8sClient := builder.Build()
		mockK8Client := &MockK8sClient{k8sClient, false}
		defer mockK8Client.assertCalled(t)
		require.NoError(t, mockK8Client.Create(context.TODO(), &ns))

		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{
			Namespace: validatableNs,
			Name:      "unavailable-pod"},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Image: validImage, Name: "container"}}}}
		require.NoError(t, mockK8Client.Create(context.TODO(), &pod))

		ctrl := NewPodReconciler(mockK8Client, scheme.Scheme, podValidator, nil, PodReconcilerConfig{
			RequeueAfter: requeueTime,
		}, testLogger.Sugar())
		req := reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: validatableNs,
			Name:      pod.Name},
		}
		expectedRes := reconcile.Result{Requeue: true}

		//WHEN
		res, err := ctrl.Reconcile(context.TODO(), req)

		//THEN
		require.NoError(t, err)
		assert.Equal(t, expectedRes, res)
	})

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
