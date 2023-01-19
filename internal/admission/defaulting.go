package admission

import (
	"context"
	"encoding/json"
	"github.com/kyma-project/warden/internal/validate"
	"github.com/kyma-project/warden/pkg"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"time"
)

const (
	DefaultingPath = "/defaulting/pods"
)

type DefaultingWebHook struct {
	validationSvc validate.PodValidator
	timeout       time.Duration
	client        k8sclient.Client
	decoder       *admission.Decoder
	logger        *zap.SugaredLogger
}

func NewDefaultingWebhook(client k8sclient.Client, ValidationSvc validate.PodValidator, timeout time.Duration, logger *zap.SugaredLogger) *DefaultingWebHook {
	return &DefaultingWebHook{
		client:        client,
		validationSvc: ValidationSvc,
		logger:        logger,
		timeout:       timeout,
	}
}

func (w *DefaultingWebHook) Handle(ctx context.Context, req admission.Request) admission.Response {
	if req.Kind.Kind != corev1.ResourcePods.String() {
		return admission.Errored(http.StatusBadRequest,
			errors.Errorf("Invalid request kind:%s, expected:%s", req.Kind.Kind, corev1.ResourcePods.String()))
	}

	pod := &corev1.Pod{}
	if err := w.decoder.Decode(req, pod); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	ns := &corev1.Namespace{}
	if err := w.client.Get(ctx, k8sclient.ObjectKey{Name: pod.Namespace}, ns); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	done := make(chan bool)
	var result validate.ValidationResult
	var err error
	go func() {
		result, err = w.validationSvc.ValidatePod(ctxTimeout, pod, ns)
		done <- true
	}()

	select {
	case <-done:
	case <-ctxTimeout.Done():
		return admission.Errored(http.StatusGatewayTimeout, ctxTimeout.Err())
	}
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	if result == validate.NoAction {
		return admission.Allowed("validation is not enabled for pod")
	}

	labeledPod := labelPod(result, pod)
	fBytes, err := json.Marshal(labeledPod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	w.logger.Infof("pod was validated: %s, %s", pod.ObjectMeta.GetName(), pod.ObjectMeta.GetNamespace())
	return admission.PatchResponseFromRaw(req.Object.Raw, fBytes)
}

func (w *DefaultingWebHook) InjectDecoder(decoder *admission.Decoder) error {
	w.decoder = decoder
	return nil
}

func labelPod(result validate.ValidationResult, pod *corev1.Pod) *corev1.Pod {
	labelToApply := LabelForValidationResult(result)
	if labelToApply == "" {
		return pod
	}
	labeledPod := pod.DeepCopy()
	if labeledPod.Labels == nil {
		labeledPod.Labels = map[string]string{}
	}

	labeledPod.Labels[pkg.PodValidationLabel] = labelToApply
	return labeledPod
}

func LabelForValidationResult(result validate.ValidationResult) string {
	switch result {
	case validate.NoAction:
		return ""
	case validate.Invalid:
		return pkg.ValidationStatusReject
	case validate.Valid:
		return pkg.ValidationStatusSuccess
	case validate.ServiceUnAvailable:
		return pkg.ValidationStatusPending
	default:
		return pkg.ValidationStatusPending
	}
}
