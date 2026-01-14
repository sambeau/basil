---
id: man-pars-duration
title: "Duration"
system: parsley
type: builtin
name: duration
created: 2025-12-15
version: 0.2.0
author: "@sam"
keywords: duration, time span, interval, period, relative time
---

## Duration

Duration values represent time spans—the difference between two points in time. Parsley tracks durations as two separate components: months (for calendar-based units) and seconds (for fixed-length units). This dual representation ensures accurate arithmetic with variable-length calendar months.

```parsley
let vacation = @2w
let project = @3mo
let meeting = @1h30m

vacation.seconds  // 1209600
project.months    // 3
meeting.format()  // "in 2 hours"
```

## Literals

Duration literals use the `@` prefix followed by one or more number-unit pairs.

### Supported Units

| Unit | Description | Storage |
|------|-------------|---------|
| `y` | Years | 12 months |
| `mo` | Months | months |
| `w` | Weeks | 604,800 seconds |
| `d` | Days | 86,400 seconds |
| `h` | Hours | 3,600 seconds |
| `m` | Minutes | 60 seconds |
| `s` | Seconds | seconds |

### Simple Durations

```parsley
@30s         // 30 seconds
@5m          // 5 minutes
@2h          // 2 hours
@7d          // 7 days
@2w          // 2 weeks
@6mo         // 6 months
@1y          // 1 year
```

### Compound Durations

Combine multiple units in a single literal (order: years, months, weeks, days, hours, minutes, seconds):

```parsley
@2h30m           // 2 hours, 30 minutes
@1d12h           // 1 day, 12 hours
@1y6mo           // 1 year, 6 months
@3w2d            // 3 weeks, 2 days
@1y2mo3w4d5h6m7s // All units combined
```

### Negative Durations

Prefix with `-` for negative durations (time in the past):

```parsley
@-1d         // 1 day ago
@-2h30m      // 2 hours 30 minutes ago
@-1y         // 1 year ago
```

## Constructor

The `duration()` function creates durations dynamically from strings or dictionaries. Use this when parsing user input or building durations from variables.

### From String

Parse a duration string using the same format as literals (without the `@` prefix):

```parsley
duration("30s")          // 30 seconds
duration("2h30m")        // 2 hours 30 minutes
duration("1y6mo")        // 1 year 6 months
duration("-1d")          // negative 1 day
```

### From Dictionary

Create a duration from named components:

```parsley
duration({seconds: 30})
duration({hours: 2, minutes: 30})
duration({years: 1, months: 6})
duration({days: 7})
```

Available keys: `years`, `months`, `weeks`, `days`, `hours`, `minutes`, `seconds`

```parsley
// All keys example
duration({
    years: 1,
    months: 2,
    weeks: 3,
    days: 4,
    hours: 5,
    minutes: 6,
    seconds: 7
})
```

### When to Use

Prefer literals for static durations; use `duration()` for dynamic values:

```parsley
// Static: use literals
let timeout = @30s
let deadline = @now + @7d

// Dynamic: use constructor
let userInput = "2h30m"
let parsed = duration(userInput)

let config = {hours: 8, minutes: 30}
let workday = duration(config)
```

## Operators

### Addition (+)

Add two durations together:

```parsley
@2h + @30m       // 2 hours 30 minutes
@1y + @6mo       // 1 year 6 months (18 months)
@1d + @-6h       // 18 hours (1 day minus 6 hours)
```

Add a duration to a datetime (commutative):

```parsley
@2024-12-25 + @7d     // January 1, 2025
@7d + @2024-12-25     // January 1, 2025 (same result)
@2024-01-15 + @1mo    // February 15, 2024
@12:00 + @2h30m       // 14:30
```

### Subtraction (-)

Subtract one duration from another:

```parsley
@1d - @6h        // 18 hours
@2y - @3mo       // 21 months
@1w - @1d        // 6 days
```

Subtract a duration from a datetime:

