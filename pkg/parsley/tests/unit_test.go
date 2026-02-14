package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/pln"
)

// Helper to evaluate unit expressions
func testEvalUnit(input string) evaluator.Object {
	return testEvalHelper(input)
}

func testExpectedUnit(t *testing.T, input string, obj evaluator.Object, expected string) {
	t.Helper()
	if obj == nil {
		t.Errorf("For input '%s': got nil object", input)
		return
	}
	if err, ok := obj.(*evaluator.Error); ok {
		t.Errorf("For input '%s': got error: [%s] %s", input, err.Code, err.Message)
		return
	}
	actual := obj.Inspect()
	if actual != expected {
		t.Errorf("For input '%s': expected %s, got %s", input, expected, actual)
	}
}

func testExpectedValue(t *testing.T, input string, obj evaluator.Object, expected string) {
	t.Helper()
	if obj == nil {
		t.Errorf("For input '%s': got nil object", input)
		return
	}
	if err, ok := obj.(*evaluator.Error); ok {
		t.Errorf("For input '%s': got error: [%s] %s", input, err.Code, err.Message)
		return
	}
	actual := obj.Inspect()
	if actual != expected {
		t.Errorf("For input '%s': expected %s, got %s", input, expected, actual)
	}
}

func testExpectedUnitError(t *testing.T, input string, obj evaluator.Object, expectedSubstring string) {
	t.Helper()
	if obj == nil {
		t.Errorf("For input '%s': expected error but got nil", input)
		return
	}
	err, ok := obj.(*evaluator.Error)
	if !ok {
		t.Errorf("For input '%s': expected error but got %T: %s", input, obj, obj.Inspect())
		return
	}
	if !strings.Contains(strings.ToLower(err.Message), strings.ToLower(expectedSubstring)) {
		t.Errorf("For input '%s': expected error containing '%s', got '%s'", input, expectedSubstring, err.Message)
	}
}

// ============================================================================
// SI Length Literals
// ============================================================================

