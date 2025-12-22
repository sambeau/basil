package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestDictionarySpreadBasic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "basic spread",
			input: `
				let attrs = {id: "test", class: "box"}
				<div ...attrs/>
			`,
			expected: `<div class="box" id="test" />`,
		},
		{
			name: "spread with boolean true",
			input: `
				let attrs = {disabled: true}
				<input ...attrs/>
			`,
			expected: `<input disabled />`,
		},
		{
			name: "spread with boolean false (omitted)",
			input: `
				let attrs = {disabled: false}
				<input ...attrs/>
			`,
			expected: `<input />`,
		},
		{
			name: "spread with null (omitted)",
			input: `
				let attrs = {placeholder: null}
				<input ...attrs/>
			`,
			expected: `<input />`,
		},
		{
			name: "spread with numbers",
			input: `
				let attrs = {maxlength: 50, min: 0}
				<input ...attrs/>
			`,
			expected: `<input maxlength="50" min="0" />`,
		},
		{
			name: "multiple spreads",
			input: `
				let base = {id: "foo", class: "base"}
				let override = {class: "override"}
				<div ...base ...override/>
			`,
			expected: `<div class="override" id="foo" />`,
		},
		{
			name: "mixed regular attrs and spreads",
			input: `
				let attrs = {class: "box"}
				<div id="x" ...attrs/>
			`,
			expected: `<div id="x" class="box" />`,
		},
		{
			name: "spread in paired tag",
			input: `
				let attrs = {class: "container"}
				<div ...attrs>"content"</div>
			`,
			expected: `<div class="container">content</div>`,
		},
		{
			name: "spread with rest destructuring",
			input: `
				let props = {name: "email", placeholder: "Email", required: true, disabled: false}
				let {name, ...inputAttrs} = props
				<input name={name} type="text" ...inputAttrs/>
			`,
			expected: `<input name="email" type="text" placeholder="Email" required />`,
		},
		{
			name: "spread with HTML escaping",
			input: `
				let attrs = {title: "A & B < C > D \"quoted\""}
				<div ...attrs/>
			`,
			expected: `<div title="A &amp; B &lt; C &gt; D &quot;quoted&quot;" />`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if result == nil {
				t.Fatalf("Eval returned nil")
			}

			if _, isErr := result.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %v", result.Inspect())
			}

			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

func TestDictionarySpreadErrors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		errorMsg string
	}{
		{
			name: "spread non-dictionary",
			input: `
				let notDict = "string"
				<input ...notDict/>
			`,
			errorMsg: "cannot spread",
		},
		{
			name: "spread undefined variable",
			input: `
				<input ...notDefined/>
			`,
			errorMsg: "undefined",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if result == nil {
				t.Fatal("expected error, got nil")
			}

			err, isErr := result.(*evaluator.Error)
			if !isErr {
				t.Fatalf("expected error, got %T", result)
			}

			if tt.errorMsg != "" && !strings.Contains(err.Message, tt.errorMsg) {
				t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Message)
			}
		})
	}
}

func TestDictionarySpreadWithComponents(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "component with spread",
			input: `
				let TextField = fn(props) {
					let {name, label, ...inputAttrs} = props
					<input name={name} type="text" ...inputAttrs/>
				}
				
				<TextField 
					name="email" 
					label="Email"
					placeholder="you@example.com"
					required={true}
					maxlength={100}
				/>
			`,
			expected: `<input name="email" type="text" maxlength="100" placeholder="you@example.com" required />`,
		},
		{
			name: "component with conditional spreads",
			input: `
				let Button = fn(props) {
					let {text, ...attrs} = props
					<button ...attrs>text</button>
				}
				
				<Button 
					text="Click" 
					disabled={false} 
					class="btn"
				/>
			`,
			expected: `<button class="btn">Click</button>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if result == nil {
				t.Fatalf("Eval returned nil")
			}

			if _, isErr := result.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %v", result.Inspect())
			}

			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}
