package admission

import (
	"context"
	"encoding/json"
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
	"net/http"
	"net/http/httptest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
	"time"
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
	decoder, err := admission.NewDecoder(scheme)
	require.NoError(t, err)
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
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns, &pod).Build()

	t.Run("Success", func(t *testing.T) {
		//GIVEN
		validationSvc := mocks.NewPodValidator(t)
		validationSvc.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
			After(timeout/2).
			Return(validate.Valid, nil).Once()
		defer validationSvc.AssertExpectations(t)
		webhook := NewDefaultingWebhook(client, validationSvc, timeout, StrictModeOff, logger.Sugar())
		require.NoError(t, webhook.InjectDecoder(decoder))

		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.Nil(t, res.Result)
		require.True(t, res.AdmissionResponse.Allowed)
	})

	t.Run("Defaulting webhook timeout, allowed", func(t *testing.T) {
		validationSvc := mocks.NewPodValidator(t)
		validationSvc.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
			After(timeout*2).
			Return(validate.Valid, nil).Once()
		defer validationSvc.AssertExpectations(t)
		webhook := NewDefaultingWebhook(client, validationSvc, timeout, StrictModeOff, logger.Sugar())
		require.NoError(t, webhook.InjectDecoder(decoder))
		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.NotNil(t, res.Result)
		assert.Equal(t, int32(http.StatusOK), res.Result.Code)
		assert.Contains(t, res.Result.Reason, "request exceeded desired timeout")
	})

	t.Run("Defaulting webhook timeout strict mode on, errored", func(t *testing.T) {
		validationSvc := mocks.NewPodValidator(t)
		validationSvc.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
			After(timeout*2).
			Return(validate.Valid, nil).Once()
		defer validationSvc.AssertExpectations(t)
		webhook := NewDefaultingWebhook(client, validationSvc, timeout, StrictModeOn, logger.Sugar())
		require.NoError(t, webhook.InjectDecoder(decoder))
		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.NotNil(t, res.Result, "response is ok")
		assert.Equal(t, int32(http.StatusRequestTimeout), res.Result.Code)
		assert.Contains(t, res.Result.Message, "context deadline exceeded")
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
		webhook := NewDefaultingWebhook(client, validationSvc, timeout, StrictModeOff, logger.Sugar())
		require.NoError(t, webhook.InjectDecoder(decoder))
		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.NotNil(t, res.Result)
		assert.Equal(t, int32(http.StatusOK), res.Result.Code)
		assert.Contains(t, res.Result.Reason, "request exceeded desired timeout")
		require.InDelta(t, timeout.Seconds(), time.Since(start).Seconds(), 0.1, "timeout duration is not respected")
	})
}

func TestStrictMode(t *testing.T) {
	logger := zap.NewNop()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	decoder, err := admission.NewDecoder(scheme)
	require.NoError(t, err)
	timeout := time.Millisecond * 30

	testNs := "test-namespace"
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNs, Labels: map[string]string{
		pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled,
	}}}

	pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: testNs},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Image: "test:test"}}},
	}
	req := newRequestFix(t, pod, admissionv1.Create)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns, &pod).Build()

	t.Run("Strict mode on", func(t *testing.T) {
		validationSvc := mocks.NewPodValidator(t)

		validationSvc.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
			Return(validate.ServiceUnavailable, nil).Once()
		defer validationSvc.AssertExpectations(t)
		webhook := NewDefaultingWebhook(client, validationSvc, timeout, StrictModeOn, logger.Sugar())
		require.NoError(t, webhook.InjectDecoder(decoder))

		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.Nil(t, res.Result)
		require.True(t, res.AdmissionResponse.Allowed)
		require.Len(t, res.Patches, 1)
		require.Len(t, res.Patches[0].Value, 1)
		require.Equal(t, "add", res.Patches[0].Operation)
		require.Equal(t, "/metadata/labels", res.Patches[0].Path)
		patchValue := (res.Patches[0].Value).(map[string]interface{})
		require.Contains(t, patchValue, pkg.PodValidationLabel)
		require.Equal(t, pkg.ValidationStatusReject, patchValue[pkg.PodValidationLabel])
	})

	t.Run("Strict mode off", func(t *testing.T) {
		validationSvc := mocks.NewPodValidator(t)

		validationSvc.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
			Return(validate.ServiceUnavailable, nil).Once()
		defer validationSvc.AssertExpectations(t)
		webhook := NewDefaultingWebhook(client, validationSvc, timeout, StrictModeOff, logger.Sugar())
		require.NoError(t, webhook.InjectDecoder(decoder))

		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.Nil(t, res.Result)
		require.True(t, res.AdmissionResponse.Allowed)
		require.Len(t, res.Patches, 1)
		require.Len(t, res.Patches[0].Value, 1)
		require.Equal(t, "add", res.Patches[0].Operation)
		require.Equal(t, "/metadata/labels", res.Patches[0].Path)
		patchValue := (res.Patches[0].Value).(map[string]interface{})
		require.Contains(t, patchValue, pkg.PodValidationLabel)
		require.Equal(t, pkg.ValidationStatusPending, patchValue[pkg.PodValidationLabel])
	})
}

