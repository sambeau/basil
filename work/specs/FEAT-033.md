---
id: FEAT-033
title: "String Sanitizer Methods"
status: implemented
priority: medium
created: 2025-12-05
implemented: 2025-12-09
author: "@human"
---

# FEAT-033: String Sanitizer Methods

## Summary
Add string methods for common sanitization tasks when cleaning user input from web forms. These methods complement the `std/valid` validation module (FEAT-032) by providing the "clean it up" counterpart to "is it valid?".

## User Story
As a web developer processing form input, I want simple string methods to clean up user data so that I can normalize whitespace, strip HTML, and extract digits without writing regex patterns.

## Target Audience
1. Web developers building small websites
2. Office workers automating tasks with web forms

## Design Decisions

### 1. Methods vs Functions
**Decision**: Implement as string methods (e.g., `s.normalizeSpace()`) not stdlib functions.
**Rationale**: Consistent with existing string methods like `s.trim()`, `s.upper()`, `s.lower()`. Chainable: `input.normalizeSpace().lower()`.

### 2. Return Type
**Decision**: All methods return new strings (immutable).
**Rationale**: Consistent with existing Parsley string behavior.

### 3. Unicode Handling
**Decision**: Methods work on UTF-8 strings but `digits()` only keeps ASCII digits 0-9.
**Rationale**: Predictable behavior for form processing. Non-ASCII digits (e.g., Arabic-Indic) are rare in web forms.

## Specification

### Whitespace Methods

| Method | Description | Example |
|--------|-------------|---------|
| `s.collapse()` | Replace runs of whitespace with single space | `"hello   world"` → `"hello world"` |
| `s.normalizeSpace()` | Trim + collapse (most common cleanup) | `"  hello   world  "` → `"hello world"` |
| `s.stripSpace()` | Remove ALL whitespace | `"hello world"` → `"helloworld"` |

**Whitespace characters affected**: space, tab, newline, carriage return (standard ASCII whitespace).

```parsley
// The whitespace trio
"  hello   world  ".trim()           // "hello   world" (edges only)
"  hello   world  ".collapse()       // "  hello world  " (internal only)
"  hello   world  ".normalizeSpace() // "hello world" (both - most common)
"  hello   world  ".stripSpace()     // "helloworld" (all gone)
```

### Content Extraction Methods

| Method | Description | Example |
|--------|-------------|---------|
| `s.stripHtml()` | Remove HTML/XML tags and decode entities | `"<b>hello</b>"` → `"hello"` |
| `s.digits()` | Keep only ASCII digits 0-9 | `"(123) 456-7890"` → `"1234567890"` |

```parsley
// Strip HTML tags and decode entities
"<p>Hello <b>world</b>!</p>".stripHtml()  // "Hello world!"
"<script>alert('x')</script>".stripHtml() // "alert('x')"
"Plain &amp; simple".stripHtml()          // "Plain & simple"
"&lt;not a tag&gt;".stripHtml()           // "<not a tag>"

// Extract digits for phone/card numbers
"(555) 123-4567".digits()                 // "5551234567"
"Card: 4111-1111-1111-1111".digits()      // "4111111111111111"
"+1 (555) 123-4567".digits()              // "15551234567"
```

### URL/Slug Method

| Method | Description | Example |
|--------|-------------|---------|
| `s.slug()` | Convert to URL-safe slug | `"Hello World!"` → `"hello-world"` |

**Slug rules:**
- Convert to lowercase
- Replace spaces and underscores with hyphens
- Remove non-alphanumeric characters (except hyphens)
- Collapse multiple hyphens to single hyphen
- Trim leading/trailing hyphens

```parsley
"Hello World!".slug()                     // "hello-world"
"  My Blog Post  ".slug()                 // "my-blog-post"
"Product: iPhone 15 Pro".slug()           // "product-iphone-15-pro"
"What's New?".slug()                      // "whats-new"
"---test---".slug()                       // "test"
"Café Münster".slug()                     // "caf-mnster" (removes accents)
```

Note: `slug()` removes accented characters rather than transliterating them (e.g., "é" → "" not "e"). This keeps the implementation simple. Users needing transliteration can pre-process with a separate function.

## Usage Examples

### Form Input Cleanup

```parsley
// Clean up a name field (trim + normalize whitespace)
let name = form.name.normalizeSpace()

// Clean up a phone number for storage
let phone = form.phone.digits()

// Generate URL slug from title
let slug = form.title.slug()

// Strip HTML from rich text input
let cleanText = form.content.stripHtml().normalizeSpace()
```

### Chaining

```parsley
// Clean and lowercase
let email = form.email.normalizeSpace().lower()

// Strip HTML, normalize, then validate length
let bio = form.bio.stripHtml().normalizeSpace()
if (len(bio) > 500) {
    // too long
}
```

### With Validation (FEAT-032)

```parsley
let valid = import("std/valid")

// Clean then validate
let phone = form.phone.digits()
if (valid.minLen(phone, 10) and valid.maxLen(phone, 11)) {
    // Valid US phone number length
}

// Normalize before checking
let email = form.email.normalizeSpace().lower()
if (valid.email(email)) {
    // Store normalized email
}
```

## Edge Cases

| Input | Method | Output | Notes |
|-------|--------|--------|-------|
| `""` | any | `""` | Empty string returns empty |
| `"   "` | `normalizeSpace()` | `""` | Whitespace-only becomes empty |
| `"   "` | `collapse()` | `" "` | Becomes single space |
| `"   "` | `stripSpace()` | `""` | All removed |
| `"no spaces"` | `normalizeSpace()` | `"no spaces"` | Already clean |
| `"<>"` | `stripHtml()` | `""` | Empty tag removed |
| `"abc"` | `digits()` | `""` | No digits = empty |
| `"---"` | `slug()` | `""` | Only hyphens = empty |
| `null` | any | Error | Methods require string receiver |

## Out of Scope

| Not Including | Reason |
|---------------|--------|
| `s.transliterate()` | Complex (accent → ASCII mapping); add later if needed |
| `s.escape()` / `s.unescape()` | HTML entities are output concern, not input sanitization |
| `s.ascii()` | Destructive for i18n; use specific methods instead |
| `s.alphanumeric()` | Use `s.slug()` or regex via `s.replace()` |

---

## Technical Context

### Affected Components
- `pkg/parsley/evaluator/methods.go` — Add new string methods
- `pkg/parsley/tests/string_methods_test.go` — Tests for new methods

### Dependencies
- Depends on: None
- Blocks: None
- Related: FEAT-032 (std/valid) — complementary functionality

### Implementation Notes

**collapse()**: 
```go
regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
```

**normalizeSpace()**:
```go
strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(s, " "))
```

**stripSpace()**:
```go
regexp.MustCompile(`\s+`).ReplaceAllString(s, "")
```

**stripHtml()**:
```go
stripped := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(s, "")
return html.UnescapeString(stripped)
```

**digits()**:
```go
regexp.MustCompile(`[^0-9]`).ReplaceAllString(s, "")
```

**slug()**:
```go
s = strings.ToLower(s)
s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
s = strings.Trim(s, "-")
```

## References
- Ruby on Rails `parameterize`: https://api.rubyonrails.org/classes/String.html#method-i-parameterize
- PHP `strip_tags`: https://www.php.net/manual/en/function.strip-tags.php
- Python `str.translate`: https://docs.python.org/3/library/stdtypes.html#str.translate

## Related
- FEAT-032: std/valid Standard Library Module (complementary)
- Plan: `docs/plans/FEAT-033-plan.md` (to be created)
