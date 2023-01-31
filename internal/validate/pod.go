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
	Invalid            = "Invalid"
	ServiceUnavailable = "ServiceUnavailable"
	Valid              = "Valid"
	NoAction           = "NoAction"
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
	logger := helpers.LoggerFromCtx(ctx) //log.FromContext(ctx)

	if ns.Name != pod.Namespace {
		return Invalid, errors.New("pod namespace mismatch with given namespace")
	}

	if enabled := IsValidationEnabledForNS(ns); !enabled {
		return NoAction, nil
	}
	matched := make(map[string]ValidationResult)

	images := getAllImages(pod)

	var admitResult ValidationResult = Valid

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
