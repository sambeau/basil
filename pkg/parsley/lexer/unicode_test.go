package lexer

import (
	"testing"
)

// TestUnicodeIdentifiers tests that Unicode letters are recognized as identifiers
func TestUnicodeIdentifiers(t *testing.T) {
	tests := []struct {
		input    string
		expected []struct {
			tokenType TokenType
			literal   string
		}
	}{
		{
			input: "let π = 3.14",
			expected: []struct {
				tokenType TokenType
				literal   string
			}{
				{LET, "let"},
				{IDENT, "π"},
				{ASSIGN, "="},
				{FLOAT, "3.14"},
				{EOF, ""},
			},
		},
		{
			input: "let α = 1",
			expected: []struct {
				tokenType TokenType
				literal   string
			}{
				{LET, "let"},
				{IDENT, "α"},
				{ASSIGN, "="},
				{INT, "1"},
				{EOF, ""},
			},
		},
		{
			input: "let 日本語 = true",
			expected: []struct {
				tokenType TokenType
				literal   string
			}{
				{LET, "let"},
				{IDENT, "日本語"},
				{ASSIGN, "="},
				{TRUE, "true"},
				{EOF, ""},
			},
		},
		{
			input: "let Ω = false",
			expected: []struct {
				tokenType TokenType
				literal   string
			}{
				{LET, "let"},
				{IDENT, "Ω"},
				{ASSIGN, "="},
				{FALSE, "false"},
				{EOF, ""},
			},
		},
		{
			input: "let αβγ = α + β",
			expected: []struct {
				tokenType TokenType
				literal   string
			}{
				{LET, "let"},
				{IDENT, "αβγ"},
				{ASSIGN, "="},
				{IDENT, "α"},
				{PLUS, "+"},
				{IDENT, "β"},
				{EOF, ""},
			},
		},
		{
			input: "fn 计算(数值) { return 数值 * 2 }",
			expected: []struct {
				tokenType TokenType
				literal   string
			}{
				{FUNCTION, "fn"},
				{IDENT, "计算"},
				{LPAREN, "("},
				{IDENT, "数值"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{RETURN, "return"},
				{IDENT, "数值"},
				{ASTERISK, "*"},
				{INT, "2"},
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
		// Mixed ASCII and Unicode
		{
			input: "let piValue = π",
			expected: []struct {
				tokenType TokenType
				literal   string
			}{
				{LET, "let"},
				{IDENT, "piValue"},
				{ASSIGN, "="},
				{IDENT, "π"},
				{EOF, ""},
			},
		},
		// Unicode followed by digits
		{
			input: "let α1 = 1",
			expected: []struct {
				tokenType TokenType
				literal   string
			}{
				{LET, "let"},
				{IDENT, "α1"},
				{ASSIGN, "="},
				{INT, "1"},
				{EOF, ""},
			},
		},
		// Russian identifier
		{
			input: "let привет = true",
			expected: []struct {
				tokenType TokenType
				literal   string
			}{
				{LET, "let"},
				{IDENT, "привет"},
				{ASSIGN, "="},
				{TRUE, "true"},
				{EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := New(tt.input)

			for i, expectedToken := range tt.expected {
				tok := l.NextToken()

				if tok.Type != expectedToken.tokenType {
					t.Fatalf("token %d: type wrong. expected=%q, got=%q (literal=%q)",
						i, expectedToken.tokenType, tok.Type, tok.Literal)
				}

				if tok.Literal != expectedToken.literal {
					t.Fatalf("token %d: literal wrong. expected=%q, got=%q",
						i, expectedToken.literal, tok.Literal)
				}
			}
		})
	}
}

// TestUnicodeInStrings tests that Unicode in string literals is preserved
func TestUnicodeInStrings(t *testing.T) {
	tests := []struct {
		input          string
		expectedString string
	}{
		{`"Hello, 世界!"`, "Hello, 世界!"},
		{`"π ≈ 3.14"`, "π ≈ 3.14"},
		{`"emoji: 🎉🎊🎈"`, "emoji: 🎉🎊🎈"},
		{`"中文字符"`, "中文字符"},
		{`"Привет мир"`, "Привет мир"},
		{`"مرحبا"`, "مرحبا"},
		{`"שלום"`, "שלום"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := New(tt.input)
			tok := l.NextToken()

			if tok.Type != STRING {
				t.Fatalf("expected STRING token, got %s", tok.Type)
			}

			if tok.Literal != tt.expectedString {
				t.Fatalf("string content wrong. expected=%q, got=%q", tt.expectedString, tok.Literal)
			}
		})
	}
}

// TestUnicodeCurrencySymbols tests that Unicode currency symbols still work
func TestUnicodeCurrencySymbols(t *testing.T) {
	tests := []struct {
		input    string
		expected struct {
			tokenType TokenType
			literal   string
		}
	}{
		{"£100", struct {
			tokenType TokenType
			literal   string
		}{MONEY, "GBP#100.00"}},
		{"€50", struct {
			tokenType TokenType
			literal   string
		}{MONEY, "EUR#50.00"}},
		{"¥1000", struct {
			tokenType TokenType
			literal   string
		}{MONEY, "JPY#1000"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := New(tt.input)
			tok := l.NextToken()

			if tok.Type != tt.expected.tokenType {
				t.Fatalf("token type wrong. expected=%q, got=%q", tt.expected.tokenType, tok.Type)
			}

			if tok.Literal != tt.expected.literal {
				t.Fatalf("literal wrong. expected=%q, got=%q", tt.expected.literal, tok.Literal)
			}
		})
	}
}
