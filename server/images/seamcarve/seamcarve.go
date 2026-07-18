// Package seamcarve provides content-aware image resizing using seam carving.
//
// Seam carving removes or inserts low-energy "seams" (connected paths of pixels) from an image
// to resize it while preserving important visual content. This is useful for resizing images
// to arbitrary aspect ratios without cropping or distortion.
//
// For reduction (shrinking):
// 1. Compute an energy map (gradient magnitude via Sobel filter)
// 2. Use dynamic programming to find the minimum-energy seam
// 3. Remove the seam (shift pixels to fill the gap)
// 4. Repeat until target dimensions are reached
//
// For enlargement (growing):
// 1. Compute an energy map
// 2. Find k minimum-energy seams on the original image (to avoid clustering)
// 3. Insert seams by duplicating pixels along each seam path
//
// Reference: Avidan & Shamir, "Seam Carving for Content-Aware Image Resizing", SIGGRAPH 2007
package seamcarve

import (
	"image"
	"image/color"
	"log"
	"math"
	"slices"
)

// MaxChangeRatio is the threshold beyond which a warning is logged.
// Changing an image by more than 30% in either dimension may produce artifacts.
const MaxChangeRatio = 0.30

// Resize performs content-aware resizing of an image to the target dimensions.
// It removes seams to reduce size or inserts seams to enlarge.
//
// For reduction: seams are removed iteratively, recomputing energy each time.
// For enlargement: k seams are found on the original image (to avoid clustering),
// then all k seams are inserted at once.
//
// If both reduction and enlargement are needed (different dimensions), reduction
// is performed first, then enlargement.
//
// A warning is logged if the change exceeds 30% in either dimension.
func Resize(img image.Image, targetWidth, targetHeight int) image.Image {
	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Clamp to minimum size
	if targetWidth < 1 {
		targetWidth = 1
	}
	if targetHeight < 1 {
		targetHeight = 1
	}

	// No change needed
	if targetWidth == srcWidth && targetHeight == srcHeight {
		return img
	}

	// Calculate changes needed (positive = reduce, negative = enlarge)
	widthReduce := srcWidth - targetWidth
	heightReduce := srcHeight - targetHeight

	// Warn if change is aggressive
	if widthReduce > 0 && float64(widthReduce)/float64(srcWidth) > MaxChangeRatio {
		log.Printf("seamcarve: width reduction of %d%% exceeds recommended 30%%, artifacts may occur",
			int(100*float64(widthReduce)/float64(srcWidth)))
	}
	if widthReduce < 0 && float64(-widthReduce)/float64(srcWidth) > MaxChangeRatio {
		log.Printf("seamcarve: width enlargement of %d%% exceeds recommended 30%%, artifacts may occur",
			int(100*float64(-widthReduce)/float64(srcWidth)))
	}
	if heightReduce > 0 && float64(heightReduce)/float64(srcHeight) > MaxChangeRatio {
		log.Printf("seamcarve: height reduction of %d%% exceeds recommended 30%%, artifacts may occur",
			int(100*float64(heightReduce)/float64(srcHeight)))
	}
	if heightReduce < 0 && float64(-heightReduce)/float64(srcHeight) > MaxChangeRatio {
		log.Printf("seamcarve: height enlargement of %d%% exceeds recommended 30%%, artifacts may occur",
			int(100*float64(-heightReduce)/float64(srcHeight)))
	}

	// Convert to RGBA for efficient pixel manipulation
	result := toRGBA(img)

	// Phase 1: Reduction (if needed)
	// Reduce the dimension with larger delta first (heuristic: better results)
	if widthReduce > 0 || heightReduce > 0 {
		if widthReduce >= heightReduce {
			if widthReduce > 0 {
				result = reduceWidth(result, widthReduce)
			}
			if heightReduce > 0 {
				result = reduceHeight(result, heightReduce)
			}
		} else {
			if heightReduce > 0 {
				result = reduceHeight(result, heightReduce)
			}
			if widthReduce > 0 {
				result = reduceWidth(result, widthReduce)
			}
		}
	}

	// Phase 2: Enlargement (if needed)
	// Recalculate deltas based on current size after reduction
	currentWidth := result.Bounds().Dx()
	currentHeight := result.Bounds().Dy()
	widthEnlarge := targetWidth - currentWidth
	heightEnlarge := targetHeight - currentHeight

	if widthEnlarge > 0 {
		result = enlargeWidth(result, widthEnlarge)
	}
	if heightEnlarge > 0 {
		result = enlargeHeight(result, heightEnlarge)
	}

	return result
}

