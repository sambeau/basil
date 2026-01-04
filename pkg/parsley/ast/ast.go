package ast

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// Inspectable is an interface for objects that can be inspected for display
// This is used to avoid circular imports between ast and evaluator packages
type Inspectable interface {
	Inspect() string
}

// Node represents any node in the AST
type Node interface {
	TokenLiteral() string
	String() string
}

// Statement represents statement nodes
type Statement interface {
	Node
	statementNode()
}

// Expression represents expression nodes
type Expression interface {
	Node
	expressionNode()
}

// Program represents the root node of every AST
type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) String() string {
	var out bytes.Buffer

	for _, s := range p.Statements {
		out.WriteString(s.String())
	}

	return out.String()
}

// LetStatement represents let statements like 'let x = 5;' or 'let [x,y,z] = [1,2,3];'
type LetStatement struct {
	Token        lexer.Token                // the lexer.LET token
	Name         *Identifier                // single identifier for simple let (mutually exclusive with patterns)
	ArrayPattern *ArrayDestructuringPattern // pattern for array destructuring
	DictPattern  *DictDestructuringPattern  // pattern for dictionary destructuring
	Value        Expression
	Export       bool // true if 'export' keyword was used
}

func (ls *LetStatement) statementNode()       {}
func (ls *LetStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *LetStatement) String() string {
	var out bytes.Buffer

	if ls.Export {
		out.WriteString("export ")
	}
	out.WriteString(ls.TokenLiteral() + " ")
	if ls.DictPattern != nil {
		out.WriteString(ls.DictPattern.String())
	} else if ls.ArrayPattern != nil {
		out.WriteString(ls.ArrayPattern.String())
	} else {
		out.WriteString(ls.Name.String())
	}
	out.WriteString(" = ")

	if ls.Value != nil {
		out.WriteString(ls.Value.String())
	}

	out.WriteString(";")
	return out.String()
}

// AssignmentStatement represents assignment statements like 'x = 5;' or '[x,y,z] = [1,2,3];'
type AssignmentStatement struct {
	Token        lexer.Token                // the identifier token
	Name         *Identifier                // single identifier for simple assignment (mutually exclusive with patterns)
	ArrayPattern *ArrayDestructuringPattern // pattern for array destructuring
	DictPattern  *DictDestructuringPattern  // pattern for dictionary destructuring
	Value        Expression
	Export       bool // true if 'export' keyword was used
}

func (as *AssignmentStatement) statementNode()       {}
func (as *AssignmentStatement) TokenLiteral() string { return as.Token.Literal }
func (as *AssignmentStatement) String() string {
	var out bytes.Buffer

	if as.Export {
		out.WriteString("export ")
	}
	if as.DictPattern != nil {
		out.WriteString(as.DictPattern.String())
	} else if as.ArrayPattern != nil {
		out.WriteString(as.ArrayPattern.String())
	} else {
		out.WriteString(as.Name.String())
	}
	out.WriteString(" = ")

	if as.Value != nil {
		out.WriteString(as.Value.String())
	}

	out.WriteString(";")
	return out.String()
}

// IndexAssignmentStatement represents assignment to index/property expressions like 'dict["key"] = value' or 'obj.prop = value'
type IndexAssignmentStatement struct {
	Token  lexer.Token // the '=' token
	Target Expression  // the IndexExpression or DotExpression being assigned to
	Value  Expression  // the value being assigned
}

func (ias *IndexAssignmentStatement) statementNode()       {}
func (ias *IndexAssignmentStatement) TokenLiteral() string { return ias.Token.Literal }
func (ias *IndexAssignmentStatement) String() string {
	var out bytes.Buffer

	out.WriteString(ias.Target.String())
	out.WriteString(" = ")
	if ias.Value != nil {
		out.WriteString(ias.Value.String())
	}
	out.WriteString(";")
	return out.String()
}

// ReturnStatement represents return statements like 'return 5;'
type ReturnStatement struct {
	Token       lexer.Token // the 'return' token
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) String() string {
	var out bytes.Buffer

	out.WriteString(rs.TokenLiteral() + " ")

	if rs.ReturnValue != nil {
		out.WriteString(rs.ReturnValue.String())
	}

	out.WriteString(";")
	return out.String()
}

// CheckStatement represents 'check CONDITION else VALUE'
type CheckStatement struct {
	Token     lexer.Token // the 'check' token
	Condition Expression  // the condition to check
	ElseValue Expression  // value to return if condition is false
}

func (cs *CheckStatement) statementNode()       {}
func (cs *CheckStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *CheckStatement) String() string {
	return "check " + cs.Condition.String() + " else " + cs.ElseValue.String()
}

// StopStatement represents 'stop' - exit for loop with accumulated results
// Implements both Statement and Expression so it can be used as: if (x > 5) stop
type StopStatement struct {
	Token lexer.Token // the 'stop' token
}

func (ss *StopStatement) statementNode()       {}
func (ss *StopStatement) expressionNode()      {}
func (ss *StopStatement) TokenLiteral() string { return ss.Token.Literal }
func (ss *StopStatement) String() string       { return "stop" }

// SkipStatement represents 'skip' - skip current iteration in for loop
// Implements both Statement and Expression so it can be used as: if (x == 0) skip
type SkipStatement struct {
	Token lexer.Token // the 'skip' token
}

func (sk *SkipStatement) statementNode()       {}
func (sk *SkipStatement) expressionNode()      {}
func (sk *SkipStatement) TokenLiteral() string { return sk.Token.Literal }
func (sk *SkipStatement) String() string       { return "skip" }

