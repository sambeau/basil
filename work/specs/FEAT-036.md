---
id: FEAT-036
title: "CLI User Management"
status: implemented
priority: high
created: 2025-12-07
author: "@sambeau"
---

# FEAT-036: CLI User Management

## Summary

Add CLI commands for managing users and API keys from the terminal. This enables bootstrap workflows (creating the first admin user) and headless administration without requiring web UI access.

```bash
# Bootstrap a new site
basil users create --name "Admin" --email "admin@example.com" --role admin
basil apikey create --user usr_abc123 --name "MacBook Git"
```

## User Story

As a **site administrator**, I want to **create users and API keys from the command line** so that I can **bootstrap a new site and manage access without needing web UI or passkey registration**.

## Acceptance Criteria

- [ ] `basil users create` creates a user record (without passkey)
- [ ] `basil users list` shows all users with roles
- [ ] `basil users set-role` changes a user's role
- [ ] `basil users update` changes name/email
- [ ] `basil users delete` removes a user and their credentials
- [ ] `basil users delete` prevents deleting the last admin
- [ ] `basil apikey create` generates an API key for a user
- [ ] `basil apikey list` shows a user's keys (prefix only)
- [ ] `basil apikey revoke` removes an API key
- [ ] Users table has a `role` column (admin/editor)
- [ ] API keys table exists per FEAT-004 Phase 2 schema
- [ ] CLI-created users can later register a passkey via web UI

## Design Decisions

### CLI for Bootstrap, Web for Self-Service

The CLI is for administrators with terminal access. Regular users manage their own API keys via the web components (`<ApiKeyList/>`, `<ApiKeyCreate/>` from FEAT-004 Phase 2).

| Task | CLI | Web UI |
|------|-----|--------|
| Create first admin | ✅ | ❌ (chicken-and-egg) |
| Create additional users | ✅ | Future (invite flow) |
| Change roles | ✅ | Future (admin panel) |
| Create own API key | ❌ | ✅ `<ApiKeyCreate/>` |
| Revoke own API key | ❌ | ✅ `<ApiKeyList/>` |
| Revoke any API key | ✅ | Future (admin panel) |

### Roles: Two-Tier

| Role | Can Clone/Push (Git) | Can Manage Users |
|------|----------------------|------------------|
| `editor` | ✅ | ❌ |
| `admin` | ✅ | ✅ |

**Default role:**
- First user created: `admin` (automatically, cannot be overridden)
- Subsequent users: `editor`

**Why no `viewer` role?** If someone has Git access, they should be able to contribute. Read-only Git access has limited utility for Basil's use case.

### User IDs

Format: `usr_<32 hex chars>` (e.g., `usr_a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4`)

This matches the existing auth system in FEAT-004. The auth system uses its own ID generation (`prefix_` + 16 random bytes as hex) separate from `std/id`, which is for application-level data.

### API Key IDs

Format: `key_<32 hex chars>` (e.g., `key_f1e2d3c4b5a6f1e2d3c4b5a6f1e2d3c4`)

### API Key Scopes (v1: None)

In v1, API keys inherit the user's role — no per-key scopes. The `--name` parameter is just a label.

### Passkey Registration for Existing Users

CLI-created users need to register a passkey to access the web UI. The existing registration flow will be modified to detect users without passkeys:

1. User visits `/register` and enters their email
2. System finds existing user with that email
3. System checks if user has any passkeys
4. If no passkeys → proceed with passkey registration for existing user
5. If has passkeys → error "Email already registered" (existing behavior)

This allows the same email for Git (API key) and web (passkey) access.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Dependencies

- **Depends on:** FEAT-004 Phase 2 schema (api_keys table)
- **Blocks:** FEAT-035 (Git over HTTPS)

### Affected Components

| File | Changes |
|------|---------|
| `cmd/basil/main.go` | Add `users` and `apikey` subcommands |
| `cmd/basil/users.go` | New — user management commands |
| `cmd/basil/apikey.go` | New — API key management commands |
| `auth/database.go` | Add `role` column to users, create `api_keys` table, enable WAL mode |
| `auth/apikeys.go` | New — API key CRUD functions |
| `auth/users.go` | New — user CRUD functions (CLI-facing) |
| `auth/handlers.go` | Modify registration to allow passkey for existing user without passkey |
| `auth/webauthn.go` | Add `BeginRegistrationForExisting()` method |

