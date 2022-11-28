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
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/passphrase"
	"github.com/theupdateframework/notary/trustpinning"
	"github.com/theupdateframework/notary/tuf/data"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	NotaryDefaultTrustDir = ".notary"
)

type NotaryConfig struct {
	Url string `json:"url"`
}

type NotaryValidator struct {
}

func NewReadOnlyRepo(img string, c NotaryConfig) (client.Repository, error) {

	base := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		TLSHandshakeTimeout: 10 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig:   nil,
		DisableKeepAlives: true,
	}
	cm := challenge.NewSimpleManager()
	ps := passwordStore{anonymous: true}
	bh := auth.NewBasicHandler(ps)
	th := auth.NewTokenHandler(base, ps, img, "pull")
	modifier := auth.NewAuthorizer(cm, bh)
	transport.NewTransport(base, auth.NewAuthorizer(cm, th))
	return client.NewFileCachedRepository(NotaryDefaultTrustDir, data.GUN(img), c.Url, transport.NewTransport(base, modifier), getPassphraseRetriever(), trustpinning.TrustPinConfig{})
}

func getPassphraseRetriever() notary.PassRetriever {
	baseRetriever := passphrase.PromptRetriever()
	env := map[string]string{
		"root":       os.Getenv("NOTARY_ROOT_PASSPHRASE"),
		"targets":    os.Getenv("NOTARY_TARGETS_PASSPHRASE"),
		"snapshot":   os.Getenv("NOTARY_SNAPSHOT_PASSPHRASE"),
		"delegation": os.Getenv("NOTARY_DELEGATION_PASSPHRASE"),
	}

	return func(keyName string, alias string, createNew bool, numAttempts int) (string, bool, error) {
		if v := env[alias]; v != "" {
			return v, numAttempts > 1, nil
		}
		// For delegation roles, we can also try the "delegation" alias if it is specified
		// Note that we don't check if the role name is for a delegation to allow for names like "user"
		// since delegation keys can be shared across repositories
		// This cannot be a base role or imported key, though.
		if v := env["delegation"]; !data.IsBaseRole(data.RoleName(alias)) && v != "" {
			return v, numAttempts > 1, nil
		}
		return baseRetriever(keyName, alias, createNew, numAttempts)
	}
}

type passwordStore struct {
	anonymous bool
}

func (p passwordStore) Basic(url *url.URL) (string, string) {
	//if p.anonymous {
	//	return "", ""
	//}
	return "", ""
}

func (p passwordStore) RefreshToken(url *url.URL, s string) string {
	return ""
}

func (p passwordStore) SetRefreshToken(realm *url.URL, service, token string) {}
