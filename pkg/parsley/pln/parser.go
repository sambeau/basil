package pln

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

// MaxNestingDepth is the maximum allowed nesting depth for PLN values
const MaxNestingDepth = 100

// SchemaResolver is a function that looks up a schema by name
type SchemaResolver func(name string) *evaluator.DSLSchema

// Parser parses PLN input into Parsley objects
type Parser struct {
	l         *Lexer
	curToken  Token
	peekToken Token
	errors    []string
	depth     int            // current nesting depth
	resolver  SchemaResolver // optional schema resolver
	env       *evaluator.Environment
}

// NewParser creates a new PLN parser
func NewParser(input string) *Parser {
	l := NewLexer(input)
	p := &Parser{l: l, errors: []string{}}
	// Read two tokens to initialize curToken and peekToken
	p.nextToken()
	p.nextToken()
	return p
}

// NewParserWithResolver creates a parser with a schema resolver
func NewParserWithResolver(input string, resolver SchemaResolver, env *evaluator.Environment) *Parser {
	p := NewParser(input)
	p.resolver = resolver
	p.env = env
	return p
}

// nextToken advances to the next token
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

// Errors returns any parsing errors
func (p *Parser) Errors() []string {
	return p.errors
}

// addError adds an error message
func (p *Parser) addError(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	p.errors = append(p.errors, fmt.Sprintf("line %d, column %d: %s",
		p.curToken.Line, p.curToken.Column, msg))
}

// Parse parses the input and returns a Parsley object
func (p *Parser) Parse() (evaluator.Object, error) {
	obj := p.parseValue()
	if obj == nil {
		if len(p.errors) > 0 {
			return nil, fmt.Errorf("parse error: %s", strings.Join(p.errors, "; "))
		}
		return nil, fmt.Errorf("parse error: unexpected token %s", p.curToken.Type)
	}

	// Check for trailing content
	if p.curToken.Type != EOF {
		return nil, fmt.Errorf("unexpected token after value: %s", p.curToken.Type)
	}

	if len(p.errors) > 0 {
		return nil, fmt.Errorf("parse errors: %s", strings.Join(p.errors, "; "))
	}

	return obj, nil
}

// parseValue parses any PLN value
func (p *Parser) parseValue() evaluator.Object {
	p.depth++
	defer func() { p.depth-- }()

	if p.depth > MaxNestingDepth {
		p.addError("maximum nesting depth (%d) exceeded", MaxNestingDepth)
		return nil
	}

	switch p.curToken.Type {
	case INT:
		return p.parseInteger()
	case FLOAT:
		return p.parseFloat()
	case STRING:
		return p.parseString()
	case TRUE:
		p.nextToken()
		return &evaluator.Boolean{Value: true}
	case FALSE:
		p.nextToken()
		return &evaluator.Boolean{Value: false}
	case NULL:
		p.nextToken()
		return &evaluator.Null{}
	case LBRACKET:
		return p.parseArray()
	case LBRACE:
		return p.parseDict()
	case AT:
		// @ followed by identifier is a record
		p.nextToken() // consume @
		if p.curToken.Type == IDENT {
			return p.parseRecord()
		}
		p.addError("expected identifier after @, got %s", p.curToken.Type)
		return nil
	case IDENT:
		// Bare identifier without @ is not allowed in PLN
		p.addError("unexpected identifier %q (did you mean to use @ for a record?)", p.curToken.Literal)
		return nil
	case DATETIME:
		return p.parseDateTime()
	case PATH:
		return p.parsePath()
	case URL:
		return p.parseURL()
	case ERRORS:
		// @errors without a preceding record
		p.addError("@errors must follow a record")
		return nil
	default:
		p.addError("unexpected token: %s", p.curToken.Type)
		return nil
	}
}

// parseInteger parses an integer literal
func (p *Parser) parseInteger() evaluator.Object {
	val, err := strconv.ParseInt(p.curToken.Literal, 10, 64)
	if err != nil {
		p.addError("invalid integer: %s", p.curToken.Literal)
		return nil
	}
	p.nextToken()
	return &evaluator.Integer{Value: val}
}

// parseFloat parses a float literal
func (p *Parser) parseFloat() evaluator.Object {
	val, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.addError("invalid float: %s", p.curToken.Literal)
		return nil
	}
	p.nextToken()
	return &evaluator.Float{Value: val}
}

// parseString parses a string literal
func (p *Parser) parseString() evaluator.Object {
	val := p.curToken.Literal
	p.nextToken()
	return &evaluator.String{Value: val}
}

