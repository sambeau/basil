package lexer

import "testing"

func TestShebangSupport(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedTokens []struct {
			tokenType TokenType
			literal   string
			line      int
		}
	}{
		{
			name: "simple shebang",
			input: `#!/usr/bin/env pars
"hello"`,
			expectedTokens: []struct {
				tokenType TokenType
				literal   string
				line      int
			}{
				{STRING, "hello", 2},
				{EOF, "", 2},
			},
		},
		{
			name: "shebang with code",
			input: `#!/usr/bin/env pars
let x = 42`,
			expectedTokens: []struct {
				tokenType TokenType
				literal   string
				line      int
			}{
				{LET, "let", 2},
				{IDENT, "x", 2},
				{ASSIGN, "=", 2},
				{INT, "42", 2},
				{EOF, "", 2},
			},
		},
		{
			name:  "shebang only",
			input: `#!/usr/bin/env pars`,
			expectedTokens: []struct {
				tokenType TokenType
				literal   string
				line      int
			}{
				{EOF, "", 2},
			},
		},
		{
			name:  "no shebang",
			input: `"hello"`,
			expectedTokens: []struct {
				tokenType TokenType
				literal   string
				line      int
			}{
				{STRING, "hello", 1},
				{EOF, "", 1},
			},
		},
		{
			name:  "hash but not shebang",
			input: `# comment`,
			expectedTokens: []struct {
				tokenType TokenType
				literal   string
				line      int
			}{
				{ILLEGAL, "#", 1},
			},
		},
		{
			name: "shebang with multiple lines",
			input: `#!/usr/bin/env pars
let a = 1
let b = 2`,
			expectedTokens: []struct {
				tokenType TokenType
				literal   string
				line      int
			}{
				{LET, "let", 2},
				{IDENT, "a", 2},
				{ASSIGN, "=", 2},
				{INT, "1", 2},
				{LET, "let", 3},
				{IDENT, "b", 3},
				{ASSIGN, "=", 3},
				{INT, "2", 3},
				{EOF, "", 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.input)

			for i, expected := range tt.expectedTokens {
				tok := l.NextToken()

				if tok.Type != expected.tokenType {
					t.Errorf("token[%d]: expected type %s, got %s", i, expected.tokenType, tok.Type)
				}

				if tok.Literal != expected.literal {
					t.Errorf("token[%d]: expected literal %q, got %q", i, expected.literal, tok.Literal)
				}

				if tok.Line != expected.line {
					t.Errorf("token[%d]: expected line %d, got %d", i, expected.line, tok.Line)
				}
			}
		})
	}
}

func TestShebangWithFilename(t *testing.T) {
	input := `#!/usr/bin/env pars
"test"`

	l := NewWithFilename(input, "script.pars")

	tok := l.NextToken()
	if tok.Type != STRING {
		t.Errorf("expected STRING, got %s", tok.Type)
	}
	if tok.Line != 2 {
		t.Errorf("expected line 2, got %d", tok.Line)
	}
}
