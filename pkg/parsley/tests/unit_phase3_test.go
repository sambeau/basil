package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/pln"
)

// ============================================================================
// 9a. Scale infrastructure (unit-level tests)
// ============================================================================

func TestScaleInfrastructure(t *testing.T) {
	// Scale=0 values behave identically to pre-Scale code
	tests := []struct {
		input    string
		expected string
	}{
		// Basic Scale=0 operations (should be identical to existing)
		{`#5m + #3m`, `#8m`},
		{`#1kg * 2`, `#2kg`},
		{`#1L + #500mL`, `#1.5L`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedUnit(t, tt.input, result, tt.expected)
		})
	}
}

// ============================================================================
// 9b. Area literal parsing
// ============================================================================

func TestAreaLiteralSI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// mm2
		{`#0mm2`, `#0mm2`},
		{`#1mm2`, `#1mm2`},
		{`#5mm2`, `#5mm2`},
		{`#100mm2`, `#100mm2`},
		// cm2
		{`#0cm2`, `#0cm2`},
		{`#1cm2`, `#1cm2`},
		{`#50cm2`, `#50cm2`},
		// m2
		{`#0m2`, `#0m2`},
		{`#1m2`, `#1m2`},
		{`#100m2`, `#100m2`},
		// km2
		{`#0km2`, `#0km2`},
		{`#1km2`, `#1km2`},
		{`#5km2`, `#5km2`},
		// Decimals
		{`#1.5m2`, `#1.5m2`},
		{`#0.5km2`, `#0.5km2`},
		{`#12.5m2`, `#12.5m2`},
		// Negative
		{`#-100m2`, `#-100m2`},
		{`#-5mm2`, `#-5mm2`},
		// Fractions
		{`#1/2m2`, `#0.5m2`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedUnit(t, tt.input, result, tt.expected)
		})
	}
}

func TestAreaLiteralUS(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// in2
		{`#0in2`, `#0in2`},
		{`#1in2`, `#1in2`},
		{`#144in2`, `#144in2`},
		// ft2
		{`#0ft2`, `#0ft2`},
		{`#1ft2`, `#1ft2`},
		{`#1500ft2`, `#1500ft2`},
		// yd2
		{`#1yd2`, `#1yd2`},
		// ac
		{`#1ac`, `#1ac`},
		{`#2.5ac`, `#2.5ac`},
		{`#640ac`, `#640ac`},
		// mi2
		{`#1mi2`, `#1mi2`},
		{`#0.5mi2`, `#0.5mi2`},
		// Fractions — US area uses decimal-only display (no HCN fractions)
		{`#1/3ac`, `#0.3333333333333333ac`}, // 6,272,640 / 3 = 2,090,880 in² (exact, displayed as decimal)
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedUnit(t, tt.input, result, tt.expected)
		})
	}
}

func TestAreaLiteralProperties(t *testing.T) {
	// Verify internal representation
	tests := []struct {
		input    string
		expected string
	}{
		{`#100m2.family`, `area`},
		{`#100m2.system`, `SI`},
		{`#100m2.unit`, `m2`},
		{`#640ac.family`, `area`},
		{`#640ac.system`, `US`},
		{`#640ac.unit`, `ac`},
		{`#5mm2.unit`, `mm2`},
		{`#1ft2.unit`, `ft2`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedValue(t, tt.input, result, tt.expected)
		})
	}
}

func TestAreaLiteralValues(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#100m2.value`, `100`},
		{`#5mm2.value`, `5`},
		{`#1.5km2.value`, `1.5`},
		{`#640ac.value`, `640`},
		{`#1500ft2.value`, `1500`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedValue(t, tt.input, result, tt.expected)
		})
	}
}

// ============================================================================
// 9c. Within-system arithmetic
// ============================================================================

func TestAreaAddSameSystem(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#1m2 + #5000cm2`, `#1.5m2`},
		{`#100mm2 + #200mm2`, `#300mm2`},
		{`#1km2 + #1km2`, `#2km2`},
		{`#1000ft2 + #500ft2`, `#1500ft2`},
		{`#1mi2 + #1mi2`, `#2mi2`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedUnit(t, tt.input, result, tt.expected)
		})
	}
}

func TestAreaSubtractSameSystem(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#1000ft2 - #500ft2`, `#500ft2`},
		{`#2m2 - #1m2`, `#1m2`},
		{`#10ac - #5ac`, `#5ac`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedUnit(t, tt.input, result, tt.expected)
		})
	}
}

