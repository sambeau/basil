// Package errors provides structured error types for the Parsley language.
//
// This package defines ParsleyError, a unified error type that can represent
// both parser and runtime errors with rich metadata for display, localization,
// and programmatic handling.
package errors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/template"
)

// ErrorClass categorizes errors for filtering and templating.
type ErrorClass string

const (
	ClassParse     ErrorClass = "parse"     // Parser/syntax errors
	ClassType      ErrorClass = "type"      // Type mismatches
	ClassArity     ErrorClass = "arity"     // Wrong argument count
	ClassUndefined ErrorClass = "undefined" // Not found/defined
	ClassIO        ErrorClass = "io"        // File operations
	ClassDatabase  ErrorClass = "database"  // DB operations
	ClassNetwork   ErrorClass = "network"   // HTTP, SSH, SFTP
	ClassSecurity  ErrorClass = "security"  // Access denied
	ClassIndex     ErrorClass = "index"     // Out of bounds
	ClassFormat    ErrorClass = "format"    // Invalid format/parse
	ClassOperator  ErrorClass = "operator"  // Invalid operations
	ClassState     ErrorClass = "state"     // Invalid state
	ClassImport    ErrorClass = "import"    // Module loading
	ClassValue     ErrorClass = "value"     // Invalid value (e.g., negative, empty)
)

// ParsleyError represents any error from parsing or evaluation.
type ParsleyError struct {
	Class   ErrorClass     `json:"class"`           // Error category
	Code    string         `json:"code"`            // Error code (e.g., "TYPE-0001")
	Message string         `json:"message"`         // Human-readable message
	Hints   []string       `json:"hints,omitempty"` // Suggestions for fixing
	Line    int            `json:"line"`            // 1-based line (0 if unknown)
	Column  int            `json:"column"`          // 1-based column (0 if unknown)
	File    string         `json:"file,omitempty"`  // File path (if known)
	Data    map[string]any `json:"data,omitempty"`  // Template variables
}

// Error implements the error interface.
func (e *ParsleyError) Error() string {
	return e.String()
}

// String returns a formatted string representation of the error.
func (e *ParsleyError) String() string {
	var sb strings.Builder

	// Location prefix
	if e.File != "" {
		sb.WriteString(e.File)
		sb.WriteString(": ")
	}
	if e.Line > 0 {
		sb.WriteString(fmt.Sprintf("line %d, column %d: ", e.Line, e.Column))
	}

	// Message
	sb.WriteString(e.Message)

	// Hints
	for _, hint := range e.Hints {
		sb.WriteString("\n  ")
		sb.WriteString(hint)
	}

	return sb.String()
}

// PrettyString returns a multi-line formatted string for display.
func (e *ParsleyError) PrettyString() string {
	var sb strings.Builder

	// Error type header
	switch e.Class {
	case ClassParse:
		sb.WriteString("Parser error")
	default:
		sb.WriteString("Runtime error")
	}

	// Location
	if e.File != "" {
		sb.WriteString(":\n  in: ")
		sb.WriteString(e.File)
		if e.Line > 0 {
			sb.WriteString(fmt.Sprintf("\n  at: line %d, column %d", e.Line, e.Column))
		}
		sb.WriteString("\n  ")
	} else if e.Line > 0 {
		sb.WriteString(fmt.Sprintf(": line %d, column %d\n  ", e.Line, e.Column))
	} else {
		sb.WriteString(":\n  ")
	}

	// Message
	sb.WriteString(e.Message)

	// Hints
	for i, hint := range e.Hints {
		sb.WriteString("\n  ")
		if i == 0 {
			sb.WriteString("Use: ")
		} else {
			sb.WriteString(" or: ")
		}
		sb.WriteString(hint)
	}

	return sb.String()
}

// ToJSON returns the error as JSON bytes.
func (e *ParsleyError) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// ToJSONIndent returns the error as indented JSON bytes.
func (e *ParsleyError) ToJSONIndent() ([]byte, error) {
	return json.MarshalIndent(e, "", "  ")
}

// WithFile returns a copy of the error with the file path set.
func (e *ParsleyError) WithFile(file string) *ParsleyError {
	copy := *e
	copy.File = file
	return &copy
}

// WithPosition returns a copy of the error with line and column set.
func (e *ParsleyError) WithPosition(line, column int) *ParsleyError {
	copy := *e
	copy.Line = line
	copy.Column = column
	return &copy
}

// IsParseError returns true if this is a parser error.
func (e *ParsleyError) IsParseError() bool {
	return e.Class == ClassParse
}

// IsRuntimeError returns true if this is a runtime error.
func (e *ParsleyError) IsRuntimeError() bool {
	return e.Class != ClassParse
}

// ErrorDef defines an error in the catalog.
type ErrorDef struct {
	Class    ErrorClass // Error category
	Template string     // Message template with {{.placeholders}}
	Hints    []string   // Hint templates (may use {{.placeholders}})
}

