package pkg

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestErrorMsg(t *testing.T) {
	//GIVEN
	err := errors.New("root err")
	err = fmt.Errorf("error: %w", err)

	err = NewServiceUnavailableError(err)
	//WHEN
	out := err.Error()

	//THEN
	require.Equal(t, "notary service unavailable error: error: root err", out)
}
