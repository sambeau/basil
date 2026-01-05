---
id: FEAT-079
title: "Query DSL with @objects Mini-Grammars"
status: draft
priority: high
created: 2026-01-04
author: "@human"
---

# FEAT-079: Query DSL with @objects Mini-Grammars

## Summary

A domain-specific language for database operations using Parsley's @object syntax. The DSL provides `@schema`, `@query`, `@insert`, `@update`, `@delete`, and `@transaction` constructs with a consistent pipe-based grammar. It covers 90% of common database operations with a minimal, memorable API while maintaining SQL-injection safety through parameterized interpolation.

## User Story

As a Parsley developer, I want a concise, readable syntax for database operations so that I can write type-safe queries without verbose method chains or raw SQL strings.

## Acceptance Criteria

### Schema Declarations
- [ ] `@schema Name { field: type }` declares a schema with typed fields
- [ ] Relations declared with `field: Type via foreign_key` (belongs-to/has-one)
- [ ] Relations declared with `field: [Type] via foreign_key` (has-many)
- [ ] Forward references work implicitly (schemas can reference each other)

### Binding
- [ ] `db.bind(Schema, "table")` binds a schema to a database table
- [ ] `db.bind(Schema, "table", {soft_delete: "field"})` enables soft deletes
- [ ] Multiple bindings to same table allowed (different filtering behavior)

### Query Operations
- [ ] `@query(Binding | conditions ??-> fields)` returns multiple rows
- [ ] `@query(Binding | conditions ?-> *)` returns single row
- [ ] `@query(Binding | conditions ?-> count)` returns row count
- [ ] `@query(Binding | conditions ?-> exists)` returns boolean

### Insert Operations
- [ ] `@insert(Binding |< field: value .)` inserts without return
- [ ] `@insert(Binding |< field: value ?-> *)` inserts and returns created row
- [ ] `@insert(Binding | update on key |< field: value .)` performs upsert
- [ ] `* each {collection} -> alias` enables batch inserts

### Update Operations
- [ ] `@update(Binding | conditions |< field: value .)` updates without return
- [ ] `@update(Binding | conditions |< field: value .-> count)` returns affected count
- [ ] `@update(Binding | conditions |< field: value ?-> *)` returns updated row
- [ ] `@update(Binding | conditions |< field: value ??-> *)` returns all updated rows

### Delete Operations
- [ ] `@delete(Binding | conditions .)` deletes without return
- [ ] `@delete(Binding | conditions .-> count)` returns deleted count
- [ ] `@delete(Binding | conditions ??-> *)` returns deleted rows
- [ ] Soft delete bindings set `deleted_at` instead of removing rows

### Conditions
- [x] Comparison operators: `==`, `!=`, `>`, `<`, `>=`, `<=`
- [x] Set operators: `in`, `not in`
- [x] Pattern matching: `like`
- [x] Range: `between X and Y`
- [ ] Null checks: `is null`, `is not null`
- [x] Logical: `and`, `or`, `not`, parentheses for grouping

### Modifiers
- [x] `| order field (asc|desc)` for sorting
- [x] `| limit N` and `| limit N offset M` for pagination
- [x] `| with relation` for eager loading relations
- [x] Nested relation loading: `| with author, comments.author`
- [x] Conditional relation loading: `| with comments(approved == true)`

### Aggregations
- [ ] `+ by field` for GROUP BY
- [ ] `count` aggregate (bare, no field required)
- [ ] `sum(field)`, `avg(field)`, `min(field)`, `max(field)` aggregates
- [ ] `| name: aggregate` for computed value definitions
- [ ] Conditions after aggregates act as HAVING

### Subqueries
- [ ] Inline subqueries: `| field in <-Table | | conditions | | ?-> column`
- [ ] Nested subqueries with `| | |` depth indicators
- [ ] CTE-style named subqueries at query start
- [ ] Correlated subqueries with `Table as alias` scoping
- [ ] `?->` for scalar subqueries, `??->` for join-like subqueries

### Transactions
- [ ] `@transaction { operations }` wraps multiple operations
- [ ] All operations succeed or all roll back
- [ ] Variables from earlier operations available to later ones

