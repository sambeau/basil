// Package seamcarve provides content-aware image resizing using seam carving.
//
// Seam carving removes low-energy "seams" (connected paths of pixels) from an image
// to reduce its dimensions while preserving important visual content. This is useful
// for resizing images to arbitrary aspect ratios without cropping or distortion.
//
// The algorithm:
// 1. Compute an energy map (gradient magnitude via Sobel filter)
// 2. Use dynamic programming to find the minimum-energy seam
// 3. Remove the seam (shift pixels to fill the gap)
// 4. Repeat until target dimensions are reached
//
// Reference: Avidan & Shamir, "Seam Carving for Content-Aware Image Resizing", SIGGRAPH 2007
package seamcarve

import (
	"image"
	"image/color"
	"log"
	"math"
)

// MaxReductionRatio is the threshold beyond which a warning is logged.
// Reducing an image by more than 30% in either dimension may produce artifacts.
const MaxReductionRatio = 0.30

// Resize performs content-aware resizing of an image to the target dimensions.
// It removes vertical and/or horizontal seams to reduce the image size.
//
// If targetWidth >= img.Width and targetHeight >= img.Height, the image is returned unchanged.
// Seam carving only supports reduction, not enlargement.
//
// If both dimensions need reduction, the dimension with the larger delta is reduced first.
// A warning is logged if reduction exceeds 30% in either dimension.
func Resize(img image.Image, targetWidth, targetHeight int) image.Image {
	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// No reduction needed
	if targetWidth >= srcWidth && targetHeight >= srcHeight {
		return img
	}

	// Clamp targets to source dimensions (no enlargement)
	if targetWidth > srcWidth {
		targetWidth = srcWidth
	}
	if targetHeight > srcHeight {
		targetHeight = srcHeight
	}
	if targetWidth < 1 {
		targetWidth = 1
	}
	if targetHeight < 1 {
		targetHeight = 1
	}

	// Calculate reductions needed
	widthDelta := srcWidth - targetWidth
	heightDelta := srcHeight - targetHeight

	// Warn if reduction is aggressive
	if float64(widthDelta)/float64(srcWidth) > MaxReductionRatio {
		log.Printf("seamcarve: width reduction of %d%% exceeds recommended 30%%, artifacts may occur",
			int(100*float64(widthDelta)/float64(srcWidth)))
	}
	if float64(heightDelta)/float64(srcHeight) > MaxReductionRatio {
		log.Printf("seamcarve: height reduction of %d%% exceeds recommended 30%%, artifacts may occur",
			int(100*float64(heightDelta)/float64(srcHeight)))
	}

	// Convert to RGBA for efficient pixel manipulation
	result := toRGBA(img)

	// Reduce the dimension with larger delta first (heuristic: better results)
	if widthDelta >= heightDelta {
		// Reduce width first, then height
		result = reduceWidth(result, widthDelta)
		result = reduceHeight(result, heightDelta)
	} else {
		// Reduce height first, then width
		result = reduceHeight(result, heightDelta)
		result = reduceWidth(result, widthDelta)
	}

	return result
}

// reduceWidth removes n vertical seams from the image.
func reduceWidth(img *image.RGBA, n int) *image.RGBA {
	for i := 0; i < n; i++ {
		energy := EnergyMap(img)
		seam := findMinVerticalSeam(energy)
		img = removeVerticalSeam(img, seam)
	}
	return img
}

// reduceHeight removes n horizontal seams from the image.
func reduceHeight(img *image.RGBA, n int) *image.RGBA {
	for i := 0; i < n; i++ {
		energy := EnergyMap(img)
		seam := findMinHorizontalSeam(energy)
		img = removeHorizontalSeam(img, seam)
	}
	return img
}

// findMinVerticalSeam finds the minimum-energy vertical seam using dynamic programming.
// Returns a slice of column indices, one per row (from top to bottom).
//
// Algorithm:
// 1. Build cumulative energy matrix M where M[y][x] = energy[y][x] + min(M[y-1][x-1], M[y-1][x], M[y-1][x+1])
// 2. Find minimum value in bottom row
// 3. Backtrack to construct the seam path
func findMinVerticalSeam(energy [][]float64) []int {
	height := len(energy)
	if height == 0 {
		return nil
	}
	width := len(energy[0])
	if width == 0 {
		return nil
	}

	// Build cumulative energy matrix
	M := make([][]float64, height)
	for y := range M {
		M[y] = make([]float64, width)
	}

	// First row: copy energy values
	copy(M[0], energy[0])

	// Fill remaining rows using DP recurrence
	for y := 1; y < height; y++ {
		for x := 0; x < width; x++ {
			minPrev := M[y-1][x]

			// Check left neighbor
			if x > 0 && M[y-1][x-1] < minPrev {
				minPrev = M[y-1][x-1]
			}

			// Check right neighbor
			if x < width-1 && M[y-1][x+1] < minPrev {
				minPrev = M[y-1][x+1]
			}

			M[y][x] = energy[y][x] + minPrev
		}
	}

	// Find minimum in bottom row
	minX := 0
	minVal := M[height-1][0]
	for x := 1; x < width; x++ {
		if M[height-1][x] < minVal {
			minVal = M[height-1][x]
			minX = x
		}
	}

	// Backtrack to find seam path
	seam := make([]int, height)
	seam[height-1] = minX

	for y := height - 2; y >= 0; y-- {
		x := seam[y+1]
		bestX := x
		bestVal := M[y][x]

		// Check left
		if x > 0 && M[y][x-1] < bestVal {
			bestVal = M[y][x-1]
			bestX = x - 1
		}

		// Check right
		if x < width-1 && M[y][x+1] < bestVal {
			bestX = x + 1
		}

		seam[y] = bestX
	}

	return seam
}

