package tests

import (
	"os"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// evalWithArgs evaluates code with @env and @args populated
func evalWithArgs(input string, args []string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return &evaluator.Error{Message: p.Errors()[0]}
	}

	env := evaluator.NewEnvironmentWithArgs(args)
	return evaluator.Eval(program, env)
}

// TestBuiltinEnv tests the @env global variable
func TestBuiltinEnv(t *testing.T) {
	// Ensure HOME is set for testing
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME environment variable not set")
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "@env.HOME returns home directory",
			input:    `@env.HOME`,
			expected: home,
		},
		{
			name:     "@env dictionary key access",
			input:    `@env["HOME"]`,
			expected: home,
		},
		{
			name:     "@env includes PATH",
			input:    `@env.PATH != null`,
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalWithArgs(tt.input, nil)

			if errObj, ok := result.(*evaluator.Error); ok {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			got := result.Inspect()
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestBuiltinArgs tests the @args global variable
func TestBuiltinArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		args     []string
		expected string
	}{
		{
			name:     "@args is empty array by default",
			input:    `@args`,
			args:     nil,
			expected: "[]",
		},
		{
			name:     "@args with single arg",
			input:    `@args[0]`,
			args:     []string{"hello"},
			expected: "hello",
		},
		{
			name:     "@args with multiple args",
			input:    `@args[1]`,
			args:     []string{"first", "second", "third"},
			expected: "second",
		},
		{
			name:     "@args length",
			input:    `@args.length()`,
			args:     []string{"a", "b", "c"},
			expected: "3",
		},
		{
			name:     "@args first",
			input:    `@args[0]`,
			args:     []string{"hello", "world"},
			expected: "hello",
		},
		{
			name:     "@args last",
			input:    `@args[@args.length() - 1]`,
			args:     []string{"hello", "world"},
			expected: "world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalWithArgs(tt.input, tt.args)

			if errObj, ok := result.(*evaluator.Error); ok {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			got := result.Inspect()
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestBuiltinEnvNotFoundReturnsNull tests that accessing non-existent env vars returns null
func TestBuiltinEnvNotFoundReturnsNull(t *testing.T) {
	// Use a definitely-not-set env var name
	input := `@env.PARSLEY_TEST_NONEXISTENT_VAR_12345`
	result := evalWithArgs(input, nil)

	if result.Type() != evaluator.NULL_OBJ {
		t.Errorf("expected NULL for non-existent env var, got %s (%s)", result.Type(), result.Inspect())
	}
}

// TestBuiltinArgsOutOfBounds tests bounds checking on @args
func TestBuiltinArgsOutOfBounds(t *testing.T) {
	input := `@args[10]`
	result := evalWithArgs(input, []string{"a", "b"})

	// Out of bounds returns an error in Parsley
	if result.Type() != evaluator.ERROR_OBJ {
		t.Errorf("expected ERROR for out-of-bounds @args access, got %s (%s)", result.Type(), result.Inspect())
	}
}

// TestBuiltinEnvReassignmentBlocked tests that @env cannot be reassigned
func TestBuiltinEnvReassignmentBlocked(t *testing.T) {
	// The @ prefix variables should be stored but can be shadowed in inner scopes
	// This is consistent with other identifiers - you can't reassign a let binding
	input := `@env = {}`
	result := evalWithArgs(input, nil)

	// Should get an error about reassignment
	if result.Type() != evaluator.ERROR_OBJ {
		t.Errorf("expected error for @env reassignment, got %s", result.Type())
	}
}
