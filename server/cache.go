package server

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

// responseCache stores rendered responses for routes with caching enabled.
// Each cache entry stores the full response (status, headers, body) keyed by
// request attributes (method, path, query string).
type responseCache struct {
	mu            sync.RWMutex
	entries       map[string]*cacheEntry
	cacheDisabled bool // true when caching is disabled (dev mode without override)
}

// cacheEntry represents a cached response with expiration time.
type cacheEntry struct {
	status    int
	headers   http.Header
	body      []byte
	expiresAt time.Time
}

// newResponseCache creates a new response cache.
// In dev mode, caching is disabled unless cacheEnabled is true.
func newResponseCache(devMode, cacheEnabled bool) *responseCache {
	return &responseCache{
		entries:       make(map[string]*cacheEntry),
		cacheDisabled: devMode && !cacheEnabled,
	}
}

// cacheKey generates a unique key for a request based on method, path, and query.
// For cache busting, we include query parameters in the key.
func cacheKey(r *http.Request) string {
	// Use SHA256 to handle long query strings efficiently
	h := sha256.New()
	h.Write([]byte(r.Method))
	h.Write([]byte(":"))
	h.Write([]byte(r.URL.Path))
	h.Write([]byte("?"))
	h.Write([]byte(r.URL.RawQuery))
	return hex.EncodeToString(h.Sum(nil))
}

// Get retrieves a cached response if available and not expired.
// Returns nil if cache miss or expired.
func (c *responseCache) Get(r *http.Request) *cacheEntry {
	// No caching when disabled
	if c.cacheDisabled {
		return nil
	}

	key := cacheKey(r)

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return nil
	}

	// Check expiration
	if time.Now().After(entry.expiresAt) {
		// Entry expired, remove it
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return nil
	}

	return entry
}

// Set stores a response in the cache with the given TTL.
func (c *responseCache) Set(r *http.Request, ttl time.Duration, status int, headers http.Header, body []byte) {
	// No caching when disabled or with zero TTL
	if c.cacheDisabled || ttl <= 0 {
		return
	}

	key := cacheKey(r)
	entry := &cacheEntry{
		status:    status,
		headers:   headers.Clone(),
		body:      body,
		expiresAt: time.Now().Add(ttl),
	}

	c.mu.Lock()
	c.entries[key] = entry
	c.mu.Unlock()
}

// Clear removes all entries from the cache.
func (c *responseCache) Clear() {
	c.mu.Lock()
	c.entries = make(map[string]*cacheEntry)
	c.mu.Unlock()
}

// Prune removes expired entries from the cache.
// This can be called periodically to prevent memory growth.
func (c *responseCache) Prune() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	pruned := 0

	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, key)
			pruned++
		}
	}

	return pruned
}

// Size returns the number of entries in the cache.
func (c *responseCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// cachedResponseWriter wraps http.ResponseWriter to capture the response
// for caching purposes.
type cachedResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func newCachedResponseWriter(w http.ResponseWriter) *cachedResponseWriter {
	return &cachedResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (c *cachedResponseWriter) WriteHeader(code int) {
	c.statusCode = code
	c.ResponseWriter.WriteHeader(code)
}

func (c *cachedResponseWriter) Write(b []byte) (int, error) {
	c.body = append(c.body, b...)
	return c.ResponseWriter.Write(b)
}
