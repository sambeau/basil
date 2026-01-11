---
id: PLAN-024
feature: FEAT-038
title: "Implementation Plan for Basil Namespace Cleanup"
status: completed
created: 2025-12-07
completed: 2025-12-07
---

# Implementation Plan: FEAT-038 Basil Namespace Cleanup

## Overview
Move Passkey auth components from global namespace into `basil.auth.*` namespace. This changes `<PasskeyLogin/>` to `<basil.auth.Login/>`, etc.

## Prerequisites
- [x] Design decision confirmed: `basil.auth.Login` (not `basil.auth.Passkey.Login`)
- [x] No deprecation period (pre-alpha, clean break)

## Current Architecture

Auth components are **post-processed** via regex in `auth/components.go`:
```go
tagPatterns: map[string]*regexp.Regexp{
    "PasskeyRegister": regexp.MustCompile(`<PasskeyRegister\s*([^>]*)/?>`),
    "PasskeyLogin":    regexp.MustCompile(`<PasskeyLogin\s*([^>]*)/?>`),
    "PasskeyLogout":   regexp.MustCompile(`<PasskeyLogout\s*([^>]*)/?>`),
}
```

This happens AFTER Parsley evaluation, not during. The evaluator doesn't know about these components.

## Implementation Options

### Option A: Keep Post-Processing (Simplest)
Just update the regex patterns to match dotted names:
```go
"basil.auth.Login": regexp.MustCompile(`<basil\.auth\.Login\s*([^>]*)/?>`),
```

**Pros**: Minimal change, no evaluator modifications
**Cons**: Components aren't real Parsley components, just string patterns

### Option B: Move to Evaluator (Cleaner)
Register components in the `basil` dictionary, handle `<basil.auth.Login/>` in evaluator.

**Pros**: Components become first-class, better error messages, can introspect
**Cons**: Requires parser/evaluator changes to handle dotted tag names

### Recommendation: Option A for FEAT-038
Keep post-processing for now. Future work can migrate to evaluator-based components.

---

## Tasks

### Task 1: Update Regex Patterns
**Files**: `auth/components.go`
**Estimated effort**: Small

Steps:
1. Change pattern keys and regexes:
   - `PasskeyLogin` → `basil.auth.Login`
   - `PasskeyRegister` → `basil.auth.Register`
   - `PasskeyLogout` → `basil.auth.Logout`
2. Update regex to escape dots: `<basil\.auth\.Login\s*([^>]*)/?>`
3. Update method references in `ExpandComponents()`

```go
// Before
tagPatterns: map[string]*regexp.Regexp{
    "PasskeyRegister": regexp.MustCompile(`<PasskeyRegister\s*([^>]*)/?>`),
    "PasskeyLogin":    regexp.MustCompile(`<PasskeyLogin\s*([^>]*)/?>`),
    "PasskeyLogout":   regexp.MustCompile(`<PasskeyLogout\s*([^>]*)/?>`),
}

// After
tagPatterns: map[string]*regexp.Regexp{
    "basil.auth.Register": regexp.MustCompile(`<basil\.auth\.Register\s*([^>]*)/?>`),
    "basil.auth.Login":    regexp.MustCompile(`<basil\.auth\.Login\s*([^>]*)/?>`),
    "basil.auth.Logout":   regexp.MustCompile(`<basil\.auth\.Logout\s*([^>]*)/?>`),
}
```

Tests:
- Existing component tests should fail (using old names)
- Update tests to use new names

---

### Task 2: Update Component Tests
**Files**: `auth/components_test.go`
**Estimated effort**: Small

Steps:
1. Update all test inputs from `<PasskeyLogin.../>` to `<basil.auth.Login.../>`
2. Run tests to verify

Tests:
- All existing auth component tests pass with new names

---

### Task 3: Update Examples
**Files**: `examples/auth/handlers/*.pars`
**Estimated effort**: Small

Steps:
1. Update `signup.pars`: `<PasskeyRegister>` → `<basil.auth.Register>`
2. Update `login.pars`: `<PasskeyLogin>` → `<basil.auth.Login>`
3. Update `logout.pars` (if exists): `<PasskeyLogout>` → `<basil.auth.Logout>`

Tests:
- Manual test: run example auth app, verify login/signup flow works

---

### Task 4: Update Documentation
**Files**: `docs/guide/authentication.md`, `examples/auth/README.md`
**Estimated effort**: Small

Steps:
1. Replace all `<PasskeyLogin>` → `<basil.auth.Login>`
2. Replace all `<PasskeyRegister>` → `<basil.auth.Register>`
3. Replace all `<PasskeyLogout>` → `<basil.auth.Logout>`
4. Update any prose referring to "PasskeyLogin component"

---

### Task 5: Update Specs/Plans (Housekeeping)
**Files**: `work/specs/FEAT-004.md`, `work/plans/FEAT-004-plan.md`, other docs
**Estimated effort**: Small

Steps:
1. Search for `PasskeyLogin` across docs/
2. Update references (or note as historical)

---

## Validation Checklist
- [ ] All tests pass: `make check`
- [ ] Auth example works: manual test signup/login/logout flow
- [ ] No references to old names in examples/
- [ ] Documentation updated

## Rollback
If issues arise, revert regex patterns in `auth/components.go`.

## Notes
- This approach keeps the post-processing architecture
- Future enhancement: move components to evaluator for first-class support
- Consider adding error message if user tries old name (search for `<PasskeyLogin` in output, warn)
