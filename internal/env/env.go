package env

import (
	"os"
	"strconv"
)

func Get(key string) string {
	return os.Getenv(key)
}

func GetBool(key string) (bool, error) {
	stringEnv := Get(key)
	if stringEnv == "" {
		return false, nil
	}

	return strconv.ParseBool(stringEnv)
}
