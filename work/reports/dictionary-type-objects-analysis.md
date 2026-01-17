# Dictionary-Type Objects Analysis

**Date:** 2026-01-16  
**Status:** Analysis / Design Discussion  
**Triggered by:** Bug where `{Person} = @~/schema/birthdays.pars` silently bound `Person = null`

## Background

While debugging a schema import issue, we discovered that path literals (`@~/path`) evaluate to dictionaries with a `__type` marker. This allows destructuring syntax to "work" but silently produce `null` for non-existent keys:

```parsley
{Person} = @~/schema/birthdays.pars  // No error! Person = null
let person = data.as(Person)          // ERROR: as expected a schema, got null
```

The user intended `{Person} = import @~/schema/birthdays.pars` but forgot the `import` keyword.

## Current State

### Objects Using `__type` Dictionary Pattern

| Type | Example | `__type` value |
|------|---------|----------------|
| Datetime | `@now` | `"datetime"` |
| Duration | `5.minutes` | `"duration"` |
| Path | `@./file.pars`, `@~/path` | `"path"` |
| URL | `@url"https://..."` | `"url"` |
| Regex | `/pattern/` | `"regex"` |
| File handle | `file(...)` | `"file"` |
| Directory handle | `dir(...)` | `"dir"` |
| Tag | HTML elements | `"tag"` |
| HTTP Request | `@request` | `"request"` |
| HTTP Response | `fetch()` result | `"response"` |
| Part module | `.part` imports | `"part"` |
| Money | currency values | `"money"` |

### Objects Already Using Real Types

- `DSLSchema` - schema definitions
- `DSLRecord` - validated records
- `DBConnection` - database connections
- `Table` - tabular data
- `Function` - closures
- `Array` - lists
- `Error` - error objects
- `String`, `Integer`, `Float`, `Boolean` - primitives

## Problem Analysis

### Issue 1: Silent Null on Missing Destructure Keys

Destructuring a non-existent key returns `null` without error:

```parsley
{foo} = {bar: 1}  // foo = null, no error
```

This is arguably valid for general dictionaries (lenient destructuring), but becomes a footgun with `__type` dictionaries where the structure is internal.

### Issue 2: Path Literals Are Destructurable

Path literals expose internal structure (`__type`, `absolute`, `segments`) that users shouldn't interact with:

```parsley
{__type, absolute, segments} = @./foo.pars  // Works but shouldn't be used
```

This creates confusion when users write `{X} = @path` expecting import-like behavior.

## Destructuring Usefulness by Type

| Type | Destructure Useful? | Rationale |
|------|---------------------|-----------|
| `datetime` | ✅ Yes | `{year, month, day} = @now` is convenient |
| `duration` | ⚠️ Maybe | `{hours, minutes} = dur` occasionally useful |
| `url` | ⚠️ Maybe | `{host, path, query} = url` occasionally useful |
| `path` | ❌ No | Internal structure, not user-facing |
| `regex` | ❌ No | Internal structure, not user-facing |
| `file` | ❌ No | Opaque handle |
| `dir` | ❌ No | Opaque handle |
| `tag` | ❌ No | Internal structure |
| `request` | ⚠️ Maybe | Headers/body access, but methods better |
| `response` | ⚠️ Maybe | Headers/body access, but methods better |

## Options

### Option A: Error on Destructuring Path Literals (Quick Fix)

Add a check in dict destructuring: if RHS has `__type: "path"`, emit error with hint.

**Error message:**
```
Cannot destructure path literal
Hint: Did you mean `import @~/schema/birthdays.pars`?
```

**Pros:**
- Catches the exact bug that triggered this investigation
- Minimal code change
- Non-breaking for other uses

**Cons:**
- Doesn't address the broader pattern

### Option B: Error on Destructuring All Internal Types

Extend Option A to all internal `__type` values: `path`, `regex`, `file`, `dir`, `tag`.

Allow destructuring for user-facing types: `datetime`, `duration`, `url`.

**Pros:**
- Prevents misuse of internal structures
- Preserves useful patterns like `{year, month} = @now`

**Cons:**
- Requires maintaining a list of "internal" vs "user-facing" types

### Option C: Convert Internal Types to Real Go Types

Make `path`, `file`, `dir`, `regex`, `tag` proper Go struct types instead of dictionaries.

**Advantages:**
1. **Type safety** - `typeof(@./foo)` → `"PATH"` not `"DICTIONARY"`
2. **Performance** - No AST expression wrapping overhead
3. **Cleaner inspection** - `@./foo.Inspect()` → `"@./foo"` not `{__type: path, ...}`
4. **Immutability** - Can't accidentally mutate internal fields
5. **Memory** - Smaller footprint
6. **No `__type` leakage** - Users can't access internal markers
7. **Method dispatch** - Faster, no runtime type checking

**Disadvantages:**
1. More Go code per type
2. Need explicit serialization methods
3. Larger refactoring effort

### Option D: Convert ALL Dictionary Types to Real Types

Extend Option C to include `datetime`, `duration`, `url`, etc.

**Additional work:**
- Add `.year`, `.month`, etc. as methods on Datetime type
- Add `.host`, `.path`, etc. as methods on URL type
- Add `toDict()` method for when dictionary form is needed

**Breaking changes:**
- `{year, month} = @now` would need to become `@now.year`, `@now.month`
- Or add special destructuring support for these types

## Recommendation

### Short Term (Bug Fix)
Implement **Option A**: Error when destructuring path literals with helpful hint.

### Medium Term
Implement **Option B**: Extend error to all internal `__type` dictionaries.

### Long Term (Consider)
Evaluate **Option C**: Convert internal types to real Go types during a planned refactoring phase. This would be a good fit for a "Type System Hardening" milestone.

Keep `datetime`, `duration`, `url` as dictionaries (or add special destructuring support if converted) since the destructuring syntax is genuinely useful for these.

## Implementation Notes

### For Option A/B

Location: `pkg/parsley/evaluator/evaluator.go` or wherever dict destructuring is evaluated.

```go
// In evalDictDestructuring or similar
if isPathDict(rightSide) {
    return &Error{
        Message: "cannot destructure path literal",
        Hints:   []string{"Did you mean: import " + pathDictToString(rightSide) + "?"},
    }
}
```

### Error Code

Suggest: `DESTRUCT-0001` - "cannot destructure {type} value"

## Related Issues

- Silent null on missing destructure keys (separate issue, more general)
- Module caching and `__type: "part"` handling
- JSON serialization of `__type` dictionaries

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-01-16 | Analysis created | Triggered by user bug report |
| | | |
