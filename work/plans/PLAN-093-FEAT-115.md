---
id: PLAN-093
feature: FEAT-115
title: "Implementation Plan for PLN Write Support"
status: draft
created: 2026-02-14
---

# Implementation Plan: PLN Write Support

## Overview
Implement PLN (Parsley Literal Notation) write support to achieve feature parity with JSON. This enables `data ==> PLN(@./file.pln)` syntax for writing Parsley data while preserving types that JSON cannot represent (dates, money, paths, durations, records).

**Based on:** FEAT-115 specification and MANUAL-JSON-PLN-REVIEW report

## Prerequisites
- [x] PLN serialization implemented (FEAT-098)
- [x] `SerializeToPLN()` function exists in `pln_hooks.go`
- [x] PLN reading works (`<== PLN(@./file.pln)`)
- [x] PLN parser handles all supported types
- [ ] Development environment set up for testing

## Tasks

### Task 1: Implement `encodePLN()` Function
**Location**: `pkg/parsley/evaluator/eval_encoders.go`
**Estimated effort**: Small (30 min)

Steps:
1. Add function after `encodeYAML()` (around line 137)
2. Call existing `SerializeToPLN()` from `pln_hooks.go`
3. Handle error cases (non-serializable values)
4. Return byte slice for file writing

Implementation:
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

Tests:
- Function compiles without errors
- Imports are correct (`SerializeToPLN` is accessible)
- Error handling preserves error codes
- UTF-8 encoding is preserved

---

### Task 2: Add PLN to Write Switch
**Location**: `pkg/parsley/evaluator/eval_file_io.go`
**Estimated effort**: Small (15 min)

Steps:
1. Locate `writeFileContent()` function
2. Find the format switch statement (around line 514)
3. Add PLN case after YAML, before default
4. Ensure `env` parameter is passed to `encodePLN()`

Implementation:
```go
case "pln":
	data, encodeErr = encodePLN(value, env)
```

Tests:
- Code compiles
- `env` is in scope at this location
- Error handling flows through existing `encodeErr` check
- Both write (`==>`) and append (`==>>`) modes work

---

### Task 3: Unit Tests - Basic Types
**Location**: `pkg/parsley/pln/file_test.go`
**Estimated effort**: Medium (1 hour)

Steps:
1. Create `TestPLNWriteBasic()` function
2. Test all basic types: integer, float, string, boolean, null
3. Test Parsley-specific types: datetime, money, path, duration
4. Test collections: array, dictionary
5. Verify output format matches expected PLN

Test cases:
- Integer: `42` → `"42"`
- String: `"hello"` → `"\"hello\""`
- Array: `[1, 2, 3]` → `"[1, 2, 3]"`
- Dict: `{a: 1, b: 2}` → pretty-printed with newlines
- Datetime: `@2024-01-15` → `"@2024-01-15T00:00:00Z"`
- Money: `$100.00` → `"$100.00"`
- Path: `@./config.pln` → `"@./config.pln"`

Tests:
- All basic types serialize correctly
- Pretty-printing works for dicts/arrays
- Parsley types preserve their format

---

### Task 4: Unit Tests - Round-Trip
**Location**: `pkg/parsley/pln/file_test.go`
**Estimated effort**: Medium (45 min)

Steps:
1. Create `TestPLNRoundTrip()` function
2. Test complex nested structures
3. Test records with schemas
4. Test mixed-type arrays
5. Verify write → read returns identical value

Test cases:
- Complex nested dict with dates, money, paths
- Record: `@Person({name: "Bob", age: 25})`
- Mixed array: `[42, "text", @2024-01-15, $100.00]`
- All combinations preserve types exactly

Tests:
- Round-trip equality: `original == loaded`
- Type preservation: `loaded.field.type() == original.field.type()`
- Nested structures maintain structure

---

### Task 5: Unit Tests - Error Cases
**Location**: `pkg/parsley/pln/file_test.go`
**Estimated effort**: Small (30 min)

Steps:
1. Create `TestPLNWriteErrors()` function
2. Test non-serializable types: functions, file handles, connections
3. Verify error codes match expected values
4. Verify error messages are clear

Test cases:
- Function: `fn(x) { x + 1 }` → `SERIALIZE-0001`
- File handle: `JSON(@./file.json)` → `SERIALIZE-0001`
- DB connection: `@sqlite(":memory:")` → `SERIALIZE-0001`

Tests:
- Correct error codes returned
- Error messages identify the problem
- No crashes or panics

---

### Task 6: Unit Tests - Append Mode
**Location**: `pkg/parsley/pln/file_test.go`
**Estimated effort**: Small (20 min)

Steps:
1. Create `TestPLNAppendMode()` function
2. Test `==>>` operator appends values
3. Verify each value is on a new line
4. Test multiple appends to same file

