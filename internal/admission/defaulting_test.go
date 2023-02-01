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
		webhook := NewDefaultingWebhook(client, validationSvc, timeout, false, logger.Sugar())
		require.NoError(t, webhook.InjectDecoder(decoder))

		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.Nil(t, res.Result)
		require.True(t, res.AdmissionResponse.Allowed)
	})

	t.Run("Defaulting webhook timeout", func(t *testing.T) {
		validationSvc := mocks.NewPodValidator(t)
		validationSvc.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
			After(timeout*2).
			Return(validate.Valid, nil).Once()
		defer validationSvc.AssertExpectations(t)
		webhook := NewDefaultingWebhook(client, validationSvc, timeout, false, logger.Sugar())
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
		webhook := NewDefaultingWebhook(client, validationSvc, timeout, false, logger.Sugar())
		require.NoError(t, webhook.InjectDecoder(decoder))
		//WHEN
		res := webhook.Handle(context.TODO(), req)

		//THEN
		require.NotNil(t, res)
		require.NotNil(t, res.Result, "response is ok")
		assert.Equal(t, int32(http.StatusRequestTimeout), res.Result.Code)
		assert.Contains(t, res.Result.Message, "context deadline exceeded")
		require.InDelta(t, timeout.Seconds(), time.Since(start).Seconds(), 0.1, "timeout duration is not respected")
	})
}

func TestStrictMode(t *testing.T) {
	tests := []struct {
		name        string
		strictMode  bool
		wantedLabel string
	}{
		{
			name:        "Strict mode on",
			strictMode:  true,
			wantedLabel: pkg.ValidationStatusReject,
		},
		{
			name:        "Strict mode off",
			strictMode:  false,
			wantedLabel: pkg.ValidationStatusPending,
		},
	}

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
	req := newRequestFix(t, pod)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns, &pod).Build()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validationSvc := mocks.NewPodValidator(t)

			validationSvc.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
				Return(validate.ServiceUnavailable, nil).Once()
			defer validationSvc.AssertExpectations(t)
			webhook := NewDefaultingWebhook(client, validationSvc, timeout, tt.strictMode, logger.Sugar())
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
			require.Equal(t, tt.wantedLabel, patchValue[pkg.PodValidationLabel])
		})
	}
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

	type args struct {
		labels               map[string]string
		shouldNotBeValidated bool
	}
	type want struct {
		patchesCount     int
		validationStatus string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "pod without label should pass with validation and set Success",
			args: args{
				labels:               nil,
				shouldNotBeValidated: false,
			},
			want: want{
				patchesCount:     1,
				validationStatus: pkg.ValidationStatusSuccess,
			},
		},
		{
			name: "pod with label Success should pass with validation",
			args: args{
				labels:               map[string]string{pkg.PodValidationLabel: pkg.ValidationStatusSuccess},
				shouldNotBeValidated: false,
			},
			want: want{
				patchesCount: 0,
			},
		},
		{
			name: "pod with label Failed should pass without validation and unchanged",
			args: args{
				labels:               map[string]string{pkg.PodValidationLabel: pkg.ValidationStatusFailed},
				shouldNotBeValidated: true,
			},
			want: want{
				patchesCount: 0,
			},
		},
		{
			name: "pod with label Reject should pass without validation and unchanged",
			args: args{
				labels:               map[string]string{pkg.PodValidationLabel: pkg.ValidationStatusReject},
				shouldNotBeValidated: true,
			},
			want: want{
				patchesCount: 0,
			},
		},
		{
			name: "pod with label Pending should pass without validation and unchanged",
			args: args{
				labels:               map[string]string{pkg.PodValidationLabel: pkg.ValidationStatusPending},
				shouldNotBeValidated: true,
			},
			want: want{
				patchesCount: 0,
			},
		},
		{
			name: "pod with unknown label should pass without validation and unchanged",
			args: args{
				labels:               map[string]string{pkg.PodValidationLabel: "some-unknown-label"},
				shouldNotBeValidated: true,
			},
			want: want{
				patchesCount: 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//GIVEN
			podValidator := setupValidatorMock(t)

			pod := newPodFix(nsName, tt.args.shouldNotBeValidated, tt.args.labels)
			req := newRequestFix(t, pod)
			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns, &pod).Build()

			timeout := time.Second
			webhook := NewDefaultingWebhook(client, podValidator, timeout, false, logger.Sugar())
			require.NoError(t, webhook.InjectDecoder(decoder))

			//WHEN
			res := webhook.Handle(context.TODO(), req)

			//THEN
			require.NotNil(t, res)
			require.True(t, res.AdmissionResponse.Allowed)
			require.Equal(t, tt.want.patchesCount, len(res.Patches))
			if tt.want.patchesCount > 0 {
				patchValue := (res.Patches[0].Value).(map[string]interface{})
				require.Contains(t, patchValue, pkg.PodValidationLabel)
				require.Equal(t, tt.want.validationStatus, patchValue[pkg.PodValidationLabel])
			}
		})
	}
}

func setupValidatorMock(t *testing.T) validate.PodValidator {
	mockValidator := mocks.ImageValidatorService{}
	mockValidator.Mock.On("Validate", mock.Anything, "test:should-not-be-validated").
		Run(func(args mock.Arguments) {
			t.Fatal("unexpected validation call!")
		})
	mockValidator.Mock.On("Validate", mock.Anything, "test:can-be-validated").
		Return(nil)
	podValidator := validate.NewPodValidator(&mockValidator)
	return podValidator
}

func newRequestFix(t *testing.T, pod corev1.Pod) admission.Request {
	raw, err := json.Marshal(pod)
	require.NoError(t, err)

	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Kind:   metav1.GroupVersionKind{Kind: PodType, Version: corev1.SchemeGroupVersion.Version},
			Object: runtime.RawExtension{Raw: raw},
		}}
	return req
}

func newPodFix(nsName string, shouldNotBeValidated bool, labels map[string]string) corev1.Pod {
	imageName := "test:can-be-validated"
	if shouldNotBeValidated {
		imageName = "test:should-not-be-validated"
	}
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod", Namespace: nsName,
			Labels: labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Image: imageName}}},
	}
	return pod
}
