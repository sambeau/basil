---
id: PLAN-094
feature: FEAT-117
title: "Implementation Plan for SQL Tag Raw Text Content"
status: complete
created: 2025-02-14
completed: 2025-02-14
---

# Implementation Plan: SQL Tag Raw Text Content

## Overview

Make `<SQL>` tags treat their content as raw text (like `<style>` and `<script>`), eliminating the need for quotes around SQL code. Additionally, block `@{}` interpolation inside SQL tags to enforce safe parameterized queries.

**Based on:** FEAT-117 specification

## Prerequisites

- [x] Understand lexer raw text mode (`inRawTextTag`, `inTagContent`)
- [x] Understand how `<style>`/`<script>` tags work in lexer
- [x] Understand `evalSQLTag()` in evaluator
- [x] Existing `<SQL>` tests work with quoted syntax

## Tasks

### Task 1: Add SQL to Raw Text Tags in Lexer (Main Token Loop)
**Location**: `pkg/parsley/lexer/lexer.go` (~L858)
**Estimated effort**: Small (15 min)

Steps:
1. Locate the `TAG_START` handling in `NextToken()` (~L850-862)
2. Find the raw text tag check: `if tagName == "style" || tagName == "script"`
3. Add `|| tagName == "SQL"` to the condition

Before:
```go
tagName := extractTagName(tagContent)
if tagName == "style" || tagName == "script" {
    l.inRawTextTag = tagName
    l.inTagContent = true
}
```

After:
```go
tagName := extractTagName(tagContent)
if tagName == "style" || tagName == "script" || tagName == "SQL" {
    l.inRawTextTag = tagName
    l.inTagContent = true
}
```

Tests:
- Lexer enters raw text mode for `<SQL>` tags
- Content is tokenized as `TAG_TEXT`

---

### Task 2: Add SQL to Raw Text Tags in Tag Content Mode
**Location**: `pkg/parsley/lexer/lexer.go` (~L2017)
**Estimated effort**: Small (10 min)

Steps:
1. Locate `nextTagContentToken()` function
2. Find the nested tag raw text check (~L2017): `if tagName == "style" || tagName == "script"`
3. Add `|| tagName == "SQL"` to the condition

This handles nested `<SQL>` tags (rare but should be consistent).

Tests:
- Nested SQL tags work correctly

---

### Task 3: Block @{} Interpolation for SQL Tags
**Location**: `pkg/parsley/lexer/lexer.go` (~L2053)
**Estimated effort**: Medium (45 min)

Steps:
1. Locate the `@` case in `nextTagContentToken()` (~L2053)
2. Find the interpolation handling: `if inRawMode && l.peekChar() == '{'`
3. Add check for SQL mode before entering interpolation
4. Return an error token or produce a lexer error for SQL

Implementation approach:
```go
case '@':
    // In raw text mode, @{ triggers interpolation
    if inRawMode && l.peekChar() == '{' {
        // Block interpolation in SQL tags for safety
        if l.inRawTextTag == "SQL" {
            // Consume @{ to show in error location
            line := l.line
            col := l.column
            l.readChar() // skip @
            l.readChar() // skip {
            // Return ILLEGAL token with error message
            return Token{
                Type:    ILLEGAL,
                Literal: "interpolation @{} is not allowed inside <SQL> tags; use attributes for parameters",
                Line:    line,
                Column:  col,
            }
        }
        // ... existing interpolation handling for style/script
    }
```

Tests:
- `<SQL>SELECT @{x}</SQL>` produces error
- `<style>.foo { color: @{c}; }</style>` still works
- Error message is clear and helpful

---

### Task 4: Add SQL Close Tag Detection in External Scanner
**Location**: `contrib/tree-sitter-parsley/src/scanner.c` (~L160)
**Estimated effort**: Medium (30 min)

Steps:
1. Locate `scan_raw_text()` function
2. Find the close tag detection for `</style>` and `</script>`
3. Add detection for `</SQL>`

