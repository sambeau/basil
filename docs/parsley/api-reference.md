# Parsley API Reference

> This document is auto-generated from the Parsley source code.
> Do not edit manually.

## Table of Contents

- [Types](#types)
- [Builtin Functions](#builtin-functions)
- [Operators](#operators)
- [Standard Library](#standard-library)

## Types

### Array

#### Methods

| Method | Description |
|--------|-------------|
| `.filter(arg)` | Filter by predicate |
| `.fmt(arg1?, arg2?)` | Format as list (and/or/unit, locale) |
| `.format(arg1?, arg2?)` | Format as list (and/or/unit, locale) (alias for fmt) |
| `.has(arg)` | Check if element exists |
| `.hasAll(arg)` | Check if all elements from array exist |
| `.hasAny(arg)` | Check if any element from array exists |
| `.insert(arg1, arg2)` | Insert at index |
| `.join(arg?)` | Join elements into string |
| `.length()` | Get element count |
| `.map(arg)` | Transform each element |
| `.pick(arg?)` | Pick random element(s) |
| `.reduce(arg1, arg2)` | Reduce to single value with accumulator function |
| `.reorder(arg, ...)` | Reorder/rename keys in each dictionary element |
| `.repr()` | Get representation string |
| `.reverse()` | Reverse order |
| `.shuffle()` | Randomly shuffle elements |
| `.sort(arg?)` | Sort elements |
| `.sortBy(arg)` | Sort by key function |
| `.take(arg)` | Take n unique random elements |
| `.toBox(...)` | Render as box diagram |
| `.toCSV(arg?)` | Convert to CSV string |
| `.toHTML(...)` | Convert to HTML |
| `.toJSON()` | Convert to JSON string |
| `.toMarkdown(...)` | Convert to Markdown |

### Boolean

#### Methods

| Method | Description |
|--------|-------------|
| `.inspect()` | Get debug dictionary |
| `.repr()` | Get representation string |
| `.toBox()` | Render as box diagram |
| `.toJSON()` | Convert to JSON string |

### Datetime

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.date` | string | Date portion (YYYY-MM-DD) |
| `.day` | integer | Day of month (1-31) |
| `.dayOfYear` | integer | Day number within year (1-366) |
| `.hour` | integer | Hour (0-23) |
| `.iso` | string | ISO 8601 datetime string |
| `.kind` | string | Datetime kind (date, datetime, time, time_seconds) |
| `.minute` | integer | Minute (0-59) |
| `.month` | integer | Month (1-12) |
| `.second` | integer | Second (0-59) |
| `.time` | string | Time portion (HH:MM or HH:MM:SS) |
| `.timestamp` | integer | Unix timestamp (alias for .unix) |
| `.unix` | integer | Unix timestamp (seconds since 1970-01-01) |
| `.week` | integer | ISO week number (1-53) |
| `.weekday` | string | Day name (Monday, Tuesday, etc.) |
| `.year` | integer | Year number |

#### Methods

| Method | Description |
|--------|-------------|
| `.dayOfYear()` | Day of year (1-366) |
| `.fmt(arg1?, arg2?)` | Format with style and/or locale |
| `.format(arg1?, arg2?)` | Format with style and/or locale (alias for fmt) |
| `.full(arg?)` | Maximum context date format |
| `.inspect()` | Return full dictionary including __type for debugging |
| `.long(arg?)` | Verbose date format |
| `.medium(arg?)` | Standard date format |
| `.repr()` | Get PLN literal representation |
| `.short(arg?)` | Compact date format |
| `.timestamp()` | Unix timestamp |
| `.toBox(...)` | Render as box diagram |
| `.toDict()` | Convert to plain dictionary (without __type marker) |
| `.toJSON()` | Convert to ISO 8601 JSON string |
| `.week()` | ISO week number |

### Dbconnection

#### Methods

| Method | Description |
|--------|-------------|
| `.begin()` | Begin a transaction |
| `.bind(arg1, arg2, arg3?)` | Bind a schema to a table (schema, tableName, options?) |
| `.close()` | Close the database connection |
| `.commit()` | Commit the current transaction |
| `.createTable(arg1, arg2?)` | Create a table from a schema (schema, tableName?) |
| `.lastInsertId()` | Get the last inserted row ID (SQLite only) |
| `.ping()` | Test the database connection |
| `.rollback()` | Rollback the current transaction |

### Dev

#### Methods

| Method | Description |
|--------|-------------|
| `.clearLog()` | Clear dev log |
| `.clearLogPage(arg)` | Clear page log for a route |
| `.log(arg1, arg2?, arg3?)` | Log value to dev panel |
| `.logPage(...)` | Log value for a specific route |
| `.setLogRoute(arg)` | Set default log route |

### Dictionary

#### Methods

| Method | Description |
|--------|-------------|
| `.as(arg)` | Cast to record using schema |
| `.delete(arg)` | Remove key |
| `.entries(...)` | Get entries as array of {key, value} dicts (keyName?, valueName?) |
| `.has(arg)` | Check if key exists |
| `.insertAfter(arg1, arg2, arg3)` | Insert key-value pair after existing key |
| `.insertBefore(arg1, arg2, arg3)` | Insert key-value pair before existing key |
| `.keys()` | Get all keys |
| `.render(arg)` | Render template string with dictionary values |
| `.reorder(arg, ...)` | Reorder and optionally rename keys |
| `.repr()` | Get representation string |
| `.toBox(...)` | Render as box diagram |
| `.toHTML(...)` | Convert to HTML |
| `.toJSON()` | Convert to JSON string |
| `.toMarkdown(...)` | Convert to Markdown |
| `.values()` | Get all values |

### Dir

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.exists` | boolean | Whether directory exists |
| `.path` | path | Directory path |

#### Methods

| Method | Description |
|--------|-------------|
| `.inspect()` | Return full dictionary including __type for debugging |
| `.mkdir(arg?)` | Create directory |
| `.rmdir(arg?)` | Remove directory |
| `.toDict()` | Convert to plain dictionary (without __type marker) |

### Duration

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.days` | integer | Total duration in days (null if months > 0) |
| `.hours` | integer | Total duration in hours (null if months > 0) |
| `.minutes` | integer | Total duration in minutes (null if months > 0) |
| `.months` | integer | Month component (years are stored as 12*years) |
| `.seconds` | integer | Seconds component (weeks/days/hours/minutes as seconds) |
| `.totalSeconds` | integer | Total seconds (only present when months == 0) |

#### Methods

| Method | Description |
|--------|-------------|
| `.fmt(arg1?, arg2?)` | Format with style and/or locale |
| `.format(arg1?, arg2?)` | Format with style and/or locale (alias for fmt) |
| `.full()` | Not supported for duration (returns error) |
| `.inspect()` | Return full dictionary including __type for debugging |
| `.long(arg?)` | Verbose duration format (2 hours 30 minutes) |
| `.medium(arg?)` | Standard duration format (2 hours) |
| `.repr()` | Get PLN literal representation |
| `.short(arg?)` | Compact duration format (2h) |
| `.toBox(...)` | Render as box diagram |
| `.toDict()` | Convert to plain dictionary (without __type marker) |
| `.toJSON()` | Convert to JSON object with components |

### File

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.exists` | boolean | Whether file exists |
| `.format` | string | File format (json, yaml, csv, etc.) |
| `.path` | path | File path |
| `.size` | integer | File size in bytes |

#### Methods

| Method | Description |
|--------|-------------|
| `.inspect()` | Return full dictionary including __type for debugging |
| `.mkdir(arg?)` | Create directory at this path |
| `.remove()` | Delete the file from filesystem |
| `.rmdir(arg?)` | Remove directory at this path |
| `.toDict()` | Convert to plain dictionary (without __type marker) |

### Float

#### Methods

| Method | Description |
|--------|-------------|
| `.abs()` | Absolute value |
| `.ceil()` | Round up |
| `.currency(arg1, arg2?)` | Format as currency (code, locale?) |
| `.floor()` | Round down |
| `.fmt(arg1?, arg2?)` | Format with style and/or locale |
| `.format(arg1?, arg2?)` | Format with style and/or locale (alias for fmt) |
| `.humanize(arg?)` | Human-readable format (1K, 1M) |
| `.inspect()` | Get debug dictionary with type info |
| `.long(arg?)` | Full precision format |
| `.medium(arg?)` | Standard format with separators |
| `.percent(arg?)` | Format as percentage |
| `.repr()` | Get representation string |
| `.round(arg?)` | Round to n decimals |
| `.short(arg?)` | Compact format (1K, 1M) |
| `.toBox()` | Render as box diagram |
| `.toJSON()` | Convert to JSON string |

### Integer

#### Methods

| Method | Description |
|--------|-------------|
| `.abs()` | Absolute value |
| `.currency(arg1, arg2?)` | Format as currency (code, locale?) |
| `.fmt(arg1?, arg2?)` | Format with style and/or locale |
| `.format(arg1?, arg2?)` | Format with style and/or locale (alias for fmt) |
| `.humanize(arg?)` | Human-readable format (1K, 1M) |
| `.inspect()` | Get debug dictionary with type info |
| `.long(arg?)` | Full precision format |
| `.medium(arg?)` | Standard format with separators |
| `.percent(arg?)` | Format as percentage |
| `.repr()` | Get representation string |
| `.short(arg?)` | Compact format (1K, 1M) |
| `.toBox()` | Render as box diagram |
| `.toJSON()` | Convert to JSON string |

### Markdown

#### Methods

| Method | Description |
|--------|-------------|
| `.ast()` | Get the raw AST dictionary |
| `.codeBlocks(arg?)` | Get all code blocks (language?) |
| `.filter(arg)` | Filter nodes by predicate fn |
| `.findAll(arg)` | Find all nodes of a given type (type or [types]) |
| `.findFirst(arg)` | Find the first node of a given type |
| `.headings(arg?)` | Get all headings (level?) |
| `.images()` | Get all images |
| `.links()` | Get all links |
| `.map(arg)` | Transform nodes with fn |
| `.text()` | Get plain text content |
| `.title()` | Get the document title (first h1) |
| `.toHTML()` | Render the document to HTML |
| `.toMarkdown()` | Render the document back to markdown |
| `.toc(arg?)` | Get table of contents (maxDepth?) |
| `.walk(arg)` | Walk the AST calling fn for each node |
| `.wordCount()` | Get word count |

### Money

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.amount` | integer | Amount in smallest currency unit (e.g., cents) |
| `.currency` | string | ISO 4217 currency code (e.g., USD, EUR) |
| `.scale` | integer | Number of decimal places for currency |

#### Methods

| Method | Description |
|--------|-------------|
| `.abs()` | Absolute value |
| `.fmt(arg1?, arg2?)` | Format with style and/or locale |
| `.format(arg1?, arg2?)` | Format with style and/or locale (alias for fmt) |
| `.full(arg?)` | Spelled out currency name |
| `.inspect()` | Get debug dictionary with internal values |
| `.long(arg?)` | Full precision format |
| `.medium(arg?)` | Standard currency format |
| `.negate()` | Negate amount |
| `.repr()` | Get representation string |
| `.short(arg?)` | Compact format ($1K) |
| `.split(arg)` | Split into n parts that sum to original |
| `.toBox(arg?)` | Render as box diagram |
| `.toDict()` | Convert to dictionary |
| `.toJSON()` | Convert to JSON string |

### Null

#### Methods

| Method | Description |
|--------|-------------|
| `.inspect()` | Get debug dictionary |
| `.repr()` | Get representation string |
| `.toBox()` | Render as box diagram |
| `.toJSON()` | Convert to JSON string |

### Path

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.absolute` | boolean | Whether path is absolute |
| `.extension` | string | File extension (without dot) |
| `.filename` | string | Last segment (file or directory name) |
| `.parent` | path | Parent directory path |
| `.segments` | array | Path segments as array of strings |

#### Methods

| Method | Description |
|--------|-------------|
| `.inspect()` | Return full dictionary including __type for debugging |
| `.isAbsolute()` | Check if absolute path |
| `.isRelative()` | Check if relative path |
| `.match(arg)` | Match against pattern, returns captures |
| `.public()` | Get public URL for this path |
| `.repr()` | Get PLN literal representation |
| `.toBox(...)` | Render as box diagram |
| `.toDict()` | Convert to plain dictionary (without __type marker) |
| `.toJSON()` | Convert to JSON string |
| `.toURL(arg)` | Convert to URL string with prefix |

### Record

#### Methods

| Method | Description |
|--------|-------------|
| `.data()` | Get all record data as a dictionary |
| `.enumValues(arg)` | Get enum options for a field |
| `.error(arg)` | Get error message for a field |
| `.errorCode(arg)` | Get error code for a field |
| `.errorList()` | Get all errors as an array |
| `.errors(arg?)` | Get validation errors (field?) |
| `.failIfInvalid(arg?)` | Fail with an error if the record is invalid (message?) |
| `.fieldProps(arg1, arg2?)` | Get form field props for a field (field, overrides?) |
| `.format(arg1, arg2?)` | Format a field value (field, options?) |
| `.hasError(arg)` | Check if a field has an error |
| `.isValid()` | Check if the record is valid |
| `.keys()` | Get all field names as an array |
| `.meta(arg1, arg2)` | Get metadata value for a field (field, key) |
| `.placeholder(arg)` | Get placeholder text for a field |
| `.schema()` | Get the record's schema |
| `.title(arg)` | Get display title for a field |
| `.toJSON()` | Serialize record data to a JSON string |
| `.update(arg)` | Merge fields and revalidate (dict) |
| `.validate()` | Validate the record against its schema |
| `.withError(arg1, arg2?, arg3?)` | Return a copy of the record with an added error (field, message?, code?) |

### Regex

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.flags` | string | Regex flags |
| `.pattern` | string | Regular expression pattern |

#### Methods

| Method | Description |
|--------|-------------|
| `.format(arg?)` | Format as string (pattern, literal, verbose) |
| `.inspect()` | Return full dictionary including __type for debugging |
| `.replace(arg1, arg2)` | Replace matches in string |
| `.split(arg)` | Split string on matches |
| `.test(arg)` | Test if string matches pattern |
| `.toBox(...)` | Render as box diagram |
| `.toDict()` | Convert to plain dictionary (without __type marker) |
| `.toJSON()` | Convert to JSON object with pattern and flags |

### Request

#### Methods

| Method | Description |
|--------|-------------|
| `.toDict()` | Convert to raw dictionary for debugging |

### Response

#### Methods

| Method | Description |
|--------|-------------|
| `.data()` | Get __data value |
| `.format()` | Get format string (json, text, etc.) |
| `.response()` | Get __response metadata dictionary |
| `.toDict()` | Convert to raw dictionary for debugging |

### Schema

#### Methods

| Method | Description |
|--------|-------------|
| `.enumValues(arg)` | Get enum options for a field |
| `.fields()` | Get all field names as an array |
| `.meta(arg1, arg2)` | Get metadata value for a field (field, key) |
| `.placeholder(arg)` | Get placeholder text for a field |
| `.title(arg)` | Get display title for a field |
| `.visibleFields()` | Get non-auto field names as an array |

### Schemadict

#### Methods

| Method | Description |
|--------|-------------|
| `.validate(arg)` | Validate a value against the schema |

### Session

#### Methods

| Method | Description |
|--------|-------------|
| `.all()` | Get all session data as dictionary |
| `.clear()` | Clear all session data |
| `.delete(arg)` | Delete session key |
| `.flash(arg1, arg2)` | Set flash message (key, value) |
| `.get(arg1, arg2?)` | Get session value (key, default?) |
| `.getAllFlash()` | Get all flash messages |
| `.getFlash(arg)` | Get and clear flash message |
| `.has(arg)` | Check if key exists |
| `.hasFlash()` | Check if any flash messages exist |
| `.regenerate()` | Regenerate session ID |
| `.set(arg1, arg2)` | Set session value (key, value) |

### Sftpconnection

#### Methods

| Method | Description |
|--------|-------------|
| `.close()` | Close the SFTP connection |

### Sftpfile

#### Methods

| Method | Description |
|--------|-------------|
| `.mkdir(arg?)` | Create a directory (options?) |
| `.remove()` | Remove a file |
| `.rmdir(arg?)` | Remove a directory (options?) |

### String

#### Methods

| Method | Description |
|--------|-------------|
| `.collapse()` | Collapse whitespace to single spaces |
| `.digits()` | Extract only digits |
| `.fromBase64()` | Decode from Base64 |
| `.highlight(arg1, arg2?)` | Wrap matches in HTML tag |
| `.htmlDecode()` | Decode HTML entities |
| `.htmlEncode()` | Encode HTML entities (<, >, &, etc.) |
| `.includes(arg)` | Check if contains substring |
| `.indent(arg)` | Add spaces to beginning of all non-blank lines |
| `.length()` | Get character count |
| `.normalizeSpace()` | Collapse and trim whitespace |
| `.outdent()` | Remove common leading whitespace from all lines |
| `.paragraphs()` | Convert blank lines to <p> tags |
| `.parseCSV(arg?)` | Parse as CSV |
| `.parseJSON()` | Parse as JSON |
| `.parseMarkdown(arg?)` | Parse markdown to {html, raw, md} |
| `.render(arg?)` | Interpolate template with values |
| `.replace(arg1, arg2)` | Replace all occurrences |
| `.repr()` | Get representation string |
| `.slug()` | Convert to URL-safe slug |
| `.split(arg)` | Split by string or regex separator into array |
| `.stripHtml()` | Remove HTML tags |
| `.stripSpace()` | Remove all whitespace |
| `.toBase64()` | Encode as Base64 |
| `.toBox()` | Render as box diagram |
| `.toCamel()` | Convert to camelCase |
| `.toJSON()` | Convert to JSON string |
| `.toKebab()` | Convert to kebab-case |
| `.toLower()` | Convert to lowercase |
| `.toPascal()` | Convert to PascalCase |
| `.toSnake()` | Convert to snake_case |
| `.toTitle()` | Convert to title case (capitalize first letter of each word) |
| `.toUpper()` | Convert to uppercase |
| `.trim()` | Remove leading/trailing whitespace |
| `.truncate(arg1, arg2?)` | Truncate to length with suffix (default '...') |
| `.urlDecode()` | Decode URL-encoded string |
| `.urlEncode()` | URL encode (spaces become +) |
| `.urlPathEncode()` | Encode URL path segment (/ becomes %2F) |
| `.urlQueryEncode()` | Encode URL query value (& and = encoded) |

### Table

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.columns` | array | Column names as array of strings |
| `.row` | dictionary | First row (or NULL if empty) |
| `.rows` | array | All rows as array of dictionaries |

#### Methods

| Method | Description |
|--------|-------------|
| `.all(arg)` | Check if all rows match predicate (fn) |
| `.any(arg)` | Check if any row matches predicate (fn) |
| `.appendCol(arg1, arg2)` | Add column at end (name, values) |
| `.appendRow(arg)` | Add a row at end (dict) |
| `.as(arg)` | Bind table to a schema (schema) |
| `.avg(arg)` | Average column values |
| `.column(arg)` | Get array of values from column |
| `.columnCount()` | Get number of columns |
| `.columnProps(arg)` | Get display props for a column (column) |
| `.copy()` | Create a copy of the table |
| `.count()` | Count rows |
| `.dropCol(arg, ...)` | Remove columns (col1, col2, ...) |
| `.errors()` | Get validation errors |
| `.find(arg)` | Find first row matching predicate (fn) |
| `.groupBy(arg1, arg2?)` | Group rows by column(s) (cols, aggregationFn?) |
| `.insertColAfter(arg1, arg2, arg3)` | Insert column after another (after, name, values) |
| `.insertColBefore(arg1, arg2, arg3)` | Insert column before another (before, name, values) |
| `.insertRowAt(arg1, arg2)` | Insert row at index (index, dict) |
| `.invalidRows()` | Get rows that fail validation |
| `.isValid()` | Check if all rows are valid |
| `.limit(arg1, arg2?)` | Limit rows (count, offset?) |
| `.map(arg)` | Transform each row (fn) |
| `.max(arg)` | Maximum column value |
| `.min(arg)` | Minimum column value |
| `.offset(arg)` | Skip rows (count) |
| `.orderBy(arg, ...)` | Sort rows by column(s) |
| `.renameCol(arg1, arg2)` | Rename a column (oldName, newName) |
| `.rowCount()` | Get number of rows |
| `.select(arg, ...)` | Select specific columns |
| `.sum(arg)` | Sum column values |
| `.toArray()` | Convert to array of row dictionaries |
| `.toBox(arg?)` | Convert to box diagram (options?) |
| `.toCSV()` | Convert to CSV string |
| `.toHTML(arg?)` | Convert to HTML table (footer?) |
| `.toJSON()` | Convert to JSON array |
| `.toMarkdown()` | Convert to Markdown table |
| `.unique(arg?)` | Remove duplicate rows (columns?) |
| `.validRows()` | Get rows that pass validation |
| `.validate()` | Validate rows against schema |
| `.where(arg)` | Filter rows by predicate |

### Tablebinding

#### Methods

| Method | Description |
|--------|-------------|
| `.all(arg?)` | Get all rows (options?) |
| `.avg(arg1, arg2?)` | Average a column (column, conditions?) |
| `.count(arg?)` | Count rows (conditions?) |
| `.delete(arg)` | Delete a row (id) |
| `.exists(arg)` | Check if a row exists (conditions) |
| `.find(arg)` | Find row by primary key (id) |
| `.findBy(arg1, arg2?)` | Find first row matching conditions (dict, options?) |
| `.first(arg1?, arg2?)` | Get first row (count?, options?) |
| `.insert(arg)` | Insert a new row (dict) |
| `.last(arg1?, arg2?)` | Get last row (count?, options?) |
| `.max(arg1, arg2?)` | Maximum value of a column (column, conditions?) |
| `.min(arg1, arg2?)` | Minimum value of a column (column, conditions?) |
| `.save(arg)` | Insert or update a row (dict) |
| `.sum(arg1, arg2?)` | Sum a column (column, conditions?) |
| `.toSQL(arg, ...)` | Generate SQL for an operation (method, args...) |
| `.update(arg1, arg2?)` | Update a row (record or table, or id and dict) |
| `.where(arg1, arg2?)` | Get rows matching conditions (dict or sql, params?) |

### Unit

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.family` | string | Unit family (length, mass, data, temperature, volume, area) |
| `.max` | unit | Maximum representable value at full (Scale=0) precision |
| `.min` | unit | Smallest representable positive value (1 base sub-unit) |
| `.system` | string | Measurement system (SI, US) |
| `.unit` | string | Display-hint unit suffix |
| `.value` | float | Decoded value in the display-hint unit |

#### Methods

| Method | Description |
|--------|-------------|
| `.abs()` | Absolute value |
| `.fmt(arg1?, arg2?)` | Format with style and/or locale |
| `.format(arg?)` | Format with optional precision (legacy, use fmt for style) |
| `.full(arg?)` | With conversion (5 metres (16.4 ft)) |
| `.inspect()` | Get debug dictionary with internal values |
| `.long(arg?)` | Spelled out unit name (5 metres) |
| `.medium(arg?)` | Standard format with precision (5.00m) |
| `.repr()` | Get parseable literal string |
| `.short(arg?)` | Compact format (5m) |
| `.to(arg)` | Convert to another unit |
| `.toBox()` | Render as box diagram |
| `.toDict()` | Convert to dictionary |
| `.toFraction()` | Get fraction string for US Customary values |
| `.toJSON()` | Convert to JSON string |

### Url

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.fragment` | string | Fragment identifier (after #) |
| `.host` | string | Hostname |
| `.path` | path | URL path as path object |
| `.port` | integer | Port number |
| `.query` | dictionary | Query parameters as dictionary |
| `.scheme` | string | URL scheme (http, https, etc.) |

#### Methods

| Method | Description |
|--------|-------------|
| `.href()` | Get full URL string |
| `.inspect()` | Return full dictionary including __type for debugging |
| `.origin()` | Get origin (scheme://host:port) |
| `.pathname()` | Get path component |
| `.repr()` | Get PLN literal representation |
| `.search()` | Get query string representation |
| `.toBox(...)` | Render as box diagram |
| `.toDict()` | Convert to plain dictionary (without __type marker) |
| `.toJSON()` | Convert to JSON string |

## Builtin Functions

### Assets

| Function | Description |
|----------|-------------|
| `asset(path)` | Get asset path with cache busting |

### Connections

| Function | Description |
|----------|-------------|
| `mysql(connectionString)` | Create MySQL database connection |
| `postgres(connectionString)` | Create PostgreSQL database connection |
| `sftp(connectionString)` | Create SFTP connection |
| `shell()` | Create shell command executor |
| `sqlite(path)` | Create SQLite database connection |

### Control Flow

| Function | Description |
|----------|-------------|
| `fail(message_or_dict)` | Throw an error with message string or error dictionary (must have 'message' key) |

### Type Conversion

| Function | Description |
|----------|-------------|
| `repr(value)` | Convert value to Parsley-parseable literal string |
| `toArray(value)` | Convert value to array |
| `toDict(pairs)` | Convert array of [key,value] pairs to dictionary |
| `toFloat(value)` | Convert value to float |
| `toInt(value)` | Convert value to integer |
| `toNumber(value)` | Convert value to number (int or float) |
| `toString(value)` | Convert value to string |

### File/Data Loading

| Function | Description |
|----------|-------------|
| `CSV(source, options?)` | Load CSV from path or URL as table |
| `JSON(source, options?)` | Load JSON from path or URL |
| `MD(path, options?)` | Load markdown file and render to HTML |
| `PLN(path, options?)` | Load PLN (Parsley Literal Notation) from path |
| `SVG(path, attributes?)` | Load SVG file with optional attributes |
| `YAML(source, options?)` | Load YAML from path or URL |
| `dir(path)` | List directory contents |
| `file(path, options?)` | Load file with auto-detected format |
| `fileList(path, pattern?)` | List files in directory recursively |
| `lines(source, options?)` | Load file as array of lines |
| `markdown(path, options?)` | Load markdown file with frontmatter |
| `raw(path, options?)` | Load file as raw byte array |
| `text(source, options?)` | Load file as text string |

### Formatting

| Function | Description |
|----------|-------------|
| `format(duration, locale?)` | Format duration as relative time |
| `tag(name, attributes?, content?)` | Create HTML tag |

### Introspection

| Function | Description |
|----------|-------------|
| `builtins(category?)` | List all builtin functions by category |
| `describe(value)` | Get human-readable description of value |
| `inspect(value)` | Get introspection data for value |

### Money

| Function | Description |
|----------|-------------|
| `money(amount, currency?)` | Create money value |

### Output

| Function | Description |
|----------|-------------|
| `log(values...)` | Log message |
| `logLine(values...)` | Log message with newline |

### Paths

| Function | Description |
|----------|-------------|
| `match(path, pattern)` | Match path/URL against pattern with named captures (:name, *name) |
| `path(pathString)` | Create path from string |

### Regular Expressions

| Function | Description |
|----------|-------------|
| `regex(pattern, flags?)` | Create regex pattern |

### Serialization

| Function | Description |
|----------|-------------|
| `deserialize(plnString)` | Parse PLN string to value |
| `serialize(value)` | Convert value to PLN string |

### Date & Time

| Function | Description |
|----------|-------------|
| `date(input, options?)` | Parse date string with locale support |
| `datetime(input, options?)` | Parse datetime from string, timestamp, or dict with locale support |
| `time(input)` | Parse time-only string (e.g., '3:45 PM') |

### Unit

| Function | Description |
|----------|-------------|
| `unit(value, suffix?)` | Create or convert a unit value |

### URLs

| Function | Description |
|----------|-------------|
| `url(urlString)` | Parse URL string into components |

## Operators

### Arithmetic

| Operator | Description |
|----------|-------------|
| `%` | Remainder after division |
| `*` | Multiply numbers or repeat strings/arrays |
| `+` | Add numbers or concatenate strings |
| `-` | Subtract numbers or compute set difference |
| `/` | Divide numbers or chunk arrays |

### Comparison

| Operator | Description |
|----------|-------------|
| `!=` | Check inequality |
| `<` | Check if less than |
| `<=` | Check if less than or equal |
| `==` | Check equality |
| `>` | Check if greater than |
| `>=` | Check if greater than or equal |

### Logical

| Operator | Description |
|----------|-------------|
| `!` | Negate boolean value |
| `&&` | Logical AND or set intersection |
| `and` | Logical AND (keyword form) |
| `or` | Logical OR (keyword form) |
| `\|\|` | Logical OR or set union |

### Collection

| Operator | Description |
|----------|-------------|
| `++` | Concatenate strings, arrays, or dictionaries |
| `..` | Create inclusive range |
| `in` | Check if value is in collection |
| `not in` | Check if value is not in collection |

### Regex

| Operator | Description |
|----------|-------------|
| `!~` | Check if string does not match regex |
| `~` | Match string against regex, returns captures or null |

### Null Handling

| Operator | Description |
|----------|-------------|
| `??` | Return left if not null, otherwise right |

## Standard Library

### @std/api

HTTP API route helpers

#### Functions

| Function | Description |
|----------|-------------|
| `adminOnly(arg)` | Require admin role for handler |
| `auth(arg1, arg2?)` | Require authentication for handler |
| `badRequest(arg?)` | Return 400 Bad Request error |
| `conflict(arg?)` | Return 409 Conflict error |
| `forbidden(arg?)` | Return 403 Forbidden error |
| `notFound(arg?)` | Return 404 Not Found error |
| `public(arg1, arg2?)` | Mark handler as public (no auth required) |
| `redirect(arg1, arg2?)` | Redirect to URL (status?) |
| `roles(arg1, arg2)` | Require specific role(s) for handler |
| `serverError(arg?)` | Return 500 Internal Server Error |
| `unauthorized(arg?)` | Return 401 Unauthorized error |

### @std/dev

Development tools (logging, debugging)

#### Functions

| Function | Description |
|----------|-------------|
| `dev()` | Development tools (log, clearLog, logPage, etc.) |

### @std/hash

Cryptographic hash functions for checksums and non-security hashing

#### Functions

| Function | Description |
|----------|-------------|
| `md5(arg)` | MD5 hash (hex string) |
| `sha1(arg)` | SHA1 hash (hex string) |
| `sha256(arg)` | SHA256 hash (hex string) |
| `sha512(arg)` | SHA512 hash (hex string) |

### @std/id

ID generation (UUID, nanoid, etc.)

#### Functions

| Function | Description |
|----------|-------------|
| `cuid()` | Generate CUID |
| `nanoid(arg?)` | Generate nanoid (length?) |
| `new()` | Generate new unique ID |
| `uuid()` | Generate UUID v4 |
| `uuidv4()` | Generate UUID v4 |
| `uuidv7()` | Generate UUID v7 |

### @std/math

Mathematical functions and constants

#### Constants

| Name | Description |
|------|-------------|
| `E` | Euler's number (2.71828...) |
| `PI` | Pi (3.14159...) |
| `TAU` | Tau (2*Pi) |

#### Functions

| Function | Description |
|----------|-------------|
| `abs(arg)` | Absolute value |
| `acos(arg)` | Arc cosine |
| `asin(arg)` | Arc sine |
| `atan(arg)` | Arc tangent |
| `atan2(arg1, arg2)` | Arc tangent of y/x |
| `avg(arg, ...)` | Average of values or array |
| `ceil(arg)` | Round up to integer |
| `clamp(arg1, arg2, arg3)` | Clamp value between min and max |
| `cos(arg)` | Cosine (radians) |
| `count(arg)` | Count elements in array |
| `degrees(arg)` | Radians to degrees |
| `dist(...)` | Distance between points |
| `exp(arg)` | e^x |
| `floor(arg)` | Round down to integer |
| `hypot(arg1, arg2)` | Hypotenuse length |
| `lerp(arg1, arg2, arg3)` | Linear interpolation |
| `log(arg)` | Natural logarithm |
| `log10(arg)` | Base-10 logarithm |
| `map(...)` | Map value from one range to another |
| `max(arg, ...)` | Maximum of values or array |
| `mean(arg, ...)` | Mean (alias for avg) |
| `median(arg)` | Median of array |
| `min(arg, ...)` | Minimum of values or array |
| `mode(arg)` | Mode of array |
| `pow(arg1, arg2)` | Power (base, exponent) |
| `product(arg, ...)` | Product of values or array |
| `radians(arg)` | Degrees to radians |
| `random()` | Random float 0-1 |
| `randomInt(arg1, arg2?)` | Random int (max) or (min, max) |
| `range(arg)` | Range (max - min) |
| `round(arg1, arg2?)` | Round to nearest (decimals?) |
| `seed(arg)` | Seed random generator |
| `sign(arg)` | Sign (-1, 0, or 1) |
| `sin(arg)` | Sine (radians) |
| `sqrt(arg)` | Square root |
| `stddev(arg)` | Standard deviation |
| `sum(arg, ...)` | Sum of values or array |
| `tan(arg)` | Tangent (radians) |
| `trunc(arg)` | Truncate to integer |
| `variance(arg)` | Variance |

### @std/mdDoc

Markdown document analysis and manipulation

#### Functions

| Function | Description |
|----------|-------------|
| `mdDoc(arg)` | Parse a Markdown string into a document object |

### @std/mddoc

Markdown document analysis and manipulation

#### Functions

| Function | Description |
|----------|-------------|
| `mdDoc(arg)` | Parse a Markdown string into a document object |

### @std/schema

Schema validation and type checking

#### Functions

| Function | Description |
|----------|-------------|
| `array(arg)` | Create array schema validator |
| `boolean()` | Create boolean schema validator |
| `date()` | Create date schema validator |
| `datetime(arg?)` | Create datetime schema validator |
| `define(arg1, arg2)` | Define named schema |
| `email()` | Create email schema validator |
| `enum(arg, ...)` | Create enum schema validator |
| `id()` | Create ID schema validator |
| `integer(arg?)` | Create integer schema validator |
| `money(arg?)` | Create money schema validator |
| `number(arg?)` | Create number schema validator |
| `object(arg)` | Create object schema validator |
| `phone()` | Create phone schema validator |
| `string(arg?)` | Create string schema validator |
| `table(arg)` | Create table schema validator |
| `url()` | Create URL schema validator |

### @std/valid

Validation predicates for IDs, financial data, and locale-specific formats

#### Functions

| Function | Description |
|----------|-------------|
| `creditCard(arg)` | Check credit card number (Luhn) |
| `cuid(arg)` | Check CUID2 format |
| `luhn(arg)` | Check Luhn algorithm |
| `nanoid(arg1, arg2?)` | Check NanoID format (length?) |
| `postalCode(arg1, arg2)` | Check postal code format (locale) |
| `ulid(arg)` | Check ULID format |
| `uuid(arg)` | Check UUID v4/v7 format |

### @basil/api

HTTP API route helpers

#### Functions

| Function | Description |
|----------|-------------|
| `adminOnly(arg)` | Require admin role for handler |
| `auth(arg1, arg2?)` | Require authentication for handler |
| `badRequest(arg?)` | Return 400 Bad Request error |
| `conflict(arg?)` | Return 409 Conflict error |
| `forbidden(arg?)` | Return 403 Forbidden error |
| `notFound(arg?)` | Return 404 Not Found error |
| `public(arg1, arg2?)` | Mark handler as public (no auth required) |
| `redirect(arg1, arg2?)` | Redirect to URL (status?) |
| `roles(arg1, arg2)` | Require specific role(s) for handler |
| `serverError(arg?)` | Return 500 Internal Server Error |
| `unauthorized(arg?)` | Return 401 Unauthorized error |

### @basil/auth

Auth context, db, session, and user shortcuts

#### Functions

| Function | Description |
|----------|-------------|
| `auth()` | Auth context |
| `session()` | Current session |
| `user()` | Current authenticated user |

### @basil/html

Pre-built HTML components (requires Basil server)

#### Functions

| Function | Description |
|----------|-------------|
| `A()` | Anchor/link element |
| `Abbr()` | Abbreviation element |
| `Accordion()` | Exclusive expandable sections |
| `Blockquote()` | Blockquote element |
| `Breadcrumb()` | Breadcrumb navigation |
| `Button()` | Button element |
| `Checkbox()` | Single checkbox |
| `CheckboxGroup()` | Checkbox group |
| `DataTable()` | Data table component |
| `Details()` | Expandable content section |
| `Dialog()` | Modal dialog |
| `ErrorSummary()` | Form validation error summary |
| `Figure()` | Figure with caption |
| `Form()` | Form wrapper |
| `Head()` | Deprecated alias for Meta |
| `Icon()` | Icon element |
| `Iframe()` | Iframe element |
| `Img()` | Image element |
| `LocalTime()` | Localized time display |
| `Meta()` | SEO and social media metadata tags |
| `Nav()` | Navigation wrapper |
| `Page()` | Page layout wrapper |
| `Pagination()` | Page navigation |
| `RadioGroup()` | Radio button group |
| `RelativeTime()` | Relative time display |
| `SelectField()` | Select dropdown field |
| `SkipLink()` | Skip to content link |
| `SrOnly()` | Screen reader only text |
| `TextField()` | Text input field |
| `TextareaField()` | Textarea input field |
| `Time()` | Time element |
| `TimeRange()` | Time range display |
| `Toast()` | Notification message |
| `Toasts()` | Toast container |

### @basil/http

HTTP request context (request, response, route, method). Use @params for query/form data.

#### Functions

| Function | Description |
|----------|-------------|
| `method()` | HTTP request method |
| `params()` | HTTP query/form parameters |
| `request()` | HTTP request object |
| `response()` | HTTP response object |
| `route()` | Current route path |

### @basil/log

Development tools (logging, debugging)

#### Functions

| Function | Description |
|----------|-------------|
| `dev()` | Development tools (log, clearLog, logPage, etc.) |

