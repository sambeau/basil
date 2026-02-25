# Parsley Formatter Audit

**Date:** 2025-01-20  
**Updated:** 2025-01-20  
**Purpose:** Pre-1.0 audit of formatting capabilities across all builtin types  
**Status:** Analysis complete, recommendations refined

## Executive Summary

Parsley has solid formatting foundations but lacks consistency across types. The main gaps are:

1. **Inconsistent API patterns** — DateTime has style parameters, Money has only locale, Units have precision/compound formats
2. **Missing locale support** — Units have no locale awareness at all
3. **Incomplete output format coverage** — Many types lack HTML/Markdown/JSON serialization
4. **No unified custom format system** — Each type has its own ad-hoc approach
5. **Verbose syntax** — Current `.format()` is too long for frequent use in templates

This document proposes a unified formatter philosophy aligned with Parsley's design principles (simplicity, minimalism, completeness, composability).

---

## Current State

### Formatter Inventory by Type

#### Numbers (Integer/Float)
| Method | Signature | Description |
|--------|-----------|-------------|
| `format` | `(locale?)` | Locale-aware with thousand separators |
| `currency` | `(code, locale?)` | Currency symbol + formatting |
| `percent` | `(locale?)` | Percentage formatting |
| `humanize` | `(locale?)` | Compact notation (1K, 1M, 1B) |
| `toBox` | `()` | CLI box diagram |
| `repr` | `()` | PLN representation |
| `toJSON` | `()` | JSON number string |

#### Money
| Method | Signature | Description |
|--------|-----------|-------------|
| `format` | `(locale?)` | Locale-aware currency formatting |
| `abs` | `()` | Absolute value |
| `split` | `(n)` | Fair division into n parts |
| `toBox` | `()` | CLI box diagram |
| `repr` | `()` | PLN representation |
| `toJSON` | `()` | JSON with amount/currency |
| `toDict` | `()` | Clean dictionary |
| `inspect` | `()` | Debug dictionary with __type |

#### DateTime
| Method | Signature | Description |
|--------|-----------|-------------|
| `format` | `(style?, locale?)` | Style: short/medium/long/full |
| `toDict` | `()` | Component dictionary |
| `inspect` | `()` | Debug dictionary |
| `toJSON` | `()` | ISO 8601 string |

**Properties:** `.date`, `.time`, `.iso`, `.unix`

#### Duration
| Method | Signature | Description |
|--------|-----------|-------------|
| `format` | `(locale?)` | Relative time formatting |
| `toDict` | `()` | Component dictionary |
| `inspect` | `()` | Debug dictionary |

#### Units
| Method | Signature | Description |
|--------|-----------|-------------|
| `format` | `(precision?)` or `(compoundFormat)` | Decimal precision or compound |
| `repr` | `()` | PLN literal |
| `toDict` | `()` | Clean dictionary |
| `inspect` | `()` | Debug dictionary |
| `toFraction` | `()` | Fraction string for US units |

#### Strings
| Method | Signature | Description |
|--------|-----------|-------------|
| `toBox` | `()` | CLI box diagram |
| `repr` | `()` | PLN representation |
| `toJSON` | `()` | JSON-encoded string |

#### Arrays
| Method | Signature | Description |
|--------|-----------|-------------|
| `format` | `(conjunction, locale?)` | List formatting |
| `toBox` | `(options?)` | CLI box with direction/style |
| `repr` | `()` | PLN representation |
| `toJSON` | `()` | JSON array |
| `toCSV` | `(hasHeader?)` | CSV string |
| `toHTML` | `()` | HTML rendering |
| `toMarkdown` | `()` | Markdown table |

#### Dictionaries
| Method | Signature | Description |
|--------|-----------|-------------|
| `toBox` | `(options?)` | CLI box diagram |
| `repr` | `()` | PLN representation |
| `toJSON` | `()` | JSON object |
| `toHTML` | `()` | HTML definition list |
| `toMarkdown` | `()` | Markdown table |

#### Booleans & Null
| Method | Signature | Description |
|--------|-----------|-------------|
| `toBox` | `()` | CLI box diagram |

#### URLs
| Method | Signature | Description |
|--------|-----------|-------------|
| `href` | `()` | Full URL string |
| `origin` | `()` | Scheme + host + port |
| `pathname` | `()` | Path portion |
| `search` | `()` | Query string |
| `toDict` | `()` | Component dictionary |
| `inspect` | `()` | Debug dictionary |

