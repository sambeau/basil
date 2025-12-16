package server

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sambeau/basil/auth"
	"github.com/sambeau/basil/config"
	"golang.org/x/crypto/acme/autocert"

	// SQLite driver (pure Go, no CGO required)
	_ "modernc.org/sqlite"
)

// serveStaticFile serves a file with appropriate caching headers for dev/production.
// In dev mode, disables caching to ensure fresh content on every request.
// In production mode, uses http.ServeFile's built-in ETag support.
func serveStaticFile(w http.ResponseWriter, r *http.Request, filePath string, devMode bool) {
	if devMode {
		// Dev mode: disable all caching to prevent stale content issues
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}
	http.ServeFile(w, r, filePath)
}

// Server represents a Basil web server instance.
type Server struct {
	config        *config.Config
	configPath    string
	version       string
	stdout        io.Writer
	stderr        io.Writer
	mux           *http.ServeMux
	server        *http.Server
	scriptCache   *scriptCache
	responseCache *responseCache
	fragmentCache *fragmentCache
	assetRegistry *assetRegistry
	assetBundle   *AssetBundle
	watcher       *Watcher
	db            *sql.DB // Database connection (nil if not configured)
	dbDriver      string  // Database driver name ("sqlite", etc.)
	rateLimiter   *rateLimiter

	// Session store (cookie-based by default)
	sessionStore  SessionStore
	sessionSecret string

	// Dev tools (nil if not in dev mode)
	devLog *DevLog

	// Auth system (nil if auth not enabled)
	authDB       *auth.DB
	authWebAuthn *auth.WebAuthnManager
	authHandlers *auth.Handlers
	authMW       *auth.Middleware

	// CSRF middleware
	csrfMW *CSRFMiddleware

	// CORS middleware
	corsMW *CORSMiddleware

	// Git server (nil if git not enabled)
	gitHandler *GitHandler
}

// New creates a new Basil server with the given configuration.
func New(cfg *config.Config, configPath string, version, commit string, stdout, stderr io.Writer) (*Server, error) {
	s := &Server{
		config:        cfg,
		configPath:    configPath,
		version:       version,
		stdout:        stdout,
		stderr:        stderr,
		mux:           http.NewServeMux(),
		scriptCache:   newScriptCache(cfg.Server.Dev),
		responseCache: newResponseCache(cfg.Server.Dev),
		fragmentCache: newFragmentCache(cfg.Server.Dev, 1000),
		rateLimiter:   newRateLimiter(60, time.Minute),
		csrfMW:        NewCSRFMiddleware(cfg.Server.Dev),
	}

	// Initialize prelude (embedded assets and Parsley files)
	if err := initPrelude(commit); err != nil {
		return nil, fmt.Errorf("initializing prelude: %w", err)
	}

	// Initialize CORS middleware if configured
	if len(cfg.CORS.Origins) > 0 {
		s.corsMW = NewCORSMiddleware(cfg.CORS)
	}

	// Initialize asset registry (logger for warnings, nil for production silent mode)
	if cfg.Server.Dev {
		s.assetRegistry = newAssetRegistry(s.logWarn)
	} else {
		s.assetRegistry = newAssetRegistry(nil)
	}

	// Initialize asset bundle (CSS/JS auto-bundling)
	if err := s.initAssetBundle(); err != nil {
		return nil, fmt.Errorf("initializing asset bundle: %w", err)
	}

	// Initialize session store
	if err := s.initSessions(); err != nil {
		return nil, fmt.Errorf("initializing sessions: %w", err)
	}

	// Initialize dev tools in dev mode
	if err := s.initDevTools(); err != nil {
		return nil, fmt.Errorf("initializing dev tools: %w", err)
	}

	// Initialize database connection if configured
	if err := s.initDatabase(); err != nil {
		s.cleanupDevTools()
		return nil, fmt.Errorf("initializing database: %w", err)
	}

	// Initialize auth system if enabled
	if err := s.initAuth(); err != nil {
		// Clean up database on auth init failure
		if s.db != nil {
			s.db.Close()
		}
		s.cleanupDevTools()
		return nil, fmt.Errorf("initializing auth: %w", err)
	}

	// Initialize Git server if enabled
	if err := s.initGit(); err != nil {
		if s.authDB != nil {
			s.authDB.Close()
		}
		if s.db != nil {
			s.db.Close()
		}
		s.cleanupDevTools()
		return nil, fmt.Errorf("initializing git server: %w", err)
	}

	// Set up routes
	if err := s.setupRoutes(); err != nil {
		// Clean up on route setup failure
		if s.authDB != nil {
			s.authDB.Close()
		}
		if s.db != nil {
			s.db.Close()
		}
		s.cleanupDevTools()
		return nil, fmt.Errorf("setting up routes: %w", err)
	}

	return s, nil
}

