package validate_test

import (
	"errors"
	"fmt"
	"github.com/kyma-project/warden/internal/validate"
	"github.com/kyma-project/warden/internal/validate/mocks"
	"github.com/kyma-project/warden/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary/client"
	"golang.org/x/net/context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type image struct {
	name string
	tag  string
	hash []byte
}

func (i image) image() string {
	return fmt.Sprintf("%s:%s", i.name, i.tag)
}

var (
	trustedImage = image{
		name: "eu.gcr.io/kyma-project/function-controller",
		tag:  "PR-16481",
		hash: []byte{243, 155, 151, 155, 35, 94, 175, 164, 30, 8, 73, 56, 233, 106, 9, 124, 3, 46, 36, 141, 41, 227, 150, 143, 207, 210, 152, 26, 190, 95, 17, 166},
	}
	differentHashImage = image{
		name: "nginx",
		tag:  "latest",
		hash: []byte{1, 2, 3, 4},
	}
	untrustedImage = image{
		name: "nginx",
		tag:  "untrusted",
	}
	unknownImage = image{
		name: "eu.gcr.io/kyma-project/function-controller",
		tag:  "unknow",
	}
)

func Test_Validate_ProperImage_ShouldPass(t *testing.T) {
	cfg := validate.ServiceConfig{NotaryConfig: validate.NotaryConfig{}}
	f := setupMockFactory()

	s := validate.NewImageValidator(&cfg, f)
	err := s.Validate(context.TODO(), trustedImage.image())
	require.NoError(t, err)
}

func Test_Validate_InvalidImageName_ShouldReturnError(t *testing.T) {
	cfg := validate.ServiceConfig{NotaryConfig: validate.NotaryConfig{}}
	f := setupMockFactory()

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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := validate.NewImageValidator(&cfg, f)

			err := s.Validate(context.TODO(), tt.imageName)

			require.ErrorContains(t, err, tt.expectedErrMsg)
		})
	}
}

func Test_Validate_ImageWithDifferentHashInNotary_ShouldReturnError(t *testing.T) {
	//GIVEN
	cfg := validate.ServiceConfig{NotaryConfig: validate.NotaryConfig{}}
	f := setupMockFactory()
	s := validate.NewImageValidator(&cfg, f)
	//WHEN
	err := s.Validate(context.TODO(), differentHashImage.image())

	//THEN
	require.ErrorContains(t, err, "unexpected image hash value")
	require.Equal(t, pkg.ValidationError, pkg.ErrorCode(err))
}

func Test_Validate_ImageWhichIsNotInNotary_ShouldReturnError(t *testing.T) {
	//GIVE
	cfg := validate.ServiceConfig{NotaryConfig: validate.NotaryConfig{}}
	f := setupMockFactory()
	s := validate.NewImageValidator(&cfg, f)

	//WHEN
	err := s.Validate(context.TODO(), untrustedImage.image())

	//THEN
	require.ErrorContains(t, err, "does not have trust data for")
	require.Equal(t, pkg.ValidationError, pkg.ErrorCode(err))
}

func Test_Validate_ImageWhichIsInNotaryButIsNotInRegistry_ShouldReturnError(t *testing.T) {
	//GIVE
	cfg := validate.ServiceConfig{NotaryConfig: validate.NotaryConfig{}}
	f := setupMockFactory()
	s := validate.NewImageValidator(&cfg, f)

	//WHEN
	err := s.Validate(context.TODO(), unknownImage.image())

	//THEN
	require.ErrorContains(t, err, "MANIFEST_UNKNOWN: Failed to fetch")
	require.Equal(t, pkg.UnknownResult, pkg.ErrorCode(err))
}

func Test_Validate_WhenNotaryNotResponding_ShouldReturnError(t *testing.T) {
	//GIVE
	f := &mocks.RepoFactory{}
	f.On("NewRepoClient", mock.Anything, mock.Anything).Return(nil, errors.New("no such host"))
	cfg := validate.ServiceConfig{NotaryConfig: validate.NotaryConfig{}}
	s := validate.NewImageValidator(&cfg, f)

	//WHEN
	err := s.Validate(context.TODO(), trustedImage.image())

	//THEN
	require.ErrorContains(t, err, "no such host")
	require.Equal(t, pkg.UnknownResult, pkg.ErrorCode(err))
}