#### Paths
| Method | Signature | Description |
|--------|-----------|-------------|
| `public` | `()` | Web-serving URL |
| `toURL` | `(prefix)` | URL with prefix |
| `toDict` | `()` | Component dictionary |
| `inspect` | `()` | Debug dictionary |

---

## Gap Analysis

### 1. Inconsistent Format Method Signatures

| Type | format() signature |
|------|-------------------|
| DateTime | `(style?, locale?)` ← **Good model** |
| Money | `(locale?)` — no style options |
| Duration | `(locale?)` — no style options |
| Unit | `(precision?)` or `(compound)` — **no locale!** |
| String | ❌ None |
| URL/Path | ❌ None |

### 2. Locale Support Coverage

| Type | Locale Support |
|------|---------------|
| Number | ✅ format, currency, humanize, percent |
| Money | ✅ format |
| DateTime | ✅ format |
| Duration | ✅ format |
| Unit | ❌ **Missing** |
| Array | ✅ format |

### 3. Output Format Coverage Matrix

| Type | repr() | toJSON() | toMarkdown() | toHTML() | toBox() |
|------|--------|----------|--------------|----------|---------|
| Integer | ✅ | ✅ | ❌ | ❌ | ✅ |
| Float | ✅ | ✅ | ❌ | ❌ | ✅ |
| Money | ✅ | ✅ | ❌ | ❌ | ✅ |
| DateTime | ❌ | ✅ | ❌ | ❌ | ❌ |
| Duration | ❌ | ❌ | ❌ | ❌ | ❌ |
| Unit | ✅ | ❌ | ❌ | ❌ | ❌ |
| String | ✅ | ✅ | ❌ | ❌ | ✅ |
| Array | ✅ | ✅ | ✅ | ✅ | ✅ |
| Dictionary | ✅ | ✅ | ✅ | ✅ | ✅ |
| Boolean | ❌ | ❌ | ❌ | ❌ | ✅ |
| Null | ❌ | ❌ | ❌ | ❌ | ✅ |
| URL | ❌ | ❌ | ❌ | ❌ | ❌ |
| Path | ❌ | ❌ | ❌ | ❌ | ❌ |

### 4. Verbosity in Templates

Current syntax is too verbose for common operations:

```parsley
// Current - verbose
`Total: {price.format("short")} due {date.format("long", "de-DE")}`

// Desired - concise
`Total: {price.short()} due {date.long("de-DE")}`
```

---

## Modern Language Research

### Key Trends
1. **Debug vs Display** — Two distinct representations (developer vs user)
2. **Named styles over format strings** — `"short"`, `"long"` are easier than `%04.2f`
3. **Locale-awareness is table stakes** — Modern apps are global
4. **Consistent patterns across types** — Same method should work similarly across types
5. **Brevity in common operations** — Frequently used methods should be short

### Why Avoid Format Strings?

Format strings like `"YYYY-MM-DD"` or `"%04.2f"` are:
- Hard to remember
- Error-prone
- Not locale-aware by default
- Against Parsley's "simplicity" principle

Instead, 90% of needs are covered by:
- Named styles: `"short"`, `"long"`
- Precision parameter: `.fmt(2)`
- Component properties for assembly: `` `{dt.year}-{dt.month}-{dt.day}` ``

---

## Recommendations

### Philosophy

Align formatters with Parsley's principles:

1. **Simplicity** — Prefer named styles over cryptic format strings
2. **Minimalism** — Small API surface, methods should be guessable
3. **Completeness** — Every type should have the formatters it needs
4. **Composability** — Formatters should work well in pipelines and templates
5. **Brevity** — Common operations should be concise

### Proposed API: `.fmt()` + Style Sugar

#### Rename: `.format()` → `.fmt()`

The method will be used constantly in templates. Save 3 characters per use:

```parsley
`{price.fmt()}`      // vs {price.format()}
```

Consistent with existing abbreviations like `repr`.

#### The `.fmt()` Signature

