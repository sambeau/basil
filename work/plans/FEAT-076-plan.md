---
id: PLAN-047
feature: FEAT-076
title: "Implementation Plan for Protected Paths and Role Enforcement"
status: completed
created: 2025-12-27
completed: 2025-12-27
---

# Implementation Plan: FEAT-076 (Protected Paths for Site Mode)

## Overview

Implement `auth.protected_paths` configuration for protecting URL path prefixes, and complete the role-based access control system so that `adminOnly()` and `roles()` wrappers actually enforce roles.

This plan has two main parts:
1. **Protected Paths** — New config-driven path protection for site mode
2. **Role Enforcement** — Fix the broken role checking in API handlers

## Prerequisites

- [x] FEAT-004 (Auth system) — implemented
- [x] FEAT-049 (Sessions) — implemented
- [x] FEAT-036 (CLI user management with roles) — implemented
- [x] `std/api` wrappers exist (`adminOnly`, `roles`) — implemented but not enforced

## Progress Summary

| Phase | Status | Notes |
|-------|--------|-------|
| Phase 1: Role Enforcement | ✅ DONE | Fixed role checking in API handlers |
| Phase 2: Protected Paths Config | ✅ DONE | Added config parsing with YAML unmarshaler |
| Phase 3: Site Mode Protection | ✅ DONE | Integrated with site handler |
| Phase 4: Routes Mode Integration | ✅ DONE | Added auth:none override and roles field |
| Phase 5: Documentation | ✅ DONE | Updated authentication.md and faq.md |

## Known Gaps

**CSRF in Site Mode**: CSRF middleware is applied for routes mode protected paths, but not for site mode. In site mode, handlers can access `basil.csrf.token` to include in forms, but POST validation is not automatically enforced at the middleware level. This is deferred to BACKLOG.md.

## Tasks

### Phase 1: Role Enforcement (Fix Existing Broken Feature)

#### Task 1.1: Populate `req.user.role` in Request Context
**Files**: `server/handler.go`, `server/api.go`
**Estimated effort**: Small

Steps:
1. In `buildRequestContext()` or equivalent, add `role` to the user dictionary
2. When `auth.User` is available, include `user.Role` in the Parsley `req.user` object
3. Verify the role comes from the session/database correctly

Tests:
- Authenticated request has `req.user.role` populated
- Role value matches user's actual role from database
- Unauthenticated request has no `req.user`

---

#### Task 1.2: Fix `enforceAuth()` Role Comparison
**Files**: `server/api.go`
**Estimated effort**: Small

Steps:
1. Locate `enforceAuth()` function (~line 206)
2. Replace the blanket deny for admin/roles with actual comparison:
   ```go
   if meta.AuthType == "admin" {
       if user.Role != auth.RoleAdmin {
           h.writeAPIError(w, &evaluator.APIError{
               Code: "HTTP-403", Message: "Forbidden: admin required", Status: 403})
           return nil, false
       }
   }
   if meta.AuthType == "roles" && len(meta.Roles) > 0 {
       if !slices.Contains(meta.Roles, user.Role) {
           h.writeAPIError(w, &evaluator.APIError{
               Code: "HTTP-403", Message: "Forbidden: insufficient role", Status: 403})
           return nil, false
       }
   }
   ```
3. Handle edge case: user with empty role → deny role-protected routes

Tests:
- `adminOnly(fn)` allows admin user
- `adminOnly(fn)` denies editor user with 403
- `roles(["editor", "admin"], fn)` allows editor
- `roles(["editor", "admin"], fn)` allows admin
- `roles(["admin"], fn)` denies editor
- User with no role denied on role-protected routes

---

#### Task 1.3: Add Role Enforcement Tests
**Files**: `server/api_test.go`
**Estimated effort**: Medium

Steps:
1. Add test for `adminOnly` wrapper with admin user → 200
2. Add test for `adminOnly` wrapper with editor user → 403
3. Add test for `roles(["editor"])` with editor → 200
4. Add test for `roles(["editor"])` with admin → 403 (no implicit hierarchy)
5. Add test for `roles(["admin", "editor"])` with either → 200

Tests: Self-testing task

---

### Phase 2: Protected Paths Configuration

#### Task 2.1: Add Config Struct Fields
**Files**: `config/config.go`
**Estimated effort**: Small

Steps:
1. Add to `AuthConfig` struct:
   ```go
   ProtectedPaths []ProtectedPath `yaml:"protected_paths"`
   ```
2. Define `ProtectedPath` type to support both simple strings and objects:
   ```go
   type ProtectedPath struct {
       Path  string   `yaml:"path"`
       Roles []string `yaml:"roles,omitempty"`
   }
   ```
3. Add custom YAML unmarshaler to handle mixed array (strings and objects)

Tests:
- Parse simple string array: `["/dashboard", "/admin"]`
- Parse object array: `[{path: "/admin", roles: ["admin"]}]`
- Parse mixed array: `["/dashboard", {path: "/admin", roles: ["admin"]}]`

---

#### Task 2.2: Config Parsing and Validation
**Files**: `config/load.go`
**Estimated effort**: Small

