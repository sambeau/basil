# Formatter Design for Parsley Builtins

**Status:** Approved  
**Version:** 1.0  
**Date:** 2025-01-20

## Overview

This document defines the unified formatting API for all Parsley builtin types. The design prioritizes brevity, consistency, and composability.

## Core Method: `.fmt()`

The primary formatting method is `.fmt()`, replacing the more verbose `.format()`.

### Signature Overloads

| Call | Type | Meaning |
|------|------|---------|
| `.fmt()` | — | Medium style, default locale |
| `.fmt(n)` | Integer | n decimal places (precision) |
| `.fmt("style")` | String | Named style |
| `.fmt("style", "locale")` | String, String | Style with locale |
| `.fmt({...})` | Dictionary | Full options |

### Examples

```parsley
$1234.56.fmt()                           // "$1,234.56"
#12.345m.fmt(2)                          // "12.35m"
@2024-12-25.fmt("short")                 // "12/25/24"
@2024-12-25.fmt("long", "de-DE")         // "25. Dezember 2024"
price.fmt({style: "short", locale: "de-DE", precision: 2})
```

## Style Methods

Shorthand methods for common styles. Each accepts an optional argument: locale string or options dictionary.

| Method | Equivalent |
|--------|------------|
| `.short()` | `.fmt("short")` |
| `.short("de-DE")` | `.fmt("short", "de-DE")` |
| `.short({locale: "de-DE"})` | `.fmt({style: "short", locale: "de-DE"})` |
| `.medium()` | `.fmt("medium")` or `.fmt()` |
| `.long()` | `.fmt("long")` |
| `.full()` | `.fmt("full")` |

### Reusable Configurations

```parsley
let german = {locale: "de-DE"}
let precise = {locale: "de-DE", precision: 2}

price.short(german)
date.long(german)
weight.fmt(precise)
```

## Standard Styles

All value types support these styles with consistent semantics:

| Style | Meaning |
|-------|---------|
| `short` | Most compact representation, may lose precision |
| `medium` | Balanced display (default) |
| `long` | Full precision, verbose |
| `full` | Maximum context (where applicable) |

### Style Output by Type

| Type | short | medium (default) | long | full |
|------|-------|------------------|------|------|
| Number | `"1.2M"` | `"1,235"` | `"1,234.57"` | — |
| Money | `"$1K"` | `"$1,235"` | `"$1,234.56"` | `"1,234.56 US dollars"` |
| DateTime | `"12/25/24"` | `"Dec 25, 2024"` | `"December 25, 2024"` | `"Wednesday, December 25, 2024"` |
| Duration | `"2h"` | `"2 hours"` | `"2 hours 30 min"` | — |
| Unit | `"5m"` | `"5.00m"` | `"5 metres"` | `"5 metres (16.4 ft)"` |

### Style Availability

| Type | short | medium | long | full | .fmt(n) |
|------|-------|--------|------|------|---------|
| Number | ✓ | ✓ | ✓ | — | ✓ |
| Money | ✓ | ✓ | ✓ | ✓ | ✓ |
| DateTime | ✓ | ✓ | ✓ | ✓ | — |
| Duration | ✓ | ✓ | ✓ | — | — |
| Unit | ✓ | ✓ | ✓ | ✓ | ✓ |

## Arrays

Arrays use conjunction-based formatting (different semantic model):

```parsley
["Alice", "Bob", "Charlie"].fmt("and")        // "Alice, Bob, and Charlie"
["Alice", "Bob", "Charlie"].fmt("or")         // "Alice, Bob, or Charlie"
["Alice", "Bob"].fmt("and", "de-DE")          // "Alice und Bob"
```

## Serialization Methods

### Universal (All Types)

| Method | Purpose | Returns |
|--------|---------|---------|
| `repr()` | PLN literal (parseable, round-trips) | String |
| `toJSON()` | JSON serialization | String |
| `inspect()` | Debug dictionary with `__type` | Dictionary |
| `toBox()` | CLI box-drawing diagram | String |

