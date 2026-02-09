// eval_parsing.go - Parsing functions for various data formats
//
// This file contains functions for parsing different data formats into Parsley objects:
// - JSON parsing
// - YAML parsing
// - Markdown parsing (with frontmatter and Parsley interpolation)
// - CSV parsing
// - Duration string parsing
//
// Also includes helper functions for:
// - Converting JSON/YAML data structures to Parsley objects
// - Goldmark extension for @{expr} interpolation in Markdown

package evaluator

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	htmllib "html"
	"strconv"
	"strings"
	"time"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
	"github.com/yuin/goldmark"
	goldmarkAst "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	goldmarkParser "github.com/yuin/goldmark/parser"
	goldmarkRenderer "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"gopkg.in/yaml.v3"
)

// parseDurationString parses a duration string like "1y2mo3w4d5h6m7s" into months and seconds
// Supports negative durations with leading minus sign
// Valid units: y (years), mo (months), w (weeks), d (days), h (hours), m (minutes), s (seconds)
func parseDurationString(s string) (int64, int64, error) {
	var months int64
	var seconds int64
	negative := false

	i := 0

	// Check for leading minus sign (negative duration)
	if i < len(s) && s[i] == '-' {
		negative = true
		i++
	}

	for i < len(s) {
		// Read number
		if !isDigit(rune(s[i])) {
			return 0, 0, fmt.Errorf("expected digit at position %d", i)
		}

		numStart := i
		for i < len(s) && isDigit(rune(s[i])) {
			i++
		}

		num, err := strconv.ParseInt(s[numStart:i], 10, 64)
		if err != nil {
			return 0, 0, err
		}

		// Read unit
		if i >= len(s) {
			return 0, 0, fmt.Errorf("missing unit after number at position %d", i)
		}

		var unit string
		// Check for "mo" (months)
		if i+1 < len(s) && s[i:i+2] == "mo" {
			unit = "mo"
			i += 2
		} else {
			// Single letter unit
			unit = string(s[i])
			i++
		}

		// Convert to months or seconds
		switch unit {
		case "y": // years = 12 months
			months += num * 12
		case "mo": // months
			months += num
		case "w": // weeks = 7 days = 7 * 24 * 60 * 60 seconds
			seconds += num * 7 * 24 * 60 * 60
		case "d": // days = 24 * 60 * 60 seconds
			seconds += num * 24 * 60 * 60
		case "h": // hours = 60 * 60 seconds
			seconds += num * 60 * 60
		case "m": // minutes = 60 seconds
			seconds += num * 60
		case "s": // seconds
			seconds += num
		default:
			return 0, 0, fmt.Errorf("unknown unit: %s", unit)
		}
	}

	// Apply negative sign if present
	if negative {
		months = -months
		seconds = -seconds
	}

	return months, seconds, nil
}

// isDigit checks if a rune is a digit
func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

// parseJSON parses a JSON string into Parsley objects
func parseJSON(content string) (Object, *Error) {
	var data any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil, newFormatError("FMT-0005", err)
	}
	return jsonToObject(data), nil
}

// parseYAML parses a YAML string into Parsley objects
func parseYAML(content string) (Object, *Error) {
	var data any
	if err := yaml.Unmarshal([]byte(content), &data); err != nil {
		return nil, newFormatError("FMT-0006", err)
	}
	return yamlToObject(data), nil
}

// jsonToObject converts a Go interface{} (from JSON) to a Parsley Object
func jsonToObject(data any) Object {
	switch v := data.(type) {
	case nil:
		return NULL
	case bool:
		return nativeBoolToParsBoolean(v)
	case float64:
		// JSON numbers are always float64
		if v == float64(int64(v)) {
			return &Integer{Value: int64(v)}
		}
		return &Float{Value: v}
	case string:
		return &String{Value: v}
	case []any:
		elements := make([]Object, len(v))
		for i, elem := range v {
			elements[i] = jsonToObject(elem)
		}
		return &Array{Elements: elements}
	case map[string]any:
		pairs := make(map[string]ast.Expression)
		for key, val := range v {
			obj := jsonToObject(val)
			pairs[key] = &ast.ObjectLiteralExpression{Obj: obj}
		}
		return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
	default:
		return NULL
	}
}

