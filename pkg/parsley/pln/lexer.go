// Package pln implements Parsley Literal Notation (PLN) - a safe data serialization format.
//
// PLN uses a subset of Parsley syntax to represent values without allowing code execution.
// It supports primitives, arrays, dictionaries, records with schemas, datetimes, and paths.
package pln

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TokenType represents different types of PLN tokens
type TokenType int

const (
	// Special tokens
	ILLEGAL TokenType = iota
	EOF

	// Literals
	INT      // 42
	FLOAT    // 3.14
	STRING   // "hello"
	TRUE     // true
	FALSE    // false
	NULL     // null
	IDENT    // fieldName, SchemaName
	DATETIME // 2024-01-20T10:30:00Z (after @)
	PATH     // /path/to/file (after @)
	URL      // https://example.com (after @)

	// Delimiters
	LBRACE   // {
	RBRACE   // }
	LBRACKET // [
	RBRACKET // ]
	LPAREN   // (
	RPAREN   // )
	COLON    // :
	COMMA    // ,
	AT       // @

	// Keywords (contextual)
	ERRORS // errors (after @)
)

// Token represents a single PLN token
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

// String returns a string representation of the token type
func (t TokenType) String() string {
	switch t {
	case ILLEGAL:
		return "ILLEGAL"
	case EOF:
		return "EOF"
	case INT:
		return "INT"
	case FLOAT:
		return "FLOAT"
	case STRING:
		return "STRING"
	case TRUE:
		return "TRUE"
	case FALSE:
		return "FALSE"
	case NULL:
		return "NULL"
	case IDENT:
		return "IDENT"
	case DATETIME:
		return "DATETIME"
	case PATH:
		return "PATH"
	case URL:
		return "URL"
	case LBRACE:
		return "LBRACE"
	case RBRACE:
		return "RBRACE"
	case LBRACKET:
		return "LBRACKET"
	case RBRACKET:
		return "RBRACKET"
	case LPAREN:
		return "LPAREN"
	case RPAREN:
		return "RPAREN"
	case COLON:
		return "COLON"
	case COMMA:
		return "COMMA"
	case AT:
		return "AT"
	case ERRORS:
		return "ERRORS"
	default:
		return fmt.Sprintf("TokenType(%d)", t)
	}
}

// Lexer tokenizes PLN input
type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int  // current line number (1-indexed)
	column       int  // current column number (1-indexed)
}

// NewLexer creates a new PLN lexer
func NewLexer(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0}
	l.readChar()
	return l
}

// readChar reads the next character and advances position
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++

	// Track line and column
	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

// peekChar returns the next character without advancing
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// peekAhead returns the character n positions ahead without advancing
func (l *Lexer) peekAhead(n int) byte {
	pos := l.readPosition + n - 1
	if pos >= len(l.input) {
		return 0
	}
	return l.input[pos]
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() Token {
	l.skipWhitespaceAndComments()

	tok := Token{Line: l.line, Column: l.column}

	switch l.ch {
	case '{':
		tok.Type = LBRACE
		tok.Literal = "{"
	case '}':
		tok.Type = RBRACE
		tok.Literal = "}"
	case '[':
		tok.Type = LBRACKET
		tok.Literal = "["
	case ']':
		tok.Type = RBRACKET
		tok.Literal = "]"
	case '(':
		tok.Type = LPAREN
		tok.Literal = "("
	case ')':
		tok.Type = RPAREN
		tok.Literal = ")"
	case ':':
		tok.Type = COLON
		tok.Literal = ":"
	case ',':
		tok.Type = COMMA
		tok.Literal = ","
	case '@':
		tok = l.readAtToken()
		return tok
	case '"':
		tok.Type = STRING
		tok.Literal = l.readString()
		return tok
	case 0:
		tok.Type = EOF
		tok.Literal = ""
	default:
		if isDigit(l.ch) || (l.ch == '-' && isDigit(l.peekChar())) {
			return l.readNumber()
		} else if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = lookupIdent(tok.Literal)
			return tok
		} else {
			tok.Type = ILLEGAL
			tok.Literal = string(l.ch)
		}
	}

	l.readChar()
	return tok
}

// skipWhitespaceAndComments skips whitespace and // comments
func (l *Lexer) skipWhitespaceAndComments() {
	for {
		// Skip whitespace
		for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
			l.readChar()
		}

		// Skip // comments
		if l.ch == '/' && l.peekChar() == '/' {
			l.skipLineComment()
			continue
		}

		break
	}
}

// skipLineComment skips a // comment until end of line
func (l *Lexer) skipLineComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

// readAtToken handles tokens starting with @
func (l *Lexer) readAtToken() Token {
	tok := Token{Line: l.line, Column: l.column}
	l.readChar() // consume '@'

	// Check what follows @
	if l.ch == 0 {
		tok.Type = AT
		tok.Literal = "@"
		return tok
	}

	// Check for URL (http:// or https://)
	if l.ch == 'h' && l.peekAhead(1) == 't' && l.peekAhead(2) == 't' && l.peekAhead(3) == 'p' {
		url := l.readURL()
		if url != "" {
			tok.Type = URL
			tok.Literal = url
			return tok
		}
	}

	// Check for path (/, ./, ../, ~/)
	if l.ch == '/' || l.ch == '~' || (l.ch == '.' && (l.peekChar() == '/' || l.peekChar() == '.')) {
		tok.Type = PATH
		tok.Literal = l.readPath()
		return tok
	}

	// Check for datetime (starts with digit)
	if isDigit(l.ch) {
		tok.Type = DATETIME
		tok.Literal = l.readDateTime()
		return tok
	}

	// Check for 'errors' keyword
	if l.ch == 'e' {
		start := l.position
		ident := l.readIdentifier()
		if ident == "errors" {
			tok.Type = ERRORS
			tok.Literal = ident
			return tok
		}
		// Not 'errors', rewind and return AT
		l.position = start
		l.readPosition = start + 1
		if start < len(l.input) {
			l.ch = l.input[start]
		}
	}

	// For all other identifiers after @, return AT token
	// The next call to NextToken will return the identifier
	tok.Type = AT
	tok.Literal = "@"
	return tok
}