// initDevTools initializes dev tools (logging, etc.) in dev mode.
func (s *Server) initDevTools() error {
	if !s.config.Server.Dev {
		return nil
	}

	// Create dev log database with config overrides
	cfg := DefaultDevLogConfig()

	// Apply config overrides
	if s.config.Dev.LogDatabase != "" {
		cfg.Path = s.config.Dev.LogDatabase
	}
	if s.config.Dev.LogMaxSize != "" {
		size, err := config.ParseSize(s.config.Dev.LogMaxSize)
		if err != nil {
			return fmt.Errorf("parsing dev.log_max_size: %w", err)
		}
		if size > 0 {
			cfg.MaxSize = size
		}
	}
	if s.config.Dev.LogTruncatePct > 0 {
		cfg.TruncatePct = s.config.Dev.LogTruncatePct
	}

	// Use a temp directory if the base directory doesn't exist (e.g., in tests)
	baseDir := s.config.BaseDir
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		baseDir = os.TempDir()
	}

	devLog, err := NewDevLog(baseDir, cfg)
	if err != nil {
		return fmt.Errorf("creating dev log: %w", err)
	}

	s.devLog = devLog
	s.logInfo("dev tools enabled, logs at: %s", devLog.Path())
	return nil
}

// cleanupDevTools closes dev tools resources.
func (s *Server) cleanupDevTools() {
	if s.devLog != nil {
		s.devLog.Close()
		s.devLog = nil
	}
}

// initAssetBundle initializes the CSS/JS asset bundle.
func (s *Server) initAssetBundle() error {
	// Determine handlers directory from routes or site config
	handlersDir := s.determineHandlersDir()
	publicDirName := filepath.Base(s.config.PublicDir)
	if handlersDir == "" {
		// No routes configured, create empty bundle
		s.assetBundle = NewAssetBundle("", s.config.Server.Dev, publicDirName)
		return nil
	}

	s.assetBundle = NewAssetBundle(handlersDir, s.config.Server.Dev, publicDirName)
	if err := s.assetBundle.Rebuild(); err != nil {
		// Log warning but don't fail - bundle just won't have content
		s.logWarn("failed to build asset bundle: %v", err)
	}

	return nil
}

// determineHandlersDir finds the handler root directory for asset bundle discovery.
// In site mode, this is the parent of the site/ directory (the handler root).
// In route mode, this is the common ancestor of all handler files.
func (s *Server) determineHandlersDir() string {
	// If using site (filesystem routing), use the parent of the site directory
	// This allows discovering CSS/JS in components/, public/, etc. at handler root level
	if s.config.Site != "" {
		dir := filepath.Dir(s.config.Site)
		// Resolve symlinks to ensure WalkDir can traverse the actual directory
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			return resolved
		}
		return dir
	}

	// Otherwise, find common parent of all route handlers
	if len(s.config.Routes) == 0 {
		return ""
	}

	// Get directory of first handler
	commonDir := filepath.Dir(s.config.Routes[0].Handler)

	// Find common ancestor with all other handlers
	for _, route := range s.config.Routes[1:] {
		handlerDir := filepath.Dir(route.Handler)
		commonDir = commonAncestor(commonDir, handlerDir)
	}

	return commonDir
}

