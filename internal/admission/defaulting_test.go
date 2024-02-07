package admission

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNs, Labels: map[string]string{
		pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled,
	}}}
	pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: testNs},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Image: "test:test"}}},
	}

	raw, err := json.Marshal(pod)
	require.NoError(t, err)

	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
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
		webhook := NewDefaultingWebhook(client, validationSvc, timeout, StrictModeOff, decoder, logger.Sugar())

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
		webhook := NewDefaultingWebhook(client, validationSvc, timeout, StrictModeOff, decoder, logger.Sugar())
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
		webhook := NewDefaultingWebhook(client, validationSvc, timeout, StrictModeOn, decoder, logger.Sugar())
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
		webhook := NewDefaultingWebhook(client, validationSvc, timeout, StrictModeOff, decoder, logger.Sugar())
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
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName, Labels: map[string]string{
		pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled,
	}}}

	t.Run("when valid image should return success", func(t *testing.T) {
		//GIVEN
		mockImageValidator := mocks.ImageValidatorService{}
		mockImageValidator.Mock.On("Validate", mock.Anything, "test:test").
			Return(nil)
		mockPodValidator := validate.NewPodValidator(&mockImageValidator)

		pod := newPodFix(nsName, nil)
		req := newRequestFix(t, pod, admissionv1.Create)
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()
		webhook := NewDefaultingWebhook(client, mockPodValidator, timeout, false, decoder, logger.Sugar())

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
		mockImageValidator := mocks.ImageValidatorService{}
		mockImageValidator.Mock.On("Validate", mock.Anything, "test:test").
			Return(nil)
		mockPodValidator := validate.NewPodValidator(&mockImageValidator)

		pod := newPodFix(nsName, nil)
		pod.ObjectMeta.Annotations = map[string]string{annotations.PodValidationRejectAnnotation: annotations.ValidationReject}
		req := newRequestFix(t, pod, admissionv1.Create)
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()
		webhook := NewDefaultingWebhook(client, mockPodValidator, timeout, false, decoder, logger.Sugar())

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
		pod := newPodFix(nsName, map[string]string{pkg.PodValidationLabel: pkg.ValidationStatusPending})
		pod.ObjectMeta.Annotations = map[string]string{annotations.PodValidationRejectAnnotation: annotations.ValidationReject}
		req := newRequestFix(t, pod, admissionv1.Update)
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()
		webhook := NewDefaultingWebhook(client, nil, timeout, false, decoder, logger.Sugar())

		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.True(t, res.AdmissionResponse.Allowed)
		require.ElementsMatch(t, withRemovedAnnotation([]jsonpatch.JsonPatchOperation{}), res.Patches)
	})

	t.Run("when invalid image should return failed and annotation reject with images list", func(t *testing.T) {
		//GIVEN
		mockImageValidator := mocks.ImageValidatorService{}
		mockImageValidator.Mock.On("Validate", mock.Anything, "test:test").
			Return(pkg.NewValidationFailedErr(errors.New("validation failed")))
		mockPodValidator := validate.NewPodValidator(&mockImageValidator)

		pod := newPodFix(nsName, nil)
		req := newRequestFix(t, pod, admissionv1.Create)
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()
		webhook := NewDefaultingWebhook(client, mockPodValidator, timeout, false, decoder, logger.Sugar())

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
		mockPodValidator := mocks.NewPodValidator(t)
		mockPodValidator.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
			Return(validate.ValidationResult{Status: validate.ServiceUnavailable}, nil)
		defer mockPodValidator.AssertExpectations(t)

		pod := newPodFix(nsName, nil)
		req := newRequestFix(t, pod, admissionv1.Create)
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()
		webhook := NewDefaultingWebhook(client, mockPodValidator, timeout, StrictModeOn, decoder, logger.Sugar())

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
		mockPodValidator := mocks.NewPodValidator(t)
		mockPodValidator.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
			Return(validate.ValidationResult{Status: validate.ServiceUnavailable}, nil)
		defer mockPodValidator.AssertExpectations(t)

		pod := newPodFix(nsName, nil)
		req := newRequestFix(t, pod, admissionv1.Create)
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns).Build()
		webhook := NewDefaultingWebhook(client, mockPodValidator, timeout, StrictModeOff, decoder, logger.Sugar())

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
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName, Labels: map[string]string{
		pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled,
	}}}

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
			webhook := NewDefaultingWebhook(client, mockPodValidator, timeout, false, decoder, logger.Sugar())

			//WHEN
			res := webhook.Handle(context.TODO(), req)

			//THEN
			require.NotNil(t, res)
			require.True(t, res.AdmissionResponse.Allowed)

			if tt.want.shouldCallValidate {
				mockImageValidator.AssertNumberOfCalls(t, "Validate", 1)
			} else {
				mockImageValidator.AssertNumberOfCalls(t, "Validate", 0)
			}
			require.Equal(t, tt.want.patches, res.Patches)
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

	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
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
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Image: "test:test"}}},
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
