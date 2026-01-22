# Parsley Pretty-Printer Design Document

**Status:** Draft  
**Created:** 2026-01-21  
**Author:** AI + Human collaboration

## Purpose

This document establishes formatting rules for a Parsley pretty-printer/code formatter. The goal is to produce consistently formatted, readable code that can be copy-pasted from the REPL and used in documentation.

## Design Philosophy

1. **Rust-inspired heuristics** — Use different thresholds for different constructs (not one fixed line limit)
2. **Minimal semicolons** — Like Go, only where grammatically required
3. **K&R brace style** — Opening brace on same line
4. **Trailing commas** — On multiline structures (like Go, modern JS)
5. **Readable defaults** — Spaces around operators and after colons

---

## Global Settings

| Setting | Value | Notes |
|---------|-------|-------|
| Max line width | 100 | Modern standard |
| Indent | 4 spaces | Matches most editors |
| Indent style | Spaces | Not tabs |

### Heuristic Thresholds (Rust-style)

| Construct | Threshold | % of max |
|-----------|-----------|----------|
| Line width | 100 | 100% |
| Array literal | 60 | 60% |
| Dict literal | 60 | 60% |
| Method chain | 60 | 60% |
| Function call args | 60 | 60% |
| Single-line if/else | 50 | 50% |

---

## 1. Basic Expressions

### 1.1 Operators — Always Spaced

```parsley
// ✅ FORMATTED
let result = a + b * c
let check = x > 0 && y < 100
let merged = dict1 ++ dict2

// ❌ UNFORMATTED
let result=a+b*c
let check=x>0&&y<100
```

### 1.2 Strings — Preserve User Choice

```parsley
// All valid, preserve as-is
"Hello, World!"
`Hello, {name}!`
'C:\Users\name'
```

### 1.3 Numbers and Literals — No Changes

```parsley
42
3.14159
$19.99
@2024-12-25
@2h30m
/\d+/i
```

---

## 2. Arrays

### 2.1 Short Arrays — Single Line (≤60 chars)

```parsley
// ✅ FORMATTED — fits in 60 chars
[1, 2, 3]
["alice", "bob", "carol"]
[{x: 1}, {x: 2}, {x: 3}]

// From docs:
let arr = [10, 20, 30, 40, 50]
let matrix = [[1, 2, 3], [4, 5, 6], [7, 8, 9]]
```

### 2.2 Long Arrays — Multi-line with Trailing Comma

```parsley
// ✅ FORMATTED — exceeds 60 chars, break to multi-line
let longNames = [
    "alice",
    "bob",
    "charlie",
    "david",
    "elizabeth",
    "frank",
]

// ✅ FORMATTED — nested arrays that exceed threshold
let matrix = [
    [1, 2, 3, 4, 5],
    [6, 7, 8, 9, 10],
    [11, 12, 13, 14, 15],
]
```

### 2.3 Array Destructuring

```parsley
// ✅ Short — single line
let [a, b, c] = [1, 2, 3]
let [first, ...rest] = items

// ✅ Long — multi-line
let [
    firstName,
    lastName,
    email,
    phone,
    address,
] = userData
```

---

## 3. Dictionaries

### 3.1 Short Dicts — Single Line (≤60 chars)

```parsley
// ✅ FORMATTED
{name: "Alice", age: 30}
{x: 1, y: 2}
{host: "localhost", port: 8080}

// Spacing rules:
// - Space after colon: `key: value`
// - Space after comma: `a: 1, b: 2`
// - No space inside braces: `{a: 1}` not `{ a: 1 }`
```

### 3.2 Long Dicts — Multi-line with Trailing Comma

```parsley
// ✅ FORMATTED — exceeds 60 chars
let config = {
    host: "localhost",
    port: 8080,
    database: "myapp",
    username: "admin",
}

// ✅ FORMATTED — nested dicts
let user = {
    name: "Alice",
    settings: {
        theme: "dark",
        notifications: true,
    },
}
```

### 3.3 Dict Destructuring

```parsley
// ✅ Short — single line
let {name, age} = person
let {name, ...rest} = person

// ✅ Long — multi-line  
let {
    name,
    email,
    phone,
    address,
    ...metadata
} = userData
```

---

## 4. Functions

### 4.1 Short Functions — Single Line

