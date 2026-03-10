package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Response Cache Benchmarks

func BenchmarkResponseCache_GetHit(b *testing.B) {
	b.ReportAllocs()

	cache := newResponseCache(false, true) // production mode, cache enabled

	// Create a representative request
	req := httptest.NewRequest(http.MethodGet, "/api/users?page=1&limit=20", http.NoBody)

	// Pre-warm the cache
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("X-Request-Id", "bench-123")
	body := []byte(`{"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}],"total":42}`)
	cache.Set(req, 5*time.Minute, http.StatusOK, headers, body)

	b.ResetTimer()
	for b.Loop() {
		entry := cache.Get(req)
		if entry == nil {
			b.Fatal("expected cache hit")
		}
	}
}

func BenchmarkResponseCache_GetMiss(b *testing.B) {
	b.ReportAllocs()

	cache := newResponseCache(false, true)

	// Request that won't be in cache
	req := httptest.NewRequest(http.MethodGet, "/api/missing", http.NoBody)

	b.ResetTimer()
	for b.Loop() {
		_ = cache.Get(req)
	}
}

func BenchmarkResponseCache_SetThenGet(b *testing.B) {
	b.ReportAllocs()

	cache := newResponseCache(false, true)

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	body := []byte(`{"status":"ok","data":{"id":123,"name":"Test"}}`)

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		// Use unique path per iteration to avoid overwriting
		req := httptest.NewRequest(http.MethodGet, "/api/item", http.NoBody)
		cache.Set(req, 5*time.Minute, http.StatusOK, headers, body)
		entry := cache.Get(req)
		if entry == nil {
			b.Fatal("expected cache hit after set")
		}
	}
}

func BenchmarkResponseCache_CacheKey(b *testing.B) {
	b.ReportAllocs()

	req := httptest.NewRequest(http.MethodGet, "/api/users?page=1&limit=20&sort=name&order=asc", http.NoBody)

	b.ResetTimer()
	for b.Loop() {
		_ = cacheKey(req)
	}
}

// Fragment Cache Benchmarks

func BenchmarkFragmentCache_GetHit(b *testing.B) {
	b.ReportAllocs()

	cache := newFragmentCache(false, 1000) // production mode

	// Pre-warm with representative HTML fragment
	key := "user-card:123"
	html := `<div class="user-card"><img src="/avatars/123.jpg" alt="User"><h3>Alice Smith</h3><p>Software Engineer</p></div>`
	cache.Set(key, html, 10*time.Minute)

	b.ResetTimer()
	for b.Loop() {
		result, ok := cache.Get(key)
		if !ok || result == "" {
			b.Fatal("expected cache hit")
		}
	}
}

func BenchmarkFragmentCache_GetMiss(b *testing.B) {
	b.ReportAllocs()

	cache := newFragmentCache(false, 1000)

	b.ResetTimer()
	for b.Loop() {
		_, _ = cache.Get("missing-key:999")
	}
}

func BenchmarkFragmentCache_SetThenGet(b *testing.B) {
	b.ReportAllocs()

	cache := newFragmentCache(false, 1000)

	html := `<article class="post"><h2>Blog Post Title</h2><p>Lorem ipsum dolor sit amet...</p></article>`

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		key := "post:1"
		cache.Set(key, html, 5*time.Minute)
		result, ok := cache.Get(key)
		if !ok || result == "" {
			b.Fatal("expected cache hit after set")
		}
	}
}

func BenchmarkFragmentCache_LargeFragment(b *testing.B) {
	b.ReportAllocs()

	cache := newFragmentCache(false, 1000)

	// Simulate a larger cached fragment (e.g., a table with many rows)
	var largeHTML string
	largeHTML = `<table class="data-table"><thead><tr><th>ID</th><th>Name</th><th>Email</th></tr></thead><tbody>`
	for range 100 {
		largeHTML += `<tr><td>123</td><td>User Name</td><td>user@example.com</td></tr>`
	}
	largeHTML += `</tbody></table>`

	key := "large-table:1"
	cache.Set(key, largeHTML, 10*time.Minute)

	b.ResetTimer()
	for b.Loop() {
		result, ok := cache.Get(key)
		if !ok || result == "" {
			b.Fatal("expected cache hit")
		}
	}
}

func BenchmarkFragmentCache_ManyKeys(b *testing.B) {
	b.ReportAllocs()

	cache := newFragmentCache(false, 10000)

	// Pre-populate with many entries
	html := `<div class="item">Content here</div>`
	for i := range 1000 {
		cache.Set("item:"+string(rune('a'+i%26))+string(rune('0'+i%10)), html, 10*time.Minute)
	}

	// Benchmark hitting one of them
	cache.Set("target-key", html, 10*time.Minute)

	b.ResetTimer()
	for b.Loop() {
		result, ok := cache.Get("target-key")
		if !ok || result == "" {
			b.Fatal("expected cache hit")
		}
	}
}
