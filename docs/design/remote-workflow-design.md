# Design Document: Remote Workflow (Git over HTTPS)

**Status:** Draft  
**Date:** December 2025  
**Related:** Developer Experience, Deployment

---

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

---

## Problem Statement

### The Developer Wants
- Access to edit Parsley scripts and public files
- A local environment that mirrors production
- Simple deployment when changes are ready
- Familiar tools (Git, their preferred editor)

### The Admin Wants
- Developers working anywhere *except* the live site
- Protection from incomplete/broken code reaching production
- Audit trail of who changed what
- No dependency on external services

### The Reality
- Developer and admin are often the same person
- Target users know Git (clone-edit-push workflow)
- External Git hosting (GitHub/GitLab) adds complexity for simple sites
- Current state: no built-in workflow—developers manually sync files

---

## Solution: Built-in Git Server over HTTPS

Basil will embed a Git HTTP server using the `go-git-http` library. This provides:

- **Clone/pull** — Developers get a full copy of the site
- **Push** — Changes deploy automatically with live reload
- **History** — Full Git history and rollback built-in
- **Auth** — Integrated with Basil's existing auth system
- **No extra ports** — Runs on the same HTTPS port as the site

### Why Git over HTTPS?

| Approach             | Pros                                            | Cons                                |
| -------------------- | ----------------------------------------------- | ----------------------------------- |
| **Git over HTTPS** ✓ | Uses existing TLS, no SSH server, familiar auth | Requires TLS for security           |
| Git over SSH         | Strong key-based auth                           | Extra daemon, port, key management  |
| SFTP                 | Immediate sync                                  | Encourages editing live, no history |
| Webhooks             | Low effort                                      | Requires external Git hosting       |
| Rsync                | Efficient sync                                  | No history, single-developer        |

Git over HTTPS best fits the "batteries included" philosophy while keeping implementation simple.

---

## Implementation

### Library: `go-git-http`

The `github.com/AaronO/go-git-http` library provides a ready-made `http.Handler`:

```go
// Simple integration
git := githttp.New("/path/to/site")
http.Handle("/.git/", authenticator(git))
```

**Key features:**
- Returns `http.Handler` — plugs directly into Basil's router
- Built-in authentication middleware hooks
- Handles Smart HTTP protocol endpoints automatically
- Apache-2.0 license (compatible)

### Routes

The Git Smart HTTP protocol requires these endpoints:

| Method | Path                                       | Purpose               |
| ------ | ------------------------------------------ | --------------------- |
| GET    | `/.git/info/refs?service=git-upload-pack`  | Clone/fetch handshake |
| POST   | `/.git/git-upload-pack`                    | Clone/fetch data      |
| GET    | `/.git/info/refs?service=git-receive-pack` | Push handshake        |
| POST   | `/.git/git-receive-pack`                   | Push data             |

### Authentication

Git over HTTPS uses **HTTP Basic Auth** — the standard that every Git client, editor, and credential manager expects. This means Basil's Git server will work out-of-the-box with VS Code, JetBrains IDEs, command-line Git, and any other Git tooling.

#### How It Works

```
User runs: git clone https://sam@mysite.example.com/.git

1. Git prompts for password (or checks OS credential helper)
2. Git sends: Authorization: Basic base64(username:password)
3. Basil validates credentials
4. Git caches in OS keychain — user never types again
```

#### Credential Storage by Platform

Git delegates credential storage to the operating system:

| Platform    | Credential Helper      | Storage                        |
| ----------- | ---------------------- | ------------------------------ |
| **macOS**   | `osxkeychain`          | macOS Keychain (encrypted)     |
| **Windows** | Git Credential Manager | Windows Credential Store       |
| **Linux**   | `cache` or `store`     | Memory or `~/.git-credentials` |

VS Code and other editors don't handle credentials directly — they shell out to `git`, which uses the OS credential helper. This means **no special integration is needed** for editor support.

#### Supported Credential Types

| Credential   | Username Field | Password Field | Recommended?    |
| ------------ | -------------- | -------------- | --------------- |
| **API Key**  | Basil username | API key        | ✅ **Preferred** |
| **Password** | Basil username | User password  | ⚠️ Fallback     |

**Why API keys are preferred:**
- Revocable without changing main password
- Can scope to Git-only operations (future)
- This is how GitHub/GitLab work — Personal Access Token as password
- Works with every credential helper and editor