// ErrorCatalog maps error codes to their definitions.
var ErrorCatalog = map[string]ErrorDef{
	// ========================================
	// Parse errors (PARSE-0xxx)
	// ========================================
	"PARSE-0001": {
		Class:    ClassParse,
		Template: "expected {{.Expected}}, got '{{.Got}}'",
	},
	"PARSE-0002": {
		Class:    ClassParse,
		Template: "unexpected '{{.Token}}'",
	},
	"PARSE-0003": {
		Class:    ClassParse,
		Template: "`for ({{.Var}} in {{.Array}})` is ambiguous without { ... }",
		Hints:    []string{"for {{.Var}} in {{.Array}} { ... }", "for ({{.Array}}) fn({{.Var}}) { ... }"},
	},
	"PARSE-0004": {
		Class:    ClassParse,
		Template: "`for {{.Array}} {{.Expr}}` is ambiguous without ()",
		Hints:    []string{"for ({{.Array}}) fn", "for x in {{.Array}} { ... }"},
	},
	"PARSE-0005": {
		Class:    ClassParse,
		Template: "invalid regex literal: {{.Literal}}",
	},
	"PARSE-0006": {
		Class:    ClassParse,
		Template: "unterminated string",
	},
	"PARSE-0007": {
		Class:    ClassParse,
		Template: "invalid number literal: {{.Literal}}",
	},
	"PARSE-0008": {
		Class:    ClassParse,
		Template: "singleton tag must be self-closing",
		Hints:    []string{"<{{.Tag}}/>"},
	},
	"PARSE-0009": {
		Class:    ClassParse,
		Template: "unclosed { in {{.Context}}",
	},
	"PARSE-0010": {
		Class:    ClassParse,
		Template: "empty interpolation {} in {{.Context}}",
	},
	"PARSE-0011": {
		Class:    ClassParse,
		Template: "error parsing {{.Context}} expression: {{.GoError}}",
	},

	// ========================================
	// Type errors (TYPE-0xxx)
	// ========================================
	"TYPE-0001": {
		Class:    ClassType,
		Template: "{{.Function}} expected {{.Expected}}, got {{.Got}}",
	},
	"TYPE-0002": {
		Class:    ClassType,
		Template: "argument to `{{.Function}}` not supported, got {{.Got}}",
	},
	"TYPE-0003": {
		Class:    ClassType,
		Template: "cannot call {{.Got}} as a function",
	},
	"TYPE-0004": {
		Class:    ClassType,
		Template: "`for ({{.Array}}) {{.Got}}` is ambiguous without { }",
		Hints:    []string{"for _ in {{.Array}} { {{.Got}} }", "for ({{.Array}}) { print {{.Got}} }"},
	},
	"TYPE-0005": {
		Class:    ClassType,
		Template: "first argument to `{{.Function}}` must be {{.Expected}}, got {{.Got}}",
	},
	"TYPE-0006": {
		Class:    ClassType,
		Template: "second argument to `{{.Function}}` must be {{.Expected}}, got {{.Got}}",
	},
	"TYPE-0007": {
		Class:    ClassType,
		Template: "cannot iterate over {{.Got}}",
		Hints:    []string{"for works with arrays, strings, and ranges"},
	},
	"TYPE-0008": {
		Class:    ClassType,
		Template: "cannot index {{.Got}} with {{.IndexType}}",
	},
	"TYPE-0009": {
		Class:    ClassType,
		Template: "comparison function must return boolean, got {{.Got}}",
	},
	"TYPE-0010": {
		Class:    ClassType,
		Template: "{{.Function}} callback must be a function, got {{.Got}}",
	},
	"TYPE-0011": {
		Class:    ClassType,
		Template: "third argument to `{{.Function}}` must be {{.Expected}}, got {{.Got}}",
	},
	"TYPE-0012": {
		Class:    ClassType,
		Template: "argument to `{{.Function}}` must be {{.Expected}}, got {{.Got}}",
	},
	"TYPE-0013": {
		Class:    ClassType,
		Template: "index operator not supported: {{.Left}}[{{.Right}}]",
		Hints:    []string{"Arrays and strings can be indexed with integers", "Dictionaries can be indexed with strings"},
	},
	"TYPE-0014": {
		Class:    ClassType,
		Template: "slice operator not supported: {{.Type}}",
		Hints:    []string{"Slicing works with arrays and strings"},
	},
	"TYPE-0015": {
		Class:    ClassType,
		Template: "cannot convert '{{.Value}}' to integer",
	},
	"TYPE-0016": {
		Class:    ClassType,
		Template: "cannot convert '{{.Value}}' to float",
	},
	"TYPE-0017": {
		Class:    ClassType,
		Template: "cannot convert '{{.Value}}' to number",
	},
	"TYPE-0018": {
		Class:    ClassType,
		Template: "slice {{.Position}} index must be an integer, got {{.Got}}",
	},
	"TYPE-0019": {
		Class:    ClassType,
		Template: "{{.Function}} element at index {{.Index}} must be {{.Expected}}, got {{.Got}}",
	},
	"TYPE-0020": {
		Class:    ClassType,
		Template: "{{.Context}} must be {{.Expected}}, got {{.Got}}",
	},
	"TYPE-0021": {
		Class:    ClassType,
		Template: "'{{.Name}}' is not a function",
	},
	"TYPE-0022": {
		Class:    ClassType,
		Template: "dot notation can only be used on dictionaries, got {{.Got}}",
	},

	// ========================================
	// Arity errors (ARITY-0xxx)
	// ========================================
	"ARITY-0001": {
		Class:    ClassArity,
		Template: "wrong number of arguments to `{{.Function}}`. got={{.Got}}, want={{.Want}}",
	},
	"ARITY-0002": {
		Class:    ClassArity,
		Template: "`{{.Function}}` expects {{.Want}} argument(s), got {{.Got}}",
	},
	"ARITY-0003": {
		Class:    ClassArity,
		Template: "comparison function must take exactly 2 parameters, got {{.Got}}",
	},
	"ARITY-0004": {
		Class:    ClassArity,
		Template: "`{{.Function}}` expects {{.Min}}-{{.Max}} arguments, got {{.Got}}",
	},
	"ARITY-0005": {
		Class:    ClassArity,
		Template: "`{{.Function}}` expects at least {{.Min}} argument(s), got {{.Got}}",
	},
	"ARITY-0006": {
		Class:    ClassArity,
		Template: "`{{.Function}}` expects exactly {{.Choice1}} or {{.Choice2}} argument(s), got {{.Got}}",
	},

	// ========================================
	// Undefined errors (UNDEF-0xxx)
	// ========================================
	"UNDEF-0001": {
		Class:    ClassUndefined,
		Template: "identifier not found: {{.Name}}",
		// Hint "Did you mean `X`?" added dynamically by fuzzy matching
	},
	"UNDEF-0002": {
		Class:    ClassUndefined,
		Template: "unknown method '{{.Method}}' for {{.Type}}",
	},
	"UNDEF-0003": {
		Class:    ClassUndefined,
		Template: "undefined component: {{.Name}}",
	},
	"UNDEF-0004": {
		Class:    ClassUndefined,
		Template: "unknown property '{{.Property}}' on {{.Type}}",
	},
	"UNDEF-0005": {
		Class:    ClassUndefined,
		Template: "unknown standard library module: @std/{{.Module}}",
	},
	"UNDEF-0006": {
		Class:    ClassUndefined,
		Template: "module does not export '{{.Name}}'",
	},

	// ========================================
	// I/O errors (IO-0xxx)
	// ========================================
	"IO-0001": {
		Class:    ClassIO,
		Template: "failed to {{.Operation}} '{{.Path}}': {{.GoError}}",
	},
	"IO-0002": {
		Class:    ClassIO,
		Template: "module not found: {{.Path}}",
	},
	"IO-0003": {
		Class:    ClassIO,
		Template: "failed to read file '{{.Path}}': {{.GoError}}",
	},
	"IO-0004": {
		Class:    ClassIO,
		Template: "failed to write file '{{.Path}}': {{.GoError}}",
	},
	"IO-0005": {
		Class:    ClassIO,
		Template: "failed to delete '{{.Path}}': {{.GoError}}",
	},
	"IO-0006": {
		Class:    ClassIO,
		Template: "failed to create directory '{{.Path}}': {{.GoError}}",
	},
	"IO-0007": {
		Class:    ClassIO,
		Template: "failed to resolve path '{{.Path}}': {{.GoError}}",
	},
	"IO-0008": {
		Class:    ClassIO,
		Template: "SFTP {{.Operation}} failed: {{.GoError}}",
	},
	"IO-0009": {
		Class:    ClassIO,
		Template: "failed to create directory '{{.Path}}': {{.GoError}}",
	},
	"IO-0010": {
		Class:    ClassIO,
		Template: "failed to remove directory '{{.Path}}': {{.GoError}}",
	},

	// ========================================
	// Database errors (DB-0xxx)
	// ========================================
	"DB-0001": {
		Class:    ClassDatabase,
		Template: "{{.Driver}} {{.Operation}} failed: {{.GoError}}",
	},
	"DB-0002": {
		Class:    ClassDatabase,
		Template: "query failed: {{.GoError}}",
	},
	"DB-0003": {
		Class:    ClassDatabase,
		Template: "failed to open {{.Driver}} database: {{.GoError}}",
	},
	"DB-0004": {
		Class:    ClassDatabase,
		Template: "failed to scan row: {{.GoError}}",
	},
	"DB-0005": {
		Class:    ClassDatabase,
		Template: "failed to ping database: {{.GoError}}",
	},
	"DB-0006": {
		Class:    ClassDatabase,
		Template: "no transaction in progress",
	},
	"DB-0007": {
		Class:    ClassDatabase,
		Template: "connection is already in a transaction",
	},
	"DB-0008": {
		Class:    ClassDatabase,
		Template: "failed to get columns: {{.GoError}}",
	},
	"DB-0009": {
		Class:    ClassState,
		Template: "cannot close server-managed database connection",
	},
	"DB-0010": {
		Class:    ClassDatabase,
		Template: "failed to close database connection: {{.GoError}}",
	},
	"DB-0011": {
		Class:    ClassDatabase,
		Template: "execute failed: {{.GoError}}",
	},

	// ========================================
	// Network errors (NET-0xxx)
	// ========================================
	"NET-0001": {
		Class:    ClassNetwork,
		Template: "{{.Operation}} to {{.URL}} failed: {{.GoError}}",
	},
	"NET-0002": {
		Class:    ClassNetwork,
		Template: "HTTP request failed: {{.GoError}}",
	},
	"NET-0003": {
		Class:    ClassNetwork,
		Template: "failed to connect to SSH server: {{.GoError}}",
	},
	"NET-0004": {
		Class:    ClassNetwork,
		Template: "HTTP {{.Method}} {{.URL}} returned {{.StatusCode}}",
	},
	"NET-0005": {
		Class:    ClassNetwork,
		Template: "SFTP: {{.GoError}}",
	},
	"NET-0006": {
		Class:    ClassNetwork,
		Template: "failed to read SSH key file: {{.GoError}}",
	},
	"NET-0007": {
		Class:    ClassNetwork,
		Template: "failed to parse SSH key: {{.GoError}}",
	},
	"NET-0008": {
		Class:    ClassNetwork,
		Template: "failed to load known_hosts: {{.GoError}}",
	},
	"NET-0009": {
		Class:    ClassNetwork,
		Template: "failed to create SFTP client: {{.GoError}}",
	},

	// ========================================
	// Security errors (SEC-0xxx)
	// ========================================
	"SEC-0001": {
		Class:    ClassSecurity,
		Template: "security: {{.Operation}} access denied",
		Hints:    []string{"use {{.Flag}} to allow this operation"},
	},
	"SEC-0002": {
		Class:    ClassSecurity,
		Template: "security: read access denied",
		Hints:    []string{"use --allow-read or -r to allow file reading"},
	},
	"SEC-0003": {
		Class:    ClassSecurity,
		Template: "security: write access denied",
		Hints:    []string{"use --allow-write or -w to allow file writing"},
	},
	"SEC-0004": {
		Class:    ClassSecurity,
		Template: "security: execute access denied",
		Hints:    []string{"use --allow-execute or -x to allow execution"},
	},
	"SEC-0005": {
		Class:    ClassSecurity,
		Template: "security: network access denied",
		Hints:    []string{"use --allow-net or -n to allow network access"},
	},
	"SEC-0006": {
		Class:    ClassSecurity,
		Template: "SFTP requires authentication: provide keyFile or password in options",
	},

	// ========================================
	// Index errors (INDEX-0xxx)
	// ========================================
	"INDEX-0001": {
		Class:    ClassIndex,
		Template: "index {{.Index}} out of range (length {{.Length}})",
	},
	"INDEX-0002": {
		Class:    ClassIndex,
		Template: "cannot {{.Operation}} from empty {{.Type}}",
	},
	"INDEX-0003": {
		Class:    ClassIndex,
		Template: "slice start index {{.Start}} is greater than end index {{.End}}",
	},
	"INDEX-0004": {
		Class:    ClassIndex,
		Template: "negative index not allowed: {{.Index}}",
	},
	"INDEX-0005": {
		Class:    ClassIndex,
		Template: "key '{{.Key}}' not found in dictionary",
	},

	// ========================================
	// Format errors (FMT-0xxx)
	// ========================================
	"FMT-0001": {
		Class:    ClassFormat,
		Template: "invalid {{.Format}}: {{.GoError}}",
	},
	"FMT-0002": {
		Class:    ClassFormat,
		Template: "invalid regex pattern: {{.GoError}}",
	},
	"FMT-0003": {
		Class:    ClassFormat,
		Template: "invalid URL: {{.GoError}}",
	},
	"FMT-0004": {
		Class:    ClassFormat,
		Template: "invalid datetime: {{.GoError}}",
	},
	"FMT-0005": {
		Class:    ClassFormat,
		Template: "invalid JSON: {{.GoError}}",
	},
	"FMT-0006": {
		Class:    ClassFormat,
		Template: "invalid YAML: {{.GoError}}",
	},
	"FMT-0007": {
		Class:    ClassFormat,
		Template: "invalid CSV: {{.GoError}}",
	},
	"FMT-0008": {
		Class:    ClassFormat,
		Template: "invalid locale: {{.Locale}}",
	},
	"FMT-0009": {
		Class:    ClassFormat,
		Template: "invalid duration: {{.GoError}}",
	},
	"FMT-0010": {
		Class:    ClassFormat,
		Template: "failed to convert markdown: {{.GoError}}",
	},

	// ========================================
	// Operator errors (OP-0xxx)
	// ========================================
	"OP-0001": {
		Class:    ClassOperator,
		Template: "unknown operator: {{.LeftType}} {{.Operator}} {{.RightType}}",
	},
	"OP-0002": {
		Class:    ClassOperator,
		Template: "division by zero",
	},
	"OP-0003": {
		Class:    ClassOperator,
		Template: "cannot compare {{.LeftType}} and {{.RightType}}",
	},
	"OP-0004": {
		Class:    ClassOperator,
		Template: "cannot negate {{.Type}}",
	},
	"OP-0005": {
		Class:    ClassOperator,
		Template: "unknown prefix operator: {{.Operator}}{{.Type}}",
	},
	"OP-0006": {
		Class:    ClassOperator,
		Template: "modulo by zero",
	},
	"OP-0007": {
		Class:    ClassOperator,
		Template: "left operand of {{.Operator}} must be {{.Expected}}, got {{.Got}}",
	},
	"OP-0008": {
		Class:    ClassOperator,
		Template: "right operand of {{.Operator}} must be {{.Expected}}, got {{.Got}}",
	},
	"OP-0009": {
		Class:    ClassOperator,
		Template: "type mismatch: {{.LeftType}} {{.Operator}} {{.RightType}}",
	},
	"OP-0010": {
		Class:    ClassOperator,
		Template: "unsupported type for mixed arithmetic: {{.Type}}",
	},
	"OP-0011": {
		Class:    ClassOperator,
		Template: "cannot add duration to datetime (use datetime + duration instead)",
		Hints:    []string{"datetime + duration is supported", "duration + datetime is not supported"},
	},
	"OP-0012": {
		Class:    ClassOperator,
		Template: "cannot intersect two {{.Kind}}s - {{.Hint}}",
	},
	"OP-0013": {
		Class:    ClassOperator,
		Template: "cannot compare durations with month components (months have variable length)",
	},
	"OP-0014": {
		Class:    ClassOperator,
		Template: "unknown operator for {{.Type}}: {{.Operator}}",
	},
	"OP-0015": {
		Class:    ClassOperator,
		Template: "unknown operator for {{.LeftType}} and {{.RightType}}: {{.Operator}} (supported: {{.Supported}})",
	},
	"OP-0016": {
		Class:    ClassOperator,
		Template: "'in' operator requires array, dictionary, or string on right side, got {{.Got}}",
	},
	"OP-0017": {
		Class:    ClassOperator,
		Template: "dictionary key must be a string, got {{.Got}}",
	},
	"OP-0018": {
		Class:    ClassOperator,
		Template: "substring must be a string, got {{.Got}}",
	},
	"OP-0019": {
		Class:    ClassOperator,
		Template: "cannot mix currencies: {{.LeftCurrency}} and {{.RightCurrency}}",
		Hints:    []string{"convert to the same currency before arithmetic"},
	},
	"OP-0020": {
		Class:    ClassOperator,
		Template: "unsupported operation between money values: {{.Operator}}",
		Hints:    []string{"only +, -, and comparison operators are allowed between money values"},
	},
	"OP-0021": {
		Class:    ClassOperator,
		Template: "unsupported operation between money and number: {{.Operator}}",
		Hints:    []string{"only * and / are allowed between money and numbers"},
	},

	// ========================================
	// State errors (STATE-0xxx)
	// ========================================
	"STATE-0001": {
		Class:    ClassState,
		Template: "{{.Resource}} is {{.ActualState}}, expected {{.ExpectedState}}",
	},
	"STATE-0002": {
		Class:    ClassState,
		Template: "SFTP connection is not connected",
	},
	"STATE-0003": {
		Class:    ClassState,
		Template: "file handle is closed",
	},

	// ========================================
	// Import errors (IMPORT-0xxx)
	// ========================================
	"IMPORT-0001": {
		Class:    ClassImport,
		Template: "in module {{.ModulePath}}: {{.NestedError}}",
	},
	"IMPORT-0002": {
		Class:    ClassImport,
		Template: "circular dependency detected when importing: {{.Path}}",
	},
	"IMPORT-0003": {
		Class:    ClassImport,
		Template: "parse errors in module {{.ModulePath}}",
	},
	"IMPORT-0004": {
		Class:    ClassImport,
		Template: "failed to resolve module path: {{.GoError}}",
	},
	"IMPORT-0005": {
		Class:    ClassImport,
		Template: "in module {{.ModulePath}}: line {{.Line}}, column {{.Column}}: {{.NestedError}}",
	},

	// ========================================
	// Command/Exec errors (CMD-0xxx)
	// ========================================
	"CMD-0001": {
		Class:    ClassState,
		Template: "command handle missing {{.Field}} field",
	},
	"CMD-0002": {
		Class:    ClassType,
		Template: "command {{.Field}} must be {{.Expected}}, got {{.Actual}}",
	},
	"CMD-0003": {
		Class:    ClassType,
		Template: "command arguments must be strings",
	},
	"CMD-0004": {
		Class:    ClassType,
		Template: "command input must be a string or null, got {{.Type}}",
	},

	// ========================================
	// Loop/iteration errors (LOOP-0xxx)
	// ========================================
	"LOOP-0001": {
		Class:    ClassType,
		Template: "for expects an array, string, or dictionary, got {{.Type}}",
	},
	"LOOP-0002": {
		Class:    ClassType,
		Template: "for expects a function, got {{.Type}}",
		Hints:    []string{"for (array) fn(x) { ... }", "for x in array { ... }"},
	},
	"LOOP-0003": {
		Class:    ClassState,
		Template: "for expression missing function or body",
	},
	"LOOP-0004": {
		Class:    ClassArity,
		Template: "function passed to for must take 1 or 2 parameters, got {{.Got}}",
	},
	"LOOP-0005": {
		Class:    ClassState,
		Template: "for loop over dictionary requires body with key, value parameters",
	},
	"LOOP-0006": {
		Class:    ClassState,
		Template: "for loop over dictionary requires function body",
	},
	"LOOP-0007": {
		Class:    ClassArity,
		Template: "for loop over dictionary requires exactly 2 parameters (key, value), got {{.Got}}",
	},

	// ========================================
	// Call errors (CALL-0xxx)
	// ========================================
	"CALL-0001": {
		Class:    ClassType,
		Template: "cannot call null as a function",
		Hints:    []string{"The value may not be exported from an imported module, or the variable is uninitialized"},
	},
	"CALL-0002": {
		Class:    ClassType,
		Template: "cannot call {{.Type}} as a function",
		Hints:    []string{"Only functions can be called with parentheses"},
	},
	"CALL-0003": {
		Class:    ClassType,
		Template: "dev module cannot be called directly, use dev.log() or other methods",
	},

	// ========================================
	// File operator errors (FILEOP-0xxx)
	// ========================================
	"FILEOP-0001": {
		Class:    ClassType,
		Template: "{{.Operator}} operator requires {{.Expected}}, got {{.Got}}",
	},
	"FILEOP-0002": {
		Class:    ClassState,
		Template: "file handle has no valid path",
	},
	"FILEOP-0003": {
		Class:    ClassState,
		Template: "file handle has no format specified",
	},
	"FILEOP-0004": {
		Class:    ClassType,
		Template: "file format must be a string, got {{.Got}}",
	},
	"FILEOP-0005": {
		Class:    ClassFormat,
		Template: "unsupported file format for {{.Operation}}: {{.Format}}",
	},
	"FILEOP-0006": {
		Class:    ClassIO,
		Template: "failed to encode data: {{.GoError}}",
	},
	"CMD-0005": {
		Class:    ClassType,
		Template: "left operand of <=#=> must be command handle, got {{.Got}}",
	},
	"CMD-0006": {
		Class:    ClassState,
		Template: "left operand of <=#=> must be command handle",
	},

	// ========================================
	// Validation errors (VAL-0xxx)
	// ========================================
	"VAL-0001": {
		Class:    ClassFormat,
		Template: "invalid currency code: {{.Code}}",
	},
	"VAL-0002": {
		Class:    ClassFormat,
		Template: "invalid style {{.Style}} for {{.Context}}, use {{.ValidOptions}}",
	},
	"VAL-0003": {
		Class:    ClassFormat,
		Template: "invalid file pattern '{{.Pattern}}': {{.GoError}}",
	},
	"VAL-0004": {
		Class:    ClassValue,
		Template: "argument to `{{.Method}}` must be non-negative, got {{.Got}}",
	},
	"VAL-0005": {
		Class:    ClassValue,
		Template: "cannot {{.Method}} from empty array",
	},
	"VAL-0006": {
		Class:    ClassValue,
		Template: "cannot take {{.Requested}} unique items from array of length {{.Length}}",
	},
	"VAL-0007": {
		Class:    ClassValue,
		Template: "invalid duration: {{.GoError}}",
	},
	"VAL-0008": {
		Class:    ClassValue,
		Template: "{{.Type}} handle has no valid path",
	},
	"VAL-0009": {
		Class:    ClassValue,
		Template: "{{.Function}}: invalid route '{{.Route}}' (use alphanumeric, hyphens, underscores)",
	},
	"VAL-0010": {
		Class:    ClassValue,
		Template: "orderBy column spec must have {{.Min}}-{{.Max}} elements, got {{.Got}}",
	},
	"VAL-0011": {
		Class:    ClassValue,
		Template: "{{.Function}} requires at least one column",
	},
	"VAL-0012": {
		Class:    ClassValue,
		Template: "chunk size must be > 0, got {{.Got}}",
	},
	"VAL-0013": {
		Class:    ClassValue,
		Template: "range start must be an integer, got {{.Got}}",
	},
	"VAL-0014": {
		Class:    ClassValue,
		Template: "range end must be an integer, got {{.Got}}",
	},
	"VAL-0015": {
		Class:    ClassValue,
		Template: "regex dictionary missing pattern field",
		Hints:    []string{"regex dictionaries must have a 'pattern' field"},
	},
	"VAL-0016": {
		Class:    ClassValue,
		Template: "regex pattern must be a string, got {{.Got}}",
	},
	"VAL-0017": {
		Class:    ClassValue,
		Template: "{{.Type}} dictionary missing {{.Field}} field",
	},
	"VAL-0018": {
		Class:    ClassValue,
		Template: "{{.Field}} field must be {{.Expected}}, got {{.Got}}",
	},
	"VAL-0019": {
		Class:    ClassValue,
		Template: "money() requires a 3-letter currency code, got '{{.Got}}'",
	},
	"VAL-0020": {
		Class:    ClassValue,
		Template: "money() scale must be between 0 and 10, got {{.Got}}",
	},
	"VAL-0021": {
		Class:    ClassValue,
		Template: "{{.Function}}() requires {{.Expected}}, got {{.Got}}",
	},

	// ========================================
	// Destructuring errors (DEST-0xxx)
	// ========================================
	"DEST-0001": {
		Class:    ClassType,
		Template: "dictionary destructuring requires a dictionary value, got {{.Got}}",
	},
	"DEST-0002": {
		Class:    ClassState,
		Template: "unsupported nested destructuring pattern",
	},

	// ========================================
	// Stdio errors (STDIO-0xxx)
	// ========================================
	"STDIO-0001": {
		Class:    ClassIO,
		Template: "cannot write to stdin",
	},
	"STDIO-0002": {
		Class:    ClassFormat,
		Template: "unknown stdio stream: {{.Name}}",
	},

	// ========================================
	// Misc/internal errors (INTERNAL-0xxx)
	// ========================================
	"INTERNAL-0001": {
		Class:    ClassState,
		Template: "{{.Context}} requires environment context",
	},
	"INTERNAL-0002": {
		Class:    ClassState,
		Template: "unknown node type: {{.Type}}",
	},
	"INTERNAL-0003": {
		Class:    ClassState,
		Template: "{{.Function}} failed: {{.GoError}}",
	},

	// ========================================
	// More database errors (DB-012+)
	// ========================================
	"DB-0012": {
		Class:    ClassType,
		Template: "{{.Operator}} requires a database connection, got {{.Got}}",
	},

	// ========================================
	// More call errors (CALL-004+)
	// ========================================
	"CALL-0004": {
		Class:    ClassType,
		Template: "cannot call '{{.Name}}' because it is null",
		Hints:    []string{"'{{.Name}}' may not be exported from the imported module. Check the export name matches."},
	},
	"CALL-0005": {
		Class:    ClassType,
		Template: "cannot call null as a function: {{.Context}}",
	},

	// ========================================
	// Component errors (COMP-0xxx)
	// ========================================
	"COMP-0001": {
		Class:    ClassType,
		Template: "cannot use '<{{.Name}}/>' because '{{.Name}}' is null",
		Hints:    []string{"'{{.Name}}' may not be exported from the imported module. Check the export name matches."},
	},
	"COMP-0002": {
		Class:    ClassType,
		Template: "cannot use '<{{.Name}}/>' because '{{.Name}}' is not a function (got {{.Got}})",
		Hints:    []string{"Components must be functions. Check that '{{.Name}}' is exported as a function."},
	},

	// ========================================
	// toDict errors (TODICT-0xxx)
	// ========================================
	"TODICT-0001": {
		Class:    ClassType,
		Template: "toDict requires array of [key, value] pairs",
	},
	"TODICT-0002": {
		Class:    ClassType,
		Template: "dictionary keys must be strings, got {{.Got}}",
	},
	"TODICT-0003": {
		Class:    ClassType,
		Template: "toDict: unsupported value type {{.Got}}",
	},

	// ========================================
	// Map/filter callback errors (CALLBACK-0xxx)
	// ========================================
	"CALLBACK-0001": {
		Class:    ClassArity,
		Template: "function passed to `{{.Function}}` must take exactly {{.Expected}} parameter(s), got {{.Got}}",
	},

	// ========================================
	// File read/write operator errors (FILEOP-007+)
	// ========================================
	"FILEOP-0007": {
		Class:    ClassType,
		Template: "{{.Operator}} requires {{.Expected}}, got {{.Got}}",
	},
	"FILEOP-0008": {
		Class:    ClassState,
		Template: "directory handle has no valid path",
	},

	// ========================================
	// SFTP format errors (SFTP-0xxx)
	// ========================================
	"SFTP-0001": {
		Class:    ClassType,
		Template: "{{.Format}} format requires {{.Expected}}, got {{.Got}}",
	},
	"SFTP-0002": {
		Class:    ClassType,
		Template: "{{.Format}} format requires {{.Expected}} at index {{.Index}}, got {{.Got}}",
	},
	"SFTP-0003": {
		Class:    ClassState,
		Template: "CSV write not yet implemented for SFTP",
	},
	"SFTP-0004": {
		Class:    ClassFormat,
		Template: "unknown format: {{.Format}}",
	},
	"SFTP-0005": {
		Class:    ClassIO,
		Template: "SFTP write failed: {{.GoError}}",
	},

	// ========================================
	// Spread errors (SPREAD-0xxx)
	// ========================================
	"SPREAD-0001": {
		Class:    ClassType,
		Template: "spread operator requires a dictionary, got {{.Got}}",
	},

	// ========================================
	// SQL errors (SQL-0xxx)
	// ========================================
	"SQL-0001": {
		Class:    ClassType,
		Template: "SQL tag content must be a string",
	},
	"SQL-0002": {
		Class:    ClassType,
		Template: "query object missing 'sql' property",
	},
	"SQL-0003": {
		Class:    ClassType,
		Template: "sql property must be a string, got {{.Got}}",
	},
	"SQL-0004": {
		Class:    ClassType,
		Template: "query must be a string or <SQL> tag, got {{.Got}}",
	},

	// ========================================
	// HTTP errors (HTTP-0xxx)
	// ========================================
	"HTTP-0001": {
		Class:    ClassState,
		Template: "request handle has no valid URL",
	},
	"HTTP-0002": {
		Class:    ClassFormat,
		Template: "failed to encode request body: {{.GoError}}",
	},
	"HTTP-0003": {
		Class:    ClassNetwork,
		Template: "failed to create request: {{.GoError}}",
	},
	"HTTP-0004": {
		Class:    ClassNetwork,
		Template: "fetch failed: {{.GoError}}",
	},
	"HTTP-0005": {
		Class:    ClassIO,
		Template: "failed to read response: {{.GoError}}",
	},
	"HTTP-0006": {
		Class:    ClassNetwork,
		Template: "HTTP error: {{.Error}}",
	},

	// ========================================
	// Stdio read errors (STDIO-0003+)
	// ========================================
	"STDIO-0003": {
		Class:    ClassIO,
		Template: "failed to read from stdin: {{.GoError}}",
	},
	"STDIO-0004": {
		Class:    ClassIO,
		Template: "cannot read from {{.Stream}}",
	},

	// ========================================
	// SFTP read errors (SFTP-0006+)
	// ========================================
	"SFTP-0006": {
		Class:    ClassIO,
		Template: "failed to list directory: {{.GoError}}",
	},
	"SFTP-0007": {
		Class:    ClassIO,
		Template: "SFTP read failed: {{.GoError}}",
	},
}

