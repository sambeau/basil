---
id: FEAT-112
title: "Unified Help System"
status: draft
priority: high
created: 2025-01-15
author: "@human"
blocking: false
---

# FEAT-112: Unified Help System

## Summary
Create a unified help system accessible via both CLI (`pars describe <topic>`) and REPL (`:describe <topic>`). This provides consistent, reliable documentation for types, modules, builtins, and operators — essential for language discoverability and AI assistance.

## User Story
As a Parsley developer, I want to look up type methods, module exports, and builtin functions from the command line or REPL so that I can learn the language without leaving my workflow.

As an AI coding assistant, I want a consistent interface to query Parsley's capabilities so that I can provide accurate help to users.

## Acceptance Criteria

### CLI
- [ ] `pars describe <topic>` displays help for the given topic
- [ ] `pars describe string` shows string methods
- [ ] `pars describe @std/math` shows module exports
- [ ] `pars describe builtins` lists all builtins by category
- [ ] `pars describe operators` lists all operators
- [ ] `pars describe JSON` shows help for specific builtin
- [ ] `pars describe --json <topic>` outputs machine-readable JSON (optional, for AI tooling)
- [ ] Unknown topic produces helpful error message

### REPL
- [ ] `:describe <topic>` works identically to CLI
- [ ] Output is formatted for terminal width
- [ ] `:describe` with no argument shows usage help

### Output Quality
- [ ] Output is consistent between CLI and REPL (identical content)
- [ ] Methods show arity and description
- [ ] Builtins show parameters and category
- [ ] Module exports show type and description

## Design Decisions

- **Topic-based, not expression-based**: `:describe string` queries the type, not a variable named `string`. This avoids the current `describe()` builtin's confusion.

- **Separate from `describe()` builtin**: The builtin remains for runtime introspection (examining actual values). Help is for static documentation.

- **Single source of truth**: Help engine reads from the declarative method registry (FEAT-111), ensuring documentation matches implementation.

- **REPL command prefix**: Use `:describe` (colon prefix) to distinguish from Parsley expressions, consistent with other REPL commands.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
| Component | Location | Changes |
|-----------|----------|---------|
| Help engine | `pkg/parsley/help/help.go` (new) | Core help lookup and formatting |
| CLI | `cmd/pars/main.go` | Add `describe` subcommand |
| REPL | `pkg/parsley/repl/repl.go` | Add `:describe` command handler |

### Dependencies
- **Depends on**: FEAT-111 (Declarative Method Registry) — help engine reads from registry
- **Blocks**: None

### Topics to Support

| Topic | Description | Data Source |
|-------|-------------|-------------|
| Type names | Methods and properties | `TypeRegistry` from FEAT-111 |
| Module paths | Exports from stdlib | Module metadata |
| `builtins` | All builtins by category | `BuiltinRegistry` |
| `operators` | All operators with descriptions | Operator metadata |
| Builtin name | Specific builtin details | `BuiltinRegistry` |
| `types` | List all types | `TypeRegistry` keys |

---

## Implementation Plan

### Phase 1: Help Engine Core

**New file: `pkg/parsley/help/help.go`**

```go
package help

import (
    "fmt"
    "strings"
    "github.com/sambeau/basil/pkg/parsley/evaluator"
)

// TopicResult represents help output
type TopicResult struct {
    Kind        string            // "type", "module", "builtin", "category"
    Name        string
    Description string
    Methods     []MethodHelp      // For types
    Properties  []PropertyHelp    // For types
    Exports     []ExportHelp      // For modules
    Params      []ParamHelp       // For builtins
}

// DescribeTopic returns help for the given topic
func DescribeTopic(topic string) (*TopicResult, error) {
    // Check if it's a type name
    if methods, ok := evaluator.TypeRegistry[topic]; ok {
        return describeType(topic, methods)
    }
    
    // Check if it's a module path (@std/math, etc.)
    if strings.HasPrefix(topic, "@std/") || strings.HasPrefix(topic, "@basil") {
        return describeModule(topic)
    }
    
    // Check special topics
    switch topic {
    case "builtins":
        return describeBuiltins()
    case "operators":
        return describeOperators()
    case "types":
        return describeTypes()
    }
    
    // Check if it's a specific builtin name
    if info, ok := evaluator.BuiltinRegistry[topic]; ok {
        return describeBuiltin(topic, info)
    }
    
    return nil, fmt.Errorf("unknown topic: %s\nTry: pars describe types", topic)
}

// FormatText formats TopicResult for terminal output
func FormatText(result *TopicResult, width int) string {
    // ... formatting logic
}

// FormatJSON formats TopicResult as JSON
func FormatJSON(result *TopicResult) ([]byte, error) {
    // ... JSON marshaling
}
```

### Phase 2: CLI Integration

**In `cmd/pars/main.go`:**

