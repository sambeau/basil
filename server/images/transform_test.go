package images

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/disintegration/imaging"
)

// createTestJPEGWithOrientation creates a JPEG file with an EXIF orientation tag.
// The image is stored with the given raw dimensions (rawWidth x rawHeight),
// and the EXIF orientation tag is set to the given value (1–8).
func createTestJPEGWithOrientation(t *testing.T, dir, name string, rawWidth, rawHeight int, orientation int) string {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, rawWidth, rawHeight))
	for y := 0; y < rawHeight; y++ {
		for x := 0; x < rawWidth; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 50, A: 255})
		}
	}

	// Encode to JPEG in memory
	var jpegBuf bytes.Buffer
	if err := jpeg.Encode(&jpegBuf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	jpegData := jpegBuf.Bytes()

	// Build minimal EXIF APP1 segment with orientation tag.
	// Structure: APP1 marker (FFE1) | size (2) | "Exif\0\0" (6) | TIFF header + IFD
	var exif bytes.Buffer

	// TIFF header: byte order (little-endian "II"), magic 42, offset to IFD0 = 8
	binary.Write(&exif, binary.LittleEndian, uint16(0x4949)) // "II" = little-endian
	binary.Write(&exif, binary.LittleEndian, uint16(42))     // TIFF magic
	binary.Write(&exif, binary.LittleEndian, uint32(8))      // offset to IFD0

	// IFD0: 1 entry
	binary.Write(&exif, binary.LittleEndian, uint16(1)) // number of entries

	// IFD entry: tag=0x0112 (orientation), type=3 (SHORT), count=1, value=orientation
	binary.Write(&exif, binary.LittleEndian, uint16(0x0112))      // tag
	binary.Write(&exif, binary.LittleEndian, uint16(3))           // type SHORT
	binary.Write(&exif, binary.LittleEndian, uint32(1))           // count
	binary.Write(&exif, binary.LittleEndian, uint16(orientation)) // value
	binary.Write(&exif, binary.LittleEndian, uint16(0))           // padding to 4 bytes

	// Next IFD offset = 0 (no more IFDs)
	binary.Write(&exif, binary.LittleEndian, uint32(0))

	// Build APP1 segment: marker + size + "Exif\0\0" + TIFF data
	var app1 bytes.Buffer
	app1.Write([]byte{0xFF, 0xE1}) // APP1 marker
	exifPayload := append([]byte("Exif\x00\x00"), exif.Bytes()...)
	binary.Write(&app1, binary.BigEndian, uint16(len(exifPayload)+2)) // size includes itself
	app1.Write(exifPayload)

	// Insert APP1 right after SOI (first 2 bytes of JPEG)
	var result bytes.Buffer
	result.Write(jpegData[:2]) // SOI marker
	result.Write(app1.Bytes()) // APP1 with EXIF
	result.Write(jpegData[2:]) // rest of JPEG

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, result.Bytes(), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return path
}

// createTestImage creates a simple test image file.
func createTestImage(t *testing.T, dir, name string, width, height int, format string) string {
	t.Helper()

	// Create a simple colored image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: 100,
				A: 255,
			})
		}
	}

	path := filepath.Join(dir, name)

	// Encode to file
	data, err := Encode(img, format, 0)
	if err != nil {
		t.Fatalf("failed to encode test image: %v", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write test image: %v", err)
	}

	return path
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name       string
		filename   string
		width      int
		height     int
		format     string
		wantFormat string
	}{
		{
			name:       "load jpeg",
			filename:   "test.jpg",
			width:      100,
			height:     80,
			format:     "jpeg",
			wantFormat: "jpeg",
		},
		{
			name:       "load png",
			filename:   "test.png",
			width:      120,
			height:     90,
			format:     "png",
			wantFormat: "png",
		},
		{
			name:       "load gif",
			filename:   "test.gif",
			width:      50,
			height:     50,
			format:     "gif",
			wantFormat: "gif",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := createTestImage(t, dir, tt.filename, tt.width, tt.height, tt.format)

			img, format, err := Load(path)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if format != tt.wantFormat {
				t.Errorf("Load() format = %q, want %q", format, tt.wantFormat)
			}

			bounds := img.Bounds()
			if bounds.Dx() != tt.width || bounds.Dy() != tt.height {
				t.Errorf("Load() dimensions = %dx%d, want %dx%d",
					bounds.Dx(), bounds.Dy(), tt.width, tt.height)
			}
		})
	}
}

