package validate_test

import (
	"testing"

	"github.com/kyma-project/warden/internal/validate"
	"github.com/kyma-project/warden/pkg"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNamespaceLabelsValidation(t *testing.T) {
	testNs := "test-namespace"

	testCases := []struct {
		name            string
		namespaceLabels map[string]string
		success         bool
	}{
		{
			name: "namespace has validation enabled",
			namespaceLabels: map[string]string{
				pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled,
			},
			success: true,
		},
		{
			name: "namespace has validation enabled (system)",
			namespaceLabels: map[string]string{
				pkg.NamespaceValidationLabel: pkg.NamespaceValidationSystem,
			},
			success: true,
		},
		{
			name: "namespace has validation enabled (user)",
			namespaceLabels: map[string]string{
				pkg.NamespaceValidationLabel: pkg.NamespaceValidationUser,
			},
			success: true,
		},
		{
			name: "namespace has validation disabled (invalid)",
			namespaceLabels: map[string]string{
				pkg.NamespaceValidationLabel: "invalid",
			},
			success: false,
		},
		{
			name:            "namespace has no validation label",
			namespaceLabels: map[string]string{},
			success:         false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			//GIVEN
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   testNs,
					Labels: testCase.namespaceLabels,
				},
			}

			//WHEN
			enabled := validate.IsValidationEnabledForNS(ns)

			//THEN
			require.Equal(t, testCase.success, enabled)
		})
	}
}

func TestUserNamespaceLabelsValidation(t *testing.T) {
	testNs := "test-namespace"

	testCases := []struct {
		name            string
		namespaceLabels map[string]string
		success         bool
	}{
		{
			name: "namespace has user validation enabled",
			namespaceLabels: map[string]string{
				pkg.NamespaceValidationLabel: pkg.NamespaceValidationUser,
			},
			success: true,
		},
		{
			name: "namespace has not user validation enabled (is set to system)",
			namespaceLabels: map[string]string{
				pkg.NamespaceValidationLabel: pkg.NamespaceValidationSystem,
			},
			success: false,
		},
		{
			name:            "namespace has no validation label",
			namespaceLabels: map[string]string{},
			success:         false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			//GIVEN
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   testNs,
					Labels: testCase.namespaceLabels,
				},
			}

			//WHEN
			enabled := validate.IsUserValidationForNS(ns)

			//THEN
			require.Equal(t, testCase.success, enabled)
		})
	}
}
