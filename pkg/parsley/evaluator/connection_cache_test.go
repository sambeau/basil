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
	)
	defer cache.close()

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent puts
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			cache.put("key", val)
		}(i)
	}

	// Concurrent gets
	for i := 0; i < iterations; i++ {
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

// TestSFTPCacheIntegration tests the actual SFTP cache
func TestSFTPCacheIntegration(t *testing.T) {
	// This test verifies the real sftpCache works
	if sftpCache == nil {
		t.Fatal("sftpCache should be initialized")
	}

	// Note: We can't easily test the SFTP cache with a mock connection
	// because the health check (Getwd) requires a real SSH client.
	// The cache itself is tested in the generic tests above.
	// Here we just verify it's initialized properly.

	initialSize := sftpCache.size()
	if initialSize < 0 {
		t.Fatal("cache size should be non-negative")
	}
}
