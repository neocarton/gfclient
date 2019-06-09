package glclient

import (
	"time"
)

type (
	// Config config
	Config interface {
		Timeout() time.Duration
	}

	// DefaultConfig config
	DefaultConfig struct{}
)

func (DefaultConfig) Timeout() time.Duration {
	return time.Second
}
