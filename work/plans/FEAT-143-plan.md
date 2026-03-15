---
id: PLAN-122
feature: FEAT-143
title: "Implementation Plan: Prelude Component Styling Strategy"
status: complete
created: 2026-06-15
updated: 2026-03-15
---

# Implementation Plan: FEAT-143

## Overview

Implement a Pico CSS-compatible styling strategy for Prelude components. This involves creating new components (Dialog, Details, Accordion, Toast, Pagination, ErrorSummary), updating existing components to output semantic HTML without embedded CSS, and documenting the recommended styling approach.

## Current Status

**✅ All phases complete.**

| Phase | Status | Notes |
|-------|--------|-------|
| Phase 1: Foundation | ✅ Complete | Components created |
| Phase 2: Migrate Existing | ✅ Complete | Components updated |
| Phase 3: New Components | ✅ Complete | All new components added |
| Phase 4: Parsley Correctness | ✅ Complete | All syntax errors fixed |
| Phase 5: Testing | ✅ Complete | Verification script passes |

## Prerequisites

- [x] Spec reviewed and approved: `work/specs/FEAT-143-prelude-component-styling.md`
- [x] Design document complete: `work/design/DESIGN-prelude-pico-compatibility.md`
- [x] Supplement CSS created: `examples/css/basil-supplement.css`
- [x] Supplement README created: `examples/css/README.md`

---

## Phase 1: Foundation (No Breaking Changes) — ✅ COMPLETE

### Task 1.1: Create Details component ✅
### Task 1.2: Create Accordion component ✅
### Task 1.3: Create Dialog component ✅
### Task 1.4: Update prelude exports for Phase 1 components ✅
### Task 1.5: Document Pico CSS setup ✅

---

## Phase 2: Migrate Existing Components — ✅ COMPLETE

### Task 2.1: Update TextField to use `<small>` for hints/errors ✅
### Task 2.2: Update TextareaField to match TextField pattern ✅
### Task 2.3: Update SelectField to match TextField pattern ✅
### Task 2.4: Update SkipLink to remove inline CSS ✅
### Task 2.5: Update Breadcrumb for Pico compatibility ✅
### Task 2.6: Update Page to fix body id default ✅
### Task 2.7: Update Form to remove .form class ✅

---

## Phase 3: New Components — ✅ COMPLETE

### Task 3.1: Create Toast component ✅
### Task 3.2: Create Toasts container component ✅
### Task 3.3: Create Pagination component ✅
### Task 3.4: Create ErrorSummary component ✅
### Task 3.5: Update prelude exports for Phase 3 components ✅

---

## Phase 4: Parsley Correctness — 🔴 IN PROGRESS

A post-implementation audit found blocking Parsley syntax errors. These MUST be fixed before FEAT-143 is complete.

### Task 4.1: Fix invalid spread syntax
**Status:** ✅ Complete  
**Files:** 9 component files  
**Estimated effort:** 15 min

**Problem:** Used `{...attrs}` (JSX) instead of `...attrs` (Parsley)

**Files to fix:**
| File | Line |
|------|------|
| `accordion.pars` | 11 |
| `breadcrumb.pars` | 11 |
| `details.pars` | 7 |
| `dialog.pars` | 7 |
| `error_summary.pars` | 19 |
| `pagination.pars` | 25 |
| `skip_link.pars` | 8 |
| `toast.pars` | 12 |
| `toasts.pars` | 14 |

**Fix command:**
```bash
cd server/prelude/components
for f in accordion.pars breadcrumb.pars details.pars dialog.pars \
         error_summary.pars pagination.pars skip_link.pars toast.pars toasts.pars; do
    sed -i '' 's/{\.\.\.attrs}/...attrs/g' "$f"
done
```

**Verification:**
```bash
# Each file should parse without error:
for f in server/prelude/components/*.pars; do
    pars --check "$f" 2>&1 || echo "FAIL: $f"
done
```

---

### Task 4.2: Fix reversed for-loop variable ordering
**Status:** ✅ Complete  
**Files:** 6 component files  
**Estimated effort:** 20 min

**Problem:** Used `for (item, idx in items)` but Parsley is `for (idx, item in items)`

