# Code Analysis Report: Dead Code & Duplicates

**Date:** January 2025  
**Tools Used:** `deadcode` (golang.org/x/tools), `dupl` (github.com/mibk/dupl)  
**Scope:** Full repository analysis

## Executive Summary

| Category | Count | Recommendation |
|----------|-------|----------------|
| Dead code - Safe to delete | ~15 functions | Delete immediately |
| Dead code - Incomplete features | ~25 functions | Keep for future work |
| Dead code - Entry point issue | ~30 functions | Fix wiring, not delete |
| Dead code - Test helpers | ~50 functions | Review case-by-case |
| Duplicate code clones | ~60 groups | ~10 worth deduplicating |

---

## Part 1: Dead Code Analysis

### 1.1 Safe to Delete (No Spec Reference, Unused)

These functions have no spec reference and appear truly unused:

| Function | File | Notes |
|----------|------|-------|
| `makeRelativePath` | server/errors.go | Superseded by other path handling |
| `makeMessageRelative` | server/errors.go | Superseded |
| `improveErrorMessage` | server/errors.go | Old error formatting |
| `renderDevErrorPage` | server/errors.go | Superseded by dev_error.pars template |
| `getSourceContext` | server/errors.go | Superseded by `extractSourceContext` |
| `highlightParsley` | server/errors.go | Old HTML highlighting |
| `escapeForCodeDisplay` | server/errors.go | Superseded |

**Recommendation:** Delete these 7 functions from `server/errors.go`. They were part of the old HTML error page system replaced by the Parsley template in FEAT-006/FEAT-102.

### 1.2 Entry Point Issue - Active Features (Don't Delete)

These are documented in specs but deadcode reports them unreachable. The functions ARE used - they're called from Parsley via the stdlib bridge, not from Go code.

#### Session Methods (FEAT-049)

| Function | File | Spec |
|----------|------|------|
| `Get` | server/session.go | FEAT-049 |
| `Set` | server/session.go | FEAT-049 |
| `Delete` | server/session.go | FEAT-049 |
| `Clear` | server/session.go | FEAT-049 |
| `All` | server/session.go | FEAT-049 |
| `Has` | server/session.go | FEAT-049 |
| `Flash` | server/session.go | FEAT-049 |
| `GetFlash` | server/session.go | FEAT-049 |
| `GetAllFlash` | server/session.go | FEAT-049 |
| `HasFlash` | server/session.go | FEAT-049 |
| `Regenerate` | server/session.go | FEAT-049 |
| `IsDirty` | server/session.go | FEAT-049 |

**Recommendation:** KEEP. These are exposed to Parsley as `basil.session.*` methods. Deadcode can't trace dynamic dispatch.

#### Format Package (FEAT-090, FEAT-100)

| Function | File | Spec |
|----------|------|------|
| `FormatValue` | pkg/parsley/format/format.go | FEAT-100 |
| `FormatValueWithOpts` | pkg/parsley/format/format.go | FEAT-100 |
| `Printer.*` methods | pkg/parsley/format/printer.go | FEAT-100 |

**Recommendation:** KEEP. These are exposed as `@std/format` or via the pretty-printer. Verify with actual usage.

#### PLN Package (FEAT-098)

| Function | File | Spec |
|----------|------|------|
| `Serialize` | pkg/parsley/pln/serializer.go | FEAT-098 |
| `SerializeWithEnv` | pkg/parsley/pln/serializer.go | FEAT-098 |
| `SerializePretty` | pkg/parsley/pln/serializer.go | FEAT-098 |
| `Parse` | pkg/parsley/pln/parser.go | FEAT-098 |
| `MustParse` | pkg/parsley/pln/parser.go | FEAT-098 |
| `Validate` | pkg/parsley/pln/validator.go | FEAT-098 |

**Recommendation:** KEEP. These form the PLN module API.

### 1.3 Incomplete Features (Keep for Future Work)

These are implemented but not yet wired into the stdlib:

#### Markdown Helpers (No Spec - Candidate for FEAT-XXX)

| Function | File | Purpose |
|----------|------|---------|
| `markdownFindAll` | markdown_helpers.go | Find nodes by type |
| `markdownFindFirst` | markdown_helpers.go | Find first node |
| `markdownHeadings` | markdown_helpers.go | Extract headings |
| `markdownLinks` | markdown_helpers.go | Extract links |
| `markdownImages` | markdown_helpers.go | Extract images |
| `markdownCodeBlocks` | markdown_helpers.go | Extract code blocks |
| `markdownTitle` | markdown_helpers.go | Get document title |
| `markdownTOC` | markdown_helpers.go | Generate TOC |
| `markdownText` | markdown_helpers.go | Extract plain text |
| `markdownWordCount` | markdown_helpers.go | Count words |
| `markdownWalk` | markdown_helpers.go | Walk AST |
| `markdownMap` | markdown_helpers.go | Map over AST |
| `markdownFilter` | markdown_helpers.go | Filter AST |

**Recommendation:** KEEP. Add to BACKLOG.md - these should be exposed via `@std/markdown` module (FEAT-072 or new spec).

### 1.4 Test Helpers (~50 functions)

Deadcode found many test helper functions unreachable. These are false positives - test files are not analyzed for reachability.

**Recommendation:** Ignore. Test helpers are fine.

---

## Part 2: Duplicate Code Analysis

### 2.1 High-Value Deduplication Opportunities

#### 2.1.1 `tableMin` / `tableMax` (stdlib_table.go:1093-1160)

**Duplication:** Two 35-line functions that differ only in comparison operator (`<` vs `>`).

```go
// Current: Two nearly identical functions
func tableMin(args []Object, env *Environment) Object { ... }
func tableMax(args []Object, env *Environment) Object { ... }
```

