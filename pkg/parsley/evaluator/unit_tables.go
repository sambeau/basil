// Package evaluator provides unit measurement tables for FEAT-118.
// This file defines suffix metadata, sub-unit bases, within-system ratios,
// cross-system bridge constants, and display configuration for measurement units.
package evaluator

import "math"

// HCN is the Highly Composite Denominator for US Customary unit storage.
// All US Customary values are stored as Amount / HCN in the base unit.
// HCN = 2⁸ × 3⁴ × 5 × 7 = 725,760
// This yields 20,160 sub-units per inch, making all common fractions exact.
const HCN int64 = 725_760

// Unit system identifiers
const (
	SystemSI = "SI"
	SystemUS = "US"
)

// Unit family identifiers
const (
	FamilyLength      = "length"
	FamilyMass        = "mass"
	FamilyData        = "data"
	FamilyTemperature = "temperature"
	FamilyVolume      = "volume"
)

// Temperature storage: all temperatures are stored as int64 sub-kelvins,
// where 1 K = 900 sub-K. This makes the 9/5 C-to-F ratio exact (900/500).
const (
	TempScaleK  int64 = 900     // sub-K per kelvin
	TempScaleC  int64 = 900     // sub-K per degree Celsius
	TempScaleF  int64 = 500     // sub-K per degree Fahrenheit
	TempOffsetC int64 = 245_835 // 273.15 × 900 — sub-K value at 0°C
	TempOffsetF int64 = 229_835 // 459.67 × 500 — sub-K value at 0°F
	TempOffsetK int64 = 0       // kelvin has no offset
)

// TempOffset returns the sub-K offset for a temperature suffix.
func TempOffset(suffix string) int64 {
	switch suffix {
	case "C":
		return TempOffsetC
	case "F":
		return TempOffsetF
	case "K":
		return TempOffsetK
	default:
		return 0
	}
}

// TempScale returns the sub-K per degree for a temperature suffix.
func TempScale(suffix string) int64 {
	switch suffix {
	case "C":
		return TempScaleC
	case "F":
		return TempScaleF
	case "K":
		return TempScaleK
	default:
		return 0
	}
}

// EncodeTempToSubK encodes a numeric temperature value to sub-kelvins.
// e.g., EncodeTempToSubK(20.0, "C") = 20*900 + 245835 = 263835
// Uses math.Round to avoid floating-point truncation errors.
func EncodeTempToSubK(value float64, suffix string) int64 {
	scale := TempScale(suffix)
	offset := TempOffset(suffix)
	return int64(math.Round(value*float64(scale))) + offset
}

// DecodeTempFromSubK decodes sub-kelvins to a numeric value in the given scale.
// e.g., DecodeTempFromSubK(263835, "C") = (263835 - 245835) / 900 = 20.0
func DecodeTempFromSubK(subK int64, suffix string) float64 {
	scale := TempScale(suffix)
	offset := TempOffset(suffix)
	if scale == 0 {
		return 0
	}
	return float64(subK-offset) / float64(scale)
}

// IsTemperatureFamily returns true if the family is temperature.
func IsTemperatureFamily(family string) bool {
	return family == FamilyTemperature
}

// UnitInfo describes a unit suffix's metadata.
type UnitInfo struct {
	Suffix string // the suffix string (e.g., "m", "in", "kg")
	Family string // "length", "mass", "data", "temperature", "volume"
	System string // "SI", "US"
}

