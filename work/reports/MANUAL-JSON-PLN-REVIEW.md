---
id: MANUAL-JSON-PLN-REVIEW
title: "Manual Review: JSON vs PLN Guidance"
date: 2026-02-14
status: post-alpha
---

# Manual Review: JSON vs PLN Guidance

## Summary

The Parsley manual currently uses JSON extensively in examples, but many cases involving **local Parsley data** would benefit from suggesting PLN (Parsley Literal Notation) instead. PLN preserves Parsley types (dates, money, paths, durations) that JSON cannot represent, making it the better choice for internal data storage.

**Key principle:** Use JSON for external interoperability (APIs, other systems), use PLN for internal Parsley data (configs, caches, data files).

## Current PLN Implementation Status

**✅ Reading:** Fully implemented
- `PLN()` builtin function exists in `evaluator.go` (L2712-2744)
- `parsePLN()` function reads PLN files via `<==` operator
- Auto-detection works: `.pln` extension recognized in `inferFormatFromExtension()`

**❌ Writing:** NOT implemented
- No `encodePLN()` function in `eval_encoders.go`
- PLN format not in write switch statement in `writeFileContent()` (eval_file_io.go L514-536)
- Cannot use `data ==> PLN(@./file.pln)` syntax

**Impact:** All examples suggesting PLN for writing will NOT work until PLN encoding is implemented. The review recommendations below assume PLN write support will be added.

---

## Recommended Changes

### High Priority: Add PLN Guidance

These sections should explicitly recommend PLN for local data:

#### 1. **features/data-formats.md**

**Current:** Section introduces JSON alongside CSV and Markdown with no clear guidance on when to use each.

**Add after JSON section (around L230):**
```markdown
### When to Use PLN vs JSON

**Use PLN for:**
- Configuration files that include dates, money, or other Parsley types
- Caching Parsley data structures
- Data files read/written by Parsley scripts
- Serializing records with schemas

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

**Location:** Reading Files section (around L85-90)

**Current:**
```markdown
### JSON

let config <== JSON(@./config.json)
config.database.host             // "localhost"
```

**Suggestion:** Add a note:
```markdown
### JSON

let config <== JSON(@./config.json)
config.database.host             // "localhost"

**Note:** For configuration files with Parsley-specific types (dates, money, paths), 
consider using PLN instead: `let config <== PLN(@./config.pln)`
See [PLN](../pln.md) for details.
```

---

**Location:** Writing Files section (around L150-158)

**Current:**
```markdown
// Write JSON
{name: "Alice", age: 30} ==> JSON(@./user.json)
```

**Add alternative:**
```markdown
// Write JSON (for external systems)
{name: "Alice", age: 30} ==> JSON(@./user.json)

// Write PLN (for Parsley data with dates, money, etc.)
{name: "Alice", joined: @2024-01-15, balance: $100.00} ==> PLN(@./user.pln)
```

---

#### 3. **builtins/paths.md**

**Location:** Paths as File Handle Sources (around L147-150)

**Current:**
```markdown
let data <== JSON(@./config.json)
let lines <== lines(@./todo.txt)
"output" ==> text(@./result.txt)
```

**Suggestion:** Change first example to PLN for config:
```markdown
let config <== PLN(@./config.pln)      // Parsley config with dates, money
let data <== JSON(@./api-cache.json)   // JSON from external API
let lines <== lines(@./todo.txt)
"output" ==> text(@./result.txt)
```

---

#### 4. **features/data-formats.md - Common Patterns**

**Location:** Read, Transform, Write (around L245-251)

**Current:**
```markdown
// Read CSV, transform, write JSON
let sales <== CSV(@./sales.csv)
let summary = for (row in sales) {
    {name: row.product, total: row.price * row.quantity}
}
summary.toJSON() ==> text(@./summary.json)
```

**Add PLN alternative:**
```markdown
// Read CSV, transform, write JSON (for external consumption)
let sales <== CSV(@./sales.csv)
let summary = for (row in sales) {
    {name: row.product, total: row.price * row.quantity}
}
summary.toJSON() ==> text(@./summary.json)

