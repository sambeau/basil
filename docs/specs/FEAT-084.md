---
id: FEAT-084
title: "Query Row Transform"
status: draft
priority: medium
created: 2026-01-05
author: "@human"
---

# FEAT-084: Query Row Transform

## Summary
Add post-query row transformation to the Query DSL, allowing each result row to be processed through a Parsley expression before being returned. This enables computed fields, data reshaping, and format conversion directly within a query, eliminating the need for a separate `for` loop.

## User Story
As a Parsley developer, I want to transform query results row-by-row within the query itself so that I can reshape data, compute derived fields, and format values without writing verbose post-processing loops.

## Motivation

### Before (current)
```parsley
people = @query(
  People 
  | order Year 
  ??-> Firstname, Surname, Day, Month, Year, id
)

result = for (person in people) {
  {Firstname, Surname, Day, Month, Year, id} = person
  dob = @({Year}-{Month}-{Day})
  age = floor((@today - dob) / @1y)
  {
    id: id,
    Name: Firstname + " " + Surname,
    Birthday: dob.format("long", "en_GB"),
    Age: age,
  }
}
```

### After (with this feature)
```parsley
people = @query(
  People 
  | order Year 
  ??-> * as {Firstname, Surname, Day, Month, Year, id} {
    dob = @({Year}-{Month}-{Day})
    age = floor((@today - dob) / @1y)
    {
      id: id,
      Name: Firstname + " " + Surname,
      Birthday: dob.format("long", "en_GB"),
      Age: age,
    }
  }
)
```

## Syntax

### Grammar
```
row_transform  = "as" binding transform_body
binding        = IDENT | destructure_pattern
destructure_pattern = "{" IDENT ("," IDENT)* ["," "..." IDENT] "}"
transform_body = block | dict_literal
```

### Position in Query
The row transform comes after the projection (`??->`):

```
@query(
  Table
  ?? where_clause
  ??-> projection
  as binding { body }
)
```

## Binding Patterns

### Simple binding
Bind the entire row to a variable:
```parsley
@query(Users ??-> * as row {
  {
    name: row.forename + " " + row.surname,
    email: row.email
  }
})
```

### Destructure binding
Extract specific fields:
```parsley
@query(Users ??-> * as {forename, surname, email} {
  {
    name: forename + " " + surname,
    email: email
  }
})
```

### Destructure with rest
Extract some fields, capture the rest:
```parsley
@query(Users ??-> * as {forename, surname, ...rest} {
  {
    name: forename + " " + surname,
    ...rest  // spread remaining columns
  }
})
```

## Transform Body

The transform body is standard Parsley code that must evaluate to a dictionary.

### Simple dict literal
```parsley
as row {
  name: row.first + " " + row.last,
  email: row.email
}
```

### Block with statements
```parsley
as {Year, Month, Day, ...rest} {
  dob = @({Year}-{Month}-{Day})
  age = floor((@today - dob) / @1y)
  {
    birthday: dob.format("long"),
    age: age,
    ...rest
  }
}
```

## Acceptance Criteria
- [ ] `as ident { ... }` binds row to identifier
- [ ] `as {a, b, c} { ... }` destructures row fields
- [ ] `as {a, ...rest} { ... }` destructures with rest capture
- [ ] Transform body can be a dict literal or a block returning dict
- [ ] `...rest` spread works in output dict
- [ ] Transform runs per-row after SQL returns
- [ ] Errors in transform report row number/context
- [ ] Works with all query types (`@query`, joins, subqueries)
- [ ] Parser produces clear errors for malformed transforms

## Design Decisions

- **`as` keyword**: Chosen because it mirrors SQL alias syntax and Parsley's batch insert syntax (`collection as alias`). Reads naturally: "select all columns *as* this shape".

- **No `=>` arrow**: After discussion, omitting `=>` between binding and body. The `as binding { body }` pattern is sufficient and cleaner. The `{` clearly starts the transform block.

- **Destructure in binding**: Allows extracting fields directly in the binding position, consistent with how destructuring works in `let` statements and function parameters. Reduces boilerplate significantly.

- **Block semantics**: The transform body is a standard Parsley block. If it contains only a dict literal, that's returned. If it contains statements, the final expression (which must be a dict) is returned.

- **Post-SQL execution**: The transform runs in Parsley *after* the SQL query returns, row by row. This is important—SQL aggregates and functions still work in the query; the transform is purely for reshaping results.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/parser/parser.go` — Parse `as binding { body }` after projection
- `pkg/parsley/ast/dsl_query.go` — Add `RowTransform` field to query AST node
- `pkg/parsley/evaluator/dsl_query.go` — Apply transform to each row after SQL execution

### AST Representation
```go
type QueryExpression struct {
    // ... existing fields ...
    RowTransform *RowTransform  // nil if no transform
}

type RowTransform struct {
    Token       lexer.Token
    Binding     RowBinding      // identifier or destructure pattern
    Body        *BlockStatement // the transform block
}

type RowBinding struct {
    Identifier  *Identifier     // simple: as row
    Destructure *DictPattern    // pattern: as {a, b, ...rest}
}
```

### Evaluation Logic
```go
func (e *Evaluator) evalQueryWithTransform(query *ast.QueryExpression) Object {
    // 1. Execute SQL query as normal
    rows := e.executeSQL(query)
    
    // 2. If no transform, return rows as-is
    if query.RowTransform == nil {
        return rows
    }
    
    // 3. Transform each row
    results := make([]Object, 0, len(rows))
    for i, row := range rows {
        // Create scope with binding
        scope := e.bindRow(query.RowTransform.Binding, row)
        
        // Evaluate transform body
        result := e.Eval(query.RowTransform.Body, scope)
        if isError(result) {
            return wrapError(result, "row %d", i+1)
        }
        
        results = append(results, result)
    }
    
    return &Array{Elements: results}
}
```

### Dependencies
- Depends on: Existing Query DSL, destructuring support (already exists)
- Blocks: None

### Edge Cases & Constraints

1. **Empty result set** — Transform is not called; returns empty array
2. **Transform returns non-dict** — Runtime error with row context
3. **Destructure missing field** — Runtime error naming the missing field
4. **Rest pattern with no remaining fields** — `rest` is empty dict, not error
5. **Transform throws error** — Include row number in error message
6. **Nested queries** — Transform applies to outermost query only

## Examples

### Basic transformation
```parsley
@query(Users ??-> forename, surname, email as u {
  {
    name: u.forename + " " + u.surname,
    email: u.email.lower()
  }
})
```

### Date formatting
```parsley
@query(
  Events 
  ?? active == true
  ??-> title, start_date, end_date
  as {title, start_date, end_date} {
    {
      title: title,
      period: start_date.format("short") + " - " + end_date.format("short"),
      duration: end_date - start_date
    }
  }
)
```

### With aggregation (transform runs on aggregate results)
```parsley
@query(
  Orders
  | group customer_id
  ??-> customer_id, total: sum(amount), count: count(*)
  as row {
    {
      customer: row.customer_id,
      total: row.total.format("$0,0.00"),
      avgOrder: (row.total / row.count).format("$0,0.00")
    }
  }
)
```

### Conditional logic in transform
```parsley
@query(Users ??-> * as {role, ...rest} {
  badge = match role {
    "admin" -> "🔴",
    "mod" -> "🟡",
    _ -> "🟢"
  }
  {
    badge: badge,
    ...rest
  }
})
```

## Implementation Notes
*Added during/after implementation*

## Related
- FEAT-081: Rich Schema Types (related: schema-aware transforms could validate output)
- Query DSL Design: `docs/design/QUERY-DSL-DESIGN-v2.md`
