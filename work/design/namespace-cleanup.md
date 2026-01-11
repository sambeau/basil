# Parsley Namespace Cleanup Design

> **Status (2025-12-09)**: Phase 1 (method-duplicate builtins) completed in FEAT-052. Phase 2 (file builtin renames) completed. Import syntax updated to `import @path`. Planning final namespace organization.

## Overview

This document proposes cleanup of the Parsley global namespace by:
1. Removing builtins that are duplicated as methods
2. Keeping type constructors and file/data operations global (core to making websites from data)
3. Moving obscure utilities to methods or stdlib modules
4. Final reorganization - rename things once, organize the global namespace once, organize stdlib structure once

## Guiding Principles

1. **No deprecation; break things, fix things** - This is the last chance to get it right before stability. Make all breaking changes now.

2. **Type constructors stay global** - Functions that create types/pseudo-types remain in namespace (e.g., `time()`, `url()`, `file()`, `tag()`, `money()`)

3. **Methods replace function forms** - If `arr.sort()` exists, remove `sort(arr)`

4. **Core mission: websites from data** - Parsley excels at making websites from data. Core functions for this stay global: tags, dates, files, directories, data files, databases, money, regexes. Anything obscure moves to a namespace.

5. **Formatting is a method** - All types should have their own `.format()` method with a standard set of formatters.

6. **Serialization standard** - Core types and pseudo-types should all have `.toJSON()`. Any custom type with a custom `.toJSON()` should be serializable to JSON.

