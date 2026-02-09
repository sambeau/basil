---
id: man-pars-strings
title: Strings
system: parsley
type: builtin
name: strings
created: 2026-02-05
version: 0.2.0
author: Basil Team
keywords:
  - string
  - template
  - raw string
  - interpolation
  - text
  - methods
---

# Strings

Parsley has **three distinct string types**, each with different interpolation and escaping rules. Choosing the right one avoids unnecessary escaping and makes intent clear.

```parsley
"Hello, World!\n"               // double-quoted: escape sequences, no interpolation
`Hello, {name}!`                // template: interpolation with {expr}
'C:\Users\raw \n stays'         // raw: no escapes, interpolation with @{expr}
```

## String Types

### Double-Quoted Strings (`"..."`)

Standard strings with escape sequences. **No interpolation** — braces are literal characters.

```parsley
"Line 1\nLine 2"                // newline between lines
"She said \"hello\""            // escaped quotes
"Tab\there"                     // tab character
```

| Escape | Meaning |
|--------|---------|
| `\n` | Newline |
| `\t` | Tab |
| `\r` | Carriage return |
| `\\` | Literal backslash |
| `\"` | Literal double quote |

### Template Strings (`` `...` ``)

Interpolated strings using `{expression}` — any valid Parsley expression works inside the braces. No escape sequences are processed.

```parsley
let name = "Alice"
`Hello, {name}!`               // "Hello, Alice!"
`2 + 2 = {2 + 2}`              // "2 + 2 = 4"
`{name.toUpper()}`             // "ALICE"
```

> ⚠️ Parsley uses `{expr}`, **not** `${expr}`. The dollar sign is not part of the syntax — this is the most common mistake when coming from JavaScript.

### Raw Strings (`'...'`)

Backslashes are literal — no escape sequences. Interpolation uses `@{expression}`.

```parsley
'C:\Users\name'                 // backslashes are literal
'regex: \d+\.\d+'              // no escaping needed
let id = 42
'id = @{id}'                    // "id = 42"
```

Raw strings are ideal for file paths, regex patterns, SQL, and templates. Use `\@` to escape a literal `@` when followed by `{`.

#### Raw Strings in `<script>` and `<style>` Tags

The content inside `<script>` and `<style>` tags uses the same raw string rules — braces `{` and `}` are **literal characters**, and interpolation uses `@{expr}`. This is by design: CSS and JavaScript both use `{` `}` as core syntax (CSS rule blocks, JS code blocks), so treating them as literal avoids conflicts:

```parsley
<style>
    .card { border: 1px solid #ccc; }
    .card:hover { background: lightblue; }
</style>
```

Use `@{expr}` when you need dynamic values:

```parsley
let accent = "tomato"
<style>
    .highlight { color: @{accent}; }
</style>

let endpoint = "/api/data"
<script>
    fetch("@{endpoint}").then(function(r) { return r.json(); });
</script>
```

### Choosing a String Type

| Need | Use | Why |
|------|-----|-----|
| Static text with special chars (`\n`, `\t`) | `"..."` | Escape sequences processed |
| Dynamic text with expressions | `` `...` `` | `{expr}` interpolation |
| Paths, regex, templates | `'...'` | Backslashes literal, `@{expr}` interpolation |

## Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `+` | Concatenation | `"Hello" + " " + "World"` → `"Hello World"` |
| `*` | Repetition | `"ab" * 3` → `"ababab"` |
| `in` | Substring test | `"ell" in "hello"` → `true` |
| `not in` | Negated substring | `"xyz" not in "hello"` → `true` |
| `==`, `!=` | Equality | `"a" == "a"` → `true` |
| `<`, `>`, `<=`, `>=` | Comparison (natural sort) | `"file2" < "file10"` → `true` |
| `~` | Regex match (returns array or null) | `"abc123" ~ /\d+/` → `["123"]` |
| `!~` | Regex no-match (returns boolean) | `"hello" !~ /\d+/` → `true` |

> ⚠️ **Natural sort order:** String comparisons use natural ordering, so `"file2" < "file10"` is `true` (not lexicographic where `"file10" < "file2"`). This is almost always what you want but differs from most languages.

> ⚠️ **`++` does not concatenate strings.** It wraps both sides into an array: `"a" ++ "b"` → `["a", "b"]`. Use `+` for string concatenation.

## Indexing & Slicing

Strings support integer indexing (0-based) and slicing, including negative indices:

```parsley
let s = "hello"
s[0]                            // "h"
s[-1]                           // "o" (last character)
s[1:3]                          // "el" (start inclusive, end exclusive)
s[:2]                           // "he" (first 2)
s[2:]                           // "llo" (from index 2)
s[?99]                          // null (optional access, no error)
```

## Methods

### Case & Formatting

| Method | Description |
|--------|-------------|
| `.toUpper()` | Uppercase: `"hello".toUpper()` → `"HELLO"` |
| `.toLower()` | Lowercase: `"HELLO".toLower()` → `"hello"` |
| `.toTitle()` | Title case: `"hello world".toTitle()` → `"Hello World"` |
| `.slug()` | URL-safe slug: `"Hello World!".slug()` → `"hello-world"` |

### Whitespace

