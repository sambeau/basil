# Pre-v1.0 Code Quality Audit Report

**Date:** 2026-02-26  
**Auditor:** Claude (AI)  
**Codebase:** Basil/Parsley  
**Lines of Code:** ~178,000 (including ~86,000 test LOC)

## Executive Summary

This audit assessed the Basil codebase for security vulnerabilities, race conditions, code quality, dead code, and modernization opportunities before the v1.0 release. The codebase is generally well-structured, but there are several issues requiring attention before public release.

### Priority Levels
- 🔴 **CRITICAL** - Must fix before v1.0
- 🟠 **HIGH** - Should fix before v1.0
- 🟡 **MEDIUM** - Fix during v1.0 development cycle
- 🟢 **LOW** - Nice to have / backlog

---

## 1. Security Issues

### 🔴 CRITICAL: Known Vulnerabilities in Dependencies

**govulncheck found 4 exploitable vulnerabilities:**

#### 1.1 Go Standard Library Vulnerabilities (go1.25.5)

| Vulnerability | Description | Fixed In |
|--------------|-------------|----------|
| GO-2026-4341 | Memory exhaustion in query parameter parsing (net/url) | go1.25.6 |
| GO-2026-4340 | TLS handshake processing at incorrect encryption level | go1.25.6 |
| GO-2026-4337 | Unexpected TLS session resumption | go1.25.7 |

**Impact:** Our code calls `http.Request.ParseForm` and uses TLS extensively.

**Recommendation:** Upgrade to Go 1.25.7+ immediately.

#### 1.2 Chi Router Vulnerability (v5.2.1)

| Vulnerability | Description | Fixed In |
|--------------|-------------|----------|
| GO-2025-3770 | Host header injection in RedirectSlashes → open redirect | v5.2.2 |

**Impact:** Chi is an indirect dependency via `mailgun-go/v4`. We don't use `RedirectSlashes` directly, but the vulnerable code is in our dependency tree via `chi.init()`.

**Recommendation:** Update mailgun or pin chi to v5.2.2+.

---

### 🟠 HIGH: Security-Sensitive Code Patterns

#### 1.3 SSH InsecureIgnoreHostKey (gosec G106)

**File:** `pkg/parsley/evaluator/evaluator.go:2272`

```go
HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Default to accept any (user can override)
```

**Current behavior:** Defaults to accepting any host key, with option to specify known_hosts file.

**Recommendation:** This is acceptable for a scripting language, but:
1. Document the security implications clearly
2. Consider warning users when no known_hosts is specified
3. Add example showing secure usage in docs

#### 1.4 Subprocess Execution (gosec G204)

**Files:** `pkg/parsley/evaluator/evaluator.go:4273, 4338`

The `exec.Command` usage appears safe - arguments are passed directly without shell interpretation. However:

**Recommendation:** 
1. Document which commands are allowed (if any restrictions exist)
2. Consider adding a security mode that disables command execution

#### 1.5 Weak Cryptographic Primitives (gosec G501, G505)

**File:** `pkg/parsley/evaluator/stdlib_hash.go`

MD5 and SHA1 are exposed in the `@std/hash` module.

**Assessment:** These are intentionally available for compatibility/interop use cases (checksums, legacy systems). This is **acceptable** but should be documented.

**Recommendation:**
1. Add documentation warning against using MD5/SHA1 for security purposes
2. Consider deprecation notices in the function descriptions

#### 1.6 File Permission Issues (gosec G301, G302, G306)

Multiple files create directories with 0755 and files with 0644 permissions.

**Files affected:**
- `cmd/basil/init.go` (project scaffolding)
- `pkg/parsley/evaluator/methods.go` (file operations)
- Various test files

**Assessment:** For a development tool creating project files, 0755/0644 is reasonable. For sensitive files, 0700/0600 would be better.

**Recommendation:** 
1. Keep 0755/0644 for user-facing project files (index.pars, etc.)
2. Use 0600 for any config files that might contain secrets
3. Document security implications

---

## 2. Race Conditions

### ✅ Race Detection Results

**Result:** `go test -race ./...` passed for all packages (except unrelated test failures).

**Assessment:** No race conditions detected under test. The codebase uses mutexes appropriately.

### 🟡 MEDIUM: Mutex Patterns Review

Several files use Lock/Unlock without defer:

**Files:**
- `server/cache.go`
- `server/assets.go`
- `server/fragment_cache.go`
- `server/handler.go`
- `server/watcher.go`
- `pkg/parsley/evaluator/connection_cache.go`
- `pkg/parsley/evaluator/stdlib_id.go`

**Assessment:** Manual review shows these are all **safe** - short critical sections with no panics or early returns between Lock() and Unlock(). This pattern is acceptable for performance-critical paths.

**Recommendation:** No action required, but consider adding comments explaining why defer isn't used in these cases.

