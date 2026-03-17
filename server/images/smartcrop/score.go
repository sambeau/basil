package smartcrop

import (
	"image"
	"image/color"
	"math"
)

// BoostRegion represents a weighted region of interest (e.g., detected face or focal point).
// Boost regions heavily influence crop selection - a crop containing a boost region
// will score much higher than one without.
type BoostRegion struct {
	Rect   image.Rectangle
	Weight float64
}

// ScoringWeights holds the weights for different scoring signals.
// These can be tuned based on test image results.
type ScoringWeights struct {
	EdgeDensity      float64 // Detail/structure signal
	Saturation       float64 // Color interest signal
	SkinTone         float64 // Skin-tone presence (secondary to face detection)
	Composition      float64 // Rule-of-thirds, centering
	EdgeClutter      float64 // Penalty for busy crop edges (negative weight)
	BoostRegion      float64 // Face detection / focal point (dominant signal)
	SubjectMargin    float64 // Penalty for clipping boost regions (negative weight)
	SizePreference   float64 // Bonus for larger crops (prefer minimal cropping)
	SubjectCentered  float64 // Bonus for centering subject within crop
	CropCentered     float64 // Bonus for crop being centered on image (prefer center position)
	SubjectContained float64 // Bonus for fully containing subjects with margin
}

// DefaultScoringWeights returns the default scoring weights.
// These are tuned to prioritize faces/focal points while still considering composition.
// Size preference is strong to avoid over-cropping pre-composed images.
// CropCentered rewards crops positioned near the image center (default behavior).
// SubjectCentered rewards crops where detected subjects are centered within the crop.
func DefaultScoringWeights() ScoringWeights {
	return ScoringWeights{
		EdgeDensity:      0.1,
		Saturation:       0.1,
		SkinTone:         0.05,
		Composition:      0.1,
		EdgeClutter:      -0.1,
		BoostRegion:      20.0,  // Moderate - faces pull crop tighter
		SubjectMargin:    -30.0, // Strong penalty for clipping subjects
		SizePreference:   30.0,  // Moderate preference for larger crops (allows tighter face crops)
		SubjectCentered:  40.0,  // Moderate - center subjects within the crop
		CropCentered:     10.0,  // Light fallback - prefer centered crops when no subjects
		SubjectContained: 25.0,  // Strong bonus for fully containing subjects with margin
	}
}

// ScoreCandidate evaluates a candidate crop rectangle and returns a score.
// Higher scores indicate better crops. The score combines multiple signals:
// - Edge density (detail/structure)
// - Color saturation
// - Skin-tone presence
// - Composition (rule-of-thirds)
// - Edge clutter penalty
// - Boost region coverage (faces, focal points)
func ScoreCandidate(img image.Image, sobelMap [][]float64, crop image.Rectangle, boosts []BoostRegion) float64 {
	return ScoreCandidateWithWeights(img, sobelMap, crop, boosts, DefaultScoringWeights())
}

// ScoreCandidateWithWeights evaluates a crop with custom weights.
func ScoreCandidateWithWeights(img image.Image, sobelMap [][]float64, crop image.Rectangle, boosts []BoostRegion, weights ScoringWeights) float64 {
	if crop.Empty() {
		return 0
	}

	bounds := img.Bounds()
	cropArea := float64(crop.Dx() * crop.Dy())
	if cropArea == 0 {
		return 0
	}

	var score float64

	// Size preference - reward larger crops (prefer minimal cropping)
	// Score is ratio of crop area to image area, so 1.0 = full image
	imgArea := float64(bounds.Dx() * bounds.Dy())
	if imgArea > 0 {
		sizeScore := cropArea / imgArea
		score += weights.SizePreference * sizeScore
	}

	// Crop centered - reward crops positioned near the image center
	// This prevents arbitrary off-center positioning when there's no strong reason
	imgCenterX := float64(bounds.Min.X+bounds.Max.X) / 2
	imgCenterY := float64(bounds.Min.Y+bounds.Max.Y) / 2
	cropCenterX := float64(crop.Min.X+crop.Max.X) / 2
	cropCenterY := float64(crop.Min.Y+crop.Max.Y) / 2

	dx := cropCenterX - imgCenterX
	dy := cropCenterY - imgCenterY
	distFromCenter := math.Sqrt(dx*dx + dy*dy)
	imgDiagonal := math.Sqrt(float64(bounds.Dx()*bounds.Dx() + bounds.Dy()*bounds.Dy()))

	if imgDiagonal > 0 {
		// Gentle decay from center - crops near center score higher
		normalizedDist := distFromCenter / imgDiagonal
		centerScore := math.Exp(-3 * normalizedDist)
		score += weights.CropCentered * centerScore
	}

	// Edge density - normalized by crop area
	edgeScore := scoreEdgeDensity(sobelMap, crop)
	score += weights.EdgeDensity * edgeScore

	// Saturation
	satScore := scoreSaturation(img, crop)
	score += weights.Saturation * satScore

	// Skin-tone
	skinScore := scoreSkinTone(img, crop)
	score += weights.SkinTone * skinScore

	// Composition (rule-of-thirds, centering)
	compScore := scoreComposition(crop, bounds)
	score += weights.Composition * compScore

	// Edge clutter penalty
	clutterScore := scoreEdgeClutter(sobelMap, crop)
	score += weights.EdgeClutter * clutterScore

	// Boost regions (faces, focal points) - dominant signal
	boostScore, marginPenalty, centeringScore, containedScore := scoreBoostRegions(crop, boosts)
	score += weights.BoostRegion * boostScore
	score += weights.SubjectMargin * marginPenalty
	score += weights.SubjectCentered * centeringScore
	score += weights.SubjectContained * containedScore

	return score
}

