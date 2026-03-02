---
id: PLAN-117
feature: FEAT-137
title: "Implementation Plan for Complete Method Registry Migration"
status: draft
created: 2026-03-02
---

# Implementation Plan: FEAT-137 Complete Method Registry Migration

## Overview

Migrate all remaining ~20 types (~263 methods) from hand-coded switch dispatch to the declarative `MethodRegistry` system introduced in FEAT-111. Work is split into three tiers by priority and dependency. Each tier is independently shippable.

**Already migrated (FEAT-111):** string (38), integer, float (29 combined), money (35), unit (46) — 5 types across 4 files totalling ~3,400 lines.

**Pattern reference:** `methods_string.go` is the canonical example of a fully migrated type.

## Prerequisites

- [x] FEAT-111 infrastructure exists (`method_registry.go`: `MethodRegistry`, `MethodEntry`, `MethodFunc`, `dispatchFromRegistry`, `checkArity`, `RegisterMethodRegistry`, `GetMethodsForType`)
- [x] 5 types already migrated — pattern is proven and stable
- [ ] Clean working tree (`git status`)
- [ ] All tests pass before starting (`go test ./pkg/parsley/...`)

## Migration Pattern (All Tasks Follow This)

Every type migration follows the same mechanical steps:

1. **Create file** `methods_<type>.go` (or add to existing file for small types)
2. **Define registry** variable and `init()` with `RegisterMethodRegistry()`
3. **Extract each switch case** into a named `MethodFunc`:
   - Signature: `func <type><Method>(receiver Object, args []Object, env *Environment) Object`
   - First line: type-assert receiver (`arr := receiver.(*Array)`)
   - Remove manual arity check (registry handles it)
   - Body otherwise unchanged
4. **Replace eval function** to use `dispatchFromRegistry` + `unknownMethodError`
5. **Remove stale data**: `TypeMethods` entry in `introspect.go`, any `xxxMethods` string slice
6. **Run tests**: `go test ./pkg/parsley/...`
7. **Verify introspection**: `pars describe <type>` shows methods from registry

### Example Transform

**Before** (switch case in `evalArrayMethod`):
```go
case "length":
    if len(args) != 0 {
        return newArityError("length", len(args), 0)
    }
    return &Integer{Value: int64(len(arr.Elements))}
```

**After** (registry entry + named function):
```go
// In registry:
"length": {
    Fn:          arrayLength,
    Arity:       "0",
    Description: "Return the number of elements",
},

// Named function:
func arrayLength(receiver Object, args []Object, env *Environment) Object {
    arr := receiver.(*Array)
    return &Integer{Value: int64(len(arr.Elements))}
}
```

**After** (dispatch function):
```go
func evalArrayMethod(arr *Array, method string, args []Object, env *Environment) Object {
    result := dispatchFromRegistry(ArrayMethodRegistry, "array", arr, method, args, env)
    if result != nil {
        return result
    }
    return unknownMethodError(method, "array", ArrayMethodRegistry.Names())
}
```

---

## Tier 1: Core Types

**Branch:** `feat/FEAT-137-tier1-core-methods`

These are the highest-value migrations — `array` and `dictionary` are the most commonly used and most commonly queried with `pars describe`.

---

### Task 1: Migrate `array` methods

**Files:** New `methods_array.go`, modify `methods.go`, modify `introspect.go`
**Estimated effort:** Medium-Large

The `array` type has 23 methods in a 388-line switch block (`methods.go` L350-738). Several methods delegate to helper functions that stay in place.

**Registry entries to create:**

