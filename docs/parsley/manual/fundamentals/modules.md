---
id: man-pars-modules
title: Modules
system: parsley
type: fundamentals
name: modules
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - import
  - export
  - module
  - computed export
  - standard library
  - stdlib
---

# Modules

Parsley modules are `.pars` files. Any file can import another, and only `export`ed values are visible to the importer. Non-exported values remain private. Modules are evaluated once and cached — subsequent imports return the same dictionary.

## Importing

`import` is an expression that returns a dictionary of a module's exports. There are three ways to use it:

### Namespace Import

Import the whole module as a named dictionary. The name is derived from the path automatically:

```parsley
import @std/math
math.floor(3.7)                  // 3
math.ceil(3.2)                   // 4
```

### Aliased Import

Rename the module with `as`:

```parsley
import @std/math as M
M.floor(3.7)                    // 3
```

### Destructured Import

Pull out specific exports with `let` destructuring:

```parsley
let {floor, ceil} = import @std/math
floor(3.7)                       // 3
ceil(3.2)                        // 4
```

## Import Paths

| Path type | Syntax | Resolves to |
|---|---|---|
| Standard library | `@std/math` | Built-in stdlib module |
| Relative | `@./utils.pars` | Relative to current file |
| Project root | `@~/lib/utils.pars` | Relative to project root |

All import paths start with `@`. The prefix determines how the path is resolved:

```parsley
import @std/math                 // stdlib
import @./helpers.pars           // sibling file
import @./lib/format.pars        // subdirectory
import @~/shared/config.pars     // project root
```

## Exporting

Use `export` to make values available to importers. Everything else in the file is private.

### export let

Declare and export in one statement:

```parsley
export let greeting = "Hello"
export let double = fn(x) { x * 2 }
```

### export assignment

Export a value by name:

```parsley
export PI = 3.14159
export square = fn(x) { x * x }
```

### Bare export

Export a value that was already defined:

```parsley
let helper = fn(x) { x + 1 }
export helper
```

### Destructured export

Export multiple values from a destructuring assignment:

```parsley
export {width, height} = getDimensions()
```

## Computed Exports

`export computed` creates an export that recalculates on every access. Useful for exposing live data like database queries or timestamps:

```parsley
export computed timestamp = @now
export computed count = items.length()
```

Block form for multi-line computations:

```parsley
export computed activeUsers {
    let query = "SELECT * FROM users WHERE active = true"
    @DB.query(query)
}
```

Computed exports look like regular values to the consumer, but each access re-evaluates the body:

```parsley
import @./data.pars

// Each access runs the query again
for (user in data.activeUsers) { user.name }  // Query 1
for (user in data.activeUsers) { user.email } // Query 2

// Snapshot by assigning to a variable
let snapshot = data.activeUsers               // Query 3
for (user in snapshot) { user.name }          // Uses snapshot
for (user in snapshot) { user.email }         // Uses snapshot
```

> ⚠️ Computed exports recalculate on **every access**. If the computation is expensive, assign the result to a local variable to avoid redundant work.

## Module Example

A module file (`mathutils.pars`):

```parsley
// Private — not visible to importers
let square = fn(x) { x * x }

// Public API
export PI = 3.14159
export pythagoras = fn(a, b) {
    math.sqrt(square(a) + square(b))
}
export cube = fn(x) { x * x * x }
```

Consuming the module:

```parsley
import @./mathutils.pars
mathutils.PI                     // 3.14159
mathutils.cube(3)                // 27

// Or destructure what you need
let {PI, pythagoras} = import @./mathutils.pars
pythagoras(3, 4)                 // 5
```

## Module Scope

Each module has its own isolated environment:

- Variables defined in a module don't leak into the importer's scope.
- The importer only sees `export`ed names, accessed through the module dictionary.
- Modules inherit security policy and database connections from the importing environment, but not variables.

## Caching

Modules are evaluated once. The first `import` runs the file and caches the resulting dictionary. All subsequent imports of the same path return the cached result, regardless of where the import appears:

```parsley
// Both get the same cached module dictionary
import @./config.pars
let {theme} = import @./config.pars    // no re-evaluation
```

## Circular Import Prevention

If module A imports module B and module B imports module A, Parsley detects the cycle and raises an import error. Restructure shared code into a third module that both can import.

## Standard Library Modules

| Module | Description |
|---|---|
| `@std/math` | Mathematical functions (floor, ceil, sqrt, abs, etc.) |
| `@std/valid` | Validation helpers |
| `@std/id` | ID generation (UUID, nanoid, etc.) |
| `@std/table` | Table constructor (deprecated — prefer `@table` literal) |
| `@std/api` | API utilities |
| `@std/mdDoc` | Markdown document processing |
| `@std/dev` | Development/debugging tools |
| `@std/html` | HTML utilities |
| `@std/schema` | Schema utilities |

## Key Differences from Other Languages

- **`import` is an expression** — it returns a dictionary, so you can destructure it, pass it around, or assign it to any name.
- **No `from` keyword** — use `let {x, y} = import @path` instead of `from path import x, y`.
- **Path prefixes are required** — `@std/`, `@./`, `@~/` make resolution explicit. No bare module names.
- **Computed exports** — a unique feature for live/reactive data that has no direct equivalent in most languages.
- **No re-exports or barrel files** — each module exports its own values. Import and re-export manually if needed.

## See Also

- [Variables & Binding](variables.md) — `let` destructuring used with imports
- [Functions](functions.md) — exporting functions as module API
- [Tags](tags.md) — components are imported functions used as custom tags
- [@std/math](../stdlib/math.md) — math standard library reference
- [@std/api](../stdlib/api.md) — API utilities for Basil handlers
- [@std/id](../stdlib/id.md) — ID generation functions