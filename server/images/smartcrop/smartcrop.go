package smartcrop

import (
	"image"

	"github.com/disintegration/imaging"
)

// FocalRegion represents a user-specified region of interest in normalised (0–1) coordinates.
// If W and H are zero, it's a point (expanded to a small boost region internally).
// If W and H are non-zero, it's a rectangle used directly as a boost region.
type FocalRegion struct {
	X float64 // 0–1, left to right
	Y float64 // 0–1, top to bottom
	W float64 // 0–1, width (0 = point)
	H float64 // 0–1, height (0 = point)
}

// CropCandidate represents a scored candidate crop rectangle.
type CropCandidate struct {
	Rect  image.Rectangle
	Score float64
}

// analysisSize is the size for heuristic scoring (Pass 2).
const analysisSize = 256

// faceDetectionSize is the size for face detection (Pass 1).
const faceDetectionSize = 640

// BestCrop analyses the image and returns the best crop rectangle for the target dimensions.
// It uses a two-pass analysis:
//   - Pass 1: Face detection at 640px (reliable for faces >= 10% of image height)
//   - Pass 2: Heuristic scoring at 256px (fast candidate search)
//
// The focal parameter allows specifying a region of interest that will be treated
// like a detected face (heavily weighted in scoring).
func BestCrop(img image.Image, targetWidth, targetHeight int, focal *FocalRegion) image.Rectangle {
	if img == nil || targetWidth <= 0 || targetHeight <= 0 {
		return image.Rectangle{}
	}

	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// If image is smaller than target, return full image bounds
	if srcWidth <= targetWidth && srcHeight <= targetHeight {
		return bounds
	}

	// Calculate target aspect ratio
	targetAspect := float64(targetWidth) / float64(targetHeight)

	// Collect boost regions from face detection and focal point
	var boosts []BoostRegion

	// Pass 1: Face detection at 640px
	faceBoosts, hadRawDetections := detectFaceBoosts(img, srcWidth, srcHeight)
	boosts = append(boosts, faceBoosts...)

	// Convert focal point/region to boost
	if focal != nil {
		focalBoost := focalToBoost(focal, srcWidth, srcHeight)
		if !focalBoost.Rect.Empty() {
			boosts = append(boosts, focalBoost)
		}
	}

	// Pass 2: Heuristic scoring at 256px
	analysisImg, analysisScale := resizeForAnalysis(img, analysisSize)
	sobelMap := Sobel(analysisImg)

	// If no faces or focal points, find the energy centroid as a fallback
	// This helps find "interesting" regions like isolated objects, boundaries, etc.
	// Note: We check hadRawDetections to avoid energy centroid overriding
	// weak but valid face detections that were filtered by confidence threshold.
	if len(boosts) == 0 && !hadRawDetections {
		energyBoost := findEnergyCentroid(sobelMap, analysisScale, srcWidth, srcHeight)
		if !energyBoost.Rect.Empty() {
			boosts = append(boosts, energyBoost)
		}
	}

	// Scale boost regions to analysis coordinates
	scaledBoosts := scaleBoosts(boosts, analysisScale)

	// Generate and score candidate crops
	analysisBounds := analysisImg.Bounds()
	candidates := generateCandidates(analysisBounds, targetAspect)

	if len(candidates) == 0 {
		// Fallback to center crop
		return centerCrop(bounds, targetAspect)
	}

	// Score all candidates
	var bestCandidate CropCandidate
	bestCandidate.Score = -1e9 // Start with very low score

	for _, rect := range candidates {
		score := ScoreCandidate(analysisImg, sobelMap, rect, scaledBoosts)
		if score > bestCandidate.Score {
			bestCandidate.Rect = rect
			bestCandidate.Score = score
		}
	}

	// Check if all scores are very similar (uniform image) - fallback to center
	if shouldFallbackToCenter(candidates, analysisImg, sobelMap, scaledBoosts, bestCandidate.Score) {
		return centerCrop(bounds, targetAspect)
	}

	// Map best crop back to original image coordinates
	return scaleRectToOriginal(bestCandidate.Rect, analysisScale, bounds)
}

