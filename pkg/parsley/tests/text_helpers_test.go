package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// ============================================================================
// Text View Helper Tests (FEAT-048)
// ============================================================================

// TestHighlightMethod tests the string.highlight(phrase, tag?) method
func TestHighlightMethod(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic highlighting with default <mark> tag
		{
			name:     "basic highlight",
			input:    `"hello world".highlight("world")`,
			expected: "hello <mark>world</mark>",
		},
		{
			name:     "highlight at start",
			input:    `"world hello".highlight("world")`,
			expected: "<mark>world</mark> hello",
		},
		{
			name:     "highlight in middle",
			input:    `"say hello friend".highlight("hello")`,
			expected: "say <mark>hello</mark> friend",
		},
		{
			name:     "no match",
			input:    `"hello world".highlight("foo")`,
			expected: "hello world",
		},

		// Case-insensitive matching
		{
			name:     "case insensitive match",
			input:    `"Hello World".highlight("hello")`,
			expected: "<mark>Hello</mark> World",
		},
		{
			name:     "case insensitive uppercase",
			input:    `"hello world".highlight("WORLD")`,
			expected: "hello <mark>world</mark>",
		},

		// Multiple occurrences
		{
			name:     "multiple occurrences",
			input:    `"hello world hello".highlight("hello")`,
			expected: "<mark>hello</mark> world <mark>hello</mark>",
		},

		// Custom tag
		{
			name:     "custom strong tag",
			input:    `"hello world".highlight("world", "strong")`,
			expected: "hello <strong>world</strong>",
		},
		{
			name:     "custom em tag",
			input:    `"hello world".highlight("world", "em")`,
			expected: "hello <em>world</em>",
		},
		{
			name:     "custom span tag",
			input:    `"hello world".highlight("world", "span")`,
			expected: "hello <span>world</span>",
		},

		// HTML escaping - text should be escaped but matches wrapped in tags
		{
			name:     "html escaping",
			input:    `"<script>alert('xss')</script> hello".highlight("hello")`,
			expected: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt; <mark>hello</mark>",
		},

		// Edge cases
		{
			name:     "empty phrase",
			input:    `"hello world".highlight("")`,
			expected: "hello world",
		},
		{
			name:     "empty string",
			input:    `"".highlight("hello")`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			str, ok := result.(*evaluator.String)
			if !ok {
				err, isErr := result.(*evaluator.Error)
				if isErr {
					t.Fatalf("expected String, got Error: %s", err.Message)
				}
				t.Fatalf("expected String, got %T (%+v)", result, result)
			}
			if str.Value != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, str.Value)
			}
		})
	}
}

// TestHighlightMethodErrors tests error cases for highlight
func TestHighlightMethodErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		errorSubstr string
	}{
		{
			name:        "no arguments",
			input:       `"hello".highlight()`,
			errorSubstr: "expects 1-2 arguments",
		},
		{
			name:        "too many arguments",
			input:       `"hello".highlight("h", "mark", "extra")`,
			errorSubstr: "expects 1-2 arguments",
		},
		{
			name:        "non-string phrase",
			input:       `"hello".highlight(123)`,
			errorSubstr: "must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			err, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected Error, got %T (%+v)", result, result)
			}
			if !strings.Contains(err.Message, tt.errorSubstr) {
				t.Errorf("expected error containing %q, got %q", tt.errorSubstr, err.Message)
			}
		})
	}
}

// TestParagraphsMethod tests the string.paragraphs() method
func TestParagraphsMethod(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic paragraph wrapping
		{
			name:     "single paragraph",
			input:    `"Hello world".paragraphs()`,
			expected: "<p>Hello world</p>",
		},

		// Multiple paragraphs (double newline)
		{
			name:     "two paragraphs",
			input:    "\"First paragraph.\\n\\nSecond paragraph.\".paragraphs()",
			expected: "<p>First paragraph.</p><p>Second paragraph.</p>",
		},
		{
			name:     "three paragraphs",
			input:    "\"One\\n\\nTwo\\n\\nThree\".paragraphs()",
			expected: "<p>One</p><p>Two</p><p>Three</p>",
		},

		// Single newlines become <br/>
		{
			name:     "line break within paragraph",
			input:    "\"Line one\\nLine two\".paragraphs()",
			expected: "<p>Line one<br/>Line two</p>",
		},
		{
			name:     "mixed breaks",
			input:    "\"Para one line one\\nPara one line two\\n\\nPara two\".paragraphs()",
			expected: "<p>Para one line one<br/>Para one line two</p><p>Para two</p>",
		},

		// HTML escaping
		{
			name:     "html escaping",
			input:    `"<script>alert('xss')</script>".paragraphs()`,
			expected: "<p>&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;</p>",
		},
		{
			name:     "ampersand escaping",
			input:    `"Tom & Jerry".paragraphs()`,
			expected: "<p>Tom &amp; Jerry</p>",
		},

		// Edge cases
		{
			name:     "empty string",
			input:    `"".paragraphs()`,
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    `"   ".paragraphs()`,
			expected: "",
		},
		{
			name:     "multiple blank lines",
			input:    "\"First\\n\\n\\n\\nSecond\".paragraphs()",
			expected: "<p>First</p><p>Second</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			str, ok := result.(*evaluator.String)
			if !ok {
				err, isErr := result.(*evaluator.Error)
				if isErr {
					t.Fatalf("expected String, got Error: %s", err.Message)
				}
				t.Fatalf("expected String, got %T (%+v)", result, result)
			}
			if str.Value != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, str.Value)
			}
		})
	}
}

