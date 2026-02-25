// Package evaluator provides decimal scale helpers for measurement unit arithmetic.
// Scale extends the range of int64-based unit storage: true value = Amount × 10^Scale.
// For all existing families (length, mass, data, volume, temperature), Scale is always 0.
// Scale > 0 occurs only for area (and large volume) values that exceed int64 range.
//
// Every helper fast-paths on Scale == 0 to guarantee zero behaviour change for existing code.
package evaluator

import "math"

// scalePow10 returns 10^n as int64. For n > 18 returns math.MaxInt64.
// For n < 0 returns 0 (we never use negative exponents in practice).
func scalePow10(n int) int64 {
	if n < 0 {
		return 0
	}
	if n > 18 {
		return math.MaxInt64
	}
	result := int64(1)
	for range n {
		result *= 10
	}
	return result
}

// scaleAlign aligns two (Amount, Scale) pairs to a common Scale for addition/subtraction/comparison.
// It tries to align to the smaller Scale first (maximum precision).
// If aligning the larger-Scale value would overflow, it falls back to the larger Scale
// (truncating the smaller-Scale value).
// Returns aligned amounts and the common Scale.
func scaleAlign(aAmount int64, aScale int, bAmount int64, bScale int) (alignedA, alignedB int64, commonScale int) {
	if aScale == bScale {
		return aAmount, bAmount, aScale
	}

	// Determine which has the larger scale
	var hiAmount, loAmount int64
	var hiScale, loScale int
	var hiIsA bool

	if aScale > bScale {
		hiAmount, hiScale = aAmount, aScale
		loAmount, loScale = bAmount, bScale
		hiIsA = true
	} else {
		hiAmount, hiScale = bAmount, bScale
		loAmount, loScale = aAmount, aScale
		hiIsA = false
	}

	diff := hiScale - loScale

	// Try to bring the higher-scale value down to the lower scale (maximum precision).
	// This means multiplying hiAmount by 10^diff.
	if diff <= 18 {
		factor := scalePow10(diff)
		if hiAmount != 0 && factor != 0 {
			// Check for overflow: |hiAmount * factor| <= MaxInt64
			limit := math.MaxInt64 / factor
			if hiAmount >= 0 && hiAmount <= limit {
				aligned := hiAmount * factor
				if hiIsA {
					return aligned, loAmount, loScale
				}
				return loAmount, aligned, loScale
			}
			if hiAmount < 0 && hiAmount >= -limit {
				aligned := hiAmount * factor
				if hiIsA {
					return aligned, loAmount, loScale
				}
				return loAmount, aligned, loScale
			}
		}
		if hiAmount == 0 {
			if hiIsA {
				return 0, loAmount, loScale
			}
			return loAmount, 0, loScale
		}
	}

	// Can't bring hi down to lo scale without overflow.
	// Fall back: bring lo up to hi scale (truncating lo).
	if diff <= 18 {
		factor := scalePow10(diff)
		loAligned := loAmount / factor
		if hiIsA {
			return hiAmount, loAligned, hiScale
		}
		return loAligned, hiAmount, hiScale
	}

	// Extreme difference (>18 orders of magnitude): lo is negligible
	if hiIsA {
		return hiAmount, 0, hiScale
	}
	return 0, hiAmount, hiScale
}

// scaleNormalize normalizes an (Amount, Scale) pair after arithmetic.
// When Scale > 0, tries to absorb scale back into amount by multiplying amount
// and decreasing scale, moving toward Scale=0 where arithmetic is fully exact.
// True value (Amount × 10^Scale) is preserved.
func scaleNormalize(amount int64, scale int) (normAmount int64, normScale int) {
	if scale == 0 {
		return amount, 0
	}

	if amount == 0 {
		return 0, 0
	}

	// Try to absorb scale into amount: multiply amount by 10 and decrease scale.
	// This moves toward Scale=0 (exact integer arithmetic) when possible.
	for scale > 0 {
		if amount > 0 {
			if amount > math.MaxInt64/10 {
				break // would overflow
			}
		} else {
			if amount < -math.MaxInt64/10 {
				break // would overflow
			}
		}
		amount *= 10
		scale--
	}

	return amount, scale
}

