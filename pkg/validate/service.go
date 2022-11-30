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
	"errors"
	"fmt"
	"strings"
)

//go:generate mockery --name=ValidatorService
type PodValidatorService interface {
	Validate(image string) error
}

type notaryService struct {
	Type         string       `json:"type"`
	NotaryConfig NotaryConfig `json:"notaryConfig"`
}

func GetPodValidatorService() PodValidatorService {
	return createNotaryValidatorService()
}

func createNotaryValidatorService() PodValidatorService {

	return &notaryService{
		Type: "",
		NotaryConfig: NotaryConfig{
			Url: "https://signing-dev.repositories.cloud.sap",
		},
	}
}

func (s *notaryService) Validate(image string) error {

	if split := strings.Split(image, ":"); len(split) == 2 {
		imgRepo := split[0]
		imgTag := split[1]

		c, err := NewRepo(imgRepo, s.NotaryConfig)

		if err != nil {
			return err
		}

		// getting sha
		target, err := c.GetTargetByName(imgTag)
		if err != nil {
			return err
		}
		fmt.Println(target.Hashes)
		// TODO next steps ???
		return nil
	} else {
		return errors.New("Docker image name is not formatted correctly.")
	}
}
