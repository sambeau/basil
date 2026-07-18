package smartcrop

import (
	"image"
	"image/color"
	"testing"
)

func TestBestCrop_NilImage(t *testing.T) {
	result := BestCrop(nil, 100, 100, nil)
	if !result.Empty() {
		t.Errorf("expected empty rect for nil image, got %v", result)
	}
}

func TestBestCrop_ZeroDimensions(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	result := BestCrop(img, 0, 100, nil)
	if !result.Empty() {
		t.Errorf("expected empty rect for zero width, got %v", result)
	}

	result = BestCrop(img, 100, 0, nil)
	if !result.Empty() {
		t.Errorf("expected empty rect for zero height, got %v", result)
	}
}

func TestBestCrop_ImageSmallerThanTarget(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))

	result := BestCrop(img, 100, 100, nil)

	// Should return full image bounds
	expected := img.Bounds()
	if result != expected {
		t.Errorf("expected full bounds %v, got %v", expected, result)
	}
}

func TestBestCrop_ReturnsValidRectangle(t *testing.T) {
	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, 400, 300))
	for y := range 300 {
		for x := range 400 {
			img.Set(x, y, color.Gray{Y: uint8((x + y) % 256)})
		}
	}

	result := BestCrop(img, 200, 200, nil)

	// Result should be non-empty
	if result.Empty() {
		t.Error("expected non-empty result")
	}

	// Result should be within image bounds
	if !result.In(img.Bounds()) {
		t.Errorf("result %v is not within bounds %v", result, img.Bounds())
	}

	// Result should have approximately correct aspect ratio (1:1)
	aspect := float64(result.Dx()) / float64(result.Dy())
	if aspect < 0.9 || aspect > 1.1 {
		t.Errorf("aspect ratio %f should be ~1.0", aspect)
	}
}

func TestBestCrop_WithFocalPoint(t *testing.T) {
	// Create a wide image where a square crop MUST choose a position
	// (can't include the whole image at 1:1 aspect ratio)
	img := image.NewRGBA(image.Rect(0, 0, 800, 400))
	for y := range 400 {
		for x := range 800 {
			// Add gradient to create variation
			img.Set(x, y, color.RGBA{
				R: uint8(x * 255 / 800),
				G: uint8(y * 255 / 400),
				B: 128,
				A: 255,
			})
		}
	}

	// Crop with focal point at left side
	focalLeft := &FocalRegion{X: 0.1, Y: 0.5}
	resultLeft := BestCrop(img, 300, 300, focalLeft)

	// Crop with focal point at right side
	focalRight := &FocalRegion{X: 0.9, Y: 0.5}
	resultRight := BestCrop(img, 300, 300, focalRight)

	// The crops should be in different horizontal positions
	// Left focal should pull crop toward left, right focal toward right
	if resultLeft.Min.X >= resultRight.Min.X {
		t.Errorf("left focal should produce crop further left than right focal: Left=%v, Right=%v", resultLeft, resultRight)
	}
}

func TestBestCrop_WithFocalRegion(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 400, 400))
	for y := range 400 {
		for x := range 400 {
			img.Set(x, y, color.Gray{Y: 128})
		}
	}

	// Focal region in top-left quadrant
	focal := &FocalRegion{X: 0.1, Y: 0.1, W: 0.2, H: 0.2}
	result := BestCrop(img, 200, 200, focal)

	// Result should be non-empty and within bounds
	if result.Empty() {
		t.Error("expected non-empty result with focal region")
	}
	if !result.In(img.Bounds()) {
		t.Errorf("result %v not within bounds", result)
	}
}

func TestBestCrop_DifferentAspectRatios(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 400, 300))
	for y := range 300 {
		for x := range 400 {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}

	testCases := []struct {
		name         string
		targetW      int
		targetH      int
		expectAspect float64
	}{
		{"square", 100, 100, 1.0},
		{"wide 16:9", 160, 90, 16.0 / 9.0},
		{"tall 3:4", 90, 120, 3.0 / 4.0},
		{"very wide", 200, 50, 4.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := BestCrop(img, tc.targetW, tc.targetH, nil)

			if result.Empty() {
				t.Fatal("expected non-empty result")
			}

			aspect := float64(result.Dx()) / float64(result.Dy())
			tolerance := 0.15 // 15% tolerance
			if aspect < tc.expectAspect*(1-tolerance) || aspect > tc.expectAspect*(1+tolerance) {
				t.Errorf("aspect ratio %f should be ~%f", aspect, tc.expectAspect)
			}
		})
	}
}

func TestFocalToBoost_Point(t *testing.T) {
	focal := &FocalRegion{X: 0.5, Y: 0.5}
	boost := focalToBoost(focal, 100, 100)

	// Should create a boost region centered at (50, 50)
	if boost.Rect.Empty() {
		t.Fatal("expected non-empty boost region")
	}

	// Check it's roughly centered
	centerX := (boost.Rect.Min.X + boost.Rect.Max.X) / 2
	centerY := (boost.Rect.Min.Y + boost.Rect.Max.Y) / 2

	if centerX < 45 || centerX > 55 {
		t.Errorf("boost center X = %d, expected ~50", centerX)
	}
	if centerY < 45 || centerY > 55 {
		t.Errorf("boost center Y = %d, expected ~50", centerY)
	}

	// Weight should be 1.0
	if boost.Weight != 5.0 {
		t.Errorf("weight = %f, expected 5.0", boost.Weight)
	}
}

