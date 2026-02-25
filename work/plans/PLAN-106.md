---
id: PLAN-106
feature: FEAT-127
title: "Implementation Plan for Parsley 1.0 Release Readiness"
status: complete
created: 2025-02-26
completed: 2025-02-26
---

# Implementation Plan: FEAT-127

## Overview
Complete all must-fix items from the 1.0 Readiness Audit: named capture groups returning dictionaries, custom failIfInvalid messages, deprecation warnings, and CLI cleanup.

## Prerequisites
- [x] 1.0 Readiness Audit complete
- [x] Test coverage gaps addressed (FEAT-126)

## Tasks

### Task 1: Named Capture Groups Return Dictionaries (#97)
**Files**: `pkg/parsley/evaluator/eval_regex.go`
**Estimated effort**: Small (1-2 hours)

Steps:
1. Locate regex match result construction (search for `SubexpNames`)
2. When named groups exist, build a Dictionary instead of Array
3. Include key `"0"` for full match
4. Named groups get their name as key
5. Unnamed groups get numeric string keys (`"1"`, `"2"`, etc.)
6. If no named groups, continue returning Array (backward compatible)

Tests:
- Named groups: `"John Smith" ~ /(?P<first>\w+)\s+(?P<last>\w+)/` → `{0: "John Smith", first: "John", last: "Smith"}`
- Mixed groups: `"abc" ~ /(?P<letter>[a-z])(\d+)/` → `{0: "abc", letter: "a", "2": "123"}`
- No named groups: `"abc" ~ /([a-z]+)/` → `["abc", "abc"]` (unchanged)
- No match: `"123" ~ /(?P<x>[a-z]+)/` → `null`
- Multiple matches with `~*`: verify each match is a dictionary

---

### Task 2: Custom Message for failIfInvalid (#90)
**Files**: `pkg/parsley/evaluator/methods_record.go`
**Estimated effort**: Small (30 minutes)

Steps:
1. Find `recordFailIfInvalid` function (~line 253-280)
2. Change arity check from `len(args) != 0` to `len(args) > 1`
3. If `len(args) == 1`, extract string message from `args[0]`
4. Use custom message in error, or default to "Validation failed"
5. Update introspection metadata for method signature

Tests:
- No message: `record.failIfInvalid()` → "Validation failed"
- Custom message: `record.failIfInvalid("Bad data")` → "Bad data"
- Valid record: no error regardless of message
- Wrong arg type: `record.failIfInvalid(123)` → type error

---

### Task 3: Add Deprecation Warning Infrastructure
**Files**: `pkg/parsley/evaluator/deprecation.go` (new file)
**Estimated effort**: Small (30 minutes)

Steps:
1. Create new file `deprecation.go` in evaluator package
2. Add package-level `var deprecationWarnings sync.Map`
3. Add `emitDeprecationWarning(code, message string)` function
4. Check if warning already emitted using sync.Map
5. If new, print to stderr and store in map
6. Format: `DEPRECATION WARNING: <message> [<code>]`

Tests:
- First call emits warning
- Second call with same code is silent
- Different codes emit separately

---

### Task 4: Add Deprecation Warning for now()
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small (15 minutes)

Steps:
1. Find `now` in builtins registration
2. Add call to `emitDeprecationWarning("DEP-001", "now() is deprecated, use @now literal instead")`
3. Function continues to work normally

Tests:
- `now()` returns datetime and emits warning (check stderr)
- `@now` works without warning

---

### Task 5: Add Deprecation Warning for @std/table
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Small (15 minutes)

Steps:
1. Find module loading function (likely `loadTableModule` or similar)
2. Add deprecation warning at module load time
3. Message: "@std/table is deprecated, use @table literal instead"

Tests:
- `import @std/table` emits warning
- `@table` literal works without warning

---

### Task 6: Add Deprecation Warnings for Uppercase Components
**Files**: `pkg/parsley/evaluator/eval_tags.go` or `form_components.go`
**Estimated effort**: Small (30 minutes)

Steps:
1. Find where `<Label>`, `<Error>`, `<Meta>` are handled
2. Add deprecation warnings when these tags are evaluated
3. Messages:
   - `<Label>` → "use <label> instead"
   - `<Error>` → "use <error> instead"  
   - `<Meta @field>` → "use <val @field @key=\"help\"/> instead"

Tests:
- Each uppercase component emits its warning
- Lowercase equivalents work without warning

---

### Task 7: Add Deprecation Warning for format(array, style)
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small (15 minutes)

Steps:
1. Find `format` builtin function
2. When first arg is Array, emit deprecation warning
3. Message: "format(array, style) is deprecated, use array.format(style) instead"
4. Duration format remains without warning (not deprecated)

Tests:
- `format([1,2,3], "and")` emits warning
- `[1,2,3].format("and")` no warning
- `format(@5d, "long")` no warning (duration not deprecated)

---

### Task 8: Remove migrate-let-var from CLI Help
**Files**: `cmd/pars/main.go`
**Estimated effort**: Small (10 minutes)

Steps:
1. Find `printHelp` function
2. Remove `migrate-let-var` from help text
3. Optionally: remove from command dispatch or leave as hidden command

Tests:
- `pars --help` does not show migrate-let-var
- `pars migrate-let-var` either shows "unknown command" or hidden error message

---

## Validation Checklist
- [x] All tests pass: `go test ./pkg/parsley/...`
- [x] Named capture groups return dict: `"a b" ~ /(?P<x>\w+)/ == {0: "a", x: "a"}`
- [x] failIfInvalid accepts message
- [x] Each deprecated item emits warning once
- [x] `pars --help` doesn't show migrate-let-var
- [ ] Linter passes: `golangci-lint run`
- [ ] Audit report action items updated

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-02-26 | Task 1 | ✅ Complete | Named capture groups return dictionaries |
| 2025-02-26 | Task 2 | ✅ Complete | failIfInvalid(msg?) with optional message |
| 2025-02-26 | Task 3 | ✅ Complete | deprecation.go with emitDeprecationWarning |
| 2025-02-26 | Task 4 | ⬚ Skipped | now() was never implemented - removed from introspect |
| 2025-02-26 | Task 5 | ✅ Complete | DEP-001 warning for @std/table |
| 2025-02-26 | Task 6 | ✅ Complete | DEP-002/003/004 for Label/Error/Meta |
| 2025-02-26 | Task 7 | ✅ Complete | DEP-005 for format(array, style) |
| 2025-02-26 | Task 8 | ✅ Complete | Removed from CLI help text |

## Deferred Items
Items to add to work/BACKLOG.md after implementation:
- None anticipated — these are all targeted fixes