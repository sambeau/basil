---
id: man-pars-paths
title: Paths
system: parsley
type: builtins
name: paths
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - path
  - file path
  - directory
  - path literal
  - interpolated path
  - segments
  - extension
  - parent
  - project root
---

# Paths

Path values represent filesystem paths. They are first-class objects (not strings) with properties and methods for manipulation. Paths are created from literals prefixed with `@`.

## Literals

```parsley
@./config.json                   // relative to current file
@~/lib/utils.pars                // relative to project root
```

| Prefix | Resolves relative to |
|---|---|
| `@./` | Current file's directory |
| `@~/` | Project root |

> ⚠️ `@~/` means the **project root**, not the user's home directory. This is different from shell conventions.

## Interpolated Paths

Use `@(...)` with `{expr}` placeholders for dynamic paths:

```parsley
let name = "config"
@(./data/{name}.json)            // ./data/config.json

let id = 42
@(./users/{id}/profile.json)     // ./users/42/profile.json
```

## path() Builtin

Create a path from a string — useful when the path is fully dynamic:

```parsley
let p = path("./relative/path")
```

Prefer literals for static paths — they're checked at parse time.

## Properties

| Property | Type | Description |
|---|---|---|
| `.segments` | array | Path segments as array of strings |
| `.filename` | string | Last segment (file or directory name) |
| `.extension` | string | File extension (without leading dot) |
| `.stem` | string | Filename without extension |
| `.parent` | path | Parent directory as a path value |
| `.absolute` | boolean | Whether the path is absolute |
| `.suffixes` | array | All extensions as array (e.g., `["tar", "gz"]`) |

```parsley
let p = @./users/123/profile.json
p.filename                       // "profile.json"
p.extension                      // "json"
p.stem                           // "profile"
p.segments                       // [".", "users", "123", "profile.json"]
p.parent.filename                // "123"
```

## Methods

### .isAbsolute() / .isRelative()

```parsley
let rel = @./config.json
rel.isAbsolute()                 // false
rel.isRelative()                 // true
```

### .match(pattern)

Match against a route-style pattern. Returns a dictionary of captures or `null`:

```parsley
let p = @./users/123/profile.json
p.match("/users/:id/:file")      // {id: "123", file: "profile.json"}
p.match("/products/:id")         // null
```

Pattern syntax: `:param` matches a single segment, `*splat` matches multiple.

### .toURL(prefix)

Convert to a URL string with a prefix:

```parsley
let p = @./images/logo.png
p.toURL("https://cdn.example.com")
// "https://cdn.example.com/images/logo.png"
```

### .public()

Get the public web-serving URL for this path:

```parsley
let p = @./assets/style.css
p.public()                       // "/assets/style.css"
```

### .toDict() / .inspect()

```parsley
let p = @./config.json
p.toDict()                       // {segments: [".", "config.json"], absolute: false}
p.inspect()                      // includes __type: "path"
```

## Path Arithmetic

Use `+` or `/` to join path segments:

```parsley
let base = @./data
let full = base + "users.json"
// ./data/users.json
```

## Paths as File Handle Sources

Paths are the primary argument to file handle constructors:

```parsley
let data <== JSON(@./config.json)
let lines <== lines(@./todo.txt)
"output" ==> text(@./result.txt)
```

See [File I/O](../features/file-io.md) for the full file operations reference.

## Paths in Import Statements

Path literals are used for module imports:

```parsley
import @./utils.pars             // relative import
import @~/lib/helpers.pars       // project root import
import @std/math                 // stdlib (not a filesystem path)
```

See [Modules](../fundamentals/modules.md) for import details.

## Key Differences from Other Languages

- **Paths are objects, not strings** — they have typed properties (`.extension`, `.parent`, `.segments`) and methods. Use `.string` or string conversion to get the string representation.
- **`@~/` is project root, not home directory** — this is the most common point of confusion. There is no shorthand for the user's home directory.
- **Interpolation uses `{expr}`** — `@(./data/{name}.json)`, not template string syntax.
- **No path separator concerns** — Parsley handles forward/backward slashes internally. Always use forward slashes in literals.

## See Also

- [File I/O](../features/file-io.md) — reading and writing files using path handles
- [URLs](urls.md) — URL literals and manipulation
- [Modules](../fundamentals/modules.md) — using paths in import statements
- [Operators](../fundamentals/operators.md) — `+` path joining operator