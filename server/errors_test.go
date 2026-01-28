package server

import (
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sambeau/basil/server/config"
)

func TestExtractLineInfo_ParseError(t *testing.T) {
	msg := "parse error in /app/test.pars: unexpected token"
	file, line, col, cleanMsg := extractLineInfo(msg)

	if file != "/app/test.pars" {
		t.Errorf("expected file '/app/test.pars', got %q", file)
	}
	if cleanMsg != "unexpected token" {
		t.Errorf("expected clean message 'unexpected token', got %q", cleanMsg)
	}
	// Line/col may be 0 if not in message
	_ = line
	_ = col
}

func TestExtractLineInfo_WithLineNumber(t *testing.T) {
	msg := "error at line 42: something went wrong"
	_, line, _, _ := extractLineInfo(msg)

	if line != 42 {
		t.Errorf("expected line 42, got %d", line)
	}
}

func TestExtractLineInfo_ScriptError(t *testing.T) {
	msg := "script error in /path/to/handler.pars: not a function: DICTIONARY"
	file, _, _, cleanMsg := extractLineInfo(msg)

	if file != "/path/to/handler.pars" {
		t.Errorf("expected file path, got %q", file)
	}
	if cleanMsg != "not a function: DICTIONARY" {
		t.Errorf("expected clean message, got %q", cleanMsg)
	}
}

func TestExtractLineInfo_ModuleParseErrors(t *testing.T) {
	// Test the multi-line parse errors format with "module" prefix
	msg := `parse errors in module ./app/pages/home.pars:
  expected identifier as dictionary key, got opening tag at line 6, column 3
  line 31, column 7: unexpected 'Page'`

	file, line, col, cleanMsg := extractLineInfo(msg)

	if file != "./app/pages/home.pars" {
		t.Errorf("expected file './app/pages/home.pars', got %q", file)
	}
	if line != 6 {
		t.Errorf("expected line 6, got %d", line)
	}
	if col != 3 {
		t.Errorf("expected column 3, got %d", col)
	}
	// Clean message should have the error details (trimmed of leading whitespace)
	if !strings.Contains(cleanMsg, "expected identifier") {
		t.Errorf("expected clean message to contain error, got %q", cleanMsg)
	}
}

func TestExtractLineInfo_ModuleRuntimeError(t *testing.T) {
	// Test runtime error from a module - the format from evaluator when a module has an error
	// This is the format: "in module <path>: line X, column Y: <message>"
	msg := "in module ./app/pages/scouts.pars: line 18, column 20: dot notation can only be used on dictionaries, got BUILTIN"

	file, line, col, cleanMsg := extractLineInfo(msg)

	if file != "./app/pages/scouts.pars" {
		t.Errorf("expected file './app/pages/scouts.pars', got %q", file)
	}
	if line != 18 {
		t.Errorf("expected line 18, got %d", line)
	}
	if col != 20 {
		t.Errorf("expected column 20, got %d", col)
	}
	// Clean message should be stripped of the module prefix and line/col info
	if strings.Contains(cleanMsg, "in module") {
		t.Errorf("clean message should not contain 'in module', got %q", cleanMsg)
	}
}

func TestHandleScriptError_DevMode(t *testing.T) {
	// Initialize prelude for dev error page
	if err := initPrelude("test"); err != nil {
		t.Fatalf("initPrelude() error = %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Dev: true,
		},
	}
	s := &Server{config: cfg, stderr: io.Discard}
	h := &parsleyHandler{
		server:     s,
		scriptPath: "/test/handler.pars",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	h.handleScriptError(w, req, "runtime", "/test/handler.pars", "test error message")

	resp := w.Result()
	body := w.Body.String()

	// Should return 500
	if resp.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	// Should be HTML (dev error page)
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html, got %s", ct)
	}

	// Should contain the error message
	if !strings.Contains(body, "test error message") {
		t.Error("expected error message in body")
	}

	// Note: live reload script is injected by middleware, not tested here
}

func TestHandleScriptErrorWithLocation_ModuleError(t *testing.T) {
	// Initialize prelude for dev error page
	if err := initPrelude("test"); err != nil {
		t.Fatalf("initPrelude() error = %v", err)
	}

	// Test that module errors show the correct file path, not the parent handler path
	cfg := &config.Config{
		Server: config.ServerConfig{
			Dev: true,
		},
	}
	s := &Server{config: cfg, stderr: io.Discard, configPath: "/app/basil.yaml"}
	h := &parsleyHandler{
		server:     s,
		scriptPath: "/app/app.pars", // The parent handler
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	// Simulate error message from a module - this is the format from evaluator
	moduleErrMsg := "in module ./app/pages/scouts.pars: line 18, column 20: dot notation can only be used on dictionaries, got BUILTIN"
	h.handleScriptErrorWithLocation(w, req, "runtime", h.scriptPath, moduleErrMsg, 0, 0)

	body := w.Body.String()

	// The error page should show the MODULE path, not the parent handler path
	if strings.Contains(body, "app.pars") && !strings.Contains(body, "scouts.pars") {
		t.Error("error page should show module path (scouts.pars), not parent handler path (app.pars)")
	}

	// Should contain the correct file path
	if !strings.Contains(body, "scouts.pars") {
		t.Error("expected error page to show scouts.pars")
	}

	// Should show line 18
	if !strings.Contains(body, ":18") && !strings.Contains(body, "18") {
		t.Error("expected error page to show line 18")
	}
}

func TestHandleScriptErrorWithLocation_ModuleNotFound(t *testing.T) {
	// Initialize prelude for dev error page
	if err := initPrelude("test"); err != nil {
		t.Fatalf("initPrelude() error = %v", err)
	}

	// Test module-not-found errors show correct module file (no line info available)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Dev: true,
		},
	}
	s := &Server{config: cfg, stderr: io.Discard, configPath: "/app/basil.yaml"}
	h := &parsleyHandler{
		server: s,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	moduleErrMsg := "in module ./app/pages/scouts.pars: module not found: ./app/pages/std/table"
	h.handleScriptErrorWithLocation(w, req, "runtime", h.scriptPath, moduleErrMsg, 0, 0)

	body := w.Body.String()

	// Should show the module where the import failed (scouts.pars), not the parent
	if !strings.Contains(body, "scouts.pars") {
		t.Errorf("expected error page to show scouts.pars, body contains: %s", body[:min(500, len(body))])
	}

	// Should NOT show app.pars as the primary file
	// (it might appear in the message but not in the file-path span)
	if strings.Contains(body, `class="file-path">./app/app.pars`) {
		t.Error("error page should not show app.pars as primary file path")
	}
}

