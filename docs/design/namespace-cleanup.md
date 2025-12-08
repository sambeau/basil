# Parsley Namespace Cleanup Design

## Overview

This document proposes cleanup of the Parsley global namespace by:
1. Removing builtins that are duplicated as methods
2. Moving utility functions to stdlib modules
3. Keeping type constructors and essential functions in the global namespace

## Guiding Principles

1. **Type constructors stay global** - Functions that create types/pseudo-types remain in namespace (e.g., `time()`, `url()`, `file()`, `money()`)
2. **Methods replace function forms** - If `arr.sort()` exists, remove `sort(arr)`
3. **Essential utilities stay global** - `len()`, `now()`, `import()`, etc.
4. **Stdlib for domain-specific functions** - Formatting, parsing, validation move to imports

---

## Current Builtins (59 total)

### Category 1: Keep as Global Builtins

#### Core Language (Essential)
| Function | Reason |
|----------|--------|
| `import` | Core language feature |
| `len` | Universal idiom, works on multiple types |
| `tag` | Core to Parsley's HTML generation |
| `fail` | Error handling |
| `log`, `logLine` | Debugging essential |
| `print`, `println` | Output essential |
| `repr` | Debugging |

#### Type Constructors (Create types/pseudo-types)
| Function | Creates | Reason |
|----------|---------|--------|
| `now` | datetime | Primary way to get current time |
| `time` | datetime | Creates datetime from components |
| `url` | url dict | Creates URL pseudo-type |
| `file` | file dict | Creates file handle pseudo-type |
| `dir` | dir dict | Creates directory handle pseudo-type |
| `regex` | regex dict | Creates compiled regex pseudo-type |
| `money` | Money | Creates Money type |
| `asset` | asset reference | Creates asset pipeline reference |

#### Database/Connection Constructors (Used with `<=>` operator)
| Function | Creates | Reason |
|----------|---------|--------|
| `SQLITE` | DBConnection | Database connection |
| `POSTGRES` | DBConnection | Database connection |
| `MYSQL` | DBConnection | Database connection |
| `SFTP` | SFTPConnection | SFTP connection |
| `COMMAND` | Command handle | Shell command handle |

#### Type Conversion (Fundamental)
| Function | Reason |
|----------|--------|
| `toInt` | Type conversion |
| `toFloat` | Type conversion |
| `toNumber` | Parse string to number |
| `toString` | Type conversion |
| `toDebug` | Debug representation |
| `toArray` | Convert to array |
| `toDict` | Convert to dict |

**Total: 28 builtins to keep**

---

### Category 2: Remove (Duplicated as Methods)

These exist both as builtins and methods. The method form is preferred.

#### String Operations
| Builtin | Method Form | Action |
|---------|-------------|--------|
| `toUpper(s)` | `s.toUpper()` | **Remove** |
| `toLower(s)` | `s.toLower()` | **Remove** |
| `replace(s, old, new)` | `s.replace(old, new)` | **Remove** |
| `split(s, delim)` | `s.split(delim)` | **Remove** |

#### Array Operations
| Builtin | Method Form | Action |
|---------|-------------|--------|
| `map(arr, fn)` | `arr.map(fn)` | **Remove** |
| `sort(arr)` | `arr.sort()` | **Remove** |
| `reverse(arr)` | `arr.reverse()` | **Remove** |
| `sortBy(arr, fn)` | `arr.sortBy(fn)` | **Remove** |

#### Dictionary Operations
| Builtin | Method Form | Action |
|---------|-------------|--------|
| `keys(dict)` | `dict.keys()` | **Remove** |
| `values(dict)` | `dict.values()` | **Remove** |
| `has(dict, key)` | `dict.has(key)` | **Remove** |

**Total: 11 builtins to remove**

---

### Category 3: Move to Standard Library

These are domain-specific utilities better served by imports.

#### → `std/format` (new module)
| Current | New | Notes |
|---------|-----|-------|
| `format(template, ...)` | `format.string(template, ...)` | String interpolation |
| `formatNumber(n, ...)` | `format.number(n, ...)` | Number formatting |
| `formatCurrency(n, ...)` | `format.currency(n, ...)` | Currency formatting |
| `formatPercent(n, ...)` | `format.percent(n, ...)` | Percent formatting |
| `formatDate(d, ...)` | `format.date(d, ...)` | Date formatting |