func TestAreaScalarMultiplication(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#1m2 * 3`, `#3m2`},
		{`3 * #1m2`, `#3m2`},
		{`#10ac / 5`, `#2ac`},
		{`#100ft2 * 2`, `#200ft2`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedUnit(t, tt.input, result, tt.expected)
		})
	}
}

func TestAreaRatio(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#2m2 / #1m2`, `2`},
		{`#10ac / #5ac`, `2`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedValue(t, tt.input, result, tt.expected)
		})
	}
}

// ============================================================================
// 9d. Scale arithmetic
// ============================================================================

func TestAreaScaleArithmetic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Large SI area that requires scale
		{`#510000000km2 + #100km2`, `#510000100km2`},
		// Large + small: small absorbed
		{`#510000000km2 + #3mm2`, `#510000000km2`},
		// Scaled multiplication
		{`#510000000km2 * 2`, `#1020000000km2`},
		// Scaled comparison
		{`#510000000km2 > #9000000km2`, `true`},
		{`#510000000km2 > #1000000000km2`, `false`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedValue(t, tt.input, result, tt.expected)
		})
	}
}

func TestAreaScaleInspect(t *testing.T) {
	// Verify that scaled values have Scale > 0
	input := `#510000000km2.inspect()`
	result := testEvalUnit(input)
	if result == nil {
		t.Fatalf("For input '%s': got nil", input)
	}
	if err, ok := result.(*evaluator.Error); ok {
		t.Fatalf("For input '%s': got error: [%s] %s", input, err.Code, err.Message)
	}
	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T", result)
	}
	// Check that scale > 0
	scaleExpr, exists := dict.Pairs["scale"]
	if !exists {
		t.Fatalf("inspect() missing 'scale' key")
	}
	env := evaluator.NewEnvironment()
	scaleVal := evaluator.Eval(scaleExpr, env)
	scaleInt, ok := scaleVal.(*evaluator.Integer)
	if !ok {
		t.Fatalf("scale should be integer, got %T", scaleVal)
	}
	if scaleInt.Value <= 0 {
		t.Errorf("expected scale > 0 for #510000000km2, got %d", scaleInt.Value)
	}
}

// ============================================================================
// 9e. Cross-system arithmetic
// ============================================================================

func TestAreaCrossSystemAdd(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Within US, different display
		{`#1ft2 + #144in2`, `#2ft2`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedUnit(t, tt.input, result, tt.expected)
		})
	}
}

// ============================================================================
// 9f. Cross-system comparison
// ============================================================================

func TestAreaWithinUSEquality(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#144in2 == #1ft2`, `true`},
		{`#1296in2 == #1yd2`, `true`},
		{`#640ac == #1mi2`, `true`},
		{`#9ft2 == #1yd2`, `true`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedValue(t, tt.input, result, tt.expected)
		})
	}
}

func TestAreaWithinSIEquality(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#100cm2 == #10000mm2`, `true`},
		{`#1m2 == #1000000mm2`, `true`},
		{`#1km2 == #1000000m2`, `true`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedValue(t, tt.input, result, tt.expected)
		})
	}
}

// ============================================================================
// 9g. Constructors
// ============================================================================

func TestAreaConstructors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`squaremetres(100)`, `#100m2`},
		{`squaremeters(100)`, `#100m2`},
		{`squaremillimetres(5)`, `#5mm2`},
		{`squaremillimeters(5)`, `#5mm2`},
		{`squarecentimetres(50)`, `#50cm2`},
		{`squarecentimeters(50)`, `#50cm2`},
		{`squarekilometres(1)`, `#1km2`},
		{`squarekilometers(1)`, `#1km2`},
		{`squareinches(1)`, `#1in2`},
		{`squarefeet(1)`, `#1ft2`},
		{`squareyards(1)`, `#1yd2`},
		{`acres(640)`, `#640ac`},
		{`squaremiles(1)`, `#1mi2`},
		// Generic constructor
		{`unit(100, "m2")`, `#100m2`},
		{`unit(500, "cm2")`, `#500cm2`},
		{`unit(640, "ac")`, `#640ac`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedUnit(t, tt.input, result, tt.expected)
		})
	}
}

func TestAreaConstructorConversion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Within-system conversion via constructor
		{`squarefeet(#1yd2)`, `#9ft2`},
		{`acres(#1mi2)`, `#640ac`},
		{`squaremetres(#1km2)`, `#1000000m2`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedUnit(t, tt.input, result, tt.expected)
		})
	}
}

