---
id: PLAN-020
feature: FEAT-032
title: "Implementation Plan for std/valid"
status: complete
created: 2025-12-05
completed: 2025-12-05
---

# Implementation Plan: FEAT-032 std/valid

## Overview
Create the `std/valid` standard library module with validation functions for form input, data checking, and common format validation.

**Total functions: 30**
- Type validators: 6
- String validators: 8
- Number validators: 5
- Format validators: 7
- Locale-aware validators: 2
- Collection validators: 2

## Prerequisites
- [x] Design decisions finalized (see FEAT-032)
- [x] Understand existing stdlib loading mechanism (see `stdlib_table.go`, `stdlib_math.go`)

## Existing Stdlib Pattern
Follow the pattern from `stdlib_math.go`:

```go
// In getStdlibModules() in stdlib_table.go
"valid": loadValidModule,

// New file: stdlib_valid.go
func loadValidModule(env *Environment) Object {
    return &StdlibModuleDict{
        Exports: map[string]Object{
            "string":  &Builtin{Fn: validString},
            "email":   &Builtin{Fn: validEmail},
            // ...
        },
    }
}
```

## Error Handling
Validators return `true` or `false` - they don't produce errors for invalid input.

Errors only occur for:
- Wrong argument types (e.g., `valid.minLen(123, 5)` - first arg not string)
- Wrong argument count
- Invalid regex in `matches()`

Use existing error helpers:
```go
newTypeError(code, function, expected, got)
newArityError(function, got, want)
```

## Tasks

