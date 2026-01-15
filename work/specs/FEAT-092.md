---
id: FEAT-092
title: "Role-Based Access Control"
status: draft
priority: medium
created: 2026-01-15
author: "@human"
---

# FEAT-092: Role-Based Access Control

## Summary
Complete the role-based access control (RBAC) system in Basil. Users already have a `role` field in the database, and the CLI can assign roles, but the auth middleware doesn't enforce role requirements. This feature connects all the pieces so that roles actually restrict access to protected resources.

## Background

### Current State
- **Database**: Users have a `role` column (values: `admin`, `editor`)
- **CLI**: `basil users create --role admin` assigns roles
- **`std/api`**: Provides `adminOnly(fn)` and `roles(["role1", "role2"], fn)` wrappers
- **Schemas**: Can reference roles for row-level permissions (e.g., only admins can see certain records)

### What's Broken
1. **`basil.auth.user.role` not populated** — Handlers can't check roles manually
2. **`enforceAuth()` doesn't check roles** — API handlers wrapped with `adminOnly()` or `roles()` currently deny ALL requests (the role comparison logic is missing)
3. **No role-based protected paths** — `auth.protected_paths` only supports authenticated/not, no role requirements
4. **No route-level `roles:` config** — Routes can require auth but can't require specific roles

## User Story
As a developer, I want to restrict certain pages and API endpoints to users with specific roles so that I can build admin dashboards and other role-restricted features.

## Acceptance Criteria

### Phase 1: Fix Core Role Enforcement
- [ ] `basil.auth.user.role` is populated for all handlers when user is authenticated
- [ ] `enforceAuth()` in API handlers checks `meta.Roles` against `user.Role`
- [ ] `adminOnly(fn)` wrapper correctly restricts to admin users only
- [ ] `roles(["role1", "role2"], fn)` wrapper restricts to listed roles
- [ ] Role check failure returns 403 Forbidden (not 401)

### Phase 2: Route-Level Role Config
- [ ] Routes support `roles:` configuration option
- [ ] `roles:` works with both `auth: required` and inheriting from protected_paths
- [ ] Invalid role in request returns 403 with helpful error

### Phase 3: Role-Based Protected Paths (Optional)
- [ ] Protected paths support object syntax with role requirements
- [ ] String syntax remains for backward compatibility (any authenticated user)

## Design Decisions

### Flat Roles (Not Hierarchical)
**Decision:** Roles are flat—`admin` does not automatically include `editor` permissions.
**Rationale:** Simpler to understand and implement. If a route needs both admins and editors, use `roles: [admin, editor]`. A hierarchy could be added later without breaking changes.

### Standard Role Values
**Decision:** Define standard roles as `admin` and `editor` (matching current database constants).
**Rationale:** Schemas and the std/api module already expect these values. Custom roles could be supported in future but aren't needed for MVP.

| Role | Description |
|------|-------------|
| `admin` | Full access to all features including user management |
| `editor` | Can create/edit content but not manage users or settings |

### 403 vs 401 for Role Failures
**Decision:** Return 403 Forbidden when user is authenticated but lacks required role. Return 401 Unauthorized only when not authenticated.
**Rationale:** Standard HTTP semantics. 403 means "I know who you are, but you can't do this." 401 means "I don't know who you are."

### Role Check Timing
**Decision:** Check role on each request from session data (not database).
**Rationale:** Performance—database lookup on every request is expensive. Role changes take effect on next login. For sensitive operations (like promoting users), handlers can explicitly re-fetch from database.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `server/auth/auth.go` — Role constants already defined
- `server/auth/database.go` — User.Role already stored
- `server/handler.go` — Add `role` to `basil.auth.user` context
- `server/api.go` — Fix `enforceAuth()` to check roles
- `config/config.go` — Add `Roles []string` to Route struct
- `server/server.go` — Apply role checks in middleware chain

### API Handler Role Enforcement

Current `enforceAuth()` in `server/api.go`:
```go
func enforceAuth(ctx context.Context, meta *stdAPIMetadata, logger loggerFunc) (context.Context, bool) {
    // ... existing auth check ...
    
    // MISSING: Role check
    // meta.Roles contains required roles from adminOnly()/roles() wrapper
    // user.Role contains the user's actual role
    // Need to compare them!
}
```

Fix:
```go
// After confirming user is authenticated:
if len(meta.Roles) > 0 {
    userRole := user.Role
    allowed := false
    for _, r := range meta.Roles {
        if r == userRole {
            allowed = true
            break
        }
    }
    if !allowed {
        // Return 403 Forbidden
        return ctx, false // with 403 response
    }
}
```

### Page Handler Role Context

In `server/handler.go`, the `basil.auth.user` dict needs `role`:
```go
userDict := map[string]any{
    "id":    user.ID,
    "name":  user.Name,
    "email": user.Email,
    "role":  user.Role,  // ADD THIS
}
```

