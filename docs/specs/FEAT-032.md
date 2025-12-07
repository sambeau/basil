---
id: FEAT-032
title: "std/valid Standard Library Module"
status: implemented
priority: medium
created: 2025-12-05
implemented: 2025-12-05
author: "@human"
---

# FEAT-032: std/valid Standard Library Module

## Summary
Create a `std/valid` standard library module containing validation functions for form input, data sanitization, and common format checking. This module provides boolean validators that are simple, composable, and cover 90% of web form validation needs.

## User Story
As a web developer building small websites, I want a simple set of validators so that I can ensure user input from forms is valid before processing or storing it.

As an office worker automating tasks with a small web site, I want easy-to-use validation functions so that I can check data without writing complex regex patterns.

## Target Audience
1. Web developers building small websites
2. Office workers needing to automate something using a small web site

## Design Decisions

### 1. Return Type
**Decision**: All validators return booleans (`true`/`false`).
**Rationale**: Simple, composable with `and`/`or`, easy to understand. Users know what they're validating so don't need error messages from the validator.

### 2. Locale Parameter
**Decision**: Use locale codes (e.g., `"GB"`, `"US"`) for locale-specific validators rather than separate functions.
**Rationale**: Allows adding new locales without API changes. Start with `"ISO"`, `"US"`, `"GB"`.

### 3. Email Validation
**Decision**: Simple regex validation, not RFC 5322 compliant.
**Rationale**: Full email validation is impossibly complex. Basic format check catches 99% of typos.

### 4. What We Don't Validate
- **Names**: Impossible to validate correctly (see "Falsehoods Programmers Believe About Names")
- **Addresses**: Requires external database/services
- **Password Strength**: Security policies vary; opinionated
- **International Formats**: Start with US/GB; others can use `matches()`

## Specification

### Import
```parsley
let valid = import("std/valid")

// Use directly
if (valid.email(form.email)) { ... }

// Or destructure
let {email, minLen, between} = import("std/valid")
```

### Type Validators (6)

| Function | Description | Example |
|----------|-------------|---------|
| `string(x)` | Is it a string? | `valid.string("hello")` → `true` |
| `number(x)` | Is it an int or float? | `valid.number(42)` → `true` |
| `integer(x)` | Is it an integer? | `valid.integer(3.14)` → `false` |
| `boolean(x)` | Is it a boolean? | `valid.boolean(true)` → `true` |
| `array(x)` | Is it an array? | `valid.array([1,2])` → `true` |
| `dict(x)` | Is it a dictionary? | `valid.dict({a:1})` → `true` |

### String Validators (8)

| Function | Description | Example |
|----------|-------------|---------|
| `empty(x)` | Is string empty or whitespace-only? | `valid.empty("  ")` → `true` |
| `minLen(x, n)` | At least n characters? | `valid.minLen("hi", 3)` → `false` |
| `maxLen(x, n)` | At most n characters? | `valid.maxLen("hi", 10)` → `true` |
| `length(x, min, max)` | Length in range (inclusive)? | `valid.length("hello", 1, 10)` → `true` |
| `matches(x, pattern)` | Matches regex pattern? | `valid.matches("abc", "^[a-z]+$")` → `true` |
| `alpha(x)` | Letters only (a-z, A-Z)? | `valid.alpha("hello")` → `true` |
| `alphanumeric(x)` | Letters and digits only? | `valid.alphanumeric("abc123")` → `true` |
| `numeric(x)` | Numeric string (can parse to number)? | `valid.numeric("123.45")` → `true` |

### Number Validators (5)

| Function | Description | Example |
|----------|-------------|---------|
| `min(x, n)` | x >= n? | `valid.min(5, 1)` → `true` |
| `max(x, n)` | x <= n? | `valid.max(5, 10)` → `true` |
| `between(x, lo, hi)` | lo <= x <= hi (inclusive)? | `valid.between(5, 1, 10)` → `true` |
| `positive(x)` | x > 0? | `valid.positive(-1)` → `false` |
| `negative(x)` | x < 0? | `valid.negative(-1)` → `true` |

### Format Validators (7)

| Function | Description | Example |
|----------|-------------|---------|
| `email(x)` | Basic email format? | `valid.email("a@b.com")` → `true` |
| `url(x)` | Valid http/https URL? | `valid.url("https://example.com")` → `true` |
| `uuid(x)` | Valid UUID format (any version)? | `valid.uuid("550e8400-e29b-41d4-a716-446655440000")` → `true` |
| `phone(x)` | Looks like a phone number? | `valid.phone("+1 (555) 123-4567")` → `true` |
| `creditCard(x)` | Valid card format (Luhn check)? | `valid.creditCard("4111111111111111")` → `true` |
| `date(x, locale?)` | Valid date string? | See below |
| `time(x)` | Valid time (HH:MM or HH:MM:SS)? | `valid.time("14:30")` → `true` |

#### Date Validation with Locale

```parsley
valid.date("2024-12-25")           // ISO 8601 (default) - YYYY-MM-DD
valid.date("25/12/2024", "GB")     // DD/MM/YYYY
valid.date("12/25/2024", "US")     // MM/DD/YYYY
```

Supported date locales:
| Locale | Format | Example |
|--------|--------|---------|
| `"ISO"` (default) | YYYY-MM-DD | 2024-12-25 |
| `"US"` | MM/DD/YYYY | 12/25/2024 |
| `"GB"` | DD/MM/YYYY | 25/12/2024 |

