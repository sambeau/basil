package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sambeau/basil/auth"
	"github.com/sambeau/basil/config"
	"github.com/sambeau/basil/pkg/parsley/ast"
	perrors "github.com/sambeau/basil/pkg/parsley/errors"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
	"github.com/sambeau/basil/pkg/parsley/parsley"
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

	// Use structured errors for better error display
	if errs := p.StructuredErrors(); len(errs) > 0 {
		return nil, errs[0]
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
	server            *Server
	route             config.Route
	scriptPath        string
	cache             *scriptCache
	responseCache     *responseCache
	componentExpander *auth.ComponentExpander
}

// newParsleyHandler creates a handler for a Parsley script route
func newParsleyHandler(s *Server, route config.Route, cache *scriptCache) (*parsleyHandler, error) {
	// Handler path is already resolved to absolute by config.Load()
	scriptPath := route.Handler

	return &parsleyHandler{
		server:            s,
		route:             route,
		scriptPath:        scriptPath,
		cache:             cache,
		responseCache:     s.responseCache,
		componentExpander: auth.NewComponentExpander(),
	}, nil
}

// ServeHTTP handles HTTP requests by executing the Parsley script
func (h *parsleyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check response cache first (only for cacheable routes with GET requests)
	if h.route.Cache > 0 && r.Method == http.MethodGet {
		if cached := h.responseCache.Get(r); cached != nil {
			// Serve from cache
			for k, v := range cached.headers {
				for _, vv := range v {
					w.Header().Add(k, vv)
				}
			}
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(cached.status)
			w.Write(cached.body)
			return
		}
	}

	// Get compiled AST (from cache in production, fresh parse in dev)
	program, err := h.cache.getAST(h.scriptPath)
	if err != nil {
		h.server.logError("failed to load script: %v", err)
		// Check if it's a structured parse error
		var parseErr *perrors.ParsleyError
		if errors.As(err, &parseErr) {
			h.handleParsleyError(w, r, h.scriptPath, parseErr)
		} else {
			h.handleScriptError(w, r, "parse", h.scriptPath, err.Error())
		}
		return
	}

	// Clear module cache so imports see fresh basil.* values for this request
	// Modules that access basil.http.request, basil.auth.user, etc. need current data
	evaluator.ClearModuleCache()

	// Get or generate CSRF token and set cookie if needed
	csrfToken, isNew := GetCSRFToken(r)
	if isNew && csrfToken != "" {
		SetCSRFCookie(w, csrfToken, h.server.config.Server.Dev)
	}

	// Load session (if session store is configured)
	var session *Session
	var sessionModule *evaluator.SessionModule
	if h.server.sessionStore != nil {
		sessionData, err := h.server.sessionStore.Load(r)
		if err != nil {
			h.server.logError("failed to load session: %v", err)
			// Continue without session on error
		} else {
			session = NewSession(sessionData, h.server.sessionStore, w)
			sessionModule = evaluator.NewSessionModule(
				sessionData.Data,
				sessionData.Flash,
				h.server.config.Session.MaxAge,
			)
		}
	}

	// Build request context for the script
	reqCtx := buildRequestContext(r, h.route)

	// Create fresh environment for this request
	env := evaluator.NewEnvironment()
	env.Filename = h.scriptPath

	// Set root path for @~/ imports (handler's directory)
	scriptDir := filepath.Dir(h.scriptPath)
	absScriptDir, _ := filepath.Abs(scriptDir)
	env.RootPath = absScriptDir

	// Set security policy
	// Allow executing Parsley files in the script's directory and subdirectories (for imports)
	env.Security = &evaluator.SecurityPolicy{
		NoRead:        false,                             // Allow reads
		AllowWrite:    []string{},                        // No write access
		AllowWriteAll: false,                             // Deny all writes
		AllowExecute:  []string{absScriptDir},            // Allow imports from handler directory tree
		RestrictRead:  []string{"/etc", "/var", "/root"}, // Basic restrictions
	}

	// Build and inject the basil namespace object (protected from reassignment)
	// Use route's public_dir for this handler
	basilObj := buildBasilContext(r, h.route, reqCtx, h.server.db, h.server.dbDriver, h.route.PublicDir, h.server.fragmentCache, h.route.Path, csrfToken, sessionModule)
	env.SetProtected("basil", basilObj)

	// Also set on environment for stdlib import (std/basil)
	env.BasilCtx = basilObj

	// Set fragment cache and handler path for <basil.cache.Cache> component
	env.FragmentCache = h.server.fragmentCache
	env.AssetRegistry = h.server.assetRegistry
	env.HandlerPath = h.route.Path
	env.DevMode = h.server.config.Server.Dev

	// Inject publicUrl() function for asset registration
	env.SetProtected("publicUrl", evaluator.NewPublicURLBuiltin())

	// Set dev log writer on environment (available to stdlib dev module via import)
	// nil in production mode - dev functions become no-ops
	if h.server.devLog != nil {
		env.DevLog = h.server.devLog
	}

	// Set up custom logger that captures script log() output
	scriptLogger := &scriptLogCapture{output: make([]string, 0)}
	env.Logger = scriptLogger

	// Execute the pre-compiled AST
	result := evaluator.Eval(program, env)

	// Check for runtime errors
	if result != nil && result.Type() == evaluator.ERROR_OBJ {
		errObj := result.(*evaluator.Error)
		h.server.logError("script error in %s: %s", h.scriptPath, errObj.Inspect())
		h.handleStructuredError(w, r, "runtime", h.scriptPath, errObj)
		return
	}

	// Check for redirect response
	if result != nil && result.Type() == evaluator.REDIRECT_OBJ {
		redirect := result.(*evaluator.Redirect)
		http.Redirect(w, r, redirect.URL, redirect.Status)
		return
	}

	// Log any captured script output
	for _, line := range scriptLogger.output {
		h.server.logInfo("[script] %s", line)
	}

	// Sync session changes from evaluator module back to session wrapper
	if session != nil && sessionModule != nil {
		// Copy data and flash from the module (which may have been modified by the script)
		session.data.Data = sessionModule.Data
		session.data.Flash = sessionModule.Flash
		// Mark dirty if the module was modified
		if sessionModule.Dirty {
			session.dirty = true
		}
		if sessionModule.Cleared {
			session.cleared = true
		}
		// Commit session (writes cookie if dirty)
		if err := session.Commit(); err != nil {
			h.server.logError("failed to commit session: %v", err)
		}
	}

	// Extract response metadata from basil.http.response
	responseMeta := extractResponseMeta(env, h.server.config.Server.Dev)

	// Handle the response (with caching if enabled)
	h.writeResponseWithCache(w, r, &parsley.Result{Value: result}, responseMeta)
}

