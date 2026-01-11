# Parsley Builtins Inventory

> Generated from source code audit of `pkg/parsley/evaluator/evaluator.go` and `pkg/parsley/evaluator/introspect.go`

## Global Builtins (from `getBuiltins()`)

### File/Data Loading

| Builtin | Arity | Description |
|---------|-------|-------------|
| `JSON` | 1-2 | Load JSON from path or URL |
| `YAML` | 1-2 | Load YAML from path or URL |
| `CSV` | 1-2 | Load CSV from path or URL as table |
| `MD` | 1-2 | Load markdown file and render to HTML |
| `markdown` | 1-2 | Parse markdown string (use MD() for files) |
| `lines` | 1-2 | Load file as array of lines |
| `text` | 1-2 | Load file as text string |
| `bytes` | 1-2 | Load file as byte array |
| `SVG` | 1-2 | Load SVG file with optional attributes |
| `file` | 1-2 | Load file with auto-detected format |
| `dir` | 1 | Create directory handle |
| `fileList` | 1 | List files matching glob pattern |

### Time

| Builtin | Arity | Description |
|---------|-------|-------------|
| `time` | 1-2 | Create datetime from string, timestamp, or dict |
| `now` | 0 | Current datetime (**DEPRECATED**: use `@now` literal) |

### URLs

| Builtin | Arity | Description |
|---------|-------|-------------|
| `url` | 1 | Parse URL string into components dict |

### Type Conversion

| Builtin | Arity | Description |
|---------|-------|-------------|
| `toInt` | 1 | Convert string to integer |
| `toFloat` | 1 | Convert string to float |
| `toNumber` | 1 | Convert string to number (int or float) |
| `toString` | 1+ | Convert value(s) to string |
| `toArray` | 1 | Convert dictionary to array of [key, value] pairs |
| `toDict` | 1 | Convert array of [key, value] pairs to dictionary |

### Introspection

| Builtin | Arity | Description |
|---------|-------|-------------|
| `inspect` | 1 | Get introspection data as dictionary |
| `describe` | 1 | Get human-readable description of value |
| `repr` | 1 | Get code/debug representation of value |
| `builtins` | 0-1 | List all builtin functions by category |

### Output

| Builtin | Arity | Description |
|---------|-------|-------------|
| `print` | 1+ | Print values without newline |
| `println` | 0+ | Print values with newline |
| `printf` | 2 | Print formatted string with dictionary values |
| `log` | 1+ | Log message to stdout |
| `logLine` | 1+ | Log message with newline (placeholder) |
| `toDebug` | 1+ | Convert value(s) to debug string |

### Control Flow

| Builtin | Arity | Description |
|---------|-------|-------------|
| `fail` | 1 | Throw a catchable error with message |

### Formatting

| Builtin | Arity | Description |
|---------|-------|-------------|
| `format` | 1-3 | Format duration or array as list |
| `tag` | 1-3 | Create HTML tag dictionary programmatically |

### Regex/Matching

| Builtin | Arity | Description |
|---------|-------|-------------|
| `regex` | 1-2 | Create regex pattern dictionary |
| `match` | 2 | Match path against pattern, extract named params |

### Money

| Builtin | Arity | Description |
|---------|-------|-------------|
| `money` | 2-3 | Create Money value from amount and currency |

### Assets

| Builtin | Arity | Description |
|---------|-------|-------------|
| `asset` | 1 | Convert path under public_dir to web URL |

---

## Connection Builtins (from `connectionBuiltins()`)

These are invoked via connection literals (`@sqlite`, `@postgres`, etc.)

| Builtin | Arity | Description |
|---------|-------|-------------|
| `sqlite` | 1-2 | Create SQLite database connection |
| `postgres` | 1-2 | Create PostgreSQL database connection |
| `mysql` | 1-2 | Create MySQL database connection |
| `sftp` | 1-2 | Create SFTP connection |
| `shell` | 0 | Create shell command executor |

---

## Removed Builtins (moved to methods)

Per code comments in evaluator.go:

| Former Builtin | Replacement |
|----------------|-------------|
| `map` | `arr.map(fn)` |
| `toUpper` | `str.toUpper()` |
| `toLower` | `str.toLower()` |
| `replace` | `str.replace(old, new)` |
| `split` | `str.split(delim)` |
| `sort` | `arr.sort()` |
| `reverse` | `arr.reverse()` |
| `sortBy` | `arr.sortBy(fn)` |
| `keys` | `dict.keys()` |
| `values` | `dict.values()` |
| `has` | `dict.has(key)` |

---

## Builtin Categories Summary

From `BuiltinMetadata`:

| Category | Count | Builtins |
|----------|-------|----------|
| file | 12 | JSON, YAML, CSV, MD, markdown, lines, text, bytes, SVG, file, dir, fileList |
| time | 2 | time, now |
| url | 1 | url |
| conversion | 6 | toInt, toFloat, toNumber, toString, toArray, toDict |
| introspection | 4 | inspect, describe, repr, builtins |
| output | 6 | print, println, printf, log, logLine, toDebug |
| control | 1 | fail |
| format | 2 | format, tag |
| regex | 2 | regex, match |
| money | 1 | money |
| asset | 1 | asset |
| connection | 5 | sqlite, postgres, mysql, sftp, shell |

**Total: 43 builtins**