func TestFocalToBoost_Rectangle(t *testing.T) {
	focal := &FocalRegion{X: 0.2, Y: 0.3, W: 0.4, H: 0.5}
	boost := focalToBoost(focal, 100, 100)

	// Should create a boost region at specified position
	if boost.Rect.Empty() {
		t.Fatal("expected non-empty boost region")
	}

	// Check position (X, Y is top-left of region)
	if boost.Rect.Min.X != 20 {
		t.Errorf("rect Min.X = %d, expected 20", boost.Rect.Min.X)
	}
	if boost.Rect.Min.Y != 30 {
		t.Errorf("rect Min.Y = %d, expected 30", boost.Rect.Min.Y)
	}

	// Check size
	if boost.Rect.Dx() != 40 {
		t.Errorf("rect width = %d, expected 40", boost.Rect.Dx())
	}
	if boost.Rect.Dy() != 50 {
		t.Errorf("rect height = %d, expected 50", boost.Rect.Dy())
	}
}

func TestFocalToBoost_Nil(t *testing.T) {
	boost := focalToBoost(nil, 100, 100)
	if !boost.Rect.Empty() {
		t.Error("nil focal should produce empty boost")
	}
}

func TestResizeForAnalysis(t *testing.T) {
	testCases := []struct {
		name      string
		width     int
		height    int
		maxSize   int
		expectW   int
		expectH   int
		expectScl float64
	}{
		{"already small", 100, 80, 256, 100, 80, 1.0},
		{"landscape", 1000, 800, 256, 256, 204, 0.256},
		{"portrait", 800, 1000, 256, 204, 256, 0.256},
		{"square", 500, 500, 256, 256, 256, 0.512},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, tc.width, tc.height))
			resized, scale := resizeForAnalysis(img, tc.maxSize)

			bounds := resized.Bounds()
			if tc.expectScl == 1.0 {
				// Should return original
				if scale != 1.0 {
					t.Errorf("scale = %f, expected 1.0", scale)
				}
				if bounds.Dx() != tc.width || bounds.Dy() != tc.height {
					t.Errorf("size = %dx%d, expected %dx%d", bounds.Dx(), bounds.Dy(), tc.width, tc.height)
				}
			} else {
				// Should be resized
				if bounds.Dx() != tc.expectW {
					t.Errorf("width = %d, expected %d", bounds.Dx(), tc.expectW)
				}
				// Height might vary slightly due to rounding
				if bounds.Dy() < tc.expectH-2 || bounds.Dy() > tc.expectH+2 {
					t.Errorf("height = %d, expected ~%d", bounds.Dy(), tc.expectH)
				}
			}
		})
	}
}

func TestScaleBoosts(t *testing.T) {
	boosts := []BoostRegion{
		{Rect: image.Rect(100, 100, 200, 200), Weight: 1.0},
		{Rect: image.Rect(300, 300, 400, 400), Weight: 0.5},
	}

	// Scale down by half
	scaled := scaleBoosts(boosts, 0.5)

	if len(scaled) != 2 {
		t.Fatalf("expected 2 scaled boosts, got %d", len(scaled))
	}

	// First boost should be at (50, 50) to (100, 100)
	if scaled[0].Rect.Min.X != 50 || scaled[0].Rect.Min.Y != 50 {
		t.Errorf("first boost Min = (%d, %d), expected (50, 50)",
			scaled[0].Rect.Min.X, scaled[0].Rect.Min.Y)
	}
	if scaled[0].Weight != 1.0 {
		t.Errorf("first boost weight = %f, expected 1.0 (face detection weight)", scaled[0].Weight)
	}

	// Second boost should be at (150, 150) to (200, 200)
	if scaled[1].Rect.Min.X != 150 || scaled[1].Rect.Min.Y != 150 {
		t.Errorf("second boost Min = (%d, %d), expected (150, 150)",
			scaled[1].Rect.Min.X, scaled[1].Rect.Min.Y)
	}
}

func TestScaleBoosts_NoScale(t *testing.T) {
	boosts := []BoostRegion{
		{Rect: image.Rect(100, 100, 200, 200), Weight: 1.0},
	}

	// Scale of 1.0 should return original
	scaled := scaleBoosts(boosts, 1.0)

	if len(scaled) != 1 {
		t.Fatalf("expected 1 boost, got %d", len(scaled))
	}
	if scaled[0].Rect != boosts[0].Rect {
		t.Error("scale 1.0 should return original boost")
	}
}

