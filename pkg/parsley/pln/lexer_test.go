package pln

import (
	"testing"
)

func TestLexPrimitives(t *testing.T) {
	tests := []struct {
		input    string
		expected []Token
	}{
		{
			"42",
			[]Token{{INT, "42", 1, 1}},
		},
		{
			"-42",
			[]Token{{INT, "-42", 1, 1}},
		},
		{
			"3.14",
			[]Token{{FLOAT, "3.14", 1, 1}},
		},
		{
			"-3.14",
			[]Token{{FLOAT, "-3.14", 1, 1}},
		},
		{
			"1.5e10",
			[]Token{{FLOAT, "1.5e10", 1, 1}},
		},
		{
			`"hello"`,
			[]Token{{STRING, "hello", 1, 1}},
		},
		{
			`"hello\nworld"`,
			[]Token{{STRING, "hello\nworld", 1, 1}},
		},
		{
			`"escaped\"quote"`,
			[]Token{{STRING, `escaped"quote`, 1, 1}},
		},
		{
			"true",
			[]Token{{TRUE, "true", 1, 1}},
		},
		{
			"false",
			[]Token{{FALSE, "false", 1, 1}},
		},
		{
			"null",
			[]Token{{NULL, "null", 1, 1}},
		},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		for i, expected := range tt.expected {
			tok := l.NextToken()
			if tok.Type != expected.Type {
				t.Errorf("input %q token %d: type wrong. expected=%v, got=%v",
					tt.input, i, expected.Type, tok.Type)
			}
			if tok.Literal != expected.Literal {
				t.Errorf("input %q token %d: literal wrong. expected=%q, got=%q",
					tt.input, i, expected.Literal, tok.Literal)
			}
		}
		// Verify EOF
		tok := l.NextToken()
		if tok.Type != EOF {
			t.Errorf("input %q: expected EOF, got %v", tt.input, tok.Type)
		}
	}
}

func TestLexCollections(t *testing.T) {
	tests := []struct {
		input    string
		expected []TokenType
	}{
		{
			"[1, 2, 3]",
			[]TokenType{LBRACKET, INT, COMMA, INT, COMMA, INT, RBRACKET},
		},
		{
			"[]",
			[]TokenType{LBRACKET, RBRACKET},
		},
		{
			"{a: 1}",
			[]TokenType{LBRACE, IDENT, COLON, INT, RBRACE},
		},
		{
			`{"a": 1, "b": 2}`,
			[]TokenType{LBRACE, STRING, COLON, INT, COMMA, STRING, COLON, INT, RBRACE},
		},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		for i, expectedType := range tt.expected {
			tok := l.NextToken()
			if tok.Type != expectedType {
				t.Errorf("input %q token %d: type wrong. expected=%v, got=%v (literal=%q)",
					tt.input, i, expectedType, tok.Type, tok.Literal)
			}
		}
	}
}

