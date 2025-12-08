package tests

import (
	"math"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// evalMathTest helper that evaluates Parsley code and handles errors
func evalMathTest(t *testing.T, input string) evaluator.Object {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if result == nil {
		t.Fatal("result is nil")
	}

	return result
}

// =============================================================================
// Module Import Tests
// =============================================================================

func TestStdMathImport(t *testing.T) {
	input := `let {floor} = import @std/math
floor`

	result := evalMathTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.BUILTIN_OBJ {
		t.Errorf("expected BUILTIN, got %s", result.Type())
	}
}

func TestStdMathImportAll(t *testing.T) {
	input := `let math = import @std/math
math.PI`

	result := evalMathTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.FLOAT_OBJ {
		t.Errorf("expected FLOAT, got %s", result.Type())
	}

	f := result.(*evaluator.Float).Value
	if math.Abs(f-math.Pi) > 0.0001 {
		t.Errorf("expected PI (~3.14159), got %f", f)
	}
}

// =============================================================================
// Constants Tests
// =============================================================================

func TestMathConstants(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{"PI", `let math = import @std/math; math.PI`, math.Pi},
		{"E", `let math = import @std/math; math.E`, math.E},
		{"TAU", `let math = import @std/math; math.TAU`, math.Pi * 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalMathTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			f := result.(*evaluator.Float).Value
			if math.Abs(f-tt.expected) > 0.0001 {
				t.Errorf("expected %f, got %f", tt.expected, f)
			}
		})
	}
}

// =============================================================================
// Rounding Functions Tests
// =============================================================================

func TestMathFloor(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`let {floor} = import @std/math; floor(3.7)`, 3},
		{`let {floor} = import @std/math; floor(3.2)`, 3},
		{`let {floor} = import @std/math; floor(3)`, 3},
		{`let {floor} = import @std/math; floor(-2.3)`, -3},
		{`let {floor} = import @std/math; floor(-2.9)`, -3},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error for %s: %s", tt.input, result.Inspect())
		}

		if result.Type() != evaluator.INTEGER_OBJ {
			t.Errorf("expected INTEGER, got %s for %s", result.Type(), tt.input)
			continue
		}

		if result.(*evaluator.Integer).Value != tt.expected {
			t.Errorf("expected %d, got %d for %s", tt.expected, result.(*evaluator.Integer).Value, tt.input)
		}
	}
}

func TestMathCeil(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`let {ceil} = import @std/math; ceil(3.2)`, 4},
		{`let {ceil} = import @std/math; ceil(3.9)`, 4},
		{`let {ceil} = import @std/math; ceil(3)`, 3},
		{`let {ceil} = import @std/math; ceil(-2.3)`, -2},
		{`let {ceil} = import @std/math; ceil(-2.9)`, -2},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error: %s", result.Inspect())
		}

		if result.Type() != evaluator.INTEGER_OBJ {
			t.Errorf("expected INTEGER, got %s", result.Type())
			continue
		}

		if result.(*evaluator.Integer).Value != tt.expected {
			t.Errorf("expected %d, got %d", tt.expected, result.(*evaluator.Integer).Value)
		}
	}
}

func TestMathRound(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`let {round} = import @std/math; round(3.4)`, 3},
		{`let {round} = import @std/math; round(3.5)`, 4},
		{`let {round} = import @std/math; round(3.6)`, 4},
		{`let {round} = import @std/math; round(-2.4)`, -2},
		{`let {round} = import @std/math; round(-2.5)`, -3}, // Go rounds away from zero
		{`let {round} = import @std/math; round(-2.6)`, -3},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error: %s", result.Inspect())
		}

		if result.Type() != evaluator.INTEGER_OBJ {
			t.Errorf("expected INTEGER, got %s", result.Type())
			continue
		}

		if result.(*evaluator.Integer).Value != tt.expected {
			t.Errorf("expected %d, got %d for %s", tt.expected, result.(*evaluator.Integer).Value, tt.input)
		}
	}
}

