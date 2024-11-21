package admission

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kyma-project/warden/internal/annotations"
	"github.com/kyma-project/warden/internal/helpers"
	"github.com/kyma-project/warden/internal/validate"
	"github.com/kyma-project/warden/pkg"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	DefaultingPath = "/defaulting/pods"
)

const PodType = "Pod"

type DefaultingWebHook struct {
	systemValidator          validate.PodValidator
	userValidationSvcFactory validate.ValidatorSvcFactory
	timeout                  time.Duration
	client                   k8sclient.Client
	reader                   k8sclient.Reader
	decoder                  *admission.Decoder
	baseLogger               *zap.SugaredLogger
	strictMode               bool
}

func NewDefaultingWebhook(client k8sclient.Client, reader k8sclient.Reader,
	systemValidator validate.PodValidator, userValidationSvcFactory validate.ValidatorSvcFactory,
	timeout time.Duration, strictMode bool,
	decoder *admission.Decoder, logger *zap.SugaredLogger) *DefaultingWebHook {
	return &DefaultingWebHook{
		client:                   client,
		reader:                   reader,
		systemValidator:          systemValidator,
		userValidationSvcFactory: userValidationSvcFactory,
		baseLogger:               logger,
		timeout:                  timeout,
		strictMode:               strictMode,
		decoder:                  decoder,
	}
}

func (w *DefaultingWebHook) Handle(ctx context.Context, req admission.Request) admission.Response {
	return HandleWithLogger(w.baseLogger,
		HandlerWithTimeMeasure(
			HandleWithTimeout(w.timeout, w.handle, w.handleTimeout)))(ctx, req)
}

