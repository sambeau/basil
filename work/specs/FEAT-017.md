---
id: FEAT-017
title: "Handler Root Path Alias (@~/)"
status: complete
priority: medium
created: 2025-12-02
author: "@sambeau"
---

# FEAT-017: Handler Root Path Alias (@~/)

## Summary
Add a `@~/` path prefix that resolves paths relative to the handler's root directory instead of the current file. This eliminates verbose relative paths like `@../../../components/page.pars` in favor of `@~/components/page.pars`.

## User Story
As a Parsley developer, I want to import modules using a root-relative path so that I don't need to count `../` segments and my imports remain stable when I move files.

## Acceptance Criteria
- [x] `@~/path` in imports resolves from the handler's directory (the directory containing the route's handler file)
- [x] `@~/path` works in `read()` and `write()` operations
- [x] Error messages show the resolved path when `@~/` paths fail
- [x] Works correctly with nested imports (handler root is preserved, not the importing file's directory)
- [x] Documentation updated with examples

## Design Decisions
- **`@~/` syntax**: Chosen over `@/` to avoid confusion with absolute filesystem paths. Familiar to webpack/Vite users.
- **Handler root = handler directory**: For a route with `handler: ./app/app.pars`, the root is `./app/`. This is intuitive and matches the project structure.
- **Not a security boundary**: `@~/` is purely a path resolution shorthand. The existing security policy controls what paths are actually allowed.
- **Preserved through imports**: If `app.pars` imports `@~/utils/helpers.pars`, and `helpers.pars` imports `@~/components/Button.pars`, both resolve from `./app/` (not from `./app/utils/`).

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/evaluator/evaluator.go` — Add `RootPath` field to `Environment`, modify `resolveModulePath()` to handle `~/` prefix
- `pkg/parsley/evaluator/methods.go` — Update `read()`, `write()` to use `resolveModulePath()` with `~/` support
- `server/handler.go` — Set `RootPath` when creating the evaluator environment

### Implementation Approach

1. **Add `RootPath` to Environment struct**:
   ```go
   type Environment struct {
       // ... existing fields
       RootPath string // Handler root directory for @~/ resolution
   }
   ```

2. **Propagate RootPath through imports**: When creating child environments for imported modules, copy `RootPath` from parent (unlike `Filename` which changes per file).

3. **Modify `resolveModulePath()`**:
   ```go
   func resolveModulePath(pathStr string, currentFile string, rootPath string) (string, error) {
       // Handle @~/ prefix - resolve from rootPath
       if strings.HasPrefix(pathStr, "~/") {
           if rootPath == "" {
               return "", fmt.Errorf("cannot use ~/ path: no handler root defined")
           }
           return filepath.Clean(filepath.Join(rootPath, pathStr[2:])), nil
       }
       // ... existing logic for absolute and relative paths
   }
   ```

4. **Update all callers of `resolveModulePath()`** to pass `env.RootPath`.

5. **Set RootPath in Basil server**: In `handler.go`, when creating the environment:
   ```go
   env.RootPath = filepath.Dir(h.scriptPath) // h.scriptPath is the handler file
   ```

### Dependencies
- None - this is a standalone feature

### Edge Cases & Constraints
1. **Standalone Parsley (pars CLI)**: `RootPath` will be empty. Using `@~/` should produce a clear error: "cannot use ~/ path: no handler root defined"
2. **Nested routes with different handlers**: Each route's handler defines its own root. This is correct behavior.
3. **Security policy interaction**: `@~/` paths are resolved first, then checked against the security policy. No special handling needed.

## Implementation Notes
*To be added during implementation*

## Related
- Plan: `docs/plans/FEAT-017-plan.md` (to be created)