#### Editor Compatibility

Because we use standard HTTP Basic Auth, these all "just work":

| Editor/Tool          | How It Works                        |
| -------------------- | ----------------------------------- |
| **VS Code**          | Built-in Git → OS credential helper |
| **JetBrains IDEs**   | Built-in Git → OS credential helper |
| **Sublime Merge**    | OS credential helper                |
| **GitKraken**        | Built-in credential store           |
| **SourceTree**       | OS credential helper                |
| **Command-line Git** | OS credential helper                |
| **GitHub Desktop**   | Works, but designed for GitHub      |

**No VS Code extension needed.** No special OAuth flow. No SSH keys. Just the username/password prompt that Git has always used.

#### User Experience

**First clone (one-time setup):**
```bash
$ git clone https://sam@mysite.example.com/.git
Password for 'https://sam@mysite.example.com': <paste API key>
Cloning into 'mysite'...
```

**All subsequent operations (credential cached):**
```bash
$ git pull    # Just works, no prompt
$ git push    # Just works, no prompt
```

**In VS Code:**
1. Command Palette → "Git: Clone"
2. Enter URL: `https://sam@mysite.example.com/.git`
3. VS Code prompts for password → paste API key
4. Credential manager caches it → never asked again

#### Implementation

The `go-git-http/auth` package provides middleware:

```go
authenticator := auth.Authenticator(func(info auth.AuthInfo) (bool, error) {
    // Try API key first (password field contains the key)
    user, err := basilAuth.ValidateAPIKey(info.Password)
    if err != nil {
        // Fall back to username/password
        user, err = basilAuth.ValidateCredentials(info.Username, info.Password)
    }
    if err != nil {
        return false, nil
    }
    
    // Check role for push operations
    if info.Push && !user.HasRole("editor") {
        return false, nil
    }
    
    return true, nil
})
```

#### Prerequisites: FEAT-004 Phase 2 (API Keys)

Basil's current auth system (FEAT-004) is **passkey-only** — there are no passwords or API keys yet. API keys are designed in FEAT-004 Phase 2 but **not yet implemented**.

**Current auth state:**

| Feature                        | Status                |
| ------------------------------ | --------------------- |
| User accounts                  | ✅ Implemented         |
| Passkey (WebAuthn) credentials | ✅ Implemented         |
| Sessions (cookie-based)        | ✅ Implemented         |
| Recovery codes                 | ✅ Implemented         |
| **API keys**                   | ❌ **Not implemented** |
| **Roles (editor/admin)**       | ❌ **Not implemented** |

**Why this matters for Git:**
- Git needs HTTP Basic Auth (username + password/token)
- Passkeys require browser interaction — they can't work with Git
- We need API keys as the credential Git will use

**FEAT-004 Phase 2 specifies:**

```sql
-- Database schema (designed, not built)
CREATE TABLE api_keys (
  id TEXT PRIMARY KEY,                -- "key_xyz789"
  user_id TEXT NOT NULL REFERENCES users(id),
  name TEXT NOT NULL,                 -- "CI Server" or "Git Access"
  key_hash TEXT NOT NULL,             -- bcrypt hash
  key_prefix TEXT NOT NULL,           -- "bsl_...k2m9" for display
  created_at TIMESTAMP,
  last_used_at TIMESTAMP,
  expires_at TIMESTAMP
);
```

**Key format:** `bsl_live_<random32chars>`

**Components (designed, not built):**
- `<ApiKeyList/>` — shows existing keys with revoke buttons
- `<ApiKeyCreate/>` — generates new key, shown once

**Endpoints (designed, not built):**

| Endpoint               | Method | Purpose          |
| ---------------------- | ------ | ---------------- |
| `/__auth/api-keys`     | GET    | List user's keys |
| `/__auth/api-keys`     | POST   | Create new key   |
| `/__auth/api-keys/:id` | DELETE | Revoke key       |

#### What We Reuse vs. Build

| Feature               | Status   | For Git                                   |
| --------------------- | -------- | ----------------------------------------- |
| User accounts         | ✅ Exists | Reuse — same users                        |
| Passkeys              | ✅ Exists | Not used — Git can't do WebAuthn          |
| Sessions              | ✅ Exists | Not used — Git uses Basic Auth            |
| **API keys**          | ❌ Build  | **Required** — this is the Git credential |
| **Roles**             | ❌ Build  | **Required** — `editor` role for push     |
| Basic Auth middleware | ❌ Build  | Bridges Git → API key validation          |

