package smartcrop

import (
	"image"
	"math"
	"sort"
)

// Detection represents a single face detection result.
type Detection struct {
	Row   int     // Y center of detected face
	Col   int     // X center of detected face
	Scale int     // Size of detected face (diameter)
	Score float32 // Confidence score (higher = more confident)
}

// Rect returns the bounding rectangle for this detection.
func (d Detection) Rect() image.Rectangle {
	halfSize := d.Scale / 2
	return image.Rect(
		d.Col-halfSize,
		d.Row-halfSize,
		d.Col+halfSize,
		d.Row+halfSize,
	)
}

// DetectParams holds parameters for the PICO face detector.
type DetectParams struct {
	MinSize     int     // Minimum face size in pixels (default: 30)
	MaxSize     int     // Maximum face size in pixels (0 = image size)
	ShiftFactor float64 // Step size as fraction of current scale (default: 0.1)
	ScaleFactor float64 // Scale step multiplier (default: 1.1)
}

// DefaultDetectParams returns the default detection parameters.
func DefaultDetectParams() DetectParams {
	return DetectParams{
		MinSize:     30,
		MaxSize:     0, // Will be set to image size
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,
	}
}

// DetectFaces runs PICO face detection on a grayscale image.
// The pixels array should be row-major grayscale (Y luminance) values.
// Returns clustered detections after non-maximum suppression.
func DetectFaces(pixels []uint8, width, height int) []Detection {
	return DetectFacesWithParams(pixels, width, height, DefaultDetectParams())
}

// DetectFacesWithParams runs PICO face detection with custom parameters.
func DetectFacesWithParams(pixels []uint8, width, height int, params DetectParams) []Detection {
	cascade, err := DefaultCascade()
	if err != nil {
		return nil
	}

	maxSize := params.MaxSize
	if maxSize == 0 || maxSize > min(width, height) {
		maxSize = min(width, height)
	}

	treeDepth := pow(2, int(cascade.treeDepth))
	var detections []Detection

	// Multi-scale sliding window
	scale := params.MinSize
	for scale <= maxSize {
		step := int(math.Max(params.ShiftFactor*float64(scale), 1))
		offset := scale/2 + 1

		// Slide window across image
		for row := offset; row <= height-offset; row += step {
			for col := offset; col <= width-offset; col += step {
				score := classifyRegion(cascade, row, col, scale, treeDepth, pixels, width)
				if score > 0 {
					detections = append(detections, Detection{
						Row:   row,
						Col:   col,
						Scale: scale,
						Score: score,
					})
				}
			}
		}

		// Scale up - avoid infinite loop for small scales
		newScale := int(float64(scale) + math.Max(2, float64(scale)*params.ScaleFactor-float64(scale)))
		scale = newScale
	}

	// Cluster overlapping detections
	return clusterDetections(detections, 0.3)
}

// DetectFacesImage is a convenience wrapper that takes an image.Image.
func DetectFacesImage(img image.Image) []Detection {
	pixels, width, height := ToGrayscaleFlat(img)
	return DetectFaces(pixels, width, height)
}