---

## 3. Dead Code

### 🟠 HIGH: Unreachable Functions (34 total)

The `deadcode` tool found 34 unreachable functions:

#### 3.1 AST Token Methods (likely intentional interface satisfaction)
- `QueryOneStatement.TokenLiteral/String`
- `QueryManyStatement.TokenLiteral/String`
- `ExecuteStatement.TokenLiteral/String`
- `QueryCTERef.TokenLiteral/String`

**Assessment:** These implement the `Node` interface but may not be called. Keep if interface conformance is needed.

#### 3.2 Evaluator Functions (should review)
| Function | File | Likely Status |
|----------|------|---------------|
| `evalDictionarySpread` | eval_string_conversions.go:106 | Unused feature |
| `objectToUserString` | eval_string_conversions.go:184 | Unused |
| `ObjectToReprString` | eval_string_conversions.go:371 | Exported but unused |
| `sqliteSupportsReturning` | evaluator.go:1852 | Future/unused |
| `GetOperatorsByCategory` | introspect.go:350 | Exported but unused |
| `IsUnit` | methods_unit.go:1531 | Unused |
| `ValidatePartialRecord` | record_validation.go:46 | Unused feature |
| `ConvertUSToSI` | unit_tables.go:306 | Unused |
| `ConvertSIToUS` | unit_tables.go:322 | Unused |

#### 3.3 Format Package
| Function | File |
|----------|------|
| `FormatValue` | format.go:23 |
| `FormatInspectable` | format.go:385 |
| `Printer.Reset` | printer.go:25 |

#### 3.4 Parsley High-Level API
| Function | File |
|----------|------|
| `StdoutLogger` | logger.go:17 |
| `writerLogger.Log/LogLine` | logger.go:26,30 |
| `WriterLogger` | logger.go:35 |
| `nullLogger.Log/LogLine` | logger.go:104,105 |
| `NullLogger` | logger.go:108 |
| `WithEnv` | options.go:30 |
| `WithSecurity` | options.go:37 |
| `WithFilename` | options.go:51 |
| `EvalFile` | parsley.go:133 |

#### 3.5 PLN Package
- `Parser.Errors` (pln/parser.go:56)
- `SerializeWithEnv` (pln/pln.go:57)

#### 3.6 Server Package
| Function | File |
|----------|------|
| `DefaultConfig` | auth/auth.go:73 |
| `GetUserFromContext` | auth/middleware.go:79 |
| `FTS5Index.Weights` | search/fts5.go:137 |
| `UpdateStats.String` | search/watcher.go:165 |

**Recommendation:** 
1. Review each function - remove if truly unused
2. Keep if part of public API or needed for interface conformance
3. Consider: some may be intended for future use or external consumers

---

## 4. Code Quality & Modernization

### 🟡 MEDIUM: Linter Issues Summary

| Linter | Count | Priority |
|--------|-------|----------|
| errcheck | 667 | Medium |
| gocritic | 660 | Low-Medium |
| staticcheck | 19 | Medium |
| modernize | 5 | Low |
| contextcheck | ~17 | Medium |

### 4.1 Error Checking (errcheck) - 667 issues

Most are `fmt.Fprintf` return values not checked. This is common and low-risk for stdout/stderr writes.

**High-priority errcheck issues:**
- `cmd/basil/main.go:269` - `db.Close()` return value not checked
- `cmd/basil/main.go:637` - `db.Close()` return value not checked

**Recommendation:**
1. Fix db.Close() error handling (log errors at minimum)
2. For fmt.Fprintf to stdout/stderr, this is acceptable to ignore
3. Consider adding `//nolint:errcheck` comments with justification

### 4.2 gocritic Issues - 660 total

| Issue Type | Count | Action |
|------------|-------|--------|
| Octal literal style (0644 → 0o644) | 183 | Auto-fix |
| `http.NoBody` should be used | 128 | Auto-fix |
| if-else → switch | 88 | Optional |
| Use %q for quoted strings | 38 | Auto-fix |
| Named return values | 24 | Optional |
| if-else → type switch | 20 | Optional |
| Commented-out code | 20 | Review & remove |
| Use min/max builtins | 20 | Auto-fix |

**Recommendation:** 
1. Run `golangci-lint run --fix` for auto-fixable issues
2. Manually review and remove commented-out code
3. Consider if-else → switch conversions case-by-case

### 4.3 staticcheck Issues - 19 total

| Code | Description | Count | Action |
|------|-------------|-------|--------|
| SA1019 | Deprecated API usage | 6 | Fix |
| QF1001 | De Morgan's law simplification | 6 | Optional |
| ST1023 | Redundant type declaration | 2 | Fix |
| SA9003 | Empty branch | 1 | Review |
| SA4011 | Ineffective break | 1 | Fix |
| QF1003 | Tagged switch suggestion | 1 | Optional |

