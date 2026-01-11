# FEAT-079 Implementation Gaps

This document details the features from the FEAT-079 Query DSL spec (based on the "Query DSL Design: @objects with Mini-Grammars" design document) that are either not implemented or only partially implemented. Each section includes the design document syntax, use cases, and implementation complexity estimates.

**Last Updated:** 2026-01-04

---

## Recently Implemented Features

### ✅ Logical Grouping and NOT (Phase 2 - Complete)

**Implemented Features:**
- `| not status == "draft"` → `WHERE NOT status = 'draft'`
- `| (a == 1 or b == 2)` → `WHERE (a = $1 OR b = $2)`
- `| not (a or b)` → `WHERE NOT (a OR b)`
- `| (a or b) and c` → `WHERE (a OR b) AND c`

### ✅ Nested Relation Loading (Phase 3 - Complete)

**Implemented Features:**
- `| with comments.author` loads authors into each comment
- Recursive loading for any depth of nesting

### ✅ Conditional Relation Loading (Phase 4 - Complete)

**Implemented Features:**
- `| with comments(approved == true)` filters relations
- `| with comments(order created_at desc)` orders relations
- `| with comments(limit 5)` limits relations
- Combined: `| with comments(approved == true | order created_at desc | limit 5)`

---

## Not Implemented Features

### ~~1. Conditional Relation Loading~~ ✅ IMPLEMENTED (Phase 4)

> **Status:** Fully implemented in Phase 4 of PLAN-052. See "Recently Implemented Features" section above.
> 
> Supports filter conditions, ordering, and limits in relation loading:
> ```parsley
> | with comments(approved == true | order created_at desc | limit 5)
> ```

---

### 2. CTE-Style Named Subqueries

**Design Reference:** Part 9 - Subqueries - CTE-Style (Named Subqueries)

**Design Syntax:**
```parsley
@query(
  Tags as food_tags
  | topic == "food"
  ??-> name

  Posts
  | status == "published"
  | tags in food_tags
  | order created_at desc
  ??-> *
)
```

**Description:**  
Define a named subquery at the top of a `@query`, then reference it in the main query. The subquery acts like a CTE (Common Table Expression) - it's evaluated once and the result can be used multiple times.

**Current Implementation:**
- ❌ This syntax is NOT implemented
- Only inline subqueries with `<-Table` work

**What Works Instead:**
```parsley
// Inline subquery (single use)
@query(
  Posts
  | author_id in <-Users | | role == "editor" | | ?-> id
  ??-> *
)
```

**Use Cases:**
1. Complex reporting queries with multiple derived tables
2. Reusing the same subquery result multiple times
3. Breaking down complex queries into readable named parts

**Implementation Complexity:** High
- Parser: Parse multiple query blocks within single `@query()`
- AST: New `QueryCTE` node type for named subqueries
- Evaluator: Build SQL WITH clause, handle reference resolution
- SQL Generation: Emit proper CTE syntax for each database dialect

**Workaround:**
```parsley
// Use Parsley variables for separate queries (but this is N+1)
let foodTagNames = @query(Tags | topic == "food" ??-> name)
    .map(fn(t) { t.name })

@query(Posts | status == "published" | tags in foodTagNames ??-> *)
```

---

### 3. Correlated Subqueries (Computed Fields from Subquery)

**Design Reference:** Part 9 - Subqueries - Correlated Subqueries

**Design Syntax:**
```parsley
// Posts with more than 5 comments
@query(
  Posts as post
  | comments <-Comments
  | | post_id == post.id
  | ?-> count
  | comments > 5
  ??-> *
)
```

**Description:**  
Define a computed field (`comments`) from a correlated subquery, then filter on it. The subquery references the outer query's alias (`post.id`).

**Current Implementation:**
- ✅ `source as alias` syntax is parsed
- ❌ Computed fields from subqueries NOT implemented
- ❌ Referencing outer alias in subquery NOT implemented

**What Partially Works:**
```parsley
// Inline subquery for IN clause (no correlation)
@query(Posts | author_id in <-Users | | role == "editor" | | ?-> id ??-> *)
```

**Use Cases:**
1. Find posts with more than N comments
2. Find users with total order value over $1000
3. Find products with average rating above 4.0
4. "Latest of each" queries (latest order per customer)

**Implementation Complexity:** High
- Parser: Track alias scope across outer/inner queries
- AST: Support computed field assignment from subquery
- Evaluator: Pass outer query context to subquery, handle correlation
- SQL: Generate correlated subquery with outer reference

**Workaround:**
```parsley
// Use aggregation with GROUP BY
@query(Comments + by post_id | comment_count: count | comment_count > 5 ??-> post_id)
// Then fetch those posts separately
```

---

## Partially Implemented Features

### ~~4. Logical Grouping with Parentheses and NOT~~ ✅ IMPLEMENTED (Phase 2)

