package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisCallTimeout backstops every Redis round trip this package makes —
// Allow's signature (matching Limiter's) has no ctx param for callers, so
// this is the only deadline a call gets.
const redisCallTimeout = 2 * time.Second

// RedisLimiter is a fixed-window limiter backed by Redis INCR+EXPIRE — same
// "at most Limit attempts per Window, per key" semantics as Limiter, but
// shared across processes. See the package doc comment for when to reach
// for this over Limiter.
type RedisLimiter struct {
	client *redis.Client
	prefix string
	limit  int
	window time.Duration
}

// NewRedisLimiter builds a RedisLimiter. prefix namespaces this limiter's
// keys from any other limiter sharing the same Redis instance (e.g.
// "ai:ratelimit:chat").
func NewRedisLimiter(client *redis.Client, prefix string, limit int, window time.Duration) *RedisLimiter {
	return &RedisLimiter{client: client, prefix: prefix, limit: limit, window: window}
}

var _ RateLimiter = (*RedisLimiter)(nil)

// Allow implements RateLimiter. A Redis error fails OPEN (allows the
// request) rather than closed — rate limiting is an abuse-defense layer,
// not correctness-critical business logic; a Redis outage must not take
// the whole feature down with it.
func (l *RedisLimiter) Allow(key string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), redisCallTimeout)
	defer cancel()

	fullKey := fmt.Sprintf("%s:%s", l.prefix, key)
	count, err := l.client.Incr(ctx, fullKey).Result()
	if err != nil {
		return true
	}
	if count == 1 {
		// First hit in this window — set the expiry exactly once, not on
		// every increment, so the window doesn't keep sliding forward.
		if err := l.client.Expire(ctx, fullKey, l.window).Err(); err != nil {
			return true
		}
	}
	return count <= int64(l.limit)
}
