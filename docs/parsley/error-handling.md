# Parsley Error Handling Guide

This guide explains how to handle errors when integrating the Parsley language into your application.

## Overview

Parsley uses structured errors that provide rich information for error handling, display, and recovery. All errors implement the standard Go `error` interface while also providing additional metadata.

## Error Structure

Every Parsley error contains:

```go
type ParsleyError struct {
    Class   ErrorClass     // Category: type, arity, io, etc.
    Code    string         // Unique code: "TYPE-0001", "IO-0003"
    Message string         // Human-readable message
    Hints   []string       // Suggestions for fixing
    Line    int            // Source line (1-based, 0 if unknown)
    Column  int            // Source column (1-based, 0 if unknown)
    File    string         // Source file path (if known)
    Data    map[string]any // Structured data for templating
}
```

## Quick Start

### Basic Usage

```go
import (
    "github.com/sambeau/basil/pkg/parsley/parsley"
)

func main() {
    result := parsley.Eval(`len(42)`)  // Wrong type!
    
    if result.Error != nil {
        fmt.Println(result.Error.Message)
        // Output: len expected array or string, got integer
    }
}
```

### Handling Different Error Types

```go
import (
    perrors "github.com/sambeau/basil/pkg/parsley/errors"
)

result := parsley.Eval(code)
if err := result.Error; err != nil {
    switch err.Class {
    case perrors.ClassType:
        // Type mismatch - show expected vs got
        fmt.Printf("Type error: expected %v, got %v\n", 
            err.Data["Expected"], err.Data["Got"])
    
    case perrors.ClassArity:
        // Wrong number of arguments
        fmt.Printf("Function %s expects %v args, got %v\n",
            err.Data["Function"], err.Data["Want"], err.Data["Got"])
    
    case perrors.ClassIO:
        // File operation failed
        fmt.Printf("Failed to %s '%s'\n",
            err.Data["Operation"], err.Data["Path"])
    
    case perrors.ClassSecurity:
        // Security policy violation
        fmt.Printf("Access denied: %s\n", err.Message)
    
    default:
        fmt.Println(err.Message)
    }
}
```

## Error Classes

| Class | Use Case | Common Codes |
|-------|----------|--------------|
| `parse` | Syntax errors | PARSE-0001 to PARSE-0011 |
| `type` | Type mismatches | TYPE-0001 to TYPE-0022 |
| `arity` | Wrong argument count | ARITY-0001 to ARITY-0006 |
| `undefined` | Unknown identifiers | UNDEF-0001 to UNDEF-0006 |
| `io` | File operations | IO-0001 to IO-0010 |
| `database` | Database operations | DB-0001 to DB-0012 |
| `network` | HTTP/SSH/SFTP | NET-0001 to NET-0009 |
| `security` | Access denied | SEC-0001 to SEC-0006 |
| `index` | Out of bounds | INDEX-0001 to INDEX-0005 |
| `format` | Parse/format errors | FMT-0001 to FMT-0010 |
| `operator` | Invalid operators | OP-0001 to OP-0018 |
| `value` | Invalid values | VAL-0001 to VAL-0018 |

## Displaying Errors

### Simple String

```go
fmt.Println(err.String())
// Output: line 5, column 12: len expected array or string, got integer
```

### Pretty Multi-line

```go
fmt.Println(err.PrettyString())
// Output:
// Runtime error: line 5, column 12
//   len expected array or string, got integer
//   Try: len([1, 2, 3]) or len("hello")
```

### JSON Serialization

```go
jsonBytes, _ := err.ToJSON()
fmt.Println(string(jsonBytes))
// Output:
// {
//   "class": "type",
//   "code": "TYPE-0001",
//   "message": "len expected array or string, got integer",
//   "hints": ["Try: len([1, 2, 3]) or len(\"hello\")"],
//   "line": 5,
//   "column": 12,
//   "data": {
//     "Function": "len",
//     "Expected": "array or string",
//     "Got": "integer"
//   }
// }
```

## Programmatic Error Handling

### By Error Code

Handle specific errors by their unique code:

```go
switch err.Code {
case "TYPE-0001":
    // Function type mismatch
    suggestTypeConversion(err.Data["Got"], err.Data["Expected"])

case "UNDEF-0001":
    // Unknown identifier - suggest similar names
    suggestSimilarIdentifiers(err.Data["Name"])

case "IO-0003":
    // File read error - check permissions
    checkFilePermissions(err.Data["Path"])

case "SEC-0001":
    // Security violation - log and alert
    logSecurityViolation(err)
}
```

### By Error Class

Handle categories of errors:

```go
switch err.Class {
case perrors.ClassParse:
    showSyntaxHelp(err)

case perrors.ClassType, perrors.ClassArity:
    showFunctionSignature(err.Data["Function"])

case perrors.ClassIO, perrors.ClassDatabase, perrors.ClassNetwork:
    showRetryOption(err)

case perrors.ClassSecurity:
    showSecurityPolicy(err)
}
```

## Common Data Fields

Different error classes provide different data fields:

### Type Errors (TYPE-xxxx)
```go
err.Data["Function"]  // Function name
err.Data["Expected"]  // Expected type(s)
err.Data["Got"]       // Actual type
err.Data["ArgNum"]    // Argument position (for positional errors)
```

### I/O Errors (IO-xxxx)
```go
err.Data["Path"]      // File path
err.Data["Operation"] // Operation: read, write, delete
err.Data["GoError"]   // Underlying Go error message
```

### Network Errors (NET-xxxx)
```go
err.Data["URL"]        // Request URL
err.Data["StatusCode"] // HTTP status code
err.Data["Status"]     // HTTP status text
err.Data["GoError"]    // Underlying error
```

### Database Errors (DB-xxxx)
```go
err.Data["Driver"]    // Database driver
err.Data["Operation"] // Operation: query, exec, connect
err.Data["GoError"]   // Underlying error
```

## Custom Error Templates

You can use the `Data` field to create custom error messages:

```go
import "text/template"

tmpl := template.Must(template.New("error").Parse(
    `Error in {{.Function}}: expected {{.Expected}}, got {{.Got}}`))

var buf bytes.Buffer
tmpl.Execute(&buf, err.Data)
fmt.Println(buf.String())
```

## Localization

Error codes enable localization by mapping codes to translated templates:

```go
var translations = map[string]map[string]string{
    "TYPE-0001": {
        "en": "{{.Function}} expected {{.Expected}}, got {{.Got}}",
        "es": "{{.Function}} esperaba {{.Expected}}, obtuvo {{.Got}}",
        "de": "{{.Function}} erwartete {{.Expected}}, erhielt {{.Got}}",
    },
}

func localizeError(err *perrors.ParsleyError, lang string) string {
    if templates, ok := translations[err.Code]; ok {
        if tmplStr, ok := templates[lang]; ok {
            tmpl := template.Must(template.New("").Parse(tmplStr))
            var buf bytes.Buffer
            tmpl.Execute(&buf, err.Data)
            return buf.String()
        }
    }
    return err.Message // Fallback to default
}
```

## IDE/Editor Integration

Errors provide all information needed for IDE integration:

```go
type DiagnosticItem struct {
    Source   string `json:"source"`
    Code     string `json:"code"`
    Message  string `json:"message"`
    Severity string `json:"severity"`
    Range    struct {
        Line   int `json:"line"`
        Column int `json:"column"`
    } `json:"range"`
}

func toDiagnostic(err *perrors.ParsleyError) DiagnosticItem {
    severity := "error"
    if err.Class == perrors.ClassParse {
        severity = "error"
    }
    
    return DiagnosticItem{
        Source:   "parsley",
        Code:     err.Code,
        Message:  err.Message,
        Severity: severity,
        Range: struct {
            Line   int `json:"line"`
            Column int `json:"column"`
        }{
            Line:   err.Line,
            Column: err.Column,
        },
    }
}
```

## Error Recovery

### Using Hints

Errors often include hints for fixing the problem:

```go
if len(err.Hints) > 0 {
    fmt.Println("Suggestions:")
    for _, hint := range err.Hints {
        fmt.Printf("  • %s\n", hint)
    }
}
```

### Checking Recoverability

Some errors are recoverable (e.g., file not found), others are not (e.g., syntax errors):

```go
func isRecoverable(err *perrors.ParsleyError) bool {
    switch err.Class {
    case perrors.ClassIO, perrors.ClassNetwork, perrors.ClassDatabase:
        return true  // Can retry
    case perrors.ClassParse, perrors.ClassType, perrors.ClassArity:
        return false // Code needs fixing
    default:
        return false
    }
}
```

## Best Practices

1. **Check error class first** - Use `err.Class` for broad categorization
2. **Use error codes for specific handling** - Use `err.Code` for precise behavior
3. **Display hints to users** - Always show `err.Hints` when available
4. **Log structured data** - Use `err.ToJSON()` for structured logging
5. **Preserve line/column** - Show source location for debugging
6. **Handle security errors specially** - Never expose sensitive details

## Reference

For a complete list of all error codes, see [Error Codes Reference](error-codes.md).
