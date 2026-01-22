// Package format provides AST-based formatting for Parsley source code.
// This file handles formatting of AST nodes for the pretty-printer.
package format

import (
	"fmt"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// getStatementToken extracts the first token from a statement for comment/blank line info
func getStatementToken(stmt ast.Statement) *lexer.Token {
	switch s := stmt.(type) {
	case *ast.LetStatement:
		return &s.Token
	case *ast.AssignmentStatement:
		return &s.Token
	case *ast.ReturnStatement:
		return &s.Token
	case *ast.ExpressionStatement:
		return getExpressionToken(s.Expression)
	case *ast.ExportNameStatement:
		return &s.Token
	case *ast.ComputedExportStatement:
		return &s.Token
	case *ast.BlockStatement:
		return &s.Token
	case *ast.IndexAssignmentStatement:
		return getExpressionToken(s.Target)
	case *ast.StopStatement:
		return &s.Token
	case *ast.SkipStatement:
		return &s.Token
	}
	return nil
}

// getNodeToken extracts the first token from any AST node (for comment/blank line info)
func getNodeToken(node ast.Node) *lexer.Token {
	switch n := node.(type) {
	case ast.Statement:
		return getStatementToken(n)
	case ast.Expression:
		return getExpressionToken(n)
	case *ast.TextNode:
		return &n.Token
	}
	return nil
}

// getExpressionToken extracts the first token from an expression
func getExpressionToken(expr ast.Expression) *lexer.Token {
	switch e := expr.(type) {
	case *ast.Identifier:
		return &e.Token
	case *ast.IntegerLiteral:
		return &e.Token
	case *ast.FloatLiteral:
		return &e.Token
	case *ast.StringLiteral:
		return &e.Token
	case *ast.Boolean:
		return &e.Token
	case *ast.FunctionLiteral:
		return &e.Token
	case *ast.IfExpression:
		return &e.Token
	case *ast.ForExpression:
		return &e.Token
	case *ast.TagLiteral:
		return &e.Token
	case *ast.TagPairExpression:
		return &e.Token
	case *ast.CallExpression:
		return getExpressionToken(e.Function)
	case *ast.InfixExpression:
		return getExpressionToken(e.Left)
	case *ast.PrefixExpression:
		return &e.Token
	case *ast.ArrayLiteral:
		return &e.Token
	case *ast.DictionaryLiteral:
		return &e.Token
	case *ast.SchemaDeclaration:
		return &e.Token
	}
	return nil
}

// writeComments outputs any leading comments for a token
func (p *Printer) writeComments(tok *lexer.Token) {
	if tok == nil {
		return
	}
	for _, comment := range tok.LeadingComments {
		p.writeIndent()
		p.write(comment)
		p.newline()
	}
}

// writeTrailingComment outputs a trailing comment (from previous statement) inline
func (p *Printer) writeTrailingComment(tok *lexer.Token) {
	if tok == nil || tok.TrailingComment == "" {
		return
	}
	// Trailing comment goes on same line as previous statement, with a space
	p.write(" ")
	p.write(tok.TrailingComment)
}

// writeBlankLineIfNeeded outputs a blank line if the token had one before it
func (p *Printer) writeBlankLineIfNeeded(tok *lexer.Token) {
	if tok != nil && tok.BlankLinesBefore > 0 {
		p.newline()
	}
}

// FormatNode formats any AST node into well-formatted Parsley code.
// This is the main entry point for AST-based formatting.
func FormatNode(node ast.Node) string {
	if node == nil {
		return ""
	}
	p := NewPrinter()
	p.formatNode(node)
	return p.String()
}

// FormatProgram formats an entire Parsley program.
func FormatProgram(prog *ast.Program) string {
	if prog == nil || len(prog.Statements) == 0 {
		return ""
	}
	p := NewPrinter()
	p.formatProgram(prog)
	return p.String()
}

// formatProgram formats a program with proper spacing between statements
func (p *Printer) formatProgram(prog *ast.Program) {
	for i, stmt := range prog.Statements {
		tok := getStatementToken(stmt)

		// For the first statement, handle leading comments and blank lines
		if i == 0 {
			// Output any leading comments for this statement
			p.writeComments(tok)

			// If there was a blank line between comments and code, preserve it
			if tok != nil && tok.BlankLinesBefore > 0 && len(tok.LeadingComments) > 0 {
				p.newline()
			}

			p.formatStatement(stmt)
			continue
		}

		// For subsequent statements:
		// 1. First, output trailing comment from previous statement (on same line)
		if tok != nil && tok.TrailingComment != "" {
			p.write(" ")
			p.write(tok.TrailingComment)
		}

		// 2. Newline after previous statement
		p.newline()

		// 3. Blank line separator if needed
		if tok != nil && tok.BlankLinesBefore > 0 {
			// Source had a blank line - output it
			p.newline()
		} else if needsBlankLineBefore(stmt, prog.Statements[i-1]) {
			// Fallback: Add blank line BEFORE certain statements (like exported function defs)
			p.newline()
		}

		// 4. Output any leading comments for this statement
		p.writeComments(tok)

		// 5. Format the statement
		p.formatStatement(stmt)

		// Add extra blank line for visual separation after functions/schemas (but not at end)
		// Only if next statement doesn't already have a blank line from source
		if i < len(prog.Statements)-1 && needsBlankLineAfter(stmt) {
			nextTok := getStatementToken(prog.Statements[i+1])
			if nextTok == nil || nextTok.BlankLinesBefore == 0 {
				p.newline()
			}
		}
	}
}

// needsBlankLineBefore determines if a statement needs an extra blank line before it
func needsBlankLineBefore(stmt ast.Statement, prevStmt ast.Statement) bool {
	// Add blank line before exported function definitions if previous wasn't one
	var isFunc, isExport bool

	switch s := stmt.(type) {
	case *ast.LetStatement:
		_, isFunc = s.Value.(*ast.FunctionLiteral)
		isExport = s.Export
	case *ast.AssignmentStatement:
		_, isFunc = s.Value.(*ast.FunctionLiteral)
		isExport = s.Export
	}

	if isFunc && isExport {
		// Check if previous was NOT a function definition
		switch ps := prevStmt.(type) {
		case *ast.LetStatement:
			if _, isPrevFunc := ps.Value.(*ast.FunctionLiteral); !isPrevFunc {
				return true
			}
		case *ast.AssignmentStatement:
			if _, isPrevFunc := ps.Value.(*ast.FunctionLiteral); !isPrevFunc {
				return true
			}
		default:
			return true
		}
	}
	return false
}

// needsBlankLineAfter determines if a statement needs an extra blank line after it
func needsBlankLineAfter(stmt ast.Statement) bool {
	switch s := stmt.(type) {
	case *ast.LetStatement:
		// Function definitions get extra spacing
		if _, ok := s.Value.(*ast.FunctionLiteral); ok {
			return true
		}
	case *ast.AssignmentStatement:
		// Exported function definitions get extra spacing
		if _, ok := s.Value.(*ast.FunctionLiteral); ok {
			return true
		}
	case *ast.ExpressionStatement:
		// Schema declarations get extra spacing
		if _, ok := s.Expression.(*ast.SchemaDeclaration); ok {
			return true
		}
	}
	return false
}

// formatNode dispatches to the appropriate formatting method
func (p *Printer) formatNode(node ast.Node) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	// Statements
	case *ast.Program:
		p.formatProgram(n)
	case *ast.LetStatement:
		p.formatLetStatement(n)
	case *ast.AssignmentStatement:
		p.formatAssignmentStatement(n)
	case *ast.ReturnStatement:
		p.formatReturnStatement(n)
	case *ast.ExpressionStatement:
		p.formatExpressionStatement(n)
	case *ast.BlockStatement:
		p.formatBlockStatement(n)
	case *ast.ExportNameStatement:
		p.formatExportNameStatement(n)
	case *ast.ComputedExportStatement:
		p.formatComputedExportStatement(n)
	case *ast.IndexAssignmentStatement:
		p.formatIndexAssignmentStatement(n)
	case *ast.CheckStatement:
		p.formatCheckStatement(n)

	// Expressions
	case *ast.Identifier:
		p.write(n.Value)
	case *ast.IntegerLiteral:
		p.write(fmt.Sprintf("%d", n.Value))
	case *ast.FloatLiteral:
		p.write(fmt.Sprintf("%v", n.Value))
	case *ast.StringLiteral:
		p.formatStringLiteral(n)
	case *ast.Boolean:
		p.write(n.Token.Literal)
	case *ast.ArrayLiteral:
		p.formatArrayLiteral(n)
	case *ast.DictionaryLiteral:
		p.formatDictionaryLiteral(n)
	case *ast.FunctionLiteral:
		p.formatFunctionLiteral(n)
	case *ast.CallExpression:
		p.formatCallExpression(n)
	case *ast.DotExpression:
		p.formatDotExpression(n)
	case *ast.IndexExpression:
		p.formatIndexExpression(n)
	case *ast.SliceExpression:
		p.formatSliceExpression(n)
	case *ast.PrefixExpression:
		p.formatPrefixExpression(n)
	case *ast.InfixExpression:
		p.formatInfixExpression(n)
	case *ast.IfExpression:
		p.formatIfExpression(n)
	case *ast.ForExpression:
		p.formatForExpression(n)
	case *ast.TagLiteral:
		p.formatTagLiteral(n)
	case *ast.TagPairExpression:
		p.formatTagPairExpression(n)
	case *ast.TextNode:
		p.write(n.Value)
	case *ast.GroupedExpression:
		p.formatGroupedExpression(n)
	case *ast.IsExpression:
		p.formatIsExpression(n)
	case *ast.InterpolationBlock:
		p.formatInterpolationBlock(n)
	case *ast.SchemaDeclaration:
		p.formatSchemaDeclaration(n)
	case *ast.StopStatement:
		p.write("stop")
	case *ast.SkipStatement:
		p.write("skip")

	// Table literal
	case *ast.TableLiteral:
		p.formatTableLiteral(n)

	// Query DSL expressions
	case *ast.QueryExpression:
		p.formatQueryExpression(n)
	case *ast.InsertExpression:
		p.formatInsertExpression(n)
	case *ast.UpdateExpression:
		p.formatUpdateExpression(n)
	case *ast.DeleteExpression:
		p.formatDeleteExpression(n)
	case *ast.TransactionExpression:
		p.formatTransactionExpression(n)

	default:
		// Fall back to the node's String() method
		p.write(node.String())
	}
}

