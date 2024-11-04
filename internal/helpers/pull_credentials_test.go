package helpers

import (
	"context"
	"reflect"
	"testing"

	cliType "github.com/docker/cli/cli/config/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_GetRemotePullCredentials(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		secrets []*corev1.Secret
		pod     *corev1.Pod
		want    map[string]cliType.AuthConfig
		wantErr bool
	}{
		{
			name:    "no secrets",
			secrets: nil,
			pod:     &corev1.Pod{},
			want:    map[string]cliType.AuthConfig{},
			wantErr: false,
		},
		{
			name:    "missing secret",
			secrets: nil,
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: "secret",
						},
					},
				},
			},
			want:    map[string]cliType.AuthConfig{},
			wantErr: true,
		},
		{
			name: "incorrect secret",
			secrets: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"incorrectKey": []byte("someData"),
					},
				},
			},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: "secret",
						},
					},
				},
			},
			want:    map[string]cliType.AuthConfig{},
			wantErr: true,
		},
		{
			name: "malformed secret",
			secrets: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						".dockerconfigjson": []byte("someData"),
					},
				},
			},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: "secret",
						},
					},
				},
			},
			want:    map[string]cliType.AuthConfig{},
			wantErr: true,
		},
		{
			name: "secret with .dockerconfigjson",
			secrets: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						".dockerconfigjson": []byte(`{"auths": {"registry": {"username": "user", "password": "password"}}}`),
					},
				},
			},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: "secret",
						},
					},
				},
			},
			want: map[string]cliType.AuthConfig{
				"registry": {
					Username: "user",
					Password: "password",
				},
			},
			wantErr: false,
		}, {
			name: "secret with config.json",
			secrets: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"config.json": []byte(`{"auths": {"registry": {"auth": "username:password"}}}`),
					},
				},
			},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: "secret",
						},
					},
				},
			},
			want: map[string]cliType.AuthConfig{
				"registry": {
					Auth: "username:password",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientBuilder := fake.NewClientBuilder()
			for _, secret := range tt.secrets {
				clientBuilder.WithObjects(secret)
			}
			client := clientBuilder.Build()

			got, err := GetRemotePullCredentials(ctx, client, tt.pod)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRemotePullCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRemotePullCredentials() = %v, want %v", got, tt.want)
			}
		})
	}
}
