---
id: FEAT-120
title: "Remove print/println builtins"
status: complete
priority: high
created: 2025-01-15
author: "@human"
---

# FEAT-120: Remove print/println builtins

## Summary

Remove the `print()`, `println()`, and `printf()` builtin functions from Parsley. These functions cause persistent confusion with console logging (their actual purpose is template output) and are redundant with Parsley's expression-based output model.

## User Story

As a Parsley developer, I want clear and consistent output semantics so that I don't confuse template output with console logging, and so that AI coding assistants generate correct idiomatic code.

## Background

### What These Functions Do

Unlike `log()` which writes immediately to stdout and returns `NULL`, `print()`/`println()` return a special `PrintValue` object that gets merged into the expression result stream. They're for *template composition*, not console output.

### Why They're Problematic

1. **Name collision** — Every programmer expects `print()` to mean console output
2. **Redundant** — Parsley's expression model makes them unnecessary:
   ```parsley
   // Expression style (idiomatic Parsley)
   for (item in items) {
       if (item.featured) {
           <li class=featured>{item.name}</li>
       } else {
           <li>{item.name}</li>
       }
   }
   
   // print style (unnecessary, confusing)
   for (item in items) {
       if (item.featured) {
           print(<li class=featured>{item.name}</li>)
       }
   }
   ```
3. **AI confusion** — AI assistants persistently generate `print()` calls because it's ingrained from Python/JS training data. This was why `print()` was originally added, but it perpetuates the wrong mental model.

### Origin

These functions were added to reduce friction when AI assistants generated code using `print()`. However, this approach teaches the wrong idiom. Better to have clear error messages that teach the Parsley way.

## Acceptance Criteria

### Part 1: Remove Builtins
- [x] Remove `print`, `println`, `printf` from `getBuiltins()` in evaluator.go
- [x] Remove `PrintValue` type and all handling code in eval functions
- [x] Remove from `BuiltinMetadata` in introspect.go
- [x] All existing tests pass (after updating test files)

### Part 2: Helpful Error Messages
- [x] Add special-case error handling for calls to `print`, `println`, `printf`
- [x] Error message teaches the correct pattern:
  ```
  Error: Unknown function 'print'.
  
  Parsley uses expression-based output. Instead of:
      print(value)
  
  Simply return the value:
      value
  
  For debugging/console output, use log().
  ```
- [x] Include "did you mean" style suggestion pointing to `log()` for debugging

### Part 3: Update Examples and Tests
- [x] Convert `pkg/parsley/tests/replace_function_test.pars` to expression style
- [x] Convert `examples/parsley/reference/21-builtins.pars`
- [x] Remove or convert `examples/parsley/temp/test_table_*.pars`
- [x] Update `.vscode-extension/test/syntax-test.pars`

### Part 4: Update Documentation
- [x] Update `docs/parsley/CHEATSHEET.md` — make "No print() function" the #1 gotcha
- [x] Update `.github/instructions/parsley.instructions.md` — prominent AI guidance
- [x] Update `.github/copilot-instructions.md` — reinforce in project rules
- [x] Update `contrib/highlightjs/README.md` — fix examples

### Part 5: Verification
- [ ] Test with fresh AI conversation to verify guidance is effective (deferred to human)
- [ ] Document any remaining AI patterns that need addressing (deferred to human)

## Scope

### In Scope
- Removing the three builtins
- Adding helpful error messages
- Updating all affected files
- AI guidance improvements

### Out of Scope
- Adding replacement functions (expression model is the replacement)
- Deprecation period (clean break for 1.0)

## Impact Analysis

### Breaking Change

This is an intentional breaking change for 1.0. Code using `print()`/`println()`/`printf()` will error with a helpful message explaining how to migrate.

### Migration Path

All uses can be mechanically converted:

| Before | After |
|--------|-------|
| `print(x)` | `x` |
| `println(x)` | `x` followed by `"\n"` or just `x` if newline not needed |
| `printf(template, dict)` | `template.render(dict)` |

### Files to Modify

**Core implementation:**
- `pkg/parsley/evaluator/evaluator.go` — remove builtins, remove `PrintValue` type
- `pkg/parsley/evaluator/introspect.go` — remove metadata

**Error handling:**
- `pkg/parsley/evaluator/evaluator.go` or errors package — add special-case error

**Tests:**
- `pkg/parsley/tests/replace_function_test.pars` — rewrite without print()
- `pkg/parsley/tests/` — any other tests using print

**Examples:**
- `examples/parsley/reference/21-builtins.pars`
- `examples/parsley/temp/test_table_builtin.pars`
- `examples/parsley/temp/test_table_error1.pars`
- `examples/parsley/temp/test_table_error2.pars`

**Documentation:**
- `docs/parsley/CHEATSHEET.md`
- `.github/instructions/parsley.instructions.md`
- `.github/copilot-instructions.md`
- `contrib/highlightjs/README.md`

## Error Message Design

The error message is critical for AI guidance. Proposed format:

```
Error at line 5, column 3: Unknown function 'print'

Parsley uses expression-based output — values are returned, not printed.

Instead of:
    print(value)
    print(<div>hello</div>)

Write:
    value
    <div>hello</div>

The last expression in a block becomes its output.

For debugging (console output), use:
    log("debug:", value)
```

Key principles:
1. **Immediate clarity** — "expression-based output" explains the paradigm
2. **Show the fix** — concrete before/after examples
3. **Address the debugging case** — point to `log()` for that use case
4. **Teach the model** — "last expression becomes output" teaches idiomatic Parsley

## AI Guidance Strategy

### Documentation Updates

1. **CHEATSHEET.md** — Reorder gotchas to put this first:
   ```markdown
   ### 1. No `print()` — Expressions ARE Output
   
   // ❌ WRONG (doesn't exist)
   print("hello")
   println(value)
   
   // ✅ CORRECT — the value IS the output
   "hello"
   value
   <div>{content}</div>
   
   // ✅ For debugging, use log()
   log("debug:", someVar)
   ```

2. **parsley.instructions.md** — Add to Major Gotchas section with examples

3. **copilot-instructions.md** — Add explicit rule about output model

### Example Corpus

Ensure ALL examples use expression style. AIs learn from examples, so:
- Audit every `.pars` file in `/examples/`
- Remove any `print()` usage
- Add comments showing the expression-based pattern

### Testing the AI Experience

After implementation:
1. Start fresh AI conversations
2. Ask for Parsley code that would naturally use print in other languages
3. Verify the AI either uses expression style OR gets a helpful error
4. Iterate on error messages based on what patterns AIs attempt

## Implementation Notes

### Removing PrintValue

The `PrintValue` type is handled in three places:
- `evalProgram()`
- `evalBlockStatement()`
- `evalInterpolationBlock()`

All three have identical handling code that expands `PrintValue` into the results array. This code can be deleted entirely.

### Special-Case Error Detection

Add detection in the function call evaluator. When resolving an identifier that doesn't exist, check if it's one of `["print", "println", "printf"]` and return the specialized error instead of the generic "unknown identifier" error.

## Test Plan

1. **Removal verification** — Confirm `print()`, `println()`, `printf()` all error
2. **Error message quality** — Verify error messages are helpful and accurate
3. **Regression** — Run full test suite after updating test files
4. **Example validation** — All example files execute without error
5. **AI testing** — Manual testing with AI assistants

## Related

- See `work/reports/FORMATTER_AUDIT.md` — contains the full audit of print/println
- FEAT-111: Declarative Method Registry (context for builtin organization)