| Method | Arity | Current helpers (stay in place) |
|--------|-------|---------------------------------|
| `length` | `0` | — |
| `reverse` | `0` | — |
| `sort` | `0-1` | `sortArrayWithOptions` |
| `sortBy` | `1` | `sortArrayByFunction` |
| `map` | `1` | `mapArrayWithFunction` |
| `filter` | `1` | `filterArrayWithFunction` |
| `reduce` | `2` | — (inline logic) |
| `format` | `0-2` | `locale.FormatList` |
| `join` | `0-1` | — |
| `toJSON` | `0` | — |
| `toCSV` | `0-1` | `encodeCSV` |
| `shuffle` | `0` | — |
| `pick` | `0-1` | — |
| `take` | `1` | — |
| `insert` | `2` | — |
| `has` | `1` | — |
| `hasAny` | `1` | — |
| `hasAll` | `1` | — |
| `toBox` | `0+` | `arrayToBox` |
| `repr` | `0` | `objectToReprString` |
| `toHTML` | `0+` | `arrayToHTML` |
| `toMarkdown` | `0+` | `arrayToMarkdown` |
| `reorder` | `1+` | `arrayReorder` |

Steps:
1. Create `methods_array.go` with `ArrayMethodRegistry` and `init()`
2. Extract each of the 23 switch cases into named functions (e.g. `arrayLength`, `arrayReverse`, `arraySort`, etc.)
3. For methods that delegate to existing helpers (`sortArrayWithOptions`, etc.), the named function becomes a thin wrapper that type-asserts and calls the helper
4. Replace `evalArrayMethod` body with `dispatchFromRegistry` pattern
5. Remove `var arrayMethods = []string{...}` from `methods.go` (L41-44)
6. Remove `"array"` entry from `TypeMethods` in `introspect.go` (L132-148)
7. Move helper functions (`sortArrayWithOptions`, `sortArrayByFunction`, `mapArrayWithFunction`, `filterArrayWithFunction`, `arrayReorder`, `arrayToBox`, `arrayToHTML`, `arrayToMarkdown`) to `methods_array.go` since they're only used by array methods

**Note on `fmt`/`format` alias:** The switch has `case "fmt", "format":`. In the registry, register both keys pointing to the same function.

Tests:
- All existing array tests pass (`go test ./pkg/parsley/... -run Array`)
- `pars describe array` shows all 23 methods from registry
- Arity errors use registry-generated messages (spot-check a few)

---

### Task 2: Migrate `dictionary` methods

**Files:** New `methods_dictionary.go`, modify `methods.go`, modify `introspect.go`
**Estimated effort:** Medium

The `dictionary` type has 15 methods in a 212-line switch block (`methods.go` L1124-1336). The dictionary dispatch has two special considerations:

1. **Schema-dict check**: `evalDictionaryMethod` starts with `if IsSchemaDict(dict)` which tries schema methods first. This stays as-is — the registry wraps the switch, and the schema check happens before registry dispatch.
2. **`nil` return for unknown methods**: The current `default` case returns `nil` (not an error) to allow user-defined function lookup in `dispatchMethodCall`. After migration, `dispatchFromRegistry` already returns `nil` for unknown methods, so this behaviour is preserved naturally.

**Registry entries to create:**

| Method | Arity | Notes |
|--------|-------|-------|
| `keys` | `0` | |
| `values` | `0` | |
| `entries` | `0` or `2` | Arity spec: `"0"` won't work — need custom handling or use arity `"0+"` and validate internally. Use `"0+"` with manual check for 0 or 2 args inside the function, matching the existing `len(args) != 0 && len(args) != 2` check. |
| `has` | `1` | |
| `delete` | `1` | |
| `insertAfter` | `3` | |
| `insertBefore` | `3` | |
| `render` | `1` | |
| `toJSON` | `0` | |
| `toBox` | `0+` | `dictToBox` |
| `repr` | `0` | `objectToReprString` |
| `toHTML` | `0+` | `dictToHTML` |
| `toMarkdown` | `0+` | `dictToMarkdown` |
| `as` | `1` | |
| `reorder` | `1+` | `dictReorder` |