func TestMathTrunc(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`let {trunc} = import @std/math; trunc(3.9)`, 3},
		{`let {trunc} = import @std/math; trunc(3.1)`, 3},
		{`let {trunc} = import @std/math; trunc(-2.9)`, -2},
		{`let {trunc} = import @std/math; trunc(-2.1)`, -2},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error: %s", result.Inspect())
		}

		if result.Type() != evaluator.INTEGER_OBJ {
			t.Errorf("expected INTEGER, got %s", result.Type())
			continue
		}

		if result.(*evaluator.Integer).Value != tt.expected {
			t.Errorf("expected %d, got %d", tt.expected, result.(*evaluator.Integer).Value)
		}
	}
}

// =============================================================================
// Comparison & Clamping Tests
// =============================================================================

func TestMathAbs(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{`let {abs} = import @std/math; abs(5)`, 5},
		{`let {abs} = import @std/math; abs(-5)`, 5},
		{`let {abs} = import @std/math; abs(3.14)`, 3.14},
		{`let {abs} = import @std/math; abs(-3.14)`, 3.14},
		{`let {abs} = import @std/math; abs(0)`, 0},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error: %s", result.Inspect())
		}

		var actual float64
		switch r := result.(type) {
		case *evaluator.Integer:
			actual = float64(r.Value)
		case *evaluator.Float:
			actual = r.Value
		default:
			t.Errorf("expected number, got %s", result.Type())
			continue
		}

		if math.Abs(actual-tt.expected) > 0.0001 {
			t.Errorf("expected %f, got %f", tt.expected, actual)
		}
	}
}

func TestMathSign(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`let {sign} = import @std/math; sign(5)`, 1},
		{`let {sign} = import @std/math; sign(-5)`, -1},
		{`let {sign} = import @std/math; sign(0)`, 0},
		{`let {sign} = import @std/math; sign(3.14)`, 1},
		{`let {sign} = import @std/math; sign(-3.14)`, -1},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error: %s", result.Inspect())
		}

		if result.Type() != evaluator.INTEGER_OBJ {
			t.Errorf("expected INTEGER, got %s", result.Type())
			continue
		}

		if result.(*evaluator.Integer).Value != tt.expected {
			t.Errorf("expected %d, got %d", tt.expected, result.(*evaluator.Integer).Value)
		}
	}
}

func TestMathClamp(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{`let {clamp} = import @std/math; clamp(5, 0, 10)`, 5},
		{`let {clamp} = import @std/math; clamp(-5, 0, 10)`, 0},
		{`let {clamp} = import @std/math; clamp(15, 0, 10)`, 10},
		{`let {clamp} = import @std/math; clamp(5.5, 0, 10)`, 5.5},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error: %s", result.Inspect())
		}

		var actual float64
		switch r := result.(type) {
		case *evaluator.Integer:
			actual = float64(r.Value)
		case *evaluator.Float:
			actual = r.Value
		default:
			t.Errorf("expected number, got %s", result.Type())
			continue
		}

		if math.Abs(actual-tt.expected) > 0.0001 {
			t.Errorf("expected %f, got %f", tt.expected, actual)
		}
	}
}

// =============================================================================
// Aggregation Functions Tests (2 args OR array)
// =============================================================================

func TestMathMin(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		// Two args mode
		{`let {min} = import @std/math; min(5, 3)`, 3},
		{`let {min} = import @std/math; min(3, 5)`, 3},
		{`let {min} = import @std/math; min(-3, 5)`, -3},
		{`let {min} = import @std/math; min(3.5, 3.2)`, 3.2},
		// Array mode
		{`let {min} = import @std/math; min([5, 3, 8, 1])`, 1},
		{`let {min} = import @std/math; min([3.5, 2.1, 4.9])`, 2.1},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error for %s: %s", tt.input, result.Inspect())
		}

		var actual float64
		switch r := result.(type) {
		case *evaluator.Integer:
			actual = float64(r.Value)
		case *evaluator.Float:
			actual = r.Value
		default:
			t.Errorf("expected number, got %s for %s", result.Type(), tt.input)
			continue
		}

		if math.Abs(actual-tt.expected) > 0.0001 {
			t.Errorf("expected %f, got %f for %s", tt.expected, actual, tt.input)
		}
	}
}