// parseArray parses an array literal
func (p *Parser) parseArray() evaluator.Object {
	arr := &evaluator.Array{Elements: []evaluator.Object{}}

	p.nextToken() // consume [

	// Empty array
	if p.curToken.Type == RBRACKET {
		p.nextToken()
		return arr
	}

	// First element
	elem := p.parseValue()
	if elem == nil {
		return nil
	}
	arr.Elements = append(arr.Elements, elem)

	// Remaining elements
	for p.curToken.Type == COMMA {
		p.nextToken() // consume ,

		// Allow trailing comma
		if p.curToken.Type == RBRACKET {
			break
		}

		elem := p.parseValue()
		if elem == nil {
			return nil
		}
		arr.Elements = append(arr.Elements, elem)
	}

	if p.curToken.Type != RBRACKET {
		p.addError("expected ], got %s", p.curToken.Type)
		return nil
	}
	p.nextToken() // consume ]

	return arr
}

// parseDict parses a dictionary literal
func (p *Parser) parseDict() evaluator.Object {
	dict := &evaluator.Dictionary{
		Pairs:    make(map[string]ast.Expression),
		KeyOrder: []string{},
		Env:      p.env,
	}

	p.nextToken() // consume {

	// Empty dict
	if p.curToken.Type == RBRACE {
		p.nextToken()
		return dict
	}

	// Parse pairs
	for {
		key, val := p.parsePair()
		if key == "" {
			return nil
		}
		dict.Pairs[key] = &ast.ObjectLiteralExpression{Obj: val}
		dict.KeyOrder = append(dict.KeyOrder, key)

		if p.curToken.Type == COMMA {
			p.nextToken() // consume ,

			// Allow trailing comma
			if p.curToken.Type == RBRACE {
				break
			}
			continue
		}

		break
	}

	if p.curToken.Type != RBRACE {
		p.addError("expected }, got %s", p.curToken.Type)
		return nil
	}
	p.nextToken() // consume }

	return dict
}

// parsePair parses a key-value pair
func (p *Parser) parsePair() (string, evaluator.Object) {
	// Key can be IDENT or STRING
	var key string
	switch p.curToken.Type {
	case IDENT:
		key = p.curToken.Literal
		p.nextToken()
	case STRING:
		key = p.curToken.Literal
		p.nextToken()
	default:
		p.addError("expected key (identifier or string), got %s", p.curToken.Type)
		return "", nil
	}

	// Expect :
	if p.curToken.Type != COLON {
		p.addError("expected :, got %s", p.curToken.Type)
		return "", nil
	}
	p.nextToken() // consume :

	// Parse value
	val := p.parseValue()
	if val == nil {
		return "", nil
	}

	return key, val
}

// parseRecord parses a record literal @Schema({...})
func (p *Parser) parseRecord() evaluator.Object {
	schemaName := p.curToken.Literal
	p.nextToken() // consume schema name

	// Expect (
	if p.curToken.Type != LPAREN {
		p.addError("expected ( after schema name, got %s", p.curToken.Type)
		return nil
	}
	p.nextToken() // consume (

	// Expect {
	if p.curToken.Type != LBRACE {
		p.addError("expected { for record data, got %s", p.curToken.Type)
		return nil
	}

	// Parse the dictionary part
	dataObj := p.parseDict()
	if dataObj == nil {
		return nil
	}
	dataDict, ok := dataObj.(*evaluator.Dictionary)
	if !ok {
		p.addError("internal error: expected dictionary")
		return nil
	}

	// Expect )
	if p.curToken.Type != RPAREN {
		p.addError("expected ) after record data, got %s", p.curToken.Type)
		return nil
	}
	p.nextToken() // consume )

	// Check for @errors suffix
	var recordErrors map[string]*evaluator.RecordError
	if p.curToken.Type == ERRORS {
		p.nextToken() // consume @errors

		// Expect {
		if p.curToken.Type != LBRACE {
			p.addError("expected { after @errors, got %s", p.curToken.Type)
			return nil
		}

		errorsObj := p.parseDict()
		if errorsObj == nil {
			return nil
		}
		errorsDict, ok := errorsObj.(*evaluator.Dictionary)
		if !ok {
			p.addError("internal error: expected dictionary for errors")
			return nil
		}

		// Convert errors dict to RecordError map
		recordErrors = make(map[string]*evaluator.RecordError)
		for field, expr := range errorsDict.Pairs {
			if objLit, ok := expr.(*ast.ObjectLiteralExpression); ok {
				if strObj, ok := objLit.Obj.(*evaluator.String); ok {
					recordErrors[field] = &evaluator.RecordError{
						Code:    "PLN",
						Message: strObj.Value,
					}
				}
			}
		}
	}

	// Look up schema if resolver is available
	var schema *evaluator.DSLSchema
	if p.resolver != nil {
		schema = p.resolver(schemaName)
	}

	if schema == nil {
		// Create a minimal schema stub with the schema name
		// This allows Records to survive round-trips even without full schema definition
		// The data fields are inferred from the parsed dictionary
		schema = &evaluator.DSLSchema{
			Name:   schemaName,
			Fields: make(map[string]*evaluator.DSLSchemaField),
		}
		// Infer fields from the data
		for key := range dataDict.Pairs {
			schema.Fields[key] = &evaluator.DSLSchemaField{
				Name: key,
				Type: "any", // Unknown type, but preserves the field
			}
		}
	}

	// Create a Record
	record := &evaluator.Record{
		Schema:    schema,
		Data:      dataDict.Pairs,
		KeyOrder:  dataDict.KeyOrder,
		Errors:    recordErrors,
		Validated: recordErrors != nil,
		Env:       p.env,
	}

	return record
}