// scoreEdgeDensity computes the mean gradient magnitude within the crop.
// Higher values indicate more detail/structure.
func scoreEdgeDensity(sobelMap [][]float64, crop image.Rectangle) float64 {
	return SobelMean(sobelMap, crop)
}

// scoreSaturation computes the mean saturation within the crop.
// Returns a value in [0, 1] where 1 is fully saturated.
func scoreSaturation(img image.Image, crop image.Rectangle) float64 {
	if crop.Empty() {
		return 0
	}

	bounds := img.Bounds()
	minX := max(crop.Min.X, bounds.Min.X)
	maxX := min(crop.Max.X, bounds.Max.X)
	minY := max(crop.Min.Y, bounds.Min.Y)
	maxY := min(crop.Max.Y, bounds.Max.Y)

	if minX >= maxX || minY >= maxY {
		return 0
	}

	var totalSat float64
	var count int

	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			c := img.At(x, y)
			_, s, _ := rgbToHSL(c)
			totalSat += s
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return totalSat / float64(count)
}

// scoreSkinTone computes the proportion of pixels that fall within skin-tone ranges.
// Uses a broad range to cover diverse skin tones.
func scoreSkinTone(img image.Image, crop image.Rectangle) float64 {
	if crop.Empty() {
		return 0
	}

	bounds := img.Bounds()
	minX := max(crop.Min.X, bounds.Min.X)
	maxX := min(crop.Max.X, bounds.Max.X)
	minY := max(crop.Min.Y, bounds.Min.Y)
	maxY := min(crop.Max.Y, bounds.Max.Y)

	if minX >= maxX || minY >= maxY {
		return 0
	}

	var skinCount int
	var totalCount int

	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			c := img.At(x, y)
			if isSkinTone(c) {
				skinCount++
			}
			totalCount++
		}
	}

	if totalCount == 0 {
		return 0
	}
	return float64(skinCount) / float64(totalCount)
}

// isSkinTone checks if a color falls within skin-tone ranges.
// Uses YCbCr color space which is better for skin detection across diverse tones.
// Tightened to exclude brown fur, orange objects, pink stripes while keeping
// good coverage for diverse human skin tones.
func isSkinTone(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	// Scale from 16-bit to 8-bit
	r8 := uint8(r >> 8)
	g8 := uint8(g >> 8)
	b8 := uint8(b >> 8)

	// Convert RGB to YCbCr
	y := 0.299*float64(r8) + 0.587*float64(g8) + 0.114*float64(b8)
	cb := 128 - 0.168736*float64(r8) - 0.331264*float64(g8) + 0.5*float64(b8)
	cr := 128 + 0.5*float64(r8) - 0.418688*float64(g8) - 0.081312*float64(b8)

	// Tightened YCbCr ranges for human skin detection
	// - Cr (red-difference): 140-165 excludes orange (high Cr) and brown (low Cr)
	// - Cb (blue-difference): 85-120 excludes pink/magenta (high Cb) and yellow/brown (low Cb)
	// - Y (luminance): 60-230 allows light to dark skin
	// These ranges still cover:
	// - Light/fair skin (high Y, moderate Cr/Cb)
	// - Medium/olive skin (medium Y, moderate Cr/Cb)
	// - Dark/deep skin (low Y, moderate Cr/Cb)
	if y > 60 && y < 230 && cb > 85 && cb < 120 && cr > 140 && cr < 165 {
		return true
	}

	// Secondary check: normalized RGB for edge cases
	// Tightened to require specific ratios that match human skin
	sum := float64(r8) + float64(g8) + float64(b8)
	if sum > 100 { // Require reasonable brightness
		rn := float64(r8) / sum
		gn := float64(g8) / sum
		bn := float64(b8) / sum
		// Human skin: R > G > B with specific ratios
		// Exclude brown (R and G too close) and orange (R too dominant)
		if rn > 0.38 && rn < 0.52 && gn > 0.28 && gn < 0.38 && bn > 0.15 && bn < 0.28 {
			if rn > gn && gn > bn { // Must have R > G > B ordering
				return true
			}
		}
	}

	return false
}