Test cases:
- Single append adds newline
- Multiple appends create multi-line file
- Each line is valid PLN

Tests:
- Append mode works correctly
- Values are separated by newlines
- File can be read back (each line parsed separately)

---

### Task 7: Integration Tests
**Location**: `pkg/parsley/tests/file_io_test.go` (create if needed)
**Estimated effort**: Medium (45 min)

Steps:
1. Create `TestPLNFileWriteRead()` integration test
2. Test full file I/O pipeline with temp files
3. Create `TestPLNAutoDetect()` for `.pln` extension
4. Test error cases (permissions, invalid paths)

Test cases:
- Write complex value, read back, verify equality
- Auto-detect: `file(@./data.pln)` uses PLN format
- Write permission error produces `IO-0004`
- Reading malformed PLN produces format error

Tests:
- End-to-end file operations work
- Auto-detection based on extension works
- Error handling is correct

---

### Task 8: Documentation - Data Formats Guide
**Location**: `docs/parsley/manual/features/data-formats.md`
**Estimated effort**: Medium (45 min)

Steps:
1. Add "When to Use PLN vs JSON" section after JSON section (~L230)
2. Include comparison table (PLN vs JSON)
3. Show code examples demonstrating type loss in JSON
4. Explain when to use each format

Content:
- "Use PLN for" list (internal data, configs, caches)
- "Use JSON for" list (APIs, external systems)
- Side-by-side example showing type preservation
- Format comparison table

Tests:
- All code examples are syntactically correct
- Examples can be copy-pasted and run
- Table formatting is correct

---

### Task 9: Documentation - File I/O Guide
**Location**: `docs/parsley/manual/features/file-io.md`
**Estimated effort**: Medium (30 min)

Steps:
1. Add PLN reading section with note about type preservation
2. Update writing examples to include PLN
3. Update file handles table to include PLN
4. Add note comparing PLN vs JSON for config files

Content:
- PLN reading example with datetime/money
- PLN writing example alongside JSON/text
- Updated file handles table with PLN row
- Note recommending PLN for Parsley-specific types

Tests:
- Examples are correct
- Table formatting matches existing style
- Links to pln.md work

---

### Task 10: Documentation - PLN Manual Page
**Location**: `docs/parsley/manual/pln.md`
**Estimated effort**: Medium (30 min)

Steps:
1. Add "File I/O" section before "See Also"
2. Include reading, writing, and round-trip examples
3. Show auto-detection based on `.pln` extension
4. Demonstrate type preservation

Content:
- Reading PLN files subsection
- Writing PLN files subsection  
- Append mode example
- Complete round-trip example with all types

Tests:
- Examples demonstrate key features
- Code is syntactically correct
- Formatting is consistent

---

### Task 11: Documentation - Paths Builtin
**Location**: `docs/parsley/manual/builtins/paths.md`
**Estimated effort**: Small (10 min)

Steps:
1. Locate "Paths as File Handle Sources" example (~L147)
2. Change first example from JSON to PLN
3. Add comment explaining choice

Content:
```parsley
let config <== PLN(@./config.pln)      // Parsley config with dates, money
let data <== JSON(@./api-cache.json)   // JSON from external API
```

Tests:
- Example is correct
- Comment clarifies the distinction

---

### Task 12: Documentation - Dictionary Builtin
**Location**: `docs/parsley/manual/builtins/dictionary.md`
**Estimated effort**: Small (15 min)

Steps:
1. Locate JSON section (~L736)
2. Add note about type loss in JSON vs PLN
3. Include short example

Content:
- Note explaining JSON converts Parsley types
- Example showing date → string conversion in JSON
- Recommendation to use PLN for preservation
- Link to pln.md

Tests:
- Note is clear and concise
- Example demonstrates the issue
- Link works

---

### Task 13: Documentation - Record Builtin
**Location**: `docs/parsley/manual/builtins/record.md`
**Estimated effort**: Small (15 min)

Steps:
1. Locate data methods table note (~L1078)
2. Add note about `toJSON()` vs PLN serialization
3. Include example showing difference

Content:
- Note explaining `toJSON()` type loss
- Example: JSON vs PLN for record serialization
- Show schema preservation with PLN

Tests:
- Note fits existing documentation style
- Example is correct
- Recommendation is clear

---

### Task 14: Manual Testing
**Location**: N/A (manual verification)
**Estimated effort**: Small (30 min)

Steps:
1. Write simple value to PLN file
2. Write complex dict with dates/money/paths
3. Read back and verify types preserved
4. Test append mode with multiple values
5. Test auto-detection for `.pln` extension
6. Attempt to write function (verify error)

