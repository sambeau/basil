package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// AssetBundle manages site-wide CSS and JavaScript bundles.
// It discovers, concatenates, and serves all .css and .js files from the handlers directory.
type AssetBundle struct {
	mu          sync.RWMutex
	cssFiles    []string // ordered absolute file paths
	jsFiles     []string // ordered absolute file paths
	cssHash     string   // first 8 chars of SHA-256
	jsHash      string   // first 8 chars of SHA-256
	cssContent  []byte   // concatenated CSS content
	jsContent   []byte   // concatenated JS content
	devMode     bool
	handlersDir string
}

// NewAssetBundle creates a new asset bundle manager.
func NewAssetBundle(handlersDir string, devMode bool) *AssetBundle {
	return &AssetBundle{
		handlersDir: handlersDir,
		devMode:     devMode,
	}
}

// Rebuild walks the handlers directory and rebuilds the CSS/JS bundles.
func (b *AssetBundle) Rebuild() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Discover files in depth-first alphabetical order
	cssFiles, jsFiles, err := b.discoverAssets()
	if err != nil {
		return fmt.Errorf("discovering assets: %w", err)
	}

	// Build CSS bundle
	var cssContent []byte
	if len(cssFiles) > 0 {
		cssContent, err = b.concatenateFiles(cssFiles, "css")
		if err != nil {
			return fmt.Errorf("building CSS bundle: %w", err)
		}
	}

	// Build JS bundle
	var jsContent []byte
	if len(jsFiles) > 0 {
		jsContent, err = b.concatenateFiles(jsFiles, "js")
		if err != nil {
			return fmt.Errorf("building JS bundle: %w", err)
		}
	}

	// Compute hashes
	b.cssFiles = cssFiles
	b.jsFiles = jsFiles
	b.cssContent = cssContent
	b.jsContent = jsContent
	b.cssHash = computeHash(cssContent)
	b.jsHash = computeHash(jsContent)

	return nil
}

// discoverAssets walks the handlers directory and returns ordered lists of CSS and JS files.
func (b *AssetBundle) discoverAssets() (cssFiles, jsFiles []string, err error) {
	// Use WalkDir for efficient directory traversal
	type fileEntry struct {
		path  string
		isCSS bool
	}
	var entries []fileEntry

	err = filepath.WalkDir(b.handlersDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files and directories
		name := d.Name()
		if strings.HasPrefix(name, ".") && path != b.handlersDir {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip public directory (if it exists under handlers for some reason)
		if d.IsDir() && name == "public" {
			return filepath.SkipDir
		}

		// Only process .css and .js files
		if !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(name))
			if ext == ".css" {
				entries = append(entries, fileEntry{path: path, isCSS: true})
			} else if ext == ".js" {
				entries = append(entries, fileEntry{path: path, isCSS: false})
			}
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	// Sort entries by depth-first alphabetical order
	// filepath.WalkDir already does depth-first, but we need to ensure alphabetical within each level
	sort.Slice(entries, func(i, j int) bool {
		// Compare directory depth first
		depthI := strings.Count(entries[i].path, string(filepath.Separator))
		depthJ := strings.Count(entries[j].path, string(filepath.Separator))
		if depthI != depthJ {
			return depthI < depthJ
		}
		// Then alphabetically
		return entries[i].path < entries[j].path
	})

	// Split into CSS and JS files while maintaining order
	for _, entry := range entries {
		if entry.isCSS {
			cssFiles = append(cssFiles, entry.path)
		} else {
			jsFiles = append(jsFiles, entry.path)
		}
	}

	return cssFiles, jsFiles, nil
}

// concatenateFiles reads and concatenates files with optional dev mode comments.
func (b *AssetBundle) concatenateFiles(files []string, fileType string) ([]byte, error) {
	var buf bytes.Buffer

	for _, path := range files {
		// Add source comment in dev mode
		if b.devMode {
			relPath, err := filepath.Rel(b.handlersDir, path)
			if err != nil {
				relPath = path
			}
			separator := fmt.Sprintf("/* ══════════════════════════════════════════════════════════════\n   handlers/%s\n   ══════════════════════════════════════════════════════════════ */\n", relPath)
			if fileType == "js" {
				separator = fmt.Sprintf("/* ══════════════════════════════════════════════════════════════\n   handlers/%s\n   ══════════════════════════════════════════════════════════════ */\n", relPath)
			}
			buf.WriteString(separator)
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}

		buf.Write(content)

		// Add newline between files
		if !bytes.HasSuffix(content, []byte("\n")) {
			buf.WriteByte('\n')
		}
	}

	return buf.Bytes(), nil
}

// computeHash computes the first 8 characters of SHA-256 hash.
func computeHash(content []byte) string {
	if len(content) == 0 {
		return ""
	}
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])[:8]
}

// CSSUrl returns the URL for the CSS bundle, or empty string if no CSS files.
func (b *AssetBundle) CSSUrl() string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.cssFiles) == 0 {
		return ""
	}
	return fmt.Sprintf("/__site.css?v=%s", b.cssHash)
}

// JSUrl returns the URL for the JS bundle, or empty string if no JS files.
func (b *AssetBundle) JSUrl() string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.jsFiles) == 0 {
		return ""
	}
	return fmt.Sprintf("/__site.js?v=%s", b.jsHash)
}

// ServeCSS serves the CSS bundle.
func (b *AssetBundle) ServeCSS(w http.ResponseWriter, r *http.Request) {
	b.mu.RLock()
	content := b.cssContent
	hash := b.cssHash
	b.mu.RUnlock()

	if len(content) == 0 {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("ETag", fmt.Sprintf(`"%s"`, hash))

	// Cache headers
	if b.devMode {
		w.Header().Set("Cache-Control", "no-cache")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=31536000")
	}

	// Handle ETag caching
	if match := r.Header.Get("If-None-Match"); match == fmt.Sprintf(`"%s"`, hash) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	http.ServeContent(w, r, "site.css", time.Time{}, bytes.NewReader(content))
}

// ServeJS serves the JS bundle.
func (b *AssetBundle) ServeJS(w http.ResponseWriter, r *http.Request) {
	b.mu.RLock()
	content := b.jsContent
	hash := b.jsHash
	b.mu.RUnlock()

	if len(content) == 0 {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("ETag", fmt.Sprintf(`"%s"`, hash))

	// Cache headers
	if b.devMode {
		w.Header().Set("Cache-Control", "no-cache")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=31536000")
	}

	// Handle ETag caching
	if match := r.Header.Get("If-None-Match"); match == fmt.Sprintf(`"%s"`, hash) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	http.ServeContent(w, r, "site.js", time.Time{}, bytes.NewReader(content))
}
