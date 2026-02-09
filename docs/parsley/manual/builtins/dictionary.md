---
id: dictionary
title: Dictionaries
system: parsley
type: builtin
name: dictionary
created: 2025-01-20
version: 0.17.0
author: Basil Team
keywords:
  - dictionary
  - dict
  - object
  - key-value
  - map
  - hash
  - ordered
  - record
  - schema
  - JSON
---

# Dictionaries

A **Dictionary** is an ordered collection of key-value pairs. It is the fundamental data structure for representing structured data in Parsley—everything from simple configuration objects to database rows, JSON responses, and template contexts.

```parsley
let person = {name: "Alice", age: 30, city: "London"}
person.name
```

**Result:** `"Alice"`

**Key characteristics:**

- **Ordered**: Dictionaries preserve insertion order. Keys are always iterated in the order they were added.
- **String keys**: All keys are strings. Unquoted identifiers are automatically treated as strings.
- **Any values**: Values can be any Parsley type—numbers, strings, arrays, other dictionaries, functions, etc.
- **Dot and bracket access**: Access values with `dict.key` or `dict["key"]`.
- **Foundation for other types**: Records, Tables, and many special types (datetime, path, url) are built on dictionaries.

---

## Creating Dictionaries

### Basic Syntax

Use curly braces with `key: value` pairs separated by commas:

```parsley
{name: "Bob", score: 100}
```

**Result:** `{name: "Bob", score: 100}`

Keys can be unquoted identifiers or quoted strings:

```parsley
{firstName: "Ada", "last-name": "Lovelace", "with spaces": true}
```

**Result:** `{firstName: "Ada", "last-name": "Lovelace", "with spaces": true}`

### Empty Dictionary

```parsley
{}
```

**Result:** `{}`

### Nested Dictionaries

Dictionaries can contain other dictionaries:

```parsley
{
    user: {name: "Alice", id: 42},
    settings: {theme: "dark", notifications: true},
}
```

**Result:** `{user: {name: "Alice", id: 42}, settings: {theme: "dark", notifications: true}}`

### Computed Keys

Use square brackets `[expr]` for keys determined at runtime:

```parsley
let field = "email"
{name: "Alice", [field]: "alice@example.com"}
```

**Result:** `{name: "Alice", email: "alice@example.com"}`

Computed keys can use variables determined at runtime:

```parsley
let keyName = "status"
{id: 1, [keyName]: "active"}
```

**Result:** `{id: 1, status: "active"}`

### Function Values and `this`

Dictionaries can contain functions as values. When a function is called as a method on a dictionary, `this` is automatically bound to the dictionary:

```parsley
let user = {name: "Sam", greet: fn() { "Hi, " + this.name }}
user.greet()
```

**Result:** `"Hi, Sam"`

Methods can accept arguments and reference multiple properties:

```parsley
let calc = {value: 10, add: fn(x) { this.value + x }}
calc.add(5)
```

**Result:** `15`

Methods can call other methods on the same dictionary:

```parsley
let p = {name: "Bob", greet: fn() { "Hi " + this.name }, hello: fn() { this.greet() + "!" }}
p.hello()
```

**Result:** `"Hi Bob!"`

Built-in methods (`.keys()`, `.has()`, etc.) continue to work alongside user-defined methods.

---

## Accessing Values

### Dot Notation

Access values using dot notation when the key is a valid identifier:

```parsley
let config = {host: "localhost", port: 8080}
config.host
```

**Result:** `"localhost"`

### Bracket Notation

Use bracket notation for any key, including those with special characters:

```parsley
let data = {"my-key": "value", "with spaces": 42}
data["my-key"]
```

**Result:** `"value"`

Bracket notation also allows dynamic key access:

```parsley
let data = {a: 1, b: 2, c: 3}
let key = "b"
data[key]
```

**Result:** `2`

### Missing Keys

Accessing a missing key returns `null`:

```parsley
let d = {name: "Alice"}
d.missing
```

**Result:** `null`

### Safe Access with `?`

Use optional access `[?key]` for explicit null-safety (behaves the same as normal access but makes intent clearer):

```parsley
let d = {a: 1}
d[?"missing"]
```

**Result:** `null`

---

## Operators

### `++` (Concatenation / Merge)

Merge two dictionaries with the `++` operator. When keys conflict, the right dictionary's values win:

```parsley
{a: 1, b: 2} ++ {b: 3, c: 4}
```

**Result:** `{a: 1, b: 3, c: 4}`