Steps:
1. Create `methods_dictionary.go` with `DictionaryMethodRegistry` and `init()`
2. Extract 15 switch cases into named functions
3. Replace the switch block in `evalDictionaryMethod` with registry dispatch, preserving:
   - The `IsSchemaDict` check at the top (before registry dispatch)
   - The `nil` return for unknown methods (so `dispatchMethodCall` can try user-defined functions)
4. The eval function becomes:
   ```go
   func evalDictionaryMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
       if IsSchemaDict(dict) {
           result := evalSchemaMethod(dict, method, args, env)
           if result != nil {
               return result
           }
       }
       result := dispatchFromRegistry(DictionaryMethodRegistry, "dictionary", dict, method, args, env)
       if result != nil {
           return result
       }
       // Return nil for unknown methods — dispatchMethodCall checks user-defined functions
       return nil
   }
   ```
5. Remove `var dictionaryMethods = []string{...}` from `methods.go` (L48-52)
6. Remove `"dictionary"` entry from `TypeMethods` in `introspect.go` (L149-158)
7. Move helper functions (`insertDictKeyAfter`, `insertDictKeyBefore`, `dictReorder`, `dictToBox`, `dictToHTML`, `dictToMarkdown`) to `methods_dictionary.go`

**Note on `entries` arity:** The `entries()` method accepts 0 or exactly 2 args. The arity spec system doesn't natively support "0 or 2" (only ranges). Two options:
- Use arity `"0+"` and do the 0-or-2 validation inside the function body (recommended — keeps the existing error message intact)
- Extend `checkArity` to support `"0,2"` syntax (over-engineering for one method)

Tests:
- All existing dictionary tests pass
- `pars describe dictionary` shows all 15 methods from registry
- Dictionary with user-defined function methods still works (the `nil` return path)
- Schema-dict methods still dispatch correctly

---

### Task 3: Tier 1 validation and commit

**Estimated effort:** Small

Steps:
1. Run full test suite: `go test ./pkg/parsley/...`
2. Run linter: `golangci-lint run`
3. Verify introspection: `pars describe array`, `pars describe dictionary`
4. Verify error messages: test an unknown method on array and dictionary — should get fuzzy-matched suggestions from registry
5. Commit: `feat(parsley): migrate array and dictionary to method registry (FEAT-137 Tier 1)`

---

## Tier 2: Dictionary-Like Types and Simple Types

**Branch:** `feat/FEAT-137-tier2-dict-subtypes`

These types all follow the same pattern. The dict-subtype types (datetime, duration, path, url, regex, file, dir) are `*Dictionary` values with a `__type` marker. The simple types (boolean, null, request, response) are trivial migrations.

**Key constraint:** The dict-subtype dispatch chain in `dispatchMethodCall` (`eval_method_dispatch.go` L253-370) must be updated to use registry dispatch while preserving the fallthrough-to-dictionary-methods semantics.

---

### Task 4: Migrate `boolean` and `null` methods

**Files:** New `methods_boolean.go` (or add to small `methods_simple.go`), modify `methods.go`, modify `introspect.go`
**Estimated effort:** Small

These are the simplest migrations — 4 methods each, no special dispatch logic.

**Boolean methods:** `toBox` (0), `repr` (0), `toJSON` (0), `inspect` (0)
**Null methods:** `toBox` (0), `repr` (0), `toJSON` (0), `inspect` (0)

Steps:
1. Create `methods_boolean.go` with `BooleanMethodRegistry` and `NullMethodRegistry`
2. Extract 4 + 4 switch cases into named functions
3. Replace `evalBooleanMethod` and `evalNullMethod` with registry dispatch
4. Remove `var booleanMethods` and `var nullMethods` string slices from `methods.go`
5. Remove/update `"boolean"` and `"null"` entries in `TypeMethods`
6. Run tests

Tests:
- Existing boolean/null tests pass
- `pars describe boolean` and `pars describe null` show methods

---

### Task 5: Migrate `datetime` methods

**Files:** New `methods_datetime.go`, modify `methods.go`, modify `introspect.go`
**Estimated effort:** Medium

