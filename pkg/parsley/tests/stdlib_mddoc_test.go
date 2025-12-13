package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// TestMdDocBasic tests basic mdDoc functionality
func TestMdDocBasic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "mdDoc title",
			input:    `let {mdDoc} = import @std/mdDoc; let doc = mdDoc("# Hello World"); doc.title()`,
			expected: "Hello World",
		},
		{
			name:     "mdDoc wordCount",
			input:    `let {mdDoc} = import @std/mdDoc; let doc = mdDoc("# Hello World"); doc.wordCount()`,
			expected: int64(2),
		},
		{
			name:     "mdDoc text",
			input:    `let {mdDoc} = import @std/mdDoc; let doc = mdDoc("# Hello"); doc.text()`,
			expected: "Hello ",
		},
		{
			name:     "mdDoc inspect",
			input:    `let {mdDoc} = import @std/mdDoc; let doc = mdDoc("# Test"); "" + doc`,
			expected: `mdDoc("Test")`,
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

			switch expected := tt.expected.(type) {
			case string:
				if result.Type() != evaluator.STRING_OBJ {
					t.Fatalf("expected STRING, got %s (%s)", result.Type(), result.Inspect())
				}
				if result.(*evaluator.String).Value != expected {
					t.Errorf("expected %q, got %q", expected, result.(*evaluator.String).Value)
				}
			case int64:
				if result.Type() != evaluator.INTEGER_OBJ {
					t.Fatalf("expected INTEGER, got %s (%s)", result.Type(), result.Inspect())
				}
				if result.(*evaluator.Integer).Value != expected {
					t.Errorf("expected %d, got %d", expected, result.(*evaluator.Integer).Value)
				}
			}
		})
	}
}

// TestMdDocToHTML tests HTML rendering
func TestMdDocToHTML(t *testing.T) {
	input := `let {mdDoc} = import @std/mdDoc; let doc = mdDoc("# Hello"); doc.toHTML()`

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

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected STRING, got %s", result.Type())
	}

	if !strings.Contains(str.Value, "<h1") {
		t.Errorf("expected HTML with h1 tag, got: %s", str.Value)
	}
}

// TestMdDocHeadings tests headings extraction
func TestMdDocHeadings(t *testing.T) {
	input := `let {mdDoc} = import @std/mdDoc; let doc = mdDoc("# Title"); doc.headings().length()`

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

	intVal, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected INTEGER, got %s (%s)", result.Type(), result.Inspect())
	}

	if intVal.Value != 1 {
		t.Errorf("expected 1 heading, got %d", intVal.Value)
	}
}

// TestMdDocFindAll tests findAll
func TestMdDocFindAll(t *testing.T) {
	input := `let {mdDoc} = import @std/mdDoc; let doc = mdDoc("# H1\n\n## H2\n\n## H3"); doc.findAll("heading").length()`

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

	intVal, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected INTEGER, got %s (%s)", result.Type(), result.Inspect())
	}

	if intVal.Value != 3 {
		t.Errorf("expected 3 headings, got %d", intVal.Value)
	}
}

// TestMdDocDoesNotShadowMarkdownBuiltin tests that importing mdDoc doesn't shadow markdown builtin
func TestMdDocDoesNotShadowMarkdownBuiltin(t *testing.T) {
	input := `let {mdDoc} = import @std/mdDoc; markdown`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	// markdown should still be the builtin function, not an error
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("markdown builtin was shadowed: %s", result.Inspect())
	}

	// Should be a builtin
	if result.Type() != evaluator.BUILTIN_OBJ {
		t.Errorf("expected markdown to be BUILTIN, got %s (%s)", result.Type(), result.Inspect())
	}
}

// TestMdDocMap tests the map transformation
func TestMdDocMap(t *testing.T) {
	input := `let {mdDoc} = import @std/mdDoc; let doc = mdDoc("# Hello"); let mapped = doc.map(fn(n) { n }); mapped.title()`

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

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected STRING, got %s (%s)", result.Type(), result.Inspect())
	}

	if str.Value != "Hello" {
		t.Errorf("expected 'Hello', got %q", str.Value)
	}
}
