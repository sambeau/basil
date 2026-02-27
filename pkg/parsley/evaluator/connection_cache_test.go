package evaluator

import (
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"
)

// TestConnectionCacheBasic tests basic cache operations
func TestConnectionCacheBasic(t *testing.T) {
	cache := newConnectionCache[string](
		10,
		1*time.Minute,
		nil, // no health check
		func(s string) error { return nil },
		nil, // no logger
	)
	defer cache.close()

	// Put and get
	cache.put("key1", "value1")
	val, found := cache.get("key1")
	if !found {
		t.Fatal("expected to find key1 in cache")
	}
	if val != "value1" {
		t.Fatalf("expected value1, got %s", val)
	}

	// Get non-existent key
	_, found = cache.get("key2")
	if found {
		t.Fatal("expected not to find key2 in cache")
	}
}

// TestConnectionCacheTTL tests TTL expiration
func TestConnectionCacheTTL(t *testing.T) {
	cache := newConnectionCache[string](
		10,
		100*time.Millisecond, // very short TTL for testing
		nil,
		func(s string) error { return nil },
		nil,
	)
	defer cache.close()

	cache.put("key1", "value1")

	// Should be found immediately
	_, found := cache.get("key1")
	if !found {
		t.Fatal("expected to find key1 immediately")
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Should not be found after expiration
	_, found = cache.get("key1")
	if found {
		t.Fatal("expected key1 to be expired")
	}
}

// TestConnectionCacheTTLResetOnUse verifies that accessing a cached entry resets its TTL.
// A connection under active use should not be evicted — only idle connections should expire.
func TestConnectionCacheTTLResetOnUse(t *testing.T) {
	cache := newConnectionCache[string](
		10,
		150*time.Millisecond, // short TTL for testing
		nil,
		func(s string) error { return nil },
		nil,
	)
	defer cache.close()

	cache.put("key1", "value1")

	// Access repeatedly at sub-TTL intervals to keep lastUsed fresh.
	// Each access resets the TTL clock; the entry should survive.
	for i := 0; i < 3; i++ {
		time.Sleep(75 * time.Millisecond) // less than TTL each iteration
		val, found := cache.get("key1")
		if !found {
			t.Fatalf("iteration %d: expected key1 to still be cached (TTL should reset on use)", i)
		}
		if val != "value1" {
			t.Fatalf("iteration %d: expected value1, got %s", i, val)
		}
	}

	// Stop accessing and wait for one full TTL to elapse since last use
	time.Sleep(200 * time.Millisecond)
	_, found := cache.get("key1")
	if found {
		t.Fatal("expected key1 to be evicted after TTL elapsed since last use")
	}
}

// TestConnectionCacheHealthCheck tests health check functionality
func TestConnectionCacheHealthCheck(t *testing.T) {
	healthCheckFails := false
	cache := newConnectionCache[string](
		10,
		1*time.Minute,
		func(s string) error {
			if healthCheckFails {
				return errors.New("health check failed")
			}
			return nil
		},
		func(s string) error { return nil },
		nil,
	)
	defer cache.close()

	cache.put("key1", "value1")

	// Health check passes
	val, found := cache.get("key1")
	if !found {
		t.Fatal("expected to find key1 when health check passes")
	}
	if val != "value1" {
		t.Fatalf("expected value1, got %s", val)
	}

	// Make health check fail
	healthCheckFails = true

	// Should not be found after health check fails
	_, found = cache.get("key1")
	if found {
		t.Fatal("expected key1 to be removed after health check failure")
	}
}

// TestConnectionCacheMaxSize tests LRU eviction
func TestConnectionCacheMaxSize(t *testing.T) {
	cache := newConnectionCache[int](
		3, // small cache for testing
		1*time.Minute,
		nil,
		func(i int) error { return nil },
		nil,
	)
	defer cache.close()

	// Fill cache to capacity
	cache.put("key1", 1)
	cache.put("key2", 2)
	cache.put("key3", 3)

	if cache.size() != 3 {
		t.Fatalf("expected cache size 3, got %d", cache.size())
	}

	// Access key1 to make it more recently used
	cache.get("key1")

	// Add key4, should evict key2 (least recently used)
	cache.put("key4", 4)

	if cache.size() != 3 {
		t.Fatalf("expected cache size 3 after eviction, got %d", cache.size())
	}

	// key2 should be evicted
	_, found := cache.get("key2")
	if found {
		t.Fatal("expected key2 to be evicted")
	}

	// key1, key3, key4 should still be present
	_, found = cache.get("key1")
	if !found {
		t.Fatal("expected key1 to still be in cache")
	}
	_, found = cache.get("key3")
	if !found {
		t.Fatal("expected key3 to still be in cache")
	}
	_, found = cache.get("key4")
	if !found {
		t.Fatal("expected key4 to still be in cache")
	}
}

// TestConnectionCacheConcurrent tests concurrent access
func TestConnectionCacheConcurrent(t *testing.T) {
	cache := newConnectionCache[int](
		100,
		1*time.Minute,
		nil,
		func(i int) error { return nil },
		nil,
	)
	defer cache.close()

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent puts
	for i := range iterations {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			cache.put("key", val)
		}(i)
	}

	// Concurrent gets
	for range iterations {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.get("key")
		}()
	}

	wg.Wait()

	// Should not crash and cache should still work
	cache.put("final", 999)
	val, found := cache.get("final")
	if !found || val != 999 {
		t.Fatal("cache corrupted after concurrent access")
	}
}