func TestAreaConstructorWrongFamily(t *testing.T) {
	input := `squaremetres(#5kg)`
	result := testEvalUnit(input)
	testExpectedUnitError(t, input, result, "convert")
}

func TestAreaConstructorScaleOverflow(t *testing.T) {
	// Very large value should activate Scale
	input := `squarekilometres(510000000)`
	result := testEvalUnit(input)
	if result == nil {
		t.Fatal("got nil")
	}
	if err, ok := result.(*evaluator.Error); ok {
		t.Fatalf("got error: [%s] %s", err.Code, err.Message)
	}
	u, ok := result.(*evaluator.Unit)
	if !ok {
		t.Fatalf("expected Unit, got %T", result)
	}
	if u.Scale <= 0 {
		t.Errorf("expected Scale > 0 for squarekilometres(510000000), got Scale=%d", u.Scale)
	}
}

// ============================================================================
// 9h. Properties
// ============================================================================

func TestAreaProperties(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#100m2.value`, `100`},
		{`#100m2.unit`, `m2`},
		{`#100m2.family`, `area`},
		{`#100m2.system`, `SI`},
		{`#640ac.system`, `US`},
		{`#5mm2.value`, `5`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedValue(t, tt.input, result, tt.expected)
		})
	}
}

// ============================================================================
// 9i. Methods
// ============================================================================

func TestAreaToMethod(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Within-system conversions
		{`#1mi2.to("ac")`, `#640ac`},
		{`#1yd2.to("ft2")`, `#9ft2`},
		{`#1km2.to("m2")`, `#1000000m2`},
		{`#1m2.to("cm2")`, `#10000cm2`},
		// Cross-system conversions (US → SI)
		{`#1in2.to("mm2")`, `#645mm2`},
		{`#1ft2.to("cm2")`, `#929cm2`},
		{`#1yd2.to("m2")`, `#0.836127m2`},
		// Cross-system conversions (SI → US)
		{`#100cm2.to("in2")`, `#15in2`},
		{`#1m2.to("ft2")`, `#10.76388888888889ft2`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedUnit(t, tt.input, result, tt.expected)
		})
	}
}

func TestAreaAbsMethod(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#100m2.abs()`, `#100m2`},
		{`#-100m2.abs()`, `#100m2`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedUnit(t, tt.input, result, tt.expected)
		})
	}
}

// ============================================================================
// 9j. PLN round-trip
// ============================================================================

func TestAreaPLNRoundTrip(t *testing.T) {
	tests := []struct {
		input string
	}{
		{`#100m2`},
		{`#5mm2`},
		{`#50cm2`},
		{`#1.5km2`},
		{`#0.5km2`},
		{`#1500ft2`},
		{`#2.5ac`},
		{`#640ac`},
		{`#1mi2`},
		{`#144in2`},
		{`#1yd2`},
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

			// Serialize to PLN
			plnStr, err := pln.Serialize(u)
			if err != nil {
				t.Fatalf("serialize error: %v", err)
			}

			// Deserialize back
			deserialized, parseErr := pln.Deserialize(plnStr, nil, nil)
			if parseErr != nil {
				t.Fatalf("deserialize error for %q: %v", plnStr, parseErr)
			}

			u2, ok := deserialized.(*evaluator.Unit)
			if !ok {
				t.Fatalf("deserialized to %T, expected Unit", deserialized)
			}

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
			if u.Scale != u2.Scale {
				t.Errorf("Scale mismatch: %d vs %d", u.Scale, u2.Scale)
			}
		})
	}
}

func TestAreaScaledPLNRoundTrip(t *testing.T) {
	// Scaled values: serialize → parse → verify Amount and Scale match
	input := `#510000000km2`
	evaluated := testEvalUnit(input)
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

	deserialized, parseErr := pln.Deserialize(plnStr, nil, nil)
	if parseErr != nil {
		t.Fatalf("deserialize error for %q: %v", plnStr, parseErr)
	}

	u2, ok := deserialized.(*evaluator.Unit)
	if !ok {
		t.Fatalf("deserialized to %T, expected Unit", deserialized)
	}

	if u.Amount != u2.Amount {
		t.Errorf("Amount mismatch: %d vs %d", u.Amount, u2.Amount)
	}
	if u.Scale != u2.Scale {
		t.Errorf("Scale mismatch: %d vs %d", u.Scale, u2.Scale)
	}
	if u.Family != u2.Family {
		t.Errorf("Family mismatch: %s vs %s", u.Family, u2.Family)
	}
}