**Suggested Fix:**
```go
func tableExtreme(args []Object, env *Environment, isMin bool) Object {
    compare := func(a, b float64) bool {
        if isMin { return a < b }
        return a > b
    }
    // ... shared logic using compare
}

func tableMin(args []Object, env *Environment) Object {
    return tableExtreme(args, env, true)
}

func tableMax(args []Object, env *Environment) Object {
    return tableExtreme(args, env, false)
}
```

**Lines saved:** ~30 lines  
**Risk:** Low

---

#### 2.1.2 `ServeCSS` / `ServeJS` (server/bundle.go:219-295)

**Duplication:** Two 40-line functions that differ only in content-type header.

**Suggested Fix:**
```go
func serveBundle(w http.ResponseWriter, r *http.Request, contentType, content, hash string) {
    // shared logic
}

func ServeCSS(w http.ResponseWriter, r *http.Request, content, hash string) {
    serveBundle(w, r, "text/css; charset=utf-8", content, hash)
}

func ServeJS(w http.ResponseWriter, r *http.Request, content, hash string) {
    serveBundle(w, r, "application/javascript; charset=utf-8", content, hash)
}
```

**Lines saved:** ~35 lines  
**Risk:** Low

---

#### 2.1.3 `isDatetimeDict` / `isPathDict` / `isURLDict` (pln/serializer.go:545-600)

**Duplication:** Three identical functions checking for typed dictionaries.

**Suggested Fix:**
```go
func isTypedDict(obj Object, typeName string) bool {
    dict, ok := obj.(*Dictionary)
    if !ok { return false }
    typeVal, hasType := dict.Get("@type")
    if !hasType { return false }
    typeStr, ok := typeVal.(*String)
    return ok && typeStr.Value == typeName
}

func isDatetimeDict(obj Object) bool { return isTypedDict(obj, "datetime") }
func isPathDict(obj Object) bool    { return isTypedDict(obj, "path") }
func isURLDict(obj Object) bool     { return isTypedDict(obj, "url") }
```

**Lines saved:** ~40 lines  
**Risk:** Low

---

#### 2.1.4 `evalDirComputedProperty` / `evalFileComputedProperty` (eval_computed_properties.go:240-360)

**Duplication:** Large switch statements with shared cases (name, stem, ext, parent, etc.)

**Current pattern:**
```go
func evalDirComputedProperty(...) { switch prop { case "name": ... case "parent": ... } }
func evalFileComputedProperty(...) { switch prop { case "name": ... case "parent": ... } }
```

**Suggested Fix:** Extract shared path property helper:
```go
func evalPathProperty(pathStr, prop string) (Object, bool) {
    switch prop {
    case "name": return &String{Value: filepath.Base(pathStr)}, true
    case "parent": return &String{Value: filepath.Dir(pathStr)}, true
    case "stem": return &String{Value: strings.TrimSuffix(filepath.Base(pathStr), filepath.Ext(pathStr))}, true
    case "ext": return &String{Value: filepath.Ext(pathStr)}, true
    // ... other shared cases
    default: return nil, false
    }
}
```

**Lines saved:** ~60 lines  
**Risk:** Medium (needs careful testing)

---

### 2.2 Low-Value Duplications (Don't Refactor)

These duplicates were found but are not worth refactoring:

| Location | Reason to Keep |
|----------|----------------|
| Test assertion patterns | Tests should be explicit |
| Error message builders | Clarity over DRY |
| Small switch statements | Readability matters |
| HTTP handler boilerplate | Go idiom |

---

## Part 3: Recommendations Summary

### Immediate Actions (Low Risk)

1. **Delete 7 obsolete functions** from `server/errors.go`:
   - `makeRelativePath`, `makeMessageRelative`, `improveErrorMessage`
   - `renderDevErrorPage`, `getSourceContext`, `highlightParsley`, `escapeForCodeDisplay`

2. **Deduplicate** `tableMin`/`tableMax` in `stdlib_table.go`

3. **Deduplicate** `ServeCSS`/`ServeJS` in `server/bundle.go`

4. **Deduplicate** `isDatetimeDict`/`isPathDict`/`isURLDict` in `pln/serializer.go`

### Backlog Items

1. **Add markdown helpers to stdlib** - Create spec for `@std/markdown` module exposing:
   - `markdown.findAll()`, `markdown.headings()`, `markdown.links()`, etc.

2. **Consider** deduplicating path computed properties (medium risk)

### Do Not Delete

- Session methods (FEAT-049) - used via Parsley
- Format package (FEAT-100) - public API
- PLN package (FEAT-098) - public API
- Test helpers - false positives

---

## Appendix: Raw Tool Output

### Deadcode Summary (Top 20)

```
server/errors.go: FromParsleyError, makeRelativePath, makeMessageRelative, 
                  improveErrorMessage, renderDevErrorPage, getSourceContext,
                  highlightParsley, escapeForCodeDisplay
server/session.go: Get, Set, Delete, Clear, All, Has, Flash, GetFlash, 
                   GetAllFlash, HasFlash, Regenerate, IsDirty
pkg/parsley/format/format.go: FormatValue, FormatValueWithOpts
pkg/parsley/format/printer.go: Printer methods
pkg/parsley/pln/*.go: Serialize, Parse, Validate, etc.
pkg/parsley/evaluator/markdown_helpers.go: 13 markdown* functions
```

### Dupl Summary (Top Clone Groups)

```
Clone 1: stdlib_table.go tableMin/tableMax (68 lines)
Clone 2: bundle.go ServeCSS/ServeJS (76 lines)  
Clone 3: eval_computed_properties.go dir/file props (120 lines)
Clone 4: pln/serializer.go isTypedDict variants (45 lines)
```
