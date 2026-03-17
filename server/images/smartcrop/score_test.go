package smartcrop

import (
	"image"
	"image/color"
	"testing"
)

func TestDefaultScoringWeights(t *testing.T) {
	w := DefaultScoringWeights()

	// Boost region weight should be dominant
	if w.BoostRegion <= w.EdgeDensity+w.Saturation+w.SkinTone+w.Composition {
		t.Error("BoostRegion weight should dominate other signals")
	}

	// Edge clutter and subject margin should be negative (penalties)
	if w.EdgeClutter >= 0 {
		t.Error("EdgeClutter should be negative (penalty)")
	}
	if w.SubjectMargin >= 0 {
		t.Error("SubjectMargin should be negative (penalty)")
	}
}

func TestScoreEdgeDensity(t *testing.T) {
	// Create a simple gradient map
	sobelMap := [][]float64{
		{10, 20, 30},
		{40, 50, 60},
		{70, 80, 90},
	}

	testCases := []struct {
		name     string
		crop     image.Rectangle
		expected float64
	}{
		{"full image", image.Rect(0, 0, 3, 3), 50}, // Mean of 10-90
		{"top-left", image.Rect(0, 0, 2, 2), 30},   // Mean of 10,20,40,50
		{"center", image.Rect(1, 1, 2, 2), 50},     // Just center pixel
		{"empty", image.Rect(0, 0, 0, 0), 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := scoreEdgeDensity(sobelMap, tc.crop)
			if score != tc.expected {
				t.Errorf("scoreEdgeDensity() = %f, expected %f", score, tc.expected)
			}
		})
	}
}

func TestScoreSaturation(t *testing.T) {
	// Create a test image with varying saturation
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))

	// Fill with gray (low saturation)
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.Gray{Y: 128})
		}
	}

	// Gray area should have low saturation
	grayCrop := image.Rect(0, 0, 5, 5)
	grayScore := scoreSaturation(img, grayCrop)

	// Now add some saturated pixels
	for y := 5; y < 10; y++ {
		for x := 5; x < 10; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255}) // Pure red
		}
	}

	redCrop := image.Rect(5, 5, 10, 10)
	redScore := scoreSaturation(img, redCrop)

	// Red area should have higher saturation
	if redScore <= grayScore {
		t.Errorf("saturated region (%f) should score higher than gray (%f)", redScore, grayScore)
	}
}

func TestScoreSaturation_Empty(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	score := scoreSaturation(img, image.Rect(0, 0, 0, 0))
	if score != 0 {
		t.Errorf("empty crop should return 0, got %f", score)
	}
}

func TestScoreSkinTone(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))

	// Fill with non-skin color (blue)
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 0, B: 255, A: 255})
		}
	}

	blueCrop := image.Rect(0, 0, 10, 10)
	blueScore := scoreSkinTone(img, blueCrop)

	// Fill with skin-like color (various skin tones)
	skinTones := []color.RGBA{
		{R: 255, G: 224, B: 189, A: 255}, // Light skin
		{R: 198, G: 134, B: 66, A: 255},  // Medium skin
		{R: 141, G: 85, B: 36, A: 255},   // Dark skin
	}

	for i, y := 0, 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, skinTones[i%len(skinTones)])
		}
	}

	skinCrop := image.Rect(0, 0, 10, 10)
	skinScore := scoreSkinTone(img, skinCrop)

	// Skin tones should score higher than blue
	if skinScore <= blueScore {
		t.Errorf("skin tone region (%f) should score higher than blue (%f)", skinScore, blueScore)
	}
}

