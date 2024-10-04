package admission

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/kyma-project/warden/internal/helpers"
	"k8s.io/utils/ptr"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/kyma-project/warden/internal/annotations"
	"github.com/kyma-project/warden/internal/test_helpers"
	"github.com/kyma-project/warden/internal/validate"
	"github.com/kyma-project/warden/internal/validate/mocks"
	"github.com/kyma-project/warden/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	StrictModeOff = false
	StrictModeOn  = true
)

func TestTimeout(t *testing.T) {
	//GIVEN
	logger := zap.NewNop()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	decoder := admission.NewDecoder(scheme)
	timeout := time.Millisecond * 100

	testNs := "test-namespace"
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNs,
		Labels: map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled}}}
	pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: testNs},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Image: "test:test"}}},
	}

	raw, err := json.Marshal(pod)
	require.NoError(t, err)

	req := admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
		Kind:   metav1.GroupVersionKind{Kind: PodType, Version: corev1.SchemeGroupVersion.Version},
		Object: runtime.RawExtension{Raw: raw},
	}}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()

	t.Run("Success", func(t *testing.T) {
		//GIVEN
		validationSvc := mocks.NewPodValidator(t)
		validationSvc.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
			After(timeout/2).
			Return(validate.ValidationResult{Status: validate.Valid}, nil).Once()
		defer validationSvc.AssertExpectations(t)
		webhook := NewDefaultingWebhook(client,
			validationSvc, nil, timeout, StrictModeOff, decoder, logger.Sugar())

		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.Nil(t, res.Result)
		assert.True(t, res.Allowed)
	})

	t.Run("Defaulting webhook timeout, allowed", func(t *testing.T) {
		validationSvc := mocks.NewPodValidator(t)
		validationSvc.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
			After(timeout*2).
			Return(validate.ValidationResult{Status: validate.Valid}, nil).Once()
		defer validationSvc.AssertExpectations(t)
		webhook := NewDefaultingWebhook(client,
			validationSvc, nil, timeout, StrictModeOff, decoder, logger.Sugar())
		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.NotNil(t, res.Result)
		assert.Contains(t, res.Result.Message, "request exceeded desired timeout")
		assert.True(t, res.Allowed)
		assert.ElementsMatch(t, patchWithAddLabel(pkg.ValidationStatusPending), res.Patches)
	})

	t.Run("Defaulting webhook timeout strict mode on, errored", func(t *testing.T) {
		validationSvc := mocks.NewPodValidator(t)
		validationSvc.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
			After(timeout*2).
			Return(validate.ValidationResult{Status: validate.Valid}, nil).Once()
		defer validationSvc.AssertExpectations(t)
		webhook := NewDefaultingWebhook(client,
			validationSvc, nil, timeout, StrictModeOn, decoder, logger.Sugar())
		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.NotNil(t, res.Result, "response is ok")
		assert.Contains(t, res.Result.Message, "request exceeded desired timeout")
		assert.True(t, res.Allowed)
		assert.ElementsMatch(t, withAddRejectAnnotation(patchWithAddLabel(pkg.ValidationStatusPending)), res.Patches)
	})

	t.Run("Defaulting webhook timeout - all layers", func(t *testing.T) {
		timeout := time.Second
		start := time.Now()
		h := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			time.Sleep(timeout * 2)
		})
		srv := httptest.NewServer(h)
		defer srv.Close()

		validateImage := validate.NewImageValidator(&validate.ServiceConfig{NotaryConfig: validate.NotaryConfig{Url: srv.URL}}, validate.NotaryRepoFactory{})
		validationSvc := validate.NewPodValidator(validateImage)
		webhook := NewDefaultingWebhook(client,
			validationSvc, nil, timeout, StrictModeOff, decoder, logger.Sugar())
		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.NotNil(t, res.Result)
		assert.True(t, res.AdmissionResponse.Allowed)
		assert.Contains(t, res.Result.Message, "request exceeded desired timeout")
		assert.InDelta(t, timeout.Seconds(), time.Since(start).Seconds(), 0.1, "timeout duration is not respected")
		assert.ElementsMatch(t, patchWithAddLabel(pkg.ValidationStatusPending), res.Patches)
	})
}

