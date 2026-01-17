package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
	perrors "github.com/sambeau/basil/pkg/parsley/errors"
	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// Precedence levels for operators
const (
	_ int = iota
	LOWEST
	COMMA_PREC  // ,
	LOGIC_OR    // or, |
	LOGIC_AND   // and, &
	EQUALS      // ==
	LESSGREATER // > or <
	SUM         // +
	CONCAT      // ++
	PRODUCT     // *
	PREFIX      // -X or !X
	INDEX       // array[index]
	CALL        // myFunction(X)
)

// precedences maps tokens to their precedence
var precedences = map[lexer.TokenType]int{
	lexer.COMMA:        COMMA_PREC,
	lexer.OR:           LOGIC_OR,
	lexer.NULLISH:      LOGIC_OR,
	lexer.AND:          LOGIC_AND,
	lexer.EQ:           EQUALS,
	lexer.NOT_EQ:       EQUALS,
	lexer.MATCH:        EQUALS,
	lexer.NOT_MATCH:    EQUALS,
	lexer.IN:           EQUALS,
	lexer.BANG:         EQUALS, // for 'not in' operator
	lexer.LT:           LESSGREATER,
	lexer.GT:           LESSGREATER,
	lexer.LTE:          LESSGREATER,
	lexer.GTE:          LESSGREATER,
	lexer.PLUS:         SUM,
	lexer.MINUS:        SUM,
	lexer.RANGE:        SUM,
	lexer.PLUSPLUS:     CONCAT,
	lexer.SLASH:        PRODUCT,
	lexer.ASTERISK:     PRODUCT,
	lexer.PERCENT:      PRODUCT,
	lexer.LBRACKET:     INDEX,
	lexer.DOT:          INDEX,
	lexer.LPAREN:       CALL,
	lexer.QUERY_ONE:    EQUALS, // Database query operators
	lexer.QUERY_MANY:   EQUALS,
	lexer.EXECUTE:      EQUALS,
	lexer.EXECUTE_WITH: EQUALS, // Process execution operator
}

// Parser represents the parser
type Parser struct {
	l *lexer.Lexer

	structuredErrors []*perrors.ParsleyError // Structured errors

	prevToken lexer.Token
	curToken  lexer.Token
	peekToken lexer.Token

	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

// New creates a new parser instance
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l: l,
	}

	// Initialize prefix parse functions
	p.prefixParseFns = make(map[lexer.TokenType]prefixParseFn)
	p.registerPrefix(lexer.IDENT, p.parseIdentifier)
	p.registerPrefix(lexer.INT, p.parseIntegerLiteral)
	p.registerPrefix(lexer.FLOAT, p.parseFloatLiteral)
	p.registerPrefix(lexer.STRING, p.parseStringLiteral)
	p.registerPrefix(lexer.TEMPLATE, p.parseTemplateLiteral)
	p.registerPrefix(lexer.RAW_TEMPLATE, p.parseRawTemplateLiteral)
	p.registerPrefix(lexer.REGEX, p.parseRegexLiteral)
	p.registerPrefix(lexer.DATETIME_NOW, p.parseDatetimeNow)
	p.registerPrefix(lexer.TIME_NOW, p.parseTimeNow)
	p.registerPrefix(lexer.DATE_NOW, p.parseDateNow)
	p.registerPrefix(lexer.DATETIME_LITERAL, p.parseDatetimeLiteral)
	p.registerPrefix(lexer.DURATION_LITERAL, p.parseDurationLiteral)
	p.registerPrefix(lexer.SQLITE_LITERAL, p.parseConnectionLiteral)
	p.registerPrefix(lexer.POSTGRES_LITERAL, p.parseConnectionLiteral)
	p.registerPrefix(lexer.MYSQL_LITERAL, p.parseConnectionLiteral)
	p.registerPrefix(lexer.SFTP_LITERAL, p.parseConnectionLiteral)
	p.registerPrefix(lexer.SHELL_LITERAL, p.parseConnectionLiteral)
	p.registerPrefix(lexer.DB_LITERAL, p.parseConnectionLiteral)
	p.registerPrefix(lexer.SEARCH_LITERAL, p.parseConnectionLiteral)
	p.registerPrefix(lexer.ENV_LITERAL, p.parseBuiltinGlobal)
	p.registerPrefix(lexer.ARGS_LITERAL, p.parseBuiltinGlobal)
	p.registerPrefix(lexer.PARAMS_LITERAL, p.parseBuiltinGlobal)
	p.registerPrefix(lexer.SCHEMA_LITERAL, p.parseSchemaDeclaration)
	p.registerPrefix(lexer.TABLE_LITERAL, p.parseTableLiteral)
	p.registerPrefix(lexer.QUERY_LITERAL, p.parseQueryExpression)
	p.registerPrefix(lexer.INSERT_LITERAL, p.parseInsertExpression)
	p.registerPrefix(lexer.UPDATE_LITERAL, p.parseUpdateExpression)
	p.registerPrefix(lexer.DELETE_LITERAL, p.parseDeleteExpression)
	p.registerPrefix(lexer.TRANSACTION_LIT, p.parseTransactionExpression)
	p.registerPrefix(lexer.MONEY, p.parseMoneyLiteral)
	p.registerPrefix(lexer.PATH_LITERAL, p.parsePathLiteral)
	p.registerPrefix(lexer.URL_LITERAL, p.parseUrlLiteral)
	p.registerPrefix(lexer.STDLIB_PATH, p.parseStdlibPathLiteral)
	p.registerPrefix(lexer.PATH_TEMPLATE, p.parsePathTemplateLiteral)
	p.registerPrefix(lexer.URL_TEMPLATE, p.parseUrlTemplateLiteral)
	p.registerPrefix(lexer.DATETIME_TEMPLATE, p.parseDatetimeTemplateLiteral)
	p.registerPrefix(lexer.TAG, p.parseTagLiteral)
	p.registerPrefix(lexer.TAG_START, p.parseTagPair)
	p.registerPrefix(lexer.BANG, p.parsePrefixExpression)
	p.registerPrefix(lexer.MINUS, p.parsePrefixExpression)
	p.registerPrefix(lexer.READ_FROM, p.parseReadExpression)
	p.registerPrefix(lexer.TRUE, p.parseBoolean)
	p.registerPrefix(lexer.FALSE, p.parseBoolean)
	p.registerPrefix(lexer.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(lexer.LBRACKET, p.parseSquareBracketArrayLiteral)
	p.registerPrefix(lexer.IF, p.parseIfExpression)
	p.registerPrefix(lexer.FUNCTION, p.parseFunctionLiteral)
	p.registerPrefix(lexer.FOR, p.parseForExpression)
	p.registerPrefix(lexer.TRY, p.parseTryExpression)
	p.registerPrefix(lexer.IMPORT, p.parseImportExpression)
	p.registerPrefix(lexer.LBRACE, p.parseDictionaryLiteral)
	p.registerPrefix(lexer.STOP, p.parseStopExpression)
	p.registerPrefix(lexer.SKIP, p.parseSkipExpression)

	// Initialize infix parse functions
	p.infixParseFns = make(map[lexer.TokenType]infixParseFn)
	p.registerInfix(lexer.PLUS, p.parseInfixExpression)
	p.registerInfix(lexer.MINUS, p.parseInfixExpression)
	p.registerInfix(lexer.SLASH, p.parseInfixExpression)
	p.registerInfix(lexer.ASTERISK, p.parseInfixExpression)
	p.registerInfix(lexer.PERCENT, p.parseInfixExpression)
	p.registerInfix(lexer.EQ, p.parseInfixExpression)
	p.registerInfix(lexer.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(lexer.LT, p.parseInfixExpression)
	p.registerInfix(lexer.GT, p.parseInfixExpression)
	p.registerInfix(lexer.LTE, p.parseInfixExpression)
	p.registerInfix(lexer.GTE, p.parseInfixExpression)
	p.registerInfix(lexer.AND, p.parseInfixExpression)
	p.registerInfix(lexer.OR, p.parseInfixExpression)
	p.registerInfix(lexer.NULLISH, p.parseInfixExpression)
	p.registerInfix(lexer.MATCH, p.parseInfixExpression)
	p.registerInfix(lexer.NOT_MATCH, p.parseInfixExpression)
	p.registerInfix(lexer.IN, p.parseInfixExpression)
	p.registerInfix(lexer.BANG, p.parseNotInExpression) // for 'not in' operator
	p.registerInfix(lexer.PLUSPLUS, p.parseInfixExpression)
	p.registerInfix(lexer.RANGE, p.parseInfixExpression)
	p.registerInfix(lexer.QUERY_ONE, p.parseInfixExpression)      // Database operators
	p.registerInfix(lexer.QUERY_MANY, p.parseInfixExpression)     // Database operators
	p.registerInfix(lexer.EXECUTE, p.parseInfixExpression)        // Database operators
	p.registerInfix(lexer.EXECUTE_WITH, p.parseExecuteExpression) // Process execution operator
	// Note: COMMA is not registered as infix - arrays must use [1,2,3] syntax
	p.registerInfix(lexer.LPAREN, p.parseCallExpression)
	p.registerInfix(lexer.LBRACKET, p.parseIndexOrSliceExpression)
	p.registerInfix(lexer.DOT, p.parseDotExpression)

	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

// Errors returns parser errors as strings (convenience method for tests).
// Prefer StructuredErrors() for production code.
func (p *Parser) Errors() []string {
	result := make([]string, len(p.structuredErrors))
	for i, err := range p.structuredErrors {
		if err.Line > 0 {
			result[i] = fmt.Sprintf("line %d, column %d: %s", err.Line, err.Column, err.Message)
		} else {
			result[i] = err.Message
		}
	}
	return result
}

// StructuredErrors returns parser errors as structured ParsleyError objects.
func (p *Parser) StructuredErrors() []*perrors.ParsleyError {
	return p.structuredErrors
}

// addError adds a structured error.
// Only the first error is recorded - subsequent errors are usually cascading noise.
func (p *Parser) addError(msg string, line, column int) {
	// Only keep the first error
	if len(p.structuredErrors) > 0 {
		return
	}

	// Add structured error
	p.structuredErrors = append(p.structuredErrors, &perrors.ParsleyError{
		Class:   perrors.ClassParse,
		Message: msg,
		Line:    line,
		Column:  column,
	})
}

// addStructuredError adds a structured error from the catalog.
// Only the first error is recorded - subsequent errors are usually cascading noise.
func (p *Parser) addStructuredError(code string, line, column int, data map[string]any) {
	// Only keep the first error
	if len(p.structuredErrors) > 0 {
		return
	}

	perr := perrors.NewWithPosition(code, line, column, data)

	// Add structured error
	p.structuredErrors = append(p.structuredErrors, perr)
}

// addErrorWithHints adds an error with hints.
// Only the first error is recorded - subsequent errors are usually cascading noise.
func (p *Parser) addErrorWithHints(msg string, line, column int, hints ...string) {
	// Only keep the first error
	if len(p.structuredErrors) > 0 {
		return
	}

	// Add structured error
	p.structuredErrors = append(p.structuredErrors, &perrors.ParsleyError{
		Class:   perrors.ClassParse,
		Message: msg,
		Hints:   hints,
		Line:    line,
		Column:  column,
	})
}

// registerPrefix registers a prefix parse function
func (p *Parser) registerPrefix(tokenType lexer.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

// registerInfix registers an infix parse function
func (p *Parser) registerInfix(tokenType lexer.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

// nextToken advances prevToken, curToken, and peekToken
func (p *Parser) nextToken() {
	p.prevToken = p.curToken
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

// ParseProgram parses the program and returns the AST
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

// parseStatement parses statements
func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case lexer.EXPORT:
		return p.parseExportStatement()
	case lexer.LET:
		return p.parseLetStatement(false)
	case lexer.RETURN:
		return p.parseReturnStatement()
	case lexer.CHECK:
		return p.parseCheckStatement()
	case lexer.STOP:
		return p.parseStopStatement()
	case lexer.SKIP:
		return p.parseSkipStatement()
	case lexer.LBRACE:
		// Check if this is a dictionary destructuring assignment
		// We need to look ahead to see if this is {a, b} = ... or just a dict literal
		// Save complete state including lexer position for proper backtracking
		savedCur := p.curToken
		savedPeek := p.peekToken
		savedPrev := p.prevToken
		savedStructuredErrors := len(p.structuredErrors)
		savedLexerState := p.l.SaveState()

		stmt := p.parseDictDestructuringAssignment()

		// If parsing failed (no = found), restore and parse as expression
		if stmt == nil || len(p.structuredErrors) > savedStructuredErrors {
			p.curToken = savedCur
			p.peekToken = savedPeek
			p.prevToken = savedPrev
			p.structuredErrors = p.structuredErrors[:savedStructuredErrors]
			p.l.RestoreState(savedLexerState)
			return p.parseExpressionStatement()
		}
		return stmt
	case lexer.IDENT:
		// Check for common keyword typos when identifier is followed by another identifier
		// This pattern (ident ident) is usually a mistake - likely a typo of a keyword
		if p.peekTokenIs(lexer.IDENT) {
			if hint := p.checkKeywordTypo(p.curToken.Literal); hint != "" {
				p.addError(hint, p.curToken.Line, p.curToken.Column)
				return nil
			}
		}
		// Check if this is an assignment statement (= or <== or <=/= or <=?=> or <=??=> or <=!=>)
		if p.peekTokenIs(lexer.ASSIGN) || p.peekTokenIs(lexer.READ_FROM) || p.peekTokenIs(lexer.FETCH_FROM) || p.peekTokenIs(lexer.QUERY_ONE) || p.peekTokenIs(lexer.QUERY_MANY) || p.peekTokenIs(lexer.EXECUTE) {
			return p.parseAssignmentStatement(false)
		}
		// Check for potential destructuring: IDENT followed by COMMA
		// We need to peek further to determine if this is `x,y = ...` or just `x,y` expression
		// For now, try parsing as assignment if comma follows
		if p.peekTokenIs(lexer.COMMA) {
			// Tentatively parse as destructuring assignment
			savedCur := p.curToken
			savedPeek := p.peekToken
			savedPrev := p.prevToken
			savedStructuredErrors := len(p.structuredErrors)
			savedLexerState := p.l.SaveState()

			stmt := p.parseAssignmentStatement(false)

			// If parsing failed (no = found), restore and parse as expression
			if stmt == nil || len(p.structuredErrors) > savedStructuredErrors {
				p.curToken = savedCur
				p.peekToken = savedPeek
				p.prevToken = savedPrev
				p.structuredErrors = p.structuredErrors[:savedStructuredErrors]
				p.l.RestoreState(savedLexerState)
				return p.parseExpressionStatement()
			}
			return stmt
		}
		// Otherwise, treat as expression statement
		return p.parseExpressionStatement()
	default:
		return p.parseExpressionStatement()
	}
}

// parseExportStatement parses export statements:
// - export let x = 5
// - export x = 5
// - export Name (bare export of already-defined binding)
// - export @schema Name { ... }
func (p *Parser) parseExportStatement() ast.Statement {
	exportToken := p.curToken

	// Move past 'export'
	p.nextToken()

	// Check if next is 'let'
	if p.curTokenIs(lexer.LET) {
		return p.parseLetStatement(true)
	}

	// Handle 'export @schema Name { ... }'
	if p.curTokenIs(lexer.SCHEMA_LITERAL) {
		schema := p.parseSchemaDeclaration()
		if schema == nil {
			return nil
		}
		// Mark schema for export and wrap in expression statement
		if schemaDecl, ok := schema.(*ast.SchemaDeclaration); ok {
			schemaDecl.Export = true
		}
		return &ast.ExpressionStatement{Token: exportToken, Expression: schema}
	}

	// Handle 'export x = 5' (assignment) or 'export x' (bare export)
	if p.curTokenIs(lexer.IDENT) {
		// Peek ahead to see if there's an assignment
		if p.peekTokenIs(lexer.ASSIGN) {
			return p.parseAssignmentStatement(true)
		}
		// Bare export: 'export Name'
		name := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		return &ast.ExportNameStatement{Token: exportToken, Name: name}
	}

	// Handle 'export {a, b} = ...' (dict destructuring)
	if p.curTokenIs(lexer.LBRACE) {
		// Save state for backtracking
		savedCur := p.curToken
		savedPeek := p.peekToken
		savedPrev := p.prevToken
		savedStructuredErrors := len(p.structuredErrors)
		savedLexerState := p.l.SaveState()

		stmt := p.parseDictDestructuringAssignment()

		// If parsing failed, restore and report error
		if stmt == nil || len(p.structuredErrors) > savedStructuredErrors {
			p.curToken = savedCur
			p.peekToken = savedPeek
			p.prevToken = savedPrev
			p.structuredErrors = p.structuredErrors[:savedStructuredErrors]
			p.l.RestoreState(savedLexerState)
			p.peekError(lexer.LET)
			return nil
		}

		// Mark as export
		if assignStmt, ok := stmt.(*ast.AssignmentStatement); ok {
			assignStmt.Export = true
		}
		return stmt
	}

	p.peekError(lexer.LET)
	return nil
}

// parseLetStatement parses let statements
func (p *Parser) parseLetStatement(export bool) ast.Statement {
	letToken := p.curToken

	// Check for dictionary destructuring pattern
	if p.peekTokenIs(lexer.LBRACE) {
		p.nextToken() // move to '{'
		dictPattern := p.parseDictDestructuringPattern()
		if dictPattern == nil {
			return nil
		}

		// Check for <== (read statement) or <=/= (fetch statement) or = (regular let)
		if p.peekTokenIs(lexer.READ_FROM) {
			p.nextToken() // consume <==
			readStmt := &ast.ReadStatement{
				Token:       p.curToken,
				DictPattern: dictPattern,
				IsLet:       true,
			}
			p.nextToken()
			readStmt.Source = p.parseExpression(LOWEST)
			if p.peekTokenIs(lexer.SEMICOLON) {
				p.nextToken()
			}
			return readStmt
		}

		if p.peekTokenIs(lexer.FETCH_FROM) {
			p.nextToken() // consume <=/=
			fetchStmt := &ast.FetchStatement{
				Token:       p.curToken,
				DictPattern: dictPattern,
				IsLet:       true,
			}
			p.nextToken()
			fetchStmt.Source = p.parseExpression(LOWEST)
			if p.peekTokenIs(lexer.SEMICOLON) {
				p.nextToken()
			}
			return fetchStmt
		}

		if !p.expectPeek(lexer.ASSIGN) {
			return nil
		}

		stmt := &ast.LetStatement{Token: letToken, Export: export}
		stmt.DictPattern = dictPattern
		p.nextToken()
		stmt.Value = p.parseExpression(LOWEST)

		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}

		return stmt
	}

	// Check for array destructuring pattern with brackets: let [a, b, c] = [1, 2, 3] or let [a, ...rest] = arr
	if p.peekTokenIs(lexer.LBRACKET) {
		p.nextToken() // move to '['

		// Parse array destructuring pattern
		arrayPattern := p.parseArrayDestructuringPattern()
		if arrayPattern == nil {
			return nil
		}

		// Check for <== (read statement) or <=/= (fetch statement) or = (regular let)
		if p.peekTokenIs(lexer.READ_FROM) {
			p.nextToken() // consume <==
			readStmt := &ast.ReadStatement{
				Token:        p.curToken,
				ArrayPattern: arrayPattern,
				IsLet:        true,
			}
			p.nextToken()
			readStmt.Source = p.parseExpression(LOWEST)
			if p.peekTokenIs(lexer.SEMICOLON) {
				p.nextToken()
			}
			return readStmt
		}

		if p.peekTokenIs(lexer.FETCH_FROM) {
			p.nextToken() // consume <=/=
			fetchStmt := &ast.FetchStatement{
				Token:        p.curToken,
				ArrayPattern: arrayPattern,
				IsLet:        true,
			}
			p.nextToken()
			fetchStmt.Source = p.parseExpression(LOWEST)
			if p.peekTokenIs(lexer.SEMICOLON) {
				p.nextToken()
			}
			return fetchStmt
		}

		if !p.expectPeek(lexer.ASSIGN) {
			return nil
		}

		stmt := &ast.LetStatement{Token: letToken, Export: export}
		stmt.ArrayPattern = arrayPattern
		p.nextToken()
		stmt.Value = p.parseExpression(LOWEST)

		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}

		return stmt
	}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	// Single identifier only (no comma-separated destructuring - use [a, b] = ... syntax instead)
	name := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Check for <== (read statement) or <=/= (fetch statement)
	if p.peekTokenIs(lexer.READ_FROM) {
		p.nextToken() // consume <==
		readStmt := &ast.ReadStatement{
			Token: p.curToken,
			Name:  name,
			IsLet: true,
		}
		p.nextToken()
		readStmt.Source = p.parseExpression(LOWEST)
		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}
		return readStmt
	}

	if p.peekTokenIs(lexer.FETCH_FROM) {
		p.nextToken() // consume <=/=
		fetchStmt := &ast.FetchStatement{
			Token: p.curToken,
			Name:  name,
			IsLet: true,
		}
		p.nextToken()
		fetchStmt.Source = p.parseExpression(LOWEST)
		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}
		return fetchStmt
	}

	// Regular let statement
	stmt := &ast.LetStatement{Token: letToken, Export: export}
	stmt.Name = name

	if !p.expectPeek(lexer.ASSIGN) {
		return nil
	}

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseAssignmentStatement parses assignment statements like 'x = 5;' or 'x <== file(...)'
func (p *Parser) parseAssignmentStatement(export bool) ast.Statement {
	firstToken := p.curToken

	// Single identifier only (no comma-separated destructuring - use [a, b] = ... syntax instead)
	name := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Check for <== (read statement) or <=/= (fetch statement)
	if p.peekTokenIs(lexer.READ_FROM) {
		p.nextToken() // consume <==
		readStmt := &ast.ReadStatement{
			Token: p.curToken,
			Name:  name,
			IsLet: false,
		}
		p.nextToken()
		readStmt.Source = p.parseExpression(LOWEST)
		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}
		return readStmt
	}

	if p.peekTokenIs(lexer.FETCH_FROM) {
		p.nextToken() // consume <=/=
		fetchStmt := &ast.FetchStatement{
			Token: p.curToken,
			Name:  name,
			IsLet: false,
		}
		p.nextToken()
		fetchStmt.Source = p.parseExpression(LOWEST)
		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}
		return fetchStmt
	}

	// Regular assignment
	stmt := &ast.AssignmentStatement{Token: firstToken, Export: export}
	stmt.Name = name

	if !p.expectPeek(lexer.ASSIGN) {
		return nil
	}

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseDictDestructuringAssignment parses dictionary destructuring assignments like '{a, b} = dict;' or '{a, b} <== file(...)'
func (p *Parser) parseDictDestructuringAssignment() ast.Statement {
	braceToken := p.curToken // the '{' token

	dictPattern := p.parseDictDestructuringPattern()
	if dictPattern == nil {
		return nil
	}

	// Check for <== (read statement)
	// Check for <== (read statement) or <=/= (fetch statement)
	if p.peekTokenIs(lexer.READ_FROM) {
		p.nextToken() // consume <==
		readStmt := &ast.ReadStatement{
			Token:       p.curToken,
			DictPattern: dictPattern,
			IsLet:       false,
		}
		p.nextToken()
		readStmt.Source = p.parseExpression(LOWEST)
		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}
		return readStmt
	}

	if p.peekTokenIs(lexer.FETCH_FROM) {
		p.nextToken() // consume <=/=
		fetchStmt := &ast.FetchStatement{
			Token:       p.curToken,
			DictPattern: dictPattern,
			IsLet:       false,
		}
		p.nextToken()
		fetchStmt.Source = p.parseExpression(LOWEST)
		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}
		return fetchStmt
	}

	// Regular assignment
	if !p.expectPeek(lexer.ASSIGN) {
		return nil
	}

	stmt := &ast.AssignmentStatement{Token: braceToken}
	stmt.DictPattern = dictPattern

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseReturnStatement parses return statements
func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken()

	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseCheckStatement parses check statements: check CONDITION else VALUE
