package images

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	cache := NewCache("/tmp/test-cache")
	if cache.Dir() != "/tmp/test-cache" {
		t.Errorf("Dir() = %q, want %q", cache.Dir(), "/tmp/test-cache")
	}
}

func TestCache_Path(t *testing.T) {
	cache := NewCache("/tmp/test-cache")

	tests := []struct {
		key  string
		ext  string
		want string
	}{
		{"abc123", ".jpg", "/tmp/test-cache/abc123.jpg"},
		{"def456", ".png", "/tmp/test-cache/def456.png"},
		{"hash", ".webp", "/tmp/test-cache/hash.webp"},
	}

	for _, tt := range tests {
		got := cache.Path(tt.key, tt.ext)
		if got != tt.want {
			t.Errorf("Path(%q, %q) = %q, want %q", tt.key, tt.ext, got, tt.want)
		}
	}
}

func TestCache_WriteAndRead(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir)

	key := "testkey123456"
	ext := ".jpg"
	data := []byte("test image data")

	// Write to cache
	path, err := cache.Write(key, ext, data)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	expectedPath := filepath.Join(dir, key+ext)
	if path != expectedPath {
		t.Errorf("Write() path = %q, want %q", path, expectedPath)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Errorf("Written file does not exist: %v", err)
	}

	// Read back
	readData, ok := cache.Read(key, ext)
	if !ok {
		t.Fatal("Read() returned not found")
	}

	if string(readData) != string(data) {
		t.Errorf("Read() = %q, want %q", string(readData), string(data))
	}
}

func TestCache_Exists(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir)

	key := "existstest"
	ext := ".png"
	data := []byte("png data")

	// Should not exist initially
	if _, exists := cache.Exists(key, ext); exists {
		t.Error("Exists() = true for non-existent file")
	}

	// Write the file
	if _, err := cache.Write(key, ext, data); err != nil {
		t.Fatal(err)
	}

	// Should exist now
	path, exists := cache.Exists(key, ext)
	if !exists {
		t.Error("Exists() = false for existing file")
	}
	if path == "" {
		t.Error("Exists() returned empty path for existing file")
	}
}

func TestCache_Read_NotFound(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir)

	data, ok := cache.Read("nonexistent", ".jpg")
	if ok {
		t.Error("Read() returned ok for non-existent file")
	}
	if data != nil {
		t.Error("Read() returned data for non-existent file")
	}
}

func TestCache_Write_CreatesDirectory(t *testing.T) {
	// Use a nested directory that doesn't exist
	dir := filepath.Join(t.TempDir(), "nested", "cache", "dir")
	cache := NewCache(dir)

	// Write should create the directory
	_, err := cache.Write("test", ".jpg", []byte("data"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("Cache directory was not created: %v", err)
	}
}

func TestCache_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir)

	key := "atomictest"
	ext := ".jpg"
	data := []byte("atomic write test data")

	// Write the file
	path, err := cache.Write(key, ext, data)
	if err != nil {
		t.Fatal(err)
	}

	// Verify content is complete
	readData, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if string(readData) != string(data) {
		t.Errorf("File content = %q, want %q", string(readData), string(data))
	}

	// Verify no temp files left behind
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ext {
			t.Errorf("Unexpected file in cache dir: %s", entry.Name())
		}
	}
}

func TestCache_IsFresh(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir)

	// Create a "source" file
	sourcePath := filepath.Join(dir, "source.jpg")
	if err := os.WriteFile(sourcePath, []byte("source"), 0644); err != nil {
		t.Fatal(err)
	}

	key := "freshtest"
	ext := ".jpg"

	// Cache doesn't exist yet
	if cache.IsFresh(key, ext, sourcePath) {
		t.Error("IsFresh() = true when cache doesn't exist")
	}

	// Wait a moment to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Write to cache (should be newer than source)
	if _, err := cache.Write(key, ext, []byte("cached")); err != nil {
		t.Fatal(err)
	}

	// Cache should be fresh
	if !cache.IsFresh(key, ext, sourcePath) {
		t.Error("IsFresh() = false when cache is newer than source")
	}

	// Wait a moment
	time.Sleep(10 * time.Millisecond)

	// Update source file (touch it)
	if err := os.WriteFile(sourcePath, []byte("updated source"), 0644); err != nil {
		t.Fatal(err)
	}

	// Cache should no longer be fresh
	if cache.IsFresh(key, ext, sourcePath) {
		t.Error("IsFresh() = true when source is newer than cache")
	}
}

func TestCache_Clear(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir)

	// Write some files
	for i := range 5 {
		key := "file" + string(rune('0'+i))
		if _, err := cache.Write(key, ".jpg", []byte("data")); err != nil {
			t.Fatal(err)
		}
	}

	// Verify files exist
	count, err := cache.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Errorf("Count() = %d, want 5", count)
	}

	// Clear the cache
	if err := cache.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Verify files are gone
	count, err = cache.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("Count() after Clear() = %d, want 0", count)
	}
}

func TestCache_Clear_NonExistentDir(t *testing.T) {
	cache := NewCache("/nonexistent/cache/dir")

	// Should not error on non-existent directory
	if err := cache.Clear(); err != nil {
		t.Errorf("Clear() error = %v, want nil for non-existent dir", err)
	}
}

func TestCache_Size(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir)

	// Empty cache
	size, err := cache.Size()
	if err != nil {
		t.Fatal(err)
	}
	if size != 0 {
		t.Errorf("Size() = %d, want 0 for empty cache", size)
	}

	// Add some data
	data1 := make([]byte, 1000)
	data2 := make([]byte, 500)

	if _, err := cache.Write("file1", ".jpg", data1); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Write("file2", ".png", data2); err != nil {
		t.Fatal(err)
	}

	size, err = cache.Size()
	if err != nil {
		t.Fatal(err)
	}

	expectedSize := int64(len(data1) + len(data2))
	if size != expectedSize {
		t.Errorf("Size() = %d, want %d", size, expectedSize)
	}
}

func TestCache_Count(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir)

	// Empty cache
	count, err := cache.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("Count() = %d, want 0 for empty cache", count)
	}

	// Add files
	for i := range 3 {
		key := "count" + string(rune('a'+i))
		if _, err := cache.Write(key, ".jpg", []byte("data")); err != nil {
			t.Fatal(err)
		}
	}

	count, err = cache.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("Count() = %d, want 3", count)
	}
}

func TestCache_CopyFile(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir)

	// Create source file
	srcContent := []byte("source file content for copy test")
	srcPath := filepath.Join(dir, "source.txt")
	if err := os.WriteFile(srcPath, srcContent, 0644); err != nil {
		t.Fatal(err)
	}

	key := "copytest"
	ext := ".txt"

	// Copy to cache
	path, err := cache.CopyFile(key, ext, srcPath)
	if err != nil {
		t.Fatalf("CopyFile() error = %v", err)
	}

	// Verify copy
	copiedContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if string(copiedContent) != string(srcContent) {
		t.Errorf("CopyFile() content = %q, want %q", string(copiedContent), string(srcContent))
	}

	// Verify it's in the cache
	if _, exists := cache.Exists(key, ext); !exists {
		t.Error("CopyFile() file not found in cache after copy")
	}
}

func TestCache_CopyFile_SourceNotFound(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir)

	_, err := cache.CopyFile("test", ".txt", "/nonexistent/file.txt")
	if err == nil {
		t.Error("CopyFile() expected error for non-existent source")
	}
}