// readString reads a double-quoted string with escape sequences
func (l *Lexer) readString() string {
	var sb strings.Builder
	l.readChar() // skip opening "

	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			case '/':
				sb.WriteByte('/')
			case 'u':
				// Unicode escape \uXXXX
				if l.readPosition+4 <= len(l.input) {
					hex := l.input[l.readPosition : l.readPosition+4]
					if r, err := parseHexRune(hex); err == nil {
						sb.WriteRune(r)
						l.readPosition += 4
						l.position = l.readPosition - 1
						l.readChar()
						continue
					}
				}
				sb.WriteByte(l.ch)
			default:
				sb.WriteByte(l.ch)
			}
		} else {
			sb.WriteByte(l.ch)
		}
		l.readChar()
	}

	l.readChar() // skip closing "
	return sb.String()
}

// parseHexRune parses a 4-character hex string into a rune
func parseHexRune(hex string) (rune, error) {
	var r rune
	for _, c := range hex {
		r <<= 4
		switch {
		case c >= '0' && c <= '9':
			r |= rune(c - '0')
		case c >= 'a' && c <= 'f':
			r |= rune(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			r |= rune(c - 'A' + 10)
		default:
			return 0, fmt.Errorf("invalid hex character: %c", c)
		}
	}
	return r, nil
}

// readNumber reads an integer or float
func (l *Lexer) readNumber() Token {
	tok := Token{Line: l.line, Column: l.column}
	var sb strings.Builder

	// Handle negative sign
	if l.ch == '-' {
		sb.WriteByte('-')
		l.readChar()
	}

	// Read integer part
	for isDigit(l.ch) {
		sb.WriteByte(l.ch)
		l.readChar()
	}

	// Check for decimal point
	if l.ch == '.' && isDigit(l.peekChar()) {
		sb.WriteByte('.')
		l.readChar()

		// Read fractional part
		for isDigit(l.ch) {
			sb.WriteByte(l.ch)
			l.readChar()
		}

		// Check for exponent
		if l.ch == 'e' || l.ch == 'E' {
			sb.WriteByte(l.ch)
			l.readChar()
			if l.ch == '+' || l.ch == '-' {
				sb.WriteByte(l.ch)
				l.readChar()
			}
			for isDigit(l.ch) {
				sb.WriteByte(l.ch)
				l.readChar()
			}
		}

		tok.Type = FLOAT
	} else {
		tok.Type = INT
	}

	tok.Literal = sb.String()
	return tok
}

// readIdentifier reads an identifier
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readDateTime reads a datetime literal after @
// Supports: 2024-01-20, 2024-01-20T10:30:00, 2024-01-20T10:30:00Z, 2024-01-20T10:30:00+05:30, 10:30:00
func (l *Lexer) readDateTime() string {
	var sb strings.Builder

	// Read date or time part
	for isDigit(l.ch) || l.ch == '-' || l.ch == ':' || l.ch == 'T' || l.ch == 'Z' || l.ch == '+' || l.ch == '.' {
		sb.WriteByte(l.ch)
		l.readChar()
	}

	return sb.String()
}

// readPath reads a path literal after @
func (l *Lexer) readPath() string {
	var sb strings.Builder

	for !isPathTerminator(l.ch) {
		sb.WriteByte(l.ch)
		l.readChar()
	}

	return sb.String()
}

// readURL reads a URL literal after @
func (l *Lexer) readURL() string {
	var sb strings.Builder

	// Check for http:// or https://
	start := l.position
	for l.ch != 0 && !isURLTerminator(l.ch) {
		sb.WriteByte(l.ch)
		l.readChar()
	}

	url := sb.String()
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}

	// Not a valid URL, rewind
	l.position = start
	l.readPosition = start + 1
	if start < len(l.input) {
		l.ch = l.input[start]
	}
	return ""
}

// lookupIdent returns the token type for an identifier
func lookupIdent(ident string) TokenType {
	switch ident {
	case "true":
		return TRUE
	case "false":
		return FALSE
	case "null":
		return NULL
	default:
		return IDENT
	}
}

// isLetter returns true if the character is a letter or underscore
func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

// isDigit returns true if the character is a digit
func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

// isPathTerminator returns true if the character terminates a path
func isPathTerminator(ch byte) bool {
	return ch == 0 || ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' ||
		ch == ')' || ch == ']' || ch == '}' || ch == ',' || ch == ':'
}

// isURLTerminator returns true if the character terminates a URL
func isURLTerminator(ch byte) bool {
	return ch == 0 || ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' ||
		ch == ')' || ch == ']' || ch == '}' || ch == ','
}

// Unused but kept for potential future Unicode support
var _ = unicode.IsLetter
var _ = utf8.DecodeRuneInString
