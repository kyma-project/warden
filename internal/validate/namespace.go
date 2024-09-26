package validate

import (
	"github.com/kyma-project/warden/pkg"
	corev1 "k8s.io/api/core/v1"
)

func IsValidationEnabledForNS(ns *corev1.Namespace) bool {
	value := ns.GetLabels()[pkg.NamespaceValidationLabel]
	return IsSupportedValidationLabelValue(value)
}

func IsSupportedValidationLabelValue(value string) bool {
	return value == pkg.NamespaceValidationEnabled ||
		value == pkg.NamespaceValidationSystem ||
		value == pkg.NamespaceValidationUser
}

func IsUserValidationForNS(ns *corev1.Namespace) bool {
	value := ns.GetLabels()[pkg.NamespaceValidationLabel]
	return value == pkg.NamespaceValidationUser
}

func IsChangedSupportedValidationLabelValue(oldValue, newValue string) bool {
	if !IsSupportedValidationLabelValue(oldValue) && !IsSupportedValidationLabelValue(newValue) {
		return false
	}
	if (oldValue == pkg.NamespaceValidationEnabled || oldValue == pkg.NamespaceValidationSystem) &&
		(newValue == pkg.NamespaceValidationEnabled || newValue == pkg.NamespaceValidationSystem) {
		return false
	}
	return oldValue != newValue
}
