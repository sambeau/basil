package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/sambeau/basil/config"
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

	// Set up routes
	if err := s.setupRoutes(); err != nil {
		return nil, fmt.Errorf("setting up routes: %w", err)
	}

	return s, nil
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
