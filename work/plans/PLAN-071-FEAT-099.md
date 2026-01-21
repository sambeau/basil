---
id: PLAN-071
feature: FEAT-099
title: "Implementation Plan for Flexible Date/Time Parsing"
status: complete
created: 2026-01-21
completed: 2026-01-21
---

# Implementation Plan: FEAT-099 Flexible Date/Time Parsing

## Overview
Implement `date()`, `time()`, and `datetime()` functions for flexible parsing of human-readable date/time strings. Uses araddon/dateparse library with locale-aware format interpretation and UTC-by-default timezone handling.

## Prerequisites
- [ ] Add `github.com/araddon/dateparse` dependency
- [ ] Review existing `time()` builtin usage in tests/examples

## Tasks

### Task 1: Add dateparse dependency
**Files**: `go.mod`, `go.sum`
**Estimated effort**: Small

Steps:
1. Run `go get github.com/araddon/dateparse`
2. Run `go mod tidy`
3. Verify import works

Tests:
- Build succeeds

---

### Task 2: Add error codes FMT-0010, FMT-0011
**Files**: `pkg/parsley/errors/codes.go`
**Estimated effort**: Small

Steps:
1. Add `FMT-0010`: "Invalid date/time: cannot parse %q"
2. Add `FMT-0011`: "Ambiguous date: %q could be interpreted multiple ways"
3. Add helper functions in evaluator for these errors

Tests:
- Error messages display correctly

---

### Task 3: Create locale-aware parsing configuration
**Files**: `pkg/parsley/evaluator/eval_datetime.go` (new section)
**Estimated effort**: Medium

Steps:
1. Define locale config struct with dayFirst, month names
2. Create locale registry for en-US, en-GB, fr-FR, de-DE, es-ES
3. Implement `getLocaleConfig(locale string)` function
4. Define month name mappings per locale:
   - French: janvier, février, mars, avril, mai, juin, juillet, août, septembre, octobre, novembre, décembre
   - German: Januar, Februar, März, April, Mai, Juni, Juli, August, September, Oktober, November, Dezember
   - Spanish: enero, febrero, marzo, abril, mayo, junio, julio, agosto, septiembre, octubre, noviembre, diciembre

```go
type LocaleConfig struct {
    DayFirst   bool
    DotSep     bool              // Use dots as date separator (German)
    MonthNames map[string]int    // Localized month name → number
}

var localeConfigs = map[string]*LocaleConfig{
    "en-US": {DayFirst: false, DotSep: false, MonthNames: englishMonths},
    "en-GB": {DayFirst: true, DotSep: false, MonthNames: englishMonths},
    "fr-FR": {DayFirst: true, DotSep: false, MonthNames: frenchMonths},
    "de-DE": {DayFirst: true, DotSep: true, MonthNames: germanMonths},
    "es-ES": {DayFirst: true, DotSep: false, MonthNames: spanishMonths},
}
```

Tests:
- `getLocaleConfig("en-US")` returns DayFirst=false
- `getLocaleConfig("en-GB")` returns DayFirst=true
- `getLocaleConfig("unknown")` returns en-US default

---

### Task 4: Implement core parsing function
**Files**: `pkg/parsley/evaluator/eval_datetime.go`
**Estimated effort**: Large

Steps:
1. Implement `parseFlexibleDateTime(input string, locale *LocaleConfig, tz string, strict bool)` 
2. Pre-process input for localized month names (replace with English)
3. Configure dateparse based on locale (dayFirst setting)
4. Call dateparse.ParseIn() with UTC or specified timezone
5. Return (time.Time, kind string, error) where kind is "date", "time", or "datetime"
6. Detect ambiguous dates when strict mode enabled

```go
func parseFlexibleDateTime(input string, locale *LocaleConfig, tzName string, strict bool) (time.Time, string, error) {
    // 1. Normalize localized month names to English
    normalized := normalizeMonthNames(input, locale)
    
    // 2. Load timezone (default UTC)
    loc := time.UTC
    if tzName != "" {
        var err error
        loc, err = time.LoadLocation(tzName)
        if err != nil {
            return time.Time{}, "", err
        }
    }
    
    // 3. Configure dateparse options
    if locale.DayFirst {
        // Use ParseIn with PreferDayFirst option
    }
    
    // 4. Parse with dateparse
    t, err := dateparse.ParseIn(normalized, loc)
    if err != nil {
        return time.Time{}, "", err
    }
    
    // 5. Detect kind based on what was parsed
    kind := detectKind(input, t)
    
    return t, kind, nil
}
```

Tests:
- "22 April 2005" → 2005-04-22 (date)
- "3:45 PM" → 15:45:00 (time)
- "April 22, 2005 3:45 PM" → full datetime
- "01/02/2005" with en-US → Jan 2
- "01/02/2005" with en-GB → Feb 1

