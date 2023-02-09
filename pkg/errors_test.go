package pkg

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestErrorMsg(t *testing.T) {
	expectedErrMsg := "notary service unknown error: error: root err"
	t.Run("Wrap", func(t *testing.T) {
		//GIVEN
		err := errors.New("root err")
		err = errors.Wrap(err, "error")

		err = NewUnknownResultErr(err)
		//WHEN
		out := err.Error()

		//THEN
		require.Equal(t, expectedErrMsg, out)
	})

	t.Run("Errorf", func(t *testing.T) {
		//GIVEN
		err := errors.New("root err")
		err = fmt.Errorf("error: %w", err)

		err = NewUnknownResultErr(err)
		//WHEN
		out := err.Error()

		//THEN
		require.Equal(t, expectedErrMsg, out)
	})

}
