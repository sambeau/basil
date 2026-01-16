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

// IsCatchable returns true if errors of this class can be caught by try expressions.
// Catchable errors are "user errors" - external factors that can't be validated before calling.
// Non-catchable errors are "developer errors" - logic bugs that should halt execution.
func (ec ErrorClass) IsCatchable() bool {
	switch ec {
	case ClassIO, ClassNetwork, ClassDatabase, ClassFormat, ClassValue, ClassSecurity:
		return true
	default:
		return false
	}
}

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
		Template: "Expected {{.Expected}}, got '{{.Got}}'",
	},
	"PARSE-0002": {
		Class:    ClassParse,
		Template: "Unexpected '{{.Token}}'",
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
		Template: "Invalid regex literal: {{.Literal}}",
	},
	"PARSE-0006": {
		Class:    ClassParse,
		Template: "Unterminated string",
	},
	"PARSE-0007": {
		Class:    ClassParse,
		Template: "Invalid number literal: {{.Literal}}",
	},
	"PARSE-0008": {
		Class:    ClassParse,
		Template: "Singleton tag must be self-closing",
		Hints:    []string{"<{{.Tag}}/>"},
	},
	"PARSE-0009": {
		Class:    ClassParse,
		Template: "Unclosed { in {{.Context}}",
	},
	"PARSE-0010": {
		Class:    ClassParse,
		Template: "Empty interpolation {} in {{.Context}}",
	},
	"PARSE-0011": {
		Class:    ClassParse,
		Template: "Error parsing {{.Context}} expression: {{.GoError}}",
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
		Template: "Argument to `{{.Function}}` not supported, got {{.Got}}",
	},
	"TYPE-0003": {
		Class:    ClassType,
		Template: "Cannot call {{.Got}} as a function",
	},
	"TYPE-0004": {
		Class:    ClassType,
		Template: "`for ({{.Array}}) {{.Got}}` is ambiguous",
		Hints:    []string{"for _ in {{.Array}} { ... }", "for ({{.Array}}) fn(x) { ... }"},
	},
	"TYPE-0005": {
		Class:    ClassType,
		Template: "First argument to `{{.Function}}` must be {{.Expected}}, got {{.Got}}",
	},
	"TYPE-0006": {
		Class:    ClassType,
		Template: "Second argument to `{{.Function}}` must be {{.Expected}}, got {{.Got}}",
	},
	"TYPE-0007": {
		Class:    ClassType,
		Template: "Cannot iterate over {{.Got}}",
		Hints:    []string{"for works with arrays, strings, and ranges"},
	},
	"TYPE-0008": {
		Class:    ClassType,
		Template: "Cannot index {{.Got}} with {{.IndexType}}",
	},
	"TYPE-0009": {
		Class:    ClassType,
		Template: "Comparison function must return boolean, got {{.Got}}",
	},
	"TYPE-0010": {
		Class:    ClassType,
		Template: "{{.Function}} callback must be a function, got {{.Got}}",
	},
	"TYPE-0011": {
		Class:    ClassType,
		Template: "Third argument to `{{.Function}}` must be {{.Expected}}, got {{.Got}}",
	},
	"TYPE-0012": {
		Class:    ClassType,
		Template: "Argument to `{{.Function}}` must be {{.Expected}}, got {{.Got}}",
	},
	"TYPE-0013": {
		Class:    ClassType,
		Template: "Index operator not supported: {{.Left}}[{{.Right}}]",
		Hints:    []string{"Arrays and strings can be indexed with integers", "Dictionaries can be indexed with strings"},
	},
	"TYPE-0014": {
		Class:    ClassType,
		Template: "Slice operator not supported: {{.Type}}",
		Hints:    []string{"Slicing works with arrays and strings"},
	},
	"TYPE-0015": {
		Class:    ClassType,
		Template: "Cannot convert '{{.Value}}' to integer",
	},
	"TYPE-0016": {
		Class:    ClassType,
		Template: "Cannot convert '{{.Value}}' to float",
	},
	"TYPE-0017": {
		Class:    ClassType,
		Template: "Cannot convert '{{.Value}}' to number",
	},
	"TYPE-0018": {
		Class:    ClassType,
		Template: "Slice {{.Position}} index must be an integer, got {{.Got}}",
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
		Template: "Dot notation can only be used on dictionaries, got {{.Got}}",
	},
	"TYPE-0023": {
		Class:    ClassType,
		Template: "Key '{{.Key}}' already exists in dictionary",
	},

	// ========================================
	// Arity errors (ARITY-0xxx)
	// ========================================
	"ARITY-0001": {
		Class:    ClassArity,
		Template: "Wrong number of arguments to `{{.Function}}`. got={{.Got}}, want={{.Want}}",
	},
	"ARITY-0002": {
		Class:    ClassArity,
		Template: "`{{.Function}}` expects {{.Want}} argument(s), got {{.Got}}",
	},
	"ARITY-0003": {
		Class:    ClassArity,
		Template: "Comparison function must take exactly 2 parameters, got {{.Got}}",
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
		Template: "Identifier not found: {{.Name}}",
		// Hint "Did you mean `X`?" added dynamically by fuzzy matching
	},
	"UNDEF-0002": {
		Class:    ClassUndefined,
		Template: "Unknown method '{{.Method}}' for {{.Type}}",
	},
	"UNDEF-0003": {
		Class:    ClassUndefined,
		Template: "Undefined component: {{.Name}}",
	},
	"UNDEF-0004": {
		Class:    ClassUndefined,
		Template: "Unknown property '{{.Property}}' on {{.Type}}",
	},
	"UNDEF-0005": {
		Class:    ClassUndefined,
		Template: "Unknown standard library module: @std/{{.Module}}",
	},
	"UNDEF-0006": {
		Class:    ClassUndefined,
		Template: "Module does not export '{{.Name}}'",
	},
	"UNDEF-0007": {
		Class:    ClassUndefined,
		Template: "Unknown basil module: @basil/{{.Module}}",
	},
	"UNDEF-0010": {
		Class:    ClassUndefined,
		Template: "@params is not available at module scope",
		// Hints added in evalIdentifier
	},

	// ========================================
	// I/O errors (IO-0xxx)
	// ========================================
	"IO-0001": {
		Class:    ClassIO,
		Template: "Failed to {{.Operation}} '{{.Path}}': {{.GoError}}",
	},
	"IO-0002": {
		Class:    ClassIO,
		Template: "Module not found: {{.Path}}",
	},
	"IO-0003": {
		Class:    ClassIO,
		Template: "Failed to read file '{{.Path}}': {{.GoError}}",
	},
	"IO-0004": {
		Class:    ClassIO,
		Template: "Failed to write file '{{.Path}}': {{.GoError}}",
	},
	"IO-0005": {
		Class:    ClassIO,
		Template: "Failed to delete '{{.Path}}': {{.GoError}}",
	},
	"IO-0006": {
		Class:    ClassIO,
		Template: "Failed to create directory '{{.Path}}': {{.GoError}}",
	},
	"IO-0007": {
		Class:    ClassIO,
		Template: "Failed to resolve path '{{.Path}}': {{.GoError}}",
	},
	"IO-0008": {
		Class:    ClassIO,
		Template: "SFTP {{.Operation}} failed: {{.GoError}}",
	},
	"IO-0009": {
		Class:    ClassIO,
		Template: "Failed to create directory '{{.Path}}': {{.GoError}}",
	},
	"IO-0010": {
		Class:    ClassIO,
		Template: "Failed to remove directory '{{.Path}}': {{.GoError}}",
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
		Template: "Query failed: {{.GoError}}",
	},
	"DB-0003": {
		Class:    ClassDatabase,
		Template: "Failed to open {{.Driver}} database: {{.GoError}}",
	},
	"DB-0004": {
		Class:    ClassDatabase,
		Template: "Failed to scan row: {{.GoError}}",
	},
	"DB-0005": {
		Class:    ClassDatabase,
		Template: "Failed to ping database: {{.GoError}}",
	},
	"DB-0006": {
		Class:    ClassDatabase,
		Template: "No transaction in progress",
	},
	"DB-0007": {
		Class:    ClassDatabase,
		Template: "Connection is already in a transaction",
	},
	"DB-0008": {
		Class:    ClassDatabase,
		Template: "Failed to get columns: {{.GoError}}",
	},
	"DB-0009": {
		Class:    ClassState,
		Template: "Cannot close server-managed database connection",
	},
	"DB-0010": {
		Class:    ClassDatabase,
		Template: "Failed to close database connection: {{.GoError}}",
	},
	"DB-0011": {
		Class:    ClassDatabase,
		Template: "Execute failed: {{.GoError}}",
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
		Template: "Failed to connect to SSH server: {{.GoError}}",
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
		Template: "Failed to read SSH key file: {{.GoError}}",
	},
	"NET-0007": {
		Class:    ClassNetwork,
		Template: "Failed to parse SSH key: {{.GoError}}",
	},
	"NET-0008": {
		Class:    ClassNetwork,
		Template: "Failed to load known_hosts: {{.GoError}}",
	},
	"NET-0009": {
		Class:    ClassNetwork,
		Template: "Failed to create SFTP client: {{.GoError}}",
	},

	// ========================================
	// Security errors (SEC-0xxx)
	// ========================================
	"SEC-0001": {
		Class:    ClassSecurity,
		Template: "Security: {{.Operation}} access denied",
		Hints:    []string{"use {{.Flag}} to allow this operation"},
	},
	"SEC-0002": {
		Class:    ClassSecurity,
		Template: "Security: read access denied",
		Hints:    []string{"use --allow-read or -r to allow file reading"},
	},
	"SEC-0003": {
		Class:    ClassSecurity,
		Template: "Security: write access denied",
		Hints:    []string{"writes are allowed by default; check if --no-write or --restrict-write was used"},
	},
	"SEC-0004": {
		Class:    ClassSecurity,
		Template: "Security: execute access denied",
		Hints:    []string{"use --allow-execute or -x to allow execution"},
	},
	"SEC-0005": {
		Class:    ClassSecurity,
		Template: "Security: network access denied",
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
		Template: "Index {{.Index}} out of range (length {{.Length}})",
	},
	"INDEX-0002": {
		Class:    ClassIndex,
		Template: "Cannot {{.Operation}} from empty {{.Type}}",
	},
	"INDEX-0003": {
		Class:    ClassIndex,
		Template: "Slice start index {{.Start}} is greater than end index {{.End}}",
	},
	"INDEX-0004": {
		Class:    ClassIndex,
		Template: "Negative index not allowed: {{.Index}}",
	},
	"INDEX-0005": {
		Class:    ClassIndex,
		Template: "Key '{{.Key}}' not found in dictionary",
	},

	// ========================================
	// Format errors (FMT-0xxx)
	// ========================================
	"FMT-0001": {
		Class:    ClassFormat,
		Template: "Invalid {{.Format}}: {{.GoError}}",
	},
	"FMT-0002": {
		Class:    ClassFormat,
		Template: "Invalid regex pattern: {{.GoError}}",
	},
	"FMT-0003": {
		Class:    ClassFormat,
		Template: "Invalid URL: {{.GoError}}",
	},
	"FMT-0004": {
		Class:    ClassFormat,
		Template: "Invalid datetime: {{.GoError}}",
	},
	"FMT-0005": {
		Class:    ClassFormat,
		Template: "Invalid JSON: {{.GoError}}",
	},
	"FMT-0006": {
		Class:    ClassFormat,
		Template: "Invalid YAML: {{.GoError}}",
	},
	"FMT-0007": {
		Class:    ClassFormat,
		Template: "Invalid CSV: {{.GoError}}",
	},
	"FMT-0008": {
		Class:    ClassFormat,
		Template: "Invalid locale: {{.Locale}}",
	},
	"FMT-0009": {
		Class:    ClassFormat,
		Template: "Invalid duration: {{.GoError}}",
	},
	"FMT-0010": {
		Class:    ClassFormat,
		Template: "Failed to convert markdown: {{.GoError}}",
	},

	// ========================================
	// Value errors (VALUE-0xxx)
	// ========================================
	"VALUE-0001": {
		Class:    ClassValue,
		Template: "`{{.Function}}` requires a non-empty array",
	},
	"VALUE-0002": {
		Class:    ClassValue,
		Template: "`{{.Function}}` requires a non-negative number, got {{.Got}}",
	},
	"VALUE-0003": {
		Class:    ClassValue,
		Template: "`{{.Function}}` domain error: {{.Reason}}",
	},

	// ========================================
	// Operator errors (OP-0xxx)
	// ========================================
	"OP-0001": {
		Class:    ClassOperator,
		Template: "Unknown operator: {{.LeftType}} {{.Operator}} {{.RightType}}",
	},
	"OP-0002": {
		Class:    ClassOperator,
		Template: "Division by zero",
	},
	"OP-0003": {
		Class:    ClassOperator,
		Template: "Cannot compare {{.LeftType}} and {{.RightType}}",
	},
	"OP-0004": {
		Class:    ClassOperator,
		Template: "Cannot negate {{.Type}}",
	},
	"OP-0005": {
		Class:    ClassOperator,
		Template: "Unknown prefix operator: {{.Operator}}{{.Type}}",
	},
	"OP-0006": {
		Class:    ClassOperator,
		Template: "Modulo by zero",
	},
	"OP-0007": {
		Class:    ClassOperator,
		Template: "Left operand of {{.Operator}} must be {{.Expected}}, got {{.Got}}",
	},
	"OP-0008": {
		Class:    ClassOperator,
		Template: "Right operand of {{.Operator}} must be {{.Expected}}, got {{.Got}}",
	},
	"OP-0009": {
		Class:    ClassOperator,
		Template: "Type mismatch: {{.LeftType}} {{.Operator}} {{.RightType}}",
	},
	"OP-0010": {
		Class:    ClassOperator,
		Template: "Unsupported type for mixed arithmetic: {{.Type}}",
	},
	"OP-0011": {
		Class:    ClassOperator,
		Template: "Cannot add duration to datetime",
		Hints:    []string{"use datetime + duration instead"},
	},
	"OP-0012": {
		Class:    ClassOperator,
		Template: "Cannot intersect two {{.Kind}}s - {{.Hint}}",
	},
	"OP-0013": {
		Class:    ClassOperator,
		Template: "Cannot compare durations with month components (months have variable length)",
	},
	"OP-0014": {
		Class:    ClassOperator,
		Template: "Unknown operator for {{.Type}}: {{.Operator}}",
	},
	"OP-0015": {
		Class:    ClassOperator,
		Template: "Unknown operator for {{.LeftType}} and {{.RightType}}: {{.Operator}} (supported: {{.Supported}})",
	},
	"OP-0016": {
		Class:    ClassOperator,
		Template: "'in' operator requires array, dictionary, or string on right side, got {{.Got}}",
	},
	"OP-0017": {
		Class:    ClassOperator,
		Template: "Dictionary key must be a string, got {{.Got}}",
	},
	"OP-0018": {
		Class:    ClassOperator,
		Template: "Substring must be a string, got {{.Got}}",
	},
	"OP-0019": {
		Class:    ClassOperator,
		Template: "Cannot mix currencies: {{.LeftCurrency}} and {{.RightCurrency}}",
		Hints:    []string{"convert to the same currency before arithmetic"},
	},
	"OP-0020": {
		Class:    ClassOperator,
		Template: "Unsupported operation between money values: {{.Operator}}",
		Hints:    []string{"only +, -, and comparison operators are allowed between money values"},
	},
	"OP-0021": {
		Class:    ClassOperator,
		Template: "Unsupported operation between money and number: {{.Operator}}",
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
		Template: "File handle is closed",
	},

	// ========================================
	// Import errors (IMPORT-0xxx)
	// ========================================
	"IMPORT-0001": {
		Class:    ClassImport,
		Template: "In module {{.ModulePath}}: {{.NestedError}}",
	},
	"IMPORT-0002": {
		Class:    ClassImport,
		Template: "Circular dependency detected when importing: {{.Path}}",
	},
	"IMPORT-0003": {
		Class:    ClassImport,
		Template: "Parse errors in module {{.ModulePath}}",
	},
	"IMPORT-0004": {
		Class:    ClassImport,
		Template: "Failed to resolve module path: {{.GoError}}",
	},
	"IMPORT-0005": {
		Class:    ClassImport,
		Template: "In module {{.ModulePath}}: line {{.Line}}, column {{.Column}}: {{.NestedError}}",
	},
	"IMPORT-0006": {
		Class:    ClassImport,
		Template: "Standard library module @std/{{.Module}} has been removed. {{.Replacement}}",
		Hints: []string{
			"Use @basil/http for request/response context",
			"Use @basil/auth for db/session/auth",
		},
	},

	// ========================================
	// Command/Exec errors (CMD-0xxx)
	// ========================================
	"CMD-0001": {
		Class:    ClassState,
		Template: "Command handle missing {{.Field}} field",
	},
	"CMD-0002": {
		Class:    ClassType,
		Template: "Command {{.Field}} must be {{.Expected}}, got {{.Actual}}",
	},
	"CMD-0003": {
		Class:    ClassType,
		Template: "Command arguments must be strings",
	},
	"CMD-0004": {
		Class:    ClassType,
		Template: "Command input must be a string or null, got {{.Type}}",
	},

	// ========================================
	// Loop/iteration errors (LOOP-0xxx)
	// ========================================
	"LOOP-0001": {
		Class:    ClassType,
		Template: "For expects an array, string, or dictionary, got {{.Type}}",
	},
	"LOOP-0002": {
		Class:    ClassType,
		Template: "For expects a function, got {{.Type}}",
		Hints:    []string{"for (array) fn(x) { ... }", "for x in array { ... }"},
	},
	"LOOP-0003": {
		Class:    ClassState,
		Template: "For expression missing function or body",
	},
	"LOOP-0004": {
		Class:    ClassArity,
		Template: "Function passed to for must take 1 or 2 parameters, got {{.Got}}",
	},
	"LOOP-0005": {
		Class:    ClassState,
		Template: "For loop over dictionary requires body with key, value parameters",
	},
	"LOOP-0006": {
		Class:    ClassState,
		Template: "For loop over dictionary requires function body",
	},
	"LOOP-0007": {
		Class:    ClassArity,
		Template: "For loop over dictionary requires exactly 2 parameters (key, value), got {{.Got}}",
	},

	// ========================================
	// Call errors (CALL-0xxx)
	// ========================================
	"CALL-0001": {
		Class:    ClassType,
		Template: "Cannot call null as a function",
		Hints:    []string{"The value may not be exported from an imported module, or the variable is uninitialized"},
	},
	"CALL-0002": {
		Class:    ClassType,
		Template: "Cannot call {{.Type}} as a function",
		Hints:    []string{"Only functions can be called with parentheses"},
	},
	"CALL-0003": {
		Class:    ClassType,
		Template: "Dev module cannot be called directly, use dev.log() or other methods",
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
		Template: "File handle has no valid path",
	},
	"FILEOP-0003": {
		Class:    ClassState,
		Template: "File handle has no format specified",
	},
	"FILEOP-0004": {
		Class:    ClassType,
		Template: "File format must be a string, got {{.Got}}",
	},
	"FILEOP-0005": {
		Class:    ClassFormat,
		Template: "Unsupported file format for {{.Operation}}: {{.Format}}",
	},
	"FILEOP-0006": {
		Class:    ClassIO,
		Template: "Failed to encode data: {{.GoError}}",
	},
	"CMD-0005": {
		Class:    ClassType,
		Template: "Left operand of <=#=> must be command handle, got {{.Got}}",
	},
	"CMD-0006": {
		Class:    ClassState,
		Template: "Left operand of <=#=> must be command handle",
	},

	// ========================================
	// Validation errors (VAL-0xxx)
	// ========================================
	"VAL-0001": {
		Class:    ClassFormat,
		Template: "Invalid currency code: {{.Code}}",
	},
	"VAL-0002": {
		Class:    ClassFormat,
		Template: "Invalid style {{.Style}} for {{.Context}}, use {{.ValidOptions}}",
	},
	"VAL-0003": {
		Class:    ClassFormat,
		Template: "Invalid file pattern '{{.Pattern}}': {{.GoError}}",
	},
	"VAL-0004": {
		Class:    ClassValue,
		Template: "Argument to `{{.Method}}` must be non-negative, got {{.Got}}",
	},
	"VAL-0005": {
		Class:    ClassValue,
		Template: "Cannot {{.Method}} from empty array",
	},
	"VAL-0006": {
		Class:    ClassValue,
		Template: "Cannot take {{.Requested}} unique items from array of length {{.Length}}",
	},
	"VAL-0007": {
		Class:    ClassValue,
		Template: "Invalid duration: {{.GoError}}",
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
		Template: "The orderBy column spec must have {{.Min}}-{{.Max}} elements, got {{.Got}}",
	},
	"VAL-0011": {
		Class:    ClassValue,
		Template: "{{.Function}} requires at least one column",
	},
	"VAL-0012": {
		Class:    ClassValue,
		Template: "Chunk size must be > 0, got {{.Got}}",
	},
	"VAL-0013": {
		Class:    ClassValue,
		Template: "Range start must be an integer, got {{.Got}}",
	},
	"VAL-0014": {
		Class:    ClassValue,
		Template: "Range end must be an integer, got {{.Got}}",
	},
	"VAL-0015": {
		Class:    ClassValue,
		Template: "Regex dictionary missing pattern field",
		Hints:    []string{"regex dictionaries must have a 'pattern' field"},
	},
	"VAL-0016": {
		Class:    ClassValue,
		Template: "Regex pattern must be a string, got {{.Got}}",
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
		Template: "The money() function requires a 3-letter currency code, got '{{.Got}}'",
	},
	"VAL-0020": {
		Class:    ClassValue,
		Template: "The money() scale must be between 0 and 10, got {{.Got}}",
	},
	"VAL-0021": {
		Class:    ClassValue,
		Template: "{{.Function}}() requires {{.Expected}}, got {{.Got}}",
	},

	// ========================================
	// Table errors (TABLE-0xxx)
	// ========================================
	"TABLE-0001": {
		Class:    ClassType,
		Template: "table() requires an array, got {{.Got}}",
		Hints:    []string{"Create a table from an array of dictionaries: table([{a: 1}, {a: 2}])"},
	},
	"TABLE-0002": {
		Class:    ClassType,
		Template: "Table row {{.Row}}: expected dictionary, got {{.Got}}",
		Hints:    []string{"Each row in a table must be a dictionary with consistent keys"},
	},
	"TABLE-0003": {
		Class:    ClassType,
		Template: "Table row {{.Row}}: missing columns [{{.Missing}}]",
		Hints:    []string{"All rows must have the same columns as the first row"},
	},
	"TABLE-0004": {
		Class:    ClassType,
		Template: "Table row {{.Row}}: unexpected columns [{{.Extra}}]",
		Hints:    []string{"All rows must have the same columns as the first row"},
	},
	"TABLE-0005": {
		Class:    ClassType,
		Template: "Table row {{.Row}}: missing required field '{{.Field}}'",
		Hints:    []string{"Required schema fields must be provided or have a default value"},
	},

	// ========================================
	// Destructuring errors (DEST-0xxx)
	// ========================================
	"DEST-0001": {
		Class:    ClassType,
		Template: "Dictionary destructuring requires a dictionary value, got {{.Got}}",
	},
	"DEST-0002": {
		Class:    ClassState,
		Template: "Unsupported nested destructuring pattern",
	},

	// ========================================
	// Stdio errors (STDIO-0xxx)
	// ========================================
	"STDIO-0001": {
		Class:    ClassIO,
		Template: "Cannot write to stdin",
	},
	"STDIO-0002": {
		Class:    ClassFormat,
		Template: "Unknown stdio stream: {{.Name}}",
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
		Template: "Unknown node type: {{.Type}}",
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
		Template: "Cannot call '{{.Name}}' because it is null",
		Hints:    []string{"'{{.Name}}' may not be exported from the imported module. Check the export name matches."},
	},
	"CALL-0005": {
		Class:    ClassType,
		Template: "Cannot call null as a function: {{.Context}}",
	},

	// ========================================
	// Component errors (COMP-0xxx)
	// ========================================
	"COMP-0001": {
		Class:    ClassType,
		Template: "Component '<{{.Name}}/>' not found - '{{.Name}}' is null or not exported",
		Hints:    []string{"Did you forget to 'export {{.Name}}' in the imported module?", "Check that the export name matches exactly (case-sensitive)", "Ensure the module file exists and is imported correctly"},
	},
	"COMP-0002": {
		Class:    ClassType,
		Template: "Cannot use '<{{.Name}}/>' because '{{.Name}}' is not a function (got {{.Got}})",
		Hints:    []string{"Components must be functions. Check that '{{.Name}}' is exported as a function."},
	},

	// ========================================
	// toDict errors (TODICT-0xxx)
	// ========================================
	"TODICT-0001": {
		Class:    ClassType,
		Template: "The toDict function requires array of [key, value] pairs",
	},
	"TODICT-0002": {
		Class:    ClassType,
		Template: "Dictionary keys must be strings, got {{.Got}}",
	},
	"TODICT-0003": {
		Class:    ClassType,
		Template: "The toDict function: unsupported value type {{.Got}}",
	},

	// ========================================
	// Map/filter callback errors (CALLBACK-0xxx)
	// ========================================
	"CALLBACK-0001": {
		Class:    ClassArity,
		Template: "Function passed to `{{.Function}}` must take exactly {{.Expected}} parameter(s), got {{.Got}}",
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
		Template: "Directory handle has no valid path",
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
		Template: "Unknown format: {{.Format}}",
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
		Template: "Spread operator requires a dictionary, got {{.Got}}",
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
		Template: "Query object missing 'sql' property",
	},
	"SQL-0003": {
		Class:    ClassType,
		Template: "The sql property must be a string, got {{.Got}}",
	},
	"SQL-0004": {
		Class:    ClassType,
		Template: "Query must be a string or <SQL> tag, got {{.Got}}",
	},

	// ========================================
	// HTTP errors (HTTP-0xxx)
	// ========================================
	"HTTP-0001": {
		Class:    ClassState,
		Template: "Request handle has no valid URL",
	},
	"HTTP-0002": {
		Class:    ClassFormat,
		Template: "Failed to encode request body: {{.GoError}}",
	},
	"HTTP-0003": {
		Class:    ClassNetwork,
		Template: "Failed to create request: {{.GoError}}",
	},
	"HTTP-0004": {
		Class:    ClassNetwork,
		Template: "Fetch failed: {{.GoError}}",
	},
	"HTTP-0005": {
		Class:    ClassIO,
		Template: "Failed to read response: {{.GoError}}",
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
		Template: "Failed to read from stdin: {{.GoError}}",
	},
	"STDIO-0004": {
		Class:    ClassIO,
		Template: "Cannot read from {{.Stream}}",
	},

	// ========================================
	// SFTP read errors (SFTP-0006+)
	// ========================================
	"SFTP-0006": {
		Class:    ClassIO,
		Template: "Failed to list directory: {{.GoError}}",
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