func (p *Parser) parseCheckStatement() *ast.CheckStatement {
	stmt := &ast.CheckStatement{Token: p.curToken}

	p.nextToken() // move past 'check'

	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.ELSE) {
		return nil
	}

	p.nextToken() // move past 'else'

	stmt.ElseValue = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseStopStatement parses stop statements
func (p *Parser) parseStopStatement() *ast.StopStatement {
	stmt := &ast.StopStatement{Token: p.curToken}

	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseSkipStatement parses skip statements
func (p *Parser) parseSkipStatement() *ast.SkipStatement {
	stmt := &ast.SkipStatement{Token: p.curToken}

	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseStopExpression parses stop as an expression (for use in: if (cond) stop)
func (p *Parser) parseStopExpression() ast.Expression {
	return &ast.StopStatement{Token: p.curToken}
}

// parseSkipExpression parses skip as an expression (for use in: if (cond) skip)
func (p *Parser) parseSkipExpression() ast.Expression {
	return &ast.SkipStatement{Token: p.curToken}
}

// parseExpressionStatement parses expression statements
func (p *Parser) parseExpressionStatement() ast.Statement {
	firstToken := p.curToken

	expr := p.parseExpression(LOWEST)

	// Check for index/property assignment: expr[key] = value or expr.prop = value
	if p.peekTokenIs(lexer.ASSIGN) {
		if p.isAssignableExpression(expr) {
			p.nextToken() // consume '='
			assignToken := p.curToken
			p.nextToken() // move to value expression
			value := p.parseExpression(LOWEST)

			if p.peekTokenIs(lexer.SEMICOLON) {
				p.nextToken()
			}

			return &ast.IndexAssignmentStatement{
				Token:  assignToken,
				Target: expr,
				Value:  value,
			}
		}
	}

	// Check for write operators ==> or ==>>
	if p.peekTokenIs(lexer.WRITE_TO) || p.peekTokenIs(lexer.APPEND_TO) {
		p.nextToken() // consume ==> or ==>>
		writeStmt := &ast.WriteStatement{
			Token:  p.curToken,
			Value:  expr,
			Append: p.curToken.Type == lexer.APPEND_TO,
		}
		p.nextToken() // move to target expression
		writeStmt.Target = p.parseExpression(LOWEST)

		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}
		return writeStmt
	}

	stmt := &ast.ExpressionStatement{Token: firstToken}
	stmt.Expression = expr

	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// isAssignableExpression returns true if the expression can be assigned to
func (p *Parser) isAssignableExpression(expr ast.Expression) bool {
	switch expr.(type) {
	case *ast.IndexExpression, *ast.DotExpression:
		return true
	}
	return false
}

// parseExpression parses expressions using Pratt parsing
func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}

	leftExp := prefix()

	for !p.peekTokenIs(lexer.SEMICOLON) && precedence < p.peekPrecedence() {
		// Special handling for DOT: only treat as infix if followed by IDENT or AS keyword
		// This allows standalone DOT to be used as a terminal operator in DSL queries
		// AS is allowed since 'as' can be a method name like dict.as(Schema)
		if p.peekTokenIs(lexer.DOT) {
			peekedAhead := p.l.PeekToken()
			if peekedAhead.Type != lexer.IDENT && peekedAhead.Type != lexer.AS {
				return leftExp
			}
		}

		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

// parseExpressionUntilBrace parses an expression but stops when we see '{'
// Used for if/for without parentheses: if condition { } or for x in arr { }
func (p *Parser) parseExpressionUntilBrace() ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}

	leftExp := prefix()

	// Stop at semicolon, LBRACE, or when precedence is exhausted
	for !p.peekTokenIs(lexer.SEMICOLON) && !p.peekTokenIs(lexer.LBRACE) && LOWEST < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

// parseExpressionUntil parses an expression but stops when the stopCondition returns true
// Used for parsing expressions in contexts where specific tokens signal the end
func (p *Parser) parseExpressionUntil(precedence int, stopCondition func() bool) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}

	leftExp := prefix()

	// Stop at semicolon, when stop condition is met, or when precedence is exhausted
	for !p.peekTokenIs(lexer.SEMICOLON) && !stopCondition() && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

