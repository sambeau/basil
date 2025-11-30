package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sambeau/basil/config"
	"github.com/sambeau/parsley/pkg/evaluator"
	"github.com/sambeau/parsley/pkg/parsley"
)

// scriptCache caches loaded Parsley scripts
type scriptCache struct {
	mu      sync.RWMutex
	scripts map[string]string // path -> source code
}

func newScriptCache() *scriptCache {
	return &scriptCache{
		scripts: make(map[string]string),
	}
}

// get returns cached script source, loading from disk if needed
func (c *scriptCache) get(path string) (string, error) {
	c.mu.RLock()
	source, ok := c.scripts[path]
	c.mu.RUnlock()
	if ok {
		return source, nil
	}

	// Load from disk
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading script %s: %w", path, err)
	}

	source = string(content)
	c.mu.Lock()
	c.scripts[path] = source
	c.mu.Unlock()

	return source, nil
}

// clear removes all cached scripts (for hot reload)
func (c *scriptCache) clear() {
	c.mu.Lock()
	c.scripts = make(map[string]string)
	c.mu.Unlock()
}

// parsleyHandler handles HTTP requests with Parsley scripts
type parsleyHandler struct {
	server     *Server
	route      config.Route
	scriptPath string
	cache      *scriptCache
}

// newParsleyHandler creates a handler for a Parsley script route
func newParsleyHandler(s *Server, route config.Route, cache *scriptCache) (*parsleyHandler, error) {
	// Resolve script path (relative to config file or absolute)
	scriptPath := route.Handler
	if !filepath.IsAbs(scriptPath) {
		// TODO: Make relative to config file location
		absPath, err := filepath.Abs(scriptPath)
		if err != nil {
			return nil, fmt.Errorf("resolving handler path: %w", err)
		}
		scriptPath = absPath
	}

	return &parsleyHandler{
		server:     s,
		route:      route,
		scriptPath: scriptPath,
		cache:      cache,
	}, nil
}

// ServeHTTP handles HTTP requests by executing the Parsley script
func (h *parsleyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Load script (from cache or disk)
	source, err := h.cache.get(h.scriptPath)
	if err != nil {
		h.server.logError("failed to load script: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Build request context for the script
	reqCtx := buildRequestContext(r, h.route)

	// Create security policy
	// By default: allow reading from script directory, no writing, no executing
	policy := &evaluator.SecurityPolicy{
		NoRead:        false,                             // Allow reads
		AllowWrite:    []string{},                        // No write access
		AllowWriteAll: false,                             // Deny all writes
		AllowExecute:  []string{},                        // No execute access
		RestrictRead:  []string{"/etc", "/var", "/root"}, // Basic restrictions
	}

	// Build evaluation options
	opts := []parsley.Option{
		parsley.WithFilename(h.scriptPath),
		parsley.WithSecurity(policy),
		parsley.WithVar("request", reqCtx),
		parsley.WithVar("method", r.Method),
		parsley.WithVar("path", r.URL.Path),
		parsley.WithVar("query", queryToMap(r.URL.Query())),
	}

	// Add custom logger that captures script log() output
	scriptLogger := &scriptLogCapture{output: make([]string, 0)}
	opts = append(opts, parsley.WithLogger(scriptLogger))

	// Execute the script
	result, err := parsley.Eval(source, opts...)
	if err != nil {
		h.server.logError("script error in %s: %v", h.scriptPath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Log any captured script output
	for _, line := range scriptLogger.output {
		h.server.logInfo("[script] %s", line)
	}

	// Handle the response
	h.writeResponse(w, result)
}

// buildRequestContext creates the request object passed to Parsley scripts
func buildRequestContext(r *http.Request, route config.Route) map[string]interface{} {
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	return map[string]interface{}{
		"method":     r.Method,
		"path":       r.URL.Path,
		"query":      queryToMap(r.URL.Query()),
		"headers":    headers,
		"host":       r.Host,
		"remoteAddr": r.RemoteAddr,
		"auth":       route.Auth, // "required", "optional", or ""
	}
}

// queryToMap converts URL query parameters to a map
func queryToMap(query map[string][]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range query {
		if len(v) == 1 {
			result[k] = v[0]
		} else {
			result[k] = v
		}
	}
	return result
}

// writeResponse writes the Parsley result to the HTTP response
func (h *parsleyHandler) writeResponse(w http.ResponseWriter, result *parsley.Result) {
	if result == nil || result.IsNull() {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Get the Go value
	value := result.GoValue()

	// Handle different response types
	switch v := value.(type) {
	case string:
		// Plain text or HTML (detect by content)
		contentType := "text/plain; charset=utf-8"
		if strings.HasPrefix(strings.TrimSpace(v), "<") {
			contentType = "text/html; charset=utf-8"
		}
		w.Header().Set("Content-Type", contentType)
		fmt.Fprint(w, v)

	case map[string]interface{}:
		// Check for special response object format
		if status, ok := v["status"].(int64); ok {
			w.WriteHeader(int(status))
		}
		if headers, ok := v["headers"].(map[string]interface{}); ok {
			for k, hv := range headers {
				w.Header().Set(k, fmt.Sprintf("%v", hv))
			}
		}
		if body, ok := v["body"]; ok {
			switch b := body.(type) {
			case string:
				fmt.Fprint(w, b)
			default:
				// JSON encode other body types
				h.writeJSON(w, b)
			}
		} else {
			// No body field, encode the whole map as JSON
			h.writeJSON(w, v)
		}

	default:
		// Encode as JSON
		h.writeJSON(w, value)
	}
}

// writeJSON writes a JSON response
func (h *parsleyHandler) writeJSON(w http.ResponseWriter, value interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// Simple JSON encoding (in production, use encoding/json)
	fmt.Fprintf(w, "%v", value)
}

// scriptLogCapture captures log() output from Parsley scripts
type scriptLogCapture struct {
	mu     sync.Mutex
	output []string
}

func (l *scriptLogCapture) Log(args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = append(l.output, fmt.Sprint(args...))
}

func (l *scriptLogCapture) LogLine(args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = append(l.output, fmt.Sprintln(args...))
}

// logInfo logs an info message
func (s *Server) logInfo(format string, args ...interface{}) {
	fmt.Fprintf(s.stdout, "[INFO] "+format+"\n", args...)
}

// logError logs an error message
func (s *Server) logError(format string, args ...interface{}) {
	fmt.Fprintf(s.stderr, "[ERROR] "+format+"\n", args...)
}