### Post-Push Reload

After a successful push, Basil triggers a handler reload:

```go
git.EventHandler = func(ev githttp.Event) {
    if ev.Type == githttp.PUSH {
        // Clear handler cache, trigger live reload
        server.ReloadHandlers()
    }
}
```

### Configuration

```yaml
git:
  enabled: true              # Enable Git server (default: false)
  require_auth: true         # Require authentication (default: true)
  # path: /.git              # URL path (default: /.git)
```

In development mode (`basil --dev`), authentication can be disabled for localhost-only access.

---

## Modes of Operation

### Production Mode (Authenticated)

For remote servers accessible over the internet:

- **TLS required** — HTTPS only
- **Authentication required** — HTTP Basic Auth over TLS
- **Role check** — `editor` role required for push
- **Audit logging** — Log clone/push events

```bash
git clone https://editor@mysite.example.com/.git
# Prompts for password
```

### Development Mode (Trusted Local)

For local development or trusted LAN environments:

- **HTTP allowed** — TLS optional on localhost
- **Auth optional** — Can disable for convenience
- **Localhost binding** — `127.0.0.1` by default in `--dev` mode

```yaml
# basil.yaml for local dev
git:
  enabled: true
  require_auth: false  # OK for localhost
```

```bash
# No credentials needed on localhost
git clone http://localhost:8080/.git
```

**Use cases for trusted mode:**
- Solo developer on their own machine
- Home/office LAN with trusted users
- Air-gapped environments
- Teaching/workshops

**Caveats:**
- Should warn if `require_auth: false` with non-localhost bind
- Production deployments should always require auth

---

## User Setup Flow

### How a User Gets Git Access