func TestLexRecords(t *testing.T) {
	input := `@Person({name: "Alice"})`
	l := NewLexer(input)

	// First we get AT token
	tok := l.NextToken()
	if tok.Type != AT {
		t.Errorf("expected AT, got %v %q", tok.Type, tok.Literal)
	}

	// Then IDENT for the schema name
	tok = l.NextToken()
	if tok.Type != IDENT || tok.Literal != "Person" {
		t.Errorf("expected IDENT 'Person', got %v %q", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != LPAREN {
		t.Errorf("expected LPAREN, got %v", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != LBRACE {
		t.Errorf("expected LBRACE, got %v", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != IDENT || tok.Literal != "name" {
		t.Errorf("expected IDENT 'name', got %v %q", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != COLON {
		t.Errorf("expected COLON, got %v", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != STRING || tok.Literal != "Alice" {
		t.Errorf("expected STRING 'Alice', got %v %q", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != RBRACE {
		t.Errorf("expected RBRACE, got %v", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != RPAREN {
		t.Errorf("expected RPAREN, got %v", tok.Type)
	}
}

func TestLexErrors(t *testing.T) {
	input := `@errors {name: "Required"}`
	l := NewLexer(input)

	tok := l.NextToken()
	if tok.Type != ERRORS || tok.Literal != "errors" {
		t.Errorf("expected ERRORS 'errors', got %v %q", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != LBRACE {
		t.Errorf("expected LBRACE, got %v", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != IDENT || tok.Literal != "name" {
		t.Errorf("expected IDENT 'name', got %v %q", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != COLON {
		t.Errorf("expected COLON, got %v", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != STRING || tok.Literal != "Required" {
		t.Errorf("expected STRING 'Required', got %v %q", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != RBRACE {
		t.Errorf("expected RBRACE, got %v", tok.Type)
	}
}

func TestLexDatetimes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"@2024-01-20", "2024-01-20"},
		{"@2024-01-20T10:30:00Z", "2024-01-20T10:30:00Z"},
		{"@2024-01-20T10:30:00+05:30", "2024-01-20T10:30:00+05:30"},
		{"@10:30:00", "10:30:00"},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		tok := l.NextToken()
		if tok.Type != DATETIME {
			t.Errorf("input %q: expected DATETIME, got %v", tt.input, tok.Type)
		}
		if tok.Literal != tt.expected {
			t.Errorf("input %q: expected literal %q, got %q", tt.input, tt.expected, tok.Literal)
		}
	}
}

func TestLexPaths(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"@/path/to/file", "/path/to/file"},
		{"@./relative/path", "./relative/path"},
		{"@../parent/path", "../parent/path"},
		{"@~/home/path", "~/home/path"},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		tok := l.NextToken()
		if tok.Type != PATH {
			t.Errorf("input %q: expected PATH, got %v (literal=%q)", tt.input, tok.Type, tok.Literal)
		}
		if tok.Literal != tt.expected {
			t.Errorf("input %q: expected literal %q, got %q", tt.input, tt.expected, tok.Literal)
		}
	}
}

func TestLexURLs(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"@https://example.com", "https://example.com"},
		{"@http://localhost:8080/api", "http://localhost:8080/api"},
		{"@https://api.example.com/v1/users?id=123", "https://api.example.com/v1/users?id=123"},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		tok := l.NextToken()
		if tok.Type != URL {
			t.Errorf("input %q: expected URL, got %v (literal=%q)", tt.input, tok.Type, tok.Literal)
		}
		if tok.Literal != tt.expected {
			t.Errorf("input %q: expected literal %q, got %q", tt.input, tt.expected, tok.Literal)
		}
	}
}

func TestLexComments(t *testing.T) {
	input := `// This is a comment
42
// Another comment
"hello"`

	l := NewLexer(input)

	tok := l.NextToken()
	if tok.Type != INT || tok.Literal != "42" {
		t.Errorf("expected INT 42, got %v %q", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != STRING || tok.Literal != "hello" {
		t.Errorf("expected STRING hello, got %v %q", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != EOF {
		t.Errorf("expected EOF, got %v", tok.Type)
	}
}

func TestLexLineNumbers(t *testing.T) {
	input := `{
    name: "Alice",
    age: 30
}`

	l := NewLexer(input)

	// {
	tok := l.NextToken()
	if tok.Line != 1 {
		t.Errorf("'{' should be on line 1, got %d", tok.Line)
	}

	// name
	tok = l.NextToken()
	if tok.Line != 2 {
		t.Errorf("'name' should be on line 2, got %d", tok.Line)
	}

	// :
	l.NextToken()
	// "Alice"
	l.NextToken()
	// ,
	l.NextToken()
	// age
	tok = l.NextToken()
	if tok.Line != 3 {
		t.Errorf("'age' should be on line 3, got %d", tok.Line)
	}
}

func TestLexTrailingComma(t *testing.T) {
	input := `[1, 2, 3,]`

	l := NewLexer(input)

	expected := []TokenType{LBRACKET, INT, COMMA, INT, COMMA, INT, COMMA, RBRACKET}
	for i, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp {
			t.Errorf("token %d: expected %v, got %v", i, exp, tok.Type)
		}
	}
}

func TestLexComplexNested(t *testing.T) {
	input := `{
    users: [
        @Person({name: "Alice", age: 30}),
        @Person({name: "Bob", age: 25})
    ],
    created: @2024-01-20T10:30:00Z
}`

	l := NewLexer(input)

	// Just verify it lexes without errors and produces expected structure
	tokenCount := 0
	for {
		tok := l.NextToken()
		if tok.Type == EOF {
			break
		}
		if tok.Type == ILLEGAL {
			t.Errorf("unexpected ILLEGAL token: %q at line %d, col %d",
				tok.Literal, tok.Line, tok.Column)
		}
		tokenCount++
	}

	if tokenCount < 30 {
		t.Errorf("expected many tokens, got %d", tokenCount)
	}
}

func TestLexUnicodeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello\u0020world"`, "hello world"},
		{`"\u4e2d\u6587"`, "中文"},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		tok := l.NextToken()
		if tok.Type != STRING {
			t.Errorf("input %q: expected STRING, got %v", tt.input, tok.Type)
		}
		if tok.Literal != tt.expected {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.expected, tok.Literal)
		}
	}
}