13 methods in `evalDatetimeMethod` (`methods.go` L1579-1768). The datetime type is a `*Dictionary` with `__type: "datetime"`.

**Registry entries:** `format` (0-2), `add` (1), `subtract` (1), `isBefore` (1), `isAfter` (1), `isEqual` (1), `dayOfYear` (0), `week` (0), `timestamp` (0), `toDict` (0), `toBox` (0+), `repr` (0), `toJSON` (0)

Steps:
1. Create `methods_datetime.go` with `DatetimeMethodRegistry` and `init()`
2. Extract 13 switch cases into named functions
3. Replace `evalDatetimeMethod` with registry dispatch
4. Update the `isDatetimeDict` branch in `dispatchMethodCall` to:
   ```go
   if isDatetimeDict(receiver) {
       result := dispatchFromRegistry(DatetimeMethodRegistry, "datetime", receiver, method, args, env)
       if result != nil {
           return result
       }
       // Fall through to dictionary methods
       result = dispatchFromRegistry(DictionaryMethodRegistry, "dictionary", receiver, method, args, env)
       if result != nil {
           return result
       }
       return unknownMethodError(method, "datetime", DatetimeMethodRegistry.Names())
   }
   ```
5. Remove `"datetime"` entry from `TypeMethods`
6. Move `getDictStringValue` helper to `methods_datetime.go` if only used there
7. Run tests

Tests:
- Existing datetime tests pass
- `pars describe datetime` shows all 13 methods
- Calling dictionary methods on a datetime value still works (e.g. `.keys()`, `.toJSON()`)

---

### Task 6: Migrate `duration` methods

**Files:** New `methods_duration.go`, modify `methods.go`, modify `introspect.go`
**Estimated effort:** Medium

10 methods in `evalDurationMethod` (`methods.go` L1786-1958) plus format helpers (`formatDurationWithStyle`, `formatDurationShort`, `formatDurationLong`, `formatDurationRepr` — L1961-2158).

**Registry entries:** `format` (0-1), `add` (1), `subtract` (1), `multiply` (1), `abs` (0), `isNegative` (0), `toDict` (0), `toBox` (0+), `repr` (0), `toJSON` (0)

Steps:
1. Create `methods_duration.go` with `DurationMethodRegistry`
2. Extract 10 switch cases into named functions
3. Move format helpers (`formatDurationWithStyle`, `formatDurationShort`, `formatDurationLong`, `formatDurationRepr`) to `methods_duration.go`
4. Replace `evalDurationMethod` with registry dispatch
5. Update the `isDurationDict` branch in `dispatchMethodCall` (same pattern as datetime — fallthrough to dictionary methods)
6. Remove `"duration"` entry from `TypeMethods`
7. Run tests

Tests:
- Existing duration tests pass
- `pars describe duration` shows all 10 methods
- Dictionary methods accessible on duration values

---

### Task 7: Migrate `path` methods

**Files:** New `methods_path.go`, modify `methods.go`, modify `introspect.go`
**Estimated effort:** Medium

10 methods in `evalPathMethod` (`methods.go` L2165-2298).

**Registry entries:** `toString` (0), `join` (1+), `parent` (0), `isAbsolute` (0), `isRelative` (0), `public` (0), `toURL` (1), `match` (1), `toDict` (0), `repr` (0)

Steps:
1. Create `methods_path.go` with `PathMethodRegistry`
2. Extract 10 switch cases into named functions
3. Replace `evalPathMethod` with registry dispatch
4. Update `isPathDict` branch in `dispatchMethodCall` (fallthrough to dictionary)
5. Remove `"path"` entry from `TypeMethods`
6. Run tests

Tests:
- Existing path tests pass
- `pars describe path` shows all methods
- `.keys()` etc. still work on path values

---

### Task 8: Migrate `url` methods