### Interpolation
- [ ] `{expression}` interpolates Parsley expressions
- [ ] All interpolated values are parameterized (SQL-injection safe)
- [ ] Field and table names must be static (not interpolated)

### Results
- [ ] Results are Parsley dictionaries/arrays, not flat rows
- [ ] Relations embedded as nested objects/arrays
- [ ] Partial selection with relations returns partial nested objects

## Design Decisions

### Operators Mirror Existing SQL Operators
- **Rationale**: `?->` mirrors `<=?=>` (single row), `??->` mirrors `<=??=>` (multiple rows), `.` mirrors `<=!=>` (execute). Consistent mental model across raw SQL and DSL.

### Parsley Operators, Not SQL Operators
- **Rationale**: Use `==` not `=`, `and`/`or` not `AND`/`OR`. Developers already know Parsley syntax; SQL syntax would require learning yet another set of conventions.

### Schema-Table Separation
- **Rationale**: Schemas are pure types with no database knowledge. Bindings add the database mapping. This allows same schema for different tables, different databases, or validation-only use.

### `| update on` for Upserts
- **Rationale**: Rather than a separate `@upsert` operation, upsert is a modifier on `@insert`. Reads naturally as "insert, update on conflict with key". Keeps operation count minimal.

### `.` and `•` for Execute-No-Return
- **Rationale**: Both period and bullet point accepted. Period is easy to type; bullet is visually distinct. The `.` visually suggests "end of statement".

### `+ by` for GROUP BY
- **Rationale**: The `+` is visually distinct from `|` conditions. "Plus by" suggests "also group by". Brief but memorable.

### `* each` for Batch Operations
- **Rationale**: `*` suggests iteration/multiplication. "Each" makes intent explicit. Combined with `->` for variable binding, reads naturally: "for each item in collection".

### Soft Deletes via Binding Options
- **Rationale**: Not schema-level (schemas are pure types), not query-level (too easy to forget). Binding-level is declared once, consistently applied. Multiple bindings allow access to deleted rows when needed.

### Explicit Interpolation with `{}`
- **Rationale**: Without markers, ambiguity between columns and variables. With `{}`, rule is simple: bare identifiers are columns, `{...}` are Parsley expressions. Enables static analysis.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components

#### New Parser Components
- `pkg/parsley/parser/schema_parser.go` — Parser for `@schema` declarations
- `pkg/parsley/parser/query_parser.go` — Parser for `@query`, `@insert`, `@update`, `@delete`
- `pkg/parsley/parser/transaction_parser.go` — Parser for `@transaction` blocks

#### New AST Nodes
- `pkg/parsley/ast/schema_node.go` — AST for schema declarations
- `pkg/parsley/ast/query_node.go` — AST for query operations
- `pkg/parsley/ast/binding_node.go` — AST for table bindings

#### New Evaluator Components
- `pkg/parsley/evaluator/schema_eval.go` — Evaluate schema definitions
- `pkg/parsley/evaluator/query_eval.go` — Compile and execute queries
- `pkg/parsley/evaluator/transaction_eval.go` — Transaction management

#### SQL Generation
- `pkg/parsley/sql/builder.go` — Build SQL from AST
- `pkg/parsley/sql/postgres.go` — PostgreSQL-specific SQL generation
- `pkg/parsley/sql/sqlite.go` — SQLite-specific SQL generation

#### Standard Library
- `pkg/parsley/stdlib/db.go` — `db.bind()` function

### Dependencies
- Depends on: FEAT-078 (TableBinding API) — shares binding concepts
- Depends on: FEAT-034 (Schema Validation) — schemas build on validation infrastructure
- Blocks: Future ORM features

### Edge Cases & Constraints

1. **Reserved word `count`** — Cannot have a column named `count` since it's a bare aggregate. Documented as limitation; escape to raw SQL if needed.

2. **Circular schema references** — Forward references are implicit. Implementation must handle parsing schemas that reference not-yet-defined schemas.

3. **Soft delete + hard delete** — Users must bind table twice (with and without soft_delete) to access both behaviors. Documented pattern.

4. **Aggregate without GROUP BY** — Allowed; produces single-row result with aggregate values.

