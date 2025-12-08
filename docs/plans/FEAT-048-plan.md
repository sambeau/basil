---
id: PLAN-029
feature: FEAT-048
title: "Implementation Plan for Text View Helpers"
status: draft
created: 2025-12-08
---

# Implementation Plan: FEAT-048 Text View Helpers

## Overview
Add three methods for common text display tasks:
1. `string.highlight(phrase, tag?)` — Wrap search matches in HTML tags with XSS protection
2. `string.paragraphs()` — Convert plain text to HTML paragraphs with XSS protection
3. `number.humanize(locale?)` — Locale-aware compact number formatting (1.2M, 1,5 Mio., etc.)

## Prerequisites
- [x] Understand existing string method pattern in `methods.go`
- [x] Understand existing number method pattern (format, currency, percent)
- [x] Verify `golang.org/x/text` is available for locale support

## Tasks

### Task 1: Implement `string.highlight(phrase, tag?)`
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Medium

Steps:
1. Add `"highlight"` case to `evalStringMethod` switch
2. Implement HTML escaping helper (or reuse existing)
3. Implement case-insensitive search and wrap logic
4. Handle edge cases: empty phrase, no matches, special chars in phrase
5. Default tag to `"mark"`, allow custom tag as second arg

Implementation details:
- Escape entire string first (HTML entities: `<`, `>`, `&`, `"`, `'`)
- Escape the search phrase too before matching
- Use case-insensitive matching (`strings.EqualFold` or lowercase compare)
- Wrap each match preserving original case
- Return escaped string if no matches

Tests:
- Basic highlight: `"hello world".highlight("world")` → `"hello <mark>world</mark>"`
- Custom tag: `"hello world".highlight("world", "strong")` → `"hello <strong>world</strong>"`
- XSS prevention: `"<script>".highlight("script")` → `"&lt;<mark>script</mark>&gt;"`
- Case insensitive: `"Hello WORLD".highlight("world")` → `"Hello <mark>WORLD</mark>"`
- Multiple matches: `"cat sat mat".highlight("at")` → `"c<mark>at</mark> s<mark>at</mark> m<mark>at</mark>"`
- No match: `"hello".highlight("xyz")` → `"hello"`
- Empty phrase: `"hello".highlight("")` → `"hello"`
- Special chars: `"a & b".highlight("&")` → `"a <mark>&amp;</mark> b"`

---

### Task 2: Implement `string.paragraphs()`
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Medium

Steps:
1. Add `"paragraphs"` case to `evalStringMethod` switch
2. Normalize line endings (`\r\n` → `\n`)
3. Split on `\n\n+` (one or more blank lines)
4. For each paragraph: trim whitespace, escape HTML, convert single `\n` to `<br/>`
5. Wrap each non-empty paragraph in `<p>...</p>`
6. Join and return

Implementation details:
- Empty input returns empty string
- Whitespace-only input returns empty string
- Skip empty paragraphs after splitting
- Preserve paragraph order

Tests:
- Basic: `"a\n\nb".paragraphs()` → `"<p>a</p><p>b</p>"`
- Line breaks: `"a\nb".paragraphs()` → `"<p>a<br/>b</p>"`
- Multiple blank lines: `"a\n\n\n\nb".paragraphs()` → `"<p>a</p><p>b</p>"`
- XSS: `"<script>".paragraphs()` → `"<p>&lt;script&gt;</p>"`
- Empty: `"".paragraphs()` → `""`
- Whitespace only: `"   \n\n   ".paragraphs()` → `""`
- Trim per paragraph: `"  a  \n\n  b  ".paragraphs()` → `"<p>a</p><p>b</p>"`
- Windows line endings: `"a\r\n\r\nb".paragraphs()` → `"<p>a</p><p>b</p>"`

---

### Task 3: Implement `number.humanize(locale?)`
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Medium

Steps:
1. Add `"humanize"` case to both `evalIntegerMethod` and `evalFloatMethod`
2. Use `golang.org/x/text/language` for locale parsing
3. Use `golang.org/x/text/message` with compact number formatting
4. Handle edge cases: zero, negative, very large numbers, unknown locale

Implementation details:
- Default locale: "en-US"
- Use CLDR compact decimal format (short form)
- Fall back to en-US if locale not recognized
- Handle negative numbers (preserve sign)
- Handle Infinity/NaN gracefully

Tests:
- Basic: `1000.humanize()` → `"1K"`
- Decimal: `1500.humanize()` → `"1.5K"`
- Million: `1000000.humanize()` → `"1M"`
- Billion: `1000000000.humanize()` → `"1B"`
- Trillion: `1000000000000.humanize()` → `"1T"`
- Zero: `0.humanize()` → `"0"`
- Negative: `(-1500).humanize()` → `"-1.5K"`
- Small: `999.humanize()` → `"999"`
- Float: `1234.5.humanize()` → `"1.2K"`
- German locale: `1234567.humanize("de-DE")` → `"1,2 Mio."`
- Unknown locale fallback: `1000.humanize("xx-XX")` → `"1K"` (en-US fallback)

---

### Task 4: Add HTML escape helper (if not exists)
**Files**: `pkg/parsley/evaluator/methods.go` or new `pkg/parsley/evaluator/html.go`
**Estimated effort**: Small

Steps:
1. Check if `html.EscapeString` from stdlib is sufficient
2. If needed, create helper that escapes: `<`, `>`, `&`, `"`, `'`
3. Ensure consistent escaping across highlight and paragraphs

Note: Go's `html.EscapeString` escapes `<`, `>`, `&`, `"` but NOT single quotes. May need custom if single quotes matter.

---

### Task 5: Update method lists for error messages
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Steps:
1. Add `"highlight"`, `"paragraphs"` to `stringMethods` slice
2. Add `"humanize"` to `integerMethods` and `floatMethods` slices

---

### Task 6: Write comprehensive tests
**Files**: `pkg/parsley/tests/methods_test.go` (or new `text_helpers_test.go`)
**Estimated effort**: Medium

Steps:
1. Add tests for `highlight` covering all edge cases
2. Add tests for `paragraphs` covering all edge cases
3. Add tests for `humanize` covering numbers and locales
4. Test error cases (wrong arg types, wrong arity)

---

### Task 7: Update documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small

Steps:
1. Add `.highlight()` to String Methods section in reference.md
2. Add `.paragraphs()` to String Methods section in reference.md
3. Add `.humanize()` to Number Methods section in reference.md
4. Add quick examples to CHEATSHEET.md if appropriate

---

## Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run` (if configured)
- [ ] Documentation updated
- [ ] Spec FEAT-048 acceptance criteria all checked
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| | Task 1: highlight | ⬜ Not started | |
| | Task 2: paragraphs | ⬜ Not started | |
| | Task 3: humanize | ⬜ Not started | |
| | Task 4: HTML helper | ⬜ Not started | |
| | Task 5: Method lists | ⬜ Not started | |
| | Task 6: Tests | ⬜ Not started | |
| | Task 7: Docs | ⬜ Not started | |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- Regex support for highlight phrase — complexity vs. value tradeoff
- Case-sensitive option for highlight — rare use case
- Custom paragraph wrapper (div instead of p) — rare use case

## Implementation Order
Recommended sequence:
1. Task 4 (HTML helper) — foundation for string methods
2. Task 1 (highlight) — simpler of the two string methods
3. Task 2 (paragraphs) — builds on same escaping
4. Task 3 (humanize) — independent, can be parallel
5. Task 5 (method lists) — quick cleanup
6. Task 6 (tests) — validate everything
7. Task 7 (docs) — final step