// detectFaceBoosts runs face detection and converts results to boost regions.
// Face detection confidence is used to weight the boost - low confidence detections
// have less influence on crop selection.
// Detections are validated with skin-tone analysis to filter out false positives
// (e.g., toy faces, patterns that look like faces).
// Returns the boost regions and whether any raw detections were found (before filtering).
func detectFaceBoosts(img image.Image, srcWidth, srcHeight int) ([]BoostRegion, bool) {
	// Resize for face detection
	faceImg, faceScale := resizeForAnalysis(img, faceDetectionSize)

	// Run face detection
	pixels, width, height := ToGrayscaleFlat(faceImg)
	detections := DetectFaces(pixels, width, height)
	hadRawDetections := len(detections) > 0

	if len(detections) == 0 {
		return nil, false
	}

	// Find max score for normalization
	var maxScore float32
	for _, det := range detections {
		if det.Score > maxScore {
			maxScore = det.Score
		}
	}

	// Minimum confidence threshold - ignore very weak detections
	// Set to 2.0 to balance catching faces in difficult conditions
	// while filtering noisy false positives that pull crops off-center
	const minConfidenceThreshold = 2.0

	// Minimum skin-tone percentage required to validate a face detection
	// This filters out toy faces, patterns, pink objects etc. that fool the detector
	const minSkinTonePercent = 0.50

	// Convert detections to boost regions in original image coordinates
	boosts := make([]BoostRegion, 0, len(detections))
	for _, det := range detections {
		// Skip low-confidence detections entirely
		if det.Score < minConfidenceThreshold {
			continue
		}

		// Get detection rectangle in face detection scale
		detRect := det.Rect()

		// Scale back to original coordinates
		origRect := image.Rect(
			int(float64(detRect.Min.X)/faceScale),
			int(float64(detRect.Min.Y)/faceScale),
			int(float64(detRect.Max.X)/faceScale),
			int(float64(detRect.Max.Y)/faceScale),
		)

		// Clamp to image bounds
		origRect = origRect.Intersect(image.Rect(0, 0, srcWidth, srcHeight))

		if origRect.Empty() {
			continue
		}

		// Validate with skin-tone analysis to filter false positives
		skinPercent := calculateSkinTonePercent(img, origRect)
		if skinPercent < minSkinTonePercent {
			// Not enough skin tone - likely a toy, pattern, or non-human face
			continue
		}

		// Weight by confidence: normalize to [0.3, 1.0] range
		// so even the weakest accepted detection has some influence
		weight := 0.3 + 0.7*float64(det.Score)/float64(maxScore)

		boosts = append(boosts, BoostRegion{
			Rect:   origRect,
			Weight: weight,
		})
	}

	return boosts, hadRawDetections
}

