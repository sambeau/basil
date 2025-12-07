---
id: FEAT-049
title: "Sessions and Flash Messages"
status: draft
priority: high
created: 2025-12-07
author: "@copilot"
---

# FEAT-049: Sessions and Flash Messages

## Summary

Add server-side session storage with encrypted cookie sessions as the default and SQLite sessions as an opt-in alternative. Include built-in flash message support for one-time notifications across redirects. Sessions enable shopping carts, form wizards, and user preferences without database storage.

## User Story

As a developer building interactive web applications, I want to store temporary user data (cart, wizard state, preferences) and display one-time feedback messages after redirects so that I can create stateful experiences without managing database records.

## Acceptance Criteria

### Sessions
- [ ] `basil.session` provides dict-like access to session data
- [ ] Session data persists across requests for the same user
- [ ] Cookie sessions work by default with no configuration
- [ ] Session data is encrypted (AES-256-GCM)
- [ ] SQLite sessions available via config (`store: sqlite`)
- [ ] Session expires after configurable `maxAge` (default: 24h)
- [ ] `clear(basil.session)` clears entire session
- [ ] Session readable immediately after setting (same request)

### Flash Messages
- [ ] `flash(type, message)` stores a flash message
- [ ] `flash()` returns all flash messages and clears them
- [ ] Flash messages survive exactly one redirect
- [ ] Common types: success, error, warning, info

### Security
- [ ] Secret key auto-generated for development
- [ ] Secret key required in production (error if missing)
- [ ] HttpOnly, Secure, SameSite=Lax cookie defaults
- [ ] Session ID regenerated on auth changes (prevent fixation)

## Design Decisions

- **Cookie sessions by default**: Zero config, stateless, scales infinitely. Matches Rails/Phoenix.
- **SQLite opt-in**: For apps needing >4KB session data or session management features.
- **Flash as session feature**: Flash is just session data with auto-clear behavior, not a separate system.
- **Secret handling**: Auto-generate for dev (with log warning), require explicit in production. Prevents accidental insecure deployments.
- **Read-all-clear-all flash**: `flash()` returns entire flash dict and clears it. Simpler than per-key clearing, matches typical usage (display all in layout).

---

## Technical Context

### Session API

```parsley
// Set session values
basil.session.userId = "123"
basil.session.cart = [{item: "Widget", qty: 2}]
basil.session.preferences = {theme: "dark", locale: "en-US"}

// Get session values
let userId = basil.session.userId
let cart = basil.session.cart ?? []

// Delete a value
basil.session.cart = null

// Clear entire session
clear(basil.session)

// Check if key exists
if (basil.session.userId != null) {
    // logged in
}
```

### Flash API

```parsley
// POST handler - set flash before redirect
basil.sqlite <=!=> "INSERT INTO items ..."
flash("success", "Item created successfully")
redirect("/items")

// GET handler - display and clear flash
let messages = flash()  // {success: "Item created successfully"}

// In template
if (messages.success) {
    <div class="alert alert-success">{messages.success}</div>
}
if (messages.error) {
    <div class="alert alert-error">{messages.error}</div>
}

// After displaying, flash is cleared
// Next request: flash() returns {}
```

### Multiple Flash Messages

```parsley
// Can set multiple types
flash("success", "Profile updated")
flash("warning", "Please verify your email")
redirect("/profile")

// Or multiple of same type (last wins)
flash("error", "Invalid email")
flash("error", "Password too short")  // Overwrites previous

// For multiple errors, use array in session
basil.session.errors = ["Invalid email", "Password too short"]
```

### Configuration

```yaml
# basil.yaml

# Minimal (uses defaults)
session:
  secret: ${SESSION_SECRET}

# Full options
session:
  # Storage: "cookie" (default) or "sqlite"
  store: cookie
  
  # Encryption secret (REQUIRED in production)
  # Auto-generated for development with warning
  secret: ${SESSION_SECRET}
  
  # Session lifetime
  maxAge: 24h
  
  # Cookie name
  name: _basil_session
  
  # Cookie flags (secure defaults)
  secure: true      # HTTPS only (auto in production)
  httpOnly: true    # No JavaScript access
  sameSite: Lax     # CSRF protection
  
# SQLite sessions
session:
  store: sqlite
  table: _sessions   # Table name (auto-created)
  cleanup: 1h        # Expired session cleanup interval
```

### Cookie Session Implementation