// scaleAdd performs scale-aware addition: (aAmount, aScale) + (bAmount, bScale).
// Returns the result (Amount, Scale).
func scaleAdd(aAmount int64, aScale int, bAmount int64, bScale int) (resultAmount int64, resultScale int) {
	// Fast path
	if aScale == 0 && bScale == 0 {
		result := aAmount + bAmount
		// Check for overflow
		if (bAmount > 0 && result < aAmount) || (bAmount < 0 && result > aAmount) {
			// Overflow: normalize by increasing scale
			return scaleAddOverflow(aAmount, 0, bAmount, 0)
		}
		return result, 0
	}

	aa, ba, s := scaleAlign(aAmount, aScale, bAmount, bScale)
	result := aa + ba

	// Check for overflow after addition
	if (ba > 0 && result < aa) || (ba < 0 && result > aa) {
		return scaleAddOverflow(aAmount, aScale, bAmount, bScale)
	}

	return scaleNormalize(result, s)
}

// scaleAddOverflow handles addition when the sum overflows at the aligned scale.
// It increases the scale by 1 and retries.
func scaleAddOverflow(aAmount int64, aScale int, bAmount int64, bScale int) (resultAmount int64, resultScale int) {
	// Increase both to a higher scale
	targetScale := max(aScale, bScale) + 1

	aDiff := targetScale - aScale
	bDiff := targetScale - bScale

	var aa, ba int64
	if aDiff > 0 && aDiff <= 18 {
		aa = aAmount / scalePow10(aDiff)
	} else if aDiff == 0 {
		aa = aAmount
	}
	if bDiff > 0 && bDiff <= 18 {
		ba = bAmount / scalePow10(bDiff)
	} else if bDiff == 0 {
		ba = bAmount
	}

	result := aa + ba

	// If still overflows, keep increasing scale
	for (ba > 0 && result < aa) || (ba < 0 && result > aa) {
		targetScale++
		aa /= 10
		ba /= 10
		result = aa + ba
	}

	return scaleNormalize(result, targetScale)
}

// scaleSub performs scale-aware subtraction: (aAmount, aScale) - (bAmount, bScale).
func scaleSub(aAmount int64, aScale int, bAmount int64, bScale int) (resultAmount int64, resultScale int) {
	// Fast path
	if aScale == 0 && bScale == 0 {
		result := aAmount - bAmount
		// Check for overflow
		if (bAmount < 0 && result < aAmount) || (bAmount > 0 && result > aAmount) {
			return scaleSubOverflow(aAmount, 0, bAmount, 0)
		}
		return result, 0
	}

	aa, ba, s := scaleAlign(aAmount, aScale, bAmount, bScale)
	result := aa - ba

	if (ba < 0 && result < aa) || (ba > 0 && result > aa) {
		return scaleSubOverflow(aAmount, aScale, bAmount, bScale)
	}

	return scaleNormalize(result, s)
}

func scaleSubOverflow(aAmount int64, aScale int, bAmount int64, bScale int) (resultAmount int64, resultScale int) {
	// Subtraction overflow: treat as addition of negative
	return scaleAddOverflow(aAmount, aScale, -bAmount, bScale)
}

// scaleMulScalar performs scale-aware multiplication by a scalar: (amount, scale) * scalar.
// The scalar is a float64 (from unit * number operations).
func scaleMulScalar(amount int64, scale int, scalar float64) (resultAmount int64, resultScale int) {
	if scale == 0 {
		result := float64(amount) * scalar
		rounded := math.Round(result)
		if rounded >= math.MinInt64 && rounded <= math.MaxInt64 {
			return int64(rounded), 0
		}
		// Overflow: increase scale
		return scaleMulScalarOverflow(amount, 0, scalar)
	}

	result := float64(amount) * scalar
	rounded := math.Round(result)
	if rounded >= math.MinInt64 && rounded <= math.MaxInt64 {
		return scaleNormalize(int64(rounded), scale)
	}

	return scaleMulScalarOverflow(amount, scale, scalar)
}