### Task 1: Create Module Structure
**Files**: `pkg/parsley/evaluator/stdlib_valid.go` (new), `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Small

Steps:
1. Create `pkg/parsley/evaluator/stdlib_valid.go`
2. Define `loadValidModule()` returning `StdlibModuleDict`
3. Register "valid" in `getStdlibModules()` in `stdlib_table.go`

Tests:
- `import("std/valid")` succeeds
- Module has expected exports

---

### Task 2: Type Validators (6 functions)
**Files**: `pkg/parsley/evaluator/stdlib_valid.go`
**Estimated effort**: Small

Implement:
- `string(x)` - returns true if x is a string
- `number(x)` - returns true if x is int or float
- `integer(x)` - returns true if x is int
- `boolean(x)` - returns true if x is boolean
- `array(x)` - returns true if x is array
- `dict(x)` - returns true if x is dictionary

```go
func validString(args ...Object) Object {
    if len(args) != 1 {
        return newArityError("valid.string", len(args), 1)
    }
    _, ok := args[0].(*String)
    return nativeBoolToBooleanObject(ok)
}
```

Tests:
- Each type validator returns true for correct type
- Each returns false for other types
- Each returns false for null

---

### Task 3: String Validators (8 functions)
**Files**: `pkg/parsley/evaluator/stdlib_valid.go`
**Estimated effort**: Medium

Implement:
- `empty(x)` - true if string is empty or whitespace-only
- `minLen(x, n)` - true if len(x) >= n
- `maxLen(x, n)` - true if len(x) <= n
- `length(x, min, max)` - true if min <= len(x) <= max
- `matches(x, pattern)` - true if x matches regex pattern
- `alpha(x)` - true if x contains only letters a-z, A-Z
- `alphanumeric(x)` - true if x contains only letters and digits
- `numeric(x)` - true if x is a numeric string (parseable)

```go
func validEmpty(args ...Object) Object {
    if len(args) != 1 {
        return newArityError("valid.empty", len(args), 1)
    }
    str, ok := args[0].(*String)
    if !ok {
        return FALSE
    }
    return nativeBoolToBooleanObject(strings.TrimSpace(str.Value) == "")
}
```

Tests:
- `empty("")` → true, `empty("  ")` → true, `empty("a")` → false
- `minLen("hello", 3)` → true, `minLen("hi", 3)` → false
- `maxLen("hi", 10)` → true, `maxLen("hello world", 5)` → false
- `length("hello", 1, 10)` → true
- `matches("abc", "^[a-z]+$")` → true
- `alpha("hello")` → true, `alpha("hello1")` → false
- `alphanumeric("abc123")` → true
- `numeric("123.45")` → true, `numeric("abc")` → false

---

### Task 4: Number Validators (5 functions)
**Files**: `pkg/parsley/evaluator/stdlib_valid.go`
**Estimated effort**: Small

Implement:
- `min(x, n)` - true if x >= n
- `max(x, n)` - true if x <= n
- `between(x, lo, hi)` - true if lo <= x <= hi
- `positive(x)` - true if x > 0
- `negative(x)` - true if x < 0

Tests:
- `min(5, 1)` → true, `min(0, 1)` → false
- `max(5, 10)` → true, `max(15, 10)` → false
- `between(5, 1, 10)` → true
- `positive(1)` → true, `positive(0)` → false
- `negative(-1)` → true, `negative(0)` → false

---

### Task 5: Basic Format Validators (4 functions)
**Files**: `pkg/parsley/evaluator/stdlib_valid.go`
**Estimated effort**: Medium

Implement:
- `email(x)` - basic email format check
- `url(x)` - valid http/https URL
- `uuid(x)` - valid UUID format
- `time(x)` - valid time string (HH:MM or HH:MM:SS)

Regex patterns:
```go
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var timeRegex = regexp.MustCompile(`^([01]?[0-9]|2[0-3]):[0-5][0-9](:[0-5][0-9])?$`)
```

Tests:
- `email("test@example.com")` → true
- `email("invalid")` → false
- `url("https://example.com")` → true
- `uuid("550e8400-e29b-41d4-a716-446655440000")` → true
- `time("14:30")` → true, `time("14:30:00")` → true

---

### Task 6: Phone and Credit Card Validators (2 functions)
**Files**: `pkg/parsley/evaluator/stdlib_valid.go`
**Estimated effort**: Medium

Implement:
- `phone(x)` - loose phone number check (digits, spaces, +, -, parens)
- `creditCard(x)` - Luhn algorithm check + length validation

```go
// Luhn algorithm for credit card validation
func luhnCheck(number string) bool {
    // Strip non-digits
    // Apply Luhn algorithm
    // Return true if valid
}
```

Tests:
- `phone("+1 (555) 123-4567")` → true
- `phone("555-1234")` → true
- `creditCard("4111111111111111")` → true (Visa test number)
- `creditCard("1234567890123456")` → false (fails Luhn)

---

### Task 7: Date Validator with Locale (1 function)
**Files**: `pkg/parsley/evaluator/stdlib_valid.go`
**Estimated effort**: Medium

Implement:
- `date(x, locale?)` - validate date string with optional locale

Supported formats:
- `"ISO"` (default): YYYY-MM-DD
- `"US"`: MM/DD/YYYY
- `"GB"`: DD/MM/YYYY

Must validate that date is real (not just format):
- `"2024-02-30"` → false (Feb 30 doesn't exist)
- `"2024-02-29"` → true (2024 is leap year)

```go
func validDate(args ...Object) Object {
    // Parse locale (default "ISO")
    // Parse date according to format
    // Validate date is real using time.Parse
}
```

Tests:
- `date("2024-12-25")` → true
- `date("2024-02-30")` → false
- `date("25/12/2024", "GB")` → true
- `date("12/25/2024", "US")` → true
- `date("25/12/2024", "US")` → false (month 25 invalid)

---

### Task 8: Postal Code Validator with Locale (1 function)
**Files**: `pkg/parsley/evaluator/stdlib_valid.go`
**Estimated effort**: Small

Implement:
- `postalCode(x, locale)` - validate postal code for locale

Patterns:
```go
var postalCodePatterns = map[string]*regexp.Regexp{
    "US": regexp.MustCompile(`^\d{5}(-\d{4})?$`),
    "GB": regexp.MustCompile(`^[A-Z]{1,2}[0-9][0-9A-Z]?\s?[0-9][A-Z]{2}$`),
}
```

Tests:
- `postalCode("90210", "US")` → true
- `postalCode("90210-1234", "US")` → true
- `postalCode("SW1A 1AA", "GB")` → true
- `postalCode("M1 1AA", "GB")` → true

---

### Task 9: parseDate Sanitizer (1 function)
**Files**: `pkg/parsley/evaluator/stdlib_valid.go`
**Estimated effort**: Medium

Implement:
- `parseDate(x, locale)` - parse date to ISO string or return null

```go
func validParseDate(args ...Object) Object {
    // Parse according to locale
    // If valid, return ISO string "YYYY-MM-DD"
    // If invalid, return NULL
}
```

Tests:
- `parseDate("12/25/2024", "US")` → `"2024-12-25"`
- `parseDate("25/12/2024", "GB")` → `"2024-12-25"`
- `parseDate("invalid", "US")` → `null`
- `parseDate("02/30/2024", "US")` → `null` (invalid date)

---

### Task 10: Collection Validators (2 functions)
**Files**: `pkg/parsley/evaluator/stdlib_valid.go`
**Estimated effort**: Small

Implement:
- `contains(arr, x)` - true if array contains value
- `oneOf(x, options)` - true if x is one of options

```go
func validContains(args ...Object) Object {
    // Check if array contains value using equality comparison
}

func validOneOf(args ...Object) Object {
    // Check if value is in options array
}
```

Tests:
- `contains([1, 2, 3], 2)` → true
- `contains(["a", "b"], "c")` → false
- `oneOf("red", ["red", "green", "blue"])` → true
- `oneOf("yellow", ["red", "green", "blue"])` → false

---

### Task 11: Documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small

Steps:
1. Add `std/valid` section to reference.md
2. Add validation examples to CHEATSHEET.md

---

## Validation Checklist
- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Documentation updated
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| | | | |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- Additional locales for date/postalCode (DE, FR, CA, AU, etc.)
- datetime validator (combined date + time)
- ISBN validator
- IBAN validator