// New creates a ParsleyError from the catalog.
// If the code is not found, creates a generic error with the message.
// Type names in common keys (Got, Type, LeftType, RightType) are automatically
// normalized to lowercase for consistent error messages.
func New(code string, data map[string]any) *ParsleyError {
	// Normalize type names to lowercase in common keys
	if data != nil {
		for _, key := range []string{"Got", "Type", "LeftType", "RightType"} {
			if v, ok := data[key]; ok {
				// Use fmt.Sprintf to handle both string and custom string types (like ObjectType)
				if s := fmt.Sprintf("%v", v); s != "" {
					data[key] = TypeName(s)
				}
			}
		}
	}

	def, ok := ErrorCatalog[code]
	if !ok {
		// Unknown code - create a generic error
		msg := code
		if data != nil {
			if m, ok := data["message"].(string); ok {
				msg = m
			}
		}
		return &ParsleyError{
			Class:   ClassType, // Default class
			Code:    code,
			Message: msg,
			Data:    data,
		}
	}

	// Render the message template
	msg := renderTemplate(def.Template, data)

	// Render hint templates
	var hints []string
	for _, hintTmpl := range def.Hints {
		rendered := renderTemplate(hintTmpl, data)
		if rendered != "" {
			hints = append(hints, rendered)
		}
	}

	return &ParsleyError{
		Class:   def.Class,
		Code:    code,
		Message: msg,
		Hints:   hints,
		Data:    data,
	}
}