// formatStatement formats a statement (with trailing newline if needed)
func (p *Printer) formatStatement(stmt ast.Statement) {
	p.formatNode(stmt)
}

// formatExpression formats an expression
func (p *Printer) formatExpression(expr ast.Expression) {
	p.formatNode(expr)
}

// formatLetStatement formats let statements: let x = value
func (p *Printer) formatLetStatement(ls *ast.LetStatement) {
	if ls.Export {
		p.write("export ")
	}
	p.write("let ")

	if ls.DictPattern != nil {
		p.formatDictDestructuringPattern(ls.DictPattern)
	} else if ls.ArrayPattern != nil {
		p.formatArrayDestructuringPattern(ls.ArrayPattern)
	} else {
		p.write(ls.Name.Value)
	}

	p.write(" = ")
	p.formatExpression(ls.Value)
}

// formatAssignmentStatement formats assignment statements: x = value
func (p *Printer) formatAssignmentStatement(as *ast.AssignmentStatement) {
	if as.Export {
		p.write("export ")
	}

	if as.DictPattern != nil {
		p.formatDictDestructuringPattern(as.DictPattern)
	} else if as.ArrayPattern != nil {
		p.formatArrayDestructuringPattern(as.ArrayPattern)
	} else {
		p.write(as.Name.Value)
	}

	p.write(" = ")
	p.formatExpression(as.Value)
}

// formatReturnStatement formats return statements
func (p *Printer) formatReturnStatement(rs *ast.ReturnStatement) {
	p.write("return")
	if rs.ReturnValue != nil {
		p.write(" ")
		p.formatExpression(rs.ReturnValue)
	}
}

// formatExpressionStatement formats expression statements
func (p *Printer) formatExpressionStatement(es *ast.ExpressionStatement) {
	p.formatExpression(es.Expression)
}

// formatBlockStatement formats block statements: { ... }
func (p *Printer) formatBlockStatement(bs *ast.BlockStatement) {
	if bs == nil || len(bs.Statements) == 0 {
		p.write("{}")
		return
	}

	// Check if we can format inline (single simple expression)
	if len(bs.Statements) == 1 {
		if es, ok := bs.Statements[0].(*ast.ExpressionStatement); ok {
			// Don't inline if body contains control flow - these deserve their own lines
			if containsControlFlow(es.Expression) {
				// Fall through to multiline
			} else {
				// Don't inline if there are comments
				tok := getStatementToken(es)
				if tok == nil || (len(tok.LeadingComments) == 0 && tok.TrailingComment == "") {
					inline := fmt.Sprintf("{ %s }", nodeString(es.Expression))
					// Use position-aware check to avoid overly long lines when nested
					if p.fitsOnLine(inline, MaxLineWidth) && !containsNewline(es.Expression) {
						p.write(inline)
						return
					}
				}
			}
		}
	}

	// Multiline format
	p.write("{")

	// Check if first statement has a trailing comment (which belongs to the { line)
	if len(bs.Statements) > 0 {
		tok := getStatementToken(bs.Statements[0])
		if tok != nil && tok.TrailingComment != "" {
			p.write(" ")
			p.write(tok.TrailingComment)
		}
	}

	p.newline()
	p.indentInc()

	for i, stmt := range bs.Statements {
		tok := getStatementToken(stmt)

		// For the first statement in block
		if i == 0 {
			// Blank line if source had one (after the { line)
			if tok != nil && tok.BlankLinesBefore > 0 {
				p.newline()
			}
			// Output any leading comments
			p.writeComments(tok)
			p.writeIndent()
			p.formatStatement(stmt)
			continue
		}

		// For subsequent statements:
		// 1. Output trailing comment from previous statement
		if tok != nil && tok.TrailingComment != "" {
			p.write(" ")
			p.write(tok.TrailingComment)
		}

		// 2. Newline
		p.newline()

		// 3. Blank line if source had one
		if tok != nil && tok.BlankLinesBefore > 0 {
			p.newline()
		}

		// 4. Leading comments
		p.writeComments(tok)

		// 5. Statement
		p.writeIndent()
		p.formatStatement(stmt)
	}

	// Handle trailing comment on last statement
	if len(bs.Statements) > 0 {
		// We need to check if there's a trailing comment that belongs to the last statement
		// This would be on the closing brace token, but we don't have access to it here
		// For now, just add the newline
		p.newline()
	}

	p.indentDec()
	p.writeIndent()
	p.write("}")
}

