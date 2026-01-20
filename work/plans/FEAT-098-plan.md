---
id: PLAN-070
feature: FEAT-098
title: "Implementation Plan for Parsley Literal Notation (PLN)"
status: draft
created: 2026-01-20
---

# Implementation Plan: FEAT-098 (PLN)

## Overview

Implement PLN (Parsley Literal Notation), a data serialization format for Parsley. This includes:
1. A dedicated PLN parser (values only, no code execution)
2. Serialization from Parsley values to PLN strings
3. `serialize()` and `deserialize()` builtin functions
4. Auto-serialization in Part props
5. `.pln` file loading support
6. HMAC signing for transport security

## Prerequisites
- [ ] Design document reviewed: `work/design/PLN-design.md`
- [ ] Feature spec approved: `work/specs/FEAT-098.md`

---

## Phase 1: Core PLN Package

### Task 1.1: PLN Lexer
**Files**: `pkg/parsley/pln/lexer.go`, `pkg/parsley/pln/lexer_test.go`
**Estimated effort**: Medium

Create a minimal lexer that tokenizes PLN-specific syntax.

Steps:
1. Define token types: `INT`, `FLOAT`, `STRING`, `TRUE`, `FALSE`, `NULL`, `LBRACE`, `RBRACE`, `LBRACKET`, `RBRACKET`, `LPAREN`, `RPAREN`, `COLON`, `COMMA`, `AT`, `IDENT`, `DATETIME`, `PATH`, `URL`, `COMMENT`, `EOF`, `ILLEGAL`
2. Implement `Lexer` struct with `input`, `position`, `readPosition`, `ch`, `line`, `column`
3. Implement `NextToken()` that:
   - Skips whitespace and comments
   - Recognizes `@` prefix for records, datetimes, paths, URLs
   - Handles string literals (double-quoted with escapes)
   - Handles numbers (int and float)
   - Handles identifiers and keywords (`true`, `false`, `null`, `errors`)
4. Implement datetime detection after `@`: ISO date/time patterns
5. Implement path detection after `@`: starts with `/`, `./`, `../`, `~`
6. Implement URL detection after `@`: starts with `http://`, `https://`

Tests:
- Lex primitives: `42`, `3.14`, `"hello"`, `true`, `false`, `null`
- Lex collections: `[1, 2]`, `{a: 1}`
- Lex records: `@Person({name: "Alice"})`
- Lex errors suffix: `@errors {name: "Required"}`
- Lex datetimes: `@2024-01-20`, `@2024-01-20T10:30:00Z`, `@10:30:00`
- Lex paths: `@/path/to/file`, `@./relative`, `@~/home`
- Lex URLs: `@https://example.com`
- Lex comments: `// comment\n42`
- Error on invalid tokens

---

### Task 1.2: PLN Parser
**Files**: `pkg/parsley/pln/parser.go`, `pkg/parsley/pln/parser_test.go`
**Estimated effort**: Medium

Create a recursive descent parser for PLN values.

Steps:
1. Define parser struct with lexer, current/peek tokens, errors
2. Implement `Parse()` → returns `object.Object` or error
3. Implement `parseValue()` dispatch:
   - `INT` → `*object.Integer`
   - `FLOAT` → `*object.Float`
   - `STRING` → `*object.String`
   - `TRUE/FALSE` → `*object.Boolean`
   - `NULL` → `*object.Null`
   - `LBRACKET` → `parseArray()`
   - `LBRACE` → `parseDict()`
   - `AT` → `parseTypedValue()` (record, datetime, path, URL)
4. Implement `parseArray()`: `[` value (`,` value)* `,`? `]`
5. Implement `parseDict()`: `{` pair (`,` pair)* `,`? `}`
6. Implement `parsePair()`: (IDENT | STRING) `:` value
7. Implement `parseTypedValue()`:
   - If next is IDENT → record: `@Name({...})` with optional `@errors {...}`
   - If next is DATETIME → datetime literal
   - If next is PATH → path literal
   - If next is URL → URL literal
8. Track nesting depth, error if > 100

Tests:
- Parse primitives round-trip
- Parse arrays: `[]`, `[1]`, `[1, 2, 3]`, `[1,]` (trailing comma)
- Parse dicts: `{}`, `{a: 1}`, `{a: 1, b: 2}`, `{"a": 1}`
- Parse nested: `{a: [1, {b: 2}]}`
- Parse records: `@Person({name: "Alice"})`
- Parse records with errors: `@Person({name: ""}) @errors {name: "Required"}`
- Parse datetimes: date, datetime, time
- Parse paths and URLs
- Error on expressions: `1 + 1`
- Error on function calls: `Person({name: "Alice"})`
- Error on variables: `{name: x}`
- Error on deep nesting (>100)