// yamlToObject converts a YAML value to a Parsley Object
func yamlToObject(value any) Object {
	switch v := value.(type) {
	case nil:
		return NULL
	case bool:
		return nativeBoolToParsBoolean(v)
	case int:
		return &Integer{Value: int64(v)}
	case int64:
		return &Integer{Value: v}
	case float64:
		if v == float64(int64(v)) {
			return &Integer{Value: int64(v)}
		}
		return &Float{Value: v}
	case time.Time:
		// YAML timestamps are parsed directly by yaml.v3
		return timeToDatetimeDict(v, NewEnvironment())
	case string:
		// Try to parse as date if it looks like ISO format
		if len(v) >= 10 && v[4] == '-' && v[7] == '-' {
			if t, err := time.Parse("2006-01-02", v[:10]); err == nil {
				return timeToDatetimeDict(t, NewEnvironment())
			}
		}
		return &String{Value: v}
	case []any:
		elements := make([]Object, len(v))
		for i, elem := range v {
			elements[i] = yamlToObject(elem)
		}
		return &Array{Elements: elements}
	case map[string]any:
		pairs := make(map[string]ast.Expression)
		for key, val := range v {
			obj := yamlToObject(val)
			pairs[key] = &ast.ObjectLiteralExpression{Obj: obj}
		}
		return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
	default:
		// Handle other YAML types (like timestamps)
		return &String{Value: fmt.Sprintf("%v", v)}
	}
}

// ========== Goldmark Parsley Interpolation Extension ==========

// KindParsleyInterpolation is the NodeKind for Parsley interpolation nodes
var KindParsleyInterpolation = goldmarkAst.NewNodeKind("ParsleyInterpolation")

// ParsleyInterpolationNode represents a @{expr} interpolation in the Goldmark AST
type ParsleyInterpolationNode struct {
	goldmarkAst.BaseInline
	Expression string
}

// Dump implements goldmarkAst.Node.Dump for debugging
func (n *ParsleyInterpolationNode) Dump(source []byte, level int) {
	goldmarkAst.DumpHelper(n, source, level, map[string]string{
		"Expression": n.Expression,
	}, nil)
}

// Kind implements goldmarkAst.Node.Kind
func (n *ParsleyInterpolationNode) Kind() goldmarkAst.NodeKind {
	return KindParsleyInterpolation
}

// parsleyInterpolationParser parses @{expr} syntax into AST nodes
type parsleyInterpolationParser struct{}

// Trigger returns the character that triggers this parser
func (p *parsleyInterpolationParser) Trigger() []byte {
	return []byte{'@'}
}

// Parse parses a @{expr} interpolation and returns an AST node
func (p *parsleyInterpolationParser) Parse(parent goldmarkAst.Node, block text.Reader, pc goldmarkParser.Context) goldmarkAst.Node {
	line, segment := block.PeekLine()

	// Check for @{
	if len(line) < 2 || line[1] != '{' {
		return nil
	}

	// Check for backslash escape: look at the character before '@' in the source
	// Note: segment.Start is the position of '@' in the source
	if segment.Start > 0 {
		source := block.Source()
		// Look at the byte before the '@'
		prevPos := segment.Start - 1
		if prevPos >= 0 && prevPos < len(source) && source[prevPos] == '\\' {
			// Check if this backslash itself is escaped
			if prevPos > 0 && source[prevPos-1] == '\\' {
				// Double backslash: \\@{ -> \@{ (backslash is literal, @ triggers us)
				// This is NOT escaped, proceed with parsing
			} else {
				// Single backslash: \@{ -> @{ (@ is escaped)
				// Don't parse as interpolation, advance past @{ and return nil
				block.Advance(2)
				return nil
			}
		}
	}

	// Find matching closing brace
	pos := findMatchingBraceInBytes(line, 2)
	if pos == -1 {
		return nil // No matching brace
	}

	// Extract expression (between @{ and })
	expr := string(line[2:pos])

	// Advance reader past the entire @{expr}
	block.Advance(pos + 1)

	return &ParsleyInterpolationNode{
		Expression: expr,
	}
}