```parsley
// ✅ FORMATTED — simple body, fits in line
let double = fn(x) { x * 2 }
let add = fn(a, b) { a + b }
let greet = fn(name) { `Hello, {name}!` }
```

### 4.2 Multi-statement Functions — Multi-line

```parsley
// ✅ FORMATTED — multiple statements
let complex = fn(x) {
    let y = x * 2
    y + 1
}

// ✅ FORMATTED — from docs
let validate = fn(x) {
    check x > 0 else "must be positive"
    check x < 100 else "must be less than 100"
    x * 2
}
```

### 4.3 Function Parameters — Break at Threshold

```parsley
// ✅ Short — single line
let process = fn(a, b, c) { a + b + c }

// ✅ Long — multi-line (exceeds 60 chars total)
let createUser = fn(
    name,
    email,
    password,
    role,
    active,
) {
    // body
}
```

### 4.4 Function Calls — Same Rules

```parsley
// ✅ Short — single line
double(5)
add(3, 4)
format("Hello", name, date)

// ✅ Long — multi-line
createUser(
    "Alice",
    "alice@example.com",
    hashedPassword,
    "admin",
    true,
)
```

---

## 5. Control Flow

### 5.1 If Expressions — Expression Form

```parsley
// ✅ Short — single line (≤50 chars)
let status = if (age >= 18) "adult" else "minor"

// ✅ Long — multi-line
let status = if (age >= 18) {
    "adult"
} else {
    "minor"
}
```

### 5.2 If-Else-If Chains — Always Multi-line

```parsley
// ✅ FORMATTED — from docs
let grade = if (score >= 90) {
    "A"
} else if (score >= 80) {
    "B"
} else if (score >= 70) {
    "C"
} else {
    "F"
}
```

### 5.3 For Expressions

```parsley
// ✅ Short body — single line
for (n in nums) { n * 2 }
for (i, v in items) { `{i}: {v}` }

// ✅ Long body — multi-line
for (n in nums) {
    let squared = n * n
    if (squared > 100) stop
    squared
}

// ✅ Filter pattern — keep inline if short
for (n in nums) { if (n % 2 == 0) n }

// ✅ Complex filter — multi-line
for (n in nums) {
    if (n % 2 == 0) {
        n * 10
    }
}
```

### 5.4 Check Guards

```parsley
// ✅ FORMATTED — from docs
let validate = fn(x) {
    check x > 0 else "must be positive"
    check x < 100 else "must be less than 100"
    x * 2
}
```

---

## 6. Method Chains

### 6.1 Short Chains — Single Line (≤60 chars)

```parsley
// ✅ FORMATTED
name.trim().toUpper()
items.filter(isActive).length()
text.split(",").join(" ")
```

### 6.2 Long Chains — Break After Each Call

```parsley
// ✅ FORMATTED — exceeds 60 chars
let result = data
    .filter(fn(x) { x.active })
    .map(fn(x) { x.name })
    .sort()
    .join(", ")

// ✅ From docs — table operations  
sales
    .where(fn(r) { r.region == "South" })
    .orderBy("amount", "desc")
    .select(["product", "amount"])
```

### 6.3 Chains with Long Arguments

```parsley
// ✅ If single method arg is long, keep method on same line
users.filter(fn(user) {
    user.active && user.role == "admin"
})

// ✅ Multiple chained with long args
users
    .filter(fn(user) {
        user.active && user.role == "admin"
    })
    .map(fn(user) {
        {name: user.name, email: user.email}
    })
```

---

## 7. Tags (HTML)

### 7.1 Self-closing Tags — Single Line

```parsley
// ✅ FORMATTED
<br/>
<hr/>
<input type=text name=email/>
```

### 7.2 Short Tags with Content — Single Line

```parsley
// ✅ FORMATTED
<p>Hello, World!</p>
<span class=highlight>{name}</span>
<a href={url}>Click here</a>
```

### 7.3 Tags with Many Attributes — Multi-line

```parsley
// ✅ FORMATTED — attributes like dict, one per line
<input
    type=text
    name=email
    placeholder="Enter email"
    required
    class=form-input
/>

// ✅ With children
<div class=container id=main>
    <h1>Title</h1>
    <p>Content here</p>
</div>
```

### 7.4 Tags with Code Children — Multi-line

Note: Parsley does not use braces around code inside tags — code is placed directly.