| Call | Meaning | Example |
|------|---------|---------|
| `.fmt()` | Medium style, default locale | `$1234.56.fmt()` → `"$1,234.56"` |
| `.fmt(n)` | n decimal places | `#12.345m.fmt(2)` → `"12.35m"` |
| `.fmt("style")` | Named style | `date.fmt("short")` → `"12/25/24"` |
| `.fmt("style", "locale")` | Style + locale | `date.fmt("long", "de-DE")` → `"25. Dezember 2024"` |
| `.fmt({...})` | Full control | `price.fmt({style: "short", locale: "de-DE"})` |

Types are unambiguous:
- Integer → precision
- String → style (or locale if 2nd arg)  
- Dictionary → options

#### Style Methods as Sugar

Add shorthand methods for common styles:

```parsley
// These pairs are equivalent:
price.short()     ←→  price.fmt("short")
price.long()      ←→  price.fmt("long")
date.full()       ←→  date.fmt("full")

// With locale string:
price.short("de-DE")  ←→  price.fmt("short", "de-DE")

// With options dictionary:
let german = {locale: "de-DE"}
price.short(german)   ←→  price.fmt({style: "short", locale: "de-DE"})

// Reusable format configs:
let priceFmt = {locale: "de-DE", precision: 2}
price.short(priceFmt)
date.long(priceFmt)
```

Style methods accept either:
- No argument → default locale
- String → locale code
- Dictionary → full options (locale, precision, etc.)

**Character savings:**

| Pattern | Before | After | Saved |
|---------|--------|-------|-------|
| Short format | `.fmt("short")` | `.short()` | 6 |
| Long format | `.fmt("long")` | `.long()` | 5 |
| Full format | `.fmt("full")` | `.full()` | 5 |
| Precision | `.fmt({precision: 2})` | `.fmt(2)` | 13 |
| Short + locale | `.fmt("short", "de")` | `.short("de")` | 6 |

**Template comparison:**

```parsley
// Before (67 chars in interpolations)
`{product.name}: {price.format("short")} - Ships {date.format("long")} ({weight.format({precision: 1})})`

// After (46 chars in interpolations)  
`{product.name}: {price.short()} - Ships {date.long()} ({weight.fmt(1)})`
```

### Standardized Styles Across Types

| Style | Meaning | Number | Money | DateTime | Duration | Unit |
|-------|---------|--------|-------|----------|----------|------|
| `short` | Most compact | "1.2M" | "$1K" | "12/25/24" | "2h" | "5m" |
| `medium` | Balanced (default) | "1,235" | "$1,235" | "Dec 25, 2024" | "2 hours" | "5.00m" |
| `long` | Full precision/verbose | "1,234.57" | "$1,234.56" | "December 25, 2024" | "2 hours 30 min" | "5 metres" |
| `full` | Maximum context | — | "1,234.56 US dollars" | "Wednesday, December 25, 2024" | — | "5 metres (16.4 ft)" |

**Rules:**
1. Every type has at least `short` and `long`
2. `medium` is the default when `.fmt()` is called with no args
3. `full` is optional — only for types where extra context adds value
4. All style methods accept `(locale?)` string or `({...})` options dictionary

### Style Methods by Type

| Type | .short(opts?) | .medium(opts?) | .long(opts?) | .full(opts?) | .fmt(n) |
|------|---------------|----------------|--------------|--------------|---------|
| Number | ✓ | ✓ | ✓ | — | ✓ |
| Money | ✓ | ✓ | ✓ | ✓ | ✓ |
| DateTime | ✓ | ✓ | ✓ | ✓ | — |
| Duration | ✓ | ✓ | ✓ | — | — |
| Unit | ✓ | ✓ | ✓ | ✓ | ✓ |

Where `opts?` can be:
- Omitted → default locale
- `"locale"` → locale string (e.g., `"de-DE"`)
- `{...}` → options dictionary (e.g., `{locale: "de-DE", precision: 2}`)

### Examples by Type

#### Numbers

```parsley
1234567.fmt()           // "1,234,567" (medium)
1234567.short()         // "1.2M"
1234567.long()          // "1,234,567.00"
3.14159.fmt(2)          // "3.14"
1234567.short("de-DE")  // "1,2 Mio."

// Reusable config
let numFmt = {locale: "de-DE", precision: 1}
1234567.short(numFmt)   // "1,2 Mio."
```

#### Money

