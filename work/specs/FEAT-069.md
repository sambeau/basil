---
id: FEAT-069
title: "Introspection Metadata for Builtin Functions"
status: draft
priority: medium
created: 2025-12-14
author: "@sambeau"
---

# FEAT-069: Introspection Metadata for Builtin Functions

## Summary
Add introspection metadata (arity, description, parameter names) to builtin functions so they have the same level of documentation as methods on primitive types. Currently, `inspect(someString)` shows detailed method info, but `inspect(JSON)` or `inspect(time)` provides minimal information.

## User Story
As a developer, I want to see documentation for builtin functions using `inspect()` and `describe()` so that I can discover available functions and understand their parameters without leaving the REPL or editor.

## Acceptance Criteria
- [ ] `inspect(JSON)` returns `{type: "builtin", name: "JSON", arity: "1-2", description: "Load JSON from path", params: ["path", "options?"]}`
- [ ] `describe(time)` shows formatted help: `time(input, delta?) - Create time from string, timestamp, or dict`
- [ ] All ~50 builtin functions have metadata (name, arity, description, params)
- [ ] Metadata stored separately from function implementations (maintainable)
- [ ] `inspect(@std)` or similar lists all builtins with descriptions
- [ ] Tab completion in REPL shows function signatures (future enhancement)

## Design Decisions
- **Separate metadata**: Store `BuiltinInfo` map separately from `getBuiltins()` function implementations. This keeps the code clean and makes it easy to add/update documentation.
- **Consistent structure**: Use same `MethodInfo` structure or similar for builtins, enabling consistent introspection API.
- **Params array**: Include parameter names for better documentation (e.g., `["path", "options?"]`).

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Current State
- `getBuiltins()` in evaluator.go returns `map[string]*Builtin`
- `Builtin` struct only has `Fn` and `FnWithEnv` fields - no metadata
- `TypeMethods` map provides `MethodInfo` for type methods (name, arity, description)
- `StdlibExports` map provides `MethodInfo` for stdlib module exports
- `inspect()` and `describe()` use these maps for documentation

### Proposed Structure
```go
// BuiltinInfo holds metadata about a builtin function
type BuiltinInfo struct {
    Name        string
    Arity       string   // e.g., "1", "1-2", "0+"
    Description string
    Params      []string // e.g., ["path", "options?"]
    Category    string   // e.g., "file", "time", "conversion"
}

// BuiltinMetadata maps builtin names to their metadata
var BuiltinMetadata = map[string]BuiltinInfo{
    "JSON":     {Name: "JSON", Arity: "1-2", Description: "Load JSON from path or URL", Params: []string{"source", "options?"}, Category: "file"},
    "time":     {Name: "time", Arity: "1-2", Description: "Create time from string, timestamp, or dict", Params: []string{"input", "delta?"}, Category: "time"},
    "url":      {Name: "url", Arity: "1", Description: "Parse URL string into components", Params: []string{"urlString"}, Category: "conversion"},
    // ... etc
}
```

### Affected Components
- `pkg/parsley/evaluator/introspect.go` — Add `BuiltinInfo` type and `BuiltinMetadata` map
- `pkg/parsley/evaluator/introspect.go` — Update `builtinInspect()` to handle builtin functions
- `pkg/parsley/evaluator/introspect.go` — Update `builtinDescribe()` for formatted output

### Builtins to Document (~50)
**File/Data Loading:**
- `JSON`, `YAML`, `CSV`, `markdown`, `lines`, `bytes`, `file`, `dir`

**Time:**
- `time`, `now` (deprecated)

**URLs:**
- `url`

**Conversion:**
- `string`, `int`, `float`, `bool`, `type`, `len`

**Output:**
- `print`, `println`, `debug`, `log`, `error`, `warn`

**Control:**
- `throw`, `assert`, `exit`

**Introspection:**
- `inspect`, `describe`, `keys`, `values`, `entries`

**Template:**
- `render`, `partial`

**Database:**
- Connection builtins (`@sqlite`, `@postgres`, etc.)

### Edge Cases & Constraints
1. **Connection literals** — `@sqlite`, `@postgres` are special; may need separate handling
2. **Deprecated functions** — `now()` is deprecated; show deprecation in metadata
3. **Variadic functions** — Arity like `1+` for functions accepting multiple args

## Implementation Notes
*Added during/after implementation*

## Related
- Plan: `docs/plans/FEAT-069-plan.md`
