package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sambeau/basil/config"
)

func TestDevToolsIndex(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = true

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", &stdout, &stderr)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer s.Close()

	// Create handler and request
	handler := newDevToolsHandler(s)
	req := httptest.NewRequest("GET", "/__", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Basil Dev Tools") {
		t.Error("expected index page to contain 'Basil Dev Tools'")
	}
	if !strings.Contains(body, "/__/logs") {
		t.Error("expected index page to contain link to logs")
	}
}

func TestDevToolsLogsHTML(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = true

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", &stdout, &stderr)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer s.Close()

	// Add some log entries
	s.devLog.Log(LogEntry{
		Route:     "",
		Level:     "info",
		Filename:  "test.pars",
		Line:      42,
		CallRepr:  "dev.log(users)",
		ValueRepr: "[{name: \"Alice\"}]",
	})

	// Request logs page
	handler := newDevToolsHandler(s)
	req := httptest.NewRequest("GET", "/__/logs", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Basil Logs") {
		t.Error("expected page to contain 'Basil Logs'")
	}
	if !strings.Contains(body, "test.pars:42") {
		t.Error("expected page to contain filename and line")
	}
	if !strings.Contains(body, "dev.log(users)") {
		t.Error("expected page to contain call repr")
	}
	if !strings.Contains(body, "Alice") {
		t.Error("expected page to contain value")
	}
}

func TestDevToolsLogsText(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = true

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", &stdout, &stderr)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer s.Close()

	// Add a log entry
	s.devLog.Log(LogEntry{
		Route:     "",
		Level:     "info",
		Filename:  "test.pars",
		Line:      42,
		CallRepr:  "dev.log(x)",
		ValueRepr: "test value",
	})

	// Request text format
	handler := newDevToolsHandler(s)
	req := httptest.NewRequest("GET", "/__/logs?text", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain") {
		t.Errorf("expected text/plain content type, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "INFO") {
		t.Error("expected text output to contain INFO")
	}
	if !strings.Contains(body, "test.pars:42") {
		t.Error("expected text output to contain filename:line")
	}
	if !strings.Contains(body, "test value") {
		t.Error("expected text output to contain value")
	}
}

func TestDevToolsLogsClear(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = true

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", &stdout, &stderr)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer s.Close()

	// Add a log entry
	s.devLog.Log(LogEntry{
		Route:     "",
		Level:     "info",
		Filename:  "test.pars",
		Line:      1,
		CallRepr:  "dev.log(x)",
		ValueRepr: "value",
	})

	// Verify entry exists
	count, _ := s.devLog.Count("")
	if count != 1 {
		t.Fatalf("expected 1 entry before clear, got %d", count)
	}

	// Request clear
	handler := newDevToolsHandler(s)
	req := httptest.NewRequest("GET", "/__/logs?clear", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should redirect
	if w.Code != http.StatusSeeOther {
		t.Errorf("expected status 303 (redirect), got %d", w.Code)
	}

	// Verify cleared
	count, _ = s.devLog.Count("")
	if count != 0 {
		t.Errorf("expected 0 entries after clear, got %d", count)
	}
}

func TestDevToolsLogsRoute(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = true

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", &stdout, &stderr)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer s.Close()

	// Add entries to different routes
	s.devLog.Log(LogEntry{Route: "", Level: "info", Filename: "test.pars", Line: 1, CallRepr: "dev.log(a)", ValueRepr: "default"})
	s.devLog.Log(LogEntry{Route: "users", Level: "info", Filename: "test.pars", Line: 2, CallRepr: "dev.log(b)", ValueRepr: "users route"})
	s.devLog.Log(LogEntry{Route: "orders", Level: "info", Filename: "test.pars", Line: 3, CallRepr: "dev.log(c)", ValueRepr: "orders route"})

	// Request users route
	handler := newDevToolsHandler(s)
	req := httptest.NewRequest("GET", "/__/logs/users", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Basil Logs: users") {
		t.Error("expected page title to contain route name")
	}
	if !strings.Contains(body, "users route") {
		t.Error("expected page to contain users route entry")
	}
	if strings.Contains(body, "orders route") {
		t.Error("expected page to NOT contain orders route entry")
	}
}

func TestDevTools404InProduction(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = false // Production mode

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", &stdout, &stderr)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer s.Close()

	// Verify devLog is nil in production
	if s.devLog != nil {
		t.Error("expected devLog to be nil in production mode")
	}

	// Create handler and test that it returns 404
	handler := newDevToolsHandler(s)

	paths := []string{"/__", "/__/", "/__/logs", "/__/logs/users"}
	for _, path := range paths {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 for %s in production, got %d", path, w.Code)
		}
	}
}

func TestDevToolsEmptyState(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = true

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", &stdout, &stderr)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer s.Close()

	// Request logs page with no entries
	handler := newDevToolsHandler(s)
	req := httptest.NewRequest("GET", "/__/logs", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "No logs yet") {
		t.Error("expected empty state message")
	}
	if !strings.Contains(body, "dev.log(value)") {
		t.Error("expected usage hint in empty state")
	}
}

func TestDevToolsWarnLevel(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = true

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", &stdout, &stderr)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer s.Close()

	// Add a warn-level entry
	s.devLog.Log(LogEntry{
		Route:     "",
		Level:     "warn",
		Filename:  "test.pars",
		Line:      42,
		CallRepr:  "dev.log(x, {level: \"warn\"})",
		ValueRepr: "warning message",
	})

	handler := newDevToolsHandler(s)
	req := httptest.NewRequest("GET", "/__/logs", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body := w.Body.String()
	// Check for warn styling
	if !strings.Contains(body, "class=\"log-entry warn\"") {
		t.Error("expected warn class on log entry")
	}
	if !strings.Contains(body, "⚠️") {
		t.Error("expected warning icon")
	}
}
