# Parsley Error Codes Reference

This document provides a complete reference of all error codes used in Parsley.

## Error Code Format

Error codes follow the pattern `PREFIX-NNNN` where:
- `PREFIX` identifies the error category (e.g., `TYPE`, `ARITY`, `IO`)
- `NNNN` is a 4-digit number within that category

## Error Classes

Each error belongs to a class that categorizes its nature:

| Class | Description |
|-------|-------------|
| `parse` | Parser/syntax errors |
| `type` | Type mismatches |
| `arity` | Wrong argument count |
| `undefined` | Not found/defined |
| `io` | File operations |
| `database` | Database operations |
| `network` | HTTP, SSH, SFTP |
| `security` | Access denied |
| `index` | Out of bounds |
| `format` | Invalid format/parse |
| `operator` | Invalid operations |
| `state` | Invalid state |
| `import` | Module loading |
| `value` | Invalid value |

---

## Type Errors (TYPE-0xxx)

Type mismatch and type conversion errors.

| Code | Template | Description |
|------|----------|-------------|
| TYPE-0001 | `{{.Function}} expected {{.Expected}}, got {{.Got}}` | General type mismatch |
| TYPE-0002 | `{{.Function}} received {{.Got}}, expected {{.Expected}}` | Argument type error |
| TYPE-0003 | `argument {{.ArgNum}} to {{.Function}} must be {{.Expected}}, got {{.Got}}` | Positional argument type |
| TYPE-0004 | `for expects a function or builtin, got {{.Got}}` | For loop function type |
| TYPE-0005 | `first argument to {{.Function}} must be {{.Expected}}, got {{.Got}}` | First argument type |
| TYPE-0006 | `second argument to {{.Function}} must be {{.Expected}}, got {{.Got}}` | Second argument type |
| TYPE-0007 | `cannot iterate over {{.Got}}` | Non-iterable type |
| TYPE-0008 | `cannot index {{.Got}} with {{.IndexType}}` | Invalid index type |
| TYPE-0009 | `comparison function must return boolean, got {{.Got}}` | Sort comparator return |
| TYPE-0010 | `{{.Function}} callback must be a function, got {{.Got}}` | Callback type |
| TYPE-0011 | `{{.Function}} element must be {{.Expected}}, got {{.Got}}` | Array element type |
| TYPE-0012 | `argument to {{.Function}} must be {{.Expected}}, got {{.Got}}` | Single argument type |
| TYPE-0013 | `index operator not supported: {{.Left}}[{{.Right}}]` | Invalid indexing |
| TYPE-0014 | `slice operator not supported: {{.Type}}` | Invalid slicing |
| TYPE-0015 | `cannot convert '{{.Value}}' to integer` | Integer conversion |
| TYPE-0016 | `cannot convert '{{.Value}}' to float` | Float conversion |
| TYPE-0017 | `cannot convert '{{.Value}}' to number` | Number conversion |
| TYPE-0018 | `slice {{.Position}} index must be an integer, got {{.Got}}` | Slice index type |
| TYPE-0019 | `{{.Function}} element at index {{.Index}} must be {{.Expected}}, got {{.Got}}` | Array element at index |
| TYPE-0020 | `{{.Context}} must be {{.Expected}}, got {{.Got}}` | Context-specific type |
| TYPE-0021 | `'{{.Name}}' is not a function` | Not callable |
| TYPE-0022 | `dot notation can only be used on dictionaries, got {{.Got}}` | Dot notation type |

---

## Arity Errors (ARITY-0xxx)

Wrong number of arguments.

