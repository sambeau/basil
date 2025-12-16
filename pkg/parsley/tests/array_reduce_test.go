package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestArrayReduce(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "sum numbers",
			input:    `[1, 2, 3, 4].reduce(fn(acc, x) { acc + x }, 0)`,
			expected: int64(10),
		},
		{
			name:     "product numbers",
			input:    `[2, 3, 4].reduce(fn(acc, x) { acc * x }, 1)`,
			expected: int64(24),
		},
		{
			name:     "concatenate strings",
			input:    `["Hello", " ", "World"].reduce(fn(acc, s) { acc + s }, "")`,
			expected: "Hello World",
		},
		{
			name:     "build array with ++",
			input:    `[1, 2, 3].reduce(fn(acc, x) { acc ++ [x * 2] }, [])`,
			expected: []int64{2, 4, 6},
		},
		{
			name:     "find maximum",
			input:    `[3, 7, 2, 9, 1].reduce(fn(max, n) { if (n > max) { n } else { max } }, 0)`,
			expected: int64(9),
		},
		{
			name:     "count elements",
			input:    `["a", "b", "c", "d"].reduce(fn(count, _) { count + 1 }, 0)`,
			expected: int64(4),
		},
		{
			name:     "empty array returns initial",
			input:    `[].reduce(fn(acc, x) { acc + x }, 42)`,
			expected: int64(42),
		},
		{
			name:     "sum with initial non-zero",
			input:    `[1, 2, 3].reduce(fn(acc, x) { acc + x }, 10)`,
			expected: int64(16),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			// Check result based on expected type
			switch exp := tt.expected.(type) {
			case int64:
				intObj, ok := result.(*evaluator.Integer)
				if !ok {
					t.Fatalf("expected INTEGER, got %s (%s)", result.Type(), result.Inspect())
				}
				if intObj.Value != exp {
					t.Errorf("expected %d, got %d", exp, intObj.Value)
				}
			case string:
				strObj, ok := result.(*evaluator.String)
				if !ok {
					t.Fatalf("expected STRING, got %s (%s)", result.Type(), result.Inspect())
				}
				if strObj.Value != exp {
					t.Errorf("expected %q, got %q", exp, strObj.Value)
				}
			case []int64:
				arrObj, ok := result.(*evaluator.Array)
				if !ok {
					t.Fatalf("expected ARRAY, got %s (%s)", result.Type(), result.Inspect())
				}
				if len(arrObj.Elements) != len(exp) {
					t.Fatalf("expected array length %d, got %d", len(exp), len(arrObj.Elements))
				}
				for i, expVal := range exp {
					intObj, ok := arrObj.Elements[i].(*evaluator.Integer)
					if !ok {
						t.Fatalf("expected INTEGER at index %d, got %s", i, arrObj.Elements[i].Type())
					}
					if intObj.Value != expVal {
						t.Errorf("at index %d: expected %d, got %d", i, expVal, intObj.Value)
					}
				}
			case map[string]int64:
				dictObj, ok := result.(*evaluator.Dictionary)
				if !ok {
					t.Fatalf("expected DICTIONARY, got %s (%s)", result.Type(), result.Inspect())
				}
				if len(dictObj.Pairs) != len(exp) {
					t.Fatalf("expected dict size %d, got %d", len(exp), len(dictObj.Pairs))
				}
				for key, expVal := range exp {
					expr, ok := dictObj.Pairs[key]
					if !ok {
						t.Fatalf("expected key %q in dictionary", key)
					}
					// Evaluate the expression to get the value
					val := evaluator.Eval(expr, dictObj.Env)
					intObj, ok := val.(*evaluator.Integer)
					if !ok {
						t.Fatalf("expected INTEGER for key %q, got %s", key, val.Type())
					}
					if intObj.Value != expVal {
						t.Errorf("for key %q: expected %d, got %d", key, expVal, intObj.Value)
					}
				}
			}
		})
	}
}

func TestArrayReduceErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedErr string
	}{
		{
			name:        "wrong arity - no args",
			input:       `[1, 2, 3].reduce()`,
			expectedErr: "wrong number",
		},
		{
			name:        "wrong arity - one arg",
			input:       `[1, 2, 3].reduce(fn(a, x) { a + x })`,
			expectedErr: "wrong number",
		},
		{
			name:        "wrong arity - three args",
			input:       `[1, 2, 3].reduce(fn(a, x) { a + x }, 0, 1)`,
			expectedErr: "wrong number",
		},
		{
			name:        "first arg not function",
			input:       `[1, 2, 3].reduce(42, 0)`,
			expectedErr: "must be a function",
		},
		{
			name:        "error in reducer function",
			input:       `[1, 2, 3].reduce(fn(acc, x) { acc / 0 }, 1)`,
			expectedErr: "division by zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			errObj, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected ERROR, got %s (%s)", result.Type(), result.Inspect())
			}

			if tt.expectedErr != "" && !containsLowercase(errObj.Message, tt.expectedErr) {
				t.Errorf("expected error containing %q, got %q", tt.expectedErr, errObj.Message)
			}
		})
	}
}

func TestArrayReduceWithVariables(t *testing.T) {
	input := `
let numbers = [1, 2, 3, 4, 5]
let sum = fn(acc, x) { acc + x }
numbers.reduce(sum, 0)
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	intObj, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected INTEGER, got %s", result.Type())
	}

	if intObj.Value != 15 {
		t.Errorf("expected 15, got %d", intObj.Value)
	}
}

func containsLowercase(s, substr string) bool {
	return contains(strings.ToLower(s), strings.ToLower(substr))
}
