# Lexer Inventory

**Source**: `pkg/parsley/lexer/lexer.go`  
**Extracted**: 2026-01-11

---

## Token Types

### Special Tokens
| Token | Description |
|-------|-------------|
| `ILLEGAL` | Invalid/unrecognized token |
| `EOF` | End of file |

### Identifiers & Literals
| Token | Description | Example |
|-------|-------------|---------|
| `IDENT` | Identifier | `add`, `foobar`, `x` |
| `INT` | Integer literal | `1343456` |
| `FLOAT` | Float literal | `3.14159` |
| `STRING` | Double-quoted string (escapes only) | `"foobar"` |
| `TEMPLATE` | Backtick template string | `` `hello {name}` `` |
| `RAW_TEMPLATE` | Single-quoted raw string with `@{}` | `'raw @{expr}'` |
| `REGEX` | Regular expression | `/pattern/flags` |
| `MONEY` | Money literal | `$12.34`, `EUR#50.00` |

### DateTime Literals
| Token | Description | Example |
|-------|-------------|---------|
| `DATETIME_LITERAL` | Date/time literal | `@2024-12-25T14:30:00Z`, `@12:30` |
| `DATETIME_NOW` | Current datetime | `@now` |
| `TIME_NOW` | Current time | `@timeNow` |
| `DATE_NOW` | Current date | `@dateNow`, `@today` |
| `DURATION_LITERAL` | Duration | `@2h30m`, `@7d`, `@-1w` |

### Path & URL Literals
| Token | Description | Example |
|-------|-------------|---------|
| `PATH_LITERAL` | Filesystem path | `@/usr/local`, `@./config`, `@~/home` |
| `URL_LITERAL` | URL | `@https://example.com` |
| `STDLIB_PATH` | Standard library path | `@std/table`, `@basil/http` |
| `PATH_TEMPLATE` | Interpolated path | `@(./path/{expr}/file)` |
| `URL_TEMPLATE` | Interpolated URL | `@(https://api.com/{id})` |
| `DATETIME_TEMPLATE` | Interpolated datetime | `@(2024-{month}-{day})` |

### Connection/Resource Literals
| Token | Description | Example |
|-------|-------------|---------|
| `SQLITE_LITERAL` | SQLite connection | `@sqlite` |
| `POSTGRES_LITERAL` | PostgreSQL connection | `@postgres` |
| `MYSQL_LITERAL` | MySQL connection | `@mysql` |
| `SFTP_LITERAL` | SFTP connection | `@sftp` |
| `SHELL_LITERAL` | Shell execution | `@shell` |
| `DB_LITERAL` | Generic DB | `@DB` |
| `SEARCH_LITERAL` | Search literal | `@SEARCH` |
| `ENV_LITERAL` | Environment variables | `@env` |
| `ARGS_LITERAL` | CLI arguments | `@args` |
| `PARAMS_LITERAL` | Parameters | `@params` |

### Query DSL Literals
| Token | Description | Example |
|-------|-------------|---------|
| `SCHEMA_LITERAL` | Schema definition | `@schema` |
| `QUERY_LITERAL` | Query DSL | `@query` |
| `INSERT_LITERAL` | Insert DSL | `@insert` |
| `UPDATE_LITERAL` | Update DSL | `@update` |
| `DELETE_LITERAL` | Delete DSL | `@delete` |
| `TRANSACTION_LIT` | Transaction DSL | `@transaction` |

### Tag Tokens (HTML/XML)
| Token | Description | Example |
|-------|-------------|---------|
| `TAG` | Self-closing tag | `<br/>`, `<img src="x"/>` |
| `TAG_START` | Opening tag | `<div>`, `<p class="x">` |
| `TAG_END` | Closing tag | `</div>`, `</p>` |
| `TAG_TEXT` | Raw text in tags | (content inside `<script>`, `<style>`) |

---

## Operators

### Arithmetic
| Symbol | Token | Description |
|--------|-------|-------------|
| `+` | `PLUS` | Addition, string concat, path join |
| `-` | `MINUS` | Subtraction |
| `*` | `ASTERISK` | Multiplication, string/array repeat |
| `/` | `SLASH` | Division, array chunk |
| `%` | `PERCENT` | Modulo |