```parsley
$1234.56.fmt()          // "$1,234.56" (medium)
$1234.56.short()        // "$1K"
$1234.56.long()         // "$1,234.56"
$1234.56.full()         // "1,234.56 US dollars"
$1234.567.fmt(2)        // "$1,234.57"
€1234.56.short("de-DE") // "1,2 Tsd. €"

// Reusable config
let euroFmt = {locale: "de-DE"}
€1234.56.short(euroFmt) // "1,2 Tsd. €"
€1234.56.long(euroFmt)  // "1.234,56 €"
```

#### DateTime

```parsley
@2024-12-25.fmt()              // "Dec 25, 2024" (medium)
@2024-12-25.short()            // "12/25/24"
@2024-12-25.long()             // "December 25, 2024"
@2024-12-25.full()             // "Wednesday, December 25, 2024"
@2024-12-25.long("de-DE")      // "25. Dezember 2024"
```

#### Duration

```parsley
@2h30m.fmt()            // "2 hours" (medium)
@2h30m.short()          // "2h"
@2h30m.long()           // "2 hours 30 minutes"
@-1d.fmt()              // "yesterday"
@7d.short("de-DE")      // "1W"
```

#### Units

```parsley
#12.345m.fmt()          // "12.35m" (medium)
#12.345m.short()        // "12m"
#12.345m.long()         // "12.345 metres"
#12.345m.full()         // "12.345 metres (40.5 ft)"
#12.345m.fmt(1)         // "12.3m"
#63in.fmt({compound: true})  // "5' 3\""
#5m.long("en-US")       // "5 meters"
#5m.long("en-GB")       // "5 metres"
```

#### Arrays

Arrays keep their conjunction-based format (different semantic):

```parsley
["Alice", "Bob", "Charlie"].fmt("and")     // "Alice, Bob, and Charlie"
["Alice", "Bob", "Charlie"].fmt("or")      // "Alice, Bob, or Charlie"
["Alice", "Bob"].fmt("and", "de-DE")       // "Alice und Bob"
```

### Tier System

#### Tier 1: Core Methods (All Types)

Every type should implement:

| Method | Purpose | Returns |
|--------|---------|---------|
| `repr()` | PLN literal (round-trips) | String |
| `toJSON()` | JSON serialization | String |
| `inspect()` | Debug dictionary | Dictionary |

#### Tier 2: Display Methods (Value Types)

Numeric, temporal, and measurement types should implement:

| Method | Purpose | Signature |
|--------|---------|-----------|
| `fmt()` | Human-readable display | `(style?, locale?)` or `(precision)` or `({...})` |
| `short()` | Compact display | `(locale?)` |
| `medium()` | Balanced display | `(locale?)` |
| `long()` | Verbose display | `(locale?)` |
| `full()` | Maximum context | `(locale?)` — where applicable |

#### Tier 3: Serialization (Collections)

Arrays and dictionaries should implement:

| Method | Purpose |
|--------|---------|
| `toMarkdown()` | Markdown table |
| `toHTML()` | HTML rendering |
| `toCSV()` | CSV (arrays only) |

#### Tier 4: CLI Output (All Types)

| Method | Purpose |
|--------|---------|
| `toBox()` | Box-drawing diagram for terminal |

### Specific Type Additions Needed

#### Numbers
- ✅ Already has `format()`, rename to `fmt()`
- Add `short()`, `medium()`, `long()` sugar

#### Money
- Rename `format()` to `fmt()`
- Add style support to `fmt()`
- Add `short()`, `medium()`, `long()`, `full()` sugar

#### DateTime
- ✅ Already has good style support
- Rename `format()` to `fmt()`
- Add `short()`, `medium()`, `long()`, `full()` sugar
- Add `repr()` for PLN: `@2024-12-25`
- Add `toBox()`

#### Duration
- Rename `format()` to `fmt()`
- Add styles: `short`, `medium`, `long`
- Add `short()`, `medium()`, `long()` sugar
- Add `repr()` for PLN: `@2h30m`
- Add `toJSON()`: `{"months": 0, "seconds": 9000}`
- Add `toBox()`

#### Units
- **Add locale support** to `fmt()`
- Add styles: `short`, `medium`, `long`, `full`
- Add `short()`, `medium()`, `long()`, `full()` sugar
- Add `toJSON()`: `{"value": 12.345, "unit": "m", "family": "length"}`
- Add `toBox()`

#### Booleans/Null
- Add `repr()`: `true`, `false`, `null`
- Add `toJSON()`: `true`, `false`, `null`

