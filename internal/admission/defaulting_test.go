package admission

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kyma-project/warden/internal/test_helpers"
	"github.com/kyma-project/warden/internal/validate"
	"github.com/kyma-project/warden/internal/validate/mocks"
	"github.com/kyma-project/warden/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
	logger := test_helpers.NewTestZapLogger(t)
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
		webhook := NewDefaultingWebhook(client, validationSvc, timeout, logger.Sugar())
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
		webhook := NewDefaultingWebhook(client, validationSvc, timeout, logger.Sugar())
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
		webhook := NewDefaultingWebhook(client, validationSvc, timeout, logger.Sugar())
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

func TestFlow(t *testing.T) {
	//GIVEN
	logger := test_helpers.NewTestZapLogger(t)
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	decoder, err := admission.NewDecoder(scheme)
	require.NoError(t, err)

	testNs := "test-namespace"
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNs, Labels: map[string]string{
		pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled,
	}}}

	type args struct {
		labels    map[string]string
		imageName string
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
			name: "label Failed should pass without validation and unchanged",
			args: args{
				labels:    map[string]string{pkg.PodValidationLabel: pkg.ValidationStatusFailed},
				imageName: pkg.ValidationStatusFailed,
			},
			want: want{
				patchesCount: 0,
			},
		},
		{
			name: "label Success should pass with validation",
			args: args{
				labels:    map[string]string{pkg.PodValidationLabel: pkg.ValidationStatusSuccess},
				imageName: pkg.ValidationStatusSuccess,
			},
			want: want{
				patchesCount: 0,
			},
		},
		{
			name: "without label should pass with validation and set Success",
			args: args{
				labels:    nil,
				imageName: pkg.ValidationStatusSuccess,
			},
			want: want{
				patchesCount:     1,
				validationStatus: pkg.ValidationStatusSuccess,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//GIVEN
			pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: testNs},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Image: fmt.Sprintf("%s:test", tt.args.imageName)}}},
			}
			if tt.args.labels != nil {
				pod.ObjectMeta.Labels = tt.args.labels
			}

			raw, err := json.Marshal(pod)
			require.NoError(t, err)

			req := admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Kind:   metav1.GroupVersionKind{Kind: PodType, Version: corev1.SchemeGroupVersion.Version},
					Object: runtime.RawExtension{Raw: raw},
				}}
			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns, &pod).Build()

			mockValidator := mocks.ImageValidatorService{}
			mockValidator.Mock.On("Validate", mock.Anything, pkg.ValidationStatusFailed+":test").
				Panic("unexpected validation call!")
			mockValidator.Mock.On("Validate", mock.Anything, pkg.ValidationStatusSuccess+":test").
				Return(nil).Once()
			podValidator := validate.NewPodValidator(&mockValidator)
			timeout := time.Second
			webhook := NewDefaultingWebhook(client, podValidator, timeout, logger.Sugar())
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