// Parse functions for different expression types
func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		p.addError(fmt.Sprintf("could not parse %q as integer", p.curToken.Literal), p.curToken.Line, p.curToken.Column)
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	lit := &ast.FloatLiteral{Token: p.curToken}

	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.addError(fmt.Sprintf("could not parse %q as float", p.curToken.Literal), p.curToken.Line, p.curToken.Column)
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseTemplateLiteral() ast.Expression {
	return &ast.TemplateLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseRawTemplateLiteral() ast.Expression {
	return &ast.RawTemplateLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseRegexLiteral() ast.Expression {
	// Token.Literal is in the form "/pattern/flags"
	literal := p.curToken.Literal
	if len(literal) < 2 || literal[0] != '/' {
		p.addError(fmt.Sprintf("invalid regex literal: %s", literal), p.curToken.Line, p.curToken.Column)
		return nil
	}

	// Find the closing / by looking from the end backwards
	// This handles /pattern/ and /pattern/flags
	lastSlash := strings.LastIndex(literal[1:], "/")
	if lastSlash == -1 {
		p.addError(fmt.Sprintf("unterminated regex literal: %s", literal), p.curToken.Line, p.curToken.Column)
		return nil
	}
	lastSlash++ // adjust for the slice offset

	pattern := literal[1:lastSlash]
	flags := ""
	if lastSlash+1 < len(literal) {
		flags = literal[lastSlash+1:]
	}

	return &ast.RegexLiteral{
		Token:   p.curToken,
		Pattern: pattern,
		Flags:   flags,
	}
}

func (p *Parser) parseDatetimeNowLiteral(kind string) ast.Expression {
	return &ast.DatetimeNowLiteral{
		Token: p.curToken,
		Kind:  kind,
	}
}

func (p *Parser) parseDatetimeNow() ast.Expression {
	return p.parseDatetimeNowLiteral("datetime")
}

func (p *Parser) parseTimeNow() ast.Expression {
	return p.parseDatetimeNowLiteral("time")
}

func (p *Parser) parseDateNow() ast.Expression {
	return p.parseDatetimeNowLiteral("date")
}

func (p *Parser) parseDatetimeLiteral() ast.Expression {
	// Token.Literal contains the ISO-8601 datetime string (without the @ prefix)
	// Determine the kind based on the literal format:
	// - Contains 'T' -> "datetime"
	// - Starts with 4-digit year (YYYY-) -> "date"
	// - Time HH:MM -> "time"
	// - Time HH:MM:SS -> "time_seconds" (to preserve precision)
	literal := p.curToken.Literal
	kind := "datetime" // default

	if len(literal) >= 5 && literal[4] == '-' {
		// Starts with YYYY- (4 digits then dash)
		if strings.Contains(literal, "T") {
			kind = "datetime"
		} else {
			kind = "date"
		}
	} else if len(literal) >= 3 && strings.Contains(literal[:3], ":") {
		// Time-only: starts with H: or HH:
		// Check if seconds are present (HH:MM:SS has 8 chars)
		colonCount := strings.Count(literal, ":")
		if colonCount >= 2 {
			kind = "time_seconds"
		} else {
			kind = "time"
		}
	}

	return &ast.DatetimeLiteral{
		Token: p.curToken,
		Value: p.curToken.Literal,
		Kind:  kind,
	}
}

func (p *Parser) parseDurationLiteral() ast.Expression {
	// Token.Literal contains the duration string (without the @ prefix)
	return &ast.DurationLiteral{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}
}

func (p *Parser) parseMoneyLiteral() ast.Expression {
	// Token.Literal contains the money literal (e.g., "USD#12.34")
	// Parse the currency, amount, and scale from the literal
	literal := p.curToken.Literal

	// Find the # separator
	hashIdx := strings.Index(literal, "#")
	if hashIdx == -1 {
		p.addError(fmt.Sprintf("invalid money literal: %s", literal), p.curToken.Line, p.curToken.Column)
		return nil
	}

	currency := literal[:hashIdx]
	numStr := literal[hashIdx+1:]

	// Calculate scale from decimal places
	scale := int8(0)
	dotIdx := strings.Index(numStr, ".")
	if dotIdx >= 0 {
		scale = int8(len(numStr) - dotIdx - 1)
	}

	// Parse amount as integer (in smallest unit)
	amount := parseMoneyAmountFromString(numStr, scale)

	return &ast.MoneyLiteral{
		Token:    p.curToken,
		Currency: currency,
		Amount:   amount,
		Scale:    scale,
	}
}

// parseMoneyAmountFromString converts a number string to an integer amount in smallest units
func parseMoneyAmountFromString(numStr string, scale int8) int64 {
	var result int64
	var seenDot bool
	var fracDigits int8
	var negative bool

	for i, ch := range numStr {
		if ch == '-' && i == 0 {
			negative = true
			continue
		}
		if ch == '.' {
			seenDot = true
			continue
		}
		if ch >= '0' && ch <= '9' {
			result = result*10 + int64(ch-'0')
			if seenDot {
				fracDigits++
			}
		}
	}

	// Pad with zeros if needed
	for fracDigits < scale {
		result *= 10
		fracDigits++
	}

	if negative {
		result = -result
	}

	return result
}

func (p *Parser) parseConnectionLiteral() ast.Expression {
	kind := ""
	switch p.curToken.Type {
	case lexer.SQLITE_LITERAL:
		kind = "sqlite"
	case lexer.POSTGRES_LITERAL:
		kind = "postgres"
	case lexer.MYSQL_LITERAL:
		kind = "mysql"
	case lexer.SFTP_LITERAL:
		kind = "sftp"
	case lexer.SHELL_LITERAL:
		kind = "shell"
	case lexer.DB_LITERAL:
		kind = "db"
	case lexer.SEARCH_LITERAL:
		kind = "search"
	}

	return &ast.ConnectionLiteral{
		Token: p.curToken,
		Kind:  kind,
	}
}

// parseBuiltinGlobal parses @env, @args, and @params as identifier lookups.
// These tokens resolve to built-in variables in the environment.
func (p *Parser) parseBuiltinGlobal() ast.Expression {
	// Return an identifier that will be looked up in the environment
	return &ast.Identifier{
		Token: p.curToken,
		Value: p.curToken.Literal, // "@env", "@args", or "@params"
	}
}

func (p *Parser) parsePathLiteral() ast.Expression {
	// Token.Literal contains the path string (without the @ prefix)
	return &ast.PathLiteral{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}
}

func (p *Parser) parseUrlLiteral() ast.Expression {
	// Token.Literal contains the URL string (without the @ prefix)
	return &ast.UrlLiteral{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}
}

func (p *Parser) parseStdlibPathLiteral() ast.Expression {
	// Token.Literal contains the stdlib path (e.g., "std/table")
	return &ast.StdlibPathLiteral{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}
}

func (p *Parser) parsePathTemplateLiteral() ast.Expression {
	// Token.Literal contains the template content (without the @( and ) delimiters)
	return &ast.PathTemplateLiteral{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}
}

func (p *Parser) parseUrlTemplateLiteral() ast.Expression {
	// Token.Literal contains the template content (without the @( and ) delimiters)
	return &ast.UrlTemplateLiteral{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}
}

func (p *Parser) parseDatetimeTemplateLiteral() ast.Expression {
	// Token.Literal contains the template content (without the @( and ) delimiters)
	return &ast.DatetimeTemplateLiteral{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}
}

func (p *Parser) parseTagLiteral() ast.Expression {
	tag := &ast.TagLiteral{
		Token:   p.curToken,
		Raw:     p.curToken.Literal,
		Spreads: extractSpreadExpressions(p.curToken.Literal),
	}
	return tag
}

func (p *Parser) parseTagPair() ast.Expression {
	tagExpr := &ast.TagPairExpression{
		Token:    p.curToken,
		Contents: []ast.Node{},
	}

	// Save opening tag position for error reporting
	openingLine := p.curToken.Line
	openingColumn := p.curToken.Column

	// Parse tag name and props from the TAG_START token literal
	// Format: "tagname attr1="value" attr2={expr}" or empty string for <>
	raw := p.curToken.Literal
	tagExpr.Name, tagExpr.Props = parseTagNameAndProps(raw)

	// Extract spread expressions from props
	tagExpr.Spreads = extractSpreadExpressions(tagExpr.Props)

	// Parse tag contents
	p.nextToken()
	tagExpr.Contents = p.parseTagContents(tagExpr.Name)

	// Current token should be TAG_END
	if !p.curTokenIs(lexer.TAG_END) {
		// Check if this is a known void/singleton element that should be self-closing
		if isVoidElement(tagExpr.Name) {
			p.addStructuredError("PARSE-0008", openingLine, openingColumn, map[string]any{"Tag": tagExpr.Name})
		} else {
			p.addError(fmt.Sprintf("expected closing tag </%s>, got %s",
				tagExpr.Name, tokenTypeToReadableName(p.curToken.Type)), openingLine, openingColumn)
		}
		return nil
	}

	// Validate closing tag matches opening tag
	closingName := p.curToken.Literal
	if closingName != tagExpr.Name {
		p.addError(fmt.Sprintf("mismatched tags: opening <%s> but closing </%s>",
			tagExpr.Name, closingName), p.curToken.Line, p.curToken.Column)
		return nil
	}

	return tagExpr
}

// parseTagContents parses the contents between opening and closing tags
// In the new syntax, tag contents are code (expressions/statements), not raw text.
// Text must be quoted: <p>"Hello"</p>
func (p *Parser) parseTagContents(tagName string) []ast.Node {
	var contents []ast.Node

	// Check if this is a raw text tag (style/script) which uses @{} for interpolation
	isRawTextTag := tagName == "style" || tagName == "script"

	for !p.curTokenIs(lexer.EOF) {
		// If we hit a closing tag, only stop when it matches the current tag
		if p.curTokenIs(lexer.TAG_END) {
			if p.curToken.Literal == tagName {
				break
			}
			// Closing tag for a nested element bubbled up; skip it and continue parsing
			p.nextToken()
			continue
		}

		switch p.curToken.Type {
		case lexer.TAG_TEXT:
			// Raw text content - still supported for style/script tags
			textNode := &ast.TextNode{
				Token: p.curToken,
				Value: p.curToken.Literal,
			}
			contents = append(contents, textNode)
			p.nextToken()

		case lexer.TAG_START:
			// Nested tag pair
			nestedTag := p.parseTagPair()
			if nestedTag != nil {
				contents = append(contents, nestedTag)
			}
			p.nextToken()

		case lexer.TAG:
			// Singleton tag
			singletonTag := p.parseTagLiteral()
			if singletonTag != nil {
				contents = append(contents, singletonTag)
			}
			p.nextToken()

		case lexer.LBRACE:
			// LBRACE in tag contents - this is from @{} interpolation in style/script tags
			if isRawTextTag {
				// Parse interpolation block for raw text tags
				// The lexer gave us { from @{, now parse the expression and expect }
				startToken := p.curToken
				p.nextToken() // move past {

				var stmts []ast.Statement
				for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
					stmt := p.parseStatement()
					if stmt != nil {
						stmts = append(stmts, stmt)
					}
					p.nextToken()
				}

				block := &ast.InterpolationBlock{
					Token:      startToken,
					Statements: stmts,
				}
				contents = append(contents, block)
				p.nextToken() // move past }
			} else {
				// For non-raw tags, { starts a dictionary literal - parse as expression
				stmt := p.parseStatement()
				if stmt != nil {
					if exprStmt, ok := stmt.(*ast.ExpressionStatement); ok {
						contents = append(contents, exprStmt.Expression)
					} else {
						block := &ast.InterpolationBlock{
							Token:      p.curToken,
							Statements: []ast.Statement{stmt},
						}
						contents = append(contents, block)
					}
				}
				// Always advance - loop start handles nested TAG_ENDs
				p.nextToken()
			}

		default:
			// Parse as a statement (expression, for loop, if statement, etc.)
			// This is the new behavior - code inside tags without { }
			stmt := p.parseStatement()
			if stmt != nil {
				// If it's an expression statement, add just the expression
				if exprStmt, ok := stmt.(*ast.ExpressionStatement); ok {
					contents = append(contents, exprStmt.Expression)
				} else {
					// For other statements (for, if, let), wrap in InterpolationBlock
					block := &ast.InterpolationBlock{
						Token:      p.curToken,
						Statements: []ast.Statement{stmt},
					}
					contents = append(contents, block)
				}
			}
			// Always advance - loop start handles nested TAG_ENDs
			p.nextToken()
		}
	}

	return contents
}

// parseTagNameAndProps splits raw tag content into name and props
// Examples: "div class=\"foo\"" -> ("div", "class=\"foo\"")
//
//	"" -> ("", "")
func parseTagNameAndProps(raw string) (string, string) {
	if raw == "" {
		return "", "" // empty grouping tag
	}

	// Find first space to separate name from props
	for i, ch := range raw {
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			return raw[:i], strings.TrimSpace(raw[i:])
		}
	}

	// No spaces, all tag name
	return raw, ""
}

// extractSpreadExpressions parses a raw props string to find spread expressions like "...attrs"
// Returns a slice of SpreadExpr nodes
func extractSpreadExpressions(raw string) []*ast.SpreadExpr {
	spreads := []*ast.SpreadExpr{}

	// Simple state machine to scan for spreads
	i := 0
	for i < len(raw) {
		// Skip whitespace
		for i < len(raw) && (raw[i] == ' ' || raw[i] == '\t' || raw[i] == '\n' || raw[i] == '\r') {
			i++
		}

		if i >= len(raw) {
			break
		}

		// Check for spread operator "..."
		if i+3 <= len(raw) && raw[i:i+3] == "..." {
			i += 3

			// Skip whitespace after ...
			for i < len(raw) && (raw[i] == ' ' || raw[i] == '\t' || raw[i] == '\n' || raw[i] == '\r') {
				i++
			}

			// Extract identifier
			start := i
			for i < len(raw) && isIdentChar(raw[i]) {
				i++
			}

			if i > start {
				identName := raw[start:i]
				spreads = append(spreads, &ast.SpreadExpr{
					Token: lexer.Token{Type: lexer.DOTDOTDOT, Literal: "..."},
					Expression: &ast.Identifier{
						Token: lexer.Token{Type: lexer.IDENT, Literal: identName},
						Value: identName,
					},
				})
			}
		} else {
			// Skip to next whitespace or potential spread
			for i < len(raw) && raw[i] != ' ' && raw[i] != '\t' && raw[i] != '\n' && raw[i] != '\r' {
				// Also check for quotes - need to skip content inside quotes
				if raw[i] == '"' {
					i++
					for i < len(raw) && raw[i] != '"' {
						if raw[i] == '\\' && i+1 < len(raw) {
							i += 2 // skip escaped character
						} else {
							i++
						}
					}
					if i < len(raw) {
						i++ // skip closing quote
					}
				} else if raw[i] == '{' {
					// Skip interpolation expressions
					depth := 1
					i++
					for i < len(raw) && depth > 0 {
						if raw[i] == '{' {
							depth++
						} else if raw[i] == '}' {
							depth--
						}
						i++
					}
				} else {
					i++
				}
			}
		}
	}

	return spreads
}

// isIdentChar returns true if the character can be part of an identifier
func isIdentChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

// isVoidElement returns true if the tag is a void/singleton element that must be self-closing
func isVoidElement(tag string) bool {
	voidElements := map[string]bool{
		"area": true, "base": true, "br": true, "col": true, "embed": true,
		"hr": true, "img": true, "input": true, "link": true, "meta": true,
		"param": true, "source": true, "track": true, "wbr": true,
	}
	return voidElements[tag]
}

func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(lexer.TRUE)}
}

func (p *Parser) parseTryExpression() ast.Expression {
	expression := &ast.TryExpression{Token: p.curToken}

	p.nextToken()

	// Parse the expression after 'try'
	call := p.parseExpression(PREFIX)

	// Validate that it's a call expression (function or method call)
	switch call.(type) {
	case *ast.CallExpression:
		// Direct function call: try func()
		expression.Call = call
	default:
		// Not a call expression - give helpful error
		p.addErrorWithHints(
			"Try requires a function or method call",
			expression.Token.Line, expression.Token.Column,
			"try can only wrap function calls like: try func()",
			"try can only wrap method calls like: try obj.method()",
		)
		return nil
	}

	return expression
}

// parseImportExpression parses import expressions:
//
//	import @std/math           -> binds to "math"
//	import @./local/file       -> binds to "file"
//	import @std/math as M      -> binds to "M"
//	import @(./path/{name})    -> dynamic import
func (p *Parser) parseImportExpression() ast.Expression {
	importToken := p.curToken

	p.nextToken() // consume 'import'

	// New syntax: import @path
	expression := &ast.ImportExpression{Token: importToken}

	// Parse the path - must be a path literal token (@std/..., @./..., etc.)
	// or a path template for dynamic imports (@(...))
	switch p.curToken.Type {
	case lexer.STDLIB_PATH:
		expression.Path = &ast.StdlibPathLiteral{
			Token: p.curToken,
			Value: p.curToken.Literal,
		}
		expression.BindName = extractBindName(p.curToken.Literal)
	case lexer.PATH_LITERAL:
		expression.Path = &ast.PathLiteral{
			Token: p.curToken,
			Value: p.curToken.Literal,
		}
		expression.BindName = extractBindName(p.curToken.Literal)
	case lexer.PATH_TEMPLATE:
		expression.Path = &ast.PathTemplateLiteral{
			Token: p.curToken,
			Value: p.curToken.Literal,
		}
		// Dynamic imports can't auto-bind (name unknown at parse time)
		expression.BindName = ""
	default:
		p.addErrorWithHints(
			"Expected path after import",
			p.curToken.Line, p.curToken.Column,
			"Use: import @std/math",
			"Use: import @./local/file",
		)
		return nil
	}

	// Check for optional 'as Alias'
	if p.peekTokenIs(lexer.AS) {
		p.nextToken() // consume 'as'
		p.nextToken() // move to alias identifier
		if !p.curTokenIs(lexer.IDENT) {
			p.addErrorWithHints(
				"Expected identifier after 'as'",
				p.curToken.Line, p.curToken.Column,
				"import @path as Alias - Alias must be an identifier",
			)
			return nil
		}
		expression.Alias = &ast.Identifier{
			Token: p.curToken,
			Value: p.curToken.Literal,
		}
		expression.BindName = p.curToken.Literal
	}

	return expression
}

// extractBindName extracts the binding name from a module path.
// "std/math" -> "math"
// "./components/Button" -> "Button"
// "../shared/utils" -> "utils"
func extractBindName(path string) string {
	// Remove file extension if present
	if idx := lastIndex(path, ".pars"); idx != -1 {
		path = path[:idx]
	}
	if idx := lastIndex(path, "."); idx != -1 && idx > lastIndex(path, "/") {
		path = path[:idx]
	}

	// Get the last segment
	if idx := lastIndex(path, "/"); idx != -1 {
		return path[idx+1:]
	}
	return path
}

// lastIndex returns the index of the last occurrence of substr in s, or -1 if not found.
func lastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()

	expression.Right = p.parseExpression(PREFIX)

	return expression
}