func TestMathMax(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		// Two args mode
		{`let {max} = import @std/math; max(5, 3)`, 5},
		{`let {max} = import @std/math; max(3, 5)`, 5},
		{`let {max} = import @std/math; max(-3, 5)`, 5},
		// Array mode
		{`let {max} = import @std/math; max([5, 3, 8, 1])`, 8},
		{`let {max} = import @std/math; max([3.5, 2.1, 4.9])`, 4.9},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error: %s", result.Inspect())
		}

		var actual float64
		switch r := result.(type) {
		case *evaluator.Integer:
			actual = float64(r.Value)
		case *evaluator.Float:
			actual = r.Value
		default:
			t.Errorf("expected number, got %s", result.Type())
			continue
		}

		if math.Abs(actual-tt.expected) > 0.0001 {
			t.Errorf("expected %f, got %f", tt.expected, actual)
		}
	}
}

func TestMathSum(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		// Two args mode
		{`let {sum} = import @std/math; sum(5, 3)`, 8},
		{`let {sum} = import @std/math; sum(3.5, 2.5)`, 6},
		// Array mode
		{`let {sum} = import @std/math; sum([1, 2, 3, 4])`, 10},
		{`let {sum} = import @std/math; sum([1.5, 2.5])`, 4},
		{`let {sum} = import @std/math; sum([])`, 0},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error for %s: %s", tt.input, result.Inspect())
		}

		var actual float64
		switch r := result.(type) {
		case *evaluator.Integer:
			actual = float64(r.Value)
		case *evaluator.Float:
			actual = r.Value
		default:
			t.Errorf("expected number, got %s for %s", result.Type(), tt.input)
			continue
		}

		if math.Abs(actual-tt.expected) > 0.0001 {
			t.Errorf("expected %f, got %f for %s", tt.expected, actual, tt.input)
		}
	}
}

func TestMathAvg(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		// Two args mode
		{`let {avg} = import @std/math; avg(4, 6)`, 5},
		{`let {avg} = import @std/math; avg(3.0, 5.0)`, 4},
		// Array mode
		{`let {avg} = import @std/math; avg([1, 2, 3, 4])`, 2.5},
		{`let {avg} = import @std/math; avg([10, 20])`, 15},
		// mean alias
		{`let {mean} = import @std/math; mean([1, 2, 3, 4])`, 2.5},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error for %s: %s", tt.input, result.Inspect())
		}

		f := result.(*evaluator.Float).Value
		if math.Abs(f-tt.expected) > 0.0001 {
			t.Errorf("expected %f, got %f for %s", tt.expected, f, tt.input)
		}
	}
}

func TestMathProduct(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		// Two args mode
		{`let {product} = import @std/math; product(4, 6)`, 24},
		{`let {product} = import @std/math; product(3.0, 2.0)`, 6},
		// Array mode
		{`let {product} = import @std/math; product([1, 2, 3, 4])`, 24},
		{`let {product} = import @std/math; product([2.5, 2])`, 5},
		{`let {product} = import @std/math; product([])`, 1},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error for %s: %s", tt.input, result.Inspect())
		}

		var actual float64
		switch r := result.(type) {
		case *evaluator.Integer:
			actual = float64(r.Value)
		case *evaluator.Float:
			actual = r.Value
		default:
			t.Errorf("expected number, got %s for %s", result.Type(), tt.input)
			continue
		}

		if math.Abs(actual-tt.expected) > 0.0001 {
			t.Errorf("expected %f, got %f for %s", tt.expected, actual, tt.input)
		}
	}
}

