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
	"time"

	"github.com/sambeau/basil/auth"
	"github.com/sambeau/basil/config"
	"golang.org/x/crypto/acme/autocert"

	// SQLite driver (pure Go, no CGO required)
	_ "modernc.org/sqlite"
)

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
	watcher       *Watcher
	db            *sql.DB // Database connection (nil if not configured)
	dbDriver      string  // Database driver name ("sqlite", etc.)

	// Auth system (nil if auth not enabled)
	authDB       *auth.DB
	authWebAuthn *auth.WebAuthnManager
	authHandlers *auth.Handlers
	authMW       *auth.Middleware
}

// New creates a new Basil server with the given configuration.
func New(cfg *config.Config, configPath string, version string, stdout, stderr io.Writer) (*Server, error) {
	s := &Server{
		config:        cfg,
		configPath:    configPath,
		version:       version,
		stdout:        stdout,
		stderr:        stderr,
		mux:           http.NewServeMux(),
		scriptCache:   newScriptCache(cfg.Server.Dev),
		responseCache: newResponseCache(cfg.Server.Dev),
	}

	// Initialize database connection if configured
	if err := s.initDatabase(); err != nil {
		return nil, fmt.Errorf("initializing database: %w", err)
	}

	// Initialize auth system if enabled
	if err := s.initAuth(); err != nil {
		// Clean up database on auth init failure
		if s.db != nil {
			s.db.Close()
		}
		return nil, fmt.Errorf("initializing auth: %w", err)
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
		return nil, fmt.Errorf("setting up routes: %w", err)
	}

	return s, nil
}

// initDatabase opens the database connection if configured.
func (s *Server) initDatabase() error {
	dbCfg := s.config.Database

	// No database configured
	if dbCfg.Driver == "" {
		return nil
	}

	switch dbCfg.Driver {
	case "sqlite":
		return s.initSQLite(dbCfg.Path)
	case "postgres", "mysql":
		return fmt.Errorf("database driver %q not yet supported", dbCfg.Driver)
	default:
		return fmt.Errorf("unknown database driver %q", dbCfg.Driver)
	}
}

// initSQLite opens a SQLite database connection.
func (s *Server) initSQLite(path string) error {
	if path == "" {
		return fmt.Errorf("sqlite requires database.path to be set")
	}

	// Resolve relative paths against config base directory
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

// setupRoutes configures the HTTP mux with static and dynamic routes.
func (s *Server) setupRoutes() error {
	// In dev mode, add live reload endpoint
	if s.config.Server.Dev {
		s.mux.Handle("/__livereload", newLiveReloadHandler(s))
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

	// Register explicit static routes (non-root paths like /favicon.ico)
	for _, static := range s.config.Static {
		if static.Path != "/" {
			if static.Root != "" {
				handler := http.StripPrefix(static.Path, http.FileServer(http.Dir(static.Root)))
				s.mux.Handle(static.Path, handler)
			} else if static.File != "" {
				filePath := static.File
				s.mux.HandleFunc(static.Path, func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, filePath)
				})
			}
		}
	}

	// Register Parsley routes (specific paths)
	for _, route := range s.config.Routes {
		if route.Path == "/" {
			continue // Handle root separately as fallback
		}
		handler, err := newParsleyHandler(s, route, s.scriptCache)
		if err != nil {
			return fmt.Errorf("creating handler for %s: %w", route.Path, err)
		}
		finalHandler := s.applyAuthMiddleware(handler, route.Auth)
		s.mux.Handle(route.Path, finalHandler)
	}

	// Create fallback handler for "/" that serves:
	// 1. Static files from public_dir (if file exists)
	// 2. Root route handler (if configured)
	// 3. 404
	s.mux.Handle("/", s.createRootHandler())

	return nil
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

// createRootHandler creates a handler that serves static files with route fallback
func (s *Server) createRootHandler() http.Handler {
	// Determine static file root
	var staticRoot string
	for _, static := range s.config.Static {
		if static.Path == "/" && static.Root != "" {
			staticRoot = static.Root
			break
		}
	}
	if staticRoot == "" && s.config.PublicDir != "" {
		staticRoot = s.config.PublicDir
	}

	// Find root route handler if configured
	var rootHandler http.Handler
	for _, route := range s.config.Routes {
		if route.Path == "/" {
			handler, err := newParsleyHandler(s, route, s.scriptCache)
			if err == nil {
				rootHandler = s.applyAuthMiddleware(handler, route.Auth)
			}
			break
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try static file first (if configured and file exists)
		if staticRoot != "" && r.URL.Path != "/" {
			filePath := filepath.Join(staticRoot, r.URL.Path)
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				http.ServeFile(w, r, filePath)
				return
			}
		}

		// Fall back to root route handler
		if rootHandler != nil {
			rootHandler.ServeHTTP(w, r)
			return
		}

		// No handler - 404
		http.NotFound(w, r)
	})
}

// ReloadScripts clears the script cache and response cache, forcing all scripts
// to be re-parsed and responses to be regenerated.
// This is useful for production deployments when scripts are updated.
// In dev mode, this also triggers browser reload via the live reload mechanism.
func (s *Server) ReloadScripts() {
	s.scriptCache.clear()
	s.responseCache.Clear()
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

	// Wrap with request logging middleware (unless level is error-only)
	if s.config.Logging.Level != "error" {
		handler = newRequestLogger(handler, s.stdout, s.config.Logging.Format)
	}

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
