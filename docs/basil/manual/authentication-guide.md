---
id: man-bas-authentication-guide
title: "The Authentication Guide"
system: basil
type: guide
name: authentication-guide
created: 2026-07-14
version: 1.0.0-alpha.4
author: "@sam"
keywords:
  - auth
  - guide
  - passkeys
  - webauthn
  - sessions
  - roles
  - protected paths
  - recovery codes
  - api keys
  - email verification
  - csrf
  - endpoints
---

# The Authentication Guide

The [Authentication page](authentication.md) covers the essentials: enabling auth, the components, and protecting routes. This guide is the deep dive — how passkeys actually work, where everything is stored, and the patterns for roles, recovery, API keys, and email verification.

## How Passkey Sign-In Works

A passkey is a public/private key pair. The private key lives on the visitor's device (or syncs through their iCloud/Google account); your server only ever stores the public half. There is nothing secret in your database, so there is nothing to leak.

When someone clicks the Register or Login button, the component's JavaScript runs a two-step dance with the server:

1. **Begin** — the browser asks the server for a challenge (`POST /__auth/register/begin` or `/__auth/login/begin`).
2. **Finish** — the browser has the device sign the challenge with the private key and sends the result back (`/__auth/register/finish` or `/__auth/login/finish`). The server verifies the signature, creates a session, and sets the cookie.

Challenges expire after five minutes, so a stalled dialog just means clicking the button again.

Basil uses *discoverable credentials*: the browser itself shows the visitor which passkeys they have for your domain, so login is one click — no username field, no "which account was that?".

### Passkeys Are Bound to Your Domain

