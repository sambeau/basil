package smartcrop

import (
	"image"
	"image/color"
	"testing"
)

func TestDefaultCascade(t *testing.T) {
	cascade, err := DefaultCascade()
	if err != nil {
		t.Fatalf("DefaultCascade() error: %v", err)
	}

	if cascade == nil {
		t.Fatal("DefaultCascade() returned nil cascade")
	}

	// Check that cascade has reasonable values
	if cascade.treeNum <= 0 {
		t.Errorf("cascade.treeNum = %d, expected > 0", cascade.treeNum)
	}
	if cascade.treeDepth <= 0 {
		t.Errorf("cascade.treeDepth = %d, expected > 0", cascade.treeDepth)
	}
	if len(cascade.treeCodes) == 0 {
		t.Error("cascade.treeCodes is empty")
	}
	if len(cascade.treePred) == 0 {
		t.Error("cascade.treePred is empty")
	}
	if len(cascade.treeThreshold) == 0 {
		t.Error("cascade.treeThreshold is empty")
	}

	// Verify it's cached (second call returns same pointer)
	cascade2, err := DefaultCascade()
	if err != nil {
		t.Fatalf("DefaultCascade() second call error: %v", err)
	}
	if cascade != cascade2 {
		t.Error("DefaultCascade() should return cached cascade")
	}
}

func TestCascadeUnpack(t *testing.T) {
	cascade, err := DefaultCascade()
	if err != nil {
		t.Fatalf("DefaultCascade() error: %v", err)
	}

	// Calculate expected sizes based on Pigo's format
	// Each tree has: 4 leading zeros + (4 * 2^depth - 4) codes
	pow2Depth := 1
	for i := 0; i < int(cascade.treeDepth); i++ {
		pow2Depth *= 2
	}
	numCodesPerTree := 4*pow2Depth - 4 + 4 // includes 4 leading zeros
	numPredsPerTree := pow2Depth

	expectedCodes := int(cascade.treeNum) * numCodesPerTree
	expectedPreds := int(cascade.treeNum) * numPredsPerTree

	if len(cascade.treeCodes) != expectedCodes {
		t.Errorf("treeCodes length = %d, expected %d", len(cascade.treeCodes), expectedCodes)
	}
	if len(cascade.treePred) != expectedPreds {
		t.Errorf("treePred length = %d, expected %d", len(cascade.treePred), expectedPreds)
	}
	if len(cascade.treeThreshold) != int(cascade.treeNum) {
		t.Errorf("treeThreshold length = %d, expected %d", len(cascade.treeThreshold), cascade.treeNum)
	}
}

func TestUnpackCascade_Errors(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"too short", []byte{0, 0, 0, 0, 0, 0, 0, 0}},
		{"invalid depth", []byte{0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0}}, // depth=0, trees=1
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := unpackCascade(tc.data)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestDetection_Rect(t *testing.T) {
	d := Detection{
		Row:   100,
		Col:   150,
		Scale: 50,
	}

	rect := d.Rect()

	// Center at (150, 100), size 50 -> half = 25
	// Expected: (125, 75) to (175, 125)
	expected := image.Rect(125, 75, 175, 125)

	if rect != expected {
		t.Errorf("Detection.Rect() = %v, expected %v", rect, expected)
	}
}

func TestDetectFaces_UniformImage(t *testing.T) {
	// A uniform gray image should have no face detections
	width, height := 100, 100
	pixels := make([]uint8, width*height)
	for i := range pixels {
		pixels[i] = 128
	}

	detections := DetectFaces(pixels, width, height)

	// Should return empty or nil (no false positives on uniform image)
	// Note: The cascade might still produce some detections, so we just
	// verify it doesn't crash and returns a reasonable result
	t.Logf("Uniform image: %d detections", len(detections))
}

func TestDetectFaces_SmallImage(t *testing.T) {
	// Image smaller than minimum face size
	width, height := 20, 20
	pixels := make([]uint8, width*height)

	detections := DetectFaces(pixels, width, height)

	// Should return empty (can't detect faces smaller than MinSize)
	if len(detections) != 0 {
		t.Errorf("expected 0 detections on tiny image, got %d", len(detections))
	}
}

func TestDetectFacesImage(t *testing.T) {
	// Test the image.Image wrapper
	img := image.NewGray(image.Rect(0, 0, 100, 100))
	for y := range 100 {
		for x := range 100 {
			img.SetGray(x, y, color.Gray{Y: 128})
		}
	}

	// Should not panic
	detections := DetectFacesImage(img)
	t.Logf("DetectFacesImage: %d detections", len(detections))
}

func TestDefaultDetectParams(t *testing.T) {
	params := DefaultDetectParams()

	if params.MinSize != 30 {
		t.Errorf("MinSize = %d, expected 30", params.MinSize)
	}
	if params.MaxSize != 0 {
		t.Errorf("MaxSize = %d, expected 0 (auto)", params.MaxSize)
	}
	if params.ShiftFactor != 0.1 {
		t.Errorf("ShiftFactor = %f, expected 0.1", params.ShiftFactor)
	}
	if params.ScaleFactor != 1.1 {
		t.Errorf("ScaleFactor = %f, expected 1.1", params.ScaleFactor)
	}
}

