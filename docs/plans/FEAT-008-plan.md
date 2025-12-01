# Implementation Plan: FEAT-008

**Feature**: Array Randomization Methods (pick, take, shuffle)
**Spec**: `docs/specs/FEAT-008.md`
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
- [ ] `go build ./cmd/basil && go build ./cmd/pars` succeeds
- [ ] `go test ./...` passes
- [ ] All edge cases tested
- [ ] Documentation updated

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| | Task 1 | ⬜ Not started | shuffle() |
| | Task 2 | ⬜ Not started | pick() |
| | Task 3 | ⬜ Not started | take() |
| | Task 4 | ⬜ Not started | Tests |
| | Task 5 | ⬜ Not started | Docs |
