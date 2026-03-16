package images

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"golang.org/x/sync/singleflight"
)

// Registry manages image transformations and caching.
// It maps content-hashed URLs to cached file paths, coordinates transforms,
// and deduplicates concurrent requests for the same transformation.
type Registry struct {
	mu     sync.RWMutex
	byHash map[string]string // cache key -> cached file path

	cache     *Cache
	infoCache sync.Map // key: "absPath|modtime" -> value: map[string]any
	group     singleflight.Group
	logger    func(format string, args ...any)

	// Config
	maxWidth       int
	maxHeight      int
	defaultQuality int
	defaultFormat  string
	devMode        bool
}

// NewRegistry creates a new image registry.
func NewRegistry(cacheDir string, maxWidth, maxHeight, defaultQuality int, defaultFormat string, devMode bool, logger func(string, ...any)) *Registry {
	return &Registry{
		byHash:         make(map[string]string),
		cache:          NewCache(cacheDir),
		logger:         logger,
		maxWidth:       maxWidth,
		maxHeight:      maxHeight,
		defaultQuality: defaultQuality,
		defaultFormat:  defaultFormat,
		devMode:        devMode,
	}
}

// Transform transforms an image with the given options, caches the result,
// and returns the public URL (e.g., /__img/{hash}.jpg).
// Concurrent requests for the same transformation are deduplicated via singleflight.
// DefaultSharpenSigma is the default sharpening sigma applied on downscale.
const DefaultSharpenSigma = 0.5

func (r *Registry) Transform(sourcePath string, opts map[string]any) (string, error) {
	// Parse and validate options
	transformOpts, err := ParseOptions(opts)
	if err != nil {
		return "", fmt.Errorf("invalid options: %w", err)
	}

	// Apply config defaults
	if transformOpts.Quality == 0 && r.defaultQuality > 0 {
		transformOpts.Quality = r.defaultQuality
	}
	if transformOpts.Format == "" && r.defaultFormat != "" {
		transformOpts.Format = r.defaultFormat
	}

	// Apply default sharpen sigma if not explicitly set or disabled
	if transformOpts.Sharpen == 0 && !transformOpts.SharpenDisabled {
		transformOpts.Sharpen = DefaultSharpenSigma
	}

	// Validate against limits
	if err := transformOpts.Validate(r.maxWidth, r.maxHeight); err != nil {
		return "", err
	}

	// Compute source hash
	sourceHash, err := SourceHash(sourcePath)
	if err != nil {
		return "", fmt.Errorf("hash source: %w", err)
	}

	// Determine output format and extension
	outputFormat := transformOpts.Format
	if outputFormat == "" {
		// Preserve source format
		ext := filepath.Ext(sourcePath)
		outputFormat = ExtensionToFormat(ext)
	}
	_, ext := NormalizeFormat(outputFormat)

	// Compute cache key
	cacheKey := CacheKey(sourceHash, transformOpts)

	// Check if already registered (fast path)
	r.mu.RLock()
	if cachedPath, ok := r.byHash[cacheKey]; ok {
		r.mu.RUnlock()
		// In dev mode, check if source is newer than cache
		if r.devMode && !r.cache.IsFresh(cacheKey, ext, sourcePath) {
			// Source changed, need to re-transform
		} else {
			_ = cachedPath // silence unused warning
			return formatImageURL(cacheKey, ext), nil
		}
	} else {
		r.mu.RUnlock()
	}

	// Check disk cache (may exist from previous server run)
	if cachedPath, exists := r.cache.Exists(cacheKey, ext); exists {
		// In dev mode, verify freshness
		if r.devMode && !r.cache.IsFresh(cacheKey, ext, sourcePath) {
			// Source changed, need to re-transform
		} else {
			// Register and return
			r.mu.Lock()
			r.byHash[cacheKey] = cachedPath
			r.mu.Unlock()
			return formatImageURL(cacheKey, ext), nil
		}
	}

	// Use singleflight to deduplicate concurrent transforms for the same key
	result, err, _ := r.group.Do(cacheKey, func() (any, error) {
		return r.doTransform(sourcePath, transformOpts, cacheKey, ext)
	})
	if err != nil {
		return "", err
	}

	return result.(string), nil
}

// doTransform performs the actual transformation and caching.
// This is called within singleflight to ensure only one transform per key.
func (r *Registry) doTransform(sourcePath string, opts TransformOptions, cacheKey, ext string) (string, error) {
	// Log if source is large
	if r.logger != nil {
		if stat, err := os.Stat(sourcePath); err == nil && stat.Size() > WarnSourceSize {
			r.logger("image(): large source file %s (%dMB)", sourcePath, stat.Size()/1024/1024)
		}
	}

	// Check if we need to transform at all
	// If no resize/crop and format is same as source, we can just copy
	needsTransform := opts.Width > 0 || opts.Height > 0 || opts.Crop != ""
	sourceFormat := ExtensionToFormat(filepath.Ext(sourcePath))
	outputFormat := opts.Format
	if outputFormat == "" {
		outputFormat = sourceFormat
	}
	formatChange := sourceFormat != outputFormat

	var cachedPath string
	var err error

	if !needsTransform && !formatChange {
		// No transformation needed, just copy to cache
		cachedPath, err = r.cache.CopyFile(cacheKey, ext, sourcePath)
	} else {
		// Perform transformation
		var data []byte
		data, _, err = Process(sourcePath, opts)
		if err != nil {
			return "", fmt.Errorf("transform: %w", err)
		}

		// Write to cache
		cachedPath, err = r.cache.Write(cacheKey, ext, data)
		if err != nil {
			return "", fmt.Errorf("cache write: %w", err)
		}
	}
	if err != nil {
		return "", err
	}

	// Register the cached file
	r.mu.Lock()
	r.byHash[cacheKey] = cachedPath
	r.mu.Unlock()

	return formatImageURL(cacheKey, ext), nil
}

