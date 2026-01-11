---
id: PLAN-049
feature: FEAT-079
title: "Implementation Plan for Query DSL with @objects Mini-Grammars"
status: draft
created: 2026-01-04
---

# Implementation Plan: FEAT-079 Query DSL

## Overview

Implement a domain-specific language for database operations using Parsley's @object syntax. This provides `@schema`, `@query`, `@insert`, `@update`, `@delete`, and `@transaction` constructs with a consistent pipe-based grammar, covering 90% of common database operations.

## Implementation Phases

The implementation is divided into 7 phases, each building on the previous. Each phase is independently testable and delivers incremental value.

| Phase | Description | Effort | Dependencies |
|-------|-------------|--------|--------------|
| 1 | Schema Declarations | Medium | None |
| 2 | Table Binding | Small | Phase 1 |
| 3 | Basic Queries | Large | Phase 2 |
| 4 | Basic Mutations | Large | Phase 2 |
| 5 | Aggregations & Grouping | Medium | Phase 3 |
| 6 | Subqueries | Large | Phase 3 |
| 7 | Transactions | Medium | Phase 4 |

---

## Prerequisites

- [ ] FEAT-034 (Schema Validation) complete — provides schema infrastructure
- [ ] FEAT-078 (TableBinding API) complete — provides binding concepts
- [ ] Existing @object parser infrastructure understood
- [ ] SQLite and PostgreSQL test databases available

---

## Phase 1: Schema Declarations

**Goal**: Parse and evaluate `@schema` declarations with fields and relations.

### Task 1.1: Lexer Tokens for @schema
**Files**: `pkg/parsley/lexer/lexer.go`
**Effort**: Small

Steps:
1. Add `TOKEN_SCHEMA` token type for `@schema` keyword
2. Add `TOKEN_VIA` token type for `via` keyword in relations
3. Update keyword lookup table

Acceptance Criteria:
- [ ] `@schema` tokenizes as TOKEN_SCHEMA
- [ ] `via` tokenizes as TOKEN_VIA
- [ ] Existing tokens unaffected

Tests:
- Tokenize `@schema` produces TOKEN_SCHEMA
- Tokenize `via` produces TOKEN_VIA
- Tokenize `@schema User { id: int }` produces correct token sequence

---

### Task 1.2: Schema AST Nodes
**Files**: `pkg/parsley/ast/schema_node.go` (new)
**Effort**: Small

Steps:
1. Create `SchemaDeclaration` node with Name, Fields
2. Create `SchemaField` node with Name, Type, RelationType (one/many/none), ForeignKey
3. Add node types to ast.go registry

```go
type SchemaDeclaration struct {
    Token  token.Token
    Name   *Identifier
    Fields []*SchemaField
}

type SchemaField struct {
    Token       token.Token
    Name        *Identifier
    TypeName    string        // "int", "string", "User", etc.
    IsArray     bool          // true for [Type]
    ForeignKey  string        // from "via fk_name"
}
```

Acceptance Criteria:
- [ ] SchemaDeclaration implements ast.Node
- [ ] SchemaField implements ast.Node
- [ ] String() methods produce readable output

Tests:
- SchemaDeclaration.String() outputs parseable representation
- SchemaField with relation outputs "field: Type via fk"

---

### Task 1.3: Schema Parser
**Files**: `pkg/parsley/parser/parser.go` or `pkg/parsley/parser/schema_parser.go` (new)
**Effort**: Medium

Steps:
1. Add parsing branch for `@schema` in parseStatement/parseExpression
2. Implement parseSchemaDeclaration():
   - Expect IDENT (schema name)
   - Expect `{`
   - Parse field list until `}`
3. Implement parseSchemaField():
   - Parse `name: type` or `name: [type]` for arrays
   - Optionally parse `via foreign_key`

Acceptance Criteria:
- [ ] `@schema Name { field: type }` parses without error
- [ ] `@schema Name { field: Type via fk }` parses relation
- [ ] `@schema Name { field: [Type] via fk }` parses has-many relation
- [ ] Missing closing brace produces clear error
- [ ] Invalid field syntax produces clear error

Tests:
- Parse simple schema with int, string, bool fields
- Parse schema with belongs-to relation
- Parse schema with has-many relation
- Parse schema with multiple fields and relations
- Parse empty schema (edge case)
- Error: missing schema name
- Error: missing opening brace
- Error: missing closing brace
- Error: invalid type syntax

---

### Task 1.4: Schema Object Type
**Files**: `pkg/parsley/object/schema_object.go` (new)
**Effort**: Small

Steps:
1. Create `Schema` object type to hold parsed schema
2. Store field definitions and relation metadata
3. Make Schema usable as a value (can be passed to functions)

```go
type Schema struct {
    Name     string
    Fields   map[string]SchemaFieldDef
    Relations map[string]SchemaRelation
}

type SchemaFieldDef struct {
    Type     string
    Required bool
}

type SchemaRelation struct {
    TargetSchema string
    ForeignKey   string
    IsMany       bool
}
```

