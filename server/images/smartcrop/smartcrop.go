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
	faceBoosts := detectFaceBoosts(img, srcWidth, srcHeight)
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
func detectFaceBoosts(img image.Image, srcWidth, srcHeight int) []BoostRegion {
	// Resize for face detection
	faceImg, faceScale := resizeForAnalysis(img, faceDetectionSize)

	// Run face detection
	pixels, width, height := ToGrayscaleFlat(faceImg)
	detections := DetectFaces(pixels, width, height)

	if len(detections) == 0 {
		return nil
	}

	// Convert detections to boost regions in original image coordinates
	boosts := make([]BoostRegion, 0, len(detections))
	for _, det := range detections {
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

		if !origRect.Empty() {
			boosts = append(boosts, BoostRegion{
				Rect:   origRect,
				Weight: 1.0,
			})
		}
	}

	return boosts
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

	return BoostRegion{
		Rect:   rect,
		Weight: 1.0,
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

// generateCandidates creates a grid of candidate crop rectangles at the target aspect ratio.
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
	stepFraction := 0.1

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

		// Step size in pixels
		stepX := max(1, int(float64(width)*stepFraction))
		stepY := max(1, int(float64(height)*stepFraction))

		// Generate candidates at this scale
		for y := bounds.Min.Y; y+cropH <= bounds.Max.Y; y += stepY {
			for x := bounds.Min.X; x+cropW <= bounds.Max.X; x += stepX {
				candidates = append(candidates, image.Rect(x, y, x+cropW, y+cropH))
			}
		}
	}

	return candidates
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
// If scores vary by less than 1%, fall back to center crop.
func shouldFallbackToCenter(candidates []image.Rectangle, img image.Image, sobelMap [][]float64, boosts []BoostRegion, bestScore float64) bool {
	if len(candidates) < 2 {
		return false
	}

	// If we have boost regions (faces/focal), trust the scoring
	if len(boosts) > 0 {
		return false
	}

	// Sample a few candidates to check score variance
	sampleSize := min(10, len(candidates))
	var minScore, maxScore float64 = bestScore, bestScore

	step := len(candidates) / sampleSize
	if step < 1 {
		step = 1
	}

	for i := 0; i < len(candidates); i += step {
		score := ScoreCandidate(img, sobelMap, candidates[i], boosts)
		if score < minScore {
			minScore = score
		}
		if score > maxScore {
			maxScore = score
		}
	}

	// If all scores are within 1% of each other, fall back to center
	if maxScore == 0 {
		return true
	}

	variance := (maxScore - minScore) / maxScore
	return variance < 0.01
}
