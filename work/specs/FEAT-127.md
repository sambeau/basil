---
id: FEAT-127
title: "Parsley 1.0 Release Readiness"
status: draft
priority: critical
created: 2025-02-26
author: "@ai"
---

# FEAT-127: Parsley 1.0 Release Readiness

## Summary
Complete all must-fix items identified in the 1.0 Readiness Audit before the Parsley 1.0 release. This includes two behavior fixes, deprecation warnings for legacy APIs, and CLI cleanup.

## User Story
As a Parsley user, I want the 1.0 release to have consistent, unsurprising behavior and clear deprecation paths so that I can write reliable code and plan migrations.

## Acceptance Criteria
- [ ] Named regex capture groups return dictionaries with named keys
- [ ] `failIfInvalid()` accepts an optional custom message parameter
- [ ] Deprecation warnings are emitted for all legacy APIs
- [ ] `migrate-let-var` command is removed from CLI help
- [ ] All existing tests continue to pass
- [ ] New tests cover the changed behavior

## Design Decisions
- **Named captures include full match**: Return `{0: "full match", name1: "...", name2: "..."}` to preserve backward compatibility for code that checks array length
- **Deprecation via stderr**: Warnings go to stderr, not stdout, so they don't interfere with program output
- **One warning per session**: Each deprecation warning is shown only once per program execution to avoid spam

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Phase 1: Named Capture Groups (#97)

**Current behavior:**
```parsley
"John Smith" ~ /(?P<first>\w+)\s+(?P<last>\w+)/
// Returns: ["John Smith", "John", "Smith"]
```

**New behavior:**
```parsley
"John Smith" ~ /(?P<first>\w+)\s+(?P<last>\w+)/
// Returns: {0: "John Smith", first: "John", last: "Smith"}
```

**Location:** `pkg/parsley/evaluator/eval_regex.go`

**Implementation notes:**
- Go's `regexp.SubexpNames()` returns group names (empty string for unnamed groups)
- Build dictionary when any named groups exist
- Include index `0` for full match to maintain compatibility
- Unnamed groups get numeric string keys ("1", "2", etc.)

---

### Phase 2: Custom failIfInvalid Message (#90)

**Current behavior:**
```parsley
record.failIfInvalid()
// Error message: "Validation failed"
```

**New behavior:**
```parsley
record.failIfInvalid("User data is invalid")
// Error message: "User data is invalid"
```

**Location:** `pkg/parsley/evaluator/methods_record.go:253-280`

**Implementation notes:**
- Add optional string parameter
- Default to current message if not provided
- Update method signature in introspection

---

### Phase 3: Deprecation Warnings

| Item | Location | Warning Text |
|------|----------|--------------|
| `now()` | `evaluator.go` (builtins) | "now() is deprecated, use @now instead" |
| `@std/table` | `stdlib_table.go` | "@std/table is deprecated, use @table literal instead" |
| `<Label>` | `eval_tags.go` | "<Label> is deprecated, use <label> instead" |
| `<Error>` | `eval_tags.go` | "<Error> is deprecated, use <error> instead" |
| `<Meta>` | `eval_tags.go` | "<Meta @field> is deprecated, use <val @field @key=\"help\"/> instead" |
| `format(array, style)` | `evaluator.go` (builtins) | "format(array, style) is deprecated, use array.format(style) instead" |

**Implementation notes:**
- Add `emitDeprecationWarning(msg string)` helper that tracks seen warnings
- Use `sync.Map` or simple map with mutex for thread safety
- Output to stderr: `fmt.Fprintf(os.Stderr, "DEPRECATION WARNING: %s\n", msg)`

---

### Phase 4: Remove migrate-let-var from CLI

**Location:** `cmd/pars/main.go`

**Changes:**
- Remove from help text (lines ~157-161 in `printHelp`)
- Keep the command handler but have it print "unknown command" or similar
- Or: remove entirely and let it fall through to unknown command handling

---

## Edge Cases & Constraints

1. **Named captures with no match**: Should return `null` (unchanged from current)
2. **Mixed named/unnamed groups**: Unnamed groups get numeric keys
3. **Regex without named groups**: Continue returning array (no breaking change)
4. **failIfInvalid on valid record**: Should be a no-op (unchanged)
5. **Deprecation in loops**: Warning shown only once, not per iteration

## Implementation Notes
*Added during/after implementation*

## Related
- Plan: `work/plans/PLAN-106.md`
- Audit: `work/reports/1.0-READINESS-AUDIT.md`
- Backlog items: #97, #90