Acceptance Criteria:
- [ ] Schema implements object.Object
- [ ] Schema.Type() returns "SCHEMA"
- [ ] Schema.Inspect() returns readable representation
- [ ] Schema is hashable (can be dict key)

Tests:
- Schema.Type() returns "SCHEMA"
- Schema.Inspect() shows name and fields
- Two schemas with same name are equal

---

### Task 1.5: Schema Evaluator
**Files**: `pkg/parsley/evaluator/schema_eval.go` (new)
**Effort**: Small

Steps:
1. Add case for SchemaDeclaration in Eval()
2. Convert AST to Schema object
3. Handle forward references (schemas referencing not-yet-defined schemas)
4. Register schema in environment

Acceptance Criteria:
- [ ] `@schema User { id: int }` creates Schema object
- [ ] Schema bound to name in environment
- [ ] Forward references resolve correctly
- [ ] Duplicate schema name produces error

Tests:
- Evaluate simple schema, check it's in environment
- Evaluate schema with forward reference to another schema
- Evaluate mutually referential schemas (User has Posts, Post has User)
- Error: duplicate schema name

---

### Task 1.6: Schema Field Type Validation
**Files**: `pkg/parsley/evaluator/schema_eval.go`
**Effort**: Small

Steps:
1. Validate primitive types (int, string, bool, float, datetime, etc.)
2. Validate relation types reference existing/forward schemas
3. Warn on unknown types

Acceptance Criteria:
- [ ] Known primitive types accepted
- [ ] Schema type references validated
- [ ] Unknown type produces error

Tests:
- Accept int, string, bool, float, datetime
- Accept reference to defined schema
- Accept forward reference to schema defined later
- Error: unknown type "foobar"

---

## Phase 2: Table Binding

**Goal**: Implement `db.bind(Schema, "table")` to connect schemas to database tables.

### Task 2.1: Binding Object Type
**Files**: `pkg/parsley/object/binding_object.go` (new)
**Effort**: Small

Steps:
1. Create `TableBinding` object type (or extend existing from FEAT-078)
2. Store: Schema reference, table name, binding options
3. Add soft_delete option support

```go
type TableBinding struct {
    Schema      *Schema
    TableName   string
    SoftDelete  string  // column name for soft delete, or ""
    Connection  *Database
}
```

Acceptance Criteria:
- [ ] TableBinding implements object.Object
- [ ] TableBinding stores schema and table name
- [ ] TableBinding stores soft_delete column name
- [ ] TableBinding.Type() returns "TABLE_BINDING"

Tests:
- Create TableBinding, verify properties
- TableBinding with soft_delete option
- TableBinding.Inspect() shows schema and table

---

### Task 2.2: db.bind() Function
**Files**: `pkg/parsley/stdlib/stdlib_db.go` (extend existing)
**Effort**: Small

Steps:
1. Add `bind` method to database object
2. Accept: Schema, table_name string, optional options dict
3. Return TableBinding object
4. Validate schema is actually a Schema object

Acceptance Criteria:
- [ ] `db.bind(Schema, "table")` returns TableBinding
- [ ] `db.bind(Schema, "table", {soft_delete: "deleted_at"})` sets soft delete
- [ ] Error if first arg is not Schema
- [ ] Error if second arg is not string

Tests:
- Basic binding without options
- Binding with soft_delete option
- Error: non-schema first argument
- Error: non-string table name
- Error: unknown option key

---

### Task 2.3: Multiple Bindings Pattern
**Files**: `pkg/parsley/evaluator/schema_eval.go`
**Effort**: Small

Steps:
1. Ensure multiple bindings to same schema work
2. Ensure multiple bindings to same table work
3. Each binding is independent (different filtering behavior)

Acceptance Criteria:
- [ ] Same schema can bind to multiple tables
- [ ] Same table can have multiple bindings (different options)
- [ ] Each binding operates independently

Tests:
- Bind same schema to two different tables
- Bind same table twice with different soft_delete settings
- Query each binding independently

---

## Phase 3: Basic Queries

**Goal**: Implement `@query` with conditions, ordering, pagination, and projections.

### Task 3.1: Query Lexer Tokens
**Files**: `pkg/parsley/lexer/lexer.go`
**Effort**: Small

Steps:
1. Add TOKEN_QUERY for `@query`
2. Add TOKEN_RETURN_ONE for `?->`
3. Add TOKEN_RETURN_MANY for `??->`
4. Add TOKEN_ORDER, TOKEN_LIMIT, TOKEN_OFFSET keywords

Acceptance Criteria:
- [ ] `@query` tokenizes as TOKEN_QUERY
- [ ] `?->` tokenizes as TOKEN_RETURN_ONE
- [ ] `??->` tokenizes as TOKEN_RETURN_MANY
- [ ] Keywords tokenize correctly

Tests:
- Tokenize `@query` produces TOKEN_QUERY
- Tokenize `?->` produces TOKEN_RETURN_ONE
- Tokenize `??->` produces TOKEN_RETURN_MANY
- Tokenize `| order name desc | limit 10 ??-> *`