// parseReadExpression parses bare <== expressions like '<== file(...)'
func (p *Parser) parseReadExpression() ast.Expression {
	expression := &ast.ReadExpression{
		Token: p.curToken,
	}

	p.nextToken()

	expression.Source = p.parseExpression(PREFIX)

	return expression
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Left:     left,
		Operator: p.curToken.Literal,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

// parseNotInExpression handles the 'not in' compound operator.
// When 'not' appears in infix position and is followed by 'in',
// it creates a 'not in' operator. Otherwise, it's a syntax error.
func (p *Parser) parseNotInExpression(left ast.Expression) ast.Expression {
	notToken := p.curToken

	// Check if next token is 'in'
	if !p.peekTokenIs(lexer.IN) {
		p.addError(fmt.Sprintf("expected 'in' after 'not', got %s", p.peekToken.Type), p.peekToken.Line, p.peekToken.Column)
		return nil
	}

	p.nextToken() // consume 'in'

	expression := &ast.InfixExpression{
		Token:    notToken,
		Left:     left,
		Operator: "not in",
	}

	precedence := precedences[lexer.IN]
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

func (p *Parser) parseExecuteExpression(left ast.Expression) ast.Expression {
	expression := &ast.ExecuteExpression{
		Token:   p.curToken,
		Command: left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Input = p.parseExpression(precedence)

	return expression
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	openParen := p.curToken
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	// Check for arrow function syntax: (a, b) => ...
	// This is not supported in Parsley, but we can give a helpful error
	if p.peekTokenIs(lexer.COMMA) {
		// Looks like (expr, ...) - could be attempted arrow function
		// Save position for error reporting
		commaPos := p.peekToken

		// Skip to find ) and =>
		depth := 1
		foundArrow := false
		for depth > 0 {
			p.nextToken()
			if p.curTokenIs(lexer.LPAREN) {
				depth++
			} else if p.curTokenIs(lexer.RPAREN) {
				depth--
			} else if p.curTokenIs(lexer.EOF) {
				break
			}
		}
		// Check if next is => (represented as = followed by >)
		if p.curTokenIs(lexer.RPAREN) && p.peekTokenIs(lexer.ASSIGN) {
			// Save and peek ahead
			p.nextToken() // consume =
			if p.peekTokenIs(lexer.GT) {
				foundArrow = true
			}
		}

		if foundArrow {
			p.addErrorWithHints(
				"Arrow function syntax is not supported",
				openParen.Line, openParen.Column,
				"use fn(a, b) { a + b } instead",
				"single-param shorthand: arr.map(fn(x) { x * 2 })")
			return nil
		}

		// Not an arrow function, give normal error at comma
		p.addError(fmt.Sprintf("expected ')', got '%s'", commaPos.Literal), commaPos.Line, commaPos.Column)
		return nil
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	// Wrap in GroupedExpression so that (expr)(args) can call the result
	return &ast.GroupedExpression{
		Token: openParen,
		Inner: exp,
	}
}

func (p *Parser) parseSquareBracketArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}
	array.Elements = []ast.Expression{}

	// Check for empty array []
	if p.peekTokenIs(lexer.RBRACKET) {
		p.nextToken()
		return array
	}

	// Parse first element - use COMMA_PREC to prevent comma from being treated as infix
	p.nextToken()
	array.Elements = append(array.Elements, p.parseExpression(COMMA_PREC))

	// Parse remaining elements separated by commas
	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume comma
		// Check for trailing comma (next token is closing bracket)
		if p.peekTokenIs(lexer.RBRACKET) {
			break
		}
		p.nextToken() // move to next element
		array.Elements = append(array.Elements, p.parseExpression(COMMA_PREC))
	}

	if !p.expectPeek(lexer.RBRACKET) {
		return nil
	}

	return array
}

func (p *Parser) parseIfExpression() ast.Expression {
	expression := &ast.IfExpression{Token: p.curToken}

	// Parentheses are optional: if (cond) { } OR if cond { }
	hasParens := p.peekTokenIs(lexer.LPAREN)
	if hasParens {
		p.nextToken() // consume '('
		p.nextToken() // move to condition
	} else {
		p.nextToken() // move to condition
	}

	// Check for assignment in condition
	if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.ASSIGN) {
		varName := p.curToken.Literal
		p.addErrorWithHints("assignment is not allowed inside if condition",
			p.curToken.Line, p.curToken.Column,
			fmt.Sprintf("let %s = ...", varName),
			fmt.Sprintf("if (%s) { ... }", varName))
		return nil
	}

	// Parse condition - if no parens, stop at LBRACE
	if hasParens {
		expression.Condition = p.parseExpression(LOWEST)
		if !p.expectPeek(lexer.RPAREN) {
			return nil
		}
	} else {
		// Without parens, we need to parse until we hit '{'
		// Use a precedence that stops at LBRACE
		expression.Condition = p.parseExpressionUntilBrace()
	}

	if p.peekTokenIs(lexer.LBRACE) {
		// Block form: if (...) { ... }
		p.nextToken()
		expression.Consequence = p.parseBlockStatement()
	} else if !hasParens {
		// Without parens, block braces are required
		p.addErrorWithHints("if without parentheses requires braces",
			expression.Token.Line, expression.Token.Column,
			"if (condition) expr", "if condition { expr }")
		return nil
	} else {
		// Single statement/expression form: if (...) expr or if (...) return expr
		p.nextToken()

		// Check if it's a return statement
		if p.curTokenIs(lexer.RETURN) {
			stmt := p.parseReturnStatement()
			expression.Consequence = &ast.BlockStatement{
				Token:      p.curToken,
				Statements: []ast.Statement{stmt},
			}
		} else {
			// Regular expression
			stmt := &ast.ExpressionStatement{Token: p.curToken}
			stmt.Expression = p.parseExpression(LOWEST)
			expression.Consequence = &ast.BlockStatement{
				Token:      p.curToken,
				Statements: []ast.Statement{stmt},
			}
		}
	}

	// Optional else clause
	if p.peekTokenIs(lexer.ELSE) {
		p.nextToken()

		// Check for 'else if'
		if p.peekTokenIs(lexer.IF) {
			p.nextToken()
			// Recursively parse the if expression
			ifExpr := p.parseIfExpression()
			// Wrap it in a block statement
			expression.Alternative = &ast.BlockStatement{
				Token: p.curToken,
				Statements: []ast.Statement{
					&ast.ExpressionStatement{
						Token:      p.curToken,
						Expression: ifExpr,
					},
				},
			}
		} else if p.peekTokenIs(lexer.LBRACE) {
			// else { ... }
			p.nextToken()
			expression.Alternative = p.parseBlockStatement()
		} else {
			// Single statement/expression form
			p.nextToken()

			// Check if it's a return statement
			if p.curTokenIs(lexer.RETURN) {
				stmt := p.parseReturnStatement()
				expression.Alternative = &ast.BlockStatement{
					Token:      p.curToken,
					Statements: []ast.Statement{stmt},
				}
			} else {
				// Parse single expression as alternative
				stmt := &ast.ExpressionStatement{Token: p.curToken}
				stmt.Expression = p.parseExpression(LOWEST)
				expression.Alternative = &ast.BlockStatement{
					Token:      p.curToken,
					Statements: []ast.Statement{stmt},
				}
			}
		}
	}

	return expression
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	p.nextToken()

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

func (p *Parser) parseFunctionLiteral() ast.Expression {
	lit := &ast.FunctionLiteral{Token: p.curToken}

	// Check if parameters are present (fn(x) {...}) or omitted (fn {...})
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // consume LPAREN
		// Use new parameter parsing that supports destructuring
		lit.Params = p.parseFunctionParametersNew()

		if !p.expectPeek(lexer.LBRACE) {
			return nil
		}
	} else if p.peekTokenIs(lexer.LBRACE) {
		// No parameters - fn {} syntax
		p.nextToken() // consume LBRACE
		lit.Params = []*ast.FunctionParameter{}
	} else {
		p.peekError(lexer.LPAREN)
		return nil
	}

	lit.Body = p.parseBlockStatement()

	return lit
}

func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	identifiers := []*ast.Identifier{}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return identifiers
	}

	p.nextToken()

	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	identifiers = append(identifiers, ident)

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		identifiers = append(identifiers, ident)
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return identifiers
}

// parseFunctionParametersNew parses function parameters with destructuring support
func (p *Parser) parseFunctionParametersNew() []*ast.FunctionParameter {
	params := []*ast.FunctionParameter{}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return params
	}

	p.nextToken()

	// Parse first parameter
	param := p.parseFunctionParameter()
	if param != nil {
		params = append(params, param)
	}

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to next parameter
		param := p.parseFunctionParameter()
		if param != nil {
			params = append(params, param)
		}
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return params
}

// parseFunctionParameter parses a single function parameter (can be identifier, array, or dict pattern)
func (p *Parser) parseFunctionParameter() *ast.FunctionParameter {
	param := &ast.FunctionParameter{}

	switch p.curToken.Type {
	case lexer.LBRACE:
		// Dictionary destructuring pattern
		param.DictPattern = p.parseDictDestructuringPattern()
		return param

	case lexer.LBRACKET:
		// Array destructuring pattern
		param.ArrayPattern = p.parseArrayDestructuringPattern()
		return param

	case lexer.IDENT:
		// Simple identifier
		param.Ident = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		return param

	default:
		return nil
	}
}

// parseForExpression parses for expressions
// Two forms: for(array) func  OR  for(var in array) body
// Parentheses are optional: for (x in arr) { } OR for x in arr { }
func (p *Parser) parseForExpression() ast.Expression {
	expression := &ast.ForExpression{Token: p.curToken}

	// Parentheses are optional
	hasParens := p.peekTokenIs(lexer.LPAREN)
	if hasParens {
		p.nextToken() // consume '('
	}
	p.nextToken() // move to first token of expression

	// Check if this is the "for(var in array)" form
	// We need to peek ahead to see if there's an IN token or COMMA (for key,value syntax)
	// But only if current token is an identifier
	if p.curToken.Type == lexer.IDENT && (p.peekTokenIs(lexer.IN) || p.peekTokenIs(lexer.COMMA)) {
		// Parse: for(var in array) body OR for(key, value in dict) body
		expression.Variable = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

		// Check for comma (key, value in dict form)
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // move to COMMA
			if !p.expectPeek(lexer.IDENT) {
				return nil
			}
			expression.ValueVariable = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		}

		if !p.expectPeek(lexer.IN) {
			return nil
		}
		p.nextToken() // move past IN to array/dict expression

		// Parse array expression - if no parens, stop at LBRACE
		if hasParens {
			expression.Array = p.parseExpression(LOWEST)
			if !p.expectPeek(lexer.RPAREN) {
				return nil
			}
		} else {
			expression.Array = p.parseExpressionUntilBrace()
		}

		// Parse body - must be a block expression
		if !p.peekTokenIs(lexer.LBRACE) {
			// Give a helpful error - user wrote for (x in arr) expr instead of for x in arr { expr }
			varName := "x"
			if expression.Variable != nil {
				varName = expression.Variable.Value
			}
			arrayStr := "array"
			if expression.Array != nil {
				arrayStr = expression.Array.String()
			}
			p.addStructuredError("PARSE-0003", expression.Token.Line, expression.Token.Column,
				map[string]any{"Var": varName, "Array": arrayStr})
			return nil
		}
		p.nextToken()

		// Create a function literal for the body
		bodyFn := &ast.FunctionLiteral{
			Token: p.curToken,
		}

		// Set parameters based on whether we have one or two variables
		if expression.ValueVariable != nil {
			bodyFn.Params = []*ast.FunctionParameter{{Ident: expression.Variable}, {Ident: expression.ValueVariable}}
		} else {
			bodyFn.Params = []*ast.FunctionParameter{{Ident: expression.Variable}}
		}

		bodyFn.Body = p.parseBlockStatement()
		expression.Body = bodyFn
	} else {
		// Parse: for(array) func
		if hasParens {
			expression.Array = p.parseExpression(LOWEST)
			if !p.expectPeek(lexer.RPAREN) {
				return nil
			}
		} else {
			// Without parens, can't determine where array ends and function begins
			// for arr fn  - is "fn" part of the array expression or the mapping function?
			// Use generic placeholders since we can't reliably extract the source
			p.addStructuredError("PARSE-0004", expression.Token.Line, expression.Token.Column,
				map[string]any{"Array": "array", "Expr": "expr"})
			return nil
		}

		p.nextToken() // move past RPAREN to function

		// Check if user wrote for (expr) { ... } which is ambiguous
		// They probably meant for x in expr { ... } or for (expr) fn(x) { ... }
		if p.curTokenIs(lexer.LBRACE) {
			arrayStr := "array"
			if expression.Array != nil {
				arrayStr = expression.Array.String()
			}
			p.addStructuredError("TYPE-0004", expression.Token.Line, expression.Token.Column,
				map[string]any{"Array": arrayStr, "Got": "{ ... }"})
			return nil
		}

		expression.Function = p.parseExpression(LOWEST)
	}

	return expression
}

func (p *Parser) parseCallExpression(fn ast.Expression) ast.Expression {
	// Only certain expression types can be called as functions.
	// This prevents `if(...){...}(x)` or `"string"(x)` from being parsed as calls.
	// Callable expressions: identifiers, member access, index access, calls (chaining), function literals, connection literals, grouped expressions
	switch fn.(type) {
	case *ast.Identifier,
		*ast.DotExpression,
		*ast.IndexExpression,
		*ast.CallExpression,
		*ast.FunctionLiteral,
		*ast.ConnectionLiteral,
		*ast.GroupedExpression:
		// These are callable - continue with call parsing
		exp := &ast.CallExpression{Token: p.curToken, Function: fn}
		exp.Arguments = p.parseExpressionList(lexer.RPAREN)
		return exp
	default:
		// Not callable - the `(` we consumed starts a grouped expression.
		// Parse what's inside the parens as an expression, then concatenate
		// with the left expression using ++.
		p.nextToken() // move past (
		inner := p.parseExpression(LOWEST)
		if !p.expectPeek(lexer.RPAREN) {
			return nil
		}
		// Create a concatenation: fn ++ inner
		return &ast.InfixExpression{
			Token:    p.curToken,
			Left:     fn,
			Operator: "++",
			Right:    inner,
		}
	}
}

func (p *Parser) parseExpressionList(end lexer.TokenType) []ast.Expression {
	args := []ast.Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return args
	}

	p.nextToken()
	args = append(args, p.parseExpression(COMMA_PREC+1))

	// Check for single-param arrow function: func(x => ...)
	// After parsing 'x', if we see '=' followed by '>', it's an arrow function attempt
	if p.peekTokenIs(lexer.ASSIGN) {
		// Save the identifier position
		identTok := p.curToken
		p.nextToken() // consume =
		if p.peekTokenIs(lexer.GT) {
			// This is x => ... pattern
			p.addErrorWithHints(
				"Arrow function syntax is not supported",
				identTok.Line, identTok.Column,
				"use fn(x) { x * 2 } instead",
				"example: arr.map(fn(x) { x * 2 })")
			return nil
		}
		// Not arrow, restore position conceptually (we've consumed =, so error will cascade)
	}

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume comma
		// Check for trailing comma (next token is closing delimiter)
		if p.peekTokenIs(end) {
			break
		}
		p.nextToken() // move to next argument
		args = append(args, p.parseExpression(COMMA_PREC+1))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return args
}

func (p *Parser) parseIndexOrSliceExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.curToken, Left: left}

	p.nextToken()

	// Check for optional index [?...]
	if p.curTokenIs(lexer.QUESTION) {
		exp.Optional = true
		p.nextToken()
	}

	// Check for slice (colon before any expression, or expression followed by colon)
	if p.curTokenIs(lexer.COLON) {
		// Slice with no start: [:end]
		return p.parseSliceExpression(left, nil)
	}

	// Parse the first expression (could be index or slice start)
	firstExp := p.parseExpression(LOWEST)

	// Check if this is a slice
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // consume colon
		return p.parseSliceExpression(left, firstExp)
	}

	// It's an index expression
	exp.Index = firstExp

	if !p.expectPeek(lexer.RBRACKET) {
		return nil
	}

	return exp
}