The result preserves order: keys from the left dictionary come first (in their original order), followed by new keys from the right:

```parsley
{z: 1, a: 2} ++ {m: 3, a: 99}
```

**Result:** `{z: 1, a: 99, m: 3}`

### `in` (Key Membership)

Test if a key exists using the `in` operator:

```parsley
let user = {name: "Alice", role: "admin"}
"name" in user
```

**Result:** `true`

```parsley
"email" in user
```

**Result:** `false`

The negated form `not in` is also available:

```parsley
"email" not in user
```

**Result:** `true`

**Note:** The `in` operator checks for keys, not values. To check if a value is in the dictionary, use `.values()` with array membership.

### `&` (Intersection)

Create a new dictionary containing only keys present in both dictionaries:

```parsley
{a: 1, b: 2, c: 3} & {b: 20, c: 30, d: 40}
```

**Result:** `{b: 2, c: 3}`

The values come from the left dictionary.

### `-` (Subtraction)

Remove keys present in the right dictionary from the left:

```parsley
{a: 1, b: 2, c: 3} - {b: 0, c: 0}
```

**Result:** `{a: 1}`

The values in the right dictionary are ignored—only the keys matter.

---

## Destructuring

Extract values from a dictionary into variables using destructuring patterns.

### Basic Destructuring

```parsley
let {name, age} = {name: "Alice", age: 30, city: "London"}
name
```

**Result:** `"Alice"`

### Renaming with `as`

Rename variables during destructuring:

```parsley
let {name as userName, age as userAge} = {name: "Bob", age: 25}
userName
```

**Result:** `"Bob"`

### Rest Operator

Capture remaining keys with `...rest`:

```parsley
let {id, ...rest} = {id: 1, name: "Alice", active: true}
rest
```

**Result:** `{name: "Alice", active: true}`

### Nested Destructuring

Destructure nested dictionaries:

```parsley
let {user: {name}} = {user: {name: "Alice", id: 1}}
name
```

**Result:** `"Alice"`

### Default Values

Missing keys become `null`:

```parsley
let {name, missing} = {name: "Alice"}
missing
```

**Result:** `null`

---

## Iteration

### For Loops

Iterate over a dictionary with `for`. The loop provides key and value:

```parsley
let scores = {alice: 95, bob: 87, carol: 92}
for (name, score in scores) {
    name ++ ": " ++ score
}
```

**Result:** `["alice: 95", "bob: 87", "carol: 92"]`

Keys are returned in insertion order.

### Single Variable Iteration

With a single variable, you get the value:

```parsley
for (v in {a: 1, b: 2}) { v }
```

**Result:** `[1, 2]`

To iterate over keys only, use `.keys()`:

```parsley
for (k in {a: 1, b: 2}.keys()) { k }
```

**Result:** `["a", "b"]`

---

## Methods

### as()

Convert a dictionary to a Record by applying a schema:

```parsley
@schema User {
    name: string(required)
    age: int
}
let data = {name: "Alice", age: 30}
let user = data.as(User)
user.isValid()
```

**Result:** `false`

The `.as()` method creates an unvalidated Record. Call `.validate()` on the result to perform validation:

```parsley
data.as(User).validate().isValid()
```

**Result:** `true`

**See also:** [Records](record.md), [Schema](schema.md)

---

### reorder()

Reorder and optionally rename dictionary keys. Returns a new dictionary.

**With an array argument** — select and reorder keys:

```parsley
let d = {a: 1, b: 2, c: 3, d: 4}
d.reorder(["c", "a"])
```

**Result:** `{c: 3, a: 1}`

Only keys listed in the array are included, in the specified order. Keys not in the array are dropped.

**With a dictionary argument** — rename and reorder keys:

```parsley
let user = {first_name: "Alice", last_name: "Smith", age: 30}
user.reorder({name: "first_name", surname: "last_name"})
```

**Result:** `{name: "Alice", surname: "Smith"}`

The dictionary maps new key names to old key names (`{newKey: "oldKey"}`). Keys are output in the order specified in the mapping. Keys not in the mapping are dropped.

This is useful for:
- Reordering columns when preparing data for display
- Renaming fields to match an API or schema
- Selecting a subset of keys in a specific order

**Errors:**
- If an array element references a non-existent key, an error is raised
- If a dictionary value references a non-existent key, an error is raised

---

### delete()

Remove a key from the dictionary. **This is the only method that mutates the original dictionary.**

