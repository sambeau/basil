---
id: FEAT-067
title: "Markdown String Parsing"
status: complete
priority: medium
created: 2025-12-14
completed: 2025-12-14
author: "@sambeau"
---

# FEAT-067: Markdown String Parsing

## Summary
Rename the `markdown` file format factory to `MD` (matching `JSON`, `YAML`, `CSV` naming convention) and repurpose `markdown()` as a string-to-markdown parsing function. This makes the API more intuitive: users naturally expect `markdown(string)` to parse a markdown string, not load a file.

## User Story
As a developer, I want to parse markdown content directly from a string so that I can test markdown processing, manipulate documents programmatically, and work with markdown from non-file sources.

## Acceptance Criteria
- [ ] `MD(path)` loads markdown files (renamed from `markdown`)
- [ ] `markdown(string)` parses markdown content from a string
- [ ] `markdown(string)` returns same structure as `MD(path)`: `{raw, html, md}`
- [ ] `string.parseMarkdown()` works as method form of `markdown(string)`
- [ ] Error message when passing path to `markdown()` suggests using `MD(path)` instead
- [ ] All existing tests updated to use `MD()` for file loading
- [ ] Documentation updated

## Design Decisions
- **Why rename to `MD`?**: Matches existing pattern (`JSON`, `YAML`, `CSV`). While "MD" isn't commonly used to refer to markdown, it's unambiguous in context and frees `markdown` for the more intuitive string-parsing role.
- **Why keep `markdown()` for strings?**: Users naturally try `markdown(string)` and are confused when it doesn't work. The file vs string distinction should be: `<==` for files, `=` for strings/values.
- **Method form**: `string.parseMarkdown()` provides fluent API and matches `parseJSON()`, `parseCSV()` patterns.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Syntax Summary
```parsley
// File loading (renamed from markdown)
{raw, html, md} <== MD(@./post.md)

// String parsing (new)
{raw, html, md} = markdown("# Hello\n\nWorld")
{raw, html, md} = myString.parseMarkdown()
```

### Affected Components
- `pkg/parsley/evaluator/evaluator.go` — Rename `markdown` builtin to `MD`, add new `markdown` string-parsing function
- `pkg/parsley/evaluator/methods.go` — Add `parseMarkdown` string method
- `pkg/parsley/tests/markdown_test.go` — Update to use `MD()`, add string parsing tests

### Dependencies
- Depends on: None
- Blocks: FEAT-068 (Markdown Heading IDs) benefits from cleaner API

### Edge Cases & Constraints
1. **Frontmatter in strings** — `markdown(string)` should support YAML frontmatter same as file loading
2. **Path-like strings** — If user passes `markdown("./file.md")`, detect this and suggest `MD(@./file.md)`
3. **Empty strings** — Return `{raw: "", html: "", md: {}}`
4. **Backwards compatibility** — Code using `markdown(@./file.md)` breaks; migration guide needed

## Implementation Notes
*Added during/after implementation*

## Related
- Plan: `docs/plans/FEAT-067-plan.md`
- Related: FEAT-068 (Markdown Heading IDs)