---

### Task 3.2: Query AST Nodes
**Files**: `pkg/parsley/ast/query_node.go` (new)
**Effort**: Medium

Steps:
1. Create QueryExpression node
2. Create QueryCondition node (operator, left, right)
3. Create QueryModifier node (order, limit, with)
4. Create QueryProjection node (fields or *)
5. Create QueryTerminal node (return type)

```go
type QueryExpression struct {
    Token      token.Token
    Source     *Identifier       // Binding name
    SourceAlias *Identifier      // Optional "as alias"
    Conditions []*QueryCondition
    Modifiers  []*QueryModifier
    Terminal   *QueryTerminal
}

type QueryCondition struct {
    Token    token.Token
    Left     Expression
    Operator string  // "==", "!=", "in", "like", etc.
    Right    Expression
    Logic    string  // "and", "or" for combining
}

type QueryTerminal struct {
    Token      token.Token
    Type       string  // "one", "many", "execute", "count"
    Projection []string // field names, or ["*"]
}
```

Acceptance Criteria:
- [ ] QueryExpression captures full query structure
- [ ] Conditions support all comparison operators
- [ ] Modifiers support order, limit, offset, with
- [ ] Terminal captures return type and projection

Tests:
- QueryExpression.String() produces readable output
- QueryCondition with various operators
- QueryTerminal with projection

---

### Task 3.3: Query Parser - Basic Structure
**Files**: `pkg/parsley/parser/query_parser.go` (new)
**Effort**: Large

Steps:
1. Implement parseQueryExpression():
   - Expect `@query` `(`
   - Parse source (binding identifier, optional alias)
   - Parse conditions (| condition)*
   - Parse modifiers (order, limit, with)
   - Parse terminal (?-> or ??-> with projection)
   - Expect `)`
2. Implement parseQueryCondition():
   - Parse left expression (column)
   - Parse operator
   - Parse right expression (value or interpolation)
3. Implement parseQueryTerminal():
   - Detect ?-> vs ??-> vs .
   - Parse projection (field list or *)

Acceptance Criteria:
- [ ] `@query(Binding ??-> *)` parses
- [ ] `@query(Binding | col == "val" ??-> *)` parses
- [ ] `@query(Binding | col == {var} ?-> id, name)` parses
- [ ] Multiple conditions with and/or parse correctly
- [ ] Parenthesized conditions parse correctly

Tests:
- Parse query with no conditions
- Parse query with single equality condition
- Parse query with multiple conditions (and)
- Parse query with multiple conditions (or)
- Parse query with mixed and/or
- Parse query with parenthesized conditions
- Parse query with `in [...]` list
- Parse query with `like` pattern
- Parse query with `between X and Y`
- Parse query with `is null` / `is not null`
- Parse query with interpolation `{var}`
- Error: missing source
- Error: missing terminal
- Error: invalid operator

---

### Task 3.4: Query Parser - Modifiers
**Files**: `pkg/parsley/parser/query_parser.go`
**Effort**: Medium

Steps:
1. Implement parseOrderClause():
   - `| order col (asc|desc)?`
   - Multiple columns: `| order col1 asc, col2 desc`
2. Implement parseLimitClause():
   - `| limit N`
   - `| limit N offset M`
3. Implement parseWithClause():
   - `| with relation`
   - `| with rel1, rel2`
   - `| with rel.nested`
   - `| with rel(conditions)`

Acceptance Criteria:
- [ ] Order clause parses with direction
- [ ] Order clause defaults to asc
- [ ] Multi-column order parses
- [ ] Limit parses
- [ ] Limit with offset parses
- [ ] With single relation parses
- [ ] With multiple relations parses
- [ ] With nested relations parses
- [ ] With conditional relations parses

Tests:
- Parse `| order name`
- Parse `| order name desc`
- Parse `| order name asc, created_at desc`
- Parse `| limit 10`
- Parse `| limit 10 offset 20`
- Parse `| with author`
- Parse `| with author, comments`
- Parse `| with comments.author`
- Parse `| with comments(approved == true)`
- Parse `| with comments(order created_at desc | limit 5)`

---

### Task 3.5: Query Evaluator - Setup
**Files**: `pkg/parsley/evaluator/query_eval.go` (new)
**Effort**: Medium

Steps:
1. Add case for QueryExpression in Eval()
2. Resolve source binding from environment
3. Evaluate interpolated expressions
4. Validate conditions reference valid columns
5. Pass to SQL builder

Acceptance Criteria:
- [ ] Query evaluator resolves binding
- [ ] Interpolated values are captured
- [ ] Invalid binding name produces error
- [ ] Query passes to SQL builder

Tests:
- Evaluate query with valid binding
- Error: undefined binding
- Error: column not in schema (optional, may defer)

---

### Task 3.6: SQL Builder - SELECT
**Files**: `pkg/parsley/sql/builder.go` (new)
**Effort**: Large