Add after the `</script>` detection block (~L210):
```c
// Check for 'SQL' (case-sensitive, uppercase)
} else if (lexer->lookahead == 'S') {
    advance(lexer);
    if (lexer->lookahead == 'Q') {
        advance(lexer);
        if (lexer->lookahead == 'L') {
            advance(lexer);
            // Check for end of tag name
            if (lexer->lookahead == '>' || lexer->lookahead == ' ' ||
                lexer->lookahead == '\t' || lexer->lookahead == '\n' ||
                lexer->lookahead == '\r') {
                // Found </SQL>!
                if (has_content) {
                    lexer->result_symbol = RAW_TEXT;
                    return true;
                }
                return false;
            }
        }
    }
}
```

Tests:
- Tree-sitter parses `<SQL>...</SQL>` correctly
- Raw text content is captured

---

### Task 5: Add sql_tag Rule to Tree-sitter Grammar
**Location**: `contrib/tree-sitter-parsley/grammar.js` (~L632)
**Estimated effort**: Small (20 min)

Steps:
1. Locate `script_tag` rule (~L632)
2. Add `sql_tag` rule after it, following same pattern
3. Add `$.sql_tag` to the `tag_expression` choice (~L608)

Implementation:
```javascript
// SQL tag with raw text content (NO interpolation allowed)
sql_tag: ($) =>
  seq(
    token(prec(PREC.TAG + 1, "<SQL")),
    repeat(choice($.tag_attribute, $.tag_spread_attribute)),
    ">",
    repeat($.raw_text),  // Note: only raw_text, no interpolation
    token(prec(PREC.TAG + 1, "</SQL>")),
  ),
```

Update `tag_expression`:
```javascript
tag_expression: ($) =>
  prec(
    PREC.TAG,
    choice(
      $.self_closing_tag,
      $.style_tag,
      $.script_tag,
      $.sql_tag,  // Add this
      seq($.open_tag, repeat($._tag_child), $.close_tag),
      seq("<>", repeat($._tag_child), "</>"),
    ),
  ),
```

Tests:
- Tree-sitter grammar compiles
- SQL tags parse correctly
- Syntax highlighting works for SQL tag structure

---

### Task 6: Trim Whitespace in evalSQLTag
**Location**: `pkg/parsley/evaluator/eval_tags.go` (~L1115)
**Estimated effort**: Small (15 min)

Steps:
1. Locate `evalSQLTag()` function (~L1101)
2. Find where `sqlStr` is used to build result (~L1122)
3. Add `strings.TrimSpace()` call

Before:
```go
sqlStr, ok := sqlContent.(*String)
if !ok {
    // error handling
}

resultPairs := map[string]ast.Expression{
    "sql": &ast.StringLiteral{Value: sqlStr.Value},
}
```

After:
```go
sqlStr, ok := sqlContent.(*String)
if !ok {
    // error handling
}

// Trim leading and trailing whitespace from SQL content
trimmedSQL := strings.TrimSpace(sqlStr.Value)

resultPairs := map[string]ast.Expression{
    "sql": &ast.StringLiteral{Value: trimmedSQL},
}
```

Tests:
- Leading whitespace is trimmed
- Trailing whitespace is trimmed
- Internal whitespace/newlines are preserved

---

### Task 7: Update Existing SQL Tag Tests
**Location**: `pkg/parsley/tests/database_test.go` (~L371)
**Estimated effort**: Medium (30 min)

Steps:
1. Locate `TestSQLTag` function
2. Update all test inputs to remove quotes from SQL content
3. Verify tests still pass with new syntax

Before:
```go
input: `<SQL>"INSERT INTO tag_users (name) VALUES ('Alice')"</SQL>`
```

After:
```go
input: `<SQL>INSERT INTO tag_users (name) VALUES ('Alice')</SQL>`
```

Update all test cases:
- "SQL tag without params in component"
- "SQL tag with params in component - insert"
- "SQL tag with params in component - query"
- "SQL tag with multiple params in component"

