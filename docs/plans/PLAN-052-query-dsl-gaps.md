---
id: PLAN-052
feature: FEAT-079
title: "Implementation Plan: Query DSL Design Alignment"
status: in-progress
created: 2026-01-04
updated: 2026-01-04
---

# Implementation Plan: FEAT-079 Query DSL Design Alignment

## Progress Summary

| Phase | Feature | Status | Date |
|-------|---------|--------|------|
| Phase 1 | Interpolation Syntax | ⏸️ Deferred | - |
| Phase 2 | Logical Grouping and NOT | ✅ Complete | 2026-01-04 |
| Phase 3 | Nested Relations | ✅ Complete | 2026-01-04 |
| Phase 4 | Conditional Relations | ✅ Complete | 2026-01-04 |
| Phase 5 | Correlated Subqueries | ✅ Complete | 2026-01-04 |
| Phase 6 | CTEs | ⏸️ Deferred (High Complexity) | - |
| Phase 7 | Join-like Subqueries | ⏸️ Deferred (High Complexity) | - |
| Phase 8 | Documentation | ✅ Complete | 2026-01-04 |

**Completed Features:**
- NOT operator: `| not status == "draft"`
- Parenthesized grouping: `| (a or b) and c`
- Nested relations: `| with comments.author`
- Conditional relations: `| with comments(approved == 1 | order created_at desc | limit 5)`
- Correlated subqueries: `| comment_count <-comments | | post_id == post.id | ?-> count`

**Deferred to Backlog:**
- Phase 1: Foundational change affecting entire DSL, needs careful design
- Phases 6-7: HIGH complexity (3-4+ days each), recommend incremental approach

---

## Overview

This plan addresses all gaps between the current FEAT-079 implementation and the "Query DSL Design: @objects with Mini-Grammars" (v2) design document. The implementation is organized into 8 phases, ordered by dependency and complexity.

**Reference Documents:**
- [FEAT-079-gaps.md](../specs/FEAT-079-gaps.md) — Gap analysis
- [QUERY-DSL-DESIGN-v2.md](../design/QUERY-DSL-DESIGN-v2.md) — Authoritative design

## Prerequisites

- [x] FEAT-079 Phase 1-7 complete (current state)
- [x] All existing DSL tests passing
- [x] Design document reviewed and understood

---

## Phase 1: Interpolation Syntax `{expression}`

**Priority:** High (foundational — other features depend on this)  
**Complexity:** Medium  
**Estimated effort:** 2-3 days

### Rationale

The design document explicitly states:
> **Rule:** Bare identifiers are columns. `{...}` are Parsley expressions.

This resolves ambiguity between columns and variables, enables static analysis, and provides consistent syntax across all DSL operations.

### Task 1.1: Lexer — Add `LBRACE_INTERP` Token

**Files:** `pkg/parsley/lexer/lexer.go`, `pkg/parsley/lexer/token.go`

Steps:
1. Add `LBRACE_INTERP` token type (or reuse existing `LBRACE` with context)
2. In DSL context, `{` starts interpolation, `}` ends it
3. Content inside `{}` is parsed as a Parsley expression

Tests:
- `{variable}` tokenizes correctly
- `{object.property}` tokenizes correctly
- `{fn(arg)}` tokenizes correctly
- Nested `{}` in dicts still works: `{key: {nested: value}}`

---

### Task 1.2: Parser — Parse Interpolated Expressions

**Files:** `pkg/parsley/parser/parser.go`

Steps:
1. In `parseQueryConditionValue()`, detect `{` and parse enclosed expression
2. Mark AST node as `IsInterpolated: true`
3. Support complex expressions: `{date.subtract(date.now(), {days: 7})}`
4. Bare identifiers without `{}` are columns (schema field lookup)

Tests:
- `| user_id == {userId}` parses as interpolated variable
- `| status == "active"` parses as string literal
- `| id == {user.id}` parses as property access
- `| created_at >= {date.now()}` parses as function call
- `| user_id == userId` **errors** or warns (ambiguous per design)

---

### Task 1.3: AST — Add Interpolation Marker

**Files:** `pkg/parsley/ast/ast.go`

Steps:
1. Add `IsInterpolated bool` field to `QueryConditionValue` (or similar)
2. Store the parsed Parsley expression AST node
3. Evaluator will evaluate the expression at query execution time

---

### Task 1.4: Evaluator — Evaluate Interpolated Expressions

**Files:** `pkg/parsley/evaluator/stdlib_dsl_query.go`

Steps:
1. When building SQL, if value `IsInterpolated`, evaluate the Parsley expression
2. Use result as parameterized value (`$1`, `$2`, etc.)
3. Ensure complex expressions like `{cart.items}` work for batch operations

