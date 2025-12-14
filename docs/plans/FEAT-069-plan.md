---
id: PLAN-044
feature: FEAT-069
title: "Implementation Plan for Builtin Function Introspection"
status: draft
created: 2025-12-14
---

# Implementation Plan: FEAT-069 Builtin Function Introspection

## Overview
Add introspection metadata to all builtin functions, enabling `inspect()` and `describe()` to show documentation, parameters, and usage information.

## Prerequisites
- [ ] Review all existing builtins in `getBuiltins()`
- [ ] Decide on categorization scheme

## Tasks

### Task 1: Define BuiltinInfo Structure
**Files**: `pkg/parsley/evaluator/introspect.go`
**Estimated effort**: Small

Create the metadata structure for builtin functions.

Steps:
1. Add `BuiltinInfo` struct (similar to `MethodInfo` but with `Params` and `Category`)
2. Place near `MethodInfo` definition for consistency

Code:
```go
// BuiltinInfo holds metadata about a builtin function
type BuiltinInfo struct {
    Name        string
    Arity       string   // e.g., "1", "1-2", "0+", "1+"
    Description string
    Params      []string // Parameter names, "?" suffix for optional
    Category    string   // Grouping: "file", "time", "conversion", etc.
    Deprecated  string   // If non-empty, deprecation message
}
```

Tests:
- Structure compiles correctly

---

### Task 2: Create BuiltinMetadata Map
**Files**: `pkg/parsley/evaluator/introspect.go`
**Estimated effort**: Medium

Document all ~50 builtin functions.

Steps:
1. Create `var BuiltinMetadata = map[string]BuiltinInfo{...}`
2. Add entries for each builtin in `getBuiltins()`
3. Group by category for readability

