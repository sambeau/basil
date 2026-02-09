package lexer

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// TokenType represents different types of tokens
type TokenType int

const (
	// Special tokens
	ILLEGAL TokenType = iota
	EOF
	COMMENT // // single line comment

	// Identifiers and literals
	IDENT             // add, foobar, x, y, ...
	INT               // 1343456
	FLOAT             // 3.14159
	STRING            // "foobar"
	TEMPLATE          // `template ${expr}`
	RAW_TEMPLATE      // 'raw with @{expr}'
	REGEX             // /pattern/flags
	DATETIME_LITERAL  // @2024-12-25T14:30:00Z
	DATETIME_NOW      // @now
	TIME_NOW          // @timeNow
	DATE_NOW          // @dateNow, @today
	DURATION_LITERAL  // @2h30m, @7d, @1y6mo
	SQLITE_LITERAL    // @sqlite
	POSTGRES_LITERAL  // @postgres
	MYSQL_LITERAL     // @mysql
	SFTP_LITERAL      // @sftp
	SHELL_LITERAL     // @shell
	DB_LITERAL        // @DB
	SEARCH_LITERAL    // @SEARCH
	ENV_LITERAL       // @env
	ARGS_LITERAL      // @args
	PARAMS_LITERAL    // @params
	SCHEMA_LITERAL    // @schema
	TABLE_LITERAL     // @table
	QUERY_LITERAL     // @query
	INSERT_LITERAL    // @insert
	UPDATE_LITERAL    // @update
	DELETE_LITERAL    // @delete
	TRANSACTION_LIT   // @transaction
	PATH_LITERAL      // @/usr/local, @./config
	URL_LITERAL       // @https://example.com
	STDLIB_PATH       // @std/table, @std/string
	PATH_TEMPLATE     // @(./path/{expr}/file)
	URL_TEMPLATE      // @(https://api.com/{expr}/path)
	DATETIME_TEMPLATE // @(2024-{month}-{day}T{hour}:00:00)
	MONEY             // $12.34, £99.99, EUR#50.00
	TAG               // <tag prop="value" />
	TAG_START         // <tag> or <tag attr="value">
	TAG_END           // </tag>
	TAG_TEXT          // raw text content within tags

	// Operators
	ASSIGN    // =
	PLUS      // +
	MINUS     // -
	BANG      // !
	ASTERISK  // *
	SLASH     // /
	PERCENT   // %
	LT        // <
	GT        // >
	LTE       // <=
	GTE       // >=
	EQ        // ==
	NOT_EQ    // !=
	AND       // & or and
	OR        // | or or
	NULLISH   // ??
	QUESTION  // ?
	MATCH     // ~
	NOT_MATCH // !~

	// File I/O operators
	READ_FROM     // <==
	FETCH_FROM    // <=/=
	WRITE_TO      // ==>
	APPEND_TO     // ==>>
	REMOTE_WRITE  // =/=>
	REMOTE_APPEND // =/=>>

	// Database operators
	QUERY_ONE  // <=?=>
	QUERY_MANY // <=??=>
	EXECUTE    // <=!=>

	// Query DSL operators
	PIPE_WRITE           // |<
	RETURN_ONE           // ?->
	RETURN_MANY          // ??->
	RETURN_ONE_EXPLICIT  // ?!->
	RETURN_MANY_EXPLICIT // ??!->
	DSL_EXECUTE          // . (in query context) or •
	EXEC_COUNT           // .->
	ARROW_PULL           // <-
	GROUP_BY             // + by (contextual)

	// Process execution operator
	EXECUTE_WITH // <=#=>

	// Delimiters
	COMMA     // ,
	SEMICOLON // ;
	COLON     // :
	DOT       // .
	DOTDOTDOT // ... (spread/rest operator)
	LPAREN    // (
	RPAREN    // )
	LBRACE    // {
	RBRACE    // }
	LBRACKET  // [
	RBRACKET  // ]
	PLUSPLUS  // ++
	RANGE     // ..

	// Keywords
	FUNCTION // "fn"
	LET      // "let"
	FOR      // "for"
	IN       // "in"
	AS       // "as"
	TRUE     // "true"
	FALSE    // "false"
	IF       // "if"
	ELSE     // "else"
	RETURN   // "return"
	EXPORT   // "export"
	TRY      // "try"
	IMPORT   // "import"
	CHECK    // "check"
	STOP     // "stop"
	SKIP     // "skip"
	VIA      // "via" (for schema relations)
	IS       // "is" (for schema checking)
	COMPUTED // "computed" (for computed exports)
)

// Token represents a single token
type Token struct {
	Type             TokenType
	Literal          string
	Line             int
	Column           int
	BlankLinesBefore int      // Number of blank lines before this token (for formatting)
	LeadingComments  []string // Comments before this token (for formatting)
	TrailingComment  string   // Comment on same line after this token (for formatting)
}

// String returns a string representation of the token
func (t Token) String() string {
	return fmt.Sprintf("{Type: %s, Literal: %s, Line: %d, Column: %d}",
		t.Type.String(), t.Literal, t.Line, t.Column)
}

// String returns a string representation of the token type
func (tt TokenType) String() string {
	switch tt {
	case ILLEGAL:
		return "ILLEGAL"
	case EOF:
		return "EOF"
	case COMMENT:
		return "COMMENT"
	case IDENT:
		return "IDENT"
	case INT:
		return "INT"
	case FLOAT:
		return "FLOAT"
	case STRING:
		return "STRING"
	case TEMPLATE:
		return "TEMPLATE"
	case RAW_TEMPLATE:
		return "RAW_TEMPLATE"
	case REGEX:
		return "REGEX"
	case DATETIME_LITERAL:
		return "DATETIME_LITERAL"
	case DATETIME_NOW:
		return "DATETIME_NOW"
	case TIME_NOW:
		return "TIME_NOW"
	case DATE_NOW:
		return "DATE_NOW"
	case DURATION_LITERAL:
		return "DURATION_LITERAL"
	case SQLITE_LITERAL:
		return "SQLITE_LITERAL"
	case POSTGRES_LITERAL:
		return "POSTGRES_LITERAL"
	case MYSQL_LITERAL:
		return "MYSQL_LITERAL"
	case SFTP_LITERAL:
		return "SFTP_LITERAL"
	case SHELL_LITERAL:
		return "SHELL_LITERAL"
	case DB_LITERAL:
		return "DB_LITERAL"
	case ENV_LITERAL:
		return "ENV_LITERAL"
	case ARGS_LITERAL:
		return "ARGS_LITERAL"
	case PARAMS_LITERAL:
		return "PARAMS_LITERAL"
	case PATH_LITERAL:
		return "PATH_LITERAL"
	case URL_LITERAL:
		return "URL_LITERAL"
	case STDLIB_PATH:
		return "STDLIB_PATH"
	case PATH_TEMPLATE:
		return "PATH_TEMPLATE"
	case URL_TEMPLATE:
		return "URL_TEMPLATE"
	case DATETIME_TEMPLATE:
		return "DATETIME_TEMPLATE"
	case MONEY:
		return "MONEY"
	case TAG:
		return "TAG"
	case TAG_START:
		return "TAG_START"
	case TAG_END:
		return "TAG_END"
	case TAG_TEXT:
		return "TAG_TEXT"
	case ASSIGN:
		return "ASSIGN"
	case PLUS:
		return "PLUS"
	case MINUS:
		return "MINUS"
	case BANG:
		return "BANG"
	case ASTERISK:
		return "ASTERISK"
	case SLASH:
		return "SLASH"
	case PERCENT:
		return "PERCENT"
	case LT:
		return "LT"
	case GT:
		return "GT"
	case LTE:
		return "LTE"
	case GTE:
		return "GTE"
	case EQ:
		return "EQ"
	case NOT_EQ:
		return "NOT_EQ"
	case AND:
		return "AND"
	case OR:
		return "OR"
	case NULLISH:
		return "NULLISH"
	case QUESTION:
		return "QUESTION"
	case MATCH:
		return "MATCH"
	case NOT_MATCH:
		return "NOT_MATCH"
	case READ_FROM:
		return "READ_FROM"
	case FETCH_FROM:
		return "FETCH_FROM"
	case WRITE_TO:
		return "WRITE_TO"
	case APPEND_TO:
		return "APPEND_TO"
	case REMOTE_WRITE:
		return "REMOTE_WRITE"
	case REMOTE_APPEND:
		return "REMOTE_APPEND"
	case QUERY_ONE:
		return "QUERY_ONE"
	case QUERY_MANY:
		return "QUERY_MANY"
	case EXECUTE:
		return "EXECUTE"
	case PIPE_WRITE:
		return "PIPE_WRITE"
	case RETURN_ONE:
		return "RETURN_ONE"
	case RETURN_MANY:
		return "RETURN_MANY"
	case RETURN_ONE_EXPLICIT:
		return "RETURN_ONE_EXPLICIT"
	case RETURN_MANY_EXPLICIT:
		return "RETURN_MANY_EXPLICIT"
	case DSL_EXECUTE:
		return "DSL_EXECUTE"
	case EXEC_COUNT:
		return "EXEC_COUNT"
	case ARROW_PULL:
		return "ARROW_PULL"
	case GROUP_BY:
		return "GROUP_BY"
	case EXECUTE_WITH:
		return "EXECUTE_WITH"
	case COMMA:
		return "COMMA"
	case SEMICOLON:
		return "SEMICOLON"
	case COLON:
		return "COLON"
	case DOT:
		return "DOT"
	case DOTDOTDOT:
		return "DOTDOTDOT"
	case LPAREN:
		return "LPAREN"
	case RPAREN:
		return "RPAREN"
	case LBRACE:
		return "LBRACE"
	case RBRACE:
		return "RBRACE"
	case LBRACKET:
		return "LBRACKET"
	case RBRACKET:
		return "RBRACKET"
	case PLUSPLUS:
		return "PLUSPLUS"
	case RANGE:
		return "RANGE"
	case FUNCTION:
		return "FUNCTION"
	case LET:
		return "LET"
	case FOR:
		return "FOR"
	case IN:
		return "IN"
	case AS:
		return "AS"
	case TRUE:
		return "TRUE"
	case FALSE:
		return "FALSE"
	case IF:
		return "IF"
	case ELSE:
		return "ELSE"
	case RETURN:
		return "RETURN"
	case EXPORT:
		return "EXPORT"
	case TRY:
		return "TRY"
	case IMPORT:
		return "IMPORT"
	case CHECK:
		return "CHECK"
	case STOP:
		return "STOP"
	case SKIP:
		return "SKIP"
	case VIA:
		return "VIA"
	case IS:
		return "IS"
	case COMPUTED:
		return "COMPUTED"
	case SCHEMA_LITERAL:
		return "SCHEMA_LITERAL"
	case TABLE_LITERAL:
		return "TABLE_LITERAL"
	case QUERY_LITERAL:
		return "QUERY_LITERAL"
	case INSERT_LITERAL:
		return "INSERT_LITERAL"
	case UPDATE_LITERAL:
		return "UPDATE_LITERAL"
	case DELETE_LITERAL:
		return "DELETE_LITERAL"
	case TRANSACTION_LIT:
		return "TRANSACTION_LIT"
	default:
		return "UNKNOWN"
	}
}

