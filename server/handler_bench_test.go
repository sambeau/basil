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

// benchServer creates a minimal server for benchmarking.
// Returns the server and a cleanup function.
func benchServer(b *testing.B, cfg *config.Config) *Server {
	b.Helper()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	srv, err := New(cfg, "", "bench", "bench-commit", stdout, stderr)
	if err != nil {
		b.Fatalf("failed to create server: %v", err)
	}

	return srv
}

// Page Handler Benchmarks

func BenchmarkParsleyHandler_ServeHTTP_Simple(b *testing.B) {
	b.ReportAllocs()

	// Create temp directory with a minimal Parsley script
	dir := b.TempDir()
	scriptPath := filepath.Join(dir, "index.parsley")
	scriptContent := `<div class="greeting"><h1>"Hello, World!"</h1></div>`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o644); err != nil {
		b.Fatalf("failed to write script: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  false, // Production mode for AST caching
		},
		Routes: []config.Route{
			{Path: "/", Handler: scriptPath},
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "text",
			Output: "stderr",
		},
	}

	srv := benchServer(b, cfg)

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	// Warm up: run once to populate AST cache
	warmRec := httptest.NewRecorder()
	srv.mux.ServeHTTP(warmRec, req)
	if warmRec.Code != http.StatusOK {
		b.Fatalf("warmup failed with status %d: %s", warmRec.Code, warmRec.Body.String())
	}

	b.ResetTimer()
	for b.Loop() {
		rec := httptest.NewRecorder()
		srv.mux.ServeHTTP(rec, req)
	}
}

func BenchmarkParsleyHandler_ServeHTTP_WithVariables(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()
	scriptPath := filepath.Join(dir, "page.parsley")
	// Script with variable bindings and iteration
	scriptContent := `
let title = "Welcome"
let items = ["one", "two", "three"]
let count = items.length()

<div class="container">
    <h1>title</h1>
    <p>"Item count: " + toString(count)</p>
    <ul>
    for (item in items) {
        <li>item</li>
    }
    </ul>
</div>
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o644); err != nil {
		b.Fatalf("failed to write script: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  false,
		},
		Routes: []config.Route{
			{Path: "/page", Handler: scriptPath},
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "text",
			Output: "stderr",
		},
	}

	srv := benchServer(b, cfg)

	req := httptest.NewRequest(http.MethodGet, "/page", http.NoBody)

	// Warm up
	warmRec := httptest.NewRecorder()
	srv.mux.ServeHTTP(warmRec, req)
	if warmRec.Code != http.StatusOK {
		b.Fatalf("warmup failed with status %d: %s", warmRec.Code, warmRec.Body.String())
	}

	b.ResetTimer()
	for b.Loop() {
		rec := httptest.NewRecorder()
		srv.mux.ServeHTTP(rec, req)
	}
}

func BenchmarkParsleyHandler_ServeHTTP_ResponseCacheHit(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()
	scriptPath := filepath.Join(dir, "cached.parsley")
	scriptContent := `<div class="cached-content"><p>"This response is cached"</p></div>`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o644); err != nil {
		b.Fatalf("failed to write script: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  false,
		},
		Routes: []config.Route{
			{
				Path:    "/cached",
				Handler: scriptPath,
				Cache:   300, // 5 minute cache TTL
			},
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "text",
			Output: "stderr",
		},
	}

	srv := benchServer(b, cfg)

	req := httptest.NewRequest(http.MethodGet, "/cached", http.NoBody)

	// Warm up: first request populates the response cache
	warmRec := httptest.NewRecorder()
	srv.mux.ServeHTTP(warmRec, req)
	if warmRec.Code != http.StatusOK {
		b.Fatalf("warmup failed with status %d: %s", warmRec.Code, warmRec.Body.String())
	}

	// Verify cache is working
	checkRec := httptest.NewRecorder()
	srv.mux.ServeHTTP(checkRec, req)
	if checkRec.Header().Get("X-Cache") != "HIT" {
		b.Log("Warning: response cache may not be active")
	}

	b.ResetTimer()
	for b.Loop() {
		rec := httptest.NewRecorder()
		srv.mux.ServeHTTP(rec, req)
	}
}

func BenchmarkParsleyHandler_ServeHTTP_WithQueryParams(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()
	scriptPath := filepath.Join(dir, "search.parsley")
	// Script that reads query parameters via @params
	scriptContent := `
let query = @params["q"] ?? "default"
let page = @params["page"] ?? "1"

<div class="search-results">
    <h2>"Search: " + query</h2>
    <p>"Page: " + page</p>
</div>
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o644); err != nil {
		b.Fatalf("failed to write script: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  false,
		},
		Routes: []config.Route{
			{Path: "/search", Handler: scriptPath},
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "text",
			Output: "stderr",
		},
	}

	srv := benchServer(b, cfg)

	req := httptest.NewRequest(http.MethodGet, "/search?q=test&page=2&limit=20", http.NoBody)

	// Warm up
	warmRec := httptest.NewRecorder()
	srv.mux.ServeHTTP(warmRec, req)
	if warmRec.Code != http.StatusOK {
		b.Fatalf("warmup failed with status %d: %s", warmRec.Code, warmRec.Body.String())
	}

	b.ResetTimer()
	for b.Loop() {
		rec := httptest.NewRecorder()
		srv.mux.ServeHTTP(rec, req)
	}
}

func BenchmarkParsleyHandler_ServeHTTP_JSONResponse(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()
	scriptPath := filepath.Join(dir, "data.parsley")
	// Script that returns a dictionary (rendered as JSON)
	scriptContent := `{
    status: "ok",
    data: {
        id: 123,
        name: "Test User",
        email: "test@example.com",
        active: true
    },
    meta: {
        version: "1.0",
        timestamp: 1234567890
    }
}`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o644); err != nil {
		b.Fatalf("failed to write script: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  false,
		},
		Routes: []config.Route{
			{Path: "/data", Handler: scriptPath},
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "text",
			Output: "stderr",
		},
	}

	srv := benchServer(b, cfg)

	req := httptest.NewRequest(http.MethodGet, "/data", http.NoBody)

	// Warm up
	warmRec := httptest.NewRecorder()
	srv.mux.ServeHTTP(warmRec, req)
	if warmRec.Code != http.StatusOK {
		b.Fatalf("warmup failed with status %d: %s", warmRec.Code, warmRec.Body.String())
	}

	b.ResetTimer()
	for b.Loop() {
		rec := httptest.NewRecorder()
		srv.mux.ServeHTTP(rec, req)
	}
}
