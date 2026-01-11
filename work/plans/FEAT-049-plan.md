---
id: PLAN-030
feature: FEAT-049
title: "Implementation Plan for Sessions and Flash Messages"
status: complete
created: 2025-12-08
completed: 2025-12-08
---

# Implementation Plan: FEAT-049 Sessions and Flash Messages

## Overview
Add server-side session storage with encrypted cookie sessions (default) and SQLite sessions (opt-in). Include flash message support for one-time notifications across redirects. This enables shopping carts, form wizards, user preferences, and post-redirect feedback.

## Prerequisites
- [x] FEAT-043 (Cookies) implemented — cookie read/write infrastructure exists
- [x] FEAT-044 (CSRF) implemented — can reference for middleware patterns
- [ ] Understand existing `basil.http` context structure in handler.go
- [ ] Review AES-256-GCM encryption patterns

## Phase 1: Cookie Sessions (Core)

### Task 1: Add session configuration
**Files**: `config/config.go`
**Estimated effort**: Small

Steps:
1. Add `SessionConfig` struct with fields:
   - `Store` (string): "cookie" or "sqlite", default "cookie"
   - `Secret` (string): encryption key (from env var)
   - `MaxAge` (duration): session lifetime, default 24h
   - `CookieName` (string): default "_basil_session"
   - `Secure` (bool): HTTPS only, default true in production
   - `HttpOnly` (bool): default true
   - `SameSite` (string): default "Lax"
2. Add `Session SessionConfig` field to main Config
3. Set sensible defaults in DefaultConfig()

Tests:
- Config loads session settings from YAML
- Defaults are applied when not specified
- Environment variable substitution works for secret

---

### Task 2: Create session encryption module
**Files**: `server/session_crypto.go` (new)
**Estimated effort**: Medium

Steps:
1. Create `SessionData` struct:
   ```go
   type SessionData struct {
       Data      map[string]interface{} `json:"d"`
       Flash     map[string]string      `json:"f,omitempty"`
       ExpiresAt time.Time              `json:"e"`
   }
   ```
2. Implement `encryptSession(data *SessionData, secret []byte) (string, error)`
   - JSON encode the data
   - Generate random 12-byte nonce
   - Encrypt with AES-256-GCM
   - Return base64(nonce + ciphertext + tag)
3. Implement `decryptSession(encoded string, secret []byte) (*SessionData, error)`
   - base64 decode
   - Extract nonce (first 12 bytes)
   - Decrypt with AES-256-GCM
   - JSON decode to SessionData
4. Implement `deriveKey(secret string) []byte`
   - SHA-256 hash of secret to get consistent 32-byte key

Tests:
- Encrypt then decrypt returns original data
- Tampered ciphertext fails to decrypt
- Invalid base64 returns error
- Expired session detection

---

### Task 3: Create cookie session store
**Files**: `server/session_cookie.go` (new)
**Estimated effort**: Medium

Steps:
1. Define `SessionStore` interface:
   ```go
   type SessionStore interface {
       Load(r *http.Request) (*SessionData, error)
       Save(w http.ResponseWriter, data *SessionData) error
       Clear(w http.ResponseWriter) error
   }
   ```
2. Implement `CookieSessionStore`:
   - `Load`: Read cookie, decrypt, check expiry
   - `Save`: Encrypt data, set cookie with proper flags
   - `Clear`: Set expired cookie to delete
3. Handle cookie size limit (~4KB):
   - Check serialized size before encryption
   - Return helpful error if too large

Tests:
- Load returns empty session for new user
- Save creates encrypted cookie
- Load after Save returns same data
- Expired session returns empty
- Large session returns size error

---

### Task 4: Create session middleware
**Files**: `server/session.go` (new)
**Estimated effort**: Medium

Steps:
1. Create `SessionMiddleware` struct with store and config
2. Implement middleware handler:
   - Load session from store at request start
   - Store session in request context
   - After handler, save if modified
3. Add session change tracking (dirty flag)
4. Handle secret key validation:
   - If production and no secret, fatal error
   - If dev and no secret, generate random + log warning

Tests:
- Session data available throughout request
- Session persists across requests (same cookie)
- Modified session triggers save
- Unmodified session doesn't set cookie
- Missing secret in production fails startup

---

### Task 5: Expose session to Parsley handlers
**Files**: `server/handler.go`
**Estimated effort**: Medium

Steps:
1. Add session data to `basil` context dict:
   ```go
   "session": sessionDataAsDict,
   ```
2. Implement session dict that tracks changes:
   - Read operations return session values
   - Write operations update session + set dirty flag
3. Support `clear(basil.session)` via special handling
4. Handle null assignment as delete

Tests:
- `basil.session.foo = "bar"` sets value
- `basil.session.foo` reads value
- `basil.session.foo = null` deletes key
- `clear(basil.session)` clears all
- Changes persist to next request

---

### Task 6: Implement flash() function
**Files**: `pkg/parsley/evaluator/stdlib_basil.go`
**Estimated effort**: Medium

Steps:
1. Add `flash` builtin function:
   - `flash(type, message)` — stores flash message in session
   - `flash()` — returns all flash messages as dict, clears them
2. Flash storage in session under `__flash` key
3. Read-all-clear-all behavior on `flash()`

Tests:
- `flash("success", "msg")` stores message
- `flash()` returns stored messages
- Second `flash()` returns empty dict
- Multiple flash types work
- Flash survives redirect (integration)

---

### Task 7: Wire up session middleware in server
**Files**: `server/server.go`
**Estimated effort**: Small

Steps:
1. Create session store based on config
2. Create session middleware
3. Add to middleware chain (after cookies, before handler)
4. Pass session to handler context

