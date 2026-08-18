package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/trvux/elc-go/internal/ai/domain"
)

// RedisCache implements domain.Cache over a Redis client — see
// docs/rfc/2026-08-18-ai-chat-redis.md.
type RedisCache struct {
	client *redis.Client
	prefix string
}

// NewRedisCache builds a RedisCache. prefix namespaces this cache's keys
// from anything else sharing the same Redis instance (e.g. "ai:cache").
func NewRedisCache(client *redis.Client, prefix string) *RedisCache {
	return &RedisCache{client: client, prefix: prefix}
}

var _ domain.Cache = (*RedisCache)(nil)

func (c *RedisCache) Get(ctx context.Context, key string) (string, bool, error) {
	value, err := c.client.Get(ctx, c.fullKey(key)).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("ai: redis cache get: %w", err)
	}
	return value, true, nil
}

func (c *RedisCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := c.client.Set(ctx, c.fullKey(key), value, ttl).Err(); err != nil {
		return fmt.Errorf("ai: redis cache set: %w", err)
	}
	return nil
}

func (c *RedisCache) fullKey(key string) string {
	return c.prefix + ":" + key
}
