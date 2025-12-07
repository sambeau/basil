---
id: PLAN-021
feature: FEAT-033
title: "Implementation Plan for String Sanitizer Methods"
status: draft
created: 2025-12-05
---

# Implementation Plan: FEAT-033 String Sanitizer Methods

## Overview
Add 6 string methods for common sanitization tasks when cleaning user input from web forms.

**Total methods: 6**
- Whitespace: `collapse()`, `normalizeSpace()`, `stripSpace()`
- Content: `stripHtml()`, `digits()`
- URL: `slug()`

## Prerequisites
- [x] Design decisions finalized (see FEAT-033)
- [x] Understand existing string method implementation (see `methods.go`)

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

### Task 1: Add collapse() Method
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Implement `collapse()` - replace runs of whitespace with single space:

```go
case "collapse":
    if len(args) != 0 {
        return newArityError("collapse", len(args), 0)
    }
    re := regexp.MustCompile(`\s+`)
    return &String{Value: re.ReplaceAllString(s.Value, " ")}
```

Tests:
- `"hello   world".collapse()` → `"hello world"`
- `"  hello   world  ".collapse()` → `" hello world "` (preserves edges)
- `"hello\t\nworld".collapse()` → `"hello world"`
- `"hello world".collapse()` → `"hello world"` (no change)
- `"".collapse()` → `""`

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
    re := regexp.MustCompile(`\s+`)
    collapsed := re.ReplaceAllString(s.Value, " ")
    return &String{Value: strings.TrimSpace(collapsed)}
```

Tests:
- `"  hello   world  ".normalizeSpace()` → `"hello world"`
- `"   ".normalizeSpace()` → `""`
- `"hello world".normalizeSpace()` → `"hello world"`
- `"\t\nhello\t\n".normalizeSpace()` → `"hello"`

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
    re := regexp.MustCompile(`\s+`)
    return &String{Value: re.ReplaceAllString(s.Value, "")}
```

Tests:
- `"hello world".stripSpace()` → `"helloworld"`
- `"  hello   world  ".stripSpace()` → `"helloworld"`
- `"hello".stripSpace()` → `"hello"`
- `"   ".stripSpace()` → `""`

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
    re := regexp.MustCompile(`<[^>]*>`)
    return &String{Value: re.ReplaceAllString(s.Value, "")}
```

Tests:
- `"<b>hello</b>".stripHtml()` → `"hello"`
- `"<p>Hello <b>world</b>!</p>".stripHtml()` → `"Hello world!"`
- `"<script>alert('x')</script>".stripHtml()` → `"alert('x')"`
- `"no tags".stripHtml()` → `"no tags"`
- `"<>".stripHtml()` → `""`
- `"a < b > c".stripHtml()` → `"a < b > c"` (not valid HTML tags)

Note: This is simple tag stripping, not a full HTML sanitizer. It removes anything between `<` and `>`.

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
    re := regexp.MustCompile(`[^0-9]`)
    return &String{Value: re.ReplaceAllString(s.Value, "")}
```

Tests:
- `"(123) 456-7890".digits()` → `"1234567890"`
- `"+1 (555) 123-4567".digits()` → `"15551234567"`
- `"Card: 4111-1111-1111-1111".digits()` → `"4111111111111111"`
- `"abc".digits()` → `""`
- `"a1b2c3".digits()` → `"123"`

---

### Task 6: Add slug() Method
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Medium

Implement `slug()` - convert to URL-safe slug:

```go
case "slug":
    if len(args) != 0 {
        return newArityError("slug", len(args), 0)
    }
    // Convert to lowercase
    result := strings.ToLower(s.Value)
    // Replace non-alphanumeric with hyphens
    re := regexp.MustCompile(`[^a-z0-9]+`)
    result = re.ReplaceAllString(result, "-")
    // Trim leading/trailing hyphens
    result = strings.Trim(result, "-")
    return &String{Value: result}
```

Tests:
- `"Hello World".slug()` → `"hello-world"`
- `"Hello World!".slug()` → `"hello-world"`
- `"  My Blog Post  ".slug()` → `"my-blog-post"`
- `"Product: iPhone 15 Pro".slug()` → `"product-iphone-15-pro"`
- `"What's New?".slug()` → `"whats-new"`
- `"---test---".slug()` → `"test"`
- `"Café".slug()` → `"caf"` (removes accented chars)
- `"".slug()` → `""`
- `"---".slug()` → `""`

---

### Task 7: Update Method List
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Add new methods to `stringMethods` slice:

```go
var stringMethods = []string{
    // ... existing methods ...
    "collapse", "normalizeSpace", "stripSpace", "stripHtml", "digits", "slug",
}
```

---

### Task 8: Documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small

Steps:
1. Add new string methods to reference.md String Methods section
2. Add sanitization examples to CHEATSHEET.md

---

## Validation Checklist
- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Documentation updated

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| | | | |

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