func TestLoad_Errors(t *testing.T) {
	dir := t.TempDir()

	t.Run("file not found", func(t *testing.T) {
		_, _, err := Load(filepath.Join(dir, "nonexistent.jpg"))
		if err == nil {
			t.Error("Load() expected error for nonexistent file")
		}
	})

	t.Run("unsupported format", func(t *testing.T) {
		path := filepath.Join(dir, "test.bmp")
		if err := os.WriteFile(path, []byte("not a real bmp"), 0644); err != nil {
			t.Fatal(err)
		}

		_, _, err := Load(path)
		if err == nil {
			t.Error("Load() expected error for unsupported format")
		}
	})

	t.Run("file too large", func(t *testing.T) {
		// We can't easily create a 50MB+ file in tests, so we skip this
		t.Skip("skipping large file test")
	})
}

func TestTransform(t *testing.T) {
	// Create a 200x100 test image
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}

	tests := []struct {
		name       string
		opts       TransformOptions
		wantWidth  int
		wantHeight int
	}{
		{
			name:       "no transform",
			opts:       TransformOptions{},
			wantWidth:  200,
			wantHeight: 100,
		},
		{
			name:       "resize by width only",
			opts:       TransformOptions{Width: 100},
			wantWidth:  100,
			wantHeight: 50, // Aspect ratio preserved
		},
		{
			name:       "resize by height only",
			opts:       TransformOptions{Height: 50},
			wantWidth:  100, // Aspect ratio preserved
			wantHeight: 50,
		},
		{
			name:       "fit within box",
			opts:       TransformOptions{Width: 80, Height: 80},
			wantWidth:  80,
			wantHeight: 40, // Fits within 80x80 preserving 2:1 aspect ratio
		},
		{
			name:       "crop center",
			opts:       TransformOptions{Width: 50, Height: 50, Crop: "center"},
			wantWidth:  50,
			wantHeight: 50, // Cropped to exact dimensions
		},
		{
			name:       "no upscaling width",
			opts:       TransformOptions{Width: 400}, // Larger than source
			wantWidth:  200,                          // Clamped to source
			wantHeight: 100,
		},
		{
			name:       "no upscaling height",
			opts:       TransformOptions{Height: 200}, // Larger than source
			wantWidth:  200,
			wantHeight: 100, // Clamped to source
		},
		{
			name:       "no upscaling both",
			opts:       TransformOptions{Width: 400, Height: 200},
			wantWidth:  200,
			wantHeight: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Transform(img, tt.opts)
			bounds := result.Bounds()

			if bounds.Dx() != tt.wantWidth || bounds.Dy() != tt.wantHeight {
				t.Errorf("Transform() = %dx%d, want %dx%d",
					bounds.Dx(), bounds.Dy(), tt.wantWidth, tt.wantHeight)
			}
		})
	}
}

func TestEncode(t *testing.T) {
	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	tests := []struct {
		name    string
		format  string
		quality int
		wantErr bool
	}{
		{
			name:   "encode jpeg default quality",
			format: "jpeg",
		},
		{
			name:    "encode jpeg with quality",
			format:  "jpeg",
			quality: 75,
		},
		{
			name:   "encode png",
			format: "png",
		},
		{
			name:   "encode gif",
			format: "gif",
		},
		{
			name:    "unsupported format",
			format:  "bmp",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := Encode(img, tt.format, tt.quality)
			if tt.wantErr {
				if err == nil {
					t.Error("Encode() expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("Encode() returned empty data")
			}
		})
	}
}

func TestEncode_QualityAffectsSize(t *testing.T) {
	// Create a larger test image for quality comparison
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			// Create some variation for compression to work with
			img.Set(x, y, color.RGBA{
				R: uint8(x % 256),
				G: uint8(y % 256),
				B: uint8((x + y) % 256),
				A: 255,
			})
		}
	}

	lowQuality, err := Encode(img, "jpeg", 10)
	if err != nil {
		t.Fatal(err)
	}

	highQuality, err := Encode(img, "jpeg", 95)
	if err != nil {
		t.Fatal(err)
	}

	if len(lowQuality) >= len(highQuality) {
		t.Errorf("Expected low quality (%d bytes) to be smaller than high quality (%d bytes)",
			len(lowQuality), len(highQuality))
	}
}