Tests:
- `@query(Posts | user_id == {userId} ??-> *)` with `let userId = 42` → SQL params `[42]`
- `@query(Posts | created_at >= {date.now()} ??-> *)` → current timestamp parameterized

---

### Task 1.5: Update Batch Insert Syntax

**Files:** `pkg/parsley/parser/parser.go`, `pkg/parsley/evaluator/stdlib_dsl_query.go`

Design syntax:
```parsley
@insert(
  OrderItems
  * each {cart.items} -> item
  |< order_id: {order.id}
  |< product_id: {item.product_id}
  .
)
```

Steps:
1. Change `* each collection as alias` to `* each {collection} -> alias`
2. Add `ARROW (->)` token to lexer if not present
3. Parse `->` as alias binding in batch context
4. Field values use `{expression}` syntax

Tests:
- `@insert(Users * each {people} -> person |< name: {person.name} .)` works
- Old syntax `* each people as person` produces deprecation warning or error

---

## Phase 2: Logical Grouping and NOT

**Priority:** High (basic boolean logic)  
**Complexity:** Medium  
**Estimated effort:** 1-2 days

### Task 2.1: Parser — Parenthesized Conditions

**Files:** `pkg/parsley/parser/parser.go`

Design syntax:
```parsley
| (status == "active" or status == "pending") and role == "admin"
```

Steps:
1. In condition parsing, detect `(` and parse grouped conditions recursively
2. Track parentheses depth for proper nesting
3. Parse `and`/`or` between groups

Tests:
- `| (a == 1 or b == 2) and c == 3` parses correctly
- `| ((a == 1 or b == 2) and c == 3) or d == 4` nested groups
- Unbalanced parentheses produce error

---

### Task 2.2: AST — Condition Groups

**Files:** `pkg/parsley/ast/ast.go`

Steps:
1. Add `QueryConditionGroup` node type
2. Contains `Conditions []QueryCondition` and `Logic string` (and/or)
3. Can nest: a group can contain other groups

---

### Task 2.3: Parser — NOT Prefix Operator

**Files:** `pkg/parsley/parser/parser.go`

Design syntax:
```parsley
| not status == "draft"
| not (deleted or archived)
```

Steps:
1. Detect `not` keyword before condition or group
2. Mark condition/group as negated in AST
3. `not` binds tighter than `and`/`or` unless grouped

Tests:
- `| not status == "draft"` → `WHERE NOT status = 'draft'`
- `| not (a or b)` → `WHERE NOT (a OR b)`

---

### Task 2.4: Evaluator — Emit Grouped SQL

**Files:** `pkg/parsley/evaluator/stdlib_dsl_query.go`

Steps:
1. When building WHERE clause, emit `(` `)` for groups
2. Emit `NOT` prefix for negated conditions/groups
3. Handle nested groups recursively

Tests:
- `| (a == 1 or b == 2) and c == 3` → `WHERE (a = $1 OR b = $2) AND c = $3`
- `| not (a == 1 and b == 2)` → `WHERE NOT (a = $1 AND b = $2)`

---

## Phase 3: Nested Relation Loading

**Priority:** High (common N+1 solution)  
**Complexity:** Medium-High  
**Estimated effort:** 2-3 days

### Task 3.1: Parser — Dot-Separated Relation Paths

**Files:** `pkg/parsley/parser/parser.go`

Design syntax:
```parsley
| with author, comments.author
```

Steps:
1. Parse relation names with `.` separator
2. Store as path: `["comments", "author"]`
3. Multiple paths comma-separated

Tests:
- `| with comments.author` parses as path
- `| with author, comments.author, comments.likes` multiple paths

---

### Task 3.2: AST — Relation Path Type

**Files:** `pkg/parsley/ast/ast.go`

Steps:
1. Change `WithRelations []string` to `WithRelations []RelationPath`
2. `RelationPath` is `[]string` (path segments)

---

### Task 3.3: Evaluator — Recursive Relation Loading

**Files:** `pkg/parsley/evaluator/stdlib_dsl_query.go`

Steps:
1. After loading first-level relation, check for nested paths
2. For each nested relation, batch-load from first-level results
3. Attach nested results to first-level objects

Example:
```
Posts -> comments (first level)
comments -> author (nested)
```

SQL sequence:
1. `SELECT * FROM posts WHERE ...`
2. `SELECT * FROM comments WHERE post_id IN ($1, $2, ...)`
3. `SELECT * FROM users WHERE id IN ($1, $2, ...)` (author_ids from comments)

Tests:
- `| with comments.author` loads authors into each comment
- `| with comments.author, comments.likes.user` multiple nested paths
- Performance: batch loads, not N+1+1

---

## Phase 4: Conditional Relation Loading

