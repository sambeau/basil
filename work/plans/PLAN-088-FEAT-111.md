# PLAN-088: FEAT-111 Declarative Method Registry

**Feature:** FEAT-111 - Declarative Method Registry  
**Created:** 2025-01-15  
**Status:** In Progress (Phase 6a)  
**Effort:** 8-12 hours (significant refactoring)

## Progress Summary

| Phase | Status | Notes |
|-------|--------|-------|
| Phase 1: Registry Infrastructure | ✅ Complete | `method_registry.go` created |
| Phase 2: String Methods | ✅ Complete | `methods_string.go` created, ~30 methods migrated |
| Phase 3: Migrate Types (partial) | ✅ Partial | integer, float, money migrated |
| Phase 4: Update Introspection | ✅ Complete | `describe()` reads from registries |
| Phase 5: Handle Special Cases | 🔲 Pending | Dictionary-based types pending |
| Phase 6a: Verification & Cleanup | 🔄 In Progress | Audit done, removing stale TypeMethods |
| Phase 6b: Final Cleanup | 🔲 Pending | After all types migrated |

**Migrated Types:** string, integer, float, money  
**Remaining Types:** array, dictionary, boolean, null, datetime, duration, path, url, regex, file, directory

---

## Overview

Refactor `methods.go` to use a declarative method registry that serves as the single source of truth for both method dispatch AND introspection. This eliminates the synchronization problem between `methods.go` (implementation) and `introspect.go` (documentation via `TypeMethods`).

---

## Current State Analysis

### Files Involved

| File | Lines | Role |
|------|-------|------|
| `pkg/parsley/evaluator/methods.go` | ~3500 | Method implementations via switch statements |
| `pkg/parsley/evaluator/introspect.go` | ~1200 | `TypeMethods` map for `describe()` output |
| `pkg/parsley/evaluator/eval_method_dispatch.go` | ~360 | Top-level `dispatchMethodCall` function |

### Current Architecture

1. **Dispatch Flow:**
   - `dispatchMethodCall()` switches on object type
   - Each type has its own `eval<Type>Method()` function
   - Each `eval<Type>Method()` uses a large switch on method name

2. **Introspection:**
   - `TypeMethods` in `introspect.go` is a static map[string][]MethodInfo
   - Manually maintained ~210 lines of metadata
   - Easily drifts from actual implementations

3. **Fuzzy Matching:**
   - Each type has a string slice (e.g., `stringMethods = []string{...}`)
   - Used for "did you mean?" suggestions in error messages

### Pain Points

- Adding a method requires changes in 3 places: implementation, `TypeMethods`, and fuzzy list
- No compile-time validation that metadata matches implementation
- FEAT-110 validation tests catch drift but don't prevent it

---

## Target Architecture

### Registry Structure

```go
// MethodEntry defines a single method with its implementation and metadata
type MethodEntry struct {
    Fn          MethodFunc
    Arity       string // "0", "1", "0-1", "1+", etc.
    Description string
}

// MethodFunc is the signature for all method implementations
// The receiver type is known from which registry the method belongs to
type MethodFunc func(receiver Object, args []Object, env *Environment) Object

// MethodRegistry maps method names to their entries for a type
type MethodRegistry map[string]MethodEntry
```

### Per-Type Registries

Each type will have its own registry with method entries:

```go
var StringMethodRegistry = MethodRegistry{
    "toUpper": {
        Fn:          stringToUpper,
        Arity:       "0",
        Description: "Convert to uppercase",
    },
    // ...
}
```

### Unified Dispatch

```go
func evalStringMethod(str *String, method string, args []Object, env *Environment) Object {
    entry, ok := StringMethodRegistry[method]
    if !ok {
        return unknownMethodError(method, "string", StringMethodRegistry.Names())
    }
    if !checkArity(entry.Arity, len(args)) {
        return newArityErrorFromSpec(method, entry.Arity, len(args))
    }
    return entry.Fn(str, args, env)
}
```

