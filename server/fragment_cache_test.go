package server

import (
	"testing"
	"time"
)

func TestFragmentCache_BasicCaching(t *testing.T) {
	cache := newFragmentCache(false, 100) // production mode, 100 max entries

	key := "/dashboard:sidebar"
	html := "<div>Sidebar content</div>"

	// Initially, cache should be empty
	if _, ok := cache.Get(key); ok {
		t.Error("expected miss from empty cache")
	}

	// Verify miss is counted
	stats := cache.Stats()
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}

	// Store a fragment
	cache.Set(key, html, 5*time.Minute)

	// Retrieve from cache
	cached, ok := cache.Get(key)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if cached != html {
		t.Errorf("expected '%s', got '%s'", html, cached)
	}

	// Verify hit is counted
	stats = cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", stats.Hits)
	}
}

func TestFragmentCache_DevMode(t *testing.T) {
	cache := newFragmentCache(true, 100) // dev mode - caching disabled

	key := "/test:sidebar"
	html := "<div>Test</div>"

	// Set should do nothing in dev mode
	cache.Set(key, html, 5*time.Minute)

	// Get should return miss in dev mode
	if _, ok := cache.Get(key); ok {
		t.Error("expected miss in dev mode")
	}

	// Verify miss is counted even in dev mode
	stats := cache.Stats()
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}
	if !stats.DevMode {
		t.Error("expected DevMode=true in stats")
	}
}

func TestFragmentCache_Expiration(t *testing.T) {
	cache := newFragmentCache(false, 100)

	key := "/test:widget"
	html := "<div>Widget</div>"

	// Store with very short TTL
	cache.Set(key, html, 10*time.Millisecond)

	// Should hit immediately
	if _, ok := cache.Get(key); !ok {
		t.Error("expected hit before expiration")
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Should be expired
	if _, ok := cache.Get(key); ok {
		t.Error("expected miss for expired entry")
	}
}

func TestFragmentCache_DifferentKeys(t *testing.T) {
	cache := newFragmentCache(false, 100)

	key1 := "/dashboard:sidebar"
	key2 := "/dashboard:header"
	html1 := "<div>Sidebar</div>"
	html2 := "<div>Header</div>"

	// Store different fragments for different keys
	cache.Set(key1, html1, 5*time.Minute)
	cache.Set(key2, html2, 5*time.Minute)

	// Should get different fragments
	cached1, ok1 := cache.Get(key1)
	cached2, ok2 := cache.Get(key2)

	if !ok1 || cached1 != html1 {
		t.Errorf("expected '%s' for key1, got '%s'", html1, cached1)
	}
	if !ok2 || cached2 != html2 {
		t.Errorf("expected '%s' for key2, got '%s'", html2, cached2)
	}
}

func TestFragmentCache_HandlerNamespacing(t *testing.T) {
	cache := newFragmentCache(false, 100)

	// Same key name, different handler paths
	key1 := "/handlers/dashboard:sidebar"
	key2 := "/handlers/profile:sidebar"
	html1 := "<div>Dashboard Sidebar</div>"
	html2 := "<div>Profile Sidebar</div>"

	cache.Set(key1, html1, 5*time.Minute)
	cache.Set(key2, html2, 5*time.Minute)

	// Keys are different, so should get different fragments
	cached1, _ := cache.Get(key1)
	cached2, _ := cache.Get(key2)

	if cached1 != html1 {
		t.Errorf("expected dashboard sidebar, got '%s'", cached1)
	}
	if cached2 != html2 {
		t.Errorf("expected profile sidebar, got '%s'", cached2)
	}
}

func TestFragmentCache_Invalidate(t *testing.T) {
	cache := newFragmentCache(false, 100)

	key := "/test:widget"
	html := "<div>Widget</div>"

	cache.Set(key, html, 5*time.Minute)

	// Verify it's cached
	if _, ok := cache.Get(key); !ok {
		t.Fatal("expected hit before invalidate")
	}

	// Invalidate
	cache.Invalidate(key)

	// Should miss now
	if _, ok := cache.Get(key); ok {
		t.Error("expected miss after invalidate")
	}
}

func TestFragmentCache_InvalidatePrefix(t *testing.T) {
	cache := newFragmentCache(false, 100)

	// Multiple keys from same handler
	cache.Set("/dashboard:sidebar", "sidebar", 5*time.Minute)
	cache.Set("/dashboard:header", "header", 5*time.Minute)
	cache.Set("/dashboard:footer", "footer", 5*time.Minute)
	cache.Set("/profile:sidebar", "profile sidebar", 5*time.Minute)

	// Invalidate all /dashboard entries
	cache.InvalidatePrefix("/dashboard:")

	// Dashboard entries should be gone
	if _, ok := cache.Get("/dashboard:sidebar"); ok {
		t.Error("expected miss for dashboard:sidebar after prefix invalidate")
	}
	if _, ok := cache.Get("/dashboard:header"); ok {
		t.Error("expected miss for dashboard:header after prefix invalidate")
	}
	if _, ok := cache.Get("/dashboard:footer"); ok {
		t.Error("expected miss for dashboard:footer after prefix invalidate")
	}

	// Profile entry should still be there
	if _, ok := cache.Get("/profile:sidebar"); !ok {
		t.Error("expected hit for profile:sidebar after prefix invalidate of different path")
	}
}

func TestFragmentCache_Clear(t *testing.T) {
	cache := newFragmentCache(false, 100)

	cache.Set("/a:x", "a", 5*time.Minute)
	cache.Set("/b:y", "b", 5*time.Minute)
	cache.Set("/c:z", "c", 5*time.Minute)

	// Verify all cached
	if _, ok := cache.Get("/a:x"); !ok {
		t.Error("expected hit before clear")
	}

	// Clear all
	cache.Clear()

	// All should miss
	if _, ok := cache.Get("/a:x"); ok {
		t.Error("expected miss after clear")
	}
	if _, ok := cache.Get("/b:y"); ok {
		t.Error("expected miss after clear")
	}
	if _, ok := cache.Get("/c:z"); ok {
		t.Error("expected miss after clear")
	}
}

func TestFragmentCache_Stats(t *testing.T) {
	cache := newFragmentCache(false, 100)

	cache.Set("/a:x", "<div>small</div>", 5*time.Minute)
	cache.Set("/b:y", "<div>bigger content here</div>", 5*time.Minute)

	// One hit, one miss
	cache.Get("/a:x") // hit
	cache.Get("/c:z") // miss

	stats := cache.Stats()

	if stats.Entries != 2 {
		t.Errorf("expected 2 entries, got %d", stats.Entries)
	}
	if stats.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}
	if stats.SizeBytes <= 0 {
		t.Error("expected positive size")
	}
	if stats.DevMode {
		t.Error("expected DevMode=false")
	}
}