Steps:
1. Implement custom `UnmarshalYAML` for `ProtectedPath` or `[]ProtectedPath`
2. Validate paths start with `/`
3. Validate roles are known values (admin, editor) or allow any string
4. Normalize paths (remove trailing slash for consistent matching)

Tests:
- Invalid path (no leading `/`) returns error
- Empty protected_paths is valid (no protection)
- Duplicate paths handled gracefully

---

#### Task 2.3: Add Helper Function for Path Matching
**Files**: `server/server.go` or `server/protected.go` (new)
**Estimated effort**: Small

Steps:
1. Create `isProtectedPath(path string, protectedPaths []ProtectedPath) *ProtectedPath`
2. Match logic: `strings.HasPrefix(requestPath, protectedPath)` or exact match for `/`
3. Handle trailing slash: `/dashboard` matches `/dashboard`, `/dashboard/`, `/dashboard/foo`
4. Return the matching `ProtectedPath` (for role info) or nil

Tests:
- `/dashboard/users` matches `/dashboard`
- `/dashboardx` does NOT match `/dashboard`
- `/dashboard` matches `/dashboard`
- `/` matches everything (if configured)
- Returns correct `ProtectedPath` with roles

---

### Phase 3: Site Mode Protection

#### Task 3.1: Integrate Protection Check in Site Handler
**Files**: `server/site.go`
**Estimated effort**: Medium

Steps:
1. In `siteHandler.ServeHTTP()`, before handler lookup:
   ```go
   if pp := s.server.isProtectedPath(r.URL.Path); pp != nil {
       user := auth.GetUser(r)
       if user == nil {
           // Redirect to login or 401 for API
           s.handleUnauthenticated(w, r)
           return
       }
       if len(pp.Roles) > 0 && !slices.Contains(pp.Roles, user.Role) {
           // 403 Forbidden
           s.handleForbidden(w, r)
           return
       }
   }
   ```
2. Create helper `handleUnauthenticated()` — redirects to `/login` for HTML, 401 for API/JSON
3. Create helper `handleForbidden()` — 403 response
4. Apply CSRF middleware for protected non-GET requests

Tests:
- Unauthenticated request to `/dashboard/page.pars` → redirect to login
- Authenticated request to `/dashboard/page.pars` → serves content
- Static file `/dashboard/style.css` under protected path → requires auth
- API request to protected path → 401 JSON (not redirect)

---

#### Task 3.2: Handle Login Redirect
**Files**: `server/site.go`, `config/config.go`
**Estimated effort**: Small

Steps:
1. Add `auth.login_path` config option (default: `/login`)
2. On unauthenticated access to protected path, redirect to login with `?next=` param
3. After login, redirect back to original URL (handled by login handler)

Tests:
- Redirect includes `?next=/dashboard/page`
- Login path is configurable

---

### Phase 4: Routes Mode Integration

#### Task 4.1: Add `auth: none` Route Option
**Files**: `config/config.go`, `server/server.go`
**Estimated effort**: Small

Steps:
1. Recognize `auth: none` in route config as explicit "not protected"
2. When processing route, if `auth: none`, skip protected_paths check
3. Document the precedence: explicit route auth > protected_paths > default

Tests:
- Route with `auth: none` under protected prefix → accessible without auth
- Route with `auth: required` → requires auth regardless of protected_paths
- Route with no auth setting under protected prefix → requires auth

---

#### Task 4.2: Add `roles` to Route Config
**Files**: `config/config.go`, `server/server.go`
**Estimated effort**: Small

Steps:
1. Add `Roles []string` to route config struct
2. When enforcing auth on route, check roles if specified
3. Works in combination with `auth: required`

Tests:
- Route with `roles: [admin]` denies editor
- Route with `roles: [admin, editor]` allows both
- Route with `auth: required` but no roles → any authenticated user

---

### Phase 5: Documentation

#### Task 5.1: Update Configuration Documentation
**Files**: `docs/guide/configuration.md` or similar
**Estimated effort**: Small

Steps:
1. Document `auth.protected_paths` config
2. Document `auth: none` route option
3. Document `roles` route option
4. Add examples for common patterns

---

#### Task 5.2: Update FAQ
**Files**: `docs/guide/faq.md`
**Estimated effort**: Small

Steps:
1. Add "How do I protect an entire section of my site?"
2. Add "How do I make one page public under a protected path?"
3. Add "How do I restrict a route to admins only?"

---

## Validation Checklist

- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Role enforcement works for `adminOnly()` and `roles()` wrappers
- [ ] Protected paths work in site mode
- [ ] Protected paths work in routes mode
- [ ] `auth: none` override works
- [ ] Static files under protected paths are protected
- [ ] API routes return 401/403 JSON (not redirect)
- [ ] Documentation updated
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| — | — | — | — |

## Deferred Items

Items to add to BACKLOG.md after implementation:
- **Role hierarchy** — `admin` implying `editor` permissions; keep flat for now
- **Custom role resolver** — Parsley function to compute roles dynamically
- **Per-request role refresh** — Check DB on each request vs session cache
- **Wildcard patterns** — `/api/*/private` style matching (not needed for MVP)
