---
id: PLAN-031
feature: FEAT-053
title: "Implementation Plan for String .render() Method"
status: completed
created: 2025-12-09
completed: 2025-12-09
---

# Implementation Plan: FEAT-053 String .render() Method

## Overview
Implement `string.render()` method for `@{...}` interpolation, along with `printf()` builtin, `dict.render()` method, and `markdown()` integration. This enables deferred/manual interpolation for templates where literal `{...}` braces are needed (CSS, JavaScript, JSON).

## Prerequisites
- [x] FEAT-053 spec finalized
- [x] Understand existing `evalTemplateLiteral()` implementation

## Tasks

### Task 1: Add `interpolateRawString()` Helper Function
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium

Steps:
1. Add `interpolateRawString(template string, env *Environment) Object` function
2. Implement `@{...}` detection with brace counting
3. Implement `\@` escape sequence handling
4. Parse and evaluate expressions using lexer/parser
5. Convert results using `objectToTemplateString()`

Tests:
- Simple variable substitution
- Math expressions in `@{...}`
- Function calls in `@{...}`
- Nested braces handling
- `\@` escape produces literal `@`
- Error propagation from invalid expressions

---

### Task 2: Add `string.render()` Method
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Steps:
1. Add `"render"` to `stringMethods` array
2. Add `case "render":` in `evalStringMethod()` 
3. Handle 0 args (use current env) and 1 arg (dictionary) cases
4. Call `interpolateRawString()` with appropriate environment

Tests:
- `"template".render()` with current scope
- `"template".render({key: value})`
- Error on invalid argument type
- Error on >1 arguments

---

### Task 3: Add `printf()` Builtin
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Add `"printf"` to `getBuiltins()` map
2. Require exactly 2 arguments (string, dictionary)
3. Create environment from dictionary
4. Call `interpolateRawString()`

Tests:
- `printf("template", {key: value})`
- Error on wrong argument count
- Error on wrong argument types

---

### Task 4: Add `dict.render()` Method
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Steps:
1. Add `"render"` to `dictionaryMethods` array
2. Add `case "render":` in `evalDictionaryMethod()`
3. Require 1 argument (string template)
4. Create environment from dictionary pairs
5. Call `interpolateRawString()`

Tests:
- `{key: value}.render("template")`
- Error on missing argument
- Error on non-string argument

---

### Task 5: Update `markdown()` Builtin
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Locate `markdown()` builtin implementation
2. After reading file content, call `interpolateRawString()` with current env
3. Check for error result before proceeding to markdown conversion
4. Convert rendered content to HTML

Tests:
- Markdown file with `@{variable}` gets interpolated
- Markdown file with literal `{...}` preserved (code blocks)
- Error in `@{...}` expression propagates correctly

---

### Task 6: Add Comprehensive Tests
**Files**: `pkg/parsley/tests/render_test.go` (new file)
**Estimated effort**: Medium

Steps:
1. Create new test file for render functionality
2. Test `string.render()` - all variants
3. Test `printf()` - all variants
4. Test `dict.render()` - all variants
5. Test examples from spec (math, conditionals, method chains)
6. Test edge cases (empty dict, missing vars, nested braces)
7. Test `\@` escaping

Tests:
- All examples from FEAT-053 spec
- Edge cases documented in spec

---

### Task 7: Update Documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small

Steps:
1. Add `string.render()` to string methods section
2. Add `dict.render()` to dictionary methods section  
3. Add `printf()` to builtins section
4. Update `markdown()` documentation to mention `@{...}` interpolation
5. Add examples showing all three syntax forms

Tests:
- N/A (documentation)

---

## Validation Checklist
- [x] All tests pass: `make test`
- [x] Build succeeds: `make build`
- [x] Linter passes: `golangci-lint run`
- [x] Documentation updated (reference.md, CHEATSHEET.md)
- [x] BACKLOG.md updated with deferrals (if any)
- [x] All three syntax forms work equivalently
- [x] Markdown integration working

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-09 | Task 1 | ✅ Completed | Added interpolateRawString() helper |
| 2025-12-09 | Task 2 | ✅ Completed | Added string.render() method |
| 2025-12-09 | Task 3 | ✅ Completed | Added printf() builtin |
| 2025-12-09 | Task 4 | ✅ Completed | Added dict.render() method |
| 2025-12-09 | Task 5 | ✅ Completed | Updated markdown() with interpolation |
| 2025-12-09 | Task 6 | ✅ Completed | Added comprehensive tests |
| 2025-12-09 | Task 7 | ✅ Completed | Updated reference.md and CHEATSHEET.md |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- None