func TestFlow(t *testing.T) {
	//GIVEN
	logger := test_helpers.NewTestZapLogger(t)
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	decoder, err := admission.NewDecoder(scheme)
	require.NoError(t, err)

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
		{ // TODO: it will be Failed status with annotation Reject (now should be impossible on webhook input - it's like unknown label)
			name:        "update pod with label Reject should pass with validation",
			operation:   admissionv1.Update,
			inputLabels: map[string]string{pkg.PodValidationLabel: pkg.ValidationStatusReject},
			want: want{
				shouldCallValidate: true,
				patches:            patchWithReplaceSuccessLabel(),
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
			mockPodValidator := validate.NewPodValidator(&mockImageValidator)

			pod := newPodFix(nsName, tt.inputLabels)
			req := newRequestFix(t, pod, tt.operation)
			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns, &pod).Build()

			timeout := time.Second
			webhook := NewDefaultingWebhook(client, mockPodValidator, timeout, false, logger.Sugar())
			require.NoError(t, webhook.InjectDecoder(decoder))

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

			//if tt.want.shouldCallValidate {
			//	mockImageValidator.AssertNumberOfCalls(t, "Validate", 1)
			//	// with validation we should have patch if input status was not Success
			//	if tt.inputLabels != nil && tt.inputLabels[pkg.PodValidationLabel] == pkg.ValidationStatusSuccess {
			//		require.Equal(t, 0, len(res.Patches))
			//	} else {
			//		require.Equal(t, 1, len(res.Patches))
			//		if res.Patches[0].Operation == "replace" {
			//			patchValue := (res.Patches[0].Value).(string)
			//			require.Equal(t, pkg.ValidationStatusSuccess, patchValue)
			//		} else {
			//			patchValue := (res.Patches[0].Value).(map[string]interface{})
			//			require.Contains(t, patchValue, pkg.PodValidationLabel)
			//			require.Equal(t, pkg.ValidationStatusSuccess, patchValue[pkg.PodValidationLabel])
			//		}
			//	}
			//} else {
			//	mockImageValidator.AssertNumberOfCalls(t, "Validate", 0)
			//	// without validation we should have no patch
			//	require.Equal(t, 0, len(res.Patches))
			//}
		})
	}
}

func setupValidatorMock() mocks.ImageValidatorService {
	mockValidator := mocks.ImageValidatorService{}
	mockValidator.Mock.On("Validate", mock.Anything, "test:test").
		Return(nil)
	return mockValidator
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

func patchWithAddSuccessLabel() []jsonpatch.JsonPatchOperation {
	return []jsonpatch.JsonPatchOperation{
		{
			Operation: "add",
			Path:      "/metadata/labels",
			Value: map[string]interface{}{
				"pods.warden.kyma-project.io/validate": "success",
			},
		},
	}
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