// commonAncestor returns the common ancestor directory of two paths.
func commonAncestor(path1, path2 string) string {
	// Clean paths
	path1 = filepath.Clean(path1)
	path2 = filepath.Clean(path2)

	// Split into components
	parts1 := strings.Split(path1, string(filepath.Separator))
	parts2 := strings.Split(path2, string(filepath.Separator))

	// Find common prefix
	var common []string
	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		if parts1[i] == parts2[i] {
			common = append(common, parts1[i])
		} else {
			break
		}
	}

	if len(common) == 0 {
		return ""
	}

	result := filepath.Join(common...)

	// Fix: If original paths were absolute (Unix), filepath.Join loses the leading /
	// because it joins ["", "Users", ...] → "Users/..." instead of "/Users/..."
	if len(common) > 0 && common[0] == "" && !filepath.IsAbs(result) {
		result = string(filepath.Separator) + result
	}

	return result
}

// Close closes all server resources. Use this in tests; in production, use Run() with a context.
func (s *Server) Close() {
	s.cleanupDevTools()
	if s.authDB != nil {
		s.authDB.Close()
	}
	if s.db != nil {
		s.db.Close()
	}
}

// initSessions initializes the session store.
func (s *Server) initSessions() error {
	cfg := &s.config.Session

	// Determine session secret
	secret := cfg.Secret
	if secret == "" {
		if s.config.Server.Dev {
			// In dev mode, generate a random secret (sessions won't persist across restarts)
			var err error
			secret, err = generateRandomSecret()
			if err != nil {
				return fmt.Errorf("generating dev session secret: %w", err)
			}
			s.logInfo("sessions: using auto-generated secret (dev mode)")
		} else {
			// In production, require explicit secret
			s.logWarn("sessions: no secret configured, sessions disabled")
			return nil
		}
	}

	s.sessionSecret = secret

	// Create cookie session store (default and currently only supported store)
	s.sessionStore = NewCookieSessionStore(cfg, secret)

	s.logInfo("sessions: cookie store initialized (max_age=%s)", cfg.MaxAge)
	return nil
}

// initDatabase opens the SQLite database connection if configured.
func (s *Server) initDatabase() error {
	// No database configured
	if s.config.SQLite == "" {
		return nil
	}

	return s.initSQLite(s.config.SQLite)
}

// initSQLite opens a SQLite database connection.
func (s *Server) initSQLite(path string) error {
	if path == "" {
		return fmt.Errorf("sqlite path is empty")
	}

	// Path should already be resolved by config loader, but handle relative just in case
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.config.BaseDir, path)
	}

	// Open database with WAL mode for better concurrency
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return fmt.Errorf("opening sqlite database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("connecting to sqlite database: %w", err)
	}

	// Configure connection pool for SQLite
	// SQLite works best with a single writer, but can handle multiple readers
	db.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0) // Keep connection open indefinitely

	s.db = db
	s.dbDriver = "sqlite"

	s.logInfo("connected to SQLite database: %s", path)
	return nil
}

// initAuth initializes the authentication system if enabled.
func (s *Server) initAuth() error {
	if !s.config.Auth.Enabled {
		return nil
	}

	// Open auth database (separate from app database)
	authDB, err := auth.OpenDB(s.config.BaseDir)
	if err != nil {
		return fmt.Errorf("opening auth database: %w", err)
	}
	s.authDB = authDB

	// Determine RP ID and origin from server config
	rpID := s.config.Server.Host
	if rpID == "" {
		rpID = "localhost"
	}

	// Build origin
	var origin string
	if s.config.Server.Dev {
		origin = fmt.Sprintf("http://localhost:%d", s.config.Server.Port)
	} else if s.config.Server.HTTPS.Auto || s.config.Server.HTTPS.Cert != "" {
		origin = fmt.Sprintf("https://%s", rpID)
		if s.config.Server.Port != 443 {
			origin = fmt.Sprintf("%s:%d", origin, s.config.Server.Port)
		}
	} else {
		origin = fmt.Sprintf("http://%s:%d", rpID, s.config.Server.Port)
	}

	// Initialize WebAuthn
	webauthn, err := auth.NewWebAuthnManager(authDB, rpID, origin, rpID)
	if err != nil {
		authDB.Close()
		return fmt.Errorf("initializing webauthn: %w", err)
	}
	s.authWebAuthn = webauthn

	// Create handlers and middleware
	sessionTTL := s.config.Auth.SessionTTL
	if sessionTTL == 0 {
		sessionTTL = 24 * time.Hour
	}
	secure := !s.config.Server.Dev // Secure cookies in production
	regOpen := s.config.Auth.Registration == "open"

	s.authHandlers = auth.NewHandlers(authDB, webauthn, sessionTTL, secure, regOpen)
	s.authMW = auth.NewMiddleware(authDB)

	s.logInfo("authentication enabled (registration: %s)", s.config.Auth.Registration)
	return nil
}

