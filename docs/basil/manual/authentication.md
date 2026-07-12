---
id: man-bas-authentication
title: "Authentication"
system: basil
type: feature
name: authentication
created: 2026-07-12
version: 1.0.0-alpha.3
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
---

# Authentication

Basil has passkey (WebAuthn) authentication built in — no passwords to store, no OAuth dance, no third-party service. Enable it, drop in the components, and protect your routes.

> **Basil-only.** Auth components and `basil.auth.*` require the Basil server environment.

## Quick Start

**1. Enable it** in `basil.yaml`:

```yaml
auth:
  enabled: true
  registration: open    # anyone can sign up
  session_ttl: 24h
```

**2. Add the components** to your pages:

```parsley
// signup page
<basil.auth.Register button_text="Create account" redirect="/dashboard"/>

// login page
<basil.auth.Login button_text="Sign in" redirect="/dashboard"/>

// anywhere
<basil.auth.Logout text="Sign out" redirect="/"/>
```

**3. Protect routes:**

```yaml
auth:
  enabled: true
  protected_paths:
    - /dashboard              # any signed-in user
    - path: /admin
      roles: [admin]          # admins only
```

Or per-route in routes mode:

```yaml
routes:
  - path: /dashboard
    handler: ./handlers/dashboard.pars
    auth: required     # 401 if not signed in
  - path: /profile
    handler: ./handlers/profile.pars
    auth: optional     # basil.auth.user set if signed in
```

**4. Use the signed-in user** in handlers:

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
| `basil.auth.user.role` | Role (`admin`, `editor`, …) |
| `basil.auth.user.created` | Account creation timestamp |

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

When `recovery_page` is set, the new user's recovery codes are placed in `sessionStorage` (`basil_recovery_codes`, `basil_recovery_user`) and the user is redirected there — display them, then clear the storage.

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
  registration: open      # "open" | "closed"
  session_ttl: 24h        # Go duration format
  login_path: /login      # redirect target for unauthenticated users
  protected_paths:
    - /dashboard
    - path: /admin
      roles: [admin]
```

## Managing Users

Users are created from the [CLI](running.md):

```bash
basil users create
basil users list
basil users set-role <id>
basil users reset <id>     # new recovery codes
```

API keys (for [Git deploys](git.md) and programmatic access):

```bash
basil apikey create
basil apikey list
basil apikey revoke <id>
```

## See Also

- [Running Basil](running.md) — user and API key CLI
- [Git Deploy](git.md) — API keys in action
- The [authentication guide](https://github.com/sambeau/basil/blob/main/docs/guide/authentication.md) — recovery flows and role-based access in depth
