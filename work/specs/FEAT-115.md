---
id: FEAT-115
title: "PLN Write Support — Feature Parity with JSON"
status: draft
priority: high
created: 2026-02-14
author: "@ai"
related: FEAT-098
blocking: false
---

# FEAT-115: PLN Write Support — Feature Parity with JSON

## Summary
Implement PLN (Parsley Literal Notation) write support to achieve feature parity with JSON. This enables `data ==> PLN(@./file.pln)` syntax for writing Parsley data to files while preserving types that JSON cannot represent (dates, money, paths, durations, records).

## User Story
As a Parsley developer, I want to write configuration files and data caches in PLN format so that I can preserve Parsley types (dates, money, paths, records) without manually converting them to JSON-compatible values and losing type information.

## Current Status

**✅ Reading:** Fully implemented (FEAT-098)
- `PLN()` builtin function creates file handles
- `<== PLN(@./file.pln)` reads and parses PLN files
- Auto-detection works: `.pln` extension recognized
- `parsePLN()` deserializes PLN to Parsley objects

**❌ Writing:** NOT implemented
- No `encodePLN()` function
- Cannot use `==> PLN(@./file.pln)` syntax
- PLN format not supported in write operators

## Acceptance Criteria

### Core Functionality
- [ ] `encodePLN()` function serializes Parsley values to PLN strings
- [ ] `data ==> PLN(@./file.pln)` writes PLN files
- [ ] `data ==>> PLN(@./file.pln)` appends to PLN files
- [ ] Round-trip preservation: `data ==> PLN(f)` then `<== PLN(f)` returns identical value
- [ ] All types that `SerializeToPLN()` supports are writable via file handles

### Error Handling
- [ ] Non-serializable values (functions, handles) produce clear errors
- [ ] File write errors propagate with context (path, format)
- [ ] Permission errors are caught and reported

### Testing
- [ ] Unit tests for `encodePLN()`
- [ ] Round-trip tests for all supported types
- [ ] File write integration tests
- [ ] Error case tests (non-serializable values, write failures)

### Documentation
- [ ] Update `docs/parsley/manual/features/data-formats.md` with PLN write examples
- [ ] Update `docs/parsley/manual/features/file-io.md` with PLN write operators
- [ ] Add "When to Use PLN vs JSON" guidance section
- [ ] Update `docs/parsley/manual/pln.md` with file I/O examples
- [ ] Update examples in `builtins/paths.md` to use PLN for config files
- [ ] Add PLN alternatives to "Read, Transform, Write" patterns

## Design Decisions

### Use Existing Serializer
PLN serialization is already implemented in `pkg/parsley/pln/serializer.go` and exposed via `SerializeToPLN()` in `pln_hooks.go`. The encoder should call this existing function rather than reimplementing serialization logic.

### Error Handling
Non-serializable values should produce `FORMAT-0008` errors (consistent with CSV/JSON encoding errors). The error message should identify which value cannot be serialized and why.

### Pretty-Printing
PLN output should be pretty-printed by default (like JSON) for readability:
- Dictionaries: one key-value pair per line, 2-space indent
- Arrays: break to multiline if >60 chars
- Records: `@Schema({...})` with formatted dict inside

This matches existing PLN serializer behavior.

### Append Mode
Appending to PLN files (`==>>`) should add values on new lines. Each value is a complete PLN expression. This allows multiple values per file (useful for logs, event streams).

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components

| Component | Location | Change Type | Notes |
|-----------|----------|-------------|-------|
| Encoder | `pkg/parsley/evaluator/eval_encoders.go` | New function | Add `encodePLN()` |
| File I/O | `pkg/parsley/evaluator/eval_file_io.go` | Modify switch | Add `case "pln"` to write logic |
| Tests | `pkg/parsley/pln/file_test.go` | New tests | Round-trip and error tests |
| Manual | `docs/parsley/manual/features/data-formats.md` | Content update | PLN write examples |
| Manual | `docs/parsley/manual/features/file-io.md` | Content update | PLN write operators |
| Manual | `docs/parsley/manual/pln.md` | Content update | File I/O section |
| Manual | `docs/parsley/manual/builtins/paths.md` | Example update | Use PLN for config |
| Manual | `docs/parsley/manual/builtins/dictionary.md` | Note addition | PLN vs JSON guidance |
| Manual | `docs/parsley/manual/builtins/record.md` | Note addition | PLN serialization |

