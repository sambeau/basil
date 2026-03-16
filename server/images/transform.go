package images

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp" // WebP decode support
)

// MaxSourceSize is the maximum source file size allowed (50MB).
const MaxSourceSize = 50 * 1024 * 1024

// WarnSourceSize is the threshold for logging a warning about large files (10MB).
const WarnSourceSize = 10 * 1024 * 1024

// Load reads and decodes an image from disk.
// It returns the decoded image, the detected format, and any error.
// For JPEG images, EXIF orientation is automatically applied.
func Load(path string) (image.Image, string, error) {
	// Check file size before loading
	stat, err := os.Stat(path)
	if err != nil {
		return nil, "", fmt.Errorf("stat %s: %w", path, err)
	}
	if stat.Size() > MaxSourceSize {
		return nil, "", fmt.Errorf("file too large: %d bytes (max %d)", stat.Size(), MaxSourceSize)
	}

	// Detect format from extension
	ext := strings.ToLower(filepath.Ext(path))
	format := ExtensionToFormat(ext)

	// For JPEG/PNG/GIF, use imaging.Open which handles EXIF auto-orientation
	switch format {
	case "jpeg", "png", "gif":
		img, err := imaging.Open(path, imaging.AutoOrientation(true))
		if err != nil {
			return nil, "", fmt.Errorf("decode %s: %w", path, err)
		}
		return img, format, nil

	case "webp":
		// WebP: use standard image.Decode (via x/image/webp import)
		f, err := os.Open(path)
		if err != nil {
			return nil, "", fmt.Errorf("open %s: %w", path, err)
		}
		defer func() { _ = f.Close() }()

		img, _, err := image.Decode(f)
		if err != nil {
			return nil, "", fmt.Errorf("decode webp %s: %w", path, err)
		}
		return img, "webp", nil

	default:
		return nil, "", fmt.Errorf("unsupported image format: %s", ext)
	}
}

// Transform applies resize/crop transformations to an image.
// It returns a new image with the transformations applied.
func Transform(img image.Image, opts TransformOptions) image.Image {
	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// If no dimensions specified, return as-is
	if opts.Width == 0 && opts.Height == 0 {
		return img
	}

	// Clamp requested dimensions to source dimensions (no upscaling)
	targetWidth := opts.Width
	targetHeight := opts.Height

	if targetWidth > srcWidth {
		targetWidth = srcWidth
	}
	if targetHeight > srcHeight {
		targetHeight = srcHeight
	}

	// Handle different resize modes
	if opts.Crop == "center" && targetWidth > 0 && targetHeight > 0 {
		// Fill the box and crop excess from center
		return imaging.Fill(img, targetWidth, targetHeight, imaging.Center, imaging.Lanczos)
	}

	if targetWidth > 0 && targetHeight > 0 {
		// Fit within box, preserve aspect ratio
		return imaging.Fit(img, targetWidth, targetHeight, imaging.Lanczos)
	}

	if targetWidth > 0 {
		// Resize to width, preserve aspect ratio
		return imaging.Resize(img, targetWidth, 0, imaging.Lanczos)
	}

	if targetHeight > 0 {
		// Resize to height, preserve aspect ratio
		return imaging.Resize(img, 0, targetHeight, imaging.Lanczos)
	}

	return img
}

// Encode writes an image to a buffer in the specified format.
// It returns the encoded bytes and any error.
func Encode(img image.Image, format string, quality int) ([]byte, error) {
	var buf bytes.Buffer

	format, _ = NormalizeFormat(format)

	switch format {
	case "jpeg":
		q := quality
		if q == 0 {
			q = DefaultQuality("jpeg")
		}
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
			return nil, fmt.Errorf("encode jpeg: %w", err)
		}

	case "png":
		encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
		if err := encoder.Encode(&buf, img); err != nil {
			return nil, fmt.Errorf("encode png: %w", err)
		}

	case "gif":
		if err := gif.Encode(&buf, img, nil); err != nil {
			return nil, fmt.Errorf("encode gif: %w", err)
		}

	case "webp":
		// WebP encoding requires gen2brain/webp which is not yet included.
		// Return an error so the caller knows WebP output isn't supported.
		return nil, fmt.Errorf("WebP output encoding not supported; use format: \"jpeg\" or \"png\" instead")

	default:
		return nil, fmt.Errorf("unsupported output format: %s", format)
	}

	return buf.Bytes(), nil
}

// Process performs the full transformation pipeline: load → transform → encode.
// It returns the encoded bytes, the output extension (with leading dot), and any error.
func Process(sourcePath string, opts TransformOptions) ([]byte, string, error) {
	// Load the image
	img, sourceFormat, err := Load(sourcePath)
	if err != nil {
		return nil, "", err
	}

	// Apply transformations
	img = Transform(img, opts)

	// Determine output format
	outputFormat := opts.Format
	if outputFormat == "" {
		outputFormat = sourceFormat
	}

	// WebP input but no explicit output format: fall back to JPEG since we can't encode WebP
	if outputFormat == "webp" && opts.Format == "" {
		outputFormat = "jpeg"
	}

	_, ext := NormalizeFormat(outputFormat)

	// Encode to output format
	data, err := Encode(img, outputFormat, opts.Quality)
	if err != nil {
		return nil, "", err
	}

	return data, ext, nil
}

// GetImageInfo returns metadata about an image without fully decoding it.
// This is more efficient than Load() when you only need dimensions/format.
func GetImageInfo(path string) (width, height int, format string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, "", fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	// Use image.DecodeConfig for efficient metadata-only read
	cfg, formatName, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, "", fmt.Errorf("decode config %s: %w", path, err)
	}

	return cfg.Width, cfg.Height, formatName, nil
}

// ImageInfo holds metadata about an image.
type ImageInfo struct {
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Format      string `json:"format"`
	Orientation string `json:"orientation"` // "landscape", "portrait", or "square"
}

// GetInfo returns full image metadata.
func GetInfo(path string) (ImageInfo, error) {
	width, height, format, err := GetImageInfo(path)
	if err != nil {
		return ImageInfo{}, err
	}

	var orientation string
	switch {
	case width > height:
		orientation = "landscape"
	case height > width:
		orientation = "portrait"
	default:
		orientation = "square"
	}

	return ImageInfo{
		Width:       width,
		Height:      height,
		Format:      format,
		Orientation: orientation,
	}, nil
}

// SourceHash computes a content hash for a file.
// The hash is truncated to 16 hex characters, matching the assetRegistry pattern.
func SourceHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	// Read file and compute hash
	data, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}

	return sha256Short(data), nil
}

// sha256Short computes SHA256 and returns first 16 hex chars.
// This matches the assetRegistry pattern for content-addressed URLs.
func sha256Short(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])[:16]
}
