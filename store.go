package idempotency

import (
	"net/http"
	"time"
)

// Store defines the interface for storing and retrieving cached responses
type Store interface {
	// Get retrieves a cached response by key
	Get(key string) (*CachedResponse, error)

	// Set stores a response with the given key and TTL
	Set(key string, response *CachedResponse, ttl time.Duration) error

	// Lock acquires a lock for the given key to prevent concurrent processing
	// Returns an unlock function that must be called to release the lock
	Lock(key string) (unlock func(), err error)
}

// CachedResponse represents a cached HTTP response
type CachedResponse struct {
	StatusCode int         `json:"status_code"`
	Headers    http.Header `json:"headers"`
	Body       []byte      `json:"body"`
	Timestamp  time.Time   `json:"timestamp"`
}
