package validate

import (
	"fmt"
	"testing"
)

func TestNewReadOnlyRepo(t *testing.T) {
	nc := NotaryConfig{
		Url: "https://signing-dev.repositories.cloud.sap",
	}
	f := NotaryRepoFactory{}
	c, err := f.NewRepo("europe-docker.pkg.dev/kyma-project/dev/bootstrap", nc)
	if err != nil {
		t.Error(err)
	}
	name, err := c.GetTargetByName("PR-6200")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(name)
}