Steps:
1. Create SQLBuilder struct to accumulate query parts
2. Implement BuildSelect():
   - SELECT columns FROM table
   - WHERE conditions
   - ORDER BY
   - LIMIT OFFSET
3. Generate parameterized SQL ($1, $2, etc.)
4. Return SQL string and parameter array

```go
type SQLBuilder struct {
    dialect   string  // "postgres" or "sqlite"
    sql       strings.Builder
    params    []interface{}
    paramIdx  int
}

func (b *SQLBuilder) BuildSelect(query *QueryExpression, binding *TableBinding) (string, []interface{}, error)
```

Acceptance Criteria:
- [ ] Generates valid SELECT statement
- [ ] Conditions become WHERE clause
- [ ] Order becomes ORDER BY
- [ ] Limit becomes LIMIT/OFFSET
- [ ] All values are parameterized
- [ ] Column names are validated/escaped

Tests:
- Build SELECT * FROM table
- Build SELECT col1, col2 FROM table
- Build SELECT with WHERE col = $1
- Build SELECT with multiple WHERE conditions (AND)
- Build SELECT with OR conditions
- Build SELECT with ORDER BY
- Build SELECT with LIMIT
- Build SELECT with LIMIT OFFSET
- Build SELECT with all clauses combined
- Parameters array matches placeholders

---

### Task 3.7: Query Execution
**Files**: `pkg/parsley/evaluator/query_eval.go`
**Effort**: Medium

Steps:
1. Execute generated SQL against database
2. Map result rows to Parsley dictionaries
3. Handle ?-> (single row or null)
4. Handle ??-> (array of rows)
5. Handle ?-> count (return integer)
6. Handle ?-> exists (return boolean)

Acceptance Criteria:
- [ ] ??-> returns array of dictionaries
- [ ] ?-> returns single dictionary or null
- [ ] ?-> count returns integer
- [ ] ?-> exists returns boolean
- [ ] Empty result: ??-> returns [], ?-> returns null

Tests:
- Query returning multiple rows
- Query returning single row
- Query returning count
- Query returning exists (true)
- Query returning exists (false)
- Query with no matches returns empty array
- Query single with no match returns null

---

### Task 3.8: Soft Delete Filtering
**Files**: `pkg/parsley/sql/builder.go`
**Effort**: Small

Steps:
1. Check binding for soft_delete option
2. If set, add implicit WHERE deleted_at IS NULL
3. Combine with user conditions using AND

Acceptance Criteria:
- [ ] Binding with soft_delete auto-filters
- [ ] Filter combines with user conditions
- [ ] Binding without soft_delete has no filter

Tests:
- Query on soft_delete binding excludes deleted rows
- Query on non-soft_delete binding sees all rows
- User conditions AND soft_delete filter combined

---

### Task 3.9: Relation Eager Loading
**Files**: `pkg/parsley/evaluator/query_eval.go`
**Effort**: Large

Steps:
1. Parse `| with` clause
2. For each relation:
   - Collect foreign keys from main query results
   - Execute batch query for related records
   - Assemble into nested structure
3. Support has-one (embed object) and has-many (embed array)
4. Support nested relations (comments.author)

Acceptance Criteria:
- [ ] `| with author` embeds author object
- [ ] `| with comments` embeds comments array
- [ ] `| with comments.author` embeds nested
- [ ] Batch loading (not N+1)
- [ ] Missing relation returns null/[]

Tests:
- Load has-one relation
- Load has-many relation
- Load nested relation (2 levels)
- Load multiple relations
- Relation with no matches (null for has-one, [] for has-many)
- Batch query uses IN clause (verify no N+1)

---

## Phase 4: Basic Mutations

**Goal**: Implement `@insert`, `@update`, `@delete` operations.

### Task 4.1: Mutation Lexer Tokens
**Files**: `pkg/parsley/lexer/lexer.go`
**Effort**: Small

Steps:
1. Add TOKEN_INSERT for `@insert`
2. Add TOKEN_UPDATE for `@update`
3. Add TOKEN_DELETE for `@delete`
4. Add TOKEN_PIPE_WRITE for `|<`
5. Add TOKEN_EXECUTE for `.` (and `•`)
6. Add TOKEN_EXEC_COUNT for `.->`

Acceptance Criteria:
- [ ] `@insert`, `@update`, `@delete` tokenize correctly
- [ ] `|<` tokenizes as TOKEN_PIPE_WRITE
- [ ] `.` tokenizes as TOKEN_EXECUTE
- [ ] `.->` tokenizes as TOKEN_EXEC_COUNT

Tests:
- Tokenize `@insert(...)` sequence
- Tokenize `|< field: value` sequence
- Tokenize `.` and `•`
- Tokenize `.-> count`

---

### Task 4.2: Mutation AST Nodes
**Files**: `pkg/parsley/ast/mutation_node.go` (new)
**Effort**: Medium

Steps:
1. Create InsertExpression, UpdateExpression, DeleteExpression nodes
2. Create WriteClause node for `|< field: value`
3. Create UpsertClause node for `| update on key`
4. Create BatchClause node for `* each {collection} -> alias`