func (p *Parser) parseSliceExpression(left ast.Expression, start ast.Expression) ast.Expression {
	exp := &ast.SliceExpression{
		Token: p.curToken,
		Left:  left,
		Start: start,
	}

	// We're at the colon, move to next token
	p.nextToken()

	// Check if there's an end expression
	if !p.curTokenIs(lexer.RBRACKET) {
		// Parse the end expression
		exp.End = p.parseExpression(LOWEST)
		// After parsing, expect the closing bracket
		if !p.expectPeek(lexer.RBRACKET) {
			return nil
		}
	}
	// If we're already at RBRACKET, this handles open-ended slices like arr[1:] or arr[:]

	return exp
}

// Helper functions
func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) peekTokenLiteral() string {
	return p.peekToken.Literal
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

func (p *Parser) peekError(t lexer.TokenType) {
	tokenName := tokenTypeToReadableName(t)
	gotName := tokenTypeToReadableName(p.peekToken.Type)
	gotLiteral := p.peekToken.Literal
	if gotLiteral == "" {
		gotLiteral = gotName
	}

	// Report error at the position after the last successfully parsed token (curToken)
	line := p.curToken.Line
	column := p.curToken.Column + len(p.curToken.Literal)

	p.addError(fmt.Sprintf("expected %s, got '%s'", tokenName, gotLiteral), line, column)
}

func (p *Parser) noPrefixParseFnError(t lexer.TokenType) {
	literal := p.curToken.Literal
	if literal == "" {
		literal = tokenTypeToReadableName(t)
	}

	// If curToken is on a new line compared to prevToken,
	// report the error at the previous token (where the expression should have been)
	line := p.curToken.Line
	column := p.curToken.Column + len(p.curToken.Literal)

	if p.prevToken.Type != lexer.ILLEGAL && p.curToken.Line > p.prevToken.Line {
		// Current token is on a new line, point to after the previous token
		line = p.prevToken.Line
		column = p.prevToken.Column + len(p.prevToken.Literal)
	} else if p.prevToken.Type != lexer.ILLEGAL {
		// Same line, point to after the previous token
		column = p.prevToken.Column + len(p.prevToken.Literal)
	}

	// ILLEGAL tokens already contain a descriptive error message, use it directly
	if t == lexer.ILLEGAL {
		p.addError(literal, line, column)
	} else {
		p.addError(fmt.Sprintf("unexpected '%s'", literal), line, column)
	}
}

// checkKeywordTypo checks if an identifier is a common misspelling of a keyword
// Returns a helpful error message if it's a typo, empty string otherwise
func (p *Parser) checkKeywordTypo(ident string) string {
	lower := strings.ToLower(ident)

	// Common typos of 'export'
	exportTypos := map[string]bool{
		"expoert": true, "exprot": true, "exort": true, "exprt": true,
		"exporrt": true, "expport": true, "exoport": true, "epxort": true,
		"eport": true, "expost": true, "expotr": true,
	}
	if exportTypos[lower] {
		return fmt.Sprintf("unknown keyword '%s'. Did you mean 'export'?\n   💡 Hint: Use 'export' to make values available when importing this module", ident)
	}

	// Common typos of 'let'
	letTypos := map[string]bool{
		"lte": true, "elt": true, "lett": true, "lat": true, "lit": true,
	}
	if letTypos[lower] {
		return fmt.Sprintf("unknown keyword '%s'. Did you mean 'let'?\n   💡 Hint: Use 'let' to declare variables: let x = 5", ident)
	}

	// Common typos of 'function' / 'fn'
	fnTypos := map[string]bool{
		"func": true, "function": true, "fuction": true, "fucntion": true,
		"funciton": true, "funtion": true, "fnn": true,
	}
	if fnTypos[lower] {
		return fmt.Sprintf("unknown keyword '%s'. Did you mean 'fn'?\n   💡 Hint: In Parsley, functions use 'fn': fn(x) { x * 2 }", ident)
	}

	// Common typos of 'return'
	returnTypos := map[string]bool{
		"retrun": true, "reutrn": true, "retrn": true, "retunr": true,
		"rerturn": true, "returm": true, "retutn": true,
	}
	if returnTypos[lower] {
		return fmt.Sprintf("unknown keyword '%s'. Did you mean 'return'?", ident)
	}

	// Common typos of 'for'
	forTypos := map[string]bool{
		"fro": true, "forr": true,
	}
	if forTypos[lower] {
		return fmt.Sprintf("unknown keyword '%s'. Did you mean 'for'?", ident)
	}

	// Common typos of 'import'
	importTypos := map[string]bool{
		"improt": true, "impoer": true, "imoprt": true, "imprt": true,
		"ipmort": true, "imort": true, "impor": true,
	}
	if importTypos[lower] {
		return fmt.Sprintf("unknown keyword '%s'. Did you mean 'import'?\n   💡 Hint: Use import(@./file.pars) or import(@std/table)", ident)
	}

	// Common typos of 'true' / 'false'
	boolTypos := map[string]string{
		"ture": "true", "treu": "true", "trrue": "true",
		"flase": "false", "fasle": "false", "fales": "false",
	}
	if correct, ok := boolTypos[lower]; ok {
		return fmt.Sprintf("unknown identifier '%s'. Did you mean '%s'?", ident, correct)
	}

	// Common typos of 'null'
	nullTypos := map[string]bool{
		"nul": true, "nulll": true, "nill": true, "nil": true,
	}
	if nullTypos[lower] {
		return fmt.Sprintf("unknown identifier '%s'. Did you mean 'null'?\n   💡 Hint: In Parsley, use 'null' (not 'nil' or 'None')", ident)
	}

	return ""
}

// tokenTypeToReadableName converts token types to human-readable names
func tokenTypeToReadableName(t lexer.TokenType) string {
	switch t {
	// Identifiers and literals
	case lexer.IDENT:
		return "identifier"
	case lexer.INT:
		return "integer"
	case lexer.FLOAT:
		return "float"
	case lexer.STRING:
		return "string"
	case lexer.TEMPLATE:
		return "string"
	case lexer.RAW_TEMPLATE:
		return "string"
	case lexer.REGEX:
		return "regex"
	case lexer.DATETIME_LITERAL:
		return "datetime"
	case lexer.DATETIME_NOW:
		return "datetime"
	case lexer.TIME_NOW:
		return "time"
	case lexer.DATE_NOW:
		return "date"
	case lexer.DURATION_LITERAL:
		return "duration"
	case lexer.PATH_LITERAL:
		return "path"
	case lexer.URL_LITERAL:
		return "URL"
	case lexer.PATH_TEMPLATE:
		return "path"
	case lexer.URL_TEMPLATE:
		return "URL"
	case lexer.DATETIME_TEMPLATE:
		return "datetime"
	case lexer.TAG:
		return "tag"
	case lexer.TAG_START:
		return "opening tag"
	case lexer.TAG_END:
		return "closing tag"
	case lexer.TAG_TEXT:
		return "text"

	// Operators
	case lexer.ASSIGN:
		return "'='"
	case lexer.PLUS:
		return "'+'"
	case lexer.MINUS:
		return "'-'"
	case lexer.BANG:
		return "'!'"
	case lexer.ASTERISK:
		return "'*'"
	case lexer.SLASH:
		return "'/'"
	case lexer.PERCENT:
		return "'%'"
	case lexer.LT:
		return "'<'"
	case lexer.GT:
		return "'>'"
	case lexer.LTE:
		return "'<='"
	case lexer.GTE:
		return "'>='"
	case lexer.EQ:
		return "'=='"
	case lexer.NOT_EQ:
		return "'!='"
	case lexer.AND:
		return "'&'"
	case lexer.OR:
		return "'|'"
	case lexer.NULLISH:
		return "'??'"
	case lexer.MATCH:
		return "'~'"
	case lexer.NOT_MATCH:
		return "'!~'"

	// File I/O operators
	case lexer.READ_FROM:
		return "'<=='"
	case lexer.FETCH_FROM:
		return "'<=/='"
	case lexer.WRITE_TO:
		return "'==>'"
	case lexer.APPEND_TO:
		return "'==>>'"

	// Database operators
	case lexer.QUERY_ONE:
		return "'<=?=>'"
	case lexer.QUERY_MANY:
		return "'<=??=>'"
	case lexer.EXECUTE:
		return "'<=!=>'"
	case lexer.EXECUTE_WITH:
		return "'<=#=>'"

	// Delimiters
	case lexer.COMMA:
		return "','"
	case lexer.SEMICOLON:
		return "';'"
	case lexer.COLON:
		return "':'"
	case lexer.DOT:
		return "'.'"
	case lexer.DOTDOTDOT:
		return "'...'"
	case lexer.LPAREN:
		return "'('"
	case lexer.RPAREN:
		return "')'"
	case lexer.LBRACE:
		return "'{'"
	case lexer.RBRACE:
		return "'}'"
	case lexer.LBRACKET:
		return "'['"
	case lexer.RBRACKET:
		return "']'"
	case lexer.PLUSPLUS:
		return "'++'"
	case lexer.RANGE:
		return "'..'"

	// Keywords
	case lexer.FUNCTION:
		return "'fn'"
	case lexer.LET:
		return "'let'"
	case lexer.TRUE:
		return "'true'"
	case lexer.FALSE:
		return "'false'"
	case lexer.IF:
		return "'if'"
	case lexer.ELSE:
		return "'else'"
	case lexer.RETURN:
		return "'return'"
	case lexer.FOR:
		return "'for'"
	case lexer.IN:
		return "'in'"
	case lexer.AS:
		return "'as'"
	case lexer.EXPORT:
		return "'export'"
	case lexer.EOF:
		return "end of file"
	case lexer.ILLEGAL:
		return "illegal character"
	default:
		return string(t.String())
	}
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

// parseDictionaryLiteral parses dictionary literals like { key: value, ... }
// Also supports computed keys: { [expr]: value, ... }
func (p *Parser) parseDictionaryLiteral() ast.Expression {
	dict := &ast.DictionaryLiteral{Token: p.curToken}
	dict.Pairs = make(map[string]ast.Expression)
	dict.KeyOrder = []string{}
	dict.ComputedPairs = []ast.ComputedKeyValue{}

	// Empty dictionary
	if p.peekTokenIs(lexer.RBRACE) {
		p.nextToken()
		return dict
	}

	// Parse key-value pairs
	for !p.curTokenIs(lexer.RBRACE) {
		p.nextToken()

		// Check for trailing comma - we might have just consumed a comma and now see RBRACE
		if p.curTokenIs(lexer.RBRACE) {
			break
		}

		// Check for computed key: [expr]
		if p.curTokenIs(lexer.LBRACKET) {
			// Parse the key expression
			p.nextToken()
			keyExpr := p.parseExpression(LOWEST)
			if keyExpr == nil {
				return nil
			}

			// Expect closing bracket
			if !p.expectPeek(lexer.RBRACKET) {
				return nil
			}

			// Expect colon
			if !p.expectPeek(lexer.COLON) {
				return nil
			}

			// Parse value expression
			p.nextToken()
			value := p.parseExpression(COMMA_PREC + 1)
			if value == nil {
				return nil
			}

			dict.ComputedPairs = append(dict.ComputedPairs, ast.ComputedKeyValue{
				Key:   keyExpr,
				Value: value,
			})
		} else {
			// Key can be an identifier or a string
			var key string
			if p.curTokenIs(lexer.IDENT) {
				key = p.curToken.Literal
			} else if p.curTokenIs(lexer.STRING) {
				key = p.curToken.Literal
			} else {
				p.addError(fmt.Sprintf("Expected identifier or string as dictionary key, got %s",
					tokenTypeToReadableName(p.curToken.Type)), p.curToken.Line, p.curToken.Column)
				return nil
			}

			// Expect colon
			if !p.expectPeek(lexer.COLON) {
				return nil
			}

			// Parse value expression with COMMA_PREC+1 to avoid consuming commas
			p.nextToken()
			value := p.parseExpression(COMMA_PREC + 1)
			if value == nil {
				return nil
			}

			dict.Pairs[key] = value
			dict.KeyOrder = append(dict.KeyOrder, key)
		}

		// Check for comma, semicolon, or closing brace
		if p.peekTokenIs(lexer.RBRACE) {
			p.nextToken()
			break
		}
		if p.peekTokenIs(lexer.COMMA) || p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}
	}

	return dict
}

// parseDotExpression parses dot notation like dict.key
func (p *Parser) parseDotExpression(left ast.Expression) ast.Expression {
	dotExpr := &ast.DotExpression{
		Token: p.curToken,
		Left:  left,
	}

	// Accept IDENT or AS keyword (since 'as' can be a method name like dict.as(Schema))
	if !p.peekTokenIs(lexer.IDENT) && !p.peekTokenIs(lexer.AS) {
		p.peekError(lexer.IDENT)
		return nil
	}
	p.nextToken()

	dotExpr.Key = p.curToken.Literal
	return dotExpr
}

// parseDictDestructuringPattern parses dictionary destructuring patterns like {a, b as c, ...rest}
func (p *Parser) parseDictDestructuringPattern() *ast.DictDestructuringPattern {
	pattern := &ast.DictDestructuringPattern{Token: p.curToken} // the '{' token

	// Check for empty pattern
	if p.peekTokenIs(lexer.RBRACE) {
		p.addError("empty dictionary destructuring pattern", p.peekToken.Line, p.peekToken.Column)
		return nil
	}

	p.nextToken() // move to first identifier or ...

	// Parse keys
	for {
		// Check for rest operator
		if p.curTokenIs(lexer.DOTDOTDOT) {
			if !p.expectPeek(lexer.IDENT) {
				p.addError("expected identifier after '...'", p.peekToken.Line, p.peekToken.Column)
				return nil
			}
			pattern.Rest = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

			// Rest must be at the end
			if !p.peekTokenIs(lexer.RBRACE) {
				p.addError("rest element must be last in destructuring pattern", p.peekToken.Line, p.peekToken.Column)
				return nil
			}
			break
		}

		// Expect identifier for key
		if !p.curTokenIs(lexer.IDENT) {
			return nil
		}

		// Parse regular key
		key := &ast.DictDestructuringKey{
			Token: p.curToken,
			Key:   &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
		}

		// Check for alias (as syntax)
		if p.peekTokenIs(lexer.AS) {
			p.nextToken() // consume 'as'
			if !p.expectPeek(lexer.IDENT) {
				p.addError("expected identifier after 'as'", p.peekToken.Line, p.peekToken.Column)
				return nil
			}
			key.Alias = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		}

		// Check for nested pattern (colon syntax)
		if p.peekTokenIs(lexer.COLON) {
			p.nextToken() // consume ':'
			p.nextToken() // move to pattern start

			// Parse nested pattern
			if p.curTokenIs(lexer.LBRACE) {
				key.Nested = p.parseDictDestructuringPattern()
			} else if p.curTokenIs(lexer.LBRACKET) {
				// For nested array destructuring, we'd need to handle this
				p.addError("nested array destructuring not yet supported", p.curToken.Line, p.curToken.Column)
				return nil
			} else {
				p.addError("expected destructuring pattern after ':'", p.curToken.Line, p.curToken.Column)
				return nil
			}
		}

		pattern.Keys = append(pattern.Keys, key)

		// Check for more keys
		if !p.peekTokenIs(lexer.COMMA) {
			break
		}
		p.nextToken() // consume comma

		// Check for trailing comma before }
		if p.peekTokenIs(lexer.RBRACE) {
			break
		}

		// Move to next key or rest operator
		p.nextToken()
	}

	if !p.expectPeek(lexer.RBRACE) {
		return nil
	}

	return pattern
}

// parseArrayDestructuringPattern parses array destructuring patterns like [a, b, ...rest]
func (p *Parser) parseArrayDestructuringPattern() *ast.ArrayDestructuringPattern {
	pattern := &ast.ArrayDestructuringPattern{Token: p.curToken} // the '[' token

	// Check for empty pattern - but allow [...rest]
	if p.peekTokenIs(lexer.RBRACKET) {
		p.addError("empty array destructuring pattern", p.peekToken.Line, p.peekToken.Column)
		return nil
	}

	p.nextToken() // move to first identifier or ...

	// Parse identifiers
	for {
		// Check for rest operator
		if p.curTokenIs(lexer.DOTDOTDOT) {
			if !p.expectPeek(lexer.IDENT) {
				p.addError("expected identifier after '...'", p.peekToken.Line, p.peekToken.Column)
				return nil
			}
			pattern.Rest = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

			// Rest must be at the end
			if !p.peekTokenIs(lexer.RBRACKET) {
				p.addError("rest element must be last in destructuring pattern", p.peekToken.Line, p.peekToken.Column)
				return nil
			}
			break
		}

		// Expect identifier
		if !p.curTokenIs(lexer.IDENT) {
			p.addError("expected identifier in array destructuring pattern", p.curToken.Line, p.curToken.Column)
			return nil
		}

		pattern.Names = append(pattern.Names, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})

		// Check for more identifiers
		if !p.peekTokenIs(lexer.COMMA) {
			break
		}
		p.nextToken() // consume comma

		// Check for trailing comma before ]
		if p.peekTokenIs(lexer.RBRACKET) {
			break
		}

		// Move to next identifier or rest operator
		p.nextToken()
	}

	if !p.expectPeek(lexer.RBRACKET) {
		return nil
	}

	return pattern
}

