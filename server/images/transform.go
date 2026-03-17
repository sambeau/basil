package images

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
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
	"github.com/gen2brain/webp"
	"github.com/sambeau/basil/server/images/seamcarve"
	"github.com/sambeau/basil/server/images/smartcrop"
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
// If sharpening is enabled (opts.Sharpen > 0) and the image was downscaled,
// sharpening is applied after the resize.
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
	var result image.Image

	// Smart scale: content-aware resizing using seam carving
	if opts.Scale == "smart" {
		// Seam carving for content-aware size reduction
		// Determine target dimensions (preserve aspect ratio if only one specified)
		seamTargetW := targetWidth
		seamTargetH := targetHeight

		if seamTargetW == 0 && seamTargetH > 0 {
			// Calculate width to preserve aspect ratio
			seamTargetW = int(float64(srcWidth) * float64(seamTargetH) / float64(srcHeight))
		} else if seamTargetH == 0 && seamTargetW > 0 {
			// Calculate height to preserve aspect ratio
			seamTargetH = int(float64(srcHeight) * float64(seamTargetW) / float64(srcWidth))
		}

		if seamTargetW > 0 && seamTargetH > 0 {
			result = seamcarve.Resize(img, seamTargetW, seamTargetH)
		} else {
			result = img
		}
	} else if opts.Crop == "smart" && targetWidth > 0 && targetHeight > 0 {
		// Smart crop: content-aware cropping with face detection
		// Convert FocalRegion to smartcrop.FocalRegion if present
		var focal *smartcrop.FocalRegion
		if opts.Focal != nil {
			focal = &smartcrop.FocalRegion{
				X: opts.Focal.X,
				Y: opts.Focal.Y,
				W: opts.Focal.W,
				H: opts.Focal.H,
			}
		}

		// Get the best crop rectangle
		cropRect := smartcrop.BestCrop(img, targetWidth, targetHeight, focal)

		// Crop to the selected region
		result = imaging.Crop(img, cropRect)

		// Resize to exact target dimensions (crop may be larger due to aspect ratio matching)
		resultBounds := result.Bounds()
		if resultBounds.Dx() != targetWidth || resultBounds.Dy() != targetHeight {
			result = imaging.Resize(result, targetWidth, targetHeight, imaging.Lanczos)
		}
	} else if opts.Crop == "center" && targetWidth > 0 && targetHeight > 0 {
		// Fill the box and crop excess from center
		result = imaging.Fill(img, targetWidth, targetHeight, imaging.Center, imaging.Lanczos)
	} else if targetWidth > 0 && targetHeight > 0 {
		// Fit within box, preserve aspect ratio
		result = imaging.Fit(img, targetWidth, targetHeight, imaging.Lanczos)
	} else if targetWidth > 0 {
		// Resize to width, preserve aspect ratio
		result = imaging.Resize(img, targetWidth, 0, imaging.Lanczos)
	} else if targetHeight > 0 {
		// Resize to height, preserve aspect ratio
		result = imaging.Resize(img, 0, targetHeight, imaging.Lanczos)
	} else {
		result = img
	}

	// Apply sharpening if enabled and image was downscaled
	if opts.Sharpen > 0 {
		resultBounds := result.Bounds()
		wasDownscaled := resultBounds.Dx() < srcWidth || resultBounds.Dy() < srcHeight
		if wasDownscaled {
			result = imaging.Sharpen(result, opts.Sharpen)
		}
	}

	return result
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
		q := quality
		if q == 0 {
			q = DefaultQuality("webp")
		}
		if err := webp.Encode(&buf, img, webp.Options{Quality: q}); err != nil {
			return nil, fmt.Errorf("encode webp: %w", err)
		}

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
// For JPEG images, the EXIF orientation tag is read and dimensions are swapped
// for orientations 5–8 (which rotate the image 90°/270°), so the returned
// dimensions match what the user sees after auto-orientation.
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

	w, h := cfg.Width, cfg.Height

	// For JPEG, check EXIF orientation and swap dimensions if needed.
	// Orientations 5–8 involve a 90° or 270° rotation that swaps width/height.
	if formatName == "jpeg" {
		orient := readJPEGOrientation(path)
		if orient >= 5 && orient <= 8 {
			w, h = h, w
		}
	}

	return w, h, formatName, nil
}