// setEnvVar converts a Go value to Parsley and sets it in the environment.
func setEnvVar(env *evaluator.Environment, name string, value interface{}) {
	obj, err := parsley.ToParsley(value)
	if err != nil {
		return // Silently ignore conversion errors
	}
	env.Set(name, obj)
}

// responseMeta holds response metadata set by the script via basil.http.response
type responseMeta struct {
	status  int
	headers map[string]string
	cookies []*http.Cookie
}

// buildBasilContext creates the basil namespace object injected into Parsley scripts
// Returns a Parsley Dictionary object that can be set directly in the environment
func buildBasilContext(r *http.Request, route config.Route, reqCtx map[string]interface{}, db *sql.DB, dbDriver string, publicDir string, fragCache *fragmentCache, routePath string, csrfToken string, sessionModule *evaluator.SessionModule) evaluator.Object {
	// Build auth context
	authCtx := map[string]interface{}{
		"required": route.Auth == "required",
	}

	// Add authenticated user if present
	user := auth.GetUser(r)
	if user != nil {
		authCtx["user"] = map[string]interface{}{
			"id":      user.ID,
			"name":    user.Name,
			"email":   user.Email,
			"created": user.CreatedAt,
		}
	} else {
		authCtx["user"] = nil
	}

	// Build the basil namespace (without sqlite - that's added separately)
	basilMap := map[string]interface{}{
		"http": map[string]interface{}{
			"request": reqCtx,
			"response": map[string]interface{}{
				"status":  int64(200),
				"headers": map[string]interface{}{},
				"cookies": map[string]interface{}{},
			},
		},
		"auth":       authCtx,
		"context":    map[string]interface{}{}, // Empty dict for user-defined globals
		"public_dir": publicDir,                // Public directory for path rewriting
		"csrf": map[string]interface{}{
			"token": csrfToken,
		},
	}

	// Convert to Parsley Dictionary
	basilObj, err := parsley.ToParsley(basilMap)
	if err != nil {
		// Fallback to empty dict on error
		return &evaluator.Dictionary{Pairs: make(map[string]ast.Expression)}
	}

	basilDict := basilObj.(*evaluator.Dictionary)

	// Add database connection if configured (as a special object, not via ToParsley)
	if db != nil {
		conn := evaluator.NewManagedDBConnection(db, dbDriver)
		// Use ast.ObjectLiteralExpression to wrap the DBConnection for Dictionary storage
		basilDict.Pairs["sqlite"] = &ast.ObjectLiteralExpression{Obj: conn}
	}

	// Add session module if configured
	if sessionModule != nil {
		basilDict.Pairs["session"] = &ast.ObjectLiteralExpression{Obj: sessionModule}
	}

	// Note: Fragment caching (FEAT-037) uses <basil.cache.Cache> tag which accesses
	// the cache via env.FragmentCache. The basil.cache.invalidate() function
	// will be added in a future task.

	return basilDict
}

