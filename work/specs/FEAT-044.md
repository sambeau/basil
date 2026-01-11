---
id: FEAT-044
title: "CSRF Protection"
status: implemented
priority: high
created: 2025-12-07
implemented: 2025-12-08
author: "@human"
---

# FEAT-044: CSRF Protection

## Summary
Add built-in Cross-Site Request Forgery (CSRF) protection for form submissions. CSRF attacks trick authenticated users into submitting malicious requests. Protection works by embedding a secret token in forms that must match a token stored in a cookie, ensuring the request originated from our site.

## User Story
As a Parsley developer, I want automatic CSRF protection so that my forms are secure against cross-site request forgery without manual implementation.

## Acceptance Criteria
- [x] `basil.csrf.token` returns a CSRF token string for embedding in forms
- [x] `<input type=hidden name=_csrf value={basil.csrf.token}/>` pattern works
- [x] POST/PUT/PATCH/DELETE requests with `auth: required` or `auth: optional` validate CSRF
- [x] Invalid/missing CSRF token returns 403 Forbidden with clear error
- [x] CSRF cookie is HttpOnly, Secure (in prod), SameSite=Strict
- [x] Token regenerated per session (not per request) for usability
- [x] AJAX requests can send token via `X-CSRF-Token` header as alternative
- [x] API routes (type: api) skip CSRF validation (use API keys instead)
- [x] Documentation updated with form example

## Design Decisions

- **Double-submit cookie pattern**: Token stored in cookie AND submitted in form/header. Server verifies they match. Simpler than session-based CSRF since it's stateless
- **Per-session tokens**: Regenerate on login, not per-request. Per-request breaks back button and multiple tabs
- **Auto-validate on auth routes**: If a route has auth, it likely has forms worth protecting
- **Skip for API routes**: APIs use API keys or bearer tokens, not cookies, so CSRF doesn't apply
- **Header alternative**: SPAs/AJAX can read the token from a meta tag and send via header
- **Clear error message**: 403 should say "CSRF token invalid" in dev mode for debugging

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### API Design

**In forms:**
```parsley
<form method=POST action="/submit">
    <input type=hidden name=_csrf value={basil.csrf.token}/>
    <input type=text name=email/>
    <button>Submit</button>
</form>
```

**For AJAX/SPA:**
```html
<!-- In head -->
<meta name=csrf-token content={basil.csrf.token}/>
```
```javascript
// In JavaScript
fetch('/api/data', {
    method: 'POST',
    headers: {
        'X-CSRF-Token': document.querySelector('meta[name=csrf-token]').content
    },
    body: JSON.stringify(data)
})
```

**Token properties:**
```parsley
basil.csrf.token     // "a1b2c3d4e5f6..." (32+ char random string)
```

### How It Works

1. **On first request**: If no CSRF cookie exists, generate a random token and set it:
   ```
   Set-Cookie: _csrf=<token>; HttpOnly; Secure; SameSite=Strict; Path=/
   ```

2. **On form render**: `basil.csrf.token` returns the token from the cookie (or generates new one)

3. **On form submit**: Middleware checks:
   - Cookie `_csrf` exists
   - Form field `_csrf` OR header `X-CSRF-Token` matches cookie value
   - If mismatch → 403 Forbidden

4. **On login**: Regenerate token to prevent session fixation

### Validation Rules

| Route Config | Method | CSRF Check |
|--------------|--------|------------|
| `auth: required` | GET/HEAD/OPTIONS | ❌ Skip |
| `auth: required` | POST/PUT/PATCH/DELETE | ✅ Validate |
| `auth: optional` | POST/PUT/PATCH/DELETE | ✅ Validate |
| `type: api` | Any | ❌ Skip |
| No auth | Any | ❌ Skip |

### Affected Components
- `server/handler.go` — Add `basil.csrf.token` to context
- `server/csrf.go` (new) — Token generation, validation middleware
- `server/server.go` — Wire up CSRF middleware for auth routes
- `auth/middleware.go` — May need to regenerate token on login

### Dependencies
- Depends on: FEAT-043 (Cookies) — needs cookie support to store CSRF token
- Blocks: None

### Edge Cases & Constraints
1. **Multiple tabs**: Same token works across tabs (per-session, not per-page)
2. **Back button**: Token doesn't change, so cached forms still work
3. **Token rotation on login**: Prevents session fixation attacks
4. **File uploads**: Multipart forms include `_csrf` field normally
5. **JSON bodies**: Check `X-CSRF-Token` header since JSON doesn't have form fields
6. **Subdomain isolation**: Cookie has `SameSite=Strict` and exact domain

### Error Response

**Dev mode (403):**
```html
<h1>403 Forbidden</h1>
<p>CSRF token validation failed.</p>
<ul>
  <li>Expected token from cookie: abc123...</li>
  <li>Received token from form: (missing)</li>
</ul>
<p>Make sure your form includes: &lt;input type=hidden name=_csrf value={basil.csrf.token}/&gt;</p>
```

**Production mode (403):**
```
403 Forbidden
```

## Implementation Notes

**Implementation Date:** 2025-12-08

**Files Changed:**
- `server/csrf.go` (new) — Token generation, validation middleware, cookie management
- `server/csrf_test.go` (new) — Comprehensive tests for all CSRF functionality
- `server/handler.go` — Added CSRF token to basil context, set cookie in ServeHTTP
- `server/api.go` — Pass empty CSRF token (API routes don't use CSRF)
- `server/server.go` — Added csrfMW field, apply CSRF middleware for auth routes
- `docs/parsley/reference.md` — Added basil.csrf section
- `docs/parsley/CHEATSHEET.md` — Added CSRF Protection section

**Key Design Points:**
- Double-submit cookie pattern (stateless)
- 32-byte random token (64 hex characters)
- Constant-time comparison to prevent timing attacks
- Cookie: `_csrf`, HttpOnly, SameSite=Strict, Secure in production
- Form field: `_csrf`
- Header: `X-CSRF-Token` (for AJAX)

## Related
- Depends on: FEAT-043 (Cookies)
