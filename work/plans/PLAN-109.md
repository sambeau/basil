# PLAN-109: Implementation Plan for FEAT-130 (Pre-v1.0 Code Quality Fixes)

## Overview

This plan details the implementation steps for FEAT-130, addressing security vulnerabilities, dead code removal, deprecated API replacement, and code modernization before the v1.0 release.

## Branch

`feat/FEAT-130-pre-v1-code-quality`

## Prerequisites

- [ ] User has Go 1.25.7+ available (or will install)
- [ ] All current tests pass on main branch
- [ ] Working tree is clean

---

## Phase 1: Security Vulnerabilities (CRITICAL)

### Step 1.1: Update Go Version

**Owner:** Human (requires system-level Go installation)

**Actions:**
1. Install Go 1.25.7+
2. Update `go.mod` to require new version:
   ```
   go 1.25.7
   ```
3. Run `go mod tidy`

**Verification:**
```bash
go version  # Should show 1.25.7+
govulncheck ./...  # Should show no stdlib vulnerabilities
```

**Commit:** `chore: upgrade to Go 1.25.7 for security fixes`

### Step 1.2: Update Chi Dependency

**Owner:** AI

**Actions:**
1. Update chi to v5.2.2+:
   ```bash
   go get github.com/go-chi/chi/v5@v5.2.2
   go mod tidy
   ```
2. Verify no breaking changes

**Verification:**
```bash
go build ./...
go test ./...
govulncheck ./...  # Should show no chi vulnerability
```

**Commit:** `chore(deps): update chi to v5.2.2 for security fix`

---

## Phase 2: Dead Code Removal (HIGH)

### Step 2.1: Remove Unit Conversion Dead Code

**Owner:** AI

**File:** `pkg/parsley/evaluator/unit_tables.go`

**Actions:**
1. Remove `ConvertUSToSI` function (lines ~306-319)
2. Remove `ConvertSIToUS` function (lines ~322-335)

**Verification:**
```bash
go build ./...
go test ./pkg/parsley/...
```

**Commit:** `refactor(evaluator): remove unused ConvertUSToSI/ConvertSIToUS`

### Step 2.2: Remove Format Package Dead Code

**Owner:** AI

**Files:** 
- `pkg/parsley/format/format.go`
- `pkg/parsley/format/printer.go`

**Actions:**
1. Remove `FormatValue` function (format.go:23-42)
2. Remove `FormatInspectable` function (format.go:385-394)
3. Remove `Printer.Reset` method (printer.go:25-27)

**Verification:**
```bash
go build ./...
go test ./pkg/parsley/format/...
```

**Commit:** `refactor(format): remove unused FormatValue, FormatInspectable, Printer.Reset`

### Step 2.3: Remove Evaluator Dead Code

**Owner:** AI

**Files:** Multiple evaluator files

**Actions:**
1. `eval_string_conversions.go`: Remove `evalDictionarySpread`, `objectToUserString`, `ObjectToReprString`
2. `evaluator.go`: Remove `sqliteSupportsReturning`
3. `introspect.go`: Remove `GetOperatorsByCategory`
4. `methods_unit.go`: Remove `IsUnit`
5. `record_validation.go`: Remove `ValidatePartialRecord`

**Verification:**
```bash
go build ./...
go test ./pkg/parsley/...
deadcode -test ./...  # Verify functions no longer listed
```

**Commit:** `refactor(evaluator): remove 7 unused functions`

---

## Phase 3: Deprecated API Replacement (HIGH)

### Step 3.1: Replace strings.Title with cases.Title

**Owner:** AI

**File:** `pkg/parsley/evaluator/methods_string.go`

**Actions:**
1. Add imports:
   ```go
   "golang.org/x/text/cases"
   "golang.org/x/text/language"
   ```
2. Create package-level caser (for performance):
   ```go
   var titleCaser = cases.Title(language.Und)
   ```
3. Replace all `strings.Title(...)` calls with `titleCaser.String(...)`

**Locations:**
- Line 231: `stringToTitle` function
- Line 589: `toTitleCase` helper
- Line 604: `toTitleCase` helper

**Verification:**
```bash
go build ./...
go test ./pkg/parsley/...
grep -r "strings.Title" pkg/  # Should return nothing
```

**Commit:** `fix(evaluator): replace deprecated strings.Title with cases.Title`

### Step 3.2: Fix Goldmark Deprecated Property

**Owner:** AI

**File:** `pkg/parsley/evaluator/markdown_helpers.go`

**Actions:**
1. Line 71: Replace `n.Text` with appropriate alternative (likely `n.Text.Value`)
2. Line 276: Same replacement

**Verification:**
```bash
go build ./...
go test ./pkg/parsley/...
```

**Commit:** `fix(evaluator): update goldmark deprecated Text property usage`

### Step 3.3: Fix Mailgun Deprecated Methods

**Owner:** AI

**File:** `server/auth/email/mailgun.go`

**Actions:**
1. Line 58: Replace `p.client.NewMessage(...)` with `mailgun.NewMessage(p.client.Domain(), ...)`
2. Line 62: Replace `m.SetHtml(...)` with `m.SetHTML(...)`

**Verification:**
```bash
go build ./...
go test ./server/auth/...
```