**Priority:** Medium (cleaner than post-filter)  
**Complexity:** Medium  
**Estimated effort:** 2 days

### Task 4.1: Parser — Relation Conditions

**Files:** `pkg/parsley/parser/parser.go`

Design syntax:
```parsley
| with comments(approved == true | order created_at desc | limit 5)
```

Steps:
1. After relation name, detect `(`
2. Parse conditions, `order`, `limit` inside parentheses
3. Use `|` as separator (same as main query conditions)

Tests:
- `| with comments(approved == true)` parses filter
- `| with comments(order votes desc)` parses ordering
- `| with comments(limit 5)` parses limit
- `| with comments(approved == true | order created_at desc | limit 5)` combined

---

### Task 4.2: AST — RelationPath with Conditions

**Files:** `pkg/parsley/ast/ast.go`

Steps:
1. Extend `RelationPath` to include optional conditions
2. Store `Filter`, `Order`, `Limit` fields

```go
type RelationPath struct {
    Path       []string
    Conditions []QueryCondition
    Order      []QueryOrder
    Limit      *int
}
```

---

### Task 4.3: Evaluator — Apply Relation Filters

**Files:** `pkg/parsley/evaluator/stdlib_dsl_query.go`

Steps:
1. In `loadHasManyRelation`/`loadBelongsToRelation`, apply filters
2. Add WHERE clause for conditions
3. Add ORDER BY if specified
4. Add LIMIT if specified

Tests:
- `| with comments(approved == true)` → `WHERE approved = true AND post_id IN (...)`
- `| with comments(order created_at desc | limit 5)` → ordered and limited

---

## Phase 5: Correlated Subqueries

**Priority:** Medium (advanced feature)  
**Complexity:** High  
**Estimated effort:** 3-4 days

### Task 5.1: Parser — Computed Field from Subquery

**Files:** `pkg/parsley/parser/parser.go`

Design syntax:
```parsley
@query(
  Posts as post
  | comments <-Comments
  | | post_id == post.id
  | ?-> count
  | comments > 5
  ??-> *
)
```

Steps:
1. Detect `| fieldName <-Table` pattern (field assignment from subquery)
2. Parse subquery conditions with `| |` prefix
3. Subquery terminal `?->` returns scalar value
4. Outer query can filter on the computed field

---

### Task 5.2: AST — CorrelatedSubquery Node

**Files:** `pkg/parsley/ast/ast.go`

Steps:
1. Add `QueryComputedField` node type
2. Contains: `Name`, `Subquery`, `Terminal`
3. Subquery can reference outer alias

```go
type QueryComputedField struct {
    Name     string        // "comments"
    Source   string        // "Comments" (table)
    Conditions []QueryCondition
    Terminal QueryTerminal // ?-> count
}
```

---

### Task 5.3: Evaluator — Outer Alias Scope

**Files:** `pkg/parsley/evaluator/stdlib_dsl_query.go`

Steps:
1. Track outer query alias in evaluation context
2. When parsing `post.id` in subquery, recognize `post` as outer alias
3. Generate qualified column reference

---

### Task 5.4: Evaluator — Generate Correlated SQL

**Files:** `pkg/parsley/evaluator/stdlib_dsl_query.go`

Steps:
1. Generate correlated subquery in SELECT clause for computed field
2. Or generate in WHERE clause if filtering

SQL output:
```sql
SELECT *, (SELECT COUNT(*) FROM comments WHERE post_id = posts.id) AS comments
FROM posts
WHERE (SELECT COUNT(*) FROM comments WHERE post_id = posts.id) > 5
```

Tests:
- `| comments <-Comments | | post_id == post.id | ?-> count` generates correlated COUNT
- `| comments > 5` filters on computed field

---

## Phase 6: CTE-Style Named Subqueries

**Priority:** Low (workaround exists)  
**Complexity:** High  
**Estimated effort:** 3-4 days

### Task 6.1: Parser — Multi-Block Query

**Files:** `pkg/parsley/parser/parser.go`

Design syntax:
```parsley
@query(
  Tags as food_tags
  | topic == "food"
  ??-> name

  Posts
  | status == "published"
  | tags in food_tags
  ??-> *
)
```

Steps:
1. Detect multiple query blocks (separated by double newline or semicolon)
2. Parse each block as a named subquery or main query
3. Named subqueries use `Table as name` syntax
4. Main query (last block) can reference named subqueries

---

### Task 6.2: AST — QueryCTE Node

**Files:** `pkg/parsley/ast/ast.go`

Steps:
1. Add `QueryCTE` node type for named subqueries
2. Main query AST includes list of CTEs