func (w *DefaultingWebHook) handle(ctx context.Context, req admission.Request) admission.Response {
	if req.Kind.Kind != PodType {
		return admission.Errored(http.StatusBadRequest,
			errors.Errorf("Invalid request kind:%s, expected:%s", req.Kind.Kind, PodType))
	}

	pod := &corev1.Pod{}
	if err := (*w.decoder).Decode(req, pod); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	logger := helpers.LoggerFromCtx(ctx)
	logger.Debugw("validation started", "operation", req.Operation, "label", pod.ObjectMeta.GetLabels()[pkg.PodValidationLabel])

	ns := &corev1.Namespace{}
	if err := w.client.Get(ctx, k8sclient.ObjectKey{Name: pod.Namespace}, ns); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if !isValidationNeeded(ctx, pod, ns, req.Operation) {
		result := cleanAnnotationIfNeeded(ctx, pod, ns, req)
		return result
	}

	validator := w.systemValidator
	if validate.IsUserValidationForNS(ns) {
		var err error
		validator, err = validate.NewUserValidationSvc(ns, w.userValidationSvcFactory)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
	}

	imagePullCredentials, err := helpers.GetRemotePullCredentials(ctx, w.reader, pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	result, err := validator.ValidatePod(ctx, pod, ns, imagePullCredentials)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	if result.Status == validate.NoAction {
		return admission.Allowed("validation is not enabled for pod")
	}
	res := w.createResponse(ctx, req, result, pod, ns, logger)
	return res
}

func cleanAnnotationIfNeeded(ctx context.Context, pod *corev1.Pod, ns *corev1.Namespace, req admission.Request) admission.Response {
	if enabled := validate.IsValidationEnabledForNS(ns); !enabled {
		return admission.Allowed("validation is not needed for pod")
	}
	if removed := removeInternalAnnotation(ctx, pod.ObjectMeta.Annotations); removed {
		fBytes, err := json.Marshal(pod)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
		return admission.PatchResponseFromRaw(req.Object.Raw, fBytes)
	}
	return admission.Allowed("validation is not needed for pod")
}

func (w DefaultingWebHook) handleTimeout(ctx context.Context, timeoutErr error, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	if err := (*w.decoder).Decode(req, pod); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	msg := fmt.Sprintf("request exceeded desired timeout: %s, reason: %s", w.timeout.String(), timeoutErr.Error())
	logger := helpers.LoggerFromCtx(ctx)
	logger.Info(msg)

	ns := &corev1.Namespace{}
	if err := w.client.Get(ctx, k8sclient.ObjectKey{Name: pod.Namespace}, ns); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	res := w.createResponse(ctx, req, validate.ValidationResult{Status: validate.ServiceUnavailable}, pod, ns, logger)
	res.Result = &metav1.Status{Message: msg}
	return res
}

func (w *DefaultingWebHook) createResponse(ctx context.Context,
	req admission.Request, result validate.ValidationResult,
	pod *corev1.Pod, ns *corev1.Namespace, logger *zap.SugaredLogger) admission.Response {

	strictMode := w.strictMode
	if validate.IsUserValidationForNS(ns) {
		var err error
		strictMode, err = helpers.GetUserValidationStrictMode(ns)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
	}

	markedPod := markPod(ctx, result, pod, strictMode)
	fBytes, err := json.Marshal(markedPod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	logger.Infow("pod was validated", "result", result)
	return admission.PatchResponseFromRaw(req.Object.Raw, fBytes)
}

func isValidationNeeded(ctx context.Context, pod *corev1.Pod, ns *corev1.Namespace, operation admissionv1.Operation) bool {
	logger := helpers.LoggerFromCtx(ctx)
	if enabled := validate.IsValidationEnabledForNS(ns); !enabled {
		logger.Debugw("pod validation skipped because validation for namespace is not enabled")
		return false
	}
	if needed := IsValidationNeededForOperation(operation); needed {
		return true
	}
	if enabled := isValidationEnabledForPodValidationLabel(pod); !enabled {
		logger.Debugw("pod validation skipped because pod checking is not enabled for the input validation label")
		return false
	}
	return true
}

func IsValidationNeededForOperation(operation admissionv1.Operation) bool {
	return operation == admissionv1.Create
}

func isValidationEnabledForPodValidationLabel(pod *corev1.Pod) bool {
	validationLabelValue := getPodValidationLabelValue(pod)
	if validationLabelValue == pkg.ValidationStatusFailed || validationLabelValue == pkg.ValidationStatusPending {
		return false
	}
	return true
}

func getPodValidationLabelValue(pod *corev1.Pod) string {
	if pod.Labels == nil {
		return ""
	}
	validationLabelValue, ok := pod.Labels[pkg.PodValidationLabel]
	if !ok {
		return ""
	}
	return validationLabelValue
}

func markPod(ctx context.Context, result validate.ValidationResult, pod *corev1.Pod, strictMode bool) *corev1.Pod {
	label, annotation := podMarkersForValidationResult(result.Status, strictMode)
	helpers.LoggerFromCtx(ctx).Infof("pod was labeled: `%s` and annotated: `%s`", label, annotation)
	if label == "" && annotation == "" {
		return pod
	}

	markedPod := pod.DeepCopy()
	if label != "" {
		if markedPod.Labels == nil {
			markedPod.Labels = map[string]string{}
		}
		markedPod.Labels[pkg.PodValidationLabel] = label
	}

	// Fixes: https://github.com/kyma-project/warden/issues/77
	removeInternalAnnotation(ctx, markedPod.Annotations)
	if annotation != "" {
		if markedPod.Annotations == nil {
			markedPod.Annotations = map[string]string{}
		}
		markedPod.Annotations[annotations.PodValidationRejectAnnotation] = annotation
		if result.InvalidImages != nil {
			markedPod.Annotations[annotations.InvalidImagesAnnotation] = strings.Join(result.InvalidImages, ", ")
		}
	}
	return markedPod
}

func podMarkersForValidationResult(result validate.ValidationStatus, strictMode bool) (label string, annotation string) {
	switch result {
	case validate.NoAction:
		return "", ""
	case validate.Invalid:
		return pkg.ValidationStatusFailed, annotations.ValidationReject
	case validate.Valid:
		return pkg.ValidationStatusSuccess, ""
	case validate.ServiceUnavailable:
		annotation = ""
		if strictMode {
			annotation = annotations.ValidationReject
		}
		return pkg.ValidationStatusPending, annotation
	default:
		return pkg.ValidationStatusPending, ""
	}
}

func removeInternalAnnotation(ctx context.Context, source map[string]string) bool {
	logger := helpers.LoggerFromCtx(ctx)
	if _, ok := source[annotations.PodValidationRejectAnnotation]; ok {
		delete(source, annotations.PodValidationRejectAnnotation)
		logger.Debug("Internal Annotation deleted")
		return true
	}
	return false
}