#### URLs
- Add `repr()`: `@https://example.com/path`
- Add `toJSON()`: URL string
- Add `toBox()`

#### Paths
- Add `repr()`: `@./path/to/file`
- Add `toJSON()`: path string
- Add `toBox()`

---

## Implementation Priority

### P0 — Must Have for 1.0

1. **Rename `.format()` to `.fmt()`** — Brevity matters
2. **Add integer overload to `.fmt(n)`** — Precision is common
3. **Add style sugar methods** — `.short()`, `.long()`, etc.
4. **Add `repr()` to all types** — Critical for PLN round-tripping
5. **Add `toJSON()` to all types** — Expected for data interchange
6. **Add locale support to Units** — Major gap

### P1 — Should Have

7. **Add `toBox()` to remaining types** — CLI completeness
8. **Standardize styles across types** — Consistency matters

### P2 — Nice to Have

9. **Add `toMarkdown()`, `toHTML()` to scalar types** — Documentation generation
10. **Format dictionary options** — For advanced use cases

---

## Migration Path

### Phase 1: Add New API (Non-Breaking)

1. Add `.fmt()` as alias for `.format()`
2. Add style methods: `.short()`, `.medium()`, `.long()`, `.full()`
3. Add precision overload: `.fmt(n)`

### Phase 2: Documentation Update

1. Update all examples to use `.fmt()` and sugar methods
2. Document `.format()` as legacy alias

### Phase 3: Deprecation (Future Major Version)

1. Mark `.format()` as deprecated
2. Eventually remove in 2.0

---

## Appendix: Complete Method Reference (Proposed)

### Universal Methods

| Method | Available On | Purpose |
|--------|-------------|---------|
| `repr()` | All types | PLN literal (parseable) |
| `toJSON()` | All types | JSON serialization |
| `inspect()` | All types | Debug dictionary with __type |
| `toBox()` | All types | CLI box diagram |

### Value Type Methods

| Method | Available On | Purpose |
|--------|-------------|---------|
| `fmt()` | Value types | Human display (style?, locale?) or (precision) or ({...}) |
| `short(opts?)` | Value types | Compact display — opts: locale string or {locale?, precision?, ...} |
| `medium(opts?)` | Value types | Balanced display (same as `fmt()`) |
| `long(opts?)` | Value types | Verbose display |
| `full(opts?)` | Money, DateTime, Unit | Maximum context |
| `toDict()` | Complex value types | Clean dictionary |

### Collection Methods

| Method | Available On | Purpose |
|--------|-------------|---------|
| `toMarkdown()` | Array, Dictionary | Markdown table |
| `toHTML()` | Array, Dictionary | HTML rendering |
| `toCSV()` | Array | CSV string |

### Format Styles Summary

| Type | short | medium (default) | long | full |
|------|-------|------------------|------|------|
| Number | "1.2M" | "1,235" | "1,234.57" | — |
| Money | "$1K" | "$1,235" | "$1,234.56" | "1,234.56 US dollars" |
| DateTime | "12/25/24" | "Dec 25, 2024" | "December 25, 2024" | "Wednesday, December 25, 2024" |
| Duration | "2h" | "2 hours" | "2 hours 30 min" | — |
| Unit | "5m" | "5.00m" | "5 metres" | "5 metres (16.4 ft)" |

---

## `print` / `println` Audit

### What They Do

These builtins return a special `PrintValue` object that gets expanded into the result stream:

```go
"print": {
    Fn: func(args ...Object) Object {
        if len(args) == 0 {
            return newArityError("print", 0, 1)
        }
        return &PrintValue{Values: args}
    },
},
"println": {
    Fn: func(args ...Object) Object {
        if len(args) == 0 {
            return &PrintValue{Values: []Object{&String{Value: "\n"}}}
        }
        values := make([]Object, len(args)+1)
        copy(values, args)
        values[len(args)] = &String{Value: "\n"}
        return &PrintValue{Values: values}
    },
},
```

The `PrintValue` is handled in `evalProgram`, `evalBlockStatement`, and `evalInterpolationBlock` by merging values into the expression result stream as user-facing strings.

### Key Insight: Template Output, Not Console Logging

Unlike `log()` which:
- Writes **immediately** to stdout via `fmt.Fprintln(os.Stdout, ...)`
- Returns `NULL`
- Uses debug formatting

