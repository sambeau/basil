package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sambeau/basil/auth"
	"github.com/sambeau/basil/config"
)

func TestAPIRouteMapping(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "todos.pars")

	script := `let api = import @std/api

export get = api.public(fn(req) { {message: "ok"} })
export getById = api.public(fn(req) { {id: req.params.id} })
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := &config.Config{
		BaseDir: dir,
		Server:  config.ServerConfig{Host: "localhost", Port: 8080, Dev: true},
		Routes:  []config.Route{{Path: "/api/todos", Handler: scriptPath, Type: "api"}},
		Logging: config.LoggingConfig{Level: "info", Format: "text", Output: "stderr"},
	}

	stdout := &noopBuffer{}
	stderr := &noopBuffer{}

	srv, err := New(cfg, "", "test", "test-commit", stdout, stderr)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/todos", nil)
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Logf("stderr: %s", stderr.String())
		t.Logf("body: %s", rec.Body.String())
		t.Fatalf("expected 200 for collection, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if body["message"] != "ok" {
		t.Fatalf("expected message 'ok', got %v", body["message"])
	}

	reqID := httptest.NewRequest(http.MethodGet, "/api/todos/abc123", nil)
	recID := httptest.NewRecorder()
	srv.mux.ServeHTTP(recID, reqID)

	if recID.Code != http.StatusOK {
		t.Fatalf("expected 200 for getById, got %d", recID.Code)
	}

	var bodyID map[string]interface{}
	if err := json.Unmarshal(recID.Body.Bytes(), &bodyID); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if bodyID["id"] != "abc123" {
		t.Fatalf("expected id 'abc123', got %v", bodyID["id"])
	}
}

func TestAPIRouteAuthDefaultsToProtected(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "secure.pars")

	script := `let api = import @std/api

export get = fn(req) { {ok: true} }
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := &config.Config{
		BaseDir: dir,
		Server:  config.ServerConfig{Host: "localhost", Port: 8080, Dev: true},
		Routes:  []config.Route{{Path: "/api/secure", Handler: scriptPath, Type: "api"}},
		Logging: config.LoggingConfig{Level: "info", Format: "text", Output: "stderr"},
	}

	stdout := &noopBuffer{}
	stderr := &noopBuffer{}

	srv, err := New(cfg, "", "test", "test-commit", stdout, stderr)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/secure", nil)
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for protected route, got %d", rec.Code)
	}
}

func TestAPIRateLimitOverride(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "limited.pars")

	script := `let api = import @std/api

export rateLimit = {requests: 1, window: "1s"}
export get = api.public(fn(req) { {ok: true} })
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := &config.Config{
		BaseDir: dir,
		Server:  config.ServerConfig{Host: "localhost", Port: 8080, Dev: true},
		Routes:  []config.Route{{Path: "/api/limited", Handler: scriptPath, Type: "api"}},
		Logging: config.LoggingConfig{Level: "info", Format: "text", Output: "stderr"},
	}

	stdout := &noopBuffer{}
	stderr := &noopBuffer{}

	srv, err := New(cfg, "", "test", "test-commit", stdout, stderr)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	req1 := httptest.NewRequest(http.MethodGet, "/api/limited", nil)
	rec1 := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected first request 200, got %d", rec1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/limited", nil)
	rec2 := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request 429, got %d", rec2.Code)
	}
}

type noopBuffer struct {
	buf []byte
}

func (b *noopBuffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *noopBuffer) String() string {
	return string(b.buf)
}

// setUserOnRequest returns a new request with the user set in the context
func setUserOnRequest(r *http.Request, user *auth.User) *http.Request {
	ctx := context.WithValue(r.Context(), auth.UserContextKey, user)
	return r.WithContext(ctx)
}

func TestAPIAdminOnlyAllowsAdmin(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "admin.pars")

	script := `let api = import @std/api

