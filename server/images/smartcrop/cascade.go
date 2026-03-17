package smartcrop

import (
	"encoding/binary"
	"errors"
	"math"
	"sync"

	_ "embed"
)

//go:embed facefinder
var cascadeData []byte

// Cascade holds the unpacked decision tree data from the cascade file.
// The PICO algorithm uses a binary decision tree cascade where each tree
// classifies whether a region contains a face based on pixel intensity comparisons.
type Cascade struct {
	treeCodes     []int8    // Encoded pixel comparison locations
	treePred      []float32 // Prediction values at leaf nodes
	treeThreshold []float32 // Threshold for each tree
	treeDepth     uint32    // Depth of each tree
	treeNum       uint32    // Number of trees in the cascade
}

var (
	defaultCascade     *Cascade
	defaultCascadeOnce sync.Once
	defaultCascadeErr  error
)

// DefaultCascade returns the embedded facefinder cascade.
// The cascade is loaded once and cached for subsequent calls.
func DefaultCascade() (*Cascade, error) {
	defaultCascadeOnce.Do(func() {
		defaultCascade, defaultCascadeErr = unpackCascade(cascadeData)
	})
	return defaultCascade, defaultCascadeErr
}

// pow computes base^exp for integers.
func pow(base, exp int) int {
	result := 1
	for i := 0; i < exp; i++ {
		result *= base
	}
	return result
}

// unpackCascade parses the binary cascade file format.
// Format (all values little-endian):
//   - 8 bytes: header (skipped)
//   - 4 bytes: tree depth (uint32)
//   - 4 bytes: number of trees (uint32)
//   - For each tree:
//   - 4 leading zero bytes
//   - (4 * 2^depth - 4) int8 codes (pixel comparison locations)
//   - 2^depth float32 predictions (leaf node values)
//   - 1 float32 threshold
func unpackCascade(data []byte) (*Cascade, error) {
	if len(data) < 16 {
		return nil, errors.New("cascade data too short")
	}

	// Skip first 8 bytes, read depth and tree count
	pos := 8
	treeDepth := binary.LittleEndian.Uint32(data[pos:])
	pos += 4
	treeNum := binary.LittleEndian.Uint32(data[pos:])
	pos += 4

	if treeDepth == 0 || treeDepth > 20 {
		return nil, errors.New("invalid tree depth")
	}
	if treeNum == 0 || treeNum > 10000 {
		return nil, errors.New("invalid tree count")
	}

	// Calculate sizes based on Pigo's format
	// Each tree has:
	// - 4 leading zero bytes for codes
	// - (4 * 2^depth - 4) code bytes
	// - 2^depth prediction values (float32)
	// - 1 threshold value (float32)
	pow2Depth := pow(2, int(treeDepth))
	numCodesPerTree := 4*pow2Depth - 4
	numPredsPerTree := pow2Depth

	// Pre-allocate slices
	cascade := &Cascade{
		treeCodes:     make([]int8, 0, int(treeNum)*(numCodesPerTree+4)),
		treePred:      make([]float32, 0, int(treeNum)*numPredsPerTree),
		treeThreshold: make([]float32, 0, treeNum),
		treeDepth:     treeDepth,
		treeNum:       treeNum,
	}

	for t := 0; t < int(treeNum); t++ {
		// Add 4 leading zeros for codes (this matches Pigo's format)
		cascade.treeCodes = append(cascade.treeCodes, 0, 0, 0, 0)

		// Read codes
		if pos+numCodesPerTree > len(data) {
			return nil, errors.New("cascade data truncated reading codes")
		}
		for i := 0; i < numCodesPerTree; i++ {
			cascade.treeCodes = append(cascade.treeCodes, int8(data[pos]))
			pos++
		}

		// Read predictions
		if pos+numPredsPerTree*4 > len(data) {
			return nil, errors.New("cascade data truncated reading predictions")
		}
		for i := 0; i < numPredsPerTree; i++ {
			bits := binary.LittleEndian.Uint32(data[pos:])
			cascade.treePred = append(cascade.treePred, math.Float32frombits(bits))
			pos += 4
		}

		// Read threshold
		if pos+4 > len(data) {
			return nil, errors.New("cascade data truncated reading threshold")
		}
		bits := binary.LittleEndian.Uint32(data[pos:])
		cascade.treeThreshold = append(cascade.treeThreshold, math.Float32frombits(bits))
		pos += 4
	}

	return cascade, nil
}
