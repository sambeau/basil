package images

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Cache manages the disk cache for transformed images.
type Cache struct {
	dir string
}

// NewCache creates a new disk cache at the specified directory.
// The directory is created if it doesn't exist.
func NewCache(dir string) *Cache {
	return &Cache{dir: dir}
}

// Dir returns the cache directory path.
func (c *Cache) Dir() string {
	return c.dir
}

// Path returns the full path for a cached file given its key and extension.
func (c *Cache) Path(key, ext string) string {
	return filepath.Join(c.dir, key+ext)
}

// Exists checks if a cached file exists and returns its path if so.
func (c *Cache) Exists(key, ext string) (string, bool) {
	path := c.Path(key, ext)
	if _, err := os.Stat(path); err == nil {
		return path, true
	}
	return "", false
}

// Read reads a cached file if it exists.
// Returns the file content and true if found, nil and false otherwise.
func (c *Cache) Read(key, ext string) ([]byte, bool) {
	path := c.Path(key, ext)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

// Write atomically writes data to the cache.
// It writes to a temporary file first, then renames to ensure atomic writes.
// This prevents serving partial files if the write is interrupted.
func (c *Cache) Write(key, ext string, data []byte) (string, error) {
	// Ensure cache directory exists
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	finalPath := c.Path(key, ext)

	// Create temporary file in the same directory (for atomic rename)
	tmpFile, err := os.CreateTemp(c.dir, "tmp-*"+ext)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Write data to temp file
	_, err = tmpFile.Write(data)
	if closeErr := tmpFile.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(tmpPath) // Clean up on error
		return "", fmt.Errorf("write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath) // Clean up on error
		return "", fmt.Errorf("rename to final: %w", err)
	}

	return finalPath, nil
}

// IsFresh checks if the cached file is newer than the source file.
// This is used in dev mode to detect when the source has changed.
func (c *Cache) IsFresh(key, ext, sourcePath string) bool {
	cachePath := c.Path(key, ext)

	cacheInfo, err := os.Stat(cachePath)
	if err != nil {
		return false // Cache doesn't exist
	}

	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return false // Source doesn't exist
	}

	return cacheInfo.ModTime().After(sourceInfo.ModTime())
}

// Clear removes all files from the cache directory.
// It does not remove the directory itself.
func (c *Cache) Clear() error {
	entries, err := os.ReadDir(c.dir)
	if os.IsNotExist(err) {
		return nil // Nothing to clear
	}
	if err != nil {
		return fmt.Errorf("read cache dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip subdirectories
		}
		path := filepath.Join(c.dir, entry.Name())
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", path, err)
		}
	}

	return nil
}

// Size returns the total size of all cached files in bytes.
func (c *Cache) Size() (int64, error) {
	var total int64

	entries, err := os.ReadDir(c.dir)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read cache dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't stat
		}
		total += info.Size()
	}

	return total, nil
}

// Count returns the number of files in the cache.
func (c *Cache) Count() (int, error) {
	entries, err := os.ReadDir(c.dir)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read cache dir: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			count++
		}
	}
	return count, nil
}

// CopyFile copies a file from src to the cache with the given key and extension.
// This is used when no transformation is needed but we still want to cache.
func (c *Cache) CopyFile(key, ext, srcPath string) (string, error) {
	// Ensure cache directory exists
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("open source: %w", err)
	}
	defer func() { _ = src.Close() }()

	finalPath := c.Path(key, ext)

	// Create temporary file for atomic write
	tmpFile, err := os.CreateTemp(c.dir, "tmp-*"+ext)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	_, err = io.Copy(tmpFile, src)
	if closeErr := tmpFile.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("copy to temp: %w", err)
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("rename to final: %w", err)
	}

	return finalPath, nil
}
