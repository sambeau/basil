package server

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sambeau/basil/config"
)

// siteHandler implements filesystem-based routing for Basil sites.
// It serves requests by walking back from the requested path to find an index.pars handler.
type siteHandler struct {
	server      *Server
	siteRoot    string // Absolute path to the site directory
	scriptCache *scriptCache
}

// newSiteHandler creates a handler for filesystem-based routing.
func newSiteHandler(s *Server, siteRoot string, cache *scriptCache) *siteHandler {
	return &siteHandler{
		server:      s,
		siteRoot:    siteRoot,
		scriptCache: cache,
	}
}

// ServeHTTP handles HTTP requests using filesystem-based routing.
// It walks back from the requested URL path to find the nearest index.pars handler.
func (h *siteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path

	// Security: reject paths with .. components (path traversal)
	if containsPathTraversal(urlPath) {
		h.server.logWarn("blocked path traversal attempt: %s", urlPath)
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}

	// Security: reject paths starting with . (dotfiles/hidden files)
	if containsDotfile(urlPath) {
		h.server.logWarn("blocked dotfile access attempt: %s", urlPath)
		http.NotFound(w, r)
		return
	}

	// Handle trailing slash redirect for directory-like paths
	// If /foo exists as a directory with index.pars but request is /foo, redirect to /foo/
	if !strings.HasSuffix(urlPath, "/") && urlPath != "/" {
		dirPath := filepath.Join(h.siteRoot, urlPath)
		indexPath := filepath.Join(dirPath, "index.pars")
		if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
			if _, err := os.Stat(indexPath); err == nil {
				http.Redirect(w, r, urlPath+"/", http.StatusFound)
				return
			}
		}
	}

	// Try static files first (from global public_dir)
	if h.server.config.PublicDir != "" && urlPath != "/" {
		staticPath := filepath.Join(h.server.config.PublicDir, urlPath)
		if info, err := os.Stat(staticPath); err == nil && !info.IsDir() {
			serveStaticFile(w, r, staticPath, h.server.devLog != nil)
			return
		}
	}

	// Check if request is for a .part file (for Part refresh/lazy-load)
	if strings.HasSuffix(urlPath, ".part") {
		// Parts can be located in two places:
		// 1. Within the site/ directory (e.g., site/results.part accessed as /results.part)
		// 2. In sibling directories at project root (e.g., parts/counter.part accessed as /parts/counter.part)
		
		// First, try within site/ directory (for Parts in same directory as handler)
		partPath := filepath.Join(h.siteRoot, urlPath)
		partPath = filepath.Clean(partPath)
		if info, err := os.Stat(partPath); err == nil && !info.IsDir() {
			h.servePartFile(w, r, partPath, urlPath)
			return
		}
		
		// Then try at project root level (for @~/parts/ style paths)
		partPath = filepath.Join(h.siteRoot, "..", urlPath)
		partPath = filepath.Clean(partPath)
		if info, err := os.Stat(partPath); err == nil && !info.IsDir() {
			h.servePartFile(w, r, partPath, urlPath)
			return
		}
		
		// Part file not found in either location
		http.NotFound(w, r)
		return
	}

	// Walk back to find the nearest index.pars handler
	handlerPath, subpath := h.findHandler(urlPath)
	if handlerPath == "" {
		// No handler found - use prelude 404 page
		h.server.handle404(w, r)
		return
	}

	// Found a handler - serve the request
	h.serveWithHandler(w, r, handlerPath, subpath)
}

// findHandler walks back from the URL path to find the nearest index.pars file.
// Returns the handler path and the subpath (portion of URL not consumed by the handler).
// Returns empty string for handlerPath if no handler is found.
func (h *siteHandler) findHandler(urlPath string) (handlerPath string, subpath string) {
	// Clean and normalize the URL path
	urlPath = filepath.Clean(urlPath)
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}

	// Split the URL path into segments
	segments := splitPath(urlPath)

	// Walk back from the deepest path to the root
	for i := len(segments); i >= 0; i-- {
		// Build the path to check
		checkPath := "/"
		if i > 0 {
			checkPath = "/" + strings.Join(segments[:i], "/")
		}

		// Look for index.pars at this path
		indexPath := filepath.Join(h.siteRoot, checkPath, "index.pars")
		if _, err := os.Stat(indexPath); err == nil {
			// Found a handler!
			// Subpath is everything after the matched portion
			if i < len(segments) {
				subpath = "/" + strings.Join(segments[i:], "/")
			} else {
				subpath = ""
			}
			return indexPath, subpath
		}
	}

	return "", ""
}

