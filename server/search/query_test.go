package search

import (
	"testing"
)

func TestSanitizeQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		raw      bool
		expected string
	}{
		{
			name:     "simple query",
			query:    "hello world",
			raw:      false,
			expected: "hello AND world",
		},
		{
			name:     "quoted phrase",
			query:    `"hello world"`,
			raw:      false,
			expected: `"hello world"`,
		},
		{
			name:     "negation",
			query:    "hello -world",
			raw:      false,
			expected: "hello AND NOT world",
		},
		{
			name:     "mixed",
			query:    `hello "quoted phrase" -excluded`,
			raw:      false,
			expected: `hello AND "quoted phrase" AND NOT excluded`,
		},
		{
			name:     "empty query",
			query:    "",
			raw:      false,
			expected: "",
		},
		{
			name:     "whitespace only",
			query:    "   ",
			raw:      false,
			expected: "",
		},
		{
			name:     "raw mode",
			query:    "title:hello OR content:world",
			raw:      true,
			expected: "title:hello OR content:world",
		},
		{
			name:     "special characters",
			query:    `hello "world"`,
			raw:      false,
			expected: `hello AND "world"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeQuery(tt.query, tt.raw)
			if result != tt.expected {
				t.Errorf("SanitizeQuery(%q, %v) = %q, want %q",
					tt.query, tt.raw, result, tt.expected)
			}
		})
	}
}

func TestEscapeToken(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{`"quoted"`, `\"quoted\"`},
		{"(parens)", `\(parens\)`},
		{"normal", "normal"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeToken(tt.input)
			if result != tt.expected {
				t.Errorf("escapeToken(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
