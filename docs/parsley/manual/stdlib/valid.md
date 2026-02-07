---
id: man-pars-std-valid
title: "@std/valid"
system: parsley
type: stdlib
name: valid
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - validation
  - validator
  - type check
  - email
  - URL
  - UUID
  - phone
  - credit card
  - format
  - string
  - number
---

# @std/valid

Validation predicates that return `true` or `false`. All validators are pure functions with no side effects — they check values without modifying them.

```parsley
let valid = import @std/valid
```

## Type Validators

| Function | Description |
|---|---|
| `string(v)` | Is string? |
| `number(v)` | Is number (integer or float)? |
| `integer(v)` | Is integer? |
| `boolean(v)` | Is boolean? |
| `array(v)` | Is array? |
| `dict(v)` | Is dictionary? |

```parsley
valid.string("hello")               // true
valid.string(42)                     // false
valid.number(3.14)                   // true
valid.integer(3.14)                  // false
valid.integer(3)                     // true
valid.boolean(true)                  // true
valid.array([1, 2])                  // true
valid.dict({a: 1})                   // true
```

## String Validators

| Function | Args | Description |
|---|---|---|
| `empty(s)` | string | Is empty or whitespace only? |
| `minLen(s, n)` | string, integer | Has at least `n` characters? |
| `maxLen(s, n)` | string, integer | Has at most `n` characters? |
| `length(s, min, max)` | string, integer, integer | Length in range `[min, max]`? |
| `matches(s, pattern)` | string, string or regex | Matches pattern? |
| `alpha(s)` | string | Only letters? |
| `alphanumeric(s)` | string | Only letters and digits? |
| `numeric(s)` | string | Only digits? |

```parsley
valid.empty("")                      // true
valid.empty("   ")                   // true
valid.empty("hi")                    // false
valid.minLen("hello", 3)             // true
valid.maxLen("hi", 5)                // true
valid.length("hello", 3, 10)         // true
valid.alpha("Hello")                 // true
valid.alpha("Hello123")              // false
valid.alphanumeric("abc123")         // true
valid.numeric("12345")               // true
valid.matches("hello", /^h/)         // true
```

## Number Validators

| Function | Args | Description |
|---|---|---|
| `min(n, min)` | number, number | At least `min`? |
| `max(n, max)` | number, number | At most `max`? |
| `between(n, min, max)` | number, number, number | In range `[min, max]`? |
| `positive(n)` | number | Greater than 0? |
| `negative(n)` | number | Less than 0? |

```parsley
valid.positive(5)                    // true
valid.positive(-1)                   // false
valid.negative(-3)                   // true
valid.min(10, 5)                     // true
valid.max(10, 20)                    // true
valid.between(10, 5, 15)             // true
valid.between(10, 20, 30)            // false
```

## Format Validators

| Function | Args | Description |
|---|---|---|
| `email(s)` | string | Valid email format? |
| `url(s)` | string | Valid URL format? |
| `uuid(s)` | string | Valid UUID format? |
| `phone(s, locale?)` | string, string? | Valid phone number? |
| `creditCard(s)` | string | Valid credit card number (Luhn check)? |
| `date(s, format?)` | string, string? | Valid date? |
| `time(s)` | string | Valid time (HH:MM or HH:MM:SS)? |
| `postalCode(s, locale?)` | string, string? | Valid postal code? |
| `parseDate(s, format?)` | string, string? | Parse date string, return dictionary or null |

```parsley
valid.email("user@example.com")      // true
valid.email("invalid")               // false
valid.url("https://example.com")     // true
valid.uuid("550e8400-e29b-41d4-a716-446655440000")  // true
valid.phone("+1-555-123-4567")       // true
valid.phone("07911 123456", "gb")    // true
valid.creditCard("4111111111111111")  // true
valid.time("14:30")                  // true
valid.time("14:30:59")               // true
valid.date("2024-06-15")             // true
valid.postalCode("90210", "us")      // true
valid.postalCode("SW1A 1AA", "gb")   // true
```

### Locale Support

`phone` and `postalCode` accept an optional locale string. Supported locales:

| Locale | Phone | Postal Code |
|---|---|---|
| `"us"` | US format | 5-digit or ZIP+4 |
| `"gb"` | UK format | UK postcode pattern |

Without a locale, the default international pattern is used.

### `parseDate`

`parseDate` returns a dictionary with parsed date components, or `null` if the string is not a valid date:

```parsley
valid.parseDate("2024-06-15")        // {year: 2024, month: 6, day: 15}
valid.parseDate("invalid")           // null
```

Supported formats: `"iso"` (default, YYYY-MM-DD), `"us"` (MM/DD/YYYY), `"gb"` (DD/MM/YYYY).

## Collection Validators

| Function | Args | Description |
|---|---|---|
| `contains(arr, item)` | array, any | Array contains item? |
| `oneOf(value, options)` | any, array | Value is one of the options? |

```parsley
valid.contains([1, 2, 3], 2)        // true
valid.contains([1, 2, 3], 5)        // false
valid.oneOf("red", ["red", "green", "blue"])  // true
valid.oneOf("pink", ["red", "green", "blue"]) // false
```

## Common Patterns

### Form Validation

```parsley
let valid = import @std/valid

let errors = []
if (!valid.email(input.email)) {
    errors = errors ++ ["Invalid email address"]
}
if (!valid.minLen(input.password, 8)) {
    errors = errors ++ ["Password must be at least 8 characters"]
}
if (!valid.oneOf(input.role, ["user", "admin"])) {
    errors = errors ++ ["Invalid role"]
}
```

### Composing Validators

```parsley
let isValidUsername = fn(s) {
    valid.string(s) and valid.minLen(s, 3) and valid.maxLen(s, 20) and valid.alphanumeric(s)
}

isValidUsername("alice123")          // true
isValidUsername("ab")                // false (too short)
```

## Key Differences from Other Languages

- **Pure predicates** — every function returns a boolean. No exceptions, no error objects. Use schema validation for structured error reporting.
- **No mutation** — validators never modify the input value.
- **Locale-aware** — phone numbers and postal codes support locale-specific patterns via an optional second argument.

## See Also

- [Data Model](../fundamentals/data-model.md) — schema-based validation with error messages
- [Types](../fundamentals/types.md) — Parsley's type system and `typeof`
- [Strings](../builtins/strings.md) — string methods including `.length()` and regex matching