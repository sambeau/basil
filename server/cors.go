package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/sambeau/basil/config"
)

// CORSMiddleware handles Cross-Origin Resource Sharing (CORS) headers
type CORSMiddleware struct {
	config config.CORSConfig
}

// NewCORSMiddleware creates a new CORS middleware with the given configuration
func NewCORSMiddleware(cfg config.CORSConfig) *CORSMiddleware {
	return &CORSMiddleware{config: cfg}
}

// Handler wraps an http.Handler to add CORS headers
func (m *CORSMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip if CORS not configured (no origins specified)
		if len(m.config.Origins) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")
		// No Origin header means same-origin request - no CORS needed
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Check if origin is allowed
		if !m.isOriginAllowed(origin) {
			// Origin not allowed - continue without CORS headers
			// Browser will block the response
			next.ServeHTTP(w, r)
			return
		}

		// Set CORS headers
		m.setCORSHeaders(w, origin)

		// Handle preflight (OPTIONS) requests
		if r.Method == http.MethodOptions {
			m.handlePreflight(w, r)
			return
		}

		// Continue with the actual request
		next.ServeHTTP(w, r)
	})
}

// isOriginAllowed checks if the given origin is in the allowed list
func (m *CORSMiddleware) isOriginAllowed(origin string) bool {
	// Wildcard allows all origins
	if m.config.Origins.Contains("*") {
		return true
	}

	// Check if origin is in the allowed list
	return m.config.Origins.Contains(origin)
}

// setCORSHeaders sets the appropriate CORS response headers
func (m *CORSMiddleware) setCORSHeaders(w http.ResponseWriter, origin string) {
	// Set allowed origin
	// Use specific origin when credentials are enabled (not "*")
	if m.config.Credentials || !m.config.Origins.Contains("*") {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}

	// Set credentials header if enabled
	if m.config.Credentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	// Expose headers to JavaScript
	if len(m.config.Expose) > 0 {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(m.config.Expose, ", "))
	}

	// Vary: Origin ensures different origins get different cached responses
	w.Header().Add("Vary", "Origin")
}

// handlePreflight handles OPTIONS preflight requests
func (m *CORSMiddleware) handlePreflight(w http.ResponseWriter, r *http.Request) {
	// Set allowed methods
	methods := m.config.Methods
	if len(methods) == 0 {
		methods = []string{"GET", "HEAD", "POST"}
	}
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ", "))

	// Set allowed headers
	if len(m.config.Headers) > 0 {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(m.config.Headers, ", "))
	} else {
		// If not configured, echo back the requested headers
		requestedHeaders := r.Header.Get("Access-Control-Request-Headers")
		if requestedHeaders != "" {
			w.Header().Set("Access-Control-Allow-Headers", requestedHeaders)
		}
	}

	// Set max age for preflight caching
	if m.config.MaxAge > 0 {
		w.Header().Set("Access-Control-Max-Age", strconv.Itoa(m.config.MaxAge))
	}

	// Return 204 No Content for preflight
	w.WriteHeader(http.StatusNoContent)
}
