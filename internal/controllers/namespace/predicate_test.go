package namespace

import (
	"testing"

	warden "github.com/kyma-project/warden/pkg"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
			name: "ns not updated - new obj has no validation lable",
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
			name: "ns not updated - old obj has validation lable",
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

	logger := newTestZapLogger(t)
	nsUpdate := buildNsUpdated(predicateOps{
		logger: logger.Sugar(),
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nsUpdate(tt.args.event); got != tt.want {
				t.Errorf("nsUpdated() = %t, want %t", got, tt.want)
			}

		})
	}
}

// testWriterSyncer is used by tests
// as an output for zap logger
type testWriterSyncer struct {
	t *testing.T
}

func (l *testWriterSyncer) Write(p []byte) (n int, err error) {
	msg := string(p)
	l.t.Logf("%s", msg)
	return len(msg), nil
}

func (l *testWriterSyncer) Sync() error {
	return nil
}

func newTestZapLogger(t *testing.T) *zap.Logger {
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		NameKey:        "logger",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		&testWriterSyncer{t: t},
		zap.DebugLevel,
	)
	return zap.New(core)
}

func Test_buildNsCreateReject(t *testing.T) {
	logger := newTestZapLogger(t)

	nsCreateReject := buildNsCreateReject(predicateOps{
		logger: logger.Sugar(),
	})

	want := false
	if got := nsCreateReject(event.CreateEvent{}); got != want {
		t.Errorf("nsCreateReject() = %t, want %t", got, want)
	}
}

func Test_buildNsDeleteReject(t *testing.T) {
	logger := newTestZapLogger(t)

	nsCreateReject := buildNsDeleteReject(predicateOps{
		logger: logger.Sugar(),
	})

	want := false
	if got := nsCreateReject(event.DeleteEvent{}); got != want {
		t.Errorf("nsDeleteReject() = %t, want %t", got, want)
	}
}

func Test_buildNsGenericReject(t *testing.T) {
	logger := newTestZapLogger(t)

	nsCreateReject := buildNsGenericReject(predicateOps{
		logger: logger.Sugar(),
	})

	want := false
	if got := nsCreateReject(event.GenericEvent{}); got != want {
		t.Errorf("nsGenericReject() = %t, want %t", got, want)
	}
}
