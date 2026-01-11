---
id: FEAT-068
title: "Markdown Heading IDs"
status: complete
priority: medium
created: 2025-12-14
completed: 2025-12-14
author: "@sambeau"
---

# FEAT-068: Markdown Heading IDs

## Summary
Add an option to automatically insert `id` attributes on heading elements in markdown-generated HTML. This enables linking to specific sections of a document (e.g., `#my-section-title`). The IDs should match the human-readable slugs already generated in the markdown AST.

## User Story
As a developer, I want heading IDs automatically added to my markdown HTML so that I can create anchor links and table-of-contents navigation to specific sections.

## Acceptance Criteria
- [ ] `MD(path, {ids: true})` adds `id` attributes to `<h1>` through `<h6>` tags
- [ ] `markdown(string, {ids: true})` same behavior for string parsing
- [ ] `string.parseMarkdown({ids: true})` same behavior for method form
- [ ] IDs are kebab-case slugs generated from heading text
- [ ] IDs match existing `id` field in markdown AST nodes (consistency)
- [ ] Duplicate headings get unique IDs (e.g., `my-heading`, `my-heading-1`, `my-heading-2`)
- [ ] Option defaults to `false` (non-breaking change)
- [ ] Documentation updated with examples

## Design Decisions
- **Why opt-in?**: Adding IDs changes HTML output, which could break existing CSS or tests. Non-breaking by default.
- **ID format**: Kebab-case, matching the existing `generateSlug()` function in stdlib_markdown.go. Goldmark's default ID style is similar.
- **Goldmark integration**: Use goldmark's `parser.WithAutoHeadingID()` option for HTML generation, but ensure ID style matches our AST slugs.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Syntax Summary
```parsley
// File loading with heading IDs
{raw, html, md} <== MD(@./post.md, {ids: true})

// String parsing with heading IDs
{raw, html, md} = markdown("# Hello\n\n## World", {ids: true})

// Method form with options
{raw, html, md} = myString.parseMarkdown({ids: true})
```

### Expected Output
Input:
```markdown
# Introduction
## Getting Started
## Getting Started
```

Output with `{ids: true}`:
```html
<h1 id="introduction">Introduction</h1>
<h2 id="getting-started">Getting Started</h2>
<h2 id="getting-started-1">Getting Started</h2>
```

### Affected Components
- `pkg/parsley/evaluator/evaluator.go` — Add `ids` option handling to `MD()`, `markdown()`, and `parseMarkdown()`
- `pkg/parsley/evaluator/methods.go` — Ensure `parseMarkdown()` method supports options
- `pkg/parsley/tests/markdown_test.go` — Add heading ID tests

### Dependencies
- Depends on: FEAT-067 (cleaner to implement with new API, but not required)
- Blocks: None

### Goldmark Integration
Goldmark supports automatic heading IDs via:
```go
import "github.com/yuin/goldmark/parser"

md := goldmark.New(
    goldmark.WithParserOptions(
        parser.WithAutoHeadingID(),
    ),
)
```

The default ID style is similar to ours (lowercase, hyphen-separated). We should verify the output matches our `generateSlug()` function, or configure Goldmark with a custom ID function for consistency:
```go
parser.WithHeadingAttribute()  // Preserve existing attributes
```

### Edge Cases & Constraints
1. **Duplicate headings** — Two `## Overview` sections need unique IDs. Goldmark handles this by appending `-1`, `-2`, etc.
2. **Special characters** — IDs should only contain alphanumeric and hyphens. Unicode should be converted or stripped.
3. **Empty headings** — Generate ID like `heading-1` based on position
4. **AST consistency** — The `id` field in `md.parse()` AST output should match the HTML `id` attribute

## Implementation Notes
*Added during/after implementation*

## Related
- Plan: `work/plans/FEAT-068-plan.md`
- Related: FEAT-067 (Markdown String Parsing)