Tests:
- Server starts with session middleware
- Session available in handlers
- Session config options respected

---

## Phase 2: SQLite Sessions (Optional)

### Task 8: Create SQLite session store
**Files**: `server/session_sqlite.go` (new)
**Estimated effort**: Medium

Steps:
1. Implement `SQLiteSessionStore`:
   - Auto-create `_sessions` table on init
   - `Load`: Query by session ID from cookie
   - `Save`: Insert/update session row
   - `Clear`: Delete session row
2. Session ID: 32-byte random hex in cookie
3. Link to user_id when authenticated

Tests:
- New session creates database row
- Session loads from database
- Session updates in database
- Clear deletes from database
- Table auto-created if missing

---

### Task 9: Add session cleanup job
**Files**: `server/session_sqlite.go`
**Estimated effort**: Small

Steps:
1. Background goroutine for cleanup
2. Run at configurable interval (default 1h)
3. Delete expired sessions:
   ```sql
   DELETE FROM _sessions WHERE expires_at < datetime('now')
   ```
4. Graceful shutdown waits for cleanup

Tests:
- Expired sessions get deleted
- Cleanup interval respected
- Graceful shutdown works

---

## Phase 3: Security & Polish

### Task 10: Session regeneration
**Files**: `server/session.go`
**Estimated effort**: Small

Steps:
1. Add `regenerateSession()` builtin function
2. Keep session data, generate new ID
3. For SQLite: delete old row, insert new
4. For cookie: just generate new session
5. Auto-call on auth state changes (login/logout)

Tests:
- Regenerate creates new session ID
- Data preserved after regenerate
- Old session ID invalid (SQLite)

---

### Task 11: Auth integration
**Files**: `server/session.go`, `auth/session.go`
**Estimated effort**: Small

Steps:
1. Track `user_id` in session when logged in
2. Auto-regenerate on login/logout
3. SQLite: update user_id column for session queries

Tests:
- Login sets user_id in session
- Logout clears user_id
- Session regenerated on auth change

---

### Task 12: Update documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`, `docs/guide/`
**Estimated effort**: Small

Steps:
1. Add session API to reference.md
2. Add flash() to builtins section
3. Add quick examples to CHEATSHEET.md
4. Add sessions guide to docs/guide/

Tests:
- Documentation builds
- Examples are accurate

---

## Validation Checklist
- [x] All tests pass: `go test ./...`
- [x] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [x] Documentation updated
- [x] Spec FEAT-049 acceptance criteria all checked
- [x] BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-08 | Task 1: Session config | ✅ Complete | Added SessionConfig to config.go |
| 2025-12-08 | Task 2: Encryption | ✅ Complete | AES-256-GCM in session_crypto.go |
| 2025-12-08 | Task 3: Cookie store | ✅ Complete | CookieSessionStore in session.go |
| 2025-12-08 | Task 4: Middleware | ✅ Complete | Integrated into handler.go |
| 2025-12-08 | Task 5: Handler exposure | ✅ Complete | SessionModule via buildBasilContext |
| 2025-12-08 | Task 6: flash() function | ✅ Complete | flash/getFlash/getAllFlash/hasFlash methods |
| 2025-12-08 | Task 7: Server wiring | ✅ Complete | initSessions() in server.go |
| | Task 8: SQLite store | ⏸️ Deferred | Added to BACKLOG.md (Phase 2) |
| | Task 9: Cleanup job | ⏸️ Deferred | Added to BACKLOG.md (Phase 2) |
| 2025-12-08 | Task 10: Regeneration | 🔄 Partial | Method exists, auth integration deferred |
| | Task 11: Auth integration | ⏸️ Deferred | Added to BACKLOG.md (Phase 3) |
| 2025-12-08 | Task 12: Documentation | ✅ Complete | reference.md, CHEATSHEET.md updated |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- Redis session store — for distributed deployments
- Session management UI in DevTools — view/clear sessions
- "Log out everywhere" for SQLite sessions — clear all sessions for user
- Session activity tracking — last_accessed_at column

## Implementation Order
Recommended sequence (Phase 1 is MVP):

**Phase 1 (Cookie Sessions):**
1. Task 1 (Config) — foundation
2. Task 2 (Encryption) — core crypto
3. Task 3 (Cookie store) — storage
4. Task 4 (Middleware) — request lifecycle
5. Task 5 (Handler exposure) — Parsley API
6. Task 6 (flash()) — flash messages
7. Task 7 (Server wiring) — integration

**Phase 2 (SQLite):**
8. Task 8 (SQLite store) — alternative storage
9. Task 9 (Cleanup) — maintenance

**Phase 3 (Polish):**
10. Task 10 (Regeneration) — security hardening
11. Task 11 (Auth integration) — user tracking
12. Task 12 (Docs) — finalize

## Notes

### Cookie Session Format
```
Cookie: _basil_session=<base64(nonce[12] + ciphertext + tag[16])>
```

Decrypted payload (JSON):
```json
{
  "d": {"userId": "123", "cart": [...]},
  "f": {"success": "Item added"},
  "e": "2025-12-09T12:00:00Z"
}
```

### Secret Key
- Development: Auto-generate 32 random bytes, warn in log
- Production: Require `session.secret` or `SESSION_SECRET` env var
- Key derivation: SHA-256(secret) to ensure 32-byte key for AES-256

### Flash Message Flow
1. POST /items (create item)
2. Handler: `flash("success", "Item created")` → stored in session.__flash
3. Handler: `redirect("/items")`
4. Cookie sent with flash in session
5. GET /items
6. Handler: `let msgs = flash()` → returns {success: "..."}, clears __flash
7. Render with messages
8. Next request: `flash()` returns {}
