---
id: FEAT-076
title: "Protected Paths for Site Mode"
status: implemented
priority: medium
created: 2025-12-27
implemented: 2025-12-27
author: "@human"
---

# FEAT-076: Protected Paths for Site Mode

## Summary
Add `auth.protected_paths` configuration to enable authentication requirements for URL path prefixes. This allows protecting entire sections of a site (e.g., `/dashboard`, `/admin`) without requiring explicit route definitions for each page. Works with both site mode (filesystem routing) and routes mode.

## User Story
As a developer, I want to protect entire URL prefixes with authentication so that I don't need to define explicit routes with `auth: required` for every protected page, especially when using site mode's filesystem-based routing.

## Acceptance Criteria
- [x] New `auth.protected_paths` config accepts a list of URL path prefixes
- [x] Any request to a protected path prefix requires authentication
- [x] Unauthenticated requests to protected paths redirect to login (or return 401 for API requests)
- [x] Protected paths work with site mode (filesystem routing)
- [x] Protected paths work with routes mode (explicit routes without `auth:` still protected)
- [x] Path matching is prefix-based: `/dashboard` protects `/dashboard/`, `/dashboard/users`, `/dashboard/users/123`
- [x] Static files under protected paths are also protected
- [ ] CSRF middleware is applied to protected paths (already applied to auth routes)
- [x] More specific route-level `auth:` settings override protected_paths (e.g., a route can be `auth: none` to exclude it)

## Design Decisions

### Config Location: `auth.protected_paths`
**Rationale:** Auth-related settings belong under the `auth:` section. This keeps all authentication configuration together and makes it clear this is an auth feature, not a routing feature.

```yaml
auth:
  enabled: true
  protected_paths:
    - /dashboard
    - /admin
    - /settings
```

### Prefix Matching (not exact)
**Rationale:** The primary use case is protecting entire sections of a site. `/dashboard` should protect `/dashboard/`, `/dashboard/users`, etc. Exact matching would require listing every path, defeating the purpose.

### Route-level override
**Rationale:** Allow exceptions. A route with explicit `auth: none` (new value) or `auth: ""` should not require auth even if under a protected prefix. This enables public pages within otherwise protected areas (e.g., `/admin/login`).

### Static files included
**Rationale:** Protected areas often have private assets. A dashboard's images/JS/CSS shouldn't be accessible to unauthenticated users. This matches Rails' approach.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `config/config.go` — Add `ProtectedPaths []string` to AuthConfig struct
- `server/server.go` — Check protected paths in request handling, apply auth middleware
- `server/site.go` — Integrate protected paths check before serving handlers
- `auth/middleware.go` — May need to expose auth check for site handler use

### Request Flow

1. Request arrives at server
2. Check if URL path starts with any protected prefix
3. If protected:
   - Apply RequiredAuth middleware
   - Apply CSRF middleware (for non-API)
4. Continue to handler (site mode or routes mode)

### For Site Mode
The `siteHandler.ServeHTTP()` needs to check protected paths before serving content. This should happen before the handler lookup to protect static files too.

### For Routes Mode
Routes without explicit `auth:` setting should check protected_paths. Routes with explicit `auth: required` or `auth: optional` continue to work as-is. Add `auth: none` as explicit "not protected" override.

### Dependencies
- Depends on: FEAT-004 (Auth system) — already implemented
- Depends on: FEAT-049 (Sessions) — already implemented

### Edge Cases & Constraints

1. **Trailing slash handling** — `/dashboard` should match `/dashboard`, `/dashboard/`, and `/dashboard/anything`
2. **Root path** — `/` as protected path would protect entire site (valid but unusual)
3. **Overlapping prefixes** — `/admin` and `/admin/users` both listed: request to `/admin/users/123` matches both, but behavior is same (protected)
4. **Case sensitivity** — Paths are case-sensitive (standard HTTP behavior)
5. **Static files** — Static routes (from `static:` config) under protected paths should also require auth
6. **API routes** — API routes under protected paths should return 401 JSON, not redirect to login

### Auth Mode Values
After this feature:
- `"required"` — Must be authenticated
- `"optional"` — Auth checked but not required, user info available if logged in
- `""` (empty) — Inherit from protected_paths or default to no auth
- `"none"` — Explicitly not protected, even if under protected_paths

### Role-Based Access Control