func TestMathCount(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`let {count} = import @std/math; count([1, 2, 3, 4])`, 4},
		{`let {count} = import @std/math; count([])`, 0},
		{`let {count} = import @std/math; count(["a", "b"])`, 2},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error: %s", result.Inspect())
		}

		if result.Type() != evaluator.INTEGER_OBJ {
			t.Errorf("expected INTEGER, got %s", result.Type())
			continue
		}

		if result.(*evaluator.Integer).Value != tt.expected {
			t.Errorf("expected %d, got %d", tt.expected, result.(*evaluator.Integer).Value)
		}
	}
}

// =============================================================================
// Statistics Functions Tests
// =============================================================================

func TestMathMedian(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{`let {median} = import @std/math; median([1, 2, 3])`, 2},
		{`let {median} = import @std/math; median([1, 2, 3, 4])`, 2.5},
		{`let {median} = import @std/math; median([5, 1, 3])`, 3},
		{`let {median} = import @std/math; median([1])`, 1},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error for %s: %s", tt.input, result.Inspect())
		}

		var actual float64
		switch r := result.(type) {
		case *evaluator.Integer:
			actual = float64(r.Value)
		case *evaluator.Float:
			actual = r.Value
		default:
			t.Errorf("expected number, got %s for %s", result.Type(), tt.input)
			continue
		}

		if math.Abs(actual-tt.expected) > 0.0001 {
			t.Errorf("expected %f, got %f for %s", tt.expected, actual, tt.input)
		}
	}
}

func TestMathMode(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{`let {mode} = import @std/math; mode([1, 2, 2, 3])`, 2},
		{`let {mode} = import @std/math; mode([1, 1, 2, 2, 3])`, 1}, // ties go to smallest
		{`let {mode} = import @std/math; mode([5])`, 5},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error for %s: %s", tt.input, result.Inspect())
		}

		var actual float64
		switch r := result.(type) {
		case *evaluator.Integer:
			actual = float64(r.Value)
		case *evaluator.Float:
			actual = r.Value
		default:
			t.Errorf("expected number, got %s for %s", result.Type(), tt.input)
			continue
		}

		if math.Abs(actual-tt.expected) > 0.0001 {
			t.Errorf("expected %f, got %f for %s", tt.expected, actual, tt.input)
		}
	}
}

func TestMathStddev(t *testing.T) {
	// stddev of [2, 4, 4, 4, 5, 5, 7, 9] = 2.0
	input := `let {stddev} = import @std/math; stddev([2, 4, 4, 4, 5, 5, 7, 9])`
	result := evalMathTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	f := result.(*evaluator.Float).Value
	if math.Abs(f-2.0) > 0.0001 {
		t.Errorf("expected stddev ~2.0, got %f", f)
	}
}

func TestMathVariance(t *testing.T) {
	// variance of [2, 4, 4, 4, 5, 5, 7, 9] = 4.0
	input := `let {variance} = import @std/math; variance([2, 4, 4, 4, 5, 5, 7, 9])`
	result := evalMathTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	f := result.(*evaluator.Float).Value
	if math.Abs(f-4.0) > 0.0001 {
		t.Errorf("expected variance ~4.0, got %f", f)
	}
}

func TestMathRange(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{`let {range} = import @std/math; range([1, 5, 3, 10, 2])`, 9},
		{`let {range} = import @std/math; range([5])`, 0},
		{`let {range} = import @std/math; range([1.5, 3.5])`, 2},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error for %s: %s", tt.input, result.Inspect())
		}

		var actual float64
		switch r := result.(type) {
		case *evaluator.Integer:
			actual = float64(r.Value)
		case *evaluator.Float:
			actual = r.Value
		default:
			t.Errorf("expected number, got %s for %s", result.Type(), tt.input)
			continue
		}

		if math.Abs(actual-tt.expected) > 0.0001 {
			t.Errorf("expected %f, got %f for %s", tt.expected, actual, tt.input)
		}
	}
}

