# Sessions & State Design

**Date:** 2025-12-07  
**Status:** Draft  
**Purpose:** Design session management and state features for Basil applications.

## Overview

Web applications need to maintain state across HTTP requests. There are several mechanisms:

| Mechanism | Storage | Lifetime | Size Limit | Use Case |
|-----------|---------|----------|------------|----------|
| **Cookies** | Client | Configurable | ~4KB | Small data, preferences |
| **Sessions** | Server | Until expiry/logout | No limit | User data, cart, wizard state |
| **Database** | Server | Permanent | No limit | Persistent user data |
| **URL params** | Client | Single request | ~2KB | Filters, pagination |

Basil currently has:
- ✅ Database (via `basil.sqlite`)
- ⏳ Cookies (FEAT-043, pending)
- ✅ URL params (via `basil.http.request.query`)
- ❌ General sessions (this document)

The auth system has internal sessions, but they're not exposed for general use.

---

## What Are Sessions?

Sessions provide server-side storage keyed by a client identifier (usually a cookie). The flow:

```
1. First request:
   Browser: GET /cart
   Server:  Set-Cookie: session_id=abc123
            Body: {cart: []}

2. Subsequent requests:
   Browser: GET /cart
            Cookie: session_id=abc123
   Server:  (looks up abc123 in session store)
            Body: {cart: [{item: "Widget", qty: 2}]}
```

The browser only stores a session ID. All data lives on the server.

### Sessions vs Cookies

| Aspect | Cookies | Sessions |
|--------|---------|----------|
| Storage | Client browser | Server |
| Size | ~4KB max | Unlimited |
| Security | Visible to client | Hidden from client |
| Tampering | Client can modify | Server controls |
| Scaling | Stateless | Requires shared store |

**Use cookies for:** Preferences, "remember me" tokens, non-sensitive flags  
**Use sessions for:** Shopping carts, form wizards, sensitive temp data

---

## What Are Flash Messages?

Flash messages are **one-time session values** that are automatically cleared after being read. They're used for feedback after redirects.

### The Problem They Solve

```
POST /items (create new item)
  → 302 Redirect to /items
GET /items
  → 200 OK, shows list

How do you show "Item created successfully" on the list page?
```

**Without flash:** Pass message in URL (`/items?msg=created`) — ugly, refreshable, bookmarkable  
**With flash:** Store in session, show once, auto-clear

### How Flash Works

```parsley
// In POST handler (before redirect)
flash("success", "Item created successfully")
redirect("/items")

// In GET handler (after redirect)
let messages = flash()  // Returns and clears: {success: "Item created..."}
if (messages.success) {
    <div class="alert success">{messages.success}</div>
}
// Next request: flash() returns {}
```

### Flash Categories

Common categories (by convention):
- `success` — Green, positive feedback
- `error` — Red, something went wrong
- `warning` — Yellow, caution
- `info` — Blue, neutral information

```parsley
flash("success", "Profile updated")
flash("error", "Invalid email address")
flash("warning", "Your subscription expires soon")
flash("info", "New features available")
```

---

## Session Store Options

### Option A: Cookie-Based Sessions (Encrypted)

Store session data in the cookie itself, encrypted.

```
Set-Cookie: session=<encrypted JSON>
```

**How it works:**
1. Server encrypts session data with secret key
2. Stores encrypted blob in cookie
3. On request, decrypts and provides to handler
4. On response, re-encrypts if changed

**Pros:**
- No server storage needed
- Scales infinitely (stateless)
- No cleanup/expiry management
- Simple deployment

**Cons:**
- Size limited (~4KB)
- All data sent on every request
- Changing secret invalidates all sessions
- Can't invalidate individual sessions server-side

**Best for:** Small session data, stateless deployments

**Example frameworks:** Rails (default), Phoenix, Express (cookie-session)

### Option B: Database Sessions

Store sessions in SQLite (same DB or separate).

```sql
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    data TEXT,  -- JSON blob
    created_at DATETIME,
    expires_at DATETIME,
    user_id TEXT  -- Optional, for auth integration
);
```

**Pros:**
- Unlimited size
- Can query sessions (admin, analytics)
- Can invalidate server-side ("log out everywhere")
- Can associate with users
- Survives server restart

