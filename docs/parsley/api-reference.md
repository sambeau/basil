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
| `.format(arg1?, arg2?)` | Format as list (and/or/unit, locale) |
| `.insert(arg1, arg2)` | Insert at index |
| `.join(arg?)` | Join elements into string |
| `.length()` | Get element count |
| `.map(arg)` | Transform each element |
| `.pick(arg?)` | Pick random element(s) |
| `.reduce(arg1, arg2)` | Reduce to single value with accumulator function |
| `.reverse()` | Reverse order |
| `.shuffle()` | Randomly shuffle elements |
| `.sort()` | Sort elements |
| `.sortBy(arg)` | Sort by key function |
| `.take(arg)` | Take n unique random elements |
| `.toCSV(arg?)` | Convert to CSV string |
| `.toJSON()` | Convert to JSON string |

### Boolean

*No properties or methods.*

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
| `.format(arg1?, arg2?)` | Format with style and locale |
| `.timestamp()` | Unix timestamp |
| `.toDict()` | Convert to dictionary |
| `.week()` | ISO week number |

### Dbconnection

#### Methods

| Method | Description |
|--------|-------------|
| `.begin()` | Begin transaction |
| `.close()` | Close connection |
| `.commit()` | Commit transaction |
| `.ping()` | Test connection |
| `.rollback()` | Rollback transaction |

### Dev

#### Methods

| Method | Description |
|--------|-------------|
| `.clearLog()` | Clear dev log |
| `.clearLogPage()` | Clear page log |
| `.log(arg1, arg2?, arg3?)` | Log value to dev panel |
| `.logPage(arg?)` | Log page content |
| `.setLogRoute(arg)` | Set log route pattern |

### Dictionary

#### Methods

| Method | Description |
|--------|-------------|
| `.delete(arg)` | Remove key |
| `.entries()` | Get [key, value] pairs |
| `.has(arg)` | Check if key exists |
| `.insertAfter(arg1, arg2)` | Insert after key |
| `.insertBefore(arg1, arg2)` | Insert before key |
| `.keys()` | Get all keys |
| `.render(arg?)` | Render template with values |
| `.toJSON()` | Convert to JSON string |
| `.values()` | Get all values |

### Dir

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.exists` | boolean | Whether directory exists |
| `.path` | path | Directory path |

### Directory

#### Methods

| Method | Description |
|--------|-------------|
| `.exists()` | Check if directory exists |
| `.list()` | List directory contents |
| `.toDict()` | Convert to dictionary |

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
| `.format(arg?)` | Format as relative time |
| `.toDict()` | Convert to dictionary |

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
| `.exists()` | Check if file exists |
| `.read()` | Read file contents |
| `.stat()` | Get file metadata |
| `.toDict()` | Convert to dictionary |

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

### Function

*No properties or methods.*

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

*No properties or methods.*

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
| `.isAbsolute()` | Check if absolute path |
| `.isRelative()` | Check if relative path |
| `.join(arg, ...)` | Join path components |
| `.match(arg)` | Match against pattern |
| `.parent()` | Get parent directory |
| `.public()` | Get public URL |
| `.toDict()` | Convert to dictionary |
| `.toString()` | Convert to string |
| `.toURL(arg)` | Convert to URL with prefix |

### Regex

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.flags` | string | Regex flags |
| `.pattern` | string | Regular expression pattern |

#### Methods

| Method | Description |
|--------|-------------|
| `.match(arg)` | Find first match |
| `.matchAll(arg)` | Find all matches |
| `.replace(arg1, arg2)` | Replace matches |
| `.split(arg)` | Split string by pattern |
| `.test(arg)` | Test if string matches |
| `.toDict()` | Convert to dictionary |

### Session

#### Methods

| Method | Description |
|--------|-------------|
| `.all()` | Get all session data |
| `.clear()` | Clear all session data |
| `.delete(arg)` | Delete session key |
| `.flash(arg1, arg2)` | Set flash message (key, value) |
| `.get(arg1, arg2?)` | Get session value (key, default?) |
| `.getAllFlash()` | Get all flash messages |
| `.getFlash(arg)` | Get and clear flash message |
| `.has(arg)` | Check if key exists |
| `.hasFlash()` | Check if flash messages exist |
| `.regenerate()` | Regenerate session ID |
| `.set(arg1, arg2)` | Set session value (key, value) |

### Sftpconnection

#### Methods

| Method | Description |
|--------|-------------|
| `.close()` | Close connection |

