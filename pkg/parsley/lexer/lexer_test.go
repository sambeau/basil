package lexer

import (
	"strings"
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `let five = 5;
let ten = 10;

let add = fn(x, y) {
  x + y;
};

let result = add(five, ten);
!-/*5;
5 < 10 > 5;

if (5 < 10) {
	return true;
} else {
	return false;
}

10 == 10;
10 != 9;
"foobar"
"foo bar"
`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{LET, "let"},
		{IDENT, "five"},
		{ASSIGN, "="},
		{INT, "5"},
		{SEMICOLON, ";"},
		{LET, "let"},
		{IDENT, "ten"},
		{ASSIGN, "="},
		{INT, "10"},
		{SEMICOLON, ";"},
		{LET, "let"},
		{IDENT, "add"},
		{ASSIGN, "="},
		{FUNCTION, "fn"},
		{LPAREN, "("},
		{IDENT, "x"},
		{COMMA, ","},
		{IDENT, "y"},
		{RPAREN, ")"},
		{LBRACE, "{"},
		{IDENT, "x"},
		{PLUS, "+"},
		{IDENT, "y"},
		{SEMICOLON, ";"},
		{RBRACE, "}"},
		{SEMICOLON, ";"},
		{LET, "let"},
		{IDENT, "result"},
		{ASSIGN, "="},
		{IDENT, "add"},
		{LPAREN, "("},
		{IDENT, "five"},
		{COMMA, ","},
		{IDENT, "ten"},
		{RPAREN, ")"},
		{SEMICOLON, ";"},
		{BANG, "!"},
		{MINUS, "-"},
		{SLASH, "/"},
		{ASTERISK, "*"},
		{INT, "5"},
		{SEMICOLON, ";"},
		{INT, "5"},
		{LT, "<"},
		{INT, "10"},
		{GT, ">"},
		{INT, "5"},
		{SEMICOLON, ";"},
		{IF, "if"},
		{LPAREN, "("},
		{INT, "5"},
		{LT, "<"},
		{INT, "10"},
		{RPAREN, ")"},
		{LBRACE, "{"},
		{RETURN, "return"},
		{TRUE, "true"},
		{SEMICOLON, ";"},
		{RBRACE, "}"},
		{ELSE, "else"},
		{LBRACE, "{"},
		{RETURN, "return"},
		{FALSE, "false"},
		{SEMICOLON, ";"},
		{RBRACE, "}"},
		{INT, "10"},
		{EQ, "=="},
		{INT, "10"},
		{SEMICOLON, ";"},
		{INT, "10"},
		{NOT_EQ, "!="},
		{INT, "9"},
		{SEMICOLON, ";"},
		{STRING, "foobar"},
		{STRING, "foo bar"},
		{EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLookupIdent(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"fn", FUNCTION},
		{"let", LET},
		{"true", TRUE},
		{"false", FALSE},
		{"if", IF},
		{"else", ELSE},
		{"return", RETURN},
		{"foobar", IDENT},
		{"foo", IDENT},
		{"bar", IDENT},
	}

	for _, tt := range tests {
		result := LookupIdent(tt.input)
		if result != tt.expected {
			t.Errorf("LookupIdent(%q) wrong. expected=%q, got=%q",
				tt.input, tt.expected, result)
		}
	}
}

func TestFloatTokens(t *testing.T) {
	input := `3.14159
	2.718
	sin(1.0)
	cos(0.5)
	`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{FLOAT, "3.14159"},
		{FLOAT, "2.718"},
		{IDENT, "sin"},
		{LPAREN, "("},
		{FLOAT, "1.0"},
		{RPAREN, ")"},
		{IDENT, "cos"},
		{LPAREN, "("},
		{FLOAT, "0.5"},
		{RPAREN, ")"},
		{EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestSQLTagBlocksInterpolation(t *testing.T) {
	// SQL tags should block @{} interpolation for safety
	// All parameters must come from attributes
	input := `<SQL>SELECT * FROM users WHERE id = @{id}</SQL>`

	l := New(input)

	// Skip TAG_START token
	tok := l.NextToken()
	if tok.Type != TAG_START {
		t.Fatalf("Expected TAG_START, got %s", tok.Type)
	}

	// Next should be TAG_TEXT for "SELECT * FROM users WHERE id = "
	tok = l.NextToken()
	if tok.Type != TAG_TEXT {
		t.Fatalf("Expected TAG_TEXT, got %s", tok.Type)
	}

	// Next should be ILLEGAL for the @{ interpolation attempt
	tok = l.NextToken()
	if tok.Type != ILLEGAL {
		t.Fatalf("Expected ILLEGAL token for @{} in SQL tag, got %s", tok.Type)
	}

	// Verify error message mentions interpolation and suggests attributes
	if !strings.Contains(tok.Literal, "interpolation") {
		t.Errorf("Error should mention 'interpolation', got: %s", tok.Literal)
	}
	if !strings.Contains(tok.Literal, "attributes") {
		t.Errorf("Error should suggest using attributes, got: %s", tok.Literal)
	}
}

func TestStyleTagAllowsInterpolation(t *testing.T) {
	// Style tags should still allow @{} interpolation
	input := `<style>.foo { color: @{color}; }</style>`

	l := New(input)

	// Skip TAG_START token
	tok := l.NextToken()
	if tok.Type != TAG_START {
		t.Fatalf("Expected TAG_START, got %s", tok.Type)
	}

	// Next should be TAG_TEXT for ".foo { color: "
	tok = l.NextToken()
	if tok.Type != TAG_TEXT {
		t.Fatalf("Expected TAG_TEXT, got %s", tok.Type)
	}

	// Next should be LBRACE for the @{ interpolation (allowed in style)
	tok = l.NextToken()
	if tok.Type != LBRACE {
		t.Fatalf("Expected LBRACE (interpolation start) in style tag, got %s", tok.Type)
	}
}
