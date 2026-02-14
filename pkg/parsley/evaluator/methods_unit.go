// Package evaluator provides unit method implementations via declarative registry.
// This file implements unit methods for FEAT-118: Measurement Units
package evaluator

import (
	"fmt"
	"math"
	"strconv"

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
		"format": {
			Fn:          unitFormatMethod,
			Arity:       "0-1",
			Description: "Format with optional precision",
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
	}
	RegisterMethodRegistry("unit", UnitMethodRegistry)
}

// evalUnitProperty handles property access on Unit values.
func evalUnitProperty(unit *Unit, key string) Object {
	switch key {
	case "value":
		return &Float{Value: unitDisplayValue(unit)}
	case "unit":
		return &String{Value: unit.DisplayHint}
	case "family":
		return &String{Value: unit.Family}
	case "system":
		return &String{Value: unit.System}
	default:
		methodNames := UnitMethodRegistry.Names()
		if _, ok := UnitMethodRegistry[key]; ok {
			return methodAsPropertyError(key, "Unit")
		}
		return unknownMethodError(key, "unit", append([]string{"value", "unit", "family", "system"}, methodNames...))
	}
}

// evalUnitMethod evaluates a method call on a Unit value using the registry.
func evalUnitMethod(unit *Unit, method string, args []Object) Object {
	result := dispatchFromRegistry(UnitMethodRegistry, "unit", unit, method, args, nil)
	if result != nil {
		return result
	}
	return unknownMethodError(method, "unit", UnitMethodRegistry.Names())
}

// --- Method implementations ---

// unitTo converts a unit value to a different unit suffix.
// Usage: #1mi.to("km"), #100cm.to("in")
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
func convertUnit(unit *Unit, targetSuffix string, targetInfo UnitInfo) *Unit {
	var newAmount int64

	switch {
	case unit.System == targetInfo.System:
		// Within-system conversion: simple ratio
		if unit.System == SystemSI {
			srcSub := SISubUnitsPerUnit(unit.DisplayHint)
			dstSub := SISubUnitsPerUnit(targetSuffix)
			if srcSub == 0 || dstSub == 0 {
				return &Unit{Amount: 0, Family: unit.Family, System: targetInfo.System, DisplayHint: targetSuffix}
			}
			// amount is in base sub-units (µm/mg/B), just change the display hint
			newAmount = unit.Amount
		} else {
			// US to US: same sub-unit base (HCN sub-yards or sub-ounces)
			newAmount = unit.Amount
		}
	case unit.System == SystemSI && targetInfo.System == SystemUS:
		// SI → US conversion
		newAmount = ConvertSIToUS(unit.Amount, unit.Family)
	default:
		// US → SI conversion
		newAmount = ConvertUSToSI(unit.Amount, unit.Family)
	}

	return &Unit{
		Amount:      newAmount,
		Family:      unit.Family,
		System:      targetInfo.System,
		DisplayHint: targetSuffix,
	}
}

// unitAbs returns the absolute value of a unit.
func unitAbs(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)
	amount := unit.Amount
	if amount < 0 {
		amount = -amount
	}
	return &Unit{
		Amount:      amount,
		Family:      unit.Family,
		System:      unit.System,
		DisplayHint: unit.DisplayHint,
	}
}

// unitFormatMethod formats a unit value with optional precision.
// Usage: #12.3m.format(), #12.3m.format(4)
func unitFormatMethod(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)
	precision := -1 // default
	if len(args) == 1 {
		switch p := args[0].(type) {
		case *Integer:
			precision = int(p.Value)
		case *Float:
			precision = int(p.Value)
		default:
			return newTypeError("TYPE-0012", "format", "an integer (precision)", args[0].Type())
		}
	}
	return &String{Value: unitFormat(unit, precision)}
}

// unitRepr returns a parseable literal string.
func unitRepr(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)
	return &String{Value: unitInspectString(unit)}
}

// unitToDict returns a dictionary with value, unit, family, system.
func unitToDict(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)
	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"value":  createLiteralExpression(&Float{Value: unitDisplayValue(unit)}),
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
	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"__type":      createLiteralExpression(&String{Value: "unit"}),
			"amount":      createLiteralExpression(&Integer{Value: unit.Amount}),
			"family":      createLiteralExpression(&String{Value: unit.Family}),
			"system":      createLiteralExpression(&String{Value: unit.System}),
			"displayHint": createLiteralExpression(&String{Value: unit.DisplayHint}),
		},
		Env: NewEnvironment(),
	}
}

// unitToFraction returns a fraction string for US Customary values.
// For SI values, returns the decimal string.
func unitToFraction(receiver Object, args []Object, env *Environment) Object {
	unit := receiver.(*Unit)
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
func UnitFromConstructor(constructorName string, value Object) Object {
	targetSuffix, ok := UnitConstructorNames[constructorName]
	if !ok {
		return newStructuredError("UNDEF-0001", map[string]any{"Name": constructorName})
	}

	targetInfo, _ := LookupUnitSuffix(targetSuffix)

	switch v := value.(type) {
	case *Integer:
		// Create a new unit from an integer value
		var amount int64
		if targetInfo.System == SystemUS {
			amount = v.Value * USSubUnitsPerUnit(targetSuffix)
		} else {
			amount = v.Value * SISubUnitsPerUnit(targetSuffix)
		}
		return &Unit{
			Amount:      amount,
			Family:      targetInfo.Family,
			System:      targetInfo.System,
			DisplayHint: targetSuffix,
		}
	case *Float:
		// Create a new unit from a float value
		var amount int64
		if targetInfo.System == SystemUS {
			amount = int64(math.Round(v.Value * float64(USSubUnitsPerUnit(targetSuffix))))
		} else {
			amount = int64(math.Round(v.Value * float64(SISubUnitsPerUnit(targetSuffix))))
		}
		return &Unit{
			Amount:      amount,
			Family:      targetInfo.Family,
			System:      targetInfo.System,
			DisplayHint: targetSuffix,
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
			var amount int64
			if info.System == SystemUS {
				amount = v.Value * USSubUnitsPerUnit(suffix)
			} else {
				amount = v.Value * SISubUnitsPerUnit(suffix)
			}
			return &Unit{
				Amount:      amount,
				Family:      info.Family,
				System:      info.System,
				DisplayHint: suffix,
			}
		case *Float:
			var amount int64
			if info.System == SystemUS {
				amount = int64(math.Round(v.Value * float64(USSubUnitsPerUnit(suffix))))
			} else {
				amount = int64(math.Round(v.Value * float64(SISubUnitsPerUnit(suffix))))
			}
			return &Unit{
				Amount:      amount,
				Family:      info.Family,
				System:      info.System,
				DisplayHint: suffix,
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

// IsUnit returns true if the object is a Unit.
func IsUnit(obj Object) bool {
	_, ok := obj.(*Unit)
	return ok
}

// --- Introspection support ---

func init() {
	// Register unit properties for introspection
	TypeProperties["unit"] = []PropertyInfo{
		{Name: "value", Type: "float", Description: "Decoded value in the display-hint unit"},
		{Name: "unit", Type: "string", Description: "Display-hint unit suffix"},
		{Name: "family", Type: "string", Description: "Unit family (length, mass, data)"},
		{Name: "system", Type: "string", Description: "Measurement system (SI, US)"},
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
