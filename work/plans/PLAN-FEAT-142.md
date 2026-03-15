# PLAN-FEAT-142: Meta Component and Page Restructure

## Overview

This plan outlines the implementation steps for FEAT-142, which restructures the `Head` and `Page` components in the Basil prelude. The goal is to replace the `Head` component with a new `Meta` component that outputs only metadata tags, and update the `Page` component to handle Open Graph and Twitter metadata automatically. This change improves the composition model and reduces duplication.

---

## Objectives

1. **Create a new `Meta` component**:
   - Outputs meta/link tags for SEO and social sharing.
   - Removes the `<head>` wrapper to allow clean composition with `Page`.

2. **Update the `Page` component**:
   - Automatically outputs Open Graph and Twitter metadata from `title` and `description` props.
   - Validates that `title` is provided using `fail()`.

3. **Ensure backward compatibility**:
   - Export `Head` as a deprecated alias for `Meta`.

4. **Maintain Parsley correctness**:
   - Follow idiomatic Parsley syntax as defined in FEAT-143.

5. **Update documentation and tests**:
   - Document the new `Meta` component and updated `Page` behavior.
   - Add tests to verify all acceptance criteria.

---

## Implementation Steps

### 1. Create `Meta` Component
- **File**: `server/prelude/components/meta.pars`
- **Responsibilities**:
  - Output meta/link tags for SEO and social sharing.
  - Support props for `image`, `url`, `type`, `author`, `published`, `modified`, `twitter`, `favicon`, `noIndex`, and `contents`.
  - Handle `datetime` objects for `published`/`modified` using `.iso`.
  - Provide sensible defaults for `og:type` (`"website"`) and favicons.
- **Backward Compatibility**:
  - Export `Head` as a deprecated alias for `Meta`.

### 2. Update `Page` Component
- **File**: `server/prelude/components/page.pars`
- **Changes**:
  - Validate that `title` is provided using `fail()`.
  - Output `<meta property="og:title">` and `<meta name="twitter:title">` from `title`.
  - Output `<meta property="og:description">` and `<meta name="twitter:description">` from `description` (if provided).
  - Ensure `head` prop renders after OG/Twitter tags to allow `Meta` composition.

### 3. Remove `Head` Component
- **File**: `server/prelude/components/head.pars`
- **Action**:
  - Delete the `Head` component file.
  - Ensure all references to `Head` are updated to use `Meta`.

### 4. Update Prelude Exports
- **Action**:
  - Add `Meta` to the prelude exports.
  - Keep `Head` as a deprecated alias.

### 5. Add Tests
- **Location**: `pkg/parsley/tests/`
- **Test Cases**:
  1. Verify `Page` outputs `og:title` and `twitter:title` from `title`.
  2. Verify `Page` outputs `og:description` and `twitter:description` from `description`.
  3. Verify `Page` raises `fail()` if `title` is missing.
  4. Verify `Meta` outputs `og:image`, `og:url`, `og:type`, and favicons correctly.
  5. Verify `Meta` handles `published`/`modified` dates using `.iso`.
  6. Verify `Meta` outputs `noindex` robots tag when `noIndex` is true.
  7. Verify `Head` alias works but logs a deprecation warning.

### 6. Update Documentation
- **Files**:
  - `docs/parsley/reference.md`
  - `docs/parsley/README.md`
  - `docs/parsley/CHEATSHEET.md`
- **Changes**:
  - Document the `Meta` component and its props.
  - Update `Page` documentation to include OG/Twitter metadata behavior.
  - Add a migration guide from `Head` to `Meta`.

### 7. Verify Parsley Correctness
- **Command**: `pars --check`
- **Ensure**:
  - All files use `+` for string concatenation.
  - All single-expression conditionals use concise form: `if (cond) expr else expr`.
  - Spread syntax uses `...attrs` (not `{...attrs}`).
  - Required prop validation uses `fail()`.

### 8. Run Benchmarks
- **Command**: `make bench-compare`
- **Ensure**:
  - No performance regressions > 5%.
  - Note any improvements > 5%.

---

## Timeline

| Task                          | Effort  | Dependencies          |
|-------------------------------|---------|-----------------------|
| Create `meta.pars`            | 30 min  | None                  |
| Update `page.pars`            | 30 min  | None                  |
| Delete `head.pars`            | 15 min  | After above           |
| Update prelude exports        | 15 min  | After above           |
| Add tests                     | 45 min  | After implementation  |
| Update documentation          | 1 hour  | After tests pass      |
| Verify Parsley correctness    | 10 min  | After implementation  |
| Run benchmarks                | 10 min  | After implementation  |

**Total Effort**: ~3.5 hours

---

## Risks and Mitigations

1. **Backward Compatibility**:
   - Risk: Breaking existing `Head` usage.
   - Mitigation: Keep `Head` as a deprecated alias for `Meta`.

2. **Developer Confusion**:
   - Risk: Developers may not understand the migration from `Head` to `Meta`.
   - Mitigation: Provide clear documentation and a migration guide.

3. **Performance Regressions**:
   - Risk: Changes to `Page` or `Meta` may introduce regressions.
   - Mitigation: Run benchmarks and optimize as needed.

---

## Deliverables

1. `Meta` component in `server/prelude/components/meta.pars`.
2. Updated `Page` component in `server/prelude/components/page.pars`.
3. Deprecated `Head` alias in `meta.pars`.
4. Updated documentation and migration guide.
5. Comprehensive tests for all acceptance criteria.
6. Benchmark results showing no regressions.

---

## References

- **Spec**: `work/specs/FEAT-142.md`
- **Design**: `work/design/DESIGN-prelude-meta-component.md`
- **Parsley Correctness**: FEAT-143
- **Prelude Review**: `work/reports/STANDARD-PRELUDE-REVIEW.md` §7.4