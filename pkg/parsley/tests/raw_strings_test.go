package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// TestRawStringBasics tests single-quoted raw string literals
func TestRawStringBasics(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple raw string",
			input:    `'hello world'`,
			expected: "hello world",
		},
		{
			name:     "raw string with double quotes",
			input:    `'hello "world"'`,
			expected: `hello "world"`,
		},
		{
			name:     "raw string with backslash n literal",
			input:    `'line1\nline2'`,
			expected: `line1\nline2`, // \n stays literal
		},
		{
			name:     "raw string with backslash t literal",
			input:    `'tab\there'`,
			expected: `tab\there`, // \t stays literal
		},
		{
			name:     "raw string with escaped single quote",
			input:    `'it\'s working'`,
			expected: `it's working`,
		},
		{
			name:     "raw string with escaped backslash",
			input:    `'path\\to\\file'`,
			expected: `path\to\file`,
		},
		{
			name:     "raw string with braces (no interpolation)",
			input:    `'value: {x}'`,
			expected: `value: {x}`, // braces stay literal
		},
		{
			name:     "javascript function call",
			input:    `'Parts.refresh("editor", {id: 1})'`,
			expected: `Parts.refresh("editor", {id: 1})`,
		},
		{
			name:     "empty raw string",
			input:    `''`,
			expected: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEvalHelper(tt.input)
			str, ok := evaluated.(*evaluator.String)
			if !ok {
				t.Fatalf("Expected String, got %T (%v)", evaluated, evaluated)
			}
			if str.Value != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, str.Value)
			}
		})
	}
}

// TestRawStringsInVariables tests using raw strings in let assignments
func TestRawStringsInVariables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "assign raw string to variable",
			input:    `let js = 'alert("hello")'; js`,
			expected: `alert("hello")`,
		},
		{
			name:     "concatenate raw string with double-quoted",
			input:    `let prefix = 'onclick='; let handler = '"alert(1)"'; prefix + handler`,
			expected: `onclick="alert(1)"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEvalHelper(tt.input)
			str, ok := evaluated.(*evaluator.String)
			if !ok {
				t.Fatalf("Expected String, got %T (%v)", evaluated, evaluated)
			}
			if str.Value != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, str.Value)
			}
		})
	}
}

// TestSingleQuotedTagAttributes tests single-quoted attributes in HTML tags
func TestSingleQuotedTagAttributes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // check that output contains this substring
	}{
		{
			name:     "simple single-quoted attribute",
			input:    `<button onclick='alert(1)'/>`,
			contains: `onclick='alert(1)'`,
		},
		{
			name:     "single-quoted attribute with double quotes inside",
			input:    `<button onclick='alert("hello")'/>`,
			contains: `onclick='alert("hello")'`,
		},
		{
			name:     "single-quoted attribute with JS object",
			input:    `<button onclick='fn({x: 1})'/>`,
			contains: `onclick='fn({x: 1})'`,
		},
		{
			name:     "complex JavaScript in single-quoted attribute",
			input:    `<button onclick='Parts.refresh("editor", {id: 1}, {view: "delete"})'/>`,
			contains: `onclick='Parts.refresh("editor", {id: 1}, {view: "delete"})'`,
		},
		{
			name:     "mixed single and double quoted attributes",
			input:    `<input type="text" onchange='validate("{name}")'/>`,
			contains: `onchange='validate("{name}")'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEvalHelper(tt.input)
			str, ok := evaluated.(*evaluator.String)
			if !ok {
				t.Fatalf("Expected String, got %T (%v)", evaluated, evaluated)
			}
			if !strings.Contains(str.Value, tt.contains) {
				t.Errorf("Expected output to contain %q, got %q", tt.contains, str.Value)
			}
		})
	}
}

// TestSingleQuotedTagPairAttributes tests single-quoted attributes in paired tags
func TestSingleQuotedTagPairAttributes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "paired tag with single-quoted onclick",
			input:    `<div onclick='alert({x: 1})'>"content"</div>`,
			contains: `onclick='alert({x: 1})'`,
		},
		{
			name: "nested tags with single-quoted attributes",
			input: `<form onsubmit='validate({})'>
				<button type='submit'>"Submit"</button>
			</form>`,
			contains: `onsubmit='validate({})'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEvalHelper(tt.input)
			str, ok := evaluated.(*evaluator.String)
			if !ok {
				t.Fatalf("Expected String, got %T (%v)", evaluated, evaluated)
			}
			if !strings.Contains(str.Value, tt.contains) {
				t.Errorf("Expected output to contain %q, got %q", tt.contains, str.Value)
			}
		})
	}
}

// TestRawStringDoesNotInterpolate verifies that {expr} is NOT evaluated in raw strings
func TestRawStringDoesNotInterpolate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "braces stay literal in raw string",
			input:    `let x = 42; 'value is {x}'`,
			expected: "value is {x}",
		},
		{
			name:     "complex expression stays literal",
			input:    `'result: {1 + 2}'`,
			expected: "result: {1 + 2}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEvalHelper(tt.input)
			str, ok := evaluated.(*evaluator.String)
			if !ok {
				t.Fatalf("Expected String, got %T (%v)", evaluated, evaluated)
			}
			if str.Value != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, str.Value)
			}
		})
	}
}

// TestRawStringErrors tests error cases for raw strings
func TestRawStringErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "unterminated raw string",
			input:         `'hello`,
			expectedError: "unterminated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use lexer/parser directly to check for parser errors
			l := lexer.New(tt.input)
			p := parser.New(l)
			_ = p.ParseProgram()
			errors := p.Errors()
			if len(errors) == 0 {
				t.Fatalf("Expected parse error, got none")
			}
			foundExpected := false
			for _, err := range errors {
				if strings.Contains(strings.ToLower(err), strings.ToLower(tt.expectedError)) {
					foundExpected = true
					break
				}
			}
			if !foundExpected {
				t.Errorf("Expected error containing %q, got %v", tt.expectedError, errors)
			}
		})
	}
}