Tests:
- All existing tests pass with new syntax
- Test coverage is maintained

---

### Task 8: Add New Test Cases
**Location**: `pkg/parsley/tests/database_test.go`
**Estimated effort**: Medium (45 min)

Steps:
1. Add test for multi-line SQL with indentation
2. Add test for SQL with comments
3. Add test for @{} interpolation error
4. Add test for whitespace trimming

New test cases:

```go
{
    name: "SQL tag with multi-line content",
    input: `
        let db = @sqlite(":memory:")
        let _ = db <=!=> "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)"
        
        let query = <SQL>
            SELECT id, name
            FROM users
            WHERE id = 1
        </SQL>
        query.sql
    `,
    check: func(t *testing.T, result evaluator.Object) {
        str, ok := result.(*evaluator.String)
        if !ok {
            t.Fatalf("Expected String, got %T", result)
        }
        // Verify whitespace trimming and internal preservation
        if !strings.Contains(str.Value, "SELECT id, name") {
            t.Errorf("Expected SQL content, got %s", str.Value)
        }
        if strings.HasPrefix(str.Value, "\n") || strings.HasPrefix(str.Value, " ") {
            t.Errorf("Leading whitespace should be trimmed")
        }
    },
},
{
    name: "SQL tag with SQL comments",
    input: `
        let query = <SQL>
            -- This is a comment
            SELECT * FROM users
        </SQL>
        query.sql
    `,
    check: func(t *testing.T, result evaluator.Object) {
        str, ok := result.(*evaluator.String)
        if !ok {
            t.Fatalf("Expected String, got %T", result)
        }
        if !strings.Contains(str.Value, "-- This is a comment") {
            t.Errorf("SQL comments should be preserved")
        }
    },
},
```

Tests:
- Multi-line SQL works correctly
- SQL comments are preserved
- Whitespace trimming verified

---

### Task 9: Add Interpolation Error Test
**Location**: `pkg/parsley/tests/database_test.go` or `pkg/parsley/lexer/lexer_test.go`
**Estimated effort**: Small (20 min)

Steps:
1. Add test that verifies `@{}` in SQL produces error
2. Verify error message is helpful

```go
{
    name: "SQL tag rejects interpolation",
    input: `<SQL>SELECT * FROM users WHERE id = @{id}</SQL>`,
    expectError: true,
    errorContains: "interpolation",
},
```

Tests:
- Interpolation is rejected
- Error message mentions interpolation
- Error message suggests using attributes

---

### Task 10: Update Documentation - Reference
**Location**: `docs/parsley/reference.md`
**Estimated effort**: Small (20 min)

Steps:
1. Find SQL tag section
2. Update examples to show unquoted syntax
3. Add note about safety (no interpolation)

Tests:
- Examples are correct
- Safety note is clear

---