// Keywords map for identifying language keywords
var keywords = map[string]TokenType{
	"fn":       FUNCTION,
	"function": FUNCTION, // alias for JS familiarity
	"let":      LET,
	"for":      FOR,
	"in":       IN,
	"as":       AS,
	"true":     TRUE,
	"false":    FALSE,
	"if":       IF,
	"else":     ELSE,
	"return":   RETURN,
	"export":   EXPORT,
	"and":      AND,
	"or":       OR,
	"not":      BANG,
	"try":      TRY,
	"import":   IMPORT,
	"check":    CHECK,
	"stop":     STOP,
	"skip":     SKIP,
	"via":      VIA,
	"is":       IS,
	"computed": COMPUTED,
}

// LookupIdent checks if an identifier is a keyword
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

// Lexer represents the lexical analyzer
type Lexer struct {
	filename               string
	input                  string
	position               int       // current position in input (points to current char)
	readPosition           int       // current reading position in input (after current char)
	ch                     byte      // current char under examination (first byte)
	chRune                 rune      // current character as a rune (for Unicode support)
	chSize                 int       // byte size of current character (1 for ASCII, 1-4 for UTF-8)
	line                   int       // current line number
	column                 int       // current column number
	inTagContent           bool      // whether we're currently lexing tag content
	tagDepth               int       // nesting depth of tags (for proper TAG_END matching)
	lastTokenType          TokenType // last token type for regex context detection
	inRawTextTag           string    // non-empty when inside <style> or <script> - stores tag name (for @{} mode)
	inRawTextInterpolate   bool      // true when inside @{} interpolation within a raw text tag
	pendingComments        []string  // comments collected before the next token
	pendingBlankLines      int       // blank lines counted before the next token
	pendingTrailingComment string    // trailing comment from previous line (for the PREVIOUS token's statement)

}

// truncate returns the first n characters of a string, adding "..." if truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// New creates a new lexer instance
func New(input string) *Lexer {
	l := &Lexer{
		filename: "<input>",
		input:    input,
		line:     1,
		column:   0,
	}
	l.readChar()
	return l
}

// NewWithFilename creates a new lexer instance with a specific filename
func NewWithFilename(input string, filename string) *Lexer {
	l := &Lexer{
		filename: filename,
		input:    input,
		line:     1,
		column:   0,
	}
	l.readChar()
	return l
}

// LexerState holds the state of a lexer for save/restore
type LexerState struct {
	position               int
	readPosition           int
	ch                     byte
	chRune                 rune
	chSize                 int
	line                   int
	column                 int
	inTagContent           bool
	tagDepth               int
	lastTokenType          TokenType
	inRawTextTag           string
	pendingComments        []string
	pendingBlankLines      int
	pendingTrailingComment string
}

// SaveState saves the current lexer state for potential restoration
func (l *Lexer) SaveState() LexerState {
	// Copy the pending comments slice
	commentsCopy := make([]string, len(l.pendingComments))
	copy(commentsCopy, l.pendingComments)
	return LexerState{
		position:               l.position,
		readPosition:           l.readPosition,
		ch:                     l.ch,
		chRune:                 l.chRune,
		chSize:                 l.chSize,
		line:                   l.line,
		column:                 l.column,
		inTagContent:           l.inTagContent,
		tagDepth:               l.tagDepth,
		lastTokenType:          l.lastTokenType,
		inRawTextTag:           l.inRawTextTag,
		pendingComments:        commentsCopy,
		pendingBlankLines:      l.pendingBlankLines,
		pendingTrailingComment: l.pendingTrailingComment,
	}
}

// RestoreState restores the lexer to a previously saved state
func (l *Lexer) RestoreState(state LexerState) {
	l.position = state.position
	l.readPosition = state.readPosition
	l.ch = state.ch
	l.chRune = state.chRune
	l.chSize = state.chSize
	l.line = state.line
	l.column = state.column
	l.inTagContent = state.inTagContent
	l.tagDepth = state.tagDepth
	l.lastTokenType = state.lastTokenType
	l.inRawTextTag = state.inRawTextTag
	l.pendingComments = state.pendingComments
	l.pendingBlankLines = state.pendingBlankLines
	l.pendingTrailingComment = state.pendingTrailingComment
}

// PeekToken returns the next token without consuming it
// This is used for lookahead when the parser needs to see beyond the current peek token
func (l *Lexer) PeekToken() Token {
	state := l.SaveState()
	tok := l.NextToken()
	l.RestoreState(state)
	return tok
}

// readChar reads the next character and advances position.
// Uses a hybrid approach: ASCII fast-path for single-byte characters,
// UTF-8 decoding for multi-byte characters (to support Unicode identifiers).
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // ASCII NUL character represents EOF
		l.chRune = 0
		l.chSize = 0
		l.position = l.readPosition
		// Don't increment readPosition past end
		return
	}

	b := l.input[l.readPosition]

	// ASCII fast-path: single-byte characters (most common case)
	if b < utf8.RuneSelf {
		l.ch = b
		l.chRune = rune(b)
		l.chSize = 1
		l.position = l.readPosition
		l.readPosition++

		if l.ch == '\n' {
			l.line++
			l.column = 0
		} else {
			l.column++
		}
		return
	}

	// Non-ASCII: decode the full UTF-8 rune
	r, size := utf8.DecodeRuneInString(l.input[l.readPosition:])
	l.ch = b // Keep first byte for backward compatibility
	l.chRune = r
	l.chSize = size
	l.position = l.readPosition
	l.readPosition += size

	l.column++
}

// appendCurrentChar appends the current character (all bytes for multi-byte UTF-8) to the given slice.
func (l *Lexer) appendCurrentChar(result []byte) []byte {
	if l.chSize == 1 {
		return append(result, l.ch)
	}
	// Multi-byte character: get the bytes from input
	return append(result, l.input[l.position:l.position+l.chSize]...)
}

// peekChar returns the next character without advancing position
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// peekCharRune returns the next character as a rune without advancing position.
// This properly handles multi-byte UTF-8 characters.
func (l *Lexer) peekCharRune() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	b := l.input[l.readPosition]
	if b < utf8.RuneSelf {
		return rune(b)
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.readPosition:])
	return r
}

// peekCharN returns the character n positions ahead without advancing position
func (l *Lexer) peekCharN(n int) byte {
	pos := l.readPosition + n - 1
	if pos >= len(l.input) {
		return 0
	}
	return l.input[pos]
}

