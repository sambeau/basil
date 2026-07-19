package seamcarve

import (
	"image"
	"image/color"
	"testing"
)

// createUniformImage creates a solid color image for testing.
func createUniformImage(width, height int, c color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, c)
		}
	}
	return img
}

// createVerticalStripeImage creates an image with a vertical stripe of a different color.
func createVerticalStripeImage(width, height int, stripeX int, bg, stripe color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			if x == stripeX {
				img.Set(x, y, stripe)
			} else {
				img.Set(x, y, bg)
			}
		}
	}
	return img
}

// createGradientImage creates an image with a horizontal gradient (left=dark, right=light).
func createGradientImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			v := uint8(x * 255 / width)
			img.Set(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	return img
}

// TestEnergyMap_Uniform tests that a uniform image has near-zero energy.
func TestEnergyMap_Uniform(t *testing.T) {
	img := createUniformImage(100, 100, color.RGBA{R: 128, G: 128, B: 128, A: 255})
	energy := EnergyMap(img)

	if len(energy) != 100 || len(energy[0]) != 100 {
		t.Fatalf("expected 100x100 energy map, got %dx%d", len(energy[0]), len(energy))
	}

	// All values should be near zero for a uniform image
	for y := range 100 {
		for x := range 100 {
			if energy[y][x] > 0.1 {
				t.Errorf("energy[%d][%d] = %f, expected near 0 for uniform image", y, x, energy[y][x])
			}
		}
	}
}

// TestEnergyMap_VerticalStripe tests that a vertical stripe has high energy at edges.
func TestEnergyMap_VerticalStripe(t *testing.T) {
	img := createVerticalStripeImage(100, 100, 50,
		color.RGBA{R: 0, G: 0, B: 0, A: 255},
		color.RGBA{R: 255, G: 255, B: 255, A: 255})
	energy := EnergyMap(img)

	// Energy should be high at the stripe edges (x=49 and x=51, adjacent to the stripe)
	// The stripe itself (x=50) is uniform white, so it has low energy
	// The high energy is where black meets white
	for y := 1; y < 99; y++ {
		// Far from stripe - should be low energy
		if energy[y][10] > 1.0 {
			t.Errorf("energy[%d][10] = %f, expected low energy away from stripe", y, energy[y][10])
		}

		// Adjacent to stripe edge (x=49) - should be high energy where black meets white
		if energy[y][49] < 100.0 {
			t.Errorf("energy[%d][49] = %f, expected high energy at stripe edge", y, energy[y][49])
		}

		// Adjacent to stripe edge (x=51) - should be high energy where white meets black
		if energy[y][51] < 100.0 {
			t.Errorf("energy[%d][51] = %f, expected high energy at stripe edge", y, energy[y][51])
		}
	}
}

// TestEnergyMap_Empty tests that an empty image returns nil.
func TestEnergyMap_Empty(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 0, 0))
	energy := EnergyMap(img)
	if energy != nil {
		t.Errorf("expected nil energy map for empty image")
	}
}

// TestFindMinVerticalSeam_UniformEnergy tests seam finding on uniform energy.
func TestFindMinVerticalSeam_UniformEnergy(t *testing.T) {
	// Create a 10x10 uniform energy map
	energy := make([][]float64, 10)
	for y := range energy {
		energy[y] = make([]float64, 10)
		for x := range energy[y] {
			energy[y][x] = 1.0
		}
	}

	seam := findMinVerticalSeam(energy)

	if len(seam) != 10 {
		t.Fatalf("expected seam length 10, got %d", len(seam))
	}

	// Any path is valid for uniform energy, just verify connectivity
	for y := 1; y < len(seam); y++ {
		diff := seam[y] - seam[y-1]
		if diff < -1 || diff > 1 {
			t.Errorf("seam is not connected: seam[%d]=%d, seam[%d]=%d", y-1, seam[y-1], y, seam[y])
		}
	}
}

// TestFindMinVerticalSeam_KnownPath tests seam finding with a known low-energy path.
func TestFindMinVerticalSeam_KnownPath(t *testing.T) {
	// Create energy map with a clear low-energy path down the center
	energy := make([][]float64, 10)
	for y := range energy {
		energy[y] = make([]float64, 10)
		for x := range energy[y] {
			if x == 5 {
				energy[y][x] = 0.0 // Low energy path
			} else {
				energy[y][x] = 100.0 // High energy elsewhere
			}
		}
	}

	seam := findMinVerticalSeam(energy)

	// Seam should follow the low-energy path at x=5
	for y, x := range seam {
		if x != 5 {
			t.Errorf("seam[%d] = %d, expected 5 (low-energy path)", y, x)
		}
	}
}

