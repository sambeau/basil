// Package smartcrop provides content-aware image cropping with face detection.
package smartcrop

import (
	"image"
	"image/color"
	"math"
)

// Sobel computes the gradient magnitude at each pixel using a Sobel filter.
// The returned 2D slice has the same dimensions as the input image.
// Higher values indicate edges/detail; uniform regions approach zero.
func Sobel(img image.Image) [][]float64 {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

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

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
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
		for x := 0; x < width; x++ {
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

// SobelSum computes the sum of gradient magnitudes within a rectangle.
// This is useful for scoring regions by edge density.
func SobelSum(sobelMap [][]float64, rect image.Rectangle) float64 {
	var sum float64
	height := len(sobelMap)
	if height == 0 {
		return 0
	}
	width := len(sobelMap[0])

	// Clamp rectangle to map bounds
	minY := clamp(rect.Min.Y, 0, height-1)
	maxY := clamp(rect.Max.Y, 0, height)
	minX := clamp(rect.Min.X, 0, width-1)
	maxX := clamp(rect.Max.X, 0, width)

	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			sum += sobelMap[y][x]
		}
	}
	return sum
}

// SobelMean computes the mean gradient magnitude within a rectangle.
func SobelMean(sobelMap [][]float64, rect image.Rectangle) float64 {
	area := rect.Dx() * rect.Dy()
	if area == 0 {
		return 0
	}
	return SobelSum(sobelMap, rect) / float64(area)
}

// ToGrayscaleFlat converts an image to a flat grayscale byte buffer.
// Returns the buffer, width, and height.
// This is used by the PICO face detector.
func ToGrayscaleFlat(img image.Image) ([]uint8, int, int) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	minX := bounds.Min.X
	minY := bounds.Min.Y

	pixels := make([]uint8, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := img.At(minX+x, minY+y)
			gray := color.GrayModel.Convert(c).(color.Gray)
			pixels[y*width+x] = gray.Y
		}
	}

	return pixels, width, height
}
