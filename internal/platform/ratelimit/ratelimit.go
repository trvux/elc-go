// Package ratelimit provides a small in-memory, fixed-window limiter for
// brute-force-sensitive endpoints (login, forgot-password, reset-password,
// accept-invite). In-memory is a deliberate, scope-appropriate choice: the
// service runs as a single container on a single VPS (see ARCHITECTURE.md
// §12) — no Redis/shared store exists or is needed today. If the service is
// ever horizontally scaled, this must move to a shared store.
package ratelimit

import (
	"sync"
	"time"
)

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