func Test_Validate_WhenRegistryNotResponding_ShouldReturnError(t *testing.T) {
	//GIVE
	notaryClient := &mocks.NotaryRepoClient{}
	response := &client.TargetWithRole{Target: client.Target{Name: "ignored",
		Hashes: map[string][]byte{"ignored": trustedImage.hash},
		Length: 1}}
	notaryClient.On("GetTargetByName", "unknown").Return(response, nil)
	f := &mocks.RepoFactory{}
	f.On("NewRepoClient", mock.Anything, mock.Anything).Return(notaryClient, nil)
	cfg := validate.ServiceConfig{NotaryConfig: validate.NotaryConfig{}}
	s := validate.NewImageValidator(&cfg, f)

	//WHEN
	err := s.Validate(context.TODO(), "some.unknown.registry/kyma-project/function-controller:unknown")

	//THEN
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
	f := &mocks.RepoFactory{}
	f.On("NewRepoClient", mock.Anything, mock.Anything).Return(nil, errors.New("Should be called"))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//GIVEN
			cfg := validate.ServiceConfig{NotaryConfig: validate.NotaryConfig{}, AllowedRegistries: tt.allowedRegistries}
			s := validate.NewImageValidator(&cfg, f)

			//WHEN
			err := s.Validate(context.TODO(), tt.imageName)

			//THEN
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

	sc := &validate.ServiceConfig{
		NotaryConfig: validate.NotaryConfig{
			Url: testServer.URL,
		},
	}
	f := validate.NotaryRepoFactory{timeout}
	validator := validate.NewImageValidator(sc, f)

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

	sc := &validate.ServiceConfig{
		NotaryConfig: validate.NotaryConfig{
			Url: testServer.URL,
		},
	}
	f := validate.NotaryRepoFactory{5 * time.Second}
	validator := validate.NewImageValidator(sc, f)

	//WHEN
	err := validator.Validate(context.TODO(), "europe-docker.pkg.dev/kyma-project/dev/bootstrap:PR-6200")

	//THEN
	require.ErrorContains(t, err, "couldn't correctly connect to notary")
	require.Equal(t, pkg.UnknownResult, pkg.ErrorCode(err))
}

func Test_Validate_DEV(t *testing.T) {
	t.Skip("for testing and debugging real notary service")
	sc := &validate.ServiceConfig{NotaryConfig: validate.NotaryConfig{
		Url: "https://signing-dev.repositories.cloud.sap"}}
	s := validate.NewImageValidator(sc, validate.NotaryRepoFactory{})

	//WHEN
	err := s.Validate(context.TODO(), untrustedImage.image())

	//THEN
	require.ErrorContains(t, err, "something")
}

func setupMockFactory() validate.RepoFactory {
	notaryClient := &mocks.NotaryRepoClient{}

	trusted := client.TargetWithRole{Target: client.Target{Name: "ignored",
		Hashes: map[string][]byte{"ignored": trustedImage.hash},
		Length: 1}}

	unknown := client.TargetWithRole{Target: client.Target{Name: "ignored",
		Hashes: map[string][]byte{"ignored": unknownImage.hash},
		Length: 1}}

	different := client.TargetWithRole{Target: client.Target{Name: "ignored",
		Hashes: map[string][]byte{"ignored": differentHashImage.hash}}}

	notaryClient.On("GetTargetByName", trustedImage.tag).Return(&trusted, nil)
	notaryClient.On("GetTargetByName", differentHashImage.tag).Return(&different, nil)
	notaryClient.On("GetTargetByName", unknownImage.tag).Return(&unknown, nil)
	notaryClient.On("GetTargetByName", untrustedImage.tag).
		Return(nil, fmt.Errorf("does not have trust data for %s", untrustedImage.name))

	f := &mocks.RepoFactory{}
	f.On("NewRepoClient", mock.Anything, mock.Anything).Return(notaryClient, nil)

	return f
}
