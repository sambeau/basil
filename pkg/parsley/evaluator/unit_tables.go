// Package evaluator provides unit measurement tables for FEAT-118.
// This file defines suffix metadata, sub-unit bases, within-system ratios,
// cross-system bridge constants, and display configuration for measurement units.
package evaluator

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
	FamilyLength = "length"
	FamilyMass   = "mass"
	FamilyData   = "data"
)

// UnitInfo describes a unit suffix's metadata.
type UnitInfo struct {
	Suffix string // the suffix string (e.g., "m", "in", "kg")
	Family string // "length", "mass", "data"
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

// ConvertUSToSI converts a US Customary amount (in sub-units) to SI sub-units.
// family must be "length" or "mass".
func ConvertUSToSI(usAmount int64, family string) int64 {
	switch family {
	case FamilyLength:
		// us_sub * 635 / 504
		return usAmount * LengthBridgeSINumerator / LengthBridgeSIDenominator
	case FamilyMass:
		// us_sub * 45,359,237 / 1,161,216,000
		return usAmount * MassBridgeSINumerator / MassBridgeSIDenominator
	default:
		return 0
	}
}

// ConvertSIToUS converts an SI amount (in sub-units) to US Customary sub-units.
// family must be "length" or "mass".
func ConvertSIToUS(siAmount int64, family string) int64 {
	switch family {
	case FamilyLength:
		// si_µm * 504 / 635
		return siAmount * LengthBridgeUSNumerator / LengthBridgeUSDenominator
	case FamilyMass:
		// si_mg * 1,161,216,000 / 45,359,237
		return siAmount * MassBridgeUSNumerator / MassBridgeUSDenominator
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
}

// AllUnitSuffixes returns all known unit suffixes for fuzzy matching in errors.
func AllUnitSuffixes() []string {
	suffixes := make([]string, 0, len(unitSuffixTable))
	for s := range unitSuffixTable {
		suffixes = append(suffixes, s)
	}
	return suffixes
}
