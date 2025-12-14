package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssetBundle_Discovery(t *testing.T) {
	// Create temporary test directory structure
	tmpDir := t.TempDir()
	handlersDir := filepath.Join(tmpDir, "handlers")

	// Create test files
	os.MkdirAll(filepath.Join(handlersDir, "components"), 0755)
	os.MkdirAll(filepath.Join(handlersDir, "pages"), 0755)

	os.WriteFile(filepath.Join(handlersDir, "base.css"), []byte("body { margin: 0; }"), 0644)
	os.WriteFile(filepath.Join(handlersDir, "utils.js"), []byte("console.log('utils');"), 0644)
	os.WriteFile(filepath.Join(handlersDir, "components", "button.css"), []byte(".button { }"), 0644)
	os.WriteFile(filepath.Join(handlersDir, "components", "card.js"), []byte("console.log('card');"), 0644)
	os.WriteFile(filepath.Join(handlersDir, "pages", "about.css"), []byte(".about { }"), 0644)

	bundle := NewAssetBundle(handlersDir, false, "public")
	err := bundle.Rebuild()
	if err != nil {
		t.Fatalf("Rebuild() failed: %v", err)
	}

	// Check CSS files discovered
	if len(bundle.cssFiles) != 3 {
		t.Errorf("Expected 3 CSS files, got %d", len(bundle.cssFiles))
	}

	// Check JS files discovered
	if len(bundle.jsFiles) != 2 {
		t.Errorf("Expected 2 JS files, got %d", len(bundle.jsFiles))
	}

	// Check CSS content
	cssContent := string(bundle.cssContent)
	if !strings.Contains(cssContent, "body { margin: 0; }") {
		t.Error("CSS content missing base.css")
	}
	if !strings.Contains(cssContent, ".button { }") {
		t.Error("CSS content missing button.css")
	}

	// Check JS content
	jsContent := string(bundle.jsContent)
	if !strings.Contains(jsContent, "console.log('utils')") {
		t.Error("JS content missing utils.js")
	}
	if !strings.Contains(jsContent, "console.log('card')") {
		t.Error("JS content missing card.js")
	}
}

func TestAssetBundle_DevModeComments(t *testing.T) {
	tmpDir := t.TempDir()
	handlersDir := filepath.Join(tmpDir, "handlers")
	os.MkdirAll(handlersDir, 0755)
	os.WriteFile(filepath.Join(handlersDir, "test.css"), []byte(".test { }"), 0644)

	bundle := NewAssetBundle(handlersDir, true, "public") // dev mode
	err := bundle.Rebuild()
	if err != nil {
		t.Fatalf("Rebuild() failed: %v", err)
	}

	cssContent := string(bundle.cssContent)
	if !strings.Contains(cssContent, "handlers/test.css") {
		t.Error("Dev mode should include source file comments")
	}
}

func TestAssetBundle_ProductionNoComments(t *testing.T) {
	tmpDir := t.TempDir()
	handlersDir := filepath.Join(tmpDir, "handlers")
	os.MkdirAll(handlersDir, 0755)
	os.WriteFile(filepath.Join(handlersDir, "test.css"), []byte(".test { }"), 0644)

	bundle := NewAssetBundle(handlersDir, false, "public") // production mode
	err := bundle.Rebuild()
	if err != nil {
		t.Fatalf("Rebuild() failed: %v", err)
	}

	cssContent := string(bundle.cssContent)
	if strings.Contains(cssContent, "handlers/test.css") {
		t.Error("Production mode should not include source file comments")
	}
}

func TestAssetBundle_ExcludesHidden(t *testing.T) {
	tmpDir := t.TempDir()
	handlersDir := filepath.Join(tmpDir, "handlers")
	os.MkdirAll(filepath.Join(handlersDir, ".hidden"), 0755)
	os.WriteFile(filepath.Join(handlersDir, ".test.css"), []byte(".hidden { }"), 0644)
	os.WriteFile(filepath.Join(handlersDir, ".hidden", "secret.css"), []byte(".secret { }"), 0644)
	os.WriteFile(filepath.Join(handlersDir, "normal.css"), []byte(".normal { }"), 0644)

	bundle := NewAssetBundle(handlersDir, false, "public")
	err := bundle.Rebuild()
	if err != nil {
		t.Fatalf("Rebuild() failed: %v", err)
	}

	if len(bundle.cssFiles) != 1 {
		t.Errorf("Expected 1 CSS file (excluding hidden), got %d", len(bundle.cssFiles))
	}

	cssContent := string(bundle.cssContent)
	if strings.Contains(cssContent, ".hidden") || strings.Contains(cssContent, ".secret") {
		t.Error("Bundle should exclude hidden files")
	}
}