// classifyRegion runs the PICO cascade classifier on a single region.
// Returns a positive score if a face is detected, negative otherwise.
//
// This implementation follows Pigo's classifyRegion algorithm:
// 1. For each tree in the cascade, walk from root to leaf
// 2. At each node, compare pixel intensities at encoded positions
// 3. Accumulate predictions from leaf nodes
// 4. Early-reject regions that fall below threshold after each tree
func classifyRegion(cascade *Cascade, r, c, s, treeDepth int, pixels []uint8, dim int) float32 {
	if cascade == nil || cascade.treeNum == 0 {
		return -1
	}

	var out float32
	root := 0

	// Scale row and column by 256 for fixed-point arithmetic
	r256 := r * 256
	c256 := c * 256

	for t := 0; t < int(cascade.treeNum); t++ {
		idx := 1

		// Walk down the tree
		for j := 0; j < int(cascade.treeDepth); j++ {
			// Get pixel comparison positions from codes
			// Each node uses 4 int8 codes for two pixel positions
			codeIdx := root + 4*idx

			// Pixel positions for the two comparison points. Each is a linear
			// index row*dim + col, where row = (r*256 + code*s) >> 8 and
			// col = (c*256 + code*s) >> 8 (Pigo's fixed-point tree encoding).
			r1 := (r256 + int(cascade.treeCodes[codeIdx+0])*s) >> 8
			c1 := (c256 + int(cascade.treeCodes[codeIdx+1])*s) >> 8
			r2 := (r256 + int(cascade.treeCodes[codeIdx+2])*s) >> 8
			c2 := (c256 + int(cascade.treeCodes[codeIdx+3])*s) >> 8

			// Clamp to image bounds
			r1 = clampInt(r1, 0, len(pixels)/dim-1)
			c1 = clampInt(c1, 0, dim-1)
			r2 = clampInt(r2, 0, len(pixels)/dim-1)
			c2 = clampInt(c2, 0, dim-1)

			// Get pixel indices
			x1 := r1*dim + c1
			x2 := r2*dim + c2

			// Bounds check
			if x1 < 0 || x1 >= len(pixels) || x2 < 0 || x2 >= len(pixels) {
				return -1
			}

			// Binary test: if pixel1 <= pixel2, go left (add 1), else go right (add 0)
			if pixels[x1] <= pixels[x2] {
				idx = 2*idx + 1
			} else {
				idx = 2*idx + 0
			}
		}

		// Get leaf prediction
		// The prediction index is: treeDepth*t + idx - treeDepth
		predIdx := treeDepth*t + idx - treeDepth
		if predIdx < 0 || predIdx >= len(cascade.treePred) {
			return -1
		}

		out += cascade.treePred[predIdx]

		// Early rejection
		if out <= cascade.treeThreshold[t] {
			return -1
		}

		// Move root pointer to next tree
		// Each tree has 4*treeDepth codes (including the leading zeros)
		root += 4 * treeDepth
	}

	return out - cascade.treeThreshold[cascade.treeNum-1]
}

// clampInt restricts a value to the range [minVal, maxVal].
func clampInt(val, minVal, maxVal int) int {
	if val < minVal {
		return minVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}

// clusterDetections merges overlapping detections using non-maximum suppression.
// This implementation follows Pigo's ClusterDetections algorithm.
func clusterDetections(detections []Detection, iouThreshold float64) []Detection {
	if len(detections) == 0 {
		return nil
	}

	// Sort by score ascending (Pigo sorts this way for clustering)
	sort.Slice(detections, func(i, j int) bool {
		return detections[i].Score < detections[j].Score
	})

	assignments := make([]bool, len(detections))
	var clusters []Detection

	for i := 0; i < len(detections); i++ {
		if assignments[i] {
			continue
		}

		var (
			sumR, sumC, sumS, n int
			sumQ                float32
		)

		for j := 0; j < len(detections); j++ {
			if computeIoU(detections[i], detections[j]) > iouThreshold {
				assignments[j] = true
				sumR += detections[j].Row
				sumC += detections[j].Col
				sumS += detections[j].Scale
				sumQ += detections[j].Score
				n++
			}
		}

		if n > 0 {
			clusters = append(clusters, Detection{
				Row:   sumR / n,
				Col:   sumC / n,
				Scale: sumS / n,
				Score: sumQ,
			})
		}
	}

	return clusters
}

// computeIoU calculates the Intersection over Union of two detections.
// This follows Pigo's IoU calculation for circular/square regions.
func computeIoU(a, b Detection) float64 {
	// Unpack positions and sizes
	r1, c1, s1 := float64(a.Row), float64(a.Col), float64(a.Scale)
	r2, c2, s2 := float64(b.Row), float64(b.Col), float64(b.Scale)

	// Calculate overlap in row and column dimensions
	overRow := math.Max(0, math.Min(r1+s1/2, r2+s2/2)-math.Max(r1-s1/2, r2-s2/2))
	overCol := math.Max(0, math.Min(c1+s1/2, c2+s2/2)-math.Max(c1-s1/2, c2-s2/2))

	// IoU = intersection / union
	intersection := overRow * overCol
	union := s1*s1 + s2*s2 - intersection

	if union <= 0 {
		return 0
	}

	return intersection / union
}