// Info returns metadata about an image without transforming it.
// Results are cached in memory, keyed by absolute path + file modification time.
func (r *Registry) Info(sourcePath string) (map[string]any, error) {
	// Get file stat for modtime
	stat, err := os.Stat(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", sourcePath, err)
	}

	// Build cache key: path + modtime
	cacheKey := sourcePath + "|" + strconv.FormatInt(stat.ModTime().UnixNano(), 10)

	// Check cache
	if cached, ok := r.infoCache.Load(cacheKey); ok {
		return cached.(map[string]any), nil
	}

	// Cache miss - get info
	info, err := GetInfo(sourcePath)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"width":       int64(info.Width),
		"height":      int64(info.Height),
		"format":      info.Format,
		"orientation": info.Orientation,
	}

	// Store in cache
	r.infoCache.Store(cacheKey, result)

	return result, nil
}

// Lookup returns the file path for a cache key, or empty string if not found.
func (r *Registry) Lookup(key string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	path, ok := r.byHash[key]
	return path, ok
}

// Clear removes all registered images (called on server reload).
// Note: This only clears the in-memory registry, not the disk cache.
func (r *Registry) Clear() {
	r.mu.Lock()
	r.byHash = make(map[string]string)
	r.mu.Unlock()

	// Clear info cache
	r.infoCache.Range(func(key, value any) bool {
		r.infoCache.Delete(key)
		return true
	})
}

// ClearCache clears both the in-memory registry and disk cache.
func (r *Registry) ClearCache() error {
	r.Clear()
	return r.cache.Clear()
}

// CacheStats returns statistics about the disk cache.
func (r *Registry) CacheStats() (count int, size int64, err error) {
	count, err = r.cache.Count()
	if err != nil {
		return 0, 0, err
	}
	size, err = r.cache.Size()
	if err != nil {
		return 0, 0, err
	}
	return count, size, nil
}

// formatImageURL returns the public URL for a cached image.
func formatImageURL(key, ext string) string {
	return "/__img/" + key + ext
}

// BlurPlaceholder generates a Low Quality Image Placeholder (LQIP) data URI.
// Results are cached to disk and deduplicated via singleflight.
// The data URI format is: "data:image/jpeg;base64,..."
func (r *Registry) BlurPlaceholder(sourcePath string) (string, error) {
	// Compute source hash for cache key
	sourceHash, err := SourceHash(sourcePath)
	if err != nil {
		return "", fmt.Errorf("hash source: %w", err)
	}

	// Cache key includes "blur" to distinguish from transform cache entries
	cacheKey := blurCacheKey(sourceHash)

	// Check disk cache (store as .txt file containing the data URI)
	if cachedPath, exists := r.cache.Exists(cacheKey, ".txt"); exists {
		// In dev mode, verify freshness
		if r.devMode && !r.cache.IsFresh(cacheKey, ".txt", sourcePath) {
			// Source changed, regenerate
		} else {
			// Read cached data URI
			data, err := os.ReadFile(cachedPath)
			if err == nil {
				return string(data), nil
			}
			// Cache read failed, regenerate
		}
	}

	// Use singleflight to deduplicate concurrent requests
	result, err, _ := r.group.Do("blur:"+cacheKey, func() (any, error) {
		return r.doBlurPlaceholder(sourcePath, cacheKey)
	})
	if err != nil {
		return "", err
	}

	return result.(string), nil
}

// doBlurPlaceholder generates and caches a blur placeholder.
func (r *Registry) doBlurPlaceholder(sourcePath, cacheKey string) (string, error) {
	// Generate the blur placeholder
	dataURI, err := GenerateBlurPlaceholder(sourcePath)
	if err != nil {
		return "", fmt.Errorf("generate blur placeholder: %w", err)
	}

	// Write to cache (store as text file)
	_, err = r.cache.Write(cacheKey, ".txt", []byte(dataURI))
	if err != nil {
		// Log but don't fail - caching is optional
		if r.logger != nil {
			r.logger("blur placeholder cache write failed: %v", err)
		}
	}

	return dataURI, nil
}

// blurCacheKey computes a cache key for blur placeholders.
func blurCacheKey(sourceHash string) string {
	canonical := sourceHash + "|blur"
	hash := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(hash[:])[:16]
}
