package idempotency

import "errors"

var (
	// ErrRequestInProgress is returned when a request with the same key is already being processed
	ErrRequestInProgress = errors.New("request with this idempotency key is already in progress")

	// ErrNotFound is returned when a cached response is not found
	ErrNotFound = errors.New("cached response not found")

	// ErrLockFailed is returned when acquiring a lock fails
	ErrLockFailed = errors.New("failed to acquire lock")
)
