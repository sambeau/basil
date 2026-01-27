package server

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/server/config"
)

func TestDevToolsIndex(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = true

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", "test-commit", &stdout, &stderr)
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
	if !strings.Contains(body, "Basil") {
		t.Errorf("expected index page to contain 'Basil', got: %s", body[:min(500, len(body))])
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
	s, err := New(cfg, "", "test", "test-commit", &stdout, &stderr)
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
	if !strings.Contains(body, "Logs") {
		t.Error("expected page to contain 'Logs'")
	}
	if !strings.Contains(body, "test.pars") {
		t.Error("expected page to contain filename")
	}
	if !strings.Contains(body, "dev.log(users)") {
		t.Error("expected page to contain call repr")
	}
	if !strings.Contains(body, "Alice") {
		t.Error("expected page to contain value")
	}
}

func TestDevToolsLogsPoll(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = true

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", "test-commit", &stdout, &stderr)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer s.Close()

	handler := newDevToolsHandler(s)

	// First poll - should return seq 0
	req := httptest.NewRequest("GET", "/__/logs/poll", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
	if !strings.Contains(w.Body.String(), `"seq":0`) {
		t.Errorf("expected seq:0, got %s", w.Body.String())
	}

	// Add a log entry
	s.devLog.Log(LogEntry{
		Route:     "",
		Level:     "info",
		Filename:  "test.pars",
		Line:      42,
		CallRepr:  "dev.log(x)",
		ValueRepr: "test",
	})

	// Second poll - should return seq 1
	req = httptest.NewRequest("GET", "/__/logs/poll", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), `"seq":1`) {
		t.Errorf("expected seq:1, got %s", w.Body.String())
	}

	// Add another log entry
	s.devLog.Log(LogEntry{
		Route:     "",
		Level:     "warn",
		Filename:  "test.pars",
		Line:      50,
		CallRepr:  "dev.log(y)",
		ValueRepr: "test2",
	})

	// Third poll - should return seq 2
	req = httptest.NewRequest("GET", "/__/logs/poll", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), `"seq":2`) {
		t.Errorf("expected seq:2, got %s", w.Body.String())
	}
}

func TestDevToolsLogsText(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = true

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", "test-commit", &stdout, &stderr)
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
	s, err := New(cfg, "", "test", "test-commit", &stdout, &stderr)
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
	s, err := New(cfg, "", "test", "test-commit", &stdout, &stderr)
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
	if !strings.Contains(body, "Logs") {
		t.Error("expected page to contain 'Logs'")
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
	s, err := New(cfg, "", "test", "test-commit", &stdout, &stderr)
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
	s, err := New(cfg, "", "test", "test-commit", &stdout, &stderr)
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
	if !strings.Contains(body, "No logs") {
		t.Error("expected empty state message")
	}
}

func TestDevToolsWarnLevel(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = true

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", "test-commit", &stdout, &stderr)
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
	// Check for warning content
	if !strings.Contains(body, "warning message") {
		t.Error("expected warning message in body")
	}
}

func TestDevToolsEnv(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = true
	cfg.Server.Port = 8080

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "v1.2.3", "test-commit", &stdout, &stderr)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer s.Close()

	handler := newDevToolsHandler(s)
	req := httptest.NewRequest("GET", "/__/env", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "v1.2.3") {
		t.Error("expected version to be displayed")
	}
	if !strings.Contains(body, "go1.") {
		t.Error("expected Go version to be displayed")
	}
	if !strings.Contains(body, "8080") {
		t.Error("expected port to be displayed")
	}
	if !strings.Contains(body, "ENV") {
		t.Error("expected ENV title")
	}
}

func TestDevToolsEnvNoSecrets(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = true
	// Set a real session secret to test it gets masked
	cfg.Session.Secret = config.NewSecretString("my-super-secret-key")

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", "test-commit", &stdout, &stderr)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer s.Close()

	handler := newDevToolsHandler(s)
	req := httptest.NewRequest("GET", "/__/env", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body := w.Body.String()
	// Session secret should be masked
	if strings.Contains(body, "my-super-secret-key") {
		t.Error("should not expose session secret")
	}
	// Should show masked version
	if !strings.Contains(body, "●●●●●●●●") {
		t.Error("should show masked session secret")
	}
}

func TestDevToolsDBFileDownload(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = true
	// Don't set SQLite in config yet - let server create it

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", "test-commit", &stdout, &stderr)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer s.Close()

	// Now set the SQLite path and create a test database file
	testDBPath := tmpDir + "/test.db"
	s.config.SQLite = testDBPath
	sqliteMagic := []byte{0x53, 0x51, 0x4c, 0x69, 0x74, 0x65, 0x20, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x20, 0x33, 0x00}
	testDBContent := append(sqliteMagic, []byte("rest of database")...)
	if err := os.WriteFile(testDBPath, testDBContent, 0644); err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}

	handler := newDevToolsHandler(s)
	req := httptest.NewRequest("GET", "/__/db/download", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/octet-stream" {
		t.Errorf("expected Content-Type application/octet-stream, got %s", w.Header().Get("Content-Type"))
	}

	disposition := w.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "attachment") || !strings.Contains(disposition, "test.db") {
		t.Errorf("expected Content-Disposition with attachment and test.db, got %s", disposition)
	}

	if !bytes.Equal(w.Body.Bytes(), testDBContent) {
		t.Error("downloaded content does not match database file")
	}
}