func TestGenerateCandidates(t *testing.T) {
	bounds := image.Rect(0, 0, 256, 256)
	aspect := 1.0 // Square

	candidates := generateCandidates(bounds, aspect)

	if len(candidates) == 0 {
		t.Fatal("expected some candidates")
	}

	// All candidates should be within bounds
	for i, c := range candidates {
		if !c.In(bounds) {
			t.Errorf("candidate %d (%v) is not within bounds %v", i, c, bounds)
		}
	}

	// All candidates should have approximately correct aspect ratio
	for i, c := range candidates {
		cAspect := float64(c.Dx()) / float64(c.Dy())
		if cAspect < 0.9 || cAspect > 1.1 {
			t.Errorf("candidate %d has aspect %f, expected ~1.0", i, cAspect)
		}
	}
}

func TestGenerateCandidates_WideAspect(t *testing.T) {
	bounds := image.Rect(0, 0, 256, 256)
	aspect := 16.0 / 9.0 // Wide

	candidates := generateCandidates(bounds, aspect)

	if len(candidates) == 0 {
		t.Fatal("expected some candidates")
	}

	// Check aspect ratios
	for i, c := range candidates {
		cAspect := float64(c.Dx()) / float64(c.Dy())
		if cAspect < aspect*0.85 || cAspect > aspect*1.15 {
			t.Errorf("candidate %d has aspect %f, expected ~%f", i, cAspect, aspect)
		}
	}
}

func TestGenerateCandidates_Empty(t *testing.T) {
	// Empty bounds should produce no candidates
	candidates := generateCandidates(image.Rect(0, 0, 0, 0), 1.0)
	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates for empty bounds, got %d", len(candidates))
	}
}

func TestCenterCrop(t *testing.T) {
	bounds := image.Rect(0, 0, 400, 300)

	testCases := []struct {
		name   string
		aspect float64
	}{
		{"square", 1.0},
		{"wide", 16.0 / 9.0},
		{"tall", 3.0 / 4.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := centerCrop(bounds, tc.aspect)

			// Should be centered
			imgCenterX := (bounds.Min.X + bounds.Max.X) / 2
			imgCenterY := (bounds.Min.Y + bounds.Max.Y) / 2
			cropCenterX := (result.Min.X + result.Max.X) / 2
			cropCenterY := (result.Min.Y + result.Max.Y) / 2

			if cropCenterX < imgCenterX-5 || cropCenterX > imgCenterX+5 {
				t.Errorf("crop center X = %d, expected ~%d", cropCenterX, imgCenterX)
			}
			if cropCenterY < imgCenterY-5 || cropCenterY > imgCenterY+5 {
				t.Errorf("crop center Y = %d, expected ~%d", cropCenterY, imgCenterY)
			}

			// Should have correct aspect ratio
			resultAspect := float64(result.Dx()) / float64(result.Dy())
			if resultAspect < tc.aspect*0.95 || resultAspect > tc.aspect*1.05 {
				t.Errorf("aspect = %f, expected ~%f", resultAspect, tc.aspect)
			}
		})
	}
}

func TestScaleRectToOriginal(t *testing.T) {
	originalBounds := image.Rect(0, 0, 1000, 1000)

	// Rect at analysis scale (scale = 0.25, so 250px analysis size)
	analysisRect := image.Rect(25, 50, 125, 150)

	result := scaleRectToOriginal(analysisRect, 0.25, originalBounds)

	// Should be scaled up 4x
	if result.Min.X != 100 {
		t.Errorf("Min.X = %d, expected 100", result.Min.X)
	}
	if result.Min.Y != 200 {
		t.Errorf("Min.Y = %d, expected 200", result.Min.Y)
	}
	if result.Max.X != 500 {
		t.Errorf("Max.X = %d, expected 500", result.Max.X)
	}
	if result.Max.Y != 600 {
		t.Errorf("Max.Y = %d, expected 600", result.Max.Y)
	}
}

func TestScaleRectToOriginal_Clamped(t *testing.T) {
	originalBounds := image.Rect(0, 0, 100, 100)

	// Rect that extends beyond bounds - should be clamped by Intersect
	analysisRect := image.Rect(80, 80, 120, 120)

	result := scaleRectToOriginal(analysisRect, 1.0, originalBounds)

	// Should be clamped to original bounds via Intersect
	// Expected: (80,80)-(100,100) after intersection
	if result.Max.X != 100 || result.Max.Y != 100 {
		t.Errorf("result Max should be clamped to (100,100), got %v", result)
	}
	if result.Min.X != 80 || result.Min.Y != 80 {
		t.Errorf("result Min should be (80,80), got %v", result)
	}
}

func BenchmarkBestCrop(b *testing.B) {
	// Create a 640x480 test image
	img := image.NewRGBA(image.Rect(0, 0, 640, 480))
	for y := range 480 {
		for x := range 640 {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 2) % 256),
				G: uint8((y * 2) % 256),
				B: 128,
				A: 255,
			})
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BestCrop(img, 300, 300, nil)
	}
}

func BenchmarkBestCrop_WithFocal(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 640, 480))
	for y := range 480 {
		for x := range 640 {
			img.Set(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}

	focal := &FocalRegion{X: 0.3, Y: 0.4}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BestCrop(img, 300, 300, focal)
	}
}

func BenchmarkGenerateCandidates(b *testing.B) {
	bounds := image.Rect(0, 0, 256, 256)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateCandidates(bounds, 1.0)
	}
}
