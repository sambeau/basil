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
<basil.auth.Register
  name_placeholder="Your name"
  email_placeholder="Email (optional)"
  button_text="Create account"
  redirect="/dashboard"
/>
```

**Login page (login.pars):**

```parsley
<basil.auth.Login
  button_text="Sign in"
  redirect="/dashboard"
/>
```

**Logout (anywhere):**

```parsley
<basil.auth.Logout
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
    auth: optional    # basil.auth.user available if logged in
```

### 4. Use `basil.auth.user`

In authenticated handlers:

```parsley
if basil.auth.user {
  <p>Hello, {basil.auth.user.name}!</p>
}
```

User object fields:
- `basil.auth.user.id` — User ID (e.g., "usr_abc123")
- `basil.auth.user.name` — Display name
- `basil.auth.user.email` — Email (may be empty)
- `basil.auth.user.role` — User role ("admin" or "editor")
- `basil.auth.user.created` — Account creation timestamp

## Components Reference

### `<basil.auth.Register/>`

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

### `<basil.auth.Login/>`

Login button with WebAuthn.

| Attribute | Default | Description |
|-----------|---------|-------------|
| `button_text` | `"Sign in"` | Button label |
| `redirect` | `"/"` | URL after successful login |
| `class` | `""` | Additional CSS classes |

### `<basil.auth.Logout/>`

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
  login_path: /login      # Where to redirect unauthenticated users (default: /login)
  protected_paths:        # URL prefixes that require authentication
    - /dashboard          # Any authenticated user
    - /settings
    - path: /admin        # With role requirement
      roles: [admin]
```

- **`enabled`**: Must be `true` to activate authentication
- **`registration`**: Set to `"closed"` to disable new signups
- **`session_ttl`**: Go duration format (e.g., `1h`, `7d`, `168h`)
- **`login_path`**: Redirect destination for unauthenticated requests (default: `/login`)
- **`protected_paths`**: List of URL prefixes requiring auth (see below)

## Protected Paths

Protect entire sections of your site by URL prefix. All paths starting with the prefix require authentication.

### Simple Protected Paths

```yaml
auth:
  enabled: true
  protected_paths:
    - /dashboard
    - /settings
    - /api/private
```

This protects:
- `/dashboard`, `/dashboard/`, `/dashboard/users`, `/dashboard/users/123`
- `/settings`, `/settings/profile`, etc.
- `/api/private`, `/api/private/data`, etc.

### Protected Paths with Roles

Restrict paths to specific user roles:

```yaml
auth:
  enabled: true
  protected_paths:
    - /dashboard              # Any authenticated user
    - path: /admin
      roles: [admin]          # Admin only
    - path: /content
      roles: [admin, editor]  # Admin or editor
```

### Login Redirect

Unauthenticated HTML requests redirect to `login_path` with a `?next=` parameter:

```yaml
auth:
  enabled: true
  login_path: /auth/signin    # Custom login page
  protected_paths:
    - /dashboard
```

Visiting `/dashboard/users` while logged out redirects to `/auth/signin?next=/dashboard/users`.

API requests (paths starting with `/api/` or `Accept: application/json`) get a 401 JSON response instead of a redirect.

### Site Mode

Protected paths work with filesystem-based routing:

```yaml
site: ./site

auth:
  enabled: true
  protected_paths:
    - /dashboard
    - path: /admin
      roles: [admin]
```

All `.pars` handlers and static files under `/dashboard/` and `/admin/` are protected.

## Role-Based Access Control

Users have a `role` field that can be `admin` or `editor`. Roles are assigned via CLI.

### Checking Roles in Handlers

Access the user's role in Parsley handlers:

```parsley
if (basil.auth.user) {
    let role = basil.auth.user.role
    if (role == "admin") {
        <p>"You have admin access"</p>
    }
}
```

### Route-Level Role Requirements

In routes mode, add `roles` to individual routes:

```yaml
routes:
  - path: /admin/users
    handler: ./handlers/admin-users.pars
    auth: required
    roles: [admin]
  
  - path: /content/edit
    handler: ./handlers/content-edit.pars
    auth: required
    roles: [admin, editor]
```

### API Module Role Wrappers

For API routes, use `adminOnly()` or `roles()` wrappers:

```parsley
let api = import @std/api

// Admin only
export get = api.adminOnly(fn(req) {
    {users: getAllUsers()}
})

// Specific roles
export post = api.roles(["admin", "editor"], fn(req) {
    {created: createContent(req.body)}
})

// Any authenticated user (default for API routes)
export delete = fn(req) {
    {deleted: true}
}
```

### Assigning Roles via CLI

```bash
# Create admin user
basil users create --name "Admin" --role admin

# Create editor
basil users create --name "Editor" --role editor

# List users with roles
basil users list
```

### Public Routes Under Protected Paths

Make specific routes public even under a protected prefix:

```yaml
auth:
  enabled: true
  protected_paths:
    - /admin

routes:
  - path: /admin/login
    handler: ./handlers/admin-login.pars
    auth: none              # Explicitly public
  
  - path: /admin/dashboard
    handler: ./handlers/admin-dashboard.pars
    # No auth: setting = inherits from protected_paths
```

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