// =============================================================================
// Random Functions Tests
// =============================================================================

func TestMathRandomBasic(t *testing.T) {
	// Just test that random() returns a float between 0 and 1
	input := `let {random} = import @std/math; random()`
	result := evalMathTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.FLOAT_OBJ {
		t.Fatalf("expected FLOAT, got %s", result.Type())
	}

	f := result.(*evaluator.Float).Value
	if f < 0 || f >= 1 {
		t.Errorf("random() should return [0, 1), got %f", f)
	}
}

func TestMathRandomWithMax(t *testing.T) {
	input := `let {random} = import @std/math; random(10)`
	result := evalMathTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	f := result.(*evaluator.Float).Value
	if f < 0 || f >= 10 {
		t.Errorf("random(10) should return [0, 10), got %f", f)
	}
}

func TestMathRandomWithRange(t *testing.T) {
	input := `let {random} = import @std/math; random(5, 10)`
	result := evalMathTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	f := result.(*evaluator.Float).Value
	if f < 5 || f >= 10 {
		t.Errorf("random(5, 10) should return [5, 10), got %f", f)
	}
}

func TestMathRandomInt(t *testing.T) {
	input := `let {randomInt} = import @std/math; randomInt(10)`
	result := evalMathTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.INTEGER_OBJ {
		t.Fatalf("expected INTEGER, got %s", result.Type())
	}

	i := result.(*evaluator.Integer).Value
	if i < 0 || i > 10 {
		t.Errorf("randomInt(10) should return [0, 10], got %d", i)
	}
}

func TestMathRandomIntWithRange(t *testing.T) {
	input := `let {randomInt} = import @std/math; randomInt(5, 10)`
	result := evalMathTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.INTEGER_OBJ {
		t.Fatalf("expected INTEGER, got %s", result.Type())
	}

	i := result.(*evaluator.Integer).Value
	if i < 5 || i > 10 {
		t.Errorf("randomInt(5, 10) should return [5, 10], got %d", i)
	}
}

func TestMathSeed(t *testing.T) {
	// Test that seeding produces reproducible results
	input := `let {seed, random} = import @std/math
seed(42)
a = random()
seed(42)
b = random()
a == b`

	result := evalMathTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.BOOLEAN_OBJ {
		t.Fatalf("expected BOOLEAN, got %s", result.Type())
	}

	if !result.(*evaluator.Boolean).Value {
		t.Error("seeded random should produce reproducible results")
	}
}

// =============================================================================
// Powers & Logarithms Tests
// =============================================================================

func TestMathSqrt(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{`let {sqrt} = import @std/math; sqrt(16)`, 4},
		{`let {sqrt} = import @std/math; sqrt(2)`, math.Sqrt(2)},
		{`let {sqrt} = import @std/math; sqrt(0)`, 0},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error: %s", result.Inspect())
		}

		var actual float64
		switch r := result.(type) {
		case *evaluator.Integer:
			actual = float64(r.Value)
		case *evaluator.Float:
			actual = r.Value
		default:
			t.Errorf("expected number, got %s", result.Type())
			continue
		}

		if math.Abs(actual-tt.expected) > 0.0001 {
			t.Errorf("expected %f, got %f", tt.expected, actual)
		}
	}
}

func TestMathSqrtNegative(t *testing.T) {
	input := `let {sqrt} = import @std/math; sqrt(-1)`
	result := evalMathTest(t, input)

	if result.Type() != evaluator.ERROR_OBJ {
		t.Fatalf("expected error for sqrt(-1), got %s", result.Type())
	}

	errMsg := result.(*evaluator.Error).Message
	if !strings.Contains(errMsg, "negative") {
		t.Errorf("error should mention negative, got: %s", errMsg)
	}
}