func scaleMulScalarOverflow(amount int64, scale int, scalar float64) (resultAmount int64, resultScale int) {
	// Keep dividing amount by 10 and increasing scale until the product fits
	s := scale
	a := float64(amount)
	for {
		s++
		a /= 10
		result := a * scalar
		rounded := math.Round(result)
		if rounded >= math.MinInt64 && rounded <= math.MaxInt64 {
			return scaleNormalize(int64(rounded), s)
		}
		if a == 0 {
			return 0, 0
		}
	}
}

// scaleDivScalar performs scale-aware division by a scalar: (amount, scale) / scalar.
func scaleDivScalar(amount int64, scale int, scalar float64) (resultAmount int64, resultScale int) {
	if scalar == 0 {
		return 0, 0 // caller should handle division by zero before calling
	}
	if scale == 0 {
		result := float64(amount) / scalar
		rounded := math.Round(result)
		if rounded >= math.MinInt64 && rounded <= math.MaxInt64 {
			return int64(rounded), 0
		}
	}
	result := float64(amount) / scalar
	rounded := math.Round(result)
	if rounded >= math.MinInt64 && rounded <= math.MaxInt64 {
		return scaleNormalize(int64(rounded), scale)
	}
	// Very unlikely: division producing overflow
	return int64(math.Round(float64(amount) / scalar)), scale
}

// scaleMulDiv performs scale-aware multiply-then-divide for bridge conversions:
// result = (amount × 10^scale) * mulNum / mulDen
// Returns (resultAmount, resultScale).
func scaleMulDiv(amount int64, scale int, mulNum, mulDen int64) (resultAmount int64, resultScale int) {
	if mulDen == 0 {
		return 0, 0
	}

	// Fast path: Scale == 0 and multiplication won't overflow
	if scale == 0 {
		if mulNum != 0 && amount != 0 {
			limit := math.MaxInt64 / absInt64(mulNum)
			if absInt64(amount) <= limit {
				// No overflow: use integer arithmetic for exactness
				return amount * mulNum / mulDen, 0
			}
		} else {
			return 0, 0
		}
	}

	// Scale-aware path: try to multiply at current scale
	if amount != 0 && mulNum != 0 {
		limit := math.MaxInt64 / absInt64(mulNum)
		if absInt64(amount) <= limit {
			result := amount * mulNum / mulDen
			return scaleNormalize(result, scale)
		}
	} else {
		return 0, 0
	}

	// Overflow: shift scale up before multiplying
	s := scale
	a := amount
	for {
		s++
		a /= 10
		if a == 0 {
			return 0, 0
		}
		limit := math.MaxInt64 / absInt64(mulNum)
		if absInt64(a) <= limit {
			result := a * mulNum / mulDen
			return scaleNormalize(result, s)
		}
	}
}

// scaleCmp compares two scaled values: returns -1 if a < b, 0 if a == b, +1 if a > b.
func scaleCmp(aAmount int64, aScale int, bAmount int64, bScale int) int {
	// Fast path
	if aScale == 0 && bScale == 0 {
		return cmpInt64(aAmount, bAmount)
	}

	// Quick sign-based comparison
	aSign := signInt64(aAmount)
	bSign := signInt64(bAmount)
	if aSign != bSign {
		if aSign < bSign {
			return -1
		}
		return 1
	}
	if aSign == 0 && bSign == 0 {
		return 0
	}

	// Same sign — try alignment
	aa, ba, _ := scaleAlign(aAmount, aScale, bAmount, bScale)
	return cmpInt64(aa, ba)
}