### Comparison
| Symbol | Token | Description |
|--------|-------|-------------|
| `<` | `LT` | Less than |
| `>` | `GT` | Greater than |
| `<=` | `LTE` | Less than or equal |
| `>=` | `GTE` | Greater than or equal |
| `==` | `EQ` | Equal |
| `!=` | `NOT_EQ` | Not equal |

### Logical
| Symbol | Token | Description |
|--------|-------|-------------|
| `&&` or `&` | `AND` | Logical AND / set intersection |
| `\|\|` or `\|` | `OR` | Logical OR / set union |
| `!` | `BANG` | Logical NOT |
| `and` | `AND` | Keyword alias for `&&` |
| `or` | `OR` | Keyword alias for `\|\|` |
| `not` | `BANG` | Keyword alias for `!` |

### Pattern Matching
| Symbol | Token | Description |
|--------|-------|-------------|
| `~` | `MATCH` | Regex match |
| `!~` | `NOT_MATCH` | Regex not match |

### Nullish
| Symbol | Token | Description |
|--------|-------|-------------|
| `??` | `NULLISH` | Null coalescing |
| `?` | `QUESTION` | Optional/ternary marker |

### Assignment
| Symbol | Token | Description |
|--------|-------|-------------|
| `=` | `ASSIGN` | Assignment |

### File I/O
| Symbol | Token | Description |
|--------|-------|-------------|
| `<==` | `READ_FROM` | Read from file |
| `<=/=` | `FETCH_FROM` | Fetch from URL |
| `==>` | `WRITE_TO` | Write to file |
| `==>>` | `APPEND_TO` | Append to file |

### Database (SQL)
| Symbol | Token | Description |
|--------|-------|-------------|
| `<=?=>` | `QUERY_ONE` | Query single row |
| `<=??=>` | `QUERY_MANY` | Query multiple rows |
| `<=!=>` | `EXECUTE` | Execute mutation |

### Query DSL
| Symbol | Token | Description |
|--------|-------|-------------|
| `\|<` | `PIPE_WRITE` | Pipe write for DSL |
| `?->` | `RETURN_ONE` | Return single result |
| `??->` | `RETURN_MANY` | Return multiple results |
| `.->` | `EXEC_COUNT` | Execute and return count |
| `<-` | `ARROW_PULL` | Subquery pull |

### Process Execution
| Symbol | Token | Description |
|--------|-------|-------------|
| `<=#=>` | `EXECUTE_WITH` | Execute command with input |

### Range
| Symbol | Token | Description |
|--------|-------|-------------|
| `..` | `RANGE` | Range operator (e.g., `1..5`) |

### Other
| Symbol | Token | Description |
|--------|-------|-------------|
| `++` | `PLUSPLUS` | Concatenation (arrays) |

---

## Delimiters

| Symbol | Token | Description |
|--------|-------|-------------|
| `,` | `COMMA` | Comma |
| `;` | `SEMICOLON` | Semicolon |
| `:` | `COLON` | Colon |
| `.` | `DOT` | Dot (member access) |
| `...` | `DOTDOTDOT` | Spread/rest operator |
| `(` | `LPAREN` | Left parenthesis |
| `)` | `RPAREN` | Right parenthesis |
| `{` | `LBRACE` | Left brace |
| `}` | `RBRACE` | Right brace |
| `[` | `LBRACKET` | Left bracket |
| `]` | `RBRACKET` | Right bracket |

---

## Keywords

| Keyword | Token | Description |
|---------|-------|-------------|
| `fn` | `FUNCTION` | Function definition |
| `function` | `FUNCTION` | Alias for `fn` (JS familiarity) |
| `let` | `LET` | Variable declaration |
| `for` | `FOR` | For loop |
| `in` | `IN` | Membership/iteration |
| `as` | `AS` | Alias in imports |
| `true` | `TRUE` | Boolean true |
| `false` | `FALSE` | Boolean false |
| `if` | `IF` | Conditional |
| `else` | `ELSE` | Else branch |
| `return` | `RETURN` | Return from function |
| `export` | `EXPORT` | Export from module |
| `try` | `TRY` | Error handling |
| `import` | `IMPORT` | Import module |
| `check` | `CHECK` | Guard expression |
| `stop` | `STOP` | Break from loop |
| `skip` | `SKIP` | Continue in loop |
| `via` | `VIA` | Schema relations |

**Note**: `and`, `or`, `not` are keyword aliases for `&&`, `||`, `!`

---

## String Types

