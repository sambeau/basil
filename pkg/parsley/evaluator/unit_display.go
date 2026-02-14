// Package evaluator provides display and formatting functions for measurement units.
// This file implements FEAT-118: unit Inspect(), repr(), string interpolation,
// SI decimal display, and US Customary fraction display via GCD.
package evaluator

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// unitInspectString returns the PLN-format literal string for a Unit value.
// This is used by both Inspect() and .repr().
// SI values display as decimals; US Customary values display as fractions where possible.
func unitInspectString(u *Unit) string {
	if u.System == SystemUS {
		return unitUSDisplay(u)
	}
	return unitSIDisplay(u)
}

// unitInterpolationString returns the string used when a unit is interpolated
// into a template string. No `#` sigil, no space: "1.83m", "3/8in".
func unitInterpolationString(u *Unit) string {
	s := unitInspectString(u)
	// Strip the leading '#'
	if s != "" && s[0] == '#' {
		return s[1:]
	}
	return s
}

// --- SI display ---

// unitSIDisplay formats an SI unit value as a decimal literal: #12.3m, #500g, #1024B
func unitSIDisplay(u *Unit) string {
	subPerUnit := SISubUnitsPerUnit(u.DisplayHint)
	if subPerUnit == 0 {
		subPerUnit = 1
	}

	amount := u.Amount
	negative := amount < 0
	if negative {
		amount = -amount
	}

	dp := SIDefaultDecimalPlaces(u.DisplayHint)

	// Special case: if the value divides evenly, use fewer decimal places
	if amount%subPerUnit == 0 {
		// Exact integer value
		whole := amount / subPerUnit
		if negative {
			return fmt.Sprintf("#-%d%s", whole, u.DisplayHint)
		}
		return fmt.Sprintf("#%d%s", whole, u.DisplayHint)
	}

	// Compute the decimal representation
	// value = amount / subPerUnit, displayed with dp decimal places
	whole := amount / subPerUnit
	remainder := amount % subPerUnit

	if dp == 0 {
		// Round to nearest integer
		if remainder*2 >= subPerUnit {
			whole++
		}
		if negative {
			return fmt.Sprintf("#-%d%s", whole, u.DisplayHint)
		}
		return fmt.Sprintf("#%d%s", whole, u.DisplayHint)
	}

	// Calculate fractional part with the right number of decimal places
	// We need: frac = remainder * 10^dp / subPerUnit
	scale := int64(1)
	for range dp {
		scale *= 10
	}
	frac := remainder * scale / subPerUnit

	// Check if there are more significant digits beyond dp
	// If so, show them (up to a reasonable limit) to avoid information loss
	actualDP := dp
	testRemainder := remainder
	testScale := scale
	for actualDP < 6 {
		if testRemainder*testScale%subPerUnit == 0 {
			break
		}
		actualDP++
		testScale *= 10
	}

	if actualDP > dp {
		// Recompute with more decimal places
		scale = int64(1)
		for range actualDP {
			scale *= 10
		}
		frac = remainder * scale / subPerUnit
		dp = actualDP
	}

	// Trim trailing zeros from the fractional part
	fracStr := fmt.Sprintf("%0*d", dp, frac)
	fracStr = strings.TrimRight(fracStr, "0")
	if fracStr == "" {
		if negative {
			return fmt.Sprintf("#-%d%s", whole, u.DisplayHint)
		}
		return fmt.Sprintf("#%d%s", whole, u.DisplayHint)
	}

	if negative {
		return fmt.Sprintf("#-%d.%s%s", whole, fracStr, u.DisplayHint)
	}
	return fmt.Sprintf("#%d.%s%s", whole, fracStr, u.DisplayHint)
}

// --- US Customary display ---