### Sftpfile

#### Methods

| Method | Description |
|--------|-------------|
| `.mkdir(arg?)` | Create directory |
| `.remove()` | Remove file |
| `.rmdir(arg?)` | Remove directory |

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
| `.split(arg)` | Split by delimiter into array |
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
| `.all(arg)` | Check if all rows match predicate (fn) - returns boolean |
| `.any(arg)` | Check if any row matches predicate (fn) - returns boolean |
| `.appendCol(arg1, arg2)` | Add column at end |
| `.appendRow(arg)` | Add row at end |
| `.avg(arg)` | Average column values |
| `.column(arg)` | Get array of values from column |
| `.columnCount()` | Get number of columns |
| `.count()` | Count rows |
| `.dropCol(arg, ...)` | Remove columns (col1, col2, ...) |
| `.find(arg)` | Find first row matching predicate (fn) - returns row or null |
| `.groupBy(arg1, arg2?)` | Group rows by column(s) (cols, aggregationFn?) |
| `.insertColAfter(arg1, arg2, arg3)` | Insert column after another |
| `.insertColBefore(arg1, arg2, arg3)` | Insert column before another |
| `.insertRowAt(arg1, arg2)` | Insert row at index |
| `.limit(arg1, arg2?)` | Limit rows (count, offset?) |
| `.map(arg)` | Transform each row (fn) - preserves schema if Records returned |
| `.max(arg)` | Maximum column value |
| `.min(arg)` | Minimum column value |
| `.orderBy(arg, ...)` | Sort rows by column(s) |
| `.renameCol(arg1, arg2)` | Rename column (oldName, newName) |
| `.rowCount()` | Get number of rows |
| `.select(arg, ...)` | Select specific columns |
| `.sum(arg)` | Sum column values |
| `.toCSV()` | Convert to CSV string |
| `.toHTML(arg?)` | Convert to HTML table (footer: string\|dict?) |
| `.toJSON()` | Convert to JSON array |
| `.toMarkdown()` | Convert to Markdown table |
| `.unique(arg?)` | Remove duplicate rows (columns?) |
| `.where(arg)` | Filter rows by predicate |

### Tablemodule

#### Methods

| Method | Description |
|--------|-------------|
| `.fromDict(arg)` | Create table from dictionary |

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
| `.origin()` | Get origin (scheme://host:port) |
| `.pathname()` | Get path component |
| `.toDict()` | Convert to dictionary |
| `.toString()` | Convert to string |
| `.withPath(arg)` | Create URL with new path |
| `.withQuery(arg)` | Create URL with query params |

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
| `format(template, values...)` | Format string with placeholders |
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
| `path(pathString)` | Create path from string |

### Regular Expressions

| Function | Description |
|----------|-------------|
| `match(string, pattern, flags?)` | Match string against pattern |
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
| `**` | Raise to power |
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

### Pipe

| Operator | Description |
|----------|-------------|
| `\|>` | Pass left value as first argument to right function |

### Null Handling

| Operator | Description |
|----------|-------------|
| `??` | Return left if not null, otherwise right |

### Control Flow

| Operator | Description |
|----------|-------------|
| `?:` | Conditional expression: condition ? then : else |

## Standard Library

### @std/api

HTTP client for API requests

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

### @std/table

Table data structure with query methods

#### Functions

| Function | Description |
|----------|-------------|
| `table(arg, ...)` | Create table from data |

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

HTTP client for API requests

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
| `Blockquote()` | Blockquote element |
| `Breadcrumb()` | Breadcrumb navigation |
| `Button()` | Button element |
| `Checkbox()` | Single checkbox |
| `CheckboxGroup()` | Checkbox group |
| `DataTable()` | Data table component |
| `Figure()` | Figure with caption |
| `Form()` | Form wrapper |
| `Head()` | HTML head section |
| `Icon()` | Icon element |
| `Iframe()` | Iframe element |
| `Img()` | Image element |
| `LocalTime()` | Localized time display |
| `Nav()` | Navigation wrapper |
| `Page()` | Page layout wrapper |
| `RadioGroup()` | Radio button group |
| `RelativeTime()` | Relative time display |
| `SelectField()` | Select dropdown field |
| `SkipLink()` | Skip to content link |
| `SrOnly()` | Screen reader only text |
| `TextField()` | Text input field |
| `TextareaField()` | Textarea input field |
| `Time()` | Time element |
| `TimeRange()` | Time range display |

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

