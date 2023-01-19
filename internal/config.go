package internal

import "time"

type Config struct {
	Timeout time.Duration `envconfig:"default=2m"`
}
