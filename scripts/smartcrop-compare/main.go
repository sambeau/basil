// Command smartcrop-compare generates side-by-side comparison grids
// of center crop vs smart crop for test images.
//
// Usage:
//
//	go run ./scripts/smartcrop-compare/main.go [flags]
//
// Flags:
//
//	-input   Input directory (default: testdata/images/smartcrop)
//	-output  Output directory (default: testdata/images/smartcrop/output)
//	-aspects Comma-separated aspect ratios to test (default: 1:1,16:9,9:16,4:3)
//	-debug   Generate debug visualizations (Sobel maps, face boxes)
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/sambeau/basil/server/images/smartcrop"
)

// AspectRatio represents a width:height ratio for cropping.
type AspectRatio struct {
	Name   string
	Width  int
	Height int
}

func main() {
	inputDir := flag.String("input", "testdata/images/smartcrop", "Input directory containing test images")
	outputDir := flag.String("output", "testdata/images/smartcrop/output", "Output directory for comparison grids")
	aspectsStr := flag.String("aspects", "1:1,16:9,9:16,4:3", "Comma-separated aspect ratios to test")
	debug := flag.Bool("debug", false, "Generate debug visualizations")
	flag.Parse()

	aspects := parseAspects(*aspectsStr)
	if len(aspects) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no valid aspect ratios provided")
		os.Exit(1)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Find all images in input directory (recursively)
	images, err := findImages(*inputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding images: %v\n", err)
		os.Exit(1)
	}

	if len(images) == 0 {
		fmt.Println("No images found in", *inputDir)
		fmt.Println("Add test images to the subdirectories (faces/, landscapes/, objects/, edge-cases/)")
		os.Exit(0)
	}

	fmt.Printf("Found %d images, generating comparisons for %d aspect ratios...\n", len(images), len(aspects))

	// Process each image
	for _, imgPath := range images {
		relPath, _ := filepath.Rel(*inputDir, imgPath)
		fmt.Printf("  Processing: %s\n", relPath)

		img, err := loadImage(imgPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "    Error loading: %v\n", err)
			continue
		}

		baseName := strings.TrimSuffix(filepath.Base(imgPath), filepath.Ext(imgPath))
		category := filepath.Base(filepath.Dir(imgPath))

		// Create category subdirectory in output
		categoryOut := filepath.Join(*outputDir, category)
		if err := os.MkdirAll(categoryOut, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "    Error creating category dir: %v\n", err)
			continue
		}

		// Generate comparison for each aspect ratio
		for _, aspect := range aspects {
			outPath := filepath.Join(categoryOut, fmt.Sprintf("%s_%s.jpg", baseName, aspect.Name))
			if err := generateComparison(img, aspect, outPath); err != nil {
				fmt.Fprintf(os.Stderr, "    Error generating %s: %v\n", aspect.Name, err)
				continue
			}
		}

		// Generate debug visualizations if requested
		if *debug {
			debugDir := filepath.Join(categoryOut, "debug")
			if err := os.MkdirAll(debugDir, 0755); err == nil {
				generateDebug(img, baseName, debugDir)
			}
		}
	}

	fmt.Println("Done! Comparisons written to:", *outputDir)
	fmt.Println("\nReview the images and note any failures for weight tuning.")
}

// parseAspects parses a comma-separated list of aspect ratios like "1:1,16:9,4:3".
func parseAspects(s string) []AspectRatio {
	var aspects []AspectRatio
	for part := range strings.SplitSeq(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		parts := strings.Split(part, ":")
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Warning: invalid aspect ratio %q, skipping\n", part)
			continue
		}

		w, err1 := strconv.Atoi(parts[0])
		h, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
			fmt.Fprintf(os.Stderr, "Warning: invalid aspect ratio %q, skipping\n", part)
			continue
		}

		aspects = append(aspects, AspectRatio{
			Name:   fmt.Sprintf("%dx%d", w, h),
			Width:  w,
			Height: h,
		})
	}
	return aspects
}

// findImages recursively finds all JPEG/PNG images in a directory.
func findImages(dir string) ([]string, error) {
	var images []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip output directory
			if info.Name() == "output" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
			images = append(images, path)
		}
		return nil
	})
	return images, err
}