func TestDevToolsDBFileUpload(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = true
	// Don't set SQLite in config yet

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", "test-commit", &stdout, &stderr)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer s.Close()

	// Now set the SQLite path and create initial database file
	testDBPath := tmpDir + "/test.db"
	s.config.SQLite = testDBPath
	sqliteMagic := []byte{0x53, 0x51, 0x4c, 0x69, 0x74, 0x65, 0x20, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x20, 0x33, 0x00}
	initialContent := append(sqliteMagic, []byte("initial database")...)
	if err := os.WriteFile(testDBPath, initialContent, 0644); err != nil {
		t.Fatalf("failed to create initial db: %v", err)
	}

	// Create a valid SQLite file content for upload (with magic bytes)
	uploadContent := append(sqliteMagic, []byte("rest of database")...)

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("database", "upload.db")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write(uploadContent); err != nil {
		t.Fatalf("failed to write upload content: %v", err)
	}
	writer.Close()

	handler := newDevToolsHandler(s)
	req := httptest.NewRequest("POST", "/__/db/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Check that response is JSON
	if !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		t.Errorf("expected JSON response, got %s", w.Header().Get("Content-Type"))
	}

	// Check that backup was created
	backupFiles, err := filepath.Glob(testDBPath + ".*.backup")
	if err != nil {
		t.Fatalf("failed to check for backup: %v", err)
	}
	if len(backupFiles) == 0 {
		t.Error("expected backup file to be created")
	}

	// Check that database was replaced
	newContent, err := os.ReadFile(testDBPath)
	if err != nil {
		t.Fatalf("failed to read updated db: %v", err)
	}
	if !bytes.Equal(newContent, uploadContent) {
		t.Error("database content was not updated correctly")
	}
}

func TestDevToolsDBFileUploadInvalidFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = true
	// Don't set SQLite in config yet

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", "test-commit", &stdout, &stderr)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer s.Close()

	// Now set the SQLite path and create initial database
	testDBPath := tmpDir + "/test.db"
	s.config.SQLite = testDBPath
	sqliteMagic := []byte{0x53, 0x51, 0x4c, 0x69, 0x74, 0x65, 0x20, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x20, 0x33, 0x00}
	initialContent := append(sqliteMagic, []byte("initial")...)
	if err := os.WriteFile(testDBPath, initialContent, 0644); err != nil {
		t.Fatalf("failed to create initial db: %v", err)
	}

	// Create invalid file (not SQLite)
	invalidContent := []byte("not a sqlite database")

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("database", "invalid.db")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write(invalidContent); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}
	writer.Close()

	handler := newDevToolsHandler(s)
	req := httptest.NewRequest("POST", "/__/db/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "not a SQLite database") {
		t.Error("expected error message about invalid SQLite file")
	}

	// Verify original database was not changed
	content, err := os.ReadFile(testDBPath)
	if err != nil {
		t.Fatalf("failed to read db: %v", err)
	}
	if !bytes.Equal(content, initialContent) {
		t.Error("database should not have been modified")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	content := []byte("test content")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	dstContent, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read destination: %v", err)
	}

	if !bytes.Equal(dstContent, content) {
		t.Error("copied content does not match source")
	}
}

func TestCopyFileNonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	src := filepath.Join(tmpDir, "nonexistent.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	err := copyFile(src, dst)
	if err == nil {
		t.Error("expected error when copying non-existent file")
	}
}

func TestBytesEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        []byte
		b        []byte
		expected bool
	}{
		{"equal slices", []byte{1, 2, 3}, []byte{1, 2, 3}, true},
		{"different lengths", []byte{1, 2}, []byte{1, 2, 3}, false},
		{"different content", []byte{1, 2, 3}, []byte{1, 2, 4}, false},
		{"empty slices", []byte{}, []byte{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bytesEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("bytesEqual(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
