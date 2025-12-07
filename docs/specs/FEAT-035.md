---
id: FEAT-035
title: "Git over HTTPS"
status: draft
priority: medium
created: 2025-12-07
author: "@sambeau"
---

# FEAT-035: Git over HTTPS

## Summary

Basil will serve Git repositories over HTTPS, allowing developers to clone, edit locally, and push changes using standard Git commands. This "batteries included" approach requires no external Git hosting and leverages Basil's existing HTTPS infrastructure.

```bash
# The developer experience
git clone https://user@mysite.example.com/.git mysite
cd mysite
# ... edit locally ...
git push origin main
# Site auto-reloads
```

## User Story

As a **Basil site developer**, I want to **clone my site via Git, edit locally, and push changes** so that I can **use familiar tools (my editor, Git) without manually syncing files or needing external Git hosting**.

## Acceptance Criteria

- [ ] `git clone https://user@site.example.com/.git` works with HTTP Basic Auth
- [ ] `git push` deploys changes and triggers live reload
- [ ] Authentication uses API keys (from FEAT-004 Phase 2)
- [ ] Only users with `editor` or `admin` role can push
- [ ] Any authenticated user can clone/pull
- [ ] Git server can be disabled via config
- [ ] Development mode supports unauthenticated access on localhost
- [ ] CLI commands exist for bootstrap: `basil users create`, `basil apikey create`

## Design Decisions

### HTTP Basic Auth with API Keys

Git over HTTPS uses HTTP Basic Auth — the universal standard supported by all Git clients, editors, and credential managers. The password field contains an API key (not a user password).

**Why API keys, not passkeys:**
- Passkeys require browser interaction — Git can't do WebAuthn
- API keys are revocable without changing main credentials
- This is how GitHub/GitLab work (PAT as password)

### `go-git-http` Library

We'll use `github.com/AaronO/go-git-http`, which provides:
- Ready-made `http.Handler`
- Built-in middleware hooks for authentication
- Smart HTTP protocol endpoints
- Apache-2.0 license

### API Key Scopes (v1: None)

In v1, all API keys have equal capabilities — they inherit the user's role (admin/editor). The `--name` parameter when creating keys is just a label for the user.

**Future consideration:** Scoped keys (`--scope git`) could be added later.

### Route: `/.git/`

Git endpoints mount at `/.git/`:
- `/.git/info/refs` — Clone/fetch/push handshake
- `/.git/git-upload-pack` — Clone/fetch data
- `/.git/git-receive-pack` — Push data

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Dependencies

- **Depends on:** FEAT-036 (CLI User Management)
- **Optional:** FEAT-004 Phase 2 (self-service key management via web UI)
- **Blocks:** None

### Affected Components

| File | Changes |
|------|---------|
| `server/git.go` | New — Git HTTP handler, auth middleware |
| `server/server.go` | Mount Git handler at `/.git/` |
| `config/config.go` | Add `git:` configuration section |
| `cmd/basil/main.go` | Add `users` and `apikey` subcommands |
| `auth/apikeys.go` | New — API key CRUD (FEAT-004 Phase 2) |
| `auth/database.go` | Add `api_keys` table, `role` column |

### Configuration

```yaml
git:
  enabled: true              # Enable Git server (default: false)
  require_auth: true         # Require authentication (default: true)
```

### CLI Commands (Bootstrap)

```bash
# Create user (no passkey, for CLI bootstrap)
basil users create --name "Admin" --email "admin@example.com" --role admin
# Output: ✓ Created user usr_abc123

# Generate API key
basil apikey create --user usr_abc123 --name "MacBook Git"
# Output: ✓ Created API key: bsl_live_abc123...

# List users
basil users list

# Change role
basil users set-role usr_abc123 editor

# List keys (shows prefix only)
basil apikey list --user usr_abc123

# Revoke key
basil apikey revoke key_xyz789
```

### Authentication Flow

```go
authenticator := auth.Authenticator(func(info auth.AuthInfo) (bool, error) {
    // Validate API key (password field contains the key)
    user, err := basilAuth.ValidateAPIKey(info.Password)
    if err != nil {
        return false, nil
    }
    
    // Check role for push operations
    if info.Push && !user.HasRole("editor", "admin") {
        return false, nil
    }
    
    return true, nil
})
```

### Post-Push Reload

```go
git.EventHandler = func(ev githttp.Event) {
    if ev.Type == githttp.PUSH {
        server.ReloadHandlers()
    }
}
```

### Role Requirements

| Operation | Required Role |
|-----------|---------------|
| Clone/Pull | Any authenticated user |
| Push | `editor` or `admin` |

### Security Constraints

1. HTTPS required when `require_auth: true`
2. Warn if `require_auth: false` with non-localhost bind
3. Rate limit authentication attempts
4. Log all push events with user identity

### Edge Cases

1. **Invalid API key** — Return 401, log attempt
2. **Valid key, wrong role for push** — Return 403
3. **Push with syntax errors** — Accept push, site may error (future: pre-receive validation)
4. **Force push** — Allow (Git default); future: optional rejection
5. **Large files** — No Git LFS support in v1; document size expectations

## Implementation Notes

*Added during/after implementation*

## Related

- Design: `docs/design/remote-workflow-design.md`
- Prerequisite: FEAT-004 Phase 2 (API Keys)