### Introspection Integration

```go
// GetMethodsForType returns method info from the registry (called by describe())
func GetMethodsForType(typeName string) []MethodInfo {
    registry := getRegistryForType(typeName)
    if registry == nil {
        return nil
    }
    methods := make([]MethodInfo, 0, len(registry))
    for name, entry := range registry {
        methods = append(methods, MethodInfo{
            Name:        name,
            Arity:       entry.Arity,
            Description: entry.Description,
        })
    }
    sort.Slice(methods, func(i, j int) bool {
        return methods[i].Name < methods[j].Name
    })
    return methods
}
```

---

## Implementation Phases

### Phase 1: Define Registry Infrastructure (1 hour)

**Goal:** Add new types and helper functions without changing existing behavior.

**Tasks:**
1. Create `pkg/parsley/evaluator/method_registry.go`:
   - Define `MethodEntry` struct
   - Define `MethodFunc` type
   - Define `MethodRegistry` type with helper methods:
     - `Names() []string` - for fuzzy matching
     - `Get(name string) (MethodEntry, bool)` - lookup
   - Implement `checkArity(spec string, got int) bool`
   - Implement `newArityErrorFromSpec(method, spec string, got int) *Error`

2. Add registry-to-type mapping infrastructure:
   ```go
   var typeRegistries = map[string]MethodRegistry{}
   
   func RegisterTypeRegistry(typeName string, registry MethodRegistry) {
       typeRegistries[typeName] = registry
   }
   
   func getRegistryForType(typeName string) MethodRegistry {
       return typeRegistries[typeName]
   }
   ```

**Verification:**
- Code compiles
- All existing tests pass
- No behavioral changes yet

**Implementation Notes (2025-01-15):**
- Created `pkg/parsley/evaluator/method_registry.go`
- Added `MethodEntry`, `MethodFunc`, `MethodRegistry` types
- Added `checkArity()` and `newArityErrorFromSpec()` helpers
- Added `dispatchFromRegistry()` for uniform dispatch
- Added `newArityErrorMin()` to `eval_errors.go` for variadic methods

---

### Phase 2: Migrate String Methods (2 hours)

**Goal:** Convert string methods to registry-based dispatch as proof of concept.

**Tasks:**
1. Extract each string method case into a standalone function:
   ```go
   func stringToUpper(receiver Object, args []Object, env *Environment) Object {
       str := receiver.(*String)
       return &String{Value: strings.ToUpper(str.Value)}
   }
   ```

2. Create `StringMethodRegistry` with all 28+ string methods

3. Update `evalStringMethod` to use registry lookup:
   ```go
   func evalStringMethod(str *String, method string, args []Object, env *Environment) Object {
       entry, ok := StringMethodRegistry[method]
       if !ok {
           return unknownMethodError(method, "string", StringMethodRegistry.Names())
       }
       if !checkArity(entry.Arity, len(args)) {
           return newArityErrorFromSpec(method, entry.Arity, len(args))
       }
       return entry.Fn(str, args, env)
   }
   ```

4. Remove the old `stringMethods = []string{...}` slice (now redundant)

5. Register in init: `RegisterTypeRegistry("string", StringMethodRegistry)`

**Verification:**
- All string method tests pass
- `describe("hello")` shows correct methods
- Error messages with fuzzy matching still work

**Implementation Notes (2025-01-15):**
- Created `pkg/parsley/evaluator/methods_string.go`
- Used `init()` function to avoid initialization cycle
- Migrated 30+ string methods to standalone functions
- Removed `stringMethods` slice from `methods.go`
- Registry now includes additional methods not in TypeMethods (toBox, repr, toJSON, parseMarkdown)

---

### Phase 3: Migrate Remaining Types (4-5 hours)

**Goal:** Convert all type-specific method handlers to registry-based dispatch.

**Order of migration** (from smallest to largest for quick wins):