// unitSuffixTable maps unit suffixes to their metadata.
var unitSuffixTable = map[string]UnitInfo{
	// Length — SI
	"mm": {Suffix: "mm", Family: FamilyLength, System: SystemSI},
	"cm": {Suffix: "cm", Family: FamilyLength, System: SystemSI},
	"m":  {Suffix: "m", Family: FamilyLength, System: SystemSI},
	"km": {Suffix: "km", Family: FamilyLength, System: SystemSI},
	// Length — US
	"in": {Suffix: "in", Family: FamilyLength, System: SystemUS},
	"ft": {Suffix: "ft", Family: FamilyLength, System: SystemUS},
	"yd": {Suffix: "yd", Family: FamilyLength, System: SystemUS},
	"mi": {Suffix: "mi", Family: FamilyLength, System: SystemUS},
	// Mass — SI
	"mg": {Suffix: "mg", Family: FamilyMass, System: SystemSI},
	"g":  {Suffix: "g", Family: FamilyMass, System: SystemSI},
	"kg": {Suffix: "kg", Family: FamilyMass, System: SystemSI},
	// Mass — US
	"oz": {Suffix: "oz", Family: FamilyMass, System: SystemUS},
	"lb": {Suffix: "lb", Family: FamilyMass, System: SystemUS},
	// Digital information — decimal
	"B":  {Suffix: "B", Family: FamilyData, System: SystemSI},
	"kB": {Suffix: "kB", Family: FamilyData, System: SystemSI},
	"MB": {Suffix: "MB", Family: FamilyData, System: SystemSI},
	"GB": {Suffix: "GB", Family: FamilyData, System: SystemSI},
	"TB": {Suffix: "TB", Family: FamilyData, System: SystemSI},
	// Digital information — binary
	"KiB": {Suffix: "KiB", Family: FamilyData, System: SystemSI},
	"MiB": {Suffix: "MiB", Family: FamilyData, System: SystemSI},
	"GiB": {Suffix: "GiB", Family: FamilyData, System: SystemSI},
	"TiB": {Suffix: "TiB", Family: FamilyData, System: SystemSI},
	// Temperature — K and C are SI, F is US
	"K": {Suffix: "K", Family: FamilyTemperature, System: SystemSI},
	"C": {Suffix: "C", Family: FamilyTemperature, System: SystemSI},
	"F": {Suffix: "F", Family: FamilyTemperature, System: SystemUS},
	// Volume — SI
	"mL": {Suffix: "mL", Family: FamilyVolume, System: SystemSI},
	"L":  {Suffix: "L", Family: FamilyVolume, System: SystemSI},
	// Volume — US
	"floz": {Suffix: "floz", Family: FamilyVolume, System: SystemUS},
	"cup":  {Suffix: "cup", Family: FamilyVolume, System: SystemUS},
	"pt":   {Suffix: "pt", Family: FamilyVolume, System: SystemUS},
	"qt":   {Suffix: "qt", Family: FamilyVolume, System: SystemUS},
	"gal":  {Suffix: "gal", Family: FamilyVolume, System: SystemUS},
}

// LookupUnitSuffix returns the UnitInfo for a suffix, or ok=false if unknown.
func LookupUnitSuffix(suffix string) (UnitInfo, bool) {
	info, ok := unitSuffixTable[suffix]
	return info, ok
}

// --- SI sub-unit conversion tables ---
// These map a suffix to the number of sub-units that one display-unit represents.
// SI Length: sub-unit = micrometre (µm). 1 m = 1,000,000 µm.
// SI Mass: sub-unit = milligram (mg). 1 g = 1,000 mg.
// SI Data: sub-unit = byte (B). 1 B = 1.

var siSubUnitsPerUnit = map[string]int64{
	// Length (base sub-unit: µm)
	"mm": 1_000,         // 1 mm = 1,000 µm
	"cm": 10_000,        // 1 cm = 10,000 µm
	"m":  1_000_000,     // 1 m  = 1,000,000 µm
	"km": 1_000_000_000, // 1 km = 1,000,000,000 µm
	// Mass (base sub-unit: mg)
	"mg": 1,         // 1 mg = 1 mg
	"g":  1_000,     // 1 g  = 1,000 mg
	"kg": 1_000_000, // 1 kg = 1,000,000 mg
	// Data (base sub-unit: byte)
	"B":   1,
	"kB":  1_000,
	"MB":  1_000_000,
	"GB":  1_000_000_000,
	"TB":  1_000_000_000_000,
	"KiB": 1 << 10, // 1,024
	"MiB": 1 << 20, // 1,048,576
	"GiB": 1 << 30, // 1,073,741,824
	"TiB": 1 << 40, // 1,099,511,627,776
	// Volume (base sub-unit: µL = microlitre)
	"mL": 1_000,     // 1 mL = 1,000 µL
	"L":  1_000_000, // 1 L  = 1,000,000 µL
}

// SISubUnitsPerUnit returns how many sub-units one display-unit represents.
func SISubUnitsPerUnit(suffix string) int64 {
	return siSubUnitsPerUnit[suffix]
}

// --- US Customary sub-unit conversion tables ---
// These map a suffix to the number of HCN sub-units per one display-unit.
// Length base: yard. 1 yard = HCN sub-yards = 725,760.
// Mass base: ounce. 1 oz = HCN sub-ounces = 725,760.