### Dependencies
- Depends on: FEAT-098 (PLN serialization)
- Blocks: Manual documentation updates (MANUAL-JSON-PLN-REVIEW)
- Related: FEAT-100 (pretty-printer uses same formatting logic)

### Effort Estimate
- Core implementation: 1-2 hours
- Testing: 1 hour
- Documentation: 2-3 hours
- **Total: 4-6 hours**

---

## Implementation Plan

### Task 1: Implement `encodePLN()`

**File:** `pkg/parsley/evaluator/eval_encoders.go`

**Location:** Add after `encodeYAML()` (around L137)

**Implementation:**
```go
// encodePLN encodes a value as PLN (Parsley Literal Notation)
func encodePLN(value Object, env *Environment) ([]byte, error) {
	// Call the existing PLN serializer
	plnObj := SerializeToPLN(value, env)
	
	// Check if serialization failed
	if err, isErr := plnObj.(*Error); isErr {
		return nil, fmt.Errorf("%s: %s", err.Code, err.Message)
	}
	
	// Extract the PLN string
	plnStr, ok := plnObj.(*String)
	if !ok {
		return nil, fmt.Errorf("PLN serialization returned %s instead of string", plnObj.Type())
	}
	
	return []byte(plnStr.Value), nil
}
```

**Validation:**
- Verify `SerializeToPLN()` is accessible from evaluator package
- Confirm error codes match existing format error patterns
- Test that byte conversion preserves UTF-8 encoding

---

### Task 2: Add PLN to Write Switch

**File:** `pkg/parsley/evaluator/eval_file_io.go`

**Location:** In `writeFileContent()` switch statement (around L535)

**Implementation:**
```go
case "pln":
	data, encodeErr = encodePLN(value, env)
```

**Context:** The switch statement already handles:
- `"text"`, `"bytes"`, `"lines"` — raw formats
- `"json"`, `"yaml"`, `"csv"` — structured formats
- `"svg"` — special format

PLN should be grouped with structured formats (after YAML, before default).

**Validation:**
- Verify `env` is available in scope (needed for schema resolution)
- Confirm error handling flows through `encodeErr` check
- Test both write (`==>`) and append (`==>>`) modes

---

### Task 3: Unit Tests

**File:** `pkg/parsley/pln/file_test.go`

**Add after existing `TestPLNBuiltinFunction`:**

```go
func TestPLNWriteBasic(t *testing.T) {
	tests := []struct{
		name  string
		value string
		want  string
	}{
		{"integer", "42", "42"},
		{"string", `"hello"`, `"hello"`},
		{"array", "[1, 2, 3]", "[1, 2, 3]"},
		{"dict", "{a: 1, b: 2}", "{\n  a: 1,\n  b: 2\n}"},
		{"datetime", "@2024-01-15", "@2024-01-15T00:00:00Z"},
		{"money", "$100.00", "$100.00"},
		{"path", "@./config.pln", "@./config.pln"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write PLN
			code := fmt.Sprintf(`%s ==> PLN(@./test_output.pln)`, tt.value)
			// ... test implementation
		})
	}
}

func TestPLNRoundTrip(t *testing.T) {
	tests := []struct{
		name  string
		value string
	}{
		{"complex_nested", `{
			name: "Alice",
			age: 30,
			joined: @2024-01-15,
			balance: $50000.00,
			dataPath: @./data/users.csv,
			tags: ["admin", "user"],
			meta: {active: true, verified: false}
		}`},
		{"record_simple", `@Person({name: "Bob", age: 25})`},
		{"mixed_types", `[
			42,
			"text",
			@2024-01-15,
			$100.00,
			{nested: true}
		]`},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write then read
			writeCode := fmt.Sprintf(`let data = %s\ndata ==> PLN(@./roundtrip.pln)`, tt.value)
			readCode := `let loaded <== PLN(@./roundtrip.pln)\nloaded`
			
			// Verify loaded == original
			// ... test implementation
		})
	}
}

func TestPLNWriteErrors(t *testing.T) {
	tests := []struct{
		name     string
		value    string
		wantCode string
	}{
		{"function", "fn(x) { x + 1 }", "SERIALIZE-0001"},
		{"file_handle", "JSON(@./file.json)", "SERIALIZE-0001"},
		{"db_connection", "@sqlite(\":memory:\")", "SERIALIZE-0001"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := fmt.Sprintf(`%s ==> PLN(@./error.pln)`, tt.value)
			// Verify error code matches expected
			// ... test implementation
		})
	}
}

func TestPLNAppendMode(t *testing.T) {
	// Test that ==>> appends values on new lines
	// ... test implementation
}
```