// calculateSkinTonePercent samples pixels in a rectangle and returns the
// percentage that match skin-tone colors. Used to validate face detections.
func calculateSkinTonePercent(img image.Image, rect image.Rectangle) float64 {
	if rect.Empty() {
		return 0
	}

	bounds := img.Bounds()
	minX := max(rect.Min.X, bounds.Min.X)
	maxX := min(rect.Max.X, bounds.Max.X)
	minY := max(rect.Min.Y, bounds.Min.Y)
	maxY := min(rect.Max.Y, bounds.Max.Y)

	if minX >= maxX || minY >= maxY {
		return 0
	}

	// Sample every 4th pixel for performance (faces are usually large enough)
	const sampleStep = 4
	var skinCount, totalCount int

	for y := minY; y < maxY; y += sampleStep {
		for x := minX; x < maxX; x += sampleStep {
			if isSkinTone(img.At(x, y)) {
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

// focalToBoost converts a focal point/region to a boost region.
func focalToBoost(focal *FocalRegion, imgWidth, imgHeight int) BoostRegion {
	if focal == nil {
		return BoostRegion{}
	}

	// Convert normalized coordinates to pixels
	centerX := int(focal.X * float64(imgWidth))
	centerY := int(focal.Y * float64(imgHeight))

	var rect image.Rectangle

	if focal.W > 0 && focal.H > 0 {
		// Focal is a rectangle
		w := int(focal.W * float64(imgWidth))
		h := int(focal.H * float64(imgHeight))
		rect = image.Rect(centerX, centerY, centerX+w, centerY+h)
	} else {
		// Focal is a point - expand to 10% of image dimensions
		expandW := imgWidth / 10
		expandH := imgHeight / 10
		rect = image.Rect(
			centerX-expandW/2,
			centerY-expandH/2,
			centerX+expandW/2,
			centerY+expandH/2,
		)
	}

	// Clamp to image bounds
	rect = rect.Intersect(image.Rect(0, 0, imgWidth, imgHeight))

	// Explicit focal points get very high weight (5.0) to override
	// the CropCentered preference and actually pull the crop toward them.
	// This is intentionally much higher than auto-detected faces (0.3-1.0)
	// because the user explicitly requested this focal point.
	return BoostRegion{
		Rect:   rect,
		Weight: 5.0,
	}
}

// resizeForAnalysis resizes an image to fit within maxSize on longest side.
// Returns the resized image and the scale factor (resized/original).
func resizeForAnalysis(img image.Image, maxSize int) (image.Image, float64) {
	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// If image is already small enough, return as-is
	if srcWidth <= maxSize && srcHeight <= maxSize {
		return img, 1.0
	}

	var newWidth, newHeight int
	if srcWidth > srcHeight {
		newWidth = maxSize
		newHeight = int(float64(srcHeight) * float64(maxSize) / float64(srcWidth))
	} else {
		newHeight = maxSize
		newWidth = int(float64(srcWidth) * float64(maxSize) / float64(srcHeight))
	}

	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}

	resized := imaging.Resize(img, newWidth, newHeight, imaging.Linear)
	scale := float64(newWidth) / float64(srcWidth)

	return resized, scale
}

// scaleBoosts scales boost regions from original coordinates to analysis coordinates.
func scaleBoosts(boosts []BoostRegion, scale float64) []BoostRegion {
	if len(boosts) == 0 || scale == 1.0 {
		return boosts
	}

	scaled := make([]BoostRegion, len(boosts))
	for i, b := range boosts {
		scaled[i] = BoostRegion{
			Rect: image.Rect(
				int(float64(b.Rect.Min.X)*scale),
				int(float64(b.Rect.Min.Y)*scale),
				int(float64(b.Rect.Max.X)*scale),
				int(float64(b.Rect.Max.Y)*scale),
			),
			Weight: b.Weight,
		}
	}
	return scaled
}

// findEnergyCentroid finds the center of high-energy regions in the Sobel map.
// This is used as a fallback when no faces are detected, to find "interesting"
// regions like isolated objects, boundaries, or anomalies.
// Returns a boost region centered on the energy centroid with moderate weight.
func findEnergyCentroid(sobelMap [][]float64, analysisScale float64, origWidth, origHeight int) BoostRegion {
	height := len(sobelMap)
	if height == 0 {
		return BoostRegion{}
	}
	width := len(sobelMap[0])
	if width == 0 {
		return BoostRegion{}
	}

	// Calculate weighted centroid of energy
	var totalEnergy, weightedX, weightedY float64

	// Find max energy for thresholding
	var maxEnergy float64
	for y := range height {
		for x := range width {
			if sobelMap[y][x] > maxEnergy {
				maxEnergy = sobelMap[y][x]
			}
		}
	}

	if maxEnergy == 0 {
		return BoostRegion{} // Uniform image
	}

	// Use a threshold to focus on high-energy regions (top 30% of energy)
	threshold := maxEnergy * 0.3

	for y := range height {
		for x := range width {
			energy := sobelMap[y][x]
			if energy > threshold {
				// Use squared energy to emphasize peaks
				weight := energy * energy
				totalEnergy += weight
				weightedX += float64(x) * weight
				weightedY += float64(y) * weight
			}
		}
	}

	if totalEnergy == 0 {
		return BoostRegion{} // No significant energy found
	}

	// Calculate centroid in analysis coordinates
	centroidX := weightedX / totalEnergy
	centroidY := weightedY / totalEnergy

	// Convert back to original image coordinates
	origCentroidX := centroidX / analysisScale
	origCentroidY := centroidY / analysisScale

	// Create a boost region around the centroid
	// Size is 15% of image dimensions - enough to influence crop but not dominate
	regionW := origWidth / 7
	regionH := origHeight / 7

	rect := image.Rect(
		int(origCentroidX)-regionW/2,
		int(origCentroidY)-regionH/2,
		int(origCentroidX)+regionW/2,
		int(origCentroidY)+regionH/2,
	)

	// Clamp to image bounds
	rect = rect.Intersect(image.Rect(0, 0, origWidth, origHeight))

	if rect.Empty() {
		return BoostRegion{}
	}

	// Use moderate weight - less than explicit focal (5.0) but enough to influence
	// Lower weight than faces so it's a gentle pull, not a hard constraint
	return BoostRegion{
		Rect:   rect,
		Weight: 1.5,
	}
}

// generateCandidates creates a grid of candidate crop rectangles at the target aspect ratio.
// Always includes the center crop position at each scale to ensure centered crops are considered.
func generateCandidates(bounds image.Rectangle, targetAspect float64) []image.Rectangle {
	width := bounds.Dx()
	height := bounds.Dy()

	if width <= 0 || height <= 0 {
		return nil
	}

	var candidates []image.Rectangle

	// Scale factors to try (1.0 = fill image, smaller = smaller crops)
	scales := []float64{1.0, 0.9, 0.8, 0.7, 0.6, 0.5}

	// Step size for sliding window (as fraction of image size)
	// Use 5% steps for finer granularity near center
	stepFraction := 0.05

	for _, scaleFactor := range scales {
		// Calculate crop dimensions at this scale
		var cropW, cropH int

		imgAspect := float64(width) / float64(height)
		if targetAspect > imgAspect {
			// Target is wider - constrain by width
			cropW = int(float64(width) * scaleFactor)
			cropH = int(float64(cropW) / targetAspect)
		} else {
			// Target is taller - constrain by height
			cropH = int(float64(height) * scaleFactor)
			cropW = int(float64(cropH) * targetAspect)
		}

		// Ensure crop fits within image
		if cropW > width {
			cropW = width
			cropH = int(float64(cropW) / targetAspect)
		}
		if cropH > height {
			cropH = height
			cropW = int(float64(cropH) * targetAspect)
		}

		if cropW <= 0 || cropH <= 0 {
			continue
		}

		// Always add the center crop at this scale first
		// This ensures the center position is always considered
		centerX := bounds.Min.X + (width-cropW)/2
		centerY := bounds.Min.Y + (height-cropH)/2
		candidates = append(candidates, image.Rect(centerX, centerY, centerX+cropW, centerY+cropH))

		// Step size in pixels
		stepX := max(1, int(float64(width)*stepFraction))
		stepY := max(1, int(float64(height)*stepFraction))

		// Generate grid candidates at this scale
		for y := bounds.Min.Y; y+cropH <= bounds.Max.Y; y += stepY {
			for x := bounds.Min.X; x+cropW <= bounds.Max.X; x += stepX {
				// Skip if this is very close to the center (already added)
				if abs(x-centerX) < stepX/2 && abs(y-centerY) < stepY/2 {
					continue
				}
				candidates = append(candidates, image.Rect(x, y, x+cropW, y+cropH))
			}
		}
	}

	return candidates
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// centerCrop returns a center crop at the target aspect ratio.
func centerCrop(bounds image.Rectangle, targetAspect float64) image.Rectangle {
	width := bounds.Dx()
	height := bounds.Dy()

	var cropW, cropH int
	imgAspect := float64(width) / float64(height)

	if targetAspect > imgAspect {
		// Target is wider - use full width
		cropW = width
		cropH = int(float64(cropW) / targetAspect)
	} else {
		// Target is taller - use full height
		cropH = height
		cropW = int(float64(cropH) * targetAspect)
	}

	// Center the crop
	x := bounds.Min.X + (width-cropW)/2
	y := bounds.Min.Y + (height-cropH)/2

	return image.Rect(x, y, x+cropW, y+cropH)
}

// scaleRectToOriginal maps a rectangle from analysis coordinates to original image coordinates.
func scaleRectToOriginal(rect image.Rectangle, scale float64, originalBounds image.Rectangle) image.Rectangle {
	var result image.Rectangle

	if scale == 1.0 {
		result = rect
	} else {
		// Scale up to original coordinates
		result = image.Rect(
			int(float64(rect.Min.X)/scale),
			int(float64(rect.Min.Y)/scale),
			int(float64(rect.Max.X)/scale),
			int(float64(rect.Max.Y)/scale),
		)
	}

	// Always clamp to original bounds
	return result.Intersect(originalBounds)
}

// shouldFallbackToCenter checks if all candidate scores are too similar (uniform image).
// If scores vary by less than the threshold, fall back to center crop.
// This prevents the algorithm from making arbitrary choices when there's no clear winner.
func shouldFallbackToCenter(candidates []image.Rectangle, img image.Image, sobelMap [][]float64, boosts []BoostRegion, bestScore float64) bool {
	if len(candidates) < 2 {
		return false
	}

	// Sample a few candidates to check score variance
	sampleSize := min(10, len(candidates))
	var minScore, maxScore = bestScore, bestScore

	step := max(len(candidates)/sampleSize, 1)

	for i := 0; i < len(candidates); i += step {
		score := ScoreCandidate(img, sobelMap, candidates[i], boosts)
		if score < minScore {
			minScore = score
		}
		if score > maxScore {
			maxScore = score
		}
	}

	// If all scores are within threshold of each other, fall back to center
	if maxScore == 0 {
		return true
	}

	variance := (maxScore - minScore) / maxScore

	// Use a higher threshold (5%) when we have boost regions (faces/focal)
	// because face detection can be unreliable and we don't want to make
	// arbitrary choices based on weak signals
	if len(boosts) > 0 {
		return variance < 0.05
	}

	// For images without boost regions, use 3% threshold
	return variance < 0.03
}
