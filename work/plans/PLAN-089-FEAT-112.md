---
id: PLAN-089
feature: FEAT-112
title: "Implementation Plan: Unified Help System"
status: draft
created: 2026-02-11
---

# PLAN-089: FEAT-112 Unified Help System

## Progress Summary

| Phase | Description | Status | Notes |
|-------|-------------|--------|-------|
| 1 | Operator Metadata | ✅ Complete | Added `OperatorInfo` / `OperatorMetadata` to `introspect.go` |
| 2 | Help Engine Core | ✅ Complete | New `pkg/parsley/help/` package with `help.go`, `format.go` |
| 3 | CLI Integration | ✅ Complete | `pars describe <topic>` subcommand with `--json` flag |
| 4 | REPL Integration | ✅ Complete | `:describe` and `:d` commands, tab completion |
| 5 | Tests | ✅ Complete | 17 test functions covering all topic types |
| 6 | JSON struct tags | ✅ Complete | Add `json:"..."` tags with `omitempty` for clean, spec-conformant JSON output |
| 7 | Module export accuracy | ✅ Complete | Fix `moduleExportsMap` against actual source; add missing `StdlibExports` entries |
| 8 | Self-describing modules | ✅ Complete | Modules carry their own `ModuleMeta`; eliminated `moduleExportsMap`, `StdlibExports`, `StdlibModuleDescriptions`, `BasilModuleDescriptions`. Fixes shared-name collision. |

## Overview

Create a unified help system that provides static, topic-based documentation for Parsley types, builtins, operators, and modules. Accessible via `pars describe <topic>` (CLI) and `:describe <topic>` (REPL). Distinct from the runtime `describe()` builtin which introspects live values.

The help engine reads from existing metadata in the `evaluator` package: method registries (FEAT-111), `TypeMethods`, `TypeProperties`, `BuiltinMetadata`, `StdlibModuleDescriptions`, `StdlibExports`. New operator metadata will be added.

## Prerequisites

- [x] FEAT-111 registry infrastructure exists (`method_registry.go`, `GetRegistryForType()`, `GetMethodsForType()`)
- [x] Introspection metadata exported (`TypeMethods`, `TypeProperties`, `BuiltinMetadata`, `StdlibExports`, etc.)
- [ ] Operator metadata (Phase 1 of this plan)

## Current State Analysis

### Existing Data Sources

| Data | Location | Exported? | Notes |
|------|----------|-----------|-------|
| Type methods (migrated) | `method_registry.go` → `typeRegistries` | Yes, via `GetRegistryForType()` / `GetMethodsForType()` | string, integer, float, money |
| Type methods (unmigrated) | `introspect.go` → `TypeMethods` | Yes | array, dictionary, datetime, duration, path, url, regex, file, directory, table, etc. |
| Type properties | `introspect.go` → `TypeProperties` | Yes | Properties for path, url, datetime, etc. |
| Builtin metadata | `introspect.go` → `BuiltinMetadata` | Yes | All builtins with arity, params, category, description |
| Stdlib module descriptions | `introspect.go` → `StdlibModuleDescriptions` | Yes | Short descriptions per module |
| Basil module descriptions | `introspect.go` → `BasilModuleDescriptions` | Yes | @basil/http, @basil/auth |
| Stdlib export metadata | `introspect.go` → `StdlibExports` | Yes | Per-export arity and description |
| Operator metadata | **Does not exist** | — | Must be created in Phase 1 |

### Key Design Constraints

- **Topic-based, not expression-based**: `describe string` looks up the type, not a variable. This is static documentation.
- **Separate from `describe()` builtin**: The builtin remains for runtime value introspection. The help system is for static documentation.
- **Single source of truth**: Reads from registries and metadata maps — no duplicate data.
- **Consistent output**: CLI and REPL produce identical content (only terminal width may differ).

## Tasks

### Phase 1: Operator Metadata

**Files**: `pkg/parsley/evaluator/introspect.go`
**Estimated effort**: Small (30 min)

Add structured operator metadata to `introspect.go`, consistent with existing `BuiltinMetadata` pattern.

Steps:
1. Define `OperatorInfo` struct with fields: `Symbol`, `Name`, `Description`, `Category`, `Example`
2. Create `OperatorMetadata` map covering all operators from `eval_infix.go`
3. Group operators by category: arithmetic, comparison, logical, collection, string, regex, range, assignment

