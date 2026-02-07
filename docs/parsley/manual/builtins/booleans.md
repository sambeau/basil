---
id: man-pars-booleans
title: Booleans & Null
system: parsley
type: builtin
name: booleans
created: 2026-02-05
version: 0.2.0
author: Basil Team
keywords:
  - boolean
  - null
  - true
  - false
  - truthiness
  - logical
  - operators
  - null coalescing
---

# Booleans & Null

Parsley has three special literal values: `true`, `false`, and `null`. Booleans drive control flow; `null` represents the intentional absence of a value.

```parsley
let active = true
let deleted = false
let nickname = null       // no value
```

Accessing a missing dictionary key returns `null` without error:

```parsley
let person = {name: "Alice"}
person.age                    // null
```

## Truthiness

Parsley uses Python-style truthiness. The following values are **falsy**:

| Value | Type |
|-------|------|
| `false` | Boolean |
| `null` | Null |
| `0` | Integer |
| `0.0` | Float |
| `""` | Empty string |
| `[]` | Empty array |
| `{}` | Empty dictionary |

**Everything else is truthy** — non-zero numbers, non-empty strings, non-empty collections.

```parsley
if (username) { "has username" }  // fails for ""
if (items) { "has items" }        // fails for []
if (config) { "has config" }      // fails for {}
if (count) { "non-zero" }         // fails for 0
```

> ⚠️ **Unlike JavaScript**, empty arrays `[]` and empty dictionaries `{}` are **falsy** in Parsley. This matches Python — you can write `if (items) { ... }` to guard against empty collections without calling `.length()`.

## Operators

### Negation: `!` / `not`

The `!` operator inverts truthiness and always returns a boolean. The `not` keyword is an identical alias:

```parsley
!true               // false
!null               // true
not ""              // true
```

### And / Or: `&&` / `||` and aliases

Standard boolean logic. Parsley offers English aliases — `and` for `&&`, `or` for `||`, `&` for `&&`, `|` for `||`:

```parsley
true && false       // false
true and true       // true
false || true       // true
false or false      // false
```

> ⚠️ **Array overloads:** When both operands are **arrays**, `&&` performs set **intersection** and `||` performs set **union**. See the [Arrays](array.md) manual page.

### Null Coalescing: `??`

Returns the left-hand value unless it is `null`, in which case it evaluates and returns the right. This is **short-circuit** — the right side is only evaluated when needed.

```parsley
null ?? "default"       // "default"
"value" ?? "default"    // "value"
```

Crucially, `??` triggers **only on `null`** — not on other falsy values:

```parsley
0 ?? "default"          // 0
"" ?? "default"         // ""
false ?? "default"      // false
```

This makes `??` ideal for providing defaults when a value might be absent, without replacing legitimate falsy values. You can chain it for multi-level fallbacks:

```parsley
let theme = config.theme ?? config.defaultTheme ?? "light"
```

### Truthiness vs. `??`

This is a common source of confusion. Use `if` when you want to catch **all falsy values**; use `??` when you only want to replace **`null`**:

```parsley
// Truthiness — replaces 0, "", [], {}, null, false
let label = if (0) { 0 } else { "none" }   // "none"

// Null coalescing — only replaces null
0 ?? "none"                                  // 0
```

## Equality

Standard `==` and `!=`. There is no `===` operator — Parsley has no strict-equality variant:

```parsley
true == true        // true
null == null        // true
true != false       // true
```

## Methods

Booleans and null have minimal methods:

| Method | Description |
|--------|-------------|
| `.type()` | Returns `"boolean"` or `"null"` |
| `.toBox()` | Box-formatted string for display |

```parsley
true.type()         // "boolean"
null.type()         // "null"
```

## Operator Precedence

Logical operators follow this precedence (lowest to highest):

| Level | Operators |
|-------|-----------|
| 1 | `??`, `\|\|`, `or` |
| 2 | `&&`, `and` |
| 3 | `==`, `!=` |
| 8 | `!`, `not` (prefix) |

Use parentheses to clarify intent when mixing operators:

```parsley
let a = true
let b = false
let c = true
(a || b) && c       // true
```

## Key Differences from Other Languages

- **Empty collections are falsy:** `[]` and `{}` are falsy (unlike JavaScript, like Python)
- **`??` is null-only:** Only triggers on `null`, not `false`, `0`, or `""`
- **No `===`:** Use `==` — there is no strict-equality variant
- **`not`/`and`/`or` are aliases:** Identical to `!`/`&&`/`||` — use whichever you prefer
- **`&&`/`||` on arrays:** Performs set intersection/union, not boolean logic

## See Also

- [Numbers](numbers.md) — numeric types and arithmetic
- [Strings](strings.md) — text values and interpolation
- [Control Flow](../fundamentals/control-flow.md) — `if`/`else` and `for` expressions
- [Operators](../fundamentals/operators.md) — complete operator reference
- [Error Handling](../fundamentals/errors.md) — `try`, `check`, and error patterns
- [Types](../fundamentals/types.md) — truthiness rules and type coercion