func TestProcess(t *testing.T) {
	dir := t.TempDir()

	// Create a test image
	path := createTestImage(t, dir, "source.jpg", 200, 100, "jpeg")

	tests := []struct {
		name    string
		opts    TransformOptions
		wantExt string
	}{
		{
			name:    "preserve format",
			opts:    TransformOptions{Width: 100},
			wantExt: ".jpg",
		},
		{
			name:    "convert to png",
			opts:    TransformOptions{Width: 100, Format: "png"},
			wantExt: ".png",
		},
		{
			name:    "with crop",
			opts:    TransformOptions{Width: 50, Height: 50, Crop: "center"},
			wantExt: ".jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, ext, err := Process(path, tt.opts)
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}

			if ext != tt.wantExt {
				t.Errorf("Process() ext = %q, want %q", ext, tt.wantExt)
			}

			if len(data) == 0 {
				t.Error("Process() returned empty data")
			}
		})
	}
}

func TestGetImageInfo(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name       string
		filename   string
		width      int
		height     int
		format     string
		wantFormat string
	}{
		{
			name:       "jpeg info",
			filename:   "test.jpg",
			width:      100,
			height:     80,
			format:     "jpeg",
			wantFormat: "jpeg",
		},
		{
			name:       "png info",
			filename:   "test.png",
			width:      120,
			height:     90,
			format:     "png",
			wantFormat: "png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := createTestImage(t, dir, tt.filename, tt.width, tt.height, tt.format)

			width, height, format, err := GetImageInfo(path)
			if err != nil {
				t.Fatalf("GetImageInfo() error = %v", err)
			}

			if width != tt.width || height != tt.height {
				t.Errorf("GetImageInfo() dimensions = %dx%d, want %dx%d",
					width, height, tt.width, tt.height)
			}

			if format != tt.wantFormat {
				t.Errorf("GetImageInfo() format = %q, want %q", format, tt.wantFormat)
			}
		})
	}
}

func TestGetInfo(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name            string
		filename        string
		width           int
		height          int
		format          string
		wantOrientation string
	}{
		{
			name:            "landscape",
			filename:        "landscape.jpg",
			width:           200,
			height:          100,
			format:          "jpeg",
			wantOrientation: "landscape",
		},
		{
			name:            "portrait",
			filename:        "portrait.jpg",
			width:           100,
			height:          200,
			format:          "jpeg",
			wantOrientation: "portrait",
		},
		{
			name:            "square",
			filename:        "square.jpg",
			width:           100,
			height:          100,
			format:          "jpeg",
			wantOrientation: "square",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := createTestImage(t, dir, tt.filename, tt.width, tt.height, tt.format)

			info, err := GetInfo(path)
			if err != nil {
				t.Fatalf("GetInfo() error = %v", err)
			}

			if info.Width != tt.width {
				t.Errorf("GetInfo() Width = %d, want %d", info.Width, tt.width)
			}

			if info.Height != tt.height {
				t.Errorf("GetInfo() Height = %d, want %d", info.Height, tt.height)
			}

			if info.Orientation != tt.wantOrientation {
				t.Errorf("GetInfo() Orientation = %q, want %q", info.Orientation, tt.wantOrientation)
			}
		})
	}
}