// ExpressionStatement represents expression statements
type ExpressionStatement struct {
	Token      lexer.Token // the first token of the expression
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

// BlockStatement represents block statements like '{...}'
type BlockStatement struct {
	Token      lexer.Token // the '{' token
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	var out bytes.Buffer

	for _, s := range bs.Statements {
		out.WriteString(s.String())
	}

	return out.String()
}

// Identifier represents identifier expressions
type Identifier struct {
	Token lexer.Token // the lexer.IDENT token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

// SpreadExpr represents spread expressions like ...attrs used in tag attributes
type SpreadExpr struct {
	Token      lexer.Token // the lexer.DOTDOTDOT token
	Expression Expression  // the identifier or expression to spread
}

func (se *SpreadExpr) expressionNode()      {}
func (se *SpreadExpr) TokenLiteral() string { return se.Token.Literal }
func (se *SpreadExpr) String() string {
	return "..." + se.Expression.String()
}

// IntegerLiteral represents integer literals
type IntegerLiteral struct {
	Token lexer.Token // the lexer.INT token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

// FloatLiteral represents floating-point literals
type FloatLiteral struct {
	Token lexer.Token // the lexer.FLOAT token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) String() string       { return fl.Token.Literal }

// StringLiteral represents string literals
type StringLiteral struct {
	Token lexer.Token // the lexer.STRING token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return sl.Token.Literal }

// TemplateLiteral represents template literals with interpolation
type TemplateLiteral struct {
	Token lexer.Token // the lexer.TEMPLATE token
	Value string      // the raw template string
}

func (tl *TemplateLiteral) expressionNode()      {}
func (tl *TemplateLiteral) TokenLiteral() string { return tl.Token.Literal }
func (tl *TemplateLiteral) String() string       { return "`" + tl.Value + "`" }

// RawTemplateLiteral represents single-quoted strings with @{} interpolation
type RawTemplateLiteral struct {
	Token lexer.Token // the lexer.RAW_TEMPLATE token
	Value string      // the raw template string with @{} markers
}

func (rtl *RawTemplateLiteral) expressionNode()      {}
func (rtl *RawTemplateLiteral) TokenLiteral() string { return rtl.Token.Literal }
func (rtl *RawTemplateLiteral) String() string       { return "'" + rtl.Value + "'" }

// RegexLiteral represents regular expression literals like /pattern/flags
type RegexLiteral struct {
	Token   lexer.Token // the lexer.REGEX token
	Pattern string      // the regex pattern
	Flags   string      // the regex flags
}

func (rl *RegexLiteral) expressionNode()      {}
func (rl *RegexLiteral) TokenLiteral() string { return rl.Token.Literal }
func (rl *RegexLiteral) String() string {
	return "/" + rl.Pattern + "/" + rl.Flags
}

// DatetimeLiteral represents datetime literals like @2024-12-25T14:30:00Z
// Kind indicates how the literal was specified: "datetime", "date", or "time"
type DatetimeLiteral struct {
	Token lexer.Token // the lexer.DATETIME_LITERAL token
	Value string      // the ISO-8601 datetime string
	Kind  string      // "datetime", "date", or "time"
}

func (dl *DatetimeLiteral) expressionNode()      {}
func (dl *DatetimeLiteral) TokenLiteral() string { return dl.Token.Literal }
func (dl *DatetimeLiteral) String() string       { return "@" + dl.Value }

// DatetimeNowLiteral represents current time/date/datetime literals like @now, @timeNow, @dateNow, @today
// Kind indicates which dictionary shape to produce: "datetime", "date", or "time".
type DatetimeNowLiteral struct {
	Token lexer.Token // the lexer.DATETIME_NOW, TIME_NOW, or DATE_NOW token
	Kind  string
}

func (dnl *DatetimeNowLiteral) expressionNode()      {}
func (dnl *DatetimeNowLiteral) TokenLiteral() string { return dnl.Token.Literal }
func (dnl *DatetimeNowLiteral) String() string       { return "@" + dnl.Token.Literal }

// DurationLiteral represents duration literals like @2h30m, @7d, @1y6mo
type DurationLiteral struct {
	Token lexer.Token // the lexer.DURATION_LITERAL token
	Value string      // the duration string (e.g., "2h30m", "7d")
}

func (dr *DurationLiteral) expressionNode()      {}
func (dr *DurationLiteral) TokenLiteral() string { return dr.Token.Literal }
func (dr *DurationLiteral) String() string       { return "@" + dr.Value }

// MoneyLiteral represents money literals like $12.34, £99.99, EUR#50.00
type MoneyLiteral struct {
	Token    lexer.Token // the lexer.MONEY token
	Currency string      // currency code: "USD", "GBP", "EUR", etc.
	Amount   int64       // amount in smallest unit (e.g., cents)
	Scale    int8        // decimal places (2 for USD, 0 for JPY)
}

func (ml *MoneyLiteral) expressionNode()      {}
func (ml *MoneyLiteral) TokenLiteral() string { return ml.Token.Literal }
func (ml *MoneyLiteral) String() string {
	// Format as CODE#amount with appropriate decimal places
	if ml.Scale == 0 {
		return ml.Currency + "#" + fmt.Sprintf("%d", ml.Amount)
	}
	divisor := int64(1)
	for i := int8(0); i < ml.Scale; i++ {
		divisor *= 10
	}
	whole := ml.Amount / divisor
	frac := ml.Amount % divisor
	if frac < 0 {
		frac = -frac
	}
	format := fmt.Sprintf("%%d.%%0%dd", ml.Scale)
	return ml.Currency + "#" + fmt.Sprintf(format, whole, frac)
}

// PathLiteral represents path literals like @/usr/local/bin or @./config.json
type PathLiteral struct {
	Token lexer.Token // the lexer.PATH_LITERAL token
	Value string      // the path string
}

func (pl *PathLiteral) expressionNode()      {}
func (pl *PathLiteral) TokenLiteral() string { return pl.Token.Literal }
func (pl *PathLiteral) String() string       { return "@" + pl.Value }

// UrlLiteral represents URL literals like @https://example.com/api
type UrlLiteral struct {
	Token lexer.Token // the lexer.URL_LITERAL token
	Value string      // the URL string
}

func (ul *UrlLiteral) expressionNode()      {}
func (ul *UrlLiteral) TokenLiteral() string { return ul.Token.Literal }
func (ul *UrlLiteral) String() string       { return "@" + ul.Value }

// ConnectionLiteral represents connection constructor literals like @sqlite, @postgres, @shell
type ConnectionLiteral struct {
	Token lexer.Token // the lexer.*_LITERAL token
	Kind  string      // "sqlite", "postgres", "mysql", "sftp", "shell", "db"
}

func (cl *ConnectionLiteral) expressionNode()      {}
func (cl *ConnectionLiteral) TokenLiteral() string { return cl.Token.Literal }
func (cl *ConnectionLiteral) String() string       { return "@" + cl.Kind }

// StdlibPathLiteral represents standard library imports like @std/table
type StdlibPathLiteral struct {
	Token lexer.Token // the lexer.STDLIB_PATH token
	Value string      // the stdlib path (e.g., "std/table")
}

func (sp *StdlibPathLiteral) expressionNode()      {}
func (sp *StdlibPathLiteral) TokenLiteral() string { return sp.Token.Literal }
func (sp *StdlibPathLiteral) String() string       { return "@" + sp.Value }

// PathTemplateLiteral represents interpolated path templates like @(./path/{expr}/file)
type PathTemplateLiteral struct {
	Token lexer.Token // the lexer.PATH_TEMPLATE token
	Value string      // the template content (e.g., "./path/{name}/file")
}

func (pt *PathTemplateLiteral) expressionNode()      {}
func (pt *PathTemplateLiteral) TokenLiteral() string { return pt.Token.Literal }
func (pt *PathTemplateLiteral) String() string       { return "@(" + pt.Value + ")" }

// UrlTemplateLiteral represents interpolated URL templates like @(https://api.com/{version}/users)
type UrlTemplateLiteral struct {
	Token lexer.Token // the lexer.URL_TEMPLATE token
	Value string      // the template content (e.g., "https://api.com/{v}/users")
}

func (ut *UrlTemplateLiteral) expressionNode()      {}
func (ut *UrlTemplateLiteral) TokenLiteral() string { return ut.Token.Literal }
func (ut *UrlTemplateLiteral) String() string       { return "@(" + ut.Value + ")" }

// DatetimeTemplateLiteral represents interpolated datetime templates like @(2024-{month}-{day})
type DatetimeTemplateLiteral struct {
	Token lexer.Token // the lexer.DATETIME_TEMPLATE token
	Value string      // the template content (e.g., "2024-{month}-{day}")
}

func (dt *DatetimeTemplateLiteral) expressionNode()      {}
func (dt *DatetimeTemplateLiteral) TokenLiteral() string { return dt.Token.Literal }
func (dt *DatetimeTemplateLiteral) String() string       { return "@(" + dt.Value + ")" }

// TagLiteral represents singleton tags like <input type="text" />
type TagLiteral struct {
	Token   lexer.Token   // the lexer.TAG token
	Raw     string        // the raw tag content (everything between < and />)
	Spreads []*SpreadExpr // spread expressions like ...attrs
}

func (tg *TagLiteral) expressionNode()      {}
func (tg *TagLiteral) TokenLiteral() string { return tg.Token.Literal }
func (tg *TagLiteral) String() string       { return "<" + tg.Raw + " />" }

// TagPairExpression represents paired tags like <div>content</div>
type TagPairExpression struct {
	Token    lexer.Token   // the lexer.TAG_START token
	Name     string        // tag name (empty string for grouping tags <>)
	Props    string        // raw props content
	Spreads  []*SpreadExpr // spread expressions like ...attrs
	Contents []Node        // mixed content: text nodes, expressions, nested tags
}

func (tp *TagPairExpression) expressionNode()      {}
func (tp *TagPairExpression) TokenLiteral() string { return tp.Token.Literal }
func (tp *TagPairExpression) String() string {
	var out bytes.Buffer
	if tp.Name == "" {
		out.WriteString("<>")
	} else {
		out.WriteString("<" + tp.Name)
		if tp.Props != "" {
			out.WriteString(" " + tp.Props)
		}
		out.WriteString(">")
	}
	for _, content := range tp.Contents {
		out.WriteString(content.String())
	}
	if tp.Name == "" {
		out.WriteString("</>")
	} else {
		out.WriteString("</" + tp.Name + ">")
	}
	return out.String()
}

// TextNode represents raw text content within tags
type TextNode struct {
	Token lexer.Token // the lexer.TAG_TEXT token
	Value string      // the text content
}

func (tn *TextNode) expressionNode()      {}
func (tn *TextNode) TokenLiteral() string { return tn.Token.Literal }
func (tn *TextNode) String() string       { return tn.Value }

// Boolean represents boolean literals
type Boolean struct {
	Token lexer.Token // the lexer.TRUE or lexer.FALSE token
	Value bool
}

func (b *Boolean) expressionNode()      {}
func (b *Boolean) TokenLiteral() string { return b.Token.Literal }
func (b *Boolean) String() string       { return b.Token.Literal }

// GroupedExpression represents a parenthesized expression like '(x + y)'
// This is used to allow calling the result of complex expressions: (if(cond){fn})(args)
type GroupedExpression struct {
	Token lexer.Token // the '(' token
	Inner Expression
}

func (ge *GroupedExpression) expressionNode()      {}
func (ge *GroupedExpression) TokenLiteral() string { return ge.Token.Literal }
func (ge *GroupedExpression) String() string {
	return "(" + ge.Inner.String() + ")"
}

// PrefixExpression represents prefix expressions like '!x' or '-x'
type PrefixExpression struct {
	Token    lexer.Token // the prefix token, e.g. !
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(pe.Operator)
	out.WriteString(pe.Right.String())
	out.WriteString(")")

	return out.String()
}

// InfixExpression represents infix expressions like 'x + y'
type InfixExpression struct {
	Token    lexer.Token // the operator token, e.g. +
	Left     Expression
	Operator string
	Right    Expression
}

func (oe *InfixExpression) expressionNode()      {}
func (oe *InfixExpression) TokenLiteral() string { return oe.Token.Literal }
func (oe *InfixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(oe.Left.String())
	out.WriteString(" " + oe.Operator + " ")
	out.WriteString(oe.Right.String())
	out.WriteString(")")

	return out.String()
}

// IfExpression represents if expressions
type IfExpression struct {
	Token       lexer.Token // the 'if' token
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (ie *IfExpression) expressionNode()      {}
func (ie *IfExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IfExpression) String() string {
	var out bytes.Buffer

	out.WriteString("if")
	out.WriteString(ie.Condition.String())
	out.WriteString(" ")
	out.WriteString(ie.Consequence.String())

	if ie.Alternative != nil {
		out.WriteString("else ")
		out.WriteString(ie.Alternative.String())
	}

	return out.String()
}

// FunctionLiteral represents function literals
// FunctionParameter represents a function parameter (identifier, array pattern, or dict pattern)
type FunctionParameter struct {
	Ident        *Identifier                // simple identifier parameter
	ArrayPattern *ArrayDestructuringPattern // array destructuring pattern
	DictPattern  *DictDestructuringPattern  // dict destructuring pattern
}

func (fp *FunctionParameter) String() string {
	if fp.DictPattern != nil {
		return fp.DictPattern.String()
	}
	if fp.ArrayPattern != nil {
		return fp.ArrayPattern.String()
	}
	return fp.Ident.String()
}

type FunctionLiteral struct {
	Token  lexer.Token          // the 'fn' token
	Params []*FunctionParameter // parameter list supporting destructuring
	Body   *BlockStatement
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FunctionLiteral) String() string {
	var out bytes.Buffer

	params := []string{}
	for _, p := range fl.Params {
		params = append(params, p.String())
	}

	out.WriteString(fl.TokenLiteral())
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") ")
	out.WriteString(fl.Body.String())

	return out.String()
}

// CallExpression represents function calls
type CallExpression struct {
	Token     lexer.Token // the '(' token
	Function  Expression  // Identifier or FunctionLiteral
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	var out bytes.Buffer

	args := []string{}
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}

	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")

	return out.String()
}

// ArrayLiteral represents array literals like [1, 2, 3]
type ArrayLiteral struct {
	Token    lexer.Token // the first element token
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) String() string {
	var out bytes.Buffer

	elements := []string{}
	for _, el := range al.Elements {
		elements = append(elements, el.String())
	}

	out.WriteString(strings.Join(elements, ", "))

	return out.String()
}

// ForExpression represents for expressions
// Two forms: for(array) func  OR  for(var in array) body
type ForExpression struct {
	Token         lexer.Token // the 'for' token
	Array         Expression  // the array to iterate over
	Function      Expression  // the function to apply (for simple form)
	Variable      *Identifier // the loop variable (for 'in' form) or key variable (for dict)
	ValueVariable *Identifier // the value variable (for dict 'key, value in dict' form)
	Body          Expression  // the body expression (for 'in' form)
}

func (fe *ForExpression) expressionNode()      {}
func (fe *ForExpression) TokenLiteral() string { return fe.Token.Literal }
func (fe *ForExpression) String() string {
	var out bytes.Buffer

	out.WriteString("for(")
	if fe.Variable != nil {
		out.WriteString(fe.Variable.String())
		out.WriteString(" in ")
	}
	out.WriteString(fe.Array.String())
	out.WriteString(")")

	if fe.Function != nil {
		out.WriteString(" ")
		out.WriteString(fe.Function.String())
	} else if fe.Body != nil {
		out.WriteString(" ")
		out.WriteString(fe.Body.String())
	}

	return out.String()
}

// IndexExpression represents array/string indexing like arr[0] or str[1]
type IndexExpression struct {
	Token    lexer.Token // the '[' token
	Left     Expression  // the array or string being indexed
	Index    Expression  // the index expression
	Optional bool        // true for [?n] optional indexing syntax
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString("[")
	if ie.Optional {
		out.WriteString("?")
	}
	out.WriteString(ie.Index.String())
	out.WriteString("])")

	return out.String()
}

// SliceExpression represents array/string slicing like arr[1:4]
type SliceExpression struct {
	Token lexer.Token // the '[' token
	Left  Expression  // the array or string being sliced
	Start Expression  // the start index (can be nil)
	End   Expression  // the end index (can be nil)
}

func (se *SliceExpression) expressionNode()      {}
func (se *SliceExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SliceExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(se.Left.String())
	out.WriteString("[")
	if se.Start != nil {
		out.WriteString(se.Start.String())
	}
	out.WriteString(":")
	if se.End != nil {
		out.WriteString(se.End.String())
	}
	out.WriteString("])")

	return out.String()
}

// DictionaryLiteral represents dictionary literals like { key: value, ... }
type DictionaryLiteral struct {
	Token    lexer.Token // the '{' token
	Pairs    map[string]Expression
	KeyOrder []string // Keys in source order
	// ComputedPairs holds key-value pairs where the key is computed at runtime
	// e.g., {[expr]: value}
	ComputedPairs []ComputedKeyValue
}

// ComputedKeyValue represents a key-value pair with a computed key expression
type ComputedKeyValue struct {
	Key   Expression
	Value Expression
}

func (dl *DictionaryLiteral) expressionNode()      {}
func (dl *DictionaryLiteral) TokenLiteral() string { return dl.Token.Literal }
func (dl *DictionaryLiteral) String() string {
	var out bytes.Buffer

	pairs := []string{}
	// Use KeyOrder if available for consistent output
	keys := dl.KeyOrder
	if len(keys) == 0 {
		for key := range dl.Pairs {
			keys = append(keys, key)
		}
	}
	for _, key := range keys {
		if value, ok := dl.Pairs[key]; ok {
			pairs = append(pairs, key+": "+value.String())
		}
	}
	// Add computed pairs
	for _, cp := range dl.ComputedPairs {
		pairs = append(pairs, "["+cp.Key.String()+"]: "+cp.Value.String())
	}

	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")

	return out.String()
}

// DotExpression represents dot notation access like dict.key
type DotExpression struct {
	Token lexer.Token // the '.' token
	Left  Expression  // the object being accessed
	Key   string      // the property name
}

func (de *DotExpression) expressionNode()      {}
func (de *DotExpression) TokenLiteral() string { return de.Token.Literal }
func (de *DotExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(de.Left.String())
	out.WriteString(".")
	out.WriteString(de.Key)
	out.WriteString(")")

	return out.String()
}

// ExecuteExpression represents command execution like: COMMAND("ls") <=#=> null
type ExecuteExpression struct {
	Token   lexer.Token // the '<=#=>' token
	Command Expression  // the command handle
	Input   Expression  // the input data (or null)
}

func (ee *ExecuteExpression) expressionNode()      {}
func (ee *ExecuteExpression) TokenLiteral() string { return ee.Token.Literal }
func (ee *ExecuteExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(ee.Command.String())
	out.WriteString(" <=#=> ")
	out.WriteString(ee.Input.String())
	out.WriteString(")")

	return out.String()
}

// ReadStatement represents read-from-file statements like 'let x <== file(...)' or '{a, b} <== file(...)'
type ReadStatement struct {
	Token        lexer.Token                // the <== token
	Name         *Identifier                // single name for let x <==
	ArrayPattern *ArrayDestructuringPattern // pattern for array destructuring
	DictPattern  *DictDestructuringPattern  // pattern for dictionary destructuring
	IsLet        bool                       // true if 'let' was used
	Source       Expression                 // the file handle expression
}

// ReadExpression represents a bare read expression like '<== file(...)' that returns the content
type ReadExpression struct {
	Token  lexer.Token // the <== token
	Source Expression  // the file handle expression
}

func (re *ReadExpression) expressionNode()      {}
func (re *ReadExpression) TokenLiteral() string { return re.Token.Literal }
func (re *ReadExpression) String() string {
	var out bytes.Buffer
	out.WriteString("<== ")
	if re.Source != nil {
		out.WriteString(re.Source.String())
	}
	return out.String()
}

func (rs *ReadStatement) statementNode()       {}
func (rs *ReadStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReadStatement) String() string {
	var out bytes.Buffer

	if rs.IsLet {
		out.WriteString("let ")
	}
	if rs.DictPattern != nil {
		out.WriteString(rs.DictPattern.String())
	} else if rs.ArrayPattern != nil {
		out.WriteString(rs.ArrayPattern.String())
	} else if rs.Name != nil {
		out.WriteString(rs.Name.String())
	}
	out.WriteString(" <== ")

	if rs.Source != nil {
		out.WriteString(rs.Source.String())
	}

	out.WriteString(";")
	return out.String()
}

// FetchStatement represents fetch-from-URL statements like 'let x <=/= jsonFile(@https://...)' or '{data, error} <=/= jsonFile(@url)'
type FetchStatement struct {
	Token        lexer.Token                // the <=/= token
	Name         *Identifier                // single name for let x <=/=
	ArrayPattern *ArrayDestructuringPattern // pattern for array destructuring
	DictPattern  *DictDestructuringPattern  // pattern for dictionary destructuring
	IsLet        bool                       // true if 'let' was used
	Source       Expression                 // the URL/request handle expression
}

func (fs *FetchStatement) statementNode()       {}
func (fs *FetchStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *FetchStatement) String() string {
	var out bytes.Buffer

	if fs.IsLet {
		out.WriteString("let ")
	}
	if fs.DictPattern != nil {
		out.WriteString(fs.DictPattern.String())
	} else if fs.ArrayPattern != nil {
		out.WriteString(fs.ArrayPattern.String())
	} else if fs.Name != nil {
		out.WriteString(fs.Name.String())
	}
	out.WriteString(" <=/= ")

	if fs.Source != nil {
		out.WriteString(fs.Source.String())
	}

	out.WriteString(";")
	return out.String()
}

// WriteStatement represents write-to-file statements like 'data ==> file(...)' or 'data ==>> file(...)'
type WriteStatement struct {
	Token  lexer.Token // the ==> or ==>> token
	Value  Expression  // the data to write
	Target Expression  // the file handle expression
	Append bool        // true for ==>> (append), false for ==> (write)
}

func (ws *WriteStatement) statementNode()       {}
func (ws *WriteStatement) TokenLiteral() string { return ws.Token.Literal }
func (ws *WriteStatement) String() string {
	var out bytes.Buffer

	if ws.Value != nil {
		out.WriteString(ws.Value.String())
	}
	if ws.Append {
		out.WriteString(" ==>> ")
	} else {
		out.WriteString(" ==> ")
	}
	if ws.Target != nil {
		out.WriteString(ws.Target.String())
	}

	out.WriteString(";")
	return out.String()
}

// QueryOneStatement represents query-one-row statements like 'let user = db <=?=> <GetUser id={1} />'
type QueryOneStatement struct {
	Token      lexer.Token   // the <=?=> token
	Names      []*Identifier // the variable names (can be identifiers or destructuring patterns)
	Connection Expression    // the database connection
	Query      Expression    // the query expression (component returning <SQL>)
	IsLet      bool          // true if this is a let statement
	Export     bool          // true if this should be exported
}

func (qos *QueryOneStatement) statementNode()       {}
func (qos *QueryOneStatement) TokenLiteral() string { return qos.Token.Literal }
func (qos *QueryOneStatement) String() string {
	var out bytes.Buffer

	if qos.IsLet {
		out.WriteString("let ")
	}

	for i, name := range qos.Names {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(name.String())
	}

	out.WriteString(" ")
	if qos.Connection != nil {
		out.WriteString(qos.Connection.String())
	}
	out.WriteString(" <=?=> ")
	if qos.Query != nil {
		out.WriteString(qos.Query.String())
	}

	out.WriteString(";")
	return out.String()
}

// QueryManyStatement represents query-many-rows statements like 'let users = db <=??=> <SearchUsers />'
type QueryManyStatement struct {
	Token      lexer.Token   // the <=??=> token
	Names      []*Identifier // the variable names (can be identifiers or destructuring patterns)
	Connection Expression    // the database connection
	Query      Expression    // the query expression (component returning <SQL>)
	IsLet      bool          // true if this is a let statement
	Export     bool          // true if this should be exported
}

func (qms *QueryManyStatement) statementNode()       {}
func (qms *QueryManyStatement) TokenLiteral() string { return qms.Token.Literal }
func (qms *QueryManyStatement) String() string {
	var out bytes.Buffer

	if qms.IsLet {
		out.WriteString("let ")
	}

	for i, name := range qms.Names {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(name.String())
	}

	out.WriteString(" ")
	if qms.Connection != nil {
		out.WriteString(qms.Connection.String())
	}
	out.WriteString(" <=??=> ")
	if qms.Query != nil {
		out.WriteString(qms.Query.String())
	}

	out.WriteString(";")
	return out.String()
}

// ExecuteStatement represents database mutation statements like 'let {affected} = db <=!=> <CreateUser />'
type ExecuteStatement struct {
	Token      lexer.Token   // the <=!=> token
	Names      []*Identifier // the variable names (can be identifiers or destructuring patterns)
	Connection Expression    // the database connection
	Query      Expression    // the query expression (component returning <SQL>)
	IsLet      bool          // true if this is a let statement
	Export     bool          // true if this should be exported
}

func (es *ExecuteStatement) statementNode()       {}
func (es *ExecuteStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExecuteStatement) String() string {
	var out bytes.Buffer

	if es.IsLet {
		out.WriteString("let ")
	}

	for i, name := range es.Names {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(name.String())
	}

	out.WriteString(" ")
	if es.Connection != nil {
		out.WriteString(es.Connection.String())
	}
	out.WriteString(" <=!=> ")
	if es.Query != nil {
		out.WriteString(es.Query.String())
	}

	out.WriteString(";")
	return out.String()
}

// DictDestructuringPattern represents a dictionary destructuring pattern like {a, b as c, ...rest}
type DictDestructuringPattern struct {
	Token lexer.Token             // the '{' token
	Keys  []*DictDestructuringKey // the keys to extract
	Rest  *Identifier             // optional rest identifier (for ...rest)
}

func (ddp *DictDestructuringPattern) expressionNode()      {}
func (ddp *DictDestructuringPattern) TokenLiteral() string { return ddp.Token.Literal }
func (ddp *DictDestructuringPattern) String() string {
	var out bytes.Buffer

	out.WriteString("{")
	for i, key := range ddp.Keys {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(key.String())
	}
	if ddp.Rest != nil {
		if len(ddp.Keys) > 0 {
			out.WriteString(", ")
		}
		out.WriteString("...")
		out.WriteString(ddp.Rest.String())
	}
	out.WriteString("}")

	return out.String()
}

// DictDestructuringKey represents a single key in a dictionary destructuring pattern
// Can be: 'a' or 'a as b' or nested pattern
type DictDestructuringKey struct {
	Token  lexer.Token // the identifier token
	Key    *Identifier // the key name from the dictionary
	Alias  *Identifier // optional alias (for 'as' syntax)
	Nested Expression  // optional nested pattern for nested destructuring
}

func (ddk *DictDestructuringKey) expressionNode()      {}
func (ddk *DictDestructuringKey) TokenLiteral() string { return ddk.Token.Literal }
func (ddk *DictDestructuringKey) String() string {
	var out bytes.Buffer

	if ddk.Nested != nil {
		out.WriteString(ddk.Key.String())
		out.WriteString(": ")
		out.WriteString(ddk.Nested.String())
	} else if ddk.Alias != nil {
		out.WriteString(ddk.Key.String())
		out.WriteString(" as ")
		out.WriteString(ddk.Alias.String())
	} else {
		out.WriteString(ddk.Key.String())
	}

	return out.String()
}

// ArrayDestructuringPattern represents an array destructuring pattern like [a, b, ...rest]
type ArrayDestructuringPattern struct {
	Token lexer.Token   // the '[' token
	Names []*Identifier // the identifiers to extract
	Rest  *Identifier   // optional rest identifier (for ...rest)
}

func (adp *ArrayDestructuringPattern) expressionNode()      {}
func (adp *ArrayDestructuringPattern) TokenLiteral() string { return adp.Token.Literal }
func (adp *ArrayDestructuringPattern) String() string {
	var out bytes.Buffer

	out.WriteString("[")
	for i, name := range adp.Names {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(name.String())
	}
	if adp.Rest != nil {
		if len(adp.Names) > 0 {
			out.WriteString(", ")
		}
		out.WriteString("...")
		out.WriteString(adp.Rest.String())
	}
	out.WriteString("]")

	return out.String()
}

// ObjectLiteralExpression wraps an evaluator Object as an AST expression
// This is used internally by the module system to convert environment values to expressions
type ObjectLiteralExpression struct {
	Obj interface{} // stores evaluator.Object, but we use interface{} to avoid circular import
}

func (ole *ObjectLiteralExpression) expressionNode()      {}
func (ole *ObjectLiteralExpression) TokenLiteral() string { return "" }
func (ole *ObjectLiteralExpression) String() string {
	// Try to get a displayable representation via Inspectable interface
	if inspectable, ok := ole.Obj.(Inspectable); ok {
		return inspectable.Inspect()
	}
	return "<object literal>"
}

// InterpolationBlock represents a block of statements inside tag interpolation {stmt; stmt; ...}
// When evaluated, it returns the value of the last statement (or null if empty)
type InterpolationBlock struct {
	Token      lexer.Token // the '{' token
	Statements []Statement
}

func (ib *InterpolationBlock) expressionNode()      {}
func (ib *InterpolationBlock) TokenLiteral() string { return ib.Token.Literal }
func (ib *InterpolationBlock) String() string {
	var out bytes.Buffer
	out.WriteString("{")
	for i, s := range ib.Statements {
		if i > 0 {
			out.WriteString("; ")
		}
		out.WriteString(s.String())
	}
	out.WriteString("}")
	return out.String()
}

// TryExpression represents a try expression that catches "user errors"
// and returns a {result, error} dictionary instead of halting.
type TryExpression struct {
	Token lexer.Token // The 'try' token
	Call  Expression  // Must be a CallExpression (function or method call)
}

func (te *TryExpression) expressionNode()      {}
func (te *TryExpression) TokenLiteral() string { return te.Token.Literal }
func (te *TryExpression) String() string {
	var out bytes.Buffer
	out.WriteString("try ")
	out.WriteString(te.Call.String())
	return out.String()
}

// ImportExpression represents an import expression: import @path or import @path as Alias
// When used as a statement, it auto-binds to the last path segment (or alias if provided).
// When used in an assignment like {x} = import @path, it returns the module for destructuring.
type ImportExpression struct {
	Token    lexer.Token // The 'import' token
	Path     Expression  // The path expression (StdlibPathLiteral, PathLiteral, PathTemplateLiteral, etc.)
	Alias    *Identifier // Optional alias from "as Alias"
	BindName string      // Name to bind to environment (computed from path or alias)
}

func (ie *ImportExpression) expressionNode()      {}
func (ie *ImportExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *ImportExpression) String() string {
	var out bytes.Buffer
	out.WriteString("import @")
	if ie.Path != nil {
		// Strip leading @ from path string if present
		pathStr := ie.Path.String()
		if len(pathStr) > 0 && pathStr[0] == '@' {
			pathStr = pathStr[1:]
		}
		out.WriteString(pathStr)
	}
	if ie.Alias != nil {
		out.WriteString(" as ")
		out.WriteString(ie.Alias.Value)
	}
	return out.String()
}

// SchemaDeclaration represents @schema Name { fields } declarations
type SchemaDeclaration struct {
	Token  lexer.Token    // the SCHEMA_LITERAL token
	Name   *Identifier    // schema name
	Fields []*SchemaField // field definitions
}

func (sd *SchemaDeclaration) expressionNode()      {}
func (sd *SchemaDeclaration) TokenLiteral() string { return sd.Token.Literal }
func (sd *SchemaDeclaration) String() string {
	var out bytes.Buffer
	out.WriteString("@schema ")
	out.WriteString(sd.Name.Value)
	out.WriteString(" { ")
	for i, f := range sd.Fields {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(f.String())
	}
	out.WriteString(" }")
	return out.String()
}

// SchemaField represents a field definition within a schema
type SchemaField struct {
	Token      lexer.Token // the field name token
	Name       *Identifier // field name
	TypeName   string      // type name: "int", "string", "User", etc.
	IsArray    bool        // true for [Type] (has-many relation)
	ForeignKey string      // from "via fk_name", empty if no relation
}

func (sf *SchemaField) expressionNode()      {}
func (sf *SchemaField) TokenLiteral() string { return sf.Token.Literal }
func (sf *SchemaField) String() string {
	var out bytes.Buffer
	out.WriteString(sf.Name.Value)
	out.WriteString(": ")
	if sf.IsArray {
		out.WriteString("[")
	}
	out.WriteString(sf.TypeName)
	if sf.IsArray {
		out.WriteString("]")
	}
	if sf.ForeignKey != "" {
		out.WriteString(" via ")
		out.WriteString(sf.ForeignKey)
	}
	return out.String()
}

// QueryExpression represents @query(source | conditions ??-> projection) expressions
type QueryExpression struct {
	Token          lexer.Token           // the QUERY_LITERAL token
	Source         *Identifier           // binding/table name
	SourceAlias    *Identifier           // optional alias from "as alias"
	Conditions     []QueryConditionNode  // WHERE conditions (can be QueryCondition or QueryConditionGroup)
	Modifiers      []*QueryModifier      // ORDER BY, LIMIT, WITH, etc.
	GroupBy        []string              // GROUP BY fields
	ComputedFields []*QueryComputedField // computed aggregations like "total: sum(amount)"
	Terminal       *QueryTerminal        // return type and projection
}

func (qe *QueryExpression) expressionNode()      {}
func (qe *QueryExpression) TokenLiteral() string { return qe.Token.Literal }
func (qe *QueryExpression) String() string {
	var out bytes.Buffer
	out.WriteString("@query(")
	out.WriteString(qe.Source.Value)
	if qe.SourceAlias != nil {
		out.WriteString(" as ")
		out.WriteString(qe.SourceAlias.Value)
	}
	for _, c := range qe.Conditions {
		out.WriteString(" | ")
		out.WriteString(c.ConditionString())
	}
	if len(qe.GroupBy) > 0 {
		out.WriteString(" + by ")
		out.WriteString(strings.Join(qe.GroupBy, ", "))
	}
	for _, cf := range qe.ComputedFields {
		out.WriteString(" | ")
		out.WriteString(cf.String())
	}
	for _, m := range qe.Modifiers {
		out.WriteString(" | ")
		out.WriteString(m.String())
	}
	if qe.Terminal != nil {
		out.WriteString(" ")
		out.WriteString(qe.Terminal.String())
	}
	out.WriteString(")")
	return out.String()
}

// QueryCondition represents a condition in a query (WHERE clause part)
type QueryCondition struct {
	Token    lexer.Token // first token of condition
	Left     Expression  // column/field reference
	Operator string      // "==", "!=", ">", "<", ">=", "<=", "in", "not in", "like", "is null", "is not null", "between"
	Right    Expression  // value/interpolation (nil for "is null"/"is not null")
	RightEnd Expression  // end value for "between X and Y" operator
	Logic    string      // "and", "or" for combining with previous condition
	Negated  bool        // true if prefixed with "not"
}

func (qc *QueryCondition) expressionNode()      {}
func (qc *QueryCondition) TokenLiteral() string { return qc.Token.Literal }
func (qc *QueryCondition) String() string {
	var out bytes.Buffer
	if qc.Logic != "" {
		out.WriteString(qc.Logic)
		out.WriteString(" ")
	}
	if qc.Negated {
		out.WriteString("not ")
	}
	out.WriteString(qc.Left.String())
	out.WriteString(" ")
	out.WriteString(qc.Operator)
	if qc.Right != nil {
		out.WriteString(" ")
		out.WriteString(qc.Right.String())
	}
	if qc.RightEnd != nil {
		out.WriteString(" and ")
		out.WriteString(qc.RightEnd.String())
	}
	return out.String()
}

// QueryConditionGroup represents a group of conditions wrapped in parentheses
// Supports: | (a == 1 or b == 2) and c == 3
type QueryConditionGroup struct {
	Token      lexer.Token          // the opening '(' token
	Conditions []QueryConditionNode // conditions or nested groups in this group
	Logic      string               // "and", "or" for combining with previous condition/group
	Negated    bool                 // true if prefixed with "not"
}

func (qcg *QueryConditionGroup) expressionNode()      {}
func (qcg *QueryConditionGroup) TokenLiteral() string { return qcg.Token.Literal }
func (qcg *QueryConditionGroup) String() string {
	var out bytes.Buffer
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
		out.WriteString(cond.ConditionString())
	}
	out.WriteString(")")
	return out.String()
}

// ConditionString returns the string representation for condition nodes
func (qcg *QueryConditionGroup) ConditionString() string {
	return qcg.String()
}

// QueryConditionNode is an interface for both QueryCondition and QueryConditionGroup
type QueryConditionNode interface {
	Expression
	ConditionString() string
}

// ConditionString returns the string representation for QueryCondition
func (qc *QueryCondition) ConditionString() string {
	return qc.String()
}

// QueryModifier represents ORDER BY, LIMIT, OFFSET, WITH, or GROUP BY clauses
type QueryModifier struct {
	Token         lexer.Token     // keyword token (order, limit, with)
	Kind          string          // "order", "limit", "offset", "with", "group"
	Fields        []string        // field names (for order, with, group)
	RelationPaths []*RelationPath // relation paths with conditions (for 'with')
	Direction     string          // "asc" or "desc" for order (optional)
	Value         int64           // numeric value for limit/offset
}

func (qm *QueryModifier) expressionNode()      {}
func (qm *QueryModifier) TokenLiteral() string { return qm.Token.Literal }
func (qm *QueryModifier) String() string {
	var out bytes.Buffer
	out.WriteString(qm.Kind)
	switch qm.Kind {
	case "order":
		out.WriteString(" ")
		out.WriteString(strings.Join(qm.Fields, ", "))
		if qm.Direction != "" {
			out.WriteString(" ")
			out.WriteString(qm.Direction)
		}
	case "limit", "offset":
		out.WriteString(" ")
		out.WriteString(fmt.Sprintf("%d", qm.Value))
	case "with":
		out.WriteString(" ")
		if len(qm.RelationPaths) > 0 {
			var parts []string
			for _, rp := range qm.RelationPaths {
				parts = append(parts, rp.String())
			}
			out.WriteString(strings.Join(parts, ", "))
		} else {
			out.WriteString(strings.Join(qm.Fields, ", "))
		}
	case "group":
		out.WriteString(" ")
		out.WriteString(strings.Join(qm.Fields, ", "))
	}
	return out.String()
}

// RelationPath represents a relation path with optional conditions, order, and limit
// Used in 'with' clauses like: | with comments.author(approved == true | order created_at desc | limit 5)
type RelationPath struct {
	Path       string               // dot-separated path like "comments.author"
	Conditions []QueryConditionNode // filter conditions (optional)
	Order      []QueryOrderField    // order by fields (optional)
	Limit      *int64               // limit value (optional)
}

// QueryOrderField represents a single field in an ORDER BY clause
type QueryOrderField struct {
	Field     string // field name
	Direction string // "asc" or "desc"
}

func (rp *RelationPath) String() string {
	var out bytes.Buffer
	out.WriteString(rp.Path)
	if len(rp.Conditions) > 0 || len(rp.Order) > 0 || rp.Limit != nil {
		out.WriteString("(")
		var parts []string
		for _, cond := range rp.Conditions {
			parts = append(parts, cond.ConditionString())
		}
		if len(rp.Order) > 0 {
			var orderParts []string
			for _, o := range rp.Order {
				orderStr := o.Field
				if o.Direction != "" {
					orderStr += " " + o.Direction
				}
				orderParts = append(orderParts, orderStr)
			}
			parts = append(parts, "order "+strings.Join(orderParts, ", "))
		}
		if rp.Limit != nil {
			parts = append(parts, fmt.Sprintf("limit %d", *rp.Limit))
		}
		out.WriteString(strings.Join(parts, " | "))
		out.WriteString(")")
	}
	return out.String()
}

// QueryComputedField represents a computed/aggregate field like "total: sum(amount)"
type QueryComputedField struct {
	Token    lexer.Token // the identifier token
	Name     string      // alias name (e.g., "total")
	Function string      // aggregate function ("count", "sum", "avg", "min", "max") or empty for plain field
	Field    string      // field to aggregate (e.g., "amount") - empty for count without field
}

func (qc *QueryComputedField) expressionNode()      {}
func (qc *QueryComputedField) TokenLiteral() string { return qc.Token.Literal }
func (qc *QueryComputedField) String() string {
	var out bytes.Buffer
	out.WriteString(qc.Name)
	out.WriteString(": ")
	if qc.Function != "" {
		out.WriteString(qc.Function)
		if qc.Field != "" {
			out.WriteString("(")
			out.WriteString(qc.Field)
			out.WriteString(")")
		}
	} else {
		out.WriteString(qc.Field)
	}
	return out.String()
}

// QuerySubquery represents a subquery in a condition (e.g., "author_id in <-Users | | role == 'admin' | | ?-> id")
type QuerySubquery struct {
	Token       lexer.Token          // the <- token
	Source      *Identifier          // table name
	SourceAlias *Identifier          // optional alias from "as alias" (for correlated subqueries)
	Conditions  []QueryConditionNode // WHERE conditions (prefixed with | | in syntax)
	Modifiers   []*QueryModifier     // ORDER BY, LIMIT, etc.
	Terminal    *QueryTerminal       // ?-> for single column (IN clause)
}

func (qs *QuerySubquery) expressionNode()      {}
func (qs *QuerySubquery) TokenLiteral() string { return qs.Token.Literal }
func (qs *QuerySubquery) String() string {
	var out bytes.Buffer
	out.WriteString("<-")
	out.WriteString(qs.Source.Value)
	if qs.SourceAlias != nil {
		out.WriteString(" as ")
		out.WriteString(qs.SourceAlias.Value)
	}
	for _, c := range qs.Conditions {
		out.WriteString(" | | ")
		out.WriteString(c.ConditionString())
	}
	for _, m := range qs.Modifiers {
		out.WriteString(" | | ")
		out.WriteString(m.String())
	}
	if qs.Terminal != nil {
		out.WriteString(" ")
		out.WriteString(qs.Terminal.String())
	}
	return out.String()
}

// QueryTerminal represents the return type and projection of a query
type QueryTerminal struct {
	Token      lexer.Token // ?-> or ??-> or . token
	Type       string      // "one", "many", "execute", "count", "exists"
	Projection []string    // field names, or ["*"] for all
}

func (qt *QueryTerminal) expressionNode()      {}
func (qt *QueryTerminal) TokenLiteral() string { return qt.Token.Literal }
func (qt *QueryTerminal) String() string {
	var out bytes.Buffer
	switch qt.Type {
	case "one":
		out.WriteString("?->")
	case "many":
		out.WriteString("??->")
	case "execute":
		out.WriteString(".")
	case "count":
		out.WriteString(".->")
	}
	if len(qt.Projection) > 0 {
		out.WriteString(" ")
		out.WriteString(strings.Join(qt.Projection, ", "))
	}
	return out.String()
}

// InsertExpression represents @insert(binding |< field: value ?-> *) expressions
type InsertExpression struct {
	Token     lexer.Token         // the INSERT_LITERAL token
	Source    *Identifier         // binding/table name
	UpsertKey []string            // fields for ON CONFLICT (from "| update on key")
	Writes    []*InsertFieldWrite // field assignments
	Batch     *InsertBatch        // batch operation (optional)
	Terminal  *QueryTerminal      // return type and projection
}

func (ie *InsertExpression) expressionNode()      {}
func (ie *InsertExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InsertExpression) String() string {
	var out bytes.Buffer
	out.WriteString("@insert(")
	out.WriteString(ie.Source.Value)
	if len(ie.UpsertKey) > 0 {
		out.WriteString(" | update on ")
		out.WriteString(strings.Join(ie.UpsertKey, ", "))
	}
	for _, w := range ie.Writes {
		out.WriteString(" |< ")
		out.WriteString(w.String())
	}
	if ie.Terminal != nil {
		out.WriteString(" ")
		out.WriteString(ie.Terminal.String())
	}
	out.WriteString(")")
	return out.String()
}

// InsertFieldWrite represents a field assignment in an insert: field: value
type InsertFieldWrite struct {
	Token lexer.Token // field name token
	Field string      // field/column name
	Value Expression  // value expression
}

func (iw *InsertFieldWrite) expressionNode()      {}
func (iw *InsertFieldWrite) TokenLiteral() string { return iw.Token.Literal }
func (iw *InsertFieldWrite) String() string {
	return iw.Field + ": " + iw.Value.String()
}

// InsertBatch represents batch insert: * each {collection} -> alias
type InsertBatch struct {
	Token      lexer.Token // the * token
	Collection Expression  // the collection to iterate
	Alias      *Identifier // loop variable name
	IndexAlias *Identifier // optional index variable
}

func (ib *InsertBatch) expressionNode()      {}
func (ib *InsertBatch) TokenLiteral() string { return ib.Token.Literal }
func (ib *InsertBatch) String() string {
	var out bytes.Buffer
	out.WriteString("* each ")
	out.WriteString(ib.Collection.String())
	out.WriteString(" -> ")
	out.WriteString(ib.Alias.Value)
	if ib.IndexAlias != nil {
		out.WriteString(", ")
		out.WriteString(ib.IndexAlias.Value)
	}
	return out.String()
}

// UpdateExpression represents @update(binding | conditions |< field: value .-> count) expressions
type UpdateExpression struct {
	Token      lexer.Token         // the UPDATE_LITERAL token
	Source     *Identifier         // binding/table name
	Conditions []*QueryCondition   // WHERE conditions
	Writes     []*InsertFieldWrite // field assignments
	Terminal   *QueryTerminal      // return type
}

func (ue *UpdateExpression) expressionNode()      {}
func (ue *UpdateExpression) TokenLiteral() string { return ue.Token.Literal }
func (ue *UpdateExpression) String() string {
	var out bytes.Buffer
	out.WriteString("@update(")
	out.WriteString(ue.Source.Value)
	for _, c := range ue.Conditions {
		out.WriteString(" | ")
		out.WriteString(c.String())
	}
	for _, w := range ue.Writes {
		out.WriteString(" |< ")
		out.WriteString(w.String())
	}
	if ue.Terminal != nil {
		out.WriteString(" ")
		out.WriteString(ue.Terminal.String())
	}
	out.WriteString(")")
	return out.String()
}

// DeleteExpression represents @delete(binding | conditions .) expressions
type DeleteExpression struct {
	Token      lexer.Token       // the DELETE_LITERAL token
	Source     *Identifier       // binding/table name
	Conditions []*QueryCondition // WHERE conditions
	Terminal   *QueryTerminal    // return type
}

func (de *DeleteExpression) expressionNode()      {}
func (de *DeleteExpression) TokenLiteral() string { return de.Token.Literal }
func (de *DeleteExpression) String() string {
	var out bytes.Buffer
	out.WriteString("@delete(")
	out.WriteString(de.Source.Value)
	for _, c := range de.Conditions {
		out.WriteString(" | ")
		out.WriteString(c.String())
	}
	if de.Terminal != nil {
		out.WriteString(" ")
		out.WriteString(de.Terminal.String())
	}
	out.WriteString(")")
	return out.String()
}

// TransactionExpression represents @transaction { statements } expressions
type TransactionExpression struct {
	Token      lexer.Token // the TRANSACTION_LIT token
	Statements []Statement // statements to execute in transaction
}

func (te *TransactionExpression) expressionNode()      {}
func (te *TransactionExpression) TokenLiteral() string { return te.Token.Literal }
func (te *TransactionExpression) String() string {
	var out bytes.Buffer
	out.WriteString("@transaction { ")
	for i, s := range te.Statements {
		if i > 0 {
			out.WriteString("; ")
		}
		out.WriteString(s.String())
	}
	out.WriteString(" }")
	return out.String()
}
