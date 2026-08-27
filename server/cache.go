package server

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// responseCache stores rendered responses for routes with caching enabled.
// Each cache entry stores the full response (status, headers, body) keyed by
// request attributes (method, path, query string) salted with a release
// generation: SwapRelease advances the generation before building the new
// release's handlers, so a write from a request still in flight on the old
// release lands under the old generation and can never be served to a
// request on the new one.
type responseCache struct {
	mu            sync.RWMutex
	entries       map[string]*cacheEntry
	cacheDisabled bool // true when caching is disabled (dev mode without override)

	// gen is the current release generation. Handlers pin the value at
	// construction and pass it to Get and Set.
	gen atomic.Uint64
}

// Generation returns the current release generation, for handlers to pin at
// construction.
func (c *responseCache) Generation() uint64 {
	return c.gen.Load()
}

// Advance starts a new release generation. Called by SwapRelease before the
// new release's handlers are built; entries of older generations become
// unreachable to them and are wiped by the post-swap Clear (or expire).
func (c *responseCache) Advance() {
	c.gen.Add(1)
}

// cacheEntry represents a cached response with expiration time.
type cacheEntry struct {
	status    int
	headers   http.Header
	body      []byte
	expiresAt time.Time
}

// newResponseCache creates a new response cache. noCache comes from
// Config.NoCache - dev mode without the dev.cache opt-in.
func newResponseCache(noCache bool) *responseCache {
	return &responseCache{
		entries:       make(map[string]*cacheEntry),
		cacheDisabled: noCache,
	}
}

// cacheKey generates a unique key for a request based on the release
// generation, method, path, and query. For cache busting, we include query
// parameters in the key.
func cacheKey(r *http.Request, gen uint64) string {
	// Use SHA256 to handle long query strings efficiently
	h := sha256.New()
	var genBytes [8]byte
	binary.BigEndian.PutUint64(genBytes[:], gen)
	h.Write(genBytes[:])
	h.Write([]byte(r.Method))
	h.Write([]byte(":"))
	h.Write([]byte(r.URL.Path))
	h.Write([]byte("?"))
	h.Write([]byte(r.URL.RawQuery))
	return hex.EncodeToString(h.Sum(nil))
}

// Get retrieves a cached response if available and not expired. gen is the
// caller's pinned release generation. Returns nil if cache miss or expired.
func (c *responseCache) Get(r *http.Request, gen uint64) *cacheEntry {
	// No caching when disabled
	if c.cacheDisabled {
		return nil
	}

	key := cacheKey(r, gen)

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

// Set stores a response in the cache with the given TTL, under the caller's
// pinned release generation.
func (c *responseCache) Set(r *http.Request, gen uint64, ttl time.Duration, status int, headers http.Header, body []byte) {
	// No caching when disabled or with zero TTL
	if c.cacheDisabled || ttl <= 0 {
		return
	}

	key := cacheKey(r, gen)
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
