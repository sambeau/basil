---
id: FEAT-082
title: "Auto-translate == null to IS NULL in Query DSL"
status: proposed
priority: low
created: 2026-01-05
author: "@human"
---

# FEAT-082: Auto-translate == null to IS NULL in Query DSL

## Summary

Automatically translate `== null` to `IS NULL` and `!= null` to `IS NOT NULL` in the Query DSL for user convenience, avoiding a common SQL pitfall.

## User Story

As a developer who may not be deeply familiar with SQL NULL semantics, I want `== null` to work intuitively in queries so that I don't have to remember the `is null` syntax.

## Background

In standard SQL, comparing anything to NULL with `=` returns `NULL` (unknown), not `true` or `false`:

```sql
-- These return NO ROWS, even if active IS NULL:
SELECT * FROM users WHERE active = NULL;
SELECT * FROM users WHERE active != NULL;

-- These work correctly:
SELECT * FROM users WHERE active IS NULL;
SELECT * FROM users WHERE active IS NOT NULL;
```

Parsley the language treats `null == null` as `true`, which creates a semantic mismatch when users expect the same behavior in the Query DSL.

## Current Behavior

```parsley
@query(Users | active is null ??-> *)      // ✓ Works
@query(Users | active == null ??-> *)      // Returns no rows (SQL behavior)
```

## Proposed Behavior

```parsley
@query(Users | active is null ??-> *)      // ✓ Works (unchanged)
@query(Users | active == null ??-> *)      // ✓ Translates to IS NULL
@query(Users | active != null ??-> *)      // ✓ Translates to IS NOT NULL
```

## Acceptance Criteria

- [ ] `column == null` translates to `column IS NULL`
- [ ] `column != null` translates to `column IS NOT NULL`
- [ ] `column == {expr}` where expr evaluates to null also uses IS NULL
- [ ] Existing `is null` / `is not null` syntax continues to work
- [ ] Documentation updated

## Design Considerations

### Arguments For

1. **Principle of least surprise**: Matches Parsley's own `== null` behavior
2. **Common pitfall**: Many developers get this wrong in SQL
3. **Other ORMs do this**: ActiveRecord, Ecto, and others auto-translate

### Arguments Against

1. **Hides SQL semantics**: Users who know SQL may be surprised
2. **Inconsistent with raw SQL**: Could cause confusion when mixing DSL and raw queries
3. **Magic behavior**: Implicit translation is harder to debug

### Implementation Notes

Detection points in `stdlib_dsl_query.go`:
- In `buildConditionSQL` when processing `==` or `!=` operators
- Check if right-hand side is `NullValue` or evaluates to `nil`
- Generate `IS NULL` / `IS NOT NULL` instead of `= ?` / `!= ?`

## Alternatives Considered

1. **Status quo**: Just document the behavior (done in FEAT-082 discussion)
2. **Warning**: Emit a warning when `== null` is used
3. **Strict mode**: Add a config option to control behavior

## Decision

Deferred for community feedback. The current workaround (`is null`) is well-documented.