7. **CSV belongs to tables** - Table should have converters to/from CSV. Nothing else needs it (it's just an array of dictionaries).

---

## Current Builtins

### Category 1: Keep as Global Builtins

#### Core Language (Essential)

| Function           | Reason                |
| ------------------ | --------------------- |
| `import`           | Core language feature |
| `fail`             | Error handling        |
| `log`, `logLine`   | Debugging essential   |
| `print`, `println` | Output essential      |
| `repr`             | Debugging             |

**Note:** `len()` removed - use `string.length()` and `array.length()` methods instead.

#### Type Constructors (Create types/pseudo-types)

| Function | Creates    | Reason                                 |
| -------- | ---------- | -------------------------------------- |
| `tag`    | tag dict   | HTML tag pseudo-type (core to Parsley) |
| `now`    | datetime   | Primary way to get current time        |
| `time`   | datetime   | Creates datetime from components       |
| `url`    | url dict   | Creates URL pseudo-type                |
| `file`   | file dict  | Creates file handle pseudo-type        |
| `dir`    | dir dict   | Creates directory handle pseudo-type   |
| `regex`  | regex dict | Creates compiled regex pseudo-type     |
| `money`  | Money      | Creates Money type                     |

#### File Reading (Core to data-driven websites)

| Function            | Returns       | Reason                           |
| ------------------- | ------------- | -------------------------------- |
| `fileList(pattern)` | Array[File]   | Glob pattern matching for files  |
| `JSON(path)`        | Any           | Read and parse JSON file         |
| `YAML(path)`        | Any           | Read and parse YAML file         |
| `CSV(path)`         | Table         | Read and parse CSV file as table |
| `lines(path)`       | Array[String] | Read file as array of lines      |
| `text(path)`        | String        | Read file as text                |
| `bytes(path)`       | Bytes         | Read file as bytes               |
| `SVG(path)`         | SVG           | Read SVG file                    |
| `markdown(path)`    | String        | Read Markdown file               |

**Note:** Uppercase names (`JSON`, `YAML`, `CSV`, `SVG`) match the convention of `JSON`, `YAML`, `CSV`, `SVG` as format names.

#### Database/Connection Constructors (Used with `<=>` operator)

| Old Name       | New Name    | Creates               | Notes                                          |
| -------------- | ----------- | --------------------- | ---------------------------------------------- |
| `basil.sqlite` | `@DB`       | Built-in DB           | Basil's always-available database (Basil-only) |
| `SQLITE`       | `@sqlite`   | SQLite connection     | External SQLite database                       |
| `POSTGRES`     | `@postgres` | PostgreSQL connection | PostgreSQL database                            |
| `MYSQL`        | `@mysql`    | MySQL connection      | MySQL database                                 |
| `SFTP`         | `@sftp`     | SFTP connection       | SFTP file system                               |
| `COMMAND`      | `@shell`    | Shell command         | Shell command execution                        |

**Rationale:** Using `@` prefix distinguishes connections/external resources from regular functions. `@DB` is Basil's built-in database (always available), while `@sqlite`, `@postgres`, etc. are external connections that must be configured.

#### Type Conversion (Fundamental)

| Function   | Reason                 |
| ---------- | ---------------------- |
| `toInt`    | Type conversion        |
| `toFloat`  | Type conversion        |
| `toNumber` | Parse string to number |
| `toString` | Type conversion        |
| `toDebug`  | Debug representation   |
| `toArray`  | Convert to array       |
| `toDict`   | Convert to dict        |

---

### Category 2: File Builtin Renames ✅ COMPLETED

> **Completed (2025-12-09)**: File reading builtins renamed for clarity and consistency.

After testing various naming patterns, settled on these final names:

| Old Name                               | Final Name   | Rationale                                           |
| -------------------------------------- | ------------ | --------------------------------------------------- |
| `files()`                              | `fileList()` | Returns array of file handles, not individual files |
| `JSON()` / `jsonFile()` / `JSONFile()` | `JSON()`     | Uppercase format name, consistent with `.toJSON()`  |
| `YAML()` / `yamlFile()` / `YAMLFile()` | `YAML()`     | Uppercase format name                               |
| `CSV()` / `csvFile()` / `CSVFile()`    | `CSV()`      | Uppercase format name, consistent with `.toCSV()`   |
| `SVG()` / `svgFile()` / `SVGFile()`    | `SVG()`      | Uppercase format name                               |
| `MD()`                                 | `markdown()` | Full name is clearer than abbreviation              |
| `lines()` / `linesFile()`              | `lines()`    | Simple, clear                                       |
| `text()` / `textFile()`                | `text()`     | Simple, clear                                       |
| `bytes()` / `bytesFile()`              | `bytes()`    | Simple, clear                                       |

**Note:** These names are very similar to the original builtins (except `markdown`), but are the clearest of many variations tested. They're consistent with method names like `.toJSON()` and `.toCSV()`.

**Note:** `file(path)` remains unchanged - it's the generic file handle constructor.

---

### Category 3: Method-Duplicate Builtins ✅ COMPLETED

> **Completed in FEAT-052 (2025-12-08)**: All 11 builtins removed. Method syntax is now the only option.

These \~\~exist\~\~ existed both as builtins and methods. The method form is \~\~preferred\~\~ now required.

#### String Operations

| Builtin                | Method Form           | Action     |
| ---------------------- | --------------------- | ---------- |
| `toUpper(s)`           | `s.toUpper()`         | **Remove** |
| `toLower(s)`           | `s.toLower()`         | **Remove** |
| `replace(s, old, new)` | `s.replace(old, new)` | **Remove** |
| `split(s, delim)`      | `s.split(delim)`      | **Remove** |

#### Array Operations

| Builtin           | Method Form      | Action     |
| ----------------- | ---------------- | ---------- |
| `map(arr, fn)`    | `arr.map(fn)`    | **Remove** |
| `sort(arr)`       | `arr.sort()`     | **Remove** |
| `reverse(arr)`    | `arr.reverse()`  | **Remove** |
| `sortBy(arr, fn)` | `arr.sortBy(fn)` | **Remove** |

#### Dictionary Operations

| Builtin          | Method Form     | Action     |
| ---------------- | --------------- | ---------- |
| `keys(dict)`     | `dict.keys()`   | **Remove** |
| `values(dict)`   | `dict.values()` | **Remove** |
| `has(dict, key)` | `dict.has(key)` | **Remove** |

---

### Category 4: Move to Methods (Proposed)

These should become methods on their respective types, not separate modules.

#### Formatting → Type Methods

| Current                         | Proposed                       | Notes                               |
| ------------------------------- | ------------------------------ | ----------------------------------- |
| `formatNumber(n, ...)`          | `n.format(...)`                | Numbers have their own formatter    |
| `formatCurrency(money, ...)`    | `money.format(...)`            | Money type has its own formatter    |
| \~\~`formatPercent(n, ...)`\~\~ | `n.format({style: "percent"})` | Percentage is a number format style |
| `formatDate(d, ...)`            | `d.format(...)`                | Datetime has its own formatter      |

**Rationale:** Each type knows how to format itself. Standard `.format()` method across all types.

#### JSON Serialization → Type Methods

| Current              | Proposed        | Notes                                   |
| -------------------- | --------------- | --------------------------------------- |
| `stringifyJSON(obj)` | `obj.toJSON()`  | Arrays and dicts serialize themselves   |
| `parseJSON(s)`       | `s.parseJSON()` | String parses to object (array or dict) |

**Rationale:** All core types should have `.toJSON()`. Custom types with `.toJSON()` are auto-serializable.

**Clarification:** `s.parseJSON()` parses a JSON string into a Parsley object (array or dictionary), not into another string.

#### CSV → Table Methods

| Current               | Proposed        | Notes                   |
| --------------------- | --------------- | ----------------------- |
| `stringifyCSV(table)` | `table.toCSV()` | Table serializes to CSV |
| `parseCSV(s)`         | `s.parseCSV()`  | String parses to table  |

**Rationale:** CSV is just an array of dictionaries - only Table needs it.

#### Path Methods

| Current                           | Proposed              | Notes                                                       |
| --------------------------------- | --------------------- | ----------------------------------------------------------- |
| `match(path, pattern)`            | `path.match(pattern)` | Path matches against pattern                                |
| `publicUrl(path)` / `asset(path)` | `path.public()`       | **Basil-only**: Converts path under `public_dir` to web URL |
| N/A (new)                         | `path.toURL(prefix)`  | **Parsley**: Converts path to URL with explicit prefix      |

**Rationale for `path.public()` vs `publicUrl()`:** 
- Parsley has no knowledge of a `public_dir` - that's Basil-specific config
- Making it a method on path keeps Basil-specific logic in Basil
- `path.toURL(prefix)` is the general Parsley version with explicit control

---

### Category 5: File Operations Stay Global ✅ DECIDED

**Decision:** File reading operations stay global. They are core to Parsley's mission of making websites from data.

| Function     | Status      | Rationale                       |
| ------------ | ----------- | ------------------------------- |
| `fileList()` | Keep global | Glob patterns for finding files |
| `JSON()`     | Keep global | Reading JSON data files         |
| `YAML()`     | Keep global | Reading YAML config/data        |
| `CSV()`      | Keep global | Reading CSV data as tables      |
| `lines()`    | Keep global | Reading line-based data         |
| `text()`     | Keep global | Reading text content            |
| `bytes()`    | Keep global | Reading binary data             |
| `SVG()`      | Keep global | Reading SVG graphics            |
| `markdown()` | Keep global | Reading Markdown content        |

**Rationale:** These are fundamental to Parsley's purpose. A `std/fs` module would add ceremony without value.

---

## Summary

| Category                          | Action                   | Status                 |
| --------------------------------- | ------------------------ | ---------------------- |
| Core language + type constructors | Keep global              | Ongoing                |
| File operations (`*File()`)       | Keep global              | ✅ Renamed (2025-12-09) |
| Method duplicates                 | Removed                  | ✅ Done (FEAT-052)      |
| Database constructors             | Rename to `@` prefix     | Planned                |
| Formatting                        | Move to type methods     | Planned                |
| JSON/CSV serialization            | Move to type methods     | Planned                |
| `match()`                         | Move to path method      | Planned                |
| `len()`                           | Remove (use `.length()`) | Planned                |

---

## Migration Path

### Phase 1: Remove Method Duplicates ✅ COMPLETED
- ✅ Remove the 11 method-duplicate builtins — Done in FEAT-052
- ✅ Users must use method syntax — Now enforced

**Decision:** No deprecation period. Pre-alpha means we break things and fix them.

### Phase 2: Rename File Builtins ✅ COMPLETED
- ✅ Rename `files()` to `fileList()` — Done (2025-12-09)
- ✅ Rename format-specific builtins to simple names — Done (2025-12-09)
  - `JSON()`, `YAML()`, `CSV()`, `SVG()` (uppercase format names)
  - `lines()`, `text()`, `bytes()`, `markdown()` (lowercase descriptive names)

### Phase 3: Final Namespace Reorganization (Planned)
- [ ] Remove `len()` - use `.length()` method
- [ ] Rename database constructors to `@` prefix (`@DB`, `@sqlite`, `@postgres`, `@mysql`, `@sftp`, `@shell`)
- [ ] Move `publicUrl()` / `asset()` to `path.public()` method (Basil-only)
- [ ] Add `path.toURL(prefix)` method (Parsley)
- [ ] Move formatting to type methods (`.format()` on numbers, dates, money)
- [ ] Move JSON serialization to type methods (`.toJSON()`, `.parseJSON()`)
- [ ] Move CSV to table methods (`.toCSV()`, `.parseCSV()`)
- [ ] Move `match()` to path method (`.match()`)

### Phase 4: Standard Type Methods (Planned)
Ensure all core types have standard methods:
- [ ] All types: `.toJSON()`, `.toString()`, `.toDebug()`
- [ ] Formattable types: `.format(options)`
- [ ] Collections: `.length()` (not `len()`)
- [ ] Strings: `.parseJSON()` (string → object), `.parseCSV()` (string → table)
- [ ] Tables: `.toCSV()`
- [ ] Paths: `.match(pattern)`, `.public()` (Basil-only), `.toURL(prefix)`
- [ ] Strings: `.parseJSON()`, `.parseCSV()`
- [ ] Tables: `.toCSV()`
- [ ] Paths: `.match(pattern)`

---

## Final Global Namespace (Target)

After all phases complete, the global namespace will contain:

### Core Language
- `import`, `fail`, `log`, `logLine`, `print`, `println`, `repr`

### Type Constructors  
- `tag`, `now`, `time`, `url`, `file`, `dir`, `regex`, `money`, `publicUrl` (Basil-only)

### File Reading (Core to data-driven sites)
- `fileList`, `JSON`, `YAML`, `CSV`, `lines`, `text`, `bytes`, `SVG`, `markdown`

### Database/External Connections
- `@DB` (Basil-only), `@sqlite`, `@postgres`, `@mysql`, `@sftp`, `@shell`

### Type Conversion
- `toInt`, `toFloat`, `toNumber`, `toString`, `toDebug`, `toArray`, `toDict`

**Total: \~35 global builtins** (down from 59, with better organization)

---

## Standard Library Structure

The stdlib remains minimal and focused:

```
std/
├── table      # Table data structure
├── math       # Math functions  
├── valid      # Validation utilities
├── schema     # Schema definitions
├── id         # ID generation
├── api        # API helpers
├── dev        # Dev logging
└── basil      # Basil server context (Basil-only)
```

**No additions needed** - formatting, JSON, CSV, and file operations are handled by type/method system and global builtins.

---

## Examples: Before and After

### Phase 1: Method Duplicates (Completed)
```parsley
// ❌ Before (FEAT-052):
let upper = toUpper(name)
let items = sort(products)

// ✅ After (now required):
let upper = name.toUpper()
let items = products.sort()
```

### Phase 2: File Builtins (Completed)
```parsley
// ❌ Before (various iterations):
let data = jsonFile(~/data.json)     // or JSONFile()
let config = yamlFile(~/config.yml)   // or YAMLFile()
let rows = csvFile(~/data.csv)       // or CSVFile()

// ✅ After (final names):
let data = JSON(~/data.json)
let config = YAML(~/config.yml)
let rows = CSV(~/data.csv)
```

**Note:** Final names are similar to original builtins but clearest after testing many variations.

### Phase 3: Final Reorganization (Planned)
```parsley
// ❌ Current:
let count = len(items)
let formatted = formatNumber(price, {decimals: 2})
let json = stringifyJSON(data)
let params = match("/users/123", "/users/:id")
let url = asset(@./public/logo.png)  // or publicUrl()

// ✅ Future (planned):
let count = items.length()
let formatted = price.format({decimals: 2})
let json = data.toJSON()
let params = @/users/123.match("/users/:id")
let url = @./public/logo.png.public()          // Basil-only
let url = @./assets/logo.png.toURL("/static")  // Parsley with explicit prefix
```

### Database Connections (Planned)
```parsley
// ❌ Current:
import @std/basil
let db = basil.sqlite <=> { /* query */ }
let external = SQLITE("path/to/db.sqlite") <=> { /* query */ }

// ✅ Future (planned):
let db = @DB <=> { /* query */ }           // Basil's built-in DB
let external = @sqlite("path/to/db.sqlite") <=> { /* query */ }
```

---

## Design Decisions

### Why Keep File Operations Global?
Parsley's core mission is making websites from data. File reading is fundamental:
- Reading JSON/YAML data files
- Reading CSV for tables
- Reading Markdown for content
- Reading SVG for graphics

Adding `import @std/fs` ceremony would work against the language's purpose.

### Why Remove `len()`?
- Only works on strings and arrays (not universal)
- Both have `.length()` methods
- `"hello".length()` is just as clear as `len("hello")`
- Reduces global namespace for marginal benefit

### Why `@` Prefix for Connections?
- Visually distinguishes external resources from functions
- `@DB` vs `@sqlite` clarifies built-in vs external
- Consistent with `@std/module` import syntax
- Groups related concepts (databases, SFTP, shell)

### Why Methods for Formatting/Serialization?
- Each type knows how to format itself
- Standard `.format()` interface across types
- `.toJSON()` enables transparent serialization
- No need for separate modules

### Why Uppercase Format Names?
- `JSON`, `YAML`, `CSV`, `SVG` match conventional format names
- Consistent with how these formats are typically written (JSON not Json)
- Consistent with method names: `obj.toJSON()`, `table.toCSV()`
- `markdown` uses full lowercase name for clarity (not `MD` or `Markdown`)
- After testing many variations (`jsonFile`, `JSONFile`, `json`, `JSON`), these are clearest

### Why `path.public()` Instead of Global `publicUrl()`?
- **Parsley has no knowledge of `public_dir`** - that's Basil-specific configuration
- Making it a method keeps Basil-specific logic in Basil, not Parsley core
- `path.toURL(prefix)` is the general Parsley equivalent with explicit control
- Methods on types are clearer than global functions with context dependencies

---

## Open Questions

1. **`format()` template function** - Keep global for `format("Hello {name}", {name: "World"})` or move to `string.format()`?

2. **`toNumber()` vs parsing** - Should this stay global or become `string.toNumber()`?

3. **Path literal syntax** - Should `@"/users/123"` create a path dict for `.match()` method, or keep `match()` global accepting strings?

---

## Next Steps

1. ✅ Complete file builtin renames (done 2025-12-09)
2. Create feature specs for Phase 3 changes:
   3. Database constructor renames (`@DB`, `@sqlite`, etc.)
   4. Remove `len()`, use `.length()`
   5. Move `publicUrl()`/`asset()` to `path.public()` (Basil-only)
   6. Add `path.toURL(prefix)` (Parsley)
   7. Move formatting to type methods
   8. Move serialization to type methods (`.toJSON()`, `.parseJSON()`)
   9. Move `match()` to path method
3. Implement and test Phase 3 changes
4. Update all documentation and examples
5. Final validation before declaring namespace stable

---

**Last Updated:** 2025-12-09  
**Status:** Phase 2 complete (file builtins renamed), Phase 3 planned
