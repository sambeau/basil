## 1. Literals

### 1.1 Numbers

#### Integers

```parsley
42
-15
0
```

#### Floats

```parsley
3.14159
-2.718
0.5
```

---

### 1.2 Strings

#### Double-Quoted Strings (`"..."`)

Standard strings with escape sequences. **No interpolation**.

```parsley
"Hello, World!"
"Line1\nLine2"
"Tab\there"
"Quote: \"hi\""
"Backslash: \\"
```

| Escape | Meaning |
|--------|---------|
| `\n` | Newline |
| `\t` | Tab |
| `\r` | Carriage return |
| `\\` | Backslash |
| `\"` | Double quote |

#### Template Strings (`` `...` ``)

Interpolated strings using `{expression}` syntax. Any valid Parsley expression can be used inside the braces.

**Note**: Unlike JavaScript template literals, Parsley uses `{expr}` not `${expr}`.

```parsley
let name = "Alice"
`Hello, {name}!`        // "Hello, Alice!"
`2 + 2 = {2 + 2}`       // "2 + 2 = 4"
`{name.toUpper()}`      // "ALICE"
```

#### Raw Strings (`'...'`)

Backslashes are literal (no escape sequences). Interpolation only with `@{expression}`.

**Use for**: Regular expressions, file paths, SQL patterns.

```parsley
'C:\Users\name'                 // Backslashes literal
'regex: \d+\.\d+'               // No escaping needed
let id = 42
'id = @{id}'                    // "id = 42"
```

---

### 1.3 Booleans and Null

```parsley
true
false
null
```

#### Truthiness

Parsley has Python-style truthiness:

**Falsy values:**
- `false`
- `null`
- `0` (integer)
- `0.0` (float)
- `""` (empty string)
- `[]` (empty array)
- `{}` (empty dictionary)

**Everything else is truthy**, including:
- `true`
- Non-zero numbers
- Non-empty strings, arrays, and dictionaries

This matches the behavior of Python, PHP, and Perl, and avoids JavaScript's confusing inconsistency where empty arrays/objects are truthy. The design makes Parsley more intuitive for common web development patterns like form validation and collection checking:

```parsley
// Intuitive for form validation
if (username) { ... }      // fails for ""
if (items) { ... }         // fails for []
if (config) { ... }        // fails for {}
if (count) { ... }         // fails for 0
```

---

### 1.4 Arrays

Arrays are ordered, zero-indexed collections that can hold values of any type.

```parsley
[1, 2, 3]
let empty = []
```

#### Nested Arrays

Arrays can contain other arrays, useful for matrices and tabular data:

```parsley
let matrix = [[1, 2, 3], [4, 5, 6], [7, 8, 9]]
matrix[1][0]                    // 4 (second row, first column)
matrix[0]                       // [1, 2, 3] (first row)
```

#### Indexing

```parsley
let arr = [10, 20, 30, 40, 50]
arr[0]                          // 10 (first element)
arr[-1]                         // 50 (last element)
arr[1:3]                        // [20, 30] (slice)
arr[:2]                         // [10, 20] (first 2)
arr[2:]                         // [30, 40, 50] (from index 2)
arr[?99]                        // null (optional, no error)
```

#### Destructuring

```parsley
let [a, b, c] = [1, 2, 3]       // a=1, b=2, c=3
let [first, ...rest] = [1, 2, 3, 4]  // first=1, rest=[2,3,4]
```

---

### 1.5 Dictionaries

Dictionaries are unordered key-value collections. Keys must be strings (quotes optional if valid identifiers).

```parsley
{name: "Alice", age: 30}
let emptyDict = {}
```

#### Access

```parsley
let person = {name: "Bob", age: 25}
person.name                     // "Bob"
person["age"]                   // 25
person.missing                  // null (no error)
```

#### Destructuring

Extract specific fields by name:

```parsley
let {name, age} = person        // Extract fields
```

Use `...rest` to collect remaining fields into a new dictionary:

