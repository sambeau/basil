---
id: PLAN-004
feature: FEAT-006
title: "Implementation Plan for Dev Mode Error Display"
status: draft
created: 2025-12-01
---

# Implementation Plan: FEAT-006

## Overview
Add a dev-mode error page that displays Parsley errors in the browser with syntax highlighting, source context, and live reload support.

## Prerequisites
- [x] Feature spec approved (FEAT-006)

## Tasks

### Task 1: Create error page renderer
**Files**: `server/errors.go` (new file)
**Estimated effort**: Medium

Steps:
1. Create `DevError` struct to hold error details (file, line, column, message, error type)
2. Create `renderDevErrorPage()` function that generates self-contained HTML
3. Add inline CSS styles (dark theme, monospace, syntax highlighting classes)
4. Include live reload script in the error page template
5. Add helper to extract line/column from error messages

Tests:
- `TestRenderDevErrorPage_ParseError` - verify HTML output with parse error
- `TestRenderDevErrorPage_RuntimeError` - verify HTML output with runtime error
- `TestExtractLineInfo` - verify line number extraction from error strings

---

### Task 2: Create syntax highlighter
**Files**: `server/errors.go`
**Estimated effort**: Medium

Steps:
1. Create `highlightParsley()` function for simple syntax highlighting
2. Highlight keywords: `let`, `fn`, `if`, `else`, `for`, `in`, `export`, `import`, `true`, `false`, `nil`
3. Highlight strings (double-quoted)
4. Highlight numbers
5. Highlight HTML tags `<...>` and `</...>`
6. Highlight comments `// ...`
7. Escape HTML entities in source code

Tests:
- `TestHighlightParsley_Keywords` - verify keyword highlighting
- `TestHighlightParsley_Strings` - verify string highlighting  
- `TestHighlightParsley_Tags` - verify HTML tag highlighting
- `TestHighlightParsley_HTMLEscape` - verify `<` and `>` in code are escaped

---

### Task 3: Create source context extractor
**Files**: `server/errors.go`
**Estimated effort**: Small

Steps:
1. Create `getSourceContext()` function that reads file and extracts lines around error
2. Handle edge cases: file not readable, line out of range, very short files
3. Return slice of `SourceLine` structs with line number, content, and isError flag
4. Default to 5 lines before and after error line

Tests:
- `TestGetSourceContext_MiddleOfFile` - verify context around line 20 in 100-line file
- `TestGetSourceContext_StartOfFile` - verify context for error on line 2
- `TestGetSourceContext_EndOfFile` - verify context for error on last line
- `TestGetSourceContext_FileNotFound` - verify graceful handling

---

### Task 4: Integrate with handler error paths
**Files**: `server/handler.go`
**Estimated effort**: Medium

Steps:
1. Add `devMode` field to `parsleyHandler` struct (passed from Server)
2. Modify parse error handling to call `renderDevErrorPage()` in dev mode
3. Modify runtime error handling to call `renderDevErrorPage()` in dev mode
4. Modify file-not-found error handling to call `renderDevErrorPage()` in dev mode
5. Ensure live reload script is included in error page response
6. Keep existing `http.Error()` behavior for production mode

Tests:
- `TestHandler_ParseError_DevMode` - verify dev error page returned
- `TestHandler_ParseError_ProdMode` - verify generic 500 returned
- `TestHandler_RuntimeError_DevMode` - verify dev error page with source context
- `TestHandler_FileNotFound_DevMode` - verify dev error page for missing handler

---

### Task 5: End-to-end testing and polish
**Files**: `server/errors_test.go`, `server/handler.go`
**Estimated effort**: Small

Steps:
1. Manual testing with real Parsley errors
2. Verify live reload works (fix error, page auto-recovers)
3. Verify error page renders correctly in browser
4. Adjust styling if needed
5. Update any relevant documentation

Tests:
- Integration test with test server in dev mode

---

## Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `go build -o basil .`
- [ ] Linter passes: `golangci-lint run`
- [ ] Manual test: introduce parse error, see error page
- [ ] Manual test: introduce runtime error, see error page with source
- [ ] Manual test: fix error, page auto-reloads
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| | Task 1 | ⬜ Not started | — |
| | Task 2 | ⬜ Not started | — |
| | Task 3 | ⬜ Not started | — |
| | Task 4 | ⬜ Not started | — |
| | Task 5 | ⬜ Not started | — |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- Column-level caret positioning (if Parsley provides column info consistently)
- Import chain display for nested import errors
- Copy-to-clipboard button for error message

