package images

import (
	"net/http"
	"path"
	"strings"
)

// Handler serves cached images at /__img/ URLs.
// It mirrors the assetHandler pattern for /__p/ URLs.
type Handler struct {
	registry *Registry
	devMode  bool
}

// NewHandler creates a new image handler.
func NewHandler(registry *Registry, devMode bool) *Handler {
	return &Handler{
		registry: registry,
		devMode:  devMode,
	}
}

// ServeHTTP handles requests to /__img/{hash}.{ext}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract hash and extension from URL: /__img/{hash}.{ext}
	urlPath := strings.TrimPrefix(r.URL.Path, "/__img/")

	// Split off extension to get hash
	ext := path.Ext(urlPath)
	hash := strings.TrimSuffix(urlPath, ext)

	if hash == "" {
		http.NotFound(w, r)
		return
	}

	// Lookup file path in registry
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