func TestFragmentCache_HitRate(t *testing.T) {
	stats := FragmentCacheStats{Hits: 75, Misses: 25}
	rate := stats.HitRate()
	if rate != 75.0 {
		t.Errorf("expected 75.0%% hit rate, got %v%%", rate)
	}

	// Zero total
	emptyStats := FragmentCacheStats{Hits: 0, Misses: 0}
	if emptyStats.HitRate() != 0 {
		t.Error("expected 0%% hit rate for empty stats")
	}
}

func TestFragmentCache_ZeroTTL(t *testing.T) {
	cache := newFragmentCache(false, 100)

	// Zero TTL should not cache
	cache.Set("/test:x", "content", 0)

	if _, ok := cache.Get("/test:x"); ok {
		t.Error("expected miss for zero TTL entry")
	}
}

func TestFragmentCache_NegativeTTL(t *testing.T) {
	cache := newFragmentCache(false, 100)

	// Negative TTL should not cache
	cache.Set("/test:x", "content", -5*time.Minute)

	if _, ok := cache.Get("/test:x"); ok {
		t.Error("expected miss for negative TTL entry")
	}
}

func TestFragmentCache_LRUEviction(t *testing.T) {
	// Small cache to force eviction
	cache := newFragmentCache(false, 10)

	// Fill the cache
	for i := range 10 {
		key := "/test:" + string(rune('a'+i))
		cache.Set(key, "content", 5*time.Minute)
	}

	// Verify all cached
	stats := cache.Stats()
	if stats.Entries != 10 {
		t.Errorf("expected 10 entries, got %d", stats.Entries)
	}

	// Add one more to trigger eviction
	cache.Set("/test:overflow", "overflow content", 5*time.Minute)

	// Some entries should have been evicted
	stats = cache.Stats()
	if stats.Entries > 10 {
		t.Errorf("expected <=10 entries after eviction, got %d", stats.Entries)
	}
}

func TestFragmentCache_EmptyContent(t *testing.T) {
	cache := newFragmentCache(false, 100)

	// Empty content is valid
	cache.Set("/test:empty", "", 5*time.Minute)

	cached, ok := cache.Get("/test:empty")
	if !ok {
		t.Error("expected hit for empty content")
	}
	if cached != "" {
		t.Errorf("expected empty string, got '%s'", cached)
	}
}

func TestFragmentCache_Disabled(t *testing.T) {
	cache := newFragmentCache(false, 100)
	cache.SetDisabled(true)

	cache.Set("/test:x", "content", 5*time.Minute)

	if _, ok := cache.Get("/test:x"); ok {
		t.Error("expected miss when cache is disabled")
	}

	// Re-enable
	cache.SetDisabled(false)

	cache.Set("/test:x", "content", 5*time.Minute)

	if _, ok := cache.Get("/test:x"); !ok {
		t.Error("expected hit when cache is re-enabled")
	}
}