A passkey is tied to the domain it was created on (this is what makes it phishing-proof — a fake site on another domain can't even ask for it). Basil uses `server.host` from your config as the domain, or `localhost` in dev mode. Two consequences:

- Passkeys created on `localhost` in dev won't work on your production domain (and vice versa). Register separately in each environment.
- If you change `server.host`, existing passkeys stop matching. Choose your domain before inviting users.

Browsers also require a *secure context* for WebAuthn: HTTPS in production, though plain HTTP is fine on `localhost`.

## Where Everything Lives

Auth data is kept in its own SQLite database, `.basil-auth.db`, in the same folder as `basil.yaml` — deliberately separate from your app database, so handler code can never accidentally query it. It holds users, credentials (public keys), sessions, hashed recovery codes, hashed API keys, and the email-verification bookkeeping.

Two things to know:

- **A production server won't start** with `auth.enabled: true` until this database exists. `basil users create` creates it — that's why the quick start makes a user first. **`basil --dev` creates it for you**, empty, so a clone of a site runs on a laptop with no setup: the release's config comes from the server, where auth is always on, and a dev partner has no data directory to have been given one. Nobody is signed in until you make a user, which is the same answer production gives.
- The CLI and the server share it safely (it runs in WAL mode, so you'll see `.basil-auth.db-wal` and `.basil-auth.db-shm` alongside). Keep all three out of git; do include them in backups.

## Sessions

Finishing a registration or login creates a server-side session and sets a cookie:

| Property | Value |
|----------|-------|
| Cookie name | `__basil_session` |
| Contents | A random 256-bit token (the session lives server-side) |
| Flags | `HttpOnly`, `SameSite=Lax`, and `Secure` in production |
| Lifetime | `auth.session_ttl` (default 24h), fixed from sign-in — no sliding renewal |

When the session expires, the visitor signs in again — one click with a passkey, so longer TTLs buy less than they would with passwords. `session_ttl` takes Go durations: `24h`, `168h` for a week (hours are the largest unit Go understands — there is no `7d`).

Logging out deletes the session server-side and clears the cookie. Deleting a user deletes their sessions too.

> **Not the same thing:** `basil.session` (the [session store](configuration.md) for your own data, cookie `_basil_session`) and auth sessions (`__basil_session`) are separate systems. Enabling auth doesn't touch your session data, and vice versa.

## Users and Roles

Basil has exactly two roles: **`admin`** and **`editor`**. There's no role editor, no permission matrix — just "runs the place" and "works here".

How roles are assigned:

- The **first user created via the CLI is always an admin** (a `--role` saying otherwise is ignored, politely).
- Later CLI users default to `editor`; pass `--role admin` to override.
- **Everyone who signs up through the web is an editor.** Always. Promotion happens via `basil users set-role <id> admin`.

### Bootstrapping Your Admin

A CLI-created user has no passkey yet — they exist, but can't sign in. The trick: when someone registers with an email that matches a passkey-less user, the new passkey attaches to that existing account instead of creating a new one. So:

```bash
basil users create --name "You" --email you@example.com   # first user → admin
```

Then, with `registration: open`, visit your signup page and register with the same email. The passkey lands on your admin account. Once you're in, you can set `registration: closed` if the site isn't meant to have public signups. (The registration endpoint always admits the *very first* user even when closed — but attaching a passkey to an existing account needs registration open at that moment.)

If an email already has a passkey, registering with it again is refused ("Email already registered") — one account per email.

## Protecting Pages

### Protected Paths

`protected_paths` guards URL prefixes, in both site mode and routes mode:

```yaml
auth:
  enabled: true
  protected_paths:
    - /dashboard              # any signed-in user
    - path: /admin
      roles: [admin]          # admins only
    - path: /content
      roles: [admin, editor]  # either role
```

A prefix matches itself and everything under it: `/dashboard` protects `/dashboard`, `/dashboard/`, and `/dashboard/reports/2026`. It does *not* match `/dashboard-widgets` — matching is by path segment, not string prefix.

### What Blocked Visitors See

- **Signed out, HTML request** — a 302 redirect to `login_path` (default `/login`) with the original destination in `?next=`: visiting `/dashboard/reports` while signed out lands on `/login?next=/dashboard/reports`.
- **Signed out, API request** — a 401 JSON error instead of a redirect. A request counts as API if its path starts with `/api/`, it sends `Accept: application/json`, or it posts JSON.
- **Signed in, wrong role** — a 403.

Your login page can honour `?next=` by passing it through to the component:

```parsley
let next = @params.next ?? "/"
<basil.auth.Login button_text="Sign in" redirect={next}/>
```

### Per-Route Auth in Routes Mode

Routes can declare their own requirements:

```yaml
routes:
  - path: /dashboard
    handler: ./handlers/dashboard.pars
    auth: required            # 401 if not signed in

  - path: /profile
    handler: ./handlers/profile.pars
    auth: optional            # public, but basil.auth.user is set when signed in

  - path: /admin/reports
    handler: ./handlers/reports.pars
    roles: [admin]            # implies sign-in; wrong role → 403

  - path: /admin/login
    handler: ./handlers/admin-login.pars
    auth: none                # explicitly public, even under a protected prefix
```

- `auth: required` returns a bare 401 rather than redirecting — it suits API-ish routes. For pages, prefer `protected_paths` (friendly redirect) or `roles:`.
- `roles:` on a route requires sign-in *and* one of the listed roles, no `auth:` needed.
- `auth: none` is the escape hatch: it exempts a route from `protected_paths`, while still filling in `basil.auth.user` for signed-in visitors.
- Routes with no `auth:` setting fall under whatever `protected_paths` says; everything else is public.

### Checking Roles in Handlers

For finer-grained decisions, check the role yourself:

```parsley
if (basil.auth.user != null && basil.auth.user.role == "admin") {
    <a href="/admin">"Admin panel"</a>
}
```

### API Modules

API-mode handlers (routes under `/api/` or `type: api`) flip the default: **every handler requires a signed-in user unless you say otherwise.** Wrappers from `@std/api` adjust it:

```parsley
let api = import @std/api

// Anyone
export get = api.public(fn(req) {
    {posts: listPosts()}
})

// Admins only
export delete = api.adminOnly(fn(req) {
    {deleted: deleteEverything()}
})

// Specific roles
export post = api.roles(["admin", "editor"], fn(req) {
    {created: createPost(req.body)}
})

// No wrapper: any signed-in user
export put = fn(req) {
    {updated: true}
}
```

Failures come back as JSON: 401 without a user, 403 for a missing role. The signed-in user is available as `req.user` (`id`, `name`, `email`, `role`).

## Forms and CSRF

When a route declares `auth: required` or `auth: optional` in routes mode, Basil validates a CSRF token on every mutating request (POST, PUT, PATCH, DELETE) to that route. Include it in your forms:

```parsley
<form method="POST" action="/profile">
    <input type="hidden" name="_csrf" value={basil.csrf.token}/>
    <input name="bio"/>
    <button>"Save"</button>
</form>
```

For `fetch()` calls, send it as a header instead: `X-CSRF-Token`. Forget the token and you'll get a 403 — in dev mode, with an error page that tells you exactly which of the two tokens went missing. A `fetch()` caller gets that diagnostic as JSON (`error.details`) rather than HTML. In production the same failure renders the standard [403 page](configuration.md#error-pages), with no token details.

## Recovery Codes

Passkeys are wonderful right up until the phone is in a river. At registration, every user gets **eight one-time recovery codes** in the form `XXXX-XXXX-XXXX` (no ambiguous characters, so no squinting at `0` vs `O`). They're bcrypt-hashed at rest and each works once.

### Showing Them at Signup

Set `recovery_page` on the Register component:

```parsley
<basil.auth.Register
    button_text="Create account"
    redirect="/dashboard"
    recovery_page="/recovery-codes"
/>
```

After registration the codes are put in `sessionStorage` (`basil_recovery_codes`, a JSON array, and `basil_recovery_user`) and the browser goes to your recovery page. That page should display the codes, offer a copy button, and clear the storage once the user moves on — `examples/auth/handlers/recovery-codes.pars` in the repo is a complete, stealable implementation.

### Using a Code

There's no built-in recovery UI, but the endpoint is one `fetch` away — `POST /__auth/recover` with the user's email and one code:

```parsley
<form id="recover">
    <input name="email" type="email" placeholder="you@example.com"/>
    <input name="code" placeholder="XXXX-XXXX-XXXX"/>
    <button>"Recover account"</button>
</form>
<script>
document.getElementById('recover').addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = new FormData(e.target);
    const res = await fetch('/__auth/recover', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({email: form.get('email'), code: form.get('code')})
    });
    if (res.ok) window.location.href = '/';
    else alert('That code didn\'t work.');
});
</script>
```

A valid code signs the user in (codes are case-insensitive and the dashes are optional). The response includes `remaining_codes`, worth surfacing when it gets low. When someone runs out, `basil users reset <id>` issues a fresh set of eight and retires the old ones.

## API Keys

API keys authenticate machines rather than browsers — today that means the built-in [Git server](git.md). Each key belongs to a user and acts with that user's role.

```bash
basil apikey create --user usr_abc123 --name "MacBook Git"
```

Keys look like `bsl_live_…` and are **shown once, at creation** — only a bcrypt hash is stored, plus a display prefix so `basil apikey list` can show you which key is which. For Git, the key goes in the password slot of HTTP Basic auth (the username is ignored):

```bash
git clone https://anything:bsl_live_...@yourserver.com/.git mysite
```

Revoking is instant and per-key — `basil apikey revoke <id>` — so losing a laptop doesn't mean resetting an account.

## Email Verification

Optionally, Basil can send verification emails through **Mailgun** or **Resend**:

```yaml
auth:
  enabled: true
  email_verification:
    enabled: true
    provider: mailgun            # or "resend"

    mailgun:
      api_key: "${MAILGUN_API_KEY}"    # ${VAR} reads environment variables
      domain: "mg.example.com"
      region: us                 # or "eu"
      from: "noreply@example.com"

    # resend:
    #   api_key: "${RESEND_API_KEY}"
    #   from: "noreply@example.com"

    token_ttl: 1h                # verification link lifetime (default: 1h)
    resend_cooldown: 5m          # minimum gap between emails (default: 5m)
    max_sends_per_day: 10        # per user (default: 10)

    template_vars:
      site_name: "My App"        # used in the email (default: "Basil")
      site_url: "https://example.com"
```

With this enabled, anyone who registers with an email gets a verification message automatically. The link (`/__auth/verify-email?token=…`) marks them verified and redirects to `/`. Tokens are 256-bit, single-use, bcrypt-hashed at rest, and expire after `token_ttl`.

### Acting on Verification

Verification state is in the user object, so gating is a handler-side decision:

```parsley
if (basil.auth.user.email_verification_pending) {
    <p>"Please check your inbox and verify your email."</p>
    <button onclick="fetch('/__auth/resend-verification', {method: 'POST'}).then(() => alert('Sent!'))">
        "Resend verification email"
    </button>
} else {
    // verified (or no email on file) — carry on
}
```

Resends are rate-limited server-side: the cooldown, the per-user daily cap, and a per-address daily cap of twice the user limit (so a typo'd address can't be firehosed). The CLI can manage all of it:

```bash
basil auth status <id>                        # verification state + pending tokens
basil auth verify-email <id>                  # mark verified by hand
basil auth resend-verification <id>           # send another (--force skips rate limits)
basil auth reset-verification <id>            # back to unverified
basil auth email-logs [<id>] [--limit N]      # the send audit log
```

### Email Recovery

Once a user's email is verified, they get a second recovery route: `POST /__auth/recover/email` with `{"email": "..."}` emails them a sign-in link. The response is deliberately the same whether or not the address exists (no fishing for registered emails), and unverified addresses are silently skipped. Clicking the link signs them in and redirects to `/?recovery=true`.

## The Endpoints

Everything the components do goes through `/__auth/*` — useful when you'd rather write your own UI:

| Endpoint | Method | Does |
|----------|--------|------|
| `/__auth/register/begin` | POST | Start registration (`{name, email}`) |
| `/__auth/register/finish` | POST | Verify the new passkey; signs in, returns recovery codes |
| `/__auth/login/begin` | POST | Start login |
| `/__auth/login/finish` | POST | Verify the assertion; signs in |
| `/__auth/logout` | POST | Delete the session |
| `/__auth/me` | GET | Current user as JSON, or `{"user": null}` |
| `/__auth/recover` | POST | Sign in with a recovery code (`{email, code}`) |
| `/__auth/recover/email` | POST | Email a recovery link (`{email}`) |
| `/__auth/recover/verify` | GET | Recovery-link target; signs in |
| `/__auth/verify-email` | GET | Verification-link target |
| `/__auth/resend-verification` | POST | Resend verification (must be signed in) |

## Styling the Components

The components ship unstyled, with classes waiting for your CSS:

```css
.basil-auth-register { }   /* the registration form */
.basil-auth-login { }      /* the login container */
.basil-auth-logout { }     /* the logout button/link */
.basil-auth-input { }      /* text inputs */
.basil-auth-button { }     /* buttons */
.basil-auth-error { }      /* error messages (hidden until needed) */
```

Each component also takes a `class` attribute for adding your own.

## Troubleshooting

**"Registration is closed"** — `auth.registration` is `closed` and at least one user exists. Open it, or create users via the CLI.

**"Email already registered"** — that email has an account with a passkey. One account per email; recover the existing one instead.

**Registration or login fails silently** — open the browser console. The usual suspects: not on `localhost` or HTTPS (WebAuthn needs a secure context), the visitor dismissed the passkey prompt, or the domain doesn't match `server.host` (passkeys made on one domain won't work on another).

**"No authentication database found"** at startup — auth is enabled but `.basil-auth.db` doesn't exist yet. Run `basil users create` once. This is a production-mode error only: `basil --dev` creates the database rather than refusing, so seeing it means you are not in dev mode — and on a server it usually means the credentials are gone, which is worth investigating before you make a new empty one.

**403 on a form post** — a CSRF token is missing; add `<input type="hidden" name="_csrf" value={basil.csrf.token}/>` to the form. Dev mode's error page shows which side was missing.

**Verification emails not arriving** — check `basil auth email-logs` for send failures, then the spam folder, then your domain's SPF/DKIM records in the provider dashboard. `basil --check-config` warns about sandbox domains and incomplete provider config.

## See Also

- [Authentication](authentication.md) — the essentials
- [Running Basil](running.md) — the `users`, `apikey`, and `auth` CLI commands
- [Git Deploy](git.md) — API keys at work
- [Configuration](configuration.md) — the full `basil.yaml` reference
- `examples/auth/` in the repo — a complete working login/signup/dashboard site
