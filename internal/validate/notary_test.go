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
	t.Skip("This is not unit test because it reach the external system")
	nc := NotaryConfig{
		Url: "https://signing-dev.repositories.cloud.sap",
	}
	f := NotaryRepoFactory{}
	c, err := f.NewRepoClient("europe-docker.pkg.dev/kyma-project/dev/bootstrap", nc)
	require.NoError(t, err)

	name, err := c.GetTargetByName("PR-6200")
	require.NoError(t, err)
	fmt.Println(name)
}

func TestNotaryHTTPTimeout(t *testing.T) {
	//GIVEN
	//The minimum value is 1 seconds everything less than it
	timeout := time.Millisecond * 1000
	start := time.Now()

	h := func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(2 * timeout)
	}
	handler := http.HandlerFunc(h)

	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	nc := NotaryConfig{
		Url: testServer.URL,
	}
	f := NotaryRepoFactory{Timeout: time.Second}

	//WHEN
	_, err := f.NewRepoClient("europe-docker.pkg.dev/kyma-project/dev/bootstrap", nc)

	//THEn
	require.Error(t, err)
	require.ErrorContains(t, err, "context deadline exceeded")
	require.InDelta(t, timeout.Milliseconds(), time.Since(start).Milliseconds(), 100, "timeout duration is not respected")

}