// NewWithPosition creates a ParsleyError with position information.
func NewWithPosition(code string, line, column int, data map[string]any) *ParsleyError {
	err := New(code, data)
	err.Line = line
	err.Column = column
	return err
}

// NewSimple creates a simple error without using the catalog.
// Use this for one-off errors or when migrating existing code.
func NewSimple(class ErrorClass, message string) *ParsleyError {
	return &ParsleyError{
		Class:   class,
		Message: message,
	}
}

// NewSimpleWithHints creates a simple error with hints.
func NewSimpleWithHints(class ErrorClass, message string, hints ...string) *ParsleyError {
	return &ParsleyError{
		Class:   class,
		Message: message,
		Hints:   hints,
	}
}

// renderTemplate renders a Go template with the given data.
func renderTemplate(tmplStr string, data map[string]any) string {
	if data == nil {
		return tmplStr
	}

	tmpl, err := template.New("").Parse(tmplStr)
	if err != nil {
		return tmplStr
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return tmplStr
	}

	return buf.String()
}

// TypeName returns a lowercase type name for error messages.
// Converts "STRING" to "string", "ARRAY" to "array", etc.
func TypeName(t string) string {
	return strings.ToLower(t)
}

// ============================================================================
// Fuzzy Matching - "Did you mean?" suggestions
// ============================================================================

