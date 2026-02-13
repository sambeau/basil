---
id: FEAT-117
title: "SQL Tag Raw Text Content"
status: complete
priority: medium
created: 2025-02-14
completed: 2025-02-14
author: "@human"
related: FEAT-098
---

# FEAT-117: SQL Tag Raw Text Content

## Summary

Make `<SQL>` tags treat their content as raw text, eliminating the need for quotes around SQL code. This aligns with how `<style>` and `<script>` tags work, providing a cleaner syntax for parameterized queries while enforcing SQL injection safety by disallowing inline interpolation.

## User Story

As a Parsley developer, I want to write SQL queries without wrapping them in quotes so that my code is cleaner and more readable, while maintaining safety through parameterized queries.

## Current Behavior

SQL content must be a quoted string:

```parsley
let InsertUser = fn(props) {
    <SQL name={props.name}>
        "INSERT INTO users (name) VALUES (?)"
    </SQL>
}
```

Without quotes, the content is parsed as Parsley code, causing errors like `Identifier not found: SELECT`.

## Proposed Behavior

SQL content is raw text, like `<style>` and `<script>`:

```parsley
let InsertUser = fn(props) {
    <SQL name={props.name}>
        INSERT INTO users (name) VALUES (?)
    </SQL>
}
```

### Key Differences from `<style>`/`<script>`

| Feature | `<style>`/`<script>` | `<SQL>` |
|---------|---------------------|---------|
| Raw text content | ✅ | ✅ |
| `@{}` interpolation | ✅ Allowed | ❌ **Disallowed** |
| Parameters | N/A | Via attributes only |

The `@{}` interpolation is intentionally disabled for `<SQL>` tags to enforce safe parameterized queries. Users who need dynamic SQL can use raw strings or template strings outside the tag.

## Acceptance Criteria

### Core Functionality
- [ ] `<SQL>` tag content is parsed as raw text (no quotes needed)
- [ ] Leading and trailing whitespace is trimmed from SQL content
- [ ] `@{}` interpolation inside `<SQL>` produces a clear error message
- [ ] All existing `<SQL>` tests continue to pass (with updated syntax)
- [ ] Parameterized queries work as before (params from attributes)

### Error Handling
- [ ] `@{` inside SQL content produces error: "Interpolation is not allowed inside <SQL> tags. Use attributes for parameters."
- [ ] Helpful hint suggests the safe pattern: `<SQL name={value}>...VALUES (?)...</SQL>`

### Whitespace Handling
- [ ] Leading whitespace (indentation) is trimmed
- [ ] Trailing whitespace is trimmed
- [ ] Internal whitespace is preserved (e.g., `INSERT INTO` stays as two words)

### Examples That Must Work

```parsley
// Simple query
let users = db <=??=> <SQL>SELECT * FROM users</SQL>

// Parameterized insert
let InsertUser = fn(props) {
    <SQL name={props.name} email={props.email}>
        INSERT INTO users (name, email) VALUES (?, ?)
    </SQL>
}

// Multi-line query
let GetActiveUsers = fn(props) {
    <SQL status={props.status} limit={props.limit}>
        SELECT id, name, email
        FROM users
        WHERE status = ?
        ORDER BY created_at DESC
        LIMIT ?
    </SQL>
}

// Query with SQL comments (preserved)
<SQL>
    -- Get all active users
    SELECT * FROM users WHERE active = 1
</SQL>
```

### Examples That Must Error

```parsley
// MUST produce error - interpolation not allowed
<SQL>
    SELECT * FROM users WHERE name = '@{name}'
</SQL>
// Error: Interpolation is not allowed inside <SQL> tags.
//        Use attributes for parameters: <SQL name={name}>...WHERE name = ?...</SQL>
```

## Design Decisions

### Why Disallow `@{}` Interpolation?

**Safety by default.** The `<SQL>` tag exists specifically to encourage parameterized queries. Allowing `@{}` would make it trivially easy to write SQL injection vulnerabilities:

```parsley
// If @{} were allowed, this would be vulnerable:
<SQL>SELECT * FROM users WHERE name = '@{userInput}'</SQL>
```

Users who genuinely need dynamic SQL (e.g., dynamic table names) can construct the query outside the tag:

```parsley
// Dynamic SQL when truly needed (rare):
let query = `SELECT * FROM {tableName} WHERE id = ?`
db <=?=> <SQL id={id}>{query}</SQL>

// Or use raw strings for fully dynamic queries (user takes responsibility)
db <=?=> `SELECT * FROM users WHERE name = '${escapedName}'`
```

### Why Trim Whitespace?

Database servers ignore leading/trailing whitespace in SQL, and trimming makes the output cleaner:

```parsley
<SQL>
    SELECT * FROM users
</SQL>
// Results in sql: "SELECT * FROM users"
// Not: sql: "\n    SELECT * FROM users\n"
```

Internal whitespace (between tokens) is preserved to maintain SQL readability.

### Consistency with `<style>`/`<script>`

The implementation mirrors `<style>` and `<script>` tags, which already use raw text mode. This provides:
- Familiar mental model for Parsley developers
- Reusable infrastructure (lexer's raw text mode)
- Consistent behavior across "content" tags

The only difference is the explicit disabling of `@{}` for safety.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components

| Component | Location | Change Type | Notes |
|-----------|----------|-------------|-------|
| Lexer | `pkg/parsley/lexer/lexer.go` | Modify | Add "SQL" to raw text tag checks |
| Lexer | `pkg/parsley/lexer/lexer.go` | Modify | Block `@{` in SQL mode with error |
| Evaluator | `pkg/parsley/evaluator/eval_tags.go` | Modify | Trim whitespace in `evalSQLTag` |
| Tests | `pkg/parsley/tests/database_test.go` | Update | Update existing tests, add new ones |
| Tree-sitter | `contrib/tree-sitter-parsley/grammar.js` | Add | Add `sql_tag` rule (optional) |
| Tree-sitter | `contrib/tree-sitter-parsley/src/scanner.c` | Modify | Add SQL to close tag detection |

### Implementation Details

#### 1. Lexer Changes (`pkg/parsley/lexer/lexer.go`)

**Add SQL to raw text tag detection (~L858):**

```go
// Current:
if tagName == "style" || tagName == "script" {
    l.inRawTextTag = tagName
    l.inTagContent = true
}

// Change to:
if tagName == "style" || tagName == "script" || tagName == "SQL" {
    l.inRawTextTag = tagName
    l.inTagContent = true
}
```

**Same change needed at ~L2017 in `nextTagContentToken`.**

**Block `@{` interpolation for SQL (~L2053):**

```go
case '@':
    // In raw text mode, @{ triggers interpolation
    if inRawMode && l.peekChar() == '{' {
        // NEW: Block interpolation in SQL tags
        if l.inRawTextTag == "SQL" {
            // Return error token or emit error
            // Details TBD based on error handling approach
        }
        // ... existing interpolation handling for style/script
    }
```

#### 2. Evaluator Changes (`pkg/parsley/evaluator/eval_tags.go`)

**Trim whitespace in `evalSQLTag` (~L1115):**

```go
sqlStr, ok := sqlContent.(*String)
if !ok {
    // ... error handling
}

// NEW: Trim leading and trailing whitespace
trimmedSQL := strings.TrimSpace(sqlStr.Value)

// Build result dictionary with trimmed sql
resultPairs := map[string]ast.Expression{
    "sql": &ast.StringLiteral{Value: trimmedSQL},
}
```

#### 3. Test Updates (`pkg/parsley/tests/database_test.go`)

Update existing `TestSQLTag` cases to remove quotes:

```go
// Before:
input: `<SQL>"INSERT INTO users (name) VALUES (?)"</SQL>`

// After:
input: `<SQL>INSERT INTO users (name) VALUES (?)</SQL>`
```

Add new test cases:
- Multi-line SQL with indentation (verify trimming)
- SQL with internal comments (verify preservation)
- `@{` in SQL content (verify error)

### Tree-sitter Grammar (Future Enhancement)

#### SQL Syntax Highlighting via Language Injection

Tree-sitter supports **language injection**, where content inside a tag can be highlighted using a different language's grammar. This would enable SQL syntax highlighting inside `<SQL>` tags.

**How it works:**

1. Define an `sql_tag` rule in `grammar.js` (similar to `style_tag` and `script_tag`)
2. Update `scanner.c` to detect `</SQL>` as a close tag
3. Create a `queries/injections.scm` file that tells editors to use SQL grammar for the content

**Example `injections.scm`:**

```scheme
; Inject SQL language into <SQL> tag content
((sql_tag
  (raw_text) @injection.content)
 (#set! injection.language "sql"))
```

**Benefits:**
- SQL keywords (`SELECT`, `FROM`, `WHERE`) get proper highlighting
- SQL strings, numbers, and operators are colored appropriately
- Editor features like SQL formatting could work inside the tag

**Considerations:**
- Requires users to have a SQL tree-sitter grammar installed
- Not all editors support language injection equally
- Implementation is independent of the core Parsley changes

**Recommendation:** Implement as a follow-up enhancement after the core feature is stable. The raw text functionality works without tree-sitter changes; syntax highlighting is purely cosmetic.

### Dependencies

- None (self-contained feature)

### Effort Estimate

| Task | Estimate |
|------|----------|
| Lexer changes (add SQL to raw text) | 30 min |
| Lexer changes (block @{} for SQL) | 1 hour |
| Evaluator changes (whitespace trim) | 15 min |
| Update existing tests | 30 min |
| Add new test cases | 1 hour |
| Documentation updates | 30 min |
| **Total** | **~4 hours** |

Tree-sitter grammar changes (optional, for syntax highlighting): +2-3 hours

## Migration

### Backward Compatibility

The quoted syntax will continue to work since `evalTagContents` evaluates any expression to a string. However, we should:

1. Update all documentation to show the new unquoted syntax
2. Update all examples in the codebase
3. Consider a lint warning for quoted SQL content (optional, low priority)

### Documentation Updates

- `docs/parsley/reference.md` — Update SQL tag section
- `docs/parsley/CHEATSHEET.md` — Update SQL examples
- `.github/skills/basil-development/references/DATABASE.md` — Update examples

## Testing Plan

### Unit Tests

1. **Basic raw text parsing**
   - `<SQL>SELECT * FROM users</SQL>` → `{sql: "SELECT * FROM users"}`

2. **Parameterized query**
   - `<SQL name={n}>INSERT INTO t (name) VALUES (?)</SQL>` → `{sql: "...", params: {name: ...}}`

3. **Multi-line with indentation**
   ```parsley
   <SQL>
       SELECT *
       FROM users
   </SQL>
   ```
   → `{sql: "SELECT *\nFROM users"}` (leading/trailing whitespace trimmed, internal preserved)

4. **SQL comments preserved**
   - `<SQL>-- comment\nSELECT 1</SQL>` → `{sql: "-- comment\nSELECT 1"}`

5. **Interpolation blocked**
   - `<SQL>SELECT @{x}</SQL>` → Error with helpful message

### Integration Tests

- Full query execution with new syntax
- Component returning `<SQL>` tag works with database operators

## Related

- Depends on: None
- Blocks: None
- Related: FEAT-098 (PLN infrastructure, similar literal handling patterns)

## Notes

- The `<SQL>` tag name is uppercase to distinguish it as a "special" tag (like custom components) rather than an HTML element
- Database servers universally ignore leading/trailing whitespace in SQL statements
- This change makes `<SQL>` consistent with JSX-style "embedded DSL" patterns used in other frameworks