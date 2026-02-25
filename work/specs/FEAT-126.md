---
id: FEAT-126
title: "Test Coverage Gap Remediation"
status: draft
priority: high
created: 2025-02-26
author: "@ai"
---

# FEAT-126: Test Coverage Gap Remediation

## Summary
Add missing tests for features identified in the 1.0 Readiness Audit test coverage analysis. Several implemented features lack test coverage, including the `with` expression, US derived area units, and area division operations.

## User Story
As a maintainer, I want all language features to have test coverage so that regressions are caught automatically and the codebase remains stable for the 1.0 release.

## Acceptance Criteria
- [ ] `with` expression has tests covering basic usage, error cases, and edge cases
- [ ] US derived area units (ft², yd²) from length multiplication have tests
- [ ] Area division operations (`#10m2 / #2m` → length) have tests
- [ ] All new tests pass
- [ ] No regressions in existing tests

## Design Decisions
- **Integration tests over unit tests**: Parsley uses integration tests in `pkg/parsley/tests/` that exercise the full parse→eval pipeline, which is more valuable for a language implementation
- **Focus on gaps, not coverage percentage**: We're adding tests for specific untested features, not chasing a coverage number

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/tests/with_test.go` — New test file for `with` expression
- `pkg/parsley/tests/unit_test.go` — Add US area unit tests
- `pkg/parsley/tests/unit_phase3_test.go` — Add area division tests

### Dependencies
- Depends on: None (features already implemented)
- Blocks: 1.0 release readiness

### Features to Test

#### 1. `with` Expression (0% coverage)
The `with` expression unpacks dictionary/record fields into local scope:
```parsley
with {a: 1, b: 2} { a + b }  // → 3
with user { name + " (" + email + ")" }
```

Implementation: `pkg/parsley/evaluator/eval_control_flow.go:386`

Test cases needed:
- Basic dictionary unpacking
- Record unpacking
- Nested `with` expressions
- Error: non-dict/record target
- Edge: empty dictionary
- Edge: keys that aren't valid identifiers (should be skipped)

#### 2. US Derived Area Units (0% coverage)
Length × Length should produce area for US units:
```parsley
#3ft * #4ft  // → #12ft2
#2yd * #3yd  // → #6yd2
```

Implementation: `pkg/parsley/evaluator/eval_unit_infix.go:119` (`multiplyLengthToAreaUS`)

Test cases needed:
- ft × ft → ft²
- yd × yd → yd²
- in × in → in² (if supported)
- Mixed US units (ft × in)

#### 3. Area Division (0% coverage)
Area ÷ Length should produce length:
```parsley
#12m2 / #3m   // → #4m
#100ft2 / #10ft  // → #10ft
```

Implementation: `pkg/parsley/evaluator/eval_unit_infix.go:225` (`divideAreaByLength`)

Test cases needed:
- SI: m² / m → m
- SI: cm² / cm → cm
- US: ft² / ft → ft
- Error: incompatible units (m² / ft)
- Error: division by zero length

## Implementation Notes
*Added during/after implementation*

## Related
- Plan: `work/plans/PLAN-105.md`
- Audit: `work/reports/1.0-READINESS-AUDIT.md` (Test Coverage Analysis section)