// findMatchingBraceInBytes finds the closing } for a { at startPos
// Handles nested braces, strings, raw strings, and escape sequences
func findMatchingBraceInBytes(input []byte, startPos int) int {
	depth := 1
	inString := false
	inRawString := false
	inChar := false
	escapeNext := false
	i := startPos

	for i < len(input) {
		ch := input[i]

		// Handle escape sequences
		if escapeNext {
			escapeNext = false
			i++
			continue
		}

		if ch == '\\' && !inRawString {
			escapeNext = true
			i++
			continue
		}

		// Check for raw string start (backtick)
		if ch == '`' && !inString && !inChar {
			inRawString = !inRawString
			i++
			continue
		}

		// Handle regular strings and chars (only when not in raw string)
		if !inRawString {
			if ch == '"' && !inChar {
				inString = !inString
				i++
				continue
			}

			if ch == '\'' && !inString {
				inChar = !inChar
				i++
				continue
			}
		}

		// Only count braces outside of strings
		if !inString && !inChar && !inRawString {
			if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
				if depth == 0 {
					return i
				}
			}
		}

		i++
	}

	return -1 // No matching brace found
}

// parsleyInterpolationRenderer renders ParsleyInterpolationNode to HTML
type parsleyInterpolationRenderer struct {
	env *Environment
}

// RegisterFuncs registers the render function for ParsleyInterpolationNode
func (r *parsleyInterpolationRenderer) RegisterFuncs(reg goldmarkRenderer.NodeRendererFuncRegisterer) {
	reg.Register(KindParsleyInterpolation, r.renderParsleyInterpolation)
}

// renderParsleyInterpolation evaluates and renders a Parsley interpolation
func (r *parsleyInterpolationRenderer) renderParsleyInterpolation(
	w util.BufWriter,
	source []byte,
	node goldmarkAst.Node,
	entering bool,
) (goldmarkAst.WalkStatus, error) {
	if !entering {
		return goldmarkAst.WalkContinue, nil
	}

	n := node.(*ParsleyInterpolationNode)

	// Parse the Parsley expression
	l := lexer.New(n.Expression)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		// Parse error - output error span
		w.WriteString(`<span class="parsley-error" title="Parse error">`)
		w.WriteString(htmllib.EscapeString(strings.Join(p.Errors(), "; ")))
		w.WriteString("</span>")
		return goldmarkAst.WalkContinue, nil
	}

	// Evaluate the expression
	var result Object
	for _, stmt := range program.Statements {
		result = Eval(stmt, r.env)
		if isError(result) {
			// Evaluation error - output error span
			w.WriteString(`<span class="parsley-error" title="Evaluation error">`)
			w.WriteString(htmllib.EscapeString(result.Inspect()))
			w.WriteString("</span>")
			return goldmarkAst.WalkContinue, nil
		}
	}

	// Convert result to string and output
	if result != nil {
		output := objectToTemplateString(result)
		w.WriteString(output)
	}

	return goldmarkAst.WalkContinue, nil
}

// ParsleyInterpolationExtension is a Goldmark extension that evaluates @{expr} syntax
type ParsleyInterpolationExtension struct {
	env *Environment
}

// NewParsleyInterpolation creates a new Parsley interpolation extension
func NewParsleyInterpolation(env *Environment) goldmark.Extender {
	return &ParsleyInterpolationExtension{env: env}
}

// Extend adds the Parsley interpolation parser and renderer to Goldmark
func (e *ParsleyInterpolationExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(goldmarkParser.WithInlineParsers(
		util.Prioritized(&parsleyInterpolationParser{}, 500),
	))
	m.Renderer().AddOptions(goldmarkRenderer.WithNodeRenderers(
		util.Prioritized(&parsleyInterpolationRenderer{env: e.env}, 500),
	))
}

// ========== End Goldmark Extension ==========

