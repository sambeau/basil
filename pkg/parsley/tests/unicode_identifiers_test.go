package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// TestUnicodeIdentifiersEndToEnd tests that Unicode identifiers work through the full pipeline
func TestUnicodeIdentifiersEndToEnd(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Greek letter pi",
			input:    `let π = 3.14; π`,
			expected: "3.14",
		},
		{
			name:     "Greek letter alpha",
			input:    `let α = 1; α + 1`,
			expected: "2",
		},
		{
			name:     "Greek letters in expression",
			input:    `let α = 2; let β = 3; α * β`,
			expected: "6",
		},
		{
			name:     "Japanese identifier",
			input:    `let 結果 = 10; 結果`,
			expected: "10",
		},
		{
			name:     "Chinese function",
			input:    `let 加法 = fn(x, y) { x + y }; 加法(2, 3)`,
			expected: "5",
		},
		{
			name:     "Russian identifier",
			input:    `let привет = "hello"; привет`,
			expected: "hello",
		},
		{
			name:     "Mixed ASCII and Unicode",
			input:    `let total = 10; let Δ = 5; total + Δ`,
			expected: "15",
		},
		{
			name:     "Unicode in function parameter",
			input:    `let f = fn(π) { π * 2 }; f(3.14)`,
			expected: "6.28",
		},
		{
			name:     "Multiple Greek letters",
			input:    `let α = 1; let β = 2; let γ = 3; α + β + γ`,
			expected: "6",
		},
		{
			name:     "Unicode identifier with digit suffix",
			input:    `let α1 = 1; let α2 = 2; α1 + α2`,
			expected: "3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if errObj, ok := result.(*evaluator.Error); ok {
				t.Fatalf("evaluation error: %s", errObj.Inspect())
			}

			if result.Inspect() != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestUnicodeIdentifiersInHTML tests Unicode identifiers in HTML context
// This test uses an ASCII variable name in the tag attribute for historical reasons.
// See TestUnicodeInTagAttributes for tests that use Unicode directly in tag attributes.
func TestUnicodeIdentifiersInHTML(t *testing.T) {
	// Use ASCII variable name in tag, but test that Unicode still works in code
	input := `let π = 3.14
let piValue = π
<div class="math" data-pi={piValue}/>`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("evaluation error: %s", errObj.Inspect())
	}

	expected := `<div class="math" data-pi="3.14" />`

	if result.Inspect() != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, result.Inspect())
	}
}

// TestUnicodeInTagAttributes tests Unicode identifiers directly in tag attribute interpolations
func TestUnicodeInTagAttributes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Greek letter in singleton tag",
			input:    `let π = 3.14; <div data={π}/>`,
			expected: `<div data="3.14" />`,
		},
		{
			name:     "Unicode in paired tag attribute",
			input:    `let 値 = "test"; <div data={値}>"content"</div>`,
			expected: `<div data="test">content</div>`,
		},
		{
			name:     "Multiple Unicode vars in attributes",
			input:    `let α = 1; let β = 2; <div a={α} b={β}/>`,
			expected: `<div a="1" b="2" />`,
		},
		{
			name:     "Unicode expression in attribute",
			input:    `let π = 3.14; <div data={π * 2}/>`,
			expected: `<div data="6.28" />`,
		},
		{
			name:     "Unicode in string inside interpolation",
			input:    `<div data={"π = 3.14"}/>`,
			expected: `<div data="π = 3.14" />`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if errObj, ok := result.(*evaluator.Error); ok {
				t.Fatalf("evaluation error: %s", errObj.Inspect())
			}

			if result.Inspect() != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestUnicodeTagNames tests Unicode characters in tag names (both custom components and closing tags)
func TestUnicodeTagNames(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Unicode in custom component name (Greek)",
			input: `let Πcircle = fn(props) { <div class="circle">props.radius</div> }
<Πcircle radius={5}/>`,
			expected: `<div class="circle">5</div>`,
		},
		{
			name: "Unicode suffix in component name",
			input: `let Showπ = fn(props) { <span>props.value</span> }
<Showπ value={3.14}/>`,
			expected: `<span>3.14</span>`,
		},
		{
			name: "Japanese component name with uppercase prefix",
			// Japanese kanji are not uppercase, so we prefix with an uppercase letter
			input: `let J表示 = fn(props) { <div>props.text</div> }
<J表示 text="hello"/>`,
			expected: `<div>hello</div>`,
		},
		{
			name: "Unicode paired component",
			input: `let Карточка = fn({contents}) { <div class="card">contents</div> }
<Карточка>"Card content"</Карточка>`,
			expected: `<div class="card">Card content</div>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if errObj, ok := result.(*evaluator.Error); ok {
				t.Fatalf("evaluation error: %s", errObj.Inspect())
			}

			if result.Inspect() != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestUnicodeCapitalDetection tests that Unicode uppercase letters correctly identify custom components
func TestUnicodeCapitalDetection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Greek capital Pi is custom component",
			// Π (Greek capital Pi) should be treated as a custom component
			input: `let Π = fn(props) { <span class="pi">props.v</span> }
<Π v="3.14"/>`,
			expected: `<span class="pi">3.14</span>`,
		},
		{
			name: "Cyrillic capital A is custom component",
			// А (Cyrillic capital A, U+0410) should be treated as custom
			input: `let А = fn(props) { <div>props.x</div> }
<А x="test"/>`,
			expected: `<div>test</div>`,
		},
		{
			name: "Greek lowercase pi is NOT custom component",
			// π (Greek lowercase pi) would be treated as standard HTML tag
			// Standard tags just pass through as-is
			input:    `<π class="circle"/>`,
			expected: `<π class="circle" />`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if errObj, ok := result.(*evaluator.Error); ok {
				t.Fatalf("evaluation error: %s", errObj.Inspect())
			}

			if result.Inspect() != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}
