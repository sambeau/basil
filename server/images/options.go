// Package images provides image transformation and caching for Basil.
package images

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// FocalRegion represents a user-specified region of interest in normalised (0–1) coordinates.
// If W and H are zero, it's a point (expanded to a small boost region internally).
// If W and H are non-zero, it's a rectangle used directly as a boost region.
type FocalRegion struct {
	X float64 // 0–1, left to right
	Y float64 // 0–1, top to bottom
	W float64 // 0–1, width (0 = point)
	H float64 // 0–1, height (0 = point)
}

// TransformOptions specifies how to transform an image.
type TransformOptions struct {
	Width           int          // Target width (0 = auto/preserve aspect ratio)
	Height          int          // Target height (0 = auto/preserve aspect ratio)
	Crop            string       // "center", "smart", or "" (no crop — fit within box)
	Quality         int          // 1-100, 0 = format default (JPEG: 85, WebP: 80, PNG: lossless)
	Format          string       // "webp", "jpeg", "png", or "" (preserve original)
	Sharpen         float64      // Sharpen sigma (0 = use default on downscale, >0 = explicit)
	SharpenDisabled bool         // Explicitly disabled via {sharpen: false}
	Scale           string       // "" or "smart" (seam carving)
	Focal           *FocalRegion // Optional focal point/region for smart crop (normalised 0–1)
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

		case "sharpen":
			switch v := val.(type) {
			case bool:
				if !v {
					t.SharpenDisabled = true
				}
				// sharpen: true means use default, leave Sharpen at 0
			case int:
				t.Sharpen = float64(v)
			case int64:
				t.Sharpen = float64(v)
			case float64:
				t.Sharpen = v
			default:
				return t, fmt.Errorf("sharpen: expected boolean or number, got %T", val)
			}

		case "scale":
			s, ok := val.(string)
			if !ok {
				return t, fmt.Errorf("scale: expected string, got %T", val)
			}
			t.Scale = s

		case "focal":
			focal, err := parseFocal(val)
			if err != nil {
				return t, fmt.Errorf("focal: %w", err)
			}
			t.Focal = focal

		default:
			return t, fmt.Errorf("unknown option: %s", key)
		}
	}

	return t, nil
}

// parseFocal parses a focal point/region from a map.
// Accepts {x, y} for a point or {x, y, w, h} for a rectangle.
func parseFocal(val any) (*FocalRegion, error) {
	m, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected dict with x, y keys, got %T", val)
	}

	focal := &FocalRegion{}

	// X is required
	xVal, ok := m["x"]
	if !ok {
		return nil, fmt.Errorf("missing required key 'x'")
	}
	x, err := toFloat(xVal)
	if err != nil {
		return nil, fmt.Errorf("x: %w", err)
	}
	focal.X = x

	// Y is required
	yVal, ok := m["y"]
	if !ok {
		return nil, fmt.Errorf("missing required key 'y'")
	}
	y, err := toFloat(yVal)
	if err != nil {
		return nil, fmt.Errorf("y: %w", err)
	}
	focal.Y = y

	// W is optional (default 0 = point)
	if wVal, ok := m["w"]; ok {
		w, err := toFloat(wVal)
		if err != nil {
			return nil, fmt.Errorf("w: %w", err)
		}
		focal.W = w
	}

	// H is optional (default 0 = point)
	if hVal, ok := m["h"]; ok {
		h, err := toFloat(hVal)
		if err != nil {
			return nil, fmt.Errorf("h: %w", err)
		}
		focal.H = h
	}

	return focal, nil
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

	// Validate crop option
	switch t.Crop {
	case "", "center", "smart":
		// Valid
	default:
		return fmt.Errorf("crop must be \"center\", \"smart\", or empty, got %q", t.Crop)
	}

	// Smart crop requires both width and height (need aspect ratio)
	if t.Crop == "smart" && (t.Width == 0 || t.Height == 0) {
		return fmt.Errorf("crop: \"smart\" requires both width and height")
	}

	// Validate scale option
	switch t.Scale {
	case "", "smart":
		// Valid
	default:
		return fmt.Errorf("scale must be \"smart\" or empty, got %q", t.Scale)
	}

	// Scale "smart" requires at least one dimension
	if t.Scale == "smart" && t.Width == 0 && t.Height == 0 {
		return fmt.Errorf("scale: \"smart\" requires at least width or height")
	}

	// Crop and scale are mutually exclusive
	if t.Crop != "" && t.Scale != "" {
		return fmt.Errorf("crop and scale cannot be used together")
	}

	// Focal requires crop: "smart"
	if t.Focal != nil && t.Crop != "smart" {
		return fmt.Errorf("focal requires crop: \"smart\"")
	}

	// Validate focal coordinates are in range 0-1
	if t.Focal != nil {
		if t.Focal.X < 0 || t.Focal.X > 1 {
			return fmt.Errorf("focal x must be 0-1, got %v", t.Focal.X)
		}
		if t.Focal.Y < 0 || t.Focal.Y > 1 {
			return fmt.Errorf("focal y must be 0-1, got %v", t.Focal.Y)
		}
		if t.Focal.W < 0 || t.Focal.W > 1 {
			return fmt.Errorf("focal w must be 0-1, got %v", t.Focal.W)
		}
		if t.Focal.H < 0 || t.Focal.H > 1 {
			return fmt.Errorf("focal h must be 0-1, got %v", t.Focal.H)
		}
	}

	if t.Format != "" {
		switch strings.ToLower(t.Format) {
		case "jpeg", "jpg", "png", "webp", "gif":
			// Valid
		default:
			return fmt.Errorf("unsupported format %q (supported: jpeg, png, webp, gif)", t.Format)
		}
	}
	if t.Sharpen < 0 {
		return fmt.Errorf("sharpen must be non-negative, got %v", t.Sharpen)
	}
	if t.Sharpen > 0 && t.SharpenDisabled {
		return fmt.Errorf("sharpen cannot be both set (%v) and disabled", t.Sharpen)
	}
	return nil
}

// Canonical returns a stable string representation of the options for cache key computation.
// The format is deterministic: options are always in the same order.
func (t *TransformOptions) Canonical() string {
	sharpenStr := "0"
	if t.SharpenDisabled {
		sharpenStr = "off"
	} else if t.Sharpen > 0 {
		sharpenStr = fmt.Sprintf("%.2f", t.Sharpen)
	}

	// Include scale in canonical representation
	scaleStr := t.Scale

	// Include focal in canonical representation
	focalStr := ""
	if t.Focal != nil {
		if t.Focal.W > 0 || t.Focal.H > 0 {
			focalStr = fmt.Sprintf("f=%.3f,%.3f,%.3f,%.3f", t.Focal.X, t.Focal.Y, t.Focal.W, t.Focal.H)
		} else {
			focalStr = fmt.Sprintf("f=%.3f,%.3f", t.Focal.X, t.Focal.Y)
		}
	}

	canonical := fmt.Sprintf("w=%d|h=%d|c=%s|q=%d|f=%s|s=%s|sc=%s",
		t.Width, t.Height, t.Crop, t.Quality, strings.ToLower(t.Format), sharpenStr, scaleStr)

	if focalStr != "" {
		canonical += "|" + focalStr
	}

	return canonical
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

// toFloat converts various numeric types to float64.
func toFloat(val any) (float64, error) {
	switch v := val.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
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
