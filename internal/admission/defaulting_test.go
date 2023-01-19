package admission

import (
	"context"
	"encoding/json"
	"github.com/kyma-project/warden/internal/validate"
	"github.com/kyma-project/warden/internal/validate/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
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
	timeout := time.Second

	testNs := "test-namespace"
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNs}}
	pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: testNs}}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&ns, &pod).Build()
	validationSvc := mocks.NewPodValidator(t)
	validationSvc.On("ValidatePod", mock.Anything, mock.Anything, mock.Anything).
		After(timeout*2).
		Return(validate.Valid, nil).Once()
	defer validationSvc.AssertExpectations(t)

	webhook := NewDefaultingWebhook(client, validationSvc, timeout, logger.Sugar())
	require.NoError(t, webhook.InjectDecoder(decoder))

	raw, err := json.Marshal(pod)
	require.NoError(t, err)

	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Kind:   metav1.GroupVersionKind{Kind: corev1.ResourcePods.String(), Version: corev1.SchemeGroupVersion.Version},
			Object: runtime.RawExtension{Raw: raw},
		}}

	//WHEN
	res := webhook.Handle(context.TODO(), req)

	//THEN
	require.NotNil(t, res)
	require.NotNil(t, res.Result, "response is ok")
	assert.Equal(t, int32(http.StatusGatewayTimeout), res.Result.Code)
	assert.Equal(t, res.Result.Message, "context deadline exceeded")
}