**Files:** New `methods_url.go`, modify `methods.go`, modify `introspect.go`
**Estimated effort:** Small-Medium

9 methods in `evalUrlMethod` (`methods.go` L2305-2445).

**Registry entries:** `origin` (0), `pathname` (0), `toString` (0), `withPath` (1), `withQuery` (1), `toDict` (0), `toBox` (0+), `repr` (0), `toJSON` (0)

Steps:
1. Create `methods_url.go` with `UrlMethodRegistry`
2. Extract 9 switch cases into named functions
3. Replace `evalUrlMethod` with registry dispatch
4. Update `isUrlDict` branch in `dispatchMethodCall`
5. Remove `"url"` entry from `TypeMethods`
6. Run tests

---

### Task 9: Migrate `regex` methods

**Files:** New `methods_regex.go`, modify `methods.go`, modify `introspect.go`
**Estimated effort:** Small-Medium

7 methods in `evalRegexMethod` (`methods.go` L2452-2605).

**Registry entries:** `test` (1), `match` (1), `matchAll` (1), `replace` (2), `split` (1), `toDict` (0), `repr` (0)

Steps:
1. Create `methods_regex.go` with `RegexMethodRegistry`
2. Extract 7 switch cases into named functions
3. Replace `evalRegexMethod` with registry dispatch
4. Update `isRegexDict` branch in `dispatchMethodCall`
5. Remove `"regex"` entry from `TypeMethods`
6. Run tests

---

### Task 10: Migrate `file` and `dir` methods

**Files:** New `methods_file.go`, modify `methods.go`, modify `introspect.go`
**Estimated effort:** Small

5 methods for file (`methods.go` L2612-2731), 4 methods for dir (`methods.go` L2738-2850). These are small enough to share a file.

**File registry entries:** `exists` (0), `read` (0), `stat` (0), `toDict` (0), `repr` (0)
**Dir registry entries:** `exists` (0), `list` (0), `toDict` (0), `repr` (0)

Steps:
1. Create `methods_file.go` with `FileMethodRegistry` and `DirMethodRegistry`
2. Extract 5 + 4 switch cases
3. Replace `evalFileMethod` and `evalDirMethod` with registry dispatch
4. Update `isFileDict` and `isDirDict` branches in `dispatchMethodCall`
5. Remove `"file"` and `"directory"` entries from `TypeMethods`
6. Run tests

---

### Task 11: Migrate `request` and `response` methods

**Files:** New `methods_http.go`, modify `methods.go`, modify `introspect.go`
**Estimated effort:** Small

1 method for request (`methods.go` L2857-2869), 4 methods for response (`methods.go` L2876-2920). Trivially small.

**Request registry entries:** `toJSON` (0)
**Response registry entries:** `toJSON` (0), `toDict` (0), `toBox` (0+), `repr` (0)

Steps:
1. Create `methods_http.go` with `RequestMethodRegistry` and `ResponseMethodRegistry`
2. Extract 1 + 4 switch cases
3. Replace `evalRequestMethod` and `evalResponseMethod` with registry dispatch
4. Update `isRequestDict` and `isResponseDict` branches in `dispatchMethodCall`
5. Remove entries from `TypeMethods`
6. Run tests

---

### Task 12: Refactor dict-subtype dispatch in `dispatchMethodCall`

**Files:** `eval_method_dispatch.go`
**Estimated effort:** Small

After Tasks 5-11, the `*Dictionary` case in `dispatchMethodCall` has a chain of `isXxxDict` checks. Each one currently calls `evalXxxMethod` and does an `UNDEF-0002` error check for fallthrough. Now that all subtypes use registries, refactor these to use a consistent helper pattern.

