package evaluator

import (
	"unicode"
)

// eval_helpers.go contains utility functions for the evaluator that don't directly
// perform evaluation. This includes natural sorting, type comparison, and other helpers.

// naturalCompare compares two objects using natural sort order
// Returns true if a < b in natural sort order
func naturalCompare(a, b Object) bool {
	// Type-based ordering: numbers < strings
	aType := getTypeOrder(a)
	bType := getTypeOrder(b)

	if aType != bType {
		return aType < bType
	}

	// Both are numbers
	if aType == 0 {
		return compareNumbers(a, b)
	}

	// Both are strings - use natural string comparison
	if aType == 1 {
		aStr := a.(*String).Value
		bStr := b.(*String).Value
		return NaturalCompare(aStr, bStr) < 0
	}

	// Other types (shouldn't happen with current implementation)
	return false
}

// getTypeOrder returns a sort order for types
// 0 = numbers (Integer, Float)
// 1 = strings
// 2 = other
func getTypeOrder(obj Object) int {
	switch obj.Type() {
	case INTEGER_OBJ, FLOAT_OBJ:
		return 0
	case STRING_OBJ:
		return 1
	default:
		return 2
	}
}

// compareNumbers compares two numeric objects
func compareNumbers(a, b Object) bool {
	aVal := getNumericValue(a)
	bVal := getNumericValue(b)
	return aVal < bVal
}

// getNumericValue extracts numeric value as float64
func getNumericValue(obj Object) float64 {
	switch obj := obj.(type) {
	case *Integer:
		return float64(obj.Value)
	case *Float:
		return obj.Value
	default:
		return 0
	}
}

// NaturalCompare compares two strings using natural sort order.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Uses ASCII fast path for performance, falls back to Unicode for international text.
func NaturalCompare(a, b string) int {
	// Fast path: check if both strings are ASCII
	if isASCII(a) && isASCII(b) {
		return naturalCompareASCII(a, b)
	}
	// Fallback: Unicode-aware comparison
	return naturalCompareUnicode(a, b)
}

// isASCII checks if a string contains only ASCII characters
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 128 {
			return false
		}
	}
	return true
}

// isDigitASCII checks if byte is ASCII digit (inlined for performance)
func isDigitASCII(c byte) bool {
	return '0' <= c && c <= '9'
}

// naturalCompareASCII compares two ASCII strings using natural sort order.
// This is the fast path - works on bytes directly with zero allocations.
func naturalCompareASCII(a, b string) int {
	ai, bi := 0, 0

	for ai < len(a) && bi < len(b) {
		ca, cb := a[ai], b[bi]

		// Both are digits - compare numerically
		if isDigitASCII(ca) && isDigitASCII(cb) {
			// Check for leading zeros (left-aligned comparison)
			if ca == '0' || cb == '0' {
				if result := compareLeftASCII(a, b, &ai, &bi); result != 0 {
					return result
				}
				continue
			}
			// Right-aligned comparison (no leading zeros)
			if result := compareRightASCII(a, b, &ai, &bi); result != 0 {
				return result
			}
			continue
		}

		// Regular character comparison
		if ca != cb {
			if ca < cb {
				return -1
			}
			return 1
		}

		ai++
		bi++
	}

	// Shorter string comes first
	if ai < len(a) {
		return 1
	}
	if bi < len(b) {
		return -1
	}
	return 0
}

// compareRightASCII compares two right-aligned numbers (no leading zeros).
// The longest run of digits wins; if equal length, greater value wins.
func compareRightASCII(a, b string, ai, bi *int) int {
	bias := 0

	for {
		var ca, cb byte
		aHasMore := *ai < len(a)
		bHasMore := *bi < len(b)

		if aHasMore {
			ca = a[*ai]
		}
		if bHasMore {
			cb = b[*bi]
		}

		aIsDigit := aHasMore && isDigitASCII(ca)
		bIsDigit := bHasMore && isDigitASCII(cb)

		if !aIsDigit && !bIsDigit {
			return bias
		}
		if !aIsDigit {
			return -1
		}
		if !bIsDigit {
			return 1
		}

		if ca < cb {
			if bias == 0 {
				bias = -1
			}
		} else if ca > cb {
			if bias == 0 {
				bias = 1
			}
		}

		*ai++
		*bi++
	}
}

