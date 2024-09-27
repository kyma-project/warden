package namespace

import (
	"github.com/kyma-project/warden/internal/test_helpers"
	"github.com/stretchr/testify/require"
	"testing"

	warden "github.com/kyma-project/warden/pkg"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func Test_nsUpdated(t *testing.T) {
	tests := []struct {
		name  string
		event event.UpdateEvent
		want  bool
	}{
		{
			name: "ns updated - added validation label",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationEnabled}}}},
			want: true,
		},
		{
			name: "ns not updated - added validation label with unsupported value",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: "disable"}}}},
			want: false,
		},
		{
			name: "ns not updated - removed validation label",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationEnabled}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{}}},
			want: false,
		},
		{
			name: "ns updated - changed validation label value (both supported)",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationSystem}}}},
			want: true,
		},
		{
			name: "ns updated - changed validation label value from unsupported to supported",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: "disable"}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationSystem}}}},
			want: true,
		},
		{
			name: "ns not updated - changed validation label value from supported to unsupported",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationSystem}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: "disable"}}}},
			want: false,
		},
		{
			name: "ns updated - changed user validation annotations (notary url) value for user validation",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser},
					Annotations: map[string]string{warden.NamespaceNotaryURLAnnotation: "notary-url"}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser},
					Annotations: map[string]string{warden.NamespaceNotaryURLAnnotation: "changed-notary-url"}}}},
			want: true,
		},
		{
			name: "ns updated - changed user validation annotations (allowed registries) value for user validation",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser},
					Annotations: map[string]string{warden.NamespaceAllowedRegistriesAnnotation: "allowed,registries"}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser},
					Annotations: map[string]string{warden.NamespaceAllowedRegistriesAnnotation: "another,allowed,registries"}}}},
			want: true,
		},
		{
			name: "ns updated - changed user validation annotations (notary timeout) value for user validation",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser},
					Annotations: map[string]string{warden.NamespaceNotaryTimeoutAnnotation: "33s"}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser},
					Annotations: map[string]string{warden.NamespaceNotaryTimeoutAnnotation: "44s"}}}},
			want: true,
		},
		{
			name: "ns updated - changed user validation annotations (strict mode) value for user validation",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser},
					Annotations: map[string]string{warden.NamespaceStrictModeAnnotation: "true"}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser},
					Annotations: map[string]string{warden.NamespaceStrictModeAnnotation: "false"}}}},
			want: true,
		},
		{
			name: "ns updated - added user validation annotations (notary url) value for user validation",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser},
					Annotations: map[string]string{warden.NamespaceNotaryURLAnnotation: "notary-url"}}}},
			want: true,
		},
		{
			name: "ns updated - added user validation annotations (allowed registries) value for user validation",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser},
					Annotations: map[string]string{warden.NamespaceAllowedRegistriesAnnotation: "allowed,registries"}}}},
			want: true,
		},
		{
			name: "ns updated - added user validation annotations (notary timeout) value for user validation",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser},
					Annotations: map[string]string{warden.NamespaceNotaryTimeoutAnnotation: "33s"}}}},
			want: true,
		},
		{
			name: "ns updated - added user validation annotations (strict mode) value for user validation",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser},
					Annotations: map[string]string{warden.NamespaceStrictModeAnnotation: "true"}}}},
			want: true,
		},
		{
			name: "ns updated - removed user validation annotations (notary url) value for user validation",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser},
					Annotations: map[string]string{warden.NamespaceNotaryURLAnnotation: "notary-url"}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser}}}},
			want: true,
		},
		{
			name: "ns updated - removed user validation annotations (allowed registries) value for user validation",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser},
					Annotations: map[string]string{warden.NamespaceAllowedRegistriesAnnotation: "allowed,registries"}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser}}}},
			want: true,
		},
		{
			name: "ns updated - removed user validation annotations (notary timeout) value for user validation",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser},
					Annotations: map[string]string{warden.NamespaceNotaryTimeoutAnnotation: "33s"}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser}}}},
			want: true,
		},
		{
			name: "ns updated - removed user validation annotations (strict mode) value for user validation",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser},
					Annotations: map[string]string{warden.NamespaceStrictModeAnnotation: "true"}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationUser}}}},
			want: true,
		},
		{
			name: "ns not updated - changed user validation annotations value for system validation",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationSystem},
					Annotations: map[string]string{
						warden.NamespaceNotaryURLAnnotation:         "notary-url",
						warden.NamespaceAllowedRegistriesAnnotation: "allowed,registries",
						warden.NamespaceNotaryTimeoutAnnotation:     "33s",
						warden.NamespaceStrictModeAnnotation:        "true"}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationSystem},
					Annotations: map[string]string{
						warden.NamespaceNotaryURLAnnotation:         "another-notary-url",
						warden.NamespaceAllowedRegistriesAnnotation: "another,allowed,registries",
						warden.NamespaceNotaryTimeoutAnnotation:     "44s",
						warden.NamespaceStrictModeAnnotation:        "false"}}}},
			want: false,
		},
		{
			name: "ns not updated - removed user validation annotations value for system validation",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationEnabled},
					Annotations: map[string]string{
						warden.NamespaceNotaryURLAnnotation:         "notary-url",
						warden.NamespaceAllowedRegistriesAnnotation: "allowed,registries",
						warden.NamespaceNotaryTimeoutAnnotation:     "33s",
						warden.NamespaceStrictModeAnnotation:        "true"}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationEnabled}}}},
			want: false,
		},
		{
			name: "ns not updated - added user validation annotations value for system validation",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationSystem}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: warden.NamespaceValidationSystem},
					Annotations: map[string]string{
						warden.NamespaceNotaryURLAnnotation:         "notary-url",
						warden.NamespaceAllowedRegistriesAnnotation: "allowed,registries",
						warden.NamespaceNotaryTimeoutAnnotation:     "33s",
						warden.NamespaceStrictModeAnnotation:        "true"}}}},
			want: false,
		},
		{
			name: "ns not updated - changed user validation annotations value for no validation label",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						warden.NamespaceNotaryURLAnnotation:         "notary-url",
						warden.NamespaceAllowedRegistriesAnnotation: "allowed,registries",
						warden.NamespaceNotaryTimeoutAnnotation:     "33s",
						warden.NamespaceStrictModeAnnotation:        "true"}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						warden.NamespaceNotaryURLAnnotation:         "another-notary-url",
						warden.NamespaceAllowedRegistriesAnnotation: "another,allowed,registries",
						warden.NamespaceNotaryTimeoutAnnotation:     "44s",
						warden.NamespaceStrictModeAnnotation:        "false"}}}},
			want: false,
		},
		{
			name: "ns not updated - removed user validation annotations value for no validation label",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: "disabled"},
					Annotations: map[string]string{
						warden.NamespaceNotaryURLAnnotation:         "notary-url",
						warden.NamespaceAllowedRegistriesAnnotation: "allowed,registries",
						warden.NamespaceNotaryTimeoutAnnotation:     "33s",
						warden.NamespaceStrictModeAnnotation:        "true"}}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{warden.NamespaceValidationLabel: "unsupported"}}}},
			want: false,
		},
		{
			name: "ns not updated - added user validation annotations value for no validation label",
			event: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{}},
				ObjectNew: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						warden.NamespaceNotaryURLAnnotation:         "notary-url",
						warden.NamespaceAllowedRegistriesAnnotation: "allowed,registries",
						warden.NamespaceNotaryTimeoutAnnotation:     "33s",
						warden.NamespaceStrictModeAnnotation:        "true"}}}},
			want: false,
		},
	}

	logger := test_helpers.NewTestZapLogger(t)
	nsUpdate := buildNsUpdated(predicateOps{
		logger: logger.Sugar(),
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldBeTriggered := nsUpdate(tt.event)
			require.Equal(t, tt.want, shouldBeTriggered)
		})
	}
}

func Test_buildNsCreateReject(t *testing.T) {
	logger := test_helpers.NewTestZapLogger(t)

	nsCreateReject := buildNsCreateReject(predicateOps{
		logger: logger.Sugar(),
	})

	want := false
	if got := nsCreateReject(event.CreateEvent{}); got != want {
		t.Errorf("nsCreateReject() = %t, want %t", got, want)
	}
}

func Test_buildNsDeleteReject(t *testing.T) {
	logger := test_helpers.NewTestZapLogger(t)

	nsCreateReject := buildNsDeleteReject(predicateOps{
		logger: logger.Sugar(),
	})

	want := false
	if got := nsCreateReject(event.DeleteEvent{}); got != want {
		t.Errorf("nsDeleteReject() = %t, want %t", got, want)
	}
}

func Test_buildNsGenericReject(t *testing.T) {
	logger := test_helpers.NewTestZapLogger(t)

	nsCreateReject := buildNsGenericReject(predicateOps{
		logger: logger.Sugar(),
	})

	want := false
	if got := nsCreateReject(event.GenericEvent{}); got != want {
		t.Errorf("nsGenericReject() = %t, want %t", got, want)
	}
}