Steps:
1. Create a helper function to reduce repetition:
   ```go
   func dispatchDictSubtype(registry MethodRegistry, typeName string, dict *Dictionary, method string, args []Object, env *Environment) Object {
       result := dispatchFromRegistry(registry, typeName, dict, method, args, env)
       if result != nil {
           return result
       }
       // Fall through to dictionary methods
       result = dispatchFromRegistry(DictionaryMethodRegistry, "dictionary", dict, method, args, env)
       if result != nil {
           return result
       }
       return unknownMethodError(method, typeName, registry.Names())
   }
   ```
2. Replace each `isXxxDict` branch body with a single `return dispatchDictSubtype(XxxMethodRegistry, "xxx", receiver, method, args, env)` call
3. Run tests — no behaviour change, just deduplication

**This task is optional** — it cleans up the dispatch chain but doesn't change functionality. Skip if the individual branches are clear enough without it.

---

### Task 13: Tier 2 validation and commit

**Estimated effort:** Small

Steps:
1. Run full test suite: `go test ./pkg/parsley/...`
2. Run linter: `golangci-lint run`
3. Verify introspection for all Tier 2 types: `pars describe datetime`, `pars describe duration`, `pars describe path`, etc.
4. Verify dict-subtype fallthrough: e.g. `pars -e '(now()).keys()'` — datetime value using dictionary method
5. Commit: `feat(parsley): migrate dict-subtypes and simple types to method registry (FEAT-137 Tier 2)`

---

## Tier 3: Server-Specific Types

**Branch:** `feat/FEAT-137-tier3-server-types`

These are Basil server types, not core Parsley types. They're lower priority for `pars describe` but completing them eliminates the old dispatch pattern entirely. `table` and `DSLSchema` are the largest; the rest are moderate to small.

---

### Task 14: Migrate `record` methods

**Files:** `methods_record.go`
**Estimated effort:** Medium

~24 methods in `evalRecordMethod` (`methods_record.go`, 749 lines). This file already exists so the registry and named functions go in the same file.

Steps:
1. Add `RecordMethodRegistry` and `init()` to `methods_record.go`
2. Extract switch cases into named functions
3. Replace `evalRecordMethod` with registry dispatch
4. Remove any `TypeMethods` entry if present (check — may not have one)
5. Run tests

---

### Task 15: Migrate `DBConnection` methods

**Files:** `eval_method_dispatch.go` or new `methods_dbconnection.go`
**Estimated effort:** Small-Medium

8 methods in `evalDBConnectionMethod` (`eval_method_dispatch.go` L12-197).

**Registry entries:** `begin` (0), `commit` (0), `rollback` (0), `close` (0), `ping` (0), `createTable` (1-2), `lastInsertId` (0), `bind` (2-3)

Steps:
1. Create `methods_dbconnection.go` with `DBConnectionMethodRegistry`
2. Extract 8 switch cases
3. Replace `evalDBConnectionMethod` with registry dispatch
4. Remove `"dbconnection"` entry from `TypeMethods`
5. Run tests (database tests)

**Note on `bind` arity:** Accepts 2 or 3 args. Use arity `"2-3"`.
**Note on `createTable` arity:** Accepts 1 or 2 args. Use arity `"1-2"`.

---

### Task 16: Migrate `SFTPConnection` and `SFTPFileHandle` methods

**Files:** `eval_network_io.go` or new `methods_sftp.go`
**Estimated effort:** Small

Both types are in `eval_network_io.go`. Small method sets.

Steps:
1. Create `methods_sftp.go` with `SFTPConnectionMethodRegistry` and `SFTPFileHandleMethodRegistry`
2. Extract switch cases into named functions
3. Replace eval functions with registry dispatch
4. Remove `"sftpconnection"` and `"sftpfile"` entries from `TypeMethods`
5. Run tests

---

### Task 17: Migrate `session` methods

**Files:** `stdlib_session.go` or new `methods_session.go`
**Estimated effort:** Small-Medium

~11 methods in `evalSessionMethod` (`stdlib_session.go`, 293 lines).

**Registry entries:** `get` (1-2), `set` (2), `delete` (1), `has` (1), `clear` (0), `all` (0), `flash` (2), `getFlash` (1), `getAllFlash` (0), `hasFlash` (0), `regenerate` (0)