```parsley
let d = {a: 1, b: 2, c: 3}
d.delete("b")
d
```

**Result:** `{a: 1, c: 3}`

Returns `null`. The dictionary is modified in place.

---

### entries()

Return an array of `{key, value}` dictionaries:

```parsley
{name: "Alice", age: 30}.entries()
```

**Result:** `[{key: "name", value: "Alice"}, {key: "age", value: 30}]`

Customize the key and value names:

```parsley
{name: "Alice", age: 30}.entries("field", "data")
```

**Result:** `[{field: "name", data: "Alice"}, {field: "age", data: 30}]`

Entries are returned in insertion order.

---

### has()

Check if a key exists:

```parsley
{name: "Alice", age: 30}.has("name")
```

**Result:** `true`

```parsley
{name: "Alice"}.has("email")
```

**Result:** `false`

---

### insertAfter()

Insert a new key-value pair after an existing key. Returns a new dictionary:

```parsley
{a: 1, c: 3}.insertAfter("a", "b", 2)
```

**Result:** `{a: 1, b: 2, c: 3}`

Errors if the existing key doesn't exist or the new key already exists.

---

### insertBefore()

Insert a new key-value pair before an existing key. Returns a new dictionary:

```parsley
{a: 1, c: 3}.insertBefore("c", "b", 2)
```

**Result:** `{a: 1, b: 2, c: 3}`

Errors if the existing key doesn't exist or the new key already exists.

---

### keys()

Return an array of all keys in insertion order:

```parsley
{name: "Alice", age: 30, city: "London"}.keys()
```

**Result:** `["name", "age", "city"]`

---

### render()

Render a template string, substituting `@{expr}` placeholders with dictionary values:

```parsley
let person = {name: "Ada", born: 1815}
person.render("@{name} was born in @{born}.")
```

**Result:** `"Ada was born in 1815."`

The content inside `@{...}` is a full Parsley expression with access to the dictionary's keys:

```parsley
let data = {price: 100, tax: 0.2}
data.render("Total: @{price * (1 + tax)}")
```

**Result:** `"Total: 120"`

---

### repr()

Return a string representation suitable for debugging. Note that dictionary values are lazily evaluated, so `repr()` may show `<unevaluated>` for complex values:

```parsley
let d = {name: "Alice", count: 3}
d.repr()
```

**Result:** `"{name: <unevaluated>, count: <unevaluated>}"`

For fully evaluated representations, use `.toJSON()` instead.

---

### toBox()

Render the dictionary as an ASCII box:

```parsley
{name: "Alice", age: 30}.toBox()
```

**Result:**

```
┌───────┬───────┐
│ name  │ Alice │
│ age   │ 30    │
└───────┴───────┘
```

Options:

```parsley
{name: "Alice", age: 30}.toBox({title: "User", style: "rounded"})
```

**Result:**

```
╭───────────────╮
│     User      │
├───────┬───────┤
│ name  │ Alice │
│ age   │ 30    │
╰───────┴───────╯
```

Available options:
- `title`: String - optional title row
- `style`: `"single"` (default), `"double"`, `"rounded"`, `"ascii"`
- `align`: `"left"` (default), `"right"`, `"center"`
- `maxWidth`: Integer - truncate values longer than this

---

### toHTML()

Convert the dictionary to an HTML definition list:

```parsley
{name: "Alice", role: "Admin"}.toHTML()
```

**Result:** `"<dl><dt>name</dt><dd>Alice</dd><dt>role</dt><dd>Admin</dd></dl>"`

---

### toJSON()

Convert the dictionary to a JSON string:

```parsley
{name: "Alice", active: true}.toJSON()
```

**Result:** `"{\"active\":true,\"name\":\"Alice\"}"`

---

### toMarkdown()

Convert the dictionary to a Markdown table:

```parsley
{name: "Alice", age: 30}.toMarkdown()
```

**Result:**

```
| Key | Value |
|-----|-------|
| name | Alice |
| age | 30 |
```

---

### values()

Return an array of all values in insertion order:

```parsley
{name: "Alice", age: 30, city: "London"}.values()
```

**Result:** `["Alice", 30, "London"]`

---

## Order Preservation

Dictionaries in Parsley are **ordered**. They maintain the order in which keys were inserted. This is different from many programming languages where hash maps/dictionaries are unordered.

```parsley
let d = {z: 1, a: 2, m: 3}
d.keys()
```

**Result:** `["z", "a", "m"]`

