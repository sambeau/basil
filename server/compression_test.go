package server

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sambeau/basil/config"
)

func TestCompressionHandler_Disabled(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>Hello, World!</body></html>"))
	})

	cfg := config.CompressionConfig{
		Enabled: false,
		Level:   "default",
		MinSize: 1024,
	}

	wrapped := newCompressionHandler(handler, cfg)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Should not be compressed
	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Expected response not to be gzipped when compression is disabled")
	}
	if rec.Body.String() != "<html><body>Hello, World!</body></html>" {
		t.Errorf("Expected uncompressed body, got: %s", rec.Body.String())
	}
}

func TestCompressionHandler_LevelNone(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>Hello, World!</body></html>"))
	})

	cfg := config.CompressionConfig{
		Enabled: true,
		Level:   "none",
		MinSize: 1024,
	}

	wrapped := newCompressionHandler(handler, cfg)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Should not be compressed
	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Expected response not to be gzipped when level is 'none'")
	}
}

func TestCompressionHandler_GzipResponse(t *testing.T) {
	// Generate content larger than MinSize
	largeContent := strings.Repeat("<p>Hello, World!</p>\n", 100)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(largeContent))
	})

	cfg := config.CompressionConfig{
		Enabled: true,
		Level:   "default",
		MinSize: 1024,
	}

	wrapped := newCompressionHandler(handler, cfg)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Should be compressed
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Expected Content-Encoding: gzip header")
	}

	// Decompress and verify content
	reader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to decompress response: %v", err)
	}

	if string(decompressed) != largeContent {
		t.Error("Decompressed content does not match original")
	}
}

func TestCompressionHandler_SmallResponse(t *testing.T) {
	// Small content (less than MinSize)
	smallContent := "Hello"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(smallContent))
	})

	cfg := config.CompressionConfig{
		Enabled: true,
		Level:   "default",
		MinSize: 1024,
	}

	wrapped := newCompressionHandler(handler, cfg)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Should not be compressed (too small)
	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Expected small response not to be gzipped")
	}
	if rec.Body.String() != smallContent {
		t.Errorf("Expected uncompressed body, got: %s", rec.Body.String())
	}
}

func TestCompressionHandler_NoAcceptEncoding(t *testing.T) {
	largeContent := strings.Repeat("<p>Hello, World!</p>\n", 100)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(largeContent))
	})

	cfg := config.CompressionConfig{
		Enabled: true,
		Level:   "default",
		MinSize: 1024,
	}

	wrapped := newCompressionHandler(handler, cfg)

	req := httptest.NewRequest("GET", "/", nil)
	// No Accept-Encoding header
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Should not be compressed (client doesn't support it)
	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Expected response not to be gzipped when client doesn't accept gzip")
	}
	if rec.Body.String() != largeContent {
		t.Errorf("Expected uncompressed body")
	}
}

func TestCompressionHandler_Levels(t *testing.T) {
	tests := []struct {
		level string
		valid bool
	}{
		{"fastest", true},
		{"default", true},
		{"best", true},
		{"invalid", true}, // Falls back to default
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(strings.Repeat("<p>Test</p>\n", 100)))
			})

			cfg := config.CompressionConfig{
				Enabled: true,
				Level:   tt.level,
				MinSize: 100,
			}

			wrapped := newCompressionHandler(handler, cfg)

			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			rec := httptest.NewRecorder()

			wrapped.ServeHTTP(rec, req)

			// Should be compressed for valid levels
			if tt.valid && rec.Header().Get("Content-Encoding") != "gzip" {
				t.Errorf("Expected Content-Encoding: gzip for level %s", tt.level)
			}
		})
	}
}