```parsley
let person = {name: "Bob", age: 25, city: "NYC"}
let {name, ...rest} = person    // name="Bob", rest={age: 25, city: "NYC"}
```

---

### 1.6 Functions

Functions are defined with the `fn` keyword (or `function` as an alias).

**Note**: Arrow function syntax (`x => x * 2`) is **not supported**. Always use `fn(x) { x * 2 }`.

```parsley
let double = fn(x) { x * 2 }
double(5)                       // 10

let add = fn(a, b) { a + b }
add(3, 4)                       // 7

let constant = fn() { 42 }
constant()                      // 42
```

**Implicit return**: The last expression is returned automatically. Unlike JavaScript, you don't need `return` for single expressions.

```parsley
let complex = fn(x) {
    let y = x * 2
    y + 1                       // Returns this
}
complex(10)                     // 21
```

---

### 1.7 DateTime Literals

All datetime literals start with `@`:

```parsley
@2024-12-25                     // Date only
@2024-12-25T14:30:00            // DateTime
@14:30                          // Time only
@14:30:45                       // Time with seconds
```

#### Special Values

```parsley
@now                            // Current datetime
@today                          // Current date
```

#### Interpolated DateTime

```parsley
let month = 12
let day = 25
@(2024-{month}-{day})           // Dynamic construction
```

---

### 1.8 Duration Literals

```parsley
@1d                             // 1 day
@2h30m                          // 2 hours 30 minutes
@1w                             // 1 week
@1y6mo                          // 1 year 6 months
@-1d                            // Negative: 1 day ago
```

| Unit | Meaning |
|------|---------|
| `y` | Year |
| `mo` | Month |
| `w` | Week |
| `d` | Day |
| `h` | Hour |
| `m` | Minute |
| `s` | Second |

---

### 1.9 Money Literals

Parsley has first-class support for monetary values with precise decimal handling.

#### Symbol Format

```parsley
$12.34                          // USD
$99.99
```

#### Unicode Currency Symbols

Parsley recognizes common currency symbols directly:

```parsley
€123.45                         // Euro (EUR)
£99.99                          // British Pound (GBP)
¥5000                           // Japanese Yen (JPY)
```

#### Compound Symbols

```parsley
CA$50.00                        // Canadian Dollar (CAD)
AU$75.00                        // Australian Dollar (AUD)
HK$100.00                       // Hong Kong Dollar (HKD)
S$88.00                         // Singapore Dollar (SGD)
CN¥200.00                       // Chinese Yuan (CNY)
```

#### CODE# Format

For currencies without symbols, use the ISO 4217 code followed by `#`:

```parsley
EUR#50.00                       // Euro
GBP#25.00                       // British Pound
CHF#100.00                      // Swiss Franc
INR#1000.00                     // Indian Rupee
```

---

### 1.10 Regex Literals

Regex literals must be assigned to a variable or used in an expression context.

```parsley
let r = /hello/
let digits = /\d+/
let caseInsensitive = /pattern/i
```

| Flag | Meaning |
|------|---------|
| `i` | Case insensitive |
| `m` | Multiline |
| `s` | Dotall (`.` matches newline) |
| `g` | Global (all matches) |

---

### 1.11 Path Literals

```parsley
@./relative/path                // Relative to current file
@~/from/root                    // Relative to project root
```

#### Interpolated Paths

```parsley
let file = "config"
@(./data/{file}.json)           // ./data/config.json
```

---

### 1.12 URL Literals

```parsley
@https://example.com
@http://localhost:3000
```

#### Interpolated URLs

```parsley
let id = 123
@(https://api.example.com/users/{id})
```

---

### 1.13 Standard Library Paths

```parsley
@std/math
@std/valid
@std/id
```

---

### 1.14 Table Literals

Table literals create structured tabular data with named columns. Rows are always dictionary literals; every row must have the same keys as the first row. A schema may be supplied to validate rows and fill in defaults.

#### Basic Syntax

```parsley
// From array of dictionaries (columns inferred from keys)
@table [
    {name: "Alice", age: 30},
    {name: "Bob", age: 25}
]
```

