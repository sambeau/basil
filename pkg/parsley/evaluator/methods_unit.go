// Package evaluator provides unit method implementations via declarative registry.
// This file implements unit methods for FEAT-118: Measurement Units
package evaluator

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// UnitMethodRegistry defines all methods available on unit values.
var UnitMethodRegistry MethodRegistry

func init() {
	UnitMethodRegistry = MethodRegistry{
		"to": {
			Fn:          unitTo,
			Arity:       "1",
			Description: "Convert to another unit",
		},
		"abs": {
			Fn:          unitAbs,
			Arity:       "0",
			Description: "Absolute value",
		},
		"fmt": {
			Fn:          unitFmt,
			Arity:       "0-2",
			Description: "Format with style and/or locale",
		},
		"format": {
			Fn:          unitFormatMethod,
			Arity:       "0-1",
			Description: "Format with optional precision (legacy, use fmt for style)",
		},
		"short": {
			Fn:          unitShort,
			Arity:       "0-1",
			Description: "Compact format (5m)",
		},
		"medium": {
			Fn:          unitMedium,
			Arity:       "0-1",
			Description: "Standard format with precision (5.00m)",
		},
		"long": {
			Fn:          unitLong,
			Arity:       "0-1",
			Description: "Spelled out unit name (5 metres)",
		},
		"full": {
			Fn:          unitFull,
			Arity:       "0-1",
			Description: "With conversion (5 metres (16.4 ft))",
		},
		"repr": {
			Fn:          unitRepr,
			Arity:       "0",
			Description: "Get parseable literal string",
		},
		"toDict": {
			Fn:          unitToDict,
			Arity:       "0",
			Description: "Convert to dictionary",
		},
		"inspect": {
			Fn:          unitInspectMethod,
			Arity:       "0",
			Description: "Get debug dictionary with internal values",
		},
		"toFraction": {
			Fn:          unitToFraction,
			Arity:       "0",
			Description: "Get fraction string for US Customary values",
		},
		"toJSON": {
			Fn:          unitToJSON,
			Arity:       "0",
			Description: "Convert to JSON string",
		},
		"toBox": {
			Fn:          unitToBox,
			Arity:       "0",
			Description: "Render as box diagram",
		},
	}
	RegisterMethodRegistry("unit", UnitMethodRegistry)
}

// evalUnitProperty handles property access on Unit values.
func evalUnitProperty(unit *Unit, key string) Object {
	switch key {
	case "value":
		if IsTemperatureFamily(unit.Family) {
			return &Float{Value: DecodeTempFromSubK(unit.Amount, unit.DisplayHint)}
		}
		return &Float{Value: unitDisplayValue(unit)}
	case "unit":
		return &String{Value: unit.DisplayHint}
	case "family":
		return &String{Value: unit.Family}
	case "system":
		return &String{Value: unit.System}
	case "max":
		return unitMaxProperty(unit)
	case "min":
		return unitMinProperty(unit)
	default:
		methodNames := UnitMethodRegistry.Names()
		if _, ok := UnitMethodRegistry[key]; ok {
			return methodAsPropertyError(key, "Unit")
		}
		return unknownMethodError(key, "unit", append([]string{"value", "unit", "family", "system", "max", "min"}, methodNames...))
	}
}

// unitMaxProperty returns the maximum representable value at full (Scale=0) precision
// for the given unit suffix. The result is a Unit with the same suffix.
func unitMaxProperty(unit *Unit) Object {
	if IsTemperatureFamily(unit.Family) {
		return unitTempMax(unit)
	}

	var subPerUnit int64
	if unit.System == SystemUS {
		subPerUnit = USSubUnitsPerUnit(unit.DisplayHint)
	} else {
		subPerUnit = SISubUnitsPerUnit(unit.DisplayHint)
	}
	if subPerUnit == 0 {
		subPerUnit = 1
	}

	// Largest exact multiple of subPerUnit that fits in int64
	maxAmount := math.MaxInt64 / subPerUnit * subPerUnit
	return &Unit{
		Amount:      maxAmount,
		Family:      unit.Family,
		System:      unit.System,
		DisplayHint: unit.DisplayHint,
	}
}

// unitMinProperty returns the smallest representable positive value for the given unit suffix.
// This is 1 base sub-unit expressed in the receiver's suffix.
func unitMinProperty(unit *Unit) Object {
	if IsTemperatureFamily(unit.Family) {
		return unitTempMin(unit)
	}

	return &Unit{
		Amount:      1,
		Family:      unit.Family,
		System:      unit.System,
		DisplayHint: unit.DisplayHint,
	}
}

// unitTempMax returns the maximum representable temperature for the given scale.
// Derived from int64 max of the sub-kelvin representation.
func unitTempMax(unit *Unit) Object {
	// Max sub-kelvin value is math.MaxInt64. Convert to the display scale.
	// For K: max = MaxInt64 / TempScaleK (in kelvins)
	// For C: max = (MaxInt64 - TempOffsetC) / TempScaleC ... but we just want the largest value
	//        that fits when encoded. Actually, since sub-kelvins can go up to MaxInt64,
	//        the max displayable value = DecodeTempFromSubK(MaxInt64, suffix).
	return &Unit{
		Amount:      math.MaxInt64,
		Family:      FamilyTemperature,
		System:      unit.System,
		DisplayHint: unit.DisplayHint,
	}
}

// unitTempMin returns the minimum representable temperature for the given scale.
// For K: smallest positive sub-kelvin increment (1 sub-K).
// For C/F: derived from sub-kelvin floor (0 sub-K = absolute zero).
func unitTempMin(unit *Unit) Object {
	switch unit.DisplayHint {
	case "K":
		// Smallest positive Kelvin: 1 sub-K = 1/900 K
		return &Unit{
			Amount:      1,
			Family:      FamilyTemperature,
			System:      unit.System,
			DisplayHint: "K",
		}
	case "C":
		// Minimum Celsius: 0 sub-K → absolute zero in Celsius
		// 0 sub-K decoded as C = (0 - TempOffsetC) / TempScaleC = -273.15
		return &Unit{
			Amount:      0,
			Family:      FamilyTemperature,
			System:      unit.System,
			DisplayHint: "C",
		}
	case "F":
		// Minimum Fahrenheit: 0 sub-K → absolute zero in Fahrenheit
		return &Unit{
			Amount:      0,
			Family:      FamilyTemperature,
			System:      unit.System,
			DisplayHint: "F",
		}
	default:
		return &Unit{
			Amount:      0,
			Family:      FamilyTemperature,
			System:      unit.System,
			DisplayHint: unit.DisplayHint,
		}
	}
}

