# FEAT-130: Pre-v1.0 Code Quality Fixes

## Overview

Before releasing v1.0, we need to address security vulnerabilities, remove dead code, fix deprecated API usage, and apply code modernization. This ensures the codebase is polished and defensible when exposed to public scrutiny.

## Background

A comprehensive code audit was performed on 2026-02-26 (see `work/reports/PRE_V1_CODE_AUDIT.md`). The audit identified:

- 4 known security vulnerabilities in dependencies
- 12 confirmed dead code functions
- 6 deprecated API usages
- ~660 auto-fixable style issues
- 2 unchecked error returns on database close

## Goals

1. Eliminate all known security vulnerabilities
2. Remove confirmed dead code
3. Replace deprecated APIs with modern alternatives
4. Apply auto-fixable linter improvements
5. Fix unchecked error returns

## Non-Goals

- Major refactoring or architecture changes
- Improving test coverage (separate effort)
- Fixing context propagation issues (lower priority, post-v1.0)

---

## Requirements

### 1. Security Vulnerabilities (CRITICAL)

#### 1.1 Go Standard Library Vulnerabilities

**Current State:** Running go1.25.5 with 3 known vulnerabilities:
- GO-2026-4341: Memory exhaustion in net/url (ParseForm)
- GO-2026-4340: TLS handshake at incorrect encryption level
- GO-2026-4337: Unexpected TLS session resumption

**Required Action:** Upgrade to Go 1.25.7+

**Verification:**
```bash
govulncheck ./...
# Should show no standard library vulnerabilities
```

#### 1.2 Chi Router Vulnerability

**Current State:** Indirect dependency `github.com/go-chi/chi/v5@v5.2.1` has host header injection vulnerability (GO-2025-3770)

**Required Action:** Update chi to v5.2.2+
```bash
go get github.com/go-chi/chi/v5@v5.2.2
go mod tidy
```

**Note:** Chi is an indirect dependency via mailgun-go. We don't use RedirectSlashes directly, but should still update.

---

### 2. Dead Code Removal (HIGH)

Remove the following 12 confirmed dead functions:

#### 2.1 Unit Conversion (pkg/parsley/evaluator/unit_tables.go)
| Function | Line | Reason |
|----------|------|--------|
| `ConvertUSToSI` | 306 | Non-scaled version, only `*Scaled` variants used |
| `ConvertSIToUS` | 322 | Non-scaled version, only `*Scaled` variants used |

#### 2.2 Format Package (pkg/parsley/format/format.go, printer.go)
| Function | Line | Reason |
|----------|------|--------|
| `FormatValue` | format.go:23 | Generic formatter, superseded by FormatObject |
| `FormatInspectable` | format.go:385 | Superseded by FormatObject |
| `Printer.Reset` | printer.go:25 | Never called externally |

#### 2.3 Evaluator Functions
| Function | File | Line | Reason |
|----------|------|------|--------|
| `evalDictionarySpread` | eval_string_conversions.go | 106 | Abandoned feature |
| `objectToUserString` | eval_string_conversions.go | 184 | Unused |
| `ObjectToReprString` | eval_string_conversions.go | 371 | Exported but never called |
| `sqliteSupportsReturning` | evaluator.go | 1852 | Unused feature check |
| `GetOperatorsByCategory` | introspect.go | 350 | Unused introspection |
| `IsUnit` | methods_unit.go | 1531 | Unused predicate |
| `ValidatePartialRecord` | record_validation.go | 46 | Unused validation |

**Verification:**
```bash
deadcode -test ./...
# Should not list any of the above functions
```

---

### 3. Deprecated API Replacement (HIGH)

#### 3.1 strings.Title → golang.org/x/text/cases

**Files affected:**
- `pkg/parsley/evaluator/methods_string.go:231` (stringToTitle)
- `pkg/parsley/evaluator/methods_string.go:589` (toTitleCase helper)
- `pkg/parsley/evaluator/methods_string.go:604` (toTitleCase helper)

**Current:**
```go
strings.Title(str.Value)
```

**Replacement:**
```go
import "golang.org/x/text/cases"
import "golang.org/x/text/language"

caser := cases.Title(language.Und)
caser.String(str.Value)
```

