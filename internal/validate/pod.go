package validate

import (
	"context"
	"errors"
	"github.com/kyma-project/warden/internal/helpers"
	"github.com/kyma-project/warden/pkg"
	corev1 "k8s.io/api/core/v1"
)

type ValidationStatus string

type ValidationResult struct {
	Status        ValidationStatus
	InvalidImages []string
}

const (
	Invalid            ValidationStatus = "Invalid"
	ServiceUnavailable ValidationStatus = "ServiceUnavailable"
	Valid              ValidationStatus = "Valid"
	NoAction           ValidationStatus = "NoAction"
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
		return ValidationResult{Invalid, nil}, errors.New("pod namespace mismatch with given namespace")
	}

	images := getAllImages(pod)

	admitResult := Valid

	invalidImages := []string{}

	for s := range images {
		result, err := a.validateImage(ctx, s)

		if result != Valid {
			admitResult = result
			invalidImages = append(invalidImages, s)
			logger.With("image", s).Info(err.Error())
		}
	}

	return ValidationResult{admitResult, invalidImages}, nil
}

func IsValidationEnabledForNS(ns *corev1.Namespace) bool {
	return ns.GetLabels()[pkg.NamespaceValidationLabel] == pkg.NamespaceValidationEnabled
}

func (a *podValidator) validateImage(ctx context.Context, image string) (ValidationStatus, error) {
	err := a.Validator.Validate(ctx, image)
	if err != nil {
		if pkg.ErrorCode(err) == pkg.UnknownResult {
			return ServiceUnavailable, err
		}
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
