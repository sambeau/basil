# Parsley Operators Inventory

> Generated from source code audit of `pkg/parsley/evaluator/eval_infix.go`, `pkg/parsley/evaluator/eval_operators.go`, and `pkg/parsley/lexer/lexer.go`

## Prefix Operators

| Operator | Types | Description |
|----------|-------|-------------|
| `!` / `not` | any | Logical negation (truthy → false, falsy → true) |
| `-` | integer, float, money | Unary minus (negation) |

---

## Arithmetic Operators

| Operator | Left | Right | Result | Description |
|----------|------|-------|--------|-------------|
| `+` | integer | integer | integer | Addition |
| `+` | float | float | float | Addition |
| `+` | integer | float | float | Mixed addition |
| `+` | string | any | string | String concatenation (coerces right) |
| `+` | any | string | string | String concatenation (coerces left) |
| `+` | path | path | path | Path join |
| `+` | path | string | path | Path append segment |
| `+` | url | string | url | URL append path |
| `+` | datetime | duration | datetime | Add duration to datetime |
| `+` | duration | datetime | datetime | Add duration to datetime (commutative) |
| `+` | duration | duration | duration | Add durations |
| `+` | money | money | money | Add money (same currency) |
| `-` | integer | integer | integer | Subtraction |
| `-` | float | float | float | Subtraction |
| `-` | integer | float | float | Mixed subtraction |
| `-` | array | array | array | Set difference |
| `-` | dict | dict | dict | Dictionary subtraction (remove keys) |
| `-` | datetime | datetime | duration | Time difference |
| `-` | datetime | duration | datetime | Subtract duration |
| `-` | datetime | integer | datetime | Subtract days |
| `-` | duration | duration | duration | Subtract durations |
| `-` | money | money | money | Subtract money (same currency) |
| `*` | integer | integer | integer | Multiplication |
| `*` | float | float | float | Multiplication |
| `*` | integer | float | float | Mixed multiplication |
| `*` | string | integer | string | String repetition |
| `*` | array | integer | array | Array repetition |
| `*` | money | integer | money | Scale money |
| `*` | money | float | money | Scale money |
| `*` | integer | money | money | Scale money (commutative) |
| `*` | float | money | money | Scale money (commutative) |
| `/` | integer | integer | integer | Integer division |
| `/` | float | float | float | Division |
| `/` | integer | float | float | Mixed division |
| `/` | array | integer | array[] | Array chunking (split into groups) |
| `/` | money | integer | money | Divide money |
| `/` | money | float | money | Divide money |
| `%` | integer | integer | integer | Modulo |

---

## Comparison Operators

| Operator | Behavior | Description |
|----------|----------|-------------|
| `==` | Structural equality | Equal |
| `!=` | Structural inequality | Not equal |
| `<` | Type-specific | Less than |
| `>` | Type-specific | Greater than |
| `<=` | Type-specific | Less than or equal |
| `>=` | Type-specific | Greater than or equal |

### String Comparison
- Uses **natural sort order** (numbers embedded in strings sort numerically)
- Example: `"file2" < "file10"` is `true`

### Datetime Comparison
- Compares Unix timestamps

### Money Comparison
- Must be same currency
- Compares minor units (cents)

---

## Logical/Set Operators

| Operator | Left | Right | Result | Description |
|----------|------|-------|--------|-------------|
| `&&` / `&` / `and` | any | any | boolean | Logical AND (short-circuit) |
| `&&` / `&` / `and` | array | array | array | Set intersection |
| `&&` / `&` / `and` | dict | dict | dict | Dictionary intersection |
| `&&` / `&` / `and` | datetime | datetime | datetime | Combine date + time |
| `\|\|` / `\|` / `or` | any | any | boolean | Logical OR (short-circuit) |
| `\|\|` / `\|` / `or` | array | array | array | Set union |

### Datetime Intersection Rules

`&&` combines date and time components:

| Left | Right | Result |
|------|-------|--------|
| date | time | datetime (combine) |
| time | date | datetime (combine) |
| datetime | time | datetime (replace time) |
| datetime | date | datetime (replace date) |
| date | date | **ERROR** (ambiguous) |
| time | time | **ERROR** (ambiguous) |
| datetime | datetime | **ERROR** (ambiguous) |