// ============================================================================
// 9k. Display
// ============================================================================

func TestAreaDisplay(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// US area: decimal display (not fractions)
		{`#1500ft2`, `#1500ft2`},
		{`#100m2`, `#100m2`},
		{`#2.5ac`, `#2.5ac`},
		{`#5mm2`, `#5mm2`},
		{`#0.5mi2`, `#0.5mi2`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedUnit(t, tt.input, result, tt.expected)
		})
	}
}

// ============================================================================
// 9l. Errors
// ============================================================================

func TestAreaErrors(t *testing.T) {
	tests := []struct {
		input          string
		expectedSubstr string
	}{
		// Cross-family error
		{`#5m2 + #5kg`, "family"},
		// Unit × unit error
		{`#5m2 * #5m2`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			if result == nil {
				t.Fatalf("expected error, got nil")
			}
			_, ok := result.(*evaluator.Error)
			if !ok {
				t.Errorf("expected error, got %T: %s", result, result.Inspect())
			}
		})
	}
}

// ============================================================================
// 9m. Existing family regression
// ============================================================================

func TestPhase3RegressionLength(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#1m + #50cm`, `#1.5m`},
		{`#1km * 2`, `#2km`},
		{`#1ft + #6in`, `#1+1/2ft`},
		{`#12in == #1ft`, `true`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedValue(t, tt.input, result, tt.expected)
		})
	}
}

func TestPhase3RegressionMass(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#500g + #500g`, `#1000g`},
		{`#1kg * 3`, `#3kg`},
		{`#16oz == #1lb`, `true`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedValue(t, tt.input, result, tt.expected)
		})
	}
}

func TestPhase3RegressionVolume(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#500mL + #500mL`, `#1000mL`},
		{`#1L + #500mL`, `#1.5L`},
		{`#1gal * 2`, `#2gal`},
		{`#128floz == #1gal`, `true`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedValue(t, tt.input, result, tt.expected)
		})
	}
}

func TestPhase3RegressionTemperature(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#100C.to("F")`, `#212F`},
		{`#0C.to("K")`, `#273.15K`},
		{`#32F.to("C")`, `#0C`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedValue(t, tt.input, result, tt.expected)
		})
	}
}

func TestPhase3RegressionData(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#1024B + #1024B`, `#2048B`},
		{`#1MB * 2`, `#2MB`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedValue(t, tt.input, result, tt.expected)
		})
	}
}

// ============================================================================
// 13a. kL literal parsing
// ============================================================================

func TestKilolitreLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#1kL`, `#1kL`},
		{`#5kL`, `#5kL`},
		{`#5.5kL`, `#5.5kL`},
		{`#0kL`, `#0kL`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedUnit(t, tt.input, result, tt.expected)
		})
	}
}

func TestKilolitreEquivalence(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#1kL == #1000L`, `true`},
		{`#1kL == #1000000mL`, `true`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedValue(t, tt.input, result, tt.expected)
		})
	}
}

func TestKilolitreProperties(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#5kL.value`, `5`},
		{`#5kL.unit`, `kL`},
		{`#5kL.family`, `volume`},
		{`#5kL.system`, `SI`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedValue(t, tt.input, result, tt.expected)
		})
	}
}

// ============================================================================
// 13b. kL constructors
// ============================================================================

func TestKilolitreConstructors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`kilolitres(1)`, `#1kL`},
		{`kiloliters(1)`, `#1kL`},
		{`kilolitres(5)`, `#5kL`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedUnit(t, tt.input, result, tt.expected)
		})
	}
}

func TestKilolitreConstructorConversion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`kilolitres(#1000L)`, `#1kL`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedUnit(t, tt.input, result, tt.expected)
		})
	}
}

// ============================================================================
// 13c. kL arithmetic
// ============================================================================

func TestKilolitreArithmetic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#1kL + #500L`, `#1.5kL`},
		{`#10kL - #3kL`, `#7kL`},
		{`#2kL * 5`, `#10kL`},
		{`#10kL / 2`, `#5kL`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedUnit(t, tt.input, result, tt.expected)
		})
	}
}

// ============================================================================
// 13e. kL methods
// ============================================================================

func TestKilolitreMethods(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`#5kL.to("L")`, `#5000L`},
		{`#1000L.to("kL")`, `#1kL`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedUnit(t, tt.input, result, tt.expected)
		})
	}
}

// ============================================================================
// 13a-f. kL PLN round-trip
// ============================================================================

