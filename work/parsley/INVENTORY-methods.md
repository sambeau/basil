# Evaluator Methods Inventory

**Source**: `pkg/parsley/evaluator/methods.go`  
**Extracted**: 2026-01-11

---

## String Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `toUpper()` | `() → string` | Convert to uppercase |
| `toLower()` | `() → string` | Convert to lowercase |
| `toTitle()` | `() → string` | Convert to title case |
| `trim()` | `() → string` | Remove leading/trailing whitespace |
| `split(delim)` | `(string) → array` | Split string by delimiter |
| `replace(search, repl)` | `(string\|regex, string\|fn) → string` | Replace occurrences |
| `length()` | `() → int` | Return character count (Unicode-aware) |
| `includes(substr)` | `(string) → bool` | Check if contains substring |
| `highlight(phrase, tag?)` | `(string, string?) → string` | Wrap matches in HTML tag (XSS-safe) |
| `paragraphs()` | `() → string` | Convert plain text to HTML paragraphs |
| `render(dict?)` | `(dict?) → string` | Interpolate template with variables |
| `parseMarkdown(options?)` | `(dict?) → dict` | Parse markdown to `{html, raw, md}` |
| `parseJSON()` | `() → any` | Parse JSON string to value |
| `parseCSV(header?)` | `(bool?) → array` | Parse CSV string (header=true returns dicts) |
| `collapse()` | `() → string` | Collapse whitespace to single spaces |
| `normalizeSpace()` | `() → string` | Collapse + trim whitespace |
| `stripSpace()` | `() → string` | Remove all whitespace |
| `stripHtml()` | `() → string` | Remove HTML tags, decode entities |
| `digits()` | `() → string` | Extract only digits |
| `slug()` | `() → string` | Create URL-safe slug |
| `htmlEncode()` | `() → string` | Escape HTML entities |
| `htmlDecode()` | `() → string` | Unescape HTML entities |
| `urlEncode()` | `() → string` | URL encode (query string style) |
| `urlDecode()` | `() → string` | URL decode |
| `urlPathEncode()` | `() → string` | URL encode for path segments |
| `urlQueryEncode()` | `() → string` | URL encode for query values |
| `outdent()` | `() → string` | Remove common leading whitespace |
| `indent(n)` | `(int) → string` | Add n spaces to line starts |

---

## Array Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `length()` | `() → int` | Return element count |
| `reverse()` | `() → array` | Return reversed copy |
| `sort(options?)` | `({natural: bool}?) → array` | Sort (natural=true by default) |
| `sortBy(fn)` | `(fn) → array` | Sort by key function result |
| `map(fn)` | `(fn) → array` | Transform elements (null filtered out) |
| `filter(fn)` | `(fn) → array` | Keep elements where fn returns truthy |
| `reduce(fn, init)` | `(fn, any) → any` | Reduce to single value |
| `format(style?, locale?)` | `(string?, string?) → string` | Format as list ("and", "or", "unit") |
| `join(sep?)` | `(string?) → string` | Join elements with separator |
| `toJSON()` | `() → string` | Convert to JSON string |
| `toCSV(header?)` | `(bool?) → string` | Convert to CSV string |
| `shuffle()` | `() → array` | Return randomly shuffled copy |
| `pick()` | `() → any` | Return single random element |
| `pick(n)` | `(int) → array` | Return n random elements (with replacement) |
| `take(n)` | `(int) → array` | Return n unique random elements |
| `insert(idx, val)` | `(int, any) → array` | Insert value at index |
| `has(item)` | `(any) → bool` | Check if item exists |
| `hasAny(arr)` | `(array) → bool` | Check if any items exist |
| `hasAll(arr)` | `(array) → bool` | Check if all items exist |

---

## Dictionary Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `keys()` | `() → array` | Return array of keys (ordered) |
| `values()` | `() → array` | Return array of values (ordered) |
| `entries(k?, v?)` | `(string?, string?) → array` | Return array of `{key, value}` dicts |
| `has(key)` | `(string) → bool` | Check if key exists |
| `delete(key)` | `(string) → null` | Remove key (mutates) |
| `insertAfter(after, key, val)` | `(string, string, any) → dict` | Insert k/v after existing key |
| `insertBefore(before, key, val)` | `(string, string, any) → dict` | Insert k/v before existing key |
| `render(template)` | `(string) → string` | Render template with dict values |
| `toJSON()` | `() → string` | Convert to JSON string |

