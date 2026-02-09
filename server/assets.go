package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

// assetEntry holds cached hash information for a file
type assetEntry struct {
	hash    string
	modTime time.Time
	size    int64
}

// assetRegistry manages public URLs for private assets.
// It maps content hashes to file paths and caches hash computations.
type assetRegistry struct {
	mu     sync.RWMutex
	byHash map[string]string     // hash -> absolute filepath
	cache  map[string]assetEntry // filepath -> cached hash info
	logger func(format string, args ...any)
}

// newAssetRegistry creates a new asset registry
func newAssetRegistry(logger func(format string, args ...any)) *assetRegistry {
	return &assetRegistry{
		byHash: make(map[string]string),
		cache:  make(map[string]assetEntry),
		logger: logger,
	}
}

// Register registers a file and returns its public URL.
// Returns error if file doesn't exist or exceeds size limits.
func (r *assetRegistry) Register(filepath string) (string, error) {
	stat, err := os.Stat(filepath)
	if err != nil {
		return "", fmt.Errorf("file not found: %s", filepath)
	}

	if stat.IsDir() {
		return "", fmt.Errorf("cannot create public URL for directory: %s", filepath)
	}

	// Size limits
	const warnSize = 10 * 1024 * 1024 // 10MB
	const maxSize = 100 * 1024 * 1024 // 100MB

	if stat.Size() > maxSize {
		return "", fmt.Errorf("file too large for publicUrl() (>100MB): %s - use public/ folder instead", filepath)
	}
	if stat.Size() > warnSize && r.logger != nil {
		r.logger("publicUrl(): large file %s (%dMB) - consider using public/ folder",
			filepath, stat.Size()/1024/1024)
	}

	// Check cache for existing hash
	r.mu.RLock()
	if entry, ok := r.cache[filepath]; ok {
		if entry.modTime.Equal(stat.ModTime()) && entry.size == stat.Size() {
			r.mu.RUnlock()
			return formatAssetURL(entry.hash, filepath), nil
		}
	}
	r.mu.RUnlock()

	// Read and hash file
	content, err := os.ReadFile(filepath)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}
	hash := sha256Short(content)

	// Update registry (write lock)
	r.mu.Lock()
	r.byHash[hash] = filepath
	r.cache[filepath] = assetEntry{
		hash:    hash,
		modTime: stat.ModTime(),
		size:    stat.Size(),
	}
	r.mu.Unlock()

	return formatAssetURL(hash, filepath), nil
}

// Lookup returns the file path for a hash, or empty string if not found
func (r *assetRegistry) Lookup(hash string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fp, ok := r.byHash[hash]
	return fp, ok
}

// Clear removes all registered assets (called on server reload)
func (r *assetRegistry) Clear() {
	r.mu.Lock()
	r.byHash = make(map[string]string)
	r.cache = make(map[string]assetEntry)
	r.mu.Unlock()
}

// sha256Short computes SHA256 and returns first 16 hex chars
func sha256Short(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])[:16]
}

// formatAssetURL returns the public URL for a hash and original filepath
func formatAssetURL(hash, filepath string) string {
	ext := path.Ext(filepath)
	return "/__p/" + hash + ext
}

// assetHandler serves registered assets at /__p/ URLs
type assetHandler struct {
	registry *assetRegistry
	devMode  bool
}

// ServeHTTP handles requests to /__p/{hash}.{ext}
func (h *assetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract hash from URL: /__p/{hash}.{ext}
	urlPath := strings.TrimPrefix(r.URL.Path, "/__p/")

	// Split off extension to get hash
	ext := path.Ext(urlPath)
	hash := strings.TrimSuffix(urlPath, ext)

	if hash == "" {
		http.NotFound(w, r)
		return
	}

	// Lookup file path
	fp, ok := h.registry.Lookup(hash)
	if !ok {
		http.NotFound(w, r)
		return
	}

	// Verify extension matches (security check)
	if path.Ext(fp) != ext {
		http.NotFound(w, r)
		return
	}

	// Set cache headers based on mode
	if h.devMode {
		// Dev mode: disable caching to prevent stale content
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	} else {
		// Production: aggressive caching (content-addressed = immutable)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}

	// Serve the file - http.ServeFile handles Content-Type and Range requests
	http.ServeFile(w, r, fp)
}

// newAssetHandler creates a new asset handler
func newAssetHandler(registry *assetRegistry, devMode bool) *assetHandler {
	return &assetHandler{
		registry: registry,
		devMode:  devMode,
	}
}