// scoreComposition evaluates how well the crop follows composition rules.
// Considers rule-of-thirds placement and overall balance.
func scoreComposition(crop, imgBounds image.Rectangle) float64 {
	if crop.Empty() || imgBounds.Empty() {
		return 0
	}

	// Calculate crop center relative to image
	cropCenterX := float64(crop.Min.X+crop.Max.X) / 2
	cropCenterY := float64(crop.Min.Y+crop.Max.Y) / 2

	imgWidth := float64(imgBounds.Dx())
	imgHeight := float64(imgBounds.Dy())
	imgCenterX := float64(imgBounds.Min.X) + imgWidth/2
	imgCenterY := float64(imgBounds.Min.Y) + imgHeight/2

	// Rule of thirds points
	thirds := []struct{ x, y float64 }{
		{imgCenterX - imgWidth/6, imgCenterY - imgHeight/6}, // Top-left third
		{imgCenterX + imgWidth/6, imgCenterY - imgHeight/6}, // Top-right third
		{imgCenterX - imgWidth/6, imgCenterY + imgHeight/6}, // Bottom-left third
		{imgCenterX + imgWidth/6, imgCenterY + imgHeight/6}, // Bottom-right third
		{imgCenterX, imgCenterY},                            // Center (also good)
	}

	// Find distance to nearest thirds point
	minDist := math.MaxFloat64
	for _, t := range thirds {
		dx := cropCenterX - t.x
		dy := cropCenterY - t.y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < minDist {
			minDist = dist
		}
	}

	// Normalize distance by image diagonal
	diagonal := math.Sqrt(imgWidth*imgWidth + imgHeight*imgHeight)
	if diagonal == 0 {
		return 0
	}
	normalizedDist := minDist / diagonal

	// Convert to score (closer to thirds = higher score)
	// Score of 1.0 at exact thirds point, decaying with distance
	score := math.Exp(-5 * normalizedDist)

	return score
}

// scoreEdgeClutter computes the mean gradient at crop edges.
// High edge clutter (busy edges) indicates partial objects being cut off.
func scoreEdgeClutter(sobelMap [][]float64, crop image.Rectangle) float64 {
	if crop.Empty() {
		return 0
	}

	height := len(sobelMap)
	if height == 0 {
		return 0
	}
	width := len(sobelMap[0])
	if width == 0 {
		return 0
	}

	// Calculate border thickness (10% of smaller dimension, min 1px)
	borderThickness := max(1, min(crop.Dx(), crop.Dy())/10)

	var borderSum float64
	var borderCount int

	// Top and bottom borders
	for y := crop.Min.Y; y < min(crop.Min.Y+borderThickness, crop.Max.Y); y++ {
		for x := crop.Min.X; x < crop.Max.X; x++ {
			if y >= 0 && y < height && x >= 0 && x < width {
				borderSum += sobelMap[y][x]
				borderCount++
			}
		}
	}
	for y := max(crop.Max.Y-borderThickness, crop.Min.Y); y < crop.Max.Y; y++ {
		for x := crop.Min.X; x < crop.Max.X; x++ {
			if y >= 0 && y < height && x >= 0 && x < width {
				borderSum += sobelMap[y][x]
				borderCount++
			}
		}
	}

	// Left and right borders (excluding corners already counted)
	for y := crop.Min.Y + borderThickness; y < crop.Max.Y-borderThickness; y++ {
		for x := crop.Min.X; x < min(crop.Min.X+borderThickness, crop.Max.X); x++ {
			if y >= 0 && y < height && x >= 0 && x < width {
				borderSum += sobelMap[y][x]
				borderCount++
			}
		}
		for x := max(crop.Max.X-borderThickness, crop.Min.X); x < crop.Max.X; x++ {
			if y >= 0 && y < height && x >= 0 && x < width {
				borderSum += sobelMap[y][x]
				borderCount++
			}
		}
	}

	if borderCount == 0 {
		return 0
	}
	return borderSum / float64(borderCount)
}

