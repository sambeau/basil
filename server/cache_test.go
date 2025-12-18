package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResponseCache_BasicCaching(t *testing.T) {
	cache := newResponseCache(false, false) // production mode

	// Create a test request
	req := httptest.NewRequest("GET", "/test?foo=bar", nil)

	// Initially, cache should be empty
	if entry := cache.Get(req); entry != nil {
		t.Error("expected nil from empty cache")
	}

	// Store a response
	headers := http.Header{}
	headers.Set("Content-Type", "text/html")
	body := []byte("<html>test</html>")
	cache.Set(req, 5*time.Minute, 200, headers, body)

	// Retrieve from cache
	entry := cache.Get(req)
	if entry == nil {
		t.Fatal("expected cached entry")
	}
	if entry.status != 200 {
		t.Errorf("expected status 200, got %d", entry.status)
	}
	if string(entry.body) != "<html>test</html>" {
		t.Errorf("expected body '<html>test</html>', got '%s'", entry.body)
	}
	if entry.headers.Get("Content-Type") != "text/html" {
		t.Errorf("expected Content-Type 'text/html', got '%s'", entry.headers.Get("Content-Type"))
	}
}

func TestResponseCache_DevMode(t *testing.T) {
	cache := newResponseCache(true, false) // dev mode - caching disabled

	req := httptest.NewRequest("GET", "/test", nil)

	// Set should do nothing in dev mode
	cache.Set(req, 5*time.Minute, 200, http.Header{}, []byte("test"))

	// Get should return nil in dev mode
	if entry := cache.Get(req); entry != nil {
		t.Error("expected nil in dev mode")
	}
}

func TestResponseCache_DevModeWithCacheEnabled(t *testing.T) {
	cache := newResponseCache(true, true) // dev mode with caching enabled

	req := httptest.NewRequest("GET", "/test", nil)

	// Set should work when cache is enabled
	cache.Set(req, 5*time.Minute, 200, http.Header{}, []byte("test"))

	// Get should return the cached entry
	if entry := cache.Get(req); entry == nil {
		t.Error("expected cached entry in dev mode with cache enabled")
	}
}

func TestResponseCache_Expiration(t *testing.T) {
	cache := newResponseCache(false, false)

	req := httptest.NewRequest("GET", "/test", nil)

	// Store with very short TTL
	cache.Set(req, 1*time.Millisecond, 200, http.Header{}, []byte("test"))

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Should be expired
	if entry := cache.Get(req); entry != nil {
		t.Error("expected nil for expired entry")
	}
}

func TestResponseCache_DifferentQueries(t *testing.T) {
	cache := newResponseCache(false, false)

	req1 := httptest.NewRequest("GET", "/test?a=1", nil)
	req2 := httptest.NewRequest("GET", "/test?a=2", nil)

	// Store different responses for different query strings
	cache.Set(req1, 5*time.Minute, 200, http.Header{}, []byte("response1"))
	cache.Set(req2, 5*time.Minute, 200, http.Header{}, []byte("response2"))

	// Should get different responses
	entry1 := cache.Get(req1)
	entry2 := cache.Get(req2)

	if entry1 == nil || string(entry1.body) != "response1" {
		t.Error("expected 'response1' for first request")
	}
	if entry2 == nil || string(entry2.body) != "response2" {
		t.Error("expected 'response2' for second request")
	}
}

func TestResponseCache_DifferentMethods(t *testing.T) {
	cache := newResponseCache(false, false)

	getReq := httptest.NewRequest("GET", "/test", nil)
	postReq := httptest.NewRequest("POST", "/test", nil)

	// Store response for GET
	cache.Set(getReq, 5*time.Minute, 200, http.Header{}, []byte("get-response"))

	// GET should hit cache
	if entry := cache.Get(getReq); entry == nil {
		t.Error("expected cache hit for GET")
	}

	// POST should miss cache (different key)
	if entry := cache.Get(postReq); entry != nil {
		t.Error("expected cache miss for POST")
	}
}

func TestResponseCache_Clear(t *testing.T) {
	cache := newResponseCache(false, false)

	req := httptest.NewRequest("GET", "/test", nil)
	cache.Set(req, 5*time.Minute, 200, http.Header{}, []byte("test"))

	// Verify it's cached
	if entry := cache.Get(req); entry == nil {
		t.Fatal("expected entry before clear")
	}

	// Clear cache
	cache.Clear()

	// Should be empty
	if entry := cache.Get(req); entry != nil {
		t.Error("expected nil after clear")
	}
}

func TestResponseCache_Prune(t *testing.T) {
	cache := newResponseCache(false, false)

	// Add expired and valid entries
	expiredReq := httptest.NewRequest("GET", "/expired", nil)
	validReq := httptest.NewRequest("GET", "/valid", nil)

	cache.Set(expiredReq, 1*time.Millisecond, 200, http.Header{}, []byte("expired"))
	cache.Set(validReq, 5*time.Minute, 200, http.Header{}, []byte("valid"))

	// Wait for first to expire
	time.Sleep(10 * time.Millisecond)

	// Prune should remove 1 entry
	pruned := cache.Prune()
	if pruned != 1 {
		t.Errorf("expected 1 pruned entry, got %d", pruned)
	}

	// Valid entry should still exist
	if entry := cache.Get(validReq); entry == nil {
		t.Error("expected valid entry to remain")
	}
}

func TestResponseCache_ZeroTTL(t *testing.T) {
	cache := newResponseCache(false, false)

	req := httptest.NewRequest("GET", "/test", nil)

	// Zero TTL should not cache
	cache.Set(req, 0, 200, http.Header{}, []byte("test"))

	if entry := cache.Get(req); entry != nil {
		t.Error("expected nil for zero TTL")
	}
}

func TestResponseCache_Size(t *testing.T) {
	cache := newResponseCache(false, false)

	if cache.Size() != 0 {
		t.Error("expected empty cache")
	}

	req := httptest.NewRequest("GET", "/test", nil)
	cache.Set(req, 5*time.Minute, 200, http.Header{}, []byte("test"))

	if cache.Size() != 1 {
		t.Errorf("expected size 1, got %d", cache.Size())
	}
}

func TestCachedResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	crw := newCachedResponseWriter(rec)

	// Write headers and body
	crw.Header().Set("X-Custom", "test")
	crw.WriteHeader(201)
	crw.Write([]byte("hello"))
	crw.Write([]byte(" world"))

	// Check captured values
	if crw.statusCode != 201 {
		t.Errorf("expected status 201, got %d", crw.statusCode)
	}
	if string(crw.body) != "hello world" {
		t.Errorf("expected body 'hello world', got '%s'", crw.body)
	}

	// Check underlying writer received everything
	if rec.Code != 201 {
		t.Errorf("expected recorder status 201, got %d", rec.Code)
	}
	if rec.Body.String() != "hello world" {
		t.Errorf("expected recorder body 'hello world', got '%s'", rec.Body.String())
	}
}
