## 2. Operators

Parsley provides a rich set of operators for arithmetic, comparison, logical operations, and more.

### 2.1 Arithmetic

| Operator | Description | Example |
|----------|-------------|---------|
| `+` | Addition / String concatenation | `1 + 2` → `3`, `"a" + "b"` → `"ab"` |
| `-` | Subtraction | `5 - 3` → `2` |
| `*` | Multiplication | `4 * 3` → `12` |
| `/` | Division | `10 / 3` → `3.333...` |
| `%` | Modulo | `10 % 3` → `1` |

### 2.2 Comparison

| Operator | Description | Example |
|----------|-------------|---------|
| `==` | Equal | `1 == 1` → `true` |
| `!=` | Not equal | `1 != 2` → `true` |
| `<` | Less than | `1 < 2` → `true` |
| `>` | Greater than | `2 > 1` → `true` |
| `<=` | Less or equal | `1 <= 1` → `true` |
| `>=` | Greater or equal | `2 >= 1` → `true` |

### 2.3 Logical

| Operator | Description | Example |
|----------|-------------|---------|
| `and` | Logical AND | `true and false` → `false` |
| `or` | Logical OR | `true or false` → `true` |
| `not` | Logical NOT | `not true` → `false` |

### 2.4 Null Handling

| Operator | Description | Example |
|----------|-------------|---------|
| `??` | Null coalescing | `null ?? "default"` → `"default"` |
| `?.` | Optional chaining | `obj?.prop` → `null` if obj is null |

### 2.5 Other

| Operator | Description | Example |
|----------|-------------|---------|
| `in` | Membership test | `"a" in ["a", "b"]` → `true` |
| `is` | Schema validation | `value is @schema` |
| `..` | Range | `1..5` → `[1, 2, 3, 4, 5]` |
| `~` | Regex match | `"hello" ~ /^h/` → `true` |