// evalUnitMethod evaluates a method call on a Unit value using the registry.
func evalUnitMethod(unit *Unit, method string, args []Object) Object {
	result := dispatchFromRegistry(UnitMethodRegistry, unit, method, args, nil)
	if result != nil {
		return result
	}
	return unknownMethodError(method, "unit", UnitMethodRegistry.Names())
}

// --- Method implementations ---

// unitTo converts a unit value to a different unit suffix.
// Usage: #1mi.to("km"), #100cm.to("in"), #100C.to("F")
func unitTo(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)
	targetStr, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "to", "a string (unit suffix)", args[0].Type())
	}
	target := targetStr.Value

	targetInfo, found := LookupUnitSuffix(target)
	if !found {
		// Try fuzzy match for suggestions
		suffixes := AllUnitSuffixes()
		suggestion := findClosestSuffix(target, suffixes)
		if suggestion != "" {
			return newStructuredError("UNIT-0007", map[string]any{
				"Suffix":     target,
				"Suggestion": suggestion,
			})
		}
		return newStructuredError("UNIT-0007", map[string]any{
			"Suffix":     target,
			"Suggestion": "m, cm, km, in, ft, etc.",
		})
	}

	if targetInfo.Family != unit.Family {
		return newStructuredError("UNIT-0006", map[string]any{
			"FromFamily":  unit.Family,
			"ToFamily":    targetInfo.Family,
			"Constructor": target,
			"Example":     exampleForFamily(targetInfo.Family),
		})
	}

	return convertUnit(unit, target, targetInfo)
}

// convertUnit converts a unit value to a target suffix within the same family.
// Scale-aware: propagates Scale through within-system and cross-system conversions.
func convertUnit(unit *Unit, targetSuffix string, targetInfo UnitInfo) *Unit {
	// Temperature conversion: sub-kelvins are absolute, just change the display hint
	if IsTemperatureFamily(unit.Family) {
		return &Unit{
			Amount:      unit.Amount,
			Family:      unit.Family,
			System:      targetInfo.System,
			DisplayHint: targetSuffix,
		}
	}

	var newAmount int64
	newScale := unit.Scale

	switch {
	case unit.System == targetInfo.System:
		// Within-system conversion: simple ratio
		if unit.System == SystemSI {
			srcSub := SISubUnitsPerUnit(unit.DisplayHint)
			dstSub := SISubUnitsPerUnit(targetSuffix)
			if srcSub == 0 || dstSub == 0 {
				return &Unit{Amount: 0, Family: unit.Family, System: targetInfo.System, DisplayHint: targetSuffix}
			}
			// amount is in base sub-units (µm/mg/B/nL/mm²), just change the display hint
			newAmount = unit.Amount
		} else {
			// US to US: same sub-unit base (HCN sub-yards or sub-ounces or sub-floz or in²)
			newAmount = unit.Amount
		}
	case unit.System == SystemSI && targetInfo.System == SystemUS:
		// SI → US conversion (scale-aware)
		newAmount, newScale = ConvertSIToUSScaled(unit.Amount, unit.Scale, unit.Family)
	default:
		// US → SI conversion (scale-aware)
		newAmount, newScale = ConvertUSToSIScaled(unit.Amount, unit.Scale, unit.Family)
	}

	return &Unit{
		Amount:      newAmount,
		Family:      unit.Family,
		System:      targetInfo.System,
		DisplayHint: targetSuffix,
		Scale:       newScale,
	}
}

// unitAbs returns the absolute value of a unit.
// For temperature, operates on the decoded display value (not raw sub-kelvins).
// Scale-aware: preserves Scale on the result.
func unitAbs(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)

	if IsTemperatureFamily(unit.Family) {
		// Abs of the display value: if decoded value is negative, re-encode the positive
		decoded := DecodeTempFromSubK(unit.Amount, unit.DisplayHint)
		if decoded < 0 {
			return &Unit{
				Amount:      EncodeTempToSubK(-decoded, unit.DisplayHint),
				Family:      unit.Family,
				System:      unit.System,
				DisplayHint: unit.DisplayHint,
			}
		}
		return unit
	}

	amount := unit.Amount
	if amount < 0 {
		amount = -amount
	}
	return &Unit{
		Amount:      amount,
		Family:      unit.Family,
		System:      unit.System,
		DisplayHint: unit.DisplayHint,
		Scale:       unit.Scale,
	}
}

// unitFmt formats a unit value with style and locale options.
// Overloads:
//   - .fmt() - medium style, default locale
//   - .fmt(n) - precision (decimal places)
//   - .fmt("style") - named style
//   - .fmt("style", "locale") - style + locale
//   - .fmt({...}) - options dictionary
func unitFmt(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)
	opts, err := parseFormatArgs(args)
	if err != nil {
		return err
	}
	return formatUnitWithOpts(unit, opts)
}

func unitShort(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)
	opts, err := parseStyleMethodArgs("short", args, "short")
	if err != nil {
		return err
	}
	return formatUnitWithOpts(unit, opts)
}

func unitMedium(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)
	opts, err := parseStyleMethodArgs("medium", args, "medium")
	if err != nil {
		return err
	}
	return formatUnitWithOpts(unit, opts)
}

func unitLong(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)
	opts, err := parseStyleMethodArgs("long", args, "long")
	if err != nil {
		return err
	}
	return formatUnitWithOpts(unit, opts)
}

func unitFull(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)
	opts, err := parseStyleMethodArgs("full", args, "full")
	if err != nil {
		return err
	}
	return formatUnitWithOpts(unit, opts)
}

