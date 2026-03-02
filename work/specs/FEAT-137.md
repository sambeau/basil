---
id: FEAT-137
title: "Complete Method Registry Migration"
status: implemented
priority: medium
created: 2026-03-02
author: "@human"
---

# FEAT-137: Complete Method Registry Migration

## Summary

FEAT-111 introduced a declarative `MethodRegistry` system for method dispatch and migrated 5 types (string, integer, float, money, unit). The remaining ~20 types still use hand-coded switch statements with inconsistent arity checking, no `pars describe` visibility, and less helpful error messages. This spec covers migrating all remaining types to the registry, completing the work started in FEAT-111.

Discovered during the [1.0 Ship Review](../reports/1.0-SHIP-REVIEW.md) (Section 5: Method Dispatch Inconsistency).

## User Story

As a Parsley developer, I want `pars describe array` (and all other types) to show accurate method listings so that I can discover and use methods without reading source code.

As a Parsley maintainer, I want a single pattern for defining methods so that adding or modifying methods is consistent across all types.

## Acceptance Criteria

### Tier 1 — Core types (highest priority)

- [ ] `array` methods migrated to `ArrayMethodRegistry` (~23 methods)
- [ ] `dictionary` methods migrated to `DictionaryMethodRegistry` (~15 methods)
- [ ] `pars describe array` shows all methods from registry
- [ ] `pars describe dictionary` shows all methods from registry
- [ ] All existing array and dictionary tests pass

### Tier 2 — Dictionary-like types and simple types

- [ ] `datetime` methods migrated (12 distinct methods, 14 registry entries)
- [ ] `duration` methods migrated (9 distinct methods, 11 registry entries)
- [ ] `path` methods migrated (10 methods)
- [ ] `url` methods migrated (9 methods)
- [ ] `regex` methods migrated (7 methods)
- [ ] `file` methods migrated (5 methods)
- [ ] `dir` methods migrated (4 methods)
- [ ] `boolean` methods migrated (~4 methods)
- [ ] `null` methods migrated (~4 methods)
- [ ] `request` methods migrated (~1 method)
- [ ] `response` methods migrated (~4 methods)
- [ ] `pars describe` works correctly for all Tier 2 types
- [ ] All existing tests pass

### Tier 3 — Server-specific types

- [x] `DBConnection` methods migrated (8 methods)
- [x] `SFTPConnection` methods migrated (1 method)
- [x] `SFTPFileHandle` methods migrated (3 methods)
- [x] `session` methods migrated (11 methods)
- [x] `record` methods migrated (19 methods)
- [x] `table` methods migrated (38 methods)
- [x] `DSLSchema` methods migrated (6 methods)
- [x] `TableBinding` methods migrated (17 methods)
- [x] `MdDoc` methods migrated (16 methods)
- [x] `DevModule` methods migrated (5 methods)
- [x] `pars describe` works correctly for all Tier 3 types
- [x] All existing tests pass

### Cleanup

- [x] Stale `TypeMethods` entries removed from `introspect.go` for all migrated types
- [ ] `TypeMethods` map removed entirely once all entries are cleared (only empty `"function"` entry remains)
- [x] `arrayMethods` and `dictionaryMethods` string slices in `methods.go` removed (replaced by registry)
- [x] `booleanMethods` and `nullMethods` string slices in `methods.go` removed (replaced by registry)
- [ ] FEAT-111 status updated to `implemented` once complete

## Design Decisions

- **Continue the FEAT-111 pattern exactly**: Each type gets a `XxxMethodRegistry` variable initialised in `init()`, registered via `RegisterMethodRegistry()`. Method bodies move to named functions with the `MethodFunc` signature. The `dispatchFromRegistry` / `unknownMethodError` pattern handles dispatch and errors.

- **Dictionary-subtype fallthrough preserved as-is**: The `*Dictionary` case in `dispatchMethodCall` has special logic for datetime, path, url, regex, file, dir, request, and response dicts. **The fallthrough behaviour is inconsistent in the existing code**: `duration` does not fall through to dictionary methods at all (bare `return`), while all others do. `datetime` and `response` use a different two-stage nil/error check pattern from the rest. After migration, each subtype's dispatch block must preserve its own existing semantics exactly — do not normalise them.