// levenshteinDistance computes the edit distance between two strings.
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create matrix
	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(a)][len(b)]
}

// FuzzyMatch represents a fuzzy match result with its distance.
type FuzzyMatch struct {
	Value    string
	Distance int
}

// FindClosestMatch finds the closest match to the given string from candidates.
// Returns the best match if the distance is within the threshold, otherwise empty string.
// The threshold is calculated dynamically based on the length of the input.
func FindClosestMatch(input string, candidates []string) string {
	if len(input) == 0 || len(candidates) == 0 {
		return ""
	}

	// Normalize input to lowercase for comparison
	inputLower := strings.ToLower(input)

	var bestMatch string
	bestDistance := -1

	for _, candidate := range candidates {
		// Normalize candidate to lowercase for comparison
		candidateLower := strings.ToLower(candidate)

		dist := levenshteinDistance(inputLower, candidateLower)

		if bestDistance == -1 || dist < bestDistance {
			bestDistance = dist
			bestMatch = candidate // Return original case
		}
	}

	// Calculate threshold based on input length
	// Short words (1-3): max 1 edit
	// Medium words (4-6): max 2 edits
	// Longer words (7+): max 3 edits
	threshold := 1
	if len(input) >= 4 && len(input) <= 6 {
		threshold = 2
	} else if len(input) >= 7 {
		threshold = 3
	}

	// Don't suggest if distance is 0 (exact match) or over threshold
	if bestDistance <= 0 || bestDistance > threshold {
		return ""
	}

	return bestMatch
}