#### With Schema

Tables can reference a schema for validation and defaults:

```parsley
@schema Person { name: string, age: integer = 0 }

// Schema validates rows and fills in defaults for omitted columns
@table(Person) [
    {name: "Alice"},
    {name: "Bob"}              // neither row gives age, so both get 0
]
```

#### Empty Tables

```parsley
// Empty table with no columns
@table []

// Empty table carrying a schema (no columns until rows are added)
@table(Person) []
```

---

### 1.15 Schema Literals

Schemas define the structure of records and tables. They specify field names, types, validation rules, default values, and metadata. Schemas are used for database table bindings, form validation, and typed data structures.

#### Basic Syntax

```parsley
@schema Person {
    name: string
    age: integer
    email: email
}
```

#### Field Types

| Type | Description | SQL Type |
|------|-------------|----------|
| `string` | Text data | `TEXT` |
| `text` | Long text data | `TEXT` |
| `int`, `integer` | Whole numbers | `INTEGER` |
| `bigint` | Large integers | `BIGINT` |
| `float`, `number` | Decimal numbers | `REAL` |
| `bool`, `boolean` | True/false | `INTEGER` (0/1) |
| `datetime` | Date and time | `DATETIME` |
| `date` | Date only | `DATE` |
| `time` | Time only | `TIME` |
| `money` | Monetary values | `REAL` |
| `uuid` | UUID strings | `TEXT` |
| `ulid` | ULID strings | `TEXT` |
| `json` | JSON data | `TEXT` |
| `email` | Email (validated) | `TEXT` |
| `url` | URL (validated) | `TEXT` |
| `phone` | Phone number | `TEXT` |
| `slug` | URL slug (validated) | `TEXT` |
| `enum` | One of specified values | `TEXT` |
| `mass` | Mass unit (kg, lb, etc.) | `TEXT` |
| `length` | Length unit (m, ft, etc.) | `TEXT` |
| `data` | Data unit (B, MB, etc.) | `TEXT` |
| `temperature` | Temperature (C, F, K) | `TEXT` |
| `volume` | Volume unit (L, gal, etc.) | `TEXT` |
| `area` | Area unit (m2, ft2, etc.) | `TEXT` |
| `unit(...)` | Unit with constraints | `TEXT` |

#### Nullable Fields

Append `?` to make a field nullable:

```parsley
@schema User {
    name: string           // Required
    nickname: string?      // Optional (nullable)
    email: email?          // Optional email
}
```

#### Default Values

Use `=` to specify default values:

```parsley
@schema Post {
    title: string
    status: string = "draft"
    views: integer = 0
    published: boolean = false
    createdAt: datetime = @now
}
```

#### Enum Types

Define allowed values inline:

```parsley
@schema Task {
    title: string
    priority: enum["low", "medium", "high"] = "medium"
    status: enum["todo", "in-progress", "done"]
}
```

#### Type Constraints

Add constraints using `(key: value)` syntax:

```parsley
@schema Profile {
    username: string(min: 3, max: 20, unique: true)
    age: integer(min: 0, max: 150)
    bio: text(max: 500)
}
```

| Constraint | Applies To | Description |
|------------|------------|-------------|
| `min` | string, integer | Minimum length or value |
| `max` | string, integer | Maximum length or value |
| `pattern` | string | Regex pattern for validation |
| `required` | any | Field must have a non-null value |
| `auto` | any | Database/server generates this value |
| `readOnly` | any | Field cannot be set from client/form input |
| `unique` | any | UNIQUE constraint in SQL |
| `suffix` | unit | Specific unit suffix required (e.g., `"kg"`) |
| `family` | unit | Unit family required (e.g., `"mass"`) |

#### Unit Types in Schemas

Schemas can declare fields as unit types. Use the family name directly, or use `unit()` with constraints:

```parsley
// Family names as types — accepts any unit of that family
@schema Product {
    weight: mass              // accepts #5kg, #2.2lb, #500g, etc.
    height: length            // accepts #1.8m, #6ft, #180cm, etc.
    storage: data             // accepts #1GB, #500MB, etc.
    temp: temperature         // accepts #100C, #212F, #373K
    capacity: volume          // accepts #2L, #1gal, etc.
    floor: area               // accepts #100m2, #1000ft2, etc.
}

// Specific unit constraint — requires exact suffix
@schema MetricProduct {
    weight: unit(suffix: "kg")     // must be in kg
    height: unit(suffix: "cm")     // must be in cm
}

// Family constraint via unit() — equivalent to family name
@schema AnyProduct {
    weight: unit(family: "mass")   // same as `mass`
}
```

**Validation behavior:**
- Family types (`mass`, `length`, etc.) accept any unit of that family
- `unit(suffix: "...")` requires the exact suffix — validation fails on mismatch
- Non-unit values (numbers, strings) are rejected

```parsley
@schema Product { weight: mass }
Product({weight: #5kg}).validate().isValid()    // true
Product({weight: #2.2lb}).validate().isValid()  // true
Product({weight: #5m}).validate().isValid()     // false (length ≠ mass)
Product({weight: 5}).validate().isValid()       // false (number ≠ unit)

@schema Metric { height: unit(suffix: "cm") }
Metric({height: #180cm}).validate().isValid()   // true
Metric({height: #1.8m}).validate().isValid()    // false (m ≠ cm)
```

#### The `auto` Constraint

The `auto` constraint marks fields whose values are generated by the database or server (e.g., auto-increment IDs, timestamps). Auto fields are skipped during validation and are immutable on updates.

```parsley
@schema User {
    id: id(auto)                     // Database generates on insert
    createdAt: datetime(auto)        // Server sets on insert
    updatedAt: datetime(auto)        // Server sets on insert/update
    name: string(required)
}

// Valid - id and timestamps are auto, don't need to be provided
let user = User({name: "Alice"})
user.validate().isValid()            // true

// Error - cannot update auto fields
user.update({id: "new-id"})          // Error: cannot update auto field 'id'
```

**Note:** `auto` and `required` cannot be combined on the same field.

#### Field Metadata (Pipe Syntax)

Add UI metadata using the pipe `|` syntax:

```parsley
@schema Contact {
    name: string | {title: "Full Name", placeholder: "Enter your name"}
    email: email | {title: "Email Address", hidden: false}
    notes: text | {title: "Notes", placeholder: "Optional notes...", hidden: true}
}
```

Common metadata keys:
- `title` — Display label for forms/tables
- `placeholder` — Input placeholder text
- `hidden` — Hide field in auto-generated UIs

#### Complete Example

```parsley
@schema User {
    id: id(auto)
    username: string(min: 3, max: 30, unique: true) | {title: "Username"}
    email: email(unique: true) | {title: "Email Address", placeholder: "user@example.com"}
    password: string | {hidden: true}
    role: enum["user", "admin", "moderator"] = "user" | {title: "Role"}
    bio: text? | {title: "Biography", placeholder: "Tell us about yourself..."}
    active: boolean = true
    createdAt: datetime(auto) | {title: "Created", hidden: true}
}
```

---

### 1.16 Unit Literals

Parsley has first-class support for measurement units with exact integer arithmetic. Unit literals use the `#` sigil followed by a numeric value and a unit suffix.

#### Basic Syntax

```parsley
#12m                            // 12 metres
#5.5kg                          // 5.5 kilograms
#100cm                          // 100 centimetres
#1024B                          // 1024 bytes
```

#### Fraction Syntax (US Customary)

US Customary units support exact fractions:

```parsley
#3/8in                          // 3/8 of an inch (exact)
#1/2lb                          // half a pound (exact)
```

#### Mixed Number Syntax

Use `+` as the mixed-number separator:

```parsley
#92+5/8in                       // 92 and 5/8 inches
#2+3/8in                        // 2 and 3/8 inches
```

#### Negative Literals

```parsley
#-6m                            // negative 6 metres (canonical)
-#6m                            // also negative 6 metres (unary negation)
```