### Database Concurrency

The CLI must work **while the server is running**. This requires SQLite WAL mode and a busy timeout:

```go
// In auth/database.go OpenDB()
db.Exec("PRAGMA journal_mode=WAL")   // Allow concurrent readers + writer
db.Exec("PRAGMA busy_timeout=5000")  // Wait up to 5s for locks
```

**Why this matters:**
- Without WAL, SQLite uses rollback journaling which blocks readers during writes
- Without busy timeout, concurrent access fails immediately with "database is locked"
- With both: CLI can create users while server handles requests — no restart needed

### Database Changes

#### Add `role` column to `users`

```sql
ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'editor';
```

Valid values: `admin`, `editor`

#### Create `api_keys` table

From FEAT-004 Phase 2 design:

```sql
CREATE TABLE api_keys (
  id TEXT PRIMARY KEY,                -- "key_xyz789"
  user_id TEXT NOT NULL REFERENCES users(id),
  name TEXT NOT NULL,                 -- "MacBook Git"
  key_hash TEXT NOT NULL,             -- bcrypt hash
  key_prefix TEXT NOT NULL,           -- "bsl_...k2m9" for display
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  last_used_at TIMESTAMP,
  expires_at TIMESTAMP                -- Optional expiry
);

CREATE INDEX idx_api_keys_user ON api_keys(user_id);
```

### CLI Commands

#### `basil users create`

```bash
basil users create --name "Sam Phillips" --email "sam@example.com" [--role admin]

# Output:
✓ Created user usr_abc123def456
```

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--name` | Yes | — | Display name |
| `--email` | No | — | Email address (not unique — same person can have multiple accounts) |
| `--role` | No | see below | One of: admin, editor |

**Role default:** First user is always `admin` (flag ignored). Subsequent users default to `editor`.

#### `basil users list`

```bash
basil users list

# Output:
ID                  NAME           EMAIL                ROLE     CREATED
usr_abc123def456    Sam Phillips   sam@example.com      admin    2025-12-07
usr_def456ghi789    Jane Doe       jane@example.com     editor   2025-12-07
```

#### `basil users set-role`

```bash
basil users set-role usr_abc123def456 editor

# Output:
✓ Set role for Sam Phillips to editor
```

#### `basil users update`

```bash
basil users update usr_abc123def456 --name "Samuel Phillips" --email "samuel@example.com"

# Output:
✓ Updated user usr_abc123def456
```

| Flag | Required | Description |
|------|----------|-------------|
| `--name` | No | New display name |
| `--email` | No | New email address |

At least one of `--name` or `--email` must be provided.

#### `basil users delete`

```bash
basil users delete usr_abc123def456

# Output:
⚠ This will delete user Sam Phillips and all their credentials.
  Continue? [y/N] y
✓ Deleted user usr_abc123def456
```

Use `--force` to skip confirmation.

#### `basil apikey create`

```bash
basil apikey create --user usr_abc123def456 --name "MacBook Git"

