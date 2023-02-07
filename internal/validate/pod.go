package validate

import (
	"context"
	"github.com/kyma-project/warden/internal/helpers"
	"github.com/kyma-project/warden/pkg"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

type ValidationResult string

const (
	Invalid            ValidationResult = "Invalid"
	ServiceUnavailable ValidationResult = "ServiceUnavailable"
	Valid              ValidationResult = "Valid"
	NoAction           ValidationResult = "NoAction"
)

//go:generate mockery --name PodValidator
type PodValidator interface {
	ValidatePod(ctx context.Context, pod *corev1.Pod, ns *corev1.Namespace) (ValidationResult, error)
}

type NamespaceChecker interface {
	IsValidationEnabledForNS(namespace string) bool
}

var _ PodValidator = &podValidator{}

type podValidator struct {
	Validator ImageValidatorService
}

func NewPodValidator(imageValidator ImageValidatorService) PodValidator {
	return &podValidator{
		imageValidator,
	}
}

func (a *podValidator) ValidatePod(ctx context.Context, pod *corev1.Pod, ns *corev1.Namespace) (ValidationResult, error) {
	logger := helpers.LoggerFromCtx(ctx)

	if ns.Name != pod.Namespace {
		return Invalid, errors.New("pod namespace mismatch with given namespace")
	}

	if enabled := IsValidationEnabledForNS(ns); !enabled {
		logger.Debugw("Pod validation skipped because validation for namespace is not enabled")
		return NoAction, nil
	}

	if enabled := IsValidationEnabledForPodValidationLabel(pod); !enabled {
		logger.Debugw("Pod verification skipped because pod checking is not enabled for the input validation label")
		return NoAction, nil
	}

	matched := make(map[string]ValidationResult)

	images := getAllImages(pod)

	admitResult := Valid

	for s := range images {
		result, err := a.validateImage(ctx, s)
		matched[s] = result

		if result == Invalid {
			admitResult = Invalid
			logger.Info(err.Error())
		}
	}

	return admitResult, nil
}

func IsValidationEnabledForNS(ns *corev1.Namespace) bool {
	return ns.GetLabels()[pkg.NamespaceValidationLabel] == pkg.NamespaceValidationEnabled
}

func IsValidationEnabledForPodValidationLabel(pod *corev1.Pod) bool {
	validationLabelValue := getPodValidationLabelValue(pod)
	if validationLabelValue == "" {
		return true
	}
	if validationLabelValue == pkg.ValidationStatusSuccess {
		return true
	}
	return false
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
func (a *podValidator) validateImage(ctx context.Context, image string) (ValidationResult, error) {
	err := a.Validator.Validate(ctx, image)
	if err != nil {
		return Invalid, err
	}

	return Valid, nil
}

func getAllImages(pod *corev1.Pod) map[string]struct{} {
	images := make(map[string]struct{})
	for _, c := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
		images[c.Image] = struct{}{}
	}
	return images
}
