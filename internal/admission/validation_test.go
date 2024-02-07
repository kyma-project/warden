package admission

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kyma-project/warden/internal/annotations"
	"github.com/kyma-project/warden/internal/test_helpers"
	"github.com/kyma-project/warden/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestValidationWebhook(t *testing.T) {
	scheme := runtime.NewScheme()
	decoder := admission.NewDecoder(scheme)
	log := test_helpers.NewTestZapLogger(t).Sugar()
	webhook := NewValidationWebhook(log, decoder)

	testCases := []struct {
		name            string
		pod             *corev1.Pod
		expectedStatus  int32
		expectedMessage string
	}{
		{
			name: "Pod should be rejected",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod",
					Annotations: map[string]string{
						annotations.PodValidationRejectAnnotation: annotations.ValidationReject,
						annotations.InvalidImagesAnnotation:       annotations.InvalidImagesAnnotation,
					}},
			},
			expectedStatus:  int32(http.StatusForbidden),
			expectedMessage: "images validation failed",
		},
		{
			name: "Pod should be allowed, any label",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod",
					Labels: map[string]string{
						pkg.PodValidationLabel: "anything",
					}},
			},
			expectedStatus:  int32(http.StatusOK),
			expectedMessage: "nothing to do",
		},
		{
			name: "Pod should be allowed, no label",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod"},
			},
			expectedStatus:  int32(http.StatusOK),
			expectedMessage: "nothing to do",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//GIVE
			rawPod, err := json.Marshal(tc.pod)
			require.NoError(t, err)

			req := admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Kind:   metav1.GroupVersionKind{Kind: PodType, Version: corev1.SchemeGroupVersion.Version},
					Object: runtime.RawExtension{Raw: rawPod},
				}}

			//WHEN
			resp := webhook.Handle(context.TODO(), req)

			//THEN
			require.NotNil(t, resp)
			require.NotNil(t, resp.Result)
			assert.Equal(t, tc.expectedStatus, resp.Result.Code)
			assert.Contains(t, resp.Result.Message, tc.expectedMessage)
		})
	}
}

func TestValidationWebhook_Errors(t *testing.T) {
	scheme := runtime.NewScheme()
	decoder := admission.NewDecoder(scheme)
	log := test_helpers.NewTestZapLogger(t).Sugar()
	webhook := NewValidationWebhook(log, decoder)
	testCases := []struct {
		name            string
		req             admission.Request
		expectedStatus  int32
		expectedMessage string
	}{{
		name: "Decode fails",
		req: admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Kind:   metav1.GroupVersionKind{Kind: PodType, Version: corev1.SchemeGroupVersion.Version},
				Object: runtime.RawExtension{Raw: []byte("")},
			}},
		expectedStatus: int32(http.StatusInternalServerError),
	},
		{name: "Invalid request kind",
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Kind:   metav1.GroupVersionKind{Kind: corev1.ResourceCPU.String(), Version: corev1.SchemeGroupVersion.Version},
					Object: runtime.RawExtension{Raw: []byte("")},
				}},
			expectedStatus: int32(http.StatusBadRequest)}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//GIVEN

			//WHEN
			resp := webhook.Handle(context.TODO(), tc.req)

			//THEN
			require.NotNil(t, resp)
			require.NotNil(t, resp.Result)
			assert.Equal(t, tc.expectedStatus, resp.Result.Code)
		})
	}
}
