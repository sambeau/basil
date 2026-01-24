package server

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	// SQLite driver (pure Go, no CGO required)
	_ "modernc.org/sqlite"
)

// DevLog manages the developer log database for dev tools.
type DevLog struct {
	mu          sync.RWMutex
	db          *sql.DB
	path        string
	maxSize     int64  // Maximum database size in bytes (default 10MB)
	truncatePct int    // Percentage to delete when truncating (default 25)
	seq         uint64 // Sequence number, incremented on each log write
}

// LogEntry represents a single log entry.
type LogEntry struct {
	ID        int64
	Route     string
	Level     string
	Filename  string
	Line      int
	Timestamp time.Time
	CallRepr  string
	ValueRepr string
}

// DevLogConfig holds configuration for the dev log.
type DevLogConfig struct {
	Path        string // Database file path
	MaxSize     int64  // Max size in bytes (default 10MB)
	TruncatePct int    // Percentage to delete when truncating (default 25%)
}

// DefaultDevLogConfig returns the default configuration.
func DefaultDevLogConfig() DevLogConfig {
	return DevLogConfig{
		MaxSize:     10 * 1024 * 1024, // 10MB
		TruncatePct: 25,
	}
}

// NewDevLog creates a new DevLog instance.
// If path is empty, creates a database named "dev_logs.db" in baseDir.
func NewDevLog(baseDir string, cfg DevLogConfig) (*DevLog, error) {
	path := cfg.Path
	if path == "" {
		// Use a fixed filename so logs persist across restarts
		path = filepath.Join(baseDir, "dev_logs.db")
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating log directory: %w", err)
	}

	// Open database with WAL mode for better concurrency
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("opening dev log database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connecting to dev log database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	dl := &DevLog{
		db:          db,
		path:        path,
		maxSize:     cfg.MaxSize,
		truncatePct: cfg.TruncatePct,
	}

	// Set defaults
	if dl.maxSize == 0 {
		dl.maxSize = 10 * 1024 * 1024 // 10MB
	}
	if dl.truncatePct == 0 {
		dl.truncatePct = 25
	}

	// Create schema
	if err := dl.createSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating dev log schema: %w", err)
	}

	return dl, nil
}

// createSchema creates the logs table if it doesn't exist.
func (dl *DevLog) createSchema() error {
	schema := `
		CREATE TABLE IF NOT EXISTS logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			route TEXT NOT NULL DEFAULT '',
			level TEXT NOT NULL DEFAULT 'info',
			filename TEXT NOT NULL,
			line INTEGER NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			call_repr TEXT NOT NULL,
			value_repr TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_logs_route ON logs(route);
		CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp);
	`
	_, err := dl.db.Exec(schema)
	return err
}

// Log writes a log entry to the database.
func (dl *DevLog) Log(entry LogEntry) error {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	// Check if we need to truncate
	if err := dl.maybeAutoTruncate(); err != nil {
		// Log truncation errors but don't fail the log operation
		fmt.Fprintf(os.Stderr, "[WARN] dev log truncation failed: %v\n", err)
	}

	_, err := dl.db.Exec(`
		INSERT INTO logs (route, level, filename, line, call_repr, value_repr)
		VALUES (?, ?, ?, ?, ?, ?)
	`, entry.Route, entry.Level, entry.Filename, entry.Line, entry.CallRepr, entry.ValueRepr)

	if err == nil {
		dl.seq++
	}

	return err
}

// LogFromEvaluator implements evaluator.DevLogWriter interface.
func (dl *DevLog) LogFromEvaluator(route, level, filename string, line int, callRepr, valueRepr string) error {
	return dl.Log(LogEntry{
		Route:     route,
		Level:     level,
		Filename:  filename,
		Line:      line,
		CallRepr:  callRepr,
		ValueRepr: valueRepr,
	})
}

