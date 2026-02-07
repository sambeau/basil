---
id: man-pars-regex
title: Regex
system: parsley
type: builtins
name: regex
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - regex
  - regular expression
  - pattern
  - match
  - replace
  - test
  - flags
  - capture group
---

# Regex

Regex values represent regular expressions. They are created from literals (`/pattern/flags`) or the `regex()` builtin, and are used for matching, replacing, and splitting strings.

## Literals

```parsley
let r = /hello/
let digits = /\d+/
let email = /\w+@\w+\.\w+/i
```

Flags follow the closing `/`:

| Flag | Meaning |
|---|---|
| `i` | Case-insensitive |
| `m` | Multi-line (`^` and `$` match line boundaries) |
| `s` | Dotall (`.` matches newline) |
| `g` | Global (match all occurrences — used by operators and methods) |

```parsley
let r = /pattern/igs              // multiple flags
```

## regex() Builtin

Create a regex from strings — useful when the pattern is dynamic:

```parsley
let r = regex("\\d+", "g")
let pattern = "hello"
let r2 = regex(pattern, "i")
```

> ⚠️ Backslashes must be doubled in strings (`"\\d+"`) but not in literals (`/\d+/`). Prefer literals for static patterns.

## Match Operator — `~`

The `~` operator tests a string against a regex and returns an array of matches (or `null` if no match). Element `[0]` is the full match; subsequent elements are capture groups:

```parsley
"hello123" ~ /(\w+?)(\d+)/      // ["hello123", "hello", "123"]
"no match" ~ /\d+/              // null
```

Because `null` is falsy and a match array is truthy, `~` works directly in conditions:

```parsley
if ("test@example.com" ~ /\w+@\w+/) {
    "valid-ish email"
}
```

### Extracting Captures

```parsley
let m = "2026-02-06" ~ /(\d{4})-(\d{2})-(\d{2})/
m[0]                             // "2026-02-06"
m[1]                             // "2026"
m[2]                             // "02"
m[3]                             // "06"
```

## Not-Match Operator — `!~`

Returns `true` when the string does **not** match:

```parsley
"hello" !~ /\d+/                 // true
"hello123" !~ /\d+/              // false
```

## Properties

| Property | Type | Description |
|---|---|---|
| `.pattern` | string | The regex pattern string |
| `.flags` | string | The flag characters |

```parsley
let r = /\d+/gi
r.pattern                        // "\\d+"
r.flags                          // "gi"
```

## Methods

### .test(str)

Returns `true` if the pattern matches anywhere in the string:

```parsley
let digits = /\d+/
digits.test("hello123")          // true
digits.test("hello")             // false
```

### .replace(str, replacement)

Replace matches in a string. Without `g` flag, replaces only the first match. With `g`, replaces all:

```parsley
let r = /\d+/g
r.replace("abc123def456", "X")   // "abcXdefX"

let first = /\d+/
first.replace("abc123def456", "X")  // "abcXdef456"
```

Replacement can be a function that receives the match and returns the replacement:

```parsley
let r = /[a-z]+/g
r.replace("hello WORLD", fn(m) { m.toUpper() })
// "HELLO WORLD"
```

### .format(style?)

Format the regex for display:

```parsley
let r = /\d+/g
r.format()                       // "/\\d+/g"
r.format("pattern")              // "\\d+"
r.format("verbose")              // pattern and flags separately
```

### .toDict() / .inspect()

```parsley
let r = /\d+/gi
r.toDict()                       // {pattern: "\\d+", flags: "gi"}
r.inspect()                      // {__type: "regex", pattern: "\\d+", flags: "gi"}
```

## String Methods with Regex

Several string methods accept regex arguments:

```parsley
"hello world" ~ /wo\w+/         // ["world"]

"abc123def".replace(/\d+/, "X") // "abcXdef"
"abc123def".replace(/\d+/g, "X") // "abcXdefX" (non-standard — use regex method)

"a,b,,c".split(/,+/)            // ["a", "b", "c"]
```

The `.replace()` string method also supports function replacement:

```parsley
"hello world".replace(/\w+/g, fn(m) { m.toTitle() })
// "Hello World"
```

## Common Patterns

### Validation

```parsley
let isEmail = fn(s) { s ~ /^[\w.+-]+@[\w-]+\.[\w.]+$/ != null }
let isNumeric = fn(s) { s ~ /^\d+$/ != null }
```

### Extract All Matches

Use the `g` flag with `~` or the `.matchAll()` approach:

```parsley
let text = "Call 555-1234 or 555-5678"
let numbers = text ~ /\d{3}-\d{4}/g
// Returns all matches when g flag is present
```

### Named Capture Groups

Go-style named captures with `(?P<name>...)`:

```parsley
let m = "2026-02-06" ~ /(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})/
// Access by index: m[1], m[2], m[3]
```

### Search and Replace

```parsley
let clean = /\s+/g
clean.replace("  too   many   spaces  ", " ")
// " too many spaces "
```

## Key Differences from Other Languages

- **`~` returns an array or null** — not a boolean. Use `!~` for a boolean "does not match" test, or check `!= null` for "does match" as a boolean.
- **`g` flag controls global matching** — without it, `~` and `.replace()` operate on the first match only.
- **No regex literal in arbitrary expression position** — regex literals must be assigned to a variable or used on the right side of `~` / `!~`. Use `regex()` for dynamic patterns.
- **Go regex engine** — Parsley uses Go's `regexp` package (RE2 syntax). No backreferences, no lookahead/lookbehind.
- **Named groups use `(?P<name>...)`** — the Go/Python syntax, not the JavaScript `(?<name>...)` syntax.

## See Also

- [Strings](strings.md) — `.replace()`, `.split()`, and `.match()` methods that accept regex
- [Operators](../fundamentals/operators.md) — `~` and `!~` match operators
- [Variables & Binding](../fundamentals/variables.md) — destructuring match results