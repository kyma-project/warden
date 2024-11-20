package validate

import (
	"reflect"
	"testing"

	cliType "github.com/docker/cli/cli/config/types"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
)

func Test_parseCredentials(t *testing.T) {
	tests := []struct {
		name        string
		credentials cliType.AuthConfig
		want        authn.Authenticator
		wantErr     bool
	}{
		{
			name:        "empty credentials",
			credentials: cliType.AuthConfig{},
			want:        nil,
			wantErr:     true,
		},
		{
			name: "basic credentials",
			credentials: cliType.AuthConfig{
				Username: "user",
				Password: "password",
			},
			want:    &authn.Basic{Username: "user", Password: "password"},
			wantErr: false,
		},
		{
			name: "basic auth credentials",
			credentials: cliType.AuthConfig{
				Auth: "dXNlcm5hbWU6cGFzc3dvcmQ=",
			},
			want:    &authn.Basic{Username: "username", Password: "password"},
			wantErr: false,
		},
		{
			name: "token credentials",
			credentials: cliType.AuthConfig{
				RegistryToken: "token",
			},
			want:    &authn.Bearer{Token: "token"},
			wantErr: false,
		},

		{
			name: "invalid basic auth credentials",
			credentials: cliType.AuthConfig{
				Auth: "Y2Fubm90IGRlY29kZQ==",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credentials, err := parseCredentials(tt.credentials)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCredentialsOption() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(credentials, tt.want) {
				t.Errorf("parseCredentialsOption() = %v, want %v", credentials, tt.want)
			}
		})
	}
}

func Test_ImageContainsTag(t *testing.T) {
	tests := []struct {
		name  string
		image string
		want  bool
	}{
		{
			name:  "image with no tag",
			image: "image",
			want:  false,
		},
		{
			name:  "image with tag",
			image: "image:tag",
			want:  true,
		},
		{
			name:  "image with digest",
			image: "image@sha256:fdd33d7bf8cc80f223e30b4aa6c2ad705ffc7cf1a77697f37ed7232bc74484b0",
			want:  true,
		},
		{
			name:  "image with tag and digest",
			image: "image:tag@sha256:fdd33d7bf8cc80f223e30b4aa6c2ad705ffc7cf1a77697f37ed7232bc74484b0",
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := name.ParseReference(tt.image)
			assert.NoError(t, err)
			containsTag := imageContainsTag(tt.image, ref)
			assert.Equal(t, tt.want, containsTag)
		})
	}
}