// loadImage loads an image from disk with automatic EXIF orientation.
func loadImage(path string) (image.Image, error) {
	return imaging.Open(path, imaging.AutoOrientation(true))
}

// generateComparison creates a side-by-side comparison of center crop vs smart crop.
func generateComparison(img image.Image, aspect AspectRatio, outPath string) error {
	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Calculate crop dimensions that fit within the source image
	targetAspect := float64(aspect.Width) / float64(aspect.Height)
	var cropW, cropH int

	imgAspect := float64(srcWidth) / float64(srcHeight)
	if targetAspect > imgAspect {
		cropW = srcWidth
		cropH = int(float64(cropW) / targetAspect)
	} else {
		cropH = srcHeight
		cropW = int(float64(cropH) * targetAspect)
	}

	// Get center crop rectangle
	centerX := (srcWidth - cropW) / 2
	centerY := (srcHeight - cropH) / 2
	centerRect := image.Rect(centerX, centerY, centerX+cropW, centerY+cropH)

	// Get smart crop rectangle
	smartRect := smartcrop.BestCrop(img, cropW, cropH, nil)

	// Extract crops
	centerCrop := imaging.Crop(img, centerRect)
	smartCrop := imaging.Crop(img, smartRect)

	// Create comparison grid
	// Layout: [Original with boxes] [Center crop] [Smart crop]
	// Each panel is 400px wide max

	panelWidth := 400
	panelHeight := int(float64(panelWidth) / targetAspect)

	// Resize crops to panel size
	centerResized := imaging.Resize(centerCrop, panelWidth, panelHeight, imaging.Lanczos)
	smartResized := imaging.Resize(smartCrop, panelWidth, panelHeight, imaging.Lanczos)

	// Create thumbnail of original with crop rectangles drawn
	thumbScale := float64(panelWidth) / float64(srcWidth)
	thumbHeight := int(float64(srcHeight) * thumbScale)
	originalThumb := imaging.Resize(img, panelWidth, thumbHeight, imaging.Lanczos)
	originalWithBoxes := drawCropBoxes(originalThumb, centerRect, smartRect, thumbScale)

	// Assemble the comparison grid
	// Use the max height for the canvas
	maxHeight := max(thumbHeight, panelHeight)

	gridWidth := panelWidth*3 + 40 // 3 panels + padding
	gridHeight := maxHeight + 60   // Extra space for labels

	grid := image.NewRGBA(image.Rect(0, 0, gridWidth, gridHeight))

	// Fill with white background
	draw.Draw(grid, grid.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Draw original thumbnail (left)
	drawImage(grid, originalWithBoxes, 10, 30)

	// Draw center crop (middle)
	drawImage(grid, centerResized, panelWidth+20, 30)

	// Draw smart crop (right)
	drawImage(grid, smartResized, panelWidth*2+30, 30)

	// Add labels (simple colored bars since we can't easily draw text)
	drawLabel(grid, 10, 10, panelWidth, color.RGBA{100, 100, 100, 255})              // "Original"
	drawLabel(grid, panelWidth+20, 10, panelWidth, color.RGBA{200, 100, 100, 255})   // "Center" (red)
	drawLabel(grid, panelWidth*2+30, 10, panelWidth, color.RGBA{100, 200, 100, 255}) // "Smart" (green)

	// Save the grid
	return saveJPEG(grid, outPath, 90)
}

// drawCropBoxes draws center (red) and smart (green) crop rectangles on the image.
func drawCropBoxes(img image.Image, centerRect, smartRect image.Rectangle, scale float64) image.Image {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, image.Point{}, draw.Src)

	// Scale rectangles
	scaledCenter := scaleRect(centerRect, scale)
	scaledSmart := scaleRect(smartRect, scale)

	// Draw rectangles
	drawRect(result, scaledCenter, color.RGBA{255, 100, 100, 255}, 2) // Red for center
	drawRect(result, scaledSmart, color.RGBA{100, 255, 100, 255}, 2)  // Green for smart

	return result
}

// scaleRect scales a rectangle by a factor.
func scaleRect(r image.Rectangle, scale float64) image.Rectangle {
	return image.Rect(
		int(float64(r.Min.X)*scale),
		int(float64(r.Min.Y)*scale),
		int(float64(r.Max.X)*scale),
		int(float64(r.Max.Y)*scale),
	)
}

