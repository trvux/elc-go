package domain

import (
	"context"
	"time"
)

// Cache is an optional key-value store ClassifyMessage and the
// search_products tool use to skip a repeat LLM call / DB query — see
// docs/rfc/2026-08-18-ai-chat-redis.md. Every caller that accepts a Cache
// treats a nil value as "caching disabled", never an error — same
// graceful-degrade rule as an unconfigured guardrail classifier.
type Cache interface {
	Get(ctx context.Context, key string) (value string, ok bool, err error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
}