| Code | Template | Description |
|------|----------|-------------|
| ARITY-0001 | `wrong number of arguments to {{.Function}}. got={{.Got}}, want={{.Want}}` | General arity |
| ARITY-0002 | `{{.Function}} expects {{.Want}} argument(s), got {{.Got}}` | Expected arguments |
| ARITY-0003 | `comparison function must take exactly 2 parameters, got {{.Got}}` | Sort comparator |
| ARITY-0004 | `wrong number of arguments. got={{.Got}}, want={{.Min}}-{{.Max}}` | Range arity |
| ARITY-0005 | `{{.Function}} requires at least {{.Min}} argument(s), got {{.Got}}` | Minimum arguments |
| ARITY-0006 | `{{.Function}} requires exactly {{.Want}} argument(s), got {{.Got}}` | Exact arguments |

---

## Operator Errors (OP-0xxx)

Invalid operator usage.

| Code | Template | Description |
|------|----------|-------------|
| OP-0001 | `unknown operator: {{.LeftType}} {{.Operator}} {{.RightType}}` | Unknown infix operator |
| OP-0002 | `division by zero` | Division by zero |
| OP-0003 | `cannot compare {{.LeftType}} and {{.RightType}}` | Incompatible comparison |
| OP-0004 | `cannot negate {{.Type}}` | Invalid negation |
| OP-0005 | `unknown prefix operator: {{.Operator}}{{.Type}}` | Unknown prefix operator |
| OP-0006 | `modulo by zero` | Modulo by zero |
| OP-0007 | `left operand of {{.Operator}} must be {{.Expected}}, got {{.Got}}` | Left operand type |
| OP-0008 | `right operand of {{.Operator}} must be {{.Expected}}, got {{.Got}}` | Right operand type |
| OP-0009 | `type mismatch: {{.LeftType}} {{.Operator}} {{.RightType}}` | Operand type mismatch |
| OP-0010 | `unsupported type for mixed arithmetic: {{.Type}}` | Mixed arithmetic type |
| OP-0011 | `cannot add duration to datetime` | Duration + datetime order |
| OP-0012 | `cannot intersect two {{.Kind}}s - {{.Hint}}` | Datetime intersection |
| OP-0013 | `cannot compare durations with month components` | Duration month comparison |
| OP-0014 | `unknown operator for {{.Type}}: {{.Operator}}` | Unknown type operator |
| OP-0015 | `unknown operator for {{.LeftType}} and {{.RightType}}: {{.Operator}}` | Unknown typed operator |
| OP-0016 | `'in' operator requires array, dictionary, or string, got {{.Got}}` | Invalid 'in' right side |
| OP-0017 | `dictionary key must be a string, got {{.Got}}` | Dict key type for 'in' |
| OP-0018 | `substring must be a string, got {{.Got}}` | String 'in' operand |

---

## Validation Errors (VAL-0xxx)

Invalid values or constraints.

| Code | Template | Description |
|------|----------|-------------|
| VAL-0001 | `{{.Function}} requires a non-empty {{.Type}}` | Empty value |
| VAL-0002 | `{{.Function}} value cannot be negative: {{.Value}}` | Negative value |
| VAL-0003 | `{{.Function}}: {{.Field}} must be {{.Constraint}}` | Field constraint |
| VAL-0004 | `{{.Function}}: invalid {{.Field}} '{{.Value}}'` | Invalid field value |
| VAL-0005 | `{{.Function}}: {{.Field}} out of range ({{.Min}}-{{.Max}})` | Out of range |
| VAL-0006 | `{{.Function}}: count must be non-negative, got {{.Got}}` | Negative count |
| VAL-0007 | `{{.Function}}: precision must be non-negative, got {{.Got}}` | Negative precision |
| VAL-0008 | `{{.Type}} handle has no valid path` | Invalid handle path |
| VAL-0009 | `{{.Function}}: invalid route '{{.Route}}'` | Invalid route |
| VAL-0010 | `orderBy column spec must have {{.Min}}-{{.Max}} elements, got {{.Got}}` | orderBy spec length |
| VAL-0011 | `{{.Function}} requires at least one column` | Missing columns |
| VAL-0012 | `chunk size must be > 0, got {{.Got}}` | Invalid chunk size |
| VAL-0013 | `range start must be an integer, got {{.Got}}` | Range start type |
| VAL-0014 | `range end must be an integer, got {{.Got}}` | Range end type |
| VAL-0015 | `regex dictionary missing pattern field` | Missing regex pattern |
| VAL-0016 | `regex pattern must be a string, got {{.Got}}` | Regex pattern type |
| VAL-0017 | `{{.Type}} dictionary missing {{.Field}} field` | Missing dict field |
| VAL-0018 | `{{.Field}} field must be {{.Expected}}, got {{.Got}}` | Dict field type |