Categories and functions:
```go
var BuiltinMetadata = map[string]BuiltinInfo{
    // === File/Data Loading ===
    "JSON":     {Name: "JSON", Arity: "1-2", Description: "Load JSON from path or URL", Params: []string{"source", "options?"}, Category: "file"},
    "YAML":     {Name: "YAML", Arity: "1-2", Description: "Load YAML from path or URL", Params: []string{"source", "options?"}, Category: "file"},
    "CSV":      {Name: "CSV", Arity: "1-2", Description: "Load CSV from path or URL", Params: []string{"source", "options?"}, Category: "file"},
    "markdown": {Name: "markdown", Arity: "1-2", Description: "Load markdown file with frontmatter", Params: []string{"path", "options?"}, Category: "file"},
    "lines":    {Name: "lines", Arity: "1-2", Description: "Load file as array of lines", Params: []string{"source", "options?"}, Category: "file"},
    "bytes":    {Name: "bytes", Arity: "1", Description: "Load file as byte array", Params: []string{"path"}, Category: "file"},
    "file":     {Name: "file", Arity: "1-2", Description: "Load file with auto-detected format", Params: []string{"path", "options?"}, Category: "file"},
    "dir":      {Name: "dir", Arity: "1", Description: "List directory contents", Params: []string{"path"}, Category: "file"},
    
    // === Time ===
    "time": {Name: "time", Arity: "1-2", Description: "Create time from string, timestamp, or dict", Params: []string{"input", "delta?"}, Category: "time"},
    "now":  {Name: "now", Arity: "0", Description: "Current datetime", Params: []string{}, Category: "time", Deprecated: "Use @now instead"},
    
    // === URLs ===
    "url": {Name: "url", Arity: "1", Description: "Parse URL string into components", Params: []string{"urlString"}, Category: "url"},
    
    // === Type Conversion ===
    "string": {Name: "string", Arity: "1", Description: "Convert value to string", Params: []string{"value"}, Category: "conversion"},
    "int":    {Name: "int", Arity: "1", Description: "Convert value to integer", Params: []string{"value"}, Category: "conversion"},
    "float":  {Name: "float", Arity: "1", Description: "Convert value to float", Params: []string{"value"}, Category: "conversion"},
    "bool":   {Name: "bool", Arity: "1", Description: "Convert value to boolean", Params: []string{"value"}, Category: "conversion"},
    
    // === Type Info ===
    "type": {Name: "type", Arity: "1", Description: "Get type name of value", Params: []string{"value"}, Category: "info"},
    "len":  {Name: "len", Arity: "1", Description: "Get length of string, array, or dict", Params: []string{"value"}, Category: "info"},
    
    // === Output ===
    "print":   {Name: "print", Arity: "1+", Description: "Print values without newline", Params: []string{"values..."}, Category: "output"},
    "println": {Name: "println", Arity: "0+", Description: "Print values with newline", Params: []string{"values..."}, Category: "output"},
    "debug":   {Name: "debug", Arity: "1+", Description: "Print debug output", Params: []string{"values..."}, Category: "output"},
    "log":     {Name: "log", Arity: "1+", Description: "Log message", Params: []string{"values..."}, Category: "output"},
    "error":   {Name: "error", Arity: "1+", Description: "Log error message", Params: []string{"values..."}, Category: "output"},
    "warn":    {Name: "warn", Arity: "1+", Description: "Log warning message", Params: []string{"values..."}, Category: "output"},
    
    // === Control Flow ===
    "throw":  {Name: "throw", Arity: "1", Description: "Throw an error", Params: []string{"message"}, Category: "control"},
    "assert": {Name: "assert", Arity: "1-2", Description: "Assert condition is true", Params: []string{"condition", "message?"}, Category: "control"},
    "exit":   {Name: "exit", Arity: "0-1", Description: "Exit program", Params: []string{"code?"}, Category: "control"},
    
    // === Introspection ===
    "inspect":  {Name: "inspect", Arity: "1", Description: "Get introspection data as dictionary", Params: []string{"value"}, Category: "introspection"},
    "describe": {Name: "describe", Arity: "1", Description: "Get formatted help text", Params: []string{"value"}, Category: "introspection"},
    "keys":     {Name: "keys", Arity: "1", Description: "Get dictionary keys", Params: []string{"dict"}, Category: "introspection"},
    "values":   {Name: "values", Arity: "1", Description: "Get dictionary values", Params: []string{"dict"}, Category: "introspection"},
    "entries":  {Name: "entries", Arity: "1", Description: "Get dictionary [key, value] pairs", Params: []string{"dict"}, Category: "introspection"},
    
    // === Import ===
    "import": {Name: "import", Arity: "1", Description: "Import module or file", Params: []string{"path"}, Category: "module"},
    
    // ... etc
}
```

Tests:
- All builtins in `getBuiltins()` have corresponding metadata entries

---

### Task 3: Update inspect() for Builtins
**Files**: `pkg/parsley/evaluator/introspect.go`
**Estimated effort**: Medium

Make `inspect()` return rich data for builtin functions.

Steps:
1. In `builtinInspect()`, detect when argument is a `*Builtin`
2. Look up metadata from `BuiltinMetadata` map
3. Return dictionary with: name, arity, description, params, category, deprecated

Expected output:
```parsley
inspect(JSON)
// Returns:
{
  type: "builtin",
  name: "JSON",
  arity: "1-2",
  description: "Load JSON from path or URL",
  params: ["source", "options?"],
  category: "file"
}
```

Tests:
- `inspect(JSON).name` returns "JSON"
- `inspect(JSON).params` returns ["source", "options?"]
- `inspect(now).deprecated` returns deprecation message

---

### Task 4: Update describe() for Builtins
**Files**: `pkg/parsley/evaluator/introspect.go`
**Estimated effort**: Small

Make `describe()` output formatted help for builtins.

Steps:
1. In `builtinDescribe()`, detect when argument is a `*Builtin`
2. Look up metadata and format nicely
3. Include deprecation warning if applicable

Expected output:
```
describe(time)
// Returns:
"time(input, delta?) - Create time from string, timestamp, or dict

Parameters:
  input   - Time source (string, integer timestamp, or dict)
  delta?  - Optional: adjustment to apply {days: 1, hours: -2}

Category: time"
```