```go
type InsertExpression struct {
    Token     token.Token
    Source    *Identifier
    Upsert    *UpsertClause     // optional
    Batch     *BatchClause      // optional
    Writes    []*WriteClause
    Terminal  *QueryTerminal
}

type WriteClause struct {
    Token token.Token
    Field *Identifier
    Value Expression
}

type UpsertClause struct {
    Token token.Token
    Keys  []*Identifier  // columns for conflict detection
}

type BatchClause struct {
    Token      token.Token
    Collection Expression
    ItemAlias  *Identifier
    IndexAlias *Identifier  // optional
}
```

Acceptance Criteria:
- [ ] InsertExpression captures all insert variants
- [ ] UpdateExpression captures conditions and writes
- [ ] DeleteExpression captures conditions
- [ ] WriteClause captures field assignments

Tests:
- AST String() methods produce readable output
- InsertExpression with writes
- InsertExpression with upsert
- InsertExpression with batch
- UpdateExpression with conditions and writes

---

### Task 4.3: Mutation Parser
**Files**: `pkg/parsley/parser/mutation_parser.go` (new)
**Effort**: Large

Steps:
1. Implement parseInsertExpression():
   - Parse source binding
   - Optionally parse `| update on key1, key2`
   - Optionally parse `* each {collection} -> alias`
   - Parse writes `|< field: value`
   - Parse terminal
2. Implement parseUpdateExpression():
   - Parse source binding
   - Parse conditions `| cond`
   - Parse writes `|< field: value`
   - Parse terminal
3. Implement parseDeleteExpression():
   - Parse source binding
   - Parse conditions
   - Parse terminal

Acceptance Criteria:
- [ ] `@insert(Binding |< f: v .)` parses
- [ ] `@insert(Binding | update on key |< f: v .)` parses
- [ ] `@insert(Binding * each {items} -> i |< f: {i.x} .)` parses
- [ ] `@update(Binding | cond |< f: v .)` parses
- [ ] `@delete(Binding | cond .)` parses

Tests:
- Parse simple insert
- Parse insert with multiple writes
- Parse insert with upsert on single key
- Parse insert with upsert on composite key
- Parse insert with batch
- Parse insert with return (?-> *)
- Parse update with single condition
- Parse update with multiple conditions
- Parse update with return (.-> count)
- Parse delete with condition
- Parse delete with return (??-> *)
- Error: insert without writes
- Error: update without conditions (dangerous)
- Error: delete without conditions (dangerous)

---

### Task 4.4: SQL Builder - INSERT
**Files**: `pkg/parsley/sql/builder.go`
**Effort**: Medium

Steps:
1. Implement BuildInsert():
   - INSERT INTO table (columns) VALUES (params)
   - Optional RETURNING clause
2. Implement BuildInsertBatch():
   - Multiple value tuples
3. Implement BuildUpsert():
   - PostgreSQL: ON CONFLICT ... DO UPDATE
   - SQLite: ON CONFLICT ... DO UPDATE SET

Acceptance Criteria:
- [ ] Generates valid INSERT statement
- [ ] Generates INSERT with RETURNING
- [ ] Generates batch INSERT with multiple rows
- [ ] Generates upsert for PostgreSQL
- [ ] Generates upsert for SQLite
- [ ] All values parameterized

Tests:
- Build simple INSERT
- Build INSERT with RETURNING *
- Build INSERT with RETURNING id
- Build batch INSERT (3 rows)
- Build PostgreSQL upsert
- Build SQLite upsert
- Build upsert with composite key

---

### Task 4.5: SQL Builder - UPDATE
**Files**: `pkg/parsley/sql/builder.go`
**Effort**: Medium

Steps:
1. Implement BuildUpdate():
   - UPDATE table SET col = val WHERE conditions
   - Optional RETURNING clause
2. Apply soft_delete filter if binding has it

Acceptance Criteria:
- [ ] Generates valid UPDATE statement
- [ ] Generates UPDATE with RETURNING
- [ ] Soft delete binding excludes deleted rows
- [ ] All values parameterized

Tests:
- Build simple UPDATE
- Build UPDATE with multiple SET clauses
- Build UPDATE with RETURNING *
- Build UPDATE with RETURNING count
- Build UPDATE with soft_delete filter

---

### Task 4.6: SQL Builder - DELETE
**Files**: `pkg/parsley/sql/builder.go`
**Effort**: Medium

Steps:
1. Implement BuildDelete():
   - DELETE FROM table WHERE conditions
   - Optional RETURNING clause
2. If soft_delete binding, generate UPDATE instead

Acceptance Criteria:
- [ ] Generates valid DELETE statement
- [ ] Generates DELETE with RETURNING
- [ ] Soft delete binding generates UPDATE
- [ ] All values parameterized

Tests:
- Build simple DELETE
- Build DELETE with RETURNING *
- Build DELETE with RETURNING count
- Build soft delete (UPDATE deleted_at)

