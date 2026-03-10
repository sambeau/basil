package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/server/config"
)

// API Handler Benchmarks

func BenchmarkAPIHandler_ServeHTTP_SimpleGET(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()
	apiPath := filepath.Join(dir, "api.parsley")
	// Minimal API module with a get handler
	apiContent := `
let api = import @std/api

export let get = api.public(fn(req) {
    {
        status: 200,
        body: {message: "Hello, API!"}
    }
})
`
	if err := os.WriteFile(apiPath, []byte(apiContent), 0o644); err != nil {
		b.Fatalf("failed to write API script: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  false,
		},
		Routes: []config.Route{
			{
				Path:    "/api/test",
				Handler: apiPath,
				Type:    "api",
			},
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "text",
			Output: "stderr",
		},
	}

	srv := benchServerAPI(b, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)

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

func BenchmarkAPIHandler_ServeHTTP_SmallResponse(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()
	apiPath := filepath.Join(dir, "users.parsley")
	// API that returns a small structured response
	apiContent := `
let api = import @std/api

export let get = api.public(fn(req) {
    {
        status: 200,
        body: {
            users: [
                {id: 1, name: "Alice", email: "alice@example.com"},
                {id: 2, name: "Bob", email: "bob@example.com"},
                {id: 3, name: "Carol", email: "carol@example.com"}
            ],
            total: 3,
            page: 1
        }
    }
})
`
	if err := os.WriteFile(apiPath, []byte(apiContent), 0o644); err != nil {
		b.Fatalf("failed to write API script: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  false,
		},
		Routes: []config.Route{
			{
				Path:    "/api/users",
				Handler: apiPath,
				Type:    "api",
			},
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "text",
			Output: "stderr",
		},
	}

	srv := benchServerAPI(b, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/users", http.NoBody)

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

func BenchmarkAPIHandler_ServeHTTP_POST(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()
	apiPath := filepath.Join(dir, "items.parsley")
	// API with POST handler that processes request body
	apiContent := `
let api = import @std/api

export let post = api.public(fn(req) {
    {
        status: 201,
        body: {
            id: 999,
            message: "Created successfully"
        }
    }
})
`
	if err := os.WriteFile(apiPath, []byte(apiContent), 0o644); err != nil {
		b.Fatalf("failed to write API script: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  false,
		},
		Routes: []config.Route{
			{
				Path:    "/api/items",
				Handler: apiPath,
				Type:    "api",
			},
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "text",
			Output: "stderr",
		},
	}

	srv := benchServerAPI(b, cfg)

	jsonBody := `{"name": "Test Item", "price": 29.99}`

	// Warm up
	warmReq := httptest.NewRequest(http.MethodPost, "/api/items", strings.NewReader(jsonBody))
	warmReq.Header.Set("Content-Type", "application/json")
	warmRec := httptest.NewRecorder()
	srv.mux.ServeHTTP(warmRec, warmReq)
	if warmRec.Code != http.StatusCreated {
		b.Fatalf("warmup failed with status %d: %s", warmRec.Code, warmRec.Body.String())
	}

	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/api/items", strings.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.mux.ServeHTTP(rec, req)
	}
}

func BenchmarkAPIHandler_ServeHTTP_WithComputation(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()
	apiPath := filepath.Join(dir, "compute.parsley")
	// API that performs some computation
	apiContent := `
let api = import @std/api

export let get = api.public(fn(req) {
    let numbers = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20]
    let doubled = numbers.map(fn(n) { n * 2 })
    let filtered = doubled.filter(fn(n) { n > 20 })
    let sum = filtered.reduce(fn(acc, n) { acc + n }, 0)

    {
        status: 200,
        body: {
            result: sum,
            count: filtered.length()
        }
    }
})
`
	if err := os.WriteFile(apiPath, []byte(apiContent), 0o644); err != nil {
		b.Fatalf("failed to write API script: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  false,
		},
		Routes: []config.Route{
			{
				Path:    "/api/compute",
				Handler: apiPath,
				Type:    "api",
			},
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "text",
			Output: "stderr",
		},
	}

	srv := benchServerAPI(b, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/compute", http.NoBody)

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

func BenchmarkAPIHandler_ServeHTTP_GetById(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()
	apiPath := filepath.Join(dir, "resource.parsley")
	// API with getById handler
	apiContent := `
let api = import @std/api

export let getById = api.public(fn(req) {
    let id = req.params.id
    {
        status: 200,
        body: {
            id: id,
            name: "Resource " + id,
            created: "2024-01-01T00:00:00Z"
        }
    }
})
`
	if err := os.WriteFile(apiPath, []byte(apiContent), 0o644); err != nil {
		b.Fatalf("failed to write API script: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  false,
		},
		Routes: []config.Route{
			{
				Path:    "/api/resource",
				Handler: apiPath,
				Type:    "api",
			},
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "text",
			Output: "stderr",
		},
	}

	srv := benchServerAPI(b, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/resource/123", http.NoBody)

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

// benchServerAPI creates a minimal server for API benchmarking.
func benchServerAPI(b *testing.B, cfg *config.Config) *Server {
	b.Helper()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	srv, err := New(cfg, "", "bench", "bench-commit", stdout, stderr)
	if err != nil {
		b.Fatalf("failed to create server: %v", err)
	}

	return srv
}