```parsley
// ✅ FORMATTED — for loop as child
<ul>
    for (item in items) {
        <li>item.name</li>
    }
</ul>

// ✅ FORMATTED — conditional
<div>
    if (user) {
        <span>Welcome, user.name</span>
    } else {
        <span>Please log in</span>
    }
</div>
```

---

## 8. Schemas

### 8.1 Short Schemas

```parsley
// ✅ Simple schema — one field per line
@schema Point {
    x: int
    y: int
}
```

### 8.2 Schemas with Constraints

```parsley
// ✅ FORMATTED — from docs
@schema User {
    id: int(auto)
    name: string(min: 2, required)
    email: email(unique: true)
    role: enum["admin", "user", "guest"] = "user"
    active: boolean = true
    createdAt: datetime(auto)
}
```

### 8.3 Schemas with Metadata

```parsley
// ✅ FORMATTED — metadata on same line if short
@schema Contact {
    name: string | {title: "Full Name"}
    email: email | {title: "Email", placeholder: "user@example.com"}
}

// ✅ FORMATTED — long metadata on next line
@schema User {
    bio: text(max: 500) | {
        title: "Biography",
        placeholder: "Tell us about yourself...",
        hidden: false,
    }
}
```

---

## 9. Query DSL

The Query DSL has unique formatting considerations due to its SQL-like nature.

### 9.1 Short Queries — Single Line (≤60 chars)

```parsley
// ✅ FORMATTED — simple queries stay inline
@query(Users | status == "active" ??-> *)
@query(Users | id == {userId} ?-> *)
@query(Users ??-> name, email)
@query(Users ?-> count)
```

### 9.2 Queries with Multiple Conditions — Multi-line

Table name goes on its own line, closing paren on newline:

```parsley
// ✅ FORMATTED — multiple conditions, one per line
@query(
    Users
    | status == "active"
    | role == "admin"
    | age >= 18
    ??-> *
)

// ✅ FORMATTED — with modifiers
@query(
    Users
    | status == "active"
    | order name asc
    | limit 10
    ??-> *
)
```

### 9.3 Complex Queries — Full Multi-line

```parsley
// ✅ FORMATTED — from docs, with eager loading
@query(
    Posts
    | status == "published"
    | with author, comments(approved == true | order created_at desc)
    | order created_at desc
    | limit 10
    ??-> *
)

// ✅ FORMATTED — aggregation
@query(
    Orders
    + by customer_id
    | total: sum(amount)
    | count: count
    | total > {1000}
    ??-> customer_id, total, count
)
```

### 9.4 Insert Statements

```parsley
// ✅ Short — single line
@insert(Users |< name: "Alice" |< email: "alice@test.com" .)

// ✅ Long — multi-line, one field per line
@insert(
    Users
    |< username: "alice-smith"
    |< email: "alice@example.com"
    |< phone: "+1 (555) 123-4567"
    |< role: "user"
    ?-> *
)
```

### 9.5 Transactions

```parsley
// ✅ FORMATTED — from docs
@transaction {
    let user = @insert(Users |< name: "Alice" ?-> *)
    @insert(Profiles |< user_id: {user.id} |< bio: "Hello" .)
    user
}
```

### 9.6 Inline Query Exception

**Rule:** Queries with 1-2 clauses that fit in 60 chars can stay inline even in larger expressions:

```parsley
// ✅ Inline — simple query in function call
let user = db.find(@query(Users | id == {id} ?-> *))

// ✅ Inline — simple query in chain
users.where(status == "active").limit(10)

// ❌ Break — 3+ clauses or exceeds 60 chars
let admins = @query(
    Users
    | role == "admin"
    | status == "active"
    | verified == true
    ??-> *
)
```

---

## 10. Tables

### 10.1 Short Table Literals

```parsley
// ✅ FORMATTED — fits threshold
@table [{x: 1, y: 2}, {x: 3, y: 4}]
```

### 10.2 Long Table Literals — Multi-line

```parsley
// ✅ FORMATTED — from docs
@table [
    {name: "Alice", age: 30, active: true},
    {name: "Bob", age: 25, active: false},
    {name: "Carol", age: 35, active: true},
]

// ✅ With schema
@table(Person) [
    {name: "Alice", age: 30},
    {name: "Bob", age: 25},
]
```

---

## 11. Semicolons

### 11.1 Rule: Only When Required

Parsley uses implicit semicolons (like Go). Explicit semicolons only where grammatically needed:

```parsley
// ✅ No semicolons needed
let x = 5
let y = 10
let result = x + y

// ✅ For loop separation (required)
for (x in items) { x * 2 }

// ❌ Avoid trailing semicolons
let x = 5;
let y = 10;
```

---

## 12. Comments

### 12.1 Single-line Comments

```parsley
// This is a comment
let x = 5  // Inline comment
```

### 12.2 Preserve User Comments

The pretty-printer should preserve comments in their relative position.

---

## Resolved Questions

1. **Attribute ordering in tags** — Preserve user order (default). See "Future: Semantic Attribute Ordering" below for potential enhancement.

2. **Schema field ordering** — Preserve user order, but `id` field should always be first if present.

3. **Empty lines** — 2 blank lines between top-level definitions.

4. **Import grouping** — Yes. Group and sort: std library first, then project imports.

5. **Query DSL inline threshold** — 2 clauses AND ≤60 chars. Single-clause queries should never need breaking (implied AND makes them naturally short).

---

## Implementation Notes

### Phase 1: AST Pretty-Printer
- Walk AST and emit formatted code
- Track indentation level
- Calculate line lengths for threshold decisions

### Phase 2: Preserving Semantics
- Handle comments (attach to nearest node)
- Preserve string quote style
- Preserve number formats

### Phase 3: REPL Integration
- `ObjectToReprString()` uses pretty-printer for functions
- REPL output is valid, copy-pasteable Parsley

---

## References

- **Prettier** (JS): 80 char default, trailing commas, bracket spacing
- **Rustfmt**: 100 char default, 60% heuristics for nested constructs
- **gofmt** (Go): No line limit, tabs, mandatory trailing commas, K&R braces
- **Biome** (JS): 80 char default, Prettier-compatible

---

## Future: Semantic Attribute Ordering

*Low priority — interesting but complex. Would need SVG support which adds significant scope.*

### Research Summary

No major formatter auto-sorts attributes (Prettier, Rustfmt, gofmt all preserve order). Sorting is typically a **linter** concern (e.g., `eslint-plugin-react`'s `jsx-sort-props`).

### Observed Patterns in HTML

The most common ordering pattern across real-world HTML:

```
id → class → type/role → name → src/href → [functional] → [handlers] → [data-*] → [aria-*] → [booleans]
```

This matches how developers mentally parse elements: identity → appearance → kind → purpose → behavior → metadata → state.

### Proposed Tiers (if implemented)

| Tier | Attributes | Rationale |
|------|------------|-----------|
| 1 | `id`, `name`, `for` | Identity |
| 2 | `class`, `style` | Styling hooks |
| 3 | `type`, `role` | Kind/semantics |
| 4 | `href`, `src`, `action` | Destination/source |
| 5 | `value`, `placeholder`, `alt`, `title` | Content/description |
| 6 | `width`, `height`, `size`, `cols`, `rows` | Dimensions |
| 7 | `min`, `max`, `pattern`, `minlength`, `maxlength` | Constraints |
| 8 | `on*`, `hx-*` | Event handlers |
| 9 | `data-*` | Custom data (alphabetical) |
| 10 | `aria-*`, `tabindex` | Accessibility |
| 11 | `required`, `disabled`, `checked`, `readonly`, `hidden`, ... | Boolean flags |

Within each tier: alphabetical order.

### Spread Boundaries

Spread operators (`...`) act as ordering boundaries — tiers reset after a spread. This preserves semantic meaning (later spreads override earlier values).

```parsley
// Spreads preserved, sorting within segments
<Input {...defaults} class=input name=email type=text {...overrides} disabled required/>
```

### Complications

1. **SVG attributes** — Completely different set (`viewBox`, `preserveAspectRatio`, `d`, `cx`, `cy`, `r`, `fill`, `stroke`, etc.). Would need separate tier lists.

2. **Framework-specific** — React wants `key`/`ref` first; Vue wants `v-if`/`v-for` first; HTMX has `hx-*` clustering.

3. **Diminishing returns** — Alphabetical-only (spread-aware) gets 80% of the diff benefits with 10% of the complexity.

### Recommendation

If implemented:
- **Default**: Preserve user order
- **Flag**: `--sort-attributes` → simple alphabetical (spread-aware)
- **Future flag**: `--sort-attributes=semantic` → tier-based ordering