func TestUnitLiteralSILength(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Millimetres
		{`#0mm`, `#0mm`},
		{`#1mm`, `#1mm`},
		{`#100mm`, `#100mm`},
		// Centimetres
		{`#0cm`, `#0cm`},
		{`#1cm`, `#1cm`},
		{`#50cm`, `#50cm`},
		// Metres
		{`#0m`, `#0m`},
		{`#1m`, `#1m`},
		{`#12m`, `#12m`},
		{`#100m`, `#100m`},
		// Kilometres
		{`#0km`, `#0km`},
		{`#1km`, `#1km`},
		{`#42km`, `#42km`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestUnitLiteralSILengthDecimal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#12.3m`, `#12.3m`},
		{`#12.34m`, `#12.34m`},
		{`#0.5m`, `#0.5m`},
		{`#1.5cm`, `#1.5cm`},
		{`#3.14km`, `#3.14km`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// US Customary Length Literals
// ============================================================================

func TestUnitLiteralUSLength(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#0in`, `#0in`},
		{`#1in`, `#1in`},
		{`#12in`, `#12in`},
		{`#1ft`, `#1ft`},
		{`#3ft`, `#3ft`},
		{`#1yd`, `#1yd`},
		{`#1mi`, `#1mi`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestUnitLiteralUSLengthFraction(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#1/2in`, `#1/2in`},
		{`#1/4in`, `#1/4in`},
		{`#3/8in`, `#3/8in`},
		{`#5/16in`, `#5/16in`},
		{`#1/3ft`, `#1/3ft`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestUnitLiteralUSLengthMixedNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#1+1/2in`, `#1+1/2in`},
		{`#92+5/8in`, `#92+5/8in`},
		{`#3+1/4ft`, `#3+1/4ft`},
		{`#2+3/8in`, `#2+3/8in`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Negative Literals
// ============================================================================

func TestUnitLiteralNegative(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#-6m`, `#-6m`},
		{`#-12.3m`, `#-12.3m`},
		{`#-1in`, `#-1in`},
		{`#-3/8in`, `#-3/8in`},
		{`#-2+1/4in`, `#-2+1/4in`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestUnitUnaryNegation(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`-#6m`, `#-6m`},
		{`-#12.3m`, `#-12.3m`},
		{`-#1in`, `#-1in`},
		{`-#3/8in`, `#-3/8in`},
		// Double negation
		{`-#-6m`, `#6m`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Mass Literals
// ============================================================================

func TestUnitLiteralMass(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// SI
		{`#1mg`, `#1mg`},
		{`#500mg`, `#500mg`},
		{`#1g`, `#1g`},
		{`#100g`, `#100g`},
		{`#1kg`, `#1kg`},
		{`#2.5kg`, `#2.5kg`},
		// US
		{`#1oz`, `#1oz`},
		{`#16oz`, `#16oz`},
		{`#1lb`, `#1lb`},
		{`#2.5lb`, `#2+1/2lb`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Data Literals
// ============================================================================

func TestUnitLiteralData(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#1B`, `#1B`},
		{`#1024B`, `#1024B`},
		{`#1kB`, `#1kB`},
		{`#1MB`, `#1MB`},
		{`#1GB`, `#1GB`},
		{`#1TB`, `#1TB`},
		// Binary
		{`#1KiB`, `#1KiB`},
		{`#1MiB`, `#1MiB`},
		{`#1GiB`, `#1GiB`},
		{`#1TiB`, `#1TiB`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Properties
// ============================================================================

func TestUnitProperties(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// SI Length
		{`#12m.value`, `12`},
		{`#12m.unit`, `m`},
		{`#12m.family`, `length`},
		{`#12m.system`, `SI`},
		// US Length
		{`#1in.unit`, `in`},
		{`#1in.family`, `length`},
		{`#1in.system`, `US`},
		// Mass
		{`#1kg.family`, `mass`},
		{`#1lb.system`, `US`},
		// Data
		{`#1MB.family`, `data`},
		{`#1MB.system`, `SI`},
		// Decimal value
		{`#12.5m.value`, `12.5`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Same-Family, Same-System Arithmetic
// ============================================================================

func TestUnitAdditionSameSystem(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#10m + #5m`, `#15m`},
		{`#1km + #1km`, `#2km`},
		{`#100cm + #50cm`, `#150cm`},
		{`#6in + #6in`, `#12in`},
		{`#1ft + #6in`, `#1+1/2ft`},
		{`#1lb + #1lb`, `#2lb`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestUnitSubtractionSameSystem(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#10m - #3m`, `#7m`},
		{`#1ft - #6in`, `#1/2ft`},
		{`#100cm - #50cm`, `#50cm`},
		{`#2lb - #1lb`, `#1lb`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Scalar Multiplication and Division
// ============================================================================

func TestUnitScalarMultiplication(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#10m * 3`, `#30m`},
		{`3 * #10m`, `#30m`},
		{`#5in * 2`, `#10in`},
		{`#1ft * 12`, `#12ft`},
		{`#1kg * 2.5`, `#2.5kg`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestUnitScalarDivision(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#10m / 5`, `#2m`},
		{`#10m / 2`, `#5m`},
		{`#1ft / 2`, `#1/2ft`},
		{`#12in / 4`, `#3in`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestUnitUnitDivision(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#10m / #5m`, `2`},
		{`#10m / #2m`, `5`},
		{`#1ft / #6in`, `2`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Cross-System Arithmetic (Left Side Wins)
// ============================================================================

func TestUnitCrossSystemAddition(t *testing.T) {
	// Left-side-wins: result uses left operand's system and display hint
	tests := []struct {
		input string
		check func(t *testing.T, obj evaluator.Object)
	}{
		{
			// #1cm + #1in: result is in cm
			input: `#1cm + #1in`,
			check: func(t *testing.T, obj evaluator.Object) {
				t.Helper()
				u, ok := obj.(*evaluator.Unit)
				if !ok {
					t.Errorf("expected Unit, got %T: %s", obj, obj.Inspect())
					return
				}
				if u.System != "SI" {
					t.Errorf("expected SI system, got %s", u.System)
				}
				if u.DisplayHint != "cm" {
					t.Errorf("expected display hint 'cm', got %s", u.DisplayHint)
				}
			},
		},
		{
			// #1in + #1cm: result is in inches
			input: `#1in + #1cm`,
			check: func(t *testing.T, obj evaluator.Object) {
				t.Helper()
				u, ok := obj.(*evaluator.Unit)
				if !ok {
					t.Errorf("expected Unit, got %T: %s", obj, obj.Inspect())
					return
				}
				if u.System != "US" {
					t.Errorf("expected US system, got %s", u.System)
				}
				if u.DisplayHint != "in" {
					t.Errorf("expected display hint 'in', got %s", u.DisplayHint)
				}
			},
		},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		if err, ok := evaluated.(*evaluator.Error); ok {
			t.Errorf("For input '%s': got error: [%s] %s", tt.input, err.Code, err.Message)
			continue
		}
		tt.check(t, evaluated)
	}
}

// ============================================================================
// Comparison Operators
// ============================================================================

func TestUnitComparison(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Same system
		{`#1m == #1m`, `true`},
		{`#1m == #100cm`, `true`},
		{`#1m != #2m`, `true`},
		{`#1m < #2m`, `true`},
		{`#2m > #1m`, `true`},
		{`#1m <= #1m`, `true`},
		{`#1m >= #1m`, `true`},
		{`#1m <= #2m`, `true`},
		{`#2m >= #1m`, `true`},
		// Different families are never equal
		{`#1m == #1kg`, `false`},
		{`#1m != #1kg`, `true`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

func TestUnitCrossSystemComparison(t *testing.T) {
	// 1 inch = 25.4mm, so 1in < 1m (which is 1000mm)
	tests := []struct {
		input    string
		expected string
	}{
		{`#1in < #1m`, `true`},
		{`#1m > #1in`, `true`},
		{`#1ft < #1m`, `true`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Constructors
// ============================================================================

func TestUnitNamedConstructors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// SI Length
		{`metres(5)`, `#5m`},
		{`meters(5)`, `#5m`},
		{`centimetres(100)`, `#100cm`},
		{`centimeters(100)`, `#100cm`},
		{`millimetres(500)`, `#500mm`},
		{`millimeters(500)`, `#500mm`},
		{`kilometres(1)`, `#1km`},
		{`kilometers(1)`, `#1km`},
		// US Length
		{`inches(12)`, `#12in`},
		{`feet(3)`, `#3ft`},
		{`yards(1)`, `#1yd`},
		{`miles(1)`, `#1mi`},
		// SI Mass
		{`grams(100)`, `#100g`},
		{`kilograms(1)`, `#1kg`},
		{`milligrams(500)`, `#500mg`},
		// US Mass
		{`ounces(16)`, `#16oz`},
		{`pounds(1)`, `#1lb`},
		// Data
		{`kilobytes(1)`, `#1kB`},
		{`megabytes(1)`, `#1MB`},
		{`gigabytes(1)`, `#1GB`},
		{`terabytes(1)`, `#1TB`},
		{`kibibytes(1)`, `#1KiB`},
		{`mebibytes(1)`, `#1MiB`},
		{`gibibytes(1)`, `#1GiB`},
		{`tebibytes(1)`, `#1TiB`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestUnitNamedConstructorConversion(t *testing.T) {
	tests := []struct {
		input string
		check func(t *testing.T, obj evaluator.Object)
	}{
		{
			// Convert inches to metres
			input: `metres(#12in)`,
			check: func(t *testing.T, obj evaluator.Object) {
				t.Helper()
				u, ok := obj.(*evaluator.Unit)
				if !ok {
					t.Errorf("expected Unit, got %T: %s", obj, obj.Inspect())
					return
				}
				if u.System != "SI" || u.DisplayHint != "m" {
					t.Errorf("expected SI/m, got %s/%s", u.System, u.DisplayHint)
				}
			},
		},
		{
			// Convert metres to inches
			input: `inches(#1m)`,
			check: func(t *testing.T, obj evaluator.Object) {
				t.Helper()
				u, ok := obj.(*evaluator.Unit)
				if !ok {
					t.Errorf("expected Unit, got %T: %s", obj, obj.Inspect())
					return
				}
				if u.System != "US" || u.DisplayHint != "in" {
					t.Errorf("expected US/in, got %s/%s", u.System, u.DisplayHint)
				}
			},
		},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		if err, ok := evaluated.(*evaluator.Error); ok {
			t.Errorf("For input '%s': got error: [%s] %s", tt.input, err.Code, err.Message)
			continue
		}
		tt.check(t, evaluated)
	}
}

func TestUnitGenericConstructor(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`unit(123, "m")`, `#123m`},
		{`unit(12, "in")`, `#12in`},
		{`unit(1, "kg")`, `#1kg`},
		{`unit(1024, "B")`, `#1024B`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestUnitGenericConstructorConversion(t *testing.T) {
	tests := []struct {
		input string
		check func(t *testing.T, obj evaluator.Object)
	}{
		{
			input: `unit(#12in, "cm")`,
			check: func(t *testing.T, obj evaluator.Object) {
				t.Helper()
				u, ok := obj.(*evaluator.Unit)
				if !ok {
					t.Errorf("expected Unit, got %T: %s", obj, obj.Inspect())
					return
				}
				if u.DisplayHint != "cm" {
					t.Errorf("expected display hint 'cm', got %s", u.DisplayHint)
				}
			},
		},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		if err, ok := evaluated.(*evaluator.Error); ok {
			t.Errorf("For input '%s': got error: [%s] %s", tt.input, err.Code, err.Message)
			continue
		}
		tt.check(t, evaluated)
	}
}

// ============================================================================
// Methods
// ============================================================================

func TestUnitToMethod(t *testing.T) {
	tests := []struct {
		input string
		check func(t *testing.T, obj evaluator.Object)
	}{
		{
			input: `#1mi.to("km")`,
			check: func(t *testing.T, obj evaluator.Object) {
				t.Helper()
				u, ok := obj.(*evaluator.Unit)
				if !ok {
					t.Errorf("expected Unit, got %T: %s", obj, obj.Inspect())
					return
				}
				if u.DisplayHint != "km" {
					t.Errorf("expected km, got %s", u.DisplayHint)
				}
			},
		},
		{
			input: `#100cm.to("m")`,
			check: func(t *testing.T, obj evaluator.Object) {
				t.Helper()
				u, ok := obj.(*evaluator.Unit)
				if !ok {
					t.Errorf("expected Unit, got %T: %s", obj, obj.Inspect())
					return
				}
				if u.DisplayHint != "m" {
					t.Errorf("expected m, got %s", u.DisplayHint)
				}
				// 100cm should be exactly 1m
				expected := "#1m"
				if u.Inspect() != expected {
					t.Errorf("expected %s, got %s", expected, u.Inspect())
				}
			},
		},
		{
			input: `#12.3m.to("cm")`,
			check: func(t *testing.T, obj evaluator.Object) {
				t.Helper()
				u, ok := obj.(*evaluator.Unit)
				if !ok {
					t.Errorf("expected Unit, got %T: %s", obj, obj.Inspect())
					return
				}
				if u.Inspect() != "#1230cm" {
					t.Errorf("expected #1230cm, got %s", u.Inspect())
				}
			},
		},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		if err, ok := evaluated.(*evaluator.Error); ok {
			t.Errorf("For input '%s': got error: [%s] %s", tt.input, err.Code, err.Message)
			continue
		}
		tt.check(t, evaluated)
	}
}

func TestUnitAbsMethod(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#12m.abs()`, `#12m`},
		{`#-6m.abs()`, `#6m`},
		{`#-3/8in.abs()`, `#3/8in`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestUnitFormatMethod(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#12.3m.format()`, `12.3m`},
		{`#12.3m.format(0)`, `12m`},
		{`#12.3m.format(4)`, `12.3000m`},
		{`#12m.format()`, `12m`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

func TestUnitReprMethod(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#12m.repr()`, `#12m`},
		{`#3/8in.repr()`, `#3/8in`},
		{`#92+5/8in.repr()`, `#92+5/8in`},
		{`#-6m.repr()`, `#-6m`},
		{`#12.3m.repr()`, `#12.3m`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

func TestUnitToDictMethod(t *testing.T) {
	evaluated := testEvalUnit(`#12m.toDict()`)
	if err, ok := evaluated.(*evaluator.Error); ok {
		t.Fatalf("got error: [%s] %s", err.Code, err.Message)
	}
	dict, ok := evaluated.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T: %s", evaluated, evaluated.Inspect())
	}

	// Check that the dict has expected keys
	expectedKeys := []string{"value", "unit", "family", "system"}
	for _, key := range expectedKeys {
		if _, exists := dict.Pairs[key]; !exists {
			t.Errorf("missing key '%s' in toDict() result", key)
		}
	}
}

func TestUnitInspectMethod(t *testing.T) {
	evaluated := testEvalUnit(`#12m.inspect()`)
	if err, ok := evaluated.(*evaluator.Error); ok {
		t.Fatalf("got error: [%s] %s", err.Code, err.Message)
	}
	dict, ok := evaluated.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T: %s", evaluated, evaluated.Inspect())
	}

	expectedKeys := []string{"__type", "amount", "family", "system", "displayHint"}
	for _, key := range expectedKeys {
		if _, exists := dict.Pairs[key]; !exists {
			t.Errorf("missing key '%s' in inspect() result", key)
		}
	}
}

func TestUnitToFractionMethod(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#3/8in.toFraction()`, `3/8"`},
		{`#1+1/2in.toFraction()`, `1+1/2"`},
		{`#12in.toFraction()`, `12"`},
		// SI values return decimal format
		{`#12m.toFraction()`, `12m`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// String Interpolation and Concatenation
// ============================================================================

func TestUnitStringConcatenation(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// No # sigil in string output
		{`"Length: " + #12m`, `Length: 12m`},
		{`"Size: " + #3/8in`, `Size: 3/8in`},
		{`#12m + " long"`, `12m long`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

func TestUnitTemplateInterpolation(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Backtick template strings interpolate units without # sigil
		{"let d = #12m; `Length: {d}`", `Length: 12m`},
		{"let d = #3/8in; `Size: {d}`", `Size: 3/8in`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Variables and Control Flow
// ============================================================================

func TestUnitVariables(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`let x = #12m; x`, `#12m`},
		{`let x = #12m; let y = #3m; x + y`, `#15m`},
		{`let x = #12m; x * 2`, `#24m`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestUnitInConditionals(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`if #10m > #5m { "big" } else { "small" }`, `big`},
		{`if #1in < #1m { "short" } else { "long" }`, `short`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

func TestUnitInArrays(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`[#1m, #2m, #3m][0]`, `#1m`},
		{`[#1m, #2m, #3m][1]`, `#2m`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// PLN Round-Trip (serialize / deserialize)
// ============================================================================

func TestUnitPLNSerialize(t *testing.T) {
	tests := []struct {
		name     string
		unit     *evaluator.Unit
		expected string
	}{
		{
			name:     "SI integer",
			unit:     &evaluator.Unit{Amount: 12_000_000, Family: "length", System: "SI", DisplayHint: "m"},
			expected: "#12m",
		},
		{
			name:     "SI decimal",
			unit:     &evaluator.Unit{Amount: 12_300_000, Family: "length", System: "SI", DisplayHint: "m"},
			expected: "#12.3m",
		},
		{
			name:     "US integer",
			unit:     &evaluator.Unit{Amount: 20160, Family: "length", System: "US", DisplayHint: "in"},
			expected: "#1in",
		},
		{
			name:     "US fraction",
			unit:     &evaluator.Unit{Amount: 20160 * 3 / 8, Family: "length", System: "US", DisplayHint: "in"},
			expected: "#3/8in",
		},
		{
			name:     "data bytes",
			unit:     &evaluator.Unit{Amount: 1024, Family: "data", System: "SI", DisplayHint: "B"},
			expected: "#1024B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pln.Serialize(tt.unit)
			if err != nil {
				t.Fatalf("serialize error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestUnitPLNRoundTrip(t *testing.T) {
	tests := []struct {
		input string
	}{
		{`#12m`},
		{`#12.3m`},
		{`#100cm`},
		{`#1km`},
		{`#3/8in`},
		{`#92+5/8in`},
		{`#1ft`},
		{`#1kg`},
		{`#500g`},
		{`#1024B`},
		{`#1MB`},
		{`#1GiB`},
		{`#-6m`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// Evaluate the literal to get a Unit object
			evaluated := testEvalUnit(tt.input)
			if err, ok := evaluated.(*evaluator.Error); ok {
				t.Fatalf("eval error: [%s] %s", err.Code, err.Message)
			}
			u, ok := evaluated.(*evaluator.Unit)
			if !ok {
				t.Fatalf("expected Unit, got %T: %s", evaluated, evaluated.Inspect())
			}

			// Serialize to PLN
			plnStr, err := pln.Serialize(u)
			if err != nil {
				t.Fatalf("serialize error: %v", err)
			}

			// Deserialize back
			deserialized, err := pln.Deserialize(plnStr, nil, nil)
			if err != nil {
				t.Fatalf("deserialize error for %q: %v", plnStr, err)
			}

			u2, ok := deserialized.(*evaluator.Unit)
			if !ok {
				t.Fatalf("deserialized to %T, expected Unit", deserialized)
			}

			// Values should match
			if u.Amount != u2.Amount {
				t.Errorf("Amount mismatch: %d vs %d", u.Amount, u2.Amount)
			}
			if u.Family != u2.Family {
				t.Errorf("Family mismatch: %s vs %s", u.Family, u2.Family)
			}
			if u.System != u2.System {
				t.Errorf("System mismatch: %s vs %s", u.System, u2.System)
			}
			if u.DisplayHint != u2.DisplayHint {
				t.Errorf("DisplayHint mismatch: %s vs %s", u.DisplayHint, u2.DisplayHint)
			}
		})
	}
}

// ============================================================================
// Error Cases
// ============================================================================

func TestUnitErrorDifferentFamilies(t *testing.T) {
	tests := []struct {
		input          string
		expectedSubstr string
	}{
		{`#1m + #1kg`, "length"},
		{`#1m - #1kg`, "length"},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnitError(t, tt.input, evaluated, tt.expectedSubstr)
	}
}

func TestUnitErrorScalarPlusUnit(t *testing.T) {
	tests := []struct {
		input          string
		expectedSubstr string
	}{
		{`5 + #5m`, "number"},
		{`5 - #5m`, "number"},
		{`#5m + 5`, "unit"},
		{`#5m - 5`, "unit"},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnitError(t, tt.input, evaluated, tt.expectedSubstr)
	}
}

func TestUnitErrorUnitTimesUnit(t *testing.T) {
	evaluated := testEvalUnit(`#5m * #5m`)
	testExpectedUnitError(t, `#5m * #5m`, evaluated, "multiply")
}

func TestUnitErrorScalarDivUnit(t *testing.T) {
	evaluated := testEvalUnit(`10 / #5m`)
	testExpectedUnitError(t, `10 / #5m`, evaluated, "divide")
}

func TestUnitErrorWrongFamilyConstructor(t *testing.T) {
	evaluated := testEvalUnit(`metres(#1kg)`)
	testExpectedUnitError(t, `metres(#1kg)`, evaluated, "convert")
}

func TestUnitErrorUnknownSuffix(t *testing.T) {
	evaluated := testEvalUnit(`unit(5, "xyz")`)
	testExpectedUnitError(t, `unit(5, "xyz")`, evaluated, "unknown")
}

func TestUnitErrorDivisionByZero(t *testing.T) {
	evaluated := testEvalUnit(`#10m / 0`)
	if evaluated == nil {
		t.Fatal("expected error, got nil")
	}
	_, ok := evaluated.(*evaluator.Error)
	if !ok {
		t.Errorf("expected error, got %T: %s", evaluated, evaluated.Inspect())
	}
}

// ============================================================================
// Within-System Conversion Exactness
// ============================================================================

func TestUnitConversionExactness(t *testing.T) {
	// 100cm should be exactly 1m
	tests := []struct {
		input    string
		expected string
	}{
		{`#100cm.to("m")`, `#1m`},
		{`#1000mm.to("m")`, `#1m`},
		{`#1000m.to("km")`, `#1km`},
		{`#12in.to("ft")`, `#1ft`},
		{`#3ft.to("yd")`, `#1yd`},
		{`#1000g.to("kg")`, `#1kg`},
		{`#16oz.to("lb")`, `#1lb`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Unit Equality Across Representations
// ============================================================================

func TestUnitEqualityAcrossUnits(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#1m == #100cm`, `true`},
		{`#1m == #1000mm`, `true`},
		{`#1km == #1000m`, `true`},
		{`#1ft == #12in`, `true`},
		{`#1yd == #3ft`, `true`},
		{`#1lb == #16oz`, `true`},
		{`#1kg == #1000g`, `true`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Cross-System Bridge Correctness
// ============================================================================

func TestUnitBridgeInchToMM(t *testing.T) {
	// 1 inch = 25.4mm exactly by international definition
	// Internally: 1in = 20,160 sub-yards, 25.4mm = 25,400 µm
	// #1in.to("mm") should be close to #25.4mm
	evaluated := testEvalUnit(`#1in.to("mm")`)
	if err, ok := evaluated.(*evaluator.Error); ok {
		t.Fatalf("got error: [%s] %s", err.Code, err.Message)
	}
	u, ok := evaluated.(*evaluator.Unit)
	if !ok {
		t.Fatalf("expected Unit, got %T", evaluated)
	}
	if u.DisplayHint != "mm" {
		t.Errorf("expected mm, got %s", u.DisplayHint)
	}
	// Check that the value is close to 25,400 µm (25.4 mm)
	// The bridge ratio is 635/504, so 20160 * 635 / 504 = 25400 exactly
	if u.Amount != 25400 {
		t.Errorf("expected 25400 µm (25.4mm), got %d µm", u.Amount)
	}
}

func TestUnitBridgeFootToM(t *testing.T) {
	// 1 foot = 12 inches = 0.3048m exactly
	// 1ft = HCN/3 = 241920 sub-yards
	// 241920 * 635 / 504 = 304800 µm = 0.3048m
	evaluated := testEvalUnit(`#1ft.to("m")`)
	if err, ok := evaluated.(*evaluator.Error); ok {
		t.Fatalf("got error: [%s] %s", err.Code, err.Message)
	}
	u, ok := evaluated.(*evaluator.Unit)
	if !ok {
		t.Fatalf("expected Unit, got %T", evaluated)
	}
	if u.Amount != 304800 {
		t.Errorf("expected 304800 µm (0.3048m), got %d µm", u.Amount)
	}
}

// ============================================================================
// Fraction Reduction
// ============================================================================

func TestUnitFractionReduction(t *testing.T) {
	// 2/4 should reduce to 1/2
	tests := []struct {
		input    string
		expected string
	}{
		{`#2/4in`, `#1/2in`},
		{`#4/8in`, `#1/2in`},
		{`#6/8in`, `#3/4in`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Type Check
// ============================================================================

func TestUnitObjectType(t *testing.T) {
	evaluated := testEvalUnit(`#12m`)
	if evaluated.Type() != evaluator.UNIT_OBJ {
		t.Errorf("expected UNIT type, got %s", evaluated.Type())
	}
}

// ============================================================================
// Phase 2: Temperature Literals
// ============================================================================

func TestTemperatureLiteralCelsius(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#0C`, `#0C`},
		{`#100C`, `#100C`},
		{`#37C`, `#37C`},
		{`#-40C`, `#-40C`},
		{`#-273C`, `#-273C`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestTemperatureLiteralFahrenheit(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#32F`, `#32F`},
		{`#212F`, `#212F`},
		{`#0F`, `#0F`},
		{`#-40F`, `#-40F`},
		{`#98.6F`, `#98.6F`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestTemperatureLiteralKelvin(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#0K`, `#0K`},
		{`#273.15K`, `#273.15K`},
		{`#373.15K`, `#373.15K`},
		{`#100K`, `#100K`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestTemperatureLiteralDecimal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#37.5C`, `#37.5C`},
		{`#98.6F`, `#98.6F`},
		{`#-273.15C`, `#-273.15C`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Temperature Cross-Scale Equivalence
// ============================================================================

func TestTemperatureCrossScaleEquivalence(t *testing.T) {
	// #0C and #32F should have the same internal Amount
	c0 := testEvalUnit(`#0C`)
	f32 := testEvalUnit(`#32F`)
	uc, ok1 := c0.(*evaluator.Unit)
	uf, ok2 := f32.(*evaluator.Unit)
	if !ok1 || !ok2 {
		t.Fatalf("expected Unit objects, got %T and %T", c0, f32)
	}
	if uc.Amount != uf.Amount {
		t.Errorf("#0C.Amount=%d != #32F.Amount=%d", uc.Amount, uf.Amount)
	}

	// #100C and #212F
	c100 := testEvalUnit(`#100C`)
	f212 := testEvalUnit(`#212F`)
	uc2 := c100.(*evaluator.Unit)
	uf2 := f212.(*evaluator.Unit)
	if uc2.Amount != uf2.Amount {
		t.Errorf("#100C.Amount=%d != #212F.Amount=%d", uc2.Amount, uf2.Amount)
	}

	// #-40C and #-40F (scales cross at -40)
	cm40 := testEvalUnit(`#-40C`)
	fm40 := testEvalUnit(`#-40F`)
	ucm := cm40.(*evaluator.Unit)
	ufm := fm40.(*evaluator.Unit)
	if ucm.Amount != ufm.Amount {
		t.Errorf("#-40C.Amount=%d != #-40F.Amount=%d", ucm.Amount, ufm.Amount)
	}

	// #0K and #-273.15C (absolute zero)
	k0 := testEvalUnit(`#0K`)
	cm273 := testEvalUnit(`#-273.15C`)
	uk := k0.(*evaluator.Unit)
	uca := cm273.(*evaluator.Unit)
	if uk.Amount != uca.Amount {
		t.Errorf("#0K.Amount=%d != #-273.15C.Amount=%d", uk.Amount, uca.Amount)
	}
}

// ============================================================================
// Temperature Properties
// ============================================================================

func TestTemperatureProperties(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#100C.value`, `100`},
		{`#100C.unit`, `C`},
		{`#100C.family`, `temperature`},
		{`#100C.system`, `SI`},
		{`#212F.value`, `212`},
		{`#212F.unit`, `F`},
		{`#212F.family`, `temperature`},
		{`#212F.system`, `US`},
		{`#0K.value`, `0`},
		{`#0K.unit`, `K`},
		{`#0K.family`, `temperature`},
		{`#0K.system`, `SI`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Temperature Arithmetic — Same Scale
// ============================================================================

func TestTemperatureAddSameScale(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#20C + #10C`, `#30C`},
		{`#100C + #0C`, `#100C`},
		{`#0F + #32F`, `#32F`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestTemperatureSubtractSameScale(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#100C - #37C`, `#63C`},
		{`#212F - #32F`, `#180F`},
		{`#50C - #50C`, `#0C`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Temperature Comparison
// ============================================================================

func TestTemperatureComparison(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#0C == #32F`, `true`},
		{`#100C == #212F`, `true`},
		{`#-40C == #-40F`, `true`},
		{`#100C > #200F`, `true`}, // #100C = #212F > #200F
		{`#20C < #20F`, `false`},  // #20C = #68F > #20F
		{`#0C != #0F`, `true`},    // 0C = 32F, 0F = -17.78C
		{`#100C >= #212F`, `true`},
		{`#100C <= #212F`, `true`},
		{`#0K == #-273.15C`, `true`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Temperature Arithmetic — Restrictions (Multiply/Divide)
// ============================================================================

func TestTemperatureMultiplyBlocked(t *testing.T) {
	tests := []struct {
		input          string
		expectedSubstr string
	}{
		{`#20C * 2`, "multiply a temperature"},
		{`2 * #20C`, "multiply a temperature"},
		{`#100F * 3`, "multiply a temperature"},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnitError(t, tt.input, evaluated, tt.expectedSubstr)
	}
}

func TestTemperatureDivideBlocked(t *testing.T) {
	tests := []struct {
		input          string
		expectedSubstr string
	}{
		{`#100F / 2`, "divide a temperature"},
		{`#20C / 5`, "divide a temperature"},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnitError(t, tt.input, evaluated, tt.expectedSubstr)
	}
}

func TestTemperatureDivideByTemperatureBlocked(t *testing.T) {
	tests := []struct {
		input          string
		expectedSubstr string
	}{
		{`#100C / #50C`, "divide temperature by temperature"},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnitError(t, tt.input, evaluated, tt.expectedSubstr)
	}
}

// ============================================================================
// Temperature Negation (allowed — negation is not multiplication)
// ============================================================================

func TestTemperatureNegation(t *testing.T) {
	// Negation negates the internal sub-K, which is allowed
	evaluated := testEvalUnit(`-#20C`)
	if err, ok := evaluated.(*evaluator.Error); ok {
		t.Fatalf("expected no error for -#20C, got [%s] %s", err.Code, err.Message)
	}
	_, ok := evaluated.(*evaluator.Unit)
	if !ok {
		t.Fatalf("expected Unit, got %T: %s", evaluated, evaluated.Inspect())
	}
}

// ============================================================================
// Temperature Constructors
// ============================================================================

func TestTemperatureConstructors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`celsius(100)`, `#100C`},
		{`fahrenheit(212)`, `#212F`},
		{`kelvins(0)`, `#0K`},
		{`celsius(0)`, `#0C`},
		{`fahrenheit(32)`, `#32F`},
		{`kelvins(273.15)`, `#273.15K`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestTemperatureConstructorConversion(t *testing.T) {
	// celsius(#212F) should give #100C
	evaluated := testEvalUnit(`celsius(#212F)`)
	if err, ok := evaluated.(*evaluator.Error); ok {
		t.Fatalf("celsius(#212F) error: [%s] %s", err.Code, err.Message)
	}
	u, ok := evaluated.(*evaluator.Unit)
	if !ok {
		t.Fatalf("expected Unit, got %T", evaluated)
	}
	if u.DisplayHint != "C" {
		t.Errorf("expected DisplayHint=C, got %s", u.DisplayHint)
	}
	val := evaluator.DecodeTempFromSubK(u.Amount, "C")
	if val != 100.0 {
		t.Errorf("celsius(#212F) value: expected 100, got %f", val)
	}

	// fahrenheit(#0C) should give #32F
	evaluated2 := testEvalUnit(`fahrenheit(#0C)`)
	u2 := evaluated2.(*evaluator.Unit)
	if u2.DisplayHint != "F" {
		t.Errorf("expected DisplayHint=F, got %s", u2.DisplayHint)
	}
	val2 := evaluator.DecodeTempFromSubK(u2.Amount, "F")
	if val2 != 32.0 {
		t.Errorf("fahrenheit(#0C) value: expected 32, got %f", val2)
	}

	// kelvins(#100C) should give #373.15K
	evaluated3 := testEvalUnit(`kelvins(#100C)`)
	u3 := evaluated3.(*evaluator.Unit)
	if u3.DisplayHint != "K" {
		t.Errorf("expected DisplayHint=K, got %s", u3.DisplayHint)
	}
	val3 := evaluator.DecodeTempFromSubK(u3.Amount, "K")
	if val3 != 373.15 {
		t.Errorf("kelvins(#100C) value: expected 373.15, got %f", val3)
	}
}

func TestTemperatureConstructorWrongFamily(t *testing.T) {
	evaluated := testEvalUnit(`celsius(#5kg)`)
	testExpectedUnitError(t, `celsius(#5kg)`, evaluated, "convert")
}

func TestTemperatureGenericConstructor(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`unit(100, "C")`, `#100C`},
		{`unit(212, "F")`, `#212F`},
		{`unit(0, "K")`, `#0K`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestTemperatureGenericConstructorConversion(t *testing.T) {
	// unit(#212F, "C") should give #100C
	evaluated := testEvalUnit(`unit(#212F, "C")`)
	if err, ok := evaluated.(*evaluator.Error); ok {
		t.Fatalf("unit(#212F, 'C') error: [%s] %s", err.Code, err.Message)
	}
	u, ok := evaluated.(*evaluator.Unit)
	if !ok {
		t.Fatalf("expected Unit, got %T", evaluated)
	}
	val := evaluator.DecodeTempFromSubK(u.Amount, "C")
	if val != 100.0 {
		t.Errorf("unit(#212F, 'C') value: expected 100, got %f", val)
	}
}

// ============================================================================
// Temperature Methods
// ============================================================================

func TestTemperatureToMethod(t *testing.T) {
	// #100C.to("F") should give #212F
	evaluated := testEvalUnit(`#100C.to("F")`)
	if err, ok := evaluated.(*evaluator.Error); ok {
		t.Fatalf("#100C.to('F') error: [%s] %s", err.Code, err.Message)
	}
	u := evaluated.(*evaluator.Unit)
	if u.DisplayHint != "F" {
		t.Errorf("expected F, got %s", u.DisplayHint)
	}
	val := evaluator.DecodeTempFromSubK(u.Amount, "F")
	if val != 212.0 {
		t.Errorf("expected 212, got %f", val)
	}

	// #32F.to("C") should give #0C
	evaluated2 := testEvalUnit(`#32F.to("C")`)
	u2 := evaluated2.(*evaluator.Unit)
	val2 := evaluator.DecodeTempFromSubK(u2.Amount, "C")
	if val2 != 0.0 {
		t.Errorf("#32F.to('C'): expected 0, got %f", val2)
	}

	// #100C.to("K") should give #373.15K
	evaluated3 := testEvalUnit(`#100C.to("K")`)
	u3 := evaluated3.(*evaluator.Unit)
	val3 := evaluator.DecodeTempFromSubK(u3.Amount, "K")
	if val3 != 373.15 {
		t.Errorf("#100C.to('K'): expected 373.15, got %f", val3)
	}
}

func TestTemperatureToMethodWrongFamily(t *testing.T) {
	evaluated := testEvalUnit(`#100C.to("m")`)
	testExpectedUnitError(t, `#100C.to("m")`, evaluated, "convert")
}

func TestTemperatureAbsMethod(t *testing.T) {
	// abs of negative display value
	evaluated := testEvalUnit(`(#-40C).abs()`)
	if err, ok := evaluated.(*evaluator.Error); ok {
		t.Fatalf("abs error: [%s] %s", err.Code, err.Message)
	}
	u := evaluated.(*evaluator.Unit)
	val := evaluator.DecodeTempFromSubK(u.Amount, "C")
	if val != 40.0 {
		t.Errorf("(#-40C).abs(): expected 40, got %f", val)
	}

	// abs of positive display value should be unchanged
	evaluated2 := testEvalUnit(`#100C.abs()`)
	u2 := evaluated2.(*evaluator.Unit)
	val2 := evaluator.DecodeTempFromSubK(u2.Amount, "C")
	if val2 != 100.0 {
		t.Errorf("#100C.abs(): expected 100, got %f", val2)
	}
}

func TestTemperatureFormatMethod(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#37.5C.format()`, `37.5C`},
		{`#37.5C.format(0)`, `38C`},
		{`#100C.format()`, `100C`},
		{`#100C.repr()`, `#100C`},
		{`#-40F.repr()`, `#-40F`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

func TestTemperatureToDictMethod(t *testing.T) {
	evaluated := testEvalUnit(`#100C.toDict()`)
	if err, ok := evaluated.(*evaluator.Error); ok {
		t.Fatalf("toDict error: [%s] %s", err.Code, err.Message)
	}
	dict, ok := evaluated.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T", evaluated)
	}
	// Check key count
	if len(dict.Pairs) != 4 {
		t.Errorf("expected 4 keys, got %d", len(dict.Pairs))
	}
}

func TestTemperatureToFractionMethod(t *testing.T) {
	// Temperature fractions don't apply — should return decimal string
	evaluated := testEvalUnit(`#100C.toFraction()`)
	if err, ok := evaluated.(*evaluator.Error); ok {
		t.Fatalf("toFraction error: [%s] %s", err.Code, err.Message)
	}
	s, ok := evaluated.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T", evaluated)
	}
	if s.Value != "100C" {
		t.Errorf("expected '100C', got '%s'", s.Value)
	}
}

// ============================================================================
// Temperature String Interpolation
// ============================================================================

func TestTemperatureInterpolation(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let t = #37.5C; `Temp: {t}`", `Temp: 37.5C`},
		{"let t = #212F; `Temp: {t}`", `Temp: 212F`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Temperature PLN Serialize
// ============================================================================

func TestTemperaturePLNSerialize(t *testing.T) {
	tests := []struct {
		name     string
		unit     *evaluator.Unit
		expected string
	}{
		{
			name:     "Celsius integer",
			unit:     &evaluator.Unit{Amount: evaluator.EncodeTempToSubK(100, "C"), Family: "temperature", System: "SI", DisplayHint: "C"},
			expected: "#100C",
		},
		{
			name:     "Fahrenheit integer",
			unit:     &evaluator.Unit{Amount: evaluator.EncodeTempToSubK(212, "F"), Family: "temperature", System: "US", DisplayHint: "F"},
			expected: "#212F",
		},
		{
			name:     "Kelvin zero",
			unit:     &evaluator.Unit{Amount: 0, Family: "temperature", System: "SI", DisplayHint: "K"},
			expected: "#0K",
		},
		{
			name:     "Negative Celsius",
			unit:     &evaluator.Unit{Amount: evaluator.EncodeTempToSubK(-40, "C"), Family: "temperature", System: "SI", DisplayHint: "C"},
			expected: "#-40C",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pln.Serialize(tt.unit)
			if err != nil {
				t.Fatalf("serialize error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// ============================================================================
// Temperature PLN Round-Trip
// ============================================================================

func TestTemperaturePLNRoundTrip(t *testing.T) {
	tests := []struct {
		input string
	}{
		{`#100C`},
		{`#212F`},
		{`#0K`},
		{`#-40C`},
		{`#-40F`},
		{`#37.5C`},
		{`#98.6F`},
		{`#273.15K`},
		{`#-273.15C`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEvalUnit(tt.input)
			if err, ok := evaluated.(*evaluator.Error); ok {
				t.Fatalf("eval error: [%s] %s", err.Code, err.Message)
			}
			u, ok := evaluated.(*evaluator.Unit)
			if !ok {
				t.Fatalf("expected Unit, got %T: %s", evaluated, evaluated.Inspect())
			}

			plnStr, err := pln.Serialize(u)
			if err != nil {
				t.Fatalf("serialize error: %v", err)
			}

			deserialized, err := pln.Deserialize(plnStr, nil, nil)
			if err != nil {
				t.Fatalf("deserialize error for %q: %v", plnStr, err)
			}

			u2, ok := deserialized.(*evaluator.Unit)
			if !ok {
				t.Fatalf("deserialized to %T, expected Unit", deserialized)
			}

			if u.Amount != u2.Amount {
				t.Errorf("Amount mismatch: %d vs %d (PLN: %q)", u.Amount, u2.Amount, plnStr)
			}
			if u.Family != u2.Family {
				t.Errorf("Family mismatch: %s vs %s", u.Family, u2.Family)
			}
			if u.DisplayHint != u2.DisplayHint {
				t.Errorf("DisplayHint mismatch: %s vs %s", u.DisplayHint, u2.DisplayHint)
			}
		})
	}
}

// ============================================================================
// Temperature Cross-Family Error
// ============================================================================

func TestTemperatureCrossFamilyError(t *testing.T) {
	tests := []struct {
		input          string
		expectedSubstr string
	}{
		{`#5m + #5C`, "length"},
		{`#1L + #1C`, "volume"},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnitError(t, tt.input, evaluated, tt.expectedSubstr)
	}
}

// ============================================================================
// Phase 2: Volume Literals
// ============================================================================

func TestVolumeLiteralSI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#500mL`, `#500mL`},
		{`#1000mL`, `#1000mL`},
		{`#2L`, `#2L`},
		{`#2.5L`, `#2.5L`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestVolumeLiteralUS(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#8floz`, `#8floz`},
		{`#1cup`, `#1cup`},
		{`#1pt`, `#1pt`},
		{`#1qt`, `#1qt`},
		{`#1gal`, `#1gal`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestVolumeLiteralUSFraction(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#1/3cup`, `#1/3cup`},
		{`#1/2gal`, `#1/2gal`},
		{`#3/8cup`, `#3/8cup`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestVolumeLiteralUSMixedNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#1+1/2gal`, `#1+1/2gal`},
		{`#2+3/4qt`, `#2+3/4qt`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Volume Properties
// ============================================================================

func TestVolumeProperties(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#1L.value`, `1`},
		{`#1L.unit`, `L`},
		{`#1L.family`, `volume`},
		{`#1L.system`, `SI`},
		{`#1gal.system`, `US`},
		{`#1gal.family`, `volume`},
		{`#1cup.unit`, `cup`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Volume Arithmetic — Same System
// ============================================================================

func TestVolumeAddSameSystem(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#500mL + #500mL`, `#1000mL`},
		{`#1L + #1L`, `#2L`},
		{`#4floz + #4floz`, `#8floz`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestVolumeSubtractSameSystem(t *testing.T) {
	// #1L - #500mL: both SI. 1,000,000 - 500,000 = 500,000 µL with DisplayHint=L → #0.5L
	eval1 := testEvalUnit(`#1L - #500mL`)
	testExpectedUnit(t, `#1L - #500mL`, eval1, `#0.5L`)

	// #1gal - #1qt: both US. 128*HCN - 32*HCN = 96*HCN with hint=gal → 96/128 = 3/4 → #3/4gal
	eval2 := testEvalUnit(`#1gal - #1qt`)
	testExpectedUnit(t, `#1gal - #1qt`, eval2, `#3/4gal`)
}

func TestVolumeScalarMultiply(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#1cup * 3`, `#3cup`},
		{`#1cup * 8`, `#8cup`},
		{`3 * #1cup`, `#3cup`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestVolumeScalarDivide(t *testing.T) {
	evaluated := testEvalUnit(`#1gal / 4`)
	testExpectedUnit(t, `#1gal / 4`, evaluated, `#1/4gal`)
}

func TestVolumeRatio(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#1gal / #1qt`, `4`},
		{`#1cup / #1floz`, `8`},
		{`#1pt / #1cup`, `2`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Volume Fraction Exactness
// ============================================================================

func TestVolumeFractionExactness(t *testing.T) {
	// #1/3cup + #1/3cup + #1/3cup should exactly equal #1cup
	evaluated := testEvalUnit(`#1/3cup + #1/3cup + #1/3cup`)
	testExpectedUnit(t, `#1/3cup + #1/3cup + #1/3cup`, evaluated, `#1cup`)
}

func TestVolumeThreeEighthsPlusFiveEighths(t *testing.T) {
	evaluated := testEvalUnit(`#3/8cup + #5/8cup`)
	testExpectedUnit(t, `#3/8cup + #5/8cup`, evaluated, `#1cup`)
}

// ============================================================================
// Volume Cross-System
// ============================================================================

func TestVolumeCrossSystemAdd(t *testing.T) {
	// #1L + #1floz: left is SI, result in L (display hint from left)
	evaluated := testEvalUnit(`#1L + #1floz`)
	if err, ok := evaluated.(*evaluator.Error); ok {
		t.Fatalf("#1L + #1floz error: [%s] %s", err.Code, err.Message)
	}
	u, ok := evaluated.(*evaluator.Unit)
	if !ok {
		t.Fatalf("expected Unit, got %T", evaluated)
	}
	if u.DisplayHint != "L" {
		t.Errorf("expected DisplayHint=L, got %s", u.DisplayHint)
	}
	if u.System != "SI" {
		t.Errorf("expected System=SI, got %s", u.System)
	}
}

func TestVolumeCrossFamilyError(t *testing.T) {
	evaluated := testEvalUnit(`#1L + #1kg`)
	testExpectedUnitError(t, `#1L + #1kg`, evaluated, "volume")
}

// ============================================================================
// Volume Constructors
// ============================================================================

func TestVolumeConstructors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`litres(2)`, `#2L`},
		{`liters(2)`, `#2L`},
		{`gallons(1)`, `#1gal`},
		{`cups(1)`, `#1cup`},
		{`pints(1)`, `#1pt`},
		{`quarts(1)`, `#1qt`},
		{`fluidounces(8)`, `#8floz`},
		{`millilitres(500)`, `#500mL`},
		{`milliliters(500)`, `#500mL`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedUnit(t, tt.input, evaluated, tt.expected)
	}
}

func TestVolumeConstructorConversion(t *testing.T) {
	// litres(#1gal) should give a value in L close to 3.785
	evaluated := testEvalUnit(`litres(#1gal)`)
	if err, ok := evaluated.(*evaluator.Error); ok {
		t.Fatalf("litres(#1gal) error: [%s] %s", err.Code, err.Message)
	}
	u, ok := evaluated.(*evaluator.Unit)
	if !ok {
		t.Fatalf("expected Unit, got %T", evaluated)
	}
	if u.DisplayHint != "L" {
		t.Errorf("expected DisplayHint=L, got %s", u.DisplayHint)
	}
	if u.Family != "volume" {
		t.Errorf("expected family=volume, got %s", u.Family)
	}
}

func TestVolumeConstructorWrongFamily(t *testing.T) {
	evaluated := testEvalUnit(`cups(#5m)`)
	testExpectedUnitError(t, `cups(#5m)`, evaluated, "convert")
}

// ============================================================================
// Volume Methods
// ============================================================================

func TestVolumeToMethod(t *testing.T) {
	evaluated := testEvalUnit(`#1gal.to("qt")`)
	if err, ok := evaluated.(*evaluator.Error); ok {
		t.Fatalf("to error: [%s] %s", err.Code, err.Message)
	}
	u := evaluated.(*evaluator.Unit)
	if u.DisplayHint != "qt" {
		t.Errorf("expected qt, got %s", u.DisplayHint)
	}
	// 1 gal = 4 qt
	val := float64(u.Amount) / float64(evaluator.USSubUnitsPerUnit("qt"))
	if val != 4.0 {
		t.Errorf("1gal.to('qt') value: expected 4, got %f", val)
	}
}

func TestVolumeReprMethod(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#500mL.repr()`, `#500mL`},
		{`#2L.repr()`, `#2L`},
		{`#1/3cup.repr()`, `#1/3cup`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

func TestVolumeToFraction(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#1/3cup.toFraction()`, `1/3cup`},
		{`#1/2gal.toFraction()`, `1/2gal`},
	}
	for _, tt := range tests {
		evaluated := testEvalUnit(tt.input)
		testExpectedValue(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Volume PLN Round-Trip
// ============================================================================

func TestVolumePLNRoundTrip(t *testing.T) {
	tests := []struct {
		input string
	}{
		{`#500mL`},
		{`#2L`},
		{`#2.5L`},
		{`#8floz`},
		{`#1cup`},
		{`#1/3cup`},
		{`#1pt`},
		{`#1qt`},
		{`#1gal`},
		{`#1+1/2gal`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEvalUnit(tt.input)
			if err, ok := evaluated.(*evaluator.Error); ok {
				t.Fatalf("eval error: [%s] %s", err.Code, err.Message)
			}
			u, ok := evaluated.(*evaluator.Unit)
			if !ok {
				t.Fatalf("expected Unit, got %T: %s", evaluated, evaluated.Inspect())
			}

			plnStr, err := pln.Serialize(u)
			if err != nil {
				t.Fatalf("serialize error: %v", err)
			}

			deserialized, err := pln.Deserialize(plnStr, nil, nil)
			if err != nil {
				t.Fatalf("deserialize error for %q: %v", plnStr, err)
			}

			u2, ok := deserialized.(*evaluator.Unit)
			if !ok {
				t.Fatalf("deserialized to %T, expected Unit", deserialized)
			}

			if u.Amount != u2.Amount {
				t.Errorf("Amount mismatch: %d vs %d (PLN: %q)", u.Amount, u2.Amount, plnStr)
			}
			if u.Family != u2.Family {
				t.Errorf("Family mismatch: %s vs %s", u.Family, u2.Family)
			}
			if u.DisplayHint != u2.DisplayHint {
				t.Errorf("DisplayHint mismatch: %s vs %s", u.DisplayHint, u2.DisplayHint)
			}
		})
	}
}

// ============================================================================
// Volume PLN Serialize
// ============================================================================

func TestVolumePLNSerialize(t *testing.T) {
	tests := []struct {
		name     string
		unit     *evaluator.Unit
		expected string
	}{
		{
			name:     "SI mL integer",
			unit:     &evaluator.Unit{Amount: 500_000_000, Family: "volume", System: "SI", DisplayHint: "mL"},
			expected: "#500mL",
		},
		{
			name:     "SI L integer",
			unit:     &evaluator.Unit{Amount: 2_000_000_000, Family: "volume", System: "SI", DisplayHint: "L"},
			expected: "#2L",
		},
		{
			name:     "US cup integer",
			unit:     &evaluator.Unit{Amount: evaluator.HCN * 8, Family: "volume", System: "US", DisplayHint: "cup"},
			expected: "#1cup",
		},
		{
			name:     "US gal integer",
			unit:     &evaluator.Unit{Amount: evaluator.HCN * 128, Family: "volume", System: "US", DisplayHint: "gal"},
			expected: "#1gal",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pln.Serialize(tt.unit)
			if err != nil {
				t.Fatalf("serialize error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// ============================================================================
// Volume Lexer Regression — #1gal should not be #1g + "al"
// ============================================================================

func TestVolumeLexerGalNotGrams(t *testing.T) {
	// #1gal should be a volume, not mass
	evaluated := testEvalUnit(`#1gal`)
	u, ok := evaluated.(*evaluator.Unit)
	if !ok {
		t.Fatalf("expected Unit, got %T: %s", evaluated, evaluated.Inspect())
	}
	if u.Family != "volume" {
		t.Errorf("#1gal: expected family=volume, got %s", u.Family)
	}
	if u.DisplayHint != "gal" {
		t.Errorf("#1gal: expected DisplayHint=gal, got %s", u.DisplayHint)
	}
}

func TestLexerGramsRegression(t *testing.T) {
	// #5g should still be grams
	evaluated := testEvalUnit(`#5g`)
	u, ok := evaluated.(*evaluator.Unit)
	if !ok {
		t.Fatalf("expected Unit, got %T: %s", evaluated, evaluated.Inspect())
	}
	if u.Family != "mass" {
		t.Errorf("#5g: expected family=mass, got %s", u.Family)
	}
}

// ============================================================================
// Phase 1 Regression — ensure existing tests still work
// ============================================================================

func TestPhase1RegressionLengthLiteral(t *testing.T) {
	evaluated := testEvalUnit(`#12m`)
	testExpectedUnit(t, `#12m`, evaluated, `#12m`)
}

func TestPhase1RegressionMassLiteral(t *testing.T) {
	evaluated := testEvalUnit(`#5kg`)
	testExpectedUnit(t, `#5kg`, evaluated, `#5kg`)
}

func TestPhase1RegressionDataLiteral(t *testing.T) {
	evaluated := testEvalUnit(`#1024B`)
	testExpectedUnit(t, `#1024B`, evaluated, `#1024B`)
}
