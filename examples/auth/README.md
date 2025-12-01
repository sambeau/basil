# Authentication Example

This example demonstrates Basil's built-in passkey authentication.

## Features

- Passkey-based registration and login (no passwords!)
- Protected routes
- User session management
- Recovery codes

## Running

```bash
cd examples/auth
basil --dev
```

Then visit http://localhost:8080

## Pages

- **/** - Homepage (public)
- **/signup** - Create an account
- **/login** - Sign in
- **/dashboard** - Protected page (requires auth)
- **/logout** - Sign out

## How It Works

### Registration

The `<PasskeyRegister/>` component handles WebAuthn registration. When a user signs up:

1. They enter their name (and optionally email)
2. Their browser prompts to create a passkey
3. The passkey is stored securely
4. They receive recovery codes (save these!)

### Login

The `<PasskeyLogin/>` component handles authentication:

1. User clicks "Sign in"
2. Browser prompts for passkey selection
3. Session cookie is set
4. User is redirected to dashboard

### Protected Routes

Routes can be protected with `auth: required`:

```yaml
routes:
  - path: /dashboard
    handler: ./handlers/dashboard.pars
    auth: required
```

Unauthenticated users get a 401 response.

### `request.user`

In authenticated handlers, access user info via:

```parsley
request.user.id      // "usr_abc123"
request.user.name    // "Sam"
request.user.email   // "sam@example.com" or ""
request.user.created // timestamp
```

## Managing Users

Use the CLI to manage users:

```bash
basil users list
basil users show usr_abc123
basil users delete usr_abc123
basil users reset usr_abc123  # New recovery codes
```