---

## Index Errors (INDEX-0xxx)

Array/string bounds errors.

| Code | Template | Description |
|------|----------|-------------|
| INDEX-0001 | `index {{.Index}} out of range (length {{.Length}})` | Index out of bounds |
| INDEX-0002 | `negative index not allowed` | Negative index |
| INDEX-0003 | `slice bounds out of range` | Slice bounds |
| INDEX-0004 | `key '{{.Key}}' not found in dictionary` | Missing dict key |
| INDEX-0005 | `column '{{.Column}}' not found in table` | Missing table column |

---

## I/O Errors (IO-0xxx)

File operation errors.

| Code | Template | Description |
|------|----------|-------------|
| IO-0001 | `failed to {{.Operation}} '{{.Path}}': {{.GoError}}` | General I/O error |
| IO-0002 | `module not found: {{.Path}}` | Module not found |
| IO-0003 | `failed to read file '{{.Path}}': {{.GoError}}` | Read error |
| IO-0004 | `failed to write file '{{.Path}}': {{.GoError}}` | Write error |
| IO-0005 | `failed to delete '{{.Path}}': {{.GoError}}` | Delete error |
| IO-0006 | `failed to create directory '{{.Path}}': {{.GoError}}` | Mkdir error |
| IO-0007 | `failed to resolve path '{{.Path}}': {{.GoError}}` | Path resolution |
| IO-0008 | `failed to create directory '{{.Path}}': {{.GoError}}` | Directory creation |
| IO-0009 | `failed to remove directory '{{.Path}}': {{.GoError}}` | Directory removal |
| IO-0010 | `failed to list directory '{{.Path}}': {{.GoError}}` | Directory listing |

---

## Format Errors (FMT-0xxx)

Parsing and formatting errors.

| Code | Template | Description |
|------|----------|-------------|
| FMT-0001 | `invalid {{.Format}}: {{.GoError}}` | General format error |
| FMT-0002 | `invalid regex pattern: {{.GoError}}` | Regex compile error |
| FMT-0003 | `invalid URL: {{.GoError}}` | URL parse error |
| FMT-0004 | `invalid datetime: {{.GoError}}` | Datetime parse error |
| FMT-0005 | `invalid number: {{.Value}}` | Number parse error |
| FMT-0006 | `invalid JSON: {{.GoError}}` | JSON parse error |
| FMT-0007 | `invalid time format: {{.Format}}` | Time format error |
| FMT-0008 | `invalid locale: {{.Locale}}` | Locale error |
| FMT-0009 | `invalid duration: {{.GoError}}` | Duration parse error |
| FMT-0010 | `invalid base for conversion: {{.Base}}` | Base conversion error |

---

## Undefined Errors (UNDEF-0xxx)

Not found errors.

| Code | Template | Description |
|------|----------|-------------|
| UNDEF-0001 | `identifier not found: {{.Name}}` | Unknown identifier |
| UNDEF-0002 | `unknown method '{{.Method}}' for type {{.Type}}` | Unknown method |
| UNDEF-0003 | `undefined component: {{.Name}}` | Unknown component |
| UNDEF-0004 | `unknown property '{{.Property}}' on {{.Type}}` | Unknown property |
| UNDEF-0005 | `unknown standard library module: @std/{{.Module}}` | Unknown stdlib module |
| UNDEF-0006 | `module does not export '{{.Name}}'` | Missing export |

---