```parsley
@2024-12-25 - @7d     // December 18, 2024
@2024-03-01 - @1mo    // February 1, 2024
```

Subtract two datetimes to get a duration:

```parsley
@2024-12-25 - @2024-12-20     // 5 days
@2024-12-25T14:00:00 - @2024-12-25T12:00:00  // 2 hours
```

### Multiplication (*)

Multiply a duration by an integer:

```parsley
@2h * 3          // 6 hours
@1d * 7          // 1 week
@1mo * 6         // 6 months
```

### Division (/)

Divide a duration by an integer:

```parsley
@1d / 2          // 12 hours
@6mo / 3         // 2 months
@1h / 4          // 15 minutes
```

Divide two durations to get a ratio:

```parsley
@7d / @1d        // 7
@6mo / @1y       // 0.5
@2h / @30m       // 4

// Practical example: calculate age
let birthdate = @1990-05-15
let today = @today
let age = (today - birthdate) / @1y  // Approximate years
```

> **Note:** Division involving months uses an approximate conversion (1 month ≈ 30.44 days) for accurate ratios.

### Comparison Operators

Compare durations of the same type (seconds-only):

```parsley
@2h > @1h        // true
@30m < @1h       // true
@1d == @24h      // true
@1w != @6d       // true
@2h <= @2h       // true
@3d >= @2d       // true
```

> **Note:** Comparisons with month-based durations are not allowed due to variable month lengths.

## Properties

### Core Properties

| Property | Type | Description |
|----------|------|-------------|
| `.months` | Integer | Month component of the duration |
| `.seconds` | Integer | Seconds component of the duration |
| `.totalSeconds` | Integer | Total seconds (only for durations without months) |
| `.__type` | String | Always `"duration"` |

```parsley
let d = @1y2mo3d4h
d.months        // 14 (12 + 2)
d.seconds       // 273600 (3*86400 + 4*3600)
d.__type        // "duration"
```

For durations without months, `.totalSeconds` provides the complete duration:

```parsley
let d = @2d12h
d.totalSeconds  // 216000 (2*86400 + 12*3600)
d.seconds       // 216000 (same value)
```

For durations with months, `.totalSeconds` is not available:

```parsley
let d = @1y2mo
d.totalSeconds  // null (months have variable length)
d.months        // 14
d.seconds       // 0
```

### Computed Properties

These properties calculate derived values from the seconds component:

| Property | Type | Description |
|----------|------|-------------|
| `.days` | Integer | Total seconds as days (integer division) |
| `.hours` | Integer | Total seconds as hours (integer division) |
| `.minutes` | Integer | Total seconds as minutes (integer division) |

```parsley
let d = @2d12h30m
d.seconds       // 217800
d.days          // 2 (217800 / 86400)
d.hours         // 60 (217800 / 3600)
d.minutes       // 3630 (217800 / 60)
```

For week-based durations:

```parsley
let w = @1w
w.days          // 7
w.hours         // 168
w.minutes       // 10080
```

> **Note:** Computed properties return `null` for month-based durations since months have variable lengths (28-31 days):

```parsley
let y = @1y
y.days          // null
y.hours         // null
y.minutes       // null

let mixed = @1mo2d
mixed.days      // null (has month component)
```

## Methods

### format()

#### Usage: format()

Format the duration as relative time in the default locale (en-US):

```parsley
@1d.format()     // "tomorrow"
@-1d.format()    // "yesterday"
@7d.format()     // "next week"
@-7d.format()    // "last week"
@1mo.format()    // "next month"
@1y.format()     // "next year"
@2h.format()     // "in 2 hours"
@-30m.format()   // "30 minutes ago"
```

#### Usage: format(locale)

Format with a specific locale:

```parsley
@1d.format("de-DE")     // "morgen"
@-1d.format("de-DE")    // "gestern"
@7d.format("fr-FR")     // "la semaine prochaine"
@1mo.format("es-ES")    // "el próximo mes"
@1y.format("ja-JP")     // "来年"
```