Operators to document (from `eval_infix.go` and parser):
```
// Arithmetic
+   Addition / string concatenation
-   Subtraction / set difference
*   Multiplication / string repetition
/   Division / array chunking
%   Modulo
**  Exponentiation

// Comparison
==  Equal
!=  Not equal
<   Less than
>   Greater than
<=  Less than or equal
>=  Greater than or equal

// Logical
&&, and   Logical AND / set intersection
||, or    Logical OR / set union
!         Logical NOT

// Collection
++  Concatenation (string, array, dictionary)
in  Membership test
not in  Negated membership test
..  Range (inclusive)

// Regex
~   Regex match (returns captures or null)
!~  Regex non-match (returns boolean)

// Pipe
|>  Pipe (pass left as first arg to right)

// Ternary
?:  Ternary (condition ? then : else)

// Null coalescing
??  Null coalescing (left ?? default)
```

Tests:
- Verify `OperatorMetadata` has entries for all operators
- Verify each entry has non-empty description

---

### Phase 2: Help Engine Core

**Files**: `pkg/parsley/help/help.go` (new), `pkg/parsley/help/format.go` (new)
**Estimated effort**: Medium (2-3 hours)

Create a new `help` package that provides topic-based documentation lookup and output formatting.

Steps:

1. **Create `pkg/parsley/help/help.go`** with:

   ```
   TopicResult struct:
     Kind        string   // "type", "module", "builtin", "builtin-list", "operator-list", "type-list"
     Name        string
     Description string
     Methods     []evaluator.MethodInfo
     Properties  []evaluator.PropertyInfo
     Builtins    []evaluator.BuiltinInfo    // For "builtins" topic
     Operators   []evaluator.OperatorInfo   // For "operators" topic
     Exports     []ExportEntry              // For modules
     TypeNames   []string                   // For "types" topic
     Params      []string                   // For specific builtin
     Arity       string                     // For specific builtin
     Category    string                     // For specific builtin

   func DescribeTopic(topic string) (*TopicResult, error)
   ```

2. **Topic resolution order** in `DescribeTopic()`:
   - Check type registries first (`GetRegistryForType(topic)`) — migrated types
   - Check `TypeMethods[topic]` — unmigrated types
   - Check module paths (`@std/*`, `@basil/*`)
   - Check special keywords: `builtins`, `operators`, `types`
   - Check specific builtin names (`BuiltinMetadata[topic]`)
   - Return error with suggestions for unknown topics

3. **Create `pkg/parsley/help/format.go`** with:
   - `FormatText(result *TopicResult, width int) string` — terminal-friendly output
   - `FormatJSON(result *TopicResult) ([]byte, error)` — machine-readable JSON

4. **Type help** (`DescribeTopic("string")`):
   - Get methods from registry or `TypeMethods`
   - Get properties from `TypeProperties`
   - Format with aligned columns showing `.method(params)  Description`

5. **Module help** (`DescribeTopic("@std/math")`):
   - Look up module name in `StdlibModuleDescriptions` or `BasilModuleDescriptions`
   - Get exports from `StdlibExports`
   - Group by constants vs functions

6. **Builtins list** (`DescribeTopic("builtins")`):
   - Read all entries from `BuiltinMetadata`
   - Group by `Category` field
   - Format: `name(params)  Description`

7. **Operators list** (`DescribeTopic("operators")`):
   - Read all entries from `OperatorMetadata`
   - Group by `Category` field
   - Format: `symbol  Description`

8. **Types list** (`DescribeTopic("types")`):
   - Collect all type names from registries + `TypeMethods` keys
   - Sort alphabetically
   - Show brief list: `string, integer, float, ...`

9. **Specific builtin** (`DescribeTopic("JSON")`):
   - Look up in `BuiltinMetadata`
   - Show signature, description, arity, category, deprecation warning if any

Tests:
- `DescribeTopic("string")` returns type result with methods
- `DescribeTopic("integer")` returns registry-sourced methods
- `DescribeTopic("@std/math")` returns module with exports
- `DescribeTopic("builtins")` returns grouped builtin list
- `DescribeTopic("operators")` returns operator list
- `DescribeTopic("types")` returns all type names
- `DescribeTopic("JSON")` returns specific builtin info
- `DescribeTopic("nonexistent")` returns error with suggestion
- `FormatText()` produces readable output with alignment
- `FormatJSON()` produces valid JSON

---

### Phase 3: CLI Integration

