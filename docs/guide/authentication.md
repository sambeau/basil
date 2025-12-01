# Authentication

Basil includes built-in passkey authentication. No passwords, no OAuth complexity—just modern, secure authentication.

## Quick Start

### 1. Enable Authentication

In your `basil.yaml`:

```yaml
auth:
  enabled: true
  registration: open    # Anyone can sign up
  session_ttl: 24h      # Session duration
```

### 2. Add Auth Components

In your Parsley handlers, use the built-in components:

**Registration page (signup.pars):**
```parsley
<PasskeyRegister
  name_placeholder="Your name"
  email_placeholder="Email (optional)"
  button_text="Create account"
  redirect="/dashboard"
/>
```

**Login page (login.pars):**
```parsley
<PasskeyLogin
  button_text="Sign in"
  redirect="/dashboard"
/>
```

**Logout (anywhere):**
```parsley
<PasskeyLogout
  text="Sign out"
  redirect="/"
/>
```

### 3. Protect Routes

In your `basil.yaml`:

```yaml
routes:
  - path: /dashboard
    handler: ./handlers/dashboard.pars
    auth: required    # 401 if not logged in
  
  - path: /profile
    handler: ./handlers/profile.pars
    auth: optional    # request.user available if logged in
```

### 4. Use `request.user`

In authenticated handlers:

```parsley
if request.user {
  <p>Hello, {request.user.name}!</p>
}
```

User object fields:
- `request.user.id` — User ID (e.g., "usr_abc123")
- `request.user.name` — Display name
- `request.user.email` — Email (may be empty)
- `request.user.created` — Account creation timestamp

## Components Reference

### `<PasskeyRegister/>`

Registration form with WebAuthn.

| Attribute | Default | Description |
|-----------|---------|-------------|
| `name` | `""` | Pre-fill name field |
| `email` | `""` | Pre-fill email field |
| `name_placeholder` | `"Your name"` | Name input placeholder |
| `email_placeholder` | `"you@example.com"` | Email input placeholder |
| `button_text` | `"Create account"` | Button label |
| `redirect` | `"/"` | URL after successful registration |
| `recovery_page` | `""` | URL to show recovery codes (recommended) |
| `class` | `""` | Additional CSS classes |

**Recovery codes:** When `recovery_page` is set, codes are stored in `sessionStorage` as `basil_recovery_codes` (JSON array) and `basil_recovery_user` (username), then the user is redirected there. Your recovery page can display them nicely and clear the storage after.

### `<PasskeyLogin/>`

Login button with WebAuthn.

| Attribute | Default | Description |
|-----------|---------|-------------|
| `button_text` | `"Sign in"` | Button label |
| `redirect` | `"/"` | URL after successful login |
| `class` | `""` | Additional CSS classes |

### `<PasskeyLogout/>`

Logout button or link.

| Attribute | Default | Description |
|-----------|---------|-------------|
| `text` | `"Sign out"` | Button/link text |
| `redirect` | `"/"` | URL after logout |
| `method` | `"button"` | `"button"` or `"link"` |
| `class` | `""` | Additional CSS classes |

## Configuration Reference

```yaml
auth:
  enabled: true           # Enable authentication (default: false)
  registration: open      # "open" or "closed" (default: open)
  session_ttl: 24h        # Session duration (default: 24h)
```

- **`enabled`**: Must be `true` to activate authentication
- **`registration`**: Set to `"closed"` to disable new signups
- **`session_ttl`**: Go duration format (e.g., `1h`, `7d`, `168h`)

## User Management CLI

Manage users from the command line:

```bash
# List all users
basil users list

# Show user details
basil users show usr_abc123

# Delete a user
basil users delete usr_abc123
basil users delete usr_abc123 --force  # Skip confirmation

# Generate new recovery codes
basil users reset usr_abc123
```

## Recovery Codes

When a user registers, they receive 8 recovery codes. These are one-time codes that can be used if they lose access to their passkey device.

To use a recovery code, users should contact you (the site administrator). You can then use the CLI to verify their identity and either:
- Generate new recovery codes with `basil users reset`
- Help them register a new passkey

**Future enhancement:** A web-based recovery flow is planned.

## CSS Styling

The auth components use these CSS classes:

```css
/* Forms */
.basil-auth-register { }
.basil-auth-login { }
.basil-auth-logout { }

/* Inputs */
.basil-auth-input { }

/* Buttons */
.basil-auth-button { }

/* Error messages */
.basil-auth-error { }
```

Add your own styles to customize the appearance.

## Security Notes

- **Passkeys are phishing-resistant**: They're bound to your domain
- **No passwords stored**: Nothing to leak in a database breach
- **Recovery codes are hashed**: Stored securely with bcrypt
- **Session tokens**: Cryptographically random, HttpOnly cookies
- **HTTPS required**: In production, always use HTTPS

## Troubleshooting

### "WebAuthn not supported"

The browser doesn't support passkeys. All modern browsers do—update or switch browsers.

### "Registration failed"

Check the browser console. Common causes:
- Not on `localhost` or HTTPS
- Browser cancelled the operation
- Invalid RP ID (domain mismatch)

### "Login failed"

- No passkey registered for this origin
- User cancelled the browser prompt
- Passkey device not available

### Users can't sign up

Check `auth.registration` in config—it might be set to `"closed"`.

## Example

See `examples/auth/` for a complete working example with:
- Public homepage
- Signup and login pages
- Protected dashboard
- Styling

```bash
cd examples/auth
basil --dev
```