// scoreBoostRegions calculates how well a crop contains boost regions.
// Returns (coverage score, margin penalty, centering score, contained score).
// Coverage score is the weighted overlap with boost regions.
// Margin penalty is applied when boost regions are clipped.
// Centering score rewards placing subjects near the crop center (very light).
// Contained score rewards crops that fully contain subjects with breathing room.
func scoreBoostRegions(crop image.Rectangle, boosts []BoostRegion) (coverage, marginPenalty, centeringScore, containedScore float64) {
	if len(boosts) == 0 {
		return 0, 0, 0, 0
	}

	cropArea := float64(crop.Dx() * crop.Dy())
	if cropArea == 0 {
		return 0, 0, 0, 0
	}

	// Safe margin: 5% of crop dimensions - subjects should be this far from edges
	safeMarginX := float64(crop.Dx()) * 0.05
	safeMarginY := float64(crop.Dy()) * 0.05

	// Crop center for centering calculation
	cropCenterX := float64(crop.Min.X+crop.Max.X) / 2
	cropCenterY := float64(crop.Min.Y+crop.Max.Y) / 2
	cropDiagonal := math.Sqrt(float64(crop.Dx()*crop.Dx() + crop.Dy()*crop.Dy()))

	var totalCenteringScore float64
	var totalContainedScore float64
	var boostCount int

	for _, boost := range boosts {
		if boost.Rect.Empty() {
			continue
		}

		// Check if this boost is inside the crop
		intersection := crop.Intersect(boost.Rect)
		if intersection.Empty() {
			// Boost region is outside crop - heavy penalty
			marginPenalty += boost.Weight * 1.0
			continue
		}

		boostCenterX := float64(boost.Rect.Min.X+boost.Rect.Max.X) / 2
		boostCenterY := float64(boost.Rect.Min.Y+boost.Rect.Max.Y) / 2

		// Centering score - very gentle, just breaks ties
		dx := boostCenterX - cropCenterX
		dy := boostCenterY - cropCenterY
		dist := math.Sqrt(dx*dx + dy*dy)

		if cropDiagonal > 0 {
			normalizedDist := dist / cropDiagonal
			centering := math.Exp(-2 * normalizedDist)
			totalCenteringScore += boost.Weight * centering
			boostCount++
		}

		interArea := float64(intersection.Dx() * intersection.Dy())
		boostArea := float64(boost.Rect.Dx() * boost.Rect.Dy())

		// Coverage: proportion of boost region contained in crop
		if boostArea > 0 {
			overlapRatio := interArea / boostArea
			coverage += boost.Weight * overlapRatio
		}

		// Check containment: is the boost fully inside with safe margin?
		if intersection == boost.Rect {
			// Boost is fully contained - check margin from edges
			leftMargin := float64(boost.Rect.Min.X - crop.Min.X)
			rightMargin := float64(crop.Max.X - boost.Rect.Max.X)
			topMargin := float64(boost.Rect.Min.Y - crop.Min.Y)
			bottomMargin := float64(crop.Max.Y - boost.Rect.Max.Y)

			// Score based on how much margin we have (more margin = better)
			// Use the minimum margin as the limiting factor
			minHorizMargin := math.Min(leftMargin, rightMargin)
			minVertMargin := math.Min(topMargin, bottomMargin)

			// Normalize by safe margin target - 1.0 means we hit the safe margin
			horizScore := math.Min(1.0, minHorizMargin/safeMarginX)
			vertScore := math.Min(1.0, minVertMargin/safeMarginY)

			// Combined containment score - reward crops where subject has breathing room
			totalContainedScore += boost.Weight * horizScore * vertScore
		} else {
			// Boost is clipped - apply margin penalty based on how much is clipped
			clippedRatio := 1.0 - (interArea / boostArea)
			marginPenalty += boost.Weight * clippedRatio
		}
	}

	// Average scores (don't let many small detections dominate)
	if boostCount > 0 {
		centeringScore = totalCenteringScore / float64(boostCount)
		containedScore = totalContainedScore / float64(boostCount)
	}

	return coverage, marginPenalty, centeringScore, containedScore
}

// rgbToHSL converts an RGB color to HSL (hue, saturation, lightness).
// Returns h in [0, 360), s and l in [0, 1].
func rgbToHSL(c color.Color) (h, s, l float64) {
	r, g, b, _ := c.RGBA()
	// Scale from 16-bit to [0, 1]
	rf := float64(r) / 65535.0
	gf := float64(g) / 65535.0
	bf := float64(b) / 65535.0

	maxC := math.Max(rf, math.Max(gf, bf))
	minC := math.Min(rf, math.Min(gf, bf))
	delta := maxC - minC

	// Lightness
	l = (maxC + minC) / 2

	if delta == 0 {
		// Achromatic (gray)
		return 0, 0, l
	}

	// Saturation
	if l < 0.5 {
		s = delta / (maxC + minC)
	} else {
		s = delta / (2 - maxC - minC)
	}

	// Hue
	switch maxC {
	case rf:
		h = (gf - bf) / delta
		if gf < bf {
			h += 6
		}
	case gf:
		h = (bf-rf)/delta + 2
	case bf:
		h = (rf-gf)/delta + 4
	}
	h *= 60

	return h, s, l
}