| Type | Est. Methods | Complexity |
|------|--------------|------------|
| boolean | 0 | Trivial |
| null | 0 | Trivial |
| integer | 8 | Low |
| float | 10 | Low |
| money | 5 | Low |
| duration | 3 | Low |
| datetime | 6 | Low |
| path | 10 | Medium |
| url | 7 | Medium |
| regex | 7 | Medium |
| file | 5 | Low |
| directory | 4 | Low |
| dictionary | 12 | Medium |
| array | 25 | High |

**Tasks per type:**
1. Extract switch cases to standalone functions
2. Create `<Type>MethodRegistry` 
3. Update `eval<Type>Method` to use registry
4. Remove old method list slice
5. Register with `RegisterTypeRegistry`
6. Run tests

**Implementation Notes (2025-01-15):**
- Created `pkg/parsley/evaluator/methods_numeric.go` for integer/float
- Created `pkg/parsley/evaluator/methods_money.go` for money
- Fixed humanize arity from "0" to "0-1" (caught by existing tests)
- Integer registry includes 8 methods (abs, format, currency, percent, humanize, toBox, repr, toJSON)
- Float registry includes 11 methods (adds round, floor, ceil)
- Money registry includes 9 methods (format, abs, negate, split, toJSON, toBox, repr, toDict, inspect)

**Notes:**
- Some methods need `env` parameter, some don't - the signature includes it for uniformity
- Dictionary-based types (datetime, duration, path, url, regex, file, directory) may need special handling for the receiver type assertion
- Array methods with closures (map, filter, reduce) are more complex

---

### Phase 4: Update Introspection (1 hour)

**Goal:** Remove `TypeMethods` from `introspect.go` and read from registries instead.

**Tasks:**
1. Add `GetMethodsForType(typeName string) []MethodInfo` function in `method_registry.go`

2. Update `builtinDescribe()` in `introspect.go`:
   ```go
   // Replace:
   methodInfos, ok := TypeMethods[methodKey]
   
   // With:
   methodInfos := GetMethodsForType(methodKey)
   ok := len(methodInfos) > 0
   ```

3. Delete `var TypeMethods = map[string][]MethodInfo{...}` (~210 lines)

4. Update FEAT-110 validation tests if needed (they should now be redundant by design)

**Verification:**
- `describe("hello")` output unchanged
- `describe([1,2,3])` output unchanged
- `describe(123)` output unchanged
- All introspection tests pass

**Implementation Notes (2025-01-15):**
- Updated `builtinDescribe()` in `introspect.go` to check registries first
- Falls back to TypeMethods for non-migrated types
- Updated validation tests to use `getMethodsForValidation()` helper
- `describe()` now shows all methods from registries (including ones not in TypeMethods)

---

### Phase 5: Handle Special Cases (1 hour)

**Goal:** Ensure edge cases and special dispatch patterns work correctly.

**Special cases to handle:**

1. **Dictionary-based types** (datetime, path, url, etc.):
   - These dispatch via type detection functions like `isDatetimeDict()`
   - Need to map subtypes to registries correctly
   - Consider: `getRegistryForType` should handle subtype mapping

2. **Universal `.type()` method**:
   - Currently handled specially in `dispatchMethodCall`
   - Keep as special case OR add to every registry
   - Decision: Keep special case (single implementation point)

3. **Methods requiring different signatures**:
   - Most methods can use `(receiver Object, args []Object, env *Environment) Object`
   - Some don't need env - that's fine, signature is uniform
   - Some don't have receiver (e.g., session methods) - handle specially

4. **DBConnection and SFTP methods**:
   - In `eval_method_dispatch.go`, not `methods.go`
   - Decision: Leave in current files, create registries there
   - They're server-context methods, logically separate

**Tasks:**
1. Create mapping from subtype names to registries
2. Verify dictionary-based type dispatch works
3. Test session/dev/table module methods (may stay as switch for now)

---

### Phase 6a: Verification & Cleanup of Migrated Types (1 hour)

