package images

import (
	"path/filepath"
	"regexp"
	"sync"
	"testing"
)

func TestRegistry_Transform(t *testing.T) {
	r, _ := newTestRegistry(t, false)
	srcDir := t.TempDir()

	src := createTestImage(t, srcDir, "test.png", 200, 100, "png")

	url, err := r.Transform(src, nil)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}

	// URL must match /__img/{16 hex chars}.{ext}
	matched, _ := regexp.MatchString(`^/__img/[0-9a-f]{16}\.\w+$`, url)
	if !matched {
		t.Fatalf("unexpected URL format: %s", url)
	}
}

func TestRegistry_Transform_CacheHit(t *testing.T) {
	r, _ := newTestRegistry(t, false)
	srcDir := t.TempDir()

	src := createTestImage(t, srcDir, "test.png", 200, 100, "png")

	url1, err := r.Transform(src, nil)
	if err != nil {
		t.Fatalf("first Transform: %v", err)
	}

	url2, err := r.Transform(src, nil)
	if err != nil {
		t.Fatalf("second Transform: %v", err)
	}

	if url1 != url2 {
		t.Fatalf("cache hit returned different URL: %s vs %s", url1, url2)
	}

	count, _, err := r.CacheStats()
	if err != nil {
		t.Fatalf("CacheStats: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 cached file, got %d", count)
	}
}

func TestRegistry_Transform_WithOptions(t *testing.T) {
	r, _ := newTestRegistry(t, false)
	srcDir := t.TempDir()

	src := createTestImage(t, srcDir, "test.png", 400, 300, "png")

	urlDefault, err := r.Transform(src, nil)
	if err != nil {
		t.Fatalf("Transform default: %v", err)
	}

	urlResized, err := r.Transform(src, map[string]any{"width": int64(100)})
	if err != nil {
		t.Fatalf("Transform resized: %v", err)
	}

	if urlDefault == urlResized {
		t.Fatal("expected different URLs for different options")
	}
}

func TestRegistry_Transform_ConfigDefaults(t *testing.T) {
	srcDir := t.TempDir()
	src := createTestImage(t, srcDir, "test.png", 200, 100, "png")

	// Registry with default format = jpeg
	cacheDir := t.TempDir()
	r := NewRegistry(cacheDir, 4096, 4096, 90, "jpeg", false, nil)

	url, err := r.Transform(src, nil)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}

	ext := filepath.Ext(url)
	if ext != ".jpg" {
		t.Fatalf("expected .jpg extension from default format, got %s", ext)
	}
}

func TestRegistry_Info(t *testing.T) {
	srcDir := t.TempDir()
	src := createTestImage(t, srcDir, "test.png", 320, 240, "png")

	r, _ := newTestRegistry(t, false)

	info, err := r.Info(src)
	if err != nil {
		t.Fatalf("Info: %v", err)
	}

	if w := info["width"].(int64); w != 320 {
		t.Fatalf("expected width 320, got %d", w)
	}
	if h := info["height"].(int64); h != 240 {
		t.Fatalf("expected height 240, got %d", h)
	}
	if f := info["format"].(string); f != "png" {
		t.Fatalf("expected format png, got %s", f)
	}
	if _, ok := info["orientation"]; !ok {
		t.Fatal("expected orientation key in info")
	}
}

func TestRegistry_Lookup(t *testing.T) {
	r, _ := newTestRegistry(t, false)
	srcDir := t.TempDir()

	src := createTestImage(t, srcDir, "test.png", 200, 100, "png")

	url, err := r.Transform(src, nil)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}

	// Extract the hash key from the URL: /__img/{key}.{ext}
	base := filepath.Base(url)
	ext := filepath.Ext(base)
	key := base[:len(base)-len(ext)]

	path, ok := r.Lookup(key)
	if !ok {
		t.Fatal("Lookup failed after Transform")
	}
	if path == "" {
		t.Fatal("Lookup returned empty path")
	}
}

func TestRegistry_Clear(t *testing.T) {
	r, _ := newTestRegistry(t, false)
	srcDir := t.TempDir()

	src := createTestImage(t, srcDir, "test.png", 200, 100, "png")

	url, err := r.Transform(src, nil)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}

	base := filepath.Base(url)
	ext := filepath.Ext(base)
	key := base[:len(base)-len(ext)]

	r.Clear()

	_, ok := r.Lookup(key)
	if ok {
		t.Fatal("Lookup succeeded after Clear, expected failure")
	}
}

func TestRegistry_CacheStats(t *testing.T) {
	r, _ := newTestRegistry(t, false)
	srcDir := t.TempDir()

	count, size, err := r.CacheStats()
	if err != nil {
		t.Fatalf("CacheStats: %v", err)
	}
	if count != 0 || size != 0 {
		t.Fatalf("expected empty cache, got count=%d size=%d", count, size)
	}

	src := createTestImage(t, srcDir, "test.png", 200, 100, "png")

	_, err = r.Transform(src, nil)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}

	count, size, err = r.CacheStats()
	if err != nil {
		t.Fatalf("CacheStats: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 cached file, got %d", count)
	}
	if size <= 0 {
		t.Fatalf("expected positive cache size, got %d", size)
	}
}

func TestRegistry_ConcurrentTransform(t *testing.T) {
	r, _ := newTestRegistry(t, false)
	srcDir := t.TempDir()

	src := createTestImage(t, srcDir, "test.png", 200, 100, "png")

	const N = 20
	var wg sync.WaitGroup
	urls := make([]string, N)
	errs := make([]error, N)

	wg.Add(N)
	for i := range N {
		go func(idx int) {
			defer wg.Done()
			urls[idx], errs[idx] = r.Transform(src, map[string]any{"width": int64(50)})
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}

	// All URLs must be identical
	for i := 1; i < N; i++ {
		if urls[i] != urls[0] {
			t.Fatalf("goroutine %d returned different URL: %s vs %s", i, urls[i], urls[0])
		}
	}

	// Singleflight should deduplicate: only 1 cached file
	count, _, err := r.CacheStats()
	if err != nil {
		t.Fatalf("CacheStats: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 cached file after concurrent transforms, got %d", count)
	}
}