Note: Validates that the date is real (e.g., `"2024-02-30"` → `false`).

### Locale-Aware Validators (2)

| Function | Description | Example |
|----------|-------------|---------|
| `postalCode(x, locale)` | Valid postal code for locale? | See below |
| `parseDate(x, locale)` | Parse date to ISO string (sanitizer) | See below |

#### Postal Code Validation

```parsley
valid.postalCode("SW1A 1AA", "GB")   // UK postcode → true
valid.postalCode("90210", "US")       // US ZIP → true
valid.postalCode("90210-1234", "US")  // US ZIP+4 → true
```

Supported postal code locales:
| Locale | Format | Example |
|--------|--------|---------|
| `"US"` | 5 digits or ZIP+4 | 90210, 90210-1234 |
| `"GB"` | UK postcode | SW1A 1AA, M1 1AA |

#### Date Parsing (Sanitizer)

Returns ISO date string or `null` if invalid:

```parsley
valid.parseDate("12/25/2024", "US")  // → "2024-12-25"
valid.parseDate("25/12/2024", "GB")  // → "2024-12-25"
valid.parseDate("invalid", "US")     // → null

// Use with time() function
let isoDate = valid.parseDate(form.date, "US")
if (isoDate) {
    let dt = time(isoDate)
}
```

### Collection Validators (2)

| Function | Description | Example |
|----------|-------------|---------|
| `contains(arr, x)` | Array contains value? | `valid.contains([1,2,3], 2)` → `true` |
| `oneOf(x, options)` | Value is one of options? | `valid.oneOf("red", ["red","green"])` → `true` |

## Usage Examples

### Basic Form Validation

```parsley
let valid = import("std/valid")

let errors = []

if (valid.empty(form.name)) {
    errors = errors ++ ["Name is required"]
}

if (not valid.email(form.email)) {
    errors = errors ++ ["Invalid email address"]
}

if (not valid.minLen(form.password, 8)) {
    errors = errors ++ ["Password must be at least 8 characters"]
}

if (not valid.between(form.age, 18, 120)) {
    errors = errors ++ ["Age must be between 18 and 120"]
}

if (len(errors) > 0) {
    // Show errors
} else {
    // Process form
}
```

### Combining Validators

```parsley
let valid = import("std/valid")

// Chain with and/or
let isValidAge = valid.integer(age) and valid.between(age, 0, 150)

// Validate enumerated values
let isValidStatus = valid.oneOf(status, ["pending", "active", "closed"])

// Custom pattern with matches()
let isProductCode = valid.matches(code, "^[A-Z]{2}-[0-9]{4}$")
```

### Locale-Aware Validation

```parsley
let valid = import("std/valid")

// US form
if (valid.date(form.dob, "US") and valid.postalCode(form.zip, "US")) {
    let dobIso = valid.parseDate(form.dob, "US")
    // Store in ISO format
}

// UK form  
if (valid.date(form.dob, "GB") and valid.postalCode(form.postcode, "GB")) {
    let dobIso = valid.parseDate(form.dob, "GB")
    // Store in ISO format
}
```

## Out of Scope

| Not Including | Reason |
|---------------|--------|
| Name validation | Impossible to do correctly |
| Address validation | Requires external services/databases |
| Password strength | Opinionated; security policies vary |
| File types/MIME | Different domain (file handling) |
| HTML/SQL injection | Should be handled by escaping at output |
| International postal codes | Start with US/GB; expand based on demand |

## Related: String Sanitizer Methods

These should be added as **string methods** (not in std/valid):

| Method | Description | Example |
|--------|-------------|---------|
| `s.collapse()` | Collapse multiple whitespace to single space | `"hello   world"` → `"hello world"` |
| `s.normalizeSpace()` | trim + collapse | `"  hello   world  "` → `"hello world"` |
| `s.stripSpace()` | Remove ALL whitespace | `"hello world"` → `"helloworld"` |
| `s.stripHtml()` | Remove HTML tags | `"<b>hi</b>"` → `"hi"` |
| `s.digits()` | Keep only digits | `"(123) 456-7890"` → `"1234567890"` |
| `s.slug()` | URL-safe slug | `"Hello World!"` → `"hello-world"` |

(String methods may be tracked in a separate feature.)

---

## Technical Context

### Affected Components
- `pkg/parsley/evaluator/stdlib_valid.go` — New file: validation module
- `pkg/parsley/evaluator/stdlib_table.go` — Register "valid" module
- `pkg/parsley/tests/stdlib_valid_test.go` — Comprehensive tests

### Dependencies
- Depends on: None (standalone module)
- Blocks: None

### Edge Cases & Constraints

1. **Empty strings**: Type validators should handle `""` (returns `true` for `string()`)
2. **Null values**: All validators should return `false` for `null` input
3. **Unicode**: `alpha()` and `alphanumeric()` should be ASCII-only for predictability
4. **Regex errors**: `matches()` with invalid regex should return error, not false
5. **Phone numbers**: Very loose validation (allows international formats)
6. **Credit cards**: Luhn algorithm + length check, not issuer validation

## Implementation Notes
*Added during/after implementation*

## References
- PHP Laminas Validator: https://docs.laminas.dev/laminas-validator/
- CakePHP Validation: https://api.cakephp.org/4.6/class-Cake.Validation.Validator.html
- "Falsehoods Programmers Believe About Names": https://www.kalzumeus.com/2010/06/17/falsehoods-programmers-believe-about-names/

## Related
- Plan: `docs/plans/FEAT-032-plan.md` (to be created)