func TestFlow_OutputStatuses_ForPodValidationResult(t *testing.T) {
	//GIVEN
	logger := test_helpers.NewTestZapLogger(t)
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	decoder := admission.NewDecoder(scheme)
	timeout := time.Second

	nsName := "test-namespace"

	t.Run("when valid image should return success", func(t *testing.T) {
		//GIVEN
		ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName,
			Labels: map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled}}}
		mockImageValidator := mocks.ImageValidatorService{}
		mockImageValidator.Mock.On("Validate", mock.Anything, "test:test").Return(nil)
		mockPodValidator := validate.NewPodValidator(&mockImageValidator)

		pod := newPodFix(nsName, nil)
		req := newRequestFix(t, pod, admissionv1.Create)
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()
		webhook := NewDefaultingWebhook(client,
			mockPodValidator, nil, timeout, false, decoder, logger.Sugar())

		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		mockImageValidator.AssertNumberOfCalls(t, "Validate", 1)
		require.NotNil(t, res)
		require.True(t, res.AdmissionResponse.Allowed)
		require.ElementsMatch(t, patchWithAddSuccessLabel(), res.Patches)
	})

	t.Run("when valid image with annotation reject should return success and remove the annotation", func(t *testing.T) {
		//GIVEN
		ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName,
			Labels: map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled}}}
		mockImageValidator := mocks.ImageValidatorService{}
		mockImageValidator.Mock.On("Validate", mock.Anything, "test:test").Return(nil)
		mockPodValidator := validate.NewPodValidator(&mockImageValidator)

		pod := newPodFix(nsName, nil)
		pod.ObjectMeta.Annotations = map[string]string{annotations.PodValidationRejectAnnotation: annotations.ValidationReject}
		req := newRequestFix(t, pod, admissionv1.Create)
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()
		webhook := NewDefaultingWebhook(client,
			mockPodValidator, nil, timeout, false, decoder, logger.Sugar())

		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		mockImageValidator.AssertNumberOfCalls(t, "Validate", 1)
		require.NotNil(t, res)
		require.True(t, res.AdmissionResponse.Allowed)
		require.ElementsMatch(t, withRemovedAnnotation(patchWithAddSuccessLabel()), res.Patches)
	})

	t.Run("when pod labeled by ns controller with pending label and annotation reject should remove the annotation", func(t *testing.T) {
		//GIVEN
		ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName,
			Labels: map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled}}}
		pod := newPodFix(nsName, map[string]string{pkg.PodValidationLabel: pkg.ValidationStatusPending})
		pod.ObjectMeta.Annotations = map[string]string{annotations.PodValidationRejectAnnotation: annotations.ValidationReject}
		req := newRequestFix(t, pod, admissionv1.Update)
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()
		webhook := NewDefaultingWebhook(client,
			nil, nil, timeout, false, decoder, logger.Sugar())

		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.True(t, res.AdmissionResponse.Allowed)
		require.ElementsMatch(t, withRemovedAnnotation([]jsonpatch.JsonPatchOperation{}), res.Patches)
	})

	t.Run("when invalid image should return failed and annotation reject with images list", func(t *testing.T) {
		//GIVEN
		ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName,
			Labels: map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled}}}
		mockImageValidator := mocks.ImageValidatorService{}
		mockImageValidator.Mock.On("Validate", mock.Anything, "test:test").
			Return(pkg.NewValidationFailedErr(errors.New("validation failed")))
		mockPodValidator := validate.NewPodValidator(&mockImageValidator)

		pod := newPodFix(nsName, nil)
		req := newRequestFix(t, pod, admissionv1.Create)
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()
		webhook := NewDefaultingWebhook(client,
			mockPodValidator, nil, timeout, false, decoder, logger.Sugar())

		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		mockImageValidator.AssertNumberOfCalls(t, "Validate", 1)
		require.NotNil(t, res)
		require.True(t, res.AdmissionResponse.Allowed)
		require.ElementsMatch(t, withAddRejectAndImagesAnnotation(patchWithAddLabel(pkg.ValidationStatusFailed)), res.Patches)
	})

	t.Run("when service unavailable and strict mode on should return pending and annotation reject", func(t *testing.T) {
		//GIVEN
		ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName,
			Labels: map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled}}}
		mockPodValidator := mocks.NewPodValidator(t)
		mockPodValidator.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
			Return(validate.ValidationResult{Status: validate.ServiceUnavailable}, nil)
		defer mockPodValidator.AssertExpectations(t)

		pod := newPodFix(nsName, nil)
		req := newRequestFix(t, pod, admissionv1.Create)
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()
		webhook := NewDefaultingWebhook(client,
			mockPodValidator, nil, timeout, StrictModeOn, decoder, logger.Sugar())

		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.Nil(t, res.Result)
		require.True(t, res.AdmissionResponse.Allowed)
		require.ElementsMatch(t, withAddRejectAnnotation(patchWithAddLabel(pkg.ValidationStatusPending)), res.Patches)
	})

	t.Run("when service unavailable and strict mode off should return pending", func(t *testing.T) {
		//GIVEN
		ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName,
			Labels: map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled}}}
		mockPodValidator := mocks.NewPodValidator(t)
		mockPodValidator.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
			Return(validate.ValidationResult{Status: validate.ServiceUnavailable}, nil)
		defer mockPodValidator.AssertExpectations(t)

		pod := newPodFix(nsName, nil)
		req := newRequestFix(t, pod, admissionv1.Create)
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()
		webhook := NewDefaultingWebhook(client,
			mockPodValidator, nil, timeout, StrictModeOff, decoder, logger.Sugar())

		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.Nil(t, res.Result)
		require.True(t, res.AdmissionResponse.Allowed)
		require.ElementsMatch(t, patchWithAddLabel(pkg.ValidationStatusPending), res.Patches)
	})

	t.Run("when service unavailable and strict mode on for user validation should return pending and annotation reject", func(t *testing.T) {
		//GIVEN
		ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name:   nsName,
			Labels: map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationUser},
			Annotations: map[string]string{
				pkg.NamespaceStrictModeAnnotation: strconv.FormatBool(true),
				pkg.NamespaceNotaryURLAnnotation:  "notary"},
		}}

		systemValidator := mocks.NewPodValidator(t)
		systemValidator.AssertNotCalled(t, "ValidatePod")
		defer systemValidator.AssertExpectations(t)

		userValidator := mocks.NewPodValidator(t)
		userValidator.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
			Return(validate.ValidationResult{Status: validate.ServiceUnavailable}, nil).Once()
		defer userValidator.AssertExpectations(t)

		userValidatorFactory := mocks.NewValidatorSvcFactory(t)
		userValidatorFactory.On("NewValidatorSvc", mock.Anything, mock.Anything, mock.Anything).
			Return(userValidator).Once()
		defer userValidatorFactory.AssertExpectations(t)

		pod := newPodFix(nsName, nil)
		req := newRequestFix(t, pod, admissionv1.Create)
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()
		webhook := NewDefaultingWebhook(client,
			systemValidator, userValidatorFactory, timeout, StrictModeOff, decoder, logger.Sugar())

		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.Nil(t, res.Result)
		require.True(t, res.AdmissionResponse.Allowed)
		require.ElementsMatch(t, withAddRejectAnnotation(patchWithAddLabel(pkg.ValidationStatusPending)), res.Patches)
	})

	t.Run("when service unavailable and strict mode off for user validation should return pending", func(t *testing.T) {
		//GIVEN
		ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name:   nsName,
			Labels: map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationUser},
			Annotations: map[string]string{
				pkg.NamespaceStrictModeAnnotation: strconv.FormatBool(false),
				pkg.NamespaceNotaryURLAnnotation:  "notary"},
		}}

		systemValidator := mocks.NewPodValidator(t)
		systemValidator.AssertNotCalled(t, "ValidatePod")
		defer systemValidator.AssertExpectations(t)

		userValidator := mocks.NewPodValidator(t)
		userValidator.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
			Return(validate.ValidationResult{Status: validate.ServiceUnavailable}, nil).Once()
		defer userValidator.AssertExpectations(t)

		userValidatorFactory := mocks.NewValidatorSvcFactory(t)
		userValidatorFactory.On("NewValidatorSvc", mock.Anything, mock.Anything, mock.Anything).
			Return(userValidator).Once()
		defer userValidatorFactory.AssertExpectations(t)

		pod := newPodFix(nsName, nil)
		req := newRequestFix(t, pod, admissionv1.Create)
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()
		webhook := NewDefaultingWebhook(client,
			systemValidator, userValidatorFactory, timeout, StrictModeOn, decoder, logger.Sugar())

		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.Nil(t, res.Result)
		require.True(t, res.AdmissionResponse.Allowed)
		require.ElementsMatch(t, patchWithAddLabel(pkg.ValidationStatusPending), res.Patches)
	})
}