// formatUnitWithOpts formats a unit using FormatOpts.
func formatUnitWithOpts(unit *Unit, opts FormatOpts) Object {
	switch opts.Style {
	case "short":
		return formatUnitShort(unit, opts)
	case "medium":
		return formatUnitMedium(unit, opts)
	case "long":
		return formatUnitLongStyle(unit, opts)
	case "full":
		return formatUnitFull(unit, opts)
	default:
		return formatUnitMedium(unit, opts)
	}
}

// formatUnitShort returns compact format: "5m", "10kg"
func formatUnitShort(unit *Unit, opts FormatOpts) Object {
	var value float64
	if IsTemperatureFamily(unit.Family) {
		value = DecodeTempFromSubK(unit.Amount, unit.DisplayHint)
	} else {
		value = unitDisplayValue(unit)
	}

	// For short format, show minimal decimal places
	formatted := formatUnitValue(value, 0, opts.Locale)
	return &String{Value: formatted + unit.DisplayHint}
}

// formatUnitMedium returns standard format with precision: "5.00m"
func formatUnitMedium(unit *Unit, opts FormatOpts) Object {
	precision := opts.Precision
	if precision < 0 {
		precision = 2 // default
	}

	var value float64
	if IsTemperatureFamily(unit.Family) {
		value = DecodeTempFromSubK(unit.Amount, unit.DisplayHint)
	} else {
		value = unitDisplayValue(unit)
	}

	formatted := formatUnitValue(value, precision, opts.Locale)
	return &String{Value: formatted + unit.DisplayHint}
}

// formatUnitLongStyle returns spelled out unit: "5 metres"
func formatUnitLongStyle(unit *Unit, opts FormatOpts) Object {
	var value float64
	if IsTemperatureFamily(unit.Family) {
		value = DecodeTempFromSubK(unit.Amount, unit.DisplayHint)
	} else {
		value = unitDisplayValue(unit)
	}

	precision := opts.Precision
	if precision < 0 {
		precision = 2
	}

	formatted := formatUnitValue(value, precision, opts.Locale)
	unitName := getUnitName(unit.DisplayHint, opts.Locale, value != 1.0)
	return &String{Value: formatted + " " + unitName}
}

// formatUnitFull returns with conversion: "5 metres (16.4 ft)"
func formatUnitFull(unit *Unit, opts FormatOpts) Object {
	var value float64
	if IsTemperatureFamily(unit.Family) {
		value = DecodeTempFromSubK(unit.Amount, unit.DisplayHint)
	} else {
		value = unitDisplayValue(unit)
	}

	precision := opts.Precision
	if precision < 0 {
		precision = 2
	}

	formatted := formatUnitValue(value, precision, opts.Locale)
	unitName := getUnitName(unit.DisplayHint, opts.Locale, value != 1.0)

	// Add conversion to other system
	convStr := getUnitConversion(unit, opts.Locale)
	if convStr != "" {
		return &String{Value: formatted + " " + unitName + " (" + convStr + ")"}
	}
	return &String{Value: formatted + " " + unitName}
}

// formatUnitValue formats the numeric value with locale-aware separators.
func formatUnitValue(value float64, precision int, locale string) string {
	negative := value < 0
	if negative {
		value = -value
	}

	thousandSep, decimalSep := getLocaleSeparators(locale)

	// Format with precision
	intPart := int64(value)
	var result string

	// Add thousand separators for large numbers
	if intPart >= 1000 {
		result = formatIntegerWithSeparator(intPart, thousandSep)
	} else {
		result = fmt.Sprintf("%d", intPart)
	}

	if precision > 0 {
		decPart := value - float64(intPart)
		decStr := fmt.Sprintf("%.*f", precision, decPart)
		if len(decStr) > 2 {
			result += decimalSep + decStr[2:]
		}
	}

	if negative {
		result = "-" + result
	}
	return result
}

