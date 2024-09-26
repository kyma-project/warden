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
	type args struct {
		event event.UpdateEvent
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "ns updated - added validation label",
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
			name: "ns not updated - added validation label with unsupported value",
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
			name: "ns not updated - removed validation label",
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
		{
			name: "ns updated - changed validation label value (both supported)",
			args: args{
				event: event.UpdateEvent{
					ObjectOld: &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								warden.NamespaceValidationLabel: warden.NamespaceValidationUser,
							},
						},
					},
					ObjectNew: &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								warden.NamespaceValidationLabel: warden.NamespaceValidationSystem,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "ns updated - changed validation label value from unsupported to supported",
			args: args{
				event: event.UpdateEvent{
					ObjectOld: &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								warden.NamespaceValidationLabel: "disable",
							},
						},
					},
					ObjectNew: &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								warden.NamespaceValidationLabel: warden.NamespaceValidationSystem,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "ns not updated - changed validation label value from supported to unsupported",
			args: args{
				event: event.UpdateEvent{
					ObjectOld: &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								warden.NamespaceValidationLabel: warden.NamespaceValidationSystem,
							},
						},
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
