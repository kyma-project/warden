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

func Test_Validate_InvalidName_ShouldReturnError(t *testing.T) {
	s := notaryService{
		NotaryConfig: NotaryConfig{},
		RepoFactory:  MockNotaryRepoFactory{},
	}
	err := s.Validate("makapaka")
	require.Error(t, err)
	require.EqualError(t, err, "image name is not formatted correctly")
}

func Test_Validate_ImageWithDifferentHashInNotary_ShouldReturnError(t *testing.T) {
	f := func(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
		return &client.TargetWithRole{
			Target: client.Target{
				Name:   "ignored",
				Hashes: map[string][]byte{"ignored": {1, 2, 3, 4}},
				Length: 1,
				Custom: nil,
			},
			Role: "ignored",
		}, nil
	}
	s := notaryService{
		NotaryConfig: NotaryConfig{},
		RepoFactory: MockNotaryRepoFactory{
			GetTargetByNameFunc: &f,
		},
	}
	err := s.Validate(TrustedImageName)
	require.Error(t, err)
	require.EqualError(t, err, "unexpected image hash value")
}

// positive path - there are: image in notary, image in registry, equal hashes
func Test_Validate_ProperImage_ShouldPass(t *testing.T) {
	f := func(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
		return &client.TargetWithRole{
			Target: client.Target{
				Name:   "ignored",
				Hashes: map[string][]byte{"ignored": {243, 155, 151, 155, 35, 94, 175, 164, 30, 8, 73, 56, 233, 106, 9, 124, 3, 46, 36, 141, 41, 227, 150, 143, 207, 210, 152, 26, 190, 95, 17, 166}},
				Length: 1,
				Custom: nil,
			},
			Role: "ignored",
		}, nil
	}
	s := notaryService{
		NotaryConfig: NotaryConfig{},
		RepoFactory: MockNotaryRepoFactory{
			GetTargetByNameFunc: &f,
		},
	}
	err := s.Validate(TrustedImageName)
	require.NoError(t, err)
}

// there isn't image in notary
func Test_Validate_ImageWhichIsNotInNotary_ShouldReturnError(t *testing.T) {
	f := func(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
		return nil, client.ErrRepositoryNotExist{}
	}
	s := notaryService{
		NotaryConfig: NotaryConfig{},
		RepoFactory: MockNotaryRepoFactory{
			GetTargetByNameFunc: &f,
		},
	}
	err := s.Validate(UntrustedImageName)
	require.Error(t, err)
	require.ErrorContains(t, err, "does not have trust data for")
}

// there isn't image in registry
func Test_Validate_ImageWhichIsInNotaryButIsNotInRegistry_ShouldReturnError(t *testing.T) {
	f := func(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
		return &client.TargetWithRole{
			Target: client.Target{
				Name:   "ignored",
				Hashes: map[string][]byte{"ignored": {1, 2, 3, 4}},
				Length: 1,
				Custom: nil,
			},
			Role: "ignored",
		}, nil
	}
	s := notaryService{
		NotaryConfig: NotaryConfig{},
		RepoFactory: MockNotaryRepoFactory{
			GetTargetByNameFunc: &f,
		},
	}
	err := s.Validate("eu.gcr.io/kyma-project/function-controller:unknown")
	require.Error(t, err)
	require.ErrorContains(t, err, "MANIFEST_UNKNOWN: Failed to fetch")
}

// notary not responding
func Test_Validate_WhenNotaryNotResponding_ShouldReturnError(t *testing.T) {
	s := notaryService{
		NotaryConfig: NotaryConfig{},
		RepoFactory:  MockNotaryRepoFactoryNoSuchHost{},
	}
	err := s.Validate(TrustedImageName)
	require.Error(t, err)
	require.ErrorContains(t, err, "no such host")
}

// registry not responding
func Test_Validate_WhenRegistryNotResponding_ShouldReturnError(t *testing.T) {
	f := func(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
		return &client.TargetWithRole{
			Target: client.Target{
				Name:   "ignored",
				Hashes: map[string][]byte{"ignored": {1, 2, 3, 4}},
				Length: 1,
				Custom: nil,
			},
			Role: "ignored",
		}, nil
	}
	s := notaryService{
		NotaryConfig: NotaryConfig{},
		RepoFactory: MockNotaryRepoFactory{
			GetTargetByNameFunc: &f,
		},
	}
	err := s.Validate("some.unknown.registry/kyma-project/function-controller:unknown")
	require.Error(t, err)
	require.ErrorContains(t, err, "lookup some.unknown.registry: no such host")
}

// image is in allowedList
func Test_Validate_ImageWhichIsNotInNotaryButIsInAllowedList_ShouldPass(t *testing.T) {
	f := func(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
		panic("it shouldn't be called")
	}
	s := notaryService{
		NotaryConfig: NotaryConfig{
			AllowedRegistries: []string{
				"qwertyuiop",
				"some-registry/allowed-image-name",
				"asdfghjkl",
			},
		},
		RepoFactory: MockNotaryRepoFactory{
			GetTargetByNameFunc: &f,
		},
	}
	err := s.Validate("some-registry/allowed-image-name:unstable")
	require.NoError(t, err)
}

// image prefix is in allowedList
func Test_Validate_ImageWhichIsNotInNotaryButHasPrefixInAllowedList_ShouldPass(t *testing.T) {
	f := func(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
		panic("it shouldn't be called")
	}
	s := notaryService{
		NotaryConfig: NotaryConfig{
			AllowedRegistries: []string{
				"some-registry/allowed-",
			},
		},
		RepoFactory: MockNotaryRepoFactory{
			GetTargetByNameFunc: &f,
		},
	}
	err := s.Validate("some-registry/allowed-image-name:unstable")
	require.NoError(t, err)
}

// notary respond after long time
func Test_Validate_WhenNotaryRespondAfterLongTime_ShouldReturnError(t *testing.T) {
	f := func(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
		time.Sleep(time.Second * 10)
		return &client.TargetWithRole{
			Target: client.Target{
				Name:   "ignored",
				Hashes: map[string][]byte{"ignored": {243, 155, 151, 155, 35, 94, 175, 164, 30, 8, 73, 56, 233, 106, 9, 124, 3, 46, 36, 141, 41, 227, 150, 143, 207, 210, 152, 26, 190, 95, 17, 166}},
				Length: 1,
				Custom: nil,
			},
		}, nil
	}
	s := notaryService{
		NotaryConfig: NotaryConfig{},
		RepoFactory: MockNotaryRepoFactory{
			GetTargetByNameFunc: &f,
		},
	}
	err := s.Validate(TrustedImageName)
	require.Error(t, err)
	require.EqualError(t, err, "some timeout error")
}

// registry respond after long time

func Test_Validate_DEV(t *testing.T) {
	s := notaryService{
		NotaryConfig: NotaryConfig{
			Url:               "https://signing-dev.repositories.cloud.sap",
			AllowedRegistries: []string{},
		},
		RepoFactory: NotaryRepoFactory{},
	}
	err := s.Validate(UntrustedImageName)
	require.Error(t, err)
	require.EqualError(t, err, "something")
}