func TestFlow_SomeInputStatuses_ShouldCallPodValidation(t *testing.T) {
	//GIVEN
	logger := test_helpers.NewTestZapLogger(t)
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	decoder := admission.NewDecoder(scheme)

	nsName := "test-namespace"
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName,
		Labels: map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled}}}

	type want struct {
		shouldCallValidate bool
		patches            []jsonpatch.JsonPatchOperation
	}
	tests := []struct {
		name        string
		operation   admissionv1.Operation
		inputLabels map[string]string
		want        want
	}{
		{
			name:        "update pod without label should pass with validation",
			operation:   admissionv1.Update,
			inputLabels: nil,
			want: want{
				shouldCallValidate: true,
				patches:            patchWithAddSuccessLabel(),
			},
		},
		{
			name:        "update pod with label Success should pass with validation",
			operation:   admissionv1.Update,
			inputLabels: map[string]string{pkg.PodValidationLabel: pkg.ValidationStatusSuccess},
			want: want{
				shouldCallValidate: true,
				patches:            []jsonpatch.JsonPatchOperation{},
			},
		},
		{
			name:        "update pod with unknown label should pass with validation",
			operation:   admissionv1.Update,
			inputLabels: map[string]string{pkg.PodValidationLabel: "some-unknown-label"},
			want: want{
				shouldCallValidate: true,
				patches:            patchWithReplaceSuccessLabel(),
			},
		},
		{
			name:        "update pod with label Failed should pass without validation",
			operation:   admissionv1.Update,
			inputLabels: map[string]string{pkg.PodValidationLabel: pkg.ValidationStatusFailed},
			want: want{
				shouldCallValidate: false,
				patches:            []jsonpatch.JsonPatchOperation(nil),
			},
		},
		{
			name:        "update pod with label Pending should pass without validation",
			operation:   admissionv1.Update,
			inputLabels: map[string]string{pkg.PodValidationLabel: pkg.ValidationStatusPending},
			want: want{
				shouldCallValidate: false,
				patches:            []jsonpatch.JsonPatchOperation(nil),
			},
		},
		// create always should be validated
		{
			name:        "create pod without label should pass with validation",
			operation:   admissionv1.Create,
			inputLabels: nil,
			want: want{
				shouldCallValidate: true,
				patches:            patchWithAddSuccessLabel(),
			},
		},
		{
			name:        "create pod with label Success should pass with validation",
			operation:   admissionv1.Create,
			inputLabels: map[string]string{pkg.PodValidationLabel: pkg.ValidationStatusSuccess},
			want: want{
				shouldCallValidate: true,
				patches:            []jsonpatch.JsonPatchOperation{},
			},
		},
		{
			name:        "create pod with unknown label should pass with validation",
			operation:   admissionv1.Create,
			inputLabels: map[string]string{pkg.PodValidationLabel: "some-unknown-label"},
			want: want{
				shouldCallValidate: true,
				patches:            patchWithReplaceSuccessLabel(),
			},
		},
		{
			name:        "create pod with label Failed should pass with validation",
			operation:   admissionv1.Create,
			inputLabels: map[string]string{pkg.PodValidationLabel: pkg.ValidationStatusFailed},
			want: want{
				shouldCallValidate: true,
				patches:            patchWithReplaceSuccessLabel(),
			},
		},
		{
			name:        "create pod with label Pending should pass wit validation",
			operation:   admissionv1.Create,
			inputLabels: map[string]string{pkg.PodValidationLabel: pkg.ValidationStatusPending},
			want: want{
				shouldCallValidate: true,
				patches:            patchWithReplaceSuccessLabel(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//GIVEN
			mockImageValidator := setupValidatorMock()
			mockPodValidator := validate.NewPodValidator(mockImageValidator)

			pod := newPodFix(nsName, tt.inputLabels)
			req := newRequestFix(t, pod, tt.operation)
			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()
			timeout := time.Second
			webhook := NewDefaultingWebhook(client,
				mockPodValidator, nil, timeout, false, decoder, logger.Sugar())

			//WHEN
			res := webhook.Handle(context.TODO(), req)

			//THEN
			require.NotNil(t, res)
			require.True(t, res.AdmissionResponse.Allowed)

			expectedValidateCalls := 0
			if tt.want.shouldCallValidate {
				expectedValidateCalls = 1
			}
			mockImageValidator.AssertNumberOfCalls(t, "Validate", expectedValidateCalls)
			require.Equal(t, tt.want.patches, res.Patches)
		})
	}
}

