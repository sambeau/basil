---
id: man-bas-authentication
title: "Authentication"
system: basil
type: feature
name: authentication
created: 2026-07-12
version: 1.0.0-alpha.4
author: "@sam"
keywords:
  - auth
  - passkeys
  - webauthn
  - login
  - users
  - roles
  - sessions
  - api keys
  - protected paths
  - email verification
---

# Authentication

Basil has passkey (WebAuthn) authentication built in — no passwords to store, no OAuth dance, no third-party service. Enable it, drop in the components, and protect your routes.

> **Basil-only.** Auth components and `basil.auth.*` require the Basil server environment.

## Quick Start

**1. Enable it** in `basil.yaml`:

```yaml
auth:
  enabled: true
  registration: open    # anyone can sign up (default: closed)
  session_ttl: 24h
```

**2. Create your first user:**

```bash
basil users create --name "You" --email you@example.com
```

This also creates the auth database (`.basil-auth.db`, next to `basil.yaml`) — the server won't start with auth enabled until it exists. The first user is always an admin.

**3. Add the components** to your pages:

```parsley
// signup page
<basil.auth.Register button_text="Create account" redirect="/dashboard"/>

// login page
<basil.auth.Login button_text="Sign in" redirect="/dashboard"/>

// anywhere
<basil.auth.Logout text="Sign out" redirect="/"/>
```

Sign up with the same email as your CLI-created user and the new passkey attaches to that account.

**4. Protect routes:**

```yaml
auth:
  enabled: true
  protected_paths:
    - /dashboard              # any signed-in user
    - path: /admin
      roles: [admin]          # admins only
```

Signed-out visitors are redirected to the login page; signed-in visitors without the right role get a 403. Or protect per-route in routes mode:

```yaml
routes:
  - path: /dashboard
    handler: ./handlers/dashboard.pars
    auth: required     # 401 if not signed in
  - path: /profile
    handler: ./handlers/profile.pars
    auth: optional     # basil.auth.user set if signed in
```

**5. Use the signed-in user** in handlers:

```parsley
if (basil.auth.user != null) {
    <p>"Hello, " + basil.auth.user.name + "!"</p>
}
```

| Field | Description |
|-------|-------------|
| `basil.auth.user.id` | User ID (e.g. `usr_abc123`) |
| `basil.auth.user.name` | Display name |
| `basil.auth.user.email` | Email (may be empty) |
| `basil.auth.user.role` | Role — `"admin"` or `"editor"` |
| `basil.auth.user.created` | Account creation timestamp |
| `basil.auth.user.email_verified_at` | Verification timestamp, or `null` |
| `basil.auth.user.email_verification_pending` | `true` if email set but unverified |

The two email fields only appear when the user has an email address. `basil.auth.user` is `null` for signed-out visitors, and `basil.auth.required` tells you whether the current route declared `auth: required`.

## Components

### `<basil.auth.Register/>`

| Attribute | Default | Description |
|-----------|---------|-------------|
| `name`, `email` | `""` | Pre-fill fields |
| `name_placeholder` | `"Your name"` | Name input placeholder |
| `email_placeholder` | `"you@example.com"` | Email input placeholder |
| `button_text` | `"Create account"` | Button label |
| `redirect` | `"/"` | URL after registration |
| `recovery_page` | `""` | Page to show recovery codes (recommended) |
| `class` | `""` | Extra CSS classes |

When `recovery_page` is set, the new user's recovery codes are placed in `sessionStorage` (`basil_recovery_codes`, `basil_recovery_user`) and the user is redirected there — display them, then clear the storage. Without it, the codes appear in a browser `alert()`, which works but won't win any design awards.

### `<basil.auth.Login/>`

| Attribute | Default | Description |
|-----------|---------|-------------|
| `button_text` | `"Sign in"` | Button label |
| `redirect` | `"/"` | URL after login |
| `class` | `""` | Extra CSS classes |

### `<basil.auth.Logout/>`

| Attribute | Default | Description |
|-----------|---------|-------------|
| `text` | `"Sign out"` | Button/link text |
| `redirect` | `"/"` | URL after logout |
| `method` | `"button"` | `"button"` or `"link"` |
| `class` | `""` | Extra CSS classes |

## Configuration Reference

```yaml
auth:
  enabled: true           # default: false
  registration: open      # "open" | "closed" (default: closed)
  session_ttl: 24h        # Go duration format (default: 24h)
  login_path: /login      # redirect target for signed-out visitors (default: /login)
  protected_paths:
    - /dashboard
    - path: /admin
      roles: [admin]
```

Even with `registration: closed`, the very first user may sign up — so you can bootstrap an account before locking the door.

## Managing Users

Users are created from the [CLI](running.md):

```bash
basil users create --name "Sam" --email sam@example.com --role admin
basil users list
basil users set-role <id> editor
basil users reset <id>     # new recovery codes
```

API keys (used by [Git deploys](git.md)):

```bash
basil apikey create --user <id> --name "MacBook Git"
basil apikey list --user <id>
basil apikey revoke <id>
```

## There's More

This page covers the essentials. Authentication also includes:

- **Recovery codes** — eight one-time codes per user, for when a passkey device goes missing
- **Email verification** — send verification emails via Mailgun or Resend
- **Email recovery** — verified users can recover their account by email
- **Role-based API modules** — `api.public()`, `api.adminOnly()`, and `api.roles()`
- **CSRF protection** — automatic token checks on authenticated form posts
- **`/__auth/*` endpoints** — the HTTP API behind the components

## See Also

- [The Authentication Guide](authentication-guide.md) — the deep dive: passkeys, sessions, roles, protected paths, recovery, API keys, and email verification
- [Running Basil](running.md) — user and API key CLI
- [Git Deploy](git.md) — API keys in action
