package smartcrop

import (
	"image"
	"image/color"
	"math"
	"testing"
)

// uniformImage creates an image with a single color.
func uniformImage(width, height int, c color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

// verticalEdgeImage creates an image with a vertical edge (black on left, white on right).
func verticalEdgeImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	midX := width / 2
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if x < midX {
				img.Set(x, y, color.Black)
			} else {
				img.Set(x, y, color.White)
			}
		}
	}
	return img
}

// horizontalEdgeImage creates an image with a horizontal edge (black on top, white on bottom).
func horizontalEdgeImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	midY := height / 2
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if y < midY {
				img.Set(x, y, color.Black)
			} else {
				img.Set(x, y, color.White)
			}
		}
	}
	return img
}

// diagonalGradientImage creates an image with a diagonal gradient.
func diagonalGradientImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Intensity increases diagonally
			intensity := uint8((x + y) * 255 / (width + height - 2))
			img.Set(x, y, color.Gray{Y: intensity})
		}
	}
	return img
}

func TestSobel_UniformImage(t *testing.T) {
	// A uniform image should have zero gradients everywhere
	img := uniformImage(10, 10, color.Gray{Y: 128})
	result := Sobel(img)

	if len(result) != 10 || len(result[0]) != 10 {
		t.Errorf("expected 10x10 result, got %dx%d", len(result), len(result[0]))
	}

	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			// Use tolerance for floating point comparison
			if result[y][x] > 1e-10 {
				t.Fatalf("expected ~0 at (%d,%d), got %.20e", x, y, result[y][x])
			}
		}
	}
}

func TestSobel_VerticalEdge(t *testing.T) {
	// A vertical edge should have high gradients along the edge, low elsewhere
	img := verticalEdgeImage(20, 20)
	result := Sobel(img)

	if len(result) != 20 || len(result[0]) != 20 {
		t.Errorf("expected 20x20 result, got %dx%d", len(result), len(result[0]))
	}

	// Check that edge region (around x=10) has high values
	edgeSum := 0.0
	for y := 0; y < 20; y++ {
		// Edge is at x=9,10 (the transition from black to white)
		edgeSum += result[y][9] + result[y][10]
	}

	// Check that non-edge regions have low values
	nonEdgeSum := 0.0
	for y := 0; y < 20; y++ {
		nonEdgeSum += result[y][0] + result[y][19]
	}

	if edgeSum <= nonEdgeSum {
		t.Errorf("edge sum (%f) should be greater than non-edge sum (%f)", edgeSum, nonEdgeSum)
	}

	// Edge values should be significantly non-zero
	if edgeSum < 1000 {
		t.Errorf("edge sum (%f) seems too low for a clear vertical edge", edgeSum)
	}
}

func TestSobel_HorizontalEdge(t *testing.T) {
	// A horizontal edge should have high gradients along the edge
	img := horizontalEdgeImage(20, 20)
	result := Sobel(img)

	// Check that edge region (around y=10) has high values
	edgeSum := 0.0
	for x := 0; x < 20; x++ {
		edgeSum += result[9][x] + result[10][x]
	}

	// Check that non-edge regions have low values
	nonEdgeSum := 0.0
	for x := 0; x < 20; x++ {
		nonEdgeSum += result[0][x] + result[19][x]
	}

	if edgeSum <= nonEdgeSum {
		t.Errorf("edge sum (%f) should be greater than non-edge sum (%f)", edgeSum, nonEdgeSum)
	}
}

func TestSobel_DiagonalGradient(t *testing.T) {
	// A diagonal gradient should have smooth non-zero values throughout
	img := diagonalGradientImage(20, 20)
	result := Sobel(img)

	// Interior pixels should have non-zero gradients
	nonZeroCount := 0
	for y := 1; y < 19; y++ {
		for x := 1; x < 19; x++ {
			if result[y][x] > 0 {
				nonZeroCount++
			}
		}
	}

	// Most interior pixels should have non-zero gradients
	expectedNonZero := 18 * 18 / 2 // At least half
	if nonZeroCount < expectedNonZero {
		t.Errorf("expected at least %d non-zero pixels, got %d", expectedNonZero, nonZeroCount)
	}
}

func TestSobel_SmallImages(t *testing.T) {
	// Test edge cases with very small images
	testCases := []struct {
		width, height int
	}{
		{1, 1},
		{2, 2},
		{3, 3},
		{1, 10},
		{10, 1},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			img := uniformImage(tc.width, tc.height, color.White)
			result := Sobel(img)

			if len(result) != tc.height {
				t.Errorf("expected height %d, got %d", tc.height, len(result))
			}
			if len(result[0]) != tc.width {
				t.Errorf("expected width %d, got %d", tc.width, len(result[0]))
			}
		})
	}
}

func TestSobel_KnownValues(t *testing.T) {
	// Create a simple 3x3 image with known values
	// [0,   0,   0]
	// [0, 255, 255]
	// [0, 255, 255]
	// This should produce a gradient at the edge
	img := image.NewGray(image.Rect(0, 0, 3, 3))
	img.SetGray(0, 0, color.Gray{Y: 0})
	img.SetGray(1, 0, color.Gray{Y: 0})
	img.SetGray(2, 0, color.Gray{Y: 0})
	img.SetGray(0, 1, color.Gray{Y: 0})
	img.SetGray(1, 1, color.Gray{Y: 255})
	img.SetGray(2, 1, color.Gray{Y: 255})
	img.SetGray(0, 2, color.Gray{Y: 0})
	img.SetGray(1, 2, color.Gray{Y: 255})
	img.SetGray(2, 2, color.Gray{Y: 255})

	result := Sobel(img)

	// The center pixel (1,1) should have the highest gradient
	// because it's at the corner of the edge
	centerGrad := result[1][1]
	if centerGrad < 100 {
		t.Errorf("center gradient should be high, got %f", centerGrad)
	}

	// Corners should have lower gradients due to clamping
	topLeftGrad := result[0][0]
	bottomRightGrad := result[2][2]
	if topLeftGrad >= centerGrad {
		t.Errorf("top-left gradient (%f) should be less than center (%f)", topLeftGrad, centerGrad)
	}
	if bottomRightGrad >= centerGrad {
		t.Errorf("bottom-right gradient (%f) should be less than center (%f)", bottomRightGrad, centerGrad)
	}
}

