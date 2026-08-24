package server

import (
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// fragmentCache stores rendered HTML fragments for the <basil.cache.Cache> component.
// It uses an in-memory LRU cache with time-based expiration.
//
// Keys are user-chosen strings, so releases would collide across a swap; the
// evaluator therefore talks to a generation-pinned view (see view), never to
// the cache directly.
type fragmentCache struct {
	mu       sync.RWMutex
	entries  map[string]*fragmentEntry
	maxSize  int
	devMode  bool
	hits     atomic.Int64
	misses   atomic.Int64
	disabled bool // For testing: when true, always returns miss

	// gen is the current release generation, advanced by SwapRelease before
	// the new release's handlers pin their views.
	gen atomic.Uint64
}

// Generation returns the current release generation.
func (c *fragmentCache) Generation() uint64 {
	return c.gen.Load()
}

// Advance starts a new release generation: entries written through views of
// older generations become unreachable to views pinned afterwards.
func (c *fragmentCache) Advance() {
	c.gen.Add(1)
}

// view returns a FragmentCacher pinned to the cache's current generation.
// Handlers capture one at construction, so a fragment written by a request
// that outlives a swap stays in its own release's keyspace.
func (c *fragmentCache) view() *fragmentCacheView {
	return &fragmentCacheView{cache: c, gen: c.gen.Load()}
}

// fragmentCacheView is a generation-pinned facade over fragmentCache. It
// implements evaluator.FragmentCacher by salting every key with the pinned
// generation.
type fragmentCacheView struct {
	cache *fragmentCache
	gen   uint64
}

func (v *fragmentCacheView) salted(key string) string {
	return strconv.FormatUint(v.gen, 10) + ":" + key
}

func (v *fragmentCacheView) Get(key string) (string, bool) {
	return v.cache.Get(v.salted(key))
}

func (v *fragmentCacheView) Set(key string, html string, maxAge time.Duration) {
	v.cache.Set(v.salted(key), html, maxAge)
}

func (v *fragmentCacheView) Invalidate(key string) {
	v.cache.Invalidate(v.salted(key))
}

// fragmentEntry represents a cached HTML fragment with expiration.
type fragmentEntry struct {
	html      string
	expiresAt time.Time
	size      int // Approximate size in bytes for LRU tracking
}

// newFragmentCache creates a new fragment cache.
// In dev mode, caching is disabled but operations are logged.
func newFragmentCache(devMode bool, maxSize int) *fragmentCache {
	if maxSize <= 0 {
		maxSize = 1000 // Default to 1000 entries
	}
	return &fragmentCache{
		entries: make(map[string]*fragmentEntry),
		maxSize: maxSize,
		devMode: devMode,
	}
}

// Get retrieves a cached fragment if available and not expired.
// Returns the HTML and true on cache hit, empty string and false on miss.
func (c *fragmentCache) Get(key string) (string, bool) {
	// No caching in dev mode
	if c.devMode || c.disabled {
		c.misses.Add(1)
		return "", false
	}

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		c.misses.Add(1)
		return "", false
	}

	// Check expiration
	if time.Now().After(entry.expiresAt) {
		// Entry expired, remove it
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		c.misses.Add(1)
		return "", false
	}

	c.hits.Add(1)
	return entry.html, true
}

// Set stores a fragment in the cache with the given TTL.
// If maxAge is 0 or negative, the entry is not stored.
func (c *fragmentCache) Set(key string, html string, maxAge time.Duration) {
	// No caching in dev mode or with zero/negative TTL
	if c.devMode || c.disabled || maxAge <= 0 {
		return
	}

	entry := &fragmentEntry{
		html:      html,
		expiresAt: time.Now().Add(maxAge),
		size:      len(html),
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple LRU: if at capacity, remove oldest expired entries first
	if len(c.entries) >= c.maxSize {
		c.evictExpired()

		// If still at capacity, remove some entries (simple eviction, not true LRU)
		if len(c.entries) >= c.maxSize {
			c.evictOldest(c.maxSize / 10) // Remove 10% of entries
		}
	}

	c.entries[key] = entry
}

// Invalidate removes a specific cache entry by key.
func (c *fragmentCache) Invalidate(key string) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

// InvalidatePrefix removes all cache entries with keys starting with the given prefix.
// Useful for invalidating all fragments from a specific handler.
func (c *fragmentCache) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.entries {
		if strings.HasPrefix(key, prefix) {
			delete(c.entries, key)
		}
	}
}

// Clear removes all entries from the cache.
func (c *fragmentCache) Clear() {
	c.mu.Lock()
	c.entries = make(map[string]*fragmentEntry)
	c.mu.Unlock()
}

// Stats returns cache statistics.
func (c *fragmentCache) Stats() FragmentCacheStats {
	c.mu.RLock()
	count := len(c.entries)
	var totalSize int
	for _, e := range c.entries {
		totalSize += e.size
	}
	c.mu.RUnlock()

	return FragmentCacheStats{
		Entries:   count,
		Hits:      c.hits.Load(),
		Misses:    c.misses.Load(),
		SizeBytes: totalSize,
		DevMode:   c.devMode,
	}
}

// FragmentCacheStats holds cache statistics for DevTools display.
type FragmentCacheStats struct {
	Entries   int
	Hits      int64
	Misses    int64
	SizeBytes int
	DevMode   bool
}

// HitRate returns the cache hit rate as a percentage (0-100).
func (s FragmentCacheStats) HitRate() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total) * 100
}

// evictExpired removes all expired entries. Caller must hold the lock.
func (c *fragmentCache) evictExpired() {
	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, key)
		}
	}
}

// evictOldest removes n entries. Not true LRU since we don't track access time,
// but removes entries that expire soonest. Caller must hold the lock.
func (c *fragmentCache) evictOldest(n int) {
	if n <= 0 {
		return
	}

	// Find entries that expire soonest
	type keyExpiry struct {
		key    string
		expiry time.Time
	}
	var candidates []keyExpiry
	for key, entry := range c.entries {
		candidates = append(candidates, keyExpiry{key, entry.expiresAt})
	}

	// Sort by expiry time (soonest first)
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].expiry.Before(candidates[i].expiry) {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// Remove first n
	for i := 0; i < n && i < len(candidates); i++ {
		delete(c.entries, candidates[i].key)
	}
}

// SetDisabled sets whether the cache is disabled (for testing).
func (c *fragmentCache) SetDisabled(disabled bool) {
	c.disabled = disabled
}