// SubUnitsPerInch is HCN / 36 = 20,160 sub-yards per inch.
const SubUnitsPerInch int64 = HCN / 36

var usSubUnitsPerUnit = map[string]int64{
	// Length (base sub-unit: sub-yard, 1 yd = HCN)
	"in": HCN / 36,   // 20,160 sub-yards per inch
	"ft": HCN / 3,    // 241,920 sub-yards per foot (12 inches)
	"yd": HCN,        // 725,760 sub-yards per yard
	"mi": HCN * 1760, // 1 mile = 1,760 yards
	// Mass (base sub-unit: sub-ounce, 1 oz = HCN)
	"oz": HCN,      // 725,760 sub-ounces per ounce
	"lb": HCN * 16, // 1 lb = 16 oz
	// Volume (base sub-unit: sub-floz, 1 floz = HCN)
	"floz": HCN,       // 725,760 sub-floz per fluid ounce
	"cup":  HCN * 8,   // 1 cup = 8 floz = 5,806,080
	"pt":   HCN * 16,  // 1 pt  = 16 floz = 11,612,160
	"qt":   HCN * 32,  // 1 qt  = 32 floz = 23,224,320
	"gal":  HCN * 128, // 1 gal = 128 floz = 92,897,280
}

// USSubUnitsPerUnit returns how many HCN sub-units one display-unit represents.
func USSubUnitsPerUnit(suffix string) int64 {
	return usSubUnitsPerUnit[suffix]
}

// --- Cross-system bridge constants ---
// These are used to convert between SI and US Customary systems.
//
// Length bridge: 1 inch = 0.0254 m = 25,400 µm (exact by international definition)
// In US sub-units: 1 inch = HCN/36 = 20,160 sub-yards
// Bridge: 25,400 µm per 20,160 sub-yards
//
// Simplified ratio for conversion:
//   SI→US: us_sub = si_µm * 20,160 / 25,400 = si_µm * 2016 / 2540 = si_µm * 504 / 635
//   US→SI: si_µm = us_sub * 25,400 / 20,160 = us_sub * 2540 / 2016 = us_sub * 635 / 504

const (
	// Length bridge ratio (simplified GCD)
	LengthBridgeSINumerator   int64 = 635 // µm per sub-yard numerator
	LengthBridgeSIDenominator int64 = 504 // µm per sub-yard denominator
	LengthBridgeUSNumerator   int64 = 504 // sub-yards per µm numerator
	LengthBridgeUSDenominator int64 = 635 // sub-yards per µm denominator
)

// Mass bridge: 1 lb = 453.59237 g = 453,592.37 mg
// 1 lb = 16 oz, so 1 oz = 453,592.37 / 16 = 28,349.523125 mg
// In US sub-units: 1 oz = HCN = 725,760 sub-ounces
// Bridge: 28,349,523.125 µg per 725,760 sub-ounces ... but we need integer math.
//
// More precisely: 1 lb = 453.59237 g exactly = 453,592.37 mg exactly.
// That's 45,359,237 / 100 mg per lb.
// 1 lb = 16 * HCN = 11,612,160 sub-ounces
// So: 1 sub-ounce = 45,359,237 / (100 * 11,612,160) mg
//                  = 45,359,237 / 1,161,216,000 mg
// Simplify: GCD(45359237, 1161216000) ... 45359237 is prime? Let's use the ratio directly.
//
// US→SI: si_mg = us_sub * 45_359_237 / (100 * 16 * HCN)
//       = us_sub * 45_359_237 / 1_161_216_000
// SI→US: us_sub = si_mg * 1_161_216_000 / 45_359_237

const (
	MassBridgeSINumerator   int64 = 45_359_237    // mg per (16*HCN) sub-ounces
	MassBridgeSIDenominator int64 = 1_161_216_000 // = 100 * 16 * HCN
	MassBridgeUSNumerator   int64 = 1_161_216_000
	MassBridgeUSDenominator int64 = 45_359_237
)

// Volume bridge: 1 gal = 3.785411784 L (exact, from 1 gal = 231 in³)
// In sub-units: 1 gal = 128 × HCN = 92,897,280 sub-floz = 3,785,411,784 µL
// GCD(3785411784, 92897280) = 168
// Simplified: 22,532,213 / 552,960
const (
	VolumeBridgeSINumerator   int64 = 22_532_213 // µL per sub-floz (numerator)
	VolumeBridgeSIDenominator int64 = 552_960    // µL per sub-floz (denominator)
	VolumeBridgeUSNumerator   int64 = 552_960    // sub-floz per µL (numerator)
	VolumeBridgeUSDenominator int64 = 22_532_213 // sub-floz per µL (denominator)
)

