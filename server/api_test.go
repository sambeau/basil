package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

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
