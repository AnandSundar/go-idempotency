package store

import (
	"sync"
	"time"

	"github.com/AnandSundar/go-idempotency"
)

// MemoryStore is an in-memory implementation of Store
type MemoryStore struct {
	mu      sync.RWMutex
	data    map[string]*entry
	locks   map[string]*sync.Mutex
	locksMu sync.Mutex
}

type entry struct {
	response  *idempotency.CachedResponse
	expiresAt time.Time
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	s := &MemoryStore{
		data:  make(map[string]*entry),
		locks: make(map[string]*sync.Mutex),
	}

	// Start cleanup goroutine
	go s.cleanup()

	return s
}

// Get retrieves a cached response
func (s *MemoryStore) Get(key string) (*idempotency.CachedResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.data[key]
	if !exists {
		return nil, idempotency.ErrNotFound
	}

	if time.Now().After(entry.expiresAt) {
		return nil, idempotency.ErrNotFound
	}

	return entry.response, nil
}

// Set stores a response with TTL
func (s *MemoryStore) Set(key string, response *idempotency.CachedResponse, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = &entry{
		response:  response,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Lock acquires a lock for the given key
func (s *MemoryStore) Lock(key string) (func(), error) {
	s.locksMu.Lock()
	mu, exists := s.locks[key]
	if !exists {
		mu = &sync.Mutex{}
		s.locks[key] = mu
	}
	s.locksMu.Unlock()

	// Try to acquire lock with timeout
	locked := make(chan struct{})
	go func() {
		mu.Lock()
		close(locked)
	}()

	select {
	case <-locked:
		return func() { mu.Unlock() }, nil
	case <-time.After(100 * time.Millisecond):
		return nil, idempotency.ErrRequestInProgress
	}
}

// cleanup periodically removes expired entries
func (s *MemoryStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, entry := range s.data {
			if now.After(entry.expiresAt) {
				delete(s.data, key)
			}
		}
		s.mu.Unlock()
	}
}