// NextToken scans the input and returns the next token
func (l *Lexer) NextToken() Token {
	var tok Token

	// Special handling when inside tag content
	if l.inTagContent {
		tok = l.nextTagContentToken()
		return l.attachPendingTrivia(tok)
	}

	// Collect whitespace, blank lines, and comments before the next token
	l.collectTrivia()

	switch l.ch {
	case '=':
		if l.peekChar() == '/' && l.peekCharN(2) == '=' && l.peekCharN(3) == '>' {
			// =/=> (remote write) and =/=>> (remote append)
			line := l.line
			col := l.column
			l.readChar() // consume '/'
			l.readChar() // consume '='
			l.readChar() // consume '>'
			if l.peekChar() == '>' {
				l.readChar() // consume second '>'
				tok = Token{Type: REMOTE_APPEND, Literal: "=/=>>", Line: line, Column: col}
			} else {
				tok = Token{Type: REMOTE_WRITE, Literal: "=/=>", Line: line, Column: col}
			}
		} else if l.peekChar() == '=' {
			ch := l.ch
			line := l.line
			col := l.column
			l.readChar() // consume second '='
			if l.peekChar() == '>' {
				l.readChar() // consume '>'
				if l.peekChar() == '>' {
					l.readChar() // consume second '>'
					tok = Token{Type: APPEND_TO, Literal: "==>>", Line: line, Column: col}
				} else {
					tok = Token{Type: WRITE_TO, Literal: "==>", Line: line, Column: col}
				}
			} else {
				tok = Token{Type: EQ, Literal: string(ch) + string(l.ch), Line: line, Column: col}
			}
		} else {
			tok = newToken(ASSIGN, l.ch, l.line, l.column)
		}
	case '+':
		if l.peekChar() == '+' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: PLUSPLUS, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = newToken(PLUS, l.ch, l.line, l.column)
		}
	case '-':
		tok = newToken(MINUS, l.ch, l.line, l.column)
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: NOT_EQ, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else if l.peekChar() == '~' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: NOT_MATCH, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = newToken(BANG, l.ch, l.line, l.column)
		}
	case '~':
		tok = newToken(MATCH, l.ch, l.line, l.column)
	case '/':
		if l.peekChar() == '/' {
			// This shouldn't happen since collectTrivia handles comments,
			// but if we somehow get here, capture the comment and recurse
			l.skipAndCaptureComment()
			return l.NextToken()
		} else if l.shouldTreatAsRegex(l.lastTokenType) {
			// This is a regex literal
			line := l.line
			column := l.column
			pattern, flags := l.readRegex()
			tok.Type = REGEX
			tok.Literal = "/" + pattern + "/" + flags
			tok.Line = line
			tok.Column = column
			l.lastTokenType = tok.Type
			return l.attachPendingTrivia(tok)
		}
		tok = newToken(SLASH, l.ch, l.line, l.column)
	case '%':
		tok = newToken(PERCENT, l.ch, l.line, l.column)
	case '*':
		tok = newToken(ASTERISK, l.ch, l.line, l.column)
	case '<':
		if l.peekChar() == '=' && l.peekCharN(2) == '?' && l.peekCharN(3) == '?' && l.peekCharN(4) == '=' && l.peekCharN(5) == '>' {
			// <=??=> (query many from database)
			line := l.line
			col := l.column
			l.readChar() // consume '='
			l.readChar() // consume first '?'
			l.readChar() // consume second '?'
			l.readChar() // consume '='
			l.readChar() // consume '>'
			tok = Token{Type: QUERY_MANY, Literal: "<=??=>", Line: line, Column: col}
		} else if l.peekChar() == '=' && l.peekCharN(2) == '?' && l.peekCharN(3) == '=' && l.peekCharN(4) == '>' {
			// <=?=> (query one from database)
			line := l.line
			col := l.column
			l.readChar() // consume '='
			l.readChar() // consume '?'
			l.readChar() // consume '='
			l.readChar() // consume '>'
			tok = Token{Type: QUERY_ONE, Literal: "<=?=>", Line: line, Column: col}
		} else if l.peekChar() == '=' && l.peekCharN(2) == '!' && l.peekCharN(3) == '=' && l.peekCharN(4) == '>' {
			// <=!=> (execute database mutation)
			line := l.line
			col := l.column
			l.readChar() // consume '='
			l.readChar() // consume '!'
			l.readChar() // consume '='
			l.readChar() // consume '>'
			tok = Token{Type: EXECUTE, Literal: "<=!=>", Line: line, Column: col}
		} else if l.peekChar() == '=' && l.peekCharN(2) == '#' && l.peekCharN(3) == '=' && l.peekCharN(4) == '>' {
			// <=#=> (execute command with input)
			line := l.line
			col := l.column
			l.readChar() // consume '='
			l.readChar() // consume '#'
			l.readChar() // consume '='
			l.readChar() // consume '>'
			tok = Token{Type: EXECUTE_WITH, Literal: "<=#=>", Line: line, Column: col}
		} else if l.peekChar() == '=' && l.peekCharN(2) == '/' && l.peekCharN(3) == '=' {
			// <=/= (fetch from URL)
			line := l.line
			col := l.column
			l.readChar() // consume '='
			l.readChar() // consume '/'
			l.readChar() // consume '='
			tok = Token{Type: FETCH_FROM, Literal: "<=/=", Line: line, Column: col}
		} else if l.peekChar() == '=' && l.peekCharN(2) == '=' {
			// <== (read from file)
			line := l.line
			col := l.column
			l.readChar() // consume first '='
			l.readChar() // consume second '='
			tok = Token{Type: READ_FROM, Literal: "<==", Line: line, Column: col}
		} else if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: LTE, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else if l.peekChar() == '?' {
			// XML processing instruction <?xml ... ?> - pass through as string
			line := l.line
			column := l.column
			content := l.readProcessingInstruction()
			tok.Type = STRING
			tok.Literal = content
			tok.Line = line
			tok.Column = column
			l.lastTokenType = tok.Type
			return l.attachPendingTrivia(tok)
		} else if l.peekChar() == '!' {
			// Could be XML comment <!-- -->, CDATA <![CDATA[]]>, or DOCTYPE <!DOCTYPE>
			if l.peekCharN(2) == '-' && l.peekCharN(3) == '-' {
				// XML comment - skip it and get next token
				l.skipXMLComment()
				return l.NextToken()
			} else if l.peekCharN(2) == '[' && l.peekCharN(3) == 'C' {
				// CDATA section - return as string
				line := l.line
				column := l.column
				content, ok := l.readCDATA()
				if ok {
					tok.Type = STRING
					tok.Literal = content
					tok.Line = line
					tok.Column = column
					l.lastTokenType = tok.Type
					return l.attachPendingTrivia(tok)
				}
			} else if l.peekCharN(2) == 'D' || l.peekCharN(2) == 'd' {
				// DOCTYPE declaration - pass through as string
				line := l.line
				column := l.column
				content := l.readDoctype()
				tok.Type = STRING
				tok.Literal = content
				tok.Line = line
				tok.Column = column
				l.lastTokenType = tok.Type
				return l.attachPendingTrivia(tok)
			}
			// Not a comment, CDATA, or DOCTYPE - treat < as less-than
			tok = newToken(LT, l.ch, l.line, l.column)
		} else if l.peekChar() == '/' {
			// This is a closing tag </tag>
			line := l.line
			column := l.column
			tok.Type = TAG_END
			tok.Literal = l.readTagEnd()
			tok.Line = line
			tok.Column = column
			// Decrement tag depth
			if l.tagDepth > 0 {
				l.tagDepth--
			}
			l.lastTokenType = tok.Type
			return l.attachPendingTrivia(tok)
		} else if isLetter(l.peekChar()) || l.peekChar() == '>' || isLetterRune(l.peekCharRune()) {
			// Could be a tag start <tag> or singleton <tag /> or Unicode tag <Π/>
			line := l.line
			column := l.column
			tagContent, isSingleton := l.readTagStartOrSingleton()
			if isSingleton {
				tok.Type = TAG
			} else {
				tok.Type = TAG_START
				// Track tag depth but DON'T enter tag content mode
				// This allows code (not raw text) inside tags
				l.tagDepth++
				// Check if this is a raw text tag (style or script)
				// For these, we DO need special handling
				tagName := extractTagName(tagContent)
				if tagName == "style" || tagName == "script" {
					l.inRawTextTag = tagName
					l.inTagContent = true // Only for raw text tags
				}
			}
			tok.Literal = tagContent
			tok.Line = line
			tok.Column = column
			l.lastTokenType = tok.Type
			return l.attachPendingTrivia(tok)
		} else if l.peekChar() == '-' {
			// <- (arrow pull for DSL subqueries)
			line := l.line
			col := l.column
			l.readChar() // consume '-'
			tok = Token{Type: ARROW_PULL, Literal: "<-", Line: line, Column: col}
		} else {
			tok = newToken(LT, l.ch, l.line, l.column)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: GTE, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = newToken(GT, l.ch, l.line, l.column)
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			line := l.line
			col := l.column
			l.readChar() // consume second '&'
			tok = Token{Type: AND, Literal: string(ch) + string(l.ch), Line: line, Column: col}
		} else {
			tok = newToken(AND, l.ch, l.line, l.column)
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			line := l.line
			col := l.column
			l.readChar() // consume second '|'
			tok = Token{Type: OR, Literal: string(ch) + string(l.ch), Line: line, Column: col}
		} else if l.peekChar() == '<' {
			// |< (pipe write for DSL)
			line := l.line
			col := l.column
			l.readChar() // consume '<'
			tok = Token{Type: PIPE_WRITE, Literal: "|<", Line: line, Column: col}
		} else {
			// Single | is also OR (shorthand)
			tok = newToken(OR, l.ch, l.line, l.column)
		}
	case '?':
		if l.peekChar() == '?' {
			// Could be ?? (nullish), ??-> (return many), or ??!-> (explicit return many)
			line := l.line
			col := l.column
			l.readChar() // consume second '?'
			if l.peekChar() == '!' && l.peekCharN(2) == '-' && l.peekCharN(3) == '>' {
				// ??!-> (explicit return many)
				l.readChar() // consume '!'
				l.readChar() // consume '-'
				l.readChar() // consume '>'
				tok = Token{Type: RETURN_MANY_EXPLICIT, Literal: "??!->", Line: line, Column: col}
			} else if l.peekChar() == '-' && l.peekCharN(2) == '>' {
				// ??-> (return many)
				l.readChar() // consume '-'
				l.readChar() // consume '>'
				tok = Token{Type: RETURN_MANY, Literal: "??->", Line: line, Column: col}
			} else {
				tok = Token{Type: NULLISH, Literal: "??", Line: line, Column: col}
			}
		} else if l.peekChar() == '!' && l.peekCharN(2) == '-' && l.peekCharN(3) == '>' {
			// ?!-> (explicit return one)
			line := l.line
			col := l.column
			l.readChar() // consume '!'
			l.readChar() // consume '-'
			l.readChar() // consume '>'
			tok = Token{Type: RETURN_ONE_EXPLICIT, Literal: "?!->", Line: line, Column: col}
		} else if l.peekChar() == '-' && l.peekCharN(2) == '>' {
			// ?-> (return one)
			line := l.line
			col := l.column
			l.readChar() // consume '-'
			l.readChar() // consume '>'
			tok = Token{Type: RETURN_ONE, Literal: "?->", Line: line, Column: col}
		} else {
			tok = newToken(QUESTION, l.ch, l.line, l.column)
		}
	case ';':
		tok = newToken(SEMICOLON, l.ch, l.line, l.column)
	case ',':
		tok = newToken(COMMA, l.ch, l.line, l.column)
	case ':':
		tok = newToken(COLON, l.ch, l.line, l.column)
	case '.':
		// Check for "..." (spread/rest operator), ".." (range), or ".->" (exec count)
		if l.peekChar() == '-' && l.peekCharN(2) == '>' {
			// .-> (execute and return count)
			line := l.line
			col := l.column
			l.readChar() // consume '-'
			l.readChar() // consume '>'
			tok = Token{Type: EXEC_COUNT, Literal: ".->", Line: line, Column: col}
		} else if l.peekChar() == '.' {
			if l.readPosition+1 < len(l.input) && l.input[l.readPosition+1] == '.' {
				// Three dots: ...
				line := l.line
				col := l.column
				l.readChar() // consume second '.'
				l.readChar() // consume third '.'
				tok = Token{Type: DOTDOTDOT, Literal: "...", Line: line, Column: col}
			} else {
				// Two dots: ..
				line := l.line
				col := l.column
				l.readChar() // consume second '.'
				tok = Token{Type: RANGE, Literal: "..", Line: line, Column: col}
			}
		} else {
			tok = newToken(DOT, l.ch, l.line, l.column)
		}
	case '[':
		tok = newToken(LBRACKET, l.ch, l.line, l.column)
	case ']':
		tok = newToken(RBRACKET, l.ch, l.line, l.column)
	case '(':
		tok = newToken(LPAREN, l.ch, l.line, l.column)
	case ')':
		tok = newToken(RPAREN, l.ch, l.line, l.column)
	case '{':
		tok = newToken(LBRACE, l.ch, l.line, l.column)
	case '}':
		tok = newToken(RBRACE, l.ch, l.line, l.column)
		// If we were in a raw text tag interpolation (@{}), re-enter tag content mode
		if l.inRawTextInterpolate {
			l.inTagContent = true
			l.inRawTextInterpolate = false
		}
	case '"':
		line := l.line
		column := l.column
		str, terminated := l.readString()
		if !terminated {
			tok.Type = ILLEGAL
			tok.Literal = fmt.Sprintf("Unterminated string starting with \"%s\"", truncate(str, 20))
		} else {
			tok.Type = STRING
			tok.Literal = str
		}
		tok.Line = line
		tok.Column = column
	case '\'':
		line := l.line
		column := l.column
		str, terminated, hasInterpolation := l.readRawString()
		if !terminated {
			tok.Type = ILLEGAL
			tok.Literal = fmt.Sprintf("Unterminated raw string starting with '%s'", truncate(str, 20))
		} else if hasInterpolation {
			tok.Type = RAW_TEMPLATE
			tok.Literal = str
		} else {
			tok.Type = STRING
			tok.Literal = str
		}
		tok.Line = line
		tok.Column = column
	case '`':
		line := l.line
		column := l.column
		tok.Type = TEMPLATE
		tok.Literal = l.readTemplate()
		tok.Line = line
		tok.Column = column
	case '@':
		line := l.line
		column := l.column
		// Peek ahead to determine the literal type
		literalType := l.detectAtLiteralType()
		switch literalType {
		case DATETIME_NOW:
			tok.Type = DATETIME_NOW
			tok.Literal = l.readNowLiteral("now")
		case TIME_NOW:
			tok.Type = TIME_NOW
			tok.Literal = l.readNowLiteral("timeNow")
		case DATE_NOW:
			tok.Type = DATE_NOW
			tok.Literal = l.readNowLiteral(l.detectDateNowKeyword())
		case DATETIME_LITERAL:
			tok.Type = DATETIME_LITERAL
			tok.Literal = l.readDatetimeLiteral()
		case DURATION_LITERAL:
			tok.Type = DURATION_LITERAL
			tok.Literal = l.readDurationLiteral()
		case PATH_LITERAL:
			tok.Type = PATH_LITERAL
			tok.Literal = l.readPathLiteral()
		case URL_LITERAL:
			tok.Type = URL_LITERAL
			tok.Literal = l.readUrlLiteral()
		case STDLIB_PATH:
			tok.Type = STDLIB_PATH
			tok.Literal = l.readStdlibPath()
		case PATH_TEMPLATE:
			tok.Type = PATH_TEMPLATE
			tok.Literal = l.readPathTemplate()
		case URL_TEMPLATE:
			tok.Type = URL_TEMPLATE
			tok.Literal = l.readUrlTemplate()
		case DATETIME_TEMPLATE:
			tok.Type = DATETIME_TEMPLATE
			tok.Literal = l.readDatetimeTemplate()
		case SQLITE_LITERAL:
			tok.Type = SQLITE_LITERAL
			tok.Literal = l.readConnectionLiteral("sqlite")
		case POSTGRES_LITERAL:
			tok.Type = POSTGRES_LITERAL
			tok.Literal = l.readConnectionLiteral("postgres")
		case MYSQL_LITERAL:
			tok.Type = MYSQL_LITERAL
			tok.Literal = l.readConnectionLiteral("mysql")
		case SFTP_LITERAL:
			tok.Type = SFTP_LITERAL
			tok.Literal = l.readConnectionLiteral("sftp")
		case SHELL_LITERAL:
			tok.Type = SHELL_LITERAL
			tok.Literal = l.readConnectionLiteral("shell")
		case DB_LITERAL:
			tok.Type = DB_LITERAL
			tok.Literal = l.readConnectionLiteral("DB")
		case SEARCH_LITERAL:
			tok.Type = SEARCH_LITERAL
			tok.Literal = l.readConnectionLiteral("SEARCH")
		case ENV_LITERAL:
			tok.Type = ENV_LITERAL
			tok.Literal = "@env"
			l.readChar()  // skip @
			for range 3 { // consume "env"
				l.readChar()
			}
		case ARGS_LITERAL:
			tok.Type = ARGS_LITERAL
			tok.Literal = "@args"
			l.readChar()  // skip @
			for range 4 { // consume "args"
				l.readChar()
			}
		case PARAMS_LITERAL:
			tok.Type = PARAMS_LITERAL
			tok.Literal = "@params"
			l.readChar()  // skip @
			for range 6 { // consume "params"
				l.readChar()
			}
		case SCHEMA_LITERAL:
			tok.Type = SCHEMA_LITERAL
			tok.Literal = l.readDSLKeyword("schema")
		case TABLE_LITERAL:
			tok.Type = TABLE_LITERAL
			tok.Literal = l.readDSLKeyword("table")
		case QUERY_LITERAL:
			tok.Type = QUERY_LITERAL
			tok.Literal = l.readDSLKeyword("query")
		case INSERT_LITERAL:
			tok.Type = INSERT_LITERAL
			tok.Literal = l.readDSLKeyword("insert")
		case UPDATE_LITERAL:
			tok.Type = UPDATE_LITERAL
			tok.Literal = l.readDSLKeyword("update")
		case DELETE_LITERAL:
			tok.Type = DELETE_LITERAL
			tok.Literal = l.readDSLKeyword("delete")
		case TRANSACTION_LIT:
			tok.Type = TRANSACTION_LIT
			tok.Literal = l.readDSLKeyword("transaction")
		default:
			tok.Type = ILLEGAL
			tok.Literal = string(l.ch)
		}
		tok.Line = line
		tok.Column = column
		l.lastTokenType = tok.Type
		return l.attachPendingTrivia(tok)
	case '$':
		// Could be $, CA$, AU$, HK$, S$ followed by a number
		line := l.line
		column := l.column
		tok = l.readMoneyLiteral()
		tok.Line = line
		tok.Column = column
		l.lastTokenType = tok.Type
		return l.attachPendingTrivia(tok)
	case 0:
		tok.Literal = ""
		tok.Type = EOF
		tok.Line = l.line
		tok.Column = l.column
	default:
		// Check for Unicode currency symbols (£, €, ¥)
		if l.chRune == '£' { // £ (GBP)
			line := l.line
			column := l.column
			tok = l.readMoneyLiteral()
			tok.Line = line
			tok.Column = column
			l.lastTokenType = tok.Type
			return l.attachPendingTrivia(tok)
		} else if l.chRune == '€' { // € (EUR)
			line := l.line
			column := l.column
			tok = l.readMoneyLiteral()
			tok.Line = line
			tok.Column = column
			l.lastTokenType = tok.Type
			return l.attachPendingTrivia(tok)
		} else if l.chRune == '¥' { // ¥ (JPY)
			line := l.line
			column := l.column
			tok = l.readMoneyLiteral()
			tok.Line = line
			tok.Column = column
			l.lastTokenType = tok.Type
			return l.attachPendingTrivia(tok)
		} else if isLetterRune(l.chRune) {
			// Check for CODE# money syntax (e.g., USD#12.34)
			if l.isCurrencyCodeStart() {
				line := l.line
				column := l.column
				tok = l.readMoneyLiteral()
				tok.Line = line
				tok.Column = column
				l.lastTokenType = tok.Type
				return l.attachPendingTrivia(tok)
			}
			// Check for compound currency symbols (CA$, AU$, HK$, S$, CN¥)
			if l.isCompoundCurrencySymbol() {
				line := l.line
				column := l.column
				tok = l.readMoneyLiteral()
				tok.Line = line
				tok.Column = column
				l.lastTokenType = tok.Type
				return l.attachPendingTrivia(tok)
			}
			// Save position before reading
			line := l.line
			column := l.column
			tok.Literal = l.readIdentifier()
			tok.Type = LookupIdent(tok.Literal)
			tok.Line = line
			tok.Column = column
			l.lastTokenType = tok.Type
			return l.attachPendingTrivia(tok) // early return to avoid readChar()
		} else if isDigit(l.ch) {
			// Save position before reading
			line := l.line
			column := l.column
			tok.Literal = l.readNumber()
			// Check if it's a float or integer
			if containsDot(tok.Literal) {
				tok.Type = FLOAT
			} else {
				tok.Type = INT
			}
			tok.Line = line
			tok.Column = column
			l.lastTokenType = tok.Type
			return l.attachPendingTrivia(tok) // early return to avoid readChar()
		} else {
			tok = newToken(ILLEGAL, l.ch, l.line, l.column)
		}
	}

	l.readChar()
	l.lastTokenType = tok.Type
	return l.attachPendingTrivia(tok)
}

