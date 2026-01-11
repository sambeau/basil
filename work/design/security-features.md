# Security Features Design

**Date:** 2025-12-07  
**Status:** Draft  
**Purpose:** Design security features needed for production-ready Basil applications.

## Overview

Web application security has several layers. Basil already handles some well (XSS escaping, SQL injection prevention, security headers, rate limiting). This document covers the remaining gaps.

---

## 0. Batteries-Included Security (Automatic Protection)

**Goal:** Protect novice developers doing naïve things without requiring any security knowledge or configuration. A developer should be able to write straightforward code and get production-grade security by default.

### What We Already Provide Automatically

| Protection | How It Works | Developer Effort |
|------------|--------------|------------------|
| **XSS Prevention** | Tag content is auto-escaped: `<p>{userInput}</p>` is safe | None — just works |
| **SQL Injection** | Parameterized queries: `db <=?=> "SELECT * WHERE id = ?" [id]` | None if using operators |
| **Security Headers** | HSTS, X-Frame-Options, X-Content-Type-Options, etc. | None — on by default |
| **HTTPS Enforcement** | Auto Let's Encrypt in production, redirects HTTP→HTTPS | None — automatic |
| **Rate Limiting** | Built-in rate limiter prevents brute force | None — on by default |
| **Secure Cookies** | HttpOnly, Secure, SameSite defaults in production | None — secure by default |
| **Path Traversal** | Site mode blocks `../` and dotfile access | None — built into router |

### What We Should Add (Automatic, No Config)

#### Auto-CSRF for Forms

**Current gap:** Developer must remember to add `{basil.csrf.token}` to every form.

**Proposed:** Automatically inject CSRF tokens into all `<form method=POST>` tags.

```parsley
// Developer writes:
<form method=POST action="/submit">
    <input type=text name=email/>
    <button>Submit</button>
</form>

// Basil automatically renders:
<form method=POST action="/submit">
    <input type="hidden" name="_csrf" value="abc123..."/>
    <input type=text name=email/>
    <button>Submit</button>
</form>
```

**Implementation:** During HTML rendering, if tag is `<form>` with method POST/PUT/PATCH/DELETE, auto-inject the hidden input as first child.

**Opt-out:** For forms that genuinely don't need CSRF (rare), allow:
```parsley
<form method=POST action="/webhook" csrf={false}>
```

#### Auto-Validate CSRF on POST

**Current gap:** Developer must configure `auth: required` to get CSRF validation.

**Proposed:** Validate CSRF on ALL non-GET requests, regardless of auth config.

- If form has `_csrf` field → validate it
- If missing and method is POST/PUT/PATCH/DELETE → reject with 403
- Exception: Requests with `Authorization` header (API calls) skip CSRF
- Exception: Requests with `Content-Type: application/json` from same-origin skip CSRF (SameSite cookies protect this)

This protects the developer who forgets to add `auth: required` to a form handler.

#### Secure Cookie Defaults

**Current:** When FEAT-043 is implemented, use these defaults automatically:

| Setting | Dev Mode | Production |
|---------|----------|------------|
| `httpOnly` | `true` | `true` |
| `secure` | `false` | `true` |
| `sameSite` | `"Lax"` | `"Lax"` |
| `path` | `"/"` | `"/"` |

Developer can override, but the naïve case is secure.

#### Input Sanitization Hints

**Not automatic (too opinionated)** but provide helpers:

```parsley
let {sanitize} = import("std/html")

// Remove all HTML tags
let clean = sanitize(userInput)

// Allow safe subset (b, i, a, p)
let clean = sanitize(userInput, {allow: ["b", "i", "a", "p"]})
```

### What We Should Add (Opt-In, But Easy)

#### Content Security Policy

**Current:** Manual CSP string in config. Easy to get wrong.

**Proposed:** Provide sensible presets:

```yaml
security:
  csp: strict    # Very restrictive, no inline
  csp: standard  # Allows 'self', nonces for inline
  csp: relaxed   # Allows 'unsafe-inline' (not recommended)
  csp: "..."     # Custom string (advanced)
```

Most developers should use `strict` or `standard` without understanding CSP details.

#### CORS Presets

```yaml
cors: public      # origins: "*", no credentials (public API)
cors: private     # No CORS (same-origin only, default)
cors:             # Custom config (advanced)
  origins: [...]
```

### Security Checklist for Basil

When a developer does these naïve things, are they protected?

| Naïve Action | Protection | Status |
|--------------|------------|--------|
| Echo user input in HTML | XSS auto-escape | ✅ Done |
| Put user input in SQL | Parameterized queries | ✅ Done |
| Create a login form | CSRF auto-inject + validate | 🔲 Proposed |
| Set a cookie | Secure defaults | 🔲 Proposed |
| Deploy to internet | HTTPS auto, security headers | ✅ Done |
| Forget rate limiting | Built-in rate limiter | ✅ Done |
| Allow file uploads | Size limits, type checking | 🔲 TODO |
| Accept JSON POST | Same-origin enforced | 🔲 Partially (SameSite) |

### Philosophy: Secure by Default, Escape Hatches Available

1. **The default is secure** — No config = maximum reasonable protection
2. **Warnings for risky config** — If developer disables protection, warn in logs
3. **Explicit opt-out** — `csrf={false}`, not `csrf={true}` to enable
4. **Dev mode is lenient** — Less strict for local development ease
5. **Production mode is strict** — Auto-enable everything in production

### Implementation Priority

| Feature | Protects Against | Effort | Priority |
|---------|------------------|--------|----------|
| Auto-inject CSRF in forms | CSRF attacks | Medium | High |
| Auto-validate CSRF on POST | CSRF attacks | Medium | High |
| Secure cookie defaults | Session hijacking | Low | High |
| CSP presets | XSS, injection | Low | Medium |
| File upload limits | DoS, storage | Medium | Medium |
| Sanitize helper | XSS in rich text | Low | Low |

---

## 1. Cookie Support (FEAT-043)

**Status:** Spec complete, ready for implementation

Cookies are foundational — needed for CSRF, sessions, and general state management.

See `docs/specs/FEAT-043.md` for full specification.

**Summary:**
```parsley
// Read
let theme = basil.http.request.cookies.theme

// Write
basil.http.response.cookies.remember = {
    value: token,
    maxAge: @30d,
    httpOnly: true,
    secure: true,
    sameSite: "Strict"
}
```

---

## 2. CSRF Protection (FEAT-044)

**Status:** Spec complete, ready for implementation (depends on FEAT-043)

### Core Protection

See `docs/specs/FEAT-044.md` for full specification.

**Summary:**
```parsley
<form method=POST>
    <input type=hidden name=_csrf value={basil.csrf.token}/>
    ...
</form>
```

### Enhanced: `<Form>` Component

Beyond the basic `basil.csrf.token`, we could provide a higher-level `<Form>` component that handles CSRF automatically:

```parsley
// Instead of:
<form method=POST action="/submit">
    <input type=hidden name=_csrf value={basil.csrf.token}/>
    <input type=text name=email/>
    <button>Submit</button>
</form>

// Use:
<Form action="/submit">
    <input type=text name=email/>
    <button>Submit</button>
</Form>
```

**What `<Form>` provides:**
1. Auto-injects CSRF token for POST/PUT/PATCH/DELETE
2. Optional: AJAX submission with `ajax={true}`
3. Optional: Loading state with `loading={<Spinner/>}`
4. Optional: Validation feedback container

**Implementation options:**

**Option A: Built into Basil core**
```parsley
// Form is a special tag that Basil recognizes
<Form method=POST action="/submit">
    ...
</Form>
```
Pro: Always available, consistent behavior  
Con: Adds complexity to core, one more thing to maintain

