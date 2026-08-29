---
id: man-pars-datetime
title: "Datetime"
system: parsley
type: builtin
name: datetime
created: 2024-12-14
version: 0.2.0
author: "@sam"
keywords: datetime, date, time, timestamp, calendar, duration, temporal
---

## Datetime

Datetime values represent points in time with year, month, day, hour, minute, and second components. Parsley tracks the "kind" of each datetime—whether it's a full datetime, date-only, or time-only—and preserves this through operations. Datetimes support arithmetic with integers (seconds) and durations, comparisons, and can be combined using the intersection operator.

```parsley
let now = @now
let christmas = @2024-12-25
let meeting = @14:30

christmas.weekday   // "Wednesday"
meeting.hour        // 14

let countdown = christmas - @today
countdown.fmt()     // "in 11 days"
```

## Literals

Parsley supports three kinds of datetime literals, each with its own display format:

### Date Literals

Date-only values (no time component):

```parsley
@2024-12-25       // Christmas Day
@1969-07-20       // Moon landing
@2000-01-01       // Y2K
```

### Datetime Literals

Full datetime values with date and time:

```parsley
@2024-12-25T14:30:00    // Christmas afternoon
@2024-12-31T23:59:59    // New Year's Eve
@2024-01-01T00:00:00Z   // With timezone (UTC)
```

### Time Literals

Time-only values (uses current UTC date internally):

```parsley
@12:30          // 12:30 PM
@09:15          // 9:15 AM  
@23:59:59       // With seconds
```

### Special Literals

```parsley
@now            // Current datetime
@today          // Current date (synonym for @dateNow)
@dateNow        // Current date only (synonym for @now.date)
@timeNow        // Current time only (synonym for @now.time)
```

### Interpolated Datetime Templates

Use `@(...)` syntax for datetime literals with embedded expressions:

```parsley
let month = "06"
let day = "15"
let dt = @(2024-{month}-{day})
dt.day   // 15

let year = "2025"
let hour = "14"
let dt2 = @({year}-12-25T{hour}:30:00)
dt2.year   // 2025
dt2.hour   // 14

// Expressions work too
let baseDay = 10
let dt3 = @(2024-12-{baseDay + 5})
dt3.day    // 15
```

### The `date()`, `time()` and `datetime()` Functions

Create datetime values programmatically. `date()` parses a date, `time()` parses a time-only value, and `datetime()` parses a full datetime from a string, a Unix timestamp, or a dictionary of components:

```parsley
date("2024-12-25")                          // Parse ISO date
time("14:30")                               // Parse time-only value
datetime("2024-12-25T14:30:00")             // Parse ISO datetime
datetime(1735142400)                        // Unix timestamp
datetime({year: 2024, month: 12, day: 25})  // From components
```

## Operators

### Addition (+)

Add seconds to a datetime:

```parsley
@2024-12-25 + 86400        // Add 1 day (86400 seconds)
@2024-12-25T12:00:00 + 3600 // Add 1 hour
```

Add a duration to a datetime:

```parsley
@2024-12-25 + @1d          // December 26
@2024-12-25 + @2h30m       // December 25 at 02:30
@2024-01-15 + @1mo         // February 15
```

### Subtraction (-)

Subtract seconds from a datetime:

```parsley
@2024-12-25 - 86400        // December 24
```

Subtract a duration:

```parsley
@2024-12-25 - @7d          // December 18
@2024-03-01 - @1mo         // February 1
```

Subtract two datetimes to get a duration:

```parsley
@2024-12-25 - @2024-12-20  // @5d (5-day duration)
@2024-12-25T14:00:00 - @2024-12-25T12:00:00  // @2h
```

### Comparison Operators

Compare any datetime values:

```parsley
@2024-12-25 > @2024-12-24   // true
@12:30 < @14:00             // true
@2024-12-25 == @2024-12-25  // true
@2024-12-25 != @2024-12-24  // true
@12:30 <= @12:30            // true
@2024-12-25 >= @2024-12-01  // true
```

### Intersection Operator (&&)

Combine date and time components:

```parsley
@2024-12-25 && @14:30       // @2024-12-25T14:30:00
@09:15 && @2024-03-15       // @2024-03-15T09:15:00
```

Replace components in a datetime:

```parsley
@2024-12-25T08:00:00 && @14:30      // @2024-12-25T14:30:00 (replace time)
@2024-12-25T08:00:00 && @2025-01-01 // @2025-01-01T08:00:00 (replace date)
```

| Expression | Result |
|------------|--------|
| `Date && Time` | DateTime (combine) |
| `Time && Date` | DateTime (combine) |
| `DateTime && Time` | DateTime (replace time) |
| `DateTime && Date` | DateTime (replace date) |
| `Date && Date` | Error |
| `Time && Time` | Error |
| `DateTime && DateTime` | Error |

## Properties

All datetime values have these properties:

| Property | Type | Description |
|----------|------|-------------|
| `.year` | Integer | Year number |
| `.month` | Integer | Month (1-12) |
| `.day` | Integer | Day of month (1-31) |
| `.hour` | Integer | Hour (0-23) |
| `.minute` | Integer | Minute (0-59) |
| `.second` | Integer | Second (0-59) |
| `.weekday` | String | Day name ("Monday", etc.) |
| `.unix` | Integer | Unix timestamp |
| `.iso` | String | ISO 8601 string |
| `.kind` | String | Literal kind |

```parsley
let dt = @2024-12-25T14:30:45
dt.year      // 2024
dt.month     // 12
dt.day       // 25
dt.hour      // 14
dt.minute    // 30
dt.second    // 45
dt.weekday   // "Wednesday"
dt.unix      // 1735137045
dt.iso       // "2024-12-25T14:30:45Z"
dt.kind      // "datetime"
```

### Computed Properties

| Property | Type | Description |
|----------|------|-------------|
| `.date` | String | Date portion ("2024-12-25") |
| `.time` | String | Time portion ("14:30:45" or "14:30") |
| `.dayOfYear` | Integer | Day number (1-366) |
| `.week` | Integer | ISO week number (1-53) |
| `.timestamp` | Integer | Alias for `.unix` |

```parsley
let dt = @2024-12-25T14:30:00
dt.date       // "2024-12-25"
dt.time       // "14:30"
dt.dayOfYear  // 360
dt.week       // 52
```

### Kind Property

The `kind` property indicates what type of datetime literal was used:

| Literal | Kind | String Output |
|---------|------|---------------|
| `@2024-12-25` | `"date"` | `"2024-12-25"` |
| `@2024-12-25T14:30:00` | `"datetime"` | `"2024-12-25T14:30:00Z"` |
| `@12:30` | `"time"` | `"12:30"` |
| `@12:30:45` | `"time_seconds"` | `"12:30:45"` |

Kind is preserved through arithmetic:

```parsley
let dateKind = (@2024-12-25 + @1d).kind            // "date"
let timeKind = (@12:30 + 3600).kind                // "time"
let datetimeKind = (@2024-12-25T14:30:00 + @1h).kind  // "datetime"
```

## Methods

### fmt()

The primary formatting method with multiple overloads.

Every style prints what the value holds and nothing more: a date gets a date, a
time gets a time, and a datetime gets its date followed by its time. The style
chooses how the *date* is written — the time is always 24-hour and the same in
every locale, after a comma.

```parsley
@2024-12-25.fmt("long")           // "December 25, 2024"       — a date
@14:30.fmt("long")                // "14:30"                   — a time
@14:30:45.fmt("long")             // "14:30:45"                — seconds kept
@2024-12-25T14:30:00.fmt("long")  // "December 25, 2024, 14:30" — both
```

A datetime shows seconds only when it has some, and shows no time at all when
it sits at exact midnight — which is where a date ends up when something widens
it into a datetime, such as `datetime("2024-12-25")` or a date read from a
database column:

```parsley
@2024-12-25T14:30:45.medium()   // "Dec 25, 2024, 14:30:45"
@2024-12-25T00:00:00.medium()   // "Dec 25, 2024"
datetime("2024-12-25").medium() // "Dec 25, 2024"
```

