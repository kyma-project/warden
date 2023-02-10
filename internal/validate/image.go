/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validate

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/kyma-project/warden/internal/helpers"
	"github.com/kyma-project/warden/pkg"
	"github.com/pkg/errors"
	"strings"
)

const (
	tagDelim = ":"
)

//go:generate mockery --name=ImageValidatorService
type ImageValidatorService interface {
	Validate(ctx context.Context, image string) error
}

type ServiceConfig struct {
	NotaryConfig      NotaryConfig
	AllowedRegistries []string
}

type notaryService struct {
	ServiceConfig
	RepoFactory RepoFactory
}

func NewImageValidator(sc *ServiceConfig, notaryClientFactory RepoFactory) ImageValidatorService {
	return &notaryService{
		ServiceConfig: ServiceConfig{
			NotaryConfig:      sc.NotaryConfig,
			AllowedRegistries: sc.AllowedRegistries,
		},
		RepoFactory: notaryClientFactory,
	}
}

func (s *notaryService) Validate(ctx context.Context, image string) error {
	logger := helpers.LoggerFromCtx(ctx).With("image", image)
	newCtx := helpers.LoggerToContext(ctx, logger)
	split := strings.Split(image, tagDelim)

	if len(split) != 2 {
		return pkg.NewValidationFailedErr(errors.New("image name is not formatted correctly"))
	}

	imgRepo := split[0]
	imgTag := split[1]

	if allowed := s.isImageAllowed(imgRepo); allowed {
		logger.Info("image validation skipped, because it's allowed")
		return nil
	}

	expectedShaBytes, err := s.loggedGetNotaryImageDigestHash(newCtx, imgRepo, imgTag)
	if err != nil {
		return err
	}

	shaBytes, err := s.loggedGetImageDigestHash(newCtx, image)
	if err != nil {
		return err
	}

	if subtle.ConstantTimeCompare(shaBytes, expectedShaBytes) == 0 {
		return pkg.NewValidationFailedErr(errors.New("unexpected image hash value"))
	}

	return nil
}

func (s *notaryService) isImageAllowed(imgRepo string) bool {
	for _, allowed := range s.AllowedRegistries {
		// repository is in allowed list
		if strings.HasPrefix(imgRepo, allowed) {
			return true
		}
	}
	return false
}

func (s *notaryService) loggedGetImageDigestHash(ctx context.Context, image string) ([]byte, error) {
	const message = "request to image registry"
	startTime := helpers.LogStartTime(ctx, message)
	result, err := s.getImageDigestHash(image)
	helpers.LogEndTime(ctx, message, startTime)
	return result, err
}

func (s *notaryService) getImageDigestHash(image string) ([]byte, error) {
	if len(image) == 0 {
		return []byte{}, pkg.NewValidationFailedErr(errors.New("empty image provided"))
	}

	ref, err := name.ParseReference(image)
	if err != nil {
		return []byte{}, pkg.NewValidationFailedErr(errors.Wrap(err, "ref parse"))
	}
	i, err := remote.Image(ref)
	if err != nil {
		return []byte{}, pkg.NewUnknownResultErr(errors.Wrap(err, "get image"))
	}
	m, err := i.Manifest()
	if err != nil {
		return []byte{}, pkg.NewUnknownResultErr(errors.Wrap(err, "image manifest"))
	}

	bytes, err := hex.DecodeString(m.Config.Digest.Hex)

	if err != nil {
		return []byte{}, pkg.NewUnknownResultErr(errors.Wrap(err, "checksum error: %w"))
	}

	return bytes, nil
}

func (s *notaryService) loggedGetNotaryImageDigestHash(ctx context.Context, imgRepo, imgTag string) ([]byte, error) {
	const message = "request to notary"
	startTime := helpers.LogStartTime(ctx, message)
	result, err := s.getNotaryImageDigestHash(ctx, imgRepo, imgTag)
	helpers.LogEndTime(ctx, message, startTime)
	return result, err
}

func (s *notaryService) getNotaryImageDigestHash(ctx context.Context, imgRepo, imgTag string) ([]byte, error) {
	if len(imgRepo) == 0 || len(imgTag) == 0 {
		return []byte{}, pkg.NewValidationFailedErr(errors.New("empty arguments provided"))
	}

	const messageNewRepoClient = "request to notary (NewRepoClient)"
	startTimeNewRepoClient := helpers.LogStartTime(ctx, messageNewRepoClient)
	c, err := s.RepoFactory.NewRepoClient(imgRepo, s.NotaryConfig)
	helpers.LogEndTime(ctx, messageNewRepoClient, startTimeNewRepoClient)
	if err != nil {
		return []byte{}, pkg.NewUnknownResultErr(err)
	}

	const messageGetTargetByName = "request to notary (GetTargetByName)"
	startTimeGetTargetByName := helpers.LogStartTime(ctx, messageGetTargetByName)
	target, err := c.GetTargetByName(imgTag)
	helpers.LogEndTime(ctx, messageGetTargetByName, startTimeGetTargetByName)
	if err != nil {
		return []byte{}, parseNotaryErr(err)
	}

	if len(target.Hashes) == 0 {
		return []byte{}, pkg.NewValidationFailedErr(errors.New("image hash is missing"))
	}

	if len(target.Hashes) > 1 {
		return []byte{}, pkg.NewValidationFailedErr(errors.New("more than one hash for image"))
	}

	key := ""
	for i := range target.Hashes {
		key = i
	}

	return target.Hashes[key], nil
}

func parseNotaryErr(err error) error {
	errMsg := err.Error()
	if strings.Contains(errMsg, "does not have trust data for") {
		return pkg.NewValidationFailedErr(err)
	}
	if strings.Contains(errMsg, "No valid trust data for") {
		return pkg.NewValidationFailedErr(err)
	}
	return pkg.NewUnknownResultErr(err)
}