func TestSourceHash(t *testing.T) {
	dir := t.TempDir()

	// Create a test file
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Hash should be deterministic
	hash1, err := SourceHash(path)
	if err != nil {
		t.Fatalf("SourceHash() error = %v", err)
	}

	hash2, err := SourceHash(path)
	if err != nil {
		t.Fatalf("SourceHash() error = %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("SourceHash() not deterministic: %q != %q", hash1, hash2)
	}

	// Hash should be 16 chars
	if len(hash1) != 16 {
		t.Errorf("SourceHash() length = %d, want 16", len(hash1))
	}

	// Different content should produce different hash
	path2 := filepath.Join(dir, "test2.txt")
	if err := os.WriteFile(path2, []byte("different content"), 0644); err != nil {
		t.Fatal(err)
	}

	hash3, err := SourceHash(path2)
	if err != nil {
		t.Fatal(err)
	}

	if hash1 == hash3 {
		t.Error("SourceHash() should produce different hashes for different content")
	}
}

func TestEncode_WebPOutput(t *testing.T) {
	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := range 50 {
		for x := range 50 {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	// WebP output should succeed now that we have the encoder
	data, err := Encode(img, "webp", 80)
	if err != nil {
		t.Fatalf("Encode(webp) error = %v", err)
	}
	if len(data) == 0 {
		t.Error("Encode(webp) returned empty data")
	}

	// Verify it starts with a RIFF/WEBP header
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		t.Error("Encode(webp) output does not have RIFF/WEBP header")
	}
}

func TestTransform_Sharpen(t *testing.T) {
	// Create a 200x100 test image with a pattern
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			// Create a gradient pattern
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / 200),
				G: uint8((y * 255) / 100),
				B: 128,
				A: 255,
			})
		}
	}

	t.Run("sharpen applied on downscale", func(t *testing.T) {
		// Transform with sharpen
		optsWithSharpen := TransformOptions{Width: 100, Sharpen: 0.5}
		resultSharpened := Transform(img, optsWithSharpen)

		// Transform without sharpen
		optsNoSharpen := TransformOptions{Width: 100, SharpenDisabled: true}
		resultUnsharpened := Transform(img, optsNoSharpen)

		// Both should have the same dimensions
		if resultSharpened.Bounds() != resultUnsharpened.Bounds() {
			t.Error("Sharpened and unsharpened should have same dimensions")
		}

		// But they should differ in pixel values (sharpening modifies pixels)
		// Compare a few pixels to verify sharpening had an effect
		sharpenedNRGBA := imaging.Clone(resultSharpened)
		unsharpenedNRGBA := imaging.Clone(resultUnsharpened)

		different := false
		bounds := sharpenedNRGBA.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y && !different; y++ {
			for x := bounds.Min.X; x < bounds.Max.X && !different; x++ {
				if sharpenedNRGBA.At(x, y) != unsharpenedNRGBA.At(x, y) {
					different = true
				}
			}
		}

		if !different {
			t.Error("Sharpening should produce different pixel values")
		}
	})

	t.Run("sharpen not applied without downscale", func(t *testing.T) {
		// No resize (same dimensions) - sharpen should NOT be applied
		optsNoResize := TransformOptions{Sharpen: 0.5}
		result := Transform(img, optsNoResize)

		// Result should be identical to input (no transform, no sharpen)
		if result.Bounds() != img.Bounds() {
			t.Error("No-resize transform should preserve dimensions")
		}
	})

	t.Run("sharpen not applied on upscale request", func(t *testing.T) {
		// Request larger than source - gets clamped, no actual resize
		optsUpscale := TransformOptions{Width: 400, Sharpen: 0.5}
		result := Transform(img, optsUpscale)

		// Should be clamped to source size (no upscaling)
		bounds := result.Bounds()
		if bounds.Dx() != 200 || bounds.Dy() != 100 {
			t.Errorf("Upscale should be clamped to source: got %dx%d, want 200x100",
				bounds.Dx(), bounds.Dy())
		}
	})

	t.Run("sharpen disabled explicitly", func(t *testing.T) {
		optsDisabled := TransformOptions{Width: 100, SharpenDisabled: true}
		result := Transform(img, optsDisabled)

		// Just verify it doesn't panic and produces correct dimensions
		bounds := result.Bounds()
		if bounds.Dx() != 100 {
			t.Errorf("Transform with sharpen disabled: got width %d, want 100", bounds.Dx())
		}
	})

	t.Run("explicit sharpen sigma", func(t *testing.T) {
		opts := TransformOptions{Width: 100, Sharpen: 1.5}
		result := Transform(img, opts)

		bounds := result.Bounds()
		if bounds.Dx() != 100 {
			t.Errorf("Transform with explicit sharpen: got width %d, want 100", bounds.Dx())
		}
	})
}