func TestIsSkinTone(t *testing.T) {
	// Test various skin tones - should all be detected
	skinTones := []color.Color{
		color.RGBA{R: 255, G: 224, B: 189, A: 255}, // Light/fair
		color.RGBA{R: 241, G: 194, B: 125, A: 255}, // Light-medium
		color.RGBA{R: 198, G: 134, B: 66, A: 255},  // Medium/olive
		color.RGBA{R: 161, G: 102, B: 56, A: 255},  // Tan
		color.RGBA{R: 141, G: 85, B: 36, A: 255},   // Brown
		color.RGBA{R: 100, G: 60, B: 30, A: 255},   // Dark brown
	}

	detectedCount := 0
	for _, c := range skinTones {
		if isSkinTone(c) {
			detectedCount++
		}
	}

	// Should detect at least some skin tones
	if detectedCount < len(skinTones)/2 {
		t.Errorf("detected only %d/%d skin tones", detectedCount, len(skinTones))
	}

	// Non-skin colors should not be detected (or rarely)
	nonSkin := []color.Color{
		color.RGBA{R: 0, G: 0, B: 255, A: 255},     // Blue
		color.RGBA{R: 0, G: 255, B: 0, A: 255},     // Green
		color.RGBA{R: 255, G: 255, B: 255, A: 255}, // White
		color.RGBA{R: 0, G: 0, B: 0, A: 255},       // Black
	}

	nonSkinDetected := 0
	for _, c := range nonSkin {
		if isSkinTone(c) {
			nonSkinDetected++
		}
	}

	// Should not detect many non-skin colors as skin
	if nonSkinDetected > 1 {
		t.Errorf("false positive: detected %d non-skin colors as skin", nonSkinDetected)
	}
}

func TestScoreComposition(t *testing.T) {
	imgBounds := image.Rect(0, 0, 300, 300)

	// Center crop should score well
	centerCrop := image.Rect(100, 100, 200, 200)
	centerScore := scoreComposition(centerCrop, imgBounds)

	// Corner crop should score lower
	cornerCrop := image.Rect(0, 0, 100, 100)
	cornerScore := scoreComposition(cornerCrop, imgBounds)

	// Thirds-aligned crop (center at 100,100 which is 1/3 point)
	thirdsCrop := image.Rect(50, 50, 150, 150)
	thirdsScore := scoreComposition(thirdsCrop, imgBounds)

	// All scores should be non-negative
	if centerScore < 0 || cornerScore < 0 || thirdsScore < 0 {
		t.Error("composition scores should be non-negative")
	}

	// Thirds and center should score better than corner
	if cornerScore >= centerScore && cornerScore >= thirdsScore {
		t.Errorf("corner crop (%f) should score lower than center (%f) or thirds (%f)",
			cornerScore, centerScore, thirdsScore)
	}
}

func TestScoreComposition_Empty(t *testing.T) {
	score := scoreComposition(image.Rect(0, 0, 0, 0), image.Rect(0, 0, 100, 100))
	if score != 0 {
		t.Errorf("empty crop should return 0, got %f", score)
	}

	score = scoreComposition(image.Rect(0, 0, 100, 100), image.Rect(0, 0, 0, 0))
	if score != 0 {
		t.Errorf("empty bounds should return 0, got %f", score)
	}
}

func TestScoreEdgeClutter(t *testing.T) {
	// Create a sobel map with high values at edges
	sobelMap := make([][]float64, 20)
	for y := range sobelMap {
		sobelMap[y] = make([]float64, 20)
		for x := 0; x < 20; x++ {
			// Low values in center, high at edges
			if x < 2 || x > 17 || y < 2 || y > 17 {
				sobelMap[y][x] = 100
			} else {
				sobelMap[y][x] = 10
			}
		}
	}

	// Crop that includes edges should have higher clutter
	edgeCrop := image.Rect(0, 0, 10, 10)
	edgeScore := scoreEdgeClutter(sobelMap, edgeCrop)

	// Crop in the center should have lower clutter
	centerCrop := image.Rect(5, 5, 15, 15)
	centerScore := scoreEdgeClutter(sobelMap, centerCrop)

	// Edge crop should have higher clutter score
	if edgeScore <= centerScore {
		t.Errorf("edge crop clutter (%f) should be higher than center (%f)",
			edgeScore, centerScore)
	}
}

