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

func Test_nsValidationLabelSet(t *testing.T) {
	type args struct {
		labels map[string]string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "namespace validation label not found - empty key",
			args: args{
				labels: map[string]string{
					warden.NamespaceValidationLabel: "",
				},
			},
			want: false,
		},
		{
			name: "namespace validation label not found - no key",
			args: args{
				labels: map[string]string{
					"some": "label",
				},
			},
			want: false,
		},
		{
			name: "namespace validation label not found - nil map",
			args: args{
				labels: nil,
			},
			want: false,
		},
		{
			name: `namespace validation label not found - non "enabled" key value`,
			args: args{
				labels: map[string]string{
					warden.NamespaceValidationLabel: "disabled",
				},
			},
			want: false,
		},
		{
			name: "namespace validation label found",
			args: args{
				labels: map[string]string{
					warden.NamespaceValidationLabel: warden.NamespaceValidationEnabled,
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nsValidationLabelSet(tt.args.labels); got != tt.want {
				t.Errorf("nsValidationLabelSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_nsUpdated(t *testing.T) {
	type args struct {
		event event.UpdateEvent
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "ns updated",
			args: args{
				event: event.UpdateEvent{
					ObjectOld: &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{},
					},
					ObjectNew: &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								warden.NamespaceValidationLabel: warden.NamespaceValidationEnabled,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "ns not updated - new obj has no validation label",
			args: args{
				event: event.UpdateEvent{
					ObjectOld: &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{},
					},
					ObjectNew: &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								warden.NamespaceValidationLabel: "disable",
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "ns not updated - old obj has validation label",
			args: args{
				event: event.UpdateEvent{
					ObjectOld: &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								warden.NamespaceValidationLabel: warden.NamespaceValidationEnabled,
							},
						},
					},
					ObjectNew: &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{},
					},
				},
			},
			want: false,
		},
	}

	logger := test_helpers.NewTestZapLogger(t)
	nsUpdate := buildNsUpdated(predicateOps{
		logger: logger.Sugar(),
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldBeTriggered := nsUpdate(tt.args.event)
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
