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
			expected: `<div id="test" class="box" />`, // Insertion order: id, then class
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
			expected: `<div id="foo" class="override" />`, // id from base, class overridden by override
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
			expected: `<input name="email" type="text" placeholder="Email" required />`, // disabled=false omitted
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
			errorMsg: "requires a dictionary",
		},
		{
			name: "spread undefined variable",
			input: `
				<input ...notDefined/>
			`,
			errorMsg: "not found",
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

// TestExpressionAttributeSyntax tests the attr={expr} syntax for HTML attributes
func TestExpressionAttributeSyntax(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "null first attribute omitted",
			input:    `<input value={null} name="test"/>`,
			expected: `<input name="test" />`,
		},
		{
			name:     "false first attribute omitted",
			input:    `<input disabled={false} name="test"/>`,
			expected: `<input name="test" />`,
		},
		{
			name:     "null middle attribute omitted",
			input:    `<input type="text" value={null} name="test"/>`,
			expected: `<input type="text" name="test" />`,
		},
		{
			name:     "null end attribute omitted",
			input:    `<input type="text" name="test" value={null}/>`,
			expected: `<input type="text" name="test" />`,
		},
		{
			name:     "multiple omitted attributes",
			input:    `<input disabled={false} value={null} name="test"/>`,
			expected: `<input name="test" />`,
		},
		{
			name:     "all attributes omitted",
			input:    `<input disabled={false} value={null}/>`,
			expected: `<input />`,
		},
		{
			name:     "true boolean renders as attribute name only",
			input:    `<input disabled={true} name="test"/>`,
			expected: `<input disabled name="test" />`,
		},
		{
			name:     "string value quoted",
			input:    `<input value={"hello"} name="test"/>`,
			expected: `<input value="hello" name="test" />`,
		},
		{
			name:     "empty string value quoted",
			input:    `<input value={""} name="test"/>`,
			expected: `<input value="" name="test" />`,
		},
		{
			name:     "value with quotes escaped",
			input:    `<input value={"say \"hello\""} name="test"/>`,
			expected: `<input value="say &quot;hello&quot;" name="test" />`,
		},
		{
			name:     "value with HTML entities escaped",
			input:    `<input value={"<>&"} name="test"/>`,
			expected: `<input value="&lt;&gt;&amp;" name="test" />`,
		},
		{
			name:     "integer value quoted",
			input:    `<input maxlength={50} name="test"/>`,
			expected: `<input maxlength="50" name="test" />`,
		},
		{
			name:     "variable expression",
			input:    `let val = "email"; <input type={val}/>`,
			expected: `<input type="email" />`,
		},
		{
			name:     "null variable omitted",
			input:    `let val = null; <input value={val} name="test"/>`,
			expected: `<input name="test" />`,
		},
		{
			name: "mixed with spread - null omitted",
			input: `
				let props = {name: "search", type: "search", placeholder: "search"}
				let {value, ...rest} = props
				<input value={value} ...rest/>
			`,
			expected: `<input name="search" placeholder="search" type="search" />`,
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

			actual := strings.TrimSpace(result.Inspect())
			if actual != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, actual)
			}
		})
	}
}