#### Supported Suffixes

- **Length — SI:** `mm`, `cm`, `m`, `km`
- **Length — US:** `in`, `ft`, `yd`, `mi`
- **Mass — SI:** `mg`, `g`, `kg`
- **Mass — US:** `oz`, `lb`
- **Data — Decimal:** `B`, `kB`, `MB`, `GB`, `TB`
- **Data — Binary:** `KiB`, `MiB`, `GiB`, `TiB`
- **Temperature:** `K` (kelvin), `C` (Celsius), `F` (Fahrenheit)
- **Volume — SI:** `mL`, `L`, `kL`
- **Volume — US:** `floz`, `cup`, `pt`, `qt`, `gal`
- **Area — SI:** `mm2`, `cm2`, `m2`, `km2`
- **Area — US:** `in2`, `ft2`, `yd2`, `ac` (acre), `mi2`

#### Internal Representation

- **SI units** are stored as an integer count of fixed sub-units (µm for length, mg for mass, B for data, µL for volume). `#12.3m` = 12,300,000 µm internally. Within-system arithmetic is always exact.
- **US Customary units** are stored as an integer numerator over a fixed denominator (HCN = 725,760). Common fractions (halves through sixty-fourths, plus thirds, fifths, sevenths) are all exact integers. Volume uses sub-floz (1 floz = HCN).
- **Temperature** uses a unified sub-kelvin representation (1 K = 900 sub-K, 1°F = 500 sub-K). All three scales (K, C, F) are stored as a single int64 sub-kelvin count, making conversions exact: `#0C == #32F`, `#100C == #212F`, `#-40C == #-40F`.
- **Area** uses 1 mm² (SI) and 1 in² (US) as base sub-units. For large areas, a decimal Scale is applied automatically: true value = Amount × 10^Scale sub-units. This allows representation of planetary-scale surfaces (Earth ≈ 510 million km²) while preserving exact mm² precision for everyday values. US area always displays as decimal (no fractions).
- **SI fractions** are syntactic sugar for division: `#1/3m` truncates to 333,333 µm. US fractions like `#1/3in` and `#1/3cup` are stored exactly.

#### Temperature Literals

Temperature literals support all three scales. Celsius (`C`) and Kelvin (`K`) are SI; Fahrenheit (`F`) is US:

```parsley
#100C                           // 100 degrees Celsius
#212F                           // 212 degrees Fahrenheit
#0K                             // absolute zero
#37.5C                          // decimal temperature
#-40C                           // negative (same as #-40F)
#-273.15C                       // absolute zero in Celsius
```

> **Temperature arithmetic restriction**: multiplication and division of temperatures are not allowed (`#20C * 2` → error). Temperature scales have arbitrary zero points, so these operations are physically meaningless. Use addition/subtraction instead.

#### Volume Literals

Volume follows the standard dual-system pattern. US volume supports exact fractions:

```parsley
#500mL                          // 500 millilitres
#2.5L                           // 2.5 litres
#1kL                            // 1 kilolitre (= 1000 L)
#8floz                          // 8 fluid ounces
#1cup                           // 1 cup (= 8 floz)
#1/3cup                         // exact fraction
#1+1/2gal                       // mixed number: 1.5 gallons
```

#### Area Literals

Area uses `2` suffixes for SI and named units for US. US area displays as decimal only (no fractions):

```parsley
#100m2                          // 100 square metres
#5.5km2                         // 5.5 square kilometres
#1500ft2                        // 1500 square feet
#640ac                          // 640 acres (= 1 mi²)
#1mi2                           // 1 square mile
#2.5ac                          // 2.5 acres (decimal display)
```

#### Cross-System Conversion

Units in the same family (e.g., length) can be mixed. The left operand's system wins:

```parsley
#1cm + #1in                     // #3.54cm (result in SI)
#1in + #1cm                     // result in inches (US)
#1in == #25.4mm                 // true
#1L + #1floz                    // result in litres (SI)
#100m2 + #100ft2                // result in m² (SI)
```