// findMinHorizontalSeam finds the minimum-energy horizontal seam using dynamic programming.
// Returns a slice of row indices, one per column (from left to right).
//
// This is the transposed version of findMinVerticalSeam.
func findMinHorizontalSeam(energy [][]float64) []int {
	height := len(energy)
	if height == 0 {
		return nil
	}
	width := len(energy[0])
	if width == 0 {
		return nil
	}

	// Build cumulative energy matrix (scanning left to right)
	M := make([][]float64, height)
	for y := range M {
		M[y] = make([]float64, width)
	}

	// First column: copy energy values
	for y := 0; y < height; y++ {
		M[y][0] = energy[y][0]
	}

	// Fill remaining columns using DP recurrence
	for x := 1; x < width; x++ {
		for y := 0; y < height; y++ {
			minPrev := M[y][x-1]

			// Check top neighbor
			if y > 0 && M[y-1][x-1] < minPrev {
				minPrev = M[y-1][x-1]
			}

			// Check bottom neighbor
			if y < height-1 && M[y+1][x-1] < minPrev {
				minPrev = M[y+1][x-1]
			}

			M[y][x] = energy[y][x] + minPrev
		}
	}

	// Find minimum in rightmost column
	minY := 0
	minVal := M[0][width-1]
	for y := 1; y < height; y++ {
		if M[y][width-1] < minVal {
			minVal = M[y][width-1]
			minY = y
		}
	}

	// Backtrack to find seam path
	seam := make([]int, width)
	seam[width-1] = minY

	for x := width - 2; x >= 0; x-- {
		y := seam[x+1]
		bestY := y
		bestVal := M[y][x]

		// Check top
		if y > 0 && M[y-1][x] < bestVal {
			bestVal = M[y-1][x]
			bestY = y - 1
		}

		// Check bottom
		if y < height-1 && M[y+1][x] < bestVal {
			bestY = y + 1
		}

		seam[x] = bestY
	}

	return seam
}

// removeVerticalSeam removes a vertical seam from an image.
// The seam slice contains one column index per row.
// Returns a new image with width reduced by 1.
func removeVerticalSeam(img *image.RGBA, seam []int) *image.RGBA {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if len(seam) != height || width <= 1 {
		return img
	}

	// Create new image with width - 1
	newImg := image.NewRGBA(image.Rect(0, 0, width-1, height))

	for y := 0; y < height; y++ {
		seamX := seam[y]
		dstX := 0

		for x := 0; x < width; x++ {
			if x == seamX {
				continue // Skip the seam pixel
			}

			c := img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			newImg.SetRGBA(dstX, y, c)
			dstX++
		}
	}

	return newImg
}

// removeHorizontalSeam removes a horizontal seam from an image.
// The seam slice contains one row index per column.
// Returns a new image with height reduced by 1.
func removeHorizontalSeam(img *image.RGBA, seam []int) *image.RGBA {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if len(seam) != width || height <= 1 {
		return img
	}

	// Create new image with height - 1
	newImg := image.NewRGBA(image.Rect(0, 0, width, height-1))

	for x := 0; x < width; x++ {
		seamY := seam[x]
		dstY := 0

		for y := 0; y < height; y++ {
			if y == seamY {
				continue // Skip the seam pixel
			}

			c := img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			newImg.SetRGBA(x, dstY, c)
			dstY++
		}
	}

	return newImg
}

// toRGBA converts an image to *image.RGBA for efficient pixel manipulation.
func toRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		// Already RGBA, but may have non-zero Min - normalize to origin
		if rgba.Bounds().Min.X == 0 && rgba.Bounds().Min.Y == 0 {
			return rgba
		}
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	rgba := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := img.At(bounds.Min.X+x, bounds.Min.Y+y)
			r, g, b, a := c.RGBA()
			rgba.SetRGBA(x, y, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			})
		}
	}

	return rgba
}

// min3 returns the minimum of three float64 values.
func min3(a, b, c float64) float64 {
	return math.Min(a, math.Min(b, c))
}
