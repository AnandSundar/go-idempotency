package idempotency

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AnandSundar/go-idempotency/store"
	"github.com/stretchr/testify/assert"
)

func TestMiddleware_CachesResponse(t *testing.T) {
	s := store.NewMemoryStore()
	handler := Middleware(s)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))

	// First request
	req1 := httptest.NewRequest(http.MethodPost, "/api/payment", bytes.NewBufferString(`{"amount":100}`))
	req1.Header.Set("Idempotency-Key", "test-123")
	rec1 := httptest.NewRecorder()

	handler.ServeHTTP(rec1, req1)

	assert.Equal(t, http.StatusOK, rec1.Code)
	assert.Equal(t, `{"success":true}`, rec1.Body.String())
	assert.Empty(t, rec1.Header().Get("X-Idempotency-Cached"))

	// Second request with same key
	req2 := httptest.NewRequest(http.MethodPost, "/api/payment", bytes.NewBufferString(`{"amount":100}`))
	req2.Header.Set("Idempotency-Key", "test-123")
	rec2 := httptest.NewRecorder()

	handler.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusOK, rec2.Code)
	assert.Equal(t, `{"success":true}`, rec2.Body.String())
	assert.Equal(t, "true", rec2.Header().Get("X-Idempotency-Cached"))
}

func TestMiddleware_DifferentBodyGivesDifferentKey(t *testing.T) {
	s := store.NewMemoryStore()
	callCount := 0
	handler := Middleware(s)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))

	// First request
	req1 := httptest.NewRequest(http.MethodPost, "/api/payment", bytes.NewBufferString(`{"amount":100}`))
	req1.Header.Set("Idempotency-Key", "test-123")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	// Second request with same key but different body
	req2 := httptest.NewRequest(http.MethodPost, "/api/payment", bytes.NewBufferString(`{"amount":200}`))
	req2.Header.Set("Idempotency-Key", "test-123")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	// Both should process (different fingerprints)
	assert.Equal(t, 2, callCount)
}

func TestMiddleware_NoKeyPassesThrough(t *testing.T) {
	s := store.NewMemoryStore()
	handler := Middleware(s)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/payment", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMiddleware_WithCustomTTL(t *testing.T) {
	s := store.NewMemoryStore()
	handler := Middleware(s, WithTTL(100*time.Millisecond))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/payment", bytes.NewBufferString(`{"amount":100}`))
	req.Header.Set("Idempotency-Key", "test-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should process again (expired)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req)
	assert.Empty(t, rec2.Header().Get("X-Idempotency-Cached"))
}
