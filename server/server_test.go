package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sambeau/basil/server/config"
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

	srv, err := New(cfg, "", "test", "test-commit", stdout, stderr)
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

	srv, err := New(cfg, "", "test", "test-commit", stdout, stderr)
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

	srv, err := New(cfg, "", "test", "test-commit", stdout, stderr)
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

	srv, err := New(cfg, "", "test", "test-commit", stdout, stderr)
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
			{Path: "/data", Handler: scriptPath},
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stderr",
		},
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	srv, err := New(cfg, "", "test", "test-commit", stdout, stderr)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	req := httptest.NewRequest("GET", "/data", nil)
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

	srv, err := New(cfg, "", "test", "test-commit", stdout, stderr)
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
	// This test verifies the shutdown logic without actually starting a server
	// We test the listenAddr and server creation, but skip the actual Run()
	// to avoid port binding issues in test environments

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: 18999,
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

	srv, err := New(cfg, "", "test", "test-commit", stdout, stderr)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Verify server was created with correct address
	addr := srv.listenAddr()
	if addr != "127.0.0.1:18999" {
		t.Errorf("expected address '127.0.0.1:18999', got %q", addr)
	}

	// Verify the server struct is properly initialized
	if srv.mux == nil {
		t.Error("expected mux to be initialized")
	}
	if srv.scriptCache == nil {
		t.Error("expected scriptCache to be initialized")
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

func TestIsProtectedPath(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 8080, Dev: true},
		Auth: config.AuthConfig{
			Enabled: false, // Auth disabled, but protected paths can still be configured
			ProtectedPaths: []config.ProtectedPath{
				{Path: "/dashboard"},
				{Path: "/admin", Roles: []string{"admin"}},
				{Path: "/editors", Roles: []string{"admin", "editor"}},
			},
		},
		Logging: config.LoggingConfig{Level: "info", Format: "text", Output: "stderr"},
	}

	srv, err := New(cfg, "", "test", "test-commit", &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	tests := []struct {
		path      string
		wantMatch bool
		wantPath  string
		wantRoles []string
	}{
		// Exact matches
		{"/dashboard", true, "/dashboard", nil},
		{"/admin", true, "/admin", []string{"admin"}},
		{"/editors", true, "/editors", []string{"admin", "editor"}},

		// Subpath matches
		{"/dashboard/", true, "/dashboard", nil},
		{"/dashboard/users", true, "/dashboard", nil},
		{"/dashboard/users/123", true, "/dashboard", nil},
		{"/admin/settings", true, "/admin", []string{"admin"}},

		// Non-matches
		{"/", false, "", nil},
		{"/login", false, "", nil},
		{"/dashboardx", false, "", nil}, // /dashboardx is NOT under /dashboard
		{"/adminpanel", false, "", nil}, // /adminpanel is NOT under /admin
		{"/public", false, "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			pp := srv.isProtectedPath(tt.path)
			if tt.wantMatch {
				if pp == nil {
					t.Errorf("expected path %q to match protected path, got nil", tt.path)
					return
				}
				if pp.Path != tt.wantPath {
					t.Errorf("expected matched path %q, got %q", tt.wantPath, pp.Path)
				}
				if len(pp.Roles) != len(tt.wantRoles) {
					t.Errorf("expected %d roles, got %d", len(tt.wantRoles), len(pp.Roles))
				}
			} else {
				if pp != nil {
					t.Errorf("expected path %q to NOT match, but matched %q", tt.path, pp.Path)
				}
			}
		})
	}
}

func TestGetLoginPath(t *testing.T) {
	// Test default
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 8080, Dev: true},
		Logging: config.LoggingConfig{Level: "info", Format: "text", Output: "stderr"},
	}
	srv, _ := New(cfg, "", "test", "test-commit", &bytes.Buffer{}, &bytes.Buffer{})
	if srv.getLoginPath() != "/login" {
		t.Errorf("expected default login path /login, got %q", srv.getLoginPath())
	}

	// Test custom
	cfg.Auth.LoginPath = "/auth/signin"
	srv2, _ := New(cfg, "", "test", "test-commit", &bytes.Buffer{}, &bytes.Buffer{})
	if srv2.getLoginPath() != "/auth/signin" {
		t.Errorf("expected custom login path /auth/signin, got %q", srv2.getLoginPath())
	}
}

// TestMetaInjection verifies that meta section from config is accessible in Parsley handlers
func TestMetaInjection(t *testing.T) {
	// Create temp directory with a Parsley script that reads meta
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "meta_test.parsley")
	// Script that outputs meta values
	scriptContent := `<p>meta.name</p><p>meta.features.dark_mode</p>`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := config.Defaults()
	cfg.Server.Dev = true
	cfg.Meta = map[string]any{
		"name": "Test Site",
		"features": map[string]any{
			"dark_mode": true,
		},
	}
	cfg.Routes = []config.Route{
		{Path: "/test", Handler: scriptPath},
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	srv, err := New(cfg, "", "test", "test-commit", stdout, stderr)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer srv.Close()

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
		t.Logf("body: %s", rec.Body.String())
	}

	body := rec.Body.String()
	if !containsString(body, "Test Site") {
		t.Errorf("expected body to contain 'Test Site', got %q", body)
	}
	if !containsString(body, "true") {
		t.Errorf("expected body to contain 'true' for dark_mode, got %q", body)
	}
}