## Security Errors (SEC-0xxx)

Security and access errors.

| Code | Template | Description |
|------|----------|-------------|
| SEC-0001 | `{{.Operation}} access denied by security policy` | Operation denied |
| SEC-0002 | `path access denied: {{.Path}}` | Path denied |
| SEC-0003 | `network access denied: {{.Host}}` | Network denied |
| SEC-0004 | `command execution denied: {{.Command}}` | Command denied |
| SEC-0005 | `security policy not configured` | No policy |
| SEC-0006 | `SFTP authentication failed: {{.GoError}}` | SFTP auth error |

---

## Database Errors (DB-0xxx)

Database operation errors.

| Code | Template | Description |
|------|----------|-------------|
| DB-0001 | `database error: {{.GoError}}` | General DB error |
| DB-0002 | `failed to connect to database: {{.GoError}}` | Connection error |
| DB-0003 | `failed to execute query: {{.GoError}}` | Query error |
| DB-0004 | `transaction error: {{.GoError}}` | Transaction error |
| DB-0005 | `database is closed` | DB closed |
| DB-0006 | `unsupported database driver: {{.Driver}}` | Unknown driver |
| DB-0007 | `failed to prepare statement: {{.GoError}}` | Prepare error |
| DB-0008 | `no rows returned` | No rows |
| DB-0009 | `database not connected` | Not connected |
| DB-0010 | `query returned multiple rows, expected one` | Multiple rows |
| DB-0011 | `failed to begin transaction: {{.GoError}}` | Begin TX error |
| DB-0012 | `failed to scan row: {{.GoError}}` | Scan error |

---

## Network Errors (NET-0xxx)

HTTP and network errors.

| Code | Template | Description |
|------|----------|-------------|
| NET-0001 | `network error: {{.GoError}}` | General network error |
| NET-0002 | `HTTP request failed: {{.GoError}}` | HTTP request error |
| NET-0003 | `HTTP {{.StatusCode}}: {{.Status}}` | HTTP status error |
| NET-0004 | `failed to parse response: {{.GoError}}` | Response parse error |
| NET-0005 | `SSH connection failed: {{.GoError}}` | SSH connect error |
| NET-0006 | `SSH authentication failed` | SSH auth error |
| NET-0007 | `SFTP session failed: {{.GoError}}` | SFTP session error |
| NET-0008 | `SFTP operation failed: {{.GoError}}` | SFTP operation error |
| NET-0009 | `SSH/SFTP not connected` | Not connected |

---

## Import Errors (IMPORT-0xxx)

Module loading errors.

| Code | Template | Description |
|------|----------|-------------|
| IMPORT-0001 | `in module {{.ModulePath}}: {{.NestedError}}` | Nested module error |
| IMPORT-0002 | `circular dependency detected: {{.Path}}` | Circular import |
| IMPORT-0003 | `parse errors in module {{.ModulePath}}` | Module parse error |
| IMPORT-0004 | `failed to resolve module path: {{.GoError}}` | Path resolution error |
| IMPORT-0005 | `in module {{.ModulePath}}: line {{.Line}}, column {{.Column}}: {{.NestedError}}` | Detailed nested error |

---

## Parse Errors (PARSE-0xxx)

Syntax errors.

| Code | Template | Description |
|------|----------|-------------|
| PARSE-0001 | `unexpected token: {{.Got}}` | Unexpected token |
| PARSE-0002 | `expected {{.Expected}}, got {{.Got}}` | Expected token |
| PARSE-0003 | `unterminated string` | Unclosed string |
| PARSE-0004 | `invalid number: {{.Value}}` | Invalid number |
| PARSE-0005 | `unexpected end of input` | Unexpected EOF |
| PARSE-0006 | `invalid escape sequence: {{.Sequence}}` | Bad escape |
| PARSE-0007 | `invalid regex literal: {{.Literal}}` | Bad regex |
| PARSE-0008 | `invalid attribute: {{.Name}}` | Bad attribute |
| PARSE-0009 | `invalid template expression: {{.GoError}}` | Template error |
| PARSE-0010 | `unclosed template expression` | Unclosed template |
| PARSE-0011 | `template syntax error: {{.GoError}}` | Template syntax |