// getUnitName returns the spelled-out unit name.
func getUnitName(suffix, locale string, plural bool) string {
	// Unit names with regional spelling variations
	names := map[string]map[string][2]string{
		// Length - SI
		"mm": {"en-US": {"millimeter", "millimeters"}, "en-GB": {"millimetre", "millimetres"}, "de-DE": {"Millimeter", "Millimeter"}},
		"cm": {"en-US": {"centimeter", "centimeters"}, "en-GB": {"centimetre", "centimetres"}, "de-DE": {"Zentimeter", "Zentimeter"}},
		"m":  {"en-US": {"meter", "meters"}, "en-GB": {"metre", "metres"}, "de-DE": {"Meter", "Meter"}},
		"km": {"en-US": {"kilometer", "kilometers"}, "en-GB": {"kilometre", "kilometres"}, "de-DE": {"Kilometer", "Kilometer"}},
		// Length - US
		"in": {"en-US": {"inch", "inches"}, "en-GB": {"inch", "inches"}},
		"ft": {"en-US": {"foot", "feet"}, "en-GB": {"foot", "feet"}},
		"yd": {"en-US": {"yard", "yards"}, "en-GB": {"yard", "yards"}},
		"mi": {"en-US": {"mile", "miles"}, "en-GB": {"mile", "miles"}},
		// Mass - SI
		"mg": {"en-US": {"milligram", "milligrams"}, "en-GB": {"milligram", "milligrams"}, "de-DE": {"Milligramm", "Milligramm"}},
		"g":  {"en-US": {"gram", "grams"}, "en-GB": {"gram", "grams"}, "de-DE": {"Gramm", "Gramm"}},
		"kg": {"en-US": {"kilogram", "kilograms"}, "en-GB": {"kilogram", "kilograms"}, "de-DE": {"Kilogramm", "Kilogramm"}},
		// Mass - US
		"oz": {"en-US": {"ounce", "ounces"}, "en-GB": {"ounce", "ounces"}},
		"lb": {"en-US": {"pound", "pounds"}, "en-GB": {"pound", "pounds"}},
		// Temperature
		"C": {"en-US": {"degree Celsius", "degrees Celsius"}, "en-GB": {"degree Celsius", "degrees Celsius"}, "de-DE": {"Grad Celsius", "Grad Celsius"}},
		"F": {"en-US": {"degree Fahrenheit", "degrees Fahrenheit"}, "en-GB": {"degree Fahrenheit", "degrees Fahrenheit"}},
		"K": {"en-US": {"kelvin", "kelvins"}, "en-GB": {"kelvin", "kelvins"}},
		// Volume - SI
		"mL": {"en-US": {"milliliter", "milliliters"}, "en-GB": {"millilitre", "millilitres"}},
		"L":  {"en-US": {"liter", "liters"}, "en-GB": {"litre", "litres"}},
		// Volume - US
		"floz": {"en-US": {"fluid ounce", "fluid ounces"}, "en-GB": {"fluid ounce", "fluid ounces"}},
		"cup":  {"en-US": {"cup", "cups"}, "en-GB": {"cup", "cups"}},
		"pt":   {"en-US": {"pint", "pints"}, "en-GB": {"pint", "pints"}},
		"qt":   {"en-US": {"quart", "quarts"}, "en-GB": {"quart", "quarts"}},
		"gal":  {"en-US": {"gallon", "gallons"}, "en-GB": {"gallon", "gallons"}},
		// Area - SI
		"mm2": {"en-US": {"square millimeter", "square millimeters"}, "en-GB": {"square millimetre", "square millimetres"}},
		"cm2": {"en-US": {"square centimeter", "square centimeters"}, "en-GB": {"square centimetre", "square centimetres"}},
		"m2":  {"en-US": {"square meter", "square meters"}, "en-GB": {"square metre", "square metres"}},
		"km2": {"en-US": {"square kilometer", "square kilometers"}, "en-GB": {"square kilometre", "square kilometres"}},
		// Area - US
		"in2": {"en-US": {"square inch", "square inches"}, "en-GB": {"square inch", "square inches"}},
		"ft2": {"en-US": {"square foot", "square feet"}, "en-GB": {"square foot", "square feet"}},
		"yd2": {"en-US": {"square yard", "square yards"}, "en-GB": {"square yard", "square yards"}},
		"ac":  {"en-US": {"acre", "acres"}, "en-GB": {"acre", "acres"}},
		"mi2": {"en-US": {"square mile", "square miles"}, "en-GB": {"square mile", "square miles"}},
	}

	if localeNames, ok := names[suffix]; ok {
		// Try exact locale match
		if pair, ok := localeNames[locale]; ok {
			if plural {
				return pair[1]
			}
			return pair[0]
		}
		// Try language-only match (e.g., "en" for "en-AU")
		lang := locale
		if idx := strings.Index(locale, "-"); idx > 0 {
			lang = locale[:idx]
		}
		for loc, pair := range localeNames {
			if strings.HasPrefix(loc, lang) {
				if plural {
					return pair[1]
				}
				return pair[0]
			}
		}
		// Fall back to en-US
		if pair, ok := localeNames["en-US"]; ok {
			if plural {
				return pair[1]
			}
			return pair[0]
		}
	}

	// Fallback: just return the suffix
	return suffix
}

// getUnitConversion returns a conversion string to the other measurement system.
func getUnitConversion(unit *Unit, locale string) string {
	if IsTemperatureFamily(unit.Family) {
		return getTemperatureConversion(unit, locale)
	}

	// Convert to the other system
	var targetSuffix string
	var converted *Unit

	if unit.System == SystemSI {
		// Convert to US
		switch unit.Family {
		case FamilyLength:
			targetSuffix = "ft"
		case FamilyMass:
			targetSuffix = "lb"
		case FamilyVolume:
			targetSuffix = "gal"
		case FamilyArea:
			targetSuffix = "ft2"
		default:
			return ""
		}
	} else {
		// Convert to SI
		switch unit.Family {
		case FamilyLength:
			targetSuffix = "m"
		case FamilyMass:
			targetSuffix = "kg"
		case FamilyVolume:
			targetSuffix = "L"
		case FamilyArea:
			targetSuffix = "m2"
		default:
			return ""
		}
	}

	targetInfo, found := LookupUnitSuffix(targetSuffix)
	if !found {
		return ""
	}

	converted = convertUnit(unit, targetSuffix, targetInfo)
	if converted == nil {
		return ""
	}

	value := unitDisplayValue(converted)
	formatted := formatUnitValue(value, 1, locale)
	return formatted + " " + targetSuffix
}

// getTemperatureConversion returns conversion for temperature units.
func getTemperatureConversion(unit *Unit, locale string) string {
	decoded := DecodeTempFromSubK(unit.Amount, unit.DisplayHint)

	var targetSuffix string
	var targetValue float64

	switch unit.DisplayHint {
	case "C":
		// Show Fahrenheit
		targetSuffix = "F"
		targetValue = decoded*9/5 + 32
	case "F":
		// Show Celsius
		targetSuffix = "C"
		targetValue = (decoded - 32) * 5 / 9
	case "K":
		// Show Celsius
		targetSuffix = "C"
		targetValue = decoded - 273.15
	default:
		return ""
	}

	formatted := formatUnitValue(targetValue, 1, locale)
	return formatted + targetSuffix
}

// unitToJSON returns a JSON representation of the unit.
func unitToJSON(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)
	var value float64
	if IsTemperatureFamily(unit.Family) {
		value = DecodeTempFromSubK(unit.Amount, unit.DisplayHint)
	} else {
		value = unitDisplayValue(unit)
	}

	// Format as JSON object
	json := fmt.Sprintf(`{"value":%g,"unit":"%s","family":"%s","system":"%s"}`,
		value, unit.DisplayHint, unit.Family, unit.System)
	return &String{Value: json}
}

// unitToBox renders the unit as an ASCII box diagram.
func unitToBox(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)
	br := NewBoxRenderer()
	return &String{Value: br.RenderSingleValue(unitInspectString(unit))}
}

// Legacy format method for backward compatibility with compound formats
// unitFormatMethod formats a unit value with optional precision or format string.
// Usage:
//
//	#12.3m.format()           - default format
//	#12.3m.format(4)          - with precision
//	#63.375in.format("ft-in") - compound feet-inches format
//	#37oz.format("lb-oz")     - compound pounds-ounces format
//	#63in.format("compound")  - auto-detect compound format
func unitFormatMethod(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)

	if len(args) == 0 {
		return &String{Value: unitFormat(unit, -1)}
	}

	switch arg := args[0].(type) {
	case *Integer:
		return &String{Value: unitFormat(unit, int(arg.Value))}
	case *Float:
		return &String{Value: unitFormat(unit, int(arg.Value))}
	case *String:
		// Check if it's a style name first
		if isStyleName(arg.Value) {
			opts := DefaultFormatOpts()
			opts.Style = arg.Value
			return formatUnitWithOpts(unit, opts)
		}
		// Compound format string
		return &String{Value: unitFormatCompound(unit, arg.Value)}
	default:
		return newTypeError("TYPE-0012", "format", "an integer (precision) or string (format)", args[0].Type())
	}
}