Steps:
1. Create `methods_session.go` with `SessionMethodRegistry`
2. Extract 11 switch cases
3. Replace `evalSessionMethod` with registry dispatch
4. Remove `"session"` entry from `TypeMethods`
5. Run tests

---

### Task 18: Migrate `table` methods

**Files:** `stdlib_table.go` or new `methods_table.go`
**Estimated effort:** Large

~48 methods in `EvalTableMethod` (`stdlib_table.go`, 2,868 lines). This is the single largest migration. Many methods have complex implementations with helper functions.

Steps:
1. Create `methods_table.go` with `TableMethodRegistry`
2. Extract switch cases into named functions — this will be tedious but mechanical
3. Move helper functions that are only used by table methods into `methods_table.go`
4. Replace `EvalTableMethod` with registry dispatch
5. Note: `EvalTableMethod` is exported (capital E). After migration, the registry dispatch wrapper can remain exported for any external callers. Check for usages outside the package.
6. Remove `"table"` entry from `TypeMethods`
7. Run tests

**Note on exported function:** Grep for `EvalTableMethod` usage outside the evaluator package. If external callers exist, keep the function exported with the same signature.

---

### Task 19: Migrate `DSLSchema` methods

**Files:** `stdlib_dsl_schema.go` or new `methods_dsl_schema.go`
**Estimated effort:** Large

~54 methods in `evalDSLSchemaMethod` (`stdlib_dsl_schema.go`, 1,027 lines). Another large migration.

Steps:
1. Create `methods_dsl_schema.go` with `DSLSchemaMethodRegistry`
2. Extract switch cases into named functions
3. Replace `evalDSLSchemaMethod` with registry dispatch
4. Remove any `TypeMethods` entry
5. Run tests

---

### Task 20: Migrate remaining minor types

**Files:** Various
**Estimated effort:** Small

Check for any remaining types dispatched via switch that aren't covered above:
- `DevModule` — `evalDevModuleMethod` (check location and method count)
- `TableBinding` — `evalTableBindingMethod`
- `MdDoc` — `evalMdDocMethod`

Steps:
1. Audit `dispatchMethodCall` in `eval_method_dispatch.go` for any remaining type cases
2. Migrate any found types using the standard pattern
3. Remove corresponding `TypeMethods` entries (e.g. `"dev"`)
4. Run tests

---

### Task 21: Final cleanup — remove `TypeMethods` map

**Files:** `introspect.go`
**Estimated effort:** Small

Once all types are migrated, the `TypeMethods` map in `introspect.go` should be empty or nearly empty.

Steps:
1. Verify `TypeMethods` has no remaining entries (or only `"function"` which has no methods)
2. If empty, remove the `TypeMethods` variable entirely
3. Update `builtinInspect` (L635-639) to use `GetMethodsForType` instead of `TypeMethods`
4. Update `builtinDescribe` (L1058-1068) to remove the fallback branch
5. Grep for any other `TypeMethods` references and update
6. Run tests

---

### Task 22: Tier 3 validation, commit, and close

**Estimated effort:** Small

Steps:
1. Run full test suite: `go test ./pkg/parsley/...`
2. Run linter: `golangci-lint run`
3. Verify introspection for all Tier 3 types
4. Commit: `feat(parsley): migrate server types to method registry (FEAT-137 Tier 3)`
5. Update FEAT-111 status to `implemented`
6. Update FEAT-137 status to `implemented`

---

## Task Order & Commits

