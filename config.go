package glclient

import (
	"time"
)

const (
	DefaultTimeout = 30 * time.Second
)

type (
	// Config config
	Config interface {
		Timeout() time.Duration
	}

	// DefaultConfig config
	DefaultConfig struct {
		timeout time.Duration
	}
)

// Timeout get timeout
func (cfg *DefaultConfig) Timeout() time.Duration {
	if cfg.timeout == 0 {
		return DefaultTimeout
	}
	return cfg.timeout
}

// SetTimeout set timeout
func (cfg *DefaultConfig) SetTimeout(timeout time.Duration) {
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	cfg.timeout = timeout
}