// formatExportNameStatement formats export statements
func (p *Printer) formatExportNameStatement(es *ast.ExportNameStatement) {
	p.write("export ")
	p.write(es.Name.Value)
}

// formatComputedExportStatement formats computed export statements
func (p *Printer) formatComputedExportStatement(ces *ast.ComputedExportStatement) {
	p.write("export computed ")
	p.write(ces.Name.Value)

	if bs, ok := ces.Body.(*ast.BlockStatement); ok {
		p.write(" ")
		p.formatBlockStatement(bs)
	} else {
		p.write(" = ")
		p.formatNode(ces.Body)
	}
}

// formatIndexAssignmentStatement formats index assignment: dict["key"] = value
func (p *Printer) formatIndexAssignmentStatement(ias *ast.IndexAssignmentStatement) {
	p.formatExpression(ias.Target)
	p.write(" = ")
	p.formatExpression(ias.Value)
}

// formatStringLiteral formats string literals with double quotes
func (p *Printer) formatStringLiteral(sl *ast.StringLiteral) {
	// Always use double quotes for string output
	p.write(`"` + escapeString(sl.Value) + `"`)
}

// formatArrayLiteral formats array literals with threshold-based inline/multiline
func (p *Printer) formatArrayLiteral(al *ast.ArrayLiteral) {
	if len(al.Elements) == 0 {
		p.write("[]")
		return
	}

	// Try inline format - use position-aware check
	inline := formatArrayLiteralInline(al)
	if p.fitsOnLine(inline, MaxLineWidth) {
		p.write(inline)
		return
	}

	// Multiline format
	p.write("[")
	p.newline()
	p.indentInc()

	for i, elem := range al.Elements {
		p.writeIndent()
		p.formatExpression(elem)
		if TrailingCommaMultiline || i < len(al.Elements)-1 {
			p.write(",")
		}
		p.newline()
	}

	p.indentDec()
	p.writeIndent()
	p.write("]")
}

