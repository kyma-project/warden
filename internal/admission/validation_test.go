package admission

import (
	"context"
	"encoding/json"
	"github.com/kyma-project/warden/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func TestValidationWebhook(t *testing.T) {
	scheme := runtime.NewScheme()
	decoder, err := admission.NewDecoder(scheme)
	require.NoError(t, err)
	webhook := NewValidationWebhook()
	require.NoError(t, webhook.InjectDecoder(decoder))

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
					Labels: map[string]string{
						pkg.PodValidationLabel: pkg.ValidationStatusReject,
					}},
			},
			expectedStatus:  int32(http.StatusForbidden),
			expectedMessage: "images validation failed",
		},
		{
			name: "Pod should be allowed, validation success",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod",
					Labels: map[string]string{
						pkg.PodValidationLabel: pkg.ValidationStatusSuccess,
					}},
			},
			expectedStatus:  int32(http.StatusOK),
			expectedMessage: "nothing to do",
		},
		{
			name: "Pod should be allowed, validation pending",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod",
					Labels: map[string]string{
						pkg.PodValidationLabel: pkg.ValidationStatusPending,
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
					Resource: metav1.GroupVersionResource{Resource: corev1.ResourcePods.String(), Version: corev1.SchemeGroupVersion.Version},
					Object:   runtime.RawExtension{Raw: rawPod},
				}}

			//WHEN
			resp := webhook.Handle(context.TODO(), req)

			//THEN
			require.NotNil(t, resp)
			require.NotNil(t, resp.Result)
			assert.Equal(t, tc.expectedStatus, resp.Result.Code)
			assert.Contains(t, resp.Result.Reason, tc.expectedMessage)
		})
	}
}

func TestValidationWebhook_Errors(t *testing.T) {
	scheme := runtime.NewScheme()
	decoder, err := admission.NewDecoder(scheme)
	require.NoError(t, err)
	webhook := NewValidationWebhook()
	require.NoError(t, webhook.InjectDecoder(decoder))
	testCases := []struct {
		name            string
		req             admission.Request
		expectedStatus  int32
		expectedMessage string
	}{{
		name: "Decode fails",
		req: admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Kind:   metav1.GroupVersionKind{Kind: corev1.ResourcePods.String(), Version: corev1.SchemeGroupVersion.Version},
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