**Commit:** `fix(auth): update mailgun deprecated method calls`

---

## Phase 4: Auto-Fix Linter Issues (MEDIUM)

### Step 4.1: Run golangci-lint Auto-Fix

**Owner:** AI

**Actions:**
1. Run auto-fixer:
   ```bash
   golangci-lint run --fix --new-from-rev=""
   ```
2. Review changes (octal literals, http.NoBody, min/max)
3. Run tests to verify no regressions

**Verification:**
```bash
go test ./...
golangci-lint run --new-from-rev=""  # Count remaining issues
```

**Commit:** `style: apply golangci-lint auto-fixes (octal literals, http.NoBody, min/max)`

---

## Phase 5: Error Handling Fixes (MEDIUM)

### Step 5.1: Fix db.Close() Error Handling

**Owner:** AI

**File:** `cmd/basil/main.go`

**Actions:**
1. Line 269: Replace `defer db.Close()` with error-checking defer
2. Line 637: Same replacement

**Pattern:**
```go
defer func() {
    if err := db.Close(); err != nil {
        fmt.Fprintf(stderr, "warning: error closing database: %v\n", err)
    }
}()
```

**Verification:**
```bash
go build ./...
go test ./cmd/basil/...
```

**Commit:** `fix(cli): check error return from db.Close()`

### Step 5.2: Fix Ineffective Assignment

**Owner:** AI

**File:** `pkg/parsley/evaluator/eval_unit_infix.go`

**Actions:**
1. Line 145: Remove unused `subPerDisplayUnit` assignment or use the variable

**Verification:**
```bash
go build ./...
golangci-lint run --new-from-rev="" | grep ineffassign  # Should be empty
```

**Commit:** `fix(evaluator): remove ineffective assignment in eval_unit_infix.go`

---

## Phase 6: Finalization

### Step 6.1: Update Linter Baseline

**Owner:** AI

**File:** `.golangci.yml`

**Actions:**
1. Update `new-from-rev` to current commit after all fixes
2. Add comment with date

**Commit:** `chore: update golangci-lint baseline after code quality fixes`

### Step 6.2: Final Verification

**Owner:** AI

**Actions:**
1. Run full test suite:
   ```bash
   go test -race ./...
   ```
2. Run vulnerability check:
   ```bash
   govulncheck ./...
   ```
3. Run dead code check:
   ```bash
   deadcode -test ./...
   ```
4. Run full lint:
   ```bash
   golangci-lint run --new-from-rev=""
   ```

### Step 6.3: Update CHANGELOG

**Owner:** AI

**File:** `CHANGELOG.md`

**Actions:**
Add entry under "Unreleased" or create new version section:

```markdown
### Security
- Upgraded to Go 1.25.7 to fix TLS and URL parsing vulnerabilities
- Updated chi router dependency to v5.2.2 (GO-2025-3770)

### Changed
- `string.title()` now uses Unicode-aware title casing (golang.org/x/text/cases)
  - Edge case behavior change: apostrophes are now treated as part of words
  - Example: `"hello'world".title()` now returns `"Hello'world"` instead of `"Hello'World"`

### Fixed
- Updated deprecated mailgun API calls
- Updated deprecated goldmark Text property usage
- Fixed unchecked error returns on database close

### Removed
- Removed unused internal functions (no public API changes)
```

**Commit:** `docs: update CHANGELOG for pre-v1.0 code quality fixes`

---

## Progress Log

| Step | Status | Date | Notes |
|------|--------|------|-------|
| 1.1 Go Version | ⬜ Pending | | Requires human action |
| 1.2 Chi Update | ⬜ Pending | | |
| 2.1 Unit Conv | ⬜ Pending | | |
| 2.2 Format | ⬜ Pending | | |
| 2.3 Evaluator | ⬜ Pending | | |
| 3.1 strings.Title | ⬜ Pending | | |
| 3.2 Goldmark | ⬜ Pending | | |
| 3.3 Mailgun | ⬜ Pending | | |
| 4.1 Auto-Fix | ⬜ Pending | | |
| 5.1 db.Close | ⬜ Pending | | |
| 5.2 Ineffassign | ⬜ Pending | | |
| 6.1 Baseline | ⬜ Pending | | |
| 6.2 Verification | ⬜ Pending | | |
| 6.3 CHANGELOG | ⬜ Pending | | |

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| strings.Title behavior change breaks user code | Low | Medium | Document in CHANGELOG, behavior is edge-case |
| Dead code removal breaks something | Very Low | High | Functions confirmed unreachable by deadcode tool |
| Auto-fix introduces bugs | Low | Medium | Run full test suite after |
| Dependency update breaks build | Low | Low | Chi is indirect, we don't use affected code |

---

## Estimated Time

| Phase | Estimate |
|-------|----------|
| Phase 1 (Security) | 15 min (+ human Go install time) |
| Phase 2 (Dead Code) | 30 min |
| Phase 3 (Deprecated) | 45 min |
| Phase 4 (Auto-Fix) | 15 min |
| Phase 5 (Errors) | 15 min |
| Phase 6 (Final) | 20 min |
| **Total** | ~2.5 hours |

---

## References

- Spec: `work/specs/FEAT-130.md`
- Audit: `work/reports/PRE_V1_CODE_AUDIT.md`