### Route Configuration

```yaml
routes:
  - path: /admin/users
    handler: ./handlers/admin-users.pars
    auth: required
    roles: [admin]          # NEW FIELD
```

Config struct addition:
```go
type Route struct {
    Path      string        `yaml:"path"`
    Handler   string        `yaml:"handler"`
    Auth      string        `yaml:"auth"`
    Roles     []string      `yaml:"roles"`   // NEW
    Cache     time.Duration `yaml:"cache"`
    PublicDir string        `yaml:"public_dir"`
    Type      string        `yaml:"type"`
}
```

### Protected Paths with Roles (Phase 3)

Mixed format support:
```yaml
auth:
  protected_paths:
    - /dashboard                # String: any authenticated user
    - path: /admin              # Object: role-restricted
      roles: [admin]
```

Config parsing needs to handle both:
```go
type ProtectedPath struct {
    Path  string   `yaml:"path"`
    Roles []string `yaml:"roles,omitempty"`
}

// AuthConfig.ProtectedPaths can be []string or []ProtectedPath
// Use yaml.Unmarshaler to handle both
```

### Integration with Schemas

Schemas define row-level access control:
```parsley
@schema User {
  id: schema.id(),
  name: schema.string(),
  email: schema.string(),
  role: schema.enum("user", "admin")
}

// Access control in schema queries
let adminUsers = Users.where({role: "admin"})

// In handlers, check user role:
{if basil.auth.user.role == "admin"}
  <a href="/admin">Admin Panel</a>
{/if}
```

The role in `basil.auth.user.role` enables Parsley handlers to:
1. Conditionally render UI elements
2. Filter schema queries based on role
3. Make role-based business logic decisions

### Dependencies
- FEAT-004: Auth system (prerequisite) — implemented
- FEAT-076: Protected paths (prerequisite) — implemented
- FEAT-036: User roles in database — implemented

### Edge Cases & Constraints

1. **No role set** — User with empty role should be denied role-protected routes (fail secure)
2. **Unknown role** — Non-standard role value (typo in config) should fail with clear error at startup
3. **Multiple roles per user** — Not supported in MVP. User has exactly one role. Future: `roles` array field.
4. **Role in session vs database** — Session stores role at login time. Role changes require re-login. Document this.
5. **API vs HTML response** — 403 for API returns JSON error. 403 for HTML could redirect to error page or show inline error.

## Test Cases

### Phase 1: Core Role Enforcement
1. `adminOnly()` wrapper with admin user → 200 OK
2. `adminOnly()` wrapper with editor user → 403 Forbidden
3. `roles(["editor", "admin"])` with editor user → 200 OK
4. `roles(["editor", "admin"])` with unknown role → 403 Forbidden
5. Unauthenticated request to role-protected route → 401 Unauthorized
6. `basil.auth.user.role` equals user's database role

### Phase 2: Route Config
7. Route with `roles: [admin]` + admin user → 200 OK
8. Route with `roles: [admin]` + editor user → 403 Forbidden
9. Route with `auth: required` but no `roles` → any authenticated user allowed

### Phase 3: Protected Paths
10. Protected path `{path: /admin, roles: [admin]}` + admin user → 200 OK
11. Protected path `{path: /admin, roles: [admin]}` + editor user → 403 Forbidden
12. String protected path `/dashboard` → any authenticated user allowed

## Example Usage

### API Handler with Role Restriction
```parsley
// handlers/api/users.pars
import @std/api

@api export let handlers = {
  // Only admins can list all users
  GET: api.adminOnly(fn(req) {
    let users = Users.all()
    {status: 200, body: users}
  }),
  
  // Only admins can create users
  POST: api.adminOnly(fn(req) {
    let user = Users.create(req.body)
    {status: 201, body: user}
  })
}
```

### Page Handler with Role Check
```parsley
// handlers/admin.pars
{if !basil.auth.user}
  <Redirect to="/login"/>
{else if basil.auth.user.role != "admin"}
  <h1>Access Denied</h1>
  <p>You need admin privileges to view this page.</p>
{else}
  <h1>Admin Dashboard</h1>
  // ... admin content ...
{/if}
```

### Config-Based Role Protection
```yaml
auth:
  enabled: true
  protected_paths:
    - /dashboard
    - path: /admin
      roles: [admin]

routes:
  - path: /admin/settings
    handler: ./handlers/admin-settings.pars
    roles: [admin]   # Redundant with protected_path but explicit
```

## Implementation Notes
*To be added during implementation*

## Related
- FEAT-004: Passkey authentication
- FEAT-076: Protected paths (includes partial role discussion)
- FEAT-036: User roles database schema
- BACKLOG #26: "Roles/permissions"
- `server/auth/auth.go`: Role constants (`RoleAdmin`, `RoleEditor`)
- `std/api`: `adminOnly()`, `roles()` wrappers