func TestFlow_NamespaceLabelsValidation(t *testing.T) {
	//GIVEN
	logger := zap.NewNop()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	decoder := admission.NewDecoder(scheme)
	timeout := time.Millisecond * 100

	testNs := "test-namespace"

	testsSkipValidation := []struct {
		name            string
		namespaceLabels map[string]string
	}{
		{
			name:            "Namespace without labels - validation is not needed",
			namespaceLabels: map[string]string{},
		},
		{
			name:            "Namespace with unknown value of validation label - validation is not needed",
			namespaceLabels: map[string]string{pkg.NamespaceValidationLabel: "unknown"},
		},
	}
	for _, tt := range testsSkipValidation {
		t.Run(tt.name, func(t *testing.T) {
			//GIVEN
			ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNs, Labels: tt.namespaceLabels}}
			pod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: testNs},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Image: "test:test"}}},
			}

			raw, err := json.Marshal(pod)
			require.NoError(t, err)

			req := admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
				Kind:   metav1.GroupVersionKind{Kind: PodType, Version: corev1.SchemeGroupVersion.Version},
				Object: runtime.RawExtension{Raw: raw},
			}}
			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()

			validationSvc := mocks.NewPodValidator(t)
			validationSvc.AssertNotCalled(t, "ValidatePod")
			defer validationSvc.AssertExpectations(t)
			webhook := NewDefaultingWebhook(client,
				validationSvc, nil, timeout, StrictModeOff, decoder, logger.Sugar())

			//WHEN
			res := webhook.Handle(context.TODO(), req)

			//THEN
			require.NotNil(t, res)
			require.NotNil(t, res.Result)
			assert.Contains(t, res.Result.Message, "validation is not needed for pod")
			assert.True(t, res.Allowed)
		})
	}

	testsWithValidation := []struct {
		name                          string
		namespaceValidationLabelValue string
	}{
		{
			name:                          "Namespace with enabled validation - validation is needed",
			namespaceValidationLabelValue: pkg.NamespaceValidationEnabled,
		},
		{
			name:                          "Namespace with enabled (system) validation - validation is needed",
			namespaceValidationLabelValue: pkg.NamespaceValidationSystem,
		},
		{
			name:                          "Namespace with enabled (user) validation - validation is needed",
			namespaceValidationLabelValue: pkg.NamespaceValidationUser,
		},
	}
	for _, tt := range testsWithValidation {
		t.Run(tt.name, func(t *testing.T) {
			//GIVEN
			ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name:        testNs,
				Labels:      map[string]string{pkg.NamespaceValidationLabel: tt.namespaceValidationLabelValue},
				Annotations: map[string]string{pkg.NamespaceNotaryURLAnnotation: "notary"},
			}}
			pod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: testNs},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Image: "test:test"}}},
			}

			raw, err := json.Marshal(pod)
			require.NoError(t, err)

			req := admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
				Kind:   metav1.GroupVersionKind{Kind: PodType, Version: corev1.SchemeGroupVersion.Version},
				Object: runtime.RawExtension{Raw: raw},
			}}
			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()

			podValidatorCallCount := 0

			systemValidator := mocks.NewPodValidator(t)
			systemValidator.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
				Return(validate.ValidationResult{Status: validate.Valid}, nil).
				Run(func(args mock.Arguments) {
					podValidatorCallCount++
				}).Maybe()
			defer systemValidator.AssertExpectations(t)

			userValidator := mocks.NewPodValidator(t)
			userValidator.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
				Return(validate.ValidationResult{Status: validate.Valid}, nil).
				Run(func(args mock.Arguments) {
					podValidatorCallCount++
				}).Maybe()
			defer userValidator.AssertExpectations(t)

			userValidatorFactory := mocks.NewValidatorSvcFactory(t)
			userValidatorFactory.On("NewValidatorSvc", mock.Anything, mock.Anything, mock.Anything).
				Return(userValidator).Maybe()
			defer userValidatorFactory.AssertExpectations(t)

			webhook := NewDefaultingWebhook(client,
				systemValidator, userValidatorFactory, timeout, StrictModeOff, decoder, logger.Sugar())

			//WHEN
			res := webhook.Handle(context.TODO(), req)

			//THEN
			require.NotNil(t, res)
			require.Nil(t, res.Result)
			assert.True(t, res.Allowed)
			// we expect that only one of the validators will be called
			assert.Equal(t, 1, podValidatorCallCount)
		})
	}
}