**Files**: `cmd/pars/main.go`
**Estimated effort**: Small (30-45 min)

Add `describe` as a subcommand alongside the existing `fmt` subcommand.

Steps:

1. Add `describe` case to the subcommand switch in `main()` (line ~55):
   ```
   case "describe":
       describeCommand(os.Args[2:])
       return
   ```

2. Implement `describeCommand(args []string)`:
   - No args → print usage and available topics, exit 1
   - Parse `--json` flag from args
   - Remaining arg is the topic
   - Call `help.DescribeTopic(topic)`
   - On error: print error to stderr, exit 1
   - On success: format with `FormatText()` (width 80) or `FormatJSON()`, print to stdout

3. Update `printHelp()` to include `describe` in the commands section:
   ```
   Commands:
     fmt                   Format Parsley source files
     describe <topic>      Show help for a type, module, or builtin
   ```

Tests:
- Build succeeds with new subcommand
- `pars describe` with no args shows usage
- `pars describe string` produces output
- `pars describe --json string` produces valid JSON
- `pars describe unknown` exits with error

---

### Phase 4: REPL Integration

**Files**: `pkg/parsley/repl/repl.go`
**Estimated effort**: Small (30 min)

Add `:describe` command to the REPL command handler.

Steps:

1. Add `:describe` case to `handleReplCommand()` switch (after `:raw` case):
   ```
   case starts with ":describe":
       parts := strings.Fields(cmd)
       if len(parts) < 2:
           print usage
           return rawMode, true
       topic := parts[1]
       result, err := help.DescribeTopic(topic)
       if err:
           print error
           return rawMode, true
       // Use 80 as default width (terminal width detection is out of scope)
       output := help.FormatText(result, 80)
       print output
       return rawMode, true
   ```

2. Update `:help` output to include `:describe`:
   ```
   :describe <topic>  Show help (types, builtins, operators, modules)
   ```

3. Add `:describe` and `:d` as aliases (`:d string` as shorthand)

4. Add `describe` and common topic names to `completionWords` for tab completion

Tests:
- `:describe string` produces output in REPL
- `:describe` with no arg shows usage
- `:describe unknown` shows error
- `:help` lists the new command

---

### Phase 5: Tests

**Files**: `pkg/parsley/help/help_test.go` (new)
**Estimated effort**: Medium (1-2 hours)

Steps:

1. **Unit tests for `DescribeTopic()`**:
   - Type topics: `string`, `array`, `dictionary`, `integer`, `float`, `money`, `datetime`, `path`, `url`, `table`
   - Module topics: `@std/math`, `@std/table`, `@basil/http`
   - Special topics: `builtins`, `operators`, `types`
   - Builtin topics: `JSON`, `CSV`, `now`, `fail`, `print`
   - Error case: unknown topic returns error
   - Error case: empty string returns error

2. **Unit tests for `FormatText()`**:
   - Type result formats with aligned methods
   - Module result shows grouped exports
   - Builtins result shows categories
   - Output respects width parameter

3. **Unit tests for `FormatJSON()`**:
   - Output is valid JSON
   - Contains expected fields
   - Round-trips correctly

4. **Integration tests** (build and run):
   - `go build ./cmd/pars/` succeeds
   - Full test suite passes: `go test ./...`

---

## Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make dev`
- [ ] Linter passes: `golangci-lint run`
- [ ] `pars describe string` shows string methods
- [ ] `pars describe array` shows array methods
- [ ] `pars describe @std/math` shows module exports
- [ ] `pars describe builtins` lists all builtins by category
- [ ] `pars describe operators` lists all operators
- [ ] `pars describe JSON` shows JSON builtin details
- [ ] `pars describe types` lists all types
- [ ] `pars describe --json string` outputs valid JSON
- [ ] `pars describe unknown` shows error with suggestions
- [ ] REPL `:describe string` works identically to CLI
- [ ] REPL `:describe` with no arg shows usage
- [ ] REPL `:help` lists `:describe` command
- [ ] Documentation updated (spec status)
- [ ] work/BACKLOG.md updated with deferrals (if any)

## File Changes Summary

| File | Change |
|------|--------|
| `pkg/parsley/evaluator/introspect.go` | Add `OperatorInfo`, `OperatorMetadata` |
| `pkg/parsley/help/help.go` | **New** — Core help engine: `TopicResult`, `DescribeTopic()` |
| `pkg/parsley/help/format.go` | **New** — Output formatting: `FormatText()`, `FormatJSON()` |
| `pkg/parsley/help/help_test.go` | **New** — Tests for help engine |
| `cmd/pars/main.go` | Add `describe` subcommand, update help text |
| `pkg/parsley/repl/repl.go` | Add `:describe` command, update `:help`, add completions |

