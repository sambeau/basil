package server

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/sambeau/basil/config"

	// SQLite driver (pure Go, no CGO required)
	_ "modernc.org/sqlite"
)

// Server represents a Basil web server instance.
type Server struct {
	config      *config.Config
	configPath  string
	stdout      io.Writer
	stderr      io.Writer
	mux         *http.ServeMux
	server      *http.Server
	scriptCache *scriptCache
	watcher     *Watcher
	db          *sql.DB // Database connection (nil if not configured)
	dbDriver    string  // Database driver name ("sqlite", etc.)
}

// New creates a new Basil server with the given configuration.
func New(cfg *config.Config, configPath string, stdout, stderr io.Writer) (*Server, error) {
	s := &Server{
		config:      cfg,
		configPath:  configPath,
		stdout:      stdout,
		stderr:      stderr,
		mux:         http.NewServeMux(),
		scriptCache: newScriptCache(cfg.Server.Dev),
	}

	// Initialize database connection if configured
	if err := s.initDatabase(); err != nil {
		return nil, fmt.Errorf("initializing database: %w", err)
	}

	// Set up routes
	if err := s.setupRoutes(); err != nil {
		// Clean up database on route setup failure
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

// setupRoutes configures the HTTP mux with static and dynamic routes.
func (s *Server) setupRoutes() error {
	// In dev mode, add live reload endpoint
	if s.config.Server.Dev {
		s.mux.Handle("/__livereload", newLiveReloadHandler(s))
	}

	// Register static routes first (more specific)
	for _, static := range s.config.Static {
		if static.Root != "" {
			// Directory serving
			handler := http.StripPrefix(static.Path, http.FileServer(http.Dir(static.Root)))
			s.mux.Handle(static.Path, handler)
		} else if static.File != "" {
			// Single file serving - capture path for closure
			filePath := static.File
			s.mux.HandleFunc(static.Path, func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, filePath)
			})
		}
	}

	// Register Parsley routes
	for _, route := range s.config.Routes {
		handler, err := newParsleyHandler(s, route, s.scriptCache)
		if err != nil {
			return fmt.Errorf("creating handler for %s: %w", route.Path, err)
		}
		s.mux.Handle(route.Path, handler)
	}

	return nil
}

// Run starts the server and blocks until the context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	addr := s.listenAddr()

	// Ensure database is closed on shutdown
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

// listenAndServeTLS starts HTTPS server.
// Placeholder for Phase 1 - will implement autocert in Phase 2.
func (s *Server) listenAndServeTLS() error {
	cfg := s.config.Server.HTTPS

	// Manual cert mode
	if cfg.Cert != "" && cfg.Key != "" {
		return s.server.ListenAndServeTLS(cfg.Cert, cfg.Key)
	}

	// Auto cert mode - placeholder for Phase 2
	return fmt.Errorf("HTTPS auto mode not yet implemented - use --dev for development")
}
