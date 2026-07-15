package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// TestUnicodeInRawTextTags tests that multi-byte UTF-8 characters inside
// <script> and <style> contents render intact (BUG-028). The raw-text
// scanner previously emitted only the first byte of each multi-byte rune.
func TestUnicodeInRawTextTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "multi-byte char in script",
			input:    `<script>var moon = "☾";</script>`,
			expected: `<script>var moon = "☾";</script>`,
		},
		{
			name:     "multi-byte char in style",
			input:    `<style>.a::before { content: "☾"; }</style>`,
			expected: `<style>.a::before { content: "☾"; }</style>`,
		},
		{
			name:     "mixed-width chars in script",
			input:    `<script>var s = "é ☾ 日本語 🌙";</script>`,
			expected: `<script>var s = "é ☾ 日本語 🌙";</script>`,
		},
		{
			name: "multi-byte chars around interpolation in script",
			input: `let theme = "dark"
<script>var sun = "☀"; var t = "@{theme}"; var moon = "☾";</script>`,
			expected: `<script>var sun = "☀"; var t = "dark"; var moon = "☾";</script>`,
		},
		{
			name: "multi-byte chars around interpolation in style",
			input: `let color = "red"
<style>.sun::before { content: "☀"; color: @{color}; }</style>`,
			expected: `<style>.sun::before { content: "☀"; color: red; }</style>`,
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

			if result.Inspect() != tt.expected {
				t.Errorf("Expected:\n%s\n\nGot:\n%s", tt.expected, result.Inspect())
			}
		})
	}
}
