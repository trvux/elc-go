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

// incrWithTTLScript atomically increments a key and sets its expiry on the
// very first hit — INCR and EXPIRE as two separate round trips (the
// original version of this file) had a real gap: if the process crashed or
// the connection dropped between them, the key would be left with no TTL
// at all, and every request after that would increment a counter that
// never resets — a permanent block for that key instead of a fixed window.
// A Lua script runs as one atomic operation on the Redis server, closing
// that gap entirely (caught by /code-review before this shipped).
var incrWithTTLScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
	redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return count
`)

// RedisLimiter is a fixed-window limiter backed by Redis — same "at most
// Limit attempts per Window, per key" semantics as Limiter, but shared
// across processes. See the package doc comment for when to reach for this
// over Limiter.
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
	count, err := incrWithTTLScript.Run(ctx, l.client, []string{fullKey}, l.window.Milliseconds()).Int64()
	if err != nil {
		return true
	}
	return count <= int64(l.limit)
}
