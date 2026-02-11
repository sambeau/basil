---
id: FEAT-113
title: "CLI Debug Flag for Expression Inspection"
status: draft
priority: medium
created: 2026-02-11
author: "@human"
---

# FEAT-113: CLI Debug Flag for Expression Inspection

## Summary
Add a debug/inspect flag to the `pars` CLI that outputs the PLN (Parsley Literal Notation) representation of an expression's result, making it easier to debug and inspect values without manually wrapping expressions in `log()`.

## User Story
As a developer debugging Parsley code, I want to quickly see the PLN representation of an expression's result so that I can understand its structure without having to wrap it in `let x = ...; log(x)`.

## Motivation
Currently, to inspect the structure of a value in Parsley, you need to:

```bash
go run ./cmd/pars -e 'let m = "hello world" ~ /(\w+)/; log(m)'
```

This is verbose for quick debugging. A simpler option would be:

```bash
go run ./cmd/pars -d '"hello world" ~ /(\w+)/'
# or
go run ./cmd/pars --inspect '"hello world" ~ /(\w+)/'
```

## Acceptance Criteria
- [ ] `pars -d "expression"` evaluates the expression and outputs PLN representation
- [ ] `pars --debug "expression"` (long form) works the same
- [ ] Output shows the full PLN structure (arrays as `[...]`, dictionaries as `{...}`, etc.)
- [ ] Works with `-e` flag: `pars -e "code" -d` outputs PLN for the final result
- [ ] Null results are shown explicitly (e.g., `null`) rather than producing no output
- [ ] Flag is documented in `pars --help`

## Design Decisions

### Flag naming options

| Option | Pros | Cons |
|--------|------|------|
| `-d` / `--debug` | Short, memorable | Could imply other debug behavior |
| `-i` / `--inspect` | Clear meaning | `-i` often means "interactive" |
| `-p` / `--print` | Matches Ruby `-p` | Could be confused with pretty-print |
| `--pln` | Explicit about format | Longer, less intuitive |
| `-v` / `--verbose` | Common flag | Usually means more logging |

**Recommendation**: `-d` / `--debug` — short and memorable, clearly indicates "show me what's happening"

### Behavior with `-e`

Two possible interpretations:

1. **Separate flag**: `-d "expr"` is shorthand for `-e "expr"` with PLN output
2. **Modifier flag**: `-e "code" -d` adds PLN output to any `-e` evaluation

**Recommendation**: Support both:
- `pars -d "expr"` — evaluate and show PLN (implies `-e`)
- `pars -e "code" -d` — evaluate code, show PLN of result

### Output format

The output should be the PLN representation as produced by `log()`:
- Strings: `"hello"` (quoted)
- Numbers: `42`, `3.14`
- Booleans: `true`, `false`
- Null: `null`
- Arrays: `["a", "b", "c"]`
- Dictionaries: `{key: "value", num: 42}`

---

## Technical Context

### Affected Components
- `cmd/pars/main.go` — Add flag definition and dispatch logic

### Implementation Sketch

```go
var (
    // ... existing flags ...
    debugFlag     = flag.Bool("d", false, "Output PLN representation of result")
    debugLongFlag = flag.Bool("debug", false, "Output PLN representation of result")
)

func main() {
    // ...
    
    debug := *debugFlag || *debugLongFlag
    
    switch {
    case evalCode != "" || debug:
        // If -d is used alone, treat remaining args as code
        if evalCode == "" && debug && len(flag.Args()) > 0 {
            evalCode = flag.Args()[0]
        }
        executeInline(evalCode, flag.Args(), prettyPrint, debug)
    // ...
    }
}

func executeInline(code string, args []string, prettyPrint bool, debug bool) {
    // ... existing evaluation code ...
    
    if debug {
        // Always output PLN, even for null
        fmt.Println(evaluated.Inspect())
    } else if evaluated != nil && evaluated.Type() != evaluator.NULL_OBJ {
        // ... existing output logic ...
    }
}
```

### Dependencies
- Depends on: FEAT-106 (CLI `-e` flag) — already implemented
- Blocks: None

## Test Plan

| Test Case | Command | Expected |
|-----------|---------|----------|
| Simple expression | `pars -d "1 + 2"` | `3` |
| String | `pars -d '"hello"'` | `"hello"` |
| Array | `pars -d "[1, 2, 3]"` | `[1, 2, 3]` |
| Dictionary | `pars -d "{a: 1, b: 2}"` | `{a: 1, b: 2}` |
| Regex match | `pars -d '"hello" ~ /(\w+)/'` | `["hello", "hello"]` |
| Null result | `pars -d "null"` | `null` |
| With -e | `pars -e "let x = 1" -d` | `null` (let returns null) |
| Long flag | `pars --debug "[1,2,3]"` | `[1, 2, 3]` |

## Related
- FEAT-106: CLI `-e` flag (foundation for this feature)
- PLN format: `docs/parsley/manual/pln.md`
