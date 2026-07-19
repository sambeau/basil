// Package seamcarve provides content-aware image resizing using seam carving.
package seamcarve

import (
	"image"
	"math"
)

// EnergyMap computes the gradient magnitude at each pixel using a Sobel filter.
// The returned 2D slice has the same dimensions as the input image.
// Higher values indicate edges/detail; uniform regions approach zero.
// This is used to identify low-energy seams that can be removed with minimal visual impact.
func EnergyMap(img image.Image) [][]float64 {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width == 0 || height == 0 {
		return nil
	}

	// Convert to grayscale luminance values
	gray := toGrayscaleMatrix(img)

	// Allocate output
	result := make([][]float64, height)
	for y := range result {
		result[y] = make([]float64, width)
	}

	// Sobel kernels
	// Gx: horizontal gradient (detects vertical edges)
	// [-1  0  1]
	// [-2  0  2]
	// [-1  0  1]
	//
	// Gy: vertical gradient (detects horizontal edges)
	// [-1 -2 -1]
	// [ 0  0  0]
	// [ 1  2  1]

	for y := range height {
		for x := range width {
			// Get 3x3 neighborhood (clamping at borders)
			p00 := gray[clamp(y-1, 0, height-1)][clamp(x-1, 0, width-1)]
			p01 := gray[clamp(y-1, 0, height-1)][x]
			p02 := gray[clamp(y-1, 0, height-1)][clamp(x+1, 0, width-1)]
			p10 := gray[y][clamp(x-1, 0, width-1)]
			p12 := gray[y][clamp(x+1, 0, width-1)]
			p20 := gray[clamp(y+1, 0, height-1)][clamp(x-1, 0, width-1)]
			p21 := gray[clamp(y+1, 0, height-1)][x]
			p22 := gray[clamp(y+1, 0, height-1)][clamp(x+1, 0, width-1)]

			// Compute gradients
			gx := -p00 + p02 - 2*p10 + 2*p12 - p20 + p22
			gy := -p00 - 2*p01 - p02 + p20 + 2*p21 + p22

			// Gradient magnitude
			result[y][x] = math.Sqrt(gx*gx + gy*gy)
		}
	}

	return result
}

// toGrayscaleMatrix converts an image to a 2D matrix of grayscale luminance values.
// Values are in the range [0, 255].
func toGrayscaleMatrix(img image.Image) [][]float64 {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	minX := bounds.Min.X
	minY := bounds.Min.Y

	gray := make([][]float64, height)
	for y := range gray {
		gray[y] = make([]float64, width)
		for x := range width {
			c := img.At(minX+x, minY+y)
			// Convert to grayscale using luminance formula
			// Y = 0.299*R + 0.587*G + 0.114*B
			r, g, b, _ := c.RGBA()
			// RGBA() returns 16-bit values, scale to 0-255
			lum := 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
			gray[y][x] = lum
		}
	}

	return gray
}

// clamp restricts a value to the range [min, max].
func clamp(val, minVal, maxVal int) int {
	if val < minVal {
		return minVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}