Order is preserved through:
- Iteration with `for`
- `.keys()`, `.values()`, `.entries()` methods
- Concatenation with `++`
- `.insertAfter()` and `.insertBefore()` methods

---

## Relationship to Other Types

### Records

A **Record** is a schema-bound dictionary. It carries type information, validation state, and field metadata. Convert a dictionary to a Record using `.as(Schema)`:

```parsley
@schema User { name: string, age: int }
let user = {name: "Alice", age: 30}.as(User)
```

**See:** [Records](record.md)

### Tables

A **Table** is an array of dictionaries (rows) with consistent column structure. Tables can be created from arrays of dictionaries:

```parsley
@table [
    {name: "Alice", age: 30}
    {name: "Bob", age: 25}
]
```

Individual table rows are dictionaries (or Records if the table has a schema).

**See:** [Tables](table.md)

### Special Dictionary Types

Many Parsley features are implemented as dictionaries with special `__type` markers:

- **datetime**: `@2024-12-25` creates `{__type: "datetime", year: 2024, ...}`
- **duration**: `3.days` creates `{__type: "duration", days: 3, ...}`
- **path**: `@path("./file.txt")` creates `{__type: "path", ...}`
- **url**: `@url("https://example.com")` creates `{__type: "url", ...}`
- **regex**: `@re("pattern")` creates `{__type: "regex", ...}`

These types have specialized methods and operators but remain dictionaries internally.

### HTML Forms

Dictionaries are the natural format for HTML form data. In Basil handlers, `props` is a dictionary of form field names and values:

```parsley
// In a Basil handler
export save = fn(props) {
    // props is a dictionary: {name: "Alice", email: "alice@example.com"}
    let form = User(props).validate()
    // ...
}
```

### JSON

Dictionaries map directly to JSON objects. Use `.toJSON()` to serialize and `@fetch` responses automatically parse JSON into dictionaries:

```parsley
let data = {name: "Alice", scores: [95, 87]}
data.toJSON()
```

**Result:** `"{\"name\":\"Alice\",\"scores\":[95,87]}"`

---

## Spread in HTML Tags

Dictionary values can be spread into HTML tag attributes using `...dict`:

```parsley
let attrs = {class: "button", id: "submit-btn"}
<button ...attrs>Click</button>
```

**Result:** `<button class="button" id="submit-btn">Click</button>`

Later attributes override earlier ones:

```parsley
let base = {class: "btn", disabled: true}
<button ...base class="btn-primary">Submit</button>
```

**Result:** `<button disabled class="btn-primary">Submit</button>`

---

## Equality

Dictionaries are compared by reference, not by value:

```parsley
{a: 1} == {a: 1}
```

**Result:** `false`

To compare dictionary contents, compare their JSON representations or check individual keys.

---

## Common Patterns

### Building Dictionaries Dynamically

Use computed keys and the merge operator:

```parsley
let base = {type: "user"}
let extra = {id: 42, name: "Alice"}
base ++ extra
```

**Result:** `{type: "user", id: 42, name: "Alice"}`

### Filtering Dictionary Keys

Keep only certain keys using destructuring:

```parsley
let {name, age, ...drop} = {name: "Alice", age: 30, password: "secret", token: "xyz"}
{name, age}
```

**Result:** `{name: "Alice", age: 30}`

### Default Values with Merge

Provide defaults by merging:

```parsley
let defaults = {theme: "light", fontSize: 14}
let userPrefs = {theme: "dark"}
defaults ++ userPrefs
```

**Result:** `{theme: "dark", fontSize: 14}`

### Converting Arrays to Dictionaries

Use `for` to transform an array into dictionary entries:

```parsley
let pairs = [["a", 1], ["b", 2], ["c", 3]]
let result = {}
for (pair in pairs) {
    let key = pair[0]
    result = result ++ {[key]: pair[1]}
}
result
```

**Result:** `{a: 1, b: 2, c: 3}`

---

## See Also

- [Records](record.md) — Schema-bound dictionaries with validation
- [Tables](table.md) — Arrays of dictionaries with tabular operations
- [Schema](schema.md) — Defining data shapes for Records and Tables
- [Variables & Binding](../fundamentals/variables.md) — Destructuring and variable binding
- [Functions](../fundamentals/functions.md) — Function values and closures
- [Types](../fundamentals/types.md) — Parsley's type system and `typeof`
- [Data Model](../fundamentals/data-model.md) — Schemas, records, and tables overview
- [@std/table](../stdlib/table.md) — SQL-like operations on arrays of dictionaries