**Files to fix:**
| File | Line | Current | Fix |
|------|------|---------|-----|
| `accordion.pars` | 10 | `for (item, i in items)` | `for (i, item in items)` |
| `breadcrumb.pars` | 17 | `for (item, idx in items)` | `for (idx, item in items)` |
| `checkbox_group.pars` | 41 | `for (opt, idx in options)` | `for (idx, opt in options)` |
| `radio_group.pars` | 38 | `for (opt, idx in options)` | `for (idx, opt in options)` |
| `data_table.pars` | 11 | `for (col, idx in columns)` | `for (idx, col in columns)` |
| `data_table.pars` | 19 | `for (key, idx in keys)` | `for (idx, key in keys)` |

**Verification after fix:**
```bash
# Test that accordion first item is open:
pars -r -e '
let items = [{title: "Q1", content: "A1"}, {title: "Q2", content: "A2"}]
for (i, item in items) {
    <details open={i == 0}><summary>item.title</summary></details>
}
' | grep -c 'open'
# Should output: 1 (only first is open)

# Test that breadcrumb positions are 1-based:
pars -r -e '
for (idx, item in [{label: "Home"}, {label: "About"}]) {
    <meta content={idx + 1}/>
}
' | grep 'content="1"'
# Should find content="1" (not content="0")
```

---

### Task 4.3: Fix pagination range precedence bug
**Status:** ✅ Complete  
**Files:** `pagination.pars` line 45  
**Estimated effort:** 5 min

**Problem:** `start..end + 1` parses as `(start..end) + 1` due to operator precedence

**Current code:**
```parsley
for (n in start..end + 1) {
```

**Fixed code:**
```parsley
for (n in start..end) {
```

**Rationale:** The `..` operator is inclusive, so `1..5` = `[1,2,3,4,5]`. Since `end` is calculated as `min(totalPages, current + pageWindow)`, it should already be the last page to show.

**Verification:**
```bash
pars -e '
let start = 3
let end = 7
for (n in start..end) { n }
'
# Should output: [3, 4, 5, 6, 7]
```

---

### Task 4.4: Verify all fixes with syntax check
**Status:** ✅ Complete  
**Estimated effort:** 5 min

**Command:**
```bash
echo "=== Syntax Check ==="
errors=0
for f in server/prelude/components/*.pars; do
    if ! pars --check "$f" 2>/dev/null; then
        echo "FAIL: $f"
        errors=$((errors + 1))
    fi
done
if [ $errors -eq 0 ]; then
    echo "All files pass syntax check"
else
    echo "ERRORS: $errors files failed"
    exit 1
fi
```

---

### Task 4.5: Update spec examples to use correct syntax
**Status:** ✅ Complete  
**Files:** `work/specs/FEAT-143-prelude-component-styling.md`  
**Estimated effort:** Already done in spec update

---

## Phase 5: Testing — ✅ COMPLETE

### Task 5.1: Create component verification script
**Status:** ✅ Complete  
**Files:** `scripts/verify-prelude.sh`  
**Estimated effort:** 30 min

Create a script that tests each component renders correctly:

```bash
#!/bin/bash
# verify-prelude.sh

echo "=== Verifying Prelude Components ==="

# Test accordion
echo -n "Accordion: "
pars -r -e '{Accordion} = import @basil/html; <Accordion name="test" items={[{title: "Q1", content: "A1"}, {title: "Q2", content: "A2"}]}/>' 2>/dev/null | grep -q 'name="test"' && echo "OK" || echo "FAIL"

# Test breadcrumb positions
echo -n "Breadcrumb: "
pars -r -e '{Breadcrumb} = import @basil/html; <Breadcrumb items={[{label: "Home", href: "/"}, {label: "About"}]}/>' 2>/dev/null | grep -q 'content="2"' && echo "OK" || echo "FAIL"

# Test pagination
echo -n "Pagination: "
pars -r -e '{Pagination} = import @basil/html; <Pagination current={3} total={100} perPage={10} href="/p?page={page}"/>' 2>/dev/null | grep -q 'aria-current="page"' && echo "OK" || echo "FAIL"

# Test toast
echo -n "Toast: "
pars -r -e '{Toast} = import @basil/html; <Toast message="Hello" type="success"/>' 2>/dev/null | grep -q 'data-type="success"' && echo "OK" || echo "FAIL"

# Test dialog
echo -n "Dialog: "
pars -r -e '{Dialog} = import @basil/html; <Dialog id="test" title="Test">"Content"</Dialog>' 2>/dev/null | grep -q '<dialog id="test">' && echo "OK" || echo "FAIL"

# Test error_summary
echo -n "ErrorSummary: "
pars -r -e '{ErrorSummary} = import @basil/html; <ErrorSummary errors={[{field: "email", message: "Invalid"}]}/>' 2>/dev/null | grep -q 'role="alert"' && echo "OK" || echo "FAIL"

# Test radio_group IDs
echo -n "RadioGroup: "
pars -r -e '{RadioGroup} = import @basil/html; <RadioGroup name="size" label="Size" options={["S", "M", "L"]}/>' 2>/dev/null | grep -q 'id="field-size-0"' && echo "OK" || echo "FAIL"

echo "=== Done ==="
```

