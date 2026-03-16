package images

import (
	"path/filepath"
	"regexp"
	"strings"
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

func TestRegistry_BlurPlaceholder(t *testing.T) {
	r, _ := newTestRegistry(t, false)
	srcDir := t.TempDir()

	src := createTestImage(t, srcDir, "blur_test.jpg", 400, 300, "jpeg")

	dataURI, err := r.BlurPlaceholder(src)
	if err != nil {
		t.Fatalf("BlurPlaceholder: %v", err)
	}

	// Should return a data URI
	if !strings.HasPrefix(dataURI, "data:image/jpeg;base64,") {
		t.Fatalf("expected data URI, got: %s", dataURI[:min(len(dataURI), 50)])
	}
}

func TestRegistry_BlurPlaceholder_CacheHit(t *testing.T) {
	r, _ := newTestRegistry(t, false)
	srcDir := t.TempDir()

	src := createTestImage(t, srcDir, "blur_cache.jpg", 200, 150, "jpeg")

	// First call
	dataURI1, err := r.BlurPlaceholder(src)
	if err != nil {
		t.Fatalf("first BlurPlaceholder: %v", err)
	}

	// Second call should return same result from cache
	dataURI2, err := r.BlurPlaceholder(src)
	if err != nil {
		t.Fatalf("second BlurPlaceholder: %v", err)
	}

	if dataURI1 != dataURI2 {
		t.Fatal("cache should return identical data URI")
	}
}

func TestRegistry_BlurPlaceholder_DifferentImages(t *testing.T) {
	r, _ := newTestRegistry(t, false)
	srcDir := t.TempDir()

	src1 := createTestImage(t, srcDir, "blur1.jpg", 200, 100, "jpeg")
	src2 := createTestImage(t, srcDir, "blur2.jpg", 100, 200, "jpeg")

	dataURI1, err := r.BlurPlaceholder(src1)
	if err != nil {
		t.Fatalf("BlurPlaceholder src1: %v", err)
	}

	dataURI2, err := r.BlurPlaceholder(src2)
	if err != nil {
		t.Fatalf("BlurPlaceholder src2: %v", err)
	}

	if dataURI1 == dataURI2 {
		t.Fatal("different images should produce different blur placeholders")
	}
}

func TestRegistry_BlurPlaceholder_NonExistent(t *testing.T) {
	r, _ := newTestRegistry(t, false)

	_, err := r.BlurPlaceholder("/nonexistent/image.jpg")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestRegistry_BlurPlaceholder_Concurrent(t *testing.T) {
	r, _ := newTestRegistry(t, false)
	srcDir := t.TempDir()

	src := createTestImage(t, srcDir, "blur_concurrent.jpg", 300, 200, "jpeg")

	const N = 10
	var wg sync.WaitGroup
	results := make([]string, N)
	errs := make([]error, N)

	wg.Add(N)
	for i := range N {
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = r.BlurPlaceholder(src)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}

	// All results must be identical
	for i := 1; i < N; i++ {
		if results[i] != results[0] {
			t.Fatalf("goroutine %d returned different result", i)
		}
	}
}

func TestRegistry_Info_MemoryCache(t *testing.T) {
	r, _ := newTestRegistry(t, false)
	srcDir := t.TempDir()

	src := createTestImage(t, srcDir, "info_cache.jpg", 400, 300, "jpeg")

	// First call
	info1, err := r.Info(src)
	if err != nil {
		t.Fatalf("first Info: %v", err)
	}

	// Second call should hit cache
	info2, err := r.Info(src)
	if err != nil {
		t.Fatalf("second Info: %v", err)
	}

	// Results should be identical
	if info1["width"] != info2["width"] || info1["height"] != info2["height"] {
		t.Fatal("cached result should match original")
	}
}

func TestRegistry_Info_CacheInvalidatedOnFileChange(t *testing.T) {
	r, _ := newTestRegistry(t, false)
	srcDir := t.TempDir()

	src := createTestImage(t, srcDir, "info_change.jpg", 200, 100, "jpeg")

	// First call
	info1, err := r.Info(src)
	if err != nil {
		t.Fatalf("first Info: %v", err)
	}

	if info1["width"].(int64) != 200 || info1["height"].(int64) != 100 {
		t.Fatalf("unexpected dimensions: %v x %v", info1["width"], info1["height"])
	}

	// Recreate the file with different dimensions
	// Need a small delay to ensure modtime changes
	createTestImage(t, srcDir, "info_change.jpg", 300, 150, "jpeg")

	// Second call should get new info (cache miss due to modtime change)
	info2, err := r.Info(src)
	if err != nil {
		t.Fatalf("second Info: %v", err)
	}

	if info2["width"].(int64) != 300 || info2["height"].(int64) != 150 {
		t.Fatalf("expected new dimensions 300x150, got %v x %v", info2["width"], info2["height"])
	}
}

func TestRegistry_Clear_ClearsInfoCache(t *testing.T) {
	r, _ := newTestRegistry(t, false)
	srcDir := t.TempDir()

	src := createTestImage(t, srcDir, "info_clear.jpg", 100, 100, "jpeg")

	// Populate cache
	_, err := r.Info(src)
	if err != nil {
		t.Fatalf("Info: %v", err)
	}

	// Clear should not panic and should clear the info cache
	r.Clear()

	// After clear, we should still be able to get info (but it will be a cache miss)
	info, err := r.Info(src)
	if err != nil {
		t.Fatalf("Info after Clear: %v", err)
	}

	if info["width"].(int64) != 100 {
		t.Fatalf("unexpected width after clear: %v", info["width"])
	}
}

func TestRegistry_Info_Concurrent(t *testing.T) {
	r, _ := newTestRegistry(t, false)
	srcDir := t.TempDir()

	src := createTestImage(t, srcDir, "info_concurrent.jpg", 250, 200, "jpeg")

	const N = 20
	var wg sync.WaitGroup
	results := make([]map[string]any, N)
	errs := make([]error, N)

	wg.Add(N)
	for i := range N {
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = r.Info(src)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}

	// All results must have the same dimensions
	for i := 1; i < N; i++ {
		if results[i]["width"] != results[0]["width"] || results[i]["height"] != results[0]["height"] {
			t.Fatalf("goroutine %d returned different dimensions", i)
		}
	}
}