**Goal:** Remove stale `TypeMethods` entries for migrated types and fix discrepancies found during spec-vs-implementation audit.

**Background — Audit Findings (2025-01-16):**

A full audit of the implementation against the FEAT-111 spec revealed that `TypeMethods` entries for all four migrated types are **stale dead code** that is significantly out of sync with the registries:

| Type | Registry Methods | TypeMethods Methods | Missing from TypeMethods | Arity Mismatches |
|------|-----------------|--------------------|--------------------------|-----------------| 
| string | 31 | 27 | `parseMarkdown`, `toBox`, `repr`, `toJSON` | — |
| integer | 8 | 3 | `currency`, `percent`, `toBox`, `repr`, `toJSON` | `humanize`: "0" vs "0-1" |
| float | 11 | 6 | `currency`, `percent`, `toBox`, `repr`, `toJSON` | `humanize`: "0" vs "0-1" |
| money | 9 | 4 | `split`, `toJSON`, `toBox`, `repr`, `inspect` | — |

This drift is exactly the problem FEAT-111 exists to solve. The registries are authoritative; the `TypeMethods` entries are dead code since `builtinDescribe()` already prefers registries.

Additional finding: `float.format()` has arity `"0-2"` with description "Format with decimals and locale" but the implementation only handles one optional arg (locale). The second arg (decimals) was never implemented. Fix: change arity to `"0-1"` and description to `"Format with locale"` to match reality. (Decimal formatting is available via `.round(n).format()`.)

Additional finding: The spec lists 15 types to migrate but the codebase has additional types not in the original list (`table`, `record`, `session`, `dev`, `tablemodule`, `DSLSchema`, `MdDoc`, `sftpfile`). These are noted for Phase 5 scoping.

**Tasks:**
1. Remove `TypeMethods` entries for `string`, `integer`, `float`, `money`
2. Fix `float.format()` arity from `"0-2"` to `"0-1"`, description to `"Format with locale"`
3. Run tests to verify no regressions
4. Update FEAT-111 spec to check off AC #3 as partial (migrated types cleaned up)

**Verification:**
- All evaluator tests pass
- All Parsley tests pass
- `describe("hello")` still shows all string methods (from registry)
- `describe(42)` still shows all integer methods (from registry)
- `describe(3.14)` still shows all float methods (from registry)
- `describe($1.00)` still shows all money methods (from registry)
- Validation tests still pass (they already prefer registries)

---

### Phase 6b: Final Cleanup and Documentation (1 hour)

**Goal:** Remove remaining dead code after all types are migrated, update documentation, finalize.

**Tasks:**
1. Remove old method list slices (`arrayMethods`, `dictionaryMethods`, etc.) as types are migrated
2. Remove entire `TypeMethods` map once all types are migrated
3. Update any comments referencing old architecture
4. Add documentation comments to new types
5. Update FEAT-111 spec with implementation notes
6. Consider: Should FEAT-110 validation tests remain? (Belt and suspenders)
7. Account for additional types not in original spec: `table`, `record`, `session`, `dev`, `tablemodule`, `DSLSchema`, `MdDoc`, `sftpfile`

---

## Risk Mitigation

### Performance

**Concern:** Map lookup may be slower than switch statements for method dispatch.

**Mitigation:**
1. Go's switch compiler optimization is good, but map[string] lookup is O(1)
2. For small maps (<30 entries), difference is negligible
3. Run benchmarks before/after:
   ```bash
   go test ./pkg/parsley/evaluator/... -bench=BenchmarkMethod -benchmem
   ```
4. If regression: consider compile-time code generation (unlikely needed)

### Type Safety

**Concern:** Receiver type assertion in generic method functions.

**Mitigation:**
- Each registry is type-specific, so assertions should always succeed
- Add debug-mode assertion checks if concerned
- Existing pattern already does this in switch cases

### Breaking Changes

**Concern:** Subtle behavioral differences after refactor.