### Collections Only

| Method | Purpose | Available On |
|--------|---------|--------------|
| `toMarkdown()` | Markdown table | Array, Dictionary |
| `toHTML()` | HTML rendering | Array, Dictionary |
| `toCSV()` | CSV string | Array |

### Complex Value Types

| Method | Purpose | Available On |
|--------|---------|--------------|
| `toDict()` | Clean dictionary (no `__type`) | Money, DateTime, Duration, Unit, URL, Path |

## Format Options Dictionary

The options dictionary supports these keys:

| Key | Type | Applies To | Description |
|-----|------|------------|-------------|
| `style` | String | All value types | `"short"`, `"medium"`, `"long"`, `"full"` |
| `locale` | String | All value types | BCP 47 locale code (e.g., `"de-DE"`) |
| `precision` | Integer | Number, Money, Unit | Decimal places |
| `compound` | Boolean | Unit | Use compound format (e.g., feet-inches) |

## Locale Support

All formatting methods support locale-aware output:

- Numbers: thousand separators, decimal marks
- Money: symbol placement, separators
- DateTime: date order, month names, weekday names
- Duration: relative time phrases
- Unit: decimal marks, unit name spelling (`"metres"` vs `"meters"`)
- Array: conjunction words (`"and"` vs `"und"`)

Default locale: `"en-US"`

## Type-Specific Notes

### Unit Long Format

The `long` style spells out unit names with correct:
- Pluralization: `"1 metre"` vs `"2 metres"`
- Regional spelling: `"metre"` (en-GB) vs `"meter"` (en-US)

### Unit Full Format

Includes cross-system conversion:
```parsley
#5m.full()      // "5 metres (16.4 ft)"
#63in.full()    // "63 inches (1.6 m)"
```

### Money Full Format

Spells out currency name:
```parsley
$1234.56.full()           // "1,234.56 US dollars"
€1234.56.full("de-DE")    // "1.234,56 Euro"
```

### Duration Styles

| Style | Positive | Negative |
|-------|----------|----------|
| short | `"2h"` | `"-2h"` |
| medium | `"2 hours"` | `"2 hours ago"` |
| long | `"2 hours 30 minutes"` | `"2 hours 30 minutes ago"` |

## Method Summary

### Value Types (Number, Money, DateTime, Duration, Unit)

```
.fmt()                    // medium style, default locale
.fmt(n)                   // precision (Number, Money, Unit only)
.fmt("style")             // named style
.fmt("style", "locale")   // style + locale
.fmt({...})               // full options

.short(opts?)             // compact
.medium(opts?)            // balanced (default)
.long(opts?)              // verbose
.full(opts?)              // maximum context (Money, DateTime, Unit only)

.repr()                   // PLN literal
.toJSON()                 // JSON string
.inspect()                // debug dictionary
.toBox()                  // CLI box diagram
.toDict()                 // clean dictionary
```

### Collections (Array, Dictionary)

```
.repr()                   // PLN literal
.toJSON()                 // JSON string
.inspect()                // debug dictionary
.toBox()                  // CLI box diagram
.toMarkdown()             // Markdown table
.toHTML()                 // HTML rendering
.toCSV()                  // CSV string (Array only)
```

### Arrays (Conjunction Format)

```
.fmt("and")               // "A, B, and C"
.fmt("or")                // "A, B, or C"
.fmt("and", "locale")     // localized conjunction
```

### Primitives (Boolean, Null, String)

```
.repr()                   // PLN literal
.toJSON()                 // JSON string
.toBox()                  // CLI box diagram
```

### Paths and URLs

```
.repr()                   // PLN literal
.toJSON()                 // JSON string (path/URL as string)
.inspect()                // debug dictionary
.toBox()                  // CLI box diagram
.toDict()                 // component dictionary
```
