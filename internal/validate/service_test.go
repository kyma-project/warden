package validate

import (
	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/tuf/data"
	"testing"
	"time"
)

const (
	UntrustedImageName = "nginx:latest"
	TrustedImageName   = "eu.gcr.io/kyma-project/function-controller:PR-16481"
)

var (
	TrustedImageHash = []byte{243, 155, 151, 155, 35, 94, 175, 164, 30, 8, 73, 56, 233, 106, 9, 124, 3, 46, 36, 141, 41, 227, 150, 143, 207, 210, 152, 26, 190, 95, 17, 166}
)

func Test_Validate_ProperImage_ShouldPass(t *testing.T) {
	s := NewDefaultMockNotaryService().WithHash(TrustedImageHash).Build()
	err := s.Validate(TrustedImageName)
	require.NoError(t, err)
}

func Test_Validate_InvalidImageName_ShouldReturnError(t *testing.T) {
	tests := []struct {
		name      string
		imageName string
	}{
		{
			name:      "image name without semicolon",
			imageName: "makapaka",
		},
		{
			name:      "image name with more than one semicolon", //TODO: IMO it's proper image name, but now is not allowed
			imageName: "repo:port/image-name:tag",
		},
	}
	s := NewDefaultMockNotaryService().Build()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.Validate(tt.imageName)
			require.Error(t, err)
			require.EqualError(t, err, "image name is not formatted correctly")
		})
	}
}

func Test_Validate_ImageWithDifferentHashInNotary_ShouldReturnError(t *testing.T) {
	s := NewDefaultMockNotaryService().Build()
	err := s.Validate(TrustedImageName)
	require.Error(t, err)
	require.EqualError(t, err, "unexpected image hash value")
}

func Test_Validate_ImageWhichIsNotInNotary_ShouldReturnError(t *testing.T) {
	f := func(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
		return nil, client.ErrRepositoryNotExist{}
	}
	s := NewDefaultMockNotaryService().WithFunc(f).Build()
	err := s.Validate(UntrustedImageName)
	require.Error(t, err)
	require.ErrorContains(t, err, "does not have trust data for")
}

func Test_Validate_ImageWhichIsInNotaryButIsNotInRegistry_ShouldReturnError(t *testing.T) {
	s := NewDefaultMockNotaryService().Build()
	err := s.Validate("eu.gcr.io/kyma-project/function-controller:unknown")
	require.Error(t, err)
	require.ErrorContains(t, err, "MANIFEST_UNKNOWN: Failed to fetch")
}

func Test_Validate_WhenNotaryNotResponding_ShouldReturnError(t *testing.T) {
	s := NewDefaultMockNotaryService().WithRepoFactory(MockNotaryRepoFactoryNoSuchHost{}).Build()
	err := s.Validate(TrustedImageName)
	require.Error(t, err)
	require.ErrorContains(t, err, "no such host")
}

func Test_Validate_WhenRegistryNotResponding_ShouldReturnError(t *testing.T) {
	s := NewDefaultMockNotaryService().Build()
	err := s.Validate("some.unknown.registry/kyma-project/function-controller:unknown")
	require.Error(t, err)
	require.ErrorContains(t, err, "no such host")
	require.ErrorContains(t, err, "lookup some.unknown.registry")
}

func Test_Validate_ImageWhichIsNotInNotaryButIsInAllowedList_ShouldPass(t *testing.T) {
	tests := []struct {
		name              string
		imageName         string
		allowedRegistries []string
	}{
		{
			name:      "image name is allowed",
			imageName: "some-registry/allowed-image-name:unstable",
			allowedRegistries: []string{
				"some-registry/allowed-image-name",
			},
		},
		{
			name:      "image name prefix is allowed",
			imageName: "some-registry/allowed-image-name:stable",
			allowedRegistries: []string{
				"some-registry/allowed-",
			},
		},
		{
			name:      "image name is one of allowed",
			imageName: "some-registry/allowed-image-name:latest",
			allowedRegistries: []string{
				"some-registry/allowed-image-1",
				"some-registry/allowed-image-name",
				"some-registry/allowed-image-3",
			},
		},
	}
	f := func(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
		panic("it shouldn't be called")
	}
	s := NewDefaultMockNotaryService().WithFunc(f).Build()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.NotaryConfig.AllowedRegistries = tt.allowedRegistries
			err := s.Validate(tt.imageName)
			require.NoError(t, err)
		})
	}
}

// TODO: now there is no timeout support
func Test_Validate_WhenNotaryRespondAfterLongTime_ShouldReturnError(t *testing.T) {
	t.Skip("now there is no timeout support")
	f := func(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
		time.Sleep(time.Second * 10)
		return &client.TargetWithRole{
			Target: client.Target{
				Name:   "ignored",
				Hashes: map[string][]byte{"ignored": TrustedImageHash},
				Length: 1,
			},
		}, nil
	}
	s := NewDefaultMockNotaryService().WithFunc(f).Build()
	err := s.Validate(TrustedImageName)
	require.Error(t, err)
	require.EqualError(t, err, "some timeout error")
}

// TODO: registry respond after long time

func Test_Validate_DEV(t *testing.T) {
	t.Skip("for testing and degugging real notary service")
	s := NewDefaultMockNotaryService().WithRepoFactory(NotaryRepoFactory{}).Build()
	s.NotaryConfig.Url = "https://signing-dev.repositories.cloud.sap"
	err := s.Validate(UntrustedImageName)
	require.Error(t, err)
	require.EqualError(t, err, "something")
}