---

### Task 4.7: Mutation Execution
**Files**: `pkg/parsley/evaluator/mutation_eval.go` (new)
**Effort**: Medium

Steps:
1. Execute INSERT and return created row(s)
2. Execute UPDATE and return affected count or rows
3. Execute DELETE and return deleted count or rows
4. Handle batch insert execution

Acceptance Criteria:
- [ ] Insert returns created row with ?->
- [ ] Insert returns null with .
- [ ] Update returns affected count with .-> count
- [ ] Update returns updated rows with ??->
- [ ] Delete returns deleted count with .-> count
- [ ] Batch insert creates all rows

Tests:
- Execute insert, verify row created
- Execute insert with return, verify returned data
- Execute batch insert, verify all rows created
- Execute upsert (insert case)
- Execute upsert (update case)
- Execute update, verify rows modified
- Execute update with return count
- Execute delete, verify rows removed
- Execute soft delete, verify deleted_at set

---

## Phase 5: Aggregations & Grouping

**Goal**: Implement `+ by` grouping, aggregate functions, and computed values.

### Task 5.1: Aggregation Lexer Tokens
**Files**: `pkg/parsley/lexer/lexer.go`
**Effort**: Small

Steps:
1. Add TOKEN_GROUP_BY for `+ by`
2. Keep `count`, `sum`, `avg`, `min`, `max` as regular identifiers
   (they're function calls in aggregate context)

Acceptance Criteria:
- [ ] `+ by` tokenizes as TOKEN_GROUP_BY
- [ ] `count` tokenizes as identifier

Tests:
- Tokenize `+ by status`
- Tokenize `| total: sum(amount)`

---

### Task 5.2: Aggregation AST Nodes
**Files**: `pkg/parsley/ast/query_node.go`
**Effort**: Small

Steps:
1. Add GroupByClause to QueryExpression
2. Create AggregateExpression for sum(), avg(), etc.
3. Create ComputedColumn for `| name: expression`

```go
type GroupByClause struct {
    Token   token.Token
    Columns []*Identifier
}

type ComputedColumn struct {
    Token token.Token
    Name  *Identifier
    Value Expression  // aggregate or expression
}
```

Acceptance Criteria:
- [ ] GroupByClause captures group columns
- [ ] ComputedColumn captures alias and expression

Tests:
- GroupByClause.String()
- ComputedColumn.String()

---

### Task 5.3: Aggregation Parser
**Files**: `pkg/parsley/parser/query_parser.go`
**Effort**: Medium

Steps:
1. Parse `+ by col1, col2` after conditions
2. Parse `| name: aggregate(col)` as computed column
3. Parse bare `count` as special aggregate
4. Parse conditions after computed columns as HAVING

Acceptance Criteria:
- [ ] `+ by status` parses
- [ ] `+ by status, category` parses
- [ ] `| total: sum(amount)` parses
- [ ] `| n: count` parses (bare count)
- [ ] `| n: count(distinct col)` parses
- [ ] Conditions after aggregates become HAVING

Tests:
- Parse `+ by status`
- Parse `+ by status, category`
- Parse `| total: sum(amount)`
- Parse `| average: avg(score)`
- Parse `| lowest: min(price)`
- Parse `| highest: max(price)`
- Parse `| n: count`
- Parse `| n: count(id)`
- Parse `| n: count(distinct category)`
- Parse `| total: sum(amount) | total > 1000` (HAVING)
- Error: aggregate without group by returns single row (valid)
- Error: non-aggregated column in select without group by

---

### Task 5.4: SQL Builder - Aggregates
**Files**: `pkg/parsley/sql/builder.go`
**Effort**: Medium

Steps:
1. Generate GROUP BY clause
2. Generate aggregate functions in SELECT
3. Generate HAVING clause from post-aggregate conditions

Acceptance Criteria:
- [ ] GROUP BY generated correctly
- [ ] Aggregate functions in SELECT
- [ ] HAVING clause from conditions after aggregates
- [ ] Column aliases in SELECT

Tests:
- Build `GROUP BY status`
- Build `GROUP BY status, category`
- Build `SELECT status, COUNT(*) FROM ... GROUP BY status`
- Build `SELECT status, SUM(amount) as total FROM ... GROUP BY status`
- Build with HAVING clause
- Build aggregate without GROUP BY (single row result)

---

### Task 5.5: Aggregate Execution
**Files**: `pkg/parsley/evaluator/query_eval.go`
**Effort**: Small

Steps:
1. Execute aggregate query
2. Map results to dictionaries with computed column names
3. Handle single-row aggregate (no GROUP BY)

Acceptance Criteria:
- [ ] Aggregated results have computed column names
- [ ] GROUP BY returns array of groups
- [ ] No GROUP BY returns single dictionary

Tests:
- Execute `+ by status | n: count ??-> status, n`
- Execute without GROUP BY `| total: sum(amount) ?-> total`
- Verify computed column names in result

---

## Phase 6: Subqueries

**Goal**: Implement inline subqueries, CTE-style, and correlated subqueries.

### Task 6.1: Subquery Lexer Tokens
**Files**: `pkg/parsley/lexer/lexer.go`
**Effort**: Small

Steps:
1. Add TOKEN_PULL for `<-`
2. `| |` is just two pipes, no special token needed

Acceptance Criteria:
- [ ] `<-` tokenizes as TOKEN_PULL

Tests:
- Tokenize `<-Users`
- Tokenize `| | conditions | |`

---

### Task 6.2: Subquery AST Nodes
**Files**: `pkg/parsley/ast/query_node.go`
**Effort**: Medium

Steps:
1. Create SubqueryExpression node
2. Support inline subquery in condition
3. Support named (CTE-style) subquery
4. Support correlated subquery with parent reference

```go
type SubqueryExpression struct {
    Token       token.Token
    Name        *Identifier    // For CTE-style, nil for inline
    Source      *Identifier    // Table/Binding
    SourceAlias *Identifier    // For correlated
    Conditions  []*QueryCondition
    Terminal    *QueryTerminal
    IsCorrelated bool
}
```

Acceptance Criteria:
- [ ] SubqueryExpression captures inline subquery
- [ ] SubqueryExpression captures CTE-style
- [ ] SubqueryExpression captures correlated

Tests:
- SubqueryExpression.String()
- Inline subquery structure
- CTE-style subquery structure

---

### Task 6.3: Subquery Parser
**Files**: `pkg/parsley/parser/query_parser.go`
**Effort**: Large

Steps:
1. Parse `| col in <-Table | | conditions | | ?-> col`
2. Track depth with `| |` markers
3. Parse CTE-style: `Table as name | | conditions | | ?-> col` at query start
4. Parse correlated: reference parent alias in subquery conditions

Acceptance Criteria:
- [ ] Inline subquery parses
- [ ] Nested subquery (2 levels) parses
- [ ] CTE-style named subquery parses
- [ ] Correlated subquery with parent reference parses

Tests:
- Parse `| author_id in <-Users | | role == "editor" | | ?-> id`
- Parse nested subquery (3 levels of `| |`)
- Parse CTE `Tags as active_tags | | active == true | | ??-> id`
- Parse correlated `Comments as c | | c.post_id == post.id | | ?-> count`
- Error: unmatched `| |` depth

---

### Task 6.4: SQL Builder - Subqueries
**Files**: `pkg/parsley/sql/builder.go`
**Effort**: Large

Steps:
1. Build inline subquery as `IN (SELECT ... FROM ... WHERE ...)`
2. Build CTE as `WITH name AS (SELECT ...) SELECT ... WHERE col IN name`
3. Build correlated subquery with parent table reference
4. Handle scalar (`?->`) vs multi-row (`??->`) subqueries

Acceptance Criteria:
- [ ] Inline subquery generates IN (SELECT ...)
- [ ] CTE generates WITH ... AS ...
- [ ] Correlated subquery references parent alias
- [ ] Scalar subquery limits to 1 row

Tests:
- Build inline subquery IN clause
- Build nested subquery
- Build CTE with named subquery
- Build correlated subquery
- Build scalar subquery (ensure single value)

---

### Task 6.5: Subquery Execution
**Files**: `pkg/parsley/evaluator/query_eval.go`
**Effort**: Medium

Steps:
1. Execute subqueries as part of main query
2. Handle CTE execution order
3. Validate correlated references

Acceptance Criteria:
- [ ] Subquery executes and filters correctly
- [ ] CTE is properly scoped
- [ ] Correlated subquery references parent rows

Tests:
- Execute query with inline subquery
- Execute query with CTE
- Execute query with correlated subquery
- Subquery returns empty list (no matches in main)

---

## Phase 7: Transactions

**Goal**: Implement `@transaction { ... }` blocks with commit/rollback semantics.

### Task 7.1: Transaction Lexer Token
**Files**: `pkg/parsley/lexer/lexer.go`
**Effort**: Small

Steps:
1. Add TOKEN_TRANSACTION for `@transaction`

Acceptance Criteria:
- [ ] `@transaction` tokenizes as TOKEN_TRANSACTION

Tests:
- Tokenize `@transaction { ... }`

---

### Task 7.2: Transaction AST Node
**Files**: `pkg/parsley/ast/transaction_node.go` (new)
**Effort**: Small

Steps:
1. Create TransactionBlock node with list of statements

```go
type TransactionBlock struct {
    Token      token.Token
    Statements []Statement
}
```

Acceptance Criteria:
- [ ] TransactionBlock holds multiple statements

Tests:
- TransactionBlock.String()

---

### Task 7.3: Transaction Parser
**Files**: `pkg/parsley/parser/transaction_parser.go` (new)
**Effort**: Small

Steps:
1. Parse `@transaction {` 
2. Parse statements until `}`
3. Statements can include @insert, @update, @delete, let bindings

Acceptance Criteria:
- [ ] `@transaction { @insert(...) @update(...) }` parses
- [ ] Variables declared inside are scoped to transaction
- [ ] Transaction returns last expression value

Tests:
- Parse empty transaction
- Parse transaction with single insert
- Parse transaction with multiple operations
- Parse transaction with let binding
- Parse transaction with conditional logic

---

### Task 7.4: Transaction Execution
**Files**: `pkg/parsley/evaluator/transaction_eval.go` (new)
**Effort**: Medium

Steps:
1. Begin database transaction
2. Execute statements in order
3. If all succeed, commit
4. If any fails, rollback
5. Return last expression's value

Acceptance Criteria:
- [ ] Transaction commits on success
- [ ] Transaction rolls back on any error
- [ ] Variables from earlier ops available to later ops
- [ ] Transaction returns last expression value

Tests:
- Execute transaction with multiple inserts, verify all committed
- Execute transaction with error in middle, verify rollback
- Execute transaction with let binding used later
- Transaction returns value of final insert
- Nested transaction error (should fail or use savepoint)

---

## Validation Checklist

### Per-Phase Validation
After each phase, run:
- [ ] `go test ./...` — All tests pass
- [ ] `go build -o basil ./cmd/basil` — Build succeeds
- [ ] `golangci-lint run` — No linter errors

### Final Validation
- [ ] All acceptance criteria from FEAT-079 checked off
- [ ] All test cases from FEAT-079 pass
- [ ] Documentation updated
- [ ] CHANGELOG.md entry added
- [ ] Example code works end-to-end

---

## Test Plan Summary

### Unit Tests (per component)

| Component | Test File | Test Count |
|-----------|-----------|------------|
| Schema lexer | `lexer/lexer_test.go` | ~5 |
| Schema parser | `parser/schema_parser_test.go` | ~10 |
| Schema evaluator | `evaluator/schema_eval_test.go` | ~8 |
| Query lexer | `lexer/lexer_test.go` | ~5 |
| Query parser | `parser/query_parser_test.go` | ~30 |
| Query evaluator | `evaluator/query_eval_test.go` | ~15 |
| Mutation parser | `parser/mutation_parser_test.go` | ~20 |
| Mutation evaluator | `evaluator/mutation_eval_test.go` | ~15 |
| Aggregation parser | `parser/query_parser_test.go` | ~15 |
| Aggregation evaluator | `evaluator/query_eval_test.go` | ~10 |
| Subquery parser | `parser/query_parser_test.go` | ~10 |
| Subquery evaluator | `evaluator/query_eval_test.go` | ~8 |
| Transaction parser | `parser/transaction_parser_test.go` | ~5 |
| Transaction evaluator | `evaluator/transaction_eval_test.go` | ~8 |
| SQL Builder | `sql/builder_test.go` | ~40 |
| **Total** | | **~200** |

### Integration Tests

| Scenario | Test File |
|----------|-----------|
| End-to-end CRUD | `tests/query_dsl_integration_test.go` |
| Soft deletes | `tests/query_dsl_soft_delete_test.go` |
| Relations & eager loading | `tests/query_dsl_relations_test.go` |
| Transactions | `tests/query_dsl_transaction_test.go` |
| Error handling | `tests/query_dsl_errors_test.go` |

### Database Tests
- SQLite: All SQL generation and execution
- PostgreSQL: Dialect-specific features (upsert syntax)

---

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-01-04 | Phase 1: Schema Declarations | Complete | AST and parser for @schema |
| 2026-01-04 | Phase 2: Table Binding | Complete | db.bind() connects schemas to tables |
| 2026-01-04 | Phase 3: Basic Queries | Complete | @query DSL with conditions, modifiers, terminals |
| 2026-01-04 | Phase 4: Basic Mutations | Complete | @insert, @update, @delete DSLs |
| 2026-01-05 | Phase 5: Aggregations | Complete | GROUP BY, count, sum, avg, min, max |
| 2026-01-05 | Phase 6: Subqueries | Complete | IN subqueries with <- syntax and `| |` nested conditions |
| 2026-01-05 | Phase 7: Transactions | Complete | @transaction { } with automatic commit/rollback |

---

## Deferred Items

Items to add to BACKLOG.md after implementation:

1. **Nested transactions / savepoints** — Complex, needs database-specific handling
2. **Cross-database queries** — Out of scope, use raw SQL
3. **Dynamic field/table names** — Security concern, intentionally unsupported
4. **Query builder API** — Programmatic query construction (alternative to DSL)
5. **Migration generation** — Generate SQL from schema changes
6. **Query optimization hints** — INDEX hints, EXPLAIN integration

---

## Related Documents

- Specification: [FEAT-079.md](../specs/FEAT-079.md)
- Design: [QUERY-DSL-DESIGN-v2.md](../design/QUERY-DSL-DESIGN-v2.md)
- Related: [FEAT-078.md](../specs/FEAT-078.md) (TableBinding API)
- Related: [FEAT-034.md](../specs/FEAT-034.md) (Schema Validation)
