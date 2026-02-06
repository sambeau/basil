package evaluator

import (
	"database/sql"
	"sync"
	"time"
)

// connectionCache manages a cache of database connections with TTL and health checks
type connectionCache[T any] struct {
	mu          sync.RWMutex
	conns       map[string]*cachedConn[T]
	maxSize     int
	ttl         time.Duration
	cleanupTick time.Duration
	healthCheck func(T) error
	closeFunc   func(T) error
	cleanupOnce sync.Once
	stopCleanup chan struct{}
}

// cachedConn wraps a connection with metadata
type cachedConn[T any] struct {
	conn      T
	createdAt time.Time
	lastUsed  time.Time
}

// newConnectionCache creates a new connection cache with the specified configuration
func newConnectionCache[T any](maxSize int, ttl time.Duration, healthCheck func(T) error, closeFunc func(T) error) *connectionCache[T] {
	return &connectionCache[T]{
		conns:       make(map[string]*cachedConn[T]),
		maxSize:     maxSize,
		ttl:         ttl,
		cleanupTick: 5 * time.Minute,
		healthCheck: healthCheck,
		closeFunc:   closeFunc,
		stopCleanup: make(chan struct{}),
	}
}

// get retrieves a connection from the cache if it exists and is still valid
// Returns (connection, found) where found indicates if a valid connection was found
func (c *connectionCache[T]) get(key string) (T, bool) {
	c.mu.RLock()
	cached, exists := c.conns[key]
	c.mu.RUnlock()

	if !exists {
		var zero T
		return zero, false
	}

	now := time.Now()

	// Check if connection has expired
	if now.Sub(cached.createdAt) > c.ttl {
		c.mu.Lock()
		if err := c.closeFunc(cached.conn); err != nil {
			// Log error but continue with eviction
		}
		delete(c.conns, key)
		c.mu.Unlock()
		var zero T
		return zero, false
	}

	// Health check the connection
	if c.healthCheck != nil {
		if err := c.healthCheck(cached.conn); err != nil {
			c.mu.Lock()
			if err := c.closeFunc(cached.conn); err != nil {
				// Log error but continue with eviction
			}
			delete(c.conns, key)
			c.mu.Unlock()
			var zero T
			return zero, false
		}
	}

	// Update last used time
	c.mu.Lock()
	cached.lastUsed = now
	c.mu.Unlock()

	return cached.conn, true
}

// put adds a connection to the cache, evicting the least recently used if at capacity
func (c *connectionCache[T]) put(key string, conn T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If already at capacity, evict least recently used
	if len(c.conns) >= c.maxSize {
		c.evictLRU()
	}

	now := time.Now()
	c.conns[key] = &cachedConn[T]{
		conn:      conn,
		createdAt: now,
		lastUsed:  now,
	}

	// Start cleanup goroutine on first put
	c.cleanupOnce.Do(func() {
		go c.cleanup()
	})
}

// evictLRU removes the least recently used connection (caller must hold lock)
func (c *connectionCache[T]) evictLRU() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, cached := range c.conns {
		if first || cached.lastUsed.Before(oldestTime) {
			oldestKey = key
			oldestTime = cached.lastUsed
			first = false
		}
	}

	if oldestKey != "" {
		if cached, exists := c.conns[oldestKey]; exists {
			if err := c.closeFunc(cached.conn); err != nil {
				// Log error but continue with eviction
			}
			delete(c.conns, oldestKey)
		}
	}
}

// cleanup runs periodically to remove expired connections
func (c *connectionCache[T]) cleanup() {
	ticker := time.NewTicker(c.cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.evictStale()
		case <-c.stopCleanup:
			return
		}
	}
}

// evictStale removes all expired connections
func (c *connectionCache[T]) evictStale() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, cached := range c.conns {
		if now.Sub(cached.createdAt) > c.ttl {
			if err := c.closeFunc(cached.conn); err != nil {
				// Log error but continue with eviction
			}
			delete(c.conns, key)
		}
	}
}

// close shuts down the cache and closes all connections
func (c *connectionCache[T]) close() error {
	// Stop cleanup goroutine
	close(c.stopCleanup)

	c.mu.Lock()
	defer c.mu.Unlock()

	var firstErr error
	for key, cached := range c.conns {
		if err := c.closeFunc(cached.conn); err != nil && firstErr == nil {
			firstErr = err
		}
		delete(c.conns, key)
	}

	return firstErr
}

// size returns the current number of cached connections
func (c *connectionCache[T]) size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.conns)
}

// Database connection cache with TTL
var dbCache = newConnectionCache[*sql.DB](
	100,            // max 100 database connections
	30*time.Minute, // 30 minute TTL
	func(db *sql.DB) error {
		return db.Ping()
	},
	func(db *sql.DB) error {
		return db.Close()
	},
)

// SFTP connection cache with TTL
var sftpCache = newConnectionCache[*SFTPConnection](
	50,             // max 50 SFTP connections
	15*time.Minute, // 15 minute TTL
	func(conn *SFTPConnection) error {
		// Health check: try to stat a known path
		// If the connection is dead, this will fail
		_, err := conn.Client.Getwd()
		return err
	},
	func(conn *SFTPConnection) error {
		if conn.Client != nil {
			if err := conn.Client.Close(); err != nil {
				return err
			}
		}
		if conn.SSHClient != nil {
			return conn.SSHClient.Close()
		}
		return nil
	},
)
