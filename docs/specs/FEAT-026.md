# FEAT-026: File Module and Global Cleanup

**Status:** Proposed  
**Created:** 2025-12-04  
**Author:** AI Assistant  
**Depends on:** None

## Summary

Reorganize file handle factories from global builtins into a cohesive `std/file` module, and remove the `basil` global in favor of `std/basil` import. This is a pre-1.0 cleanup to establish consistent module-based APIs.

## Motivation

Currently, file operations use a mix of inconsistent globals:

```parsley
// Current (inconsistent)
let f = file(@./data.json)  // Lowercase
let f = jsonFile(@./data.json)  // Uppercase! Why?
let f = csvFile(@./data.csv)    // Uppercase
let f = yamlFile(@./config.yml) // Uppercase
```

Similarly, `basil` exists as both a global AND a module:

```parsley
// Current (confusing)
basil.version           // Global access
let {basil} = import("std/basil")  // Also works!
```

Problems:
1. **Naming inconsistency**: `file()` is lowercase, format-specific factories are UPPERCASE
2. **Global pollution**: 10+ globals for file operations, plus `basil`
3. **Discoverability**: No obvious namespace to explore
4. **Duplicate APIs**: `basil` global duplicates `std/basil` module
5. **No callable module pattern**: Unlike other std modules

## Terminology

**Important distinction:**

- **File handle** (or "factory"): An object describing a file and how to parse it
- **Reading**: The `<==` operator actually reads the file and parses its contents

The `file` module methods **create handles**. They do NOT read files:

```parsley
// Creates a handle (no I/O yet)
let handle = file.json(@./data.json)

// Reading happens here (actual I/O)
let data <== handle
```

## Proposed Design

### Module Import

```parsley
let {file} = import("std/file")
```

### Creating File Handles

All methods create file handles. The format determines how `<==` will parse the content:

| Method | Creates Handle For | `<==` Returns |
|--------|-------------------|---------------|
| `file(@path)` | Auto-detect from extension | Varies |
| `file.json(@path)` | JSON file | Dictionary/Array |
| `file.yaml(@path)` | YAML file | Dictionary/Array |
| `file.csv(@path)` | CSV file | Table |
| `file.textFile(@path)` | Plain text | String |
| `file.linesFile(@path)` | Line-delimited text | Array of strings |
| `file.bytesFile(@path)` | Binary file | Byte array |
| `file.md(@path)` | Markdown file | HTML string |
| `file.svg(@path)` | SVG file | HTML element |

### Directory Operations

```parsley
file.dir(@./folder)        // List directory contents
file.glob(@./src/**/*.go)  // Glob pattern matching
```

### Callable Module Pattern

The `file` module is **callable**, acting as shorthand for auto-detected file handles:

```parsley
file(@./data.json)      // Same as file.json() due to extension
file.json(@./data.json) // Explicit format
```

Calling `file()` directly uses extension-based format detection.

### Complete Examples

```parsley
let {file} = import("std/file")

// Create handles (no I/O)
let jsonHandle = file.json(@./config.json)
let csvHandle = file.csv(@./data.csv)
let textHandle = file.textFile(@./readme.txt)

// Read files (I/O happens here)
let config <== jsonHandle
let data <== csvHandle  
let readme <== textHandle

// One-liner pattern (most common)
let config <== file.json(@./config.json)
let users <== file.csv(@./users.csv)

// Auto-detect from extension
let config <== file(@./config.json)  // Detects JSON
let data <== file(@./data.csv)       // Detects CSV

// Directory listing
let files <== file.dir(@./src)
let goFiles <== file.glob(@./src/**/*.go)
```

### Format Auto-Detection

When using `file(@path)`, the format is inferred from the file extension:

| Extension | Format |
|-----------|--------|
| `.json` | JSON |
| `.yaml`, `.yml` | YAML |
| `.csv` | CSV |
| `.txt` | Text |
| `.md`, `.markdown` | Markdown |
| `.svg` | SVG |
| *(other)* | Text (default) |

## Migration

### Removed Globals

These globals will be **removed** (hard removal, pre-1.0):

| Old Global | New Method |
|------------|------------|
| `file()` | `file()` (via module) |
| `jsonFile()` | `file.json()` |
| `yamlFile()` | `file.yaml()` |
| `csvFile()` | `file.csv()` |
| `textFile()` | `file.textFile()` |
| `linesFile()` | `file.linesFile()` |
| `bytesFile()` | `file.bytesFile()` |
| `markdownFile()` | `file.md()` |
| `svgFile()` | `file.svg()` |
| `dir()` | `file.dir()` |
| `glob()` | `file.glob()` |

### Migration Pattern

```parsley
// Before
let data <== jsonFile(@./data.json)

// After
let {file} = import("std/file")
let data <== file.json(@./data.json)
```

## Basil Global Removal

### Current State

The `basil` global provides access to request context in handlers:

```parsley
// Current (global)
basil.version
basil.request.path
basil.request.method
```

### New Pattern

Require explicit import:

```parsley
// After
let {basil} = import("std/basil")
basil.version
basil.request.path
```

### Rationale

1. **Consistency**: All std modules use import pattern
2. **Explicitness**: Clear where `basil` comes from
3. **Testing**: Easier to mock/stub when imported
4. **No magic**: Handlers are just Parsley files, no special globals

## Implementation Notes

### Callable Module

The module implements `__call` method to support `file(@path)` syntax:

```go
// When module is called as function, delegate to auto-detect
func (m *FileModule) Call(args ...Object) Object {
    return m.autoDetect(args...)
}
```

### File Handle Structure

File handles remain dictionaries with `__type: "file"`:

```parsley
{
    __type: "file",
    path: "/absolute/path/to/file.json",
    format: "json",
    // ... other metadata
}
```

### Integration Points

- `pkg/parsley/evaluator/modules/file.go` - New module implementation
- `pkg/parsley/evaluator/builtins.go` - Remove old globals
- `pkg/parsley/evaluator/evaluator.go` - Handle callable modules

## Resolved Questions

1. **`read` vs `open`?** → Neither in method names; methods create handles, `<==` reads
2. **Callable module?** → Yes, `file(@path)` delegates to auto-detect
3. **Deprecation period?** → No, hard removal (pre-1.0)
4. **Keep global `file()`?** → No, removes confusion with module name
5. **Keep `basil` global?** → No, require `import("std/basil")`

## Future Work

- **Write operations**: `file.write(@path, content)` or `==>` operator
- **Streaming**: Large file handling without full memory load
- **File watching**: `file.watch(@path)` for live reload scenarios

## Related

- File handle implementation in `evaluator.go`
- `<==` operator semantics
- Other std modules (`std/table`, `std/http`, etc.)
