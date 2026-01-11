---
id: FEAT-070
title: "Datetime Timezone Support"
status: draft
priority: medium
created: 2025-12-14
author: "@sambeau"
---

# FEAT-070: Datetime Timezone Support

## Summary
Add timezone support to datetime values, allowing conversion from UTC to any IANA timezone or fixed offset for display purposes. All storage and arithmetic remains UTC-based; timezones are applied as a presentation layer. Also includes a Basil middleware for detecting browser timezone and a `<local-time>` web component for client-side localization.

## User Story
As a developer building web applications, I want to display datetime values in users' local timezones so that event times, deadlines, and timestamps are meaningful to users regardless of their location.

## Acceptance Criteria
- [ ] `dt.inZone("America/New_York")` returns datetime with timezone metadata
- [ ] `dt.inZone("+05:30")` supports fixed UTC offset syntax
- [ ] `.hour`, `.minute`, `.time`, `.date` properties respect the timezone
- [ ] `.format()` respects the timezone
- [ ] `.iso` outputs offset format when timezone is set (e.g., `2024-12-25T09:30:00-05:00`)
- [ ] `.zone` property returns the timezone name or offset
- [ ] `.utc()` method returns datetime without timezone (back to UTC)
- [ ] Basil middleware adds `X-Timezone` header from browser detection
- [ ] `<local-time>` web component auto-localizes UTC datetimes client-side
- [ ] Invalid timezone names produce clear error messages
- [ ] Documentation updated

## Design Decisions

### Storage: Always UTC
Datetimes remain stored as UTC internally (the `unix` field). The `inZone()` method doesn't change the instant in time—it adds metadata for how to *present* that instant.

```parsley
let utc = @2024-12-25T14:30:00
let ny = utc.inZone("America/New_York")

utc.unix == ny.unix    // true (same instant)
utc.hour != ny.hour    // 14 vs 9 (different presentation)
```

### Method: `inZone()` vs `inTimezone()`
Using `inZone()` for brevity—matches Luxon and is shorter than "timezone". The zone can be an IANA name or offset.

### No Magic "local"
There's no `inZone("local")` because "local" is ambiguous:
- Server local? (where Basil runs)
- Browser local? (where user is—only known client-side)
- Data local? (where event occurred—semantic, must be stored)

Instead, explicitly pass the timezone from the request.

### Offset Syntax
Support both IANA names and fixed offsets:
```parsley
dt.inZone("America/New_York")  // IANA name (handles DST)
dt.inZone("Europe/London")
dt.inZone("+05:30")            // Fixed offset (India)
dt.inZone("-08:00")            // Fixed offset (PST)
dt.inZone("UTC")               // Explicit UTC
```

### ISO Output Format
When timezone is set, `.iso` includes the offset:
```parsley
@2024-12-25T14:30:00.iso                        // "2024-12-25T14:30:00Z"
@2024-12-25T14:30:00.inZone("America/New_York").iso  // "2024-12-25T09:30:00-05:00"
```

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Syntax Summary
```parsley
// Convert to timezone for display
let ny = @now.inZone("America/New_York")
ny.hour          // Local hour (e.g., 9)
ny.time          // "09:30"
ny.date          // "2024-12-25"
ny.zone          // "America/New_York"
ny.iso           // "2024-12-25T09:30:00-05:00"
ny.format("long", "en-US")  // Uses timezone

// Return to UTC
ny.utc().hour    // 14

// In Basil handlers
let userTz = request.headers["X-Timezone"] ?? "UTC"
event.inZone(userTz).format("long")

// Fixed offset
dt.inZone("+05:30").format("long")
```

### Affected Components
- `pkg/parsley/evaluator/evaluator.go` — `timeToDictWithKind()` to store optional timezone
- `pkg/parsley/evaluator/methods.go` — Add `inZone()`, `utc()` methods to `evalDatetimeMethod()`
- `pkg/parsley/evaluator/evaluator.go` — `evalDatetimeComputedProperty()` to respect timezone
- `pkg/parsley/evaluator/evaluator.go` — `formatDateWithStyleAndLocale()` to apply timezone
- `server/middleware.go` — Timezone detection middleware
- New: `pkg/parsley/components/local-time.js` — Web component

### Go Implementation

Timezone conversion is straightforward in Go:

```go
// IANA timezone
loc, err := time.LoadLocation("America/New_York")
if err != nil {
    return newValidationError("VAL-0030", map[string]any{"Timezone": tzString})
}
localTime := time.Unix(unixTimestamp, 0).In(loc)

// Fixed offset (parse "+05:30" or "-08:00")
func parseFixedOffset(offset string) (*time.Location, error) {
    // Parse sign, hours, minutes
    // Return time.FixedZone(offset, totalSeconds)
}
```

