package server

import (
	"sync"
	"time"
)

// rateLimiter implements a simple in-memory token bucket keyed by user or IP.
type rateLimiter struct {
	mu            sync.Mutex
	buckets       map[string]*tokenBucket
	defaultLimit  int
	defaultWindow time.Duration
}

type tokenBucket struct {
	tokens     int
	lastRefill time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	if limit <= 0 {
		limit = 60
	}
	if window <= 0 {
		window = time.Minute
	}
	return &rateLimiter{
		buckets:       make(map[string]*tokenBucket),
		defaultLimit:  limit,
		defaultWindow: window,
	}
}

// Allow returns true if a request is permitted for the given key.
// limit/window override defaults when positive.
func (rl *rateLimiter) Allow(key string, limit int, window time.Duration) bool {
	if rl == nil {
		return true
	}
	if key == "" {
		key = "__global__"
	}
	if limit <= 0 {
		limit = rl.defaultLimit
	}
	if window <= 0 {
		window = rl.defaultWindow
	}

	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, ok := rl.buckets[key]
	if !ok {
		rl.buckets[key] = &tokenBucket{tokens: limit - 1, lastRefill: now}
		return true
	}

	// Refill tokens based on elapsed windows.
	elapsed := now.Sub(bucket.lastRefill)
	if elapsed >= window {
		refill := int(elapsed/window) * limit
		bucket.tokens += refill
		if bucket.tokens > limit {
			bucket.tokens = limit
		}
		bucket.lastRefill = now
	}

	if bucket.tokens <= 0 {
		return false
	}

	bucket.tokens--
	return true
}
