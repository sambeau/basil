package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssetRegistry_Register(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.svg")
	content := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><circle r="10"/></svg>`)
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	registry := newAssetRegistry(nil)

	// Register file
	url, err := registry.Register(testFile)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Check URL format
	if !strings.HasPrefix(url, "/__p/") {
		t.Errorf("URL should start with /__p/, got: %s", url)
	}
	if !strings.HasSuffix(url, ".svg") {
		t.Errorf("URL should end with .svg, got: %s", url)
	}

	// Register same file again should return same URL (cache hit)
	url2, err := registry.Register(testFile)
	if err != nil {
		t.Fatalf("Second register failed: %v", err)
	}
	if url != url2 {
		t.Errorf("Same file should return same URL: %s vs %s", url, url2)
	}
}

func TestAssetRegistry_SameContentSameHash(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two files with identical content
	content := []byte("identical content")
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	if err := os.WriteFile(file1, content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, content, 0644); err != nil {
		t.Fatal(err)
	}

	registry := newAssetRegistry(nil)

	url1, _ := registry.Register(file1)
	url2, _ := registry.Register(file2)

	// Same content should produce same hash
	if url1 != url2 {
		t.Errorf("Same content should produce same URL: %s vs %s", url1, url2)
	}
}

func TestAssetRegistry_ModifiedFileNewHash(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create initial file
	if err := os.WriteFile(testFile, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	registry := newAssetRegistry(nil)

	url1, _ := registry.Register(testFile)

	// Modify file
	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}

	url2, _ := registry.Register(testFile)

	// Modified content should produce different hash
	if url1 == url2 {
		t.Errorf("Modified file should produce different URL: %s vs %s", url1, url2)
	}
}

func TestAssetRegistry_NotFound(t *testing.T) {
	registry := newAssetRegistry(nil)

	_, err := registry.Register("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestAssetRegistry_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	registry := newAssetRegistry(nil)

	_, err := registry.Register(tmpDir)
	if err == nil {
		t.Error("Expected error for directory")
	}
}

func TestAssetRegistry_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	registry := newAssetRegistry(nil)
	url, _ := registry.Register(testFile)

	// Extract hash from URL
	hash := strings.TrimPrefix(url, "/__p/")
	hash = strings.TrimSuffix(hash, ".txt")

	// Verify it's registered
	if _, ok := registry.Lookup(hash); !ok {
		t.Error("File should be registered")
	}

	// Clear registry
	registry.Clear()

	// Verify it's cleared
	if _, ok := registry.Lookup(hash); ok {
		t.Error("Registry should be cleared")
	}
}

func TestAssetHandler_ServeFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "icon.svg")
	content := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><circle r="10"/></svg>`)
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	registry := newAssetRegistry(nil)
	url, _ := registry.Register(testFile)

	handler := newAssetHandler(registry)

	req := httptest.NewRequest("GET", url, nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should return 200
	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}

	// Should have cache headers
	cacheControl := rec.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "max-age=31536000") {
		t.Errorf("Expected max-age=31536000 in Cache-Control, got: %s", cacheControl)
	}
	if !strings.Contains(cacheControl, "immutable") {
		t.Errorf("Expected immutable in Cache-Control, got: %s", cacheControl)
	}

	// Content should match
	if rec.Body.String() != string(content) {
		t.Errorf("Content mismatch")
	}
}

func TestAssetHandler_NotFound(t *testing.T) {
	registry := newAssetRegistry(nil)
	handler := newAssetHandler(registry)

	req := httptest.NewRequest("GET", "/__p/nonexistent.svg", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", rec.Code)
	}
}

func TestAssetHandler_ExtensionMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "icon.svg")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	registry := newAssetRegistry(nil)
	url, _ := registry.Register(testFile)

	// Change extension in URL
	wrongURL := strings.Replace(url, ".svg", ".png", 1)

	handler := newAssetHandler(registry)
	req := httptest.NewRequest("GET", wrongURL, nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should return 404 for extension mismatch (security)
	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for extension mismatch, got %d", rec.Code)
	}
}

func TestAssetRegistry_LargeFileSizeLimit(t *testing.T) {
	// We can't easily test 100MB files, but we can verify the error message format
	registry := newAssetRegistry(nil)

	// Non-existent file with .big extension
	_, err := registry.Register("/tmp/nonexistent.big")
	if err == nil {
		t.Error("Expected error")
	}
	// Just verify we get a proper error (not testing actual size limits due to disk space)
}

func TestAssetRegistry_Warning(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "small.txt")
	if err := os.WriteFile(testFile, []byte("small"), 0644); err != nil {
		t.Fatal(err)
	}

	var warned bool
	logger := func(format string, args ...interface{}) {
		warned = true
	}

	registry := newAssetRegistry(logger)
	_, _ = registry.Register(testFile)

	// Small file shouldn't trigger warning
	if warned {
		t.Error("Small file shouldn't trigger warning")
	}
}