- **Helper functions stay in place**: Functions like `sortArrayWithOptions`, `filterArrayWithFunction`, `formatDurationWithStyle`, etc. remain as-is. Only the switch-case dispatch and arity checks are replaced by registry entries pointing to new wrapper functions.

- **Tiers can be shipped independently**: Each tier is a self-contained unit of work. Tier 1 delivers the highest user-facing value. Tiers 2 and 3 can follow in separate branches.

- **No performance concern**: Registry dispatch uses `map[string]MethodEntry` lookup (O(1) hash) which is comparable to Go's switch-on-string compilation. The only new per-call overhead is `checkArity` parsing a short spec string, which is sub-microsecond and dwarfed by method body execution.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Migration Pattern

For each type, the migration follows these steps (using `array` as example):

1. **Create registry file** `methods_array.go` with:
   ```go
   var ArrayMethodRegistry MethodRegistry

   func init() {
       ArrayMethodRegistry = MethodRegistry{
           "length": {
               Fn:          arrayLength,
               Arity:       "0",
               Description: "Return the number of elements",
           },
           // ... all methods
       }
       RegisterMethodRegistry("array", ArrayMethodRegistry)
   }
   ```

2. **Extract method bodies** from switch cases into named `MethodFunc` functions:
   ```go
   func arrayLength(receiver Object, args []Object, env *Environment) Object {
       arr := receiver.(*Array)
       return &Integer{Value: int64(len(arr.Elements))}
   }
   ```

3. **Replace the eval function** to use registry dispatch:
   ```go
   func evalArrayMethod(arr *Array, method string, args []Object, env *Environment) Object {
       result := dispatchFromRegistry(ArrayMethodRegistry, "array", arr, method, args, env)
       if result != nil {
           return result
       }
       return unknownMethodError(method, "array", ArrayMethodRegistry.Names())
   }
   ```

4. **Remove stale data**: Delete the `arrayMethods` string slice and any `TypeMethods` entry in `introspect.go`.

5. **Run tests**: `go test ./pkg/parsley/...`

### Affected Components

| Component | Location | Changes |
|-----------|----------|---------|
| Array methods | `methods.go` L350-738 → new `methods_array.go` | Extract ~23 switch cases to registry |
| Dictionary methods | `methods.go` L1124-1336 → new `methods_dictionary.go` | Extract ~15 switch cases to registry |
| Boolean methods | `methods.go` L1461-1506 | Small — 4 methods |
| Null methods | `methods.go` L1512-1550 | Small — 4 methods |
| Datetime methods | `methods.go` L1579-1768 | 13 methods |
| Duration methods | `methods.go` L1786-2158 | 10 methods + format helpers |
| Path methods | `methods.go` L2165-2298 | 10 methods |
| URL methods | `methods.go` L2305-2445 | 9 methods |
| Regex methods | `methods.go` L2452-2605 | 7 methods |
| File methods | `methods.go` L2612-2731 | 5 methods |
| Dir methods | `methods.go` L2738-2850 | 4 methods |
| Request methods | `methods.go` L2857-2869 | 1 method |
| Response methods | `methods.go` L2876-2920 | 4 methods |
| DB methods | `eval_method_dispatch.go` L11-197 | 8 methods |
| SFTP methods | `eval_network_io.go` | 4 methods across two types (SFTPConnection: 1, SFTPFileHandle: 3) |
| Session methods | `stdlib_session.go` | ~11 methods |
| Record methods | `methods_record.go` | ~24 methods |
| Table methods | `stdlib_table.go` | 44 methods |
| DSLSchema methods | `stdlib_dsl_schema.go` | 41 unique methods (54 total cases) |
| TableBinding methods | `stdlib_schema_table_binding.go` | ~22 unique methods (35 total cases) |
| MdDoc methods | `stdlib_mddoc.go` | 16 methods |
| DevModule methods | `stdlib_dev.go` | 5 methods |
| Dispatch | `eval_method_dispatch.go` L202-380 | Dict-subtype fallthrough chain unchanged |
| Introspection | `introspect.go` | Remove stale `TypeMethods` entries |
| Method lists | `methods.go` L41-52 | Remove `arrayMethods`, `dictionaryMethods` slices |

### Dictionary-Subtype Dispatch Detail