## Risk Mitigation

### Data Completeness
- **Risk**: Some types/builtins may lack metadata, producing sparse help output
- **Mitigation**: The help engine gracefully handles missing data — shows what's available. `TypeMethods` covers all unmigrated types. `BuiltinMetadata` is comprehensive. Only operator metadata is new.

### FEAT-111 Migration In Progress
- **Risk**: As types migrate from `TypeMethods` to registries, help engine must handle both
- **Mitigation**: `DescribeTopic()` checks registries first, falls back to `TypeMethods` — same pattern as `builtinDescribe()` in `introspect.go`. No code changes needed as migration continues.

### Module Export Metadata Gaps
- **Risk**: `StdlibExports` may not cover all exports for all modules
- **Mitigation**: Help engine shows what metadata exists. Module help can note "some exports may not be documented" if gaps are detected. This is acceptable for v1.

### Shared Export Names Across Modules
- **Risk**: `StdlibExports` is a flat map keyed by export name. Modules like `@std/valid`, `@std/schema`, and `@std/id` share export names (e.g., `string`, `email`, `uuid`) with different semantics. The help system shows whichever description is in the flat map, which may be wrong for a given module context (e.g., `@std/id`'s `uuid` shows "Check UUID format" from `@std/valid` instead of "Generate UUID v4").
- **Mitigation**: ~~Accepted for v1.~~ **Resolved by Phase 8** — module-scoped `ModuleMeta` eliminates the flat map and its collision problem.

## Tasks

### Phase 8: Self-Describing Modules

**Goal:** Make modules self-describing so the help system doesn't maintain a separate list. Each module carries its own `ModuleMeta` with description and per-export metadata, eliminating the three-way sync between `moduleExportsMap` (help.go), `StdlibExports` (introspect.go), and `StdlibModuleDescriptions` (introspect.go).

**Problem:**
- `moduleExportsMap` in `help.go` — hardcoded list of export names per module (already drifted once)
- `StdlibExports` in `introspect.go` — flat map of export name → metadata (causes collisions for shared names like `string`, `uuid`)
- `StdlibModuleDescriptions` / `BasilModuleDescriptions` — separate module description maps
- Adding a new export requires updating 3 places; forgetting one produces incorrect help output

**Design:**
1. New `ExportMeta` and `ModuleMeta` types in evaluator package
2. `StdlibModuleDict` gains a `Meta *ModuleMeta` field
3. Each module file defines a `var xxxModuleMeta = ModuleMeta{...}` alongside its loader
4. A registry (`GetModuleMeta`) provides metadata without instantiating the module (help system runs at CLI time without an Environment)
5. Help system and runtime introspection both use the new registry
6. Old maps (`StdlibExports`, `StdlibModuleDescriptions`, `BasilModuleDescriptions`, `moduleExportsMap`) are deleted

**Tasks:**

- [ ] **8.1** Create `pkg/parsley/evaluator/module_meta.go`
  - `ExportMeta` struct: `Kind`, `Arity`, `Description` (with JSON tags)
  - `ModuleMeta` struct: `Description`, `Exports map[string]ExportMeta`
  - `GetStdlibModuleMeta(name string) *ModuleMeta`
  - `GetBasilModuleMeta(name string) *ModuleMeta`
  - `GetStdlibModuleNames() []string`
  - `GetBasilModuleNames() []string`
  - Registry maps populated from per-module vars

- [ ] **8.2** Add `Meta *ModuleMeta` field to `StdlibModuleDict` (in `stdlib_table.go`)

- [ ] **8.3** Add metadata vars and attach to `StdlibModuleDict` in each module file:
  - `stdlib_math.go` — `mathModuleMeta` (38 exports)
  - `stdlib_id.go` — `idModuleMeta` (6 exports)
  - `stdlib_valid.go` — `validModuleMeta` (27 exports)
  - `stdlib_schema.go` — `schemaModuleMeta` (16 exports)
  - `stdlib_api.go` — `apiModuleMeta` (11 exports)
  - `stdlib_dev.go` — `devModuleMeta` (1 export)
  - `stdlib_table.go` — `tableModuleMeta` (1 export), `basilHTTPModuleMeta` (5 exports), `basilAuthModuleMeta` (3 exports)

- [ ] **8.4** Update `introspect.go` runtime consumers:
  - `inspectStdlibModule()` — use `mod.Meta.Exports[name]` instead of `StdlibExports[name]`
  - `describeStdlibModule()` — use `mod.Meta.Exports[name]` instead of `StdlibExports[name]`
  - `inspectStdlibRoot()` / `describeStdlibRoot()` — use `GetStdlibModuleMeta()`
  - `inspectBasilRoot()` / `describeBasilRoot()` — use `GetBasilModuleMeta()`
  - Delete `StdlibExports`, `StdlibModuleDescriptions`, `BasilModuleDescriptions`

- [ ] **8.5** Update `help/help.go`:
  - `describeModule()` — use `GetStdlibModuleMeta()` / `GetBasilModuleMeta()`
  - `getModuleExports()` — read from `ModuleMeta.Exports` (delete `moduleExportsMap`)
  - `getStdlibModuleNames()` / `getBasilModuleNames()` — use registry functions
  - `findSuggestions()` — use registry

- [ ] **8.6** Update tests, build, and lint

**Validation:**
- `pars describe @std/math` shows correct, module-scoped descriptions
- `pars describe @std/valid` shows `uuid` as "Check UUID format" (not id module's description)
- `pars describe @std/id` shows `uuid` as "Generate UUID v4" (not valid module's description)
- `pars describe --json @std/schema` outputs correct JSON with module-scoped metadata
- All existing help tests pass
- `go build` succeeds for both binaries
- `go test ./...` passes

## Deferred Items
Items to add to work/BACKLOG.md after implementation:
- `pars introspect --json` — dump entire language catalog for AI context windows
- Interactive help browser in REPL (fuzzy search across topics)
- Examples in help output (requires example metadata on methods/builtins)
- Terminal hyperlinks in help output (where supported)
- Search across all topics (`:describe --search format`)
- Terminal width auto-detection for REPL help formatting

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-02-11 | Phase 1: Operator Metadata | ✅ Complete | Added `OperatorInfo` struct and `OperatorMetadata` map with 22 operators across 8 categories |
| 2026-02-11 | Phase 2: Help Engine Core | ✅ Complete | Created `pkg/parsley/help/` with `help.go` (405 lines) and `format.go` (428 lines) |
| 2026-02-11 | Phase 3: CLI Integration | ✅ Complete | Added `describe` subcommand to `cmd/pars/main.go` with `--json` flag support |
| 2026-02-11 | Phase 4: REPL Integration | ✅ Complete | Added `:describe` and `:d` commands, updated `:help`, added tab completion words |
| 2026-02-11 | Phase 5: Tests | ✅ Complete | Created `help_test.go` with 17 test functions (514 lines), all passing |
| 2026-02-11 | Linter fixes | ✅ Complete | Fixed modernize and staticcheck warnings (fmt.Fprintf, slices.Contains, CutPrefix, deprecated strings.Title) |
| 2026-02-11 | Phase 6: JSON struct tags | ✅ Complete | Added `json` tags to `TopicResult`, `ExportEntry` (help.go), `MethodInfo`, `PropertyInfo`, `BuiltinInfo`, `OperatorInfo` (introspect.go). Keys now lowercase, `omitempty` suppresses null/empty fields. Updated test assertions. |
| 2026-02-11 | Phase 7: Module export accuracy | ✅ Complete | Audited all 7 modules against `load*Module()` source. Removed 18 phantom exports (cbrt, sinh, ulid, isbn, fromCSV, etc.). Added 50+ missing real exports (median, stddev, degrees, cuid, adminOnly, redirect, etc.). Added 31 new `StdlibExports` entries for id, schema, api, table, dev modules. Known limitation: shared names across modules show flat-map description (see Risk section). |
| | Phase 8: Self-describing modules | ✅ Complete | Created `module_meta.go` with `ExportMeta`/`ModuleMeta` types and registry. Added metadata vars to all 7 stdlib + 2 basil module files. Updated `introspect.go` and `help.go` to use registry. Deleted `StdlibExports`, `StdlibModuleDescriptions`, `BasilModuleDescriptions`, `moduleExportsMap`. Added `TestModuleScopedDescriptions` and `TestBasilModuleExports` tests (19 total help tests, all passing). Shared-name collision (e.g. `uuid` in id vs valid) is now resolved. |