// ConvertUSToSI converts a US Customary amount (in sub-units) to SI sub-units.
func ConvertUSToSI(usAmount int64, family string) int64 {
	switch family {
	case FamilyLength:
		return usAmount * LengthBridgeSINumerator / LengthBridgeSIDenominator
	case FamilyMass:
		return usAmount * MassBridgeSINumerator / MassBridgeSIDenominator
	case FamilyVolume:
		return usAmount * VolumeBridgeSINumerator / VolumeBridgeSIDenominator
	default:
		return 0
	}
}

// ConvertSIToUS converts an SI amount (in sub-units) to US Customary sub-units.
func ConvertSIToUS(siAmount int64, family string) int64 {
	switch family {
	case FamilyLength:
		return siAmount * LengthBridgeUSNumerator / LengthBridgeUSDenominator
	case FamilyMass:
		return siAmount * MassBridgeUSNumerator / MassBridgeUSDenominator
	case FamilyVolume:
		return siAmount * VolumeBridgeUSNumerator / VolumeBridgeUSDenominator
	default:
		return 0
	}
}

// --- Display configuration ---

// SIDefaultDecimalPlaces returns the default number of decimal places for display.
func SIDefaultDecimalPlaces(suffix string) int {
	switch suffix {
	case "m":
		return 2
	case "cm":
		return 1
	case "mm":
		return 0
	case "km":
		return 2
	case "kg":
		return 2
	case "g":
		return 0
	case "mg":
		return 0
	case "B", "kB", "MB", "GB", "TB", "KiB", "MiB", "GiB", "TiB":
		return 0
	// Temperature
	case "C":
		return 1
	case "F":
		return 1
	case "K":
		return 2
	// Volume — SI
	case "mL":
		return 0
	case "L":
		return 2
	default:
		return 2
	}
}

// CommonFractionDenominators is the set of denominators that display as fractions
// in US Customary output. If GCD reduction yields a denominator not in this set,
// the value falls back to decimal display.
var CommonFractionDenominators = map[int64]bool{
	2: true, 3: true, 4: true, 5: true, 6: true, 7: true, 8: true,
	10: true, 12: true, 16: true, 32: true, 64: true,
}

// GCD computes the greatest common divisor of two non-negative integers.
func GCD(a, b int64) int64 {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// --- Constructor name tables ---

// UnitConstructorNames maps constructor function names to their target suffix.
// Plural forms are primary; SI spelling primary, US spelling alias.
var UnitConstructorNames = map[string]string{
	// Length — SI
	"millimetres": "mm",
	"millimeters": "mm",
	"centimetres": "cm",
	"centimeters": "cm",
	"metres":      "m",
	"meters":      "m",
	"kilometres":  "km",
	"kilometers":  "km",
	// Length — US
	"inches": "in",
	"feet":   "ft",
	"yards":  "yd",
	"miles":  "mi",
	// Mass — SI
	"milligrams": "mg",
	"grams":      "g",
	"kilograms":  "kg",
	// Mass — US
	"ounces": "oz",
	"pounds": "lb",
	// Data
	"bytes":     "B",
	"kilobytes": "kB",
	"megabytes": "MB",
	"gigabytes": "GB",
	"terabytes": "TB",
	"kibibytes": "KiB",
	"mebibytes": "MiB",
	"gibibytes": "GiB",
	"tebibytes": "TiB",
	// Temperature
	"celsius":    "C",
	"fahrenheit": "F",
	"kelvins":    "K",
	// Volume — SI
	"millilitres": "mL",
	"milliliters": "mL",
	"litres":      "L",
	"liters":      "L",
	// Volume — US
	"fluidounces": "floz",
	"cups":        "cup",
	"pints":       "pt",
	"quarts":      "qt",
	"gallons":     "gal",
}

// AllUnitSuffixes returns all known unit suffixes for fuzzy matching in errors.
func AllUnitSuffixes() []string {
	suffixes := make([]string, 0, len(unitSuffixTable))
	for s := range unitSuffixTable {
		suffixes = append(suffixes, s)
	}
	return suffixes
}