// newToken creates a new token with the given parameters
func newToken(tokenType TokenType, ch byte, line, column int) Token {
	return Token{Type: tokenType, Literal: string(ch), Line: line, Column: column}
}

// readIdentifier reads an identifier or keyword.
// Supports Unicode identifiers (e.g., π, α, 日本語) via isLetterRune.
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetterRune(l.chRune) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber reads a number (integer or float)
func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}

	// Check for decimal point
	if l.ch == '.' && isDigit(l.peekChar()) {
		l.readChar() // consume the '.'
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return l.input[position:l.position]
}

// Currency symbol mappings
var symbolToCurrency = map[string]string{
	"$":   "USD",
	"CA$": "CAD",
	"AU$": "AUD",
	"HK$": "HKD",
	"S$":  "SGD",
	"£":   "GBP",
	"€":   "EUR",
	"¥":   "JPY",
	"CN¥": "CNY",
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

// currencySymbolToDisplay converts a currency code to its display symbol if known
func currencySymbolToDisplay(code string) string {
	// Reverse lookup from symbolToCurrency
	for symbol, c := range symbolToCurrency {
		if c == code {
			return symbol
		}
	}
	// If not found, return the code itself
	return code
}

// isCurrencyCodeStart checks if current position starts a CODE# money literal
func (l *Lexer) isCurrencyCodeStart() bool {
	// Must be uppercase letter
	if l.ch < 'A' || l.ch > 'Z' {
		return false
	}

	// Look for pattern: 2-3 uppercase letters followed by #
	// We need to peek ahead without consuming
	pos := 0
	for {
		ch := l.peekCharN(pos + 1)
		if ch >= 'A' && ch <= 'Z' {
			pos++
			if pos > 2 { // Max 3 letters total (including current)
				return false
			}
		} else if ch == '#' && pos >= 1 { // Need at least 2 more chars (total 3) for valid code
			// Check that next char after # is a digit
			nextCh := l.peekCharN(pos + 2)
			return isDigit(nextCh)
		} else {
			return false
		}
	}
}

// isCompoundCurrencySymbol checks if current position starts a compound currency symbol
// like CA$, AU$, HK$, S$, or CN¥ followed by a digit
func (l *Lexer) isCompoundCurrencySymbol() bool {
	switch l.ch {
	case 'C':
		// CA$ or CN¥
		if l.peekChar() == 'A' && l.peekCharN(2) == '$' && isDigit(l.peekCharN(3)) {
			return true
		}
		if l.peekChar() == 'N' && l.peekCharN(2) == 0xC2 && l.peekCharN(3) == 0xA5 {
			// CN¥ - check for digit after ¥
			return isDigit(l.peekCharN(4))
		}
	case 'A':
		// AU$
		if l.peekChar() == 'U' && l.peekCharN(2) == '$' && isDigit(l.peekCharN(3)) {
			return true
		}
	case 'H':
		// HK$
		if l.peekChar() == 'K' && l.peekCharN(2) == '$' && isDigit(l.peekCharN(3)) {
			return true
		}
	case 'S':
		// S$
		if l.peekChar() == '$' && isDigit(l.peekCharN(2)) {
			return true
		}
	}
	return false
}

// readMoneyLiteral reads a money literal and returns a MONEY token
// Handles: $12.34, £99.99, EUR#50.00, CA$25.00, etc.
func (l *Lexer) readMoneyLiteral() Token {
	var currency string
	var negative bool

	// Handle currency symbol or CODE# prefix
	switch {
	case l.ch == '$':
		// Could be $, CA$, AU$, HK$, or S$
		l.readChar()
		switch {
		case l.ch == 'A' && l.peekChar() == '$': // Could be CA$ but need to check previous
			// This case won't happen since we already consumed $
			currency = "USD"
		default:
			currency = "USD"
		}
	case l.ch == 'C' && l.peekChar() == 'A' && l.peekCharN(2) == '$':
		l.readChar() // C
		l.readChar() // A
		l.readChar() // $
		currency = "CAD"
	case l.ch == 'A' && l.peekChar() == 'U' && l.peekCharN(2) == '$':
		l.readChar() // A
		l.readChar() // U
		l.readChar() // $
		currency = "AUD"
	case l.ch == 'H' && l.peekChar() == 'K' && l.peekCharN(2) == '$':
		l.readChar() // H
		l.readChar() // K
		l.readChar() // $
		currency = "HKD"
	case l.ch == 'S' && l.peekChar() == '$':
		l.readChar() // S
		l.readChar() // $
		currency = "SGD"
	case l.chRune == '£': // £ (GBP)
		l.readChar() // consume £ (now a single readChar for multi-byte)
		currency = "GBP"
	case l.chRune == '€': // € (EUR)
		l.readChar() // consume € (now a single readChar for multi-byte)
		currency = "EUR"
	case l.chRune == '¥': // ¥ (JPY)
		l.readChar() // consume ¥ (now a single readChar for multi-byte)
		currency = "JPY"
	case l.ch == 'C' && l.peekChar() == 'N': // Check for CN¥
		// Need to check for CN¥ - peek past CN to see if ¥ follows
		// Since readChar now advances by rune size, we need a different approach
		if l.readPosition+1 < len(l.input) {
			r, _ := utf8.DecodeRuneInString(l.input[l.readPosition+1:])
			if r == '¥' {
				l.readChar() // C
				l.readChar() // N
				l.readChar() // ¥
				currency = "CNY"
			} else {
				// Must be CODE# format
				currency = l.readCurrencyCode()
			}
		} else {
			// Must be CODE# format
			currency = l.readCurrencyCode()
		}
	default:
		// Must be CODE# format (e.g., USD#, EUR#, BTC#)
		currency = l.readCurrencyCode()
	}

	// Handle optional negative sign after currency symbol
	if l.ch == '-' {
		negative = true
		l.readChar()
	}

	// Read the number
	numStr := l.readNumber()
	if numStr == "" {
		// Currency symbol found but no valid number follows
		return Token{Type: ILLEGAL, Literal: fmt.Sprintf("currency symbol '%s' must be followed by a number", currencySymbolToDisplay(currency))}
	}

	// Calculate scale from decimal places in the literal
	literalScale := int8(0)
	dotIdx := -1
	for i, ch := range numStr {
		if ch == '.' {
			dotIdx = i
			break
		}
	}
	if dotIdx >= 0 {
		literalScale = int8(len(numStr) - dotIdx - 1)
	}

	// Determine the final scale:
	// - Use known currency scale if available (e.g., 2 for USD, 0 for JPY)
	// - Otherwise default to 2 for unknown currencies
	scale := int8(2) // Default for unknown currencies
	if knownScale, ok := CurrencyScales[currency]; ok {
		scale = knownScale
		// Validate: currencies like JPY shouldn't have decimals in literal
		if literalScale > 0 && knownScale == 0 {
			return Token{Type: ILLEGAL, Literal: fmt.Sprintf("%s does not allow decimal places (got %s)", currency, numStr)}
		}
		// Validate: literal can't have more decimal places than currency allows
		if literalScale > knownScale {
			return Token{Type: ILLEGAL, Literal: fmt.Sprintf("%s allows max %d decimal places (got %d in %s)", currency, knownScale, literalScale, numStr)}
		}
	}

	// Convert to integer amount (smallest unit) using the final scale
	amount := parseMoneyAmount(numStr, literalScale, scale)
	if negative {
		amount = -amount
	}

	// Build the literal string for display
	literal := buildMoneyLiteral(currency, amount, scale)

	return Token{
		Type:    MONEY,
		Literal: literal,
	}
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

// parseMoneyAmount converts a number string to an integer amount in smallest units
// literalScale is the number of decimal places in the input string
// targetScale is the currency's scale (decimal places in smallest unit)
func parseMoneyAmount(numStr string, literalScale, targetScale int8) int64 {
	// Remove the decimal point and parse as integer
	var result int64

	for _, ch := range numStr {
		if ch == '.' {
			continue
		}
		if ch >= '0' && ch <= '9' {
			result = result*10 + int64(ch-'0')
		}
	}

	// Scale up from literal scale to target scale
	// e.g., "100" (literalScale=0) with targetScale=2 means 100.00 = 10000 cents
	for literalScale < targetScale {
		result *= 10
		literalScale++
	}

	return result
}

// buildMoneyLiteral creates a display string for a money literal
func buildMoneyLiteral(currency string, amount int64, scale int8) string {
	if scale == 0 {
		return fmt.Sprintf("%s#%d", currency, amount)
	}

	// Calculate whole and fractional parts
	divisor := int64(1)
	for range scale {
		divisor *= 10
	}

	negative := amount < 0
	if negative {
		amount = -amount
	}

	whole := amount / divisor
	frac := amount % divisor

	// Format with leading zeros in fractional part
	format := fmt.Sprintf("%%s#%%d.%%0%dd", scale)
	result := fmt.Sprintf(format, currency, whole, frac)

	if negative {
		// Insert negative sign after currency code
		return currency + "#-" + result[len(currency)+1:]
	}
	return result
}

// readString reads a string literal with escape sequence support.
// Returns the string content and whether it was terminated properly.
// Strings cannot span multiple lines (use template literals for that).
func (l *Lexer) readString() (string, bool) {
	var result []byte
	l.readChar() // skip opening quote

	for l.ch != '"' && l.ch != 0 && l.ch != '\n' {
		if l.ch == '\\' {
			l.readChar() // consume backslash
			switch l.ch {
			case 'n':
				result = append(result, '\n')
			case 't':
				result = append(result, '\t')
			case '\\':
				result = append(result, '\\')
			case '"':
				result = append(result, '"')
			default:
				// Unknown escape, keep as-is
				result = append(result, '\\')
				result = append(result, l.ch)
			}
		} else {
			result = l.appendCurrentChar(result)
		}
		l.readChar()
	}

	// Check if string was properly terminated
	terminated := l.ch == '"'
	return string(result), terminated
}

// readRawString reads a single-quoted raw string literal.
// Only \' and \\ are processed as escapes; everything else is literal.
// Returns the string content, whether it was terminated properly, and whether it contains @{} interpolation.
func (l *Lexer) readRawString() (string, bool, bool) {
	var result []byte
	hasInterpolation := false
	l.readChar() // skip opening single quote

	for l.ch != '\'' && l.ch != 0 && l.ch != '\n' {
		if l.ch == '\\' {
			l.readChar() // consume backslash
			switch l.ch {
			case '\'':
				result = append(result, '\'')
			case '\\':
				result = append(result, '\\')
			case '@':
				// \@ is literal @ (escapes the interpolation sigil)
				result = append(result, '@')
			default:
				// Not a recognized escape - keep both backslash and char literal
				result = append(result, '\\')
				result = append(result, l.ch)
			}
		} else {
			// Check for @{ interpolation marker
			if l.ch == '@' && l.peekChar() == '{' {
				hasInterpolation = true
			}
			result = l.appendCurrentChar(result)
		}
		l.readChar()
	}

	// Check if string was properly terminated
	terminated := l.ch == '\''
	return string(result), terminated, hasInterpolation
}

// readTemplate reads a template literal (backtick string)
func (l *Lexer) readTemplate() string {
	var result []byte
	l.readChar() // skip opening backtick

	for l.ch != '`' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar() // consume backslash
			switch l.ch {
			case '`':
				result = append(result, '`')
			default:
				// Unknown escape, keep as-is
				result = append(result, '\\')
				result = append(result, l.ch)
			}
		} else {
			result = l.appendCurrentChar(result)
		}
		l.readChar()
	}

	return string(result)
}