5. **Multiple databases** — Each `db.bind()` associates with a specific db connection. Cross-database queries not supported in DSL (use raw SQL).

6. **Transaction scope** — Variables defined inside `@transaction` are available to later operations in same transaction. Transaction returns last expression's value.

7. **Nested transactions** — Not supported in v1. Use savepoints for partial rollback if database supports them.

8. **Subquery depth** — `| |` nesting theoretically unlimited but practical limit ~3-4 levels for readability. Deeper queries should use CTE-style.

9. **Join vs Subquery** — DSL doesn't expose explicit JOINs. `| with` does eager loading (N+1 or batch), `<-` does subqueries. True JOINs with row multiplication require `??->` return.

10. **NULL in conditions** — Must use `is null`/`is not null`, not `== null`. Matches SQL semantics.

11. **SQLite RETURNING support** — SQLite 3.35.0+ supports `RETURNING` clause. Basil automatically detects version and falls back to `INSERT` + `SELECT last_insert_rowid()` on older versions. Users can also explicitly use `db.lastInsertId()` for maximum compatibility.

## Grammar Specification

### Terminals

```
PIPE        = "|"
PIPE_WRITE  = "|<"
ARROW_PULL  = "<-"
ARROW_PUSH  = "->"
RETURN_ONE  = "?->"
RETURN_MANY = "??->"
EXECUTE     = "." | "•"
EXEC_COUNT  = ".->"
GROUP       = "+ by"
EACH        = "* each"
```

### Productions

```
schema_decl     = "@schema" IDENT "{" field_list "}"
field_list      = (field_def)*
field_def       = IDENT ":" type_expr ("via" IDENT)?
type_expr       = IDENT | "[" IDENT "]"

query_expr      = "@query" "(" query_body ")"
insert_expr     = "@insert" "(" insert_body ")"
update_expr     = "@update" "(" update_body ")"
delete_expr     = "@delete" "(" delete_body ")"
transaction     = "@transaction" "{" statement* "}"

query_body      = source conditions? modifiers? terminal
insert_body     = source upsert? writes terminal
update_body     = source conditions writes terminal
delete_body     = source conditions terminal

source          = IDENT ("as" IDENT)?
conditions      = (PIPE condition)*
condition       = expr | subquery | modifier
subquery        = IDENT "in" ARROW_PULL IDENT subquery_body
subquery_body   = (PIPE PIPE condition)* (RETURN_ONE | RETURN_MANY) projection

modifiers       = (modifier)*
modifier        = order_clause | limit_clause | with_clause | group_clause | define_clause
order_clause    = PIPE "order" IDENT ("asc" | "desc")? ("," IDENT ("asc" | "desc")?)*
limit_clause    = PIPE "limit" NUMBER ("offset" NUMBER)?
with_clause     = PIPE "with" relation_list
group_clause    = GROUP IDENT ("," IDENT)*
define_clause   = PIPE IDENT ":" aggregate_expr

writes          = (PIPE_WRITE IDENT ":" expr)*
upsert          = PIPE "update" "on" IDENT ("," IDENT)*
batch           = EACH "{" expr "}" ARROW_PUSH IDENT ("," IDENT)?

terminal        = RETURN_ONE projection
                | RETURN_MANY projection
                | EXECUTE
                | EXEC_COUNT "count"

projection      = "*" | field_list_proj
field_list_proj = IDENT ("," IDENT)*

aggregate_expr  = "count"
                | "count" "(" IDENT ")"
                | "count" "(" "distinct" IDENT ")"
                | "sum" "(" expr ")"
                | "avg" "(" expr ")"
                | "min" "(" expr ")"
                | "max" "(" expr ")"
```

## SQL Generation Examples

### Basic Query
```parsley
@query(Posts | status == "published" | limit 10 ??-> id, title)
```
```sql
SELECT id, title FROM posts WHERE status = $1 LIMIT 10
-- params: ["published"]
```

### Query with Relations
```parsley
@query(Posts | id == {postId} | with author, comments ?-> *)
```
```sql
-- Query 1: Main
SELECT * FROM posts WHERE id = $1 LIMIT 1
-- Query 2: Author (batch)
SELECT * FROM users WHERE id IN ($2)
-- Query 3: Comments (batch)
SELECT * FROM comments WHERE post_id IN ($3)
-- Results assembled into nested structure
```