**Current State:**
- Users have a `role` field in the database (`admin` or `editor`, see FEAT-036)
- CLI tool (`basil users create --role admin`) can assign roles
- `std/api` module provides `adminOnly(fn)` and `roles(["role1", "role2"], fn)` wrappers
- **Server-side enforcement is incomplete**: Role-protected routes currently **deny all requests** because `enforceAuth()` in [server/api.go](server/api.go#L219) doesn't check the user's actual role

**What's Missing:**
1. `req.user.role` is not populated — handlers can't check roles manually
2. `enforceAuth()` doesn't compare `meta.Roles` against `user.Role`
3. Protected paths have no role concept (only authenticated vs not)

**Proposed Design:**

1. **Populate `req.user.role`** — When building the request context for Parsley handlers, include the user's role from the session/database. This enables manual role checks in handler code.

2. **Fix `enforceAuth()` in API handlers** — Compare `user.Role` against `meta.Roles`:
   ```go
   if meta.AuthType == "admin" {
       if user.Role != "admin" {
           // Return 403 Forbidden
       }
   }
   if meta.AuthType == "roles" && len(meta.Roles) > 0 {
       if !contains(meta.Roles, user.Role) {
           // Return 403 Forbidden
       }
   }
   ```

3. **Add role-based protected paths (optional enhancement)**:
   ```yaml
   auth:
     protected_paths:
       - /dashboard           # Any authenticated user
       - path: /admin
         roles: [admin]       # Only admins
   ```
   This extends protected_paths from simple prefixes to objects with role requirements. Simpler cases remain strings for backward compatibility.

4. **Route-level role requirements**:
   ```yaml
   routes:
     - path: /admin/users
       handler: ./handlers/admin-users.pars
       auth: required
       roles: [admin]
   ```

**Integration with `std/api` wrappers:**

The `adminOnly()` and `roles()` wrappers already set metadata on handlers:
- `adminOnly(fn)` → `AuthType: "admin", Roles: ["admin"]`
- `roles(["editor", "admin"], fn)` → `AuthType: "roles", Roles: ["editor", "admin"]`

This metadata is read by `readAuthMetadata()` in server/api.go. The fix is completing the enforcement logic to actually check roles.

**Role Hierarchy (Future Consideration):**

Current design has flat roles (admin, editor). A future enhancement could support hierarchical roles where `admin` implies `editor` permissions. For now, keep it simple: exact role matching only. Handlers needing both can use `roles(["admin", "editor"], fn)`.

**Edge Cases:**
- User with no role set → Treat as no role, deny role-protected routes
- Unknown role value → Deny role-protected routes (fail secure)
- Role changed mid-session → Decision: Check role on each request from DB, or accept stale session data? Recommend checking DB for admin operations, session data for normal requests.

## Implementation Notes
*To be added during implementation*

## Test Cases
1. Protected path with authenticated user → serves content
2. Protected path with unauthenticated user → redirects to login
3. Protected path subpath `/dashboard/users/123` → protected
4. Non-protected path → accessible without auth
5. Route with explicit `auth: none` under protected prefix → accessible
6. Static file under protected path → protected
7. API route under protected path → returns 401 JSON (not redirect)

## Example Configuration

```yaml
auth:
  enabled: true
  registration: open
  session_ttl: 24h
  login_path: /login              # Optional, defaults to /login
  protected_paths:
    - /dashboard                   # Any authenticated user
    - path: /admin
      roles: [admin]               # Only admins
    - /api/private

routes:
  - path: /
    handler: ./handlers/index.pars
  
  - path: /login
    handler: ./handlers/login.pars
  
  # No explicit auth needed - protected by protected_paths
  - path: /dashboard
    handler: ./handlers/dashboard.pars
  
  # Override: public page within protected area
  - path: /admin/login
    handler: ./handlers/admin-login.pars
    auth: none
  
  # Route with explicit role requirement
  - path: /admin/users
    handler: ./handlers/admin-users.pars
    auth: required
    roles: [admin]
```

Or in site mode:
```yaml
site: ./site

auth:
  enabled: true
  protected_paths:
    - /dashboard
    - /admin
```

## Implementation Notes

### Files Changed
- [config/config.go](config/config.go) — Added `ProtectedPaths`, `LoginPath` to AuthConfig, `Roles` to Route
- [config/config_test.go](config/config_test.go) — Added protected path parsing tests
- [server/server.go](server/server.go) — Added `isProtectedPath()`, `protectedPathMiddleware()`, auth helpers
- [server/server_test.go](server/server_test.go) — Added path matching tests
- [server/site.go](server/site.go) — Added protected path check in ServeHTTP
- [server/api.go](server/api.go) — Fixed `enforceAuth()` role comparison, added `req.user.role`
- [server/api_test.go](server/api_test.go) — Added role enforcement tests
- [server/handler.go](server/handler.go) — Added `role` to `basil.auth.user` context

### Key Implementation Details
1. **Role enforcement fixed**: `enforceAuth()` now compares `user.Role` against `meta.Roles`
2. **`req.user.role` populated**: Both API and page handlers now have access to user role
3. **Protected path matching**: Prefix-based matching (e.g., `/dashboard` matches `/dashboard/users`)
4. **Mixed config format**: `protected_paths` supports both strings and objects with roles
5. **Login redirect**: Unauthenticated HTML requests redirect to `login_path` with `?next=` param
6. **API responses**: API requests get 401/403 JSON instead of redirects

## Related
- BACKLOG.md: "Auth integration in site mode" — this feature addresses that item
- FEAT-004: Passkey authentication (prerequisite)
- FEAT-049: Sessions (prerequisite)
