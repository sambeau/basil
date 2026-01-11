---
id: FEAT-054
title: "Replace now() with @now/@timeNow/@dateNow/@today Datetime Literals"
status: complete
priority: medium
created: 2025-12-09
completed: 2025-12-09
author: "@human"
---

# FEAT-054: Replace now() with @now/@timeNow/@dateNow/@today Datetime Literals

## Summary
Replace the `now()` builtin function with datetime literal syntax for creating current date/time values. Introduces `@now` (full datetime), `@timeNow` (time only), `@dateNow` (date only), and `@today` (synonym for `@dateNow`) to provide consistent, concise syntax for common temporal operations while aligning with Parsley's literal-based approach to values.

## User Story
As a Parsley developer, I want a simple, consistent way to get the current date or time using the same `@` literal syntax I use for fixed dates and times, so that my code is more readable and follows Parsley's principle of literals for values rather than functions.

## Acceptance Criteria
- [ ] `@now` creates a datetime dictionary with current date and time (replaces `now()`)
- [ ] `@timeNow` creates a time-only dictionary with current time
- [ ] `@dateNow` creates a date-only dictionary with current date
- [ ] `@today` works as synonym for `@dateNow`
- [ ] All four literals return properly typed dictionaries matching datetime/time/date structures
- [ ] `now()` builtin is deprecated (kept for backward compatibility with deprecation warning)
- [ ] Kind field is correctly set: "datetime" for `@now`, "time" for `@timeNow`, "date" for `@dateNow` and `@today`
- [ ] Tests cover all four literal forms
- [ ] Documentation updated to show new syntax as primary, `now()` as deprecated
- [ ] Error handling matches existing datetime literal behavior

## Design Decisions
- **Literal syntax over functions** — Aligns with Parsley's philosophy: literals for values (`@now`), functions for operations
- **`@` prefix** — Consistent with existing datetime literals (`@2024-12-09`, `@14:30`, `@1d`)
- **Multiple variants** — Different use cases need different granularity (full datetime vs date-only vs time-only)
- **`@today` synonym** — Natural language for date-only "today" is more intuitive than `@dateNow`
- **Deprecate but keep `now()`** — Graceful migration path; can remove in future major version
- **No arguments** — Literals are fixed-point references to "current moment"; timezone/format handled by methods

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/lexer/lexer.go` — Add recognition for `@now`, `@timeNow`, `@dateNow`, `@today` tokens
- `pkg/parsley/parser/parser.go` — Parse new datetime literal forms
- `pkg/parsley/ast/ast.go` — Extend DatetimeLiteral or add new node types if needed
- `pkg/parsley/evaluator/evaluator.go` — Evaluate new literals to current time dictionaries, add deprecation warning to `now()`
- `pkg/parsley/tests/datetime_test.go` — Add tests for all four literal forms
- `docs/parsley/reference.md` — Document new literals, mark `now()` as deprecated
- `docs/parsley/CHEATSHEET.md` — Update datetime section with new syntax

### Dependencies
- None — Uses existing datetime infrastructure

### Implementation Details

#### Lexer Changes
Add token recognition after `@` prefix:
```go
// In lexer when encountering @
if l.ch == '@' {
    l.readChar()
    
    // Check for special datetime literals
    if l.matchKeyword("now") {
        return token.Token{Type: token.DATETIME_NOW, Literal: "now"}
    }
    if l.matchKeyword("timeNow") {
        return token.Token{Type: token.TIME_NOW, Literal: "timeNow"}
    }
    if l.matchKeyword("dateNow") {
        return token.Token{Type: token.DATE_NOW, Literal: "dateNow"}
    }
    if l.matchKeyword("today") {
        return token.Token{Type: token.DATE_NOW, Literal: "today"} // same as dateNow
    }
    
    // ... existing datetime literal parsing
}
```

#### Token Types
```go
// In token package
const (
    DATETIME_NOW = "DATETIME_NOW"  // @now
    TIME_NOW     = "TIME_NOW"      // @timeNow
    DATE_NOW     = "DATE_NOW"      // @dateNow or @today
)
```

#### Parser
```go
// Register prefix parsers
p.registerPrefix(token.DATETIME_NOW, p.parseDatetimeNowLiteral)
p.registerPrefix(token.TIME_NOW, p.parseTimeNowLiteral)
p.registerPrefix(token.DATE_NOW, p.parseDateNowLiteral)

func (p *Parser) parseDatetimeNowLiteral() ast.Expression {
    return &ast.DatetimeNowLiteral{Token: p.curToken, Kind: "datetime"}
}

func (p *Parser) parseTimeNowLiteral() ast.Expression {
    return &ast.DatetimeNowLiteral{Token: p.curToken, Kind: "time"}
}

func (p *Parser) parseDateNowLiteral() ast.Expression {
    return &ast.DatetimeNowLiteral{Token: p.curToken, Kind: "date"}
}
```

#### AST Node
```go
// In ast package
type DatetimeNowLiteral struct {
    Token token.Token
    Kind  string // "datetime", "time", or "date"
}

