package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yourusername/go-idempotency"
)

// RedisStore is a Redis-backed implementation of Store
type RedisStore struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisStore creates a new Redis store
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: client,
		ctx:    context.Background(),
	}
}

// Get retrieves a cached response from Redis
func (s *RedisStore) Get(key string) (*idempotency.CachedResponse, error) {
	data, err := s.client.Get(s.ctx, key).Bytes()
	if err == redis.Nil {
		return nil, idempotency.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var response idempotency.CachedResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Set stores a response in Redis with TTL
func (s *RedisStore) Set(key string, response *idempotency.CachedResponse, ttl time.Duration) error {
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}

	return s.client.Set(s.ctx, key, data, ttl).Err()
}

// Lock acquires a distributed lock using Redis
func (s *RedisStore) Lock(key string) (func(), error) {
	lockKey := "lock:" + key
	acquired, err := s.client.SetNX(s.ctx, lockKey, "1", 30*time.Second).Result()
	if err != nil {
		return nil, err
	}

	if !acquired {
		return nil, idempotency.ErrRequestInProgress
	}

	unlock := func() {
		s.client.Del(s.ctx, lockKey)
	}

	return unlock, nil
}