func TestAssetBundle_EmptyBundle(t *testing.T) {
	tmpDir := t.TempDir()
	handlersDir := filepath.Join(tmpDir, "handlers")
	os.MkdirAll(handlersDir, 0755)

	bundle := NewAssetBundle(handlersDir, false, "public")
	err := bundle.Rebuild()
	if err != nil {
		t.Fatalf("Rebuild() failed: %v", err)
	}

	if bundle.CSSUrl() != "" {
		t.Error("Empty bundle should return empty CSS URL")
	}
	if bundle.JSUrl() != "" {
		t.Error("Empty bundle should return empty JS URL")
	}
}

func TestAssetBundle_HashComputation(t *testing.T) {
	tmpDir := t.TempDir()
	handlersDir := filepath.Join(tmpDir, "handlers")
	os.MkdirAll(handlersDir, 0755)
	os.WriteFile(filepath.Join(handlersDir, "test.css"), []byte(".test { }"), 0644)

	bundle := NewAssetBundle(handlersDir, false, "public")
	err := bundle.Rebuild()
	if err != nil {
		t.Fatalf("Rebuild() failed: %v", err)
	}

	hash1 := bundle.cssHash

	// Modify content
	os.WriteFile(filepath.Join(handlersDir, "test.css"), []byte(".test { color: red; }"), 0644)
	err = bundle.Rebuild()
	if err != nil {
		t.Fatalf("Rebuild() failed: %v", err)
	}

	hash2 := bundle.cssHash

	if hash1 == hash2 {
		t.Error("Hash should change when content changes")
	}

	if len(hash1) != 8 {
		t.Errorf("Hash should be 8 characters, got %d", len(hash1))
	}
}

func TestAssetBundle_URLs(t *testing.T) {
	tmpDir := t.TempDir()
	handlersDir := filepath.Join(tmpDir, "handlers")
	os.MkdirAll(handlersDir, 0755)
	os.WriteFile(filepath.Join(handlersDir, "test.css"), []byte(".test { }"), 0644)
	os.WriteFile(filepath.Join(handlersDir, "test.js"), []byte("console.log('test');"), 0644)

	bundle := NewAssetBundle(handlersDir, false, "public")
	err := bundle.Rebuild()
	if err != nil {
		t.Fatalf("Rebuild() failed: %v", err)
	}

	cssURL := bundle.CSSUrl()
	if !strings.HasPrefix(cssURL, "/__site.css?v=") {
		t.Errorf("Unexpected CSS URL format: %s", cssURL)
	}

	jsURL := bundle.JSUrl()
	if !strings.HasPrefix(jsURL, "/__site.js?v=") {
		t.Errorf("Unexpected JS URL format: %s", jsURL)
	}
}

func TestAssetBundle_ExcludesConfiguredPublicDir(t *testing.T) {
	tmpDir := t.TempDir()
	handlersDir := filepath.Join(tmpDir, "handlers")
	os.MkdirAll(filepath.Join(handlersDir, "static"), 0755)
	os.MkdirAll(filepath.Join(handlersDir, "public"), 0755)
	os.WriteFile(filepath.Join(handlersDir, "app.css"), []byte(".app { }"), 0644)
	os.WriteFile(filepath.Join(handlersDir, "static", "vendor.css"), []byte(".vendor { }"), 0644)
	os.WriteFile(filepath.Join(handlersDir, "public", "bootstrap.css"), []byte(".bootstrap { }"), 0644)

	// Test with "static" as public directory name
	bundle := NewAssetBundle(handlersDir, false, "static")
	err := bundle.Rebuild()
	if err != nil {
		t.Fatalf("Rebuild() failed: %v", err)
	}

	cssContent := string(bundle.cssContent)

	// Should include app.css and public/bootstrap.css
	if !strings.Contains(cssContent, ".app") {
		t.Error("Bundle should include app.css")
	}
	if !strings.Contains(cssContent, ".bootstrap") {
		t.Error("Bundle should include public/bootstrap.css (not configured as public dir)")
	}

	// Should exclude static/vendor.css
	if strings.Contains(cssContent, ".vendor") {
		t.Error("Bundle should exclude static/ directory (configured as public dir)")
	}
}
