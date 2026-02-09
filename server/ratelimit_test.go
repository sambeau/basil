package server

import (
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := newRateLimiter(3, time.Second)

	// Should allow first 3 requests
	for i := range 3 {
		if !rl.Allow("user1", 0, 0) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th request should be blocked
	if rl.Allow("user1", 0, 0) {
		t.Error("4th request should be blocked")
	}

	// Wait for refill
	time.Sleep(1100 * time.Millisecond)

	// Should allow again after refill window
	if !rl.Allow("user1", 0, 0) {
		t.Error("request after refill should be allowed")
	}
}

func TestRateLimiter_MultipleKeys(t *testing.T) {
	rl := newRateLimiter(2, time.Second)

	// user1 uses their quota
	rl.Allow("user1", 0, 0)
	rl.Allow("user1", 0, 0)

	// user1 should be blocked
	if rl.Allow("user1", 0, 0) {
		t.Error("user1 3rd request should be blocked")
	}

	// user2 should still have quota (independent bucket)
	if !rl.Allow("user2", 0, 0) {
		t.Error("user2 should have independent quota")
	}
	if !rl.Allow("user2", 0, 0) {
		t.Error("user2 2nd request should be allowed")
	}
}

func TestRateLimiter_CustomLimits(t *testing.T) {
	rl := newRateLimiter(10, time.Minute)

	// Override with stricter limit
	if !rl.Allow("user1", 1, time.Second) {
		t.Error("first request should be allowed")
	}

	// Second request should be blocked (limit=1)
	if rl.Allow("user1", 1, time.Second) {
		t.Error("second request should be blocked with limit=1")
	}
}

func TestRateLimiter_NilRateLimiter(t *testing.T) {
	var rl *rateLimiter = nil

	// Nil rate limiter should always allow
	if !rl.Allow("user1", 0, 0) {
		t.Error("nil rate limiter should allow all requests")
	}
}

func TestRateLimiter_EmptyKey(t *testing.T) {
	rl := newRateLimiter(2, time.Second)

	// Empty key should use global bucket
	if !rl.Allow("", 0, 0) {
		t.Error("first request with empty key should be allowed")
	}
	if !rl.Allow("", 0, 0) {
		t.Error("second request with empty key should be allowed")
	}
	if rl.Allow("", 0, 0) {
		t.Error("third request with empty key should be blocked")
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := newRateLimiter(100, time.Second)

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Spawn 10 goroutines making 10 requests each
	for i := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for range 10 {
				rl.Allow("concurrent_user", 0, 0)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	// Check for errors (shouldn't be any)
	for err := range errors {
		t.Errorf("concurrent access error: %v", err)
	}

	// Should have consumed 100 tokens, next should be blocked
	if rl.Allow("concurrent_user", 0, 0) {
		t.Error("101st request should be blocked after concurrent usage")
	}
}

func TestRateLimiter_TokenRefill(t *testing.T) {
	rl := newRateLimiter(2, 500*time.Millisecond)

	// Use up tokens
	rl.Allow("user1", 0, 0)
	rl.Allow("user1", 0, 0)

	// Should be blocked
	if rl.Allow("user1", 0, 0) {
		t.Error("should be blocked after quota exhausted")
	}

	// Wait for one refill window
	time.Sleep(600 * time.Millisecond)

	// Should have 2 more tokens
	if !rl.Allow("user1", 0, 0) {
		t.Error("should allow after one refill window")
	}
	if !rl.Allow("user1", 0, 0) {
		t.Error("should allow second request after refill")
	}

	// Should be blocked again
	if rl.Allow("user1", 0, 0) {
		t.Error("should be blocked after using refilled tokens")
	}
}

func TestRateLimiter_InvalidParameters(t *testing.T) {
	// Test with zero limit (should default to 60)
	rl := newRateLimiter(0, time.Minute)
	if rl.defaultLimit != 60 {
		t.Errorf("expected default limit 60, got %d", rl.defaultLimit)
	}

	// Test with zero window (should default to 1 minute)
	rl = newRateLimiter(10, 0)
	if rl.defaultWindow != time.Minute {
		t.Errorf("expected default window 1m, got %v", rl.defaultWindow)
	}
}