// readJPEGOrientation reads the EXIF orientation tag from a JPEG file.
// Returns 0 if the file is not JPEG, has no EXIF data, or the orientation
// tag is missing or unreadable. Valid values are 1–8.
func readJPEGOrientation(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer func() { _ = f.Close() }()

	const (
		markerSOI      = 0xffd8
		markerAPP1     = 0xffe1
		exifHeader     = 0x45786966 // "Exif"
		byteOrderBE    = 0x4d4d     // "MM"
		byteOrderLE    = 0x4949     // "II"
		orientationTag = 0x0112
	)

	// Check JPEG SOI marker
	var soi uint16
	if err := binary.Read(f, binary.BigEndian, &soi); err != nil || soi != markerSOI {
		return 0
	}

	// Find APP1 marker containing EXIF data
	for {
		var marker, size uint16
		if err := binary.Read(f, binary.BigEndian, &marker); err != nil {
			return 0
		}
		if err := binary.Read(f, binary.BigEndian, &size); err != nil {
			return 0
		}
		if marker>>8 != 0xff {
			return 0
		}
		if marker == markerAPP1 {
			break
		}
		if size < 2 {
			return 0
		}
		if _, err := io.CopyN(io.Discard, f, int64(size-2)); err != nil {
			return 0
		}
	}

	// Check EXIF header ("Exif\0\0")
	var header uint32
	if err := binary.Read(f, binary.BigEndian, &header); err != nil || header != exifHeader {
		return 0
	}
	if _, err := io.CopyN(io.Discard, f, 2); err != nil {
		return 0
	}

	// Read byte order
	var byteOrderTag uint16
	if err := binary.Read(f, binary.BigEndian, &byteOrderTag); err != nil {
		return 0
	}
	var byteOrder binary.ByteOrder
	switch byteOrderTag {
	case byteOrderBE:
		byteOrder = binary.BigEndian
	case byteOrderLE:
		byteOrder = binary.LittleEndian
	default:
		return 0
	}
	if _, err := io.CopyN(io.Discard, f, 2); err != nil {
		return 0
	}

	// Read IFD offset and skip to it
	var offset uint32
	if err := binary.Read(f, byteOrder, &offset); err != nil {
		return 0
	}
	if offset < 8 {
		return 0
	}
	if _, err := io.CopyN(io.Discard, f, int64(offset-8)); err != nil {
		return 0
	}

	// Read number of IFD entries
	var numTags uint16
	if err := binary.Read(f, byteOrder, &numTags); err != nil {
		return 0
	}

	// Search for orientation tag
	for i := 0; i < int(numTags); i++ {
		var tag uint16
		if err := binary.Read(f, byteOrder, &tag); err != nil {
			return 0
		}
		if tag != orientationTag {
			// Skip rest of this IFD entry (10 bytes: type(2) + count(4) + value(4))
			if _, err := io.CopyN(io.Discard, f, 10); err != nil {
				return 0
			}
			continue
		}
		// Skip type(2) and count(4)
		if _, err := io.CopyN(io.Discard, f, 6); err != nil {
			return 0
		}
		var val uint16
		if err := binary.Read(f, byteOrder, &val); err != nil {
			return 0
		}
		if val < 1 || val > 8 {
			return 0
		}
		return int(val)
	}

	return 0
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
// Uses streaming to avoid loading the entire file into memory.
func SourceHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}

	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

// BlurPlaceholderWidth is the target width for LQIP thumbnails.
const BlurPlaceholderWidth = 20

// BlurPlaceholderSigma is the Gaussian blur sigma for LQIP.
const BlurPlaceholderSigma = 10.0

// BlurPlaceholderQuality is the JPEG quality for LQIP encoding.
const BlurPlaceholderQuality = 20

// GenerateBlurPlaceholder creates a Low Quality Image Placeholder (LQIP) data URI.
// The pipeline: resize to 20px wide → Gaussian blur (σ=10) → JPEG q=20 → base64.
// Returns a data URI string: "data:image/jpeg;base64,..."
func GenerateBlurPlaceholder(sourcePath string) (string, error) {
	// Load the full image
	img, _, err := Load(sourcePath)
	if err != nil {
		return "", fmt.Errorf("load image: %w", err)
	}

	// Resize to thumbnail width (preserving aspect ratio)
	thumbnail := imaging.Resize(img, BlurPlaceholderWidth, 0, imaging.Lanczos)

	// Apply Gaussian blur
	blurred := imaging.Blur(thumbnail, BlurPlaceholderSigma)

	// Encode as JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, blurred, &jpeg.Options{Quality: BlurPlaceholderQuality}); err != nil {
		return "", fmt.Errorf("encode jpeg: %w", err)
	}

	// Base64 encode and return as data URI
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return "data:image/jpeg;base64," + encoded, nil
}