### toDict()

Returns a clean dictionary for reconstruction (without `__type`):

```parsley
@2h30m.toDict()
// {months: 0, seconds: 9000}

@1y6mo.toDict()
// {months: 18, seconds: 0}
```

Useful for serialization or passing duration data to other systems.

### inspect()

Returns the full dictionary representation with `__type` for debugging:

```parsley
@2h30m.inspect()
// {__type: "duration", months: 0, seconds: 9000}

@1y6mo.inspect()
// {__type: "duration", months: 18, seconds: 0}
```

## String Conversion

Durations automatically convert to human-readable strings in output contexts:

```parsley
let d = @2h30m
d                        // "2 hours 30 minutes"
toString(d)              // "2 hours 30 minutes"
log("Time: " + toString(d))  // Time: 2 hours 30 minutes

let long = @1y2mo3d
toString(long)           // "1 year 2 months 3 days"
```

The string format uses plural forms appropriately:

```parsley
@1d             // "1 day"
@2d             // "2 days"
@1h             // "1 hour"
@3h             // "3 hours"
```

Zero duration:

```parsley
@0s             // "0 seconds"
```

## Month vs. Second Components

Parsley separates durations into two components to handle the fact that months have variable lengths (28-31 days):

### Seconds-Only Durations

Units `w`, `d`, `h`, `m`, `s` are stored as seconds:

```parsley
@1w             // 604800 seconds
@1d             // 86400 seconds
@1h             // 3600 seconds
@1m             // 60 seconds
@1s             // 1 second
```

These can be compared and have `.totalSeconds`:

```parsley
@7d == @1w      // true
@24h == @1d     // true
@60m == @1h     // true
```

### Month-Based Durations

Units `y` and `mo` are stored as months:

```parsley
@1y             // 12 months, 0 seconds
@6mo            // 6 months, 0 seconds
@1y6mo          // 18 months, 0 seconds
```

These cannot be compared (variable month lengths):

```parsley
@1y > @365d     // Error: cannot compare durations with month components
```

### Mixed Durations

Compound durations can have both:

```parsley
@1y2mo3d        // 14 months, 259200 seconds
```

## Common Patterns

### Calculate Time Until Event

```parsley
let deadline = @2025-01-01
let remaining = deadline - @today

remaining.format()              // "in 2 weeks"
remaining.seconds / 86400       // Days remaining
```

### Schedule Future Dates

```parsley
let nextReview = @today + @3mo
let followUp = @now + @2w
let reminder = @now + @1h30m
```

### Calculate Age

```parsley
let birthdate = @1990-05-15
let age = (@today - birthdate) / @1y
age.round()     // Approximate age in years
```

### Work Duration Calculations

```parsley
let taskTime = @2h30m
let tasks = 5
let totalTime = taskTime * tasks     // 12 hours 30 minutes

let workDay = @8h
let workers = 3
let perWorker = workDay / workers    // 2 hours 40 minutes each
```

### Time Zone-Safe Scheduling

Since durations add exact seconds (except months), they're predictable across time zones:

```parsley
let meeting = @2024-12-15T10:00:00
let buffer = @30m
let start = meeting - buffer         // 09:30 (exact)
```

### Relative Time Display

```parsley
let posted = @2024-12-10T14:00:00
let now = @now
let ago = now - posted

ago.format()    // "5 days ago" (depends on current date)
```

## Duration vs. Datetime Arithmetic

| Operation | Result |
|-----------|--------|
| `duration + duration` | Duration |
| `duration - duration` | Duration |
| `duration * integer` | Duration |
| `duration / integer` | Duration |
| `duration / duration` | Float (ratio) |
| `datetime + duration` | Datetime |
| `duration + datetime` | Datetime (commutative) |
| `datetime - duration` | Datetime |
| `datetime - datetime` | Duration |
| `integer + datetime` | Datetime |
| `datetime + integer` | Datetime |