func TestKilolitrePLNRoundTrip(t *testing.T) {
	tests := []struct {
		input string
	}{
		{`#1kL`},
		{`#5.5kL`},
		{`#100kL`},
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

			deserialized, parseErr := pln.Deserialize(plnStr, nil, nil)
			if parseErr != nil {
				t.Fatalf("deserialize error for %q: %v", plnStr, parseErr)
			}

			u2, ok := deserialized.(*evaluator.Unit)
			if !ok {
				t.Fatalf("deserialized to %T, expected Unit", deserialized)
			}

			if u.Amount != u2.Amount {
				t.Errorf("Amount mismatch: %d vs %d", u.Amount, u2.Amount)
			}
			if u.Scale != u2.Scale {
				t.Errorf("Scale mismatch: %d vs %d", u.Scale, u2.Scale)
			}
			if u.Family != u2.Family {
				t.Errorf("Family mismatch: %s vs %s", u.Family, u2.Family)
			}
		})
	}
}

// ============================================================================
// 13f. Volume regression
// ============================================================================

func TestKilolitreVolumeRegression(t *testing.T) {
	// Existing volume operations should be completely unaffected
	tests := []struct {
		input    string
		expected string
	}{
		{`#500mL`, `#500mL`},
		{`#2L`, `#2L`},
		{`#2.5L`, `#2.5L`},
		{`#8floz`, `#8floz`},
		{`#1cup`, `#1cup`},
		{`#1pt + #1pt`, `#2pt`},
		{`#1gal * 2`, `#2gal`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedUnit(t, tt.input, result, tt.expected)
		})
	}
}

// ============================================================================
// Phase 3c: Unit Limits — .max and .min Properties
// ============================================================================

func TestUnitMaxProperty(t *testing.T) {
	tests := []struct {
		input string
		check func(t *testing.T, result evaluator.Object)
	}{
		{
			input: `#0m.max`,
			check: func(t *testing.T, result evaluator.Object) {
				t.Helper()
				u, ok := result.(*evaluator.Unit)
				if !ok {
					t.Fatalf("expected Unit, got %T: %s", result, result.Inspect())
				}
				if u.DisplayHint != "m" {
					t.Errorf("expected hint 'm', got '%s'", u.DisplayHint)
				}
				if u.Amount <= 0 {
					t.Errorf("expected positive amount, got %d", u.Amount)
				}
				if u.Family != "length" {
					t.Errorf("expected family 'length', got '%s'", u.Family)
				}
			},
		},
		{
			input: `#0km2.max`,
			check: func(t *testing.T, result evaluator.Object) {
				t.Helper()
				u, ok := result.(*evaluator.Unit)
				if !ok {
					t.Fatalf("expected Unit, got %T: %s", result, result.Inspect())
				}
				if u.DisplayHint != "km2" {
					t.Errorf("expected hint 'km2', got '%s'", u.DisplayHint)
				}
				// km2 subPerUnit = 10^12. Max = MaxInt64 / 10^12 * 10^12 ≈ 9.223e18
				// .value should be approximately 9,223,372 km²
				val := evaluator.Eval(
					nil, evaluator.NewEnvironment(),
				)
				_ = val
				if u.Amount <= 0 {
					t.Errorf("expected positive amount, got %d", u.Amount)
				}
			},
		},
		{
			input: `#0mm2.max`,
			check: func(t *testing.T, result evaluator.Object) {
				t.Helper()
				u, ok := result.(*evaluator.Unit)
				if !ok {
					t.Fatalf("expected Unit, got %T: %s", result, result.Inspect())
				}
				// mm2 subPerUnit = 1, so max amount = MaxInt64
				if u.Amount != 9223372036854775807 {
					t.Errorf("expected MaxInt64 amount for mm2.max, got %d", u.Amount)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			if err, ok := result.(*evaluator.Error); ok {
				t.Fatalf("got error: [%s] %s", err.Code, err.Message)
			}
			tt.check(t, result)
		})
	}
}

func TestUnitMinProperty(t *testing.T) {
	tests := []struct {
		input string
		check func(t *testing.T, result evaluator.Object)
	}{
		{
			input: `#0m.min`,
			check: func(t *testing.T, result evaluator.Object) {
				t.Helper()
				u, ok := result.(*evaluator.Unit)
				if !ok {
					t.Fatalf("expected Unit, got %T: %s", result, result.Inspect())
				}
				if u.Amount != 1 {
					t.Errorf("expected amount 1, got %d", u.Amount)
				}
				if u.DisplayHint != "m" {
					t.Errorf("expected hint 'm', got '%s'", u.DisplayHint)
				}
			},
		},
		{
			input: `#0mm2.min`,
			check: func(t *testing.T, result evaluator.Object) {
				t.Helper()
				u, ok := result.(*evaluator.Unit)
				if !ok {
					t.Fatalf("expected Unit, got %T: %s", result, result.Inspect())
				}
				if u.Amount != 1 {
					t.Errorf("expected amount 1, got %d", u.Amount)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			if err, ok := result.(*evaluator.Error); ok {
				t.Fatalf("got error: [%s] %s", err.Code, err.Message)
			}
			tt.check(t, result)
		})
	}
}

func TestUnitMaxMinValues(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// .max returns a Unit, so .value gives the numeric value
		{`#0m.min.value`, `1e-06`},
		{`#0mm2.min.value`, `1`},
		// .max and .min return Unit values usable in comparison
		{`#100m2 < #0m2.max`, `true`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedValue(t, tt.input, result, tt.expected)
		})
	}
}