// unitFormatCompound formats a unit value using a compound format string.
// Supported formats:
//
//	"ft-in"    - feet and inches (e.g., 5' 3+3/8")
//	"lb-oz"    - pounds and ounces (e.g., 2lb 5oz)
//	"gal-qt-pt" - gallons, quarts, and pints (e.g., 1gal 2qt 1pt)
//	"L-mL"     - litres and millilitres (e.g., 1L 500mL)
//	"compound" - auto-detect based on unit family and system
func unitFormatCompound(u *Unit, formatStr string) string {
	switch formatStr {
	case "ft-in":
		return formatFeetInches(maybeConvertForFormat(u, "in"))
	case "lb-oz":
		return formatPoundsOunces(maybeConvertForFormat(u, "oz"))
	case "gal-qt-pt":
		return formatGalQuartPint(maybeConvertForFormat(u, "floz"))
	case "L-mL":
		return formatLitresMillilitres(maybeConvertForFormat(u, "mL"))
	case "compound":
		return formatCompoundAuto(u)
	default:
		// Unknown format, return default
		return unitInterpolationString(u)
	}
}

// maybeConvertForFormat converts a unit to the target suffix's system if needed.
// This allows e.g. #1.5m.format("ft-in") to convert to inches first, then format.
// Returns the original unit unchanged if it's already in the right system/family.
func maybeConvertForFormat(u *Unit, targetSuffix string) *Unit {
	targetInfo, found := LookupUnitSuffix(targetSuffix)
	if !found {
		return u
	}
	// Must be the same family
	if u.Family != targetInfo.Family {
		return u
	}
	// Already in the right system
	if u.System == targetInfo.System {
		return u
	}
	// Convert cross-system
	return convertUnit(u, targetSuffix, targetInfo)
}

// formatFeetInches formats a length unit as feet and inches.
// Example: #63.375in → "5' 3+3/8\""
func formatFeetInches(u *Unit) string {
	// Only works for US length (caller should convert SI→US beforehand)
	if u.Family != FamilyLength || u.System != SystemUS {
		return unitInterpolationString(u)
	}

	// Convert to inches
	subYardsPerInch := int64(HCN / 36)
	var totalSubYards int64
	if u.Scale > 0 {
		totalSubYards = int64(scaleTrueValue(u.Amount, u.Scale))
	} else {
		totalSubYards = u.Amount
	}

	// Handle negative values
	negative := totalSubYards < 0
	if negative {
		totalSubYards = -totalSubYards
	}

	// Convert to sub-inches then to inches and feet
	// totalInches = totalSubYards / subYardsPerInch
	feet := totalSubYards / (subYardsPerInch * 12)
	remainingSubYards := totalSubYards % (subYardsPerInch * 12)
	inchesSubYards := remainingSubYards

	// Build the result
	var result string
	if negative {
		result = "-"
	}

	if feet > 0 {
		result += fmt.Sprintf("%d'", feet)
	}

	// Format inches with fractions
	if inchesSubYards > 0 || feet == 0 {
		inchesWhole := inchesSubYards / subYardsPerInch
		inchesRemainder := inchesSubYards % subYardsPerInch

		if feet > 0 {
			result += " "
		}

		if inchesRemainder == 0 {
			result += fmt.Sprintf("%d\"", inchesWhole)
		} else {
			// Reduce to common fraction
			g := GCD(inchesRemainder, subYardsPerInch)
			fracNum := inchesRemainder / g
			fracDen := subYardsPerInch / g

			// Check if it's a common fraction denominator
			if CommonFractionDenominators[fracDen] {
				if inchesWhole > 0 {
					result += fmt.Sprintf("%d+%d/%d\"", inchesWhole, fracNum, fracDen)
				} else {
					result += fmt.Sprintf("%d/%d\"", fracNum, fracDen)
				}
			} else {
				// Decimal fallback
				inches := float64(inchesSubYards) / float64(subYardsPerInch)
				result += fmt.Sprintf("%.3f\"", inches)
			}
		}
	}

	return result
}

// formatPoundsOunces formats a mass unit as pounds and ounces.
// Example: #37oz → "2lb 5oz"
func formatPoundsOunces(u *Unit) string {
	// Only works for US mass (caller should convert SI→US beforehand)
	if u.Family != FamilyMass || u.System != SystemUS {
		return unitInterpolationString(u)
	}

	// US mass is stored in sub-ounces (HCN per ounce)
	var totalSubOunces int64
	if u.Scale > 0 {
		totalSubOunces = int64(scaleTrueValue(u.Amount, u.Scale))
	} else {
		totalSubOunces = u.Amount
	}

	// Handle negative values
	negative := totalSubOunces < 0
	if negative {
		totalSubOunces = -totalSubOunces
	}

	// 1 lb = 16 oz = 16 * HCN sub-ounces
	subOuncesPerLb := int64(HCN * 16)
	pounds := totalSubOunces / subOuncesPerLb
	remainingSubOunces := totalSubOunces % subOuncesPerLb
	ounces := remainingSubOunces / HCN
	ounceFrac := remainingSubOunces % HCN

	var result string
	if negative {
		result = "-"
	}

	if pounds > 0 {
		result += fmt.Sprintf("%dlb", pounds)
	}

	if ounces > 0 || ounceFrac > 0 || pounds == 0 {
		if pounds > 0 {
			result += " "
		}
		if ounceFrac == 0 {
			result += fmt.Sprintf("%doz", ounces)
		} else {
			// Include fractional ounces
			g := GCD(ounceFrac, HCN)
			fracNum := ounceFrac / g
			fracDen := HCN / g
			if CommonFractionDenominators[fracDen] {
				if ounces > 0 {
					result += fmt.Sprintf("%d+%d/%doz", ounces, fracNum, fracDen)
				} else {
					result += fmt.Sprintf("%d/%doz", fracNum, fracDen)
				}
			} else {
				totalOz := float64(remainingSubOunces) / float64(HCN)
				result += fmt.Sprintf("%.3foz", totalOz)
			}
		}
	}

	return result
}

