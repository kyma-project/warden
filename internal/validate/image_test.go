package validate_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kyma-project/warden/internal/validate"
	"github.com/kyma-project/warden/internal/validate/mocks"
	"github.com/kyma-project/warden/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary/client"
	"golang.org/x/net/context"
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
		name: "europe-docker.pkg.dev/kyma-project/prod/function-controller",
		tag:  "v20230428-1ea34f8e",
		// image hash
		hash: []byte{223, 6, 148, 15, 106, 95, 90, 178, 129, 233, 166, 72, 164, 160, 88, 104, 72, 130, 62, 48, 240, 49, 177, 42, 108, 15, 138, 138, 255, 113, 176, 239},
	}
	trustedImageLegacy = image{
		name: "europe-docker.pkg.dev/kyma-project/prod/function-controller",
		tag:  "v20240731-b8af3f9c",
		// manifest hash
		hash: []byte{157, 125, 211, 253, 79, 175, 129, 184, 184, 72, 163, 165, 92, 251, 19, 70, 92, 162, 125, 90, 135, 102, 39, 28, 194, 201, 221, 188, 72, 73, 136, 239},
	}
	trustedIndex = image{
		name: "europe-docker.pkg.dev/kyma-project/prod/external/golang",
		tag:  "1.22.2-alpine3.19",
		// index hash
		hash: []byte{205, 200, 109, 159, 54, 62, 135, 134, 132, 91, 234, 32, 64, 49, 43, 78, 250, 50, 27, 130, 138, 205, 235, 38, 243, 147, 250, 168, 100, 216, 135, 176},
	}
	differentHashIndex = image{
		name: "europe-docker.pkg.dev/kyma-project/prod/external/alpine",
		tag:  "3.20.0",
		// image hash instead of index hash
		hash: []byte{33, 98, 102, 200, 111, 196, 220, 239, 86, 25, 147, 11, 211, 148, 36, 88, 36, 194, 175, 82, 253, 33, 186, 124, 111, 160, 230, 24, 101, 125, 76, 59},
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

func Test_Validate_ProperImageLegacy_ShouldPass(t *testing.T) {
	cfg := validate.ServiceConfig{NotaryConfig: validate.NotaryConfig{}}
	f := setupMockFactory()

	s := validate.NewImageValidator(&cfg, f)
	err := s.Validate(context.TODO(), trustedImageLegacy.image())
	require.NoError(t, err)
}

func Test_Validate_ProperIndex_ShouldPass(t *testing.T) {
	cfg := validate.ServiceConfig{NotaryConfig: validate.NotaryConfig{}}
	f := setupMockFactory()

	s := validate.NewImageValidator(&cfg, f)
	err := s.Validate(context.TODO(), trustedIndex.image())
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

func Test_Validate_IndexWithDifferentHashInNotary_ShouldReturnError(t *testing.T) {
	//GIVEN
	cfg := validate.ServiceConfig{NotaryConfig: validate.NotaryConfig{}}
	f := setupMockFactory()
	s := validate.NewImageValidator(&cfg, f)
	//WHEN
	err := s.Validate(context.TODO(), differentHashIndex.image())

	//THEN
	require.ErrorContains(t, err, "unexpected image hash value")
	require.Equal(t, pkg.ValidationError, pkg.ErrorCode(err))
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
	f := validate.NotaryRepoFactory{Timeout: 5 * time.Second}
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

	trustedLegacy := client.TargetWithRole{Target: client.Target{Name: "ignored",
		Hashes: map[string][]byte{"ignored": trustedImageLegacy.hash},
		Length: 1}}

	trustedImageIndex := client.TargetWithRole{Target: client.Target{Name: "ignored",
		Hashes: map[string][]byte{"ignored": trustedIndex.hash},
		Length: 1}}

	unknown := client.TargetWithRole{Target: client.Target{Name: "ignored",
		Hashes: map[string][]byte{"ignored": unknownImage.hash},
		Length: 1}}

	different := client.TargetWithRole{Target: client.Target{Name: "ignored",
		Hashes: map[string][]byte{"ignored": differentHashImage.hash}}}

	differentIndex := client.TargetWithRole{Target: client.Target{Name: "ignored",
		Hashes: map[string][]byte{"ignored": differentHashIndex.hash},
		Length: 1}}

	notaryClient.On("GetTargetByName", trustedImage.tag).Return(&trusted, nil)
	notaryClient.On("GetTargetByName", trustedImageLegacy.tag).Return(&trustedLegacy, nil)
	notaryClient.On("GetTargetByName", trustedIndex.tag).Return(&trustedImageIndex, nil)
	notaryClient.On("GetTargetByName", differentHashIndex.tag).Return(&differentIndex, nil)
	notaryClient.On("GetTargetByName", differentHashImage.tag).Return(&different, nil)
	notaryClient.On("GetTargetByName", unknownImage.tag).Return(&unknown, nil)
	notaryClient.On("GetTargetByName", untrustedImage.tag).
		Return(nil, fmt.Errorf("does not have trust data for %s", untrustedImage.name))

	f := &mocks.RepoFactory{}
	f.On("NewRepoClient", mock.Anything, mock.Anything).Return(notaryClient, nil)

	return f
}