func TestSobelSum(t *testing.T) {
	// Create a simple gradient map
	sobelMap := [][]float64{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}

	testCases := []struct {
		rect     image.Rectangle
		expected float64
	}{
		{image.Rect(0, 0, 3, 3), 45},            // Full image: sum of 1-9
		{image.Rect(0, 0, 1, 1), 1},             // Single pixel
		{image.Rect(1, 1, 3, 3), 5 + 6 + 8 + 9}, // Bottom-right 2x2
		{image.Rect(0, 0, 2, 2), 1 + 2 + 4 + 5}, // Top-left 2x2
		{image.Rect(-1, -1, 1, 1), 1},           // Clamped to valid area
		{image.Rect(2, 2, 5, 5), 9},             // Extends past bounds
	}

	for _, tc := range testCases {
		sum := SobelSum(sobelMap, tc.rect)
		if sum != tc.expected {
			t.Errorf("SobelSum(%v) = %f, expected %f", tc.rect, sum, tc.expected)
		}
	}
}

func TestSobelMean(t *testing.T) {
	sobelMap := [][]float64{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}

	// Full image: mean of 1-9 = 45/9 = 5
	mean := SobelMean(sobelMap, image.Rect(0, 0, 3, 3))
	if mean != 5 {
		t.Errorf("SobelMean(full) = %f, expected 5", mean)
	}

	// Empty rectangle
	mean = SobelMean(sobelMap, image.Rect(0, 0, 0, 0))
	if mean != 0 {
		t.Errorf("SobelMean(empty) = %f, expected 0", mean)
	}
}

func TestSobelSum_EmptyMap(t *testing.T) {
	var emptyMap [][]float64
	sum := SobelSum(emptyMap, image.Rect(0, 0, 10, 10))
	if sum != 0 {
		t.Errorf("SobelSum(empty map) = %f, expected 0", sum)
	}
}

func TestToGrayscaleFlat(t *testing.T) {
	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	img.Set(0, 0, color.White)
	img.Set(1, 0, color.Black)
	img.Set(2, 0, color.Gray{Y: 128})
	img.Set(0, 1, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	img.Set(1, 1, color.RGBA{R: 0, G: 255, B: 0, A: 255})
	img.Set(2, 1, color.RGBA{R: 0, G: 0, B: 255, A: 255})

	pixels, width, height := ToGrayscaleFlat(img)

	if width != 3 || height != 2 {
		t.Errorf("expected 3x2, got %dx%d", width, height)
	}

	if len(pixels) != 6 {
		t.Errorf("expected 6 pixels, got %d", len(pixels))
	}

	// White should be 255
	if pixels[0] != 255 {
		t.Errorf("white pixel = %d, expected 255", pixels[0])
	}

	// Black should be 0
	if pixels[1] != 0 {
		t.Errorf("black pixel = %d, expected 0", pixels[1])
	}

	// Gray should be 128
	if pixels[2] != 128 {
		t.Errorf("gray pixel = %d, expected 128", pixels[2])
	}

	// Red, green, blue should have different grayscale values based on luminance
	// Y = 0.299*R + 0.587*G + 0.114*B
	redGray := pixels[3]
	greenGray := pixels[4]
	blueGray := pixels[5]

	// Red: ~76, Green: ~150, Blue: ~29
	if redGray < 70 || redGray > 80 {
		t.Errorf("red grayscale = %d, expected ~76", redGray)
	}
	if greenGray < 145 || greenGray > 155 {
		t.Errorf("green grayscale = %d, expected ~150", greenGray)
	}
	if blueGray < 25 || blueGray > 35 {
		t.Errorf("blue grayscale = %d, expected ~29", blueGray)
	}
}

func TestToGrayscaleFlat_NonZeroOrigin(t *testing.T) {
	// Test with an image that doesn't start at (0,0)
	img := image.NewRGBA(image.Rect(10, 20, 13, 22))
	img.Set(10, 20, color.White)
	img.Set(11, 20, color.Black)
	img.Set(12, 20, color.Gray{Y: 100})
	img.Set(10, 21, color.Gray{Y: 50})
	img.Set(11, 21, color.Gray{Y: 150})
	img.Set(12, 21, color.Gray{Y: 200})

	pixels, width, height := ToGrayscaleFlat(img)

	if width != 3 || height != 2 {
		t.Errorf("expected 3x2, got %dx%d", width, height)
	}

	if pixels[0] != 255 {
		t.Errorf("pixel[0] = %d, expected 255 (white)", pixels[0])
	}
	if pixels[1] != 0 {
		t.Errorf("pixel[1] = %d, expected 0 (black)", pixels[1])
	}
}

func BenchmarkSobel(b *testing.B) {
	img := diagonalGradientImage(256, 256)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sobel(img)
	}
}

func BenchmarkSobel_Large(b *testing.B) {
	img := diagonalGradientImage(640, 480)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sobel(img)
	}
}

func BenchmarkToGrayscaleFlat(b *testing.B) {
	img := diagonalGradientImage(640, 480)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ToGrayscaleFlat(img)
	}
}

// Helpers for floating point comparison
func approxEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}