// initGit initializes the Git HTTP server if enabled.
func (s *Server) initGit() error {
	if !s.config.Git.Enabled {
		return nil
	}

	// Security warnings
	if !s.config.Git.RequireAuth && !s.config.Server.Dev {
		s.logWarn("git server is enabled without authentication - this is insecure!")
	}
	if s.config.Git.RequireAuth && s.authDB == nil {
		return fmt.Errorf("git server requires auth but auth is not enabled - enable auth.enabled or set git.require_auth: false")
	}

	// Git handler needs the site directory (where .git repo is)
	siteDir := s.config.BaseDir

	// Create reload callback
	onPush := func() {
		s.logInfo("git push received, reloading handlers...")
		s.scriptCache.clear()
		s.responseCache.Clear()
		s.fragmentCache.Clear()
	}

	gitHandler, err := NewGitHandler(siteDir, s.authDB, s.config, onPush, s.stdout, s.stderr)
	if err != nil {
		return fmt.Errorf("creating git handler: %w", err)
	}

	s.gitHandler = gitHandler
	s.logInfo("git server enabled at /.git/")
	return nil
}

// setupRoutes configures the HTTP mux with static and dynamic routes.
func (s *Server) setupRoutes() error {
	// Register asset handler for publicUrl() files at /__p/
	s.mux.Handle("/__p/", newAssetHandler(s.assetRegistry, s.devLog != nil))

	// Register asset bundle routes
	s.mux.HandleFunc("/__site.css", func(w http.ResponseWriter, r *http.Request) {
		s.assetBundle.ServeCSS(w, r)
	})
	s.mux.HandleFunc("/__site.js", func(w http.ResponseWriter, r *http.Request) {
		s.assetBundle.ServeJS(w, r)
	})

	// Register prelude asset handlers
	s.mux.HandleFunc("/__/js/", s.handlePreludeAsset)
	s.mux.HandleFunc("/__/css/", s.handlePreludeAsset)
	s.mux.HandleFunc("/__/public/", s.handlePreludeAsset)

	// In dev mode, add dev tools endpoints
	if s.config.Server.Dev {
		s.mux.Handle("/__livereload", newLiveReloadHandler(s))
		// Dev tools handler for /__/* routes (logs, etc.)
		devTools := newDevToolsHandler(s)
		s.mux.Handle("/__/", devTools)
		s.mux.Handle("/__", devTools)
	}

	// Register auth endpoints if auth is enabled
	if s.authHandlers != nil {
		s.mux.HandleFunc("/__auth/register/begin", s.authHandlers.BeginRegisterHandler)
		s.mux.HandleFunc("/__auth/register/finish", s.authHandlers.FinishRegisterHandler)
		s.mux.HandleFunc("/__auth/login/begin", s.authHandlers.BeginLoginHandler)
		s.mux.HandleFunc("/__auth/login/finish", s.authHandlers.FinishLoginHandler)
		s.mux.HandleFunc("/__auth/logout", s.authHandlers.LogoutHandler)
		s.mux.HandleFunc("/__auth/recover", s.authHandlers.RecoverHandler)
		s.mux.HandleFunc("/__auth/me", s.authHandlers.MeHandler)
	}

	// Register Git server if enabled
	if s.gitHandler != nil {
		s.mux.Handle("/.git/", s.gitHandler)
	}

	// Register explicit static routes (non-root paths like /favicon.ico)
	for _, static := range s.config.Static {
		if static.Path != "/" {
			if static.Root != "" {
				handler := http.StripPrefix(static.Path, http.FileServer(http.Dir(static.Root)))
				s.mux.Handle(static.Path, handler)
			} else if static.File != "" {
				filePath := static.File
				devMode := s.devLog != nil
				s.mux.HandleFunc(static.Path, func(w http.ResponseWriter, r *http.Request) {
					serveStaticFile(w, r, filePath, devMode)
				})
			}
		}
	}

	// Site mode: use filesystem-based routing
	if s.config.Site != "" {
		s.mux.Handle("/", newSiteHandler(s, s.config.Site, s.scriptCache))
		s.logInfo("site mode enabled at %s", s.config.Site)
		return nil
	}

	// Routes mode: explicit route-based routing
	// Register Parsley routes (specific paths)
	for _, route := range s.config.Routes {
		if route.Path == "/" {
			continue // Handle root separately as fallback
		}

		isAPI := isAPIRoute(route)
		var handler http.Handler
		var err error

		if isAPI {
			handler, err = newAPIHandler(s, route, s.scriptCache)
		} else {
			handler, err = newParsleyHandler(s, route, s.scriptCache)
		}
		if err != nil {
			return fmt.Errorf("creating handler for %s: %w", route.Path, err)
		}

		authMode := route.Auth
		if isAPI && authMode == "" {
			// For API routes, always run OptionalAuth to populate context without forcing login;
			// handler-level wrappers will enforce.
			authMode = "optional"
		}

		finalHandler := s.applyAuthMiddleware(handler, authMode)

		// Apply CSRF middleware for non-API routes with auth
		// API routes use API keys/bearer tokens, not cookies, so CSRF doesn't apply
		if !isAPI && (authMode == "required" || authMode == "optional") {
			finalHandler = s.csrfMW.Validate(finalHandler)
		}

		// If route has public_dir, wrap with static file fallback
		if route.PublicDir != "" {
			finalHandler = s.createRouteWithStaticFallback(route, finalHandler)
		}

		s.mux.Handle(route.Path, finalHandler)
		// For API routes, also register with trailing slash to handle sub-paths (e.g., /api/todos/123)
		if isAPI && !strings.HasSuffix(route.Path, "/") {
			s.mux.Handle(route.Path+"/", finalHandler)
		}
	}

	// Create fallback handler for "/" that serves:
	// 1. Static files from public_dir (if file exists)
	// 2. Root route handler (if configured)
	// 3. 404
	s.mux.Handle("/", s.createRootHandler())

	return nil
}