// formatGalQuartPint formats a US volume as gallons, quarts, and pints.
// Example: #13pt → "1gal 2qt 1pt"
func formatGalQuartPint(u *Unit) string {
	// Only works for US volume (caller should convert SI→US beforehand)
	if u.Family != FamilyVolume || u.System != SystemUS {
		return unitInterpolationString(u)
	}

	var totalSubFloz int64
	if u.Scale > 0 {
		totalSubFloz = int64(scaleTrueValue(u.Amount, u.Scale))
	} else {
		totalSubFloz = u.Amount
	}

	negative := totalSubFloz < 0
	if negative {
		totalSubFloz = -totalSubFloz
	}

	// 1 gal = 128 floz, 1 qt = 32 floz, 1 pt = 16 floz
	subFlozPerGal := int64(HCN * 128)
	subFlozPerQt := int64(HCN * 32)
	subFlozPerPt := int64(HCN * 16)

	gallons := totalSubFloz / subFlozPerGal
	remaining := totalSubFloz % subFlozPerGal
	quarts := remaining / subFlozPerQt
	remaining = remaining % subFlozPerQt
	pints := remaining / subFlozPerPt
	remaining = remaining % subFlozPerPt
	floz := remaining / HCN
	flozFrac := remaining % HCN

	var result string
	if negative {
		result = "-"
	}

	parts := 0
	if gallons > 0 {
		result += fmt.Sprintf("%dgal", gallons)
		parts++
	}
	if quarts > 0 {
		if parts > 0 {
			result += " "
		}
		result += fmt.Sprintf("%dqt", quarts)
		parts++
	}
	if pints > 0 {
		if parts > 0 {
			result += " "
		}
		result += fmt.Sprintf("%dpt", pints)
		parts++
	}
	if floz > 0 || flozFrac > 0 || parts == 0 {
		if parts > 0 {
			result += " "
		}
		if flozFrac == 0 {
			result += fmt.Sprintf("%dfloz", floz)
		} else {
			g := GCD(flozFrac, HCN)
			fracNum := flozFrac / g
			fracDen := HCN / g
			if CommonFractionDenominators[fracDen] {
				if floz > 0 {
					result += fmt.Sprintf("%d+%d/%dfloz", floz, fracNum, fracDen)
				} else {
					result += fmt.Sprintf("%d/%dfloz", fracNum, fracDen)
				}
			} else {
				totalFloz := float64(remaining) / float64(HCN)
				result += fmt.Sprintf("%.3ffloz", totalFloz)
			}
		}
	}

	return result
}

// formatLitresMillilitres formats an SI volume as litres and millilitres.
// Example: #1500mL → "1L 500mL"
func formatLitresMillilitres(u *Unit) string {
	// Only works for SI volume (caller should convert US→SI beforehand)
	if u.Family != FamilyVolume || u.System != SystemSI {
		return unitInterpolationString(u)
	}

	// SI volume is stored in nanolitres (nL)
	var totalNL int64
	if u.Scale > 0 {
		totalNL = int64(scaleTrueValue(u.Amount, u.Scale))
	} else {
		totalNL = u.Amount
	}

	negative := totalNL < 0
	if negative {
		totalNL = -totalNL
	}

	nlPerML := int64(1_000_000)
	nlPerL := int64(1_000_000_000)

	litres := totalNL / nlPerL
	remaining := totalNL % nlPerL
	ml := remaining / nlPerML
	mlFrac := remaining % nlPerML

	var result string
	if negative {
		result = "-"
	}

	parts := 0
	if litres > 0 {
		result += fmt.Sprintf("%dL", litres)
		parts++
	}
	if ml > 0 || mlFrac > 0 || parts == 0 {
		if parts > 0 {
			result += " "
		}
		if mlFrac == 0 {
			result += fmt.Sprintf("%dmL", ml)
		} else {
			// Decimal fallback for sub-mL fractions
			totalML := float64(remaining) / float64(nlPerML)
			result += fmt.Sprintf("%.3fmL", totalML)
		}
	}

	return result
}

// formatCompoundAuto auto-detects the appropriate compound format.
func formatCompoundAuto(u *Unit) string {
	switch u.Family {
	case FamilyLength:
		if u.System == SystemUS {
			// Use feet-inches if >= 1 foot
			subYardsPerFoot := int64(HCN / 3)
			amount := u.Amount
			if u.Scale > 0 {
				amount = int64(scaleTrueValue(u.Amount, u.Scale))
			}
			if amount < 0 {
				amount = -amount
			}
			if amount >= subYardsPerFoot {
				return formatFeetInches(u)
			}
		}
	case FamilyMass:
		if u.System == SystemUS {
			// Use pounds-ounces if >= 1 pound
			subOuncesPerLb := int64(HCN * 16)
			amount := u.Amount
			if u.Scale > 0 {
				amount = int64(scaleTrueValue(u.Amount, u.Scale))
			}
			if amount < 0 {
				amount = -amount
			}
			if amount >= subOuncesPerLb {
				return formatPoundsOunces(u)
			}
		}
	case FamilyVolume:
		switch u.System {
		case SystemUS:
			// Use gal-qt-pt if >= 1 pint
			subFlozPerPt := int64(HCN * 16)
			amount := u.Amount
			if u.Scale > 0 {
				amount = int64(scaleTrueValue(u.Amount, u.Scale))
			}
			if amount < 0 {
				amount = -amount
			}
			if amount >= subFlozPerPt {
				return formatGalQuartPint(u)
			}
		case SystemSI:
			// Use L-mL if >= 1 litre
			nlPerL := int64(1_000_000_000)
			amount := u.Amount
			if u.Scale > 0 {
				amount = int64(scaleTrueValue(u.Amount, u.Scale))
			}
			if amount < 0 {
				amount = -amount
			}
			if amount >= nlPerL {
				return formatLitresMillilitres(u)
			}
		}
	}
	// Default: use standard format
	return unitInterpolationString(u)
}

