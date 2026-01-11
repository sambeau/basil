---
id: PLAN-042
feature: FEAT-067
title: "Implementation Plan for Markdown String Parsing"
status: complete
created: 2025-12-14
completed: 2025-12-14
---

# Implementation Plan: FEAT-067 Markdown String Parsing

## Overview
Rename the `markdown` file format factory to `MD` and repurpose `markdown()` as a string-to-markdown parsing function, with `toMarkdown()` method on strings.

## Prerequisites
- [ ] Design reviewed and approved
- [ ] Decide on migration strategy (breaking change)

## Tasks

### Task 1: Add `MD` File Format Factory
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Add `MD` as an alias/replacement for file-based markdown loading.

Steps:
1. Locate the `"markdown"` entry in builtins (around line 5106)
2. Copy the entire builtin to a new `"MD"` entry
3. Update internal references from "markdown" to "MD" in error messages
4. Keep `"markdown"` temporarily for backwards compatibility with deprecation warning

Tests:
- `let doc <== MD(@./test.md)` loads markdown file
- Deprecation warning when using `markdown(@./path)`

---

### Task 2: Implement `markdown()` for String Parsing  
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium

Repurpose `markdown()` to parse strings instead of files.

Steps:
1. Modify the `"markdown"` builtin to accept strings instead of paths
2. For string input: call existing `parseMarkdown(content, env)` function
3. For path input: show error suggesting `MD(@./path)` instead
4. Return same structure: `{raw, html, md}`

Tests:
- `markdown("# Hello")` returns `{html: "<h1>Hello</h1>\n", raw: "# Hello", md: {}}`
- `markdown("---\ntitle: Test\n---\n# Hello")` parses frontmatter into `md`
- `markdown(@./file.md)` shows helpful error message

---

### Task 3: Implement `string.parseMarkdown()` Method
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Add `parseMarkdown()` method on strings.

Steps:
1. Locate string method handling (search for `case *String:` in method dispatch)
2. Add case for "parseMarkdown" method
3. Call same `parseMarkdown()` function used by `markdown()` builtin
4. Support optional options dictionary argument

Tests:
- `"# Hello".parseMarkdown()` returns same as `markdown("# Hello")`
- `myVar.parseMarkdown()` works with string variables

---

### Task 4: Update Existing Tests
**Files**: `pkg/parsley/tests/markdown_test.go`
**Estimated effort**: Small

Update all tests using file-based markdown to use `MD()`.

Steps:
1. Search for `markdown(@` patterns in test files
2. Replace with `MD(@`
3. Verify all tests still pass

Tests:
- All existing markdown tests pass
- New tests for `markdown(string)` and `string.toMarkdown()`

---

### Task 5: Add String Parsing Tests
**Files**: `pkg/parsley/tests/markdown_test.go`
**Estimated effort**: Medium

Add comprehensive tests for the new string-based API.

Steps:
1. Add `TestMarkdownStringParsing` for basic functionality
2. Add `TestMarkdownStringWithFrontmatter` for YAML frontmatter
3. Add `TestToMarkdownMethod` for method form
4. Add `TestMarkdownPathError` for helpful error on path input

Tests:
```go
// Basic string parsing
{input: `markdown("# Hello")`, expected: ...}

// With frontmatter
{input: `markdown("---\ntitle: Test\n---\n# Body")`, ...}

// Method form
{input: `"# Hello".parseMarkdown().html`, expected: "<h1>Hello</h1>\n"}

// Path error
{input: `markdown(@./file.md)`, expectError: true}
```

---

### Task 6: Documentation
**Files**: `docs/parsley/reference.md`, `docs/guide/`
**Estimated effort**: Small

Update documentation with new API.

Steps:
1. Update markdown section in reference docs
2. Add migration note about `markdown()` → `MD()` for files
3. Add examples for string parsing

---

## Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `go build -o basil ./cmd/basil`
- [ ] Linter passes: `golangci-lint run`
- [ ] Documentation updated
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| | | | |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- Consider `MD` → `Markdown` alias for discoverability
- Consider `markdown()` accepting URL strings for remote markdown
