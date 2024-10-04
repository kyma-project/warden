package helpers

import (
	"github.com/kyma-project/warden/pkg"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"strconv"
	"time"
)

const (
	DefaultUserAllowedRegistries   = ""
	DefaultUserNotaryTimeoutString = "30s"
	DefaultUserStrictMode          = true
)

type UserValidationNotaryConfig struct {
	NotaryURL         string
	AllowedRegistries string
	NotaryTimeout     time.Duration
}

func GetUserValidationNotaryConfig(ns *corev1.Namespace) (UserValidationNotaryConfig, error) {
	userNotaryURL, okNotaryURL := ns.GetAnnotations()[pkg.NamespaceNotaryURLAnnotation]
	if !okNotaryURL {
		return UserValidationNotaryConfig{}, errors.New("notary URL is not set")
	}
	userAllowedRegistries, okAllowedRegistries := ns.GetAnnotations()[pkg.NamespaceAllowedRegistriesAnnotation]
	if !okAllowedRegistries {
		userAllowedRegistries = DefaultUserAllowedRegistries
	}
	userNotaryTimeoutString, okUserNotaryTimeoutString := ns.GetAnnotations()[pkg.NamespaceNotaryTimeoutAnnotation]
	if !okUserNotaryTimeoutString {
		userNotaryTimeoutString = DefaultUserNotaryTimeoutString
	}
	userNotaryTimeout, errNotaryTimeoutParse := time.ParseDuration(userNotaryTimeoutString)
	if errNotaryTimeoutParse != nil {
		return UserValidationNotaryConfig{}, errNotaryTimeoutParse
	}
	return UserValidationNotaryConfig{
		NotaryURL:         userNotaryURL,
		AllowedRegistries: userAllowedRegistries,
		NotaryTimeout:     userNotaryTimeout,
	}, nil
}

func GetUserValidationStrictMode(ns *corev1.Namespace) (bool, error) {
	strictModeString, ok := ns.GetAnnotations()[pkg.NamespaceStrictModeAnnotation]
	if !ok {
		return DefaultUserStrictMode, nil
	}
	strictMode, err := strconv.ParseBool(strictModeString)
	if err != nil {
		return true, errors.Wrapf(err, "failed to parse %s annotation", pkg.NamespaceStrictModeAnnotation)
	}
	return strictMode, nil
}