```go
type QueryExpression struct {
    CTEs       []QueryCTE
    MainQuery  QueryBlock
}

type QueryCTE struct {
    Name    string      // "food_tags"
    Source  string      // "Tags"
    // ... conditions, terminal
}
```

---

### Task 6.3: Evaluator — Generate WITH Clause

**Files:** `pkg/parsley/evaluator/stdlib_dsl_query.go`

Steps:
1. For each CTE, generate `WITH name AS (SELECT ...)`
2. Chain multiple CTEs with commas
3. Main query follows WITH clause
4. Reference resolution: `food_tags` in main query → CTE name

SQL output:
```sql
WITH food_tags AS (
  SELECT name FROM tags WHERE topic = 'food'
)
SELECT * FROM posts
WHERE status = 'published' AND tags IN (SELECT name FROM food_tags)
```

Tests:
- Single CTE referenced in main query
- Multiple CTEs with inter-dependencies
- CTE referenced multiple times

---

## Phase 7: Join-Like Subqueries (`??->` in subqueries)

**Priority:** Low (use eager loading instead)  
**Complexity:** High  
**Estimated effort:** 2-3 days

### Task 7.1: Parser — Subquery with `??->` Terminal

**Files:** `pkg/parsley/parser/parser.go`

Design syntax:
```parsley
@query(
  Orders as o
  | items <-OrderItems
  | | order_id == o.id
  | ??-> *
  ??-> *
)
```

Steps:
1. Allow `??->` terminal in subqueries (currently only `?->`)
2. `??->` indicates join-like expansion (multiple rows)

---

### Task 7.2: Evaluator — Lateral Join Semantics

**Files:** `pkg/parsley/evaluator/stdlib_dsl_query.go`

Steps:
1. For `??->` subqueries, generate LATERAL JOIN (PostgreSQL) or equivalent
2. Each outer row may produce multiple result rows
3. Result includes both outer and inner columns

SQL output (PostgreSQL):
```sql
SELECT o.*, items.*
FROM orders o
CROSS JOIN LATERAL (
  SELECT * FROM order_items WHERE order_id = o.id
) AS items
```

Tests:
- Subquery with `??->` produces multiple rows per outer row
- Columns from both outer and inner are accessible

---

## Phase 8: Documentation and Migration

**Priority:** Required  
**Complexity:** Low  
**Estimated effort:** 1 day

### Task 8.1: Update Reference Documentation

**Files:** `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`

Steps:
1. Document `{expression}` interpolation syntax
2. Document parenthesized conditions and `not`
3. Document nested relation loading
4. Document conditional relation loading
5. Document correlated subqueries
6. Document CTE syntax (if implemented)

---

### Task 8.2: Migration Guide

**Files:** `docs/guide/migration-079.md` (new)

Steps:
1. Document breaking changes (if `{expression}` required)
2. Provide before/after examples
3. Deprecation warnings for old syntax (if supporting both)

---

### Task 8.3: Update Test Suite

**Files:** `pkg/parsley/tests/dsl_*.pars`

Steps:
1. Add comprehensive tests for all new features
2. Ensure existing tests still pass
3. Add edge case tests

---

## Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Documentation updated
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log

| Date | Phase | Task | Status | Notes |
|------|-------|------|--------|-------|
| — | — | — | — | — |

## Deferred Items

Items to add to BACKLOG.md if not implemented:
- **Multi-column IN:** `| (a, b) in <-Subquery ??-> a, b` — Complex SQL generation
- **Recursive CTEs:** `WITH RECURSIVE` — Advanced use case
- **Window functions:** `row_number() over (partition by ...)` — Out of scope for DSL

---

## Estimated Total Effort

| Phase | Effort | Priority |
|-------|--------|----------|
| Phase 1: Interpolation | 2-3 days | High |
| Phase 2: Logical grouping | 1-2 days | High |
| Phase 3: Nested relations | 2-3 days | High |
| Phase 4: Conditional relations | 2 days | Medium |
| Phase 5: Correlated subqueries | 3-4 days | Medium |
| Phase 6: CTE subqueries | 3-4 days | Low |
| Phase 7: Join-like subqueries | 2-3 days | Low |
| Phase 8: Documentation | 1 day | Required |

**Total: 16-22 days**

### Recommended Implementation Order

1. **Phase 1** — Interpolation (foundational, other phases use this)
2. **Phase 2** — Logical grouping (high impact, medium complexity)
3. **Phase 3** — Nested relations (high impact, common use case)
4. **Phase 4** — Conditional relations (builds on Phase 3)
5. **Phase 5** — Correlated subqueries (advanced, lower priority)
6. **Phase 6** — CTE subqueries (workaround exists)
7. **Phase 7** — Join-like subqueries (use eager loading instead)
8. **Phase 8** — Documentation (throughout and at end)