**Coverage targets:**
- All basic types (integers, strings, arrays, dicts)
- Parsley-specific types (dates, money, paths, durations)
- Records with schemas
- Nested structures
- Error cases (functions, handles)
- Append mode (`==>>`)

---

### Task 4: Integration Tests

**File:** `pkg/parsley/tests/file_io_test.go` (or create if not exists)

Add integration tests that exercise the full file I/O pipeline:

```go
func TestPLNFileWriteRead(t *testing.T) {
	// Create temp file
	// Write complex Parsley value
	// Read back
	// Verify equality
}

func TestPLNAutoDetect(t *testing.T) {
	// Test that .pln extension is auto-detected
	// file(@./data.pln) should use PLN format
}
```

---

### Task 5: Documentation Updates

#### 5.1 Data Formats Guide

**File:** `docs/parsley/manual/features/data-formats.md`

**Add after JSON section (around L230):**

```markdown
### When to Use PLN vs JSON

**Use PLN for:**
- Configuration files that include dates, money, or other Parsley types
- Caching Parsley data structures
- Data files read/written by Parsley scripts
- Serializing records with schemas
- Any internal Parsley data storage

**Use JSON for:**
- API requests and responses
- Data exchange with non-Parsley systems
- When compatibility with JSON parsers is required
- External integrations

**Key differences:**

PLN round-trips all Parsley types losslessly:
```parsley
// Using PLN (preserves types)
let config = {
    launchDate: @2024-06-01,
    budget: $50000.00,
    dataPath: @./data/users.csv
}
config ==> PLN(@./config.pln)

// Later...
let loaded <== PLN(@./config.pln)
loaded.launchDate                // @2024-06-01 (datetime ✓)
loaded.budget                    // $50000.00 (money ✓)
loaded.dataPath                  // @./data/users.csv (path ✓)
```

JSON converts Parsley types to primitives:
```parsley
// Using JSON (loses types)
config ==> JSON(@./config.json)

let loaded <== JSON(@./config.json)
loaded.launchDate                // "2024-06-01" (string ✗)
loaded.budget                    // 50000 (number ✗)
loaded.dataPath                  // "./data/users.csv" (string ✗)
```

**Format comparison:**

| Feature | PLN | JSON |
|---------|-----|------|
| Datetimes | `@2024-01-15T10:30:00Z` | `"2024-01-15T10:30:00Z"` (string) |
| Money | `$100.00`, `EUR#50.00` | `100`, `50` (number) |
| Paths | `@./config.pln` | `"./config.pln"` (string) |
| Durations | `@2h30m` | `"2h30m"` (string) |
| Records | `@Person({...})` | `{...}` (plain object) |
| Comments | `// supported` | Not allowed |
```

---

#### 5.2 File I/O Guide

**File:** `docs/parsley/manual/features/file-io.md`

**Update Reading section (around L85-90):**

```markdown
### PLN

PLN (Parsley Literal Notation) preserves all Parsley types:

```parsley
let config <== PLN(@./config.pln)
config.launchDate                // datetime object
config.budget                    // money object
```

**Note:** For configuration files with Parsley-specific types (dates, money, paths), 
PLN is preferred over JSON to avoid type loss.
See [PLN](../pln.md) for details.
```

**Update Writing section (around L150-158):**

```markdown
// Write JSON (for external systems)
{name: "Alice", age: 30} ==> JSON(@./user.json)