func TestMathPow(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{`let {pow} = import @std/math; pow(2, 3)`, 8},
		{`let {pow} = import @std/math; pow(2, 0)`, 1},
		{`let {pow} = import @std/math; pow(2, -1)`, 0.5},
		{`let {pow} = import @std/math; pow(9, 0.5)`, 3},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error: %s", result.Inspect())
		}

		var actual float64
		switch r := result.(type) {
		case *evaluator.Integer:
			actual = float64(r.Value)
		case *evaluator.Float:
			actual = r.Value
		default:
			t.Errorf("expected number, got %s", result.Type())
			continue
		}

		if math.Abs(actual-tt.expected) > 0.0001 {
			t.Errorf("expected %f, got %f for %s", tt.expected, actual, tt.input)
		}
	}
}

func TestMathExp(t *testing.T) {
	input := `let {exp} = import @std/math; exp(1)`
	result := evalMathTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	f := result.(*evaluator.Float).Value
	if math.Abs(f-math.E) > 0.0001 {
		t.Errorf("expected e (~2.718), got %f", f)
	}
}

func TestMathLog(t *testing.T) {
	// Use module.function access pattern to avoid conflict with builtin 'log' function
	input := `let math = import @std/math; math.log(2.718281828459045)`
	result := evalMathTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.FLOAT_OBJ {
		t.Fatalf("expected FLOAT, got %s: %s", result.Type(), result.Inspect())
	}

	f := result.(*evaluator.Float).Value
	if math.Abs(f-1) > 0.0001 {
		t.Errorf("expected log(e) = 1, got %f", f)
	}
}

func TestMathLogNonPositive(t *testing.T) {
	// Use math.log() to avoid conflict with builtin log function
	tests := []string{
		`let math = import @std/math; math.log(0)`,
		`let math = import @std/math; math.log(-1)`,
	}

	for _, input := range tests {
		result := evalMathTest(t, input)

		if result.Type() != evaluator.ERROR_OBJ {
			t.Errorf("expected error for %s, got %s", input, result.Type())
		}
	}
}

func TestMathLog10(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{`let {log10} = import @std/math; log10(10)`, 1},
		{`let {log10} = import @std/math; log10(100)`, 2},
		{`let {log10} = import @std/math; log10(1)`, 0},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error: %s", result.Inspect())
		}

		f := result.(*evaluator.Float).Value
		if math.Abs(f-tt.expected) > 0.0001 {
			t.Errorf("expected %f, got %f", tt.expected, f)
		}
	}
}

// =============================================================================
// Trigonometry Tests
// =============================================================================

func TestMathTrig(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{"sin(0)", `let {sin} = import @std/math; sin(0)`, 0},
		{"sin(PI/2)", `let {sin, PI} = import @std/math; sin(PI/2)`, 1},
		{"cos(0)", `let {cos} = import @std/math; cos(0)`, 1},
		{"cos(PI)", `let {cos, PI} = import @std/math; cos(PI)`, -1},
		{"tan(0)", `let {tan} = import @std/math; tan(0)`, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalMathTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			var actual float64
			switch r := result.(type) {
			case *evaluator.Integer:
				actual = float64(r.Value)
			case *evaluator.Float:
				actual = r.Value
			default:
				t.Fatalf("expected number, got %s", result.Type())
			}

			if math.Abs(actual-tt.expected) > 0.0001 {
				t.Errorf("expected %f, got %f", tt.expected, actual)
			}
		})
	}
}

func TestMathInverseTrig(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{"asin(0)", `let {asin} = import @std/math; asin(0)`, 0},
		{"asin(1)", `let {asin} = import @std/math; asin(1)`, math.Pi / 2},
		{"acos(1)", `let {acos} = import @std/math; acos(1)`, 0},
		{"acos(0)", `let {acos} = import @std/math; acos(0)`, math.Pi / 2},
		{"atan(0)", `let {atan} = import @std/math; atan(0)`, 0},
		{"atan(1)", `let {atan} = import @std/math; atan(1)`, math.Pi / 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalMathTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			f := result.(*evaluator.Float).Value
			if math.Abs(f-tt.expected) > 0.0001 {
				t.Errorf("expected %f, got %f", tt.expected, f)
			}
		})
	}
}

