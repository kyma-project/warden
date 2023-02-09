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

	err = NewUnknownResultErr(err)
	//WHEN
	out := err.Error()

	//THEN
	require.Equal(t, "notary service unknown error: error: root err", out)
}
