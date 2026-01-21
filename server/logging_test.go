package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLoggerText(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create logger with text format
	var buf bytes.Buffer
	logger := newRequestLogger(handler, &buf, "text")

	// Make a request
	req := httptest.NewRequest("GET", "/test/path", nil)
	rec := httptest.NewRecorder()
	logger.ServeHTTP(rec, req)

	// Check log output
	log := buf.String()
	if !strings.Contains(log, "GET") {
		t.Errorf("log should contain method GET: %s", log)
	}
	if !strings.Contains(log, "/test/path") {
		t.Errorf("log should contain path /test/path: %s", log)
	}
	if !strings.Contains(log, "200") {
		t.Errorf("log should contain status 200: %s", log)
	}
}

func TestRequestLoggerJSON(t *testing.T) {
	// Create a handler that returns 201
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Created"))
	})

	// Create logger with JSON format
	var buf bytes.Buffer
	logger := newRequestLogger(handler, &buf, "json")

	// Make a request
	req := httptest.NewRequest("POST", "/api/users", nil)
	req.Header.Set("User-Agent", "test-agent")
	rec := httptest.NewRecorder()
	logger.ServeHTTP(rec, req)

	// Parse JSON log
	var entry RequestLogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON log: %v", err)
	}

	if entry.Method != "POST" {
		t.Errorf("expected method POST, got %s", entry.Method)
	}
	if entry.Path != "/api/users" {
		t.Errorf("expected path /api/users, got %s", entry.Path)
	}
	if entry.Status != 201 {
		t.Errorf("expected status 201, got %d", entry.Status)
	}
	if entry.UserAgent != "test-agent" {
		t.Errorf("expected user-agent test-agent, got %s", entry.UserAgent)
	}
	if entry.Timestamp == "" {
		t.Error("timestamp should not be empty")
	}
	if entry.DurationMs < 0 {
		t.Error("duration_ms should be non-negative")
	}
}

func TestRequestLoggerXForwardedFor(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	var buf bytes.Buffer
	logger := newRequestLogger(handler, &buf, "json")

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.195")
	rec := httptest.NewRecorder()
	logger.ServeHTTP(rec, req)

	var entry RequestLogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON log: %v", err)
	}

	if entry.ClientIP != "203.0.113.195" {
		t.Errorf("expected client IP from X-Forwarded-For, got %s", entry.ClientIP)
	}
}

func TestRequestLoggerCapturesStatus(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedStatus int
	}{
		{"OK", http.StatusOK, 200},
		{"Not Found", http.StatusNotFound, 404},
		{"Server Error", http.StatusInternalServerError, 500},
		{"Redirect", http.StatusFound, 302},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			var buf bytes.Buffer
			logger := newRequestLogger(handler, &buf, "json")

			req := httptest.NewRequest("GET", "/", nil)
			rec := httptest.NewRecorder()
			logger.ServeHTTP(rec, req)

			var entry RequestLogEntry
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("failed to parse JSON log: %v", err)
			}

			if entry.Status != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, entry.Status)
			}
		})
	}
}

func TestRequestLoggerDefaultFormat(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Empty format should default to text
	var buf bytes.Buffer
	logger := newRequestLogger(handler, &buf, "")

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	logger.ServeHTTP(rec, req)

	// Text format should NOT be valid JSON
	var entry RequestLogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err == nil {
		t.Error("default format should be text, not JSON")
	}
}

func TestRequestLoggerSkipsDevtools(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`OK`))
	})

	var buf bytes.Buffer
	logger := newRequestLogger(handler, &buf, "text")

	// All /__* paths should not be logged
	devPaths := []string{
		"/__livereload",
		"/__/logs",
		"/__/db",
		"/__/env",
		"/__",
	}

	for _, path := range devPaths {
		buf.Reset()
		req := httptest.NewRequest("GET", path, nil)
		rec := httptest.NewRecorder()
		logger.ServeHTTP(rec, req)

		// Response should still work
		if rec.Code != http.StatusOK {
			t.Errorf("%s: expected status 200, got %d", path, rec.Code)
		}

		// But log should be empty
		if buf.Len() != 0 {
			t.Errorf("%s: expected no log output, got: %s", path, buf.String())
		}
	}
}