export get = api.adminOnly(fn(req) { {admin: true} })
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := &config.Config{
		BaseDir: dir,
		Server:  config.ServerConfig{Host: "localhost", Port: 8080, Dev: true},
		Routes:  []config.Route{{Path: "/api/admin", Handler: scriptPath, Type: "api"}},
		Logging: config.LoggingConfig{Level: "info", Format: "text", Output: "stderr"},
	}

	srv, err := New(cfg, "", "test", "test-commit", &noopBuffer{}, &noopBuffer{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Admin user should be allowed
	adminUser := &auth.User{ID: "usr_admin", Name: "Admin", Role: auth.RoleAdmin}
	req := httptest.NewRequest(http.MethodGet, "/api/admin", nil)
	req = setUserOnRequest(req, adminUser)
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin user, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPIAdminOnlyDeniesEditor(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "admin.pars")

	script := `let api = import @std/api

export get = api.adminOnly(fn(req) { {admin: true} })
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := &config.Config{
		BaseDir: dir,
		Server:  config.ServerConfig{Host: "localhost", Port: 8080, Dev: true},
		Routes:  []config.Route{{Path: "/api/admin", Handler: scriptPath, Type: "api"}},
		Logging: config.LoggingConfig{Level: "info", Format: "text", Output: "stderr"},
	}

	srv, err := New(cfg, "", "test", "test-commit", &noopBuffer{}, &noopBuffer{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Editor user should be denied
	editorUser := &auth.User{ID: "usr_editor", Name: "Editor", Role: auth.RoleEditor}
	req := httptest.NewRequest(http.MethodGet, "/api/admin", nil)
	req = setUserOnRequest(req, editorUser)
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for editor user on admin-only route, got %d", rec.Code)
	}
}

func TestAPIRolesAllowsMatchingRole(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "editors.pars")

	script := `let api = import @std/api

export get = api.roles(["editor", "admin"], fn(req) { {allowed: true} })
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := &config.Config{
		BaseDir: dir,
		Server:  config.ServerConfig{Host: "localhost", Port: 8080, Dev: true},
		Routes:  []config.Route{{Path: "/api/editors", Handler: scriptPath, Type: "api"}},
		Logging: config.LoggingConfig{Level: "info", Format: "text", Output: "stderr"},
	}

	srv, err := New(cfg, "", "test", "test-commit", &noopBuffer{}, &noopBuffer{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Editor should be allowed
	editorUser := &auth.User{ID: "usr_editor", Name: "Editor", Role: auth.RoleEditor}
	req := httptest.NewRequest(http.MethodGet, "/api/editors", nil)
	req = setUserOnRequest(req, editorUser)
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for editor user, got %d: %s", rec.Code, rec.Body.String())
	}

	// Admin should also be allowed (listed in roles)
	adminUser := &auth.User{ID: "usr_admin", Name: "Admin", Role: auth.RoleAdmin}
	req2 := httptest.NewRequest(http.MethodGet, "/api/editors", nil)
	req2 = setUserOnRequest(req2, adminUser)
	rec2 := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin user, got %d", rec2.Code)
	}
}

func TestAPIRolesDeniesNonMatchingRole(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "admins.pars")

	script := `let api = import @std/api

export get = api.roles(["admin"], fn(req) { {allowed: true} })
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := &config.Config{
		BaseDir: dir,
		Server:  config.ServerConfig{Host: "localhost", Port: 8080, Dev: true},
		Routes:  []config.Route{{Path: "/api/admins", Handler: scriptPath, Type: "api"}},
		Logging: config.LoggingConfig{Level: "info", Format: "text", Output: "stderr"},
	}

	srv, err := New(cfg, "", "test", "test-commit", &noopBuffer{}, &noopBuffer{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Editor should be denied when only admin role is allowed
	editorUser := &auth.User{ID: "usr_editor", Name: "Editor", Role: auth.RoleEditor}
	req := httptest.NewRequest(http.MethodGet, "/api/admins", nil)
	req = setUserOnRequest(req, editorUser)
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for editor on admin-only roles route, got %d", rec.Code)
	}
}

func TestAPIUserWithNoRoleDeniedOnRoleProtectedRoute(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "protected.pars")

	script := `let api = import @std/api

export get = api.roles(["editor"], fn(req) { {allowed: true} })
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := &config.Config{
		BaseDir: dir,
		Server:  config.ServerConfig{Host: "localhost", Port: 8080, Dev: true},
		Routes:  []config.Route{{Path: "/api/protected", Handler: scriptPath, Type: "api"}},
		Logging: config.LoggingConfig{Level: "info", Format: "text", Output: "stderr"},
	}

	srv, err := New(cfg, "", "test", "test-commit", &noopBuffer{}, &noopBuffer{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// User with no role should be denied
	noRoleUser := &auth.User{ID: "usr_norole", Name: "No Role", Role: ""}
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req = setUserOnRequest(req, noRoleUser)
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for user with no role, got %d", rec.Code)
	}
}
