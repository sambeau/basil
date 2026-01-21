---
id: FEAT-099
title: "Flexible Date/Time/Duration Parsing"
status: implemented
priority: high
created: 2026-01-21
completed: 2026-01-21
author: "@human"
---

# FEAT-099: Flexible Date/Time/Duration Parsing

## Summary
Add flexible parsing for dates, times, and datetimes using natural human-readable formats like "22 April 2005", "3:45 PM", and "April 22, 2005 at 3pm". Provides three separate functions (`date()`, `time()`, `datetime()`) to reduce ambiguity, with options to handle regional format differences.

## User Story
As a developer working with user-provided dates, I want to parse dates in various human-readable formats so that I can accept natural input without requiring strict ISO formats.

## Acceptance Criteria
- [ ] `date("22 April 2005")` parses to a date dict
- [ ] `time("3:45 PM")` parses to a time dict
- [ ] `datetime("April 22, 2005 3:45 PM")` parses to a datetime dict
- [ ] All functions support 100+ common formats via araddon/dateparse
- [ ] `{locale: "en-GB"}` option for DD/MM/YYYY and localized month names
- [ ] `{strict: true}` option to error on ambiguous dates
- [ ] `{timezone: "America/New_York"}` option for UTC offset interpretation
- [ ] Default timezone is UTC (server location does not affect results)
- [ ] Duration parsing enhanced for human-readable formats
- [ ] Existing `@2024-12-25` literal syntax unchanged

## Design Decisions

### Three Separate Functions
**Rationale**: Using `date()`, `time()`, `datetime()` separately provides a hint about expected input type, reducing ambiguity. When the function knows you expect a date, it won't try to parse "April" as a time.

### UTC by Default, Location as Hint
**Rationale**: Server location should not affect results—in cloud environments you often don't know where code runs. All parsing defaults to UTC. Users can provide a `timezone` hint based on user preferences or business logic.

### Use araddon/dateparse Library  
**Rationale**: Mature Go library handling 100+ date formats via state machine parsing. Much faster than regex-based shotgun approaches. MIT licensed, 2.1k stars.

### No Backward Compatibility for `time()`
**Rationale**: Pre-alpha status—better to get the API right now. Current `time()` becomes `datetime()`.

### Locale vs Timezone
**Rationale**: These are separate concerns:
- **Locale** (`en-US`, `en-GB`, `fr-FR`) — How the date was *written* (format, month names)
- **Timezone** (`America/New_York`) — When the date/time *occurred* (UTC offset)

A British person in New York writes "22/04/2005" (UK locale) for an event at 3pm Eastern (NYC timezone). Using the same locale codes as date formatting ensures consistency.

### Strict Mode Available but Not Default
**Rationale**: Convenience by default (guess when possible), safety when requested (error on ambiguity).

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### API Design

```parsley
// Date-only parsing (returns date dict with __type: "date")
date("22 April 2005")              // → {__type: "date", year: 2005, month: 4, day: 22}
date("2005-04-22")                 // ISO format
date("04/22/2005")                 // US format (default)
date("22/04/2005", {dayFirst: true}) // UK/EU format

// Time-only parsing (returns time dict with __type: "time")
time("3:45 PM")                    // → {__type: "time", hour: 15, minute: 45, second: 0}
time("15:45:30")                   // 24-hour format
time("3:45:30.123 PM")             // With milliseconds

// Full datetime parsing (returns datetime dict with __type: "datetime")
datetime("April 22, 2005 3:45 PM") // → full datetime dict
datetime("2005-04-22T15:45:00Z")   // ISO 8601
datetime(1682157900)               // Unix timestamp
datetime({year: 2005, month: 4, day: 22}) // From dict

// Options (apply to all three functions)
{
  locale: string,      // Locale code for format interpretation (default: "en-US")
                       // Determines: date order (DD/MM vs MM/DD), month names
                       // Examples: "en-US", "en-GB", "fr-FR", "de-DE"
  strict: bool,        // Error on ambiguous input (default: false)
  timezone: string,    // IANA timezone for UTC offset (default: "UTC")
                       // Only affects time interpretation, not format
}

// Examples
date("01/02/2005")                      // Jan 2nd (en-US default)
date("01/02/2005", {locale: "en-GB"})   // Feb 1st (day first)
date("22 avril 2005", {locale: "fr-FR"}) // French month names
datetime("3pm EST", {timezone: "America/New_York"}) // Timezone for offset
```