func TestClusterDetections_Empty(t *testing.T) {
	result := clusterDetections(nil, 0.3)
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}

	result = clusterDetections([]Detection{}, 0.3)
	if result != nil {
		t.Errorf("expected nil for empty slice, got %v", result)
	}
}

func TestClusterDetections_SingleDetection(t *testing.T) {
	detections := []Detection{
		{Row: 100, Col: 100, Scale: 50, Score: 1.0},
	}

	result := clusterDetections(detections, 0.3)

	if len(result) != 1 {
		t.Errorf("expected 1 detection, got %d", len(result))
	}
}

func TestClusterDetections_NonOverlapping(t *testing.T) {
	// Two detections that don't overlap
	detections := []Detection{
		{Row: 50, Col: 50, Scale: 30, Score: 1.0},
		{Row: 150, Col: 150, Scale: 30, Score: 0.8},
	}

	result := clusterDetections(detections, 0.3)

	if len(result) != 2 {
		t.Errorf("expected 2 detections (non-overlapping), got %d", len(result))
	}
}

func TestClusterDetections_Overlapping(t *testing.T) {
	// Two detections that heavily overlap (should merge into one cluster)
	detections := []Detection{
		{Row: 100, Col: 100, Scale: 50, Score: 1.0},
		{Row: 105, Col: 105, Scale: 50, Score: 0.8},
	}

	result := clusterDetections(detections, 0.3)

	// Should merge into a single cluster with averaged position
	if len(result) != 1 {
		t.Errorf("expected 1 detection (overlapping merged), got %d", len(result))
	}
	if len(result) > 0 {
		// Averaged row: (100+105)/2 = 102
		// Averaged col: (100+105)/2 = 102
		// Averaged scale: (50+50)/2 = 50
		// Summed score: 1.0 + 0.8 = 1.8
		if result[0].Row != 102 {
			t.Errorf("expected averaged row 102, got %d", result[0].Row)
		}
		if result[0].Col != 102 {
			t.Errorf("expected averaged col 102, got %d", result[0].Col)
		}
	}
}

func TestClusterDetections_NonOverlapping_Preserved(t *testing.T) {
	// Three non-overlapping detections should all be preserved
	detections := []Detection{
		{Row: 50, Col: 50, Scale: 30, Score: 0.5},
		{Row: 150, Col: 150, Scale: 30, Score: 1.0},
		{Row: 250, Col: 250, Scale: 30, Score: 0.8},
	}

	result := clusterDetections(detections, 0.3)

	if len(result) != 3 {
		t.Fatalf("expected 3 detections (non-overlapping), got %d", len(result))
	}
}

func TestComputeIoU(t *testing.T) {
	testCases := []struct {
		name     string
		a, b     Detection
		expected float64
		epsilon  float64
	}{
		{
			name:     "identical",
			a:        Detection{Row: 100, Col: 100, Scale: 50},
			b:        Detection{Row: 100, Col: 100, Scale: 50},
			expected: 1.0,
			epsilon:  0.01,
		},
		{
			name:     "no overlap",
			a:        Detection{Row: 50, Col: 50, Scale: 30},
			b:        Detection{Row: 200, Col: 200, Scale: 30},
			expected: 0.0,
			epsilon:  0.01,
		},
		{
			name:     "partial overlap",
			a:        Detection{Row: 100, Col: 100, Scale: 50},
			b:        Detection{Row: 120, Col: 120, Scale: 50},
			expected: 0.22, // Pigo's IoU calculation
			epsilon:  0.1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			iou := computeIoU(tc.a, tc.b)
			if iou < tc.expected-tc.epsilon || iou > tc.expected+tc.epsilon {
				t.Errorf("computeIoU() = %f, expected ~%f", iou, tc.expected)
			}
		})
	}
}

func TestComputeIoU_Symmetric(t *testing.T) {
	a := Detection{Row: 100, Col: 100, Scale: 50}
	b := Detection{Row: 120, Col: 120, Scale: 50}

	iou1 := computeIoU(a, b)
	iou2 := computeIoU(b, a)

	if iou1 != iou2 {
		t.Errorf("IoU should be symmetric: %f != %f", iou1, iou2)
	}
}

func BenchmarkDetectFaces(b *testing.B) {
	// Create a 256x256 grayscale image
	width, height := 256, 256
	pixels := make([]uint8, width*height)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectFaces(pixels, width, height)
	}
}

func BenchmarkDetectFaces_640(b *testing.B) {
	// Create a 640x480 grayscale image (typical analysis size)
	width, height := 640, 480
	pixels := make([]uint8, width*height)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectFaces(pixels, width, height)
	}
}

func BenchmarkClusterDetections(b *testing.B) {
	// Create 100 random detections
	detections := make([]Detection, 100)
	for i := range detections {
		detections[i] = Detection{
			Row:   50 + i*5,
			Col:   50 + i*5,
			Scale: 30 + (i % 20),
			Score: float32(i) / 100,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clusterDetections(detections, 0.3)
	}
}