**Mitigation:**
- FEAT-110 validation tests provide safety net
- Run full test suite at each phase
- Compare `describe()` output before/after for each type
- Snapshot testing for error messages

---

## Test Plan

### Unit Tests

| Test | Description |
|------|-------------|
| Registry lookup | `StringMethodRegistry["toUpper"]` exists |
| Arity checking | `checkArity("0-1", 0)` returns true |
| Method dispatch | String methods work via registry |
| Unknown method | Returns error with suggestions |
| Introspection | `describe()` returns correct info |

### Integration Tests

| Test | Description |
|------|-------------|
| All existing method tests | Must pass unchanged |
| FEAT-110 validation tests | Should pass (no drift by design) |
| CLI tests | `pars -e '"hello".toUpper()'` works |

### Benchmark Tests

```go
func BenchmarkStringMethodDispatch(b *testing.B) {
    str := &String{Value: "hello"}
    for i := 0; i < b.N; i++ {
        evalStringMethod(str, "toUpper", []Object{}, nil)
    }
}
```

---

## File Changes Summary

| File | Action | Lines Changed (Est.) |
|------|--------|---------------------|
| `pkg/parsley/evaluator/method_registry.go` | Create | +150 |
| `pkg/parsley/evaluator/methods.go` | Major refactor | ~±500 |
| `pkg/parsley/evaluator/introspect.go` | Delete `TypeMethods` | -210 |
| `pkg/parsley/evaluator/eval_method_dispatch.go` | Minor updates | ±20 |

---

## Rollback Plan

If issues are discovered after merge:

1. Git revert the merge commit
2. All changes are contained in evaluator package
3. No external API changes - rollback is clean

---

## Success Criteria

- [ ] All existing tests pass
- [ ] No performance regression (benchmark within 10%)
- [ ] Adding a new method requires only ONE code change (registry entry)
- [ ] `describe()` output identical to before
- [ ] `TypeMethods` map deleted from `introspect.go`
- [ ] FEAT-110 validation tests pass (now redundant but kept as safety net)

---

## Dependencies

- **Requires:** FEAT-110 complete (validation tests as safety net)
- **Blocks:** FEAT-112 (help system will read from registries)

---

## Notes

### Files Created
- `pkg/parsley/evaluator/method_registry.go` - Registry infrastructure
- `pkg/parsley/evaluator/methods_string.go` - String method registry
- `pkg/parsley/evaluator/methods_numeric.go` - Integer/Float method registries
- `pkg/parsley/evaluator/methods_money.go` - Money method registry

### Files Modified
- `pkg/parsley/evaluator/methods.go` - Removed switch statements for migrated types
- `pkg/parsley/evaluator/eval_errors.go` - Added `newArityErrorMin()`
- `pkg/parsley/evaluator/introspect.go` - Updated `builtinDescribe()` to use registries
- `pkg/parsley/evaluator/introspect_validation_test.go` - Updated to use registries

### Key Decisions
1. Used `init()` functions to avoid initialization cycles (registry -> method -> Eval -> dispatch -> registry)
2. Kept TypeMethods entries for backward compatibility during migration
3. Updated validation tests to read from registries when available
4. Registries include methods that weren't documented in TypeMethods (toBox, repr, toJSON, etc.)

### Remaining Work
1. ~~Remove stale TypeMethods entries for migrated types~~ → Phase 6a
2. ~~Fix float.format() arity "0-2" → "0-1"~~ → Phase 6a
3. Migrate array methods (largest, ~25 methods with closures)
4. Migrate dictionary methods (~12 methods)
5. Migrate dictionary-based types (datetime, duration, path, url, regex, file, dir)
6. Migrate additional types not in original spec: table, record, session, dev, tablemodule, DSLSchema, MdDoc, sftpfile
7. Remove entire TypeMethods map after all types migrated (Phase 6b)
8. Consider removing FEAT-110 validation tests (now redundant by construction)