func TestMathAsinAcosOutOfRange(t *testing.T) {
	tests := []string{
		`let {asin} = import @std/math; asin(2)`,
		`let {asin} = import @std/math; asin(-2)`,
		`let {acos} = import @std/math; acos(2)`,
		`let {acos} = import @std/math; acos(-2)`,
	}

	for _, input := range tests {
		result := evalMathTest(t, input)

		if result.Type() != evaluator.ERROR_OBJ {
			t.Errorf("expected error for %s, got %s", input, result.Type())
		}
	}
}

func TestMathAtan2(t *testing.T) {
	input := `let {atan2, PI} = import @std/math; atan2(1, 1)`
	result := evalMathTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	f := result.(*evaluator.Float).Value
	expected := math.Pi / 4
	if math.Abs(f-expected) > 0.0001 {
		t.Errorf("expected %f, got %f", expected, f)
	}
}

// =============================================================================
// Angular Conversion Tests
// =============================================================================

func TestMathDegrees(t *testing.T) {
	input := `let {degrees, PI} = import @std/math; degrees(PI)`
	result := evalMathTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	var actual float64
	switch r := result.(type) {
	case *evaluator.Integer:
		actual = float64(r.Value)
	case *evaluator.Float:
		actual = r.Value
	}

	if math.Abs(actual-180) > 0.0001 {
		t.Errorf("expected 180, got %f", actual)
	}
}

func TestMathRadians(t *testing.T) {
	input := `let {radians, PI} = import @std/math; radians(180)`
	result := evalMathTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	f := result.(*evaluator.Float).Value
	if math.Abs(f-math.Pi) > 0.0001 {
		t.Errorf("expected PI, got %f", f)
	}
}

// =============================================================================
// Geometry & Interpolation Tests
// =============================================================================

func TestMathHypot(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{`let {hypot} = import @std/math; hypot(3, 4)`, 5},
		{`let {hypot} = import @std/math; hypot(5, 12)`, 13},
		{`let {hypot} = import @std/math; hypot(0, 5)`, 5},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error: %s", result.Inspect())
		}

		var actual float64
		switch r := result.(type) {
		case *evaluator.Integer:
			actual = float64(r.Value)
		case *evaluator.Float:
			actual = r.Value
		}

		if math.Abs(actual-tt.expected) > 0.0001 {
			t.Errorf("expected %f, got %f for %s", tt.expected, actual, tt.input)
		}
	}
}

func TestMathDist(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{`let {dist} = import @std/math; dist(0, 0, 3, 4)`, 5},
		{`let {dist} = import @std/math; dist(1, 1, 4, 5)`, 5},
		{`let {dist} = import @std/math; dist(0, 0, 0, 0)`, 0},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error: %s", result.Inspect())
		}

		var actual float64
		switch r := result.(type) {
		case *evaluator.Integer:
			actual = float64(r.Value)
		case *evaluator.Float:
			actual = r.Value
		}

		if math.Abs(actual-tt.expected) > 0.0001 {
			t.Errorf("expected %f, got %f for %s", tt.expected, actual, tt.input)
		}
	}
}

func TestMathLerp(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{`let {lerp} = import @std/math; lerp(0, 10, 0)`, 0},
		{`let {lerp} = import @std/math; lerp(0, 10, 1)`, 10},
		{`let {lerp} = import @std/math; lerp(0, 10, 0.5)`, 5},
		{`let {lerp} = import @std/math; lerp(0, 10, 0.25)`, 2.5},
		{`let {lerp} = import @std/math; lerp(10, 20, 0.5)`, 15},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error: %s", result.Inspect())
		}

		var actual float64
		switch r := result.(type) {
		case *evaluator.Integer:
			actual = float64(r.Value)
		case *evaluator.Float:
			actual = r.Value
		}

		if math.Abs(actual-tt.expected) > 0.0001 {
			t.Errorf("expected %f, got %f for %s", tt.expected, actual, tt.input)
		}
	}
}