// skipXMLComment skips an XML comment <!-- ... -->
// Returns true if a comment was successfully skipped
func (l *Lexer) skipXMLComment() bool {
	// We're at '<', peek for '!--'
	if l.peekChar() != '!' || l.peekCharN(2) != '-' || l.peekCharN(3) != '-' {
		return false
	}

	// Skip the <!--
	l.readChar() // skip <
	l.readChar() // skip !
	l.readChar() // skip -
	l.readChar() // skip -

	// Read until we find -->
	for l.ch != 0 {

		if l.ch == '-' && l.peekChar() == '-' && l.peekCharN(2) == '>' {
			l.readChar() // skip -
			l.readChar() // skip -
			l.readChar() // skip >
			break
		}
		l.readChar()
	}

	return true
}

// readCDATA reads a CDATA section <![CDATA[ ... ]]> and returns its content
func (l *Lexer) readCDATA() (string, bool) {
	// We're at '<', check for '![CDATA['
	if l.peekChar() != '!' || l.peekCharN(2) != '[' || l.peekCharN(3) != 'C' ||
		l.peekCharN(4) != 'D' || l.peekCharN(5) != 'A' || l.peekCharN(6) != 'T' ||
		l.peekCharN(7) != 'A' || l.peekCharN(8) != '[' {
		return "", false
	}

	// Skip the <![CDATA[
	for range 9 {
		l.readChar()
	}

	var content []byte
	// Read until we find ]]>
	for l.ch != 0 {

		if l.ch == ']' && l.peekChar() == ']' && l.peekCharN(2) == '>' {
			l.readChar() // skip ]
			l.readChar() // skip ]
			l.readChar() // skip >
			break
		}
		content = append(content, l.ch)
		l.readChar()
	}

	return string(content), true
}

// readProcessingInstruction reads a processing instruction <?...?>
// Returns the full content including delimiters
func (l *Lexer) readProcessingInstruction() string {
	var result []byte
	result = append(result, '<', '?')
	l.readChar() // skip <
	l.readChar() // skip ?

	// Read until we find ?>
	for l.ch != 0 {

		if l.ch == '?' && l.peekChar() == '>' {
			result = append(result, '?', '>')
			l.readChar() // skip ?
			l.readChar() // skip >
			break
		}
		result = append(result, l.ch)
		l.readChar()
	}

	return string(result)
}

// readDoctype reads a DOCTYPE declaration <!DOCTYPE...>
// Returns the full content including delimiters
func (l *Lexer) readDoctype() string {
	var result []byte
	result = append(result, '<', '!')
	l.readChar() // skip <
	l.readChar() // skip !

	// Read until we find >
	for l.ch != 0 {

		if l.ch == '>' {
			result = append(result, '>')
			l.readChar() // skip >
			break
		}
		result = append(result, l.ch)
		l.readChar()
	}

	return string(result)
}

// readTagEnd reads a closing tag like </div> or </my-component> or </basil.cache.Cache>
func (l *Lexer) readTagEnd() string {
	var result []byte
	l.readChar() // skip <
	l.readChar() // skip /

	// Read tag name (allow Unicode letters, hyphens for web components like my-component, dots for namespaced components like basil.cache.Cache)
	for isLetterRune(l.chRune) || isDigit(l.ch) || l.ch == '-' || l.ch == '.' {
		result = l.appendCurrentChar(result)
		l.readChar()
	}

	// Skip whitespace before >
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}

	// Expect >
	if l.ch != '>' {
		// Error: expected >
		return string(result)
	}
	l.readChar() // consume >

	return string(result)
}

// readTagStartOrSingleton reads a tag start <tag> or singleton <tag />
// Returns the tag content and a boolean indicating if it's a singleton
func (l *Lexer) readTagStartOrSingleton() (string, bool) {
	var result []byte
	l.readChar() // skip opening <

	// Check for empty grouping tag <>
	if l.ch == '>' {
		l.readChar()     // consume >
		return "", false // empty grouping tag is a tag start
	}

	// Read until we find > or />
	isSingleton := false
	for l.ch != 0 {

		// Check for closing />
		if l.ch == '/' && l.peekChar() == '>' {
			l.readChar() // consume /
			l.readChar() // consume >
			isSingleton = true
			break
		}

		// Check for just >
		if l.ch == '>' {
			l.readChar() // consume >
			break
		}

		// Handle double-quoted string literals within the tag
		if l.ch == '"' {
			result = append(result, l.ch)
			l.readChar()
			// Read until closing quote
			for l.ch != '"' && l.ch != 0 {
				if l.ch == '\\' {
					result = append(result, l.ch)
					l.readChar()
					if l.ch != 0 {
						result = l.appendCurrentChar(result)
						l.readChar()
					}
				} else {
					result = l.appendCurrentChar(result)
					l.readChar()
				}
			}
			if l.ch == '"' {
				result = append(result, l.ch)
				l.readChar()
			}
			continue
		}

		// Handle single-quoted string literals within the tag (raw - no escape processing)
		if l.ch == '\'' {
			result = append(result, l.ch)
			l.readChar()
			// Read until closing single quote - only \' needs escaping
			for l.ch != '\'' && l.ch != 0 {
				if l.ch == '\\' && l.peekChar() == '\'' {
					// Escaped single quote - include just the quote
					l.readChar() // skip backslash
					result = append(result, l.ch)
					l.readChar()
				} else {
					result = l.appendCurrentChar(result)
					l.readChar()
				}
			}
			if l.ch == '\'' {
				result = append(result, l.ch)
				l.readChar()
			}
			continue
		}

		// Handle interpolation braces {}
		if l.ch == '{' {
			result = append(result, l.ch)
			l.readChar()
			braceDepth := 1
			// Read until matching closing brace
			for braceDepth > 0 && l.ch != 0 {
				if l.ch == '{' {
					braceDepth++
				} else if l.ch == '}' {
					braceDepth--
				} else if l.ch == '"' {
					// Handle string inside interpolation
					result = append(result, l.ch)
					l.readChar()
					for l.ch != '"' && l.ch != 0 {
						if l.ch == '\\' {
							result = append(result, l.ch)
							l.readChar()
							if l.ch != 0 {
								result = l.appendCurrentChar(result)
								l.readChar()
							}
							continue
						}
						result = l.appendCurrentChar(result)
						l.readChar()
					}
					if l.ch == '"' {
						result = append(result, l.ch)
						l.readChar()
					}
					continue
				}
				result = l.appendCurrentChar(result)
				l.readChar()
			}
			continue
		}

		result = l.appendCurrentChar(result)
		l.readChar()
	}

	return string(result), isSingleton
}