---

## Membership Operators

| Operator | Left | Right | Result | Description |
|----------|------|-------|--------|-------------|
| `in` | any | array | boolean | Array contains element |
| `in` | string | string | boolean | String contains substring |
| `in` | string | dict | boolean | Dictionary has key |
| `not in` | any | any | boolean | Negation of `in` |

---

## Pattern Matching Operators

| Operator | Left | Right | Result | Description |
|----------|------|-------|--------|-------------|
| `~` | string | regex | array/null | Match, return captures or null |
| `!~` | string | regex | boolean | Not match, return true/false |

---

## Range Operator

| Operator | Left | Right | Result | Description |
|----------|------|-------|--------|-------------|
| `..` | integer | integer | array | Inclusive range |

Example: `1..5` → `[1, 2, 3, 4, 5]`

---

## Concatenation Operator

| Operator | Left | Right | Result | Description |
|----------|------|-------|--------|-------------|
| `++` | array | array | array | Array concatenation |
| `++` | string | string | string | String concatenation |

**Note**: `++` differs from `+` for arrays. `+` with array+string coerces, `++` preserves types.

---

## Null Coalescing

| Operator | Description |
|----------|-------------|
| `??` | Return left if not null, otherwise right |

Example: `value ?? "default"`

---

## Optional Chaining

| Operator | Description |
|----------|-------------|
| `?.` | Optional property access (return null if base is null) |
| `?[` | Optional index access (return null if base is null) |

Example: `user?.name`, `items?[0]`

---

## File I/O Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `<==` | Read from file | `data <== @./file.json` |
| `<=/=` | Fetch from URL | `data <=/= @https://api.com/data` |
| `==>` | Write to file | `data ==> @./output.txt` |
| `==>>` | Append to file | `line ==>> @./log.txt` |

---

## Database Operators

| Operator | Description | Returns |
|----------|-------------|---------|
| `<=?=>` | Query single row | dict or null |
| `<=??=>` | Query multiple rows | array of dicts |
| `<=!=>` | Execute mutation | result dict |

Example:
```parsley
db = @sqlite(@./app.db)
user = db <=?=> "SELECT * FROM users WHERE id = ?"(id)
users = db <=??=> "SELECT * FROM users"
result = db <=!=> "INSERT INTO users (name) VALUES (?)"(name)
```

---

## Query DSL Operators

| Operator | Description |
|----------|-------------|
| `\|<` | Pipe write (DSL composition) |
| `?->` | Return single result |
| `??->` | Return multiple results |
| `.->` | Execute and return count |
| `<-` | Subquery pull |

---

## Process Execution

| Operator | Description |
|----------|-------------|
| `<=#=>` | Execute command with input |

---

## Precedence Table

From parser (highest to lowest):

| Level | Operators |
|-------|-----------|
| 1 | `??` (null coalescing) |
| 2 | `\|\|`, `or` (logical OR) |
| 3 | `&&`, `and` (logical AND) |
| 4 | `==`, `!=` (equality) |
| 5 | `<`, `>`, `<=`, `>=`, `in`, `not in` (comparison) |
| 6 | `~`, `!~` (pattern matching) |
| 7 | `++`, `..` (concatenation, range) |
| 8 | `+`, `-` (additive) |
| 9 | `*`, `/`, `%` (multiplicative) |
| 10 | `-`, `!`, `not` (prefix/unary) |
| 11 | `?.`, `.`, `[`, `(` (call/access) |

---

## Type Coercion Rules

### String Concatenation with `+`
When either operand is a string:
- Numbers → decimal string
- Booleans → `"true"` / `"false"`
- Null → `"null"`
- Arrays → JSON-like representation
- Dicts → JSON-like representation

### Truthy/Falsy Values
For logical operators (`&&`, `||`, `!`):

**Falsy**:
- `null`
- `false`
- `0` (integer)
- `0.0` (float)
- `""` (empty string)
- `[]` (empty array)

**Truthy**: Everything else