| Method | Description |
|--------|-------------|
| `.trim()` | Remove leading/trailing whitespace |
| `.collapse()` | Collapse runs of whitespace to single spaces |
| `.normalizeSpace()` | Collapse + trim (combines both) |
| `.stripSpace()` | Remove all whitespace entirely |
| `.indent(n)` | Add `n` spaces to the start of each non-blank line |
| `.outdent()` | Remove the common leading indent from all lines |

```parsley
"  hello  world  ".normalizeSpace()   // "hello world"
"  hello  world  ".stripSpace()       // "helloworld"
```

### Search & Transform

| Method | Description |
|--------|-------------|
| `.length()` | Character count (Unicode-aware): `"café".length()` → `4` |
| `.includes(sub)` | Contains substring: `"hello".includes("ell")` → `true` |
| `.split(delim)` | Split into array: `"a,b,c".split(",")` → `["a", "b", "c"]` |
| `.replace(old, new)` | Replace all occurrences: `"hello".replace("l", "L")` → `"heLLo"` |
| `.digits()` | Extract only digits: `"abc123def".digits()` → `"123"` |

The `replace` method also accepts a **regex** as the first argument and a **function** as the second. With a regex, only the **first match** is replaced by default — add the `g` flag for global replacement:

```parsley
"hello".replace("l", "L")                // "heLLo" (string: replaces all)
"hello".replace(/l/, "L")                // "heLlo" (regex: first match only)
"hello".replace(/l/g, "L")               // "heLLo" (regex + g flag: all matches)
"hello world".replace(/\w+/g, fn(m) { m.toTitle() })  // "Hello World"
```

### HTML

| Method | Description |
|--------|-------------|
| `.htmlEncode()` | Escape `<`, `>`, `&`, `"` for safe HTML output |
| `.htmlDecode()` | Decode HTML entities back to characters |
| `.stripHtml()` | Remove all HTML tags |
| `.paragraphs()` | Convert blank-line-separated text to `<p>` tags |
| `.highlight(phrase, tag?)` | Wrap matches in HTML tag (default: `<mark>`) |

```parsley
"hello & world".htmlEncode()          // "hello &amp; world"
"<b>hello</b>".stripHtml()            // "hello"
```

### URL Encoding

| Method | Description |
|--------|-------------|
| `.urlEncode()` | Query-string encode (spaces → `+`) |
| `.urlDecode()` | Decode URL-encoded string |
| `.urlPathEncode()` | Encode path segments (`/` → `%2F`) |
| `.urlQueryEncode()` | Encode query values (`&`, `=` encoded) |

### Parsing

| Method | Returns | Description |
|--------|---------|-------------|
| `.parseJSON()` | any | Parse string as JSON into Parsley values |
| `.parseCSV(hasHeader?)` | table | Parse CSV (default: first row is header) |
| `.parseMarkdown(opts?)` | dictionary | Returns `{html, md, raw}` from Markdown source |

```parsley
let data = '{"name": "Bob"}'.parseJSON()
data.name                               // "Bob"
let doc = "# Title\n\nBody".parseMarkdown()
doc.html                                // "<h1>Title</h1>\n<p>Body</p>\n"
```

### Templating

#### .render(dict?)

Evaluates `@{expr}` placeholders in a string using values from the provided dictionary. Use **double-quoted strings** to hold the template (they preserve the `@{...}` syntax literally):

```parsley
let tpl = "Hello @{name}, you have @{count} items."
tpl.render({name: "Alice", count: 3})
// "Hello Alice, you have 3 items."
```

Expressions inside `@{...}` are full Parsley — arithmetic, method calls, and conditionals all work:

```parsley
"Total: @{price * qty}".render({price: 10, qty: 5})
// "Total: 50"
```

> ⚠️ Don't use raw strings (`'...'`) for templates you intend to `.render()` later — the `@{expr}` placeholders get interpolated immediately when the raw string is created. Use double-quoted strings instead to keep the placeholders intact.

### Display & Serialization

| Method | Description |
|--------|-------------|
| `.toJSON()` | JSON-encode the string (adds quotes, escapes special chars) |
| `.toBox(opts?)` | Render in a box with box-drawing characters |
| `.repr()` | Debug representation with escapes visible |

**toBox options:** `style` (`"single"`, `"double"`, `"ascii"`, `"rounded"`), `title`, `maxWidth`, `align` (`"left"`, `"right"`, `"center"`).

## Key Differences from Other Languages

- **Three string types:** Most languages have one or two. Parsley's raw strings (`'...'`) avoid the backslash-escaping pain of regex and paths
- **`{expr}` not `${expr}`:** Template string interpolation has no dollar sign
- **`'...'` is raw, not a char:** Single quotes create raw strings, not character literals
- **Natural sort comparison:** `"file2" < "file10"` is `true` — string comparisons use natural ordering, not lexicographic
- **`+` concatenates, `++` does not:** Use `+` for string joining. `++` wraps into an array
- **No `.toString()` method:** Parsley coerces automatically in template strings and `+` concatenation. Use `.type()` to check a value's type

## See Also

- [Numbers](numbers.md) — numeric types and formatting
- [Booleans & Null](booleans.md) — truthiness and null coalescing
- [Arrays](array.md) — `.split()` returns arrays; `.join()` is on arrays
- [Regex](regex.md) — pattern matching and `.replace()` with regex
- [Operators](../fundamentals/operators.md) — complete operator reference
- [Tags](../fundamentals/tags.md) — string interpolation inside HTML tags
- [Data Formats](../features/data-formats.md) — parsing and generating CSV, JSON, Markdown