func TestGenerateBlurPlaceholder(t *testing.T) {
	dir := t.TempDir()

	t.Run("generates valid data URI", func(t *testing.T) {
		path := createTestImage(t, dir, "blur_test.jpg", 400, 300, "jpeg")

		dataURI, err := GenerateBlurPlaceholder(path)
		if err != nil {
			t.Fatalf("GenerateBlurPlaceholder() error = %v", err)
		}

		// Should start with data URI prefix
		prefix := "data:image/jpeg;base64,"
		if !strings.HasPrefix(dataURI, prefix) {
			t.Errorf("GenerateBlurPlaceholder() should start with %q, got %q", prefix, dataURI[:min(len(dataURI), 50)])
		}

		// Should be reasonable size (typically 600-900 bytes encoded)
		if len(dataURI) < 100 || len(dataURI) > 2000 {
			t.Errorf("GenerateBlurPlaceholder() unexpected length = %d, want 100-2000", len(dataURI))
		}
	})

	t.Run("decodes to valid JPEG", func(t *testing.T) {
		path := createTestImage(t, dir, "blur_decode.jpg", 200, 150, "jpeg")

		dataURI, err := GenerateBlurPlaceholder(path)
		if err != nil {
			t.Fatalf("GenerateBlurPlaceholder() error = %v", err)
		}

		// Extract base64 data
		prefix := "data:image/jpeg;base64,"
		if !strings.HasPrefix(dataURI, prefix) {
			t.Fatal("Invalid data URI prefix")
		}
		b64Data := dataURI[len(prefix):]

		// Decode base64
		jpegData, err := base64.StdEncoding.DecodeString(b64Data)
		if err != nil {
			t.Fatalf("Base64 decode error = %v", err)
		}

		// Decode JPEG
		img, err := jpeg.Decode(bytes.NewReader(jpegData))
		if err != nil {
			t.Fatalf("JPEG decode error = %v", err)
		}

		// Should be BlurPlaceholderWidth wide (20px)
		bounds := img.Bounds()
		if bounds.Dx() != BlurPlaceholderWidth {
			t.Errorf("Blur placeholder width = %d, want %d", bounds.Dx(), BlurPlaceholderWidth)
		}
	})

	t.Run("works with PNG source", func(t *testing.T) {
		path := createTestImage(t, dir, "blur_png.png", 300, 200, "png")

		dataURI, err := GenerateBlurPlaceholder(path)
		if err != nil {
			t.Fatalf("GenerateBlurPlaceholder() error = %v", err)
		}

		// Should still output JPEG data URI
		if !strings.HasPrefix(dataURI, "data:image/jpeg;base64,") {
			t.Error("PNG source should still produce JPEG data URI")
		}
	})

	t.Run("works with GIF source", func(t *testing.T) {
		path := createTestImage(t, dir, "blur_gif.gif", 100, 100, "gif")

		dataURI, err := GenerateBlurPlaceholder(path)
		if err != nil {
			t.Fatalf("GenerateBlurPlaceholder() error = %v", err)
		}

		if !strings.HasPrefix(dataURI, "data:image/jpeg;base64,") {
			t.Error("GIF source should still produce JPEG data URI")
		}
	})

	t.Run("error on non-existent file", func(t *testing.T) {
		_, err := GenerateBlurPlaceholder(filepath.Join(dir, "nonexistent.jpg"))
		if err == nil {
			t.Error("GenerateBlurPlaceholder() expected error for non-existent file")
		}
	})

	t.Run("error on unsupported format", func(t *testing.T) {
		// Create a file with unsupported extension
		unsupportedPath := filepath.Join(dir, "test.bmp")
		if err := os.WriteFile(unsupportedPath, []byte("not a real image"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := GenerateBlurPlaceholder(unsupportedPath)
		if err == nil {
			t.Error("GenerateBlurPlaceholder() expected error for unsupported format")
		}
	})

	t.Run("preserves aspect ratio", func(t *testing.T) {
		// Wide image (400x100 = 4:1 aspect ratio)
		widePath := createTestImage(t, dir, "blur_wide.jpg", 400, 100, "jpeg")

		dataURI, err := GenerateBlurPlaceholder(widePath)
		if err != nil {
			t.Fatalf("GenerateBlurPlaceholder() error = %v", err)
		}

		// Decode and check dimensions
		prefix := "data:image/jpeg;base64,"
		b64Data := dataURI[len(prefix):]
		jpegData, _ := base64.StdEncoding.DecodeString(b64Data)
		img, _ := jpeg.Decode(bytes.NewReader(jpegData))

		bounds := img.Bounds()
		// 20px wide with 4:1 ratio should be 20x5
		if bounds.Dx() != 20 {
			t.Errorf("Wide image width = %d, want 20", bounds.Dx())
		}
		if bounds.Dy() != 5 {
			t.Errorf("Wide image height = %d, want 5", bounds.Dy())
		}
	})
}

func TestProcess_WebPOutput(t *testing.T) {
	dir := t.TempDir()

	// Create a JPEG test image
	path := createTestImage(t, dir, "test.jpg", 100, 100, "jpeg")

	// Process with explicit WebP format should produce WebP output
	data, ext, err := Process(path, TransformOptions{Width: 50, Format: "webp"})
	if err != nil {
		t.Fatalf("Process() with webp format error = %v", err)
	}
	if ext != ".webp" {
		t.Errorf("Process() ext = %q, want .webp", ext)
	}
	if len(data) == 0 {
		t.Error("Process() returned empty data")
	}
	// Verify RIFF/WEBP header
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		t.Error("Process() WebP output does not have RIFF/WEBP header")
	}

	// Process with explicit jpeg format should still work
	data, ext, err = Process(path, TransformOptions{Width: 50, Format: "jpeg"})
	if err != nil {
		t.Fatalf("Process() with explicit jpeg format error = %v", err)
	}
	if ext != ".jpg" {
		t.Errorf("Process() ext = %q, want .jpg", ext)
	}
	if len(data) == 0 {
		t.Error("Process() returned empty data")
	}

	// Process with explicit png format should still work
	data, ext, err = Process(path, TransformOptions{Width: 50, Format: "png"})
	if err != nil {
		t.Fatalf("Process() with explicit png format error = %v", err)
	}
	if ext != ".png" {
		t.Errorf("Process() ext = %q, want .png", ext)
	}
	if len(data) == 0 {
		t.Error("Process() returned empty data")
	}
}

func TestProcess_WebPSourcePreservesWebPByDefault(t *testing.T) {
	dir := t.TempDir()

	// Create a WebP source image
	path := createTestImage(t, dir, "test.webp", 100, 100, "webp")

	// Process with no explicit format should preserve WebP output
	data, ext, err := Process(path, TransformOptions{Width: 50})
	if err != nil {
		t.Fatalf("Process() with implicit source webp format error = %v", err)
	}
	if ext != ".webp" {
		t.Errorf("Process() ext = %q, want .webp", ext)
	}
	if len(data) == 0 {
		t.Error("Process() returned empty data")
	}
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		t.Error("Process() implicit WebP output does not have RIFF/WEBP header")
	}
}

