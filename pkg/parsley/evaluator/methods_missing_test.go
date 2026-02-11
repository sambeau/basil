package evaluator

import (
	"math"
	"testing"
)

// Tests for the 6 methods that were missing and have now been implemented

func TestIntegerAbs(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"(42).abs()", 42},
		{"(-42).abs()", 42},
		{"(0).abs()", 0},
		{"(-1).abs()", 1},
		{"(1).abs()", 1},
	}

	for _, tt := range tests {
		result := testEval(tt.input)
		intObj, ok := result.(*Integer)
		if !ok {
			t.Errorf("Expected Integer for %q, got %T", tt.input, result)
			continue
		}
		if intObj.Value != tt.expected {
			t.Errorf("For %q: expected %d, got %d", tt.input, tt.expected, intObj.Value)
		}
	}
}

func TestFloatAbs(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"(3.14).abs()", 3.14},
		{"(-3.14).abs()", 3.14},
		{"(0.0).abs()", 0.0},
		{"(-0.5).abs()", 0.5},
		{"(100.25).abs()", 100.25},
		{"(-100.25).abs()", 100.25},
	}

	for _, tt := range tests {
		result := testEval(tt.input)
		floatObj, ok := result.(*Float)
		if !ok {
			t.Errorf("Expected Float for %q, got %T", tt.input, result)
			continue
		}
		if floatObj.Value != tt.expected {
			t.Errorf("For %q: expected %f, got %f", tt.input, tt.expected, floatObj.Value)
		}
	}
}

func TestFloatRound(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"(3.14159).round()", 3.0},
		{"(3.14159).round(2)", 3.14},
		{"(3.14159).round(4)", 3.1416},
		{"(3.5).round()", 4.0},
		{"(2.5).round()", 3.0}, // Go's math.Round uses round-half-away-from-zero
		{"(-3.7).round()", -4.0},
		{"(123.456).round(1)", 123.5},
		{"(0.0).round()", 0.0},
	}

	for _, tt := range tests {
		result := testEval(tt.input)
		floatObj, ok := result.(*Float)
		if !ok {
			t.Errorf("Expected Float for %q, got %T", tt.input, result)
			continue
		}
		if math.Abs(floatObj.Value-tt.expected) > 0.0001 {
			t.Errorf("For %q: expected %f, got %f", tt.input, tt.expected, floatObj.Value)
		}
	}
}

func TestFloatFloor(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"(3.14).floor()", 3.0},
		{"(3.99).floor()", 3.0},
		{"(-3.14).floor()", -4.0},
		{"(-3.99).floor()", -4.0},
		{"(0.0).floor()", 0.0},
		{"(5.0).floor()", 5.0},
	}

	for _, tt := range tests {
		result := testEval(tt.input)
		floatObj, ok := result.(*Float)
		if !ok {
			t.Errorf("Expected Float for %q, got %T", tt.input, result)
			continue
		}
		if floatObj.Value != tt.expected {
			t.Errorf("For %q: expected %f, got %f", tt.input, tt.expected, floatObj.Value)
		}
	}
}

func TestFloatCeil(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"(3.14).ceil()", 4.0},
		{"(3.99).ceil()", 4.0},
		{"(-3.14).ceil()", -3.0},
		{"(-3.99).ceil()", -3.0},
		{"(0.0).ceil()", 0.0},
		{"(5.0).ceil()", 5.0},
	}

	for _, tt := range tests {
		result := testEval(tt.input)
		floatObj, ok := result.(*Float)
		if !ok {
			t.Errorf("Expected Float for %q, got %T", tt.input, result)
			continue
		}
		if floatObj.Value != tt.expected {
			t.Errorf("For %q: expected %f, got %f", tt.input, tt.expected, floatObj.Value)
		}
	}
}

func TestMoneyNegate(t *testing.T) {
	tests := []struct {
		input          string
		expectedAmount int64
		expectedCurr   string
	}{
		{"($50.00).negate()", -5000, "USD"},
		{"($-50.00).negate()", 5000, "USD"},
		{"(€100.00).negate()", -10000, "EUR"},
		{"(£0.00).negate()", 0, "GBP"},
	}

	for _, tt := range tests {
		result := testEval(tt.input)
		moneyObj, ok := result.(*Money)
		if !ok {
			t.Errorf("Expected Money for %q, got %T", tt.input, result)
			continue
		}
		if moneyObj.Amount != tt.expectedAmount {
			t.Errorf("For %q: expected amount %d, got %d", tt.input, tt.expectedAmount, moneyObj.Amount)
		}
		if moneyObj.Currency != tt.expectedCurr {
			t.Errorf("For %q: expected currency %s, got %s", tt.input, tt.expectedCurr, moneyObj.Currency)
		}
	}
}

// Test error cases for the new methods

func TestIntegerAbsArity(t *testing.T) {
	result := testEval("(42).abs(1)")
	errObj, ok := result.(*Error)
	if !ok {
		t.Errorf("Expected Error, got %T", result)
		return
	}
	if errObj.Code != "ARITY-0001" {
		t.Errorf("Expected arity error code ARITY-0001, got %s", errObj.Code)
	}
}

func TestFloatRoundArityAndType(t *testing.T) {
	// Too many arguments
	result := testEval("(3.14).round(2, 3)")
	_, ok := result.(*Error)
	if !ok {
		t.Errorf("Expected Error for too many args, got %T", result)
	}

	// Wrong type
	result = testEval(`(3.14).round("2")`)
	_, ok = result.(*Error)
	if !ok {
		t.Errorf("Expected Error for wrong type, got %T", result)
	}
}

func TestFloatFloorArity(t *testing.T) {
	result := testEval("(3.14).floor(1)")
	errObj, ok := result.(*Error)
	if !ok {
		t.Errorf("Expected Error, got %T", result)
		return
	}
	if errObj.Code != "ARITY-0001" {
		t.Errorf("Expected arity error code ARITY-0001, got %s", errObj.Code)
	}
}

func TestFloatCeilArity(t *testing.T) {
	result := testEval("(3.14).ceil(1)")
	errObj, ok := result.(*Error)
	if !ok {
		t.Errorf("Expected Error, got %T", result)
		return
	}
	if errObj.Code != "ARITY-0001" {
		t.Errorf("Expected arity error code ARITY-0001, got %s", errObj.Code)
	}
}

func TestMoneyNegateArity(t *testing.T) {
	result := testEval("($50.00).negate(1)")
	errObj, ok := result.(*Error)
	if !ok {
		t.Errorf("Expected Error, got %T", result)
		return
	}
	if errObj.Code != "ARITY-0001" {
		t.Errorf("Expected arity error code ARITY-0001, got %s", errObj.Code)
	}
}

// Test method chaining with new methods

func TestMethodChaining(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"(-42).abs().toJSON()", "42"},
		{"(3.14159).round(2).toJSON()", "3.14"},
		{"(3.7).floor().toJSON()", "3"},
		{"(3.2).ceil().toJSON()", "4"},
		{"($50.00).negate().format()", "$ -50.00"},
	}

	for _, tt := range tests {
		result := testEval(tt.input)
		strObj, ok := result.(*String)
		if !ok {
			t.Errorf("Expected String for %q, got %T", tt.input, result)
			continue
		}
		if strObj.Value != tt.expected {
			t.Errorf("For %q: expected %q, got %q", tt.input, tt.expected, strObj.Value)
		}
	}
}