# Output:
✓ Created API key: bsl_live_a8f3k2m9x7p2q1w5e8r4t6y9...
  (save this now — it won't be shown again)
```

| Flag | Required | Description |
|------|----------|-------------|
| `--user` | Yes | User ID to create key for |
| `--name` | Yes | Label for the key |

#### `basil apikey list`

```bash
basil apikey list --user usr_abc123def456

# Output:
ID              NAME           PREFIX          CREATED       LAST USED
key_xyz789      MacBook Git    bsl_...k2m9     2025-12-07    2025-12-07
key_abc123      CI Server      bsl_...p2q1     2025-12-06    never
```

#### `basil apikey revoke`

```bash
basil apikey revoke key_xyz789

# Output:
✓ Revoked API key "MacBook Git"
```

### Key Generation

```go
func GenerateAPIKey() (plaintext string, hash string, prefix string) {
    // Generate 32 random bytes
    random := make([]byte, 32)
    crypto/rand.Read(random)
    
    // Format: bsl_live_<base62 encoded>
    plaintext = "bsl_live_" + base62Encode(random)
    
    // Store only the hash
    hash = bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
    
    // Prefix for display: first 4 + last 4 chars
    prefix = plaintext[:12] + "..." + plaintext[len(plaintext)-4:]
    
    return plaintext, hash, prefix
}
```

### Key Validation

```go
func ValidateAPIKey(key string) (*User, error) {
    // Key format check
    if !strings.HasPrefix(key, "bsl_live_") {
        return nil, ErrInvalidKeyFormat
    }
    
    // Look up all keys and check hash (bcrypt doesn't allow direct lookup)
    // This is O(n) but API keys per site should be small
    keys, _ := db.GetAllAPIKeys()
    for _, k := range keys {
        if bcrypt.CompareHashAndPassword(k.Hash, key) == nil {
            // Update last_used_at
            db.UpdateAPIKeyLastUsed(k.ID)
            return db.GetUser(k.UserID)
        }
    }
    
    return nil, ErrInvalidKey
}
```

**Performance note:** For sites with many API keys, consider adding a key_prefix index for initial filtering before bcrypt comparison.

### Passkey Registration for Existing Users

Modify `auth/handlers.go` `BeginRegisterHandler`:

```go
// Check if email already exists
if req.Email != "" {
    existing, _ := h.db.GetUserByEmail(req.Email)
    if existing != nil {
        // Check if they have any passkeys
        creds, _ := h.db.GetCredentialsForUser(existing.ID)
        if len(creds) == 0 {
            // User exists but has no passkey — allow registration
            options, challengeID, err := h.webauthn.BeginRegistrationForExisting(existing)
            if err != nil {
                jsonError(w, "Failed to start registration", http.StatusInternalServerError)
                return
            }
            jsonResponse(w, map[string]any{
                "options":      options,
                "challenge_id": challengeID,
            })
            return
        }
        // Has passkeys — reject
        jsonError(w, "Email already registered", http.StatusConflict)
        return
    }
}
// ... continue with new user registration
```

Add `auth/webauthn.go` `BeginRegistrationForExisting`:

```go
// BeginRegistrationForExisting starts passkey registration for an existing user.
func (m *WebAuthnManager) BeginRegistrationForExisting(user *User) (*protocol.CredentialCreation, string, error) {
    wanUser := &webAuthnUser{
        id:          []byte(user.ID),
        name:        user.Email,
        displayName: user.Name,
        credentials: nil, // No existing credentials
    }
    if user.Email == "" {
        wanUser.name = user.Name
    }
    
    options, sessionData, err := m.webauthn.BeginRegistration(wanUser)
    if err != nil {
        return nil, "", fmt.Errorf("beginning registration: %w", err)
    }
    
    challengeID := generateID("chal")
    m.mu.Lock()
    m.challenges[challengeID] = &challengeData{
        sessionData:  sessionData,
        user:         wanUser,
        existingUser: user, // Flag that this is an existing user
        expiresAt:    time.Now().Add(5 * time.Minute),
    }
    m.mu.Unlock()
    
    return options, challengeID, nil
}
```

Modify `FinishRegistration` to skip user creation if `existingUser` is set.

### Edge Cases

1. **Duplicate email** — Return error: "A user with that email already exists"
2. **Invalid role** — Return error: "Invalid role. Use: admin, editor"
3. **Delete last admin** — Return error: "Cannot delete the last admin user"
4. **User not found** — Return error: "User not found: usr_xxx"
5. **Create key for nonexistent user** — Return error: "User not found: usr_xxx"

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | User not found |
| 3 | Validation error (duplicate email, invalid role) |

## Implementation Notes

*Added during/after implementation*

## Related

- FEAT-004: Authentication (Phase 2 API Keys web components)
- FEAT-035: Git over HTTPS (uses API keys for auth)
- Design: `work/design/remote-workflow-design.md`
