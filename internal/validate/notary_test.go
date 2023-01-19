package validate

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewReadOnlyRepo(t *testing.T) {
	nc := NotaryConfig{
		Url: "https://signing-dev.repositories.cloud.sap",
	}
	f := NotaryRepoFactory{}
	c, err := f.NewClient("europe-docker.pkg.dev/kyma-project/dev/bootstrap", nc)
	require.NoError(t, err)

	name, err := c.GetTargetByName("PR-6200")
	require.NoError(t, err)
	fmt.Println(name)
}

func TestTimeout(t *testing.T) {
	//GIVEN
	h := func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(40 * time.Second)
	}
	handler := http.HandlerFunc(h)

	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	nc := NotaryConfig{
		Url: testServer.URL,
	}
	f := NotaryRepoFactory{}
	c, err := f.NewClient("europe-docker.pkg.dev/kyma-project/dev/bootstrap", nc)
	require.NoError(t, err)
	//WHEN
	name, err := c.GetTargetByName("PR-6200")

	//THEn
	require.NoError(t, err)
	fmt.Println(name)
}
