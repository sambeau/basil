---
id: PLAN-021
feature: FEAT-033
title: "Implementation Plan for String Sanitizer Methods"
status: ready
created: 2025-12-05
updated: 2025-12-09
---

# Implementation Plan: FEAT-033 String Sanitizer Methods

## Overview
Add 6 string methods for common sanitization tasks when cleaning user input from web forms.

**Total methods: 6**
- Whitespace: `collapse()`, `normalizeSpace()`, `stripSpace()`
- Content: `stripHtml()`, `digits()`
- URL: `slug()`

**Estimated effort**: 1-2 hours
**Risk**: Low (additive change, no breaking changes)

## Prerequisites
- [x] Design decisions finalized (see FEAT-033)
- [x] Understand existing string method implementation (see `methods.go`)
- [x] Test file pattern identified (`pkg/parsley/tests/methods_test.go`)

## Existing String Method Pattern
String methods are defined in `pkg/parsley/evaluator/methods.go`:

```go
// In evalStringMethod()
case "trim":
    return &String{Value: strings.TrimSpace(s.Value)}
case "upper":
    return &String{Value: strings.ToUpper(s.Value)}
// Add new methods here
```

Methods are also listed in `stringMethods` slice for documentation/completion.

## Tasks

### Task 0: Add Pre-compiled Regex Patterns
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Add compiled regex patterns at package level for performance:

```go
// Near top of file, after imports
var (
    whitespaceRegex = regexp.MustCompile(`\s+`)
    htmlTagRegex    = regexp.MustCompile(`<[^>]*>`)
    nonDigitRegex   = regexp.MustCompile(`[^0-9]`)
    nonSlugRegex    = regexp.MustCompile(`[^a-z0-9]+`)
)
```

---

### Task 1: Add collapse() Method
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Implement `collapse()` - replace runs of whitespace with single space:

```go
case "collapse":
    if len(args) != 0 {
        return newArityError("collapse", len(args), 0)
    }
    return &String{Value: whitespaceRegex.ReplaceAllString(str.Value, " ")}
```

---

### Task 2: Add normalizeSpace() Method
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Implement `normalizeSpace()` - trim + collapse:

```go
case "normalizeSpace":
    if len(args) != 0 {
        return newArityError("normalizeSpace", len(args), 0)
    }
    collapsed := whitespaceRegex.ReplaceAllString(str.Value, " ")
    return &String{Value: strings.TrimSpace(collapsed)}
```

---

### Task 3: Add stripSpace() Method
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Implement `stripSpace()` - remove all whitespace:

```go
case "stripSpace":
    if len(args) != 0 {
        return newArityError("stripSpace", len(args), 0)
    }
    return &String{Value: whitespaceRegex.ReplaceAllString(str.Value, "")}
```

---

### Task 4: Add stripHtml() Method
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Implement `stripHtml()` - remove HTML tags:

```go
case "stripHtml":
    if len(args) != 0 {
        return newArityError("stripHtml", len(args), 0)
    }
    return &String{Value: htmlTagRegex.ReplaceAllString(str.Value, "")}
```

---

### Task 5: Add digits() Method
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Implement `digits()` - keep only ASCII digits 0-9:

```go
case "digits":
    if len(args) != 0 {
        return newArityError("digits", len(args), 0)
    }
    return &String{Value: nonDigitRegex.ReplaceAllString(str.Value, "")}
```

---

### Task 6: Add slug() Method
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Implement `slug()` - convert to URL-safe slug:

```go
case "slug":
    if len(args) != 0 {
        return newArityError("slug", len(args), 0)
    }
    result := strings.ToLower(str.Value)
    result = nonSlugRegex.ReplaceAllString(result, "-")
    result = strings.Trim(result, "-")
    return &String{Value: result}
```

---

### Task 7: Update Method List
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Add new methods to `stringMethods` slice (line ~30):

```go
var stringMethods = []string{
    "toUpper", "toLower", "trim", "split", "replace", "length", "includes",
    "render", "highlight", "paragraphs", "parseJSON", "parseCSV",
    "collapse", "normalizeSpace", "stripSpace", "stripHtml", "digits", "slug",
}
```

---

### Task 8: Add Tests
**Files**: `pkg/parsley/tests/methods_test.go`
**Estimated effort**: Medium

Add comprehensive tests for all 6 new methods. Add to `TestStringMethods`:

```go
// collapse()
{`"hello   world".collapse()`, "hello world"},
{`"  hello   world  ".collapse()`, " hello world "},
{`"hello\t\nworld".collapse()`, "hello world"},
{`"hello world".collapse()`, "hello world"},
{`"".collapse()`, ""},

// normalizeSpace()
{`"  hello   world  ".normalizeSpace()`, "hello world"},
{`"   ".normalizeSpace()`, ""},
{`"hello world".normalizeSpace()`, "hello world"},
{`"\t\nhello\t\n".normalizeSpace()`, "hello"},

// stripSpace()
{`"hello world".stripSpace()`, "helloworld"},
{`"  hello   world  ".stripSpace()`, "helloworld"},
{`"hello".stripSpace()`, "hello"},
{`"   ".stripSpace()`, ""},

// stripHtml()
{`"<b>hello</b>".stripHtml()`, "hello"},
{`"<p>Hello <b>world</b>!</p>".stripHtml()`, "Hello world!"},
{`"no tags".stripHtml()`, "no tags"},
{`"<>".stripHtml()`, ""},

// digits()
{`"(123) 456-7890".digits()`, "1234567890"},
{`"+1 (555) 123-4567".digits()`, "15551234567"},
{`"abc".digits()`, ""},
{`"a1b2c3".digits()`, "123"},

// slug()
{`"Hello World".slug()`, "hello-world"},
{`"Hello World!".slug()`, "hello-world"},
{`"  My Blog Post  ".slug()`, "my-blog-post"},
{`"Product: iPhone 15 Pro".slug()`, "product-iphone-15-pro"},
{`"What's New?".slug()`, "whats-new"},
{`"---test---".slug()`, "test"},
{`"".slug()`, ""},
{`"---".slug()`, ""},
```

Also test chaining:
```go
// Chaining sanitizers
{`"  <b>Hello</b>   World  ".stripHtml().normalizeSpace()`, "Hello World"},
{`"  Form Title  ".normalizeSpace().slug()`, "form-title"},
```

---

### Task 9: Documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small

**reference.md** - Add to String Methods section:

```markdown
### Sanitization Methods

| Method | Description | Example |
|--------|-------------|---------|
| `s.collapse()` | Replace whitespace runs with single space | `"a   b"` → `"a b"` |
| `s.normalizeSpace()` | Trim + collapse whitespace | `"  a   b  "` → `"a b"` |
| `s.stripSpace()` | Remove all whitespace | `"a b"` → `"ab"` |
| `s.stripHtml()` | Remove HTML tags | `"<b>hi</b>"` → `"hi"` |
| `s.digits()` | Keep only digits 0-9 | `"(123) 456"` → `"123456"` |
| `s.slug()` | Convert to URL slug | `"Hello World!"` → `"hello-world"` |
```

**CHEATSHEET.md** - Add sanitization section with pitfall warning:

```markdown
## String Sanitization
// Form input cleanup
"  hello   world  ".normalizeSpace()  // "hello world"
"<b>text</b>".stripHtml()             // "text"
"(555) 123-4567".digits()             // "5551234567"
"Blog Post Title!".slug()             // "blog-post-title"

// ⚠️ slug() removes accented characters, doesn't transliterate
"Café".slug()  // "caf" not "cafe"
```

---

## Validation Checklist
- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make build`
- [ ] Full validation: `make check`
- [ ] Documentation updated (reference.md, CHEATSHEET.md)
- [ ] Spec status updated to `implemented`

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| | Task 0: Regex vars | | |
| | Task 1: collapse() | | |
| | Task 2: normalizeSpace() | | |
| | Task 3: stripSpace() | | |
| | Task 4: stripHtml() | | |
| | Task 5: digits() | | |
| | Task 6: slug() | | |
| | Task 7: Update method list | | |
| | Task 8: Tests | | |
| | Task 9: Documentation | | |

## Deferred Items
- `s.transliterate()` — Complex accent→ASCII mapping; add later if needed (per spec)

## Notes
- All methods return new strings (immutable)
- Compile regex patterns once at package init for performance
- Consider adding compiled regex vars at package level:

```go
var (
    whitespaceRegex = regexp.MustCompile(`\s+`)
    htmlTagRegex    = regexp.MustCompile(`<[^>]*>`)
    nonDigitRegex   = regexp.MustCompile(`[^0-9]`)
    nonSlugRegex    = regexp.MustCompile(`[^a-z0-9]+`)
)
```
