---
id: man-bas-session
title: "Session Management"
system: basil
type: stdlib
name: session
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - session
  - flash
  - cookie
  - state
  - authentication
  - user data
  - server
  - basil.session
---

# Session Management

Session management for Basil server handlers. Provides key-value storage that persists across requests using cookie-based sessions.

> ⚠️ Sessions are only available inside Basil server handlers. They are not available in standalone Parsley scripts.

> **Note:** There is no `@std/session` module to import. Sessions are accessed via the `basil.session` object which is automatically available in handler code.

## Session Data

### `.get(key, default?)`

Retrieve a value from the session. Returns `null` (or the default) if the key does not exist:

```parsley
let username = basil.session.get("username")
let theme = basil.session.get("theme", "light")  // "light" if not set
```

### `.set(key, value)`

Store a value in the session:

```parsley
basil.session.set("username", "Alice")
basil.session.set("preferences", {theme: "dark", lang: "en"})
```

Values are serialized for storage — strings, numbers, booleans, arrays, and dictionaries are supported.

### `.delete(key)`

Remove a key from the session:

```parsley
basil.session.delete("username")
```

### `.has(key)`

Check if a key exists in the session:

```parsley
basil.session.has("username")        // true or false
```

### `.clear()`

Remove all session data and flash messages:

```parsley
basil.session.clear()
```

### `.all()`

Return all session data as a dictionary:

```parsley
let data = basil.session.all()
data.username                        // "Alice"
```

## Flash Messages

Flash messages are one-time values that are automatically removed after being read. They are commonly used to pass status messages across redirects (e.g., "Item saved successfully").

### `.flash(key, message)`

Set a flash message:

```parsley
basil.session.flash("success", "Profile updated!")
basil.session.flash("error", "Invalid email address")
```

### `.getFlash(key)`

Retrieve and remove a flash message. Returns `null` if no message exists for the key:

```parsley
let msg = basil.session.getFlash("success")
// msg is "Profile updated!" (and the flash is now deleted)

let again = basil.session.getFlash("success")
// again is null (already consumed)
```

### `.getAllFlash()`

Retrieve and remove all flash messages as a dictionary:

```parsley
let flashes = basil.session.getAllFlash()
// {success: "Profile updated!", error: "Invalid email"}
// All flash messages are now cleared
```

### `.hasFlash()`

Check if any flash messages exist:

```parsley
basil.session.hasFlash()             // true or false
```

## Session Regeneration

### `.regenerate()`

Regenerate the session, preserving existing data. Use this after authentication to prevent session fixation attacks:

```parsley
// After successful login
basil.session.set("userId", user.id)
basil.session.regenerate()
```

## Methods Summary

| Method | Args | Returns | Description |
|---|---|---|---|
| `.get(key, default?)` | string, any? | any | Get session value |
| `.set(key, value)` | string, any | null | Set session value |
| `.delete(key)` | string | null | Remove session value |
| `.has(key)` | string | boolean | Check if key exists |
| `.clear()` | none | null | Remove all data and flash messages |
| `.all()` | none | dictionary | Get all session data |
| `.flash(key, msg)` | string, string | null | Set a flash message |
| `.getFlash(key)` | string | string or null | Get and remove a flash message |
| `.getAllFlash()` | none | dictionary | Get and remove all flash messages |
| `.hasFlash()` | none | boolean | Any flash messages exist? |
| `.regenerate()` | none | null | Regenerate the session |

## Common Patterns

### Login / Logout

```parsley
let api = import @basil/api

// Login handler
let handleLogin = fn(req) {
    let user = authenticate(req.body.email, req.body.password)
    check user != null else api.unauthorized("Invalid credentials")

    basil.session.set("userId", user.id)
    basil.session.set("role", user.role)
    basil.session.regenerate()
    basil.session.flash("success", "Welcome back, " + user.name + "!")

    api.redirect("/dashboard")
}

// Logout handler
let handleLogout = fn(req) {
    basil.session.clear()
    basil.session.flash("info", "You have been logged out")
    api.redirect("/")
}
```

### Displaying Flash Messages

```parsley
// In a page template
let flashes = basil.session.getAllFlash()

if (flashes.success) {
    <div class=alert-success>flashes.success</div>
}
if (flashes.error) {
    <div class=alert-error>flashes.error</div>
}
```

## Key Differences from Other Languages

- **No explicit save** — session data is automatically persisted at the end of the request. Setting a value with `.set()` marks the session as dirty.
- **Flash is built in** — no need for a separate flash middleware. Flash messages are a first-class part of the session API.
- **Cookie-based** — sessions are stored in signed cookies by default. No server-side session store is required.

## See Also

- [@basil/api](api.md) — auth wrappers and HTTP error helpers
- [Security Model](../../parsley/manual/features/security.md) — security policies
- [Error Handling](../../parsley/manual/fundamentals/errors.md) — `check` for guard-style preconditions