// nextTagContentToken returns the next token while in tag content mode
func (l *Lexer) nextTagContentToken() Token {
	var tok Token

	// In raw text mode (style/script), check for @{ which triggers interpolation
	inRawMode := l.inRawTextTag != ""

	// Consolidate multiple newlines into single spaces (but not in raw mode)
	if !inRawMode && (l.ch == '\n' || l.ch == '\r') {
		for l.ch == '\n' || l.ch == '\r' || l.ch == ' ' || l.ch == '\t' {
			l.readChar()
		}
		// Don't return whitespace token, just continue to next content
		if l.ch == 0 || l.ch == '<' || l.ch == '{' {
			// Fall through to handle these special cases
		} else {
			// Start with a space before the next text
			line := l.line
			column := l.column
			text := l.readTagText()
			tok = Token{Type: TAG_TEXT, Literal: " " + text, Line: line, Column: column}
			return tok
		}
	}

	line := l.line
	column := l.column

	switch l.ch {
	case 0:
		tok = Token{Type: EOF, Literal: "", Line: l.line, Column: l.column}
		l.inTagContent = false

	case '<':
		if l.peekChar() == '/' {
			// Closing tag
			tok.Type = TAG_END
			closingTagName := l.readTagEnd()
			tok.Literal = closingTagName
			tok.Line = line
			tok.Column = column
			l.tagDepth--
			if l.tagDepth == 0 {
				l.inTagContent = false
			}
			// If we're closing a raw text tag, exit raw text mode AND tag content mode
			if l.inRawTextTag != "" && closingTagName == l.inRawTextTag {
				l.inRawTextTag = ""
				l.inTagContent = false // Exit tag content mode when closing style/script
			}
			return tok
		} else if l.peekChar() == '!' {
			// Could be XML comment <!-- --> or CDATA <![CDATA[]]>
			if l.peekCharN(2) == '-' && l.peekCharN(3) == '-' {
				// XML comment - skip it and get next token
				l.skipXMLComment()
				return l.nextTagContentToken()
			} else if l.peekCharN(2) == '[' && l.peekCharN(3) == 'C' {
				// CDATA section - return as TAG_TEXT
				content, ok := l.readCDATA()
				if ok {
					tok.Type = TAG_TEXT
					tok.Literal = content
					tok.Line = line
					tok.Column = column
					return tok
				}
			}
			// Not a comment or CDATA, treat as literal text
			tok.Type = TAG_TEXT
			tok.Literal = string(l.ch)
			tok.Line = line
			tok.Column = column
			l.readChar()
			return tok
		} else if l.peekChar() == '>' {
			// Empty grouping tag start
			l.readChar() // skip <
			l.readChar() // skip >
			tok = Token{Type: TAG_START, Literal: "", Line: line, Column: column}
			l.tagDepth++
			return tok
		} else if isLetter(l.peekChar()) {
			// Nested tag (start or singleton)
			tagContent, isSingleton := l.readTagStartOrSingleton()
			if isSingleton {
				tok.Type = TAG
			} else {
				tok.Type = TAG_START
				l.tagDepth++
				// Check if this is a raw text tag (style or script)
				tagName := extractTagName(tagContent)
				if tagName == "style" || tagName == "script" {
					l.inRawTextTag = tagName
				}
			}
			tok.Literal = tagContent
			tok.Line = line
			tok.Column = column
			return tok
		} else {
			// Literal < character in content
			tok.Type = TAG_TEXT
			tok.Literal = string(l.ch)
			tok.Line = line
			tok.Column = column
			l.readChar()
			return tok
		}

	case '/':
		// In normal mode, skip Parsley comments //
		// In raw mode (style/script), keep // as literal text (valid JS comments)
		if !inRawMode && l.peekChar() == '/' {
			l.skipAndCaptureComment()
			return l.nextTagContentToken()
		}
		// Not a comment (or in raw mode), treat as regular text
		tok.Type = TAG_TEXT
		if inRawMode {
			tok.Literal = l.readRawTagText()
		} else {
			tok.Literal = l.readTagText()
		}
		tok.Line = line
		tok.Column = column
		return tok

	case '@':
		// In raw text mode, @{ triggers interpolation
		if inRawMode && l.peekChar() == '{' {
			l.readChar() // skip @
			tok = newToken(LBRACE, l.ch, l.line, l.column)
			l.readChar() // skip {
			l.inTagContent = false
			l.inRawTextInterpolate = true // Remember to re-enter tag mode after }
			return tok
		}
		// Not @{, treat @ as regular text
		tok.Type = TAG_TEXT
		if inRawMode {
			tok.Literal = l.readRawTagText()
		} else {
			tok.Literal = l.readTagText()
		}
		tok.Line = line
		tok.Column = column
		return tok

	case '{':
		if inRawMode {
			// In raw text mode, { is literal - read as text
			tok.Type = TAG_TEXT
			tok.Literal = l.readRawTagText()
			tok.Line = line
			tok.Column = column
			return tok
		}
		// Normal mode: interpolation - temporarily exit tag content mode
		tok = newToken(LBRACE, l.ch, l.line, l.column)
		l.readChar()
		l.inTagContent = false
		return tok

	default:
		// Regular text content
		tok.Type = TAG_TEXT
		if inRawMode {
			tok.Literal = l.readRawTagText()
		} else {
			tok.Literal = l.readTagText()
		}
		tok.Line = line
		tok.Column = column
	}

	return tok
}

// readTagText reads text content until we hit <, {, or EOF
func (l *Lexer) readTagText() string {
	var result []byte

	for l.ch != 0 && l.ch != '<' && l.ch != '{' {
		// Skip Parsley comments (//) and capture them
		if l.ch == '/' && l.peekChar() == '/' {
			l.skipAndCaptureComment()
			continue
		}
		result = append(result, l.ch)
		l.readChar()
	}

	return string(result)
}

// readRawTagText reads text content in raw text mode (style/script)
// In raw text mode, {} is literal and @{} is used for interpolation
// // comments are preserved (valid in JavaScript, harmless in CSS)
// Stops at <, @{, or EOF
func (l *Lexer) readRawTagText() string {
	var result []byte

	for l.ch != 0 && l.ch != '<' {
		// Check for @{ which starts interpolation in raw text mode
		// This works even inside // comments, allowing datestamps etc.
		if l.ch == '@' && l.peekChar() == '{' {
			break
		}
		result = append(result, l.ch)
		l.readChar()
	}

	return string(result)
}

// extractTagName extracts the tag name from tag content (e.g., "div class=\"foo\"" -> "div")
func extractTagName(tagContent string) string {
	var name []byte
	for i := 0; i < len(tagContent); i++ {
		ch := tagContent[i]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			break
		}
		name = append(name, ch)
	}
	return string(name)
}

// EnterTagContentMode sets the lexer into tag content mode
func (l *Lexer) EnterTagContentMode() {
	if l.tagDepth > 0 {
		l.inTagContent = true
	}
}

// collectTrivia collects whitespace (counting blank lines) and comments
// before the next token. This loop handles alternating whitespace and comments.
// It also detects trailing comments (comments on same line as previous token).
func (l *Lexer) collectTrivia() {
	// First, check for trailing comment (comment before any newline)
	// Only check if we've already emitted at least one token (lastTokenType != ILLEGAL)
	// Otherwise, any comment at the start of input is a leading comment, not trailing
	if l.lastTokenType != ILLEGAL {
		l.skipHorizontalWhitespace()
		if l.ch == '/' && l.peekChar() == '/' {
			l.captureTrailingComment()
		}
	}

	// Now collect any remaining trivia (newlines, blank lines, leading comments)
	for {
		// Skip whitespace (counting blank lines)
		l.skipWhitespace()

		// Check for comment
		if l.ch == '/' && l.peekChar() == '/' {
			l.skipAndCaptureComment()
			continue // Keep collecting - there may be more whitespace/comments
		}

		// No more trivia to collect
		break
	}
}

// skipHorizontalWhitespace skips spaces and tabs only (not newlines)
func (l *Lexer) skipHorizontalWhitespace() {
	for l.ch == ' ' || l.ch == '\t' {
		l.readChar()
	}
}

// captureTrailingComment captures a comment that's on the same line as the previous token
func (l *Lexer) captureTrailingComment() {
	startPos := l.position

	// Skip the two slashes
	l.readChar()
	l.readChar()

	// Read until end of line or EOF
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}

	// Store as pending trailing comment (will be attached to previous token's statement)
	l.pendingTrailingComment = l.input[startPos:l.position]
}

// attachPendingTrivia attaches any pending comments and blank line count to a token,
// then clears the pending state.
func (l *Lexer) attachPendingTrivia(tok Token) Token {
	tok.LeadingComments = l.pendingComments
	tok.BlankLinesBefore = l.pendingBlankLines
	tok.TrailingComment = l.pendingTrailingComment // This is the trailing comment from the PREVIOUS line
	l.pendingComments = nil
	l.pendingBlankLines = 0
	l.pendingTrailingComment = ""
	return tok
}

// skipWhitespace skips whitespace characters and counts blank lines.
// A blank line is defined as two or more consecutive newlines (possibly with
// whitespace between them). This is used to preserve intentional spacing.
func (l *Lexer) skipWhitespace() {
	newlineCount := 0
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		if l.ch == '\n' {
			newlineCount++
		}
		l.readChar()
	}
	// Two newlines = one blank line, three newlines = two blank lines, etc.
	// But we collapse multiple blank lines to just one (like gofmt)
	if newlineCount >= 2 {
		l.pendingBlankLines = 1
	}
}

// skipAndCaptureComment reads a single-line comment and captures its text.
// The comment text is added to pendingComments for attachment to the next token.
func (l *Lexer) skipAndCaptureComment() {
	startPos := l.position

	// Skip the two slashes
	l.readChar()
	l.readChar()

	// Read until end of line or EOF
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}

	// Capture the comment text (including the //)
	commentText := l.input[startPos:l.position]
	l.pendingComments = append(l.pendingComments, commentText)
}