// scaleRatio computes (aAmount, aScale) / (bAmount, bScale) as a float64.
// Used for unit/unit division that returns a dimensionless number.
func scaleRatio(aAmount int64, aScale int, bAmount int64, bScale int) float64 {
	if bAmount == 0 {
		return 0 // caller handles division by zero
	}

	// Fast path
	if aScale == 0 && bScale == 0 {
		return float64(aAmount) / float64(bAmount)
	}

	// General case: use true values
	aTrue := scaleTrueValue(aAmount, aScale)
	bTrue := scaleTrueValue(bAmount, bScale)
	if bTrue == 0 {
		return 0
	}
	return aTrue / bTrue
}

// scaleTrueValue returns the true value as float64: amount * 10^scale.
// Used for display and .value property.
func scaleTrueValue(amount int64, scale int) float64 {
	if scale == 0 {
		return float64(amount)
	}
	return float64(amount) * math.Pow10(scale)
}

// scaleAmountForSuffix computes the amount in base sub-units for a given value and suffix,
// applying Scale when the multiplication would overflow int64.
// Returns (amount, scale).
// Used by both the parser and constructors to handle large values.
func scaleAmountForSuffix(whole, subPerUnit int64) (amount int64, scale int) {
	if subPerUnit <= 1 {
		return whole, 0
	}

	// Check if whole * subPerUnit fits in int64
	if whole != 0 {
		limit := math.MaxInt64 / subPerUnit
		if whole >= 0 && whole <= limit {
			return whole * subPerUnit, 0
		}
		if whole < 0 && whole >= -limit {
			return whole * subPerUnit, 0
		}
	} else {
		return 0, 0
	}

	// Overflow: factor out powers of 10 from subPerUnit into Scale
	scale = 0
	reducedSub := subPerUnit
	for reducedSub > 1 && reducedSub%10 == 0 {
		reducedSub /= 10
		scale++
	}

	// Try with reduced subPerUnit
	if reducedSub <= 1 {
		// subPerUnit was a pure power of 10 — whole stays as-is, scale carries the exponent
		return whole * reducedSub, scale
	}

	// subPerUnit has non-10 factors. Try multiplying by the reduced value.
	if whole != 0 {
		limit := math.MaxInt64 / reducedSub
		if absInt64(whole) <= limit {
			return whole * reducedSub, scale
		}
	}

	// Still overflows: factor out from whole as well
	for absInt64(whole) > math.MaxInt64/reducedSub && whole != 0 {
		whole /= 10
		scale++
	}

	return whole * reducedSub, scale
}

// scaleAmountForSuffixFloat computes the amount in base sub-units for a float value and suffix,
// applying Scale when the result would overflow int64.
// Returns (amount, scale).
func scaleAmountForSuffixFloat(value float64, subPerUnit int64) (amount int64, scale int) {
	result := value * float64(subPerUnit)
	rounded := math.Round(result)
	if rounded >= math.MinInt64 && rounded <= math.MaxInt64 {
		return int64(rounded), 0
	}

	// Overflow: use scaleAmountForSuffix on the integer part
	// and handle fractional part separately
	whole := int64(value)
	frac := value - float64(whole)
	a, s := scaleAmountForSuffix(whole, subPerUnit)
	if frac != 0 && s == 0 {
		fracAmount := int64(math.Round(frac * float64(subPerUnit)))
		result := a + fracAmount
		if (fracAmount > 0 && result < a) || (fracAmount < 0 && result > a) {
			// Overflow on addition — just return the whole part
			return a, s
		}
		return result, s
	}
	return a, s
}

// --- internal helpers ---

func absInt64(v int64) int64 {
	if v < 0 {
		if v == math.MinInt64 {
			return math.MaxInt64 // avoid overflow
		}
		return -v
	}
	return v
}

func cmpInt64(a, b int64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func signInt64(v int64) int {
	if v < 0 {
		return -1
	}
	if v > 0 {
		return 1
	}
	return 0
}