**Note:** This is a minor behavior change for edge cases involving Unicode punctuation. Document in CHANGELOG.

#### 3.2 Goldmark n.Text Deprecated Property

**Files affected:**
- `pkg/parsley/evaluator/markdown_helpers.go:71`
- `pkg/parsley/evaluator/markdown_helpers.go:276`

**Required Action:** Use `Text.Value` instead of `Text` property.

#### 3.3 Mailgun Deprecated Methods

**File:** `server/auth/email/mailgun.go`

| Line | Current | Replacement |
|------|---------|-------------|
| 58 | `p.client.NewMessage(...)` | `mailgun.NewMessage(...)` |
| 62 | `m.SetHtml(...)` | `m.SetHTML(...)` |

---

### 4. Auto-Fixable Linter Issues (MEDIUM)

Run golangci-lint with auto-fix for:

#### 4.1 Octal Literal Style (183 instances)
```go
// Before
os.MkdirAll(path, 0755)

// After  
os.MkdirAll(path, 0o755)
```

#### 4.2 http.NoBody Usage (128 instances)
```go
// Before
http.NewRequest("GET", url, nil)

// After
http.NewRequest("GET", url, http.NoBody)
```

#### 4.3 min/max Builtins (20 instances)
```go
// Before
if a < b { return a } else { return b }

// After
return min(a, b)
```

**Command:**
```bash
golangci-lint run --fix --new-from-rev=""
```

---

### 5. Error Handling Fixes (MEDIUM)

#### 5.1 Database Close Error Handling

**Files affected:**
- `cmd/basil/main.go:269`
- `cmd/basil/main.go:637`

**Current:**
```go
defer db.Close()
```

**Required:**
```go
defer func() {
    if err := db.Close(); err != nil {
        log.Printf("error closing database: %v", err)
    }
}()
```

---

### 6. Ineffective Assignment Fix (LOW)

**File:** `pkg/parsley/evaluator/eval_unit_infix.go:145`

Remove or use the `subPerDisplayUnit` variable.

---

## Out of Scope (Backlog)

The following items were identified but deferred to post-v1.0:

1. **Context propagation issues** (~17 instances) - Would require significant refactoring
2. **Commented-out code** (20 instances) - Review individually later
3. **Test coverage improvements** - Separate effort
4. **if-else → switch conversions** - Style preference, not blocking

---

## Implementation Notes

### Order of Operations

1. **Upgrade Go version first** - Ensures we're building with patched stdlib
2. **Update dependencies** - Chi vulnerability fix
3. **Remove dead code** - Clean slate before other changes
4. **Fix deprecated APIs** - May affect behavior slightly
5. **Run auto-fixer** - Bulk style improvements
6. **Fix error handling** - Minor changes
7. **Update golangci-lint baseline** - Reflect new clean state

### Testing Strategy

1. Run full test suite after each major step
2. Run `govulncheck` to verify vulnerability fixes
3. Run `deadcode -test ./...` to verify dead code removal
4. Run `golangci-lint run --new-from-rev=""` to verify lint fixes

### Breaking Changes

The `strings.Title` → `cases.Title` change may affect edge cases:
- Old: `"hello'world"` → `"Hello'World"`
- New: `"hello'world"` → `"Hello'world"` (apostrophe treated as part of word)

Document this in CHANGELOG under "Changed" section.

---

## Acceptance Criteria

- [ ] `govulncheck ./...` reports no vulnerabilities in our code paths
- [ ] `deadcode -test ./...` reports no unreachable functions in pkg/parsley/evaluator, pkg/parsley/format, or server/
- [ ] `golangci-lint run --new-from-rev=""` reports < 50 issues (down from ~1400)
- [ ] All tests pass with `go test -race ./...`
- [ ] No uses of `strings.Title` remain
- [ ] No uses of deprecated mailgun methods remain
- [ ] All `db.Close()` calls check returned error

---

## References

- Audit Report: `work/reports/PRE_V1_CODE_AUDIT.md`
- govulncheck: https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck
- cases package: https://pkg.go.dev/golang.org/x/text/cases
- Go 1.25.7 release notes: (link when available)