Tests:
- `describe(JSON)` includes parameter list
- `describe(now)` shows deprecation warning

---

### Task 5: Add Builtin Listing Function
**Files**: `pkg/parsley/evaluator/introspect.go`, `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Add way to list all available builtins with descriptions.

Options:
1. `builtins()` function returning array
2. `describe(builtins)` special form
3. `inspect(@builtins)` or similar

Steps:
1. Choose approach (recommend option 1: `builtins()` function)
2. Return array of builtin info dictionaries, sorted by category then name
3. Filter by category optionally: `builtins("file")`

Tests:
- `builtins()` returns array with all builtins
- `builtins("file")` returns only file-related builtins

---

### Task 6: Add Tests
**Files**: `pkg/parsley/tests/introspect_test.go` (new or existing)
**Estimated effort**: Medium

Comprehensive tests for builtin introspection.

Test cases:
```go
// inspect returns correct structure
{input: `inspect(JSON).type`, expected: "builtin"}
{input: `inspect(JSON).name`, expected: "JSON"}
{input: `inspect(JSON).arity`, expected: "1-2"}

// describe formats correctly  
{input: `describe(time).includes("input")`, expected: true}

// deprecated functions marked
{input: `inspect(now).deprecated`, notEmpty: true}

// builtins() list
{input: `builtins().length() > 30`, expected: true}
{input: `builtins("file").every(fn(b){b.category == "file"})`, expected: true}
```

---

### Task 7: Documentation
**Files**: `docs/parsley/reference.md`
**Estimated effort**: Small

Document the introspection features.

Steps:
1. Add section on `inspect()` for builtins
2. Add section on `describe()` for builtins
3. Add section on `builtins()` function
4. Add examples

---

## Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `go build -o basil ./cmd/basil`
- [ ] Linter passes: `golangci-lint run`
- [ ] Every builtin in `getBuiltins()` has metadata
- [ ] Documentation updated
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| | | | |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- REPL tab completion showing function signatures
- LSP hover documentation for builtins
- Auto-generate API reference from metadata
- Connection literal metadata (`@sqlite`, `@postgres`)

---

## Ongoing Maintenance

### When Adding New Builtins
**Every time a builtin is added** to `getBuiltins()`, immediately add corresponding entry to `BuiltinMetadata`:

```go
"new_function": {
    Name:        "new_function",
    Arity:       "1-2",  // Actual arity
    Description: "Clear description of what it does",
    Params:      []string{"param1", "param2?"},  // ? for optional
    Category:    "appropriate-category",
},
```

### Periodic Audit (Monthly or When Touching Builtins)
Run this checklist to ensure introspection data stays synchronized:

1. **Completeness check**:
   ```bash
   # Count builtins in getBuiltins()
   grep -c '".*": {$' pkg/parsley/evaluator/evaluator.go
   
   # Count entries in BuiltinMetadata
   grep -c '".*": {Name:' pkg/parsley/evaluator/introspect.go
   
   # Numbers should match (minus internal-only functions like 'import')
   ```

2. **Spot check** random builtins:
   - Verify arity matches implementation
   - Confirm parameter names are accurate
   - Check category is appropriate

3. **Test introspection**:
   ```go
   func TestAllBuiltinsHaveMetadata(t *testing.T) {
       builtins := getBuiltins()
       for name := range builtins {
           if name == "import" { continue } // Skip internal
           metadata, exists := BuiltinMetadata[name]
           if !exists {
               t.Errorf("Missing metadata for builtin: %s", name)
           }
           // Verify metadata fields are populated...
       }
   }
   ```

### Deprecation Process
When deprecating a builtin:
1. Add `Deprecated: "Use X instead"` to metadata
2. Keep the function working for backwards compatibility
3. Document in CHANGELOG under deprecation section
4. Remove in major version bump

**Reference**: See `.github/instructions/code.instructions.md` for integration with development workflow.
