// Package ratelimit provides fixed-window limiters for abuse-sensitive
// endpoints. Limiter (in-memory) is the default for brute-force-sensitive
// admin endpoints (login, forgot-password, reset-password, accept-invite)
// and remains scope-appropriate for them: the service runs as a single
// container on a single VPS (see ARCHITECTURE.md §12).
//
// RedisLimiter (see redis_limiter.go) exists for internal/ai's public
// /ai/chat specifically, where a shared counter matters more (see
// docs/rfc/2026-08-18-ai-chat-redis.md) — it's opt-in, not a replacement:
// every other caller in this codebase keeps using in-memory Limiter
// unchanged. Both satisfy the RateLimiter interface below.
package ratelimit

import (
	"sync"
	"time"
)

// RateLimiter is the shared interface Limiter and RedisLimiter both
// satisfy — callers that want to accept either (see internal/ai's
// AIHandler) depend on this instead of the concrete *Limiter type.
type RateLimiter interface {
	// Allow reports whether the caller identified by key may proceed, and
	// increments its counter as a side effect if so.
	Allow(key string) bool
}

var _ RateLimiter = (*Limiter)(nil)

type window struct {
	count     int
	expiresAt time.Time
}

// Limiter enforces "at most Limit attempts per Window, per key".
type Limiter struct {
	mu      sync.Mutex
	windows map[string]*window
	limit   int
	window  time.Duration
}

func New(limit int, per time.Duration) *Limiter {
	l := &Limiter{
		windows: make(map[string]*window),
		limit:   limit,
		window:  per,
	}
	go l.evictExpiredLoop()
	return l
}

// evictExpiredLoop periodically drops expired windows so the map doesn't
// grow without bound over the process's lifetime (every distinct
// identifier|IP ever seen would otherwise stay in memory forever). Runs
// for the lifetime of the process — Limiter is a long-lived singleton with
// no Close/Stop, same as the rest of this package's design (see the
// package doc comment).
func (l *Limiter) evictExpiredLoop() {
	ticker := time.NewTicker(l.window)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		l.mu.Lock()
		for key, w := range l.windows {
			if now.After(w.expiresAt) {
				delete(l.windows, key)
			}
		}
		l.mu.Unlock()
	}
}

// Allow reports whether the caller identified by key may proceed, and
// increments its counter as a side effect if so.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	w, ok := l.windows[key]
	if !ok || now.After(w.expiresAt) {
		l.windows[key] = &window{count: 1, expiresAt: now.Add(l.window)}
		return true
	}

	if w.count >= l.limit {
		return false
	}
	w.count++
	return true
}