```
┌─────────────────────────────────────────────────────────────┐
│                    BASIL GIT SETUP                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. REGISTER (one-time)                                     │
│     Visit site's signup page (e.g., /register or /signup)   │
│     Create account with passkey (fingerprint/Face ID)       │
│                                                              │
│  2. GET EDITOR ROLE (admin grants)                          │
│     Site admin assigns "editor" role to your account        │
│     (via CLI: basil users set-role <user> editor)           │
│                                                              │
│  3. GENERATE GIT KEY (one-time)                             │
│     Log in, then visit https://mysite.example.com/__/       │
│     Go to Settings → API Keys                               │
│     Click "Generate new key"                                │
│     Name it "Git Access" or similar                         │
│     Copy the key (shown only once!)                         │
│                                                              │
│  4. CLONE                                                   │
│     git clone https://you@mysite.example.com/.git           │
│     Password: <paste your API key>                          │
│                                                              │
│  5. DONE                                                    │
│     Git caches the key. You won't be prompted again.        │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Note on URL conventions:**
- `/__/` — Admin panel (logs, settings, API keys)
- `/__auth/*` — Auth API endpoints (used by components internally)
- `/register`, `/login` — Site pages you create with auth components

### Role Requirements

| Operation    | Required Role          |
| ------------ | ---------------------- |
| Clone (read) | Any authenticated user |
| Pull (read)  | Any authenticated user |
| Push (write) | `editor` or `admin`    |

### Revoking Access

1. User can revoke their own API keys in `/__/` panel
2. Admin can delete user account (revokes all access)
3. Changing roles takes effect immediately (next Git operation)

---

## Bootstrap: First-Time Site Setup

**The chicken-and-egg problem:** A new Basil site has no users. How does the first admin get Git access?

### CLI-Based Bootstrap

The admin with SSH/terminal access uses CLI commands to create the first user and API key:

```bash
# Create first admin user
$ basil users create --name "Admin" --email "admin@example.com" --role admin
✓ Created user usr_abc123

# Generate their API key
$ basil apikey create --user usr_abc123 --name "MacBook Git"
✓ Created API key: bsl_live_abc123def456...
  (save this now — it won't be shown again)
```

The admin can now clone and push:

```bash
git clone https://admin@mysite.example.com/.git
Password: <paste bsl_live_abc123def456...>
```

### API Key Scopes (v1: None)

In v1, **the `--name` parameter is just a label** for the user to remember what the key is for. All API keys have equal capabilities — they inherit the user's role permissions.

```bash
# These all create identical-capability keys
$ basil apikey create --user usr_abc123 --name "MacBook Git"
$ basil apikey create --user usr_abc123 --name "CI/CD Pipeline"
$ basil apikey create --user usr_abc123 --name "Work laptop"
```

**Why no scopes in v1:**
- Simpler implementation
- The user's role (admin/editor) is the permission boundary
- Follows GitHub's original PAT model (fine-grained tokens came later)

**Future consideration:** Scoped keys (`--scope git`, `--scope api`, `--scope admin`) could be added if use cases emerge.

### CLI Commands Needed

These commands don't exist yet and must be implemented:

| Command                                       | Purpose                       |
| --------------------------------------------- | ----------------------------- |
| `basil users create --name --email --role`    | Create user without passkey   |
| `basil users list`                            | Show all users                |
| `basil users set-role <user> <role>`          | Change user role              |
| `basil apikey create --user --name`           | Generate API key for user     |
| `basil apikey list --user`                    | Show user's keys (prefix only)|
| `basil apikey revoke <key-id>`                | Revoke an API key             |

### What About Web Access?

The CLI-created user has no passkey — they can't log into the web UI yet. Options:

1. **Git-only workflow** — If the user only needs Git access, they're done.
2. **Register passkey later** — User visits `/register` and adds a passkey to their existing account.
3. **CLI generates registration link** — Future enhancement: `basil users invite` emails a one-time registration URL.

For v1, option 1 (Git-only) or option 2 (manual passkey registration) are sufficient.

---

## Developer Workflow

### Solo Developer

```bash
# One-time setup
git clone https://me@mysite.example.com/.git mysite
cd mysite

# Daily workflow
vim handlers/home.pars
git add -A
git commit -m "Update homepage"
git push origin main
# Site reloads automatically
```

### Team Workflow

```bash
# Developer A
git clone https://dev@team-site.example.com/.git
git checkout -b feature/new-page
# ... edit ...
git push origin feature/new-page

# Developer B (with merge rights)
git fetch
git checkout feature/new-page
# Review changes
git checkout main
git merge feature/new-page
git push origin main
# Site reloads
```

### Local Development with Remote Push

```bash
# Clone from production
git clone https://me@mysite.example.com/.git mysite
cd mysite

# Run local Basil for testing
basil --dev

# Test changes locally at http://localhost:8080
# When ready, push to production
git push origin main
```

---

## Effort Estimate

### Prerequisite: FEAT-004 Phase 2 (API Keys)

This work must be completed before Git over HTTPS can function:

| Component                       | Effort            | Notes                          |
| ------------------------------- | ----------------- | ------------------------------ |
| `api_keys` table + migrations   | 1 hour            | Schema already designed        |
| Key generation + bcrypt hashing | 2 hours           | Secure random, store hash only |
| Key validation function         | 1 hour            | Lookup by key hash             |
| `/__auth/api-keys` endpoints    | 2-3 hours         | CRUD operations                |
| `<ApiKeyList/>` component       | 2 hours           | List + revoke UI               |
| `<ApiKeyCreate/>` component     | 2 hours           | Generate + copy-once UI        |
| Add `role` column to users      | 1 hour            | Simple schema change           |
| **Subtotal (API Keys)**         | **\~10-12 hours** |                                |

### Git Server Implementation

| Component                         | Effort           | Complexity |
| --------------------------------- | ---------------- | ---------- |
| Mount git handler on routes       | 1-2 hours        | Low        |
| Basic Auth → API key middleware   | 2-3 hours        | Medium     |
| Role check (editor for push)      | 1 hour           | Low        |
| Post-receive hook for live reload | 2-3 hours        | Medium     |
| Config options (enable/disable)   | 1 hour           | Low        |
| Trusted mode (no auth)            | 1-2 hours        | Low        |
| Documentation                     | 1-2 hours        | Low        |
| **Subtotal (Git Server)**         | **\~9-14 hours** | **Medium** |

### Total

| Phase                       | Effort                         |
| --------------------------- | ------------------------------ |
| FEAT-004 Phase 2 (API Keys) | \~10-12 hours                  |
| Git Server + Integration    | \~9-14 hours                   |
| **Total**                   | **\~19-26 hours** (\~3-4 days) |

---

## Security Considerations

| Concern                  | Mitigation                               |
| ------------------------ | ---------------------------------------- |
| Credentials over network | Require HTTPS in production              |
| Auth bypass              | Use existing Basil session/role system   |
| Force push overwrites    | Optional: reject non-fast-forward pushes |
| Sensitive files pushed   | `.gitignore` / server-side hooks         |
| Brute force              | Rate limiting on auth failures           |

### Security Checklist

- [ ] HTTPS required when `require_auth: true`
- [ ] Warn/refuse `require_auth: false` with non-localhost bind
- [ ] Rate limit authentication attempts
- [ ] Log all push events with user identity
- [ ] Validate pushed content doesn't escape handler root

---

## Risks & Mitigations

| Risk                            | Likelihood | Mitigation                                      |
| ------------------------------- | ---------- | ----------------------------------------------- |
| `go-git-http` unmaintained      | Medium     | Library is simple/stable; could vendor or fork  |
| Performance with large repos    | Low        | Basil sites are small by design                 |
| Merge conflicts on shared sites | Medium     | Document workflows; add locking later if needed |
| User confusion with Git         | Low        | Target users already know Git                   |

---

## Open Questions

1. **Multiple environments (staging/production)?**
   2. Could deploy different branches to different paths
   3. Or run separate Basil instances (simpler)

2. **Failed deploys?**
   2. Syntax check Parsley before accepting push?
   3. Automatic rollback on error?
   4. Keep previous version available?

3. **Database migrations?**
   2. Should Git workflow include migration scripts?
   3. Run migrations on push?

4. **Large assets?**
   2. Git LFS support?
   3. Or keep large assets out of Git (separate upload mechanism)

5. **Notifications?**
   2. Email/Slack on push success/failure?
   3. Deploy log visible in `/__/` panel?

---

## Next Steps

1. Create FEAT spec for Git over HTTPS
2. Add `go-git-http` dependency
3. Implement `server/git.go` with handler
4. Integrate with auth middleware
5. Add post-push reload hook
6. Add configuration options
7. Document developer workflow

---

## Appendix: Other Options Considered

These alternatives were evaluated but don't fit the use case as well as Git over HTTPS.

### Webhooks (External Git + Deploy Hook)

**How it works:** Developers use GitHub/GitLab, push triggers webhook, Basil pulls.

**Pros:** Low implementation effort, leverages existing Git infrastructure.

**Cons:** Requires external Git hosting, less "batteries included", pull-based delay.

**Verdict:** Good option if external hosting is acceptable, but doesn't meet the "no external dependencies" goal.

### SFTP Server

**How it works:** Basil runs embedded SSH/SFTP, developers connect with SFTP clients.

**Pros:** Immediate sync, familiar tooling (VS Code Remote, etc.).

**Cons:** Encourages editing live sites, no history/rollback, SSH daemon complexity.

**Verdict:** Wrong workflow—encourages the risky behavior we want to prevent.

### Git over SSH

**How it works:** Same as Git over HTTPS but using SSH transport.

**Pros:** Strong key-based auth, familiar to developers.

**Cons:** Requires SSH daemon, extra port, key management complexity.

**Verdict:** Could add later if demand exists, but HTTPS is simpler and sufficient.

### Rsync over SSH

**How it works:** SSH server + rsync for directory sync.

**Pros:** Efficient incremental sync, familiar to sysadmins.

**Cons:** No history, single-developer workflow, same "edit live" risk.

**Verdict:** Same problems as SFTP.

### Hybrid (SFTP for Assets, Git for Code)

**How it works:** Git for handlers, SFTP for large assets in `/public/`.

**Pros:** Best of both for sites with heavy asset management.

**Cons:** Two systems to maintain, more complex mental model.

**Verdict:** Potential future enhancement if large asset handling becomes a pain point.

---

## Appendix: Database Synchronization

Separate but related concern: how do developers get database access for local development?

**Current approach:** Export/import via `/__/db` (manual process).

**Future options:**
- Schema-only export (table definitions without data)
- Seed data scripts (Parsley scripts that populate dev DB)
- `basil db:clone --schema-only` CLI command

This is out of scope for the Git workflow feature but noted for future consideration.