// parseDateTime parses a datetime literal
func (p *Parser) parseDateTime() evaluator.Object {
	literal := p.curToken.Literal
	p.nextToken()

	// Try parsing as full datetime
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02",
		"15:04:05",
	}

	var t time.Time
	var err error
	var kind string

	for _, format := range formats {
		t, err = time.Parse(format, literal)
		if err == nil {
			// Determine kind based on format
			switch format {
			case "2006-01-02":
				kind = "date"
			case "15:04:05":
				kind = "time"
			default:
				kind = "datetime"
			}
			break
		}
	}

	if err != nil {
		p.addError("invalid datetime: %s", literal)
		return nil
	}

	// Return as a datetime dictionary (matching Parsley's datetime format)
	return timeToDictWithKind(t, kind, p.env)
}

// timeToDictWithKind converts a Go time.Time to a Parsley datetime dictionary
func timeToDictWithKind(t time.Time, kind string, env *evaluator.Environment) *evaluator.Dictionary {
	pairs := make(map[string]ast.Expression)

	pairs["__type"] = &ast.ObjectLiteralExpression{Obj: &evaluator.String{Value: "datetime"}}
	pairs["kind"] = &ast.ObjectLiteralExpression{Obj: &evaluator.String{Value: kind}}
	pairs["year"] = &ast.ObjectLiteralExpression{Obj: &evaluator.Integer{Value: int64(t.Year())}}
	pairs["month"] = &ast.ObjectLiteralExpression{Obj: &evaluator.Integer{Value: int64(t.Month())}}
	pairs["day"] = &ast.ObjectLiteralExpression{Obj: &evaluator.Integer{Value: int64(t.Day())}}
	pairs["hour"] = &ast.ObjectLiteralExpression{Obj: &evaluator.Integer{Value: int64(t.Hour())}}
	pairs["minute"] = &ast.ObjectLiteralExpression{Obj: &evaluator.Integer{Value: int64(t.Minute())}}
	pairs["second"] = &ast.ObjectLiteralExpression{Obj: &evaluator.Integer{Value: int64(t.Second())}}
	pairs["unix"] = &ast.ObjectLiteralExpression{Obj: &evaluator.Integer{Value: t.Unix()}}
	pairs["weekday"] = &ast.ObjectLiteralExpression{Obj: &evaluator.String{Value: t.Weekday().String()}}
	pairs["iso"] = &ast.ObjectLiteralExpression{Obj: &evaluator.String{Value: t.Format(time.RFC3339)}}

	return &evaluator.Dictionary{
		Pairs:    pairs,
		KeyOrder: []string{"__type", "kind", "year", "month", "day", "hour", "minute", "second", "unix", "weekday", "iso"},
		Env:      env,
	}
}

// parsePath parses a path literal
func (p *Parser) parsePath() evaluator.Object {
	path := p.curToken.Literal
	p.nextToken()

	// Return as a dictionary with __type: "path" for now
	// TODO: Return proper Path object when Parsley supports it
	pairs := make(map[string]ast.Expression)
	pairs["__type"] = &ast.ObjectLiteralExpression{Obj: &evaluator.String{Value: "path"}}
	pairs["value"] = &ast.ObjectLiteralExpression{Obj: &evaluator.String{Value: path}}

	return &evaluator.Dictionary{
		Pairs:    pairs,
		KeyOrder: []string{"__type", "value"},
		Env:      p.env,
	}
}

// parseURL parses a URL literal
func (p *Parser) parseURL() evaluator.Object {
	url := p.curToken.Literal
	p.nextToken()

	// Return as a dictionary with __type: "url" for now
	// TODO: Return proper URL object when Parsley supports it
	pairs := make(map[string]ast.Expression)
	pairs["__type"] = &ast.ObjectLiteralExpression{Obj: &evaluator.String{Value: "url"}}
	pairs["value"] = &ast.ObjectLiteralExpression{Obj: &evaluator.String{Value: url}}

	return &evaluator.Dictionary{
		Pairs:    pairs,
		KeyOrder: []string{"__type", "value"},
		Env:      p.env,
	}
}
