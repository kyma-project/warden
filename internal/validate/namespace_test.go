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
			name:            "namespace has validation enabled",
			namespaceLabels: map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled},
			success:         true,
		},
		{
			name:            "namespace has validation enabled (system)",
			namespaceLabels: map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationSystem},
			success:         true,
		},
		{
			name:            "namespace has validation enabled (user)",
			namespaceLabels: map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationUser},
			success:         true,
		},
		{
			name:            "namespace has validation disabled (invalid)",
			namespaceLabels: map[string]string{pkg.NamespaceValidationLabel: "invalid"},
			success:         false,
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
			ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name:   testNs,
				Labels: testCase.namespaceLabels,
			}}

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
			name:            "namespace has user validation enabled",
			namespaceLabels: map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationUser},
			success:         true,
		},
		{
			name:            "namespace has not user validation enabled (is set to system)",
			namespaceLabels: map[string]string{pkg.NamespaceValidationLabel: pkg.NamespaceValidationSystem},
			success:         false,
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
			ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name:   testNs,
				Labels: testCase.namespaceLabels,
			}}

			//WHEN
			enabled := validate.IsUserValidationForNS(ns)

			//THEN
			require.Equal(t, testCase.success, enabled)
		})
	}
}

func TestIsChangedSupportedValidationLabelValue(t *testing.T) {
	tests := []struct {
		name     string
		oldValue string
		newValue string
		want     bool
	}{
		{
			name:     "changed (System->User)",
			oldValue: pkg.NamespaceValidationSystem,
			newValue: pkg.NamespaceValidationUser,
			want:     true,
		},
		{
			name:     "changed (User->System)",
			oldValue: pkg.NamespaceValidationUser,
			newValue: pkg.NamespaceValidationSystem,
			want:     true,
		},
		{
			name:     "changed (Enabled->User)",
			oldValue: pkg.NamespaceValidationEnabled,
			newValue: pkg.NamespaceValidationUser,
			want:     true,
		},
		{
			name:     "changed (User->Enabled)",
			oldValue: pkg.NamespaceValidationUser,
			newValue: pkg.NamespaceValidationEnabled,
			want:     true,
		},
		{
			name:     "not changed (System->System)",
			oldValue: pkg.NamespaceValidationSystem,
			newValue: pkg.NamespaceValidationSystem,
			want:     false,
		},
		{
			name:     "not changed (System->Enabled - equivalent values)",
			oldValue: pkg.NamespaceValidationSystem,
			newValue: pkg.NamespaceValidationEnabled,
			want:     false,
		},
		{
			name:     "not changed (Enabled->System - equivalent values)",
			oldValue: pkg.NamespaceValidationEnabled,
			newValue: pkg.NamespaceValidationSystem,
			want:     false,
		},
		{
			name:     "changed (System->unsupported)",
			oldValue: pkg.NamespaceValidationSystem,
			newValue: "unsupported",
			want:     true,
		},
		{
			name:     "changed (unsupported->User)",
			oldValue: "unsupported",
			newValue: pkg.NamespaceValidationUser,
			want:     true,
		},
		{
			name:     "not changed (unsupported->unsupported)",
			oldValue: "unsupported",
			newValue: "unsupported",
			want:     false,
		},
		{
			name:     "not changed (unsupported->another-unsupported)",
			oldValue: "blekota",
			newValue: "mlekota",
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validate.IsChangedSupportedValidationLabelValue(tt.oldValue, tt.newValue)
			require.Equal(t, tt.want, result)
		})
	}
}