### Task 11: Update Documentation - Cheatsheet
**Location**: `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small (10 min)

Steps:
1. Find SQL-related examples
2. Update to use unquoted syntax
3. Add note about `@{}` being blocked

Tests:
- Examples compile
- Note is visible

---

### Task 12: Update Documentation - Database Reference
**Location**: `.github/skills/basil-development/references/DATABASE.md`
**Estimated effort**: Small (15 min)

Steps:
1. Find SQL tag examples
2. Update to unquoted syntax
3. Add safety note

Tests:
- Examples are correct
- Consistent with other docs

---

## Validation Checklist

### Code
- [ ] Lexer enters raw text mode for `<SQL>` tags
- [ ] `@{}` interpolation blocked with clear error
- [ ] Whitespace trimming works correctly
- [ ] All existing tests pass (updated syntax)
- [ ] New tests pass (multi-line, comments, error)
- [ ] No compiler warnings

### Tree-sitter (Optional)
- [ ] Grammar compiles: `tree-sitter generate`
- [ ] Scanner detects `</SQL>` close tag
- [ ] SQL tags parse correctly
- [ ] Tests pass: `tree-sitter test`

### Full Validation
- [ ] `make check` passes
- [ ] `golangci-lint run` passes
- [ ] Manual testing confirms functionality

### Documentation
- [ ] reference.md updated
- [ ] CHEATSHEET.md updated
- [ ] DATABASE.md updated
- [ ] All examples tested

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| — | Plan created | ✅ Complete | — |
| 2025-02-14 | Task 1: Lexer main loop | ✅ Complete | Added SQL to raw text tags |
| 2025-02-14 | Task 2: Lexer tag content | ✅ Complete | Added SQL to nested tag handling |
| 2025-02-14 | Task 3: Block @{} | ✅ Complete | Returns ILLEGAL token with helpful message |
| 2025-02-14 | Task 4: Scanner.c | ✅ Complete | Added </SQL> close tag detection |
| 2025-02-14 | Task 5: Grammar.js | ✅ Complete | Added sql_tag rule, 5 new tests pass |
| 2025-02-14 | Task 6: Whitespace trim | ✅ Complete | Added strings.TrimSpace |
| 2025-02-14 | Task 7: Update tests | ✅ Complete | Removed quotes from 4 existing tests |
| 2025-02-14 | Task 8: New tests | ✅ Complete | Added 3 new tests (multiline, comments, inline) |
| 2025-02-14 | Task 9: Error test | ✅ Complete | Added lexer tests for @{} blocking |
| 2025-02-14 | Task 10: reference.md | ✅ Complete | Already uses unquoted syntax |
| 2025-02-14 | Task 11: CHEATSHEET.md | ✅ Complete | Uses raw strings (acceptable) |
| 2025-02-14 | Task 12: DATABASE.md | ✅ Complete | Added SQL Tags section |
| 2025-02-14 | Manual: database.md | ✅ Complete | Updated SQL tag examples, added safety notes |
| 2025-02-14 | Manual: tags.md | ✅ Complete | Added raw text and no-interpolation sections |

## Deferred Items

Items to add to work/BACKLOG.md after implementation:

- **SQL syntax highlighting via language injection** — Tree-sitter can inject SQL grammar for highlighting inside `<SQL>` tags. Requires `queries/injections.scm` file. Low priority, purely cosmetic.

## Implementation Notes

### Task Order

Recommended order for implementation:

1. **Lexer changes first** (Tasks 1-3) — Core functionality
2. **Evaluator change** (Task 6) — Whitespace trimming
3. **Test updates** (Tasks 7-9) — Verify everything works
4. **Documentation** (Tasks 10-12) — After code is stable
5. **Tree-sitter** (Tasks 4-5) — Optional, can be done later

### Testing Strategy

- Run `go test ./pkg/parsley/lexer/...` after Tasks 1-3
- Run `go test ./pkg/parsley/evaluator/...` after Task 6
- Run `go test ./pkg/parsley/tests/...` after Tasks 7-9
- Run `make check` before documentation updates

### Backward Compatibility

The quoted syntax will continue to work since `evalTagContents` evaluates any expression. Old code won't break, but documentation should show only the new syntax.

## Success Criteria

Implementation is successful when:

1. `<SQL>SELECT * FROM users</SQL>` works (no quotes needed)
2. `<SQL>SELECT @{x}</SQL>` produces clear error
3. Whitespace is trimmed from SQL content
4. All tests pass
5. Documentation shows new syntax
6. `make check` passes

## Timeline Estimate

| Phase | Tasks | Time | Notes |
|-------|-------|------|-------|
| Lexer Changes | 1-3 | 1 hour | Core functionality |
| Evaluator | 6 | 15 min | Whitespace trimming |
| Tests | 7-9 | 1.5 hours | Update + new tests |
| Documentation | 10-12 | 45 min | Three files |
| Tree-sitter | 4-5 | 50 min | Optional |
| **Total Core** | **1-3, 6-12** | **~3.5 hours** | Without tree-sitter |
| **Total All** | **1-12** | **~4.5 hours** | With tree-sitter |