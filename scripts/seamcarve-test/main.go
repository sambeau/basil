// Script to test seam carving on images.
// Usage: go run ./scripts/seamcarve-test/main.go [options] <image>
//
// Examples:
//
//	go run ./scripts/seamcarve-test/main.go testdata/images/smartcrop/landscapes/landscape-minimal-1.jpg
//	go run ./scripts/seamcarve-test/main.go -enlarge 20 testdata/images/smartcrop/landscapes/landscape-minimal-1.jpg
//	go run ./scripts/seamcarve-test/main.go -reduce 20 -dir height testdata/images/smartcrop/objects/object-centered-1.jpg
package main

import (
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/sambeau/basil/server/images/seamcarve"
)

func main() {
	// Flags
	enlargePercent := flag.Int("enlarge", 0, "Enlarge by percentage (e.g., 20 for 20%)")
	reducePercent := flag.Int("reduce", 0, "Reduce by percentage (e.g., 20 for 20%)")
	direction := flag.String("dir", "width", "Direction to modify: width, height, or both")
	outputDir := flag.String("out", "testdata/images/seamcarve-output", "Output directory")
	maxSize := flag.Int("max", 800, "Max dimension before processing (downscale large images)")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: go run ./scripts/seamcarve-test/main.go [options] <image>")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  go run ./scripts/seamcarve-test/main.go -enlarge 20 testdata/images/smartcrop/landscapes/landscape-minimal-1.jpg")
		fmt.Println("  go run ./scripts/seamcarve-test/main.go -reduce 15 -dir both testdata/images/smartcrop/objects/object-centered-1.jpg")
		os.Exit(1)
	}

	if *enlargePercent == 0 && *reducePercent == 0 {
		*enlargePercent = 15 // Default: enlarge by 15%
	}

	imagePath := flag.Arg(0)

	// Load image
	img, err := imaging.Open(imagePath, imaging.AutoOrientation(true))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading image: %v\n", err)
		os.Exit(1)
	}

	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()
	fmt.Printf("Loaded: %s (%dx%d)\n", imagePath, srcWidth, srcHeight)

	// Downscale if too large (for faster processing)
	if srcWidth > *maxSize || srcHeight > *maxSize {
		img = imaging.Fit(img, *maxSize, *maxSize, imaging.Lanczos)
		bounds = img.Bounds()
		srcWidth = bounds.Dx()
		srcHeight = bounds.Dy()
		fmt.Printf("Downscaled to: %dx%d\n", srcWidth, srcHeight)
	}

	// Calculate target dimensions
	targetWidth := srcWidth
	targetHeight := srcHeight

	if *enlargePercent > 0 {
		delta := float64(*enlargePercent) / 100.0
		if *direction == "width" || *direction == "both" {
			targetWidth = int(float64(srcWidth) * (1 + delta))
		}
		if *direction == "height" || *direction == "both" {
			targetHeight = int(float64(srcHeight) * (1 + delta))
		}
	} else if *reducePercent > 0 {
		delta := float64(*reducePercent) / 100.0
		if *direction == "width" || *direction == "both" {
			targetWidth = int(float64(srcWidth) * (1 - delta))
		}
		if *direction == "height" || *direction == "both" {
			targetHeight = int(float64(srcHeight) * (1 - delta))
		}
	}

	fmt.Printf("Target dimensions: %dx%d\n", targetWidth, targetHeight)

	// Run seam carving
	start := time.Now()
	result := seamcarve.Resize(img, targetWidth, targetHeight)
	elapsed := time.Since(start)

	resultBounds := result.Bounds()
	fmt.Printf("Result: %dx%d (took %v)\n", resultBounds.Dx(), resultBounds.Dy(), elapsed)

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Generate output filename
	baseName := filepath.Base(imagePath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	var suffix string
	if *enlargePercent > 0 {
		suffix = fmt.Sprintf("_enlarge%d_%s", *enlargePercent, *direction)
	} else {
		suffix = fmt.Sprintf("_reduce%d_%s", *reducePercent, *direction)
	}

	// Save original (downscaled) for comparison
	originalPath := filepath.Join(*outputDir, nameWithoutExt+"_original.jpg")
	if err := saveImage(img, originalPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving original: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Saved original: %s\n", originalPath)

	// Save result
	resultPath := filepath.Join(*outputDir, nameWithoutExt+suffix+".jpg")
	if err := saveImage(result, resultPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving result: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Saved result: %s\n", resultPath)

	// Also save a standard resize for comparison
	var standardResize image.Image
	if *enlargePercent > 0 {
		standardResize = imaging.Resize(img, targetWidth, targetHeight, imaging.Lanczos)
	} else {
		standardResize = imaging.Resize(img, targetWidth, targetHeight, imaging.Lanczos)
	}
	standardPath := filepath.Join(*outputDir, nameWithoutExt+suffix+"_standard.jpg")
	if err := saveImage(standardResize, standardPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving standard resize: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Saved standard resize: %s\n", standardPath)

	fmt.Println()
	fmt.Println("Compare the results:")
	fmt.Printf("  Original:        %s\n", originalPath)
	fmt.Printf("  Seam carved:     %s\n", resultPath)
	fmt.Printf("  Standard resize: %s\n", standardPath)
}

func saveImage(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return png.Encode(f, img)
	default:
		return jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
	}
}
