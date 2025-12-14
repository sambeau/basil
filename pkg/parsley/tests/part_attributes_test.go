package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestPartAttributesRender(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "part-click attribute",
			input: `<button part-click="increment">"+"</button>`,
			contains: []string{
				`part-click="increment"`,
				`<button`,
				`>+</button>`,
			},
		},
		{
			name:  "part-click with part-count",
			input: `<button part-click="increment" part-count={5}>"+"</button>`,
			contains: []string{
				`part-click="increment"`,
				`part-count="5"`,
			},
		},
		{
			name:  "part-submit attribute",
			input: `<form part-submit="save"><input name="title"/></form>`,
			contains: []string{
				`part-submit="save"`,
				`<form`,
			},
		},
		{
			name:  "multiple part attributes",
			input: `<button part-click="edit" part-id={123} part-name={"item"}>"Edit"</button>`,
			contains: []string{
				`part-click="edit"`,
				`part-id="123"`,
				`part-name="item"`,
			},
		},
		{
			name:  "part attribute with special characters",
			input: `<button part-click="save" part-message={"Hello & goodbye"}>"Save"</button>`,
			contains: []string{
				`part-click="save"`,
				// HTML escaping happens in objectToTemplateString, test actual value
				`part-message`,
			},
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

			str, ok := result.(*evaluator.String)
			if !ok {
				t.Fatalf("expected String, got %T", result)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(str.Value, expected) {
					t.Errorf("expected output to contain %q, got: %s", expected, str.Value)
				}
			}
		})
	}
}

func TestPartAttributesInTagPairs(t *testing.T) {
	input := `
		<div class="container">
			<button part-click="increment" part-count={5}>"+"</button>
			<button part-click="decrement" part-count={5}>"-"</button>
		</div>
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

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	// Verify both buttons have part attributes
	if !strings.Contains(str.Value, `part-click="increment"`) {
		t.Errorf("expected increment button with part-click attribute, got: %s", str.Value)
	}

	if !strings.Contains(str.Value, `part-click="decrement"`) {
		t.Errorf("expected decrement button with part-click attribute, got: %s", str.Value)
	}

	if strings.Count(str.Value, `part-count="5"`) != 2 {
		t.Errorf("expected two part-count attributes, got: %s", str.Value)
	}
}
