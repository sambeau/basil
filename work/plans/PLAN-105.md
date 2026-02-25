---
id: PLAN-105
feature: FEAT-126
title: "Implementation Plan for Test Coverage Gap Remediation"
status: complete
created: 2025-02-26
---

# Implementation Plan: FEAT-126

## Overview
Add tests for language features identified as having zero coverage in the 1.0 Readiness Audit: `with` expression, US derived area units, and area division operations.

## Prerequisites
- [x] Features are already implemented and working
- [x] Test infrastructure exists in `pkg/parsley/tests/`

## Tasks

### Task 1: Add `with` Expression Tests
**Files**: `pkg/parsley/tests/with_test.go` (new file)
**Estimated effort**: Small

Steps:
1. Create new test file `with_test.go`
2. Add test helper using existing `testEval` pattern
3. Implement test cases for all scenarios

Tests:
- Basic dictionary unpacking: `with {a: 1, b: 2} { a + b }` → `3`
- Record unpacking: `with @record { field1 + field2 }`
- Nested with expressions
- Computed values in dictionary: `with {a: 1, b: a + 1} { b }` → `2`
- Error case: non-dict/record target → TYPE-0020 error
- Edge case: empty dictionary → body evaluates with no new bindings
- Edge case: keys with invalid identifier names are skipped

---

### Task 2: Add US Area Unit Multiplication Tests
**Files**: `pkg/parsley/tests/unit_test.go` (extend existing)
**Estimated effort**: Small

Steps:
1. Add new test function `TestUSAreaFromMultiplication`
2. Test ft × ft, yd × yd, in × in combinations
3. Verify display hints are preserved

Tests:
- `#3ft * #4ft` → `#12ft2`
- `#2yd * #3yd` → `#6yd2`
- `#6in * #6in` → `#36in2`
- `#1ft * #12in` → area in appropriate unit
- Verify `.format()` on results

---

### Task 3: Add Area Division Tests
**Files**: `pkg/parsley/tests/unit_phase3_test.go` (extend existing)
**Estimated effort**: Small

Steps:
1. Add new test function `TestAreaDivisionSI`
2. Add new test function `TestAreaDivisionUS`
3. Add error case tests

Tests:
- SI division: `#12m2 / #3m` → `#4m`
- SI division: `#100cm2 / #10cm` → `#10cm`
- SI division: `#1m2 / #50cm` → appropriate result
- US division: `#12ft2 / #3ft` → `#4ft`
- US division: `#36in2 / #6in` → `#6in`
- Error: incompatible systems `#10m2 / #5ft` → error
- Error: division by zero-length unit

---

### Task 4: Verify and Document
**Files**: `work/reports/1.0-READINESS-AUDIT.md`
**Estimated effort**: Small

Steps:
1. Run full test suite to verify no regressions
2. Run coverage report to confirm gaps are filled
3. Update audit report to mark items complete

Tests:
- `go test ./pkg/parsley/...` passes
- Coverage for target functions > 0%

---

## Validation Checklist
- [x] All tests pass: `go test ./pkg/parsley/...`
- [x] `with` expression has > 0% coverage (88.9%)
- [x] `multiplyLengthToAreaUS` has > 0% coverage (76.2%)
- [x] `divideAreaByLength` functions have > 0% coverage (73.7%-100%)
- [x] Linter passes: `golangci-lint run`
- [x] Audit report updated

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-02-26 | Task 1 | ✅ Complete | Added `with_test.go` with 25 test cases |
| 2025-02-26 | Task 2 | ✅ Complete | Added US area multiplication tests (ft×ft, yd×yd, in×in, mi×mi) |
| 2025-02-26 | Task 3 | ✅ Complete | Added SI and US area division tests |
| 2025-02-26 | Task 4 | ✅ Complete | Verified coverage, updated audit report |

## Deferred Items
Items to add to work/BACKLOG.md after implementation:
- None anticipated — this is a test-only change