// formatArrayLiteralInline formats an array as a single line
func formatArrayLiteralInline(al *ast.ArrayLiteral) string {
	parts := make([]string, len(al.Elements))
	for i, elem := range al.Elements {
		parts[i] = nodeString(elem)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// formatDictionaryLiteral formats dictionary literals with threshold-based inline/multiline
func (p *Printer) formatDictionaryLiteral(dl *ast.DictionaryLiteral) {
	hasEntries := len(dl.KeyOrder) > 0 || len(dl.ComputedPairs) > 0
	if !hasEntries {
		p.write("{}")
		return
	}

	// Try inline format - use position-aware check
	inline := formatDictLiteralInline(dl)
	if p.fitsOnLine(inline, MaxLineWidth) {
		p.write(inline)
		return
	}

	// Multiline format
	p.write("{")
	p.newline()
	p.indentInc()

	// Format regular pairs first
	totalItems := len(dl.KeyOrder) + len(dl.ComputedPairs)
	idx := 0

	for _, key := range dl.KeyOrder {
		p.writeIndent()
		p.write(formatDictKey(key))
		p.write(": ")
		if val, ok := dl.Pairs[key]; ok {
			p.formatExpression(val)
		} else {
			p.write("null")
		}
		if TrailingCommaMultiline || idx < totalItems-1 {
			p.write(",")
		}
		p.newline()
		idx++
	}

	// Format computed pairs
	for _, cp := range dl.ComputedPairs {
		p.writeIndent()
		p.write("[")
		p.formatExpression(cp.Key)
		p.write("]: ")
		p.formatExpression(cp.Value)
		if TrailingCommaMultiline || idx < totalItems-1 {
			p.write(",")
		}
		p.newline()
		idx++
	}

	p.indentDec()
	p.writeIndent()
	p.write("}")
}

// formatDictLiteralInline formats a dictionary as a single line
func formatDictLiteralInline(dl *ast.DictionaryLiteral) string {
	parts := make([]string, 0, len(dl.KeyOrder)+len(dl.ComputedPairs))

	for _, key := range dl.KeyOrder {
		val := "null"
		if v, ok := dl.Pairs[key]; ok {
			val = nodeString(v)
		}
		parts = append(parts, formatDictKey(key)+": "+val)
	}

	for _, cp := range dl.ComputedPairs {
		parts = append(parts, "["+nodeString(cp.Key)+"]: "+nodeString(cp.Value))
	}

	return "{" + strings.Join(parts, ", ") + "}"
}

// formatFunctionLiteral formats function literals with threshold-based inline/multiline
func (p *Printer) formatFunctionLiteral(fl *ast.FunctionLiteral) {
	// Format params
	paramStrs := make([]string, len(fl.Params))
	for i, param := range fl.Params {
		paramStrs[i] = param.String()
	}
	paramsLine := strings.Join(paramStrs, ", ")

	// Check if we can format inline - use indent-aware check
	bodyStr := ""
	if fl.Body != nil && len(fl.Body.Statements) == 1 {
		if es, ok := fl.Body.Statements[0].(*ast.ExpressionStatement); ok {
			bodyStr = nodeString(es.Expression)
		}
	}

	if bodyStr != "" && !strings.Contains(bodyStr, "\n") {
		inline := fmt.Sprintf("fn(%s) { %s }", paramsLine, bodyStr)
		if p.fitsOnLine(inline, MaxLineWidth) {
			p.write(inline)
			return
		}
	}

	// Multiline format
	p.write("fn(")

	// Check if params need multiline
	if len(fl.Params) > 3 || !p.fitsOnLine("fn("+paramsLine+")", MaxLineWidth) {
		p.newline()
		p.indentInc()
		for i, param := range fl.Params {
			p.writeIndent()
			p.write(param.String())
			if TrailingCommaFuncParams || i < len(fl.Params)-1 {
				p.write(",")
			}
			p.newline()
		}
		p.indentDec()
		p.writeIndent()
	} else {
		p.write(paramsLine)
	}

	p.write(") ")
	p.formatBlockStatement(fl.Body)
}

// formatCallExpression formats function calls with threshold-based formatting
func (p *Printer) formatCallExpression(ce *ast.CallExpression) {
	// Check if this is a method call that's part of a chain
	if dot, ok := ce.Function.(*ast.DotExpression); ok {
		chain := collectMethodChain(ce)
		if len(chain) > 1 {
			// This is a method chain - check if it fits inline
			// Break chains with >2 method calls for readability, or if line is too long
			inline := formatChainInline(chain)
			methodCallCount := countMethodCalls(chain)
			if methodCallCount <= 2 && p.fitsOnLine(inline, MaxLineWidth) && !strings.Contains(inline, "\n") {
				p.write(inline)
				return
			}
			// Doesn't fit or too many calls - format as multiline chain
			p.formatMethodChain(chain)
			return
		}
		// Single method call, not a chain - fall through to normal formatting
		_ = dot // silence unused warning
	}

	// Format the function being called
	p.formatExpression(ce.Function)
	p.write("(")

	if len(ce.Arguments) == 0 {
		p.write(")")
		return
	}

	// Try inline format
	argStrs := make([]string, len(ce.Arguments))
	for i, arg := range ce.Arguments {
		argStrs[i] = nodeString(arg)
	}
	argsLine := strings.Join(argStrs, ", ")

	funcStr := nodeString(ce.Function)
	fullInline := funcStr + "(" + argsLine + ")"

	// Use position-aware check for inline formatting
	if p.fitsOnLine(fullInline, MaxLineWidth) && !strings.Contains(argsLine, "\n") {
		p.write(argsLine)
		p.write(")")
		return
	}

	// Multiline format
	p.newline()
	p.indentInc()
	for i, arg := range ce.Arguments {
		p.writeIndent()
		p.formatExpression(arg)
		if TrailingCommaFuncCalls || i < len(ce.Arguments)-1 {
			p.write(",")
		}
		p.newline()
	}
	p.indentDec()
	p.writeIndent()
	p.write(")")
}

// formatDotExpression formats dot/method access expressions
func (p *Printer) formatDotExpression(de *ast.DotExpression) {
	// Check if this is part of a method chain
	chain := collectMethodChain(de)
	if len(chain) <= 1 {
		// Not really a chain, format normally
		p.formatExpression(de.Left)
		p.write(".")
		p.write(de.Key)
		return
	}

	// Format as chain - check if it fits inline (position-aware)
	// Break chains with >2 method calls for readability, or if line is too long
	inline := formatChainInline(chain)
	methodCallCount := countMethodCalls(chain)
	if methodCallCount <= 2 && p.fitsOnLine(inline, MaxLineWidth) && !strings.Contains(inline, "\n") {
		p.write(inline)
		return
	}

	// Doesn't fit or too many calls - format as multiline chain
	p.formatMethodChain(chain)
}

// countMethodCalls counts how many links in the chain are method calls (not property accesses)
func countMethodCalls(chain []chainLink) int {
	count := 0
	for _, link := range chain {
		if link.call != nil {
			count++
		}
	}
	return count
}

// formatMethodChain formats a method chain with one call per line
func (p *Printer) formatMethodChain(chain []chainLink) {
	// Base expression on first line
	p.formatExpression(chain[0].base)
	p.newline()
	p.indentInc()
	p.indentInc() // Extra indent for chain continuation
	for i, link := range chain {
		p.writeIndent()
		p.write(".")
		p.write(link.property)
		if link.call != nil {
			p.write("(")
			if len(link.call.Arguments) > 0 {
				argStrs := make([]string, len(link.call.Arguments))
				for j, arg := range link.call.Arguments {
					argStrs[j] = nodeString(arg)
				}
				p.write(strings.Join(argStrs, ", "))
			}
			p.write(")")
		}
		// Only add newline between links, not after the last one
		if i < len(chain)-1 {
			p.newline()
		}
	}
	p.indentDec()
	p.indentDec()
}

// chainLink represents one link in a method chain
type chainLink struct {
	base     ast.Expression      // the base expression (only for first link)
	property string              // the property name
	call     *ast.CallExpression // nil if property access, non-nil if method call
}

// collectMethodChain collects all links in a method chain
func collectMethodChain(expr ast.Expression) []chainLink {
	var chain []chainLink
	current := expr

	for {
		switch e := current.(type) {
		case *ast.CallExpression:
			// Check if this is a method call
			if dot, ok := e.Function.(*ast.DotExpression); ok {
				chain = append([]chainLink{{
					property: dot.Key,
					call:     e,
				}}, chain...)
				current = dot.Left
				continue
			}
			// Function call but not a method
			return nil

		case *ast.DotExpression:
			chain = append([]chainLink{{
				property: e.Key,
			}}, chain...)
			current = e.Left
			continue

		default:
			// Base of chain
			if len(chain) > 0 {
				chain[0].base = current
			}
			return chain
		}
	}
}

// formatChainInline formats a method chain as a single line
func formatChainInline(chain []chainLink) string {
	if len(chain) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(nodeString(chain[0].base))

	for _, link := range chain {
		sb.WriteString(".")
		sb.WriteString(link.property)
		if link.call != nil {
			sb.WriteString("(")
			for i, arg := range link.call.Arguments {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(nodeString(arg))
			}
			sb.WriteString(")")
		}
	}

	return sb.String()
}

// formatIndexExpression formats index access: arr[0]
func (p *Printer) formatIndexExpression(ie *ast.IndexExpression) {
	p.formatExpression(ie.Left)
	p.write("[")
	if ie.Optional {
		p.write("?")
	}
	p.formatExpression(ie.Index)
	p.write("]")
}

// formatSliceExpression formats slice access: arr[1:4]
func (p *Printer) formatSliceExpression(se *ast.SliceExpression) {
	p.formatExpression(se.Left)
	p.write("[")
	if se.Start != nil {
		p.formatExpression(se.Start)
	}
	p.write(":")
	if se.End != nil {
		p.formatExpression(se.End)
	}
	p.write("]")
}

// formatPrefixExpression formats prefix expressions: !x, -x
func (p *Printer) formatPrefixExpression(pe *ast.PrefixExpression) {
	p.write(pe.Operator)
	p.formatExpression(pe.Right)
}

// formatInfixExpression formats infix expressions: x + y
func (p *Printer) formatInfixExpression(ie *ast.InfixExpression) {
	// Special handling for operators that don't need spaces
	noSpaceOps := map[string]bool{}

	p.formatExpression(ie.Left)
	if noSpaceOps[ie.Operator] {
		p.write(ie.Operator)
	} else {
		p.write(" " + ie.Operator + " ")
	}
	p.formatExpression(ie.Right)
}

// formatIfExpression formats if expressions with threshold-based formatting
func (p *Printer) formatIfExpression(ie *ast.IfExpression) {
	// Check if we can format as single-line ternary-style (position-aware)
	if ie.Alternative != nil && canInlineIf(ie) {
		inline := formatIfInline(ie)
		if p.fitsOnLine(inline, MaxLineWidth) {
			p.write(inline)
			return
		}
	}

	// For if-without-else with a single simple expression, use braceless form
	if ie.Alternative == nil && canBracelessIf(ie) {
		inline := formatBracelessIf(ie)
		if p.fitsOnLine(inline, MaxLineWidth) {
			p.write(inline)
			return
		}
	}

	// Multiline format
	p.write("if (")
	p.formatExpression(ie.Condition)
	p.write(") ")
	p.formatBlockStatement(ie.Consequence)

	if ie.Alternative != nil {
		// Check if alternative is an else-if
		if len(ie.Alternative.Statements) == 1 {
			if es, ok := ie.Alternative.Statements[0].(*ast.ExpressionStatement); ok {
				if elseIf, ok := es.Expression.(*ast.IfExpression); ok {
					p.write(" else ")
					p.formatIfExpression(elseIf)
					return
				}
			}
		}

		p.write(" else ")
		p.formatBlockStatement(ie.Alternative)
	}
}

// canBracelessIf checks if an if-without-else can be formatted without braces
func canBracelessIf(ie *ast.IfExpression) bool {
	// Must have no alternative
	if ie.Alternative != nil {
		return false
	}
	// Must have single expression in consequence
	if ie.Consequence == nil || len(ie.Consequence.Statements) != 1 {
		return false
	}
	// Must be an expression statement
	if _, ok := ie.Consequence.Statements[0].(*ast.ExpressionStatement); !ok {
		return false
	}
	// Must not contain newlines
	return !containsNewline(ie.Consequence)
}

// formatBracelessIf formats an if-without-else as braceless single line
func formatBracelessIf(ie *ast.IfExpression) string {
	cond := nodeString(ie.Condition)
	body := ""
	if es, ok := ie.Consequence.Statements[0].(*ast.ExpressionStatement); ok {
		body = nodeString(es.Expression)
	}
	return fmt.Sprintf("if (%s) %s", cond, body)
}

// canInlineIf checks if an if expression can be formatted inline
func canInlineIf(ie *ast.IfExpression) bool {
	// Must have simple single-expression consequence and alternative
	if ie.Consequence == nil || len(ie.Consequence.Statements) != 1 {
		return false
	}
	if ie.Alternative == nil || len(ie.Alternative.Statements) != 1 {
		return false
	}

	// Both must be expression statements
	if _, ok := ie.Consequence.Statements[0].(*ast.ExpressionStatement); !ok {
		return false
	}
	if _, ok := ie.Alternative.Statements[0].(*ast.ExpressionStatement); !ok {
		return false
	}

	// Neither can contain newlines
	return !containsNewline(ie.Consequence) && !containsNewline(ie.Alternative)
}

// formatIfInline formats an if expression as a single line
func formatIfInline(ie *ast.IfExpression) string {
	cond := nodeString(ie.Condition)
	cons := ""
	alt := ""

	if es, ok := ie.Consequence.Statements[0].(*ast.ExpressionStatement); ok {
		cons = nodeString(es.Expression)
	}
	if es, ok := ie.Alternative.Statements[0].(*ast.ExpressionStatement); ok {
		alt = nodeString(es.Expression)
	}

	return fmt.Sprintf("if (%s) %s else %s", cond, cons, alt)
}

// formatForExpression formats for expressions
func (p *Printer) formatForExpression(fe *ast.ForExpression) {
	p.write("for (")

	if fe.ValueVariable != nil {
		// key, value in dict form
		p.write(fe.Variable.Value)
		p.write(", ")
		p.write(fe.ValueVariable.Value)
		p.write(" in ")
	} else if fe.Variable != nil {
		// value in array form
		p.write(fe.Variable.Value)
		p.write(" in ")
	}

	p.formatExpression(fe.Array)
	p.write(")")

	if fe.Function != nil {
		// Simple form: for(arr) fn
		p.write(" ")
		p.formatExpression(fe.Function)
	} else if fe.Body != nil {
		// The parser wraps the body in a FunctionLiteral - we need to unwrap it
		// and format just the block statement inside
		if fnBody, ok := fe.Body.(*ast.FunctionLiteral); ok && fnBody.Body != nil {
			// Format the block statement from inside the function literal
			p.write(" ")
			p.formatBlockStatement(fnBody.Body)
		} else {
			// Fallback: format body directly (shouldn't happen)
			bodyStr := nodeString(fe.Body)
			varStr := ""
			if fe.Variable != nil {
				varStr = fe.Variable.Value + " in "
			}
			inline := fmt.Sprintf("for (%s%s) { %s }",
				varStr, nodeString(fe.Array), bodyStr)

			// Use position-aware check for inline formatting
			if p.fitsOnLine(inline, MaxLineWidth) && !strings.Contains(bodyStr, "\n") {
				p.write(" { ")
				p.formatExpression(fe.Body)
				p.write(" }")
			} else {
				p.write(" {")
				p.newline()
				p.indentInc()
				p.writeIndent()
				p.formatExpression(fe.Body)
				p.newline()
				p.indentDec()
				p.writeIndent()
				p.write("}")
			}
		}
	}
}

// formatGroupedExpression formats grouped/parenthesized expressions
func (p *Printer) formatGroupedExpression(ge *ast.GroupedExpression) {
	p.write("(")
	p.formatExpression(ge.Inner)
	p.write(")")
}

// formatIsExpression formats is/is not expressions
func (p *Printer) formatIsExpression(ie *ast.IsExpression) {
	p.formatExpression(ie.Value)
	if ie.Negated {
		p.write(" is not ")
	} else {
		p.write(" is ")
	}
	p.formatExpression(ie.Schema)
}

// formatInterpolationBlock formats interpolation blocks (statements inside tags)
// These are let/assignment statements that appear directly inside tag content
func (p *Printer) formatInterpolationBlock(ib *ast.InterpolationBlock) {
	// For single statements that are expressions or simple statements,
	// we can format them without the block wrapper
	if len(ib.Statements) == 1 {
		p.formatStatement(ib.Statements[0])
		return
	}

	// Multiple statements need the block wrapper
	p.write("{")
	p.newline()
	p.indentInc()
	for _, stmt := range ib.Statements {
		p.writeIndent()
		p.formatStatement(stmt)
		p.newline()
	}
	p.indentDec()
	p.writeIndent()
	p.write("}")
}

// Helper functions

// nodeString returns the string representation of a node
func nodeString(node ast.Node) string {
	if node == nil {
		return ""
	}
	// Use the formatting function recursively
	return FormatNode(node)
}

// containsNewline checks if a node's string contains newlines
func containsNewline(node ast.Node) bool {
	if node == nil {
		return false
	}
	return strings.Contains(node.String(), "\n")
}

// containsControlFlow checks if an expression contains control flow (if, for)
// These expressions deserve their own lines for readability
func containsControlFlow(expr ast.Expression) bool {
	switch expr.(type) {
	case *ast.IfExpression, *ast.ForExpression:
		return true
	}
	return false
}

// escapeString escapes special characters in a string
func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\t", "\\t")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}

// formatArrayDestructuringPattern formats array destructuring: [a, b, ...rest]
func (p *Printer) formatArrayDestructuringPattern(ap *ast.ArrayDestructuringPattern) {
	p.write("[")
	for i, name := range ap.Names {
		if i > 0 {
			p.write(", ")
		}
		p.write(name.Value)
	}
	if ap.Rest != nil {
		if len(ap.Names) > 0 {
			p.write(", ")
		}
		p.write("...")
		p.write(ap.Rest.Value)
	}
	p.write("]")
}

// formatDictDestructuringPattern formats dict destructuring: {a, b, ...rest}
func (p *Printer) formatDictDestructuringPattern(dp *ast.DictDestructuringPattern) {
	p.write("{")
	for i, key := range dp.Keys {
		if i > 0 {
			p.write(", ")
		}
		p.write(key.String())
	}
	if dp.Rest != nil {
		if len(dp.Keys) > 0 {
			p.write(", ")
		}
		p.write("...")
		p.write(dp.Rest.Value)
	}
	p.write("}")
}

// formatCheckStatement formats check guards: check condition else value
func (p *Printer) formatCheckStatement(cs *ast.CheckStatement) {
	p.write("check ")
	p.formatExpression(cs.Condition)
	p.write(" else ")
	p.formatExpression(cs.ElseValue)
}

// formatTagLiteral formats self-closing tags: <input type=text/>
func (p *Printer) formatTagLiteral(tl *ast.TagLiteral) {
	p.write("<")
	// Raw already contains all attributes including spreads
	p.write(tl.Raw)
	p.write("/>")
}

// formatTagPairExpression formats paired tags: <div>content</div>
func (p *Printer) formatTagPairExpression(tp *ast.TagPairExpression) {
	// Opening tag
	if tp.Name == "" {
		p.write("<>")
	} else {
		p.write("<")
		p.write(tp.Name)
		// Props already contains the raw attributes including spreads
		// So we just output Props as-is (spreads are NOT separate)
		if tp.Props != "" {
			p.write(" ")
			p.write(tp.Props)
		}
		p.write(">")
	}

	// Special handling for style/script tags - normalize whitespace in text content
	isPreformattedTag := tp.Name == "style" || tp.Name == "script"

	// Check if contents can fit inline
	hasComplexContent := false
	for _, content := range tp.Contents {
		switch c := content.(type) {
		case *ast.TextNode:
			if strings.Contains(c.Value, "\n") {
				hasComplexContent = true
			}
		case *ast.ForExpression, *ast.IfExpression, *ast.TagPairExpression, *ast.InterpolationBlock:
			hasComplexContent = true
		}
	}

	if !hasComplexContent && len(tp.Contents) <= 1 {
		// Inline content
		for _, content := range tp.Contents {
			p.formatNode(content)
		}
	} else if isPreformattedTag {
		// For style/script tags, normalize whitespace in text content
		p.newline()
		for _, content := range tp.Contents {
			switch c := content.(type) {
			case *ast.TextNode:
				// Dedent the text, then re-indent at current level + 1
				dedented := strings.TrimSpace(dedentText(c.Value))
				if dedented != "" {
					reindented := reindentText(dedented, p.indent+1)
					p.write(reindented)
					p.newline()
				}
			default:
				// Check for leading comments on this content
				tok := getNodeToken(content)
				p.writeComments(tok)
				p.writeIndent()
				p.indentInc()
				p.formatNode(content)
				p.indentDec()
				p.newline()
			}
		}
		p.writeIndent()
	} else {
		// Multiline content
		p.newline()
		p.indentInc()
		for i, content := range tp.Contents {
			// Check for leading comments and blank lines on this content
			tok := getNodeToken(content)
			
			// Blank line separator (but not before first child)
			if i > 0 && tok != nil && tok.BlankLinesBefore > 0 {
				p.newline()
			}
			
			// Leading comments
			p.writeComments(tok)
			
			p.writeIndent()
			p.formatNode(content)
			p.newline()
		}
		p.indentDec()
		p.writeIndent()
	}

	// Closing tag
	if tp.Name == "" {
		p.write("</>")
	} else {
		p.write("</")
		p.write(tp.Name)
		p.write(">")
	}
}

// formatSchemaDeclaration formats schema definitions: @schema Name { fields }
func (p *Printer) formatSchemaDeclaration(sd *ast.SchemaDeclaration) {
	if sd.Export {
		p.write("export ")
	}
	p.write("@schema ")
	p.write(sd.Name.Value)
	p.write(" {")

	if len(sd.Fields) == 0 {
		p.write("}")
		return
	}

	p.newline()
	p.indentInc()

	for _, field := range sd.Fields {
		p.writeIndent()
		p.formatSchemaField(field)
		p.newline()
	}

	p.indentDec()
	p.writeIndent()
	p.write("}")
}

// formatSchemaField formats a single schema field
func (p *Printer) formatSchemaField(sf *ast.SchemaField) {
	p.write(sf.Name.Value)
	p.write(": ")

	if sf.IsArray {
		p.write("[")
	}
	p.write(sf.TypeName)
	if sf.IsArray {
		p.write("]")
	}
	if sf.Nullable {
		p.write("?")
	}

	// Type options
	if len(sf.TypeOptions) > 0 && len(sf.EnumValues) == 0 {
		p.write("(")
		first := true
		for k, v := range sf.TypeOptions {
			if !first {
				p.write(", ")
			}
			p.write(k)
			p.write(": ")
			p.formatExpression(v)
			first = false
		}
		p.write(")")
	}

	// Enum values
	if len(sf.EnumValues) > 0 {
		p.write("[")
		for i, v := range sf.EnumValues {
			if i > 0 {
				p.write(", ")
			}
			p.write(`"`)
			p.write(v)
			p.write(`"`)
		}
		p.write("]")
	}

	// Metadata
	if sf.Metadata != nil {
		p.write(" | ")
		p.formatDictionaryLiteral(sf.Metadata)
	}

	// Foreign key
	if sf.ForeignKey != "" {
		p.write(" via ")
		p.write(sf.ForeignKey)
	}

	// Default value
	if sf.DefaultValue != nil {
		p.write(" = ")
		p.formatExpression(sf.DefaultValue)
	}
}

// ============================================================================
// Table Literal Formatting
// ============================================================================

// formatTableLiteral formats @table literals with multiline row layout
func (p *Printer) formatTableLiteral(tl *ast.TableLiteral) {
	p.write("@table")
	if tl.Schema != nil {
		p.write("(")
		p.write(tl.Schema.Value)
		p.write(")")
	}
	p.write(" [")

	if len(tl.Rows) == 0 {
		p.write("]")
		return
	}

	// Check if table can fit inline (single row, short) - position-aware
	if len(tl.Rows) == 1 {
		inline := formatTableLiteralInline(tl)
		if p.fitsOnLine(inline, MaxLineWidth) {
			p.write(inline[len("@table ["):]) // Skip the "@table [" prefix we already wrote
			return
		}
	}

	// Multiline format - each row on its own line
	p.newline()
	p.indentInc()

	for i, row := range tl.Rows {
		p.writeIndent()
		p.formatDictionaryLiteral(row)
		if TrailingCommaMultiline || i < len(tl.Rows)-1 {
			p.write(",")
		}
		p.newline()
	}

	p.indentDec()
	p.writeIndent()
	p.write("]")
}

// formatTableLiteralInline formats a table literal inline
func formatTableLiteralInline(tl *ast.TableLiteral) string {
	var parts []string
	for _, row := range tl.Rows {
		parts = append(parts, formatDictLiteralInline(row))
	}
	result := "@table"
	if tl.Schema != nil {
		result += "(" + tl.Schema.Value + ")"
	}
	result += " [" + strings.Join(parts, ", ") + "]"
	return result
}

// ============================================================================
// Query DSL Formatting
// ============================================================================

// queryClauseCount counts the number of clauses in a query for threshold decisions
func queryClauseCount(qe *ast.QueryExpression) int {
	count := len(qe.Conditions) + len(qe.Modifiers) + len(qe.ComputedFields)
	if len(qe.GroupBy) > 0 {
		count++
	}
	if qe.Terminal != nil {
		count++
	}
	return count
}

// formatQueryExpression formats @query expressions with proper indentation
// Design rules:
// - Short queries (≤2 clauses AND ≤60 chars): single line
// - Longer queries: table name on own line, one clause per line
func (p *Printer) formatQueryExpression(qe *ast.QueryExpression) {
	// First, check if we can do inline format
	inline := p.queryExpressionInline(qe)
	if len(inline) <= QueryInlineThreshold && queryClauseCount(qe) <= 2 {
		p.write(inline)
		return
	}

	// Multi-line format
	p.write("@query(")
	p.indent++
	p.newline()
	p.writeIndent()

	// CTEs first
	for _, cte := range qe.CTEs {
		p.formatQueryCTE(cte)
		p.newline()
		p.newline()
		p.writeIndent()
	}

	// Source table
	p.write(qe.Source.Value)
	if qe.SourceAlias != nil {
		p.write(" as ")
		p.write(qe.SourceAlias.Value)
	}

	// Conditions - one per line
	for _, cond := range qe.Conditions {
		p.newline()
		p.writeIndent()
		p.write("| ")
		p.formatQueryCondition(cond)
	}

	// Group by
	if len(qe.GroupBy) > 0 {
		p.newline()
		p.writeIndent()
		p.write("+ by ")
		p.write(joinStrings(qe.GroupBy, ", "))
	}

	// Computed fields
	for _, cf := range qe.ComputedFields {
		p.newline()
		p.writeIndent()
		p.write("| ")
		p.formatQueryComputedField(cf)
	}

	// Modifiers (order, limit, with, etc.)
	for _, mod := range qe.Modifiers {
		p.newline()
		p.writeIndent()
		p.write("| ")
		p.formatQueryModifier(mod)
	}

	// Terminal
	if qe.Terminal != nil {
		p.newline()
		p.writeIndent()
		p.formatQueryTerminal(qe.Terminal)
	}

	p.indent--
	p.newline()
	p.writeIndent()
	p.write(")")
}

// queryExpressionInline returns a single-line representation of the query
func (p *Printer) queryExpressionInline(qe *ast.QueryExpression) string {
	var parts []string
	parts = append(parts, qe.Source.Value)
	if qe.SourceAlias != nil {
		parts[0] += " as " + qe.SourceAlias.Value
	}

	// Build inline clause string
	var clauses []string

	for _, cond := range qe.Conditions {
		clauses = append(clauses, "| "+conditionToString(cond))
	}

	if len(qe.GroupBy) > 0 {
		clauses = append(clauses, "+ by "+joinStrings(qe.GroupBy, ", "))
	}

	for _, cf := range qe.ComputedFields {
		clauses = append(clauses, "| "+computedFieldToString(cf))
	}

	for _, mod := range qe.Modifiers {
		clauses = append(clauses, "| "+modifierToString(mod))
	}

	result := "@query(" + parts[0]
	for _, clause := range clauses {
		result += " " + clause
	}

	if qe.Terminal != nil {
		result += " " + terminalToString(qe.Terminal)
	}
	result += ")"

	return result
}

// formatQueryCTE formats a Common Table Expression
func (p *Printer) formatQueryCTE(cte *ast.QueryCTE) {
	p.write(cte.Source.Value)
	p.write(" as ")
	p.write(cte.Name)

	for _, cond := range cte.Conditions {
		p.newline()
		p.writeIndent()
		p.write("| ")
		p.formatQueryCondition(cond)
	}

	for _, mod := range cte.Modifiers {
		p.newline()
		p.writeIndent()
		p.write("| ")
		p.formatQueryModifier(mod)
	}

	if cte.Terminal != nil {
		p.newline()
		p.writeIndent()
		p.formatQueryTerminal(cte.Terminal)
	}
}

// formatQueryCondition formats a query condition (WHERE clause part)
func (p *Printer) formatQueryCondition(cond ast.QueryConditionNode) {
	p.write(conditionToString(cond))
}

// conditionToString converts a condition node to string with proper value formatting
func conditionToString(cond ast.QueryConditionNode) string {
	switch c := cond.(type) {
	case *ast.QueryCondition:
		return formatQueryConditionString(c)
	case *ast.QueryConditionGroup:
		return formatQueryConditionGroupString(c)
	default:
		return cond.ConditionString()
	}
}

// formatQueryConditionString formats a single query condition
func formatQueryConditionString(qc *ast.QueryCondition) string {
	var out strings.Builder
	if qc.Logic != "" {
		out.WriteString(qc.Logic)
		out.WriteString(" ")
	}
	if qc.Negated {
		out.WriteString("not ")
	}
	out.WriteString(formatQueryExpressionValue(qc.Left))
	out.WriteString(" ")
	out.WriteString(qc.Operator)
	if qc.Right != nil {
		out.WriteString(" ")
		out.WriteString(formatQueryExpressionValue(qc.Right))
	}
	if qc.RightEnd != nil {
		out.WriteString(" and ")
		out.WriteString(formatQueryExpressionValue(qc.RightEnd))
	}
	return out.String()
}

// formatQueryConditionGroupString formats a condition group
func formatQueryConditionGroupString(qcg *ast.QueryConditionGroup) string {
	var out strings.Builder
	if qcg.Logic != "" {
		out.WriteString(qcg.Logic)
		out.WriteString(" ")
	}
	if qcg.Negated {
		out.WriteString("not ")
	}
	out.WriteString("(")
	for i, cond := range qcg.Conditions {
		if i > 0 {
			out.WriteString(" ")
		}
		out.WriteString(conditionToString(cond))
	}
	out.WriteString(")")
	return out.String()
}

// formatQueryExpressionValue formats an expression value in a query context
// This handles proper quoting of strings and formatting of other types
func formatQueryExpressionValue(expr ast.Expression) string {
	if expr == nil {
		return ""
	}

	switch e := expr.(type) {
	case *ast.StringLiteral:
		// String values need to be quoted
		return `"` + escapeString(e.Value) + `"`
	case *ast.IntegerLiteral:
		return e.String()
	case *ast.FloatLiteral:
		return e.String()
	case *ast.Boolean:
		return e.String()
	case *ast.QueryColumnRef:
		return e.Column
	case *ast.QueryInterpolation:
		return "{" + nodeString(e.Expression) + "}"
	case *ast.Identifier:
		return e.Value
	case *ast.DotExpression:
		// table.column reference
		return e.String()
	case *ast.ArrayLiteral:
		// For IN clauses
		var elements []string
		for _, elem := range e.Elements {
			elements = append(elements, formatQueryExpressionValue(elem))
		}
		return "[" + strings.Join(elements, ", ") + "]"
	case *ast.QuerySubquery:
		// Subquery in condition
		return e.String()
	default:
		return expr.String()
	}
}

// formatQueryComputedField formats computed/aggregate fields
func (p *Printer) formatQueryComputedField(cf *ast.QueryComputedField) {
	p.write(computedFieldToString(cf))
}

// computedFieldToString converts a computed field to string
func computedFieldToString(cf *ast.QueryComputedField) string {
	return cf.String()
}

// formatQueryModifier formats ORDER BY, LIMIT, WITH, etc.
func (p *Printer) formatQueryModifier(mod *ast.QueryModifier) {
	p.write(modifierToString(mod))
}

// modifierToString converts a modifier to string
func modifierToString(mod *ast.QueryModifier) string {
	return mod.String()
}

// formatQueryTerminal formats the return type and projection
func (p *Printer) formatQueryTerminal(qt *ast.QueryTerminal) {
	p.write(terminalToString(qt))
}

// terminalToString converts a terminal to string
func terminalToString(qt *ast.QueryTerminal) string {
	return qt.String()
}

// joinStrings joins strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}

// insertClauseCount counts clauses in an insert expression
func insertClauseCount(ie *ast.InsertExpression) int {
	count := len(ie.Writes)
	if len(ie.UpsertKey) > 0 {
		count++
	}
	if ie.Terminal != nil {
		count++
	}
	return count
}

// formatInsertExpression formats @insert expressions
func (p *Printer) formatInsertExpression(ie *ast.InsertExpression) {
	// Check if we can do inline format
	inline := p.insertExpressionInline(ie)
	if len(inline) <= QueryInlineThreshold && insertClauseCount(ie) <= 3 {
		p.write(inline)
		return
	}

	// Multi-line format
	p.write("@insert(")
	p.indent++
	p.newline()
	p.writeIndent()

	// Source table
	p.write(ie.Source.Value)

	// Upsert key
	if len(ie.UpsertKey) > 0 {
		p.newline()
		p.writeIndent()
		p.write("| update on ")
		p.write(joinStrings(ie.UpsertKey, ", "))
	}

	// Batch operation
	if ie.Batch != nil {
		p.newline()
		p.writeIndent()
		p.write("* each ")
		p.formatExpression(ie.Batch.Collection)
		p.write(" -> ")
		p.write(ie.Batch.Alias.Value)
		if ie.Batch.IndexAlias != nil {
			p.write(", ")
			p.write(ie.Batch.IndexAlias.Value)
		}
	}

	// Field writes - one per line
	for _, w := range ie.Writes {
		p.newline()
		p.writeIndent()
		p.write("|< ")
		p.write(w.Field)
		p.write(": ")
		p.formatExpression(w.Value)
	}

	// Terminal
	if ie.Terminal != nil {
		p.newline()
		p.writeIndent()
		p.formatQueryTerminal(ie.Terminal)
	}

	p.indent--
	p.newline()
	p.writeIndent()
	p.write(")")
}

// insertExpressionInline returns a single-line representation of the insert
func (p *Printer) insertExpressionInline(ie *ast.InsertExpression) string {
	result := "@insert(" + ie.Source.Value

	if len(ie.UpsertKey) > 0 {
		result += " | update on " + joinStrings(ie.UpsertKey, ", ")
	}

	if ie.Batch != nil {
		result += " * each " + formatQueryExpressionValue(ie.Batch.Collection) + " -> " + ie.Batch.Alias.Value
		if ie.Batch.IndexAlias != nil {
			result += ", " + ie.Batch.IndexAlias.Value
		}
	}

	for _, w := range ie.Writes {
		result += " |< " + w.Field + ": " + formatQueryExpressionValue(w.Value)
	}

	if ie.Terminal != nil {
		result += " " + terminalToString(ie.Terminal)
	}
	result += ")"

	return result
}

// updateClauseCount counts clauses in an update expression
func updateClauseCount(ue *ast.UpdateExpression) int {
	count := len(ue.Conditions) + len(ue.Writes)
	if ue.Terminal != nil {
		count++
	}
	return count
}

// formatUpdateExpression formats @update expressions
func (p *Printer) formatUpdateExpression(ue *ast.UpdateExpression) {
	// Check if we can do inline format
	inline := p.updateExpressionInline(ue)
	if len(inline) <= QueryInlineThreshold && updateClauseCount(ue) <= 3 {
		p.write(inline)
		return
	}

	// Multi-line format
	p.write("@update(")
	p.indent++
	p.newline()
	p.writeIndent()

	// Source table
	p.write(ue.Source.Value)

	// Conditions - one per line
	for _, cond := range ue.Conditions {
		p.newline()
		p.writeIndent()
		p.write("| ")
		p.write(formatQueryConditionString(cond))
	}

	// Field writes - one per line
	for _, w := range ue.Writes {
		p.newline()
		p.writeIndent()
		p.write("|< ")
		p.write(w.Field)
		p.write(": ")
		p.formatExpression(w.Value)
	}

	// Terminal
	if ue.Terminal != nil {
		p.newline()
		p.writeIndent()
		p.formatQueryTerminal(ue.Terminal)
	}

	p.indent--
	p.newline()
	p.writeIndent()
	p.write(")")
}

// updateExpressionInline returns a single-line representation of the update
func (p *Printer) updateExpressionInline(ue *ast.UpdateExpression) string {
	result := "@update(" + ue.Source.Value

	for _, cond := range ue.Conditions {
		result += " | " + formatQueryConditionString(cond)
	}

	for _, w := range ue.Writes {
		result += " |< " + w.Field + ": " + formatQueryExpressionValue(w.Value)
	}

	if ue.Terminal != nil {
		result += " " + terminalToString(ue.Terminal)
	}
	result += ")"

	return result
}

// deleteClauseCount counts clauses in a delete expression
func deleteClauseCount(de *ast.DeleteExpression) int {
	count := len(de.Conditions)
	if de.Terminal != nil {
		count++
	}
	return count
}

// formatDeleteExpression formats @delete expressions
func (p *Printer) formatDeleteExpression(de *ast.DeleteExpression) {
	// Check if we can do inline format
	inline := p.deleteExpressionInline(de)
	if len(inline) <= QueryInlineThreshold && deleteClauseCount(de) <= 2 {
		p.write(inline)
		return
	}

	// Multi-line format
	p.write("@delete(")
	p.indent++
	p.newline()
	p.writeIndent()

	// Source table
	p.write(de.Source.Value)

	// Conditions - one per line
	for _, cond := range de.Conditions {
		p.newline()
		p.writeIndent()
		p.write("| ")
		p.write(formatQueryConditionString(cond))
	}

	// Terminal
	if de.Terminal != nil {
		p.newline()
		p.writeIndent()
		p.formatQueryTerminal(de.Terminal)
	}

	p.indent--
	p.newline()
	p.writeIndent()
	p.write(")")
}

// deleteExpressionInline returns a single-line representation of the delete
func (p *Printer) deleteExpressionInline(de *ast.DeleteExpression) string {
	result := "@delete(" + de.Source.Value

	for _, cond := range de.Conditions {
		result += " | " + formatQueryConditionString(cond)
	}

	if de.Terminal != nil {
		result += " " + terminalToString(de.Terminal)
	}
	result += ")"

	return result
}

// formatTransactionExpression formats @transaction { statements } expressions
func (p *Printer) formatTransactionExpression(te *ast.TransactionExpression) {
	p.write("@transaction {")
	p.indent++

	for _, stmt := range te.Statements {
		p.newline()
		p.writeIndent()
		p.formatStatement(stmt)
	}

	p.indent--
	p.newline()
	p.writeIndent()
	p.write("}")
}

// dedentText removes common leading whitespace from all lines in text.
// This is used to normalize CSS/script content inside tags.
// It finds the minimum indentation across all non-empty lines and removes it.
func dedentText(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return text
	}

	// Find minimum indentation (ignoring empty lines)
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := 0
		for _, ch := range line {
			if ch == ' ' {
				indent++
			} else if ch == '\t' {
				indent++ // Count tabs as 1 for comparison purposes
			} else {
				break
			}
		}
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent <= 0 {
		return text
	}

	// Remove the minimum indentation from each line
	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			result = append(result, "")
		} else {
			// Remove minIndent characters from start
			remaining := minIndent
			i := 0
			for i < len(line) && remaining > 0 {
				if line[i] == ' ' || line[i] == '\t' {
					remaining--
					i++
				} else {
					break
				}
			}
			result = append(result, line[i:])
		}
	}
	return strings.Join(result, "\n")
}

// reindentText takes dedented text and adds the specified indentation level.
// It adds one tab per indent level to the start of each non-empty line.
func reindentText(text string, indentLevel int) string {
	lines := strings.Split(text, "\n")
	indent := strings.Repeat(IndentString, indentLevel)

	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			result = append(result, "")
		} else {
			result = append(result, indent+line)
		}
	}
	return strings.Join(result, "\n")
}