For full precision every time, use [`.iso`](#properties) or `toString()`.

#### Usage: fmt()

Format with default style (medium) and locale (en-US):

```parsley
@2024-12-25.fmt()       // "Dec 25, 2024"
@14:30.fmt()            // "14:30"
```

#### Usage: fmt(style)

Format with a named style:

```parsley
let dt = @2024-12-25

dt.fmt("short")         // "12/25/24"
dt.fmt("medium")        // "Dec 25, 2024"
dt.fmt("long")          // "December 25, 2024"
dt.fmt("full")          // "Wednesday, December 25, 2024"
```

#### Usage: fmt(style, locale)

Format with style and locale:

```parsley
let dt = @2024-12-25

dt.fmt("long", "de-DE")   // "25. Dezember 2024"
dt.fmt("long", "fr-FR")   // "25 décembre 2024"
dt.fmt("full", "de-DE")   // "Mittwoch, 25. Dezember 2024"
dt.fmt("full", "es-ES")   // "miércoles, 25 de diciembre de 2024"
```

#### Usage: fmt(options)

Format with an options dictionary:

```parsley
@2024-12-25.fmt({style: "full"})                    // "Wednesday, December 25, 2024"
@2024-12-25.fmt({style: "long", locale: "de-DE"})   // "25. Dezember 2024"
```

### short()

Compact date format:

```parsley
@2024-12-25.short()           // "12/25/24"
@2024-12-25T14:30:00.short()  // "12/25/24, 14:30"

// With locale:
@2024-12-25.short("de-DE")    // "25.12.24"
```

### medium()

Balanced format (default style):

```parsley
@2024-12-25.medium()          // "Dec 25, 2024"
@2024-12-25T14:30:00.medium() // "Dec 25, 2024, 14:30"

// With locale:
@2024-12-25.medium("de-DE")   // "25. Dez. 2024"
```

### long()

Verbose format with full month name:

```parsley
@2024-12-25.long()            // "December 25, 2024"
@2024-12-25T14:30:00.long()   // "December 25, 2024, 14:30"

// With locale:
@2024-12-25.long("de-DE")     // "25. Dezember 2024"
@2024-12-25.long("fr-FR")     // "25 décembre 2024"
```

### full()

Maximum context with day of week:

```parsley
@2024-12-25.full()            // "Wednesday, December 25, 2024"
@2024-12-25T14:30:00.full()   // "Wednesday, December 25, 2024, 14:30"

// With locale:
@2024-12-25.full("de-DE")     // "Mittwoch, 25. Dezember 2024"
@2024-12-25.full("es-ES")     // "miércoles, 25 de diciembre de 2024"
```

### format()

Alias for `fmt()`. Retained for backward compatibility:

```parsley
@2024-12-25.format()          // "Dec 25, 2024"
@2024-12-25.format("long")    // "December 25, 2024"
@2024-12-25.format("long", "de-DE")  // "25. Dezember 2024"
```

### repr()

Returns a parseable literal representation:

```parsley
@2024-12-25.repr()            // "@2024-12-25"
@2024-12-25T14:30:00.repr()   // "@2024-12-25T14:30:00Z"
@14:30.repr()                 // "@14:30"
```

### toJSON()

Returns the JSON representation (ISO 8601 string):

```parsley
@2024-12-25.toJSON()          // "\"2024-12-25\""
@2024-12-25T14:30:00.toJSON() // "\"2024-12-25T14:30:00Z\""
```

### inspect()

Returns a debug dictionary with type information:

```parsley
@2024-12-25.inspect()
// {__type: "datetime", kind: "date", year: 2024, month: 12, day: 25, hour: 0, minute: 0, second: 0,
//  weekday: "Wednesday", unix: 1735084800, iso: "2024-12-25T00:00:00Z"}
```

### toDict()

Returns a clean dictionary for reconstruction (without `__type`):

```parsley
@2024-12-25.toDict()
// {kind: "date", year: 2024, month: 12, day: 25, hour: 0, minute: 0, second: 0,
//  weekday: "Wednesday", unix: 1735084800, iso: "2024-12-25T00:00:00Z"}
```

Useful for serialization or passing datetime data to other systems.

### toBox()

Renders the datetime in a box diagram:

```parsley
@2024-12-25.toBox()
```

```
┌─────────┬───────────┐
│       datetime        │
├─────────┬───────────┤
│ year    │ 2024      │
├─────────┼───────────┤
│ month   │ 12        │
├─────────┼───────────┤
│ day     │ 25        │
├─────────┼───────────┤
│ hour    │ 0         │
├─────────┼───────────┤
│ minute  │ 0         │
├─────────┼───────────┤
│ second  │ 0         │
├─────────┼───────────┤
│ weekday │ Wednesday │
└─────────┴───────────┘
```

### dayOfYear()

Returns the day number within the year (1-366):

```parsley
@2024-01-01.dayOfYear()  // 1
@2024-12-31.dayOfYear()  // 366 (leap year)
```

### week()

Returns the ISO week number (1-53):

```parsley
@2024-01-01.week()   // 1
@2024-12-25.week()   // 52
```

### timestamp()

Returns the Unix timestamp (alias for `.unix`):

```parsley
@2024-12-25T00:00:00.timestamp()  // 1735084800
```

## Formatting Styles Summary

| Style | Example Output |
|-------|----------------|
| `short` | `"12/25/24"` |
| `medium` (default) | `"Dec 25, 2024"` |
| `long` | `"December 25, 2024"` |
| `full` | `"Wednesday, December 25, 2024"` |

## Duration Arithmetic

Durations represent time spans and work naturally with datetimes:

### Duration Literals

```parsley
@1d          // 1 day
@2h          // 2 hours
@30m         // 30 minutes
@1d2h30m     // Combined: 1 day, 2 hours, 30 minutes
@1mo         // 1 month
@1y          // 1 year
@-1d         // Negative (yesterday)
```

### Adding Durations

```parsley
let today = @today
today + @7d          // Next week
today + @1mo         // Next month
today + @1y          // Next year

@2024-02-28 + @1d    // February 29 (leap year)
@2024-01-31 + @1mo   // March 2 (month arithmetic overflows a short month)
```

### Duration Results

When subtracting datetimes, you get a duration:

```parsley
let future = @2025-01-01
let now = @today
let remaining = future - now

remaining.fmt()      // "in 18 days"
```

## Common Patterns

### Calculate Age

```parsley
let birthdate = @1990-05-15
let today = @today
let age = (today - birthdate) / @1y  // Approximate years
```

### Schedule Future Dates

```parsley
let deadline = @today + @14d         // Two weeks from now
let nextMeeting = @today + @1mo      // One month from now
```

### Format for Display

```parsley
let event = @2024-12-25T19:00:00

// Different formats for different contexts
event.short()                        // "12/25/24, 19:00"
event.medium()                       // "Dec 25, 2024, 19:00"
event.long()                         // "December 25, 2024, 19:00"
event.full()                         // "Wednesday, December 25, 2024, 19:00"
event.long("de-DE")                  // "25. Dezember 2024, 19:00"
```

### Combine Date and Time

```parsley
let eventDate = @2024-12-25
let eventTime = @19:00

let eventDatetime = eventDate && eventTime
// @2024-12-25T19:00:00
```

### Compare Dates

```parsley
let deadline = @2024-12-31
let today = @today

if (today > deadline) {
    "Deadline passed!"
} else {
    "Still time remaining"
}
```

## Time Zones

All datetime values in Parsley are stored in UTC. When parsing datetime strings with timezone indicators (e.g., `Z` suffix or `+00:00`), they are converted to UTC. Time-only literals use the current UTC date.

```parsley
@2024-12-25T14:30:00Z     // Explicit UTC
@2024-12-25T14:30:00      // Interpreted as UTC
@12:30                    // Current UTC date at 12:30
```

## See Also

- [Duration](duration.md) — time durations and date arithmetic
- [Numbers](numbers.md) — numeric types used in date components
- [Strings](strings.md) — `.fmt()` for date formatting
- [Types](../fundamentals/types.md) — datetime in the type system
- [@std/valid](../stdlib/valid.md) — `date()` and `time()` format validators