---

## Integer Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `format(locale?)` | `(string?) → string` | Format with locale (e.g., "1,234,567") |
| `currency(code, locale?)` | `(string, string?) → string` | Format as currency |
| `percent(locale?)` | `(string?) → string` | Format as percentage |
| `humanize(locale?)` | `(string?) → string` | Compact format (e.g., "1.2M") |

---

## Float Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `format(locale?)` | `(string?) → string` | Format with locale |
| `currency(code, locale?)` | `(string, string?) → string` | Format as currency |
| `percent(locale?)` | `(string?) → string` | Format as percentage |
| `humanize(locale?)` | `(string?) → string` | Compact format (e.g., "1.2M") |

---

## DateTime Methods

DateTime is represented as a dictionary with properties like `year`, `month`, `day`, etc.

### Properties (Computed)
| Property | Type | Description |
|----------|------|-------------|
| `year` | int | Year (e.g., 2024) |
| `month` | int | Month (1-12) |
| `day` | int | Day of month (1-31) |
| `hour` | int | Hour (0-23) |
| `minute` | int | Minute (0-59) |
| `second` | int | Second (0-59) |
| `weekday` | string | Day name (e.g., "Monday") |
| `dayOfYear` | int | Day of year (1-366) |
| `week` | int | ISO week number |
| `timestamp` | int | Unix timestamp (seconds) |
| `unix` | int | Alias for timestamp |

### Methods
| Method | Signature | Description |
|--------|-----------|-------------|
| `format(style?, locale?)` | `(string?, string?) → string` | Format datetime |
| `toDict()` | `() → dict` | Return raw dictionary |

**Format Styles:**
- `"short"` → "1/11/26"
- `"medium"` → "Jan 11, 2026"
- `"long"` → "January 11, 2026" (default)
- `"full"` → "Sunday, January 11, 2026"
- `"time"` → "2:30 PM"
- `"timeShort"` → "14:30"
- `"datetime"` → "Jan 11, 2026, 2:30 PM"
- `"iso"` → "2026-01-11T14:30:00Z"
- Custom format strings also supported

---

## Duration Methods

Duration is represented as a dictionary with components like `days`, `hours`, etc.

### Properties
| Property | Type | Description |
|----------|------|-------------|
| `years` | int | Years component |
| `months` | int | Months component |
| `weeks` | int | Weeks component |
| `days` | int | Days component |
| `hours` | int | Hours component |
| `minutes` | int | Minutes component |
| `seconds` | int | Seconds component |

### Methods
| Method | Signature | Description |
|--------|-----------|-------------|
| `format(locale?)` | `(string?) → string` | Format as relative time |
| `toDict()` | `() → dict` | Return raw dictionary |

---

## Path Methods

Path is represented as a dictionary with path components.

### Properties
| Property | Type | Description |
|----------|------|-------------|
| `path` | string | Full path string |
| `base` | string | Filename without directory |
| `ext` | string | File extension |
| `dir` | string | Directory portion |
| `absolute` | bool | Whether path is absolute |

### Methods
| Method | Signature | Description |
|--------|-----------|-------------|
| `toDict()` | `() → dict` | Return raw dictionary |
| `isAbsolute()` | `() → bool` | Check if absolute path |
| `isRelative()` | `() → bool` | Check if relative path |
| `public()` | `() → string` | Get public URL for asset |
| `toURL(prefix)` | `(string) → string` | Convert to URL with prefix |
| `match(pattern)` | `(string) → dict\|null` | Match route pattern, extract params |

### match() Pattern Syntax
```parsley
@./users/123.match("users/:id")      // {id: "123"}
@./files/a/b/c.match("files/*")      // {rest: "a/b/c"}
@./api/v1/users.match("api/:version/*")  // {version: "v1", rest: "users"}
```

---

## URL Methods

URL is represented as a dictionary with URL components.

