package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sambeau/basil/config"
	"github.com/sambeau/parsley/pkg/ast"
	"github.com/sambeau/parsley/pkg/evaluator"
	"github.com/sambeau/parsley/pkg/lexer"
	"github.com/sambeau/parsley/pkg/parser"
	"github.com/sambeau/parsley/pkg/parsley"
)

// scriptCache caches compiled Parsley ASTs for production performance.
// In dev mode, caching is disabled and scripts are always read and parsed from disk.
type scriptCache struct {
	mu       sync.RWMutex
	programs map[string]*ast.Program // path -> compiled AST
	devMode  bool
}

func newScriptCache(devMode bool) *scriptCache {
	return &scriptCache{
		programs: make(map[string]*ast.Program),
		devMode:  devMode,
	}
}

// getAST returns the compiled AST for a script, using cache in production mode.
// Returns the AST and any parse errors.
func (c *scriptCache) getAST(path string) (*ast.Program, error) {
	// In dev mode, always read and parse from disk (no caching)
	if c.devMode {
		return c.parseScript(path)
	}

	// Production mode: check cache first
	c.mu.RLock()
	program, ok := c.programs[path]
	c.mu.RUnlock()
	if ok {
		return program, nil
	}

	// Parse from disk
	program, err := c.parseScript(path)
	if err != nil {
		return nil, err
	}

	// Cache the compiled AST
	c.mu.Lock()
	c.programs[path] = program
	c.mu.Unlock()

	return program, nil
}

// parseScript reads and parses a Parsley script file.
func (c *scriptCache) parseScript(path string) (*ast.Program, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading script %s: %w", path, err)
	}

	l := lexer.NewWithFilename(string(content), path)
	p := parser.New(l)
	program := p.ParseProgram()

	if errors := p.Errors(); len(errors) != 0 {
		return nil, fmt.Errorf("parse error in %s: %s", path, errors[0])
	}

	return program, nil
}

// clear removes all cached ASTs (for hot reload)
func (c *scriptCache) clear() {
	c.mu.Lock()
	c.programs = make(map[string]*ast.Program)
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
	// Handler path is already resolved to absolute by config.Load()
	scriptPath := route.Handler

	return &parsleyHandler{
		server:     s,
		route:      route,
		scriptPath: scriptPath,
		cache:      cache,
	}, nil
}

