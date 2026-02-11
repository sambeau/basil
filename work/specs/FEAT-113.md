---
id: FEAT-113
title: "CLI: Change -e to Output PLN by Default"
status: implemented
priority: medium
created: 2026-02-11
author: "@human"
---

# FEAT-113: CLI: Change -e to Output PLN by Default

## Summary
Change the `-e` flag to output PLN (Parsley Literal Notation) by default, matching REPL behavior. Add a `--raw` flag for when file-like output is needed.

## User Story
As a developer debugging Parsley code with `-e`, I want to see the PLN representation of results so that I can understand the structure of values without having to wrap expressions in `log()`.

## Motivation
The `-e` flag is primarily used for quick testing and debugging—the same use case as the REPL. However, `-e` currently outputs "print string" format (like file execution), which loses structural information:

```bash
# Current behavior (unhelpful for debugging)
pars -e '[1, 2, 3]'        # Output: 123
pars -e '{a: 1, b: 2}'     # Output: a1b2
pars -e '"hello" ~ /(\w+)/' # Output: hellohello

# REPL behavior (what we want)
> [1, 2, 3]
[1, 2, 3]
> {a: 1, b: 2}
{a: 1, b: 2}
```

By making `-e` match REPL behavior, the common case (debugging) becomes easy, and users don't have to discover a separate `-d` flag.

## Acceptance Criteria
- [x] `pars -e "expression"` outputs PLN representation (like REPL)
- [x] `pars -e "expression" --raw` outputs print string (like file execution)
- [x] `-r` works as short form of `--raw`
- [x] Null results display as `null` (not silent)
- [x] `--raw` works with `-pp` for pretty-printed HTML
- [x] Help text updated to document new behavior
- [x] REPL behavior unchanged

## Design Decisions

### Why change `-e` instead of adding `-d`?

1. **Discoverability** — Users (and AIs) reach for `-e` first. If it doesn't show structure, they're confused and must hunt for another flag.

2. **Use case alignment** — `-e` is for quick testing/debugging, same as REPL. Show what you got, don't render it.

3. **REPL precedent** — The REPL already made this choice: interactive = exploration = show PLN.

4. **Simplicity** — One flag to learn, not two.

### `--raw` for file-like output

For scripting or when you genuinely want rendered output:

```bash
pars -e '"<html><body>Hello</body></html>"' --raw
# Output: <html><body>Hello</body></html>

pars -e '"<html><body>Hello</body></html>"' --raw -pp
# Output: (pretty-printed HTML)
```

### Output format (PLN)

Same as REPL and `log()`:
- Strings: `"hello"` (quoted)
- Numbers: `42`, `3.14`
- Booleans: `true`, `false`
- Null: `null`
- Arrays: `[1, 2, 3]`
- Dictionaries: `{key: "value", num: 42}`

---

## Technical Context

### Affected Components
- `cmd/pars/main.go` — Modify `executeInline()`, add `--raw` flag

### Implementation Sketch

```go
var (
    // ... existing flags ...
    rawFlag     = flag.Bool("r", false, "Output raw print string instead of PLN")
    rawLongFlag = flag.Bool("raw", false, "Output raw print string instead of PLN")
)

func executeInline(code string, args []string, prettyPrint bool, raw bool) {
    // ... existing evaluation code ...
    
    if evaluated == nil {
        if !raw {
            fmt.Println("null")
        }
        return
    }
    
    if evaluated.Type() == evaluator.ERROR_OBJ {
        // ... error handling unchanged ...
    }
    
    if raw {
        // File-like behavior (current)
        if evaluated.Type() != evaluator.NULL_OBJ {
            output := evaluator.ObjectToPrintString(evaluated)
            if prettyPrint {
                output = formatter.FormatHTML(output)
            }
            fmt.Println(output)
        }
    } else {
        // REPL-like behavior (new default)
        fmt.Println(evaluated.Inspect())
    }
}
```

### Dependencies
- Depends on: FEAT-106 (CLI `-e` flag) — already implemented
- Blocks: None

## Test Plan

| Test Case | Command | Expected |
|-----------|---------|----------|
| Number | `pars -e "1 + 2"` | `3` |
| String | `pars -e '"hello"'` | `"hello"` |
| Array | `pars -e "[1, 2, 3]"` | `[1, 2, 3]` |
| Dictionary | `pars -e "{a: 1}"` | `{a: 1}` |
| Regex match | `pars -e '"hi" ~ /(\w+)/'` | `["hi", "hi"]` |
| Null | `pars -e "null"` | `null` |
| Raw string | `pars -e '"hello"' --raw` | `hello` |
| Raw array | `pars -e "[1,2,3]" --raw` | `123` |
| Raw HTML | `pars -e '"<b>hi</b>"' -r` | `<b>hi</b>` |
| Raw + pretty | `pars -e '"<div>x</div>"' -r -pp` | Pretty HTML |
| Raw null | `pars -e "null" --raw` | (no output) |

## Migration Notes

This is a **breaking change** for anyone relying on `-e` output format in scripts. However:

1. `-e` is primarily used interactively, not in scripts
2. Scripts needing raw output can add `--raw`
3. The new behavior is more useful for the primary use case

## Related
- FEAT-106: CLI `-e` flag implementation
- PLN format: `docs/parsley/manual/pln.md`

---

## Implementation Notes

**Status**: Implemented in PLAN-086 (2026-02-11)

**Branch**: `feat/FEAT-113-cli-pln-output`

**Changes Made**:
1. Added `--raw` and `-r` flags to `cmd/pars/main.go`
2. Modified `executeInline()` to use `ObjectToFormattedReprString()` for PLN output (default)
3. Preserved `ObjectToPrintString()` for raw output (with `--raw` flag)
4. Updated help text with new flag documentation and examples
5. Added comprehensive integration tests in `cmd/pars/main_test.go`

**Key Implementation Detail**:
- PLN output uses `evaluator.ObjectToFormattedReprString()` (same as REPL)
- Raw output uses `evaluator.ObjectToPrintString()` (original behavior)
- Null displays as `null` in PLN mode, silent in raw mode
- All test cases from spec passing

**Breaking Change**:
This changes the default output format for `-e`. Users relying on the current behavior in scripts must add `--raw` flag.