// reduceWidth removes n vertical seams from the image.
func reduceWidth(img *image.RGBA, n int) *image.RGBA {
	for range n {
		energy := EnergyMap(img)
		seam := findMinVerticalSeam(energy)
		img = removeVerticalSeam(img, seam)
	}
	return img
}

// reduceHeight removes n horizontal seams from the image.
func reduceHeight(img *image.RGBA, n int) *image.RGBA {
	for range n {
		energy := EnergyMap(img)
		seam := findMinHorizontalSeam(energy)
		img = removeHorizontalSeam(img, seam)
	}
	return img
}

// enlargeWidth inserts n vertical seams into the image.
// Seams are found on the original image to avoid clustering.
func enlargeWidth(img *image.RGBA, n int) *image.RGBA {
	if n <= 0 {
		return img
	}
	seams := findKMinVerticalSeams(img, n)
	return insertVerticalSeams(img, seams)
}

// enlargeHeight inserts n horizontal seams into the image.
// Seams are found on the original image to avoid clustering.
func enlargeHeight(img *image.RGBA, n int) *image.RGBA {
	if n <= 0 {
		return img
	}
	seams := findKMinHorizontalSeams(img, n)
	return insertHorizontalSeams(img, seams)
}

// findKMinVerticalSeams finds k minimum-energy vertical seams on the image.
// Unlike iterative removal, this finds all seams on the original image to avoid clustering.
// Each seam is marked with high energy after being found to prevent reselection.
func findKMinVerticalSeams(img image.Image, k int) [][]int {
	energy := EnergyMap(img)
	width := len(energy[0])

	seams := make([][]int, k)

	for i := range k {
		seam := findMinVerticalSeam(energy)
		seams[i] = seam

		// Mark this seam with very high energy to prevent reselection
		for y, x := range seam {
			if x >= 0 && x < width {
				energy[y][x] = math.MaxFloat64 / 2 // High but not overflow
			}
		}
	}

	return seams
}

// findKMinHorizontalSeams finds k minimum-energy horizontal seams on the image.
func findKMinHorizontalSeams(img image.Image, k int) [][]int {
	energy := EnergyMap(img)
	height := len(energy)

	seams := make([][]int, k)

	for i := range k {
		seam := findMinHorizontalSeam(energy)
		seams[i] = seam

		// Mark this seam with very high energy to prevent reselection
		for x, y := range seam {
			if y >= 0 && y < height {
				energy[y][x] = math.MaxFloat64 / 2
			}
		}
	}

	return seams
}