func (dnl *DatetimeNowLiteral) expressionNode()      {}
func (dnl *DatetimeNowLiteral) TokenLiteral() string { return dnl.Token.Literal }
func (dnl *DatetimeNowLiteral) String() string       { return "@" + dnl.Token.Literal }
```

#### Evaluator
```go
// In Eval() switch
case *ast.DatetimeNowLiteral:
    return evalDatetimeNowLiteral(node, env)

func evalDatetimeNowLiteral(node *ast.DatetimeNowLiteral, env *Environment) Object {
    now := time.Now()
    
    switch node.Kind {
    case "datetime":
        // @now - full datetime
        return &Dictionary{
            Pairs: map[string]ast.Expression{
                "year":   &ast.IntegerLiteral{Value: int64(now.Year())},
                "month":  &ast.IntegerLiteral{Value: int64(now.Month())},
                "day":    &ast.IntegerLiteral{Value: int64(now.Day())},
                "hour":   &ast.IntegerLiteral{Value: int64(now.Hour())},
                "minute": &ast.IntegerLiteral{Value: int64(now.Minute())},
                "second": &ast.IntegerLiteral{Value: int64(now.Second())},
                "kind":   &ast.StringLiteral{Value: "datetime"},
            },
            Env: env,
        }
        
    case "time":
        // @timeNow - time only
        return &Dictionary{
            Pairs: map[string]ast.Expression{
                "hour":   &ast.IntegerLiteral{Value: int64(now.Hour())},
                "minute": &ast.IntegerLiteral{Value: int64(now.Minute())},
                "second": &ast.IntegerLiteral{Value: int64(now.Second())},
                "kind":   &ast.StringLiteral{Value: "time"},
            },
            Env: env,
        }
        
    case "date":
        // @dateNow or @today - date only
        return &Dictionary{
            Pairs: map[string]ast.Expression{
                "year":  &ast.IntegerLiteral{Value: int64(now.Year())},
                "month": &ast.IntegerLiteral{Value: int64(now.Month())},
                "day":   &ast.IntegerLiteral{Value: int64(now.Day())},
                "kind":  &ast.StringLiteral{Value: "date"},
            },
            Env: env,
        }
        
    default:
        return newError("Invalid datetime now kind: %s", node.Kind)
    }
}

// Deprecate now() builtin
"now": {
    Fn: func(args ...Object) Object {
        if len(args) != 0 {
            return newArityError("now", len(args), 0)
        }
        
        // Log deprecation warning (implementation-specific)
        logDeprecation("now() is deprecated, use @now instead")
        
        // Return same as @now for compatibility
        now := time.Now()
        return createDatetimeDictionary(now, "datetime")
    },
},
```

### Examples

#### Basic Usage
```parsley
// Full datetime (replacement for now())
let currentTime = @now
// → {year: 2025, month: 12, day: 9, hour: 14, minute: 30, second: 45, kind: "datetime"}

// Time only
let time = @timeNow
// → {hour: 14, minute: 30, second: 45, kind: "time"}

// Date only
let date = @dateNow
// → {year: 2025, month: 12, day: 9, kind: "date"}

// Today (synonym)
let today = @today
// → {year: 2025, month: 12, day: 9, kind: "date"}
```

#### Comparisons
```parsley
// Check if event is today
let eventDate = @2025-12-25
if eventDate == @today {
    print("Event is today!")
}

// Check if time is after 5 PM
if @timeNow > @17:00 {
    print("It's evening!")
}
```

#### Calculations
```parsley
// Days until Christmas
let daysUntil = @2025-12-25 - @today
print("Days until Christmas: {daysUntil}")

// Time remaining in day
let endOfDay = @23:59:59
let timeLeft = endOfDay - @timeNow
```

#### Formatting
```parsley
// Format current date
@today.format("long", "en-US")
// → "December 9, 2025"

// Format current time
@timeNow.format("short")
// → "2:30 PM"

// ISO format
@now.toISO()
// → "2025-12-09T14:30:45Z"
```

### Edge Cases & Constraints
1. **Evaluation timing** — Each reference to `@now` creates a new "current time" snapshot; multiple references in same expression may have microsecond differences
2. **Timezone** — Uses system timezone (same as existing `now()`); timezone conversion via methods
3. **Precision** — Second-level precision (matches existing datetime infrastructure)
4. **Kind field** — Must be set correctly for type system and method dispatch
5. **Backward compatibility** — `now()` continues to work but logs deprecation warning
6. **No arguments** — Literals cannot take parameters; use methods for timezone/format customization

## Implementation Notes
*To be added during implementation*

### Migration Guide
```parsley
// Old syntax (deprecated)
let current = now()

// New syntax
let current = @now       // Full datetime
let date = @today        // Date only
let time = @timeNow      // Time only
```

## Related
- Plan: `work/plans/FEAT-054-plan.md` (to be created)
- Related: Datetime literal parsing in `lexer.go` and `parser.go`
- Context: Part of namespace cleanup and builtin reorganization
- Deprecates: `now()` builtin function
