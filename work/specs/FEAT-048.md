---
id: FEAT-048
title: "Text View Helpers"
status: implemented
priority: low
created: 2025-12-07
author: "@copilot"
---

# FEAT-048: Text View Helpers

## Summary

Add three string/number methods for common text display tasks: `.highlight()` for wrapping search matches in HTML tags, `.paragraphs()` for converting plain text to HTML paragraphs, and `.humanize()` for locale-aware compact number formatting. These are tasks developers frequently implement incorrectly (XSS vulnerabilities, locale issues).

## User Story

As a developer building content-heavy pages, I want simple methods to safely display user text and format numbers so that I don't have to worry about XSS escaping or locale-specific formatting.

## Acceptance Criteria

- [x] `string.highlight(phrase)` wraps matches in `<mark>` tags, escaping HTML in the string
- [x] `string.highlight(phrase, tag)` allows custom wrapper tag
- [x] `string.paragraphs()` converts newline-separated text to `<p>` tags, escaping HTML
- [x] `number.humanize()` returns compact format ("1.2M") using default locale
- [x] `number.humanize(locale)` returns locale-specific compact format
- [x] All methods handle edge cases gracefully (empty strings, zero, negative numbers)

## Design Decisions

- **Methods not functions**: Fits Parsley's existing pattern (`.format()`, `.currency()`, `.percent()`)
- **HTML escaping built-in**: The primary value of `.highlight()` and `.paragraphs()` is safe HTML generation
- **Locale via CLDR**: `.humanize()` uses Go's `golang.org/x/text` for proper locale support, consistent with existing datetime/money formatting

---

## Technical Context

### String Methods

#### `.highlight(phrase, tag?)`

Finds all occurrences of `phrase` in the string, wraps them in an HTML tag.

```parsley
"Search results for dogs".highlight("dogs")
// → "Search results for <mark>dogs</mark>"

"Find dogs and cats".highlight("dogs", "strong")
// → "Find <strong>dogs</strong> and cats"

// HTML in source is escaped
"Find <script>alert(1)</script>".highlight("script")
// → "Find &lt;<mark>script</mark>&gt;alert(1)&lt;/<mark>script</mark>&gt;"

// Case-insensitive matching
"Hello WORLD".highlight("world")
// → "Hello <mark>WORLD</mark>"

// Multiple matches
"the cat sat on the mat".highlight("at")
// → "the c<mark>at</mark> s<mark>at</mark> on the m<mark>at</mark>"

// No match - returns escaped original
"hello world".highlight("xyz")
// → "hello world"

// Empty phrase - returns escaped original
"hello world".highlight("")
// → "hello world"
```

**Implementation notes:**
- Escape the entire string first (HTML entities)
- Then find and wrap matches (phrase itself is also escaped before matching)
- Case-insensitive by default (most common use case: search results)
- Returns raw HTML string suitable for template interpolation

#### `.paragraphs()`

Converts plain text with blank lines to HTML paragraphs.

```parsley
"First paragraph.\n\nSecond paragraph.".paragraphs()
// → "<p>First paragraph.</p><p>Second paragraph.</p>"

// Single newlines become <br/>
"Line one.\nLine two.".paragraphs()
// → "<p>Line one.<br/>Line two.</p>"

// Multiple blank lines treated as single paragraph break
"Para one.\n\n\n\nPara two.".paragraphs()
// → "<p>Para one.</p><p>Para two.</p>"

// HTML is escaped
"Hello <script>".paragraphs()
// → "<p>Hello &lt;script&gt;</p>"

// Empty string
"".paragraphs()
// → ""

// Whitespace only
"   \n\n   ".paragraphs()
// → ""

// Trims leading/trailing whitespace per paragraph
"  Hello world  \n\n  Goodbye world  ".paragraphs()
// → "<p>Hello world</p><p>Goodbye world</p>"
```

**Implementation notes:**
- Split on `\n\n` (or `\r\n\r\n`) for paragraph breaks
- Convert single `\n` to `<br/>`
- Escape HTML entities in content
- Trim whitespace from each paragraph
- Skip empty paragraphs

### Number Method

#### `.humanize(locale?)`

Returns compact/abbreviated number format.

```parsley
// Default locale (system or en-US)
999.humanize()              // "999"
1000.humanize()             // "1K"
1500.humanize()             // "1.5K"
1000000.humanize()          // "1M"
1234567.humanize()          // "1.2M"
1000000000.humanize()       // "1B"

// Negative numbers
(-1500).humanize()          // "-1.5K"

// Small decimals
0.5.humanize()              // "0.5"

// Zero
0.humanize()                // "0"

// With locale
1234567.humanize("de-DE")   // "1,2 Mio."
1234567.humanize("ja")      // "123万"
1234567.humanize("fr-FR")   // "1,2 M"

// Very large numbers
1000000000000.humanize()    // "1T"
```

**Implementation notes:**
- Use `golang.org/x/text/language` and `golang.org/x/text/message` with compact format
- Follows CLDR rules for each locale
- Consistent with existing `.format(locale)` pattern on numbers

### Affected Components

- `pkg/parsley/evaluator/builtins_string.go` — Add `highlight`, `paragraphs` methods
- `pkg/parsley/evaluator/builtins_number.go` — Add `humanize` method
- `pkg/parsley/evaluator/evaluator.go` — Register new methods

### Dependencies

- Depends on: None
- Blocks: None

### Edge Cases & Constraints

1. **Empty input** — Returns empty string (no error)
2. **Nil/null input** — Runtime error (consistent with other methods)
3. **Non-string phrase to highlight** — Coerce to string or error?
4. **Regex in phrase** — Treated as literal text (no regex support)
5. **Locale not found** — Fall back to en-US (consistent with datetime)
6. **Very large numbers** — Follow CLDR (may vary by locale)
7. **Infinity/NaN** — Return "∞" / "NaN" strings

### Test Cases

```parsley
// highlight
assert("hello world".highlight("world") == "hello <mark>world</mark>")
assert("a & b".highlight("&") == "a <mark>&amp;</mark> b")
assert("test".highlight("x") == "test")
assert("AAA".highlight("a") == "<mark>A</mark><mark>A</mark><mark>A</mark>")
assert("hi".highlight("", "b") == "hi")

// paragraphs
assert("a\n\nb".paragraphs() == "<p>a</p><p>b</p>")
assert("a\nb".paragraphs() == "<p>a<br/>b</p>")
assert("".paragraphs() == "")
assert("<b>hi</b>".paragraphs() == "<p>&lt;b&gt;hi&lt;/b&gt;</p>")

// humanize
assert(1000.humanize() == "1K")
assert(1500.humanize() == "1.5K")
assert(1000000.humanize() == "1M")
assert(0.humanize() == "0")
assert((-1500).humanize() == "-1.5K")
```

## Implementation Notes

*To be added during implementation*

## Related

- Design: `work/design/rails-inspired-ux.md` — Original discussion
- Similar: `.format()`, `.currency()`, `.percent()` — Existing locale-aware methods