### Supported Formats (via araddon/dateparse)

```
// Named months
"May 8, 2009 5:57:51 PM"
"oct 7, 1970"
"October 7th, 1970"
"22 April 2005"
"1 July 2013"
"12 Feb 2006, 19:17"

// Numeric with separators  
"3/31/2014"           // MM/DD/YYYY (default)
"31/03/2014"          // DD/MM/YYYY (with dayFirst)
"2014/03/31"          // YYYY/MM/DD
"2014-03-31"          // ISO
"2014.03.31"          // Dots

// ISO 8601
"2009-08-12T22:15:09Z"
"2009-08-12T22:15:09-07:00"
"2009-08-12T22:15:09.988"

// Unix timestamps
1332151919            // Seconds
1384216367189         // Milliseconds

// Time formats
"3:45 PM"
"15:45"  
"15:45:30"
"3:45:30.123 PM"
```

### Implementation Notes

1. **Override dateparse location behavior**: The library respects `time.Local`. We must:
   - Set `time.Local = time.UTC` for default parsing
   - For explicit timezone, parse first, then convert

2. **Return types**: All three functions return dictionaries with `__type` field:
   - `date()` → `{__type: "date", year, month, day, iso, unix, weekday}`
   - `time()` → `{__type: "time", hour, minute, second, iso}`
   - `datetime()` → `{__type: "datetime", year, month, day, hour, minute, second, unix, iso, weekday}`

3. **Error handling**: Return structured errors for:
   - Unparseable input: `FMT-0010` (new code for date parsing)
   - Ambiguous input in strict mode: `FMT-0011`

### Affected Components
- `pkg/parsley/evaluator/evaluator.go` — Replace `time()` with `datetime()`, add `date()`, `time()`
- `pkg/parsley/evaluator/eval_datetime.go` — Core parsing logic
- `pkg/parsley/evaluator/introspect.go` — Update builtin metadata
- `pkg/parsley/errors/codes.go` — Add FMT-0010, FMT-0011
- `go.mod` — Add `github.com/araddon/dateparse` dependency

### Dependencies
- External: `github.com/araddon/dateparse` (MIT license)
- Internal: None

### Edge Cases & Constraints
1. **Ambiguous numeric dates (01/02/03)** — Uses locale to determine order; `strict` to error
2. **Localized month names** — "avril", "März", "gennaio" recognized with appropriate locale
3. **Two-digit years** — Follows dateparse conventions (50-year window)
4. **Timezone-free input** — Always interpreted as UTC unless `timezone` specified
5. **Invalid formats** — Return FMT-0010 error with original input in message
6. **Time-only with date()** — Error: "Expected date, got time-only input"
7. **Date-only with time()** — Error: "Expected time, got date-only input"
8. **Unknown locale** — Fall back to en-US with warning in dev mode

### Locale Support
Initial locales to support (matching existing format() locales):
- `en-US` — MM/DD/YYYY (default)
- `en-GB` — DD/MM/YYYY
- `fr-FR` — DD/MM/YYYY, French month names
- `de-DE` — DD.MM.YYYY, German month names
- `es-ES` — DD/MM/YYYY, Spanish month names

Additional locales can be added incrementally.

### Duration Enhancement (Future)
Consider enhancing `duration()` for human-readable input:
```parsley
duration("2 hours 30 minutes")  // Human readable
duration("2h30m")               // Already supported
```
This is a separate smaller change, can be done as follow-up.

## Related
- Existing datetime literals: `@2024-12-25`, `@now`, `@12:30`
- Existing `duration()` function
- Current `time()` function (being replaced by `datetime()`)
