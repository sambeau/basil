---
id: PLAN-065
feature: FEAT-090
title: "Implementation Plan for Universal Builtin Interface"
status: draft
created: 2026-01-14
---

# Implementation Plan: FEAT-090 Universal Builtin Interface

## Overview
Standardize method interface across all Parsley builtin types. This involves cleanup (removing dead code), adding universal methods (`repr()`, `toJSON()`, `toBox()`), standardizing pseudo-type methods (`toDict()`, `inspect()`), adding collection rendering methods, and filling constructor gaps (`path()`).

**Philosophy**: Break, don't deprecate. Remove immediately and fix all tests, docs, and examples.

## Prerequisites
- [ ] FEAT-089 merged (toBox Phase 2) — provides foundation for toBox options
- [ ] Audit complete: know exactly what methods exist vs declared

## Phases

---

### Phase 1: Cleanup — Remove Dead Code
**Estimated effort**: Small (1-2 hours)
**Risk**: Low (removing unused code)

#### 1.1 Remove Integer `abs` declaration
**Files**: `pkg/parsley/evaluator/methods.go`

Steps:
1. Find Integer method table
2. Remove `abs` from declared methods (if present)
3. Verify no implementation exists (grep for Integer abs method body)

#### 1.2 Remove Float math method declarations
**Files**: `pkg/parsley/evaluator/methods.go`

Steps:
1. Find Float method table
2. Remove `abs`, `round`, `floor`, `ceil` declarations
3. Verify no implementations exist

#### 1.3 Remove `toDebug` builtin (if exists)
**Files**: `pkg/parsley/evaluator/evaluator.go`

Steps:
1. Search for `toDebug` in builtins
2. Remove if found
3. Update any tests that use it

#### 1.4 Update tests
**Files**: `pkg/parsley/tests/methods_test.go` (or similar)

Steps:
1. Remove any tests for Integer.abs()
2. Remove any tests for Float.abs(), .round(), .floor(), .ceil()
3. Remove any tests for toDebug()
4. Run `make test` to verify

#### 1.5 Update documentation
**Files**: 
- `docs/parsley/reference.md`
- `docs/parsley/manual/builtins/integer.md` (if exists)
- `docs/parsley/manual/builtins/float.md` (if exists)

Steps:
1. Remove abs from Integer section
2. Remove abs, round, floor, ceil from Float section
3. Add note pointing to `@std/math` for math operations
4. Remove toDebug from builtins list

#### 1.6 Verify cleanup complete
```bash
grep -r "\.abs()" docs/ examples/
grep -r "\.round()" docs/ examples/
grep -r "\.floor()" docs/ examples/
grep -r "\.ceil()" docs/ examples/
grep -r "toDebug" docs/ examples/ pkg/
```

**Commit**: `refactor(parsley): remove dead method declarations (abs, round, floor, ceil, toDebug)`

---

### Phase 2: Implement `repr()` Method
**Estimated effort**: Medium (3-4 hours)
**Risk**: Low (additive)

#### 2.1 Create objectToReprString function
**Files**: `pkg/parsley/evaluator/eval_string_conversions.go`

Steps:
1. Add `objectToReprString(obj Object) string` function
2. Handle each type to return Parsley-parseable literal:
   - Null → `"null"`
   - Boolean → `"true"` / `"false"`
   - Integer → `"42"`
   - Float → `"3.14"`
   - String → `"\"hello\"` (with escaping)
   - Array → `"[1, 2, 3]"` (recursive)
   - Dictionary → `"{a: 1, b: 2}"` (recursive)
   - DateTime → `"@2024-01-15T10:30:00Z"`
   - Duration → `"@1d2h30m"`
   - Money → `"$50.00"` or `"£50.00"`
   - Path → `"@./path/to/file"`
   - URL → `"@https://example.com"`
   - Regex → `"/pattern/flags"`
   - Function → `"<function name>"` (non-parseable marker)
   - Table → `"<table rows=N cols=M>"` (non-parseable marker)
3. Handle cycles with seen map, return `"<circular>"`

#### 2.2 Add repr() to all type method tables
**Files**: `pkg/parsley/evaluator/methods.go`

Steps:
1. Add "repr" to each type's method list
2. Implement method dispatch to call objectToReprString

#### 2.3 Create comprehensive tests
**Files**: `pkg/parsley/tests/repr_test.go` (new)

Tests:
- Primitives: null, true, false, integers, floats
- Strings: simple, with escapes, with quotes, with newlines
- Collections: arrays (nested), dictionaries (nested)
- Pseudo-types: datetime, duration, money, path, url, regex
- Edge cases: empty array, empty dict, circular reference, functions

#### 2.4 Update documentation
**Files**:
- `docs/parsley/reference.md` — Add repr() to each type section
- `docs/parsley/CHEATSHEET.md` — Add repr() to string conversion table

**Commit**: `feat(parsley): add repr() method for Parsley-parseable literal output`

