// Package images provides image transformation and caching for Basil.
package images

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// TransformOptions specifies how to transform an image.
type TransformOptions struct {
	Width   int    // Target width (0 = auto/preserve aspect ratio)
	Height  int    // Target height (0 = auto/preserve aspect ratio)
	Crop    string // "center" or "" (no crop — fit within box)
	Quality int    // 1-100, 0 = format default (JPEG: 85, WebP: 80, PNG: lossless)
	Format  string // "webp", "jpeg", "png", or "" (preserve original)
}

// DefaultQuality returns the default quality for a given format.
func DefaultQuality(format string) int {
	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		return 85
	case "webp":
		return 80
	case "png", "gif":
		return 0 // Lossless
	default:
		return 85
	}
}

// ParseOptions converts a map of options (from Parsley dict) to TransformOptions.
// Returns an error if any option is invalid.
func ParseOptions(opts map[string]any) (TransformOptions, error) {
	var t TransformOptions

	for key, val := range opts {
		switch key {
		case "width":
			w, err := toInt(val)
			if err != nil {
				return t, fmt.Errorf("width: %w", err)
			}
			t.Width = w

		case "height":
			h, err := toInt(val)
			if err != nil {
				return t, fmt.Errorf("height: %w", err)
			}
			t.Height = h

		case "crop":
			c, ok := val.(string)
			if !ok {
				return t, fmt.Errorf("crop: expected string, got %T", val)
			}
			t.Crop = c

		case "quality":
			q, err := toInt(val)
			if err != nil {
				return t, fmt.Errorf("quality: %w", err)
			}
			t.Quality = q

		case "format":
			f, ok := val.(string)
			if !ok {
				return t, fmt.Errorf("format: expected string, got %T", val)
			}
			t.Format = f

		default:
			return t, fmt.Errorf("unknown option: %s", key)
		}
	}

	return t, nil
}

// Validate checks that all options are within valid ranges.
func (t *TransformOptions) Validate(maxWidth, maxHeight int) error {
	if t.Width < 0 {
		return fmt.Errorf("width must be non-negative, got %d", t.Width)
	}
	if t.Height < 0 {
		return fmt.Errorf("height must be non-negative, got %d", t.Height)
	}
	if maxWidth > 0 && t.Width > maxWidth {
		return fmt.Errorf("width %d exceeds maximum %d", t.Width, maxWidth)
	}
	if maxHeight > 0 && t.Height > maxHeight {
		return fmt.Errorf("height %d exceeds maximum %d", t.Height, maxHeight)
	}
	if t.Quality < 0 || t.Quality > 100 {
		return fmt.Errorf("quality must be 0-100, got %d", t.Quality)
	}
	if t.Crop != "" && t.Crop != "center" {
		return fmt.Errorf("crop must be \"center\" or empty, got %q", t.Crop)
	}
	if t.Format != "" {
		switch strings.ToLower(t.Format) {
		case "jpeg", "jpg", "png", "webp", "gif":
			// Valid
		default:
			return fmt.Errorf("unsupported format %q (supported: jpeg, png, webp, gif)", t.Format)
		}
	}
	return nil
}

// Canonical returns a stable string representation of the options for cache key computation.
// The format is deterministic: options are always in the same order.
func (t *TransformOptions) Canonical() string {
	return fmt.Sprintf("w=%d|h=%d|c=%s|q=%d|f=%s",
		t.Width, t.Height, t.Crop, t.Quality, strings.ToLower(t.Format))
}

// CacheKey computes a cache key from a source hash and transform options.
// The key is a truncated SHA256 hash (16 hex chars).
func CacheKey(sourceHash string, opts TransformOptions) string {
	canonical := sourceHash + "|" + opts.Canonical()
	hash := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(hash[:])[:16]
}

// toInt converts various numeric types to int.
func toInt(val any) (int, error) {
	switch v := val.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("expected number, got %T", val)
	}
}

// NormalizeFormat normalizes format strings to lowercase canonical form.
// Returns the normalized format and the file extension.
func NormalizeFormat(format string) (normalized, ext string) {
	format = strings.ToLower(format)
	switch format {
	case "jpeg", "jpg":
		return "jpeg", ".jpg"
	case "png":
		return "png", ".png"
	case "webp":
		return "webp", ".webp"
	case "gif":
		return "gif", ".gif"
	default:
		return format, "." + format
	}
}

// ExtensionToFormat converts a file extension to a format name.
func ExtensionToFormat(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	switch ext {
	case "jpg", "jpeg":
		return "jpeg"
	case "png":
		return "png"
	case "webp":
		return "webp"
	case "gif":
		return "gif"
	default:
		return ext
	}
}