// ServeHTTP handles HTTP requests by executing the Parsley script
func (h *parsleyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get compiled AST (from cache in production, fresh parse in dev)
	program, err := h.cache.getAST(h.scriptPath)
	if err != nil {
		h.server.logError("failed to load script: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Build request context for the script
	reqCtx := buildRequestContext(r, h.route)

	// Create fresh environment for this request
	env := evaluator.NewEnvironment()
	env.Filename = h.scriptPath

	// Set security policy
	// Allow executing Parsley files in the script's directory (for imports)
	scriptDir := filepath.Dir(h.scriptPath)
	env.Security = &evaluator.SecurityPolicy{
		NoRead:        false,                             // Allow reads
		AllowWrite:    []string{},                        // No write access
		AllowWriteAll: false,                             // Deny all writes
		AllowExecute:  []string{scriptDir},               // Allow imports from handler directory
		RestrictRead:  []string{"/etc", "/var", "/root"}, // Basic restrictions
	}

	// Set request variables
	setEnvVar(env, "request", reqCtx)
	setEnvVar(env, "method", r.Method)
	setEnvVar(env, "path", r.URL.Path)
	setEnvVar(env, "query", queryToMap(r.URL.Query()))

	// Add database connection if configured
	if h.server.db != nil {
		conn := evaluator.NewManagedDBConnection(h.server.db, h.server.dbDriver)
		env.Set("db", conn)
	}

	// Set up custom logger that captures script log() output
	scriptLogger := &scriptLogCapture{output: make([]string, 0)}
	env.Logger = scriptLogger

	// Execute the pre-compiled AST
	result := evaluator.Eval(program, env)

	// Check for runtime errors
	if result != nil && result.Type() == evaluator.ERROR_OBJ {
		errObj := result.(*evaluator.Error)
		h.server.logError("script error in %s: %s", h.scriptPath, errObj.Message)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Log any captured script output
	for _, line := range scriptLogger.output {
		h.server.logInfo("[script] %s", line)
	}

	// Handle the response
	h.writeResponse(w, &parsley.Result{Value: result})
}

// setEnvVar converts a Go value to Parsley and sets it in the environment.
func setEnvVar(env *evaluator.Environment, name string, value interface{}) {
	obj, err := parsley.ToParsley(value)
	if err != nil {
		return // Silently ignore conversion errors
	}
	env.Set(name, obj)
}

// buildRequestContext creates the request object passed to Parsley scripts
func buildRequestContext(r *http.Request, route config.Route) map[string]interface{} {
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	ctx := map[string]interface{}{
		"method":     r.Method,
		"path":       r.URL.Path,
		"query":      queryToMap(r.URL.Query()),
		"headers":    headers,
		"host":       r.Host,
		"remoteAddr": r.RemoteAddr,
		"auth":       route.Auth, // "required", "optional", or ""
	}

	// Parse body for POST/PUT/PATCH requests
	if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
		body, form, files := parseRequestBody(r)
		ctx["body"] = body
		ctx["form"] = form
		ctx["files"] = files
	}

	return ctx
}

// parseRequestBody parses the request body based on content type
// Returns: raw body (string), form data (map), file uploads (map)
func parseRequestBody(r *http.Request) (string, map[string]interface{}, map[string]interface{}) {
	contentType := r.Header.Get("Content-Type")

	// Handle multipart form data (file uploads)
	if strings.HasPrefix(contentType, "multipart/form-data") {
		return parseMultipartForm(r)
	}

	// Handle URL-encoded form data
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		return parseURLEncodedForm(r)
	}

	// Handle JSON body
	if strings.HasPrefix(contentType, "application/json") {
		return parseJSONBody(r)
	}

	// Default: read raw body as string
	return parseRawBody(r), nil, nil
}

// parseMultipartForm handles multipart/form-data (file uploads)
func parseMultipartForm(r *http.Request) (string, map[string]interface{}, map[string]interface{}) {
	// 32MB max memory, rest goes to temp files
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return "", nil, nil
	}

	form := make(map[string]interface{})
	files := make(map[string]interface{})

	// Extract form values
	if r.MultipartForm != nil {
		for k, v := range r.MultipartForm.Value {
			if len(v) == 1 {
				form[k] = v[0]
			} else {
				form[k] = v
			}
		}

		// Extract file metadata (not the actual file contents for safety)
		for k, fileHeaders := range r.MultipartForm.File {
			fileList := make([]map[string]interface{}, 0, len(fileHeaders))
			for _, fh := range fileHeaders {
				fileList = append(fileList, map[string]interface{}{
					"filename": fh.Filename,
					"size":     fh.Size,
					"headers":  headerToMap(fh.Header),
				})
			}
			if len(fileList) == 1 {
				files[k] = fileList[0]
			} else {
				files[k] = fileList
			}
		}
	}

	return "", form, files
}

// parseURLEncodedForm handles application/x-www-form-urlencoded
func parseURLEncodedForm(r *http.Request) (string, map[string]interface{}, map[string]interface{}) {
	if err := r.ParseForm(); err != nil {
		return "", nil, nil
	}

	form := make(map[string]interface{})
	for k, v := range r.PostForm {
		if len(v) == 1 {
			form[k] = v[0]
		} else {
			form[k] = v
		}
	}

	return "", form, nil
}

// parseJSONBody handles application/json
func parseJSONBody(r *http.Request) (string, map[string]interface{}, map[string]interface{}) {
	body := parseRawBody(r)

	// Try to parse as JSON map
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(body), &data); err == nil {
		return body, data, nil
	}

	// If not a map, just return raw body
	return body, nil, nil
}

// parseRawBody reads the entire body as a string
func parseRawBody(r *http.Request) string {
	if r.Body == nil {
		return ""
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return ""
	}
	return string(body)
}

// headerToMap converts http.Header to a simple map
func headerToMap(h map[string][]string) map[string]string {
	result := make(map[string]string)
	for k, v := range h {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result
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
	data, err := json.Marshal(value)
	if err != nil {
		h.server.logError("failed to marshal JSON: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Write(data)
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