// isAPIRoute determines whether the route should be handled as an API module.
func isAPIRoute(route config.Route) bool {
	if strings.EqualFold(route.Type, "api") {
		return true
	}

	path := strings.TrimSuffix(route.Path, "/")
	if path == "" {
		path = "/"
	}

	if path == "/api" || strings.HasPrefix(path, "/api/") {
		return true
	}

	return false
}

// applyAuthMiddleware wraps a handler with appropriate auth middleware
func (s *Server) applyAuthMiddleware(handler http.Handler, authMode string) http.Handler {
	if s.authMW == nil {
		return handler
	}
	switch authMode {
	case "required":
		return s.authMW.RequireAuth(handler)
	case "optional":
		return s.authMW.OptionalAuth(handler)
	default:
		return s.authMW.OptionalAuth(handler)
	}
}

// createRouteWithStaticFallback wraps a route handler with static file fallback.
// For a route like /admin with public_dir ./admin/public:
// - /admin/styles.css will try the handler, then ./admin/public/styles.css
func (s *Server) createRouteWithStaticFallback(route config.Route, handler http.Handler) http.Handler {
	routePath := strings.TrimSuffix(route.Path, "/")
	staticRoot := route.PublicDir

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For exact route path or paths that would be handled by sub-routes, use handler
		// Static files are for paths under this route that don't match other routes
		urlPath := r.URL.Path

		// Strip route prefix to get the file path within public_dir
		relativePath := strings.TrimPrefix(urlPath, routePath)
		if relativePath == "" {
			relativePath = "/"
		}

		// Try static file first (if not the route root itself)
		if relativePath != "/" && staticRoot != "" {
			filePath := filepath.Join(staticRoot, relativePath)
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				serveStaticFile(w, r, filePath, s.devLog != nil)
				return
			}
		}

		// Fall back to route handler
		handler.ServeHTTP(w, r)
	})
}