func TestFlow_UseSystemOrUserValidator(t *testing.T) {
	//GIVEN
	logger := zap.NewNop()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	decoder := admission.NewDecoder(scheme)
	timeout := time.Millisecond * 100

	testNs := "test-namespace"

	t.Run("Namespace with system validation", func(t *testing.T) {
		//GIVEN
		namespaceLabels := map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationSystem}
		ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNs, Labels: namespaceLabels}}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: testNs},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Image: "test:test"}}},
		}

		raw, err := json.Marshal(pod)
		require.NoError(t, err)

		req := admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
			Kind:   metav1.GroupVersionKind{Kind: PodType, Version: corev1.SchemeGroupVersion.Version},
			Object: runtime.RawExtension{Raw: raw},
		}}
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()

		// system validator should be called
		validationSvc := mocks.NewPodValidator(t)
		validationSvc.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
			Return(validate.ValidationResult{Status: validate.Valid}, nil).Once()
		defer validationSvc.AssertExpectations(t)

		// user validator (factory) should not be called
		userValidatorFactory := mocks.NewValidatorSvcFactory(t)
		userValidatorFactory.AssertNotCalled(t, "NewValidatorSvc")
		defer userValidatorFactory.AssertExpectations(t)

		webhook := NewDefaultingWebhook(client,
			validationSvc, userValidatorFactory, timeout, StrictModeOff, decoder, logger.Sugar())

		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.Nil(t, res.Result)
		assert.True(t, res.Allowed)
	})

	t.Run("Namespace with user validation", func(t *testing.T) {
		//GIVEN
		namespaceLabels := map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationUser}
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:        testNs,
				Labels:      namespaceLabels,
				Annotations: map[string]string{pkg.NamespaceNotaryURLAnnotation: "notary"}},
		}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: testNs},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Image: "test:test"}}},
		}

		raw, err := json.Marshal(pod)
		require.NoError(t, err)

		req := admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
			Kind:   metav1.GroupVersionKind{Kind: PodType, Version: corev1.SchemeGroupVersion.Version},
			Object: runtime.RawExtension{Raw: raw},
		}}
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()

		// system validator should not be called
		systemValidator := mocks.NewPodValidator(t)
		systemValidator.AssertNotCalled(t, "ValidatePod")
		defer systemValidator.AssertExpectations(t)

		// user validator and its factory should be called exactly once
		userValidator := mocks.NewPodValidator(t)
		userValidator.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
			Return(validate.ValidationResult{Status: validate.Valid}, nil).Once()
		defer userValidator.AssertExpectations(t)

		userValidatorFactory := mocks.NewValidatorSvcFactory(t)
		userValidatorFactory.On("NewValidatorSvc", mock.Anything, mock.Anything, mock.Anything).
			Return(userValidator).Once()
		defer userValidatorFactory.AssertExpectations(t)

		webhook := NewDefaultingWebhook(client,
			systemValidator, userValidatorFactory, timeout, StrictModeOff, decoder, logger.Sugar())

		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.Nil(t, res.Result)
		assert.True(t, res.Allowed)
	})
}