// drawRect draws a rectangle outline on an image.
func drawRect(img *image.RGBA, r image.Rectangle, c color.Color, thickness int) {
	bounds := img.Bounds()

	// Clamp rectangle to image bounds
	r = r.Intersect(bounds)
	if r.Empty() {
		return
	}

	// Top edge
	for y := r.Min.Y; y < r.Min.Y+thickness && y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.Set(x, y, c)
		}
	}

	// Bottom edge
	for y := r.Max.Y - thickness; y < r.Max.Y; y++ {
		if y >= r.Min.Y {
			for x := r.Min.X; x < r.Max.X; x++ {
				img.Set(x, y, c)
			}
		}
	}

	// Left edge
	for x := r.Min.X; x < r.Min.X+thickness && x < r.Max.X; x++ {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			img.Set(x, y, c)
		}
	}

	// Right edge
	for x := r.Max.X - thickness; x < r.Max.X; x++ {
		if x >= r.Min.X {
			for y := r.Min.Y; y < r.Max.Y; y++ {
				img.Set(x, y, c)
			}
		}
	}
}

// drawImage draws an image onto a canvas at the specified position.
func drawImage(canvas *image.RGBA, img image.Image, x, y int) {
	r := image.Rect(x, y, x+img.Bounds().Dx(), y+img.Bounds().Dy())
	draw.Draw(canvas, r, img, image.Point{}, draw.Src)
}

// drawLabel draws a colored label bar.
func drawLabel(canvas *image.RGBA, x, y, width int, c color.Color) {
	height := 15
	for dy := range height {
		for dx := range width {
			canvas.Set(x+dx, y+dy, c)
		}
	}
}

// generateDebug creates debug visualizations (Sobel map, face detection boxes).
func generateDebug(img image.Image, baseName, debugDir string) {
	// Generate Sobel edge map
	sobelPath := filepath.Join(debugDir, baseName+"_sobel.jpg")
	generateSobelVis(img, sobelPath)

	// Generate face detection visualization
	facesPath := filepath.Join(debugDir, baseName+"_faces.jpg")
	generateFaceVis(img, facesPath)
}

// generateSobelVis creates a visualization of the Sobel edge map.
func generateSobelVis(img image.Image, outPath string) {
	// Resize for analysis (same as BestCrop does)
	analysisImg := imaging.Resize(img, 400, 0, imaging.Linear)

	// Compute Sobel
	sobelMap := smartcrop.Sobel(analysisImg)

	// Convert to grayscale image
	bounds := analysisImg.Bounds()
	result := image.NewGray(bounds)

	// Find max value for normalization
	var maxVal float64
	for y := range sobelMap {
		for _, v := range sobelMap[y] {
			if v > maxVal {
				maxVal = v
			}
		}
	}

	if maxVal > 0 {
		for y := range sobelMap {
			for x := 0; x < len(sobelMap[y]); x++ {
				normalized := uint8(sobelMap[y][x] / maxVal * 255)
				result.SetGray(x, y, color.Gray{Y: normalized})
			}
		}
	}

	saveJPEG(result, outPath, 90)
}

// generateFaceVis creates a visualization with detected faces highlighted.
func generateFaceVis(img image.Image, outPath string) {
	// Resize for face detection
	faceImg := imaging.Resize(img, 640, 0, imaging.Linear)

	// Detect faces
	pixels, width, height := smartcrop.ToGrayscaleFlat(faceImg)
	detections := smartcrop.DetectFaces(pixels, width, height)

	// Draw face boxes on the image
	result := image.NewRGBA(faceImg.Bounds())
	draw.Draw(result, result.Bounds(), faceImg, image.Point{}, draw.Src)

	for _, det := range detections {
		rect := det.Rect()
		drawRect(result, rect, color.RGBA{0, 255, 0, 255}, 3)
	}

	saveJPEG(result, outPath, 90)
}

// saveJPEG saves an image as JPEG with the specified quality.
func saveJPEG(img image.Image, path string, quality int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return jpeg.Encode(f, img, &jpeg.Options{Quality: quality})
}
