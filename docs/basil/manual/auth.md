---
id: man-bas-auth
title: "@basil/auth"
system: basil
type: stdlib
name: auth
created: 2026-07-15
version: 0.2.0
author: Basil Team
keywords:
  - auth
  - authentication
  - session
  - user
  - role
  - handler
  - server
  - basil
---

# @basil/auth

The per-request authentication context for Basil handlers: the current `session`, the `auth` state (whether the route requires login and who is signed in), and a `user` shortcut.

```parsley
let {session, auth, user} = import @basil/auth
```

> ⚠️ **Server-only.** Available when auth is configured in `basil.yaml`. All three exports are dynamic accessors, so a value read at module scope always reflects the current request.

## session

The current session module — key-value storage and flash messages scoped to the visitor.

**Type:** `SessionModule`

The session's methods (`get`, `set`, `flash`, `regenerate`, …) are documented in full on the [Session Management](session.md) page.

```parsley
let {session} = import @basil/auth
session.set("cart", ["apple", "pear"])
session.get("cart", [])
```

## auth

The authentication context for the current request.

**Type:** `Dictionary`

| Property | Type | Description |
|---|---|---|
| `required` | `Boolean` | `true` if the route requires authentication |
| `user` | `Dictionary` \| `null` | The authenticated user, or `null` if not logged in |

```parsley
let {auth} = import @basil/auth
if (auth.required && !auth.user) {
    redirect("/login")
}
```

## user

A shortcut to `auth.user` — the currently authenticated user, or `null`.

**Type:** `Dictionary` or `null`

**User properties (when authenticated):**

| Property | Type | Description |
|---|---|---|
| `id` | `Integer` | User ID |
| `name` | `String` | Display name |
| `email` | `String` | Email address |
| `role` | `String` | User role (e.g. `"admin"`, `"user"`) |
| `created` | `String` | Account creation timestamp |
| `email_verified_at` | `String` \| `null` | Email verification timestamp |
| `email_verification_pending` | `Boolean` | `true` if the email is not yet verified |

```parsley
let {user} = import @basil/auth
if (user) {
    `Welcome, {user.name}!`
} else {
    "Please log in"
}
```

## See Also

- [Authentication](authentication.md) — configuring auth, passkeys, roles, and protected paths
- [The Authentication Guide](authentication-guide.md) — auth in depth: sessions, recovery, email verification, CSRF
- [Session Management](session.md) — the full `session` method reference
- [@basil/http](http.md) — the raw request/response context