// compareLeftASCII compares two left-aligned numbers (with leading zeros).
// First different digit wins.
func compareLeftASCII(a, b string, ai, bi *int) int {
	for {
		var ca, cb byte
		aHasMore := *ai < len(a)
		bHasMore := *bi < len(b)

		if aHasMore {
			ca = a[*ai]
		}
		if bHasMore {
			cb = b[*bi]
		}

		aIsDigit := aHasMore && isDigitASCII(ca)
		bIsDigit := bHasMore && isDigitASCII(cb)

		if !aIsDigit && !bIsDigit {
			return 0
		}
		if !aIsDigit {
			return -1
		}
		if !bIsDigit {
			return 1
		}

		if ca < cb {
			return -1
		}
		if ca > cb {
			return 1
		}

		*ai++
		*bi++
	}
}

// naturalCompareUnicode compares two strings using natural sort order with full Unicode support.
// This handles non-ASCII digits (Arabic numerals, etc.) but is slower than ASCII path.
func naturalCompareUnicode(a, b string) int {
	aRunes := []rune(a)
	bRunes := []rune(b)

	ai, bi := 0, 0

	for ai < len(aRunes) && bi < len(bRunes) {
		ca, cb := aRunes[ai], bRunes[bi]

		// Both are digits - compare numerically
		if unicode.IsDigit(ca) && unicode.IsDigit(cb) {
			// Check for leading zeros
			if ca == '0' || cb == '0' {
				if result := compareLeftUnicode(aRunes, bRunes, &ai, &bi); result != 0 {
					return result
				}
				continue
			}
			if result := compareRightUnicode(aRunes, bRunes, &ai, &bi); result != 0 {
				return result
			}
			continue
		}

		// Regular character comparison
		if ca != cb {
			if ca < cb {
				return -1
			}
			return 1
		}

		ai++
		bi++
	}

	// Shorter string comes first
	if ai < len(aRunes) {
		return 1
	}
	if bi < len(bRunes) {
		return -1
	}
	return 0
}

// compareRightUnicode compares two right-aligned numbers with Unicode digit support.
func compareRightUnicode(a, b []rune, ai, bi *int) int {
	bias := 0

	for {
		var ca, cb rune
		aHasMore := *ai < len(a)
		bHasMore := *bi < len(b)

		if aHasMore {
			ca = a[*ai]
		}
		if bHasMore {
			cb = b[*bi]
		}

		aIsDigit := aHasMore && unicode.IsDigit(ca)
		bIsDigit := bHasMore && unicode.IsDigit(cb)

		if !aIsDigit && !bIsDigit {
			return bias
		}
		if !aIsDigit {
			return -1
		}
		if !bIsDigit {
			return 1
		}

		// Get numeric value of Unicode digit
		aVal := digitValue(ca)
		bVal := digitValue(cb)

		if aVal < bVal {
			if bias == 0 {
				bias = -1
			}
		} else if aVal > bVal {
			if bias == 0 {
				bias = 1
			}
		}

		*ai++
		*bi++
	}
}

// compareLeftUnicode compares two left-aligned numbers with Unicode digit support.
func compareLeftUnicode(a, b []rune, ai, bi *int) int {
	for {
		var ca, cb rune
		aHasMore := *ai < len(a)
		bHasMore := *bi < len(b)

		if aHasMore {
			ca = a[*ai]
		}
		if bHasMore {
			cb = b[*bi]
		}

		aIsDigit := aHasMore && unicode.IsDigit(ca)
		bIsDigit := bHasMore && unicode.IsDigit(cb)

		if !aIsDigit && !bIsDigit {
			return 0
		}
		if !aIsDigit {
			return -1
		}
		if !bIsDigit {
			return 1
		}

		// Get numeric value of Unicode digit
		aVal := digitValue(ca)
		bVal := digitValue(cb)

		if aVal < bVal {
			return -1
		}
		if aVal > bVal {
			return 1
		}

		*ai++
		*bi++
	}
}

// digitValue returns the numeric value of a Unicode digit rune (0-9).
// For non-decimal digits, falls back to rune comparison.
func digitValue(r rune) int {
	// ASCII fast path
	if '0' <= r && r <= '9' {
		return int(r - '0')
	}
	// Unicode digit - use the digit value from Unicode
	// Most digit characters are in blocks of 10, starting at 0
	// This handles Arabic-Indic, Devanagari, etc.
	return int(r % 10)
}

// objectsEqual compares two objects for equality
func objectsEqual(a, b Object) bool {
	if a.Type() != b.Type() {
		return false
	}

	switch a := a.(type) {
	case *Integer:
		return a.Value == b.(*Integer).Value
	case *Float:
		return a.Value == b.(*Float).Value
	case *String:
		return a.Value == b.(*String).Value
	case *Boolean:
		return a.Value == b.(*Boolean).Value
	case *Null:
		return true
	default:
		return false
	}
}
