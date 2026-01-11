---
id: FEAT-004
title: "Authentication"
status: implemented
priority: medium
created: 2025-11-30
updated: 2025-12-01
author: "@sambeau"
---

# FEAT-004: Authentication

## Summary

Add passkey-based authentication to Basil via built-in Parsley components. No passwords, no CSRF tokens, no OAuth complexity—just modern, phishing-resistant authentication that "just works."

## Design Decisions

### Passkeys First (No Passwords)

We're skipping password-based auth entirely:

- **Simpler for users** - No passwords to remember or manage
- **Simpler for developers** - No bcrypt, no password reset flows, no "forgot password" emails
- **More secure** - Phishing-proof, no credentials to steal from database
- **Future-proof** - This is where auth is heading (Apple, Google, Microsoft all pushing passkeys)

### Parsley Components (Not Built-in Pages)

Basil provides auth as **Parsley components** that developers embed in their own pages:

```parsley
// signup.pars - Developer controls the entire page
<!DOCTYPE html>
<html>
<head>
  <link rel=stylesheet href="/static/styles.css"/>
</head>
<body>
  <h1>Join Us</h1>
  
  <PasskeyRegister
    button_text="Create account"
    class="my-form"
  />
  
  <p>Already have an account? <a href="/login">Sign in</a></p>
</body>
</html>
```

**Benefits:**
- User controls all styling (their CSS, their page structure)
- No CSS config needed in Basil
- Components render semantic HTML with predictable classes
- Full flexibility for custom flows

### Air Gap: Parsley Can't Touch Auth Database

The auth database (`.basil-auth.db`) is completely separate:

- Parsley **cannot** query it via `SQLITE()` or `<=?=>`
- Parsley **only** sees `request.user` (read-only)
- All credential management happens in Go

**Security model:**
```
┌─────────────────────────────────────────┐
│  Parsley Handler                        │
│  - Reads request.user (read-only)       │
│  - Uses <PasskeyRegister/> etc.         │
│  - Cannot query auth tables             │
└─────────────────────────────────────────┘
                  │
                  ▼ (read-only user object)
┌─────────────────────────────────────────┐
│  Basil Auth Layer                       │
│  - Handles /__auth/* API endpoints      │
│  - Manages sessions                     │
│  - Owns auth database exclusively       │
└─────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────┐
│  .basil-auth.db (separate file)         │
│  - Users, credentials, sessions         │
│  - File permissions: 0600               │
└─────────────────────────────────────────┘
```

---

## API Design

### Config

```yaml
# basil.yaml
auth:
  enabled: true
  registration: open    # open | closed

routes:
  - path: /
    handler: ./public.pars
    
  - path: /admin/
    handler: ./admin.pars
    auth: required        # Must be logged in
    
  - path: /profile/
    handler: ./profile.pars
    auth: optional        # request.user available if logged in
```

### Parsley Components

#### `<PasskeyRegister/>`

Registration form with WebAuthn:

```parsley
<PasskeyRegister
  name={request.form.name ?? ""}    // Pre-fill name (optional)
  email={request.form.email ?? ""}  // Pre-fill email (optional)
  name_placeholder="Your name"
  email_placeholder="you@example.com"
  button_text="Create account"
  redirect="/"                       // After success
  class="my-form-class"              // For styling
/>
```

Renders semantic HTML:
```html
<form class="basil-auth-register my-form-class">
  <input type="text" name="name" class="basil-auth-input" placeholder="Your name" required/>
  <input type="email" name="email" class="basil-auth-input" placeholder="you@example.com" required/>
  <button type="submit" class="basil-auth-button">Create account</button>
  <div class="basil-auth-error" hidden></div>
</form>
<script>/* WebAuthn registration logic */</script>
```

#### `<PasskeyLogin/>`

Login button with WebAuthn:

```parsley
<PasskeyLogin
  button_text="Sign in"
  redirect="/"
  class="login-btn"
/>
```

Renders:
```html
<div class="basil-auth-login login-btn">
  <button type="button" class="basil-auth-button">Sign in</button>
  <div class="basil-auth-error" hidden></div>
</div>
<script>/* WebAuthn authentication logic */</script>
```

#### `<PasskeyLogout/>`

Logout button or link:

```parsley
<PasskeyLogout
  text="Sign out"
  redirect="/"
  method="link"       // "link" | "button"
  class="logout-link"
/>
```

### `request.user` Object

When authenticated, available in all handlers:

```parsley
request.user.id           // "usr_abc123"
request.user.name         // "Sam Phillips"
request.user.email        // "sam@example.com" (may be null)
request.user.created      // @2025-12-01T10:00:00
```