```go
func main() {
    if len(os.Args) > 1 {
        switch os.Args[1] {
        case "fmt":
            fmtCommand(os.Args[2:])
            return
        case "describe":
            describeCommand(os.Args[2:])
            return
        }
    }
    // ... rest of main
}

func describeCommand(args []string) {
    if len(args) == 0 {
        fmt.Println("Usage: pars describe <topic>")
        fmt.Println("Topics: string, array, @std/math, builtins, operators, types")
        os.Exit(1)
    }
    
    // Parse flags
    jsonOutput := false
    topic := args[0]
    for _, arg := range args {
        if arg == "--json" {
            jsonOutput = true
        } else {
            topic = arg
        }
    }
    
    result, err := help.DescribeTopic(topic)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
    
    if jsonOutput {
        data, _ := help.FormatJSON(result)
        fmt.Println(string(data))
    } else {
        fmt.Println(help.FormatText(result, 80))
    }
}
```

### Phase 3: REPL Integration

**In `pkg/parsley/repl/repl.go`:**

```go
func (r *Repl) handleCommand(cmd string) bool {
    parts := strings.Fields(cmd)
    if len(parts) == 0 {
        return false
    }
    
    switch parts[0] {
    case ":describe":
        if len(parts) < 2 {
            r.out.WriteString("Usage: :describe <topic>\n")
            r.out.WriteString("Topics: string, array, @std/math, builtins, operators\n")
            return true
        }
        result, err := help.DescribeTopic(parts[1])
        if err != nil {
            r.out.WriteString(fmt.Sprintf("Error: %v\n", err))
            return true
        }
        r.out.WriteString(help.FormatText(result, r.termWidth))
        return true
    
    // ... other commands
    }
    return false
}
```

---

## Output Examples

### Type Help
```
$ pars describe string

Type: string

Methods:
  .toUpper()              Convert to uppercase
  .toLower()              Convert to lowercase
  .trim()                 Remove leading/trailing whitespace
  .trimLeft()             Remove leading whitespace
  .trimRight()            Remove trailing whitespace
  .split(delim)           Split by delimiter into array
  .replace(old, new)      Replace occurrences of old with new
  .contains(substr)       Check if string contains substring
  .startsWith(prefix)     Check if string starts with prefix
  .endsWith(suffix)       Check if string ends with suffix
  .length()               Return string length
  .charAt(index)          Return character at index
  .substring(start, end?) Extract substring
  .padLeft(len, char?)    Pad left to length
  .padRight(len, char?)   Pad right to length
```

### Module Help
```
$ pars describe @std/math

Module: @std/math

Exports:
  pi              constant    Mathematical constant π (3.14159...)
  e               constant    Mathematical constant e (2.71828...)
  abs(x)          function    Absolute value
  sqrt(x)         function    Square root
  pow(base, exp)  function    Raise base to exponent
  sin(x)          function    Sine (radians)
  cos(x)          function    Cosine (radians)
  tan(x)          function    Tangent (radians)
  floor(x)        function    Round down to integer
  ceil(x)         function    Round up to integer
  round(x)        function    Round to nearest integer
  min(a, b, ...)  function    Minimum value
  max(a, b, ...)  function    Maximum value
```

### Builtins by Category
```
$ pars describe builtins

Builtin Functions:

  File/Data:
    JSON(source)            Parse JSON from string or file
    YAML(source)            Parse YAML from string or file
    CSV(source, opts?)      Parse CSV from string or file
    text(source)            Read text from file
    lines(source)           Read lines from file as array
    bytes(source)           Read bytes from file

  Time:
    now()                   Current datetime
    date(str)               Parse date string
    datetime(str)           Parse datetime string

  Conversion:
    toInt(value)            Convert to integer
    toFloat(value)          Convert to float
    toString(value)         Convert to string
    toArray(value)          Convert to array
    toDict(value)           Convert to dictionary

  Output:
    print(value)            Print without newline
    println(value)          Print with newline
    printf(fmt, args...)    Formatted print
    log(value)              Log to stderr
    fail(message)           Fail with error

  Introspection:
    inspect(value)          Inspect value structure
    describe(value)         Describe value (use :describe for types)
    builtins()              List all builtins
```

### JSON Output (for AI/tooling)
```
$ pars describe string --json
{
  "kind": "type",
  "name": "string",
  "methods": [
    {"name": "toUpper", "arity": "0", "description": "Convert to uppercase"},
    {"name": "toLower", "arity": "0", "description": "Convert to lowercase"},
    ...
  ],
  "properties": []
}
```

---

## Test Plan

| Test | Expected |
|------|----------|
| `pars describe string` | Shows string methods |
| `pars describe array` | Shows array methods |
| `pars describe @std/math` | Shows module exports |
| `pars describe builtins` | Lists all builtins by category |
| `pars describe operators` | Lists all operators |
| `pars describe JSON` | Shows JSON builtin details |
| `pars describe unknown` | Error with suggestion |
| `pars describe --json string` | Valid JSON output |
| REPL `:describe string` | Same output as CLI |
| REPL `:describe` | Usage help |

---

## Future Enhancements (Out of Scope)

- `pars introspect --json` — dump entire language catalog for AI context
- Interactive help browser in REPL
- Hyperlinks in terminal (where supported)
- Examples in help output
- Search across all topics

## Implementation Notes
*To be added during implementation*

## Related
- Report: `work/reports/PARSLEY-1.0-ALPHA-READINESS.md` (Section 2, Section 3.3)
- Depends on: FEAT-111 (Declarative Method Registry)
- Related: FEAT-110 (Introspection Validation Tests)
- Current implementation: `pkg/parsley/evaluator/introspect.go`