// extractResponseMeta reads basil.http.response from the environment after script execution
func extractResponseMeta(env *evaluator.Environment, devMode bool) *responseMeta {
	meta := &responseMeta{
		status:  200,
		headers: make(map[string]string),
		cookies: make([]*http.Cookie, 0),
	}

	// Get basil object from environment
	basilObj, ok := env.Get("basil")
	if !ok || basilObj == nil {
		return meta
	}

	// Convert to Go map using parsley's conversion
	basilMap, ok := parsley.FromParsley(basilObj).(map[string]interface{})
	if !ok {
		return meta
	}

	// Navigate to basil.http.response
	httpMap, ok := basilMap["http"].(map[string]interface{})
	if !ok {
		return meta
	}

	responseMap, ok := httpMap["response"].(map[string]interface{})
	if !ok {
		return meta
	}

	// Extract status
	if status, ok := responseMap["status"]; ok {
		switch s := status.(type) {
		case int64:
			meta.status = int(s)
		case int:
			meta.status = s
		case float64:
			meta.status = int(s)
		}
	}

	// Extract headers
	if headers, ok := responseMap["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			meta.headers[k] = fmt.Sprintf("%v", v)
		}
	}

	// Extract cookies
	if cookies, ok := responseMap["cookies"].(map[string]interface{}); ok {
		for name, value := range cookies {
			cookie := buildCookie(name, value, devMode)
			if cookie != nil {
				meta.cookies = append(meta.cookies, cookie)
			}
		}
	}

	return meta
}