// Write PLN (for Parsley data with dates, money, etc.)
{
    name: "Alice", 
    joined: @2024-01-15, 
    balance: $100.00
} ==> PLN(@./user.pln)

// Write plain text
"Hello, world!" ==> text(@./greeting.txt)

// Append to a log
"New entry\n" ==>> text(@./app.log)
```

**Update file handles table (around L55-65):**

| Function | Format | Read type | Description |
|---|---|---|---|
| `JSON(path)` | JSON | dictionary/array | JSON file (external data) |
| `YAML(path)` | YAML | dictionary/array | YAML file |
| `CSV(path)` | CSV | table | CSV file (returns a table) |
| `PLN(path)` | PLN | any | Parsley Literal Notation (preserves all types) |
| `text(path)` | Plain text | string | Raw text content |
| ... | | |
```

---

#### 5.3 PLN Manual Page

**File:** `docs/parsley/manual/pln.md`

**Add File I/O section (before "See Also"):**

```markdown
## File I/O

PLN files can be read and written using the `PLN()` file handle:

### Reading PLN Files

```parsley
let config <== PLN(@./config.pln)
```

The `.pln` extension is auto-detected:
```parsley
let data <== file(@./data.pln)     // Uses PLN format automatically
```

### Writing PLN Files

```parsley
// Write data
{
    name: "Alice",
    joined: @2024-01-15,
    balance: $50000.00,
    dataPath: @./data/users.csv
} ==> PLN(@./config.pln)

// Append to log
{
    event: "login",
    timestamp: @now,
    user: "alice"
} ==>> PLN(@./events.pln)
```

### Round-Trip Example

PLN preserves all Parsley types:

```parsley
// Original data
let original = {
    launchDate: @2024-06-01,
    budget: $50000.00,
    dataPath: @./data/users.csv,
    duration: @2h30m
}

// Write and read back
original ==> PLN(@./config.pln)
let loaded <== PLN(@./config.pln)

// All types preserved
loaded.launchDate.type()         // "datetime"
loaded.budget.type()             // "money"
loaded.dataPath.type()           // "path"
loaded.duration.type()           // "duration"
```
```

---

#### 5.4 Paths Builtin

**File:** `docs/parsley/manual/builtins/paths.md`

**Update "Paths as File Handle Sources" example (around L147-150):**

```markdown
let config <== PLN(@./config.pln)      // Parsley config with dates, money
let data <== JSON(@./api-cache.json)   // JSON from external API
let lines <== lines(@./todo.txt)
"output" ==> text(@./result.txt)
```

---

#### 5.5 Dictionary Builtin

**File:** `docs/parsley/manual/builtins/dictionary.md`

**Update JSON section (around L736-737):**

```markdown
### JSON

Dictionaries map directly to JSON objects. Use `.toJSON()` to serialize 
and `@fetch` responses automatically parse JSON into dictionaries.

**Note:** `.toJSON()` converts Parsley types to JSON-compatible values 
(dates become strings, money becomes numbers). For lossless serialization 
of Parsley data, use PLN instead:

```parsley
// JSON loses type information
{name: "Alice", joined: @2024-01-15}.toJSON()
// → {"name":"Alice","joined":"2024-01-15"}

// PLN preserves types
{name: "Alice", joined: @2024-01-15} ==> PLN(@./user.pln)
let loaded <== PLN(@./user.pln)
loaded.joined.type()             // "datetime" ✓
```

See [PLN](../pln.md) for details.
```

---

#### 5.6 Record Builtin

**File:** `docs/parsley/manual/builtins/record.md`

**Add note to data methods table (after L1078):**

```markdown
**Note:** `toJSON()` serializes data fields as JSON (dates become strings, 
money becomes numbers). For lossless record serialization, use PLN:

```parsley
// JSON loses schema and types
user.toJSON()   // → {"name":"Alice","joined":"2024-01-15"}

// PLN preserves schema and types
user ==> PLN(@./user.pln)
let loaded <== PLN(@./user.pln)
loaded.schema().name             // "User" ✓
loaded.joined.type()             // "datetime" ✓
```
```

---

## Validation Checklist