> **Status:** Fully implemented in Phase 2 of PLAN-052. See "Recently Implemented Features" section above.
>
> Supports:
> - `| not status == "draft"` → `WHERE NOT status = 'draft'`
> - `| (a == 1 or b == 2)` → `WHERE (a = $1 OR b = $2)`
> - `| (a or b) and c` → `WHERE (a OR b) AND c`
> - `| not (a or b)` → `WHERE NOT (a OR b)`

---

### ~~5. Nested Relation Loading~~ ✅ IMPLEMENTED (Phase 3)

> **Status:** Fully implemented in Phase 3 of PLAN-052. See "Recently Implemented Features" section above.
> 
> Supports dot-notation for nested relation loading:
> ```parsley
> | with comments.author
> ```
> Returns posts with comments, where each comment has its author loaded.

---

### 6. Scalar vs Join Subqueries (`??->` in subqueries)

**Design Reference:** Part 9 - Subqueries - Scalar vs Join Subqueries

**Design Syntax:**
```parsley
// Scalar lookup (one value per row)
@query(
  OrderItems as item
  | category <-Products
  | | id == item.product_id
  | ?-> category
  + by category
  ??-> category, count
)

// Join-like (multiple rows per item would expand rows)
@query(
  Orders as o
  | items <-OrderItems
  | | order_id == o.id
  | ??-> *
  ??-> *
)
```

**Current Implementation:**
- ✅ `?->` in subqueries for IN clause (scalar list)
- ❌ `??->` in subqueries for join-like behavior NOT implemented
- ❌ Computed field assignment from subquery NOT implemented

**What Works:**
```parsley
// Scalar subquery for IN clause (works)
@query(Posts | author_id in <-Users | | role == "admin" | | ?-> id ??-> *)
// SQL: SELECT * FROM posts WHERE author_id IN (SELECT id FROM users WHERE role = 'admin')
```

**What Doesn't Work:**
```parsley
// Computed field from scalar subquery
@query(
  OrderItems as item
  | category <-Products | | id == item.product_id | ?-> category
  + by category
  ??-> category, count
)

// Join-like subquery returning multiple values
@query(
  Orders as o
  | items <-OrderItems | | order_id == o.id | ??-> *
  ??-> *
)
```

**Use Cases:**
1. Computed columns from related tables
2. Join-like expansions without explicit JOIN syntax
3. Denormalized data retrieval

