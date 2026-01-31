// Package idempotency provides HTTP middleware for idempotent request handling.
// It prevents duplicate processing of requests by caching responses based on
// idempotency keys, commonly used in payment and financial APIs.
package idempotency

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// DefaultHeaderName is the default HTTP header for idempotency keys
	DefaultHeaderName = "Idempotency-Key"
	// DefaultTTL is the default time-to-live for cached responses
	DefaultTTL = 24 * time.Hour
)

// Middleware returns an HTTP middleware that enforces idempotency.
// It checks for an idempotency key in the request header, and if found,
// either returns a cached response or processes and caches the new response.
func Middleware(store Store, opts ...Option) func(http.Handler) http.Handler {
	config := &Config{
		HeaderName: DefaultHeaderName,
		TTL:        DefaultTTL,
		KeyFunc:    defaultKeyFunc,
	}

	for _, opt := range opts {
		opt(config)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only apply to non-idempotent methods
			if !isIdempotentMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Header.Get(config.HeaderName)
			if key == "" {
				// No idempotency key, process normally
				next.ServeHTTP(w, r)
				return
			}

			// Generate full key including request fingerprint
			fullKey, err := config.KeyFunc(r, key)
			if err != nil {
				http.Error(w, "Invalid idempotency key", http.StatusBadRequest)
				return
			}

			// Try to acquire lock
			unlock, err := store.Lock(fullKey)
			if err != nil {
				if err == ErrRequestInProgress {
					http.Error(w, "Request already in progress", http.StatusConflict)
					return
				}
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Check if response is cached
			cached, err := store.Get(fullKey)
			if err == nil && cached != nil {
				unlock()
				// Return cached response
				writeCachedResponse(w, cached)
				return
			}

			// Capture response
			recorder := &responseRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				body:           &bytes.Buffer{},
			}

			// Process request
			next.ServeHTTP(recorder, r)

			// Cache response
			cached = &CachedResponse{
				StatusCode: recorder.statusCode,
				Headers:    recorder.Header().Clone(),
				Body:       recorder.body.Bytes(),
				Timestamp:  time.Now(),
			}

			if err := store.Set(fullKey, cached, config.TTL); err != nil {
				// Log error but don't fail the request
				// Response has already been sent
			}

			unlock()
		})
	}
}

// isIdempotentMethod returns true for HTTP methods that should use idempotency
func isIdempotentMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPatch || method == http.MethodPut
}

// defaultKeyFunc generates a unique key combining the idempotency key and request fingerprint
func defaultKeyFunc(r *http.Request, idempotencyKey string) (string, error) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	// Create fingerprint: method + path + body hash
	h := sha256.New()
	h.Write([]byte(r.Method))
	h.Write([]byte(r.URL.Path))
	h.Write(body)
	fingerprint := fmt.Sprintf("%x", h.Sum(nil))

	return fmt.Sprintf("%s:%s", idempotencyKey, fingerprint), nil
}

// writeCachedResponse writes a cached response to the response writer
func writeCachedResponse(w http.ResponseWriter, cached *CachedResponse) {
	// Copy headers
	for key, values := range cached.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Add cache hit header
	w.Header().Set("X-Idempotency-Cached", "true")

	// Write status and body
	w.WriteHeader(cached.StatusCode)
	w.Write(cached.Body)
}

// responseRecorder captures HTTP response for caching
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}