// TestRawTemplateInterpolation tests @{} interpolation in single-quoted strings
func TestRawTemplateInterpolation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple @{} interpolation",
			input:    `let x = 42; 'value is @{x}'`,
			expected: "value is 42",
		},
		{
			name:     "multiple @{} interpolations",
			input:    `let a = 1; let b = 2; 'a=@{a}, b=@{b}'`,
			expected: "a=1, b=2",
		},
		{
			name:     "expression in @{}",
			input:    `'result: @{1 + 2}'`,
			expected: "result: 3",
		},
		{
			name:     "@{} with double quotes preserved",
			input:    `let id = 5; 'fn("arg", @{id})'`,
			expected: `fn("arg", 5)`,
		},
		{
			name:     "@{} with JS braces preserved",
			input:    `let id = 42; 'Parts.refresh("editor", {id: @{id}})'`,
			expected: `Parts.refresh("editor", {id: 42})`,
		},
		{
			name:     "escaped @ with \\@",
			input:    `let x = 42; 'email: user\\@domain.com, value: @{x}'`,
			expected: "email: user@domain.com, value: 42",
		},
		{
			name:     "complex JS with interpolation",
			input:    `let myId = 99; let view = "delete"; 'Parts.refresh("editor", {id: @{myId}}, {view: "@{view}"})'`,
			expected: `Parts.refresh("editor", {id: 99}, {view: "delete"})`,
		},
		{
			name:     "no @{} means plain raw string",
			input:    `'no interpolation here: {x}'`,
			expected: "no interpolation here: {x}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEvalHelper(tt.input)
			str, ok := evaluated.(*evaluator.String)
			if !ok {
				t.Fatalf("Expected String, got %T (%v)", evaluated, evaluated)
			}
			if str.Value != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, str.Value)
			}
		})
	}
}

// TestRawTemplateInTagAttributes tests @{} interpolation in single-quoted tag attributes
func TestRawTemplateInTagAttributes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "simple @{} in onclick",
			input:    `let id = 42; <button onclick='deleteItem(@{id})'/>`,
			contains: `onclick='deleteItem(42)'`,
		},
		{
			name:     "@{} with JS object syntax",
			input:    `let myId = 5; <button onclick='fn({id: @{myId}})'/>`,
			contains: `onclick='fn({id: 5})'`,
		},
		{
			name:     "complex Parts.refresh with @{}",
			input:    `let id = 1; let view = "delete"; <button onclick='Parts.refresh("editor", {id: @{id}}, {view: "@{view}"})'/>`,
			contains: `onclick='Parts.refresh("editor", {id: 1}, {view: "delete"})'`,
		},
		{
			name:     "multiple @{} in attribute",
			input:    `let a = 10; let b = 20; <div data-info='a=@{a}, b=@{b}'/>`,
			contains: `data-info='a=10, b=20'`,
		},
		{
			name:     "mixed attribute types",
			input:    `let x = 5; <input type="text" onclick='handle(@{x})'/>`,
			contains: `onclick='handle(5)'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEvalHelper(tt.input)
			str, ok := evaluated.(*evaluator.String)
			if !ok {
				t.Fatalf("Expected String, got %T (%v)", evaluated, evaluated)
			}
			if !strings.Contains(str.Value, tt.contains) {
				t.Errorf("Expected output to contain %q, got %q", tt.contains, str.Value)
			}
		})
	}
}

// TestRawTemplateInPairedTagAttributes tests @{} in paired tag attributes
func TestRawTemplateInPairedTagAttributes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "paired tag with @{} onclick",
			input:    `let id = 7; <button onclick='submit(@{id})'>"Click"</button>`,
			contains: `onclick='submit(7)'`,
		},
		{
			name:     "nested tags with @{}",
			input:    `let formId = 123; <form onsubmit='validate(@{formId})'><button>"Go"</button></form>`,
			contains: `onsubmit='validate(123)'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEvalHelper(tt.input)
			str, ok := evaluated.(*evaluator.String)
			if !ok {
				t.Fatalf("Expected String, got %T (%v)", evaluated, evaluated)
			}
			if !strings.Contains(str.Value, tt.contains) {
				t.Errorf("Expected output to contain %q, got %q", tt.contains, str.Value)
			}
		})
	}
}