`print()`/`println()`:
- Return a `PrintValue` merged into the **expression result stream**
- Values converted to **user-facing strings**
- Don't produce console output — they produce **template output**

### The Problem

1. **Name collision with mental model** — Every programmer expects `print()` to mean console output
2. **Redundant with expression model** — Parsley's expression-based design makes them unnecessary:
   ```parsley
   // Expression style (idiomatic)
   for (item in items) {
       if (item.featured) {
           <li class=featured>{item.name}</li>
       } else {
           <li>{item.name}</li>
       }
   }
   
   // print style (unnecessary escape hatch)
   for (item in items) {
       if (item.featured) {
           print(<li class=featured>{item.name}</li>)
       } else {
           print(<li>{item.name}</li>)
       }
   }
   ```
3. **Documentation confusion** — We tell users "`log()` not `print()`" but `print()` exists for a different purpose, causing confusion

### Assessment

| Criterion | Verdict |
|-----------|---------|
| **Are they used?** | Yes, in some examples and tests |
| **Are they useful?** | Marginally — expression model makes them redundant |
| **Are they confusing?** | YES — easily confused with console logging |
| **Is there overlap?** | Yes, with just returning values from expressions |

### Decision: Remove for 1.0

**Rationale:**
- The expression-based model is cleaner and more idiomatic
- Their existence suggests an imperative mental model Parsley doesn't need
- The naming causes persistent confusion
- Users should learn the Parsley way rather than having an escape hatch

### Implementation Plan

1. **Remove from builtins** — Delete `print`, `println`, `printf` from `getBuiltins()`
2. **Remove `PrintValue` type** — Delete the type and all handling code in eval functions
3. **Update examples** — Convert any examples using `print()` to expression style
4. **Update tests** — Remove or rewrite tests that use these functions
5. **Update introspect.go** — Remove from builtin metadata

### AI Guidance Challenge

**The Problem:** These functions were originally added because AI coding assistants (Claude, GPT, etc.) persistently generate code using `print()` — it's deeply ingrained in their training data from Python, JavaScript, and virtually every other language. When `print()` didn't exist, AI-generated Parsley code was frequently broken.

**Mitigation Strategies:**

1. **Clear error messages with guidance**
   ```
   Error: Unknown function 'print'. 
   
   Parsley uses expression-based output. Instead of:
       print(value)
   
   Simply return the value:
       value
   
   For debugging, use log() to write to console.
   ```

2. **Update all AI instruction files**
   - `.github/instructions/parsley.instructions.md` — Add prominent warning
   - `.github/copilot-instructions.md` — Reinforce in project rules
   - `docs/parsley/CHEATSHEET.md` — Lead with this gotcha

3. **Cheatsheet prominence** — Make "No print() function" the #1 gotcha, with clear explanation:
   ```markdown
   ### 1. No `print()` Function — Expressions ARE Output
   
   // ❌ WRONG (doesn't exist)
   print("hello")
   println("hello")
   
   // ✅ CORRECT — just write the value
   "hello"           // String is the output
   <div>hello</div>  // Tag is the output
   
   // ✅ For debugging/console, use log()
   log("debug:", someVar)
   ```

4. **"Did you mean" suggestions** — Enhance error handling to detect `print`/`println` calls and provide targeted help

5. **Example corpus** — Ensure ALL example files in `/examples/parsley/` use expression style, never `print()`. AIs learn from examples.

6. **Test the AI experience** — After removal, test with fresh AI conversations to verify the guidance is effective. Iterate on error messages and documentation based on what patterns AIs still attempt.

### Files to Update

- `pkg/parsley/evaluator/evaluator.go` — Remove builtins and `PrintValue` type
- `pkg/parsley/evaluator/introspect.go` — Remove from metadata
- `pkg/parsley/tests/replace_function_test.pars` — Uses `print()` extensively
- `examples/parsley/reference/21-builtins.pars` — Uses `print()`/`println()`
- `examples/parsley/temp/test_table_*.pars` — Use `print()`
- `.github/instructions/parsley.instructions.md` — Update guidance
- `docs/parsley/CHEATSHEET.md` — Update gotchas section
- `contrib/highlightjs/README.md` — Update examples

---

## Related Work

- FEAT-111: Declarative Method Registry (completed)
- FEAT-118: Measurement Units (completed)
- See `work/BACKLOG.md` for deferred formatting items