// unitRepr returns a parseable literal string.
func unitRepr(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)
	return &String{Value: unitInspectString(unit)}
}

// unitToDict returns a dictionary with value, unit, family, system.
func unitToDict(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)
	var value float64
	if IsTemperatureFamily(unit.Family) {
		value = DecodeTempFromSubK(unit.Amount, unit.DisplayHint)
	} else {
		value = unitDisplayValue(unit)
	}
	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"value":  createLiteralExpression(&Float{Value: value}),
			"unit":   createLiteralExpression(&String{Value: unit.DisplayHint}),
			"family": createLiteralExpression(&String{Value: unit.Family}),
			"system": createLiteralExpression(&String{Value: unit.System}),
		},
		Env: NewEnvironment(),
	}
}

// unitInspectMethod returns a debug dictionary including internal representation.
func unitInspectMethod(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)
	pairs := map[string]ast.Expression{
		"__type":      createLiteralExpression(&String{Value: "unit"}),
		"amount":      createLiteralExpression(&Integer{Value: unit.Amount}),
		"family":      createLiteralExpression(&String{Value: unit.Family}),
		"system":      createLiteralExpression(&String{Value: unit.System}),
		"displayHint": createLiteralExpression(&String{Value: unit.DisplayHint}),
		"scale":       createLiteralExpression(&Integer{Value: int64(unit.Scale)}),
	}
	if IsTemperatureFamily(unit.Family) {
		pairs["subKelvins"] = createLiteralExpression(&Integer{Value: unit.Amount})
	}
	return &Dictionary{
		Pairs: pairs,
		Env:   NewEnvironment(),
	}
}

// unitToFraction returns a fraction string for US Customary values.
// For SI values, returns the decimal string.
// For temperature values, fractions don't apply — returns the decimal string.
func unitToFraction(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)
	if IsTemperatureFamily(unit.Family) {
		return &String{Value: unitInterpolationString(unit)}
	}
	if unit.System != SystemUS {
		// SI values: return decimal string
		return &String{Value: unitInterpolationString(unit)}
	}

	subPerUnit := USSubUnitsPerUnit(unit.DisplayHint)
	if subPerUnit == 0 {
		subPerUnit = HCN
	}

	amount := unit.Amount
	negative := amount < 0
	if negative {
		amount = -amount
	}

	if amount == 0 {
		return &String{Value: "0" + unitSuffixSymbol(unit.DisplayHint)}
	}

	// Reduce to fraction
	g := GCD(amount, subPerUnit)
	num := amount / g
	den := subPerUnit / g

	suffix := unitSuffixSymbol(unit.DisplayHint)

	prefix := ""
	if negative {
		prefix = "-"
	}

	if den == 1 {
		return &String{Value: prefix + strconv.FormatInt(num, 10) + suffix}
	}

	whole := num / den
	fracNum := num % den
	if fracNum == 0 {
		return &String{Value: prefix + strconv.FormatInt(whole, 10) + suffix}
	}

	g2 := GCD(fracNum, den)
	fracNum /= g2
	fracDen := den / g2

	if whole > 0 {
		return &String{Value: fmt.Sprintf("%s%d+%d/%d%s", prefix, whole, fracNum, fracDen, suffix)}
	}
	return &String{Value: fmt.Sprintf("%s%d/%d%s", prefix, fracNum, fracDen, suffix)}
}

// unitSuffixSymbol returns a display symbol for a unit suffix.
// For US Customary length, uses traditional symbols (", ', etc.)
func unitSuffixSymbol(suffix string) string {
	switch suffix {
	case "in":
		return "\""
	case "ft":
		return "'"
	default:
		return suffix
	}
}

// exampleForFamily returns an example unit literal for a given family.
func exampleForFamily(family string) string {
	switch family {
	case FamilyLength:
		return "#5in or #100cm"
	case FamilyMass:
		return "#5lb or #100g"
	case FamilyData:
		return "#1024B or #1MB"
	case FamilyTemperature:
		return "#100C or #212F"
	case FamilyVolume:
		return "#1L or #1gal"
	case FamilyArea:
		return "#100m2 or #640ac"
	default:
		return "#5m"
	}
}