// TestParagraphsMethodErrors tests error cases for paragraphs
func TestParagraphsMethodErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		errorSubstr string
	}{
		{
			name:        "with arguments",
			input:       `"hello".paragraphs("arg")`,
			errorSubstr: "got=1, want=0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			err, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected Error, got %T (%+v)", result, result)
			}
			if !strings.Contains(err.Message, tt.errorSubstr) {
				t.Errorf("expected error containing %q, got %q", tt.errorSubstr, err.Message)
			}
		})
	}
}

// TestHumanizeMethod tests the number.humanize(locale?) method
func TestHumanizeMethod(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Integer humanize - basic cases
		{
			name:     "small integer",
			input:    `123.humanize()`,
			expected: "123",
		},
		{
			name:     "thousands",
			input:    `1234.humanize()`,
			expected: "1.2K",
		},
		{
			name:     "millions",
			input:    `1234567.humanize()`,
			expected: "1.2M",
		},
		{
			name:     "billions",
			input:    `1234567890.humanize()`,
			expected: "1.2B",
		},
		{
			name:     "negative thousands",
			input:    `(-1234).humanize()`,
			expected: "-1.2K",
		},
		{
			name:     "zero",
			input:    `0.humanize()`,
			expected: "0",
		},

		// Float humanize
		{
			name:     "small float",
			input:    `123.45.humanize()`,
			expected: "123.5", // rounds to nearest
		},
		{
			name:     "float thousands",
			input:    `1234.56.humanize()`,
			expected: "1.2K",
		},
		{
			name:     "float millions",
			input:    `1500000.0.humanize()`,
			expected: "1.5M",
		},

		// With locale
		{
			name:     "german locale thousands",
			input:    `1234.humanize("de")`,
			expected: "1,2K", // German uses comma for decimal
		},
		{
			name:     "english locale",
			input:    `1234567.humanize("en")`,
			expected: "1.2M",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			str, ok := result.(*evaluator.String)
			if !ok {
				err, isErr := result.(*evaluator.Error)
				if isErr {
					t.Fatalf("expected String, got Error: %s", err.Message)
				}
				t.Fatalf("expected String, got %T (%+v)", result, result)
			}
			if str.Value != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, str.Value)
			}
		})
	}
}

// TestHumanizeMethodErrors tests error cases for humanize
func TestHumanizeMethodErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		errorSubstr string
	}{
		{
			name:        "too many arguments",
			input:       `1234.humanize("en", "extra")`,
			errorSubstr: "expects 0-1 arguments",
		},
		{
			name:        "non-string locale",
			input:       `1234.humanize(123)`,
			errorSubstr: "must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			err, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected Error, got %T (%+v)", result, result)
			}
			if !strings.Contains(err.Message, tt.errorSubstr) {
				t.Errorf("expected error containing %q, got %q", tt.errorSubstr, err.Message)
			}
		})
	}
}

// TestTextHelpersInTemplate tests using text helpers within templates
func TestTextHelpersInTemplate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "highlight in template",
			input: `
query = "search"
text = "Search results for your search query"
<span>text.highlight(query)</span>
`,
			expected: "<span><mark>Search</mark> results for your <mark>search</mark> query</span>",
		},
		{
			name: "paragraphs in template",
			input: `
bio = "First line.\n\nSecond paragraph."
<div class="bio">bio.paragraphs()</div>
`,
			expected: `<div class="bio"><p>First line.</p><p>Second paragraph.</p></div>`,
		},
		{
			name: "humanize in template",
			input: `
count = 1500000
<span>count.humanize() " views"</span>
`,
			expected: "<span>1.5M views</span>",
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

			str, ok := result.(*evaluator.String)
			if !ok {
				err, isErr := result.(*evaluator.Error)
				if isErr {
					t.Fatalf("expected String, got Error: %s", err.Message)
				}
				t.Fatalf("expected String, got %T (%+v)", result, result)
			}
			if str.Value != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, str.Value)
			}
		})
	}
}