func TestFlow_UserValidatorGetValuesFromNamespaceAnnotations(t *testing.T) {
	//GIVEN
	logger := zap.NewNop()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	decoder := admission.NewDecoder(scheme)
	timeout := time.Millisecond * 100

	testNs := "test-namespace"

	tests := []struct {
		name              string
		notaryUrl         *string
		allowedRegistries *string
		notaryTimeout     *string
		success           bool
		errorMessage      string
	}{
		{
			name:      "User validation get notary url from namespace annotation",
			notaryUrl: ptr.To("http://test.notary.url"),
			success:   true,
		},
		{
			name:              "User validation get allowed registries from namespace annotation",
			notaryUrl:         ptr.To("http://test.notary.url"),
			allowedRegistries: ptr.To("ala,ma,    \nkota"),
			success:           true,
		},
		{
			name:          "User validation get notary timeout from namespace annotation",
			notaryUrl:     ptr.To("http://test.notary.url"),
			notaryTimeout: ptr.To("22s"),
			success:       true,
		},
		{
			name:              "User validation get all params from namespace annotation",
			notaryUrl:         ptr.To("http://another.test.notary.url"),
			allowedRegistries: ptr.To("maka,paka"),
			notaryTimeout:     ptr.To("77h"),
			success:           true,
		},
		{
			name:              "User validation return error for namespace without notary url annotation",
			allowedRegistries: ptr.To("maka,paka"),
			notaryTimeout:     ptr.To("77h"),
			success:           false,
			errorMessage:      "notary URL is not set",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//GIVEN
			namespaceLabels := map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationUser}
			namespaceAnnotations := map[string]string{}
			expectedNotaryURL := ""
			expectedAllowedRegistries := helpers.DefaultUserAllowedRegistries
			expectedNotaryTimeout, _ := time.ParseDuration(helpers.DefaultUserNotaryTimeoutString)
			if tt.notaryUrl != nil {
				expectedNotaryURL = *tt.notaryUrl
				namespaceAnnotations[pkg.NamespaceNotaryURLAnnotation] = *tt.notaryUrl
			}
			if tt.allowedRegistries != nil {
				expectedAllowedRegistries = *tt.allowedRegistries
				namespaceAnnotations[pkg.NamespaceAllowedRegistriesAnnotation] = *tt.allowedRegistries
			}
			if tt.notaryTimeout != nil {
				expectedNotaryTimeout, _ = time.ParseDuration(*tt.notaryTimeout)
				namespaceAnnotations[pkg.NamespaceNotaryTimeoutAnnotation] = *tt.notaryTimeout
			}
			ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNs, Labels: namespaceLabels, Annotations: namespaceAnnotations}}
			pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: testNs},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Image: "test:test"}}},
			}

			raw, err := json.Marshal(pod)
			require.NoError(t, err)

			req := admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
				Kind:   metav1.GroupVersionKind{Kind: PodType, Version: corev1.SchemeGroupVersion.Version},
				Object: runtime.RawExtension{Raw: raw},
			}}
			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()

			// system validator should not be called
			systemValidator := mocks.NewPodValidator(t)
			systemValidator.AssertNotCalled(t, "ValidatePod")
			defer systemValidator.AssertExpectations(t)

			userValidator := mocks.NewPodValidator(t)
			userValidator.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
				Return(validate.ValidationResult{Status: validate.Valid}, nil).Maybe()
			defer userValidator.AssertExpectations(t)

			// user validator factory should be called with proper data
			userValidatorFactory := mocks.NewValidatorSvcFactory(t)
			userValidatorFactory.On("NewValidatorSvc", mock.Anything, mock.Anything, mock.Anything).
				Return(userValidator).
				Run(func(args mock.Arguments) {
					argNotaryURL := args.Get(0)
					argAllowedRegistries := args.Get(1)
					argNotaryTimeout := args.Get(2)
					require.Equal(t, expectedNotaryURL, argNotaryURL)
					require.Equal(t, expectedAllowedRegistries, argAllowedRegistries)
					require.Equal(t, expectedNotaryTimeout, argNotaryTimeout)
				}).Maybe()
			defer userValidatorFactory.AssertExpectations(t)

			webhook := NewDefaultingWebhook(client,
				systemValidator, userValidatorFactory, timeout, StrictModeOff, decoder, logger.Sugar())

			//WHEN
			res := webhook.Handle(context.TODO(), req)

			//THEN
			require.NotNil(t, res)
			if tt.success {
				assert.True(t, res.Allowed)
				require.Nil(t, res.Result)
			} else {
				assert.False(t, res.Allowed)
				require.NotNil(t, res.Result)
				require.Contains(t, res.Result.Message, tt.errorMessage)
			}
		})
	}
}

