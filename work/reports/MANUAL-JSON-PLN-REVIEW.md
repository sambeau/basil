---
id: MANUAL-JSON-PLN-REVIEW
title: "Manual Review: JSON vs PLN Guidance"
date: 2026-02-14
updated: 2026-02-12
status: complete
---

# Manual Review: JSON vs PLN Guidance

## Summary

The Parsley manual currently uses JSON extensively in examples, but many cases involving **local Parsley data** would benefit from suggesting PLN (Parsley Literal Notation) instead. PLN preserves Parsley types (dates, money, paths, durations) that JSON cannot represent, making it the better choice for internal data storage.

**Key principle:** Use JSON for external interoperability (APIs, other systems), use PLN for internal Parsley data (configs, caches, data files).

## Current PLN Implementation Status

**✅ Reading:** Fully implemented
- `PLN()` builtin function exists in `evaluator.go`
- `parsePLN()` function reads PLN files via `<==` operator
- Auto-detection works: `.pln` extension recognized in `inferFormatFromExtension()`

**✅ Writing:** Implemented (FEAT-115)
- `encodePLN()` function added to `eval_encoders.go`
- PLN format added to write switch in `writeFileContent()` (eval_file_io.go)
- `data ==> PLN(@./file.pln)` syntax works
- Append mode supported: `data ==>> PLN(@./file.pln)`

**✅ Native Type Support:** Implemented (FEAT-116)
- Money literals: `USD#19.99`, `JPY#500`
- DateTime literals: `@2024-01-15`, `@2024-01-15T10:30:00`
- Path literals: `@./config/file.json`
- URL literals: `@https://example.com/api`
- Records: `@Person({name: "Alice"})`
- All types round-trip correctly

---

## Recommended Changes

### High Priority: Add PLN Guidance

These sections should explicitly recommend PLN for local data:

#### 1. **features/data-formats.md**

**Current:** Section introduces JSON alongside CSV and Markdown with no clear guidance on when to use each.

**Add after JSON section:**
```markdown
### When to Use PLN vs JSON

**Use PLN for:**
- Configuration files that include dates, money, or other Parsley types
- Caching Parsley data structures
- Data files read/written by Parsley scripts
- Serializing records with schemas
- Debugging Parsley data (PLN output is valid Parsley syntax)

**Use JSON for:**
- API requests and responses
- Data exchange with non-Parsley systems
- When compatibility with JSON parsers is required

PLN round-trips all Parsley types losslessly, while JSON converts:
- Datetimes to strings (must be reparsed)
- Money to numbers (loses currency information)
- Paths and URLs to strings (loses path operations)
- Durations to strings (must be reparsed)
```

---

#### 2. **features/file-io.md**

**Location:** Reading Files section

**Add note after JSON example:**
```markdown
> **Tip:** For configuration files with Parsley-specific types (dates, money, paths), 
> use PLN instead: `let config <== PLN(@./config.pln)`. PLN preserves all Parsley types.
```

**Location:** Writing Files section

**Add PLN example:**
```markdown
// Write PLN (for Parsley data - preserves types)
{name: "Alice", joined: @2024-01-15, balance: $100.00} ==> PLN(@./user.pln)

// Write JSON (for external systems)
{name: "Alice", age: 30} ==> JSON(@./user.json)
```

---

#### 3. **builtins/paths.md**

**Location:** Paths as File Handle Sources

**Change example to prefer PLN for config:**
```markdown
let config <== PLN(@./config.pln)      // Parsley config with dates, money
let data <== JSON(@./api-cache.json)   // JSON from external API
let lines <== lines(@./todo.txt)
"output" ==> text(@./result.txt)
```

---

#### 4. **features/data-formats.md - Common Patterns**

**Location:** Read, Transform, Write

**Add PLN alternative:**
```markdown
// Write as PLN (preserves Parsley types for later use)
summary ==> PLN(@./summary.pln)

// Write as JSON (for external consumption)
summary ==> JSON(@./summary.json)
```

---

### Medium Priority: Clarify Context

#### 5. **builtins/dictionary.md**

**Location:** Relationship to Other Types → JSON section

**Add note:**
```markdown
> **Note:** `.toJSON()` converts Parsley types to JSON-compatible values 
> (dates become strings, money becomes numbers). For lossless serialization 
> of Parsley data, write to PLN instead: `data ==> PLN(@./file.pln)`.
```

---