```go
// Session structure (encrypted in cookie)
type CookieSession struct {
    Data      map[string]interface{} `json:"d"`
    Flash     map[string]string      `json:"f,omitempty"`
    ExpiresAt time.Time              `json:"e"`
}

// Request flow:
// 1. Middleware reads cookie, decrypts → CookieSession
// 2. Expose as basil.session dict to handler
// 3. After handler, if changed, encrypt → Set-Cookie
// 4. Flash is moved to session._flash, cleared on read
```

Encryption: AES-256-GCM with random nonce. Cookie value is `base64(nonce + ciphertext + tag)`.

### SQLite Session Implementation

```sql
CREATE TABLE IF NOT EXISTS _sessions (
    id TEXT PRIMARY KEY,           -- Random 32-byte hex
    data TEXT NOT NULL,            -- JSON blob
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    user_id TEXT,                  -- Links to auth.users if logged in
    
    -- Indexes
    CREATE INDEX idx_sessions_expires ON _sessions(expires_at);
    CREATE INDEX idx_sessions_user ON _sessions(user_id);
);
```

Cleanup job runs at configured interval:
```sql
DELETE FROM _sessions WHERE expires_at < datetime('now');
```

### Secret Key Handling

```go
// On startup:
if config.Session.Secret == "" {
    if config.Mode == "production" {
        log.Fatal("session.secret is required in production mode")
    }
    // Development: generate random secret, warn
    secret = generateRandomKey(32)
    log.Warn("Using auto-generated session secret. Sessions will not persist across restarts.")
}
```

### Session Regeneration

Prevent session fixation attacks by regenerating session ID on privilege changes:

```parsley
// Automatically called by auth system on login/logout
// Or manually:
regenerateSession()  // New session ID, preserves data
```

```go
// Implementation:
// 1. Generate new session ID
// 2. Copy data from old session
// 3. Delete old session (SQLite) or just use new ID (cookie)
// 4. Set new cookie
```

### Affected Components

- `server/session.go` — New file: session middleware, encryption, storage interface
- `server/session_cookie.go` — Cookie session store implementation
- `server/session_sqlite.go` — SQLite session store implementation  
- `server/middleware.go` — Add session middleware to chain
- `server/handler.go` — Expose `basil.session` to handlers
- `pkg/parsley/evaluator/stdlib_basil.go` — Add `flash()` function
- `config/config.go` — Add session configuration

### Dependencies

- **Depends on**: Cookies (FEAT-043) — Need cookie read/write infrastructure
- **Blocks**: None (but enables future features like shopping carts, wizards)

### Edge Cases & Constraints

1. **Cookie size limit (~4KB)** — Error if session exceeds limit with helpful message suggesting SQLite store
2. **No secret in production** — Fatal error on startup (fail-fast)
3. **Concurrent requests** — Last-write-wins for cookie sessions (acceptable)
4. **Flash without redirect** — Flash persists until next `flash()` read (could be same request)
5. **Session without cookies** — Error: sessions require cookies to be enabled
6. **SQLite session table exists** — Check and skip creation if exists
7. **Cleanup job on shutdown** — Graceful shutdown waits for cleanup to complete

### Test Cases

```parsley
// Session basic operations
basil.session.foo = "bar"
assert(basil.session.foo == "bar")

basil.session.foo = null
assert(basil.session.foo == null)

clear(basil.session)
assert(basil.session.keys().length() == 0)

// Session persists across requests (integration test)
// Request 1: basil.session.counter = 1
// Request 2: assert(basil.session.counter == 1)

// Flash basic operations
flash("success", "Hello")
let messages = flash()
assert(messages.success == "Hello")

let empty = flash()
assert(empty.keys().length() == 0)

// Flash survives redirect (integration test)
// Request 1 (POST): flash("success", "Created"), redirect("/list")
// Request 2 (GET /list): assert(flash().success == "Created")
// Request 3 (GET /list): assert(flash().keys().length() == 0)

// Cookie size limit
basil.session.huge = "x" * 10000  // Should error
```

## Implementation Notes

*To be added during implementation*

### Phase 1: Cookie Sessions
1. Session middleware (encrypt/decrypt)
2. `basil.session` dict exposure
3. Secret key handling
4. Basic flash support

### Phase 2: SQLite Sessions  
1. SQLite store implementation
2. Session table auto-creation
3. Cleanup job
4. Store switching via config

### Phase 3: Polish
1. Session regeneration
2. Auth integration (user_id tracking)
3. FlashMessages component (html-components.md)
4. "Log out everywhere" for SQLite sessions

## Related

- **Depends on**: FEAT-043 (Cookies)
- **Design doc**: `docs/design/sessions-state.md`
- **Related**: FEAT-044 (CSRF) — May use session for CSRF tokens
- **Related**: `docs/design/html-components.md` — FlashMessages component