**Cons:**
- Database load on every request
- Needs cleanup job for expired sessions
- Scaling requires shared database

**Best for:** Large session data, need to query sessions, enterprise features

**Example frameworks:** Rails (ActiveRecord store), Django

### Option C: In-Memory Sessions

Store sessions in server memory (Go map).

**Pros:**
- Fastest access
- Simple implementation

**Cons:**
- Lost on server restart
- Doesn't scale to multiple servers
- Memory grows with users

**Best for:** Development, single-server deployments

**Example frameworks:** Express (default memory store)

### Option D: External Store (Redis, etc.)

Store sessions in Redis or similar.

**Pros:**
- Fast, scalable
- Survives server restart
- Built-in expiry (TTL)
- Shared across servers

**Cons:**
- External dependency
- More complex deployment

**Best for:** High-scale production deployments

---

## Recommendation for Basil

Given Basil's philosophy (single binary, batteries included, SQLite-first):

### Primary: Encrypted Cookie Sessions

Default to encrypted cookie sessions. Zero configuration, stateless, works immediately.

```yaml
# basil.yaml - optional config
session:
  secret: ${SESSION_SECRET}  # Auto-generated if not set
  maxAge: 24h                # Session lifetime
  name: _session             # Cookie name
```

```parsley
// In handlers
session.cart = [{item: "Widget", qty: 2}]
let cart = session.cart ?? []

// Flash is just session with auto-clear
flash("success", "Item added to cart")
```

### Secondary: SQLite Sessions (Opt-in)

For apps that need larger sessions or session management:

```yaml
session:
  store: sqlite              # Instead of cookie (default)
  table: sessions            # Table name
  cleanup: 1h                # Run cleanup every hour
```

Same API, different backend:
```parsley
// Works the same whether cookie or SQLite
session.largeData = {...}    // Now can be larger than 4KB
```

### No External Stores (Initially)

Redis/Memcached can be added later if demand exists. Most Basil apps won't need it.

---

## Proposed API

### Session Object

```parsley
// basil.session is a dict-like object
// Changes are automatically persisted

// Set values
basil.session.userId = "123"
basil.session.cart = [{item: "Widget", qty: 2}]

// Get values
let userId = basil.session.userId
let cart = basil.session.cart ?? []

// Delete values
basil.session.cart = null  // or delete(basil.session, "cart")

// Clear entire session
clear(basil.session)
```

### Flash Messages

```parsley
// Set flash (writes to session._flash)
flash("success", "Item created")
flash("error", "Invalid input")

// Get and clear flash
let messages = flash()  // {success: "...", error: "..."}

// Check for specific flash
if (let msg = flash("success")) {
    <div class="success">{msg}</div>
}
```

### Session Configuration

```yaml
# basil.yaml
session:
  # Storage backend: "cookie" (default) or "sqlite"
  store: cookie
  
  # Secret key for encryption (auto-generated if not set)
  # IMPORTANT: Set this in production for session persistence across restarts
  secret: ${SESSION_SECRET}
  
  # Session lifetime (default: 24h)
  maxAge: 24h
  
  # Cookie name (default: _session)
  name: _session
  
  # Cookie settings (inherit secure defaults)
  secure: true      # HTTPS only (auto in production)
  httpOnly: true    # No JavaScript access
  sameSite: Lax     # CSRF protection

# For SQLite sessions
session:
  store: sqlite
  table: sessions   # Table name (auto-created)
  cleanup: 1h       # Cleanup interval for expired sessions
```

---

## Implementation Design

### Cookie Sessions

```go
// Session data structure
type Session struct {
    Data      map[string]interface{} `json:"d"`
    Flash     map[string]string      `json:"f,omitempty"`
    CreatedAt time.Time              `json:"c"`
    ExpiresAt time.Time              `json:"e"`
}

// Encryption: AES-256-GCM
// Format: base64(nonce + ciphertext + tag)

// Flow:
// 1. On request: decrypt cookie → Session struct → basil.session dict
// 2. Handler runs, may modify basil.session
// 3. On response: if changed, encrypt → Set-Cookie header
```

### SQLite Sessions

