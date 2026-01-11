---
id: FEAT-043
title: "Cookie Support"
status: implemented
priority: high
created: 2025-12-07
author: "@human"
---

# FEAT-043: Cookie Support

## Summary
Add the ability to read and set HTTP cookies from Parsley handlers. Cookies are essential for implementing features like "remember me", user preferences, tracking consent, and CSRF tokens. Currently there is no way to access or manipulate cookies from Parsley code.

## User Story
As a Parsley developer, I want to read and set cookies so that I can persist client-side state across requests without requiring authentication.

## Acceptance Criteria
- [x] `basil.http.request.cookies` is a dict of cookie name → value (strings)
- [x] `basil.http.response.cookies` can be assigned to set cookies
- [x] Cookie options supported: `value`, `maxAge`, `expires`, `path`, `domain`, `secure`, `httpOnly`, `sameSite`
- [x] Setting a cookie with `maxAge: 0` or past `expires` deletes it
- [x] Cookies are properly URL-encoded/decoded (via Go's http package)
- [x] HttpOnly and Secure default to `true` in production mode
- [x] Documentation updated

## Design Decisions

- **Read as simple dict**: Request cookies are just `{name: value}` — no need for metadata on read since browsers don't send it
- **Write as dict of options**: Response cookies use `{name: {value, maxAge, ...}}` for full control
- **Secure defaults**: In production, default to `httpOnly: true`, `secure: true`, `sameSite: "Lax"` to prevent common vulnerabilities
- **No automatic JSON**: Cookie values are strings. Use `JSON.stringify()`/`JSON.parse()` if you need objects
- **Duration for maxAge**: Accept Parsley duration literals like `@30d` for ergonomics

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### API Design

**Reading cookies:**
```parsley
// All cookies as dict
let cookies = basil.http.request.cookies
let theme = cookies.theme ?? "light"

// Direct access
let sessionId = basil.http.request.cookies.session_id
```

**Setting cookies:**
```parsley
// Simple value (uses secure defaults)
basil.http.response.cookies.theme = "dark"

// With options
basil.http.response.cookies.remember_token = {
    value: token,
    maxAge: @30d,           // Duration literal
    path: "/",
    httpOnly: true,
    secure: true,
    sameSite: "Strict"
}

// Delete a cookie
basil.http.response.cookies.old_cookie = {value: "", maxAge: @0s}
```

**Cookie options:**
| Option | Type | Default (prod) | Default (dev) | Description |
|--------|------|----------------|---------------|-------------|
| `value` | String | required | required | Cookie value |
| `maxAge` | Duration | session | session | How long until expiry |
| `expires` | DateTime | — | — | Absolute expiry (alternative to maxAge) |
| `path` | String | `"/"` | `"/"` | URL path scope |
| `domain` | String | — | — | Domain scope |
| `secure` | Bool | `true` | `false` | HTTPS only |
| `httpOnly` | Bool | `true` | `true` | No JavaScript access |
| `sameSite` | String | `"Lax"` | `"Lax"` | `"Strict"`, `"Lax"`, or `"None"` |

### Affected Components
- `server/handler.go` — Parse cookies into `basil.http.request.cookies`, extract response cookies
- `server/handler.go` (`buildRequestContext`) — Add cookies to request context
- `server/handler.go` (`extractResponseMeta`) — Read cookies from response and set headers
- `pkg/parsley/evaluator/evaluator.go` — May need duration-to-seconds conversion

### Dependencies
- Depends on: None
- Blocks: FEAT-044 (CSRF Protection) — needs cookies to store CSRF token

### Edge Cases & Constraints
1. **Cookie size limit** — Browsers limit ~4KB per cookie. Don't validate in Basil, let browser handle
2. **Special characters** — Values must be URL-encoded if they contain special chars
3. **Multiple cookies same name** — Last one wins (standard HTTP behavior)
4. **Expired cookies** — Setting `maxAge: @0s` or past `expires` tells browser to delete
5. **SameSite=None requires Secure** — Automatically sets `secure: true` when `sameSite: "None"` (auto-fix rather than error)

## Implementation Notes

- URL encoding/decoding handled by Go's `net/http` package (no custom encoding)
- SameSite=None auto-fixes by setting Secure=true (better UX than erroring)
- Duration dicts use `totalSeconds` field for accurate conversion
- Dev mode detected via `h.server.config.Server.Dev`
- Cookie helpers: `buildCookie()`, `durationToSeconds()` in server/handler.go
- Tests in server/cookies_test.go

## Related
- Blocks: FEAT-044 (CSRF Protection)