// FindTopMatches returns the top N closest matches to the input.
// Useful for showing multiple suggestions.
func FindTopMatches(input string, candidates []string, n int) []string {
	if len(input) == 0 || len(candidates) == 0 || n <= 0 {
		return nil
	}

	// Normalize input to lowercase for comparison
	inputLower := strings.ToLower(input)

	// Calculate distances for all candidates
	var matches []FuzzyMatch
	for _, candidate := range candidates {
		candidateLower := strings.ToLower(candidate)
		dist := levenshteinDistance(inputLower, candidateLower)
		// Exclude exact matches
		if dist > 0 {
			matches = append(matches, FuzzyMatch{Value: candidate, Distance: dist})
		}
	}

	// Sort by distance
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Distance < matches[j].Distance
	})

	// Calculate threshold based on input length
	threshold := 1
	if len(input) >= 4 && len(input) <= 6 {
		threshold = 2
	} else if len(input) >= 7 {
		threshold = 3
	}

	// Return top N matches within threshold
	var result []string
	for i := 0; i < len(matches) && i < n; i++ {
		if matches[i].Distance <= threshold {
			result = append(result, matches[i].Value)
		}
	}

	return result
}

// NewUndefinedIdentifier creates an undefined identifier error with optional fuzzy matching.
func NewUndefinedIdentifier(name string, availableIdentifiers []string) *ParsleyError {
	data := map[string]any{"Name": name}
	err := New("UNDEF-0001", data)

	// Try fuzzy matching for "Did you mean?" hint
	if suggestion := FindClosestMatch(name, availableIdentifiers); suggestion != "" {
		err.Hints = append(err.Hints, "Did you mean `"+suggestion+"`?")
	}

	return err
}

// NewUndefinedMethod creates an undefined method error with optional fuzzy matching.
func NewUndefinedMethod(method, typeName string, availableMethods []string) *ParsleyError {
	data := map[string]any{
		"Method": method,
		"Type":   typeName,
	}
	err := New("UNDEF-0002", data)

	// Try fuzzy matching for "Did you mean?" hint
	if suggestion := FindClosestMatch(method, availableMethods); suggestion != "" {
		err.Hints = append(err.Hints, "Did you mean `"+suggestion+"`?")
	}

	return err
}

// Parsley reserved keywords for fuzzy matching against typos
var ParsleyKeywords = []string{
	"if", "else", "for", "in", "fn", "let", "const", "return",
	"true", "false", "null", "and", "or", "not", "import", "export",
	"break", "continue", "switch", "case", "default",
}
