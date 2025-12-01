---
id: FEAT-006
title: "Dev Mode Error Display in Browser"
status: draft
priority: high
created: 2025-12-01
author: "@human"
---

# FEAT-006: Dev Mode Error Display in Browser

## Summary
Display Parsley errors directly in the browser during development mode, with syntax highlighting, line numbers, and source code context. The error page includes live reload so when the error is fixed, the page automatically recovers.

## User Story
As a developer using Basil in dev mode, I want to see Parsley errors directly in my browser with syntax highlighting and the relevant source code so that I can quickly identify and fix issues without switching to the terminal, and have the page automatically reload when I save a fix.

## Acceptance Criteria
- [ ] Parse errors display in browser with file path and line/column numbers
- [ ] Runtime errors display in browser with file path and error message
- [ ] Error page shows source code context (5 lines before/after error line)
- [ ] Error line is highlighted with a visual indicator (caret or highlight)
- [ ] Source code has syntax highlighting (at minimum: keywords, strings, numbers)
- [ ] Live reload script is injected so page recovers when error is fixed
- [ ] Only enabled in dev mode (production shows generic "Internal Server Error")
- [ ] Error page is styled and readable (dark theme, monospace code)
- [ ] Template/handler not found errors also display in browser

## Design Decisions
- **HTML error page, not JSON**: Developers see errors in browser, not just API responses
- **Inline styles**: Error page should be self-contained (no external CSS that might also fail)
- **Dev mode only**: Security - never leak error details, file paths, or source code in production
- **Live reload preserved**: Error page includes the live reload script so auto-refresh continues
- **Simple syntax highlighting**: CSS-based, no external JS libraries required

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `server/handler.go` — Modify error handling to render dev error page
- `server/errors.go` — New file: error page rendering with syntax highlighting
- `server/livereload.go` — May need adjustment to ensure script injection works on error pages

### Error Types to Handle

1. **Parse errors** (from `parser.Errors()`)
   - Have file path, line number, column, error message
   - Example: `parse error in app.pars: unexpected token at line 12, col 5`

2. **Runtime errors** (from `evaluator.Error`)
   - Have message, may include file:line info in message string
   - Example: `dictionary destructuring requires a dictionary value, got BUILTIN`

3. **File not found errors** (from `os.ReadFile`)
   - Handler file doesn't exist
   - Example: `reading script /path/to/missing.pars: no such file`

### Error Page Structure

```html
<!DOCTYPE html>
<html>
<head>
  <title>Error - Basil Dev</title>
  <style>/* inline styles */</style>
</head>
<body>
  <div class="error-container">
    <h1>🌿 Parsley Error</h1>
    <div class="error-message">
      <span class="file">app.pars</span>:<span class="line">12</span>
      <p>unexpected token '}' - expected expression</p>
    </div>
    <div class="source-code">
      <pre>
        <code>
          10 │ let x = 1
          11 │ let y = {
       →  12 │ }
          13 │ 
          14 │ <div>Hello</div>
        </code>
      </pre>
    </div>
  </div>
  <!-- Live reload script injected here -->
</body>
</html>
```

### Syntax Highlighting (CSS classes)
- `.kw` — keywords: `let`, `fn`, `if`, `else`, `for`, `in`, `export`, `import`
- `.str` — strings: `"..."` 
- `.num` — numbers: `123`, `3.14`
- `.tag` — HTML tags: `<div>`, `</p>`
- `.attr` — tag attributes: `class=`, `id=`
- `.comment` — comments: `// ...`

### Dependencies
- Depends on: None
- Blocks: None

### Edge Cases & Constraints
1. **Nested errors** — Import fails in imported file: show the import chain
2. **Binary/non-text responses** — Only inject error page for HTML-expected routes
3. **Very long lines** — Truncate display, show where error is
4. **Missing line number** — Some errors may not have line info; show what we have
5. **Source file unreadable** — If we can't read source for context, show error without code

## Implementation Notes
*Added during/after implementation*

## Related
- Plan: `docs/plans/FEAT-006-plan.md`
- Related bugs: BUG-005 (import inconsistencies - errors surfaced this need)