---

### Task 5: Implement `date()` builtin
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium

Steps:
1. Add `date` to getBuiltins() map
2. Accept (input, options?) arguments
3. Extract locale, strict, timezone from options
4. Call parseFlexibleDateTime()
5. Verify result is date-only (or accept datetime and strip time)
6. Return date dict via existing `timeToDictWithKind(t, "date", env)`

```go
"date": {
    Fn: func(args ...Object) Object {
        if len(args) < 1 || len(args) > 2 {
            return newArityErrorRange("date", len(args), 1, 2)
        }
        
        env := NewEnvironment()
        
        // Parse input
        input, ok := args[0].(*String)
        if !ok {
            return newTypeError("TYPE-0012", "date", "a string", args[0].Type())
        }
        
        // Extract options
        locale := "en-US"
        strict := false
        timezone := "UTC"
        if len(args) == 2 {
            opts, ok := args[1].(*Dictionary)
            if !ok {
                return newTypeError("TYPE-0006", "date", "a dictionary", args[1].Type())
            }
            // ... extract locale, strict, timezone
        }
        
        // Parse
        localeConfig := getLocaleConfig(locale)
        t, kind, err := parseFlexibleDateTime(input.Value, localeConfig, timezone, strict)
        if err != nil {
            return newDateParseError("FMT-0010", input.Value, err)
        }
        
        // Return as date
        return timeToDictWithKind(t, "date", env)
    },
},
```

Tests:
- `date("22 April 2005")` → date dict
- `date("2005-04-22")` → date dict
- `date("01/02/2005")` → Jan 2 (US)
- `date("01/02/2005", {locale: "en-GB"})` → Feb 1 (UK)
- `date("22 avril 2005", {locale: "fr-FR"})` → April 22
- `date("invalid")` → FMT-0010 error

---

### Task 6: Implement `time()` builtin (new, time-only)
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium

Steps:
1. Add new `time` builtin (replacing old one)
2. Accept (input, options?) arguments
3. Parse time-only strings: "3:45 PM", "15:45", "15:45:30.123"
4. Return time dict with `__type: "time"`

```go
"time": {
    Fn: func(args ...Object) Object {
        if len(args) < 1 || len(args) > 2 {
            return newArityErrorRange("time", len(args), 1, 2)
        }
        
        env := NewEnvironment()
        
        input, ok := args[0].(*String)
        if !ok {
            return newTypeError("TYPE-0012", "time", "a string", args[0].Type())
        }
        
        // Parse time-only
        t, err := parseTimeOnly(input.Value)
        if err != nil {
            return newDateParseError("FMT-0010", input.Value, err)
        }
        
        return timeToDictWithKind(t, "time", env)
    },
},
```

Tests:
- `time("3:45 PM")` → {hour: 15, minute: 45, second: 0}
- `time("15:45")` → {hour: 15, minute: 45, second: 0}
- `time("15:45:30.123")` → includes milliseconds
- `time("April 22")` → FMT-0010 error (not a time)

---

### Task 7: Implement `datetime()` builtin
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium

Steps:
1. Rename/replace existing `time` builtin logic as `datetime`
2. Add locale and strict options support
3. Keep existing functionality: string, integer (unix), dictionary input
4. Return datetime dict

```go
"datetime": {
    Fn: func(args ...Object) Object {
        if len(args) < 1 || len(args) > 2 {
            return newArityErrorRange("datetime", len(args), 1, 2)
        }
        
        env := NewEnvironment()
        
        // Extract options first if provided
        locale := "en-US"
        strict := false  
        timezone := "UTC"
        var delta *Dictionary
        
        if len(args) == 2 {
            opts, ok := args[1].(*Dictionary)
            if ok {
                // Extract locale, strict, timezone, or treat as delta
            }
        }
        
        switch arg := args[0].(type) {
        case *String:
            localeConfig := getLocaleConfig(locale)
            t, _, err := parseFlexibleDateTime(arg.Value, localeConfig, timezone, strict)
            if err != nil {
                return newDateParseError("FMT-0010", arg.Value, err)
            }
            if delta != nil {
                t = applyDelta(t, delta, env)
            }
            return timeToDictWithKind(t, "datetime", env)
            
        case *Integer:
            // Unix timestamp
            t := time.Unix(arg.Value, 0).UTC()
            return timeToDictWithKind(t, "datetime", env)
            
        case *Dictionary:
            // From dict components
            t, err := dictToTime(arg, env)
            if err != nil {
                return newFormatError("FMT-0004", err)
            }
            return timeToDictWithKind(t, "datetime", env)
        }
        
        return newTypeError("TYPE-0012", "datetime", "a string, integer, or dictionary", args[0].Type())
    },
},
```