// parseSchemaDeclaration parses @schema Name { field: type, ... }
func (p *Parser) parseSchemaDeclaration() ast.Expression {
	schema := &ast.SchemaDeclaration{Token: p.curToken}

	// Expect schema name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	schema.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Expect opening brace
	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	// Parse fields
	schema.Fields = []*ast.SchemaField{}
	for !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		field := p.parseSchemaField()
		if field != nil {
			schema.Fields = append(schema.Fields, field)
		}

		// Check for comma or closing brace
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume comma
		} else if !p.peekTokenIs(lexer.RBRACE) {
			// Allow newline-separated fields without commas
			if p.curTokenIs(lexer.IDENT) || p.curTokenIs(lexer.VIA) {
				continue
			}
		}
	}

	// Expect closing brace
	if !p.expectPeek(lexer.RBRACE) {
		return nil
	}

	return schema
}

// parseSchemaField parses a field definition:
// - name: type
// - name: type(option: value, ...)
// - name: enum["value1", "value2", ...]
// - name: enum["value1", "value2"](option: value, ...)
// - name: type via fk
// - name: [type] via fk
func (p *Parser) parseSchemaField() *ast.SchemaField {
	if !p.curTokenIs(lexer.IDENT) {
		return nil
	}

	field := &ast.SchemaField{
		Token:       p.curToken,
		Name:        &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
		TypeOptions: make(map[string]ast.Expression),
	}

	// Expect colon
	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Check for array type [Type]
	if p.peekTokenIs(lexer.LBRACKET) {
		p.nextToken() // consume [
		field.IsArray = true
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		field.TypeName = p.curToken.Literal
		if !p.expectPeek(lexer.RBRACKET) {
			return nil
		}
	} else {
		// Regular type
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		field.TypeName = p.curToken.Literal
	}

	// Check for nullable marker: type?
	if p.peekTokenIs(lexer.QUESTION) {
		p.nextToken() // consume ?
		field.Nullable = true
	}

	// Check for enum values in square brackets: enum["a", "b", "c"]
	if field.TypeName == "enum" && p.peekTokenIs(lexer.LBRACKET) {
		p.nextToken() // consume [
		field.EnumValues = p.parseEnumValues()
		if !p.expectPeek(lexer.RBRACKET) {
			return nil
		}
	}

	// Check for type options: type(min: 1, max: 100) or enum["a", "b"](serverOnly)
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // consume (
		field.TypeOptions = p.parseTypeOptions()

		if !p.expectPeek(lexer.RPAREN) {
			return nil
		}
	}

	// Check for default value: = expression
	// This must come BEFORE metadata pipe check because the syntax is:
	// type = default | {metadata}
	if p.peekTokenIs(lexer.ASSIGN) {
		p.nextToken() // consume =
		p.nextToken() // move to expression start
		expr := p.parseExpression(LOWEST)

		// Check if the expression is a binary OR with a dictionary on the right.
		// This happens when the user writes: type = default | {metadata}
		// The expression parser sees this as: default | {metadata}
		// We need to split it: default becomes DefaultValue, {metadata} becomes Metadata
		if infixExpr, ok := expr.(*ast.InfixExpression); ok && infixExpr.Operator == "|" {
			if dictLit, ok := infixExpr.Right.(*ast.DictionaryLiteral); ok {
				field.DefaultValue = infixExpr.Left
				field.Metadata = dictLit
			} else {
				field.DefaultValue = expr
			}
		} else {
			field.DefaultValue = expr
		}
	}

	// Check for metadata pipe syntax: | {title: "...", placeholder: "..."}
	// This handles the case with no default value: type | {metadata}
	// We use OR token since single | is lexed as OR
	if field.Metadata == nil && p.peekTokenIs(lexer.OR) && p.peekTokenLiteral() == "|" {
		p.nextToken() // consume |
		if !p.expectPeek(lexer.LBRACE) {
			return nil
		}
		meta := p.parseDictionaryLiteral()
		if dictLit, ok := meta.(*ast.DictionaryLiteral); ok {
			field.Metadata = dictLit
		}
	}

	// Check for "via foreign_key"
	if p.peekTokenIs(lexer.VIA) {
		p.nextToken() // consume via
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		field.ForeignKey = p.curToken.Literal
	}

	return field
}

// parseTableLiteral parses @table [...] or @table(Schema) [...]
func (p *Parser) parseTableLiteral() ast.Expression {
	table := &ast.TableLiteral{Token: p.curToken}

	// Check for optional schema reference: @table(Schema)
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // consume (
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		table.Schema = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		if !p.expectPeek(lexer.RPAREN) {
			return nil
		}
	}

	// Expect opening bracket for row array
	if !p.expectPeek(lexer.LBRACKET) {
		return nil
	}

	// Parse rows (array of dictionaries)
	table.Rows = []*ast.DictionaryLiteral{}
	rowIndex := 0

	for !p.peekTokenIs(lexer.RBRACKET) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		rowIndex++

		// Each element must be a dictionary literal
		if !p.curTokenIs(lexer.LBRACE) {
			p.addError(
				fmt.Sprintf("@table row %d: expected dictionary literal, got %s", rowIndex, p.curToken.Type),
				p.curToken.Line, p.curToken.Column,
			)
			return nil
		}

		dict := p.parseDictionaryLiteral()
		if dict == nil {
			return nil
		}

		dictLit, ok := dict.(*ast.DictionaryLiteral)
		if !ok {
			p.addError(
				fmt.Sprintf("@table row %d: expected dictionary literal", rowIndex),
				p.curToken.Line, p.curToken.Column,
			)
			return nil
		}

		// Extract column names from first row
		if rowIndex == 1 {
			table.Columns = make([]string, 0, len(dictLit.Pairs))
			for key := range dictLit.Pairs {
				table.Columns = append(table.Columns, key)
			}
		} else {
			// Validate subsequent rows have same columns
			rowKeys := make(map[string]bool)
			for key := range dictLit.Pairs {
				rowKeys[key] = true
			}

			// Check for missing columns
			var missing []string
			for _, col := range table.Columns {
				if !rowKeys[col] {
					missing = append(missing, col)
				}
			}
			if len(missing) > 0 {
				p.addError(
					fmt.Sprintf("@table row %d: missing columns: %s", rowIndex, strings.Join(missing, ", ")),
					p.curToken.Line, p.curToken.Column,
				)
				return nil
			}

			// Check for extra columns
			var extra []string
			for key := range rowKeys {
				found := false
				for _, col := range table.Columns {
					if col == key {
						found = true
						break
					}
				}
				if !found {
					extra = append(extra, key)
				}
			}
			if len(extra) > 0 {
				p.addError(
					fmt.Sprintf("@table row %d: extra columns not in first row: %s", rowIndex, strings.Join(extra, ", ")),
					p.curToken.Line, p.curToken.Column,
				)
				return nil
			}
		}

		table.Rows = append(table.Rows, dictLit)

		// Check for comma between rows
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume comma
		}
	}

	// Expect closing bracket
	if !p.expectPeek(lexer.RBRACKET) {
		return nil
	}

	return table
}

// parseEnumValues parses comma-separated string literals for enum: "a", "b", "c"
// Called after [ is consumed, returns before ] is consumed
func (p *Parser) parseEnumValues() []string {
	var values []string

	// Handle empty enum
	if p.peekTokenIs(lexer.RBRACKET) {
		return values
	}

	for {
		if !p.expectPeek(lexer.STRING) {
			p.addError("expected string literal in enum values", p.peekToken.Line, p.peekToken.Column)
			return values
		}
		values = append(values, p.curToken.Literal)

		if !p.peekTokenIs(lexer.COMMA) {
			break
		}
		p.nextToken() // consume comma
	}

	return values
}

// parseTypeOptions parses type options: min: 1, max: 100, unique: true
// Supports full expressions: min: earliestYear, max: @now.year
func (p *Parser) parseTypeOptions() map[string]ast.Expression {
	options := make(map[string]ast.Expression)

	// Handle empty options
	if p.peekTokenIs(lexer.RPAREN) {
		return options
	}

	for {
		// Expect option name
		if !p.expectPeek(lexer.IDENT) {
			return options
		}
		optionName := p.curToken.Literal

		// Check if this is a bare boolean flag (no colon follows)
		// e.g., string(required) or id(auto)
		if p.peekTokenIs(lexer.COMMA) || p.peekTokenIs(lexer.RPAREN) {
			// Bare identifier = boolean true
			options[optionName] = &ast.Boolean{
				Token: lexer.Token{Type: lexer.TRUE, Literal: "true"},
				Value: true,
			}
		} else if p.peekTokenIs(lexer.COLON) {
			// Option with value: name: value
			p.nextToken() // consume colon

			// Parse value as a full expression (supports variables, @objects, arithmetic, etc.)
			p.nextToken()
			value := p.parseExpressionUntil(LOWEST, func() bool {
				// Stop at comma (next option) or rparen (end of options)
				return p.peekTokenIs(lexer.COMMA) || p.peekTokenIs(lexer.RPAREN)
			})

			if value == nil {
				p.addError(fmt.Sprintf("expected expression for type option '%s'", optionName),
					p.curToken.Line, p.curToken.Column)
				return options
			}

			options[optionName] = value
		} else {
			p.addError(fmt.Sprintf("expected ':' or ',' after option '%s'", optionName),
				p.peekToken.Line, p.peekToken.Column)
			return options
		}

		if !p.peekTokenIs(lexer.COMMA) {
			break
		}
		p.nextToken() // consume comma
	}

	return options
}

// parseQueryExpression parses @query(source | conditions + by group ??-> projection)
// Also supports CTEs: @query(Source as name | conditions ??-> cols  MainSource | conditions ??-> *)
func (p *Parser) parseQueryExpression() ast.Expression {
	query := &ast.QueryExpression{Token: p.curToken}
	query.CTEs = []*ast.QueryCTE{}

	// Expect opening paren
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	// Parse query blocks - CTEs followed by main query
	// A block with "Source as alias" followed by more content is a CTE
	// The final block (with or without alias) is the main query
	for {
		// Expect source identifier
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		sourceIdent := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

		// Check for alias "as alias"
		var aliasIdent *ast.Identifier
		if p.peekTokenIs(lexer.AS) {
			p.nextToken() // consume as
			if !p.expectPeek(lexer.IDENT) {
				return nil
			}
			aliasIdent = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		}

		// Parse conditions, modifiers, group by, and computed fields for this block
		conditions := []ast.QueryConditionNode{}
		modifiers := []*ast.QueryModifier{}
		computedFields := []*ast.QueryComputedField{}
		var groupBy []string

		// Main parsing loop for query clauses
		for {
			// Check for GROUP BY: + by
			if p.peekTokenIs(lexer.PLUS) {
				p.nextToken() // consume +
				if p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "by" {
					p.nextToken() // consume "by"
					groupBy = p.parseGroupByFields()
					continue
				} else {
					// Not a GROUP BY, error
					p.addError("expected 'by' after '+' in query", p.peekToken.Line, p.peekToken.Column)
					return nil
				}
			}

			// Check for pipe-based clauses
			if p.peekTokenIs(lexer.OR) {
				p.nextToken() // consume |

				// Check if this is a modifier (order, limit, with)
				if p.peekTokenIs(lexer.IDENT) && p.isQueryModifierKeyword(p.peekToken.Literal) {
					mod := p.parseQueryModifier()
					if mod != nil {
						modifiers = append(modifiers, mod)
					}
				} else if p.peekTokenIs(lexer.LPAREN) || p.peekTokenIs(lexer.BANG) {
					// This is a condition group or NOT-prefixed condition
					// Note: "not" keyword is tokenized as BANG
					node := p.parseQueryConditionExpr()
					if node != nil {
						conditions = append(conditions, node)
					}
				} else if p.peekTokenIs(lexer.IDENT) {
					// Peek ahead to determine if this is a computed field (IDENT COLON or IDENT <-) or condition
					// Save complete state for proper backtracking
					savedCur := p.curToken
					savedPeek := p.peekToken
					savedLexerState := p.l.SaveState()

					// Consume the identifier to check what follows
					p.nextToken() // now curToken is the IDENT
					identToken := p.curToken

					if p.peekTokenIs(lexer.COLON) {
						// This is a computed field: name: function(field)
						cf := p.parseComputedFieldFromIdent(identToken)
						if cf != nil {
							computedFields = append(computedFields, cf)
						}
					} else if p.peekTokenIs(lexer.ARROW_PULL) {
						// This is a correlated subquery computed field: name <-Table | | conditions | ?-> aggregate
						cf := p.parseCorrelatedSubqueryField(identToken, aliasIdent)
						if cf != nil {
							computedFields = append(computedFields, cf)
						}
					} else {
						// This is a condition - restore complete state and parse as condition
						p.curToken = savedCur
						p.peekToken = savedPeek
						p.l.RestoreState(savedLexerState)
						node := p.parseQueryConditionExpr()
						if node != nil {
							conditions = append(conditions, node)
						}
					}
				} else {
					// Parse condition (for other cases)
					node := p.parseQueryConditionExpr()
					if node != nil {
						conditions = append(conditions, node)
					}
				}
				continue
			}

			// No more clauses to parse
			break
		}

		// Parse terminal
		terminal := p.parseQueryTerminal()

		// Check if this is a CTE or the main query
		// A CTE has an alias and is followed by another IDENT (the next query block)
		// After parsing terminal, peek to see if there's another IDENT
		if aliasIdent != nil && p.peekTokenIs(lexer.IDENT) {
			// This is a CTE - save it and continue to parse the next block
			cte := &ast.QueryCTE{
				Token:      sourceIdent.Token,
				Name:       aliasIdent.Value,
				Source:     sourceIdent,
				Conditions: conditions,
				Modifiers:  modifiers,
				Terminal:   terminal,
			}
			query.CTEs = append(query.CTEs, cte)
			// Continue to parse the next block
			continue
		}

		// This is the main query
		query.Source = sourceIdent
		query.SourceAlias = aliasIdent
		query.Conditions = conditions
		query.Modifiers = modifiers
		query.GroupBy = groupBy
		query.ComputedFields = computedFields
		query.Terminal = terminal
		break
	}

	// Expect closing paren
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return query
}

// parseGroupByFields parses field list after "+ by"
func (p *Parser) parseGroupByFields() []string {
	fields := []string{}
	if !p.expectPeek(lexer.IDENT) {
		return fields
	}
	fields = append(fields, p.curToken.Literal)

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume comma
		if !p.expectPeek(lexer.IDENT) {
			return fields
		}
		fields = append(fields, p.curToken.Literal)
	}
	return fields
}