func TestHandleTimeout(t *testing.T) {
	//GIVEN
	logger := zap.NewNop()
	ctxLogger := helpers.LoggerToContext(context.TODO(), logger.Sugar())
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	decoder := admission.NewDecoder(scheme)
	timeout := time.Millisecond * 100

	testNs := "test-namespace"

	tests := []struct {
		name                          string
		systemStrictMode              bool
		userStrictMode                bool // opposite value of system strict mode for the test if we get proper value
		namespaceValidationLabelValue string
		expectedPatches               []jsonpatch.Operation
	}{
		{
			name:                          "Handle timeout for system validation and strict mode off",
			namespaceValidationLabelValue: pkg.NamespaceValidationSystem,
			systemStrictMode:              StrictModeOff,
			userStrictMode:                StrictModeOn,
			expectedPatches: []jsonpatch.Operation{
				{
					Operation: "add",
					Path:      "/metadata/labels",
					Value:     map[string]interface{}{"pods.warden.kyma-project.io/validate": "pending"},
				}},
		},
		{
			name:                          "Handle timeout for system validation and strict mode on",
			namespaceValidationLabelValue: pkg.NamespaceValidationSystem,
			systemStrictMode:              StrictModeOn,
			userStrictMode:                StrictModeOff,
			expectedPatches: []jsonpatch.Operation{
				{
					Operation: "add",
					Path:      "/metadata/labels",
					Value:     map[string]interface{}{"pods.warden.kyma-project.io/validate": "pending"},
				},
				{
					Operation: "add",
					Path:      "/metadata/annotations",
					Value:     map[string]interface{}{"pods.warden.kyma-project.io/validate-reject": "reject"},
				}},
		},
		{
			name:                          "Handle timeout for user validation and strict mode off",
			namespaceValidationLabelValue: pkg.NamespaceValidationUser,
			systemStrictMode:              StrictModeOn,
			userStrictMode:                StrictModeOff,
			expectedPatches: []jsonpatch.Operation{
				{
					Operation: "add",
					Path:      "/metadata/labels",
					Value:     map[string]interface{}{"pods.warden.kyma-project.io/validate": "pending"},
				}},
		},
		{
			name:                          "Handle timeout for user validation and strict mode on",
			namespaceValidationLabelValue: pkg.NamespaceValidationUser,
			systemStrictMode:              StrictModeOff,
			userStrictMode:                StrictModeOn,
			expectedPatches: []jsonpatch.Operation{
				{
					Operation: "add",
					Path:      "/metadata/labels",
					Value:     map[string]interface{}{"pods.warden.kyma-project.io/validate": "pending"},
				},
				{
					Operation: "add",
					Path:      "/metadata/annotations",
					Value:     map[string]interface{}{"pods.warden.kyma-project.io/validate-reject": "reject"},
				}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//GIVEN
			ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name:        testNs,
				Labels:      map[string]string{pkg.NamespaceValidationLabel: tt.namespaceValidationLabelValue},
				Annotations: map[string]string{pkg.NamespaceStrictModeAnnotation: strconv.FormatBool(tt.userStrictMode)},
			}}
			pod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: testNs},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Image: "test:test"}}},
			}

			raw, err := json.Marshal(pod)
			require.NoError(t, err)

			req := admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
				Kind:   metav1.GroupVersionKind{Kind: PodType, Version: corev1.SchemeGroupVersion.Version},
				Object: runtime.RawExtension{Raw: raw},
			}}
			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()

			webhook := NewDefaultingWebhook(client,
				nil, nil, timeout, tt.systemStrictMode, decoder, logger.Sugar())

			//WHEN
			res := webhook.handleTimeout(ctxLogger, errors.New(""), req)

			//THEN
			require.NotNil(t, res)
			assert.True(t, res.Allowed)
			require.NotNil(t, res.Result)
			require.Contains(t, res.Result.Message, "request exceeded desired timeout")
			require.NotNil(t, res.Patches)
			require.ElementsMatch(t, tt.expectedPatches, res.Patches)
		})
	}
}