---

### Task 1.3: PLN Serializer
**Files**: `pkg/parsley/pln/serializer.go`, `pkg/parsley/pln/serializer_test.go`
**Estimated effort**: Medium

Convert Parsley objects to PLN strings.

Steps:
1. Implement `Serialize(obj object.Object) (string, error)`
2. Handle primitives: int, float, string, bool, null
3. Handle arrays: recursively serialize elements
4. Handle dicts: recursively serialize key-value pairs
5. Handle records:
   - Get schema name from record
   - Serialize as `@SchemaName({...})`
   - If record has errors, append `@errors {...}`
6. Handle datetime: format as `@ISO8601`
7. Handle path: format as `@/path/literal`
8. Handle URL: format as `@https://...`
9. Error on non-serializable: function, builtin, db connection, file handle, module
10. Track visited objects for circular reference detection
11. Optionally support pretty-printing (indent parameter)

Tests:
- Serialize all primitive types
- Serialize arrays (empty, nested)
- Serialize dicts (empty, nested)
- Serialize records (with and without errors)
- Serialize datetimes (date, datetime with TZ)
- Serialize paths and URLs
- Error on function
- Error on circular reference
- Round-trip: serialize then parse equals original

---

### Task 1.4: Public API
**Files**: `pkg/parsley/pln/pln.go`
**Estimated effort**: Small

Expose clean public API.

Steps:
1. Export `Serialize(obj object.Object) (string, error)`
2. Export `Deserialize(pln string, schemaResolver func(string) *object.Schema) (object.Object, error)`
3. Export `SerializePretty(obj object.Object, indent string) (string, error)`
4. Schema resolver allows looking up schemas by name during deserialization

Tests:
- API integration test: full round-trip
- Schema resolution test: `@Person({...})` with resolver returns Record
- Unknown schema test: returns dict with `__schema` field

---

## Phase 2: Builtin Functions

### Task 2.1: Add `serialize()` builtin
**Files**: `pkg/parsley/evaluator/stdlib_core.go`, `pkg/parsley/evaluator/stdlib_core_test.go`
**Estimated effort**: Small

Steps:
1. Register `serialize` builtin in `init()`
2. Implement: takes one argument, calls `pln.Serialize()`
3. Return string or error

Tests:
- `serialize(42)` → `"42"`
- `serialize({a: 1})` → `"{a: 1}"`
- `serialize(Person({name: "Alice"}))` → `"@Person({name: \"Alice\"})"`
- `serialize(fn(x) { x })` → error

---

### Task 2.2: Add `deserialize()` builtin
**Files**: `pkg/parsley/evaluator/stdlib_core.go`, `pkg/parsley/evaluator/stdlib_core_test.go`
**Estimated effort**: Small

Steps:
1. Register `deserialize` builtin in `init()`
2. Implement: takes one string argument
3. Create schema resolver from current environment
4. Call `pln.Deserialize()` with resolver
5. Return parsed value or error

Tests:
- `deserialize("42")` → `42`
- `deserialize("{a: 1}")` → `{a: 1}`
- `deserialize("@Person({name: \"Alice\"})")` with Person schema → Person record
- `deserialize("@Unknown({x: 1})")` → `{x: 1, __schema: "Unknown"}` + warning
- `deserialize("1 + 1")` → error

---

## Phase 3: File Loading

### Task 3.1: Support `.pln` file loading
**Files**: `pkg/parsley/evaluator/stdlib_io.go`, `pkg/parsley/evaluator/stdlib_io_test.go`
**Estimated effort**: Small

Steps:
1. In `load()` function, check file extension
2. If `.pln`, read file contents and call `pln.Deserialize()`
3. Return parsed value

Tests:
- `load(@/path/to/data.pln)` returns parsed PLN
- Error on invalid PLN file

---

## Phase 4: Part Integration

### Task 4.1: Auto-serialize Part props
**Files**: `server/parts.go`, `server/parts_test.go`
**Estimated effort**: Medium

Steps:
1. When rendering `<Part>` tag, check each prop value
2. For complex types (Record, DateTime, etc.), serialize to PLN
3. Store serialized props in `data-part-props` attribute
4. Sign the serialized data with HMAC

Tests:
- Part with integer prop: passed as-is
- Part with record prop: serialized to PLN, HMAC signed
- Part with datetime prop: serialized to PLN
- Part with function prop: error (cannot serialize)

---

