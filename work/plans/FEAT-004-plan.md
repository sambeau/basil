---
id: PLAN-003
feature: FEAT-004
title: "Implementation Plan for Authentication"
status: draft
created: 2025-12-01
---

# Implementation Plan: FEAT-004 Authentication

## Overview

Implement passkey-based authentication for Basil:
- WebAuthn registration and login
- Session management with secure cookies
- Recovery codes for account recovery
- Parsley components (`<PasskeyRegister/>`, `<PasskeyLogin/>`, `<PasskeyLogout/>`)
- `request.user` object in handlers
- Route protection (`auth: required | optional`)
- CLI user management commands

## Prerequisites

- [x] FEAT-004 spec approved
- [ ] `go-webauthn/webauthn` library evaluated

## Architecture

```
auth/
├── auth.go          # Core types and interfaces
├── database.go      # Auth database operations (.basil-auth.db)
├── session.go       # Session management
├── webauthn.go      # WebAuthn ceremony handlers
├── recovery.go      # Recovery code generation/validation
├── middleware.go    # Auth middleware for routes
└── components.go    # Parsley component expansion
```

## Tasks

### Task 1: Auth Database Schema & Setup
**Files**: `auth/database.go`, `auth/auth.go`
**Estimated effort**: Medium

Create the separate auth database (`.basil-auth.db`) with users, credentials, sessions, and recovery codes tables.

Steps:
1. Define core types: `User`, `Credential`, `Session`, `RecoveryCode`
2. Create `AuthDB` struct with connection management
3. Implement `InitSchema()` to create tables
4. Implement CRUD operations for users
5. Ensure database is separate from app database