// GetSeq returns the current sequence number for polling.
// The sequence increments each time a log entry is written.
func (dl *DevLog) GetSeq() uint64 {
	dl.mu.RLock()
	defer dl.mu.RUnlock()
	return dl.seq
}

// GetLogs retrieves log entries, optionally filtered by route.
// If route is empty, returns all logs.
func (dl *DevLog) GetLogs(route string, limit int) ([]LogEntry, error) {
	dl.mu.RLock()
	defer dl.mu.RUnlock()

	if limit <= 0 {
		limit = 1000 // Default limit
	}

	var rows *sql.Rows
	var err error

	if route == "" {
		rows, err = dl.db.Query(`
			SELECT id, route, level, filename, line, timestamp, call_repr, value_repr
			FROM logs
			ORDER BY timestamp DESC, id DESC
			LIMIT ?
		`, limit)
	} else {
		rows, err = dl.db.Query(`
			SELECT id, route, level, filename, line, timestamp, call_repr, value_repr
			FROM logs
			WHERE route = ?
			ORDER BY timestamp DESC, id DESC
			LIMIT ?
		`, route, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("querying logs: %w", err)
	}
	defer rows.Close()

	var entries []LogEntry
	for rows.Next() {
		var e LogEntry
		var ts string
		if err := rows.Scan(&e.ID, &e.Route, &e.Level, &e.Filename, &e.Line, &ts, &e.CallRepr, &e.ValueRepr); err != nil {
			return nil, fmt.Errorf("scanning log entry: %w", err)
		}
		// Parse timestamp - try multiple formats SQLite might use
		for _, layout := range []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05",
			time.RFC3339,
		} {
			if t, err := time.Parse(layout, ts); err == nil {
				e.Timestamp = t
				break
			}
		}
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

// ClearLogs removes log entries, optionally filtered by route.
// If route is empty, clears all logs.
func (dl *DevLog) ClearLogs(route string) error {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	var err error
	if route == "" {
		_, err = dl.db.Exec("DELETE FROM logs")
	} else {
		_, err = dl.db.Exec("DELETE FROM logs WHERE route = ?", route)
	}
	return err
}

// Count returns the number of log entries, optionally filtered by route.
func (dl *DevLog) Count(route string) (int, error) {
	dl.mu.RLock()
	defer dl.mu.RUnlock()

	var count int
	var err error

	if route == "" {
		err = dl.db.QueryRow("SELECT COUNT(*) FROM logs").Scan(&count)
	} else {
		err = dl.db.QueryRow("SELECT COUNT(*) FROM logs WHERE route = ?", route).Scan(&count)
	}

	return count, err
}

// maybeAutoTruncate checks database size and truncates if needed.
// Must be called with lock held.
func (dl *DevLog) maybeAutoTruncate() error {
	// Check file size
	info, err := os.Stat(dl.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, nothing to truncate
		}
		return err
	}

	if info.Size() < dl.maxSize {
		return nil // Under limit, no truncation needed
	}

	// Get total count
	var total int
	if err := dl.db.QueryRow("SELECT COUNT(*) FROM logs").Scan(&total); err != nil {
		return err
	}

	if total == 0 {
		return nil
	}

	// Calculate how many to delete
	deleteCount := (total * dl.truncatePct) / 100
	if deleteCount == 0 {
		deleteCount = 1
	}

	// Delete oldest entries
	_, err = dl.db.Exec(`
		DELETE FROM logs WHERE id IN (
			SELECT id FROM logs ORDER BY timestamp ASC, id ASC LIMIT ?
		)
	`, deleteCount)

	if err != nil {
		return fmt.Errorf("truncating logs: %w", err)
	}

	// Vacuum to reclaim space (do this occasionally, not every truncation)
	// For now, skip vacuum to avoid blocking - the space will be reused
	return nil
}

// Close closes the database connection.
func (dl *DevLog) Close() error {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	return dl.db.Close()
}

// Path returns the path to the database file.
func (dl *DevLog) Path() string {
	return dl.path
}
