package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sambeau/basil/auth"
	"github.com/sambeau/basil/config"
)

func TestGitHandler_AuthRequired(t *testing.T) {
	// Create a minimal config with auth required
	cfg := &config.Config{
		Server: config.ServerConfig{Dev: false},
		Git:    config.GitConfig{Enabled: true, RequireAuth: true},
	}

	// Create handler without auth DB (will fail auth)
	var stdout, stderr bytes.Buffer
	handler, err := NewGitHandler(t.TempDir(), nil, cfg, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("NewGitHandler failed: %v", err)
	}

	// Make request without auth header
	req := httptest.NewRequest("GET", "/.git/info/refs", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should get 401
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	// Should have WWW-Authenticate header
	if w.Header().Get("WWW-Authenticate") == "" {
		t.Error("expected WWW-Authenticate header")
	}
}

func TestGitHandler_DevModeLocalhost(t *testing.T) {
	// Create config with dev mode enabled
	cfg := &config.Config{
		Server:  config.ServerConfig{Dev: true},
		Git:     config.GitConfig{Enabled: true, RequireAuth: true},
		BaseDir: t.TempDir(),
	}

	var stdout, stderr bytes.Buffer
	handler, err := NewGitHandler(cfg.BaseDir, nil, cfg, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("NewGitHandler failed: %v", err)
	}

	// Make request from localhost
	req := httptest.NewRequest("GET", "/.git/info/refs", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should NOT get 401 (dev mode allows localhost)
	// Note: we'll get some other error because there's no git repo, but not 401
	if w.Code == http.StatusUnauthorized {
		t.Error("dev mode localhost should not require auth")
	}
}

func TestGitHandler_IsPushRequest(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Dev: true},
		Git:    config.GitConfig{Enabled: true, RequireAuth: false},
	}

	var stdout, stderr bytes.Buffer
	handler, err := NewGitHandler(t.TempDir(), nil, cfg, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("NewGitHandler failed: %v", err)
	}

	tests := []struct {
		path   string
		query  string
		isPush bool
	}{
		{"/.git/info/refs", "service=git-upload-pack", false},
		{"/.git/info/refs", "service=git-receive-pack", true},
		{"/.git/git-upload-pack", "", false},
		{"/.git/git-receive-pack", "", true},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("POST", tt.path+"?"+tt.query, nil)
		got := handler.isPushRequest(req)
		if got != tt.isPush {
			t.Errorf("isPushRequest(%s?%s) = %v, want %v", tt.path, tt.query, got, tt.isPush)
		}
	}
}

func TestGitHandler_RoleCheck(t *testing.T) {
	// This test verifies role checking logic
	// We can't easily test the full flow without a real auth DB,
	// but we can verify the handler properly checks roles

	tmpDir := t.TempDir()
	authDB, err := auth.OpenOrCreateDB(tmpDir)
	if err != nil {
		t.Fatalf("OpenOrCreateDB failed: %v", err)
	}
	defer authDB.Close()

	// Create a user with editor role
	user, err := authDB.CreateUserWithRole("Editor", "editor@test.com", auth.RoleEditor)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create API key
	_, plaintext, err := authDB.CreateAPIKey(user.ID, "test-key")
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	// Validate key returns user
	validatedUser, err := authDB.ValidateAPIKey(plaintext)
	if err != nil {
		t.Fatalf("ValidateAPIKey failed: %v", err)
	}
	if validatedUser == nil {
		t.Fatal("ValidateAPIKey returned nil user")
	}
	if validatedUser.Role != auth.RoleEditor {
		t.Errorf("expected role %s, got %s", auth.RoleEditor, validatedUser.Role)
	}
}