// TestConnectionCacheCleanup tests automatic cleanup
func TestConnectionCacheCleanup(t *testing.T) {
	cache := newConnectionCache[string](
		10,
		100*time.Millisecond, // short TTL
		nil,
		func(s string) error { return nil },
		nil,
	)
	cache.cleanupTick = 50 * time.Millisecond // fast cleanup for testing
	defer cache.close()

	// Add items
	cache.put("key1", "value1")
	cache.put("key2", "value2")

	if cache.size() != 2 {
		t.Fatalf("expected cache size 2, got %d", cache.size())
	}

	// Wait for TTL to expire and cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Cleanup should have removed expired items
	if cache.size() != 0 {
		t.Fatalf("expected cache size 0 after cleanup, got %d", cache.size())
	}
}

// TestConnectionCacheClose tests cache shutdown
func TestConnectionCacheClose(t *testing.T) {
	closeCount := 0
	cache := newConnectionCache[string](
		10,
		1*time.Minute,
		nil,
		func(s string) error {
			closeCount++
			return nil
		},
		nil,
	)

	cache.put("key1", "value1")
	cache.put("key2", "value2")
	cache.put("key3", "value3")

	err := cache.close()
	if err != nil {
		t.Fatalf("unexpected error from close: %v", err)
	}

	// All connections should have been closed
	if closeCount != 3 {
		t.Fatalf("expected 3 closes, got %d", closeCount)
	}

	// Cache should be empty
	if cache.size() != 0 {
		t.Fatalf("expected cache size 0 after close, got %d", cache.size())
	}
}

// TestDBCacheIntegration tests the actual DB cache
func TestDBCacheIntegration(t *testing.T) {
	// This test verifies the real dbCache works
	// We don't want to interfere with other tests, so we'll just check the cache exists
	if dbCache == nil {
		t.Fatal("dbCache should be initialized")
	}

	// Create a memory database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	// Put in cache
	dbCache.put("test:memory:", db)

	// Get from cache
	cachedDB, found := dbCache.get("test:memory:")
	if !found {
		t.Fatal("expected to find db in cache")
	}

	// Should be the same database (health check should pass)
	if err := cachedDB.Ping(); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

// TestDBCacheNoHealthCheck verifies that dbCache is configured without a health check.
// db.Ping() on every retrieval is a synchronous network round-trip that database/sql
// makes unnecessary via its internal retry logic. This test is a regression guard to
// ensure the health check is not accidentally re-added.
func TestDBCacheNoHealthCheck(t *testing.T) {
	if dbCache == nil {
		t.Fatal("dbCache should be initialised")
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	key := "test-no-healthcheck:" + t.Name()
	dbCache.put(key, db)
	defer func() {
		dbCache.mu.Lock()
		delete(dbCache.conns, key)
		dbCache.mu.Unlock()
	}()

	cached, found := dbCache.get(key)
	if !found {
		t.Fatal("expected to find db in cache")
	}
	if cached != db {
		t.Fatal("expected to get back the same *sql.DB")
	}

	// Verify the health check is nil — if it were db.Ping(), it would still
	// succeed here (memory db is healthy), so we check the field directly.
	if dbCache.healthCheck != nil {
		t.Error("dbCache must have nil health check — db.Ping() on every retrieval adds a network round-trip per request")
	}
}

// TestSFTPCacheIntegration tests the actual SFTP cache
func TestSFTPCacheIntegration(t *testing.T) {
	// This test verifies the real sftpCache works
	if sftpCache == nil {
		t.Fatal("sftpCache should be initialized")
	}

	// Note: Full SFTP cache integration tests (including health-check eviction)
	// are in eval_sftp_integration_test.go using a real fake SSH/SFTP server
	// from testenv. Here we just verify the cache is initialized properly.

	initialSize := sftpCache.size()
	if initialSize < 0 {
		t.Fatal("cache size should be non-negative")
	}
}
