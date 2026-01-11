---
id: PLAN-043
feature: FEAT-068
title: "Implementation Plan for Markdown Heading IDs"
status: complete
created: 2025-12-14
completed: 2025-12-14
---

# Implementation Plan: FEAT-068 Markdown Heading IDs

## Overview
Add `{ids: true}` option to markdown parsing that automatically inserts `id` attributes on heading elements, enabling anchor links and TOC navigation.

## Prerequisites
- [ ] FEAT-067 complete (optional, but cleaner API)
- [ ] Verify Goldmark's ID generation matches our `generateSlug()` function

## Tasks

### Task 1: Research Goldmark Heading ID Generation
**Files**: N/A (research task)
**Estimated effort**: Small

Verify how Goldmark generates heading IDs and ensure compatibility with our existing `generateSlug()` function.

Steps:
1. Test Goldmark's `parser.WithAutoHeadingID()` output format
2. Compare with `generateSlug()` in `stdlib_markdown.go`
3. Document any differences
4. Decide: use Goldmark's built-in, or implement custom renderer

Expected findings:
- Goldmark uses lowercase, hyphen-separated IDs
- Goldmark appends `-1`, `-2` for duplicates
- Should match or be close to our `generateSlug()`

---

### Task 2: Add Options Parsing to `parseMarkdown()`
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Modify `parseMarkdown()` to accept and process options dictionary.

Steps:
1. Change signature: `parseMarkdown(content string, options *Dictionary, env *Environment)`
2. Extract `ids` boolean from options (default: false)
3. Pass options to Goldmark configuration

Code sketch:
```go
func parseMarkdown(content string, options *Dictionary, env *Environment) (Object, *Error) {
    includeIDs := false
    if options != nil {
        if val, ok := options.Get("ids"); ok {
            if b, ok := val.(*Boolean); ok {
                includeIDs = b.Value
            }
        }
    }
    
    // Configure Goldmark with or without auto heading IDs
    parserOpts := []parser.Option{}
    if includeIDs {
        parserOpts = append(parserOpts, parser.WithAutoHeadingID())
    }
    // ...
}
```

Tests:
- `parseMarkdown("# Hello", nil, env)` works (no options)
- `parseMarkdown("# Hello", {ids: true}, env)` includes IDs

---

### Task 3: Update MD() and markdown() Builtins
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Pass options dictionary through to `parseMarkdown()`.

Steps:
1. Locate `MD` and `markdown` builtins
2. Extract options from second argument (if present)
3. Pass to `parseMarkdown()`

Tests:
- `MD(@./file.md, {ids: true})` generates IDs
- `markdown("# Hello", {ids: true})` generates IDs

---

### Task 4: Update string.parseMarkdown() Method
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Support options argument in `parseMarkdown()` method.

Steps:
1. Check for optional dictionary argument
2. Pass to `parseMarkdown()`

Tests:
- `"# Hello".parseMarkdown({ids: true}).html` contains `id="hello"`

---

### Task 5: Add Heading ID Tests
**Files**: `pkg/parsley/tests/markdown_test.go`
**Estimated effort**: Medium

Comprehensive tests for heading ID generation.

Steps:
1. Test basic heading ID generation
2. Test duplicate heading handling
3. Test special characters in headings
4. Test with and without option (default behavior)

Test cases:
```go
// Basic ID generation
{
    input: `markdown("# Hello World", {ids: true}).html`,
    expected: `<h1 id="hello-world">Hello World</h1>`,
}

// Duplicate headings
{
    input: `markdown("# Intro\n\n# Intro", {ids: true}).html`,
    contains: [`id="intro"`, `id="intro-1"`],
}

// Default: no IDs
{
    input: `markdown("# Hello").html`,
    notContains: `id=`,
}

// Special characters
{
    input: `markdown("# Hello, World!", {ids: true}).html`,
    contains: `id="hello-world"`,
}
```

---

### Task 6: Ensure AST Consistency
**Files**: `pkg/parsley/evaluator/stdlib_markdown.go`
**Estimated effort**: Small

Verify that `md.parse()` AST IDs match HTML output IDs.

Steps:
1. Check `generateSlug()` implementation
2. Compare with Goldmark's ID generation
3. If different, either:
   a. Update `generateSlug()` to match Goldmark, OR
   b. Use custom Goldmark ID generator to match our slugs

Tests:
- Parse markdown with `md.parse()`, get heading ID
- Parse same markdown with `{ids: true}`, extract ID from HTML
- IDs should match

---

### Task 7: Documentation
**Files**: `docs/parsley/reference.md`, `docs/guide/`
**Estimated effort**: Small

Document the `{ids: true}` option.

Steps:
1. Add `ids` option to markdown function documentation
2. Add example showing anchor links
3. Document duplicate heading behavior

Example documentation:
```markdown
### Heading IDs

Add `{ids: true}` to generate `id` attributes on headings:

```parsley
{html} = markdown("# Getting Started\n\n## Installation", {ids: true})
// html contains: <h1 id="getting-started">Getting Started</h1>...
```

This enables anchor links like `<a href="#installation">Jump to Installation</a>`.
```

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
- Option to specify custom ID prefix (e.g., `{ids: "section-"}`)
- Option to specify ID generation function
- Expose ID generation as standalone function `md.slug("My Heading")`