// createRootHandler creates a handler that serves static files with route fallback
func (s *Server) createRootHandler() http.Handler {
	// Determine static file root - prefer route's public_dir, fall back to static config
	var staticRoot string
	var rootRoute *config.Route

	// Find root route to get its public_dir
	for i := range s.config.Routes {
		if s.config.Routes[i].Path == "/" {
			rootRoute = &s.config.Routes[i]
			staticRoot = rootRoute.PublicDir
			break
		}
	}

	// Fall back to explicit static route config if no route public_dir
	if staticRoot == "" {
		for _, static := range s.config.Static {
			if static.Path == "/" && static.Root != "" {
				staticRoot = static.Root
				break
			}
		}
	}

	// Find root route handler if configured
	var rootHandler http.Handler
	if rootRoute != nil {
		handler, err := newParsleyHandler(s, *rootRoute, s.scriptCache)
		if err == nil {
			rootHandler = s.applyAuthMiddleware(handler, rootRoute.Auth)
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try static file first (if configured and file exists)
		if staticRoot != "" && r.URL.Path != "/" {
			filePath := filepath.Join(staticRoot, r.URL.Path)
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				serveStaticFile(w, r, filePath, s.devLog != nil)
				return
			}
		}

		// Fall back to root route handler
		if rootHandler != nil {
			rootHandler.ServeHTTP(w, r)
			return
		}

		// No handler - 404
		if s.config.Server.Dev {
			// Styled 404 in dev mode
			info := Dev404Info{
				RequestPath: r.URL.Path,
				StaticRoot:  staticRoot,
				HasHandler:  rootHandler != nil,
				RoutePath:   "/",
				BasePath:    filepath.Dir(s.configPath),
			}
			// Add checked paths
			if staticRoot != "" && r.URL.Path != "/" {
				relStatic := makeRelativePath(staticRoot, info.BasePath)
				info.CheckedPaths = append(info.CheckedPaths, relStatic+r.URL.Path)
			}
			s.handle404(w, r)
			return
		}
		s.handle404(w, r)
	})
}

// ReloadScripts clears the script cache, response cache, and fragment cache,
// forcing all scripts to be re-parsed and responses to be regenerated.
// This is useful for production deployments when scripts are updated.
// In dev mode, this also triggers browser reload via the live reload mechanism.
func (s *Server) ReloadScripts() {
	s.scriptCache.clear()
	s.responseCache.Clear()
	s.fragmentCache.Clear()
	s.assetRegistry.Clear()
	// Rebuild asset bundle
	if s.assetBundle != nil {
		if err := s.assetBundle.Rebuild(); err != nil {
			s.logWarn("failed to rebuild asset bundle: %v", err)
		}
	}
	// Trigger browser reload if watcher is active (dev mode)
	if s.watcher != nil {
		s.watcher.TriggerReload()
	}
	s.logInfo("caches cleared - scripts will be re-parsed on next request")
}

// Run starts the server and blocks until the context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	// Log version first
	fmt.Fprintf(s.stdout, "basil %s\n", s.version)

	addr := s.listenAddr()

	// Ensure databases are closed on shutdown
	if s.authDB != nil {
		defer func() {
			s.logInfo("closing auth database connection")
			s.authDB.Close()
		}()
	}
	if s.db != nil {
		defer func() {
			s.logInfo("closing database connection")
			s.db.Close()
		}()
	}

	// In dev mode, start file watcher for hot reload
	if s.config.Server.Dev {
		watcher, err := NewWatcher(s, s.configPath, s.stdout, s.stderr)
		if err != nil {
			s.logError("failed to create watcher: %v", err)
		} else {
			s.watcher = watcher
			if err := s.watcher.Start(ctx); err != nil {
				s.logError("failed to start watcher: %v", err)
			}
			defer s.watcher.Close()
		}
	}

	// Build handler chain
	var handler http.Handler = s.mux

	// In dev mode, inject live reload script into HTML responses
	if s.config.Server.Dev {
		handler = injectLiveReload(handler)
	}

	// Add proxy header handling (must be before logging to get real IPs)
	handler = newProxyAware(handler, s.config.Server.Proxy)

	// Add security headers
	handler = newSecurityHeaders(handler, s.config.Security, s.config.Server.Dev)

	// Add CORS middleware if configured
	if s.corsMW != nil {
		handler = s.corsMW.Handler(handler)
	}

	// Wrap with request logging middleware (unless level is error-only)
	if s.config.Logging.Level != "error" {
		handler = newRequestLogger(handler, s.stdout, s.config.Logging.Format)
	}

	// Wrap with compression (outermost - compresses all responses)
	handler = newCompressionHandler(handler, s.config.Compression)

	s.server = &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if s.config.Server.Dev {
			fmt.Fprintf(s.stdout, "Starting Basil in development mode on http://%s\n", addr)
			errCh <- s.server.ListenAndServe()
		} else {
			fmt.Fprintf(s.stdout, "Starting Basil on https://%s\n", addr)
			errCh <- s.listenAndServeTLS()
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		fmt.Fprintf(s.stdout, "\nShutting down gracefully...\n")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err != http.ErrServerClosed {
			return err
		}
		return nil
	}
}