func TestReadJPEGOrientation(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name        string
		orientation int
	}{
		{"normal (1)", 1},
		{"flip horizontal (2)", 2},
		{"rotate 180 (3)", 3},
		{"flip vertical (4)", 4},
		{"transpose (5)", 5},
		{"rotate 270 (6)", 6},
		{"transverse (7)", 7},
		{"rotate 90 (8)", 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := createTestJPEGWithOrientation(t, dir, "orient.jpg", 100, 80, tt.orientation)
			got := readJPEGOrientation(path)
			if got != tt.orientation {
				t.Errorf("readJPEGOrientation() = %d, want %d", got, tt.orientation)
			}
		})
	}
}

func TestReadJPEGOrientation_NoEXIF(t *testing.T) {
	dir := t.TempDir()

	// A plain JPEG without EXIF should return 0
	path := createTestImage(t, dir, "plain.jpg", 100, 80, "jpeg")
	got := readJPEGOrientation(path)
	if got != 0 {
		t.Errorf("readJPEGOrientation() on plain JPEG = %d, want 0", got)
	}
}

func TestReadJPEGOrientation_PNG(t *testing.T) {
	dir := t.TempDir()

	// A PNG should return 0 (not JPEG)
	path := createTestImage(t, dir, "test.png", 100, 80, "png")
	got := readJPEGOrientation(path)
	if got != 0 {
		t.Errorf("readJPEGOrientation() on PNG = %d, want 0", got)
	}
}

