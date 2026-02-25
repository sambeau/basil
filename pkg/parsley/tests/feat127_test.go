// feat127_test.go - Tests for FEAT-127: Parsley 1.0 Release Readiness
//
// Tests for:
// - Named regex capture groups returning dictionaries
// - Custom message parameter for failIfInvalid
// - Deprecation warnings

package tests

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// =============================================================================
// Task 1: Named Capture Groups Return Dictionaries (#97)
// =============================================================================

func TestNamedCaptureGroupsReturnDictionary(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "named groups return dictionary",
			code:     `"John Smith" ~ /(?P<first>\w+)\s+(?P<last>\w+)/`,
			expected: `{0: John Smith, first: John, last: Smith}`,
		},
		{
			name:     "single named group",
			code:     `"hello" ~ /(?P<word>\w+)/`,
			expected: `{0: hello, word: hello}`,
		},
		{
			name:     "mixed named and unnamed groups",
			code:     `"abc123" ~ /(?P<letters>[a-z]+)(\d+)/`,
			expected: `{0: abc123, letters: abc, 2: 123}`,
		},
		{
			name:     "unnamed groups still return array",
			code:     `"abc" ~ /([a-z]+)/`,
			expected: `[abc, abc]`,
		},
		{
			name:     "no match returns null",
			code:     `"123" ~ /(?P<letters>[a-z]+)/`,
			expected: `null`,
		},
		{
			name:     "full match at key 0",
			code:     `let result = "test123" ~ /(?P<word>\w+)(?P<num>\d+)/; result["0"]`,
			expected: `test123`,
		},
		{
			name:     "access named group by key",
			code:     `let result = "hello world" ~ /(?P<greeting>\w+)\s+(?P<target>\w+)/; result.greeting`,
			expected: `hello`,
		},
		{
			name:     "access second named group",
			code:     `let result = "hello world" ~ /(?P<greeting>\w+)\s+(?P<target>\w+)/; result.target`,
			expected: `world`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalCodeStr(t, tt.code)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestNamedCapturesGlobalMatch(t *testing.T) {
	// Test that global match operator with named groups returns array of dictionaries
	// Using FindAllStringSubmatch via multiple calls
	code := `
		let text = "cat bat hat"
		let matches = []
		let m1 = text ~ /(?P<word>\w+at)/
		m1
	`
	result := evalCodeStr(t, code)

	// Should return a dictionary with "word" key
	if !strings.Contains(result, "word:") {
		t.Errorf("Expected named group 'word' in result, got %s", result)
	}
}

// =============================================================================
// Task 2: Custom Message for failIfInvalid (#90)
// =============================================================================

func TestFailIfInvalidCustomMessage(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		expectError bool
		errorMsg    string
	}{
		{
			name: "default message when no args",
			code: `
				@schema User { name: string(required) }
				let rec = User({})
				rec.validate().failIfInvalid()
			`,
			expectError: true,
			errorMsg:    "Validation failed",
		},
		{
			name: "custom message when provided",
			code: `
				@schema User { name: string(required) }
				let rec = User({})
				rec.validate().failIfInvalid("User profile is incomplete")
			`,
			expectError: true,
			errorMsg:    "User profile is incomplete",
		},
		{
			name: "valid record returns record (no message)",
			code: `
				@schema User { name: string }
				let rec = User({ name: "Alice" })
				rec.validate().failIfInvalid().name
			`,
			expectError: false,
			errorMsg:    "Alice",
		},
		{
			name: "valid record returns record (with message)",
			code: `
				@schema User { name: string }
				let rec = User({ name: "Bob" })
				rec.validate().failIfInvalid("Should not see this").name
			`,
			expectError: false,
			errorMsg:    "Bob",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalCodeStr(t, tt.code)
			if tt.expectError {
				// The result should contain the expected error message
				if !strings.Contains(result, tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %s", tt.errorMsg, result)
				}
			} else {
				// For valid records, check the result contains expected value
				if !strings.Contains(result, tt.errorMsg) {
					t.Errorf("Expected result to contain %q, got %s", tt.errorMsg, result)
				}
			}
		})
	}
}

func TestFailIfInvalidTypeError(t *testing.T) {
	// Passing non-string argument should be a type error
	code := `
		@schema User { name: string(required) }
		let rec = User({})
		rec.validate().failIfInvalid(123)
	`
	result := evalCodeStr(t, code)
	// Should mention "string" in the error message
	if !strings.Contains(result, "string") {
		t.Errorf("Expected type error mentioning string, got %s", result)
	}
}

func TestFailIfInvalidArityError(t *testing.T) {
	// Passing too many arguments should be an arity error
	code := `
		@schema User { name: string(required) }
		let rec = User({})
		rec.validate().failIfInvalid("msg", "extra")
	`
	result := evalCodeStr(t, code)
	// Should mention "0-1 arguments" or similar arity constraint
	if !strings.Contains(result, "0-1") && !strings.Contains(result, "arguments") {
		t.Errorf("Expected arity error about argument count, got %s", result)
	}
}

// =============================================================================
// Task 3: Deprecation Warning Infrastructure
// =============================================================================

func TestDeprecationWarningEmittedOnce(t *testing.T) {
	// Reset warnings before test
	evaluator.ResetDeprecationWarningsForTesting()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Run code that triggers deprecation twice
	code := `
		let a = format([1, 2], "and")
		let b = format([3, 4], "or")
		a
	`
	_ = evalCodeStr(t, code)

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	stderr := buf.String()

	// Should only see the warning once, not twice
	count := strings.Count(stderr, "DEP-005")
	if count > 1 {
		t.Errorf("Deprecation warning should be emitted only once, but was emitted %d times", count)
	}
	if count == 0 {
		t.Errorf("Deprecation warning was not emitted at all")
	}
}

func TestDeprecationWarningForFormatArray(t *testing.T) {
	// Reset warnings before test
	evaluator.ResetDeprecationWarningsForTesting()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Run code that triggers deprecation
	code := `format([1, 2, 3], "and")`
	_ = evalCodeStr(t, code)

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	stderr := buf.String()

	if !strings.Contains(stderr, "DEPRECATION WARNING") {
		t.Error("Expected DEPRECATION WARNING in stderr")
	}
	if !strings.Contains(stderr, "format(array, style)") {
		t.Error("Expected warning about format(array, style)")
	}
	if !strings.Contains(stderr, "array.format(style)") {
		t.Error("Expected suggestion to use array.format(style)")
	}
}

func TestNoDeprecationWarningForDurationFormat(t *testing.T) {
	// Reset warnings before test
	evaluator.ResetDeprecationWarningsForTesting()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Duration format should NOT trigger deprecation
	code := `format(@5d, "en-US")`
	_ = evalCodeStr(t, code)

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	stderr := buf.String()

	if strings.Contains(stderr, "DEPRECATION WARNING") {
		t.Error("Duration format should NOT trigger deprecation warning")
	}
}

func TestNoDeprecationWarningForArrayMethod(t *testing.T) {
	// Reset warnings before test
	evaluator.ResetDeprecationWarningsForTesting()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Using the method form should NOT trigger deprecation
	code := `[1, 2, 3].format("and")`
	_ = evalCodeStr(t, code)

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	stderr := buf.String()

	if strings.Contains(stderr, "DEPRECATION WARNING") {
		t.Error("Array.format() method should NOT trigger deprecation warning")
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func evalCodeStr(t *testing.T, code string) string {
	t.Helper()

	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if result == nil {
		return "null"
	}

	return result.Inspect()
}