// findClosestSuffix finds the closest matching suffix using Levenshtein distance.
func findClosestSuffix(input string, candidates []string) string {
	bestMatch := ""
	bestDist := math.MaxInt32
	for _, c := range candidates {
		d := levenshtein(input, c)
		if d < bestDist && d <= 3 {
			bestDist = d
			bestMatch = c
		}
	}
	return bestMatch
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la := len(a)
	lb := len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	matrix := make([][]int, la+1)
	for i := range matrix {
		matrix[i] = make([]int, lb+1)
		matrix[i][0] = i
	}
	for j := range lb + 1 {
		matrix[0][j] = j
	}

	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			matrix[i][j] = min3(
				matrix[i-1][j]+1,
				matrix[i][j-1]+1,
				matrix[i-1][j-1]+cost,
			)
		}
	}
	return matrix[la][lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// --- Constructor helper ---

// UnitFromConstructor creates a Unit from a constructor call like metres(123) or inches(#1cm).
// value can be a number (creates a new unit) or an existing Unit (converts).
// Scale-aware: uses scaleAmountForSuffix to handle overflow for large values.
func UnitFromConstructor(constructorName string, value Object) Object {
	targetSuffix, ok := UnitConstructorNames[constructorName]
	if !ok {
		return newStructuredError("UNDEF-0001", map[string]any{"Name": constructorName})
	}

	targetInfo, _ := LookupUnitSuffix(targetSuffix)

	switch v := value.(type) {
	case *Integer:
		// Create a new unit from an integer value
		if IsTemperatureFamily(targetInfo.Family) {
			amount := EncodeTempToSubK(float64(v.Value), targetSuffix)
			return &Unit{
				Amount:      amount,
				Family:      targetInfo.Family,
				System:      targetInfo.System,
				DisplayHint: targetSuffix,
			}
		}
		var subPerUnit int64
		if targetInfo.System == SystemUS {
			subPerUnit = USSubUnitsPerUnit(targetSuffix)
		} else {
			subPerUnit = SISubUnitsPerUnit(targetSuffix)
		}
		amount, scale := scaleAmountForSuffix(v.Value, subPerUnit)
		return &Unit{
			Amount:      amount,
			Family:      targetInfo.Family,
			System:      targetInfo.System,
			DisplayHint: targetSuffix,
			Scale:       scale,
		}
	case *Float:
		// Create a new unit from a float value
		if IsTemperatureFamily(targetInfo.Family) {
			amount := EncodeTempToSubK(v.Value, targetSuffix)
			return &Unit{
				Amount:      amount,
				Family:      targetInfo.Family,
				System:      targetInfo.System,
				DisplayHint: targetSuffix,
			}
		}
		var subPerUnit int64
		if targetInfo.System == SystemUS {
			subPerUnit = USSubUnitsPerUnit(targetSuffix)
		} else {
			subPerUnit = SISubUnitsPerUnit(targetSuffix)
		}
		amount, scale := scaleAmountForSuffixFloat(v.Value, subPerUnit)
		return &Unit{
			Amount:      amount,
			Family:      targetInfo.Family,
			System:      targetInfo.System,
			DisplayHint: targetSuffix,
			Scale:       scale,
		}
	case *Unit:
		// Convert an existing unit to the target
		if v.Family != targetInfo.Family {
			return newStructuredError("UNIT-0006", map[string]any{
				"FromFamily":  v.Family,
				"ToFamily":    targetInfo.Family,
				"Constructor": constructorName + "()",
				"Example":     exampleForFamily(targetInfo.Family),
			})
		}
		return convertUnit(v, targetSuffix, targetInfo)
	default:
		return newTypeError("TYPE-0012", constructorName, "a number or unit value", value.Type())
	}
}

// GenericUnitConstructor implements the unit(value, suffix) constructor.
// Scale-aware: uses scaleAmountForSuffix to handle overflow for large values.
func GenericUnitConstructor(args []Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return newArityErrorRange("unit", len(args), 1, 2)
	}

	if len(args) == 2 {
		// Two-argument form: value and suffix string
		suffixStr, ok := args[1].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "unit", "a string (unit suffix)", args[1].Type())
		}
		suffix := suffixStr.Value

		info, found := LookupUnitSuffix(suffix)
		if !found {
			suffixes := AllUnitSuffixes()
			suggestion := findClosestSuffix(suffix, suffixes)
			if suggestion != "" {
				return newStructuredError("UNIT-0007", map[string]any{
					"Suffix":     suffix,
					"Suggestion": suggestion,
				})
			}
			return newStructuredError("UNIT-0007", map[string]any{
				"Suffix":     suffix,
				"Suggestion": "m, cm, km, in, ft, etc.",
			})
		}

		switch v := args[0].(type) {
		case *Integer:
			if IsTemperatureFamily(info.Family) {
				amount := EncodeTempToSubK(float64(v.Value), suffix)
				return &Unit{
					Amount:      amount,
					Family:      info.Family,
					System:      info.System,
					DisplayHint: suffix,
				}
			}
			var subPerUnit int64
			if info.System == SystemUS {
				subPerUnit = USSubUnitsPerUnit(suffix)
			} else {
				subPerUnit = SISubUnitsPerUnit(suffix)
			}
			amount, scale := scaleAmountForSuffix(v.Value, subPerUnit)
			return &Unit{
				Amount:      amount,
				Family:      info.Family,
				System:      info.System,
				DisplayHint: suffix,
				Scale:       scale,
			}
		case *Float:
			if IsTemperatureFamily(info.Family) {
				amount := EncodeTempToSubK(v.Value, suffix)
				return &Unit{
					Amount:      amount,
					Family:      info.Family,
					System:      info.System,
					DisplayHint: suffix,
				}
			}
			var subPerUnit int64
			if info.System == SystemUS {
				subPerUnit = USSubUnitsPerUnit(suffix)
			} else {
				subPerUnit = SISubUnitsPerUnit(suffix)
			}
			amount, scale := scaleAmountForSuffixFloat(v.Value, subPerUnit)
			return &Unit{
				Amount:      amount,
				Family:      info.Family,
				System:      info.System,
				DisplayHint: suffix,
				Scale:       scale,
			}
		case *Unit:
			if v.Family != info.Family {
				return newStructuredError("UNIT-0006", map[string]any{
					"FromFamily":  v.Family,
					"ToFamily":    info.Family,
					"Constructor": "unit()",
					"Example":     exampleForFamily(info.Family),
				})
			}
			return convertUnit(v, suffix, info)
		default:
			return newTypeError("TYPE-0012", "unit", "a number or unit value", args[0].Type())
		}
	}

	// unit(existingUnit) — just returns it as-is (identity)
	if u, ok := args[0].(*Unit); ok {
		return u
	}
	return newTypeError("TYPE-0012", "unit", "a unit value, or (value, suffix)", args[0].Type())
}

// --- String conversion support ---

// UnitToString converts a unit to its string representation for string concatenation.
func UnitToString(u *Unit) string {
	return unitInterpolationString(u)
}

// --- Introspection support ---

func init() {
	// Register unit properties for introspection
	TypeProperties["unit"] = []PropertyInfo{
		{Name: "value", Type: "float", Description: "Decoded value in the display-hint unit"},
		{Name: "unit", Type: "string", Description: "Display-hint unit suffix"},
		{Name: "family", Type: "string", Description: "Unit family (length, mass, data, temperature, volume, area)"},
		{Name: "system", Type: "string", Description: "Measurement system (SI, US)"},
		{Name: "max", Type: "unit", Description: "Maximum representable value at full (Scale=0) precision"},
		{Name: "min", Type: "unit", Description: "Smallest representable positive value (1 base sub-unit)"},
	}

	// Register unit constructor for introspection
	BuiltinMetadata["unit"] = BuiltinInfo{
		Name:        "unit",
		Arity:       "1-2",
		Description: "Create or convert a unit value",
		Params:      []string{"value", "suffix?"},
		Category:    "unit",
	}
}