// parseMarkdown parses markdown content with optional YAML frontmatter
// Returns a dictionary with: html (rendered HTML), raw (original markdown), md (frontmatter metadata)
func parseMarkdown(content string, options *Dictionary, env *Environment) (Object, *Error) {
	pairs := make(map[string]ast.Expression)

	// Extract options
	includeIDs := false
	if options != nil {
		if idsExpr, ok := options.Pairs["ids"]; ok {
			idsVal := Eval(idsExpr, options.Env)
			if b, ok := idsVal.(*Boolean); ok {
				includeIDs = b.Value
			}
		}
	}

	// Check for YAML frontmatter (starts with ---)
	body := content
	metadataPairs := make(map[string]ast.Expression)
	if strings.HasPrefix(strings.TrimSpace(content), "---") {
		// Find the closing ---
		trimmed := strings.TrimSpace(content)
		rest := trimmed[3:] // Skip opening ---

		before, after, ok := strings.Cut(rest, "\n---")
		if ok {
			// Extract frontmatter YAML
			frontmatterYAML := before
			body = strings.TrimSpace(after) // Skip closing ---\n

			// Parse YAML frontmatter
			var frontmatter map[string]any
			if err := yaml.Unmarshal([]byte(frontmatterYAML), &frontmatter); err != nil {
				return nil, newFormatError("FMT-0006", err)
			}

			// Add frontmatter fields to metadata dict
			for key, value := range frontmatter {
				obj := yamlToObject(value)
				metadataPairs[key] = &ast.ObjectLiteralExpression{Obj: obj}
				// Also add to environment so interpolations can access frontmatter vars
				env.Set(key, obj)
			}
		}
	}

	// Convert markdown to HTML using goldmark with Parsley interpolation extension
	var htmlBuf bytes.Buffer

	// Configure parser options
	parserOptions := []goldmarkParser.Option{}
	if includeIDs {
		parserOptions = append(parserOptions, goldmarkParser.WithAutoHeadingID())
	}

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			NewParsleyInterpolation(env), // Custom extension for @{expr} syntax
		),
		goldmark.WithParserOptions(parserOptions...),
		goldmark.WithRendererOptions(html.WithUnsafe()), // Allow raw HTML from interpolations
	)
	if err := md.Convert([]byte(body), &htmlBuf); err != nil {
		return nil, newFormatError("FMT-0010", err)
	}

	finalHTML := htmlBuf.String()

	// Add html and raw fields
	pairs["html"] = &ast.ObjectLiteralExpression{Obj: &String{Value: finalHTML}}
	pairs["raw"] = &ast.ObjectLiteralExpression{Obj: &String{Value: body}}

	// Add md field containing metadata (frontmatter)
	pairs["md"] = &ast.DictionaryLiteral{
		Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
		Pairs: metadataPairs,
	}

	return &Dictionary{Pairs: pairs, Env: env}, nil
}

// parseCSV parses CSV data into a Table (if header) or array of arrays
func parseCSV(data []byte, hasHeader bool) (Object, *Error) {
	reader := csv.NewReader(strings.NewReader(string(data)))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, newFormatError("FMT-0007", err)
	}

	if len(records) == 0 {
		if hasHeader {
			// Empty CSV with headers returns empty Table
			return &Table{Rows: []*Dictionary{}, Columns: []string{}}, nil
		}
		return &Array{Elements: []Object{}}, nil
	}

	if hasHeader {
		// First row is headers (column names)
		headers := records[0]
		rows := make([]*Dictionary, 0, len(records)-1)

		for _, record := range records[1:] {
			pairs := make(map[string]ast.Expression)
			for i, value := range record {
				if i < len(headers) {
					pairs[headers[i]] = &ast.ObjectLiteralExpression{Obj: parseCSVValue(value)}
				}
			}
			rows = append(rows, &Dictionary{Pairs: pairs, Env: NewEnvironment()})
		}
		// Return Table instead of Array
		return &Table{Rows: rows, Columns: headers}, nil
	}

	// No header - return array of arrays
	rows := make([]Object, len(records))
	for i, record := range records {
		elements := make([]Object, len(record))
		for j, value := range record {
			elements[j] = parseCSVValue(value)
		}
		rows[i] = &Array{Elements: elements}
	}
	return &Array{Elements: rows}, nil
}

// parseCSVValue converts a CSV string value to the appropriate type
// Tries integer, float, boolean, then falls back to string
func parseCSVValue(value string) Object {
	// Try integer first (stricter than float)
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return &Integer{Value: i}
	}
	// Try float
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return &Float{Value: f}
	}
	// Try boolean
	lower := strings.ToLower(value)
	if lower == "true" {
		return TRUE
	}
	if lower == "false" {
		return FALSE
	}
	// Keep as string
	return &String{Value: value}
}