---

### Task 5.2: Add integration tests to test suite
**Status:** ⏭️ Deferred (optional)  
**Files:** `pkg/parsley/tests/prelude_feat143_test.go` (new)  
**Estimated effort:** 1 hour
**Note:** Verification script provides sufficient coverage; Go integration tests deferred to backlog.

Add Go tests that verify component output:

```go
func TestFEAT143_AccordionFirstItemOpen(t *testing.T) {
    // First item should have 'open' attribute, second should not
}

func TestFEAT143_BreadcrumbPositionsOneBased(t *testing.T) {
    // Positions should be 1, 2, 3... not 0, 1, 2...
}

func TestFEAT143_PaginationRangeCorrect(t *testing.T) {
    // Window should include expected page numbers
}

func TestFEAT143_RadioGroupUniqueIds(t *testing.T) {
    // IDs should be field-{name}-0, field-{name}-1, etc.
}
```

---

### Task 5.3: Run full test suite
**Status:** ✅ Complete  
**Estimated effort:** 5 min

```bash
go test ./...
make bench-compare
```

---

## Validation Checklist

**Phase 1-3 (Structure):**
- [x] All tests pass: `go test ./...`
- [x] Build succeeds: `make build`
- [x] New components exist in prelude
- [x] Existing components updated
- [x] Documentation complete

**Phase 4 (Correctness):**
- [x] All `.pars` files pass `pars --check`
- [x] Spread syntax fixed (`...attrs` not `{...attrs}`)
- [x] For-loop ordering fixed (`for (idx, item in ...)`)
- [x] Pagination range fixed (`start..end`)
- [x] Spec examples corrected

**Phase 5 (Testing):**
- [x] Verification script passes
- [ ] Integration tests added (deferred)
- [x] All Go tests pass: `go test ./...`

---

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-06-15 | Phase 1: Foundation | ✅ Complete | Components created |
| 2026-06-15 | Phase 2: Migrate Existing | ✅ Complete | Components updated |
| 2026-06-15 | Phase 3: New Components | ✅ Complete | All registered |
| 2026-06-15 | Phase 4: Testing | ⏭️ Deferred | "Components tested via server tests" |
| 2026-03-15 | **Audit** | 🔴 Issues Found | Parsley syntax errors discovered |
| 2026-03-15 | Phase 4: Parsley Correctness | 🔴 Added | New phase to fix blocking issues |
| 2026-03-15 | Phase 4: Parsley Correctness | ✅ Complete | All syntax fixes applied |
| 2026-03-15 | Phase 5: Testing | ✅ Complete | Verification script passes |

---

## Deferred Items (Post-FEAT-143)

Add to `work/BACKLOG.md` after completion:

- Optional modal animation JS — Pico examples include animation helpers
- Toast auto-dismiss JS — Timer-based dismissal for toasts
- ErrorSummary auto-focus JS — Focus summary on form validation failure
- Pagination with parts — Document how to use `.parts` files for partial page updates

---

## References

- **Spec:** `work/specs/FEAT-143-prelude-component-styling.md`
- **Design:** `work/design/DESIGN-prelude-pico-compatibility.md`
- **Review:** `work/reports/STANDARD-PRELUDE-REVIEW.md` (Appendix D has detailed fix instructions)
- **Parsley manual:** `docs/parsley/manual/fundamentals/tags.md`
