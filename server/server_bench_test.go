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

// Root/Static Fallback Benchmarks

func BenchmarkRootHandler_StaticHit(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()
	staticDir := filepath.Join(dir, "public")
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		b.Fatalf("failed to create static dir: %v", err)
	}

	// Create a static file
	staticContent := "body { margin: 0; padding: 0; }"
	staticFile := filepath.Join(staticDir, "style.css")
	if err := os.WriteFile(staticFile, []byte(staticContent), 0o644); err != nil {
		b.Fatalf("failed to write static file: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  false,
		},
		Static: []config.StaticRoute{
			{Path: "/static/", Root: staticDir},
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "text",
			Output: "stderr",
		},
	}

	srv := benchServerRoot(b, cfg)

	req := httptest.NewRequest(http.MethodGet, "/static/style.css", http.NoBody)

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

func BenchmarkRootHandler_StaticMiss_FallbackToRoute(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()

	// Create static directory (but without the file we'll request)
	staticDir := filepath.Join(dir, "public")
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		b.Fatalf("failed to create static dir: %v", err)
	}

	// Create a dummy file so the directory isn't empty
	dummyFile := filepath.Join(staticDir, "exists.txt")
	if err := os.WriteFile(dummyFile, []byte("dummy"), 0o644); err != nil {
		b.Fatalf("failed to write dummy file: %v", err)
	}

	// Create a Parsley handler for the fallback
	scriptPath := filepath.Join(dir, "index.parsley")
	scriptContent := `<div>"Dynamic content"</div>`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o644); err != nil {
		b.Fatalf("failed to write script: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  false,
		},
		PublicDir: staticDir,
		Routes: []config.Route{
			{Path: "/", Handler: scriptPath, PublicDir: staticDir},
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "text",
			Output: "stderr",
		},
	}

	srv := benchServerRoot(b, cfg)

	// Request a path that doesn't exist as static, falls back to route
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)

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

func BenchmarkRootHandler_StaticMiss_404(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()
	staticDir := filepath.Join(dir, "public")
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		b.Fatalf("failed to create static dir: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  false,
		},
		Static: []config.StaticRoute{
			{Path: "/static/", Root: staticDir},
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "text",
			Output: "stderr",
		},
	}

	srv := benchServerRoot(b, cfg)

	// Request a file that doesn't exist
	req := httptest.NewRequest(http.MethodGet, "/static/nonexistent.css", http.NoBody)

	b.ResetTimer()
	for b.Loop() {
		rec := httptest.NewRecorder()
		srv.mux.ServeHTTP(rec, req)
	}
}

func BenchmarkRootHandler_MultipleStaticFiles(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()
	staticDir := filepath.Join(dir, "public")
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		b.Fatalf("failed to create static dir: %v", err)
	}

	// Create multiple static files
	files := []string{"app.js", "style.css", "logo.png", "data.json"}
	for _, f := range files {
		path := filepath.Join(staticDir, f)
		if err := os.WriteFile(path, []byte("content of "+f), 0o644); err != nil {
			b.Fatalf("failed to write %s: %v", f, err)
		}
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  false,
		},
		Static: []config.StaticRoute{
			{Path: "/static/", Root: staticDir},
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "text",
			Output: "stderr",
		},
	}

	srv := benchServerRoot(b, cfg)

	// Warm up all files
	for _, f := range files {
		req := httptest.NewRequest(http.MethodGet, "/static/"+f, http.NoBody)
		rec := httptest.NewRecorder()
		srv.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("warmup failed for %s with status %d", f, rec.Code)
		}
	}

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		// Cycle through different files
		f := files[i%len(files)]
		req := httptest.NewRequest(http.MethodGet, "/static/"+f, http.NoBody)
		rec := httptest.NewRecorder()
		srv.mux.ServeHTTP(rec, req)
	}
}

func BenchmarkRootHandler_SingleFileRoute(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()

	// Create a favicon file
	faviconContent := []byte{0x00, 0x00, 0x01, 0x00} // Minimal ICO header
	faviconPath := filepath.Join(dir, "favicon.ico")
	if err := os.WriteFile(faviconPath, faviconContent, 0o644); err != nil {
		b.Fatalf("failed to write favicon: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  false,
		},
		Static: []config.StaticRoute{
			{Path: "/favicon.ico", File: faviconPath},
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "text",
			Output: "stderr",
		},
	}

	srv := benchServerRoot(b, cfg)

	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", http.NoBody)

	// Warm up
	warmRec := httptest.NewRecorder()
	srv.mux.ServeHTTP(warmRec, req)
	if warmRec.Code != http.StatusOK {
		b.Fatalf("warmup failed with status %d", warmRec.Code)
	}

	b.ResetTimer()
	for b.Loop() {
		rec := httptest.NewRecorder()
		srv.mux.ServeHTTP(rec, req)
	}
}

func BenchmarkRootHandler_NestedStaticPath(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()
	staticDir := filepath.Join(dir, "public")

	// Create nested directory structure
	nestedDir := filepath.Join(staticDir, "assets", "images", "icons")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		b.Fatalf("failed to create nested dir: %v", err)
	}

	// Create a file in the nested path
	iconContent := "fake icon data"
	iconPath := filepath.Join(nestedDir, "menu.svg")
	if err := os.WriteFile(iconPath, []byte(iconContent), 0o644); err != nil {
		b.Fatalf("failed to write icon: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Dev:  false,
		},
		Static: []config.StaticRoute{
			{Path: "/static/", Root: staticDir},
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "text",
			Output: "stderr",
		},
	}

	srv := benchServerRoot(b, cfg)

	req := httptest.NewRequest(http.MethodGet, "/static/assets/images/icons/menu.svg", http.NoBody)

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

// benchServerRoot creates a minimal server for root/static benchmarking.
func benchServerRoot(b *testing.B, cfg *config.Config) *Server {
	b.Helper()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	srv, err := New(cfg, "", "bench", "bench-commit", stdout, stderr)
	if err != nil {
		b.Fatalf("failed to create server: %v", err)
	}

	return srv
}