// Or write as PLN (preserves money types if sales data includes them)
summary ==> PLN(@./summary.pln)
```

---

### Medium Priority: Clarify Context

These sections are fine but could benefit from brief notes about when JSON is appropriate:

#### 5. **builtins/dictionary.md**

**Location:** Relationship to Other Types → JSON (L734-737)

**Current:**
```markdown
### JSON

Dictionaries map directly to JSON objects. Use `.toJSON()` to serialize 
and `@fetch` responses automatically parse JSON into dictionaries:
```

**Add note:**
```markdown
### JSON

Dictionaries map directly to JSON objects. Use `.toJSON()` to serialize 
and `@fetch` responses automatically parse JSON into dictionaries.

**Note:** `.toJSON()` converts Parsley types to JSON-compatible values 
(dates become strings, money becomes numbers). For lossless serialization 
of Parsley data, use PLN instead. See [PLN](../pln.md).
```

---

#### 6. **builtins/record.md**

**Location:** Data Methods table (L1072-1078)

**Current table includes:** `toJSON()` method

**Add to table note:**
```markdown
**Note:** `toJSON()` serializes data fields as JSON (dates become strings, 
money becomes numbers). For lossless record serialization, use PLN: 
`user ==> PLN(@./user.pln)` preserves all Parsley types.
```

---

### Low Priority: Examples with `.json` Extensions

These examples work correctly but using `.pln` would be more idiomatic for local Parsley data:

**Files to review:**
- `builtins/paths.md` - Multiple examples use `.json` for local files
- `features/security.md` - Example uses `@./secrets/keys.json` (could be `.pln`)

**Recommendation:** Leave as-is for now (JSON is familiar), but consider updating in a future consistency pass. The high-priority additions above provide sufficient guidance.

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
- [ ] **Implement PLN write support** — add `encodePLN()` to `eval_encoders.go`
- [ ] **Add PLN to write switch** — update `writeFileContent()` in `eval_file_io.go`
- [ ] **Test PLN round-trip** — verify `data ==> PLN(@./f.pln)` then `<== PLN(@./f.pln)` preserves all types

### Documentation Updates (Post-PLN-Write)
- [ ] Add "When to Use PLN vs JSON" section to `features/data-formats.md`
- [ ] Add PLN notes to `features/file-io.md` (reading and writing)
- [ ] Update config example in `builtins/paths.md` to use PLN
- [ ] Add PLN alternative to "Read, Transform, Write" pattern
- [ ] Add notes to `builtins/dictionary.md` JSON section
- [ ] Add notes to `builtins/record.md` data methods table

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

## Technical Implementation Notes

### Files to Modify for PLN Write Support

**1. `pkg/parsley/evaluator/eval_encoders.go`**
Add after `encodeYAML()`:
```go
// encodePLN encodes a value as PLN (Parsley Literal Notation)
func encodePLN(value Object, env *Environment) ([]byte, error) {
    plnObj := SerializeToPLN(value, env)
    if err, isErr := plnObj.(*Error); isErr {
        return nil, fmt.Errorf("%s: %s", err.Code, err.Message)
    }
    plnStr, ok := plnObj.(*String)
    if !ok {
        return nil, fmt.Errorf("PLN serialization returned %s instead of string", plnObj.Type())
    }
    return []byte(plnStr.Value), nil
}
```

**2. `pkg/parsley/evaluator/eval_file_io.go`**
Add case to `writeFileContent()` switch (around L535):
```go
case "pln":
    data, encodeErr = encodePLN(value, env)
```

**3. Testing**
Add to `pkg/parsley/pln/file_test.go`:
```go
func TestPLNWriteRoundTrip(t *testing.T) {
    // Test that writing and reading PLN preserves all types
}
```

---

## Related

- PLN specification: `docs/parsley/manual/pln.md`
- PLN implementation: `pkg/parsley/pln/pln.go`
- PLN hooks: `pkg/parsley/evaluator/pln_hooks.go`
- Data formats guide: `docs/parsley/manual/features/data-formats.md`
- File I/O guide: `docs/parsley/manual/features/file-io.md`