// isLetter checks if a byte represents a letter (ASCII fast-path).
// For non-ASCII bytes (>=0x80), this returns false - use isLetterRune for Unicode.
func isLetter(ch byte) bool {
	// ASCII fast-path: handles a-z, A-Z, _
	return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

// isLetterRune checks if a rune is a valid identifier character (letter or underscore).
// This supports Unicode letters like π, α, 日本語, etc.
func isLetterRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

// isDigit checks if the character is a digit
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// containsDot checks if a string contains a decimal point
func containsDot(s string) bool {
	for _, ch := range s {
		if ch == '.' {
			return true
		}
	}
	return false
}

// readRegex reads a regex literal like /pattern/flags
func (l *Lexer) readRegex() (string, string) {
	var pattern []byte
	l.readChar() // skip opening /

	// Read pattern until we find unescaped /
	for l.ch != '/' && l.ch != 0 && l.ch != '\n' {
		if l.ch == '\\' {
			pattern = append(pattern, l.ch)
			l.readChar()
			if l.ch != 0 {
				pattern = append(pattern, l.ch)
				l.readChar()
			}
		} else {
			pattern = append(pattern, l.ch)
			l.readChar()
		}
	}

	if l.ch != '/' {
		// Invalid regex (unterminated)
		return string(pattern), ""
	}

	l.readChar() // consume closing /

	// Read flags (letters immediately after closing /)
	var flags []byte
	for isLetter(l.ch) {
		flags = append(flags, l.ch)
		l.readChar()
	}

	return string(pattern), string(flags)
}

// readDatetimeLiteral reads a datetime literal after @
// Supports formats: @2024-12-25, @2024-12-25T14:30:00, @2024-12-25T14:30:00Z, @2024-12-25T14:30:00-05:00
// Also supports time-only: @12:30, @12:30:00, @9:30
func (l *Lexer) readDatetimeLiteral() string {
	var datetime []byte
	l.readChar() // skip @

	// Check if this is a time-only literal (starts with 1-2 digits then ':')
	// vs a date literal (starts with 4 digits then '-')
	isTimeOnly := false
	startPos := l.position

	// Count initial digits to determine if time-only
	digitCount := 0
	for isDigit(l.ch) {
		digitCount++
		datetime = append(datetime, l.ch)
		l.readChar()
	}

	if digitCount <= 2 && l.ch == ':' {
		// Time-only literal: HH:MM or HH:MM:SS
		isTimeOnly = true
		datetime = append(datetime, l.ch)
		l.readChar()
		// Read minutes
		for isDigit(l.ch) {
			datetime = append(datetime, l.ch)
			l.readChar()
		}
		// Check for seconds: :SS
		if l.ch == ':' {
			datetime = append(datetime, l.ch)
			l.readChar()
			for isDigit(l.ch) {
				datetime = append(datetime, l.ch)
				l.readChar()
			}
		}
	} else if digitCount == 4 && l.ch == '-' {
		// Date literal: YYYY-MM-DD potentially with time
		// Continue reading date part
		for l.ch == '-' || isDigit(l.ch) {
			datetime = append(datetime, l.ch)
			l.readChar()
		}

		// Check for time part: T14:30:00
		if l.ch == 'T' {
			datetime = append(datetime, l.ch)
			l.readChar()
			// Read time: HH:MM:SS
			for isDigit(l.ch) || l.ch == ':' {
				datetime = append(datetime, l.ch)
				l.readChar()
			}
		}

		// Check for fractional seconds (.123)
		if l.ch == '.' && isDigit(l.peekChar()) {
			datetime = append(datetime, l.ch)
			l.readChar()
			// Read the fractional part
			for isDigit(l.ch) {
				datetime = append(datetime, l.ch)
				l.readChar()
			}
		}

		// Check for timezone: Z or +05:00 or -05:00
		switch l.ch {
		case 'Z':
			datetime = append(datetime, l.ch)
			l.readChar()
		case '+', '-':
			// Only consume if followed by digit (timezone offset)
			if isDigit(l.peekChar()) {
				datetime = append(datetime, l.ch)
				l.readChar()
				// Read timezone offset
				for isDigit(l.ch) || l.ch == ':' {
					datetime = append(datetime, l.ch)
					l.readChar()
				}
			}
		}
	}

	// Avoid unused variable warning
	_ = isTimeOnly
	_ = startPos

	return string(datetime)
}

// readDurationLiteral reads a duration literal after @
// Supports formats: @2h30m, @7d, @1y6mo, @30s, @-1d, @-2w (negative durations)
// Units: y (years), mo (months), w (weeks), d (days), h (hours), m (minutes), s (seconds)
func (l *Lexer) readDurationLiteral() string {
	var duration []byte
	l.readChar() // skip @

	// Check for negative duration
	if l.ch == '-' {
		duration = append(duration, l.ch)
		l.readChar()
	}

	// Read pairs of number + unit
	for isDigit(l.ch) {
		// Read number

		for isDigit(l.ch) {
			duration = append(duration, l.ch)
			l.readChar()
		}

		// Read unit (could be single letter or "mo" for months)
		if !isLetter(l.ch) {
			break
		}

		// Check for "mo" (months)
		if l.ch == 'm' && l.peekChar() == 'o' {
			duration = append(duration, l.ch)
			l.readChar()
			duration = append(duration, l.ch)
			l.readChar()
		} else {
			// Single letter unit
			duration = append(duration, l.ch)
			l.readChar()
		}
	}

	return string(duration)
}

// isWhitespace checks if the given byte is whitespace
func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

// detectAtLiteralType determines what type of @ literal this is
// Returns the appropriate TokenType for the literal
func (l *Lexer) detectAtLiteralType() TokenType {
	pos := l.readPosition

	if pos >= len(l.input) {
		return ILLEGAL
	}

	// Check for @( which indicates a template path/URL
	if l.input[pos] == '(' {
		return l.detectTemplateAtLiteralType()
	}

	// Check for @std/ which indicates a standard library import
	if pos+4 <= len(l.input) && l.input[pos:pos+4] == "std/" {
		return STDLIB_PATH
	}

	// Check for @std (without slash) - the stdlib root
	if pos+3 <= len(l.input) && l.input[pos:pos+3] == "std" {
		// Make sure it's just "std" and not "stdout" or similar
		if pos+3 >= len(l.input) || !isLetter(l.input[pos+3]) {
			return STDLIB_PATH
		}
	}

	// Check for @basil/ which indicates a basil namespace import
	if pos+6 <= len(l.input) && l.input[pos:pos+6] == "basil/" {
		return STDLIB_PATH
	}

	// Check for @basil (without slash) - the basil root
	if pos+5 <= len(l.input) && l.input[pos:pos+5] == "basil" {
		// Ensure it's not followed by identifier characters
		if pos+5 >= len(l.input) || !isLetter(l.input[pos+5]) {
			return STDLIB_PATH
		}
	}

	// Check for @now-style literals
	if l.isKeywordAt(pos, "now") {
		return DATETIME_NOW
	}
	if l.isKeywordAt(pos, "timeNow") {
		return TIME_NOW
	}
	if l.isKeywordAt(pos, "dateNow") || l.isKeywordAt(pos, "today") {
		return DATE_NOW
	}

	// Check for connection literals
	for _, conn := range []struct {
		keyword string
		token   TokenType
	}{
		{"sqlite", SQLITE_LITERAL},
		{"postgres", POSTGRES_LITERAL},
		{"mysql", MYSQL_LITERAL},
		{"sftp", SFTP_LITERAL},
		{"shell", SHELL_LITERAL},
		{"DB", DB_LITERAL},
		{"SEARCH", SEARCH_LITERAL},
		{"env", ENV_LITERAL},
		{"args", ARGS_LITERAL},
		{"params", PARAMS_LITERAL},
	} {
		if l.isKeywordAt(pos, conn.keyword) {
			return conn.token
		}
	}

	// Check for Query DSL literals
	for _, dsl := range []struct {
		keyword string
		token   TokenType
	}{
		{"schema", SCHEMA_LITERAL},
		{"table", TABLE_LITERAL},
		{"query", QUERY_LITERAL},
		{"insert", INSERT_LITERAL},
		{"update", UPDATE_LITERAL},
		{"delete", DELETE_LITERAL},
		{"transaction", TRANSACTION_LIT},
	} {
		if l.isKeywordAt(pos, dsl.keyword) {
			return dsl.token
		}
	}

	// Check for @- (stdin/stdout) - must be just "-" not followed by path char or digit
	if l.input[pos] == '-' {
		// Check next char - if it's not a digit and not a path char, it's stdin
		if pos+1 >= len(l.input) || (!isDigit(l.input[pos+1]) && !isPathChar(l.input[pos+1])) {
			return PATH_LITERAL
		}
	}

	// Check for @stdin, @stdout, @stderr aliases
	stdioAliases := []string{"stdin", "stdout", "stderr"}
	for _, alias := range stdioAliases {
		if pos+len(alias) <= len(l.input) {
			match := true
			for i, ch := range alias {
				if l.input[pos+i] != byte(ch) {
					match = false
					break
				}
			}
			// Make sure it's not followed by more identifier chars
			if match && (pos+len(alias) >= len(l.input) || !isLetter(l.input[pos+len(alias)]) && !isDigit(l.input[pos+len(alias)])) {
				return PATH_LITERAL
			}
		}
	}

	// Check for URL: @scheme://
	// Look for characters followed by ://
	colonPos := pos
	for colonPos < len(l.input) && colonPos < pos+20 {
		if l.input[colonPos] == ':' {
			if colonPos+2 < len(l.input) && l.input[colonPos+1] == '/' && l.input[colonPos+2] == '/' {
				return URL_LITERAL
			}
			break
		}
		if !isLetter(l.input[colonPos]) && l.input[colonPos] != '+' && l.input[colonPos] != '-' {
			break
		}
		colonPos++
	}

	// Check for path: @/ or @./ or @~/ or @../ or @.filename (dotfiles)
	firstChar := l.input[pos]
	if firstChar == '/' {
		return PATH_LITERAL
	}
	if firstChar == '.' {
		// Could be ./, .., or a dotfile like .config
		if pos+1 < len(l.input) {
			nextChar := l.input[pos+1]
			// ./ or .. → path
			if nextChar == '/' || nextChar == '.' {
				return PATH_LITERAL
			}
			// .letter → dotfile path (like .config, .bashrc)
			if isLetter(nextChar) || isDigit(nextChar) {
				return PATH_LITERAL
			}
		}
		// Just a dot by itself → path (current directory)
		return PATH_LITERAL
	}
	if firstChar == '~' && pos+1 < len(l.input) && l.input[pos+1] == '/' {
		return PATH_LITERAL
	}

	// Check for negative duration: @-1d, @-2w, etc.
	// A minus followed by a digit indicates negative duration
	if firstChar == '-' && pos+1 < len(l.input) && isDigit(l.input[pos+1]) {
		return DURATION_LITERAL
	}

	// Check for datetime: 4 digits followed by '-'
	digitCount := 0
	checkPos := pos
	for checkPos < len(l.input) && isDigit(l.input[checkPos]) {
		digitCount++
		checkPos++
	}

	if digitCount == 4 && checkPos < len(l.input) && l.input[checkPos] == '-' {
		return DATETIME_LITERAL
	}

	// Check for time-only literal: 1-2 digits followed by ':'
	// e.g., @12:30, @9:30, @12:30:00
	if (digitCount == 1 || digitCount == 2) && checkPos < len(l.input) && l.input[checkPos] == ':' {
		return DATETIME_LITERAL
	}

	// Default to duration
	return DURATION_LITERAL
}

// isKeywordAt reports whether the given keyword appears at position pos
// and is not followed by additional identifier characters.
func (l *Lexer) isKeywordAt(pos int, keyword string) bool {
	end := pos + len(keyword)
	if end > len(l.input) {
		return false
	}
	if string(l.input[pos:end]) != keyword {
		return false
	}
	if end < len(l.input) && (isLetter(l.input[end]) || isDigit(l.input[end])) {
		return false
	}
	return true
}

// detectDateNowKeyword identifies whether the current @date literal keyword
// is "dateNow" or the synonym "today".
func (l *Lexer) detectDateNowKeyword() string {
	pos := l.readPosition
	if l.isKeywordAt(pos, "dateNow") {
		return "dateNow"
	}
	return "today"
}

// readNowLiteral consumes the @ prefix and advances past the keyword,
// returning the matched literal keyword.
func (l *Lexer) readNowLiteral(keyword string) string {
	l.readChar() // skip @
	for i := 0; i < len(keyword); i++ {
		l.readChar()
	}
	return keyword
}

// readConnectionLiteral consumes the @ prefix and advances past the connection keyword.
// The returned literal is the keyword itself (e.g., "sqlite").
func (l *Lexer) readConnectionLiteral(keyword string) string {
	l.readChar() // skip @
	for i := 0; i < len(keyword); i++ {
		l.readChar()
	}
	return keyword
}

// readDSLKeyword consumes the @ prefix and advances past the DSL keyword.
// The returned literal is the keyword itself (e.g., "schema", "query").
func (l *Lexer) readDSLKeyword(keyword string) string {
	l.readChar() // skip @
	for i := 0; i < len(keyword); i++ {
		l.readChar()
	}
	return keyword
}

// readPathLiteral reads a path literal after @
// Supports: @/absolute/path, @./relative/path, @~/home/path, @-, @stdin, @stdout, @stderr
func (l *Lexer) readPathLiteral() string {
	l.readChar() // skip @

	// Check for @- (stdin/stdout special path)
	if l.ch == '-' && !isDigit(l.peekChar()) && !isPathChar(l.peekChar()) {
		l.readChar()
		return "-"
	}

	// Check for @stdin, @stdout, @stderr aliases
	if l.ch == 's' {
		// Try to match stdin, stdout, stderr
		for _, alias := range []string{"stdin", "stdout", "stderr"} {
			if l.matchKeyword(alias) {
				return alias
			}
		}
	}

	var path []byte
	// Read until whitespace or delimiter
	for l.ch != 0 && !isWhitespace(l.ch) {
		// Stop at delimiters that can't be in a path literal
		if l.ch == ')' || l.ch == ']' || l.ch == '}' || l.ch == ',' || l.ch == ';' {
			break
		}
		// Stop at dot in specific cases to handle property access
		// Examples:
		//   @/path/to/file.txt.length  → stop at .length
		//   @./file.txt.extension      → stop at .extension
		// But allow:
		//   @./.config           → dotfile after ./
		//   @dir/.hidden         → dotfile in a directory
		//   @file.txt            → file with extension
		if l.ch == '.' && len(path) > 0 {
			nextCh := l.peekChar()

			// If next char is / or ., this is part of the path (../ or ./)
			if nextCh == '/' || nextCh == '.' {
				path = append(path, l.ch)
				l.readChar()
				continue
			}

			// If previous char is '/' and next is a letter, could be:
			// - A dotfile: /. followed by filename → allow
			// - Property access after path → stop
			// We distinguish by checking if it looks like a known property name
			if path[len(path)-1] == '/' && isLetter(nextCh) {
				// This looks like a dotfile (e.g., /.config, /.bashrc)
				// Continue reading - don't break
				path = append(path, l.ch)
				l.readChar()
				continue
			}

			// If next char is not /, ., or a path char, and IS a letter,
			// this might be property access
			if !isPathChar(nextCh) && isLetter(nextCh) {
				break
			}
		}
		path = append(path, l.ch)
		l.readChar()
	}

	return string(path)
}

// readStdlibPath reads a standard library path after @
// Supports: @std/table, @std/string, etc.
// Returns the path WITH the "std/" prefix (e.g., "std/table")
func (l *Lexer) readStdlibPath() string {
	l.readChar() // skip @

	var path []byte
	// Read until whitespace or delimiter
	for l.ch != 0 && !isWhitespace(l.ch) {
		// Stop at delimiters that can't be in a module name
		if l.ch == ')' || l.ch == ']' || l.ch == '}' || l.ch == ',' || l.ch == ';' {
			break
		}
		path = append(path, l.ch)
		l.readChar()
	}

	return string(path)
}

// matchKeyword tries to match a keyword at current position, returns true if matched and advances
func (l *Lexer) matchKeyword(keyword string) bool {
	// Check if we can match the keyword
	savedPos := l.position
	savedReadPos := l.readPosition
	savedCh := l.ch

	for i := 0; i < len(keyword); i++ {
		if l.ch != keyword[i] {
			// Restore position
			l.position = savedPos
			l.readPosition = savedReadPos
			l.ch = savedCh
			return false
		}
		l.readChar()
	}

	// Make sure keyword isn't followed by more identifier chars
	if isLetter(l.ch) || isDigit(l.ch) {
		// Restore position
		l.position = savedPos
		l.readPosition = savedReadPos
		l.ch = savedCh
		return false
	}

	return true
}

// isPathChar checks if a character is valid in a path (but not at the start of a property)
func isPathChar(ch byte) bool {
	return ch == '/' || ch == '-' || ch == '_' || ch == '~' || isLetter(ch) || isDigit(ch)
}

// readUrlLiteral reads a URL literal after @
// Supports: @scheme://host/path?query#fragment
func (l *Lexer) readUrlLiteral() string {
	l.readChar() // skip @

	var url []byte
	hasScheme := false // Track if we've seen ://

	// Read until whitespace or delimiter
	for l.ch != 0 && !isWhitespace(l.ch) {
		// Track if we've seen the :// pattern
		if !hasScheme && l.ch == ':' && len(url) > 0 {
			if l.peekChar() == '/' {
				hasScheme = true
			}
		}

		// Stop at delimiters that can't be in a URL literal
		if l.ch == ')' || l.ch == ']' || l.ch == '}' || l.ch == ',' || l.ch == ';' {
			break
		}

		// For dots: if we've seen ://, ALL dots are part of the URL until we hit a delimiter or whitespace
		// This handles .com, .org, file.html, etc.
		// We only stop at . for property access if there's no scheme (edge case)
		if l.ch == '.' && isLetter(l.peekChar()) && !hasScheme {
			// No :// seen yet, so this might be property access
			break
		}

		url = append(url, l.ch)
		l.readChar()
	}

	return string(url)
}

// shouldTreatAsRegex determines if / should be regex or division
// Regex context: after operators, keywords, commas, open parens/brackets
// But NOT after complete expressions like identifiers, numbers, close parens
func (l *Lexer) shouldTreatAsRegex(lastToken TokenType) bool {
	switch lastToken {
	case ASSIGN, EQ, NOT_EQ, LT, GT, LTE, GTE,
		AND, OR, MATCH, NOT_MATCH,
		LPAREN, LBRACKET, LBRACE,
		COMMA, SEMICOLON, COLON,
		RETURN, LET, IF, ELSE, FOR, IN,
		PLUSPLUS:
		return true
	case 0: // Start of input
		return true
	// Don't treat as regex after arithmetic operators that could be infix
	// These appear in expressions like: x - /y/ which is (x - /) then /y/
	// Instead they're more likely: x-1/2 (division)
	default:
		return false
	}
}

// detectTemplateAtLiteralType determines if @(...) is a path, URL, or datetime template
// Called when we've detected @( - peeks inside to determine type
func (l *Lexer) detectTemplateAtLiteralType() TokenType {
	pos := l.readPosition + 1 // skip past (

	if pos >= len(l.input) {
		return ILLEGAL
	}

	// Look for :// pattern within the first 20 chars (URL indicator)
	// Also check for scheme pattern like http:// https:// ftp://
	scanPos := pos
	for scanPos < len(l.input) && scanPos < pos+20 {
		if l.input[scanPos] == ':' {
			if scanPos+2 < len(l.input) && l.input[scanPos+1] == '/' && l.input[scanPos+2] == '/' {
				return URL_TEMPLATE
			}
			break
		}
		// Stop if we hit closing paren or non-scheme character early
		if l.input[scanPos] == ')' || l.input[scanPos] == '{' {
			break
		}
		if !isLetter(l.input[scanPos]) && l.input[scanPos] != '+' && l.input[scanPos] != '-' && l.input[scanPos] != '.' && l.input[scanPos] != '/' && l.input[scanPos] != '~' {
			break
		}
		scanPos++
	}

	// Check for datetime template: starts with 4 digits followed by '-' or an interpolation
	// e.g., @(2024-{month}-{day}) or @({year}-12-25)
	digitCount := 0
	checkPos := pos

	// Count leading digits or check for { (interpolation start)
	for checkPos < len(l.input) && isDigit(l.input[checkPos]) {
		digitCount++
		checkPos++
	}

	// 4 digits followed by '-' is a date pattern (datetime template)
	if digitCount == 4 && checkPos < len(l.input) && l.input[checkPos] == '-' {
		return DATETIME_TEMPLATE
	}

	// Check for time-only template: 1-2 digits followed by ':'
	// e.g., @(12:{min}:00) or @({hour}:30:00)
	if (digitCount == 1 || digitCount == 2) && checkPos < len(l.input) && l.input[checkPos] == ':' {
		return DATETIME_TEMPLATE
	}

	// Check for interpolated datetime that starts with { followed by date/time pattern
	// e.g., @({year}-12-25) or @({hour}:30)
	if pos < len(l.input) && l.input[pos] == '{' {
		// Find the closing brace and check what follows
		bracePos := pos + 1
		for bracePos < len(l.input) && l.input[bracePos] != '}' {
			bracePos++
		}
		if bracePos+1 < len(l.input) {
			nextChar := l.input[bracePos+1]
			// If followed by '-' or ':', it's likely a datetime template
			if nextChar == '-' || nextChar == ':' {
				return DATETIME_TEMPLATE
			}
		}
	}

	// Default to path template
	return PATH_TEMPLATE
}

// readPathTemplate reads a path template after @(
// Returns the content between the parentheses
func (l *Lexer) readPathTemplate() string {
	l.readChar() // skip @
	l.readChar() // skip (

	var result []byte
	parenCount := 1
	braceCount := 0

	for parenCount > 0 && l.ch != 0 {
		if l.ch == '(' && braceCount == 0 {
			parenCount++
		} else if l.ch == ')' && braceCount == 0 {
			parenCount--
			if parenCount == 0 {
				// Don't append the closing ), just consume it and break
				l.readChar()
				break
			}
		} else if l.ch == '{' {
			braceCount++
		} else if l.ch == '}' {
			braceCount--
		} else if l.ch == '"' {
			// Handle string literals within expressions
			result = append(result, l.ch)
			l.readChar()
			for l.ch != '"' && l.ch != 0 {
				if l.ch == '\\' {
					result = append(result, l.ch)
					l.readChar()
					if l.ch != 0 {
						result = append(result, l.ch)
						l.readChar()
					}
				} else {
					result = append(result, l.ch)
					l.readChar()
				}
			}
		}
		result = append(result, l.ch)
		l.readChar()
	}

	return string(result)
}

// readUrlTemplate reads a URL template after @(
// Returns the content between the parentheses
func (l *Lexer) readUrlTemplate() string {
	// Same logic as readPathTemplate - the content is identical
	return l.readPathTemplate()
}

// readDatetimeTemplate reads a datetime template after @(
// Returns the content between the parentheses
// Examples: @(2024-{month}-{day}), @({hour}:30:00)
func (l *Lexer) readDatetimeTemplate() string {
	// Same logic as readPathTemplate - the content is identical
	return l.readPathTemplate()
}