When not authenticated: `request.user` is `null`.

**Usage patterns:**

```parsley
// Simple check
if request.user {
  <p>Welcome, {request.user.name}!</p>
}

// Destructuring
let {name, email} = request.user ?? {name: "Guest", email: null}

// Nullish coalescing
let displayName = request.user.name ?? "Anonymous"
```

### Internal API Endpoints

Components communicate with these (users don't need to know about them):

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/__auth/register` | POST | WebAuthn registration |
| `/__auth/login` | POST | WebAuthn authentication |
| `/__auth/logout` | POST | End session |
| `/__auth/challenge` | GET | Get WebAuthn challenge |

---

## Database Schema

Separate file: `.basil-auth.db`

```sql
CREATE TABLE users (
  id TEXT PRIMARY KEY,              -- "usr_abc123"
  name TEXT NOT NULL,
  email TEXT,                        -- Optional
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE credentials (
  id BLOB PRIMARY KEY,               -- WebAuthn credential ID
  user_id TEXT NOT NULL REFERENCES users(id),
  public_key BLOB NOT NULL,          -- NOT secret
  sign_count INTEGER DEFAULT 0,      -- Replay protection
  transports TEXT,                   -- JSON array: ["internal", "usb"]
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
  id TEXT PRIMARY KEY,               -- Random token
  user_id TEXT NOT NULL REFERENCES users(id),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_credentials_user ON credentials(user_id);
```

**Security notes:**
- Public keys are not secret (that's the point of public key crypto)
- Session tokens are the main target if database is stolen
- Sessions have short TTL (24h default) and can be invalidated

---

## Implementation Notes

### Dependencies

- `github.com/go-webauthn/webauthn` - WebAuthn server library (well-maintained, FIDO2 conformant)

### Component Implementation

Components are special tags that Basil recognizes and expands:

```go
// When Parsley evaluates <PasskeyRegister .../>
// Basil intercepts and returns the HTML + JS

func (s *Server) expandAuthComponent(name string, attrs map[string]string) string {
    switch name {
    case "PasskeyRegister":
        return s.renderRegisterForm(attrs)
    case "PasskeyLogin":
        return s.renderLoginButton(attrs)
    case "PasskeyLogout":
        return s.renderLogoutButton(attrs)
    }
    return ""
}
```

### Session Cookie

```
Name: __basil_session
HttpOnly: true
Secure: true (production)
SameSite: Lax
Path: /
MaxAge: 86400 (24h)
```

---

## Example: Complete Auth Flow

```parsley
// pages/signup.pars
<!DOCTYPE html>
<html>
<head>
  <title>Sign Up</title>
  <link rel=stylesheet href="/static/auth.css"/>
</head>
<body>
  <main>
    <h1>Create Account</h1>
    
    <PasskeyRegister
      button_text="Sign up with passkey"
      redirect="/welcome"
    />
    
    <p>Already have an account? <a href="/login">Sign in</a></p>
  </main>
</body>
</html>
```

```parsley
// pages/login.pars
<!DOCTYPE html>
<html>
<head>
  <title>Sign In</title>
  <link rel=stylesheet href="/static/auth.css"/>
</head>
<body>
  <main>
    <h1>Welcome Back</h1>
    
    <PasskeyLogin button_text="Sign in"/>
    
    <p>New here? <a href="/signup">Create account</a></p>
  </main>
</body>
</html>
```

```parsley
// components/header.pars
<header>
  <nav>
    <a href="/">Home</a>
    if request.user {
      <span>{request.user.name}</span>
      <PasskeyLogout text="Sign out" method="link"/>
    } else {
      <a href="/login">Sign in</a>
    }
  </nav>
</header>
```

```css
/* static/auth.css - User's styles */
.basil-auth-register,
.basil-auth-login {
  max-width: 400px;
  margin: 2rem auto;
}

.basil-auth-input {
  display: block;
  width: 100%;
  padding: 0.75rem;
  margin-bottom: 1rem;
  border: 1px solid #ddd;
  border-radius: 4px;
}

.basil-auth-button {
  width: 100%;
  padding: 0.75rem;
  background: #2563eb;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.basil-auth-error {
  color: #dc2626;
  margin-top: 0.5rem;
}
```

---

## Design Decisions

1. **Component attribute naming**: snake_case (`button_text`) — kebab-case would parse as subtraction in Parsley.

2. **Multiple passkeys per user**: No for V1 — one device per user keeps it simple.

3. **User management**: CLI for V1 (`basil users list`, `basil users delete`), UI later.

4. **Email**: Optional profile data, not required. Passkey IS the identity.

5. **Account recovery**: Recovery codes shown at signup (8 one-time codes). No email infrastructure needed. If user loses codes AND device, site owner resets via CLI.

---

## Phase 2: API Keys (Machines)

Passkeys are for humans. API keys are for machines.

### Design Philosophy

Same component pattern, same `request.user` object—just a different auth method:

| Aspect | Passkeys | API Keys |
|--------|----------|----------|
| **Who uses it** | Human in browser | Scripts, CI, apps |
| **How it works** | WebAuthn ceremony | Bearer token in header |
| **Secret shown** | Never (lives in device) | Once at creation |
| **Revocation** | Per-device | Per-key |
| **Session** | Cookie-based | Stateless |

### Components

#### `<ApiKeyList/>`

Shows the user's existing API keys with revoke buttons:

```parsley
// settings/api-keys.pars
<h2>Your API Keys</h2>

<ApiKeyList
  empty_message="No API keys yet"
  class="my-key-list"
/>
```

Renders:
```html
<div class="basil-apikey-list my-key-list">
  <table>
    <tr>
      <td>CI Server</td>
      <td><code>bsl_...k2m9</code></td>
      <td>Created 2 days ago</td>
      <td>Last used 1 hour ago</td>
      <td><button class="basil-apikey-revoke">Revoke</button></td>
    </tr>
    <!-- ... -->
  </table>
</div>
```

#### `<ApiKeyCreate/>`

Form to generate a new API key:

```parsley
<ApiKeyCreate
  name_placeholder="Key name (e.g., 'CI Server')"
  button_text="Generate new key"
  class="my-form"
/>
```

After creation, shows the key **exactly once**:

```html
<div class="basil-apikey-created">
  <p>Your new API key (copy it now, you won't see it again):</p>
  <code class="basil-apikey-value">bsl_live_a8f3k2m9x7p2...</code>
  <button class="basil-apikey-copy">Copy</button>
</div>
```

### Usage

Client sends key in Authorization header:

```bash
curl -H "Authorization: Bearer bsl_live_a8f3k2m9x7p2..." \
  https://mysite.com/api/posts
```

### `request.user` with API Keys

Same object, different auth method:

```parsley
request.user.id           // "usr_abc123"
request.user.name         // "Sam Phillips"
request.user.email        // "sam@example.com"
request.user.auth_method  // "api_key" (vs "passkey" for humans)
request.user.api_key_name // "CI Server" (only set for API key auth)
```

### Database Schema Addition

```sql
CREATE TABLE api_keys (
  id TEXT PRIMARY KEY,                -- "key_xyz789"
  user_id TEXT NOT NULL REFERENCES users(id),
  name TEXT NOT NULL,                 -- "CI Server"
  key_hash TEXT NOT NULL,             -- bcrypt hash of the key
  key_prefix TEXT NOT NULL,           -- "bsl_...k2m9" for display
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  last_used_at TIMESTAMP,
  expires_at TIMESTAMP                -- Optional expiry
);

CREATE INDEX idx_api_keys_user ON api_keys(user_id);
```

**Security notes:**
- Only store hash of key (like passwords)
- Store prefix for display/identification
- Track last_used_at for auditing
- Optional expiry for security-conscious users

### Key Format

```
bsl_live_<random32chars>
bsl_test_<random32chars>  // Future: test mode keys?
```

Prefix makes keys:
- Identifiable if leaked (grep logs for `bsl_live_`)
- Self-documenting
- Distinguishable from other tokens

### Internal API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/__auth/api-keys` | GET | List user's keys (requires session) |
| `/__auth/api-keys` | POST | Create new key (requires session) |
| `/__auth/api-keys/:id` | DELETE | Revoke key (requires session) |

Note: You need to be logged in with a passkey to manage API keys. The keys themselves are for machine use.

---

## Future Considerations (Not MVP)

- **Roles/permissions** (`request.user.role`)
- **OAuth2/OIDC** providers (Google, GitHub, etc.)
- **Admin interface** for user management
- **Invite codes** for closed registration
- **Key scopes** - limit what API keys can access

---

## References

- [WebAuthn Guide](https://webauthn.guide/)
- [MDN Web Authentication API](https://developer.mozilla.org/en-US/docs/Web/API/Web_Authentication_API)
- [go-webauthn/webauthn](https://github.com/go-webauthn/webauthn)
- [Passkeys.dev](https://passkeys.dev/)
- [OWASP Session Management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