### Upsert
```parsley
@insert(Users | update on email |< email: {e} |< name: {n} .)
```
```sql
-- PostgreSQL
INSERT INTO users (email, name) VALUES ($1, $2)
ON CONFLICT (email) DO UPDATE SET email = $1, name = $2

-- SQLite
INSERT INTO users (email, name) VALUES ($1, $2)
ON CONFLICT (email) DO UPDATE SET email = excluded.email, name = excluded.name
```

### Aggregation with GROUP BY
```parsley
@query(Orders | status == "completed" + by customer_id | total: sum(total) | total > 1000 ??-> customer_id, total)
```
```sql
SELECT customer_id, SUM(total) as total
FROM orders
WHERE status = $1
GROUP BY customer_id
HAVING SUM(total) > 1000
-- params: ["completed"]
```

### Subquery
```parsley
@query(Posts | author_id in <-Users | | role == "editor" | | ?-> id ??-> *)
```
```sql
SELECT * FROM posts
WHERE author_id IN (SELECT id FROM users WHERE role = $1)
-- params: ["editor"]
```

### Soft Delete
```parsley
// With soft_delete binding
@delete(Posts | id == {postId} .)
```
```sql
UPDATE posts SET deleted_at = NOW() WHERE id = $1
-- params: [postId]
```

### Transaction
```parsley
@transaction {
  let order = @insert(Orders |< user_id: {uid} ?-> *)
  @insert(OrderItems * each {items} -> i |< order_id: {order.id} |< product_id: {i.pid} .)
}
```
```sql
BEGIN;
INSERT INTO orders (user_id) VALUES ($1) RETURNING *;
INSERT INTO order_items (order_id, product_id) VALUES ($2, $3), ($2, $4), ($2, $5);
COMMIT;
```

## Test Cases

### Schema Parsing
- Parse simple schema with primitive types
- Parse schema with has-many relation
- Parse schema with belongs-to relation
- Parse mutually referential schemas
- Error on unknown type

### Query Parsing
- Parse simple select
- Parse select with multiple conditions (and/or)
- Parse select with ordering
- Parse select with limit/offset
- Parse select with relations
- Parse select with nested relations
- Parse select with aggregate
- Parse select with GROUP BY
- Parse inline subquery
- Parse nested subquery
- Parse CTE-style query
- Parse correlated subquery

### Insert Parsing
- Parse simple insert
- Parse insert with return
- Parse insert with upsert
- Parse batch insert

### Update Parsing
- Parse simple update
- Parse update with multiple conditions
- Parse update with return

### Delete Parsing
- Parse simple delete
- Parse delete with return count
- Parse delete with return rows

### SQL Generation
- Generate SELECT with WHERE
- Generate SELECT with ORDER BY
- Generate SELECT with LIMIT OFFSET
- Generate INSERT
- Generate INSERT RETURNING
- Generate INSERT ON CONFLICT (PostgreSQL)
- Generate INSERT ON CONFLICT (SQLite)
- Generate UPDATE
- Generate UPDATE RETURNING
- Generate DELETE
- Generate soft delete UPDATE
- Generate GROUP BY with HAVING
- Generate subquery IN clause
- Generate correlated subquery

### Execution
- Execute query returns dictionary
- Execute query returns array
- Execute query returns count
- Execute query returns exists boolean
- Execute insert returns created row
- Execute update returns affected count
- Execute delete with soft delete
- Execute transaction commits on success
- Execute transaction rolls back on error
- Interpolation is parameterized

### Integration
- Query with eager-loaded has-many
- Query with eager-loaded belongs-to
- Query with nested eager loading
- Soft delete filtering
- Multiple bindings to same table

## Implementation Notes

*To be added during implementation*

## Related

- Design: `docs/design/QUERY-DSL-DESIGN-v2.md`
- Plan: `docs/plans/FEAT-079-plan.md` (to be created)
- FEAT-078: TableBinding API (related binding concepts)
- FEAT-034: Schema Validation (schema infrastructure)
