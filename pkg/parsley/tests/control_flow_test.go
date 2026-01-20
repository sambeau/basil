package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// testEvalControlFlow evaluates Parsley code for control flow tests
func testEvalControlFlow(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

// TestStopInForLoop tests that stop exits a for loop early with accumulated results
func TestStopInForLoop(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "stop on condition",
			input:    `for(x in [1, 2, 3, 4, 5]) { if(x == 3) { stop }; x }`,
			expected: "[1, 2]",
		},
		{
			name:     "stop immediately",
			input:    `for(x in [1, 2, 3]) { stop; x }`,
			expected: "[]",
		},
		{
			name:     "stop after one item",
			input:    `for(x in [10, 20, 30]) { x; stop }`,
			expected: "[10]",
		},
		{
			name:     "stop with nested if",
			input:    `for(x in [1, 2, 3, 4]) { if(x > 2) { stop }; x * 10 }`,
			expected: "[10, 20]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalControlFlow(tt.input)
			if result == nil {
				t.Fatalf("result is nil")
			}
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("got error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// TestSkipInForLoop tests that skip skips the current iteration
func TestSkipInForLoop(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "skip on condition",
			input:    `for(x in [1, 2, 3, 4, 5]) { if(x == 3) { skip }; x }`,
			expected: "[1, 2, 4, 5]",
		},
		{
			name:     "skip all even numbers",
			input:    `for(x in [1, 2, 3, 4, 5, 6]) { if(x % 2 == 0) { skip }; x }`,
			expected: "[1, 3, 5]",
		},
		{
			name:     "skip after value keeps the value",
			input:    `for(x in [1, 2, 3]) { let v = x; skip; v }`,
			expected: "[]",
		},
		{
			name:     "skip immediately",
			input:    `for(x in [1, 2, 3]) { skip; x }`,
			expected: "[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalControlFlow(tt.input)
			if result == nil {
				t.Fatalf("result is nil")
			}
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("got error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// TestCheckStatement tests the check statement for precondition validation
func TestCheckStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "check passes",
			input:    `if(true) { check true else "error"; "success" }`,
			expected: "success",
		},
		{
			name:     "check fails",
			input:    `if(true) { check false else "failed"; "success" }`,
			expected: "failed",
		},
		{
			name:     "check with expression",
			input:    `let x = 5; if(true) { check x > 0 else "negative"; x * 2 }`,
			expected: "10",
		},
		{
			name:     "check fails with expression",
			input:    `let x = -5; if(true) { check x > 0 else "negative"; x * 2 }`,
			expected: "negative",
		},
		{
			name:     "multiple checks first fails",
			input:    `if(true) { check false else "first"; check true else "second"; "done" }`,
			expected: "first",
		},
		{
			name:     "multiple checks second fails",
			input:    `if(true) { check true else "first"; check false else "second"; "done" }`,
			expected: "second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalControlFlow(tt.input)
			if result == nil {
				t.Fatalf("result is nil")
			}
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("got error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// TestCheckInForLoop tests check statement inside for loops
func TestCheckInForLoop(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "check as filter",
			input:    `for(x in [1, 2, 3, 4, 5]) { check x % 2 == 1 else null; x }`,
			expected: "[1, 3, 5]",
		},
		{
			name:     "check with value",
			input:    `for(x in [1, -2, 3, -4]) { check x > 0 else 0; x * 10 }`,
			expected: "[10, 0, 30, 0]",
		},
		{
			name:     "check returns early value",
			input:    `for(x in [1, 2, 3]) { check x != 2 else "skipped"; x }`,
			expected: `[1, skipped, 3]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalControlFlow(tt.input)
			if result == nil {
				t.Fatalf("result is nil")
			}
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("got error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// TestStopSkipOutsideLoop tests that stop/skip produce errors outside for loops
func TestStopSkipOutsideLoop(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "stop outside loop",
			input:         `if(true) { stop }`,
			expectedError: "stop",
		},
		{
			name:          "skip outside loop",
			input:         `if(true) { skip }`,
			expectedError: "skip",
		},
		{
			name:          "stop in function",
			input:         `let f = fn() { stop }; f()`,
			expectedError: "stop",
		},
		{
			name:          "skip in function",
			input:         `let f = fn() { skip }; f()`,
			expectedError: "skip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalControlFlow(tt.input)
			if result == nil {
				t.Fatalf("result is nil")
			}
			if result.Type() != evaluator.ERROR_OBJ {
				t.Fatalf("expected error, got %s", result.Inspect())
			}
			errMsg := result.Inspect()
			if !strings.Contains(strings.ToLower(errMsg), strings.ToLower(tt.expectedError)) {
				t.Errorf("expected error to contain %q, got %q", tt.expectedError, errMsg)
			}
		})
	}
}

// TestCheckInFunction tests check statement in functions (acts like early return)
func TestCheckInFunction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "check fails returns else value",
			input:    `let validate = fn(x) { check x > 0 else "invalid"; x * 2 }; validate(-5)`,
			expected: "invalid",
		},
		{
			name:     "check passes continues",
			input:    `let validate = fn(x) { check x > 0 else "invalid"; x * 2 }; validate(5)`,
			expected: "10",
		},
		{
			name:     "multiple checks",
			input:    `let f = fn(x) { check x > 0 else "not positive"; check x < 10 else "too big"; x }; f(5)`,
			expected: "5",
		},
		{
			name:     "first check fails",
			input:    `let f = fn(x) { check x > 0 else "not positive"; check x < 10 else "too big"; x }; f(-5)`,
			expected: "not positive",
		},
		{
			name:     "second check fails",
			input:    `let f = fn(x) { check x > 0 else "not positive"; check x < 10 else "too big"; x }; f(15)`,
			expected: "too big",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalControlFlow(tt.input)
			if result == nil {
				t.Fatalf("result is nil")
			}
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("got error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// TestStopSkipWithDictIteration tests stop/skip with dictionary iteration
func TestStopSkipWithDictIteration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "stop in dict iteration",
			input:    `for(k, v in {a: 1, b: 2, c: 3}) { if(k == "b") { stop }; v }`,
			expected: "[1]",
		},
		{
			name:     "skip in dict iteration",
			input:    `for(k, v in {a: 1, b: 2, c: 3}) { if(k == "b") { skip }; v }`,
			expected: "[1, 3]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalControlFlow(tt.input)
			if result == nil {
				t.Fatalf("result is nil")
			}
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("got error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// TestBracelessStopSkip tests stop/skip without braces: if (cond) stop
func TestBracelessStopSkip(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "braceless stop",
			input:    `for(x in [1, 2, 3, 4, 5]) { if (x > 3) stop; x }`,
			expected: "[1, 2, 3]",
		},
		{
			name:     "braceless skip",
			input:    `for(x in [1, 2, 3, 4, 5]) { if (x == 3) skip; x }`,
			expected: "[1, 2, 4, 5]",
		},
		{
			name:     "braceless stop and skip together",
			input:    `for(x in [1, 2, 3, 4, 5, 6, 7, 8]) { if (x > 5) stop; if (x == 3) skip; x }`,
			expected: "[1, 2, 4, 5]",
		},
		{
			name:     "braceless with else clause",
			input:    `for(x in [1, 2, 3, 4]) { if (x > 2) stop else x * 10 }`,
			expected: "[10, 20]",
		},
		{
			name:     "braceless skip with else",
			input:    `for(x in [1, 2, 3, 4]) { if (x == 2) skip else x * 10 }`,
			expected: "[10, 30, 40]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalControlFlow(tt.input)
			if result == nil {
				t.Fatalf("result is nil")
			}
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("got error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// TestCheckExitPropagation tests that CheckExit signals propagate correctly
// and don't get stored as variable values (regression test for wrong line number bug)
func TestCheckExitPropagation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "check exit propagates through let",
			input:    `let test = fn() { let result = if(true) { check false else "early"; "late" }; result }; test()`,
			expected: "early",
		},
		{
			name:     "check exit propagates through assignment",
			input:    `let test = fn() { let result = ""; result = if(true) { check false else "early"; "late" }; result }; test()`,
			expected: "early",
		},
		{
			name:     "check exit propagates through nested blocks",
			input:    `let test = fn() { let x = if(true) { let y = if(true) { check false else "inner"; "never" }; y }; x }; test()`,
			expected: "inner",
		},
		{
			name:     "check exit propagates through return",
			input:    `let test = fn() { return if(true) { check false else "returned"; "not returned" } }; test()`,
			expected: "returned",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalControlFlow(tt.input)
			if result == nil {
				t.Fatalf("result is nil")
			}
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("got error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}
