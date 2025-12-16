package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestStringOutdent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic common indent",
			input:    "    Hello\n    World\n    Test",
			expected: "Hello\nWorld\nTest",
		},
		{
			name:     "mixed indents",
			input:    "        Line 1\n      Line 2\n        Line 3",
			expected: "  Line 1\nLine 2\n  Line 3",
		},
		{
			name:     "with blank lines",
			input:    "    First\n    \n    Second",
			expected: "First\n\nSecond",
		},
		{
			name:     "with column 0 lines",
			input:    "No indent\n    Some indent\n    More indent",
			expected: "No indent\nSome indent\nMore indent",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no indent",
			input:    "Hello\nWorld",
			expected: "Hello\nWorld",
		},
		{
			name:     "whitespace only lines ignored in measurement",
			input:    "        \n    Line 1\n    Line 2\n        ",
			expected: "\nLine 1\nLine 2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := "`" + tt.input + "`.outdent()"
			l := lexer.New(code)
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

			str, ok := result.(*evaluator.String)
			if !ok {
				t.Fatalf("expected STRING, got %s", result.Type())
			}

			if str.Value != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, str.Value)
			}
		})
	}
}

func TestStringIndent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		spaces   int
		expected string
	}{
		{
			name:     "basic indent",
			input:    "Hello\nWorld",
			spaces:   4,
			expected: "    Hello\n    World",
		},
		{
			name:     "indent with blank lines",
			input:    "First\n\nSecond",
			spaces:   2,
			expected: "  First\n\n  Second",
		},
		{
			name:     "zero spaces",
			input:    "Hello\nWorld",
			spaces:   0,
			expected: "Hello\nWorld",
		},
		{
			name:     "empty string",
			input:    "",
			spaces:   4,
			expected: "",
		},
		{
			name:     "preserve existing indent",
			input:    "  Hello\n  World",
			spaces:   2,
			expected: "    Hello\n    World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := "`" + tt.input + "`.indent(" + string(rune('0'+tt.spaces)) + ")"
			l := lexer.New(code)
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

			str, ok := result.(*evaluator.String)
			if !ok {
				t.Fatalf("expected STRING, got %s", result.Type())
			}

			if str.Value != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, str.Value)
			}
		})
	}
}

func TestStringOutdentIndentRoundtrip(t *testing.T) {
	input := `    Hello
    World
    Test`
	expected := "  Hello\n  World\n  Test"

	code := "`" + input + "`.outdent().indent(2)"
	l := lexer.New(code)
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

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected STRING, got %s", result.Type())
	}

	if str.Value != expected {
		t.Errorf("expected %q, got %q", expected, str.Value)
	}
}