### Code
- [ ] `encodePLN()` function compiles without errors
- [ ] `case "pln"` added to write switch in `eval_file_io.go`
- [ ] Unit tests pass: `TestPLNWriteBasic`, `TestPLNRoundTrip`, `TestPLNWriteErrors`
- [ ] Integration tests pass: file write/read cycle works
- [ ] Error handling produces clear messages for non-serializable values
- [ ] Append mode works: `==>> PLN(@./file.pln)` adds values on new lines

### Documentation
- [ ] "When to Use PLN vs JSON" section added to data-formats.md
- [ ] PLN write examples added to file-io.md
- [ ] File I/O section added to pln.md
- [ ] Config example in paths.md updated to use PLN
- [ ] Note added to dictionary.md JSON section
- [ ] Note added to record.md data methods table
- [ ] All code examples tested and verified

### Manual Testing
- [ ] Write simple values: `42 ==> PLN(@./test.pln)`
- [ ] Write complex dict with dates/money: `{...} ==> PLN(@./config.pln)`
- [ ] Read back and verify types preserved
- [ ] Test append mode: `value ==>> PLN(@./log.pln)` multiple times
- [ ] Verify `.pln` auto-detection: `data <== file(@./data.pln)`
- [ ] Test error: `fn(x) { x } ==> PLN(@./error.pln)` produces clear error

---

## Test Plan

### Unit Tests

| Test | Expected Behavior |
|------|-------------------|
| Write integer | `42 ==> PLN(f)` writes `"42"` |
| Write string | `"hello" ==> PLN(f)` writes `"\"hello\""` |
| Write array | `[1,2,3] ==> PLN(f)` writes `"[1, 2, 3]"` |
| Write dict | `{a:1} ==> PLN(f)` writes pretty-printed dict |
| Write datetime | `@2024-01-15 ==> PLN(f)` writes `"@2024-01-15T00:00:00Z"` |
| Write money | `$100 ==> PLN(f)` writes `"$100.00"` |
| Write path | `@./config ==> PLN(f)` writes `"@./config"` |
| Write record | `@Person({...}) ==> PLN(f)` writes `"@Person({...})"` |
| Write function (error) | `fn(x){x} ==> PLN(f)` returns `SERIALIZE-0001` error |
| Append mode | `1 ==>> PLN(f); 2 ==>> PLN(f)` writes `"1\n2"` |

### Integration Tests

| Test | Expected Behavior |
|------|-------------------|
| Round-trip simple | `42 ==> PLN(f); <== PLN(f)` returns `42` |
| Round-trip complex | Complex dict with all types preserves exactly |
| Auto-detect `.pln` | `file(@./data.pln)` uses PLN format |
| Write permission error | Writing to restricted path produces `IO-0004` error |
| Invalid PLN in file | Reading malformed PLN produces `FMT-0009` error |

---

## Error Codes

### New Errors
None — reuse existing error codes:
- `SERIALIZE-0001` — value cannot be serialized (from PLN serializer)
- `FILEOP-0006` — encoding failed (generic file operation error)
- `IO-0004` — file write failed (I/O error)

### Error Messages
When `encodePLN()` fails:
```
Error serializing value to PLN: functions cannot be serialized
Code: SERIALIZE-0001
```

When file write fails:
```
Failed to write file: permission denied
Code: IO-0004
Path: @./restricted/file.pln
```

---

## Related

- FEAT-098 — PLN specification and serialization
- FEAT-100 — Pretty-printer (uses same formatting rules)
- MANUAL-JSON-PLN-REVIEW — Manual review report that identified this gap
- `docs/parsley/manual/pln.md` — PLN specification
- `pkg/parsley/pln/serializer.go` — Serialization implementation
- `pkg/parsley/evaluator/pln_hooks.go` — Evaluator integration

---

## Post-Implementation

After completing this feature:
1. Update `work/reports/MANUAL-JSON-PLN-REVIEW.md` status to "implemented"
2. Mark all checklist items in the review report as complete
3. Test all documentation examples manually
4. Consider adding PLN support to SFTP operations (future enhancement)
5. Consider adding PLN support to network operations (future, if needed)