func TestMathMap(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		// map(value, inMin, inMax, outMin, outMax)
		{`let {map} = import @std/math; map(5, 0, 10, 0, 100)`, 50},
		{`let {map} = import @std/math; map(0, 0, 10, 0, 100)`, 0},
		{`let {map} = import @std/math; map(10, 0, 10, 0, 100)`, 100},
		{`let {map} = import @std/math; map(25, 0, 100, 0, 1)`, 0.25},
		// Temperature conversion example: 32°F = 0°C, 212°F = 100°C
		{`let {map} = import @std/math; map(32, 32, 212, 0, 100)`, 0},
		{`let {map} = import @std/math; map(212, 32, 212, 0, 100)`, 100},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error for %s: %s", tt.input, result.Inspect())
		}

		var actual float64
		switch r := result.(type) {
		case *evaluator.Integer:
			actual = float64(r.Value)
		case *evaluator.Float:
			actual = r.Value
		}

		if math.Abs(actual-tt.expected) > 0.0001 {
			t.Errorf("expected %f, got %f for %s", tt.expected, actual, tt.input)
		}
	}
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestMathArityErrors(t *testing.T) {
	tests := []struct {
		input       string
		errContains string
	}{
		{`let {floor} = import @std/math; floor()`, "argument"},
		{`let {floor} = import @std/math; floor(1, 2)`, "argument"},
		{`let {pow} = import @std/math; pow(2)`, "argument"},
		{`let {clamp} = import @std/math; clamp(1, 2)`, "argument"},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() != evaluator.ERROR_OBJ {
			t.Errorf("expected error for %s, got %s", tt.input, result.Type())
			continue
		}

		errMsg := result.(*evaluator.Error).Message
		if !strings.Contains(strings.ToLower(errMsg), tt.errContains) {
			t.Errorf("error should contain %q for %s, got: %s", tt.errContains, tt.input, errMsg)
		}
	}
}

func TestMathTypeErrors(t *testing.T) {
	tests := []struct {
		input       string
		errContains string
	}{
		{`let {floor} = import @std/math; floor("hello")`, "number"},
		{`let {pow} = import @std/math; pow("a", 2)`, "number"},
		{`let {min} = import @std/math; min("a", "b")`, "number"},
	}

	for _, tt := range tests {
		result := evalMathTest(t, tt.input)

		if result.Type() != evaluator.ERROR_OBJ {
			t.Errorf("expected error for %s, got %s", tt.input, result.Type())
			continue
		}

		errMsg := result.(*evaluator.Error).Message
		if !strings.Contains(strings.ToLower(errMsg), tt.errContains) {
			t.Errorf("error should contain %q for %s, got: %s", tt.errContains, tt.input, errMsg)
		}
	}
}

func TestMathEmptyArrayErrors(t *testing.T) {
	// Functions that require non-empty arrays
	tests := []string{
		`let {min} = import @std/math; min([])`,
		`let {max} = import @std/math; max([])`,
		`let {avg} = import @std/math; avg([])`,
		`let {median} = import @std/math; median([])`,
		`let {mode} = import @std/math; mode([])`,
		`let {stddev} = import @std/math; stddev([])`,
		`let {variance} = import @std/math; variance([])`,
		`let {range} = import @std/math; range([])`,
	}

	for _, input := range tests {
		result := evalMathTest(t, input)

		if result.Type() != evaluator.ERROR_OBJ {
			t.Errorf("expected error for %s, got %s", input, result.Type())
		}
	}
}
