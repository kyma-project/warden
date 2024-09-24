package validate

import (
	"github.com/kyma-project/warden/pkg"
	corev1 "k8s.io/api/core/v1"
)

func IsValidationEnabledForNS(ns *corev1.Namespace) bool {
	validationLabel := ns.GetLabels()[pkg.NamespaceValidationLabel]
	return validationLabel == pkg.NamespaceValidationEnabled ||
		validationLabel == pkg.NamespaceValidationSystem ||
		validationLabel == pkg.NamespaceValidationUser
}

func IsUserValidationForNS(ns *corev1.Namespace) bool {
	validationLabel := ns.GetLabels()[pkg.NamespaceValidationLabel]
	return validationLabel == pkg.NamespaceValidationUser
}