func TestHandleScriptError_ProdMode(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Dev: false,
		},
	}
	s := &Server{config: cfg, stderr: io.Discard}
	h := &parsleyHandler{
		server:     s,
		scriptPath: "/test/handler.pars",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	h.handleScriptError(w, req, "runtime", "/test/handler.pars", "test error message")

	resp := w.Result()
	body := w.Body.String()

	// Should return 500
	if resp.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	// Should be HTML (prelude error page)
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected HTML in prod mode, got %s", ct)
	}

	// Should contain 500 error message
	if !strings.Contains(body, "500") {
		t.Error("expected body to contain '500'")
	}

	// Should NOT contain detailed error info (test error message shouldn't appear)
	if strings.Contains(body, "test error message") {
		t.Error("should not expose detailed error details in production")
	}
}

func TestCreateErrorEnv(t *testing.T) {
	// Initialize prelude before running tests
	if err := initPrelude("test"); err != nil {
		t.Fatalf("initPrelude() error = %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Dev: true,
		},
	}
	s := &Server{config: cfg, stderr: io.Discard}
	req := httptest.NewRequest("GET", "/test/path?foo=bar", nil)
	err := fmt.Errorf("test error")

	env := s.createErrorEnv(req, 404, err)

	// Check that error object was set
	errorObj, ok := env.Get("error")
	if !ok {
		t.Fatal("expected 'error' to be set in environment")
	}
	if errorObj == nil {
		t.Fatal("error object should not be nil")
	}

	// Check that basil object was set on BasilCtx
	if env.BasilCtx == nil {
		t.Fatal("expected BasilCtx to be set in environment")
	}
}

func TestRenderPreludeError_404(t *testing.T) {
	// Initialize prelude before running tests
	if err := initPrelude("test"); err != nil {
		t.Fatalf("initPrelude() error = %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Dev: true,
		},
	}
	s := &Server{config: cfg, stderr: io.Discard}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/missing", nil)
	err := fmt.Errorf("not found")

	success := s.renderPreludeError(w, req, 404, err)

	if !success {
		t.Fatal("expected renderPreludeError to succeed")
	}

	resp := w.Result()
	body := w.Body.String()

	// Should return 404
	if resp.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}

	// Should be HTML
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected HTML content type, got %s", ct)
	}

	// Should contain 404 error page content
	if !strings.Contains(body, "404") {
		t.Errorf("expected body to contain '404', got: %s", body)
	}
	if !strings.Contains(body, "not found") {
		t.Errorf("expected body to contain 'not found', got: %s", body)
	}
}

func TestRenderPreludeError_500(t *testing.T) {
	// Initialize prelude before running tests
	if err := initPrelude("test"); err != nil {
		t.Fatalf("initPrelude() error = %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Dev: true,
		},
	}
	s := &Server{config: cfg, stderr: io.Discard}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/error", nil)
	err := fmt.Errorf("server error")

	success := s.renderPreludeError(w, req, 500, err)

	if !success {
		t.Fatal("expected renderPreludeError to succeed")
	}

	resp := w.Result()
	body := w.Body.String()

	// Should return 500
	if resp.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	// Should be HTML
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected HTML content type, got %s", ct)
	}

	// Should contain 500 error page content
	if !strings.Contains(body, "500") {
		t.Error("expected body to contain '500'")
	}
	if !strings.Contains(body, "Internal Server Error") {
		t.Error("expected body to contain 'Internal Server Error'")
	}
}

func TestHandle404(t *testing.T) {
	// Initialize prelude before running tests
	if err := initPrelude("test"); err != nil {
		t.Fatalf("initPrelude() error = %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Dev: true,
		},
	}
	s := &Server{config: cfg, stderr: io.Discard}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/missing", nil)

	s.handle404(w, req)

	resp := w.Result()
	body := w.Body.String()

	// Should return 404
	if resp.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}

	// Should be HTML
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected HTML content type, got %s", ct)
	}

	// Should contain 404 content
	if !strings.Contains(body, "404") {
		t.Error("expected body to contain '404'")
	}
}

func TestHandle500(t *testing.T) {
	// Initialize prelude before running tests
	if err := initPrelude("test"); err != nil {
		t.Fatalf("initPrelude() error = %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Dev: true,
		},
	}
	s := &Server{config: cfg, stderr: io.Discard}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/error", nil)
	err := fmt.Errorf("test error")

	s.handle500(w, req, err)

	resp := w.Result()
	body := w.Body.String()

	// Should return 500
	if resp.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	// Should be HTML
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected HTML content type, got %s", ct)
	}

	// Should contain 500 content
	if !strings.Contains(body, "500") {
		t.Error("expected body to contain '500'")
	}
}