### Properties
| Property | Type | Description |
|----------|------|-------------|
| `scheme` | string | Protocol (e.g., "https") |
| `host` | string | Hostname |
| `port` | int | Port number |
| `path` | array | Path segments |
| `query` | dict | Query parameters |
| `fragment` | string | Hash fragment |

### Methods
| Method | Signature | Description |
|--------|-----------|-------------|
| `toDict()` | `() → dict` | Return raw dictionary |
| `origin()` | `() → string` | Get origin (scheme://host:port) |
| `pathname()` | `() → string` | Get path as string |
| `search()` | `() → string` | Get query string (?key=val) |
| `href()` | `() → string` | Get full URL string |

---

## Regex Methods

Regex is represented as a dictionary with `pattern` and `flags`.

### Properties
| Property | Type | Description |
|----------|------|-------------|
| `pattern` | string | Regex pattern |
| `flags` | string | Flags (i, m, s, g) |

### Methods
| Method | Signature | Description |
|--------|-----------|-------------|
| `toDict()` | `() → dict` | Return raw dictionary |
| `format(style?)` | `(string?) → string` | Format as string |
| `test(str)` | `(string) → bool` | Test if string matches |
| `replace(str, repl)` | `(string, string\|fn) → string` | Replace in string |

**Format Styles:**
- `"pattern"` → just the pattern
- `"literal"` → `/pattern/flags` (default)
- `"verbose"` → "pattern: X, flags: Y"

---

## Money Methods

Money is a first-class type with `Amount`, `Currency`, and `Scale`.

### Properties
| Property | Type | Description |
|----------|------|-------------|
| `currency` | string | ISO currency code (e.g., "USD") |
| `amount` | int | Amount in smallest units (cents) |
| `scale` | int | Decimal places for currency |

### Methods
| Method | Signature | Description |
|--------|-----------|-------------|
| `format(locale?)` | `(string?) → string` | Format with currency symbol |
| `abs()` | `() → money` | Get absolute value |
| `split(n)` | `(int) → array` | Split into n parts (penny-accurate) |

**split() Example:**
```parsley
$100.00.split(3)  // [$33.34, $33.33, $33.33]
```

---

## File Methods

File dictionary represents a file reference.

| Method | Signature | Description |
|--------|-----------|-------------|
| `toDict()` | `() → dict` | Return raw dictionary |
| `remove()` | `() → null` | Delete file from filesystem |
| `mkdir(opts?)` | `({parents: bool}?) → null` | Create directory |
| `rmdir(opts?)` | `({recursive: bool}?) → null` | Remove directory |

---

## Dir (Directory) Methods

Directory dictionary represents directory listing.

| Method | Signature | Description |
|--------|-----------|-------------|
| `toDict()` | `() → dict` | Return raw dictionary |
| `mkdir(opts?)` | `({parents: bool}?) → null` | Create directory |
| `rmdir(opts?)` | `({recursive: bool}?) → null` | Remove directory |

---

## Response Methods

Response dictionary from HTTP requests.

| Method | Signature | Description |
|--------|-----------|-------------|
| `toDict()` | `() → dict` | Return raw dictionary |
| `response()` | `() → dict` | Get response metadata |
| `format()` | `() → string` | Get format (json, text, etc.) |
| `data()` | `() → any` | Get response data directly |

---

## Type Detection

The evaluator uses special properties to detect dictionary types:

| Type | Detection |
|------|-----------|
| DateTime | Has `year`, `month`, `day` |
| Duration | Has `seconds`, `minutes`, or `hours` |
| Money | Has `currency` and `amount` |
| Path | Has `__type: "path"` |
| URL | Has `__type: "url"` |
| Regex | Has `__type: "regex"` |
| File | Has `__type: "file"` |
| Response | Has `__type: "response"` |

---

## Universal Methods

All types support:
- `.type` → Returns type name as string
- String conversion via `+ ""` or template interpolation

---

## Natural Sort Order

Default sorting uses natural order:
1. null
2. Numbers (mixed int/float)
3. Strings (natural alphanumeric)
4. Booleans
5. Dates
6. Durations
7. Money
8. Arrays
9. Dictionaries
