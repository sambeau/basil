---
id: FEAT-072
title: "Natural Sort Order"
status: draft
priority: high
created: 2025-12-17
author: "@sam"
---

# FEAT-072: Natural Sort Order

## Summary
Implement Natural Sort Order throughout Parsley for human-friendly sorting and string comparison. Natural sort treats embedded numbers numerically, so "file2" sorts before "file10" instead of after. This affects array `sort()`, `sortBy()`, and enables string comparison operators (`<`, `>`, `<=`, `>=`) which currently don't work on strings.

## User Story
As a Parsley developer, I want strings to sort in natural order so that version numbers, filenames, and numbered items appear in the order humans expect.

## Acceptance Criteria
- [ ] `["file10", "file2", "file1"].sort()` returns `["file1", "file2", "file10"]`
- [ ] `["v2.0", "v10.0", "v1.5"].sort()` returns `["v1.5", "v2.0", "v10.0"]`
- [ ] `"file2" < "file10"` returns `true`
- [ ] `"file10" > "file2"` returns `true`
- [ ] `sort({natural: false})` provides lexicographic sort as escape hatch
- [ ] Mixed-type arrays sort correctly: nulls < numbers < strings < booleans < dates
- [ ] Unicode digits (Arabic ٠١٢, etc.) are handled correctly
- [ ] Performance: ASCII strings use fast path (no allocations)
- [ ] Existing `sortBy()` uses natural comparison for string keys

## Design Decisions

### 1. Natural Sort by Default
**Decision:** `sort()` uses natural sort by default; `sort({natural: false})` for lexicographic.
**Rationale:** Natural sort is almost always what users want. Breaking change is acceptable as current string sort behavior is rarely relied upon for lexicographic ordering.

### 2. Enable String Comparison Operators
**Decision:** Enable `<`, `>`, `<=`, `>=` on strings using natural comparison.
**Rationale:** These operators currently return errors on strings, so enabling them is not a breaking change. Natural comparison makes them useful for version checking, etc.

### 3. ASCII Fast Path with Unicode Fallback
**Decision:** Use byte-based comparison for ASCII strings, fall back to rune-based for Unicode.
**Rationale:** 95%+ of strings are ASCII. Byte comparison is 3-5x faster than rune conversion. Unicode fallback ensures correctness for international text.

### 4. Mixed-Type Sort Order
**Decision:** Sort order by type: `null < numbers < strings < booleans < dates < durations < money < arrays < dicts < other`
**Rationale:** Nulls first is standard. Numbers before strings allows numeric-string mixed arrays to sort sensibly. Other types follow logical grouping.

### 5. Three-Way Compare Function
**Decision:** Internal `NaturalCompare(a, b string) int` returns -1/0/1.
**Rationale:** More efficient than bool - allows single comparison for sort and operators. Matches Go's `strings.Compare` signature.

### 6. Case Sensitivity
**Decision:** Case-sensitive by default. Future: add `sort({ignoreCase: true})`.
**Rationale:** Matches current behavior. Case-insensitive can be added later without breaking changes.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/evaluator/evaluator.go` — Replace `naturalStringCompare` with efficient `NaturalCompare`
- `pkg/parsley/evaluator/methods.go` — Update `compareObjects` to use `NaturalCompare`; add options to `sort()`
- `pkg/parsley/evaluator/infix.go` or `evaluator.go` — Enable `<`, `>`, `<=`, `>=` for strings
- `pkg/parsley/tests/methods_test.go` — Add natural sort tests
- `pkg/parsley/tests/operators_test.go` — Add string comparison tests
- `docs/manual/builtins/array.md` — Update sort documentation

### Algorithm: Efficient Natural Compare

Based on strnatcmp.c with ASCII fast path:

```go
// NaturalCompare compares two strings using natural sort order.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func NaturalCompare(a, b string) int {
    // Fast path: check if both strings are ASCII
    if isASCII(a) && isASCII(b) {
        return naturalCompareASCII(a, b)
    }
    // Fallback: Unicode-aware comparison
    return naturalCompareUnicode(a, b)
}

func isASCII(s string) bool {
    for i := 0; i < len(s); i++ {
        if s[i] >= 128 {
            return false
        }
    }
    return true
}
```

**ASCII path:** Works on bytes, inline digit check `'0' <= c && c <= '9'`, uses "bias" technique for equal-length number runs.

**Unicode path:** Converts to runes, uses `unicode.IsDigit()` for full digit detection.

### Edge Cases & Constraints

1. **Leading zeros:** `"007"` vs `"7"` — Compare left-aligned (first different digit wins)
2. **Very long numbers:** Numbers exceeding int64 — Compare digit-by-digit without parsing
3. **Empty strings:** `""` sorts before any non-empty string
4. **Same-prefix different length:** `"file"` < `"file1"` (shorter first)
5. **Mixed currency sort:** Error - cannot compare different currencies
6. **Duration with months:** Error - cannot compare durations containing months

### Dependencies
- Depends on: None
- Blocks: None

## Implementation Notes
*Added during/after implementation*

## Related
- Plan: `docs/plans/FEAT-072-plan.md`
- Wikipedia: https://en.wikipedia.org/wiki/Natural_sort_order
- Reference: https://sourcefrog.net/projects/natsort/