```sql
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,           -- Random session ID
    data TEXT NOT NULL,            -- JSON blob
    flash TEXT,                    -- JSON blob for flash messages
    created_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL,
    user_id TEXT,                  -- Optional: link to auth user
    
    INDEX idx_sessions_expires (expires_at),
    INDEX idx_sessions_user (user_id)
);
```

```go
// Cleanup job runs periodically
DELETE FROM sessions WHERE expires_at < NOW()
```

### Flash Implementation

Flash is stored in the session with special handling:

```go
// On flash() read:
// 1. Get session._flash
// 2. Delete session._flash
// 3. Return the values

// On flash("key", "value") write:
// 1. Get or create session._flash
// 2. Set session._flash[key] = value
```

---

## Edge Cases

### Session Size Limits

**Cookie sessions:** ~4KB total after encryption. If exceeded:
- Option 1: Error with clear message
- Option 2: Silently truncate (bad, data loss)
- Option 3: Auto-switch to SQLite (complex)

**Recommendation:** Error with message suggesting SQLite store.

```
Error: Session data exceeds cookie size limit (4KB).
Consider using SQLite session store:
  session:
    store: sqlite
```

### Session Fixation

Attack where attacker gives victim a known session ID.

**Prevention:** Regenerate session ID on login.

```parsley
// After successful login
regenerateSession()  // New ID, keeps data
```

Or automatically regenerate when `basil.auth.user` changes.

### Concurrent Requests

Two requests modify session simultaneously → race condition.

**Cookie sessions:** Last write wins (acceptable for most cases)  
**SQLite sessions:** Could use transactions (complexity vs. benefit)

For most Basil apps, last-write-wins is fine.

### Auth Integration

Link sessions to authenticated users:

```parsley
// After login, session is associated with user
basil.session.userId = basil.auth.user.id

// Or automatically by auth system
// basil.session internally tracks user association
```

For SQLite sessions, enables "log out everywhere":
```sql
DELETE FROM sessions WHERE user_id = ?
```

---

## Comparison with Other Frameworks

### Rails
- Cookie sessions by default (encrypted, signed)
- Can switch to ActiveRecord, Redis, Memcached
- Flash built-in: `flash[:notice] = "..."`
- Session accessed via `session[:key]`

### Express
- Memory store by default (bad for production)
- Many stores available: Redis, MongoDB, PostgreSQL
- Flash via `connect-flash` middleware
- Session accessed via `req.session.key`

### Django
- Database sessions by default
- Can use cookies, cache, file
- Flash via "messages" framework
- Session accessed via `request.session['key']`

### Phoenix/Elixir
- Cookie sessions by default (encrypted)
- Can use ETS (in-memory), Redis
- Flash built-in: `put_flash(conn, :info, "...")`
- Session accessed via `get_session(conn, :key)`

### Proposed Basil
- Cookie sessions by default (encrypted) ← Like Rails, Phoenix
- Can switch to SQLite ← Consistent with Basil's SQLite-first approach
- Flash built-in ← Like Rails, Phoenix
- Session accessed via `basil.session.key` ← Consistent with other basil.* APIs

---

## Summary

### Design Decisions

1. **Cookie sessions by default** — Zero config, stateless, scales
2. **SQLite sessions opt-in** — For larger data or session management
3. **Flash built-in** — Common need, simple to implement
4. **Consistent API** — `basil.session.*` matches other basil namespaces
5. **Secure defaults** — Encrypted, HttpOnly, Secure, SameSite

### Implementation Order

1. **Cookie sessions** (FEAT-048)
   - Encryption/decryption
   - `basil.session` integration
   - Basic flash support
   
2. **SQLite sessions** (FEAT-049)
   - Table schema
   - Cleanup job
   - Same API, different backend

3. **Session management** (future)
   - "Log out everywhere"
   - Session listing in admin
   - Session analytics

### Spec Summary

| Feature | Priority | Complexity | Depends On |
|---------|----------|------------|------------|
| Cookie Sessions | High | Medium | Cookies (FEAT-043) |
| Flash Messages | High | Low | Sessions |
| SQLite Sessions | Medium | Medium | Sessions, Database |
| Session Regeneration | Medium | Low | Sessions |
| "Log Out Everywhere" | Low | Low | SQLite Sessions |