Tests:
- Database creation and schema initialization
- User CRUD operations
- Connection isolation (can't access from app db path)

---

### Task 2: Session Management
**Files**: `auth/session.go`
**Estimated effort**: Medium

Implement secure session creation, validation, and cookie handling.

Steps:
1. Implement `CreateSession(userID)` → session token
2. Implement `ValidateSession(token)` → user or nil
3. Implement `DeleteSession(token)` for logout
4. Implement `CleanExpiredSessions()` for maintenance
5. Cookie helpers: `SetSessionCookie()`, `GetSessionFromRequest()`

Cookie settings:
- `__basil_session`
- HttpOnly, Secure (production), SameSite=Lax
- 24h expiry

Tests:
- Session creation and retrieval
- Session expiry
- Cookie parsing
- Invalid/expired session handling

---

### Task 3: Recovery Codes
**Files**: `auth/recovery.go`
**Estimated effort**: Small

Generate and validate one-time recovery codes.

Steps:
1. Implement `GenerateRecoveryCodes(userID, count)` → []string
2. Implement `ValidateRecoveryCode(userID, code)` → bool (burns code on success)
3. Implement `GetRemainingCodeCount(userID)` → int
4. Store hashed codes, not plaintext

Format: 8 codes, format `XXXX-XXXX-XXXX` (alphanumeric, easy to type)

Tests:
- Code generation (correct count, format)
- Code validation (success burns code)
- Code reuse fails
- All codes used

---

### Task 4: WebAuthn Integration
**Files**: `auth/webauthn.go`
**Estimated effort**: Large

Integrate `go-webauthn/webauthn` library for passkey registration and authentication.

Steps:
1. Add dependency: `go get github.com/go-webauthn/webauthn`
2. Implement `User` interface required by library
3. Implement `BeginRegistration(name, email)` → challenge
4. Implement `FinishRegistration(response)` → user + credentials
5. Implement `BeginLogin()` → challenge (discoverable credentials)
6. Implement `FinishLogin(response)` → user

Notes:
- Store challenges in memory (short-lived)
- Use discoverable credentials (no username needed for login)
- Configure proper RP ID and origins

Tests:
- Registration flow (mock WebAuthn responses)
- Login flow (mock WebAuthn responses)
- Challenge expiry
- Invalid response handling

---

### Task 5: Auth API Endpoints
**Files**: `auth/handlers.go`, `server/server.go`
**Estimated effort**: Medium

Add internal `/__auth/*` endpoints for WebAuthn ceremonies.

Endpoints:
| Path | Method | Purpose |
|------|--------|---------|
| `/__auth/register/begin` | POST | Start registration |
| `/__auth/register/finish` | POST | Complete registration |
| `/__auth/login/begin` | POST | Start login |
| `/__auth/login/finish` | POST | Complete login |
| `/__auth/logout` | POST | End session |
| `/__auth/recover` | POST | Use recovery code |

Steps:
1. Create handlers for each endpoint
2. Register routes in server setup
3. Return proper JSON responses with challenges
4. Handle errors gracefully

Tests:
- Each endpoint responds correctly
- Error cases return appropriate status codes
- CORS headers if needed

---

### Task 6: Auth Middleware
**Files**: `auth/middleware.go`, `server/handler.go`
**Estimated effort**: Medium

Add middleware to check authentication and populate `request.user`.

Steps:
1. Implement `AuthMiddleware(required bool)` → http.Handler wrapper
2. Extract session from cookie
3. Validate session, load user
4. Attach user to request context
5. If `required` and no user → 401/redirect
6. If `optional` → continue with nil user

Integration:
- Wrap route handlers based on `auth` config value
- Pass user to Parsley execution context

Tests:
- Required route blocks unauthenticated
- Optional route allows both
- User available in context when authenticated

---

### Task 7: Config Extensions
**Files**: `config/config.go`, `config/load.go`
**Estimated effort**: Small

Add auth configuration options.

```yaml
auth:
  enabled: true
  registration: open  # open | closed
  session_ttl: 24h    # optional, default 24h
```

Steps:
1. Add `AuthConfig` struct
2. Add to main `Config` struct
3. Add defaults
4. Add validation

Tests:
- Config loading with auth section
- Defaults applied when missing
- Validation errors for invalid values

---

### Task 8: Parsley Components
**Files**: `auth/components.go`, `server/handler.go`
**Estimated effort**: Large

Implement `<PasskeyRegister/>`, `<PasskeyLogin/>`, `<PasskeyLogout/>` component expansion.

Steps:
1. Detect component tags during Parsley output processing
2. Expand to semantic HTML + inline JS
3. JS calls `/__auth/*` endpoints
4. Handle success/error states in JS
5. Support all documented attributes

Component attributes:
- `button_text`, `class`, `redirect`
- `name`, `email`, `name_placeholder`, `email_placeholder` (Register only)

Tests:
- Component expansion produces valid HTML
- JS includes correct endpoint URLs
- Attributes map to correct HTML attributes
- Missing required attributes error

---

### Task 9: request.user in Parsley
**Files**: `server/handler.go`
**Estimated effort**: Medium

Pass authenticated user to Parsley context as `request.user`.

Steps:
1. Get user from request context (from middleware)
2. Convert to Parsley dictionary format
3. Add to `request` object passed to Parsley
4. Handle nil user (not authenticated)

Structure:
```parsley
request.user.id       // "usr_abc123"
request.user.name     // "Sam Phillips"
request.user.email    // "sam@example.com" or null
request.user.created  // @2025-12-01T10:00:00
```

Tests:
- Authenticated request has user object
- Unauthenticated request has null user
- All fields populated correctly

---

### Task 10: CLI User Management
**Files**: `main.go` (or new `cmd/` structure)
**Estimated effort**: Medium

Add CLI commands for user management.

Commands:
```bash
basil users list              # List all users
basil users show <id>         # Show user details
basil users delete <id>       # Delete user (with confirmation)
basil users reset <id>        # Generate new recovery codes
```

Steps:
1. Add subcommand parsing to main
2. Implement each command
3. Format output nicely (table for list)
4. Require `--config` to find auth database

Tests:
- Commands work with test database
- Delete requires confirmation (unless --force)
- Reset outputs new codes

---

### Task 11: Example & Documentation
**Files**: `examples/auth/`, `docs/guide/`
**Estimated effort**: Small

Create working example and documentation.

Steps:
1. Create `examples/auth/` with complete auth example
2. Include signup, login, logout, protected pages
3. Update `docs/guide/` with auth documentation
4. Add to FAQ if relevant

Example structure:
```
examples/auth/
├── basil.yaml
├── handlers/
│   ├── home.pars      # Public
│   ├── signup.pars    # Registration page
│   ├── login.pars     # Login page
│   ├── dashboard.pars # Protected page
│   └── logout.pars    # Logout action
└── static/
    └── styles.css
```

---

## Implementation Order

Suggested order (dependencies):

1. **Task 1: Database** — Foundation, everything depends on this
2. **Task 2: Sessions** — Needed for login
3. **Task 3: Recovery** — Can be parallel with WebAuthn
4. **Task 4: WebAuthn** — Core auth logic
5. **Task 7: Config** — Needed before server integration
6. **Task 5: API Endpoints** — Ties WebAuthn to HTTP
7. **Task 6: Middleware** — Route protection
8. **Task 9: request.user** — Parsley integration
9. **Task 8: Components** — User-facing pieces
10. **Task 10: CLI** — Management tools
11. **Task 11: Example** — Proves it works

## Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `go build -o basil .`
- [ ] Linter passes: `golangci-lint run`
- [ ] Example works end-to-end
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-01 | Task 1: Database | ✅ Complete | auth/auth.go, auth/database.go with tests |
| 2025-12-01 | Task 2: Sessions | ✅ Complete | auth/session.go with tests |
| 2025-12-01 | Task 3: Recovery | ✅ Complete | auth/recovery.go with tests |
| 2025-12-01 | Task 4: WebAuthn | ✅ Complete | auth/webauthn.go with tests |
| 2025-12-01 | Task 5: Endpoints | ✅ Complete | auth/handlers.go |
| 2025-12-01 | Task 6: Middleware | ✅ Complete | auth/middleware.go with tests |
| 2025-12-01 | Task 7: Config | ✅ Complete | Added AuthConfig to config/config.go + server integration |
| 2025-12-01 | Task 8: Components | ✅ Complete | auth/components.go with tests |
| 2025-12-01 | Task 9: request.user | ✅ Complete | Added to buildRequestContext in handler.go |
| 2025-12-01 | Task 10: CLI | ✅ Complete | basil users list/show/delete/reset |
| 2025-12-01 | Task 11: Docs | ✅ Complete | examples/auth/, docs/guide/authentication.md |

## Deferred Items

Items identified during planning (already in BACKLOG.md):
- Multiple passkeys per user
- API keys (Phase 2)
- OAuth2/OIDC providers
- SMS recovery (Twilio)
- Email recovery
- Roles/permissions

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| WebAuthn browser support | Low | High | All modern browsers support it; provide clear error for old browsers |
| go-webauthn library issues | Low | Medium | Well-maintained, many dependents; fallback: implement raw WebAuthn |
| Component expansion complexity | Medium | Medium | Start simple, iterate; test with real Parsley scripts |
| Dev mode without HTTPS | Medium | Low | WebAuthn works on localhost; document clearly |