---

## State Errors (STATE-0xxx)

Invalid state errors.

| Code | Template | Description |
|------|----------|-------------|
| STATE-0001 | `{{.Resource}} is {{.ActualState}}, expected {{.ExpectedState}}` | Wrong state |
| STATE-0002 | `SFTP connection is not connected` | SFTP not connected |
| STATE-0003 | `file handle is closed` | Handle closed |

---

## Loop Errors (LOOP-0xxx)

For loop errors.

| Code | Template | Description |
|------|----------|-------------|
| LOOP-0001 | `for expects function or builtin as callback` | For callback type |
| LOOP-0002 | `for callback must be a function` | Callback not function |
| LOOP-0003 | `for requires an iterable (array, string, range)` | Not iterable |
| LOOP-0004 | `for-in requires identifier` | Missing identifier |
| LOOP-0005 | `for-in body must be a block` | Missing block |
| LOOP-0006 | `break used outside of loop` | Break outside loop |
| LOOP-0007 | `continue used outside of loop` | Continue outside loop |

---

## Command Errors (CMD-0xxx)

Command execution errors.

| Code | Template | Description |
|------|----------|-------------|
| CMD-0001 | `command handle missing {{.Field}} field` | Missing field |
| CMD-0002 | `command execution failed: {{.GoError}}` | Exec error |
| CMD-0003 | `command timed out after {{.Timeout}}` | Timeout |
| CMD-0004 | `command exited with status {{.ExitCode}}` | Non-zero exit |
| CMD-0005 | `invalid command: {{.Command}}` | Invalid command |
| CMD-0006 | `command not found: {{.Command}}` | Command not found |

---

## Internal Errors (INTERNAL-0xxx)

Internal/unexpected errors.

| Code | Template | Description |
|------|----------|-------------|
| INTERNAL-0001 | `internal error: {{.Message}}` | General internal |
| INTERNAL-0002 | `unexpected nil value` | Nil value |
| INTERNAL-0003 | `{{.Function}} failed: {{.GoError}}` | Function failure |

---

## Additional Error Categories

### SFTP Errors (SFTP-0xxx)
For SFTP-specific operations.

### HTTP Errors (HTTP-0xxx)
For HTTP client operations.

### SQL Errors (SQL-0xxx)
For SQL-specific operations.

### STDIO Errors (STDIO-0xxx)
For standard I/O operations.

### Call Errors (CALL-0xxx)
For function call errors.

### Callback Errors (CALLBACK-0xxx)
For callback function errors.

### Component Errors (COMP-0xxx)
For component/tag errors.

### Destructuring Errors (DEST-0xxx)
For destructuring pattern errors.

### File Operation Errors (FILEOP-0xxx)
For file/directory operations.

### Spread Errors (SPREAD-0xxx)
For spread operator errors.

### ToDict Errors (TODICT-0xxx)
For dictionary conversion errors.

---

## Using Error Codes in Code

### Creating Errors

```go
import perrors "github.com/sambeau/basil/pkg/parsley/errors"

// Create a type error
err := perrors.New("TYPE-0001", map[string]any{
    "Function": "len",
    "Expected": "array or string",
    "Got":      "integer",
})

// Create an arity error
err := perrors.New("ARITY-0001", map[string]any{
    "Function": "push",
    "Got":      3,
    "Want":     2,
})
```

### Checking Error Types

```go
if err.Class == perrors.ClassType {
    // Handle type errors
}

if err.Code == "TYPE-0001" {
    // Handle specific error
}
```

### Accessing Error Data

```go
if fn, ok := err.Data["Function"].(string); ok {
    fmt.Printf("Error in function: %s\n", fn)
}
```
