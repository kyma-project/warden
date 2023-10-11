package env

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetBool(t *testing.T) {
	t.Run("get bool env", func(t *testing.T) {
		envKey := "TEST_GETBOOL_KEY"
		t.Setenv(envKey, "true")

		value, err := GetBool(envKey)
		require.NoError(t, err)
		require.Equal(t, true, value)
	})

	t.Run("get default false when env does not exist", func(t *testing.T) {
		envKey := "TEST_GETBOOL_KEY_NOT_EXISTS"

		value, err := GetBool(envKey)
		require.NoError(t, err)
		require.Equal(t, false, value)
	})

	t.Run("env value not bool error", func(t *testing.T) {
		envKey := "TEST_GETBOOL_KEY_NOT_BOOL"
		t.Setenv(envKey, "123123")

		value, err := GetBool(envKey)
		require.Error(t, err)
		require.Equal(t, false, value)
	})
}