// isComputedFieldStart checks if next tokens look like "name: func(" or "name: ident"
func (p *Parser) isComputedFieldStart() bool {
	// Look ahead: we need IDENT COLON to identify a computed field
	// We're currently peeking at the first IDENT
	if !p.peekTokenIs(lexer.IDENT) {
		return false
	}

	// Save position and look ahead
	// The pattern is: IDENT COLON (aggregate_function | IDENT)
	// We need to check if after the IDENT there's a COLON
	// This is tricky without proper lookahead, so let's check if
	// the identifier is followed by a colon by looking at peek2
	return p.peekNTokenIs(2, lexer.COLON)
}

// parseComputedField parses "name: function(field)" or "name: count"
func (p *Parser) parseComputedField() *ast.QueryComputedField {
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	cf := &ast.QueryComputedField{
		Token: p.curToken,
		Name:  p.curToken.Literal,
	}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	p.nextToken() // move to function name or field

	// Check if it's an aggregate function
	if p.curTokenIs(lexer.IDENT) {
		switch p.curToken.Literal {
		case "count":
			cf.Function = "count"
			// count can be bare or count(field)
			if p.peekTokenIs(lexer.LPAREN) {
				p.nextToken() // consume (
				if p.peekTokenIs(lexer.IDENT) {
					p.nextToken()
					cf.Field = p.curToken.Literal
				}
				if !p.expectPeek(lexer.RPAREN) {
					return nil
				}
			}
		case "sum", "avg", "min", "max":
			cf.Function = p.curToken.Literal
			// These require a field: sum(field)
			if !p.expectPeek(lexer.LPAREN) {
				return nil
			}
			if !p.expectPeek(lexer.IDENT) {
				return nil
			}
			cf.Field = p.curToken.Literal
			if !p.expectPeek(lexer.RPAREN) {
				return nil
			}
		default:
			// Just a field reference
			cf.Field = p.curToken.Literal
		}
	}

	return cf
}

// parseComputedFieldFromIdent parses computed field when identifier is already consumed
func (p *Parser) parseComputedFieldFromIdent(identToken lexer.Token) *ast.QueryComputedField {
	cf := &ast.QueryComputedField{
		Token: identToken,
		Name:  identToken.Literal,
	}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	p.nextToken() // move to function name or field

	// Check if it's an aggregate function
	if p.curTokenIs(lexer.IDENT) {
		switch p.curToken.Literal {
		case "count":
			cf.Function = "count"
			// count can be bare or count(field)
			if p.peekTokenIs(lexer.LPAREN) {
				p.nextToken() // consume (
				if p.peekTokenIs(lexer.IDENT) {
					p.nextToken()
					cf.Field = p.curToken.Literal
				}
				if !p.expectPeek(lexer.RPAREN) {
					return nil
				}
			}
		case "sum", "avg", "min", "max":
			cf.Function = p.curToken.Literal
			// These require a field: sum(field)
			if !p.expectPeek(lexer.LPAREN) {
				return nil
			}
			if !p.expectPeek(lexer.IDENT) {
				return nil
			}
			cf.Field = p.curToken.Literal
			if !p.expectPeek(lexer.RPAREN) {
				return nil
			}
		default:
			// Just a field reference
			cf.Field = p.curToken.Literal
		}
	}

	return cf
}

// parseCorrelatedSubqueryField parses a correlated subquery computed field:
// name <-Table | | conditions | ?-> aggregate  (scalar, correlated subquery)
// name <-Table | | conditions | ??-> *         (join-like, row expansion)
// where conditions can reference the outer query alias
func (p *Parser) parseCorrelatedSubqueryField(identToken lexer.Token, outerAlias *ast.Identifier) *ast.QueryComputedField {
	cf := &ast.QueryComputedField{
		Token: identToken,
		Name:  identToken.Literal,
	}

	// Current token is the identifier name, peek token should be <-
	if !p.expectPeek(lexer.ARROW_PULL) {
		return nil
	}

	// Parse the subquery, which starts at the <- token
	subquery := p.parseQuerySubquery()
	if subquery == nil {
		return nil
	}

	cf.Subquery = subquery

	// Check if this is a join-like subquery (??-> returns multiple rows)
	if subquery.Terminal != nil && subquery.Terminal.Type == "many" {
		cf.IsJoinSubquery = true
		// For join subqueries, the projection specifies which columns from the subquery to include
		// * means all columns from the joined table
	}

	// Extract function from subquery terminal if present (for scalar subqueries)
	if subquery.Terminal != nil && len(subquery.Terminal.Projection) > 0 {
		proj := subquery.Terminal.Projection[0]
		switch proj {
		case "count":
			cf.Function = "count"
		case "sum", "avg", "min", "max":
			// These would be like ?-> sum(field), but our syntax uses ?-> count
			// For now, treat as scalar projection
			cf.Function = proj
		default:
			// Just a field projection
			cf.Field = proj
		}
	}

	return cf
}

// parseQueryCondition parses a condition like "field == value" or "field in {values}"
// Also supports "table.field == value" for correlated subqueries
func (p *Parser) parseQueryCondition() *ast.QueryCondition {
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	cond := &ast.QueryCondition{
		Token: p.curToken,
	}

	// Check if this is a dot expression (table.field for correlated subqueries)
	if p.peekTokenIs(lexer.DOT) {
		// Parse as dot expression: table.field
		left := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		p.nextToken() // consume .
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		cond.Left = &ast.DotExpression{
			Token: p.curToken,
			Left:  left,
			Key:   p.curToken.Literal,
		}
	} else {
		cond.Left = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

	// Parse operator
	p.nextToken()
	switch p.curToken.Type {
	case lexer.EQ:
		cond.Operator = "=="
	case lexer.NOT_EQ:
		cond.Operator = "!="
	case lexer.GT:
		cond.Operator = ">"
	case lexer.LT:
		cond.Operator = "<"
	case lexer.GTE:
		cond.Operator = ">="
	case lexer.LTE:
		cond.Operator = "<="
	case lexer.IN:
		cond.Operator = "in"
	case lexer.BANG:
		// Handle "not in" - BANG is the token for "not" keyword
		if p.peekTokenIs(lexer.IN) {
			p.nextToken() // consume in
			cond.Operator = "not in"
		} else {
			p.addError("expected 'in' after 'not' in query condition", p.curToken.Line, p.curToken.Column)
			return nil
		}
	case lexer.IDENT:
		// Handle "is null", "is not null", "like", "not in", "between"
		switch p.curToken.Literal {
		case "is":
			if p.peekTokenIs(lexer.BANG) || (p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "not") {
				p.nextToken() // consume not
				if p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "null" {
					p.nextToken() // consume null
					cond.Operator = "is not null"
				}
			} else if p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "null" {
				p.nextToken() // consume null
				cond.Operator = "is null"
			}
		case "like":
			cond.Operator = "like"
		case "not":
			if p.peekTokenIs(lexer.IN) {
				p.nextToken() // consume in
				cond.Operator = "not in"
			}
		case "between":
			cond.Operator = "between"
		}
	default:
		p.addError("expected comparison operator in query condition", p.curToken.Line, p.curToken.Column)
		return nil
	}

	// Parse right side (unless is null/is not null)
	if cond.Operator != "is null" && cond.Operator != "is not null" {
		// Check for subquery: <- Table
		if p.peekTokenIs(lexer.ARROW_PULL) {
			p.nextToken() // consume <-
			cond.Right = p.parseQuerySubquery()
		} else {
			p.nextToken()
			// Parse the right side value
			// For correlated subqueries, we may have "outer.field" so we need to handle dot expressions
			cond.Right = p.parseQueryConditionValue()

			// For "between X and Y", parse the second value after "and"
			if cond.Operator == "between" {
				if p.peekTokenIs(lexer.AND) {
					p.nextToken() // consume "and"
					p.nextToken() // move to value
					cond.RightEnd = p.parseQueryConditionValue()
				} else {
					p.addError("expected 'and' after first value in 'between' condition", p.curToken.Line, p.curToken.Column)
					return nil
				}
			}
		}
	}

	return cond
}

// parseQueryConditionValue parses the right-hand side of a query condition
// Rule: Bare identifiers are columns. {expression} are Parsley expressions.
// - {userId}      → QueryInterpolation (evaluate Parsley variable)
// - "active"      → StringLiteral
// - 42            → IntegerLiteral
// - status        → QueryColumnRef (database column)
// - post.id       → DotExpression (column reference with table qualifier)
// - [1, 2, 3]     → ArrayLiteral (for IN clauses)
func (p *Parser) parseQueryConditionValue() ast.Expression {
	// Check for interpolation: {expression}
	if p.curTokenIs(lexer.LBRACE) {
		return p.parseQueryInterpolation()
	}

	// Check for array literal: [1, 2, 3] for IN clauses
	if p.curTokenIs(lexer.LBRACKET) {
		return p.parseSquareBracketArrayLiteral()
	}

	// Check for string literal
	if p.curTokenIs(lexer.STRING) {
		return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
	}

	// Check for integer literal
	if p.curTokenIs(lexer.INT) {
		value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
		if err != nil {
			p.addError(fmt.Sprintf("could not parse %q as integer", p.curToken.Literal), p.curToken.Line, p.curToken.Column)
			return nil
		}
		return &ast.IntegerLiteral{Token: p.curToken, Value: value}
	}

	// Check for float literal
	if p.curTokenIs(lexer.FLOAT) {
		value, err := strconv.ParseFloat(p.curToken.Literal, 64)
		if err != nil {
			p.addError(fmt.Sprintf("could not parse %q as float", p.curToken.Literal), p.curToken.Line, p.curToken.Column)
			return nil
		}
		return &ast.FloatLiteral{Token: p.curToken, Value: value}
	}

	// Check for boolean literals
	if p.curTokenIs(lexer.TRUE) {
		return &ast.Boolean{Token: p.curToken, Value: true}
	}
	if p.curTokenIs(lexer.FALSE) {
		return &ast.Boolean{Token: p.curToken, Value: false}
	}

	// Check for identifier (column reference)
	if p.curTokenIs(lexer.IDENT) {
		ident := p.curToken

		// Check if followed by DOT - this could be "table.column" for correlated subqueries
		if p.peekTokenIs(lexer.DOT) {
			peekedAhead := p.l.PeekToken()
			if peekedAhead.Type == lexer.IDENT {
				// This is a table.column reference - keep as DotExpression
				left := &ast.Identifier{Token: ident, Value: ident.Literal}
				p.nextToken() // consume .
				p.nextToken() // move to column name
				return &ast.DotExpression{
					Token: p.curToken,
					Left:  left,
					Key:   p.curToken.Literal,
				}
			}
		}

		// Bare identifier - this is a column reference
		return &ast.QueryColumnRef{
			Token:  ident,
			Column: ident.Literal,
		}
	}

	p.addError(fmt.Sprintf("unexpected token in query condition value: %s", p.curToken.Literal), p.curToken.Line, p.curToken.Column)
	return nil
}

// parseQueryInterpolation parses an interpolated expression: {expression}
// The current token should be '{' when called
func (p *Parser) parseQueryInterpolation() ast.Expression {
	interp := &ast.QueryInterpolation{Token: p.curToken}

	p.nextToken() // move past '{'

	// Parse the contained expression
	interp.Expression = p.parseExpression(LOWEST)
	if interp.Expression == nil {
		return nil
	}

	// Expect closing '}'
	if !p.expectPeek(lexer.RBRACE) {
		return nil
	}

	return interp
}

// parseQueryConditionExpr parses a complete condition expression after a pipe.
// This can be:
// - A single condition: field == value
// - A grouped condition: (a or b)
// - A combination: (a or b) and c
// - Complex: a and b or c (evaluates left-to-right without explicit grouping)
func (p *Parser) parseQueryConditionExpr() ast.QueryConditionNode {
	// Parse first term
	first := p.parseQueryConditionNode()
	if first == nil {
		return nil
	}

	// Check for and/or at top level
	// If we find them, we need to build a group containing all terms
	// Note: lexer.OR represents BOTH "|" (pipe) and "or" (keyword)
	// We only want to continue if it's the "or" keyword, not the pipe
	isOrKeyword := p.peekTokenIs(lexer.OR) && p.peekToken.Literal == "or"
	if !p.peekTokenIs(lexer.AND) && !isOrKeyword {
		// No further terms - return the single node
		return first
	}

	// We have more terms - create a group to hold them
	group := &ast.QueryConditionGroup{
		Token:      p.curToken,
		Conditions: []ast.QueryConditionNode{first},
		Logic:      "",
		Negated:    false,
	}

	// Parse remaining terms
	for p.peekTokenIs(lexer.AND) || (p.peekTokenIs(lexer.OR) && p.peekToken.Literal == "or") {
		var logic string
		if p.peekTokenIs(lexer.AND) {
			p.nextToken() // consume "and"
			logic = "and"
		} else {
			p.nextToken() // consume "or"
			logic = "or"
		}

		// Parse next term
		next := p.parseQueryConditionNode()
		if next == nil {
			return nil
		}

		// Set logic on the next node
		switch n := next.(type) {
		case *ast.QueryCondition:
			n.Logic = logic
		case *ast.QueryConditionGroup:
			n.Logic = logic
		}

		group.Conditions = append(group.Conditions, next)
	}

	return group
}

// parseQueryConditionNode parses a single condition term which can be:
// - A simple condition: field == value
// - A NOT-prefixed condition: not field == value
// - A parenthesized group: (condition1 or condition2)
// - A NOT-prefixed group: not (condition1 or condition2)
func (p *Parser) parseQueryConditionNode() ast.QueryConditionNode {
	// Check for NOT prefix
	// Note: "not" keyword is tokenized as BANG
	negated := false
	if p.peekTokenIs(lexer.BANG) {
		negated = true
		p.nextToken() // consume "not" or "!"
	}

	// Check for parenthesized group
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // consume "("
		return p.parseQueryConditionGroup(negated)
	}

	// Otherwise, parse a simple condition
	cond := p.parseQueryCondition()
	if cond != nil {
		cond.Negated = negated
	}
	return cond
}

// parseQueryConditionGroup parses a group of conditions wrapped in parentheses
// The opening "(" has already been consumed
func (p *Parser) parseQueryConditionGroup(negated bool) *ast.QueryConditionGroup {
	group := &ast.QueryConditionGroup{
		Token:      p.curToken, // the '(' token
		Conditions: []ast.QueryConditionNode{},
		Negated:    negated,
	}

	// Parse first condition (no logic prefix)
	firstNode := p.parseQueryConditionNode()
	if firstNode == nil {
		return nil
	}
	group.Conditions = append(group.Conditions, firstNode)

	// Parse remaining conditions with logic operators
	for !p.peekTokenIs(lexer.RPAREN) {
		// Require a logic operator (and/or) between conditions
		// Note: "or" is tokenized as lexer.OR, "and" is tokenized as lexer.AND
		var logic string
		if p.peekTokenIs(lexer.AND) {
			p.nextToken() // consume "and"
			logic = "and"
		} else if p.peekTokenIs(lexer.OR) {
			p.nextToken() // consume "or"
			logic = "or"
		} else {
			// No logic operator - might be end of group or syntax error
			break
		}

		// Parse the next condition node
		node := p.parseQueryConditionNode()
		if node == nil {
			return nil
		}

		// Set logic on the node
		switch n := node.(type) {
		case *ast.QueryCondition:
			n.Logic = logic
		case *ast.QueryConditionGroup:
			n.Logic = logic
		}

		group.Conditions = append(group.Conditions, node)
	}

	// Expect closing paren
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return group
}

