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
	"github.com/docker/distribution/registry/client/auth"
	"github.com/docker/distribution/registry/client/auth/challenge"
	"github.com/docker/distribution/registry/client/transport"
	"github.com/kyma-project/warden/pkg"
	"github.com/pkg/errors"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/trustpinning"
	"github.com/theupdateframework/notary/tuf/data"
	"net"
	"net/http"
	"time"
)

const (
	NotaryDefaultTrustDir = "/tmp/.notary"
)

type NotaryConfig struct {
	Url string `json:"url"`
}

type NotaryValidator struct {
}

//go:generate mockery --name NotaryRepoClient
type NotaryRepoClient interface {
	client.Repository
}

//go:generate mockery --name RepoFactory
type RepoFactory interface {
	NewRepoClient(string, NotaryConfig) (NotaryRepoClient, error)
}

type NotaryRepoFactory struct {
	Timeout time.Duration
}

func (f NotaryRepoFactory) NewRepoClient(img string, c NotaryConfig) (NotaryRepoClient, error) {
	base := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		TLSHandshakeTimeout: 10 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   f.Timeout,
			KeepAlive: f.Timeout,
		}).DialContext,
		DisableKeepAlives: true,
	}
	th := auth.NewTokenHandlerWithOptions(auth.TokenHandlerOptions{
		Transport: base,
		Scopes: []auth.Scope{
			auth.RepositoryScope{
				Repository: img,
				Actions:    []string{"pull"},
			},
		},
	})

	// challenge manager expects to connect to /v2/ endpoint to obtain the challenges:
	// https://github.com/notaryproject/notary/blob/master/vendor/github.com/docker/distribution/registry/client/auth/session.go#L75
	u := c.Url + "/v2/"
	pingClient := &http.Client{
		Transport: base,
		Timeout:   f.Timeout,
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := pingClient.Do(req)
	if err != nil {
		return nil, err
	}
	// nil err means we must close body
	defer resp.Body.Close()
	if (resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices) &&
		resp.StatusCode != http.StatusUnauthorized {
		// If we didn't get a 2XX range or 401 status code, we're not talking to a notary server.
		// The http client should be configured to handle redirects so at this point, 3XX is
		// not a valid status code.
		return nil, pkg.NewUnknownResultErr(errors.Errorf("couldn't correctly connect to notary, status code: %d", resp.StatusCode))
	}

	cm := challenge.NewSimpleManager()
	if err = cm.AddResponse(resp); err != nil {
		return nil, err
	}
	modifier := auth.NewAuthorizer(cm, th)
	return client.NewFileCachedRepository(NotaryDefaultTrustDir, data.GUN(img), c.Url, transport.NewTransport(base, modifier), nil, trustpinning.TrustPinConfig{})
}
