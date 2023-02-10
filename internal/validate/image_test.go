package validate

import (
	"github.com/kyma-project/warden/pkg"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/tuf/data"
	"golang.org/x/net/context"
	"net/http"
	"net/http/httptest"
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
	err := s.Validate(context.TODO(), TrustedImageName)
	require.NoError(t, err)
}

func Test_Validate_InvalidImageName_ShouldReturnError(t *testing.T) {
	tests := []struct {
		name           string
		imageName      string
		expectedErrMsg string
	}{
		{
			name:           "image name without semicolon",
			imageName:      "makapaka",
			expectedErrMsg: "image name is not formatted correctly",
		},
		{
			name:           "",
			imageName:      ":",
			expectedErrMsg: "empty arguments provided",
		},
		{
			name:           "image name with more than one semicolon", //TODO: IMO it's proper image name, but now is not allowed
			imageName:      "repo:port/image-name:tag",
			expectedErrMsg: "image name is not formatted correctly",
		},
	}
	s := NewDefaultMockNotaryService().Build()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.Validate(context.TODO(), tt.imageName)

			require.ErrorContains(t, err, tt.expectedErrMsg)
		})
	}
}

func Test_Validate_ImageWithDifferentHashInNotary_ShouldReturnError(t *testing.T) {
	s := NewDefaultMockNotaryService().Build()
	err := s.Validate(context.TODO(), TrustedImageName)
	require.ErrorContains(t, err, "unexpected image hash value")
	require.Equal(t, pkg.ValidationError, pkg.ErrorCode(err))
}

func Test_Validate_ImageWhichIsNotInNotary_ShouldReturnError(t *testing.T) {
	f := func(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
		return nil, client.ErrRepositoryNotExist{}
	}
	s := NewDefaultMockNotaryService().WithFunc(f).Build()
	err := s.Validate(context.TODO(), UntrustedImageName)
	require.ErrorContains(t, err, "does not have trust data for")
	require.Equal(t, pkg.ValidationError, pkg.ErrorCode(err))
}

func Test_Validate_ImageWhichIsInNotaryButIsNotInRegistry_ShouldReturnError(t *testing.T) {
	s := NewDefaultMockNotaryService().Build()
	err := s.Validate(context.TODO(), "eu.gcr.io/kyma-project/function-controller:unknown")
	require.ErrorContains(t, err, "MANIFEST_UNKNOWN: Failed to fetch")
	require.Equal(t, pkg.UnknownResult, pkg.ErrorCode(err))
}

func Test_Validate_WhenNotaryNotResponding_ShouldReturnError(t *testing.T) {
	s := NewDefaultMockNotaryService().WithRepoFactory(MockNotaryRepoFactoryNoSuchHost{}).Build()
	err := s.Validate(context.TODO(), TrustedImageName)
	require.ErrorContains(t, err, "no such host")
	require.Equal(t, pkg.UnknownResult, pkg.ErrorCode(err))
}

func Test_Validate_WhenRegistryNotResponding_ShouldReturnError(t *testing.T) {
	s := NewDefaultMockNotaryService().Build()
	err := s.Validate(context.TODO(), "some.unknown.registry/kyma-project/function-controller:unknown")
	require.ErrorContains(t, err, "no such host")
	require.ErrorContains(t, err, "lookup some.unknown.registry")
	require.Equal(t, pkg.UnknownResult, pkg.ErrorCode(err))
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
		return nil, errors.New("it shouldn't be called")
	}
	s := NewDefaultMockNotaryService().WithFunc(f).Build()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.AllowedRegistries = tt.allowedRegistries
			err := s.Validate(context.TODO(), tt.imageName)
			require.NoError(t, err)
		})
	}
}

func Test_Validate_WhenNotaryRespondAfterLongTime_ShouldReturnError(t *testing.T) {
	//GIVEN
	timeout := time.Second * 1
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	h := func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(2 * timeout)
	}
	handler := http.HandlerFunc(h)

	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	sc := &ServiceConfig{
		NotaryConfig: NotaryConfig{
			Url: testServer.URL,
		},
	}
	f := NotaryRepoFactory{timeout}
	validator := NewImageValidator(sc, f)

	//WHEN
	err := validator.Validate(ctx, "europe-docker.pkg.dev/kyma-project/dev/bootstrap:PR-6200")

	//THEN
	assert.ErrorContains(t, err, "context deadline exceeded")
	require.InDelta(t, timeout.Seconds(), time.Since(start).Seconds(), 0.1, "timeout duration is not respected")
	require.Equal(t, pkg.UnknownResult, pkg.ErrorCode(err))
}

func Test_Validate_WhenNotaryRespondWithError_ShouldReturnServiceNotAvailable(t *testing.T) {

	h := func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(404)
	}
	handler := http.HandlerFunc(h)

	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	sc := &ServiceConfig{
		NotaryConfig: NotaryConfig{
			Url: testServer.URL,
		},
	}
	f := NotaryRepoFactory{5 * time.Second}
	validator := NewImageValidator(sc, f)

	//WHEN
	err := validator.Validate(context.TODO(), "europe-docker.pkg.dev/kyma-project/dev/bootstrap:PR-6200")

	//THEN
	require.ErrorContains(t, err, "couldn't correctly connect to notary")
	require.Equal(t, pkg.UnknownResult, pkg.ErrorCode(err))
}

func Test_Validate_DEV(t *testing.T) {
	t.Skip("for testing and debugging real notary service")
	s := NewDefaultMockNotaryService().WithRepoFactory(NotaryRepoFactory{}).Build()
	s.NotaryConfig.Url = "https://signing-dev.repositories.cloud.sap"
	err := s.Validate(context.TODO(), UntrustedImageName)
	require.ErrorContains(t, err, "something")
}
