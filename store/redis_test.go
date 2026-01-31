package store

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/AnandSundar/go-idempotency"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedis creates a mock Redis server for testing
func setupTestRedis(t *testing.T) (*RedisStore, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	store := NewRedisStore(client)

	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})

	return store, mr
}

func TestRedisStore_SetAndGet(t *testing.T) {
	store, _ := setupTestRedis(t)

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
	assert.Equal(t, "application/json", cached.Headers.Get("Content-Type"))
}

func TestRedisStore_GetNotFound(t *testing.T) {
	store, _ := setupTestRedis(t)

	_, err := store.Get("nonexistent")
	assert.ErrorIs(t, err, idempotency.ErrNotFound)
}

func TestRedisStore_Expiration(t *testing.T) {
	store, mr := setupTestRedis(t)

	response := &idempotency.CachedResponse{
		StatusCode: 200,
		Body:       []byte(`{"success":true}`),
		Timestamp:  time.Now(),
	}

	err := store.Set("test-key", response, 100*time.Millisecond)
	require.NoError(t, err)

	// Fast-forward time in miniredis
	mr.FastForward(150 * time.Millisecond)

	_, err = store.Get("test-key")
	assert.ErrorIs(t, err, idempotency.ErrNotFound)
}

func TestRedisStore_Lock(t *testing.T) {
	store, _ := setupTestRedis(t)

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

func TestRedisStore_LockAutoExpires(t *testing.T) {
	store, mr := setupTestRedis(t)

	unlock, err := store.Lock("test-key")
	require.NoError(t, err)
	defer unlock()

	// Lock should auto-expire after 30 seconds
	mr.FastForward(31 * time.Second)

	// Should be able to acquire lock again
	unlock2, err := store.Lock("test-key")
	require.NoError(t, err)
	unlock2()
}

func TestRedisStore_MultipleKeys(t *testing.T) {
	store, _ := setupTestRedis(t)

	response1 := &idempotency.CachedResponse{
		StatusCode: 200,
		Body:       []byte(`{"id":1}`),
		Timestamp:  time.Now(),
	}

	response2 := &idempotency.CachedResponse{
		StatusCode: 201,
		Body:       []byte(`{"id":2}`),
		Timestamp:  time.Now(),
	}

	err := store.Set("key1", response1, 1*time.Hour)
	require.NoError(t, err)

	err = store.Set("key2", response2, 1*time.Hour)
	require.NoError(t, err)

	cached1, err := store.Get("key1")
	require.NoError(t, err)
	assert.Equal(t, 200, cached1.StatusCode)

	cached2, err := store.Get("key2")
	require.NoError(t, err)
	assert.Equal(t, 201, cached2.StatusCode)
}

func TestRedisStore_ConcurrentLocks(t *testing.T) {
	store, _ := setupTestRedis(t)

	const numGoroutines = 10
	successCount := 0
	done := make(chan bool, numGoroutines)

	// Try to acquire lock from multiple goroutines
	for i := 0; i < numGoroutines; i++ {
		go func() {
			unlock, err := store.Lock("concurrent-test")
			if err == nil {
				successCount++
				time.Sleep(10 * time.Millisecond)
				unlock()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Only one should have succeeded initially
	// (others may succeed after the first unlocks)
	assert.Greater(t, successCount, 0)
}

func TestRedisStore_LargeResponse(t *testing.T) {
	store, _ := setupTestRedis(t)

	// Create a large response body
	largeBody := make([]byte, 1024*1024) // 1MB
	for i := range largeBody {
		largeBody[i] = byte(i % 256)
	}

	response := &idempotency.CachedResponse{
		StatusCode: 200,
		Body:       largeBody,
		Timestamp:  time.Now(),
	}

	err := store.Set("large-key", response, 1*time.Hour)
	require.NoError(t, err)

	cached, err := store.Get("large-key")
	require.NoError(t, err)
	assert.Equal(t, len(largeBody), len(cached.Body))
	assert.Equal(t, largeBody, cached.Body)
}

// TestRedisStore_RealRedis tests against a real Redis instance
// Skip this test if Redis is not available
func TestRedisStore_RealRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real Redis test in short mode")
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Ping to check if Redis is available
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available:", err)
	}

	store := NewRedisStore(client)

	response := &idempotency.CachedResponse{
		StatusCode: 200,
		Body:       []byte(`{"success":true}`),
		Timestamp:  time.Now(),
	}

	testKey := "test:real-redis:" + time.Now().Format("20060102150405")

	err := store.Set(testKey, response, 10*time.Second)
	require.NoError(t, err)

	cached, err := store.Get(testKey)
	require.NoError(t, err)
	assert.Equal(t, response.StatusCode, cached.StatusCode)

	// Cleanup
	client.Del(ctx, testKey)
	client.Close()
}