func TestScoreBoostRegions_Empty(t *testing.T) {
	crop := image.Rect(0, 0, 100, 100)
	coverage, penalty := scoreBoostRegions(crop, nil)

	if coverage != 0 || penalty != 0 {
		t.Errorf("empty boosts should return (0, 0), got (%f, %f)", coverage, penalty)
	}
}

func TestScoreBoostRegions_FullContainment(t *testing.T) {
	crop := image.Rect(0, 0, 100, 100)
	boosts := []BoostRegion{
		{Rect: image.Rect(25, 25, 75, 75), Weight: 1.0},
	}

	coverage, penalty := scoreBoostRegions(crop, boosts)

	// Full containment should give full coverage
	if coverage != 1.0 {
		t.Errorf("full containment should give coverage 1.0, got %f", coverage)
	}
	// No clipping, so no penalty
	if penalty != 0 {
		t.Errorf("full containment should have no penalty, got %f", penalty)
	}
}

func TestScoreBoostRegions_NoOverlap(t *testing.T) {
	crop := image.Rect(0, 0, 50, 50)
	boosts := []BoostRegion{
		{Rect: image.Rect(100, 100, 150, 150), Weight: 1.0},
	}

	coverage, penalty := scoreBoostRegions(crop, boosts)

	// No overlap should give zero coverage
	if coverage != 0 {
		t.Errorf("no overlap should give coverage 0, got %f", coverage)
	}
	// Missing boost region should incur penalty
	if penalty <= 0 {
		t.Errorf("missing boost region should incur penalty, got %f", penalty)
	}
}

func TestScoreBoostRegions_PartialOverlap(t *testing.T) {
	crop := image.Rect(0, 0, 100, 100)
	// Boost region half inside, half outside
	boosts := []BoostRegion{
		{Rect: image.Rect(50, 50, 150, 150), Weight: 1.0},
	}

	coverage, penalty := scoreBoostRegions(crop, boosts)

	// Should have partial coverage
	if coverage <= 0 || coverage >= 1.0 {
		t.Errorf("partial overlap should give partial coverage, got %f", coverage)
	}
	// Clipping should incur penalty
	if penalty <= 0 {
		t.Errorf("clipping should incur penalty, got %f", penalty)
	}
}

func TestScoreBoostRegions_MultipleBoosts(t *testing.T) {
	crop := image.Rect(0, 0, 100, 100)
	boosts := []BoostRegion{
		{Rect: image.Rect(10, 10, 30, 30), Weight: 1.0},
		{Rect: image.Rect(60, 60, 90, 90), Weight: 0.5},
	}

	coverage, penalty := scoreBoostRegions(crop, boosts)

	// Both fully contained, coverage should be sum of weights
	expectedCoverage := 1.0 + 0.5
	if coverage != expectedCoverage {
		t.Errorf("multiple fully contained boosts should sum, got %f expected %f",
			coverage, expectedCoverage)
	}
	if penalty != 0 {
		t.Errorf("fully contained boosts should have no penalty, got %f", penalty)
	}
}

func TestScoreCandidate_BoostDominates(t *testing.T) {
	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.Gray{Y: 128})
		}
	}

	// Simple sobel map
	sobelMap := make([][]float64, 100)
	for y := range sobelMap {
		sobelMap[y] = make([]float64, 100)
		for x := 0; x < 100; x++ {
			sobelMap[y][x] = 10
		}
	}

	// Crop without boost
	cropNoBoost := image.Rect(0, 0, 50, 50)
	scoreNoBoost := ScoreCandidate(img, sobelMap, cropNoBoost, nil)

	// Crop with boost
	cropWithBoost := image.Rect(0, 0, 50, 50)
	boosts := []BoostRegion{
		{Rect: image.Rect(10, 10, 40, 40), Weight: 1.0},
	}
	scoreWithBoost := ScoreCandidate(img, sobelMap, cropWithBoost, boosts)

	// Crop with boost should score much higher
	if scoreWithBoost <= scoreNoBoost {
		t.Errorf("crop with boost (%f) should score much higher than without (%f)",
			scoreWithBoost, scoreNoBoost)
	}

	// The boost should dominate - at least 10x higher
	if scoreWithBoost < scoreNoBoost*10 {
		t.Errorf("boost should dominate: %f vs %f (expected 10x difference)",
			scoreWithBoost, scoreNoBoost)
	}
}