### Dictionary Changes

Add optional `zone` field to datetime dictionaries:

```go
// In timeToDictWithKind, if zone is provided:
pairs["zone"] = &ast.StringLiteral{Value: zone}

// Store the location for offset calculation
pairs["_zoneOffset"] = &ast.IntegerLiteral{Value: offsetSeconds}
```

### Property Computation

In `evalDatetimeComputedProperty()`, apply timezone before computing:

```go
func evalDatetimeComputedProperty(dict *Dictionary, key string, env *Environment) Object {
    // Get unix timestamp
    unixTime := getUnixFromDict(dict, env)
    
    // Apply timezone if present
    t := time.Unix(unixTime, 0).UTC()
    if zoneExpr, ok := dict.Pairs["zone"]; ok {
        zone := Eval(zoneExpr, env).(*String).Value
        loc, _ := time.LoadLocation(zone)  // or parseFixedOffset
        t = t.In(loc)
    }
    
    switch key {
    case "hour":
        return &Integer{Value: int64(t.Hour())}
    // ... etc
    }
}
```

### Basil Middleware

Detect timezone from JavaScript and send as header:

```javascript
// Injected by Basil when timezone detection is enabled
(function() {
    const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
    // Store for subsequent requests
    sessionStorage.setItem('tz', tz);
    // Add to all fetch requests
    const originalFetch = window.fetch;
    window.fetch = function(url, options = {}) {
        options.headers = options.headers || {};
        options.headers['X-Timezone'] = tz;
        return originalFetch(url, options);
    };
})();
```

For traditional form submissions/page loads, use a cookie (despite mobility concerns, it's the only option for non-JS requests) or require the first request to redirect after setting.

### `<local-time>` Web Component

A web component that auto-localizes displayed datetimes:

```html
<!-- Server renders UTC -->
<local-time datetime="2024-12-25T14:30:00Z">December 25, 2024 at 14:30 UTC</local-time>

<!-- JS enhances to show local time -->
<local-time datetime="2024-12-25T14:30:00Z">December 25, 2024 at 9:30 AM</local-time>
```

```javascript
class LocalTime extends HTMLElement {
    connectedCallback() {
        const dt = new Date(this.getAttribute('datetime'));
        const format = this.getAttribute('format') || 'long';
        const locale = navigator.language;
        
        this.textContent = dt.toLocaleString(locale, {
            dateStyle: format,
            timeStyle: 'short'
        });
    }
}
customElements.define('local-time', LocalTime);
```

Usage in Parsley:
```parsley
<local-time datetime={event.iso}>
    event.format("long")  // Fallback for no-JS
</local-time>
```

### Edge Cases & Constraints
1. **DST transitions** — IANA zones handle this automatically; fixed offsets don't
2. **Invalid zones** — Return clear error with suggestion if close match exists
3. **Arithmetic with zoned datetimes** — Result keeps the zone, or reverts to UTC?
4. **Zone preservation** — Does `zonedDt + @1d` keep the zone? (Recommend: yes)

### Dependencies
- Go's `time.LoadLocation()` requires timezone database (usually bundled)
- Browser `Intl.DateTimeFormat` for client-side (universal support)

## HTML Components (Related)

Two HTML components for client-side timezone localization are designed in [html-components.md](../design/html-components.md):

### `<LocalTime>`
Client-side localized datetime display using `Intl.DateTimeFormat`:
```parsley
<LocalTime datetime={event.startTime}>
    event.startTime.format("long")  // Server fallback
</LocalTime>
```
Supports `format` (`short`, `long`, `full`, `date`, `time`) and `weekday` attributes.

### `<TimeRange>`
Smart datetime span display that collapses redundant information:
```parsley
<TimeRange start={session.start} end={session.end}>
    session.start.format("long") + " – " + session.end.time
</TimeRange>
// Same-day: "December 25, 2024, 9:00 AM – 11:00 AM"
// Multi-day: "December 25 – 27, 2024"
```

Both render custom elements (`<local-time>`, `<time-range>`) that JavaScript enhances. Server-rendered content provides no-JS fallback.

**Implementation**: These components will be implemented as part of the std/HTML library alongside other HTML components, not as part of FEAT-070 core.

## Implementation Notes
*Added during/after implementation*

## Related
- Plan: `work/plans/FEAT-070-plan.md` (to be created)
- Related: Datetime manual page (`docs/manual/builtins/datetime.md`)