Test checklist:
- [ ] `42 ==> PLN(@./test.pln)` works
- [ ] Complex config with dates/money writes correctly
- [ ] Read back preserves all types
- [ ] `value ==>> PLN(@./log.pln)` appends on new line
- [ ] `file(@./data.pln)` auto-detects PLN format
- [ ] `fn(x){x} ==> PLN(@./err.pln)` produces clear error

Tests:
- All manual tests pass
- No unexpected errors
- Type preservation verified

---

## Validation Checklist

### Code
- [ ] `encodePLN()` compiles without errors
- [ ] `case "pln"` added to write switch
- [ ] All unit tests pass (basic, round-trip, errors, append)
- [ ] Integration tests pass
- [ ] No compiler warnings
- [ ] Error codes are consistent

### Tests
- [ ] `TestPLNWriteBasic` - 7+ test cases
- [ ] `TestPLNRoundTrip` - 3+ test cases
- [ ] `TestPLNWriteErrors` - 3+ test cases
- [ ] `TestPLNAppendMode` - append functionality verified
- [ ] `TestPLNFileWriteRead` - integration test passes
- [ ] `TestPLNAutoDetect` - extension detection works

### Documentation
- [ ] "When to Use PLN vs JSON" section in data-formats.md
- [ ] PLN examples in file-io.md (read + write)
- [ ] File I/O section in pln.md
- [ ] Config example updated in paths.md
- [ ] Note added to dictionary.md
- [ ] Note added to record.md
- [ ] All code examples tested

### Manual Verification
- [ ] Write/read simple values
- [ ] Write/read complex structures with all types
- [ ] Append mode creates multi-line files
- [ ] Auto-detection works for `.pln` files
- [ ] Error handling for non-serializable values
- [ ] Round-trip preserves types exactly

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| — | Plan created | ✅ Complete | — |
| — | Task 1: encodePLN() | ⏸️ Pending | — |
| — | Task 2: Write switch | ⏸️ Pending | — |
| — | Task 3: Basic tests | ⏸️ Pending | — |
| — | Task 4: Round-trip tests | ⏸️ Pending | — |
| — | Task 5: Error tests | ⏸️ Pending | — |
| — | Task 6: Append tests | ⏸️ Pending | — |
| — | Task 7: Integration tests | ⏸️ Pending | — |
| — | Task 8: data-formats.md | ⏸️ Pending | — |
| — | Task 9: file-io.md | ⏸️ Pending | — |
| — | Task 10: pln.md | ⏸️ Pending | — |
| — | Task 11: paths.md | ⏸️ Pending | — |
| — | Task 12: dictionary.md | ⏸️ Pending | — |
| — | Task 13: record.md | ⏸️ Pending | — |
| — | Task 14: Manual testing | ⏸️ Pending | — |

## Deferred Items
None anticipated. All functionality is straightforward.

## Implementation Notes

### Testing Strategy
- Unit tests first (Tasks 3-6) to verify core functionality
- Integration tests (Task 7) to verify file I/O pipeline
- Manual tests (Task 14) for final verification
- All tests should pass before documentation updates

### Documentation Strategy
- Update docs after code is working and tested
- Test all code examples in documentation
- Link related pages together (cross-references)
- Maintain consistency with existing manual style

### Code Organization
- `encodePLN()` in eval_encoders.go with other encoders
- PLN case in write switch alongside JSON/YAML/CSV
- Tests in pln/file_test.go with existing PLN tests
- Integration tests in tests/file_io_test.go

### Dependencies
- Requires `SerializeToPLN()` from pln_hooks.go
- Requires access to `Environment` for schema resolution
- Follows same pattern as JSON/YAML/CSV encoders

## Success Criteria

Implementation is successful when:
1. `data ==> PLN(@./file.pln)` works for all supported types
2. Round-trip preserves types: write then read returns identical value
3. Append mode works: `==>>` adds values on new lines
4. Auto-detection works: `.pln` files use PLN format automatically
5. Error handling is clear for non-serializable values
6. All tests pass
7. Documentation is complete with working examples
8. Manual testing confirms all functionality

## Timeline Estimate

| Phase | Tasks | Time | Notes |
|-------|-------|------|-------|
| Core Implementation | 1-2 | 45 min | encodePLN() + switch case |
| Testing | 3-7 | 3.5 hours | Unit + integration tests |
| Documentation | 8-13 | 2.5 hours | 6 manual pages |
| Manual Verification | 14 | 30 min | Final testing |
| **Total** | **14 tasks** | **~5 hours** | Can be split across sessions |

## Post-Implementation

After completing this feature:
1. Update FEAT-115 status to "implemented"
2. Update MANUAL-JSON-PLN-REVIEW report status
3. Mark all review checklist items complete
4. Consider adding PLN examples to more manual pages
5. Update CHANGELOG.md with PLN write support
6. Consider SFTP/network PLN support (future enhancement)