// parseQueryModifier parses ORDER BY, LIMIT, OFFSET, or WITH clauses
func (p *Parser) parseQueryModifier() *ast.QueryModifier {
	p.nextToken() // move to keyword

	mod := &ast.QueryModifier{Token: p.curToken}

	// Check keyword as identifier literal
	switch p.curToken.Literal {
	case "order":
		mod.Kind = "order"
		mod.OrderFields = []ast.QueryOrderField{}
		// Parse first field
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		orderField := ast.QueryOrderField{Field: p.curToken.Literal}
		// Check for asc/desc after first field
		if p.peekTokenIs(lexer.IDENT) && (p.peekToken.Literal == "asc" || p.peekToken.Literal == "desc") {
			p.nextToken()
			orderField.Direction = p.curToken.Literal
		}
		mod.OrderFields = append(mod.OrderFields, orderField)
		// Parse additional comma-separated fields
		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume comma
			if !p.expectPeek(lexer.IDENT) {
				return nil
			}
			orderField = ast.QueryOrderField{Field: p.curToken.Literal}
			// Check for asc/desc after this field
			if p.peekTokenIs(lexer.IDENT) && (p.peekToken.Literal == "asc" || p.peekToken.Literal == "desc") {
				p.nextToken()
				orderField.Direction = p.curToken.Literal
			}
			mod.OrderFields = append(mod.OrderFields, orderField)
		}
	case "limit":
		mod.Kind = "limit"
		if !p.expectPeek(lexer.INT) {
			return nil
		}
		val, _ := parseInt(p.curToken.Literal)
		mod.Value = val
	case "offset":
		mod.Kind = "offset"
		if !p.expectPeek(lexer.INT) {
			return nil
		}
		val, _ := parseInt(p.curToken.Literal)
		mod.Value = val
	case "with":
		mod.Kind = "with"
		mod.Fields = []string{}
		mod.RelationPaths = []*ast.RelationPath{}
		// Parse relation list (supports dot-separated paths like "comments.author"
		// and conditional syntax like "comments(approved == true | order created_at desc | limit 5)")
		relationPath := p.parseRelationPath()
		if relationPath == nil {
			return nil
		}
		mod.RelationPaths = append(mod.RelationPaths, relationPath)
		// For backward compatibility, also store in Fields
		mod.Fields = append(mod.Fields, relationPath.Path)
		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume comma
			relationPath = p.parseRelationPath()
			if relationPath == nil {
				return nil
			}
			mod.RelationPaths = append(mod.RelationPaths, relationPath)
			mod.Fields = append(mod.Fields, relationPath.Path)
		}
	}

	return mod
}

// parseRelationPath parses a relation path with optional conditions
// Syntax: relation.path or relation.path(conditions | order | limit)
func (p *Parser) parseRelationPath() *ast.RelationPath {
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	path := p.curToken.Literal

	// Parse additional segments separated by dots
	for p.peekTokenIs(lexer.DOT) {
		p.nextToken() // consume dot
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		path += "." + p.curToken.Literal
	}

	relationPath := &ast.RelationPath{
		Path:       path,
		Conditions: []ast.QueryConditionNode{},
		Order:      []ast.QueryOrderField{},
		Limit:      nil,
	}

	// Check for optional conditions: (cond1 | order field | limit n)
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // consume (

		// Parse clauses inside parentheses
		for !p.peekTokenIs(lexer.RPAREN) && !p.peekTokenIs(lexer.EOF) {
			if p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "order" {
				// Parse order clause
				p.nextToken() // consume "order"
				p.parseRelationOrder(relationPath)
			} else if p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "limit" {
				// Parse limit clause
				p.nextToken() // consume "limit"
				if p.expectPeek(lexer.INT) {
					val, _ := parseInt(p.curToken.Literal)
					relationPath.Limit = &val
				}
			} else if p.peekTokenIs(lexer.OR) && p.peekToken.Literal == "|" {
				// Separator between clauses
				p.nextToken() // consume |
			} else if !p.peekTokenIs(lexer.RPAREN) {
				// Parse condition expression
				cond := p.parseRelationCondition()
				if cond != nil {
					relationPath.Conditions = append(relationPath.Conditions, cond)
				}
			}
		}

		if !p.expectPeek(lexer.RPAREN) {
			return nil
		}
	}

	return relationPath
}

// parseRelationOrder parses order fields inside a relation condition
func (p *Parser) parseRelationOrder(rp *ast.RelationPath) {
	// Order followed by field name
	if !p.expectPeek(lexer.IDENT) {
		return
	}
	orderField := ast.QueryOrderField{
		Field: p.curToken.Literal,
	}
	// Optional direction
	if p.peekTokenIs(lexer.IDENT) && (p.peekToken.Literal == "asc" || p.peekToken.Literal == "desc") {
		p.nextToken()
		orderField.Direction = p.curToken.Literal
	}
	rp.Order = append(rp.Order, orderField)
}

// parseRelationCondition parses a single condition inside a relation filter
func (p *Parser) parseRelationCondition() ast.QueryConditionNode {
	// Expect field name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	leftToken := p.curToken
	left := &ast.Identifier{Token: leftToken, Value: leftToken.Literal}

	// Get operator
	p.nextToken()
	op := p.curToken.Literal

	// Get value
	p.nextToken()
	var right ast.Expression
	switch p.curToken.Type {
	case lexer.STRING:
		right = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
	case lexer.INT:
		val, _ := parseInt(p.curToken.Literal)
		right = &ast.IntegerLiteral{Token: p.curToken, Value: val}
	case lexer.FLOAT:
		val, _ := strconv.ParseFloat(p.curToken.Literal, 64)
		right = &ast.FloatLiteral{Token: p.curToken, Value: val}
	case lexer.TRUE:
		right = &ast.Boolean{Token: p.curToken, Value: true}
	case lexer.FALSE:
		right = &ast.Boolean{Token: p.curToken, Value: false}
	case lexer.IDENT:
		right = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	default:
		return nil
	}

	return &ast.QueryCondition{
		Token:    leftToken,
		Left:     left,
		Operator: op,
		Right:    right,
	}
}

// isQueryModifierKeyword checks if a string is a query DSL modifier keyword
func (p *Parser) isQueryModifierKeyword(literal string) bool {
	switch literal {
	case "order", "limit", "offset", "with":
		return true
	default:
		return false
	}
}

// parseQuerySubquery parses a subquery: <-Table | | cond1 | | cond2 | | ?-> field
// The <- token has already been consumed when this is called
func (p *Parser) parseQuerySubquery() *ast.QuerySubquery {
	subquery := &ast.QuerySubquery{
		Token:      p.curToken, // <- token
		Conditions: []ast.QueryConditionNode{},
		Modifiers:  []*ast.QueryModifier{},
	}

	// Parse table name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	subquery.Source = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Check for optional "as alias"
	if p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "as" {
		p.nextToken() // consume "as"
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		subquery.SourceAlias = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

	// Parse subquery conditions and modifiers (prefixed with | |)
	for p.peekTokenIs(lexer.OR) {
		// Save state to check for double pipe
		savedCur := p.curToken
		savedPeek := p.peekToken
		savedLexerState := p.l.SaveState()

		p.nextToken() // consume first |

		// Check if this is a terminal (single pipe before ?-> or ??->)
		if p.peekTokenIs(lexer.RETURN_ONE) || p.peekTokenIs(lexer.RETURN_MANY) {
			// This is the subquery terminal - don't restore, just break to parse it
			break
		}

		// Check for second | to confirm this is a subquery clause
		if !p.peekTokenIs(lexer.OR) {
			// Not a double pipe and not a terminal, restore state and break
			p.curToken = savedCur
			p.peekToken = savedPeek
			p.l.RestoreState(savedLexerState)
			break
		}
		p.nextToken() // consume second |

		// Check for terminal after double pipe (| | ?->)
		if p.peekTokenIs(lexer.RETURN_ONE) || p.peekTokenIs(lexer.RETURN_MANY) {
			// Don't consume - let parseQueryTerminal handle it
			break
		}

		// Check if this is a modifier or condition
		if p.peekTokenIs(lexer.IDENT) && p.isQueryModifierKeyword(p.peekToken.Literal) {
			mod := p.parseQueryModifier()
			if mod != nil {
				subquery.Modifiers = append(subquery.Modifiers, mod)
			}
		} else if p.peekTokenIs(lexer.IDENT) || p.peekTokenIs(lexer.LPAREN) || p.peekTokenIs(lexer.BANG) {
			// Parse condition (can be simple, grouped, or NOT-prefixed)
			node := p.parseQueryConditionExpr()
			if node != nil {
				subquery.Conditions = append(subquery.Conditions, node)
			}
		}
	}

	// Parse terminal (required for subqueries, typically ?-> field)
	subquery.Terminal = p.parseQueryTerminal()

	return subquery
}

// parseQueryTerminal parses ?-> , ??-> , ?!-> , ??!-> , . , or .-> with projection
func (p *Parser) parseQueryTerminal() *ast.QueryTerminal {
	// Check for terminal operator
	if !p.peekTokenIs(lexer.RETURN_ONE) && !p.peekTokenIs(lexer.RETURN_MANY) &&
		!p.peekTokenIs(lexer.RETURN_ONE_EXPLICIT) && !p.peekTokenIs(lexer.RETURN_MANY_EXPLICIT) &&
		!p.peekTokenIs(lexer.DOT) && !p.peekTokenIs(lexer.EXEC_COUNT) {
		return nil
	}

	p.nextToken()
	terminal := &ast.QueryTerminal{Token: p.curToken}

	switch p.curToken.Type {
	case lexer.RETURN_ONE:
		terminal.Type = "one"
		terminal.Explicit = false
	case lexer.RETURN_MANY:
		terminal.Type = "many"
		terminal.Explicit = false
	case lexer.RETURN_ONE_EXPLICIT:
		terminal.Type = "one"
		terminal.Explicit = true
	case lexer.RETURN_MANY_EXPLICIT:
		terminal.Type = "many"
		terminal.Explicit = true
	case lexer.DOT:
		terminal.Type = "execute"
	case lexer.EXEC_COUNT:
		terminal.Type = "count"
	}

	// Parse projection for non-execute terminals
	if terminal.Type != "execute" {
		terminal.Projection = []string{}
		if p.peekTokenIs(lexer.ASTERISK) {
			p.nextToken()
			terminal.Projection = append(terminal.Projection, "*")
		} else if p.peekTokenIs(lexer.IDENT) {
			p.nextToken()
			terminal.Projection = append(terminal.Projection, p.curToken.Literal)
			for p.peekTokenIs(lexer.COMMA) {
				p.nextToken() // consume comma
				if !p.expectPeek(lexer.IDENT) {
					return nil
				}
				terminal.Projection = append(terminal.Projection, p.curToken.Literal)
			}
		}
	}

	return terminal
}

// parseInsertExpression parses @insert(source |< field: value ?-> *)
// or @insert(source * each collection -> alias |< field: value .)
func (p *Parser) parseInsertExpression() ast.Expression {
	insert := &ast.InsertExpression{Token: p.curToken}

	// Expect opening paren
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	// Expect source identifier
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	insert.Source = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Check for batch insert "* each collection as alias"
	if p.peekTokenIs(lexer.ASTERISK) {
		p.nextToken() // consume *
		if p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "each" {
			p.nextToken() // consume each
			p.nextToken() // move to collection expression
			insert.Batch = &ast.InsertBatch{Token: p.curToken}
			// Use INDEX precedence to stop before AS keyword
			insert.Batch.Collection = p.parseExpression(INDEX)

			// Expect "as alias"
			if !p.expectPeek(lexer.AS) {
				return nil
			}
			if !p.expectPeek(lexer.IDENT) {
				return nil
			}
			insert.Batch.Alias = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		} else {
			p.addError("expected 'each' after '*' in batch insert", p.curToken.Line, p.curToken.Column)
			return nil
		}
	}

	// Check for upsert "| update on key"
	if p.peekTokenIs(lexer.OR) {
		p.nextToken() // consume |
		if p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "update" {
			p.nextToken() // consume update
			if p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "on" {
				p.nextToken() // consume on
				insert.UpsertKey = []string{}
				if !p.expectPeek(lexer.IDENT) {
					return nil
				}
				insert.UpsertKey = append(insert.UpsertKey, p.curToken.Literal)
				for p.peekTokenIs(lexer.COMMA) {
					p.nextToken() // consume comma
					if !p.expectPeek(lexer.IDENT) {
						return nil
					}
					insert.UpsertKey = append(insert.UpsertKey, p.curToken.Literal)
				}
			}
		}
	}

	// Parse field writes
	insert.Writes = []*ast.InsertFieldWrite{}
	for p.peekTokenIs(lexer.PIPE_WRITE) {
		p.nextToken() // consume |<
		write := p.parseInsertFieldWrite()
		if write != nil {
			insert.Writes = append(insert.Writes, write)
		}
	}

	// Parse terminal
	insert.Terminal = p.parseQueryTerminal()

	// Expect closing paren
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return insert
}

// parseInsertFieldWrite parses "field: value"
func (p *Parser) parseInsertFieldWrite() *ast.InsertFieldWrite {
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	write := &ast.InsertFieldWrite{
		Token: p.curToken,
		Field: p.curToken.Literal,
	}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	p.nextToken()
	// Use LOWEST precedence so that property access like person.name is fully parsed
	// The terminal parser will look for standalone DOT not followed by IDENT
	write.Value = p.parseExpression(LOWEST)

	return write
}

// parseUpdateExpression parses @update(source | conditions |< field: value .-> count)
func (p *Parser) parseUpdateExpression() ast.Expression {
	update := &ast.UpdateExpression{Token: p.curToken}

	// Expect opening paren
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	// Expect source identifier
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	update.Source = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Parse conditions until we hit |<
	update.Conditions = []*ast.QueryCondition{}
	for p.peekTokenIs(lexer.OR) && !p.peekNTokenIs(2, lexer.PIPE_WRITE) {
		p.nextToken() // consume |
		cond := p.parseQueryCondition()
		if cond != nil {
			update.Conditions = append(update.Conditions, cond)
		}
	}

	// Parse field writes
	update.Writes = []*ast.InsertFieldWrite{}
	for p.peekTokenIs(lexer.PIPE_WRITE) {
		p.nextToken() // consume |<
		write := p.parseInsertFieldWrite()
		if write != nil {
			update.Writes = append(update.Writes, write)
		}
	}

	// Parse terminal
	update.Terminal = p.parseQueryTerminal()

	// Expect closing paren
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return update
}

// parseDeleteExpression parses @delete(source | conditions .)
func (p *Parser) parseDeleteExpression() ast.Expression {
	del := &ast.DeleteExpression{Token: p.curToken}

	// Expect opening paren
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	// Expect source identifier
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	del.Source = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Parse conditions
	del.Conditions = []*ast.QueryCondition{}
	for p.peekTokenIs(lexer.OR) {
		p.nextToken() // consume |
		cond := p.parseQueryCondition()
		if cond != nil {
			del.Conditions = append(del.Conditions, cond)
		}
	}

	// Parse terminal
	del.Terminal = p.parseQueryTerminal()

	// Expect closing paren
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return del
}

// parseTransactionExpression parses @transaction { statements }
func (p *Parser) parseTransactionExpression() ast.Expression {
	trans := &ast.TransactionExpression{Token: p.curToken}

	// Expect opening brace
	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	// Parse statements
	trans.Statements = []ast.Statement{}
	for !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		stmt := p.parseStatement()
		if stmt != nil {
			trans.Statements = append(trans.Statements, stmt)
		}
	}

	// Expect closing brace
	if !p.expectPeek(lexer.RBRACE) {
		return nil
	}

	return trans
}

// peekNTokenIs looks ahead N tokens from curToken
// n=1 is the same as peekTokenIs, n=2 looks at the token after peek, etc.
func (p *Parser) peekNTokenIs(n int, t lexer.TokenType) bool {
	if n <= 0 {
		return p.curTokenIs(t)
	}
	if n == 1 {
		return p.peekTokenIs(t)
	}

	// For n >= 2, we need to save state, advance, check, and restore
	// Save complete state including lexer
	savedCur := p.curToken
	savedPeek := p.peekToken
	savedLexerState := p.l.SaveState()

	// Advance n-1 times to get to position n
	for i := 1; i < n; i++ {
		p.nextToken()
	}

	// Check if peek token matches
	result := p.peekTokenIs(t)

	// Restore complete state
	p.curToken = savedCur
	p.peekToken = savedPeek
	p.l.RestoreState(savedLexerState)

	return result
}

// parseInt parses a string to int64
func parseInt(s string) (int64, error) {
	var result int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int64(c-'0')
		}
	}
	return result, nil
}