Tests:
- `datetime("April 22, 2005 3:45 PM")` → full datetime
- `datetime("2005-04-22T15:45:00Z")` → ISO 8601
- `datetime(1682157900)` → Unix timestamp
- `datetime({year: 2005, month: 4, day: 22})` → from dict
- `datetime("01/02/2005 3pm", {locale: "en-GB"})` → Feb 1

---

### Task 8: Update introspect.go metadata
**Files**: `pkg/parsley/evaluator/introspect.go`
**Estimated effort**: Small

Steps:
1. Update `time` entry to describe new time-only parsing
2. Add `date` entry
3. Add `datetime` entry
4. Update valid.time if needed

Tests:
- `inspect(date)` returns correct metadata
- `inspect(time)` returns correct metadata
- `inspect(datetime)` returns correct metadata

---

### Task 9: Update existing tests
**Files**: `pkg/parsley/tests/*_test.go`
**Estimated effort**: Medium

Steps:
1. Find all tests using `time()` builtin
2. Update to use `datetime()` where appropriate
3. Ensure no regressions

Tests:
- All existing tests pass with updated function names

---

### Task 10: Add comprehensive test suite
**Files**: `pkg/parsley/tests/datetime_parse_test.go` (new)
**Estimated effort**: Large

Steps:
1. Create new test file for date/time parsing
2. Test date() with various formats
3. Test time() with various formats
4. Test datetime() with various formats
5. Test locale options
6. Test timezone options
7. Test strict mode
8. Test error cases

```go
func TestDateParsing(t *testing.T) {
    tests := []struct {
        input    string
        options  string
        expected string
    }{
        // Basic formats
        {`date("22 April 2005")`, "", `{__type: "date", year: 2005, month: 4, day: 22...}`},
        {`date("2005-04-22")`, "", `{__type: "date", year: 2005, month: 4, day: 22...}`},
        
        // Locale tests
        {`date("01/02/2005")`, "", `month: 1, day: 2`},  // US default
        {`date("01/02/2005", {locale: "en-GB"})`, "", `month: 2, day: 1`},  // UK
        {`date("22 avril 2005", {locale: "fr-FR"})`, "", `month: 4, day: 22`},
        
        // Error cases
        {`date("invalid")`, "", `error`},
    }
    // ...
}

func TestTimeParsing(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {`time("3:45 PM")`, `{__type: "time", hour: 15, minute: 45...}`},
        {`time("15:45:30")`, `{__type: "time", hour: 15, minute: 45, second: 30...}`},
    }
    // ...
}

func TestDatetimeParsing(t *testing.T) {
    // Full datetime tests
}

func TestStrictMode(t *testing.T) {
    // Ambiguous date tests with strict: true
}

func TestTimezoneOption(t *testing.T) {
    // Timezone conversion tests
}
```

---

### Task 11: Update documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`, `docs/parsley/manual/builtins/datetime.md`, `
**Estimated effort**: Medium

Steps:
1. Document `date()` function with examples
2. Document `time()` function with examples  
3. Document `datetime()` function with examples
4. Add locale and timezone option documentation
5. Update CHEATSHEET with date parsing patterns

---

## Validation Checklist
- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Documentation updated
- [ ] work/BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-01-21 | Task 1: Add dependency | ✅ Complete | Added github.com/araddon/dateparse |
| 2026-01-21 | Task 2: Error codes | ✅ Complete | Added FMT-0011, FMT-0012 to errors.go |
| 2026-01-21 | Task 3: Locale config | ✅ Complete | 5 locales: en-US, en-GB, fr-FR, de-DE, es-ES |
| 2026-01-21 | Task 4: Core parsing | ✅ Complete | parseFlexibleDateTime with locale & timezone |
| 2026-01-21 | Task 5: date() | ✅ Complete | Returns Date kind with options |
| 2026-01-21 | Task 6: time() | ✅ Complete | Time-only parsing (breaking change) |
| 2026-01-21 | Task 7: datetime() | ✅ Complete | Replaces old time() behavior |
| 2026-01-21 | Task 8: Introspect | ✅ Complete | Updated metadata for all 3 functions |
| 2026-01-21 | Task 9: Update tests | ✅ Complete | 7 test files updated time()→datetime() |
| 2026-01-21 | Task 10: New tests | ✅ Complete | flexible_datetime_test.go with 30+ tests |
| 2026-01-21 | Task 11: Documentation | ✅ Complete | Updated reference.md and CHEATSHEET.md |

## Deferred Items
Items to add to work/BACKLOG.md after implementation:
- Human-readable duration parsing ("2 hours 30 minutes") — Separate enhancement
- Additional locales beyond initial 5 — Add as needed
- Natural language parsing ("next Friday", "2 days ago") — Future chrono-style enhancement
- Strict mode for time() and datetime() — May need refinement based on usage