**Option B: Standard library component**
```parsley
let {Form} = import("std/html")

<Form method=POST action="/submit">
    ...
</Form>
```
Pro: Opt-in, can evolve separately, good example of component patterns  
Con: Requires import, might be forgotten

**Option C: Documentation pattern only**
Just document the `<input name=_csrf value={basil.csrf.token}/>` pattern.

Pro: Simple, no new code  
Con: Developers must remember, easy to forget

**Recommendation:** Option B — standard library component. It's the right balance of convenience and simplicity. Creates a good precedent for `std/html` helpers.

### `std/html` Library Scope

If we're adding `<Form>`, consider what else belongs in `std/html`:

```parsley
let {Form, Link, Meta, Script} = import("std/html")

// Form with CSRF
<Form action="/submit">...</Form>

// Link with active state detection
<Link href="/dashboard" activeClass="current">Dashboard</Link>

// Meta tags helper
<Meta title="Page Title" description="..." />

// Script with nonce for CSP
<Script src="/app.js"/>
```

This could grow organically based on need. Start with just `Form`.

---

## 3. CORS Configuration (NEW)

### What is CORS?

Cross-Origin Resource Sharing controls which external domains can make requests to your API. Browsers enforce this for JavaScript `fetch()` calls.

**Without CORS:** Only same-origin requests work  
**With CORS:** Specified origins can make requests

### When You Need CORS

- API consumed by a separate frontend (e.g., React SPA on different domain)
- Public API that others can call from their sites
- Microservices on different subdomains

### When You DON'T Need CORS

- Traditional server-rendered apps (Basil serves both HTML and handles forms)
- API only called from same origin
- Mobile apps (not subject to browser CORS)

### Design Options

**Option A: Config-based**
```yaml
# basil.yaml
cors:
  origins:
    - https://app.example.com
    - https://admin.example.com
  methods: [GET, POST, PUT, DELETE]
  headers: [Content-Type, Authorization]
  credentials: true
  maxAge: 86400
```

Pro: Declarative, applies globally, no code needed  
Con: Less flexible, can't vary per route

**Option B: Per-route config**
```yaml
routes:
  - path: /api/*
    handler: ./api.pars
    cors:
      origins: ["https://app.example.com"]
      credentials: true
```

Pro: Fine-grained control  
Con: Verbose, repetitive

**Option C: Parsley function for manual control**
```parsley
// In handler
cors({
    origin: "https://app.example.com",
    credentials: true
})

// Or check and set headers manually
if (basil.http.request.headers.Origin == "https://app.example.com") {
    basil.http.response.headers["Access-Control-Allow-Origin"] = "https://app.example.com"
    basil.http.response.headers["Access-Control-Allow-Credentials"] = "true"
}
```

Pro: Maximum flexibility  
Con: Easy to get wrong, verbose

**Option D: Hybrid — config with Parsley override**
```yaml
# basil.yaml - global defaults
cors:
  origins: ["https://app.example.com"]
```
```parsley
// In specific handler - override or extend
cors({origin: "*"})  // Override for this route
```

**Recommendation:** Option D (hybrid). Most apps need simple global config, but some routes need different rules (e.g., public API endpoint).

### CORS Implementation Details

CORS involves two types of requests:

**Simple requests** (GET, POST with simple content-types):
```
Browser: GET /api/data
         Origin: https://app.example.com

Server:  200 OK
         Access-Control-Allow-Origin: https://app.example.com
```

**Preflight requests** (PUT, DELETE, custom headers):
```
Browser: OPTIONS /api/data
         Origin: https://app.example.com
         Access-Control-Request-Method: DELETE
         Access-Control-Request-Headers: Authorization

Server:  204 No Content
         Access-Control-Allow-Origin: https://app.example.com
         Access-Control-Allow-Methods: GET, POST, PUT, DELETE
         Access-Control-Allow-Headers: Authorization
         Access-Control-Max-Age: 86400
```

