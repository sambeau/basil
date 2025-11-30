package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sambeau/basil/config"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  true,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stderr",
		},
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	srv, err := New(cfg, stdout, stderr)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if srv == nil {
		t.Fatal("New() returned nil server")
	}
}

func TestStaticFileServing(t *testing.T) {
	// Create temp directory with test files
	dir := t.TempDir()
	staticDir := filepath.Join(dir, "public")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("failed to create static dir: %v", err)
	}

	// Create a test file
	testContent := "Hello, static world!"
	testFile := filepath.Join(staticDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  true,
		},
		Static: []config.StaticRoute{
			{Path: "/static/", Root: staticDir},
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stderr",
		},
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	srv, err := New(cfg, stdout, stderr)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Test the static file serving
	req := httptest.NewRequest("GET", "/static/test.txt", nil)
	rec := httptest.NewRecorder()

	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != testContent {
		t.Errorf("expected body %q, got %q", testContent, rec.Body.String())
	}
}

func TestSingleFileServing(t *testing.T) {
	// Create temp file
	dir := t.TempDir()
	faviconContent := "fake favicon"
	faviconPath := filepath.Join(dir, "favicon.ico")
	if err := os.WriteFile(faviconPath, []byte(faviconContent), 0644); err != nil {
		t.Fatalf("failed to write favicon: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  true,
		},
		Static: []config.StaticRoute{
			{Path: "/favicon.ico", File: faviconPath},
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stderr",
		},
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	srv, err := New(cfg, stdout, stderr)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	req := httptest.NewRequest("GET", "/favicon.ico", nil)
	rec := httptest.NewRecorder()

	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != faviconContent {
		t.Errorf("expected body %q, got %q", faviconContent, rec.Body.String())
	}
}

func TestParsleyRouteExecution(t *testing.T) {
	// Create temp directory with a Parsley script
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "index.parsley")
	scriptContent := `"Hello from Parsley!"`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  true,
		},
		Routes: []config.Route{
			{Path: "/", Handler: scriptPath},
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stderr",
		},
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	srv, err := New(cfg, stdout, stderr)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if !containsString(rec.Body.String(), "Hello from Parsley!") {
		t.Errorf("expected body to contain 'Hello from Parsley!', got %q", rec.Body.String())
	}
}

func TestParsleyRouteWithMapResponse(t *testing.T) {
	// Create temp directory with a Parsley script that returns a dict
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "api.parsley")
	// Parsley uses {key: value} syntax (not "key": value)
	scriptContent := `{message: "Hello", count: 42}`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  true,
		},
		Routes: []config.Route{
			{Path: "/api", Handler: scriptPath},
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stderr",
		},
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	srv, err := New(cfg, stdout, stderr)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	req := httptest.NewRequest("GET", "/api", nil)
	rec := httptest.NewRecorder()

	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	// Should be JSON content type
	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Errorf("expected JSON content type, got %q", contentType)
	}
}

func TestParsleyRouteMissingScript(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  true,
		},
		Routes: []config.Route{
			{Path: "/", Handler: "/nonexistent/script.parsley"},
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stderr",
		},
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	srv, err := New(cfg, stdout, stderr)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	srv.mux.ServeHTTP(rec, req)

	// Should return 500 for missing script
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
}

func TestListenAddr(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		expected string
	}{
		{
			name: "dev mode defaults",
			cfg: &config.Config{
				Server: config.ServerConfig{Dev: true},
			},
			expected: "localhost:8080",
		},
		{
			name: "dev mode with port 443 (override to 8080)",
			cfg: &config.Config{
				Server: config.ServerConfig{Dev: true, Port: 443},
			},
			expected: "localhost:8080",
		},
		{
			name: "dev mode with custom port",
			cfg: &config.Config{
				Server: config.ServerConfig{Dev: true, Port: 3000},
			},
			expected: "localhost:3000",
		},
		{
			name: "production mode",
			cfg: &config.Config{
				Server: config.ServerConfig{Host: "example.com", Port: 443},
			},
			expected: "example.com:443",
		},
		{
			name: "production mode empty host",
			cfg: &config.Config{
				Server: config.ServerConfig{Port: 443},
			},
			expected: ":443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := &Server{config: tt.cfg}
			addr := srv.listenAddr()
			if addr != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, addr)
			}
		})
	}
}

func TestGracefulShutdown(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0, // Random port
			Dev:  true,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stderr",
		},
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	srv, err := New(cfg, stdout, stderr)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Create a context that will be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		// Override port to use a random available port
		srv.config.Server.Port = 18080 // Use high port to avoid conflicts
		errCh <- srv.Run(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()

	// Wait for server to shut down
	select {
	case err := <-errCh:
		// Should shut down cleanly
		if err != nil && err != context.Canceled {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("server did not shut down in time")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && contains(s, substr)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