**Implementation Complexity:** High
- This requires correlated subquery support (see #3)
- For join-like `??->`, need lateral join semantics

---

### 7. Batch Insert with `{expression}` and `->`

**Design Reference:** Part 10 - Batch Operations, Part 12 - Interpolation

**Design Syntax:**
```parsley
@insert(
  OrderItems
  * each {cart.items} -> item
  |< order_id: {order.id}
  |< product_id: {item.product_id}
  |< quantity: {item.quantity}
  |< price: {item.price}
  .
)
```

**Current Implementation:**
- ✅ Batch insert works
- ⚠️ Syntax differs: uses `as` instead of `->`
- ⚠️ Syntax differs: no `{}` around collection or values

**What's Implemented:**
```parsley
// Actual implemented syntax
@insert(Users * each people as person |< name: person.name |< age: person.age .)
```

**Design Syntax:**
```parsley
// Design document syntax
@insert(Users * each {people} -> person |< name: {person.name} |< age: {person.age} .)
```

**Differences:**
| Aspect | Design | Implementation |
|--------|--------|----------------|
| Collection wrapper | `{people}` | `people` |
| Alias binding | `-> person` | `as person` |
| Field value interpolation | `{person.name}` | `person.name` |

**Why the Design Uses `{}`:**

From the design document (Part 12 - "Why Interpolation Markers?"):

> Without `{}`, there's ambiguity between columns and variables:
> ```parsley
> // Is 'status' a column or a variable?
> | status == published   // Ambiguous!
> 
> // Clear with {}
> | status == {published}  // published is a Parsley variable
> | status == "published"  // "published" is a literal
> ```
> **Rule:** Bare identifiers are columns. `{...}` are Parsley expressions.

**Impact:** Medium - The deviation creates ambiguity the design explicitly avoids

**Recommendation:** Align with design by implementing `{expression}` interpolation syntax. This would:
1. Resolve column vs variable ambiguity consistently
2. Match the design's explicit rationale
3. Enable future static analysis (the design notes this as a benefit)

**For `->` vs `as`:** The `as` deviation is less problematic (stylistic choice), but `{expression}` markers address a real ambiguity concern.

---

### 8. Interpolation with `{expression}`

**Design Reference:** Part 12 - Interpolation

**Design Syntax:**
```parsley
@query(
  Posts
  | user_id == {currentUser.id}
  | created_at >= {date.subtract(date.now(), {days: 7})}
  ??-> *
)
```

**Current Implementation:**
- ✅ Variable references work without `{}`
- ⚠️ `{expression}` syntax NOT required (bare identifiers work)
- ✅ All values properly parameterized (SQL-safe)

**What Works:**
```parsley
// Bare variable reference (works)
let userId = 42
@query(Posts | user_id == userId ??-> *)
// SQL: SELECT * FROM posts WHERE user_id = $1, params: [42]
```

**Design Expected:**
```parsley
// With {} markers
@query(Posts | user_id == {userId} ??-> *)
```

**Impact:** Low - The implementation is arguably cleaner

**Note:** The design document says:
> **Rule:** Bare identifiers are columns. `{...}` are Parsley expressions.

Current implementation treats bare identifiers as either columns (if schema field) or variables (if in scope). This works but differs from the explicit design.

---

## Summary: Design vs Implementation

| Feature | Design Doc | Implementation | Status |
|---------|-----------|----------------|--------|
| Schema declarations | ✅ | ✅ | **Match** |
| Relations (via) | ✅ | ✅ | **Match** |
| db.bind() | ✅ | ✅ | **Match** |
| Soft deletes | ✅ | ✅ | **Match** |
| `@query` basic | ✅ | ✅ | **Match** |
| `@insert` basic | ✅ | ✅ | **Match** |
| `@update` basic | ✅ | ✅ | **Match** |
| `@delete` basic | ✅ | ✅ | **Match** |
| `?->` / `??->` terminals | ✅ | ✅ | **Match** |
| `.` / `.->` terminals | ✅ | ✅ | **Match** |
| Conditions (`==`, `!=`, etc.) | ✅ | ✅ | **Match** |
| `in` / `not in` | ✅ | ✅ | **Match** |
| `like` | ✅ | ✅ | **Match** |
| `between X and Y` | ✅ | ✅ | **Match** |
| `is null` / `is not null` | ✅ | ✅ | **Match** |
| `and` / `or` | ✅ | ✅ | **Match** |
| `not expr` | ✅ | ❌ | **Missing** |
| `(a or b) and c` | ✅ | ❌ | **Missing** |
| `order by` | ✅ | ✅ | **Match** |
| `limit` / `offset` | ✅ | ✅ | **Match** |
| `\| with relation` | ✅ | ✅ | **Match** |
| `\| with rel(condition)` | ✅ | ❌ | **Missing** |
| `\| with a.b` (nested) | ✅ | ⚠️ | **Partial** |
| `+ by` (GROUP BY) | ✅ | ✅ | **Match** |
| `count`, `sum`, `avg`, etc. | ✅ | ✅ | **Match** |
| `\| name: aggregate` | ✅ | ✅ | **Match** |
| HAVING (post-aggregate filter) | ✅ | ✅ | **Match** |
| Inline subquery `<-Table` | ✅ | ✅ | **Match** |
| Nested subquery `\| \|` | ✅ | ✅ | **Match** |
| CTE-style named subquery | ✅ | ❌ | **Missing** |
| Correlated subquery | ✅ | ❌ | **Missing** |
| Computed field from subquery | ✅ | ❌ | **Missing** |
| `??->` in subquery (join-like) | ✅ | ❌ | **Missing** |
| `@transaction` | ✅ | ✅ | **Match** |
| Upsert (`\| update on`) | ✅ | ✅ | **Match** |
| Batch insert (`* each`) | `{c} -> a` | `c as a` | **Syntax differs** |
| Interpolation | `{expr}` | bare ident | **Syntax differs** |

---

## Implementation Priority Recommendations

Based on design document analysis:

| Priority | Feature | Complexity | Impact | Notes |
|----------|---------|------------|--------|-------|
| 1 | Parentheses `(a or b)` | Medium | High | Basic boolean logic |
| 2 | `not expr` | Low | Medium | Simple negation |
| 3 | Nested relations `a.b` | Medium | High | Common N+1 solution |
| 4 | Conditional `with(...)` | Medium | Medium | Cleaner than post-filter |
| 5 | Correlated subqueries | High | Medium | Advanced feature |
| 6 | CTE named subqueries | High | Low | Workaround exists |

---

## Syntax Deviation Summary

The implementation deviates from the design in two areas:

### 1. Batch Insert Alias Binding

**Design:** `* each {collection} -> alias`  
**Implementation:** `* each collection as alias`

**Assessment:** 
- The `->` syntax was chosen deliberately: using `as` makes this look like SQL aliasing, when it's actually a Parsley construct iterating to generate multiple inserts
- `->` visually signals "binding into" which matches Parsley's data flow semantics
- Missing `{collection}` wrapper is more significant (see #2 below)

### 2. Value Interpolation

**Design:** `{expression}` markers required  
**Implementation:** Bare identifiers resolve to variables if in scope

**Design Rationale (Part 12):**
> Without `{}`, there's ambiguity between columns and variables.
> **Rule:** Bare identifiers are columns. `{...}` are Parsley expressions.

**Assessment:** The design explicitly addresses this ambiguity. The current implementation's context-based resolution (schema field = column, in scope = variable) works but:
- Creates implicit behavior that may surprise users
- Prevents static analysis the design intended to enable
- Could cause subtle bugs when variable names match column names

**Recommendation:** Consider aligning with the design's `{expression}` syntax to resolve ambiguity explicitly.