func setupValidatorMock() *mocks.ImageValidatorService {
	mockValidator := mocks.ImageValidatorService{}
	mockValidator.Mock.On("Validate", mock.Anything, "test:test").
		Return(nil)
	return &mockValidator
}

func newRequestFix(t *testing.T, pod corev1.Pod, operation admissionv1.Operation) admission.Request {
	raw, err := json.Marshal(pod)
	require.NoError(t, err)

	req := admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
		Kind:      metav1.GroupVersionKind{Kind: PodType, Version: corev1.SchemeGroupVersion.Version},
		Operation: operation,
		Object:    runtime.RawExtension{Raw: raw},
	}}
	return req
}

func newPodFix(nsName string, labels map[string]string) corev1.Pod {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod", Namespace: nsName,
			Labels: labels,
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Image: "test:test"}}},
	}
	return pod
}

func patchWithAddLabel(labelValue string) []jsonpatch.JsonPatchOperation {
	return []jsonpatch.JsonPatchOperation{
		{
			Operation: "add",
			Path:      "/metadata/labels",
			Value: map[string]interface{}{
				"pods.warden.kyma-project.io/validate": labelValue,
			},
		},
	}
}

func patchWithAddSuccessLabel() []jsonpatch.JsonPatchOperation {
	return patchWithAddLabel(pkg.ValidationStatusSuccess)
}

func patchWithReplaceSuccessLabel() []jsonpatch.JsonPatchOperation {
	return []jsonpatch.JsonPatchOperation{
		{
			Operation: "replace",
			Path:      "/metadata/labels/pods.warden.kyma-project.io~1validate",
			Value:     "success",
		},
	}
}

func withRemovedAnnotation(patch []jsonpatch.JsonPatchOperation) []jsonpatch.JsonPatchOperation {
	return append(patch, jsonpatch.JsonPatchOperation{
		Operation: "remove",
		Path:      "/metadata/annotations",
	})
}

func withAddRejectAnnotation(patch []jsonpatch.JsonPatchOperation) []jsonpatch.JsonPatchOperation {
	return append(patch, jsonpatch.JsonPatchOperation{
		Operation: "add",
		Path:      "/metadata/annotations",
		Value: map[string]interface{}{
			"pods.warden.kyma-project.io/validate-reject": "reject",
		},
	})
}

func withAddRejectAndImagesAnnotation(patch []jsonpatch.JsonPatchOperation) []jsonpatch.JsonPatchOperation {
	return append(patch, jsonpatch.JsonPatchOperation{
		Operation: "add",
		Path:      "/metadata/annotations",
		Value: map[string]interface{}{
			"pods.warden.kyma-project.io/invalid-images":  "test:test",
			"pods.warden.kyma-project.io/validate-reject": "reject",
		},
	})
}