```
TIER 1 (branch: feat/FEAT-137-tier1-core-methods)
═══════
Task 1 (array) ──────────┐
                          ├──► Task 3 (validate & commit)
Task 2 (dictionary) ─────┘


TIER 2 (branch: feat/FEAT-137-tier2-dict-subtypes)
═══════
Task 4 (boolean, null) ──────────────────────────────────────┐
                                                              │
Task 5 (datetime) ───┐                                       │
Task 6 (duration) ───┤                                       │
Task 7 (path) ───────┤ can be done in any order              ├──► Task 13 (validate & commit)
Task 8 (url) ────────┤ or in parallel                        │
Task 9 (regex) ──────┤                                       │
Task 10 (file, dir) ─┤                                       │
Task 11 (req, resp) ─┘                                       │
                                                              │
Task 12 (refactor dispatch — optional) ──────────────────────┘


TIER 3 (branch: feat/FEAT-137-tier3-server-types)
═══════
Task 14 (record) ────────┐
Task 15 (DBConnection) ──┤
Task 16 (SFTP) ──────────┤ can be done in any order
Task 17 (session) ───────┤ or in parallel              ──► Task 21 (remove TypeMethods) ──► Task 22 (validate & commit)
Task 18 (table) ─────────┤
Task 19 (DSLSchema) ─────┤
Task 20 (remaining) ─────┘
```

**Suggested commit sequence:**

Tier 1:
1. `feat(parsley): migrate array methods to declarative registry`
2. `feat(parsley): migrate dictionary methods to declarative registry`

Tier 2:
3. `feat(parsley): migrate boolean and null methods to declarative registry`
4. `feat(parsley): migrate datetime and duration methods to declarative registry`
5. `feat(parsley): migrate path, url, and regex methods to declarative registry`
6. `feat(parsley): migrate file, dir, request, response methods to declarative registry`
7. `refactor(parsley): deduplicate dict-subtype dispatch` (optional, Task 12)

Tier 3:
8. `feat(parsley): migrate record and session methods to declarative registry`
9. `feat(parsley): migrate DBConnection and SFTP methods to declarative registry`
10. `feat(parsley): migrate table methods to declarative registry`
11. `feat(parsley): migrate DSLSchema methods to declarative registry`
12. `feat(parsley): migrate remaining types and remove TypeMethods` (Tasks 20+21)
13. `docs: update FEAT-111 and FEAT-137 status to implemented`

Grouping related small types into single commits is fine. The key rule: **each commit should leave all tests passing**.

## Validation Checklist

- [ ] All tests pass: `go test ./pkg/parsley/...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] `pars describe array` shows all methods from registry
- [ ] `pars describe dictionary` shows all methods from registry
- [ ] `pars describe datetime` shows methods (and dict methods accessible)
- [ ] All other types' `pars describe` output is accurate
- [ ] `TypeMethods` map removed from `introspect.go` (or reduced to empty types only)
- [ ] `arrayMethods` and `dictionaryMethods` string slices removed
- [ ] FEAT-111 status updated to `implemented`
- [ ] FEAT-137 status updated to `implemented`

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Dictionary `nil` return semantics broken | High — user-defined functions on dicts would stop working | Preserve `nil` return explicitly in `evalDictionaryMethod`; test with dict that has function-valued keys |
| Dict-subtype fallthrough broken | Medium — e.g. `datetime.keys()` would error | Test dictionary methods on each subtype after migration |
| `entries()` arity (0 or 2) doesn't fit spec system | Low — cosmetic | Use `"0+"` with internal validation; same error message as before |
| `EvalTableMethod` external callers break | Medium | Grep for usage before changing; keep exported wrapper if needed |
| Large types (table, DSLSchema) introduce subtle regressions | Medium | These types have many methods; test thoroughly, commit per-type |
| `methods.go` merge conflicts if other work touches it | Low | Tier 1 should go first while the file is stable |

## Deferred Items

Items to add to `work/BACKLOG.md` after implementation:
- Consider pre-parsing arity specs into a struct at registration time (avoids string parsing per call). Not needed unless profiling shows it's hot — it won't.
- Consider generating method documentation from registry metadata (Description field is already there)
- Consider adding a `pars describe --all-methods` command that dumps every type's registry

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| | | | |