// insertVerticalSeams inserts multiple vertical seams into the image.
// Seams are processed from right to left to preserve index validity.
func insertVerticalSeams(img *image.RGBA, seams [][]int) *image.RGBA {
	if len(seams) == 0 {
		return img
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	numSeams := len(seams)

	// Create a map of which seams pass through each pixel
	// seamIndex[y][x] = list of seam indices that pass through or to the left
	// We need to track how many seams are at or before each x position per row

	// For each row, collect all seam x-positions and sort them
	seamPositions := make([][]int, height)
	for y := range height {
		seamPositions[y] = make([]int, numSeams)
		for i, seam := range seams {
			seamPositions[y][i] = seam[y]
		}
		// Sort positions for this row
		sortInts(seamPositions[y])
	}

	// Create new image with width + numSeams
	newWidth := width + numSeams
	newImg := image.NewRGBA(image.Rect(0, 0, newWidth, height))

	for y := range height {
		positions := seamPositions[y]
		seamIdx := 0
		dstX := 0

		for srcX := range width {
			// Copy original pixel
			c := img.RGBAAt(srcX, y)
			newImg.SetRGBA(dstX, y, c)
			dstX++

			// Check if any seams should be inserted after this x position
			for seamIdx < numSeams && positions[seamIdx] == srcX {
				// Insert a new pixel (average of neighbors)
				newC := averageColor(img, srcX, y, width, height)
				newImg.SetRGBA(dstX, y, newC)
				dstX++
				seamIdx++
			}
		}
	}

	return newImg
}

// insertHorizontalSeams inserts multiple horizontal seams into the image.
func insertHorizontalSeams(img *image.RGBA, seams [][]int) *image.RGBA {
	if len(seams) == 0 {
		return img
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	numSeams := len(seams)

	// For each column, collect all seam y-positions and sort them
	seamPositions := make([][]int, width)
	for x := range width {
		seamPositions[x] = make([]int, numSeams)
		for i, seam := range seams {
			seamPositions[x][i] = seam[x]
		}
		sortInts(seamPositions[x])
	}

	// Create new image with height + numSeams
	newHeight := height + numSeams
	newImg := image.NewRGBA(image.Rect(0, 0, width, newHeight))

	for x := range width {
		positions := seamPositions[x]
		seamIdx := 0
		dstY := 0

		for srcY := range height {
			// Copy original pixel
			c := img.RGBAAt(x, srcY)
			newImg.SetRGBA(x, dstY, c)
			dstY++

			// Check if any seams should be inserted after this y position
			for seamIdx < numSeams && positions[seamIdx] == srcY {
				// Insert a new pixel (average of neighbors)
				newC := averageColorVertical(img, x, srcY, width, height)
				newImg.SetRGBA(x, dstY, newC)
				dstY++
				seamIdx++
			}
		}
	}

	return newImg
}

// averageColor computes the average color of a pixel and its horizontal neighbors.
func averageColor(img *image.RGBA, x, y, width, height int) color.RGBA {
	c := img.RGBAAt(x, y)

	// Get right neighbor if available
	if x+1 < width {
		right := img.RGBAAt(x+1, y)
		return color.RGBA{
			R: uint8((uint16(c.R) + uint16(right.R)) / 2),
			G: uint8((uint16(c.G) + uint16(right.G)) / 2),
			B: uint8((uint16(c.B) + uint16(right.B)) / 2),
			A: uint8((uint16(c.A) + uint16(right.A)) / 2),
		}
	}

	// No right neighbor, just return the pixel
	return c
}

// averageColorVertical computes the average color of a pixel and its vertical neighbors.
func averageColorVertical(img *image.RGBA, x, y, width, height int) color.RGBA {
	c := img.RGBAAt(x, y)

	// Get bottom neighbor if available
	if y+1 < height {
		bottom := img.RGBAAt(x, y+1)
		return color.RGBA{
			R: uint8((uint16(c.R) + uint16(bottom.R)) / 2),
			G: uint8((uint16(c.G) + uint16(bottom.G)) / 2),
			B: uint8((uint16(c.B) + uint16(bottom.B)) / 2),
			A: uint8((uint16(c.A) + uint16(bottom.A)) / 2),
		}
	}

	// No bottom neighbor, just return the pixel
	return c
}

func sortInts(a []int) {
	slices.Sort(a)
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
		for x := range width {
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
	for y := range height {
		M[y][0] = energy[y][0]
	}

	// Fill remaining columns using DP recurrence
	for x := 1; x < width; x++ {
		for y := range height {
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

	for y := range height {
		seamX := seam[y]
		dstX := 0

		for x := range width {
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

	for x := range width {
		seamY := seam[x]
		dstY := 0

		for y := range height {
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

	for y := range height {
		for x := range width {
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