#### 6. **builtins/record.md**

**Location:** After Data Methods table

**Add note:**
```markdown
> **Tip:** `toJSON()` converts dates to strings and money to numbers. 
> For lossless record serialization, use PLN: `user ==> PLN(@./user.pln)`.
```

---

### Low Priority: Examples with `.json` Extensions

These examples work correctly but using `.pln` would be more idiomatic for local Parsley data. Leave as-is for now—the high-priority additions provide sufficient guidance.

---

## Examples That Should Stay JSON

The following uses are **correct** and should remain unchanged:

### External APIs
- `let users <=/= JSON(@https://api.example.com/users)` ✅
- `payload =/=> JSON(@https://api.example.com/items)` ✅
- All `features/network.md` examples ✅

### HTTP Protocol
- Headers: `Accept: application/json` ✅
- Request bodies: `body: {name: "Alice"}` (auto-serialized as JSON) ✅

### String Methods
- `.parseJSON()` - parsing JSON strings ✅
- `.toJSON()` - encoding to JSON format ✅

### Schema Field Type
- `json` field type for storing JSON blobs in databases ✅

### SFTP Examples
- Reading remote JSON files from external systems ✅

---

## Implementation Checklist

### Prerequisites (Code Changes Required)
- [x] **Implement PLN write support** — FEAT-115 complete
- [x] **Add PLN to write switch** — FEAT-115 complete
- [x] **Native Money serialization** — FEAT-116 complete
- [x] **Fix DateTime/Path/URL serialization** — FEAT-116 complete
- [x] **Test PLN round-trip** — All types verified

### Documentation Updates
- [x] Add "When to Use PLN vs JSON" section to `features/data-formats.md`
- [x] Add PLN notes to `features/file-io.md` (reading and writing sections)
- [x] Update config example in `builtins/paths.md` to use PLN
- [x] Add PLN alternative to "Read, Transform, Write" pattern
- [x] Add notes to `builtins/dictionary.md` JSON section
- [x] Add notes to `builtins/record.md` data methods table
- [x] Update `pln.md` with file I/O examples

---

## Rationale

**Why this matters:**

1. **Data Loss Prevention:** New users might use JSON for config files containing `@2024-01-15` or `$100.00`, then be confused when these become strings/numbers.

2. **Best Practices:** Establishing clear guidance early prevents technical debt (JSON files that should be PLN).

3. **Type Safety:** PLN preserves Parsley's rich type system; JSON flattens it.

**Example of the problem:**

```parsley
// Using JSON (loses types)
let config = {
    launchDate: @2024-06-01,
    budget: $50000.00,
    dataPath: @./data/users.csv
}
config ==> JSON(@./config.json)

// Later...
let loaded <== JSON(@./config.json)
loaded.launchDate                // "2024-06-01" (string, not datetime!)
loaded.budget                    // 50000 (number, not money!)
loaded.dataPath                  // "./data/users.csv" (string, not path!)

// Using PLN (preserves types)
config ==> PLN(@./config.pln)
let loaded <== PLN(@./config.pln)
loaded.launchDate                // @2024-06-01 (datetime ✓)
loaded.budget                    // $50000.00 (money ✓)
loaded.dataPath                  // @./data/users.csv (path ✓)
```

---

## PLN Type Support Summary

| Type | PLN Literal | Round-trips |
|------|-------------|-------------|
| Integer | `42` | ✅ |
| Float | `3.14` | ✅ |
| String | `"hello"` | ✅ |
| Boolean | `true` / `false` | ✅ |
| Null | `null` | ✅ |
| Array | `[1, 2, 3]` | ✅ |
| Dictionary | `{a: 1, b: 2}` | ✅ |
| Money | `USD#19.99` | ✅ |
| Date | `@2024-01-15` | ✅ |
| DateTime | `@2024-01-15T10:30:00` | ✅ |
| Path | `@./config/file.json` | ✅ |
| URL | `@https://example.com/api` | ✅ |
| Record | `@Person({name: "Alice"})` | ✅ |
| Table | `[{...}, {...}]` | ✅ |

---

## Related

- FEAT-115: PLN Write Support
- FEAT-116: PLN Native Money and DateTime Serialization
- PLN specification: `docs/parsley/manual/pln.md`
- Data formats guide: `docs/parsley/manual/features/data-formats.md`
- File I/O guide: `docs/parsley/manual/features/file-io.md`
