package namespace

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"

	warden "github.com/kyma-project/warden/pkg"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_labelWithValidationPending(t *testing.T) {
	type args struct {
		patch patch
		pod   *corev1.Pod
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "validation label already set to pending",
			args: args{
				patch: buildTestPatch(nil),
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							warden.PodValidationLabel: warden.ValidationStatusPending,
						},
					},
				},
			},
		},
		{
			name: "pending validation label should be added",
			args: args{
				patch: buildTestPatch(nil),
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{},
				},
			},
		},
		{
			name: "validation label reset from success to pending",
			args: args{
				patch: buildTestPatch(nil),
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							warden.PodValidationLabel: warden.ValidationStatusSuccess,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//WHEN
			err := labelWithValidationPending(context.Background(), tt.args.pod, tt.args.patch)

			//THEN
			require.NoError(t, err)
		})
	}
}

var errNoValidationLabel = fmt.Errorf("patched object should contain '%s' validation label", warden.PodValidationLabel)

func buildTestPatch(errResult error) patch {
	return func(_ context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) error {
		if _, found := obj.GetLabels()[warden.PodValidationLabel]; !found {
			return errNoValidationLabel
		}
		return errResult
	}
}
