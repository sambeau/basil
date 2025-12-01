---
id: FEAT-008
title: "Array Randomization Methods (pick, take, shuffle)"
status: draft
priority: medium
created: 2025-12-01
author: "@human"
---

# FEAT-008: Array Randomization Methods (pick, take, shuffle)

## Summary
Add `pick()`, `take()`, and `shuffle()` methods to Parsley arrays for random element selection and array randomization. These are common operations needed for games, sampling, randomized displays, and other use cases requiring randomness.

## User Story
As a Parsley developer, I want to randomly select elements from arrays and shuffle arrays so that I can build games, randomized UIs, and sampling applications without writing my own randomization logic.

## Acceptance Criteria
- [ ] `array.pick()` returns a single random element from the array
- [ ] `array.pick(n)` returns an array of n random elements (duplicates allowed, can exceed length)
- [ ] `array.take(n)` returns an array of n unique random elements (no duplicates)
- [ ] `array.take(n)` errors if n > array length (can't take more unique items than exist)
- [ ] `array.shuffle()` returns a new array with elements in random order
- [ ] Original array is not modified by any method
- [ ] Empty array handling: `[].pick()` returns null, `[].take(0)` and `[].shuffle()` return `[]`
- [ ] `shuffle()` uses Fisher-Yates algorithm for proper randomization
- [ ] Tests cover all methods and edge cases
- [ ] Documentation updated

## Design Decisions
- **Return new arrays, don't mutate**: Consistent with functional style, avoids surprises
- **`pick()` returns value, `pick(n)` returns array**: Natural API - asking for one thing gets one thing
- **`pick(n)` allows duplicates and exceeding length**: Random sampling with replacement
- **`take(n)` enforces uniqueness**: Random sampling without replacement, errors if impossible
- **Fisher-Yates for shuffle**: O(n), unbiased, industry standard algorithm
- **`pick()` on empty array returns null**: Consistent with other Parsley "not found" patterns

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/evaluator/methods.go` — Add pick(), take(), and shuffle() method implementations
- `pkg/parsley/tests/` — Add test file for randomization methods
- `docs/parsley/CHEATSHEET.md` — Document new methods
- `docs/parsley/reference.md` — Add to array methods reference

### Dependencies
- Depends on: Nothing
- Blocks: Nothing

### Edge Cases & Constraints

#### pick()
1. **Empty array**: `[].pick()` → null
2. **Single element**: Returns that element
3. **`pick(0)`**: Return empty array
4. **`pick(n)` any n**: Always valid, returns array with possible duplicates
5. **`pick(n)` where n < 0**: Return error
6. **Non-integer argument**: Return error

#### take()
1. **Empty array**: `[].take(0)` → `[]`, `[].take(n)` where n > 0 → error
2. **`take(0)`**: Return empty array
3. **`take(n)` where n == length**: Return shuffled copy of entire array
4. **`take(n)` where n > length**: Return error
5. **`take(n)` where n < 0**: Return error
6. **Non-integer argument**: Return error

#### shuffle()
1. **Empty array**: Return empty array
2. **Single element**: Return array with that element

### Algorithm Reference
Fisher-Yates shuffle (modern version):
```
for i from n−1 down to 1 do
    j := random integer such that 0 ≤ j ≤ i
    swap a[i] and a[j]
```
See: https://en.wikipedia.org/wiki/Fisher–Yates_shuffle

## Implementation Notes
*Added during/after implementation*

## Related
- Plan: `docs/plans/FEAT-008-plan.md`
