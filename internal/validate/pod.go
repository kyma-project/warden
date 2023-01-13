package validate

import (
	"context"
	"github.com/kyma-project/warden/internal/util/sets"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kyma-project/warden/pkg"
	corev1 "k8s.io/api/core/v1"
)

type ValidationResult int

const (
	Invalid ValidationResult = iota
	ServiceUnAvailable
	Valid
	NoAction
)

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
	l := log.FromContext(ctx)

	if ns.Name != pod.Namespace {
		return Invalid, errors.New("pod namespace mismatch with given namespace")
	}

	if enabled := a.IsValidationEnabledForNS(ns); !enabled {
		return NoAction, nil
	}
	matched := make(map[string]ValidationResult)

	images := GetAllImages(pod)

	admitResult := Valid

	images.Walk(func(s string) {
		result, err := a.admitPodImage(s)
		matched[s] = result

		if result == Invalid {
			admitResult = Invalid
			l.Info(err.Error())
		}
	})

	return admitResult, nil
}

func (a *podValidator) IsValidationEnabledForNS(ns *corev1.Namespace) bool {
	return ns.GetLabels()[pkg.NamespaceValidationLabel] == pkg.NamespaceValidationEnabled
}

func (a *podValidator) admitPodImage(image string) (ValidationResult, error) {
	err := a.Validator.Validate(image)
	if err != nil {
		return Invalid, err
	}

	return Valid, nil
}

func GetAllImages(pod *corev1.Pod) *sets.Strings {
	var images sets.Strings
	for _, c := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
		images.Add(c.Image)
	}
	return &images
}