The server must handle OPTIONS requests automatically when CORS is enabled.

### Proposed CORS Config

```yaml
# basil.yaml
cors:
  # Which origins can make requests
  # Can be: "*" (any), single origin, or list
  origins:
    - https://app.example.com
    - https://staging.example.com
  
  # Which methods are allowed (default: simple methods)
  methods: [GET, POST, PUT, PATCH, DELETE]
  
  # Which request headers are allowed
  headers: [Content-Type, Authorization, X-Requested-With]
  
  # Which response headers the browser can access
  expose: [X-Total-Count, X-Page-Count]
  
  # Allow credentials (cookies, auth headers)
  # Note: Can't use with origins: "*"
  credentials: true
  
  # How long browser can cache preflight response (seconds)
  maxAge: 86400  # 24 hours
```

### Parsley Override

```parsley
// Disable CORS for this handler (even if globally enabled)
cors(false)

// Enable with specific settings
cors({
    origin: "*",           // Allow any origin (no credentials)
    methods: ["GET"],      // Read-only
    maxAge: @1h
})

// Dynamic origin based on request
let origin = basil.http.request.headers.Origin
if (allowedOrigins.includes(origin)) {
    cors({origin: origin, credentials: true})
}
```

---

## 4. Content Security Policy (CSP) Enhancement

Basil already supports CSP via config:
```yaml
security:
  csp: "default-src 'self'; script-src 'self' 'unsafe-inline'"
```

### The Nonce Problem

Inline scripts require either `'unsafe-inline'` (security risk) or a nonce:
```html
<script nonce="abc123">
    // This script is allowed
</script>
```

The nonce must be random per-request and match the CSP header.

### Proposed Enhancement

**In config:**
```yaml
security:
  csp: "default-src 'self'; script-src 'self' 'nonce-{nonce}'"
```

**In Parsley:**
```parsley
// basil.csp.nonce is auto-generated per request
<script nonce={basil.csp.nonce}>
    alert('Safe inline script');
</script>
```

Basil would:
1. Generate random nonce per request
2. Replace `{nonce}` in CSP header with actual value
3. Expose as `basil.csp.nonce`

**With std/html:**
```parsley
let {Script} = import("std/html")

// Automatically includes nonce
<Script>
    alert('Safe inline script');
</Script>
```

---

## 5. Security Headers (Already Implemented)

For reference, Basil already handles these in `security:` config:

| Header | Default | Purpose |
|--------|---------|---------|
| `Strict-Transport-Security` | Enabled in prod | Force HTTPS |
| `X-Content-Type-Options` | `nosniff` | Prevent MIME sniffing |
| `X-Frame-Options` | `DENY` | Prevent clickjacking |
| `X-XSS-Protection` | `1; mode=block` | Legacy XSS filter |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Control referrer |
| `Content-Security-Policy` | Configurable | Control resource loading |
| `Permissions-Policy` | Configurable | Control browser features |

---

## Implementation Roadmap

### Phase 1: Foundation (Required)
1. **FEAT-043: Cookies** — Everything else depends on this
2. **FEAT-044: CSRF** — Required for secure forms

### Phase 2: API Support
3. **FEAT-047: CORS** — Required for API-first applications
4. **CSP nonces** — Better inline script security

### Phase 3: Developer Experience
5. **`std/html` library** — `<Form>`, `<Script>` with auto-security
6. **Security audit helper** — `basil check-security` command?

---

## Summary

| Feature | Priority | Complexity | Depends On |
|---------|----------|------------|------------|
| Cookies (FEAT-043) | Critical | Medium | — |
| CSRF (FEAT-044) | Critical | Medium | Cookies |
| CORS (FEAT-047) | High | Medium | — |
| CSP nonces | Medium | Low | — |
| `std/html` Form | Medium | Low | CSRF |

The security foundation is cookies → CSRF. CORS is independent and can be done in parallel. CSP nonces and `std/html` are polish.
