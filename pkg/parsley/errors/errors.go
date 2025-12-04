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
)

// ParsleyError represents any error from parsing or evaluation.
type ParsleyError struct {
	Class   ErrorClass     `json:"class"`             // Error category
	Code    string         `json:"code"`              // Error code (e.g., "TYPE-0001")
	Message string         `json:"message"`           // Human-readable message
	Hints   []string       `json:"hints,omitempty"`   // Suggestions for fixing
	Line    int            `json:"line"`              // 1-based line (0 if unknown)
	Column  int            `json:"column"`            // 1-based column (0 if unknown)
	File    string         `json:"file,omitempty"`    // File path (if known)
	Data    map[string]any `json:"data,omitempty"`    // Template variables
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
	// Parse errors (PARSE-0xxx)
	"PARSE-0001": {
		Class:    ClassParse,
		Template: "expected {{.Expected}}, got '{{.Got}}'",
	},
	"PARSE-0002": {
		Class:    ClassParse,
		Template: "unexpected token '{{.Token}}'",
	},
	"PARSE-0003": {
		Class:    ClassParse,
		Template: "for (var in array) requires a { } block body, not an expression",
		Hints:    []string{"for {{.Var}} in {{.Array}} { ... }"},
	},

	// Type errors (TYPE-0xxx)
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

	// Arity errors (ARITY-0xxx)
	"ARITY-0001": {
		Class:    ClassArity,
		Template: "wrong number of arguments to `{{.Function}}`. got={{.Got}}, want={{.Want}}",
	},

	// Undefined errors (UNDEF-0xxx)
	"UNDEF-0001": {
		Class:    ClassUndefined,
		Template: "identifier not found: {{.Name}}",
		Hints:    []string{}, // "Did you mean `{{.Suggestion}}`?" added dynamically
	},
	"UNDEF-0002": {
		Class:    ClassUndefined,
		Template: "unknown method '{{.Method}}' for {{.Type}}",
	},
	"UNDEF-0003": {
		Class:    ClassUndefined,
		Template: "undefined component: {{.Name}}",
	},

	// I/O errors (IO-0xxx)
	"IO-0001": {
		Class:    ClassIO,
		Template: "failed to {{.Operation}} '{{.Path}}': {{.GoError}}",
	},
	"IO-0002": {
		Class:    ClassIO,
		Template: "module not found: {{.Path}}",
	},

	// Database errors (DB-0xxx)
	"DB-0001": {
		Class:    ClassDatabase,
		Template: "{{.Driver}} {{.Operation}} failed: {{.GoError}}",
	},
	"DB-0002": {
		Class:    ClassDatabase,
		Template: "query failed: {{.GoError}}",
	},

	// Network errors (NET-0xxx)
	"NET-0001": {
		Class:    ClassNetwork,
		Template: "{{.Operation}} to {{.URL}} failed: {{.GoError}}",
	},
	"NET-0002": {
		Class:    ClassNetwork,
		Template: "HTTP request failed: {{.GoError}}",
	},

	// Security errors (SEC-0xxx)
	"SEC-0001": {
		Class:    ClassSecurity,
		Template: "security: {{.Operation}} access denied",
		Hints:    []string{"use {{.Flag}} to allow this operation"},
	},

	// Index errors (INDEX-0xxx)
	"INDEX-0001": {
		Class:    ClassIndex,
		Template: "index {{.Index}} out of range (length {{.Length}})",
	},
	"INDEX-0002": {
		Class:    ClassIndex,
		Template: "cannot {{.Operation}} from empty {{.Type}}",
	},

	// Format errors (FMT-0xxx)
	"FMT-0001": {
		Class:    ClassFormat,
		Template: "invalid {{.Format}}: {{.GoError}}",
	},

	// Operator errors (OP-0xxx)
	"OP-0001": {
		Class:    ClassOperator,
		Template: "unknown operator: {{.LeftType}} {{.Operator}} {{.RightType}}",
	},
	"OP-0002": {
		Class:    ClassOperator,
		Template: "division by zero",
	},

	// State errors (STATE-0xxx)
	"STATE-0001": {
		Class:    ClassState,
		Template: "{{.Resource}} is {{.ActualState}}, expected {{.ExpectedState}}",
	},

	// Import errors (IMPORT-0xxx)
	"IMPORT-0001": {
		Class:    ClassImport,
		Template: "in module {{.ModulePath}}: {{.NestedError}}",
	},
	"IMPORT-0002": {
		Class:    ClassImport,
		Template: "circular dependency detected: {{.Chain}}",
	},
}

// New creates a ParsleyError from the catalog.
// If the code is not found, creates a generic error with the message.
func New(code string, data map[string]any) *ParsleyError {
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
