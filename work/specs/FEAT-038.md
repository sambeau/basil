---
id: FEAT-038
title: "Basil Namespace Cleanup"
status: implemented
priority: medium
created: 2025-12-07
implemented: 2025-12-07
author: "@human"
---

# FEAT-038: Basil Namespace Cleanup

## Summary
Move Basil-provided components from the global namespace into the `basil.*` namespace structure. This cleans up the global namespace and groups related functionality together for better organization and documentation.

## User Story
As a developer, I want Basil components organized under clear namespaces so that I can easily find related functionality and my global namespace stays clean for my own components.

## Acceptance Criteria
- [x] `PasskeyLogin` → `basil.auth.Login`
- [x] `PasskeyRegister` → `basil.auth.Register`
- [x] `PasskeyLogout` → `basil.auth.Logout`
- [x] Old component names removed (no deprecation period—we're pre-alpha)
- [x] All examples updated to use new names
- [x] Documentation updated

## Design Decisions
- **`basil.auth.Login` not `basil.auth.Passkey.Login`**: Basil is opinionated—passkeys are *the* auth mechanism, not one of many. Users don't need to think about auth implementation.
- **Components live alongside data**: `basil.auth.user` (data) and `basil.auth.Login` (component) coexist in the same namespace. They're both "auth stuff."
- **No import required**: These are auto-injected by Basil runtime, accessed via the `basil.*` namespace.

## Migration Path

### Before
```parsley
<PasskeyLogin redirect="/dashboard"/>
<PasskeyRegister redirect="/welcome"/>
<PasskeyLogout redirect="/"/>
```

### After
```parsley
<basil.auth.Login redirect="/dashboard"/>
<basil.auth.Register redirect="/welcome"/>
<basil.auth.Logout redirect="/"/>
```

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `auth/components.go` — Component registration/naming
- `server/handler.go` — Component injection into environment
- `pkg/parsley/evaluator/` — May need to support dotted component names
- `examples/auth/` — Update all example handlers
- `docs/guide/authentication.md` — Update documentation

### Current Component Registration
Components are currently registered as top-level names in the evaluator environment. Need to change to register under `basil.auth.*` namespace.

### Dotted Component Names
The evaluator needs to resolve `<basil.auth.Login/>` by:
1. Looking up `basil` in environment
2. Accessing `.auth` property
3. Accessing `.Login` property
4. Treating result as a component

This may require parser/evaluator changes if not already supported.

### Dependencies
- Depends on: None
- Blocks: FEAT-037 (Fragment Caching uses same namespace pattern)

### Edge Cases & Constraints
1. **Existing user code** — Will break (acceptable: pre-alpha, only test code exists)
2. **Component name resolution** — Need to verify evaluator can handle dotted names in tag position
3. **Error messages** — Clear error for unknown component (no special handling for old names)

## Implementation Notes

**Implemented: 2025-12-07**

The implementation uses regex-based post-processing in `auth/components.go`, which means dotted component names work without any parser/evaluator changes. The regex patterns simply match the literal text `<basil.auth.Login.../>` etc. in the HTML output and replace them with the expanded form.

Files changed:
- `auth/components.go` — Updated tag patterns from `PasskeyX` to `basil.auth.X`
- `auth/components_test.go` — Updated all test cases to use new syntax
- `examples/auth/handlers/*.pars` — Updated to use new component names
- `docs/guide/authentication.md` — Updated component documentation
- `examples/auth/README.md` — Updated example documentation

Note: The dots in the regex patterns need escaping (`\\.`) since `.` is a regex metacharacter.

## Related
- FEAT-037: Fragment Caching (will use `basil.cache.Cache`)
- FEAT-039: Enhanced Import Syntax (future, allows `{Login} = import @basil/auth`)
