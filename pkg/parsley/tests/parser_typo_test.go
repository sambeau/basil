package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// TestKeywordTypoDetection tests that the parser detects common keyword typos
// Note: The typo detection only works when a typo identifier is followed by another identifier
// This is the pattern: `expoert value = 5` (identifier identifier) triggers detection
func TestKeywordTypoDetection(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string // Substring that should appear in error
	}{
		// Export typos - these work because "expoert value" is ident ident
		{
			name:          "expoert_typo",
			input:         `expoert value = 5`,
			expectedError: "Did you mean 'export'",
		},
		{
			name:          "exprot_typo",
			input:         `exprot name = "test"`,
			expectedError: "Did you mean 'export'",
		},
		{
			name:          "exort_typo",
			input:         `exort x = 10`,
			expectedError: "Did you mean 'export'",
		},

		// Let typos - these work because "lte x" is ident ident
		{
			name:          "lte_typo",
			input:         `lte x = 5`,
			expectedError: "Did you mean 'let'",
		},
		{
			name:          "lett_typo",
			input:         `lett y = 10`,
			expectedError: "Did you mean 'let'",
		},

		// Return typos - need ident ident pattern
		{
			name:          "retrun_typo",
			input:         `let f = fn(x) { retrun value }`,
			expectedError: "Did you mean 'return'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			_ = p.ParseProgram()

			errors := p.Errors()

			// Should have at least one error
			if len(errors) == 0 {
				t.Fatalf("expected parser error for typo, got none")
			}

			// Check that one of the errors contains our expected message
			found := false
			for _, err := range errors {
				if strings.Contains(err, tt.expectedError) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected error containing %q, got errors: %v", tt.expectedError, errors)
			}
		})
	}
}

// TestCorrectKeywordsStillWork tests that correct keywords work normally
func TestCorrectKeywordsStillWork(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "correct_export",
			input: `export value = 42`,
		},
		{
			name:  "correct_let",
			input: `let x = 5`,
		},
		{
			name:  "correct_fn",
			input: `let f = fn(x) { x * 2 }`,
		},
		{
			name:  "correct_function_alias",
			input: `let g = function(x) { x + 1 }`,
		},
		{
			name:  "correct_return",
			input: `let f = fn() { return 5 }`,
		},
		{
			name:  "correct_for",
			input: `for (x in [1,2,3]) { x }`,
		},
		{
			name:  "correct_import",
			input: `import(@./test.pars)`,
		},
		{
			name:  "correct_true",
			input: `let flag = true`,
		},
		{
			name:  "correct_false",
			input: `let disabled = false`,
		},
		{
			name:  "correct_null",
			input: `let empty = null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if program == nil {
				t.Fatal("ParseProgram returned nil")
			}

			// Should have no errors
			if len(p.Errors()) > 0 {
				t.Errorf("unexpected parser errors: %v", p.Errors())
			}
		})
	}
}

// TestTypoHintMessages tests that hint messages are helpful
func TestTypoHintMessages(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // Expected hint in the message
	}{
		{
			name:     "export_hint_mentions_importing",
			input:    `expoert x = 5`,
			expected: "export",
		},
		{
			name:     "let_hint_shows_example",
			input:    `lte x = 5`,
			expected: "let",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			_ = p.ParseProgram()

			errors := p.Errors()
			if len(errors) == 0 {
				t.Fatal("expected parser error")
			}

			// Join all errors and check for expected hint
			allErrors := strings.Join(errors, " ")
			if !strings.Contains(allErrors, tt.expected) {
				t.Errorf("expected error to contain %q, got: %s", tt.expected, allErrors)
			}
		})
	}
}
