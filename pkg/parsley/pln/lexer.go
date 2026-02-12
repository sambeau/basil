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
	MONEY    // $19.99, £50.00, EUR#100.00

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
	case MONEY:
		return "MONEY"
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
	case '$':
		tok = l.readMoneyLiteral()
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
		} else if l.ch == 0xC2 || l.ch == 0xA3 || l.ch == 0xA5 {
			// Check for UTF-8 encoded £ (C2 A3), € (E2 82 AC), or ¥ (C2 A5)
			if l.isUnicodeCurrencySymbol() {
				tok = l.readMoneyLiteral()
				return tok
			}
			tok.Type = ILLEGAL
			tok.Literal = string(l.ch)
		} else if l.ch == 0xE2 && l.peekChar() == 0x82 && l.peekAhead(2) == 0xAC {
			// € (E2 82 AC)
			tok = l.readMoneyLiteral()
			return tok
		} else if isLetter(l.ch) {
			// Check for CODE# money syntax (e.g., USD#12.34)
			if l.isCurrencyCodeStart() {
				tok = l.readMoneyLiteral()
				return tok
			}
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

// isUnicodeCurrencySymbol checks if current position is a Unicode currency symbol
func (l *Lexer) isUnicodeCurrencySymbol() bool {
	// £ is C2 A3 in UTF-8
	if l.ch == 0xC2 && l.peekChar() == 0xA3 {
		return true
	}
	// ¥ is C2 A5 in UTF-8
	if l.ch == 0xC2 && l.peekChar() == 0xA5 {
		return true
	}
	// € is E2 82 AC in UTF-8
	if l.ch == 0xE2 && l.peekChar() == 0x82 && l.peekAhead(2) == 0xAC {
		return true
	}
	return false
}

// isCurrencyCodeStart checks if current position starts a CODE# money literal
func (l *Lexer) isCurrencyCodeStart() bool {
	// Must be uppercase letter
	if l.ch < 'A' || l.ch > 'Z' {
		return false
	}

	// Look for pattern: 2-3 uppercase letters followed by #
	pos := 0
	for {
		ch := l.peekAhead(pos + 1)
		if ch >= 'A' && ch <= 'Z' {
			pos++
			if pos > 3 {
				return false
			}
			continue
		}
		if ch == '#' && pos >= 2 {
			return true
		}
		return false
	}
}

// CurrencyScales contains known currency decimal places from ISO 4217
var CurrencyScales = map[string]int8{
	"USD": 2, "EUR": 2, "GBP": 2, "JPY": 0, "CHF": 2,
	"CAD": 2, "AUD": 2, "CNY": 2, "HKD": 2, "SGD": 2,
	"KRW": 0, "INR": 2, "BRL": 2, "KWD": 3, "BHD": 3,
	"OMR": 3, "JOD": 3, "MXN": 2, "NZD": 2, "SEK": 2,
	"NOK": 2, "DKK": 2, "ZAR": 2, "RUB": 2, "PLN": 2,
	"THB": 2, "MYR": 2, "PHP": 2, "IDR": 2, "VND": 0,
	"CLP": 0, "COP": 2, "PEN": 2, "ARS": 2, "CZK": 2,
	"HUF": 2, "ILS": 2, "TRY": 2, "TWD": 2, "AED": 2,
	"SAR": 2, "QAR": 2, "EGP": 2, "PKR": 2, "NGN": 2,
}

// readMoneyLiteral reads a money literal and returns a MONEY token
// Handles: $12.34, £99.99, EUR#50.00, etc.
func (l *Lexer) readMoneyLiteral() Token {
	tok := Token{Line: l.line, Column: l.column}
	var currency string

	// Handle currency symbol or CODE# prefix
	switch {
	case l.ch == '$':
		l.readChar()
		currency = "USD"
	case l.ch == 0xC2 && l.peekChar() == 0xA3: // £
		l.readChar()
		l.readChar()
		currency = "GBP"
	case l.ch == 0xC2 && l.peekChar() == 0xA5: // ¥
		l.readChar()
		l.readChar()
		currency = "JPY"
	case l.ch == 0xE2 && l.peekChar() == 0x82 && l.peekAhead(2) == 0xAC: // €
		l.readChar()
		l.readChar()
		l.readChar()
		currency = "EUR"
	default:
		// Must be CODE# format (e.g., USD#, EUR#)
		currency = l.readCurrencyCode()
	}

	// Handle optional negative sign after currency symbol (or after # for CODE# format)
	negative := false
	if l.ch == '-' {
		negative = true
		l.readChar()
	}

	// Read the number (may also detect negative sign after # for CODE# format)
	numStr, literalScale, numNegative := l.readMoneyNumber()
	if numStr == "" {
		tok.Type = ILLEGAL
		tok.Literal = "currency symbol must be followed by a number"
		return tok
	}

	// Combine negative flags (either before or after #)
	if numNegative {
		negative = true
	}

	// Determine the scale
	scale := int8(2) // Default for unknown currencies
	if knownScale, ok := CurrencyScales[currency]; ok {
		scale = knownScale
		// Validate: currencies like JPY shouldn't have decimals
		if literalScale > 0 && knownScale == 0 {
			tok.Type = ILLEGAL
			tok.Literal = fmt.Sprintf("%s does not allow decimal places", currency)
			return tok
		}
		if literalScale > knownScale {
			tok.Type = ILLEGAL
			tok.Literal = fmt.Sprintf("%s allows max %d decimal places", currency, knownScale)
			return tok
		}
	}

	// Convert to integer amount (smallest unit)
	amount := parseMoneyAmount(numStr, literalScale, scale)
	if negative {
		amount = -amount
	}

	// Build the literal string: CODE#amount.decimals
	tok.Type = MONEY
	tok.Literal = buildMoneyLiteral(currency, amount, scale)
	return tok
}

// readCurrencyCode reads a 3-letter currency code followed by #
func (l *Lexer) readCurrencyCode() string {
	var code []byte
	for l.ch >= 'A' && l.ch <= 'Z' && len(code) < 3 {
		code = append(code, l.ch)
		l.readChar()
	}
	if l.ch == '#' {
		l.readChar() // consume the #
	}
	return string(code)
}

// readMoneyNumber reads the numeric part of a money literal
// Returns the number string, the number of decimal places, and whether it's negative
func (l *Lexer) readMoneyNumber() (string, int8, bool) {
	var sb strings.Builder
	var decimalPlaces int8 = -1

	// Handle negative sign that may appear after # in CODE# format
	negative := false
	if l.ch == '-' {
		negative = true
		l.readChar()
	}

	for isDigit(l.ch) || l.ch == '.' {
		if l.ch == '.' {
			if decimalPlaces >= 0 {
				break // Already saw a decimal point
			}
			decimalPlaces = 0
		} else if decimalPlaces >= 0 {
			decimalPlaces++
		}
		sb.WriteByte(l.ch)
		l.readChar()
	}

	if decimalPlaces < 0 {
		decimalPlaces = 0
	}

	return sb.String(), decimalPlaces, negative
}

// parseMoneyAmount converts a number string to an integer amount in smallest units
func parseMoneyAmount(numStr string, literalScale, targetScale int8) int64 {
	var result int64

	for _, ch := range numStr {
		if ch == '.' {
			continue
		}
		if ch >= '0' && ch <= '9' {
			result = result*10 + int64(ch-'0')
		}
	}

	// Scale up if literal has fewer decimal places than target
	for i := literalScale; i < targetScale; i++ {
		result *= 10
	}

	return result
}

// buildMoneyLiteral creates a display string for a money literal
func buildMoneyLiteral(currency string, amount int64, scale int8) string {
	if scale == 0 {
		return fmt.Sprintf("%s#%d", currency, amount)
	}

	negative := amount < 0
	if negative {
		amount = -amount
	}

	divisor := int64(1)
	for i := int8(0); i < scale; i++ {
		divisor *= 10
	}

	whole := amount / divisor
	frac := amount % divisor

	format := fmt.Sprintf("%%s#%%d.%%0%dd", scale)
	result := fmt.Sprintf(format, currency, whole, frac)

	if negative {
		return currency + "#-" + result[len(currency)+1:]
	}
	return result
}

// Unused but kept for potential future Unicode support
var _ = unicode.IsLetter
var _ = utf8.DecodeRuneInString
