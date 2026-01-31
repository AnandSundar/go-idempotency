package store

import (
	"net/http"
	"testing"
	"time"

	"github.com/AnandSundar/go-idempotency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStore_SetAndGet(t *testing.T) {
	store := NewMemoryStore()

	response := &idempotency.CachedResponse{
		StatusCode: 200,
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       []byte(`{"success":true}`),
		Timestamp:  time.Now(),
	}

	err := store.Set("test-key", response, 1*time.Hour)
	require.NoError(t, err)

	cached, err := store.Get("test-key")
	require.NoError(t, err)
	assert.Equal(t, response.StatusCode, cached.StatusCode)
	assert.Equal(t, response.Body, cached.Body)
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.Get("nonexistent")
	assert.ErrorIs(t, err, idempotency.ErrNotFound)
}

func TestMemoryStore_Expiration(t *testing.T) {
	store := NewMemoryStore()

	response := &idempotency.CachedResponse{
		StatusCode: 200,
		Body:       []byte(`{"success":true}`),
	}

	err := store.Set("test-key", response, 100*time.Millisecond)
	require.NoError(t, err)

	time.Sleep(150 * time.Millisecond)

	_, err = store.Get("test-key")
	assert.ErrorIs(t, err, idempotency.ErrNotFound)
}

func TestMemoryStore_Lock(t *testing.T) {
	store := NewMemoryStore()

	unlock1, err := store.Lock("test-key")
	require.NoError(t, err)

	// Second lock should fail
	_, err = store.Lock("test-key")
	assert.ErrorIs(t, err, idempotency.ErrRequestInProgress)

	unlock1()

	// After unlock, should succeed
	unlock2, err := store.Lock("test-key")
	require.NoError(t, err)
	unlock2()
}