// unitUSDisplay formats a US Customary unit value as a fraction literal: #3/8in, #92+5/8in
// Falls back to decimal if the reduced denominator is not a common fraction.
func unitUSDisplay(u *Unit) string {
	subPerUnit := USSubUnitsPerUnit(u.DisplayHint)
	if subPerUnit == 0 {
		subPerUnit = HCN
	}

	amount := u.Amount
	negative := amount < 0
	if negative {
		amount = -amount
	}

	prefix := "#"
	if negative {
		prefix = "#-"
	}

	if amount == 0 {
		return prefix + "0" + u.DisplayHint
	}

	// Convert to display-unit fraction: amount / subPerUnit
	// Reduce via GCD
	g := GCD(amount, subPerUnit)
	num := amount / g
	den := subPerUnit / g

	if den == 1 {
		// Exact integer
		return prefix + strconv.FormatInt(num, 10) + u.DisplayHint
	}

	// Extract whole part
	whole := num / den
	fracNum := num % den

	if fracNum == 0 {
		return prefix + strconv.FormatInt(whole, 10) + u.DisplayHint
	}

	// Reduce the fractional part
	g2 := GCD(fracNum, den)
	fracNum /= g2
	fracDen := den / g2

	// Check if denominator is a common fraction denominator
	if !CommonFractionDenominators[fracDen] {
		// Fall back to decimal display
		return unitUSDecimalFallback(u, negative)
	}

	if whole > 0 {
		// Mixed number: #92+5/8in
		return fmt.Sprintf("%s%d+%d/%d%s", prefix, whole, fracNum, fracDen, u.DisplayHint)
	}
	// Simple fraction: #3/8in
	return fmt.Sprintf("%s%d/%d%s", prefix, fracNum, fracDen, u.DisplayHint)
}

// unitUSDecimalFallback formats a US Customary value as a decimal when the
// fraction denominator is not in the common set.
func unitUSDecimalFallback(u *Unit, negative bool) string {
	subPerUnit := USSubUnitsPerUnit(u.DisplayHint)
	if subPerUnit == 0 {
		subPerUnit = HCN
	}

	amount := u.Amount
	if amount < 0 {
		amount = -amount
	}

	value := float64(amount) / float64(subPerUnit)

	prefix := "#"
	if negative {
		prefix = "#-"
	}

	// Format with enough precision to be useful
	s := strconv.FormatFloat(value, 'f', -1, 64)
	return prefix + s + u.DisplayHint
}

// --- Value extraction ---

// unitDisplayValue returns the float64 value in display-hint units.
// Used for the .value property.
func unitDisplayValue(u *Unit) float64 {
	if u.System == SystemUS {
		subPerUnit := USSubUnitsPerUnit(u.DisplayHint)
		if subPerUnit == 0 {
			return 0
		}
		return float64(u.Amount) / float64(subPerUnit)
	}
	subPerUnit := SISubUnitsPerUnit(u.DisplayHint)
	if subPerUnit == 0 {
		return 0
	}
	return float64(u.Amount) / float64(subPerUnit)
}

// --- Formatting with options ---

// unitFormat formats a unit value with optional precision override.
// If precision is -1, uses defaults.
func unitFormat(u *Unit, precision int) string {
	if precision < 0 {
		// Use default formatting (same as interpolation)
		return unitInterpolationString(u)
	}

	// Custom precision
	if u.System == SystemUS {
		return unitFormatUSPrecision(u, precision)
	}
	return unitFormatSIPrecision(u, precision)
}

func unitFormatSIPrecision(u *Unit, dp int) string {
	subPerUnit := SISubUnitsPerUnit(u.DisplayHint)
	if subPerUnit == 0 {
		subPerUnit = 1
	}

	value := float64(u.Amount) / float64(subPerUnit)
	formatted := strconv.FormatFloat(math.Abs(value), 'f', dp, 64)

	if u.Amount < 0 {
		return "-" + formatted + u.DisplayHint
	}
	return formatted + u.DisplayHint
}

func unitFormatUSPrecision(u *Unit, dp int) string {
	subPerUnit := USSubUnitsPerUnit(u.DisplayHint)
	if subPerUnit == 0 {
		subPerUnit = HCN
	}

	value := float64(u.Amount) / float64(subPerUnit)
	formatted := strconv.FormatFloat(math.Abs(value), 'f', dp, 64)

	if u.Amount < 0 {
		return "-" + formatted + u.DisplayHint
	}
	return formatted + u.DisplayHint
}