// getCheckedPaths returns the list of paths checked during handler lookup (for dev 404 page).
func (h *siteHandler) getCheckedPaths(urlPath string) []string {
	urlPath = filepath.Clean(urlPath)
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}

	segments := splitPath(urlPath)
	var paths []string

	for i := len(segments); i >= 0; i-- {
		checkPath := "/"
		if i > 0 {
			checkPath = "/" + strings.Join(segments[:i], "/")
		}
		indexPath := filepath.Join(checkPath, "index.pars")
		paths = append(paths, indexPath)
	}

	return paths
}

// serveWithHandler executes the found handler with the calculated subpath.
func (h *siteHandler) serveWithHandler(w http.ResponseWriter, r *http.Request, handlerPath string, subpath string) {
	// Calculate the route path (URL path minus subpath) for publicUrl() context
	routePath := strings.TrimSuffix(r.URL.Path, subpath)
	if !strings.HasSuffix(routePath, "/") && routePath != "" {
		routePath += "/"
	}
	if routePath == "" {
		routePath = "/"
	}

	// Determine handler root (parent of site directory) for @~ resolution
	// This allows handlers to access sibling directories like public/, components/, etc.
	handlerRoot := filepath.Dir(h.siteRoot)

	// Create a synthetic route for the handler
	route := config.Route{
		Path:      routePath,
		Handler:   handlerPath,
		PublicDir: handlerRoot, // Use handler root, not handler's directory
		Cache:     h.server.config.SiteCache,
	}

	// Create the handler using existing infrastructure
	handler, err := newParsleyHandler(h.server, route, h.scriptCache)
	if err != nil {
		h.server.logError("failed to create handler for %s: %v", handlerPath, err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Store subpath in request context for buildRequestContext to pick up
	ctx := r.Context()
	ctx = withSubpath(ctx, subpath)
	r = r.WithContext(ctx)

	// Apply auth middleware (optional auth for now - can be enhanced later)
	finalHandler := h.server.applyAuthMiddleware(handler, "optional")
	finalHandler.ServeHTTP(w, r)
}

// splitPath splits a URL path into non-empty segments.
// e.g., "/reports/2025/Q4/" -> ["reports", "2025", "Q4"]
func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

// containsPathTraversal checks if a path contains .. components.
func containsPathTraversal(path string) bool {
	segments := strings.Split(path, "/")
	for _, seg := range segments {
		if seg == ".." {
			return true
		}
	}
	return false
}

// containsDotfile checks if any path segment starts with a dot (hidden files).
// Excludes the empty segment from "/path" which would be before the first slash.
func containsDotfile(path string) bool {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for _, seg := range segments {
		if seg != "" && strings.HasPrefix(seg, ".") {
			return true
		}
	}
	return false
}

// subpathContextKey is the context key for storing the subpath in site routing.
type subpathContextKey struct{}

// withSubpath adds the subpath to a context.
func withSubpath(ctx context.Context, subpath string) context.Context {
	return context.WithValue(ctx, subpathContextKey{}, subpath)
}

// getSubpath retrieves the subpath from a context.
// Returns empty string if not in site routing mode.
func getSubpath(ctx context.Context) string {
	if v, ok := ctx.Value(subpathContextKey{}).(string); ok {
		return v
	}
	return ""
}

// servePartFile serves a .part file for Part component refresh/lazy-load.
func (h *siteHandler) servePartFile(w http.ResponseWriter, r *http.Request, partPath string, urlPath string) {
	// Determine handler root (parent of site directory)
	handlerRoot := filepath.Dir(h.siteRoot)

	// Calculate the route path (URL path minus .part file itself)
	routePath := filepath.Dir(urlPath)
	if routePath == "." {
		routePath = "/"
	} else if !strings.HasPrefix(routePath, "/") {
		routePath = "/" + routePath
	}
	if !strings.HasSuffix(routePath, "/") {
		routePath += "/"
	}

	// Create a synthetic route for the Part handler
	route := config.Route{
		Path:      routePath,
		Handler:   partPath,
		PublicDir: handlerRoot,
	}

	// Create the handler using existing infrastructure
	handler, err := newParsleyHandler(h.server, route, h.scriptCache)
	if err != nil {
		h.server.logError("failed to create Part handler for %s: %v", partPath, err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Apply auth middleware (optional auth for now)
	finalHandler := h.server.applyAuthMiddleware(handler, "optional")
	finalHandler.ServeHTTP(w, r)
}
