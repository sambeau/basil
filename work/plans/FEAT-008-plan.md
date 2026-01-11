# Implementation Plan: FEAT-008

**Feature**: Array Randomization Methods (pick, take, shuffle)
**Spec**: `work/specs/FEAT-008.md`
**Plan ID**: PLAN-006

## Overview
Add three array methods for random selection and shuffling.

## Tasks

### Task 1: Implement shuffle() method
**File**: `pkg/parsley/evaluator/methods.go`
**Effort**: Small

Implement Fisher-Yates shuffle returning a new array.

### Task 2: Implement pick() method
**File**: `pkg/parsley/evaluator/methods.go`
**Effort**: Small

- `pick()` - return random element (null if empty)
- `pick(n)` - return array of n random elements (with replacement)

### Task 3: Implement take() method
**File**: `pkg/parsley/evaluator/methods.go`
**Effort**: Small

- `take(n)` - return array of n unique random elements
- Error if n > array length

### Task 4: Add tests
**File**: `pkg/parsley/tests/array_random_test.go`
**Effort**: Medium

Test all methods and edge cases.

### Task 5: Update documentation
**Files**: `docs/parsley/CHEATSHEET.md`, `docs/parsley/reference.md`
**Effort**: Small

Document new methods with examples.

## Validation Checklist
- [x] `go build ./cmd/basil && go build ./cmd/pars` succeeds
- [x] `go test ./...` passes
- [x] All edge cases tested
- [x] Documentation updated

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-01 | Task 1 | ✅ Complete | shuffle() with Fisher-Yates |
| 2025-12-01 | Task 2 | ✅ Complete | pick() and pick(n) |
| 2025-12-01 | Task 3 | ✅ Complete | take(n) |
| 2025-12-01 | Task 4 | ✅ Complete | 27 test cases |
| 2025-12-01 | Task 5 | ✅ Complete | CHEATSHEET.md and reference.md |
