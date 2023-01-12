package validate

import (
	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/tuf/data"
	"testing"
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
				Name:   "foo",
				Hashes: map[string][]byte{"bar": {1, 2, 3, 4}},
				Length: 1,
				Custom: nil,
			},
			Role: "rumburak",
		}, nil
	}
	s := notaryService{
		NotaryConfig: NotaryConfig{},
		RepoFactory: MockNotaryRepoFactory{
			GetTargetByNameFunc: &f,
		},
	}
	err := s.Validate("eu.gcr.io/kyma-project/function-controller:PR-16481")
	require.Error(t, err)
	require.EqualError(t, err, "unexpected image hash value")
}

// positive path - there are: image in notary, image in registry, equal hashes
// there isn't image in notary
// there isn't image in registry
// notary not responding
// registry not responding
// notary / registry respond after long time