---

### Phase 3: Fill `toJSON()` Gaps
**Estimated effort**: Small-Medium (2-3 hours)
**Risk**: Low (additive)

#### 3.1 Audit current toJSON coverage
First, determine which types already have toJSON:
```bash
grep -r "toJSON" pkg/parsley/evaluator/methods*.go
```

Expected: Array, Dictionary, Table have it. Likely missing: DateTime, Duration, Money, Path, URL, Regex.

#### 3.2 Implement toJSON for missing types
**Files**: Per-type method files or centralized

For each missing type, toJSON should return valid JSON:
- DateTime → `"\"2024-01-15T10:30:00Z\""` (ISO string in quotes)
- Duration → `"{\"days\":1,\"hours\":2,...}"` (object)
- Money → `"{\"amount\":50.00,\"currency\":\"USD\"}"` (object)
- Path → `"\"./path/to/file\""` (string in quotes)
- URL → `"\"https://example.com\""` (string in quotes)
- Regex → `"{\"pattern\":\"...\",\"flags\":\"...\"}"` (object)

#### 3.3 Add tests
**Files**: `pkg/parsley/tests/json_test.go` or per-type files

Tests:
- Each pseudo-type's toJSON output
- Verify output is valid JSON (parseable)
- Nested structures

#### 3.4 Update documentation
**Files**: Per-type manual pages

**Commit**: `feat(parsley): add toJSON() to DateTime, Duration, Money, Path, URL, Regex`

---

### Phase 4: Add `toBox()` to Pseudo-types
**Estimated effort**: Medium (2-3 hours)
**Risk**: Low (additive, leverages FEAT-089 infrastructure)

#### 4.1 Implement toBox for pseudo-types
**Files**: `pkg/parsley/evaluator/eval_box.go`

Add handlers for:
- DateTime → Single-value box with formatted date
- Duration → Single-value box with formatted duration
- Money → Single-value box with formatted amount
- Path → Single-value box or property table (absolute, segments, etc.)
- URL → Property table (scheme, host, path, query, etc.)
- Regex → Property table (pattern, flags)

Use existing toBox infrastructure from FEAT-089.

#### 4.2 Add to method tables
**Files**: `pkg/parsley/evaluator/methods.go`

Add "toBox" to DateTime, Duration, Money, Path, URL, Regex method lists.

#### 4.3 Add tests
**Files**: `pkg/parsley/tests/tobox_test.go`

Tests:
- Each type produces valid box output
- Options (style, title) work

#### 4.4 Update documentation
**Files**: Per-type manual pages

**Commit**: `feat(parsley): add toBox() to DateTime, Duration, Money, Path, URL, Regex`

---

### Phase 5: Standardize `toDict()` and `inspect()`
**Estimated effort**: Medium (3-4 hours)
**Risk**: Medium (may change existing behavior)

#### 5.1 Define contracts

**toDict()**: Returns clean dictionary for reconstruction
- No `__type` marker
- Keys match constructor parameter names
- Value can be passed back to constructor

**inspect()**: Returns debug dictionary with internals
- Includes `__type` marker
- May include computed/derived properties
- For debugging, not reconstruction

#### 5.2 Audit current implementations
```bash
grep -rn "toDict\|inspect" pkg/parsley/evaluator/
```

Determine current behavior per type.

#### 5.3 Implement/fix per type
**Files**: Per-type evaluator files

| Type | toDict() | inspect() |
|------|----------|-----------|
| DateTime | `{year, month, day, hour, minute, second, timezone}` | `{__type: "datetime", ...same..., unix}` |
| Duration | `{days, hours, minutes, seconds, milliseconds}` | `{__type: "duration", ...same..., totalMs}` |
| Money | `{amount, currency}` | `{__type: "money", ...same..., formatted}` |
| Path | `{path}` | `{__type: "path", absolute, segments, extension, filename, parent}` |
| URL | `{url}` | `{__type: "url", scheme, host, port, path, query, fragment}` |
| Regex | `{pattern, flags}` | `{__type: "regex", pattern, flags}` |

#### 5.4 Update tests
**Files**: Per-type test files

Tests:
- toDict() output can reconstruct via constructor: `type(value.toDict()) == type(value)`
- inspect() includes __type
- Round-trip: `time(someTime.toDict()).format() == someTime.format()`

#### 5.5 Update documentation
**Files**: Per-type manual pages, reference.md

Document the difference between toDict() (data) and inspect() (debug).

**Commit**: `feat(parsley): standardize toDict() and inspect() for pseudo-types`

---

### Phase 6: Add `toHTML()` and `toMarkdown()` to Collections
**Estimated effort**: Medium (3-4 hours)
**Risk**: Low (additive)

#### 6.1 Implement Array.toHTML(opts?)
**Files**: `pkg/parsley/evaluator/methods_array.go` (or similar)