### Task 4.2: Auto-deserialize Part props on fetch
**Files**: `server/parts.go`, `server/parts_test.go`
**Estimated effort**: Medium

Steps:
1. When Part view is fetched, check for HMAC signature on props
2. Validate HMAC before deserializing
3. Deserialize PLN props back to Parsley values
4. Pass deserialized values to view function

Tests:
- Part request with PLN prop: deserialized to record
- Part request with tampered HMAC: rejected
- Part request with plain props: handled normally

---

### Task 4.3: HMAC utilities for PLN
**Files**: `server/session_crypto.go`, `server/session_crypto_test.go`
**Estimated effort**: Small

Steps:
1. Add `SignPLN(pln string, secret []byte) string`
2. Add `VerifyPLN(signed string, secret []byte) (string, bool)`
3. Format: `HMAC:BASE64_PLN`

Tests:
- Sign and verify round-trip
- Tampered data fails verification
- Different secret fails verification

---

## Phase 5: Tests

### Task 5.1: Integration tests
**Files**: `pkg/parsley/tests/pln_test.go`
**Estimated effort**: Medium

Parsley-level integration tests:

Tests:
- Serialize record, deserialize, compare
- Serialize record with validation errors, round-trip
- Serialize nested records
- `load()` PLN file with records
- Error messages are clear and actionable

---

### Task 5.2: Part integration tests
**Files**: `server/parts_pln_test.go`
**Estimated effort**: Medium

End-to-end Part tests:

Tests:
- Part with record prop renders correctly
- Part interaction preserves record across views
- Part with validation errors preserves errors across views
- Part prop tampering is rejected

---

## Phase 6: Documentation

### Task 6.1: PLN reference documentation
**Files**: `docs/parsley/manual/pln.md`
**Estimated effort**: Medium

Steps:
1. Copy structure from design doc
2. Add complete syntax reference
3. Add serialize/deserialize API docs
4. Add examples for each use case
5. Add security section

---

### Task 6.2: Update Parts documentation
**Files**: `docs/guide/parts.md`
**Estimated effort**: Small

Steps:
1. Add section on passing complex props
2. Show record with errors example
3. Mention auto-serialization

---

### Task 6.3: Update Parsley reference
**Files**: `docs/parsley/reference.md`
**Estimated effort**: Small

Steps:
1. Add `serialize()` and `deserialize()` to builtins
2. Add PLN file loading to `load()` documentation

---

### Task 6.4: Update cheatsheet
**Files**: `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small

Steps:
1. Add PLN syntax quick reference
2. Note differences from JSON

---

## Validation Checklist
- [x] All tests pass: `make test`
- [x] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Design doc: `work/design/PLN-design.md`
- [x] Reference doc: `docs/parsley/manual/pln.md`
- [x] Parts doc updated: `docs/guide/parts.md` (PLN props section needed)
- [x] Parsley reference updated: `docs/parsley/reference.md`
- [x] Cheatsheet updated: `docs/parsley/CHEATSHEET.md`
- [x] work/BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-01-20 | Phase 1.1: PLN Lexer | Complete | 12 tests |
| 2026-01-20 | Phase 1.2: PLN Parser | Complete | 24 tests |
| 2026-01-20 | Phase 1.3: PLN Serializer | Complete | 20+ tests |
| 2026-01-20 | Phase 1.4: Public API | Complete | 12 tests |
| 2026-01-20 | Phase 2: Builtin functions | Complete | serialize/deserialize builtins |
| 2026-01-20 | Phase 3: File loading | Complete | PLN() builtin, .pln auto-detection |
| 2026-01-15 | Phase 4: Part integration | Complete | HMAC signing, JS runtime, parsePartProps |
| 2026-01-20 | Phase 5: Integration tests | Complete | 26 tests in pkg/parsley/tests/pln_test.go |
| 2026-01-20 | Phase 6: Documentation | Complete | PLN manual page, reference updates |

## Deferred Items
Items to add to work/BACKLOG.md after implementation:
- ~~Part props auto-serialization with PLN (Phase 4)~~ ✅ Completed
- ~~HMAC signing for PLN in transport (Phase 4)~~ ✅ Completed
- ~~Record serialization with @Schema syntax~~ ✅ Completed in Phase 3
- ~~DateTime serialization with @ISO format~~ ✅ Completed in Phase 3
- ~~Path/URL serialization with @ prefix~~ ✅ Completed in Phase 3
- PLN pretty-print CLI tool (`pars fmt file.pln`)
- PLN syntax highlighting for VS Code
- PLN schema validation mode (strict vs permissive)
- Binary PLN format for large data (performance optimization)