The `*Dictionary` case in `dispatchMethodCall` checks `isDatetimeDict()`, `isPathDict()`, etc. and tries subtype-specific methods first, falling through to `evalDictionaryMethod` on `UNDEF-0002`. After migration, this becomes:

```go
case *Dictionary:
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
    // ... same pattern for other subtypes
```

This preserves the existing semantics: subtype methods shadow dictionary methods of the same name, but dictionary methods are available on all dict-typed values.

### Dependencies

- **Continues**: FEAT-111 (Declarative Method Registry)
- **References**: 1.0 Ship Review Section 5
- **Infrastructure**: `method_registry.go` — `MethodRegistry`, `MethodEntry`, `MethodFunc`, `dispatchFromRegistry`, `checkArity`, `RegisterMethodRegistry`, `GetMethodsForType` — all already exist and are stable

### Method Counts by Type

| Type | Methods | File | Tier |
|------|---------|------|------|
| `array` | 23 (24 registry entries — `fmt`/`format` alias) | `methods.go` | 1 |
| `dictionary` | 15 | `methods.go` | 1 |
| `datetime` | 12 (14 registry entries — `fmt`/`format` alias) | `methods.go` | 2 |
| `duration` | 9 (11 registry entries — `fmt`/`format` alias) | `methods.go` | 2 |
| `path` | 10 | `methods.go` | 2 |
| `url` | 9 | `methods.go` | 2 |
| `regex` | 7 | `methods.go` | 2 |
| `file` | 5 | `methods.go` | 2 |
| `dir` | 4 | `methods.go` | 2 |
| `boolean` | 4 | `methods.go` | 2 |
| `null` | 4 | `methods.go` | 2 |
| `response` | 4 | `methods.go` | 2 |
| `request` | 1 | `methods.go` | 2 |
| `DSLSchema` | 41 unique (54 total cases — some names overloaded) | `stdlib_dsl_schema.go` | 3 |
| `table` | 44 | `stdlib_table.go` | 3 |
| `record` | 24 | `methods_record.go` | 3 |
| `session` | 11 | `stdlib_session.go` | 3 |
| `TableBinding` | ~22 unique (35 total cases — fluent API aliases) | `stdlib_schema_table_binding.go` | 3 |
| `MdDoc` | 16 | `stdlib_mddoc.go` | 3 |
| `DevModule` | 5 | `stdlib_dev.go` | 3 |
| `SFTPConnection` | 1 | `eval_network_io.go` | 3 |
| `SFTPFileHandle` | 3 | `eval_network_io.go` | 3 |
| `DBConnection` | 8 | `eval_method_dispatch.go` | 3 |

### Edge Cases & Constraints

- **`env` parameter**: Some methods need `env`, some don't. The `MethodFunc` signature always includes `env *Environment` — methods that don't need it simply ignore it. Already solved in FEAT-111.

2. **Dictionary method fallthrough**: After migrating dict-subtypes, the fallthrough logic must be preserved. Each subtype tries its own registry first, then the dictionary registry. Test with e.g. `datetime_value.keys()` (a dictionary method called on a datetime).

3. **Receiver type assertions**: Each method function must assert the receiver to the concrete type. For dict-subtypes, the receiver is `*Dictionary` — same as today.

4. **`methods.go` shrinkage**: After Tiers 1 and 2, `methods.go` (currently 3,333 lines) should shrink substantially as method bodies move to dedicated files. Shared helpers (`sortArrayWithOptions`, `formatDurationWithStyle`, etc.) stay in `methods.go` or move alongside the methods that use them. Note that `getDictStringValue` is shared between `evalDatetimeMethod` and `evalDurationMethod` — leave it in `methods.go` until both are migrated.

5. **`unknownMethodError` signature**: Currently takes `[]string`. After migration, callers use `registry.Names()` which returns `[]string`. Compatible — no change needed.

## Related

- Plan: `work/plans/PLAN-117-method-registry-migration.md`
- Continues: FEAT-111 (Declarative Method Registry)
- Ship review: `work/reports/1.0-SHIP-REVIEW.md` Section 5
- Registry infrastructure: `pkg/parsley/evaluator/method_registry.go`
- Already migrated: `methods_string.go` (string, 38 entries), `methods_numeric.go` (integer 13, float 16), `methods_money.go` (money, 14 entries), `methods_unit.go` (unit, 14 entries)
- Introspection: `pkg/parsley/evaluator/introspect.go`
