---
id: PLAN-046
feature: FEAT-072
title: "Implementation Plan for Natural Sort Order"
status: in-progress
created: 2025-12-17
---

# Implementation Plan: FEAT-072 Natural Sort Order

## Overview
Implement efficient Natural Sort Order for Parsley arrays and enable string comparison operators. The implementation uses an ASCII fast path with Unicode fallback for optimal performance.

## Prerequisites
- [x] Specification approved (FEAT-072)
- [x] Existing `naturalStringCompare` function identified (in evaluator.go)
- [x] Current `compareObjects` function located (in methods.go, line 1041)

## Tasks

### Task 1: Implement Efficient NaturalCompare Function
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium

Steps:
1. Add `isASCII(s string) bool` helper function
2. Implement `naturalCompareASCII(a, b string) int` - byte-based, zero-allocation
3. Implement `naturalCompareUnicode(a, b string) int` - rune-based for international text
4. Implement `NaturalCompare(a, b string) int` - dispatcher with ASCII fast path
5. Add `compareRight` and `compareLeft` helpers for digit run comparison
6. Remove old `naturalStringCompare` and `extractNumber` functions

Tests:
- Basic: `"file1"` < `"file2"` < `"file10"`
- Versions: `"v1.5"` < `"v2.0"` < `"v10.0"`
- Leading zeros: `"007"` vs `"7"` (left-aligned comparison)
- Unicode digits: `"file١٠"` vs `"file٢"` (Arabic numerals)
- Mixed: `"abc123def456"` vs `"abc123def45"`
- Edge: empty strings, single chars, numbers only

---

### Task 2: Update compareObjects to Use NaturalCompare
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Steps:
1. Replace `strings.Compare(av.Value, bv.Value)` with `NaturalCompare(av.Value, bv.Value)` in `compareObjects` function (around line 1091)
2. Verify all callers of `compareObjects` benefit: `sort()`, `sortBy()`, table sorting

Tests:
- `[3, 1, 2].sort()` → `[1, 2, 3]` (numbers unchanged)
- `["file10", "file2", "file1"].sort()` → `["file1", "file2", "file10"]`
- `["v2.0", "v10.0", "v1.5"].sort()` → `["v1.5", "v2.0", "v10.0"]`

---

### Task 3: Add Options Parameter to sort() Method
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Medium

Steps:
1. Modify `case "sort":` to accept optional dictionary argument
2. Parse `{natural: false}` option to use lexicographic sort
3. Add `compareObjectsLexicographic` function using `strings.Compare`
4. Default to natural sort when no options provided

Tests:
- `["b", "a", "c"].sort()` → `["a", "b", "c"]` (natural, same result)
- `["file10", "file2"].sort()` → `["file2", "file10"]` (natural)
- `["file10", "file2"].sort({natural: false})` → `["file10", "file2"]` (lexicographic)

---

### Task 4: Enable String Comparison Operators
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Locate `evalStringInfixExpression` function
2. Add cases for `<`, `>`, `<=`, `>=` operators
3. Use `NaturalCompare` for all comparisons
4. Return appropriate boolean results

Tests:
- `"file2" < "file10"` → `true`
- `"file10" > "file2"` → `true`
- `"file10" >= "file10"` → `true`
- `"file10" <= "file2"` → `false`
- `"abc" < "abd"` → `true`
- `"" < "a"` → `true`

---

### Task 5: Handle Mixed-Type Array Sorting
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Medium

Steps:
1. Update `compareObjects` to handle cross-type comparisons
2. Define type ordering: `null < numbers < strings < booleans < dates < durations < money < arrays < dicts`
3. Add `typeOrder(obj Object) int` helper function
4. When types differ, compare by type order

Tests:
- `[null, 3, 1, "b", "a", 2].sort()` → `[null, 1, 2, 3, "a", "b"]`
- `[true, false, 1, "x"].sort()` → `[1, "x", false, true]`
- `["a", 1, null].sort()` → `[null, 1, "a"]`

---

### Task 6: Update sortBy() for Consistency  
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Steps:
1. Verify `sortBy()` uses `compareObjects` for key comparison
2. If not, update to use `compareObjects` (which now uses NaturalCompare)
3. Consider adding `{natural: false}` option for consistency

Tests:
- `[{n: "v10"}, {n: "v2"}].sortBy(fn(x) { x.n })` → `[{n: "v2"}, {n: "v10"}]`
- `["file10", "file2"].sortBy(fn(x) { x })` → `["file2", "file10"]`

---

### Task 7: Add Comprehensive Test Suite
**Files**: `pkg/parsley/tests/sort_test.go` (new), `pkg/parsley/tests/operators_test.go`
**Estimated effort**: Medium

Steps:
1. Create `sort_test.go` with natural sort test cases
2. Add string comparison operator tests to `operators_test.go`
3. Include edge cases: empty strings, very long numbers, Unicode
4. Add performance benchmark tests

Tests:
- All acceptance criteria from FEAT-072
- Regression tests for existing sort behavior on numbers
- Unicode digit tests (Arabic, Devanagari numerals)

---

### Task 8: Update Documentation
**Files**: `docs/manual/builtins/array.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small

Steps:
1. Update `sort()` documentation in array.md to mention natural sort
2. Add `{natural: false}` option documentation
3. Add string comparison operators section or update existing
4. Update CHEATSHEET.md with natural sort notes

Tests:
- Documentation review (manual)

---

## Validation Checklist
- [x] All tests pass: `go test ./...`
- [x] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [x] Array sort tests pass with natural order
- [x] String comparison operators work
- [x] Mixed-type sorting works
- [x] `sort({natural: false})` provides escape hatch
- [ ] Documentation updated
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-17 | Task 1: NaturalCompare | ✅ Complete | ASCII fast path + Unicode fallback implemented |
| 2025-12-17 | Task 2: Update compareObjects | ✅ Complete | Now uses NaturalCompare for strings |
| 2025-12-17 | Task 3: sort() options | ✅ Complete | `{natural: false}` option added |
| 2025-12-17 | Task 4: String operators | ✅ Complete | `<`, `>`, `<=`, `>=` now work on strings |
| 2025-12-17 | Task 5: Mixed-type sorting | ✅ Complete | Type ordering: null < numbers < strings < booleans < dates < durations < money < arrays < dicts |
| 2025-12-17 | Task 6: sortBy() update | ✅ Complete | Already uses compareObjects, inherited fix |
| 2025-12-17 | Task 7: Test suite | ✅ Complete | Created pkg/parsley/tests/sort_test.go with 32 test cases |
| | Task 8: Documentation | ⬜ Not Started | — |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- `sort({ignoreCase: true})` — Case-insensitive natural sort option
- `compare(a, b)` builtin function — Expose three-way comparison to users
- Locale-aware collation — Full ICU collation support for international sorting
