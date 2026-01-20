---
id: FEAT-098
title: "Parsley Literal Notation (PLN)"
status: draft
priority: medium
created: 2026-01-20
author: "@human"
---

# FEAT-098: Parsley Literal Notation (PLN)

## Summary
PLN is a data serialization format for Parsley that uses a safe subset of Parsley syntax to represent values—including schema-bound records and validation errors—without allowing code execution. It enables seamless passing of complex data between Parts, typed data files, and API responses.

## User Story
As a Basil developer, I want to pass rich data (records with schemas, validation errors, dates) between Parts so that I don't have to manually serialize/deserialize or lose type information at component boundaries.

## Acceptance Criteria
- [x] `serialize(value)` converts supported values to PLN strings
- [x] `deserialize(pln)` parses PLN strings back to Parsley values
- [ ] Records preserve schema association: `@Person({...})` → Person record
- [ ] Records preserve validation errors: `@Person({...}) @errors {...}`
- [ ] Native datetime support: `@2024-01-20T10:30:00Z`
- [ ] Native path/URL support: `@/path/to/file`, `@https://example.com`
- [x] Comments supported: `// comment`
- [x] Non-serializable values (functions, handles) produce clear errors
- [ ] Part props auto-serialize complex values to PLN
- [ ] Part props auto-deserialize PLN on receipt
- [ ] HMAC signing protects PLN in transit
- [x] `.pln` files can be loaded via `file(@/path/to/data.pln)` or `PLN(@/path/to/data.pln)`

## Design Decisions

- **Safe subset, not eval**: PLN uses a dedicated parser that only accepts values. No expressions, variables, or function calls. This eliminates code injection risks.

- **`@` prefix for typed values**: Records use `@Schema({...})` to distinguish from function calls `Schema({...})`. This is consistent with existing Parsley datetime syntax `@2024-01-20`.

- **Errors attached, not embedded**: Validation errors use `@errors {...}` suffix rather than embedding in the record. Keeps data clean, errors are metadata.

- **Schema by name, not embedded**: PLN references schemas by name (`@Person`) rather than embedding schema definitions. Schemas must be in scope at deserialization time. This avoids XML Schema complexity.

- **Graceful unknown schemas**: When deserializing `@UnknownSchema({...})`, return a dictionary with `__schema: "UnknownSchema"` and warn in dev mode. Don't fail silently, don't crash.

- **HMAC for transport only**: PLN in files is not signed (files are trusted). PLN in HTTP/Part props is HMAC-signed to prevent tampering.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components

**New files:**
- `pkg/parsley/pln/lexer.go` — PLN tokenizer (subset of Parsley lexer)
- `pkg/parsley/pln/parser.go` — PLN parser (values only, no expressions)
- `pkg/parsley/pln/serializer.go` — Value → PLN string conversion
- `pkg/parsley/pln/pln.go` — Public API: `Serialize()`, `Deserialize()`

**Modified files:**
- `pkg/parsley/evaluator/stdlib_core.go` — Add `serialize()`, `deserialize()` builtins
- `pkg/parsley/evaluator/stdlib_io.go` — Handle `.pln` file loading
- `server/parts.go` — Auto-serialize/deserialize Part props
- `server/session_crypto.go` — HMAC signing for PLN in transport

### Serializable Types

| Go Type | PLN Representation |
|---------|-------------------|
| `*object.Integer` | `42` |
| `*object.Float` | `3.14` |
| `*object.String` | `"hello"` |
| `*object.Boolean` | `true`, `false` |
| `*object.Null` | `null` |
| `*object.Array` | `[1, 2, 3]` |
| `*object.Dict` | `{a: 1, b: 2}` |
| `*object.Record` | `@Schema({...})` or `@Schema({...}) @errors {...}` |
| `*object.DateTime` | `@2024-01-20T10:30:00Z` |
| `*object.Path` | `@/path/to/file` |
| `*object.URL` | `@https://example.com` |

Non-serializable: `*object.Function`, `*object.Builtin`, `*object.DBConnection`, `*object.FileHandle`, `*object.Module`

### Grammar

```ebnf
value       = primitive | array | dict | record | datetime | path | url
primitive   = INTEGER | FLOAT | STRING | 'true' | 'false' | 'null'
array       = '[' (value (',' value)* ','?)? ']'
dict        = '{' (pair (',' pair)* ','?)? '}'
pair        = (IDENT | STRING) ':' value
record      = '@' IDENT '(' dict ')' errors?
errors      = '@errors' dict
datetime    = '@' ISO_DATETIME
path        = '@' PATH_LITERAL
url         = '@' URL_LITERAL
comment     = '//' .* NEWLINE
```

### Dependencies
- Depends on: None (standalone feature)
- Blocks: None

### Edge Cases & Constraints

1. **Circular references** — Detect and error: "Cannot serialize circular reference"
2. **Deep nesting** — Limit to 100 levels, error if exceeded
3. **Large values** — No hard limit, but HMAC overhead scales with size
4. **Unknown schema on deserialize** — Return dict with `__schema` field, warn in dev
5. **Schema mismatch** — If schema exists but data doesn't match, validate and attach errors
6. **String escaping** — Use Parsley string escaping rules (same as JSON + raw strings)

### Security Considerations

1. **Parser isolation** — PLN parser is separate from main Parsley parser. No code paths lead to expression evaluation.
2. **HMAC validation** — On deserialize from transport, validate HMAC first. Reject tampered data before parsing.
3. **Schema validation** — Even with valid HMAC, validate against schema. Defense in depth.
4. **No eval fallback** — Never fall back to `eval()` or main parser for "convenience"

## Implementation Notes
*Added during/after implementation*

## Related
- Design doc: `work/design/PLN-design.md`
- Plan: `work/plans/FEAT-098-plan.md` (to be created)