// buildCookie creates an http.Cookie from a Parsley cookie value.
// The value can be a simple string (uses secure defaults) or a dict with options.
//
// Supported options:
//   - value: string (required if dict)
//   - maxAge: duration dict or int64 (seconds)
//   - expires: datetime dict
//   - path: string (default: "/")
//   - domain: string
//   - secure: bool (default: false in dev, true in prod)
//   - httpOnly: bool (default: true)
//   - sameSite: string ("Strict", "Lax", "None") (default: "Lax")
func buildCookie(name string, value interface{}, devMode bool) *http.Cookie {
	cookie := &http.Cookie{
		Name:     name,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	// Set Secure default based on mode
	if !devMode {
		cookie.Secure = true
	}

	switch v := value.(type) {
	case string:
		// Simple string value
		cookie.Value = v
	case map[string]interface{}:
		// Dict with options
		if val, ok := v["value"].(string); ok {
			cookie.Value = val
		} else if val, ok := v["value"]; ok {
			cookie.Value = fmt.Sprintf("%v", val)
		}

		// maxAge can be a duration dict or int64
		if maxAge, ok := v["maxAge"]; ok {
			cookie.MaxAge = durationToSeconds(maxAge)
		}

		// expires is a datetime dict
		if expires, ok := v["expires"].(map[string]interface{}); ok {
			if unix, ok := expires["unix"].(int64); ok {
				cookie.Expires = time.Unix(unix, 0)
			}
		}

		// path
		if path, ok := v["path"].(string); ok {
			cookie.Path = path
		}

		// domain
		if domain, ok := v["domain"].(string); ok {
			cookie.Domain = domain
		}

		// secure
		if secure, ok := v["secure"].(bool); ok {
			cookie.Secure = secure
		}

		// httpOnly
		if httpOnly, ok := v["httpOnly"].(bool); ok {
			cookie.HttpOnly = httpOnly
		}

		// sameSite
		if sameSite, ok := v["sameSite"].(string); ok {
			switch strings.ToLower(sameSite) {
			case "strict":
				cookie.SameSite = http.SameSiteStrictMode
			case "lax":
				cookie.SameSite = http.SameSiteLaxMode
			case "none":
				cookie.SameSite = http.SameSiteNoneMode
				// SameSite=None requires Secure=true
				cookie.Secure = true
			}
		}
	default:
		// Unknown type, convert to string
		cookie.Value = fmt.Sprintf("%v", value)
	}

	return cookie
}

// durationToSeconds converts a Parsley duration value to seconds.
// Accepts duration dicts (with months/seconds or totalSeconds) or int64.
func durationToSeconds(value interface{}) int {
	switch v := value.(type) {
	case int64:
		return int(v)
	case int:
		return v
	case float64:
		return int(v)
	case map[string]interface{}:
		// Parsley duration dict - check for totalSeconds first
		if totalSeconds, ok := v["totalSeconds"].(int64); ok {
			return int(totalSeconds)
		}
		// Fall back to seconds field (for simple durations)
		if seconds, ok := v["seconds"].(int64); ok {
			// If months are present, we can't accurately convert to seconds
			// (months have variable length). Use an approximation of 30 days.
			if months, ok := v["months"].(int64); ok && months > 0 {
				return int(months*30*24*60*60 + seconds)
			}
			return int(seconds)
		}
	}
	return 0
}

// buildRequestContext creates the request object passed to Parsley scripts
func buildRequestContext(r *http.Request, route config.Route) map[string]interface{} {
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	// Parse cookies into a simple name→value map
	cookies := make(map[string]interface{})
	for _, c := range r.Cookies() {
		cookies[c.Name] = c.Value
	}

	ctx := map[string]interface{}{
		"method":     r.Method,
		"path":       r.URL.Path,
		"query":      queryToMap(r.URL.Query()),
		"headers":    headers,
		"cookies":    cookies,
		"host":       r.Host,
		"remoteAddr": r.RemoteAddr,
	}

	// Add subpath if in site routing mode
	// subpath is set by siteHandler via context when using filesystem-based routing
	if subpath := getSubpath(r.Context()); subpath != "" || r.Context().Value(subpathContextKey{}) != nil {
		// Convert subpath to Path object format for Parsley
		ctx["subpath"] = buildSubpathObject(subpath)
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

// buildSubpathObject creates a Path object for the subpath in site routing.
// The subpath is the portion of the URL path not consumed by the matched handler.
// Returns a map that will be converted to a Parsley Path dictionary via ToParsley.
func buildSubpathObject(subpath string) map[string]interface{} {
	// Parse segments from subpath (e.g., "/2025/Q4" -> ["2025", "Q4"])
	segments := []interface{}{}
	if subpath != "" && subpath != "/" {
		parts := strings.Split(strings.Trim(subpath, "/"), "/")
		for _, part := range parts {
			if part != "" {
				segments = append(segments, part)
			}
		}
	}

	return map[string]interface{}{
		"__type":   "path",
		"absolute": false, // Subpath is always relative
		"segments": segments,
	}
}

// writeResponseWithCache writes the response and caches it if the route has caching enabled.
func (h *parsleyHandler) writeResponseWithCache(w http.ResponseWriter, r *http.Request, result *parsley.Result, meta *responseMeta) {
	// If caching is enabled for this route and it's a GET request, capture the response
	if h.route.Cache > 0 && r.Method == http.MethodGet {
		crw := newCachedResponseWriter(w)
		crw.Header().Set("X-Cache", "MISS")
		h.writeResponse(crw, r, result, meta)

		// Only cache successful responses (2xx)
		if crw.statusCode >= 200 && crw.statusCode < 300 {
			h.responseCache.Set(r, h.route.Cache, crw.statusCode, crw.Header(), crw.body)
		}
		return
	}

	// No caching, write directly
	h.writeResponse(w, r, result, meta)
}

// writeResponse writes the Parsley result to the HTTP response
func (h *parsleyHandler) writeResponse(w http.ResponseWriter, r *http.Request, result *parsley.Result, meta *responseMeta) {
	// Apply response headers from basil.http.response.headers
	for k, v := range meta.headers {
		w.Header().Set(k, v)
	}

	// Apply response cookies from basil.http.response.cookies
	for _, cookie := range meta.cookies {
		http.SetCookie(w, cookie)
	}

	// Determine if we need a custom status code
	customStatus := meta.status != 200

	if result == nil || result.IsNull() {
		if customStatus {
			w.WriteHeader(meta.status)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
		return
	}

	// Get the Go value
	value := result.GoValue()

	// Handle different response types
	switch v := value.(type) {
	case string:
		// Plain text or HTML (detect by content)
		contentType := "text/plain; charset=utf-8"
		output := v
		if strings.HasPrefix(strings.TrimSpace(v), "<") {
			contentType = "text/html; charset=utf-8"
			// Expand auth components in HTML output
			output = h.componentExpander.ExpandComponents(v)
		}
		w.Header().Set("Content-Type", contentType)
		if customStatus {
			w.WriteHeader(meta.status)
		}
		fmt.Fprint(w, output)

	case map[string]interface{}:
		// Check for special response object format (legacy support)
		if status, ok := v["status"].(int64); ok {
			w.WriteHeader(int(status))
			customStatus = false // Already written
		} else if customStatus {
			w.WriteHeader(meta.status)
			customStatus = false
		}
		if headers, ok := v["headers"].(map[string]interface{}); ok {
			for k, hv := range headers {
				w.Header().Set(k, fmt.Sprintf("%v", hv))
			}
		}
		if body, ok := v["body"]; ok {
			switch b := body.(type) {
			case string:
				// Expand auth components if it looks like HTML
				output := b
				if strings.HasPrefix(strings.TrimSpace(b), "<") {
					output = h.componentExpander.ExpandComponents(b)
				}
				fmt.Fprint(w, output)
			default:
				// JSON encode other body types
				h.writeJSON(w, r, b)
			}
		} else {
			// No body field, encode the whole map as JSON
			h.writeJSON(w, r, v)
		}

	default:
		// Encode as JSON
		if customStatus {
			w.WriteHeader(meta.status)
		}
		h.writeJSON(w, r, value)
	}
}

// writeJSON writes a JSON response
func (h *parsleyHandler) writeJSON(w http.ResponseWriter, r *http.Request, value interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	data, err := json.Marshal(value)
	if err != nil {
		h.server.logError("failed to marshal JSON: %v", err)
		h.server.handle500(w, r, err)
		return
	}
	w.Write(data)
}

// handleScriptError handles errors during script execution.
// In dev mode, it renders a detailed error page. In production, it returns a generic 500.
func (h *parsleyHandler) handleScriptError(w http.ResponseWriter, r *http.Request, errType, filePath, message string) {
	// In production mode, always return generic error
	if !h.server.config.Server.Dev {
		h.server.handle500(w, r, fmt.Errorf("%s", message))
		return
	}

	// In dev mode, render detailed error page
	// Try to extract line info from the error message
	extractedFile, line, col, cleanMsg := extractLineInfo(message)

	// Use extracted file if we found one, otherwise use the handler file
	if extractedFile != "" {
		filePath = extractedFile
	}

	// If no clean message was extracted, use the original
	if cleanMsg == "" {
		cleanMsg = message
	}

	// Get base path for making paths relative (directory of config file)
	basePath := filepath.Dir(h.server.configPath)

	devErr := &DevError{
		Type:     errType,
		File:     filePath,
		Line:     line,
		Column:   col,
		Message:  cleanMsg,
		BasePath: basePath,
	}

	renderDevErrorPage(w, devErr)
}

// handleParsleyError handles structured ParsleyError from the parser.
// This provides clean error display without regex parsing of error messages.
func (h *parsleyHandler) handleParsleyError(w http.ResponseWriter, r *http.Request, filePath string, parseErr *perrors.ParsleyError) {
	if !h.server.config.Server.Dev {
		h.server.handle500(w, r, fmt.Errorf("parse error: %s", parseErr.Message))
		return
	}

	// Get base path for making paths relative (directory of config file)
	basePath := filepath.Dir(h.server.configPath)

	// Use file from error if available, otherwise use handler file
	file := parseErr.File
	if file == "" {
		file = filePath
	}

	devErr := FromParsleyError(parseErr, basePath)
	devErr.File = file // Ensure file is set if it was missing

	renderDevErrorPage(w, devErr)
}

// handleStructuredError handles errors with structured error information from Parsley.
func (h *parsleyHandler) handleStructuredError(w http.ResponseWriter, r *http.Request, errType, filePath string, errObj *evaluator.Error) {
	if !h.server.config.Server.Dev {
		h.server.handle500(w, r, fmt.Errorf("%s", errObj.Message))
		return
	}

	// Get base path for making paths relative (directory of config file)
	basePath := filepath.Dir(h.server.configPath)

	// Use file from error if available, otherwise use handler file
	file := errObj.File
	if file == "" {
		file = filePath
	}

	// Determine error type from class if available
	if errObj.Class == evaluator.ClassParse {
		errType = "parse"
	}

	devErr := &DevError{
		Type:     errType,
		File:     file,
		Line:     errObj.Line,
		Column:   errObj.Column,
		Message:  errObj.Message,
		Hints:    errObj.Hints,
		BasePath: basePath,
	}

	renderDevErrorPage(w, devErr)
}

// handleScriptErrorWithLocation handles errors with explicit line/column info from Parsley.
func (h *parsleyHandler) handleScriptErrorWithLocation(w http.ResponseWriter, r *http.Request, errType, filePath, message string, line, col int) {
	if !h.server.config.Server.Dev {
		h.server.handle500(w, r, fmt.Errorf("%s", message))
		return
	}

	// Try to extract more specific location from the error message
	// This handles cases like import errors where the message contains
	// the actual file/line of the error (e.g., "parse errors in module ./path.pars:")
	extractedFile, extractedLine, extractedCol, cleanMsg := extractLineInfo(message)
	if extractedFile != "" {
		filePath = extractedFile
		// If we extracted a file from a "parse errors in module" message,
		// this is really a parse error, not a runtime error
		if strings.Contains(message, "parse error") {
			errType = "parse"
		}
	}
	if extractedLine > 0 {
		line = extractedLine
	}
	if extractedCol > 0 {
		col = extractedCol
	}
	if cleanMsg != "" && cleanMsg != message {
		message = cleanMsg
	}

	// Get base path for making paths relative (directory of config file)
	basePath := filepath.Dir(h.server.configPath)

	devErr := &DevError{
		Type:     errType,
		File:     filePath,
		Line:     line,
		Column:   col,
		Message:  message,
		BasePath: basePath,
	}

	renderDevErrorPage(w, devErr)
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

// logWarn logs a warning message
func (s *Server) logWarn(format string, args ...interface{}) {
	fmt.Fprintf(s.stderr, "[WARN] "+format+"\n", args...)
}

// logError logs an error message
func (s *Server) logError(format string, args ...interface{}) {
	fmt.Fprintf(s.stderr, "[ERROR] "+format+"\n", args...)
}
