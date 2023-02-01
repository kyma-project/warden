package namespace

import (
	"context"
	"github.com/kyma-project/warden/internal/controllers/test_suite"
	"testing"
	"time"

	warden "github.com/kyma-project/warden/pkg"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	validatableNs   = "warden-enabled"
	unvalidatableNs = "some-ns"
)

func Test_NamespaceReconcile(t *testing.T) {
	testEnv, k8sClient := test_suite.Setup(t)
	defer test_suite.TearDown(t, testEnv)

	type args struct {
		client             client.Client
		pod                corev1.Pod
		ns                 *corev1.Namespace
		expectedLabelValue string
	}

	podSpec := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Image: "test-image",
				Name:  "container",
			},
		},
	}
	//TODO: create namespaces before

	tests := []struct {
		name string
		args args
	}{
		{
			name: "Happy Path with pod having no label",
			args: args{
				client: k8sClient,
				pod: corev1.Pod{ObjectMeta: metav1.ObjectMeta{
					Namespace: validatableNs,
					Name:      "valid-pod"},
					Spec: podSpec,
				},
				ns: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Name: validatableNs,
					Labels: map[string]string{
						warden.NamespaceValidationLabel: warden.NamespaceValidationEnabled,
					}},
				},
				expectedLabelValue: warden.ValidationStatusPending,
			},
		},
		{
			name: "Happy Path with label reset",
			args: args{
				client: k8sClient,
				pod: corev1.Pod{ObjectMeta: metav1.ObjectMeta{
					Namespace: validatableNs,
					Name:      "valid-pod-2",
					Labels: map[string]string{
						warden.PodValidationLabel: warden.ValidationStatusSuccess,
					},
				},
					Spec: podSpec,
				},
				expectedLabelValue: warden.ValidationStatusPending,
			},
		},
		//TODO: investigate this
		{
			name: "Happy Path, no pods",
			args: args{
				client: k8sClient,
			},
		},
		{
			name: "Namespace not labeled",
			args: args{
				client: k8sClient,
				pod: corev1.Pod{ObjectMeta: metav1.ObjectMeta{
					Namespace: unvalidatableNs,
					Name:      "valid-pod-3"},
					Spec: podSpec,
				},
				ns: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: unvalidatableNs,
						Labels: map[string]string{
							"some": "label",
						},
					},
				},
			},
		},
	}

	timeout := time.Second * 60
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create ns if needed
			if tt.args.ns != nil {
				err := client.IgnoreAlreadyExists(k8sClient.Create(ctx, tt.args.ns))
				require.NoError(t, err)
			}

			ctrl := Reconciler{
				Client: tt.args.client,
				Scheme: scheme.Scheme,
				Log:    newTestZapLogger(t).Sugar(),
			}

			require.NoError(t, k8sClient.Create(ctx, &tt.args.pod))

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: validatableNs,
				},
			}

			_, err := ctrl.Reconcile(ctx, req)

			require.NoError(t, err)
			key := client.ObjectKeyFromObject(&tt.args.pod)

			finalPod := corev1.Pod{}
			require.NoError(t, k8sClient.Get(ctx, key, &finalPod))

			labelValue := finalPod.Labels[warden.PodValidationLabel]
			require.Equal(t, tt.args.expectedLabelValue, labelValue)
		})
	}
}