```parsley
[1, 2, 3].toHTML()           // <ul><li>1</li><li>2</li><li>3</li></ul>
[1, 2, 3].toHTML({ordered: true})  // <ol><li>1</li>...</ol>
```

#### 6.2 Implement Array.toMarkdown(opts?)
```parsley
[1, 2, 3].toMarkdown()       // - 1\n- 2\n- 3
[1, 2, 3].toMarkdown({ordered: true})  // 1. 1\n2. 2\n3. 3
```

#### 6.3 Implement Dictionary.toHTML(opts?)
```parsley
{a: 1, b: 2}.toHTML()  // <dl><dt>a</dt><dd>1</dd>...</dl>
// Or: <table><tr><th>Key</th><th>Value</th></tr>...</table>
```

#### 6.4 Implement Dictionary.toMarkdown(opts?)
```parsley
{a: 1, b: 2}.toMarkdown()  // | Key | Value |\n|-----|-------|\n| a | 1 |...
```

#### 6.5 Verify Table already has these
Check Table has toHTML() and toMarkdown() from @std/table.

#### 6.6 Add tests
**Files**: `pkg/parsley/tests/array_test.go`, `pkg/parsley/tests/dictionary_test.go`

#### 6.7 Update documentation
**Files**: `docs/parsley/manual/builtins/array.md`, `docs/parsley/manual/builtins/dictionary.md`

**Commit**: `feat(parsley): add toHTML() and toMarkdown() to Array and Dictionary`

---

### Phase 7: Add `path()` Constructor
**Estimated effort**: Small (1-2 hours)
**Risk**: Low (additive)
**Resolves**: Backlog #59

#### 7.1 Implement path() builtin
**Files**: `pkg/parsley/evaluator/evaluator.go`

```parsley
let p = path("./some/path")
let p2 = path("/absolute/path")
let p3 = path(someVariable)
```

Steps:
1. Add to builtins map
2. Accept string argument
3. Create Path object
4. Respect sandbox restrictions (no escaping allowed paths)

#### 7.2 Add tests
**Files**: `pkg/parsley/tests/path_test.go`

Tests:
- Create path from string literal
- Create path from variable
- Relative paths
- Absolute paths
- Invalid paths (empty, null)
- Sandbox restrictions (if applicable)

#### 7.3 Update documentation
**Files**:
- `docs/parsley/reference.md` — Add path() to constructors section
- `docs/parsley/manual/builtins/path.md` — Update with constructor

#### 7.4 Update backlog
Mark #59 as completed with reference to FEAT-090.

**Commit**: `feat(parsley): add path() constructor for dynamic path creation`

---

## Validation Checklist

After all phases:
- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Full check: `make check`

### Code Verification
- [ ] `grep -r "abs.*Integer\|Integer.*abs" pkg/` returns no method declarations
- [ ] `grep -r "round.*Float\|Float.*round" pkg/` returns no method declarations
- [ ] `grep -r "toDebug" pkg/` returns no builtin definitions
- [ ] `grep -r "\.repr()" pkg/parsley/tests/` shows tests for all types

### Documentation Verification
- [ ] `grep -r "\.abs()" docs/` returns no hits
- [ ] `grep -r "toDebug" docs/` returns no hits
- [ ] `docs/parsley/reference.md` includes repr() for all types
- [ ] `docs/parsley/reference.md` includes path() constructor
- [ ] Each pseudo-type manual page documents toDict() and inspect()

### Example Verification
- [ ] `grep -r "\.abs()\|\.round()\|\.floor()\|\.ceil()" examples/` returns no hits
- [ ] `grep -r "toDebug" examples/` returns no hits
- [ ] Run representative examples to verify they work

## Progress Log
| Date | Phase | Status | Notes |
|------|-------|--------|-------|
| | Phase 1: Cleanup | ⬜ Not started | |
| | Phase 2: repr() | ⬜ Not started | |
| | Phase 3: toJSON() | ⬜ Not started | |
| | Phase 4: toBox() | ⬜ Not started | |
| | Phase 5: toDict/inspect | ⬜ Not started | |
| | Phase 6: toHTML/toMarkdown | ⬜ Not started | |
| | Phase 7: path() | ⬜ Not started | |

## Estimated Total Effort
| Phase | Estimate |
|-------|----------|
| Phase 1: Cleanup | 1-2 hours |
| Phase 2: repr() | 3-4 hours |
| Phase 3: toJSON() | 2-3 hours |
| Phase 4: toBox() | 2-3 hours |
| Phase 5: toDict/inspect | 3-4 hours |
| Phase 6: toHTML/toMarkdown | 3-4 hours |
| Phase 7: path() | 1-2 hours |
| **Total** | **15-22 hours** |

## Deferred Items
Items to add to work/BACKLOG.md after implementation:
- `toBool()` builtin — Consider for completeness but not blocking
- Advanced repr() options (indentation, max depth) — If needed