### Double-Quoted Strings (`"..."`)
- Standard strings with escape sequences
- **NO interpolation** - `{var}` is literal text
- Escapes: `\n`, `\t`, `\\`, `\"`, etc.

### Backtick Template Strings (`` `...` ``)
- **Interpolation** with `{expr}`
- Multi-line supported
- Escapes: `\{` for literal brace

### Single-Quoted Raw Strings (`'...'`)
- Backslashes are literal (no escapes)
- **Interpolation** with `@{expr}` only
- Escape `@` with `\@` for literal
- Perfect for JavaScript embedding: `'Parts.refresh("x", {id: @{id}})'`

---

## Money Literals

### Symbol Formats
| Symbol | Currency | Example |
|--------|----------|---------|
| `$` | USD | `$12.34` |
| `£` | GBP | `£99.99` |
| `€` | EUR | `€50.00` |
| `¥` | JPY | `¥1000` |
| `CA$` | CAD | `CA$25.00` |
| `AU$` | AUD | `AU$25.00` |
| `HK$` | HKD | `HK$25.00` |
| `S$` | SGD | `S$25.00` |
| `CN¥` | CNY | `CN¥100.00` |

### CODE# Format
Any 3-letter ISO currency code followed by `#`:
- `USD#12.34`
- `EUR#50.00`
- `BTC#0.00100000` (8 decimal places for crypto)

### Currency Scales
Zero-decimal currencies (no decimals allowed):
- JPY, KRW, VND, IDR, etc.

Two-decimal currencies (default):
- USD, EUR, GBP, CAD, AUD, etc.

Extended precision:
- BTC: 8 decimals
- ETH: 18 decimals

---

## Path Literals (@)

### Static Paths
| Pattern | Description |
|---------|-------------|
| `@./relative` | Relative to current file |
| `@~/from/root` | Relative to project root |
| `@/absolute` | Absolute filesystem path |
| `@-` | stdin/stdout |
| `@stdin` | Explicit stdin |
| `@stdout` | Explicit stdout |
| `@stderr` | Explicit stderr |

### URLs
| Pattern | Description |
|---------|-------------|
| `@http://...` | HTTP URL |
| `@https://...` | HTTPS URL |
| `@ftp://...` | FTP URL |

### Standard Library
| Pattern | Description |
|---------|-------------|
| `@std/table` | Standard library module |
| `@std/math` | Standard library module |
| `@basil/http` | Basil framework module |

### Interpolated Templates
| Pattern | Description |
|---------|-------------|
| `@(./path/{var}/file)` | Path with interpolation |
| `@(https://api/{id})` | URL with interpolation |
| `@(2024-{month}-{day})` | DateTime with interpolation |

---

## DateTime Literals (@)

### Dates & Times
| Pattern | Description |
|---------|-------------|
| `@2024-12-25` | Date |
| `@2024-12-25T14:30:00` | DateTime |
| `@2024-12-25T14:30:00Z` | DateTime UTC |
| `@14:30` | Time only |
| `@14:30:00` | Time with seconds |

### Special Values
| Literal | Description |
|---------|-------------|
| `@now` | Current datetime |
| `@today` | Current date |
| `@dateNow` | Current date (alias) |
| `@timeNow` | Current time |

### Durations
| Pattern | Description |
|---------|-------------|
| `@1d` | 1 day |
| `@2h30m` | 2 hours 30 minutes |
| `@1w` | 1 week |
| `@1y6mo` | 1 year 6 months |
| `@-1d` | Negative: 1 day ago |
| `@-2w` | Negative: 2 weeks ago |

**Duration units**: `y` (year), `mo` (month), `w` (week), `d` (day), `h` (hour), `m` (minute), `s` (second)

---

## Regular Expressions

Format: `/pattern/flags`

| Flag | Description |
|------|-------------|
| `i` | Case insensitive |
| `m` | Multiline |
| `s` | Dotall (`.` matches newline) |
| `g` | Global (all matches) |

Example: `/\d+/g`, `/hello/i`

---

## Comments

```parsley
// Single-line comment only
// No block comments
```

---

## Lexer State Notes

- **Tag depth tracking**: Lexer tracks nesting for proper `TAG_END` matching
- **Raw text tags**: `<script>` and `<style>` contents are lexed as `TAG_TEXT`
- **Regex context**: `/` is treated as regex when following operators/keywords, not when following values
- **State save/restore**: Lexer supports save/restore for lookahead