// listenAddr returns the address to listen on based on configuration.
func (s *Server) listenAddr() string {
	host := s.config.Server.Host
	port := s.config.Server.Port

	if s.config.Server.Dev {
		if host == "" {
			host = "localhost"
		}
		if port == 0 || port == 443 {
			port = 8080
		}
	}

	return fmt.Sprintf("%s:%d", host, port)
}

// listenAndServeTLS starts HTTPS server with TLS.
// Supports automatic Let's Encrypt certificates or manual certificate files.
func (s *Server) listenAndServeTLS() error {
	cfg := s.config.Server.HTTPS

	// Manual cert mode
	if cfg.Cert != "" && cfg.Key != "" {
		s.logInfo("using manual TLS certificates")
		return s.server.ListenAndServeTLS(cfg.Cert, cfg.Key)
	}

	// Auto cert mode using Let's Encrypt
	if !cfg.Auto {
		return fmt.Errorf("HTTPS requires either auto: true or cert/key paths")
	}

	return s.listenAndServeAutocert()
}

// listenAndServeAutocert configures and starts the server with Let's Encrypt certificates.
func (s *Server) listenAndServeAutocert() error {
	cfg := s.config.Server.HTTPS

	// Determine cache directory for certificates
	cacheDir := "certs"
	if cfg.CacheDir != "" {
		cacheDir = cfg.CacheDir
	}

	// Create autocert manager
	manager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache(cacheDir),
		HostPolicy: s.hostPolicy(),
	}

	if cfg.Email != "" {
		manager.Email = cfg.Email
	}

	// Configure TLS
	s.server.TLSConfig = &tls.Config{
		GetCertificate: manager.GetCertificate,
		NextProtos:     []string{"h2", "http/1.1"}, // Enable HTTP/2
		MinVersion:     tls.VersionTLS12,
	}

	// Start HTTP redirect server on port 80 for ACME challenges and redirects
	go s.runHTTPRedirect(manager)

	s.logInfo("automatic TLS enabled via Let's Encrypt (cache: %s)", cacheDir)

	// ListenAndServeTLS with empty cert/key uses TLSConfig
	return s.server.ListenAndServeTLS("", "")
}

// hostPolicy returns a function that validates hostnames for certificate requests.
func (s *Server) hostPolicy() autocert.HostPolicy {
	host := s.config.Server.Host

	// If no host configured, allow any
	if host == "" {
		return nil
	}

	// Allow only the configured host
	return autocert.HostWhitelist(host)
}

// runHTTPRedirect starts an HTTP server on port 80 that:
// 1. Handles ACME HTTP-01 challenges for Let's Encrypt
// 2. Redirects all other requests to HTTPS
func (s *Server) runHTTPRedirect(manager *autocert.Manager) {
	redirectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Build HTTPS URL
		target := "https://" + r.Host + r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})

	// Use autocert's handler which passes ACME challenges to manager
	// and delegates everything else to our redirect handler
	httpServer := &http.Server{
		Addr:              ":80",
		Handler:           manager.HTTPHandler(redirectHandler),
		ReadHeaderTimeout: 10 * time.Second,
	}

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logError("HTTP redirect server error: %v", err)
	}
}