func TestUnitTempMinMax(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// C.min = absolute zero = -273.15°C
		{`#0C.min.value`, `-273.15`},
		// F.min = absolute zero = -459.67°F
		{`#0F.min.value`, `-459.67`},
		// K.min = smallest positive kelvin (1/900 K)
		{`#0K.min.value > 0`, `true`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			testExpectedValue(t, tt.input, result, tt.expected)
		})
	}
}

// ============================================================================
// Suffix disambiguation
// ============================================================================

func TestAreaSuffixDisambiguation(t *testing.T) {
	// Ensure area suffixes don't interfere with length suffixes
	tests := []struct {
		input  string
		family string
	}{
		{`#5m2`, "area"},
		{`#5m`, "length"},
		{`#5mm2`, "area"},
		{`#5mm`, "length"},
		{`#5km2`, "area"},
		{`#5km`, "length"},
		{`#5in2`, "area"},
		{`#5in`, "length"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			if err, ok := result.(*evaluator.Error); ok {
				t.Fatalf("got error: [%s] %s", err.Code, err.Message)
			}
			u, ok := result.(*evaluator.Unit)
			if !ok {
				t.Fatalf("expected Unit, got %T: %s", result, result.Inspect())
			}
			if u.Family != tt.family {
				t.Errorf("expected family '%s', got '%s'", tt.family, u.Family)
			}
		})
	}
}

// ============================================================================
// US Area display: decimal (not fractions)
// ============================================================================

func TestUSAreaDecimalDisplay(t *testing.T) {
	// US area should use decimal display, not fraction display
	tests := []struct {
		input      string
		contains   string // what the output should contain
		notContain string // what the output should NOT contain
	}{
		{`#1500ft2`, "1500ft2", "/"},
		{`#2.5ac`, "2.5ac", "/"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := testEvalUnit(tt.input)
			if err, ok := result.(*evaluator.Error); ok {
				t.Fatalf("got error: [%s] %s", err.Code, err.Message)
			}
			display := result.Inspect()
			if !strings.Contains(display, tt.contains) {
				t.Errorf("expected display to contain '%s', got '%s'", tt.contains, display)
			}
			if tt.notContain != "" && strings.Contains(display, tt.notContain) {
				t.Errorf("expected display NOT to contain '%s', got '%s'", tt.notContain, display)
			}
		})
	}
}

// ============================================================================
// PLN parser fix: existing suffixes still work after the backward-scan replacement
// ============================================================================

func TestPLNParserFixRegression(t *testing.T) {
	// Verify all existing PLN round-trips still work after the suffix extraction fix
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
		{`#100C`},
		{`#212F`},
		{`#0K`},
		{`#500mL`},
		{`#2L`},
		{`#8floz`},
		{`#1gal`},
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

			deserialized, parseErr := pln.Deserialize(plnStr, nil, nil)
			if parseErr != nil {
				t.Fatalf("deserialize error for %q: %v", plnStr, parseErr)
			}

			u2, ok := deserialized.(*evaluator.Unit)
			if !ok {
				t.Fatalf("deserialized to %T, expected Unit", deserialized)
			}

			if u.Amount != u2.Amount {
				t.Errorf("Amount mismatch for %s: %d vs %d (PLN: %s)", tt.input, u.Amount, u2.Amount, plnStr)
			}
			if u.DisplayHint != u2.DisplayHint {
				t.Errorf("DisplayHint mismatch for %s: %s vs %s", tt.input, u.DisplayHint, u2.DisplayHint)
			}
		})
	}
}