#### → `std/fs` (new module)
| Current | New | Notes |
|---------|-----|-------|
| `files(pattern)` | `fs.glob(pattern)` | Glob file search |
| `JSON(path)` | `fs.readJSON(path)` | Read JSON file |
| `YAML(path)` | `fs.readYAML(path)` | Read YAML file |
| `CSV(path)` | `fs.readCSV(path)` | Read CSV file |
| `lines(path)` | `fs.readLines(path)` | Read lines |
| `text(path)` | `fs.readText(path)` | Read text |
| `bytes(path)` | `fs.readBytes(path)` | Read bytes |
| `SVG(path)` | `fs.readSVG(path)` | Read SVG |
| `MD(path)` | `fs.readMarkdown(path)` | Read Markdown |

**Note**: `file()` and `dir()` remain as global type constructors. The `std/fs` module provides file reading utilities.

#### → `std/json` (new module)
| Current | New | Notes |
|---------|-----|-------|
| `parseJSON(s)` | `json.parse(s)` | Parse JSON string |
| `stringifyJSON(obj)` | `json.stringify(obj)` | Serialize to JSON |

#### → `std/csv` (new module)
| Current | New | Notes |
|---------|-----|-------|
| `parseCSV(s)` | `csv.parse(s)` | Parse CSV string |
| `stringifyCSV(arr)` | `csv.stringify(arr)` | Serialize to CSV |

**Total: 16 builtins to move to stdlib**

---

### Category 4: Keep but Consider Future

| Function | Current Status | Notes |
|----------|----------------|-------|
| `match(path, pattern)` | Keep global | Path pattern matching - useful everywhere |

---

## Summary

| Category | Count | Action |
|----------|-------|--------|
| Keep as global | 28 | No change |
| Remove (method duplicates) | 11 | Delete, use methods |
| Move to stdlib | 16 | Create new modules |
| **Total** | 55 | (4 already in stdlib or special) |

---

## Migration Path

### Phase 1: Deprecation Warnings (Pre-1.0)
Add deprecation warnings to builtins that will be removed:
```
Warning: toUpper() is deprecated. Use "string".toUpper() instead.
```

### Phase 2: Remove Deprecated (1.0)
- Remove the 11 method-duplicate builtins
- Users must use method syntax

### Phase 3: Stdlib Modules (1.0 or Post-1.0)
- Create `std/format`, `std/fs`, `std/json`, `std/csv`
- Add deprecation warnings to moved functions
- Eventually remove from global namespace

---

## New Standard Library Structure

After cleanup:

```
std/
├── table      # Table data structure (exists)
├── math       # Math functions (exists)
├── valid      # Validation (exists)
├── schema     # Schema definition (exists)
├── id         # ID generation (exists)
├── api        # API helpers (exists)
├── dev        # Dev logging (exists)
├── basil      # Basil context (exists)
├── format     # NEW: Formatting functions
├── fs         # NEW: File system utilities
├── json       # NEW: JSON parse/stringify
└── csv        # NEW: CSV parse/stringify
```

---

## Example: Before and After

### Before (Current)
```parsley
let upper = toUpper(name)
let items = sort(products)
let k = keys(config)
let data = JSON(~/data.json)
let formatted = formatNumber(price, {decimals: 2})
```

### After (Proposed)
```parsley
let upper = name.toUpper()
let items = products.sort()
let k = config.keys()

let {fs} = import("std/fs")
let data = fs.readJSON(~/data.json)

let {format} = import("std/format")
let formatted = format.number(price, {decimals: 2})
```

---

## Open Questions

1. **`format()` template function** - Is this used enough to stay global? It's powerful for `format("Hello {name}", {name: "World"})`.

2. **Backwards compatibility period** - How long should deprecation warnings run before removal?

3. **`match()` placement** - Currently does URL path matching. Should it move to `std/path` or `std/url`, or stay global?

4. **File readers (`JSON`, `CSV`, etc.)** - These use uppercase by convention. Should they stay as-is for familiarity, or lowercase in `std/fs`?

---

## Recommendation

For 1.0:
1. **Remove** the 11 method duplicates (breaking change, do before 1.0)
2. **Keep** type constructors and essentials global
3. **Defer** stdlib moves to post-1.0 (non-breaking, can add modules alongside existing builtins)

This minimizes breaking changes while cleaning up the most obvious redundancy.