**Priority fixes:**
1. `strings.Title` deprecated → use `golang.org/x/text/cases`
2. `mailgun.NewMessage` deprecated method usage
3. Goldmark `n.Text` deprecated property

### 4.4 modernize Issues - 5 total

All are minor and auto-fixable:
- `range int` modernization (2)
- `SplitSeq` usage (3)

### 4.5 Context Propagation (contextcheck) - ~17 issues

Multiple functions don't pass context parameters properly through call chains.

**Example:** `server/server.go:1028` - Non-inherited context in shutdown

**Recommendation:** Review context propagation in server package. This affects cancellation and timeout behavior.

### 4.6 Ineffective Assignment - 1 issue

**File:** `pkg/parsley/evaluator/eval_unit_infix.go:145`
```go
subPerDisplayUnit // assigned but never used
```

**Recommendation:** Remove or use the variable.

---

## 5. Test Coverage

| Package | Coverage | Assessment |
|---------|----------|------------|
| pkg/parsley/errors | 90.4% | ✅ Good |
| pkg/parsley/tests | 100.0% | ✅ Excellent |
| server/search | 80.0% | ✅ Good |
| server/config | 78.0% | ✅ Good |
| pkg/parsley/help | 78.1% | ✅ Good |
| pkg/parsley/pln | 63.0% | 🟡 Acceptable |
| server | 59.9% | 🟡 Acceptable |
| pkg/parsley/format | 55.4% | 🟡 Acceptable |
| server/auth | 52.2% | 🟡 Needs improvement |
| pkg/parsley/parsley | 36.1% | 🟠 Low |
| pkg/parsley/lexer | 27.1% | 🟠 Low |
| pkg/parsley/evaluator | 12.6% | 🔴 Critical - needs integration tests |
| cmd/basil | 12.6% | 🟠 Low |
| cmd/pars | 0.0% | 🔴 No tests |

**Note:** The evaluator has low unit test coverage but is heavily tested via integration tests in `pkg/parsley/tests`.

### Test Failures Observed

Two tests failed during the audit (likely environment-specific):
1. `TestDatabaseShutdown` - port 8080 already in use
2. `TestSiteHandler_RootPath` - security restriction error

**Recommendation:** Investigate these failures; they may indicate test isolation issues.

---

## 6. Architecture Observations

### ✅ Positive Patterns
1. Clean separation between Parsley language and Basil server
2. Good use of interfaces for abstraction
3. Consistent error handling patterns with structured errors
4. Security-conscious file path handling

### 🟡 Areas for Improvement
1. Some very large files (evaluator.go) could benefit from further splitting
2. Consider extracting common mutex patterns into helper types
3. Some duplicated error creation patterns

---

## 7. Recommended Actions

### Before v1.0 Release (Priority Order)

#### 1. 🔴 CRITICAL - Upgrade Go Version
```bash
# Update go.mod to require go1.25.7+
go get go@1.25.7
```

#### 2. 🔴 CRITICAL - Update Vulnerable Dependencies
```bash
go get github.com/go-chi/chi/v5@v5.2.2
go mod tidy
```

#### 3. 🟠 HIGH - Remove Dead Code
Review and remove the 34 unreachable functions identified above.

#### 4. 🟠 HIGH - Fix Deprecated API Usage
- Replace `strings.Title` with `cases.Title`
- Update goldmark `n.Text` usage
- Update mailgun deprecated methods

#### 5. 🟡 MEDIUM - Auto-Fix Linter Issues
```bash
golangci-lint run --fix --new-from-rev=""
```

#### 6. 🟡 MEDIUM - Fix db.Close() Error Handling
```go
if err := db.Close(); err != nil {
    log.Printf("error closing database: %v", err)
}
```

#### 7. 🟢 LOW - Review Commented-Out Code
Remove or document the 20 instances of commented-out code.

### Post v1.0 (Backlog)

1. Improve test coverage for evaluator unit tests
2. Add tests for cmd/pars
3. Review and improve context propagation
4. Consider splitting large files
5. Add security documentation for SSH and command execution

---

## 8. Tools Used

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25.5 | Compiler/runtime |
| golangci-lint | 2.7.2 | Linting suite |
| govulncheck | 1.1.4 | Vulnerability scanning |
| deadcode | latest | Unreachable code detection |
| go test -race | built-in | Race condition detection |

---

## Appendix: Full Vulnerability Report

```
=== govulncheck output ===

Vulnerability #1: GO-2026-4341 (net/url)
Vulnerability #2: GO-2026-4340 (crypto/tls)
Vulnerability #3: GO-2026-4337 (crypto/tls)
Vulnerability #4: GO-2025-3770 (chi RedirectSlashes)

All vulnerabilities have available fixes.
```

---

*End of Report*