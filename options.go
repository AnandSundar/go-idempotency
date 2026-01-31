package idempotency

import (
	"net/http"
	"time"
)

// Config holds middleware configuration
type Config struct {
	HeaderName string
	TTL        time.Duration
	KeyFunc    KeyFunc
}

// KeyFunc generates a unique key from the request and idempotency key
type KeyFunc func(r *http.Request, idempotencyKey string) (string, error)

// Option is a functional option for configuring the middleware
type Option func(*Config)

// WithHeaderName sets the HTTP header name for idempotency keys
func WithHeaderName(name string) Option {
	return func(c *Config) {
		c.HeaderName = name
	}
}

// WithTTL sets the time-to-live for cached responses
func WithTTL(ttl time.Duration) Option {
	return func(c *Config) {
		c.TTL = ttl
	}
}

// WithKeyFunc sets a custom key generation function
func WithKeyFunc(fn KeyFunc) Option {
	return func(c *Config) {
		c.KeyFunc = fn
	}
}