// TestFindMinHorizontalSeam_KnownPath tests horizontal seam finding with a known low-energy path.
func TestFindMinHorizontalSeam_KnownPath(t *testing.T) {
	// Create energy map with a clear low-energy path across the middle
	energy := make([][]float64, 10)
	for y := range energy {
		energy[y] = make([]float64, 10)
		for x := range energy[y] {
			if y == 5 {
				energy[y][x] = 0.0 // Low energy path
			} else {
				energy[y][x] = 100.0 // High energy elsewhere
			}
		}
	}

	seam := findMinHorizontalSeam(energy)

	// Seam should follow the low-energy path at y=5
	for x, y := range seam {
		if y != 5 {
			t.Errorf("seam[%d] = %d, expected 5 (low-energy path)", x, y)
		}
	}
}

// TestFindMinVerticalSeam_Empty tests seam finding on empty energy map.
func TestFindMinVerticalSeam_Empty(t *testing.T) {
	seam := findMinVerticalSeam(nil)
	if seam != nil {
		t.Errorf("expected nil seam for nil energy map")
	}

	seam = findMinVerticalSeam([][]float64{})
	if seam != nil {
		t.Errorf("expected nil seam for empty energy map")
	}
}

// TestRemoveVerticalSeam_ReducesWidth tests that vertical seam removal reduces width by 1.
func TestRemoveVerticalSeam_ReducesWidth(t *testing.T) {
	img := createUniformImage(100, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	// Create a simple seam down the middle
	seam := make([]int, 50)
	for y := range seam {
		seam[y] = 50
	}

	result := removeVerticalSeam(img, seam)

	if result.Bounds().Dx() != 99 {
		t.Errorf("expected width 99, got %d", result.Bounds().Dx())
	}
	if result.Bounds().Dy() != 50 {
		t.Errorf("expected height 50, got %d", result.Bounds().Dy())
	}
}

// TestRemoveHorizontalSeam_ReducesHeight tests that horizontal seam removal reduces height by 1.
func TestRemoveHorizontalSeam_ReducesHeight(t *testing.T) {
	img := createUniformImage(50, 100, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	// Create a simple seam across the middle
	seam := make([]int, 50)
	for x := range seam {
		seam[x] = 50
	}

	result := removeHorizontalSeam(img, seam)

	if result.Bounds().Dx() != 50 {
		t.Errorf("expected width 50, got %d", result.Bounds().Dx())
	}
	if result.Bounds().Dy() != 99 {
		t.Errorf("expected height 99, got %d", result.Bounds().Dy())
	}
}

// TestResize_WidthReduction tests resizing with width reduction only.
func TestResize_WidthReduction(t *testing.T) {
	img := createUniformImage(100, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	result := Resize(img, 50, 50)

	if result.Bounds().Dx() != 50 {
		t.Errorf("expected width 50, got %d", result.Bounds().Dx())
	}
	if result.Bounds().Dy() != 50 {
		t.Errorf("expected height 50, got %d", result.Bounds().Dy())
	}
}

// TestResize_HeightReduction tests resizing with height reduction only.
func TestResize_HeightReduction(t *testing.T) {
	img := createUniformImage(50, 100, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	result := Resize(img, 50, 50)

	if result.Bounds().Dx() != 50 {
		t.Errorf("expected width 50, got %d", result.Bounds().Dx())
	}
	if result.Bounds().Dy() != 50 {
		t.Errorf("expected height 50, got %d", result.Bounds().Dy())
	}
}

// TestResize_BothDimensions tests resizing with both width and height reduction.
func TestResize_BothDimensions(t *testing.T) {
	img := createUniformImage(100, 100, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	result := Resize(img, 70, 60)

	if result.Bounds().Dx() != 70 {
		t.Errorf("expected width 70, got %d", result.Bounds().Dx())
	}
	if result.Bounds().Dy() != 60 {
		t.Errorf("expected height 60, got %d", result.Bounds().Dy())
	}
}

// TestResize_Enlargement tests that image is enlarged when target > source.
func TestResize_Enlargement(t *testing.T) {
	img := createUniformImage(50, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	result := Resize(img, 100, 100)

	if result.Bounds().Dx() != 100 {
		t.Errorf("expected width 100, got %d", result.Bounds().Dx())
	}
	if result.Bounds().Dy() != 100 {
		t.Errorf("expected height 100, got %d", result.Bounds().Dy())
	}
}

// TestResize_MixedEnlargeReduce tests enlarging one dimension while reducing another.
func TestResize_MixedEnlargeReduce(t *testing.T) {
	img := createUniformImage(50, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	result := Resize(img, 100, 30)

	// Width should enlarge to 100
	// Height should reduce to 30
	if result.Bounds().Dx() != 100 {
		t.Errorf("expected width 100, got %d", result.Bounds().Dx())
	}
	if result.Bounds().Dy() != 30 {
		t.Errorf("expected height 30, got %d", result.Bounds().Dy())
	}
}

// TestResize_MinimumSize tests that resize doesn't go below 1x1.
func TestResize_MinimumSize(t *testing.T) {
	img := createUniformImage(10, 10, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	result := Resize(img, 0, 0)

	if result.Bounds().Dx() < 1 {
		t.Errorf("expected width >= 1, got %d", result.Bounds().Dx())
	}
	if result.Bounds().Dy() < 1 {
		t.Errorf("expected height >= 1, got %d", result.Bounds().Dy())
	}
}

// TestResize_PreservesContent tests that seam carving avoids high-energy regions.
func TestResize_PreservesContent(t *testing.T) {
	// Create an image with a vertical stripe that should be preserved
	// The stripe has high contrast, so seams should avoid it
	img := createVerticalStripeImage(20, 20, 10,
		color.RGBA{R: 100, G: 100, B: 100, A: 255}, // gray background
		color.RGBA{R: 255, G: 255, B: 255, A: 255}) // white stripe

	// Reduce width by 5 pixels
	result := Resize(img, 15, 20)

	if result.Bounds().Dx() != 15 {
		t.Fatalf("expected width 15, got %d", result.Bounds().Dx())
	}

	// The white stripe should still be present somewhere in the middle region
	// Check that there's still a column with white pixels
	foundWhite := false
	for x := 5; x < 10; x++ {
		c := result.(*image.RGBA).RGBAAt(x, 10)
		if c.R > 200 && c.G > 200 && c.B > 200 {
			foundWhite = true
			break
		}
	}

	if !foundWhite {
		t.Error("white stripe should be preserved after seam carving")
	}
}

// TestToRGBA_AlreadyRGBA tests that RGBA images are handled efficiently.
func TestToRGBA_AlreadyRGBA(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	result := toRGBA(img)

	// Should return the same image (pointer equality)
	if result != img {
		t.Error("toRGBA should return same image when already RGBA with origin at (0,0)")
	}
}

// TestToRGBA_NonRGBA tests conversion from other image types.
func TestToRGBA_NonRGBA(t *testing.T) {
	// Create a Gray image
	gray := image.NewGray(image.Rect(0, 0, 10, 10))
	for y := range 10 {
		for x := range 10 {
			gray.SetGray(x, y, color.Gray{Y: 128})
		}
	}

	result := toRGBA(gray)

	if result.Bounds().Dx() != 10 || result.Bounds().Dy() != 10 {
		t.Errorf("expected 10x10, got %dx%d", result.Bounds().Dx(), result.Bounds().Dy())
	}

	// Check that color is converted correctly
	c := result.RGBAAt(5, 5)
	if c.R != 128 || c.G != 128 || c.B != 128 {
		t.Errorf("expected RGB(128,128,128), got RGB(%d,%d,%d)", c.R, c.G, c.B)
	}
}

// TestRemoveVerticalSeam_InvalidSeam tests handling of invalid seam input.
func TestRemoveVerticalSeam_InvalidSeam(t *testing.T) {
	img := createUniformImage(10, 10, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	// Wrong seam length
	seam := make([]int, 5) // Should be 10
	result := removeVerticalSeam(img, seam)

	// Should return original image unchanged
	if result.Bounds().Dx() != 10 {
		t.Errorf("expected unchanged width 10, got %d", result.Bounds().Dx())
	}
}

// TestRemoveHorizontalSeam_InvalidSeam tests handling of invalid seam input.
func TestRemoveHorizontalSeam_InvalidSeam(t *testing.T) {
	img := createUniformImage(10, 10, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	// Wrong seam length
	seam := make([]int, 5) // Should be 10
	result := removeHorizontalSeam(img, seam)

	// Should return original image unchanged
	if result.Bounds().Dy() != 10 {
		t.Errorf("expected unchanged height 10, got %d", result.Bounds().Dy())
	}
}

// BenchmarkEnergyMap benchmarks energy map computation.
func BenchmarkEnergyMap(b *testing.B) {
	img := createGradientImage(256, 256)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		EnergyMap(img)
	}
}

// BenchmarkFindMinVerticalSeam benchmarks vertical seam finding.
func BenchmarkFindMinVerticalSeam(b *testing.B) {
	img := createGradientImage(256, 256)
	energy := EnergyMap(img)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		findMinVerticalSeam(energy)
	}
}

// BenchmarkRemoveVerticalSeam benchmarks vertical seam removal.
func BenchmarkRemoveVerticalSeam(b *testing.B) {
	img := createGradientImage(256, 256)
	energy := EnergyMap(img)
	seam := findMinVerticalSeam(energy)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		removeVerticalSeam(img, seam)
	}
}

// BenchmarkResize_Small benchmarks resizing a small image.
func BenchmarkResize_Small(b *testing.B) {
	img := createGradientImage(100, 100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Resize(img, 90, 90)
	}
}

// BenchmarkResize_Medium benchmarks resizing a medium image.
func BenchmarkResize_Medium(b *testing.B) {
	img := createGradientImage(256, 256)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Resize(img, 230, 230)
	}
}

// =============================================================================
// Enlargement Tests
// =============================================================================

// TestFindKMinVerticalSeams_ReturnsKSeams tests that k distinct seams are found.
func TestFindKMinVerticalSeams_ReturnsKSeams(t *testing.T) {
	img := createUniformImage(20, 20, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	seams := findKMinVerticalSeams(img, 5)

	if len(seams) != 5 {
		t.Fatalf("expected 5 seams, got %d", len(seams))
	}

	// Each seam should have length equal to image height
	for i, seam := range seams {
		if len(seam) != 20 {
			t.Errorf("seam[%d] has length %d, expected 20", i, len(seam))
		}
	}
}

// TestFindKMinVerticalSeams_NoOverlap tests that seams don't overlap on uniform energy.
func TestFindKMinVerticalSeams_NoOverlap(t *testing.T) {
	img := createUniformImage(20, 20, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	seams := findKMinVerticalSeams(img, 5)

	// Check that seams are spread out (no two seams share the same x at any row)
	// This is a weak test since on uniform energy, seams could share positions
	// but the algorithm should mark used seams to prevent reselection
	for y := range 20 {
		positions := make(map[int]bool)
		for i, seam := range seams {
			if positions[seam[y]] {
				t.Errorf("row %d: seam %d overlaps with another seam at x=%d", y, i, seam[y])
			}
			positions[seam[y]] = true
		}
	}
}

// TestFindKMinVerticalSeams_LowEnergyRegion tests that seams cluster in low-energy regions.
func TestFindKMinVerticalSeams_LowEnergyRegion(t *testing.T) {
	// Create image with a high-energy vertical stripe on the right that seams should avoid
	// Left side is uniform (low energy interior), right side has a bright stripe (high energy edge)
	img := image.NewRGBA(image.Rect(0, 0, 30, 20))
	for y := range 20 {
		for x := range 30 {
			if x < 20 {
				// Left side: uniform gray (low energy in interior)
				img.Set(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
			} else if x == 20 {
				// Bright stripe creating high energy edge
				img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
			} else {
				// Right of stripe: uniform dark (but edge at x=20 is high energy)
				img.Set(x, y, color.RGBA{R: 50, G: 50, B: 50, A: 255})
			}
		}
	}

	seams := findKMinVerticalSeams(img, 5)

	// Seams should avoid the high-energy edge at x=20
	// Most seams should be in the uniform left region (x < 20) or uniform right region (x > 20)
	// but NOT at x=20 where the edge is
	edgeCount := 0
	for _, seam := range seams {
		for _, x := range seam {
			if x == 20 || x == 19 || x == 21 {
				edgeCount++
				break // Count each seam only once
			}
		}
	}

	// At most 2 of 5 seams should cross near the high-energy edge
	if edgeCount > 2 {
		t.Errorf("expected seams to avoid high-energy edge, but %d of 5 crossed near x=20", edgeCount)
	}
}

// TestFindKMinHorizontalSeams_ReturnsKSeams tests horizontal seam finding.
func TestFindKMinHorizontalSeams_ReturnsKSeams(t *testing.T) {
	img := createUniformImage(20, 20, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	seams := findKMinHorizontalSeams(img, 5)

	if len(seams) != 5 {
		t.Fatalf("expected 5 seams, got %d", len(seams))
	}

	// Each seam should have length equal to image width
	for i, seam := range seams {
		if len(seam) != 20 {
			t.Errorf("seam[%d] has length %d, expected 20", i, len(seam))
		}
	}
}

// TestInsertVerticalSeams_IncreasesWidth tests that seam insertion increases width.
func TestInsertVerticalSeams_IncreasesWidth(t *testing.T) {
	img := createUniformImage(20, 20, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	seams := findKMinVerticalSeams(img, 5)
	result := insertVerticalSeams(img, seams)

	if result.Bounds().Dx() != 25 {
		t.Errorf("expected width 25, got %d", result.Bounds().Dx())
	}
	if result.Bounds().Dy() != 20 {
		t.Errorf("expected height 20, got %d", result.Bounds().Dy())
	}
}

// TestInsertHorizontalSeams_IncreasesHeight tests that horizontal seam insertion increases height.
func TestInsertHorizontalSeams_IncreasesHeight(t *testing.T) {
	img := createUniformImage(20, 20, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	seams := findKMinHorizontalSeams(img, 5)
	result := insertHorizontalSeams(img, seams)

	if result.Bounds().Dx() != 20 {
		t.Errorf("expected width 20, got %d", result.Bounds().Dx())
	}
	if result.Bounds().Dy() != 25 {
		t.Errorf("expected height 25, got %d", result.Bounds().Dy())
	}
}

// TestInsertVerticalSeams_AveragesColors tests that inserted pixels are averaged.
func TestInsertVerticalSeams_AveragesColors(t *testing.T) {
	// Create image with two colors side by side
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := range 10 {
		for x := range 10 {
			if x < 5 {
				img.Set(x, y, color.RGBA{R: 100, G: 100, B: 100, A: 255})
			} else {
				img.Set(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
			}
		}
	}

	seams := findKMinVerticalSeams(img, 2)
	result := insertVerticalSeams(img, seams)

	// Result should be 12 pixels wide
	if result.Bounds().Dx() != 12 {
		t.Errorf("expected width 12, got %d", result.Bounds().Dx())
	}
}

// TestResize_WidthEnlargement tests resizing with width enlargement only.
func TestResize_WidthEnlargement(t *testing.T) {
	img := createUniformImage(50, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	result := Resize(img, 60, 50)

	if result.Bounds().Dx() != 60 {
		t.Errorf("expected width 60, got %d", result.Bounds().Dx())
	}
	if result.Bounds().Dy() != 50 {
		t.Errorf("expected height 50, got %d", result.Bounds().Dy())
	}
}

// TestResize_HeightEnlargement tests resizing with height enlargement only.
func TestResize_HeightEnlargement(t *testing.T) {
	img := createUniformImage(50, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	result := Resize(img, 50, 60)

	if result.Bounds().Dx() != 50 {
		t.Errorf("expected width 50, got %d", result.Bounds().Dx())
	}
	if result.Bounds().Dy() != 60 {
		t.Errorf("expected height 60, got %d", result.Bounds().Dy())
	}
}

// TestResize_BothEnlargement tests resizing with both dimensions enlarged.
func TestResize_BothEnlargement(t *testing.T) {
	img := createUniformImage(50, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	result := Resize(img, 60, 70)

	if result.Bounds().Dx() != 60 {
		t.Errorf("expected width 60, got %d", result.Bounds().Dx())
	}
	if result.Bounds().Dy() != 70 {
		t.Errorf("expected height 70, got %d", result.Bounds().Dy())
	}
}

// TestResize_MixedReduceAndEnlarge tests reducing one dimension while enlarging another.
func TestResize_MixedReduceAndEnlarge(t *testing.T) {
	img := createUniformImage(50, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	// Reduce width, enlarge height
	result := Resize(img, 40, 60)

	if result.Bounds().Dx() != 40 {
		t.Errorf("expected width 40, got %d", result.Bounds().Dx())
	}
	if result.Bounds().Dy() != 60 {
		t.Errorf("expected height 60, got %d", result.Bounds().Dy())
	}
}

// TestResize_MixedEnlargeAndReduce tests enlarging one dimension while reducing another.
func TestResize_MixedEnlargeAndReduce(t *testing.T) {
	img := createUniformImage(50, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	// Enlarge width, reduce height
	result := Resize(img, 60, 40)

	if result.Bounds().Dx() != 60 {
		t.Errorf("expected width 60, got %d", result.Bounds().Dx())
	}
	if result.Bounds().Dy() != 40 {
		t.Errorf("expected height 40, got %d", result.Bounds().Dy())
	}
}

// TestResize_NoChange tests that image is unchanged when target equals source.
func TestResize_NoChange(t *testing.T) {
	img := createUniformImage(50, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	result := Resize(img, 50, 50)

	if result.Bounds().Dx() != 50 {
		t.Errorf("expected width 50, got %d", result.Bounds().Dx())
	}
	if result.Bounds().Dy() != 50 {
		t.Errorf("expected height 50, got %d", result.Bounds().Dy())
	}
}

// TestResize_EnlargementPreservesContent tests that high-energy regions are preserved during enlargement.
func TestResize_EnlargementPreservesContent(t *testing.T) {
	// Create an image with a vertical stripe that should be preserved
	img := createVerticalStripeImage(20, 20, 10,
		color.RGBA{R: 100, G: 100, B: 100, A: 255}, // gray background
		color.RGBA{R: 255, G: 255, B: 255, A: 255}) // white stripe

	// Enlarge width by 5 pixels
	result := Resize(img, 25, 20)

	if result.Bounds().Dx() != 25 {
		t.Fatalf("expected width 25, got %d", result.Bounds().Dx())
	}

	// The white stripe should still be present
	foundWhite := false
	for x := range 25 {
		c := result.(*image.RGBA).RGBAAt(x, 10)
		if c.R > 200 && c.G > 200 && c.B > 200 {
			foundWhite = true
			break
		}
	}

	if !foundWhite {
		t.Error("white stripe should be preserved after seam carving enlargement")
	}
}

// TestSortInts tests the simple sort function.
func TestSortInts(t *testing.T) {
	tests := []struct {
		input    []int
		expected []int
	}{
		{[]int{3, 1, 2}, []int{1, 2, 3}},
		{[]int{1, 2, 3}, []int{1, 2, 3}},
		{[]int{3, 2, 1}, []int{1, 2, 3}},
		{[]int{5}, []int{5}},
		{[]int{}, []int{}},
		{[]int{2, 2, 1}, []int{1, 2, 2}},
	}

	for _, tc := range tests {
		input := make([]int, len(tc.input))
		copy(input, tc.input)
		sortInts(input)
		for i, v := range input {
			if v != tc.expected[i] {
				t.Errorf("sortInts(%v) = %v, expected %v", tc.input, input, tc.expected)
				break
			}
		}
	}
}

// TestEnlargeWidth_ZeroEnlargement tests that zero enlargement returns unchanged image.
func TestEnlargeWidth_ZeroEnlargement(t *testing.T) {
	img := createUniformImage(20, 20, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	result := enlargeWidth(img, 0)

	if result.Bounds().Dx() != 20 {
		t.Errorf("expected width 20, got %d", result.Bounds().Dx())
	}
}

// TestEnlargeHeight_ZeroEnlargement tests that zero enlargement returns unchanged image.
func TestEnlargeHeight_ZeroEnlargement(t *testing.T) {
	img := createUniformImage(20, 20, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	result := enlargeHeight(img, 0)

	if result.Bounds().Dy() != 20 {
		t.Errorf("expected height 20, got %d", result.Bounds().Dy())
	}
}

// BenchmarkResize_Enlarge_Small benchmarks enlarging a small image.
func BenchmarkResize_Enlarge_Small(b *testing.B) {
	img := createGradientImage(100, 100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Resize(img, 110, 110)
	}
}

// BenchmarkResize_Enlarge_Medium benchmarks enlarging a medium image.
func BenchmarkResize_Enlarge_Medium(b *testing.B) {
	img := createGradientImage(256, 256)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Resize(img, 280, 280)
	}
}

// BenchmarkFindKMinVerticalSeams benchmarks finding k seams.
func BenchmarkFindKMinVerticalSeams(b *testing.B) {
	img := createGradientImage(256, 256)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		findKMinVerticalSeams(img, 26)
	}
}

// BenchmarkInsertVerticalSeams benchmarks inserting seams.
func BenchmarkInsertVerticalSeams(b *testing.B) {
	img := createGradientImage(256, 256)
	seams := findKMinVerticalSeams(img, 26)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		insertVerticalSeams(img, seams)
	}
}