func TestScoreCandidate_EmptyCrop(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	sobelMap := make([][]float64, 100)
	for y := range sobelMap {
		sobelMap[y] = make([]float64, 100)
	}

	score := ScoreCandidate(img, sobelMap, image.Rect(0, 0, 0, 0), nil)
	if score != 0 {
		t.Errorf("empty crop should return 0, got %f", score)
	}
}

func TestRgbToHSL(t *testing.T) {
	testCases := []struct {
		name                              string
		color                             color.Color
		expectedH, minS, maxS, minL, maxL float64
	}{
		{
			name:      "red",
			color:     color.RGBA{R: 255, G: 0, B: 0, A: 255},
			expectedH: 0,
			minS:      0.9, maxS: 1.1,
			minL: 0.4, maxL: 0.6,
		},
		{
			name:      "green",
			color:     color.RGBA{R: 0, G: 255, B: 0, A: 255},
			expectedH: 120,
			minS:      0.9, maxS: 1.1,
			minL: 0.4, maxL: 0.6,
		},
		{
			name:      "blue",
			color:     color.RGBA{R: 0, G: 0, B: 255, A: 255},
			expectedH: 240,
			minS:      0.9, maxS: 1.1,
			minL: 0.4, maxL: 0.6,
		},
		{
			name:      "white",
			color:     color.RGBA{R: 255, G: 255, B: 255, A: 255},
			expectedH: 0, // Achromatic
			minS:      -0.1, maxS: 0.1,
			minL: 0.9, maxL: 1.1,
		},
		{
			name:      "gray",
			color:     color.RGBA{R: 128, G: 128, B: 128, A: 255},
			expectedH: 0, // Achromatic
			minS:      -0.1, maxS: 0.1,
			minL: 0.4, maxL: 0.6,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h, s, l := rgbToHSL(tc.color)

			// Check hue (with tolerance for red wrapping)
			if tc.expectedH == 0 {
				if h > 10 && h < 350 {
					t.Errorf("hue = %f, expected ~%f", h, tc.expectedH)
				}
			} else if h < tc.expectedH-10 || h > tc.expectedH+10 {
				t.Errorf("hue = %f, expected ~%f", h, tc.expectedH)
			}

			if s < tc.minS || s > tc.maxS {
				t.Errorf("saturation = %f, expected in [%f, %f]", s, tc.minS, tc.maxS)
			}

			if l < tc.minL || l > tc.maxL {
				t.Errorf("lightness = %f, expected in [%f, %f]", l, tc.minL, tc.maxL)
			}
		})
	}
}

func BenchmarkScoreCandidate(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}

	sobelMap := Sobel(img)
	crop := image.Rect(50, 50, 200, 200)
	boosts := []BoostRegion{
		{Rect: image.Rect(100, 100, 150, 150), Weight: 1.0},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ScoreCandidate(img, sobelMap, crop, boosts)
	}
}

func BenchmarkScoreSaturation(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}

	crop := image.Rect(50, 50, 200, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scoreSaturation(img, crop)
	}
}

func BenchmarkScoreSkinTone(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 150, B: 100, A: 255})
		}
	}

	crop := image.Rect(50, 50, 200, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scoreSkinTone(img, crop)
	}
}