func TestGetImageInfo_EXIFOrientationSwapsDimensions(t *testing.T) {
	dir := t.TempDir()

	// Orientations 1–4 do NOT swap width/height
	// Orientations 5–8 DO swap width/height
	tests := []struct {
		name        string
		rawWidth    int
		rawHeight   int
		orientation int
		wantWidth   int
		wantHeight  int
	}{
		{
			name:     "orientation 1 (normal) - no swap",
			rawWidth: 200, rawHeight: 100,
			orientation: 1,
			wantWidth:   200, wantHeight: 100,
		},
		{
			name:     "orientation 3 (rotate 180) - no swap",
			rawWidth: 200, rawHeight: 100,
			orientation: 3,
			wantWidth:   200, wantHeight: 100,
		},
		{
			name:     "orientation 6 (rotate 270) - swap",
			rawWidth: 200, rawHeight: 100,
			orientation: 6,
			wantWidth:   100, wantHeight: 200,
		},
		{
			name:     "orientation 8 (rotate 90) - swap",
			rawWidth: 200, rawHeight: 100,
			orientation: 8,
			wantWidth:   100, wantHeight: 200,
		},
		{
			name:     "orientation 5 (transpose) - swap",
			rawWidth: 300, rawHeight: 150,
			orientation: 5,
			wantWidth:   150, wantHeight: 300,
		},
		{
			name:     "orientation 7 (transverse) - swap",
			rawWidth: 300, rawHeight: 150,
			orientation: 7,
			wantWidth:   150, wantHeight: 300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := createTestJPEGWithOrientation(t, dir, "test.jpg", tt.rawWidth, tt.rawHeight, tt.orientation)

			width, height, format, err := GetImageInfo(path)
			if err != nil {
				t.Fatalf("GetImageInfo() error = %v", err)
			}
			if format != "jpeg" {
				t.Errorf("GetImageInfo() format = %q, want %q", format, "jpeg")
			}
			if width != tt.wantWidth || height != tt.wantHeight {
				t.Errorf("GetImageInfo() = %dx%d, want %dx%d", width, height, tt.wantWidth, tt.wantHeight)
			}
		})
	}
}

func TestGetInfo_EXIFOrientationAffectsOrientationField(t *testing.T) {
	dir := t.TempDir()

	// A 200x100 image stored with orientation=6 (rotate 270°) should report
	// post-rotation dimensions 100x200, making it "portrait".
	path := createTestJPEGWithOrientation(t, dir, "rotated.jpg", 200, 100, 6)

	info, err := GetInfo(path)
	if err != nil {
		t.Fatalf("GetInfo() error = %v", err)
	}

	if info.Width != 100 || info.Height != 200 {
		t.Errorf("GetInfo() dimensions = %dx%d, want 100x200", info.Width, info.Height)
	}
	if info.Orientation != "portrait" {
		t.Errorf("GetInfo() Orientation = %q, want %q", info.Orientation, "portrait")
	}
}

func TestGetImageInfo_NoEXIF_DimensionsUnchanged(t *testing.T) {
	dir := t.TempDir()

	// A plain JPEG (no EXIF) should return raw dimensions unchanged
	path := createTestImage(t, dir, "plain.jpg", 200, 100, "jpeg")

	width, height, _, err := GetImageInfo(path)
	if err != nil {
		t.Fatalf("GetImageInfo() error = %v", err)
	}
	if width != 200 || height != 100 {
		t.Errorf("GetImageInfo() = %dx%d, want 200x100", width, height)
	}
}

func TestGetImageInfo_PNG_DimensionsUnchanged(t *testing.T) {
	dir := t.TempDir()

	// PNG has no EXIF orientation — dimensions should be unchanged
	path := createTestImage(t, dir, "test.png", 300, 150, "png")

	width, height, _, err := GetImageInfo(path)
	if err != nil {
		t.Fatalf("GetImageInfo() error = %v", err)
	}
	if width != 300 || height != 150 {
		t.Errorf("GetImageInfo() = %dx%d, want 300x150", width, height)
	}
}
