# Backlog Investigation Report

**Date:** 2026-03-16
**Scope:** Items #107, #16b, #18, #10, #13, #15, #19, #26, #97, #17, #14, #42, #43, #44

---

## Executive Summary

Fourteen backlog items were investigated in depth. Key findings:

| # | Item | Verdict | Effort |
|---|------|---------|--------|
| **#97** | Named capture groups → dictionaries | **Already done** — move to Completed | 0 (bookkeeping) |
| **#26** | Roles/permissions | **Already done** — move to Completed | 0 (bookkeeping) |
| **#15** | CSRF middleware for site mode | **Security gap** — fix ASAP | ~1–2 hours |
| **#14** | Auth integration in site mode | **Security gap** — design needed | ~3–4 days |
| **#43** | API key expiry flag | Security hygiene — ready to implement | ~0.5–1 day |
| **#42** | API key scopes | Security hardening — needs design | ~2–3 days |
| **#44** | Argon2 for API key hashing | Security hardening — defer | ~2 days |
| **#16b** | Function rest parameters | Small, well-scoped | ~2–3 hours |
| **#18** | CSV upload merge mode | Small, well-scoped | ~3–5 hours |
| **#107** | Unit arithmetic overflow detection | Partially done, gaps remain | ~7–10 hours |
| **#10** | Session auth integration | Medium, architectural plumbing needed | ~1–2 days |
| **#13** | Per-route caching in site mode | Medium, needs design decision | ~2–6 hours |
| **#19** | HTTP-only production mode (behind proxy) | Medium, touches many subsystems | ~10–13 hours |
| **#17** | Standardize locale support | Large, multi-week effort | ~8–12 days |

Two items (#97 and #26) are already implemented and just need to be moved to Completed. Three items (#15, #14, #10) are security gaps — #15 has a trivial fix, #14 needs design but has a clear path, and #10 requires architectural plumbing. Items #42, #43, #44 are API key hardening with #43 being the most ready to implement. The rest range from small enhancements to a multi-week standardization effort.

### Security Items at a Glance

| Priority | Item | Nature | Fix Complexity |
|----------|------|--------|----------------|
| **Fix now** | **#15** — CSRF in site mode | Active vulnerability — mutating requests bypass CSRF | ~5 lines in `site.go` |
| **Fix soon** | **#14** — Auth in site mode | Missing per-handler auth — relies on `protected_paths` not being misconfigured | Design + implementation |
| **Fix soon** | **#10** — Session fixation | Session not regenerated on login/logout | Architectural plumbing |
| **Should do** | **#43** — API key expiry | Keys are immortal — schema supports expiry but CLI doesn't expose it | CLI flag + parameter |
| **Should do** | **#42** — API key scopes | Leaked key has full access — no way to limit blast radius | Schema + enforcement |
| **Nice to have** | **#44** — Argon2 hashing | bcrypt is fine for now; O(n) key scan is the bigger perf issue | Algorithm swap + migration |

---

## Table of Contents

1. [#97 — Named Capture Groups (ALREADY DONE)](#97--named-capture-groups-should-return-dictionaries)
2. [#26 — Roles/Permissions (ALREADY DONE)](#26--rolespermissions)
3. [#15 — CSRF Middleware for Site Mode](#15--csrf-middleware-for-site-mode)
4. [#14 — Auth Integration in Site Mode](#14--auth-integration-in-site-mode)
5. [#13 — Per-Route Caching in Site Mode](#13--per-route-caching-in-site-mode)
6. [#16b — Function Rest Parameters](#16b--function-rest-parameters)
7. [#18 — CSV Upload Merge Mode](#18--csv-upload-merge-mode-for-db)
8. [#107 — Unit Arithmetic Overflow Detection](#107--overflow-detection-for-unit-arithmetic)
9. [#10 — Session Auth Integration](#10--session-auth-integration)
10. [#19 — HTTP-Only Production Mode](#19--http-only-production-mode-behind-proxy)
11. [#17 — Standardize Locale Support](#17--standardize-locale-support-across-stdlib)
12. [#42 — API Key Scopes](#42--api-key-scopes)
13. [#43 — API Key Expiry Flag](#43--api-key-expiry-flag)
14. [#44 — Argon2 for API Key Hashing](#44--argon2-for-api-key-hashing)

---

## #97 — Named Capture Groups Should Return Dictionaries

| Field | Value |
|-------|-------|
| **Source** | Backlog (Enhancement) |
| **Deferred from** | N/A — originated as a standalone enhancement |
| **Status** | ✅ **ALREADY IMPLEMENTED** — bookkeeping error |

### Details

The feature was fully implemented as part of **FEAT-127** (Parsley 1.0 Release Readiness), completed on 2025-02-26. The implementation in `pkg/parsley/evaluator/eval_regex.go` branches on whether a regex has named groups:

- **Without named groups** → returns an `Array` (backward compatible)
- **With named groups** → returns a `Dictionary` via `buildMatchDictionary()`, using Go's `re.SubexpNames()` to construct keys

The dictionary includes key `"0"` for the full match, named groups as their name, and unnamed groups in mixed regexes as numeric string keys. The design avoids breaking changes — only regexes with named groups get the new behavior.

Nine test cases exist in `pkg/parsley/tests/feat127_test.go` covering named groups, single groups, mixed named/unnamed, backward compatibility, no-match returning null, and dot-notation access.

### What's Blocking

Nothing. The entry was simply never moved to Completed. Move to archive.

### Cost

Zero — backlog housekeeping only.

---

## #26 — Roles/Permissions

| Field | Value |
|-------|-------|
| **Source** | FEAT-004 (Authentication) |
| **Deferred from** | FEAT-004 — listed as "Future Considerations (Not MVP)" |
| **Status** | ✅ **ALREADY IMPLEMENTED** — bookkeeping error |

### Details

The literal description — "`request.user.role` and role-based route protection" — is already fully implemented across four layers:

**1. Database:** The `users` table has a `role TEXT NOT NULL DEFAULT 'editor'` column. `SetUserRole()`, `CountAdmins()`, `CreateUserWithRole()` all exist. First user is always admin.

**2. User model:** `auth.User` struct has a `Role` field with constants `RoleAdmin = "admin"` and `RoleEditor = "editor"`.

**3. Route protection (config):** `Route` has `Roles []string` and `ProtectedPath` has `Roles []string`. The `protectedPathMiddleware` in `server.go` checks `user.Role` against `requiredRoles` and returns 403 on mismatch.

**4. API module auth:** `@std/api` provides `api.public()`, `api.adminOnly()`, `api.roles()`, and `api.auth()` wrappers. `enforceAuth` in `api.go` checks role matching with tests for admin-only, role-matching, role-denying, and no-role scenarios.

**5. Parsley access:** `basil.auth.user.role` is populated in `handler.go` and accessible to all handlers.

**6. CLI:** `basil users set-role <id> <role>` and `basil users create --role <role>` exist.

### What Could Be a _New_ Item

What's _not_ implemented (and would warrant a separate backlog entry if desired):

- **Custom roles beyond admin/editor** — currently hardcoded to two roles
- **Fine-grained permissions** — no `can_edit_posts` style permission system
- **Full RBAC with UI** — no admin panel for role/permission management

### What's Blocking

Nothing for the existing scope. The entry should be moved to Completed.

### Cost

Zero for current scope. If custom roles are desired later: ~4–6 hours. Full permissions system: ~2–3 days. Full RBAC with UI: ~1–2 weeks.

---

## #15 — CSRF Middleware for Site Mode

| Field | Value |
|-------|-------|
| **Source** | FEAT-076 |
| **Deferred from** | FEAT-076 — explicitly deferred, unchecked acceptance criterion |
| **Status** | ⚠️ **Security gap** — should be fixed promptly |

### Details

In **routes mode**, CSRF middleware is automatically applied to any non-API route with auth (required or optional). The `setupRoutes()` function wraps handlers with `s.csrfMW.Validate(finalHandler)`, which intercepts POST/PUT/PATCH/DELETE and validates the CSRF token.

In **site mode**, CSRF validation is **never applied**. The protected path check in `site.go` enforces authentication (redirects unauthenticated users), but no CSRF middleware is in the chain. A malicious site can submit a POST to a protected path, and as long as the user has an active session (cookies sent automatically), the request reaches the handler unchecked.

Handlers can manually validate via `basil.csrf.token`, but there's no server-side enforcement — it relies entirely on developer diligence.

### Do We Have Something Similar?

Yes — routes mode has exactly the right behavior. The gap is that site mode doesn't apply it.

### What's Blocking

Nothing. The fix is surgical (~5 lines in `site.go`): check if the path is protected and apply `csrfMW.Validate` around the handler in `serveWithHandler()`. This mirrors what routes mode already does.

### Cost

**~1–2 hours** including tests. This is the highest-priority item in this report due to the security implications.

---

## #14 — Auth Integration in Site Mode

| Field | Value |
|-------|-------|
| **Source** | FEAT-040 (File-Based Routing) |
| **Deferred from** | FEAT-040 — "Needs design" |
| **Status** | ⚠️ **Security gap** — mitigable but error-prone |

### Details

In **routes mode**, each route declares its own auth requirement: `auth: required` (401 if no user), `auth: optional` (user populated if present), or `auth: none` (explicitly unprotected). The middleware chain in `setupRoutes` applies the correct wrapper per route, plus CSRF validation for authenticated routes.

In **site mode**, auth handling has two disconnected layers:

1. **`protected_paths` check** — runs early in `ServeHTTP` before handler lookup. Checks URL prefix against `auth.protected_paths` config entries and blocks unauthenticated users. Supports role requirements.

2. **Handler auth middleware** — always hardcoded to `"optional"` in `serveWithHandler()`. Every site-mode handler executes regardless of auth state; `basil.auth.user` is `null` if not logged in.

There is **no way to declare `auth: required` per handler** in site mode. Protection depends entirely on the developer correctly configuring every sensitive path prefix in the global `protected_paths` list.

### The Risk

The gap is **omission by default**. A developer adds `/admin/users/index.pars` and forgets to add `/admin/users` to `protected_paths` in `basil.yaml`. In routes mode, you'd add `auth: required` right on the route definition — it's obvious and co-located. In site mode, the auth config is in a completely different section of the YAML, disconnected from the handler.

The current state is **safe if `protected_paths` is correctly configured**, but lacks defense-in-depth at the handler level.

### Do We Have Something Similar?

`protected_paths` provides a coarse safety net — it blocks unauthenticated access to URL prefixes. But it doesn't distinguish `required` from `optional`, can't handle mixed auth within a subtree (e.g., `/dashboard/public-report/` public but `/dashboard/` protected), and the config lives far from the handlers.

### Design Options

| Option | Approach | Pro | Con |
|--------|----------|-----|-----|
| **A: `site.paths` config** | YAML section with per-path auth, cache, etc. | Mirrors routes mode, centralized, auditable | Some duplication with `protected_paths` |
| **B: Comment directive** | `// @auth required` in `.pars` files | Co-located with handler | New parsing concept, fragile |
| **C: Handler-level check** | Check `basil.auth.user` in Parsley code | Works today, no changes needed | Boilerplate, easy to forget, handler still executes |

**Recommendation:** Option A (`site.paths` config) is strongest. It mirrors routes mode, keeps security config centralized and auditable, and could serve as the unified mechanism for **#13 (per-route caching)**, **#15 (CSRF)**, and this item — all three need "per-path config for site mode."

A combined `site.paths` design:

```/dev/null/example.yaml#L1-L6
site:
  path: ./site
  paths:
    /dashboard: { auth: required, cache: 5m }
    /admin: { auth: required, roles: [admin] }
    /public/*: { auth: none, cache: 1h }
```

### What's Blocking

Design decision on approach. If Option A is chosen, it could be designed to also cover #13 and #15 — combined effort for all three would be ~5–6 days instead of doing them separately.

### Cost

**~3–4 days** standalone. If combined with #13 and #15 under a unified `site.paths` design, **~5–6 days** total for all three.

---

## #13 — Per-Route Caching in Site Mode

| Field | Value |
|-------|-------|
| **Source** | FEAT-040 (File-Based Routing) |
| **Deferred from** | FEAT-040 — "Needs design" |
| **Status** | Open — needs design decision |

### Details

**Routes mode** has a `Cache` duration field per route in `basil.yaml`. Each `parsleyHandler` stores the route's cache TTL and checks it at request time, serving from cache on hit and capturing responses on miss.

**Site mode** has a single global `Site.Cache` TTL applied to every `index.pars` in the directory tree. There's no way to say "cache the homepage for 60s but the dashboard for 0s."

### Do We Have Something Similar?

The global `site.cache` provides a single TTL for all site routes. The per-route mechanism exists in routes mode but isn't available to site mode.

### Design Options

| Option | Approach | Pro | Con |
|--------|----------|-----|-----|
| **A: Comment directive** | `// basil:cache 300` in `.pars` files | Co-located with handler | Magic comments, must parse before execution |
| **B: YAML rules** | `site.cache_rules: [{path: /dashboard, cache: 0}]` | All config in one place | Duplicates filesystem structure, can drift |
| **C: Runtime config** | `basil.http.response.cache = 300` in handler | Maximum flexibility, consistent with `status`/`headers` pattern | First request always executes (but this is how cache miss works anyway) |

**Recommendation:** Option C (runtime) is the strongest choice. It's consistent with how handlers already set `status`, `headers`, and `cookies`. The handler sets a TTL during execution, and `writeResponseWithCache` uses it instead of the route config's value. Falls back naturally to `site.cache` global default when not set.

### What's Blocking

Design decision on which approach to use. No technical blockers.

### Cost

- **Option C (runtime):** ~2–4 hours — add `cache` to response meta extraction, use it in `writeResponseWithCache`, add Parsley namespace entry
- **Option B (YAML rules):** ~4–6 hours — add config parsing, path matching logic, integration with cache check
- Both could complement each other (B for static rules, C for dynamic overrides)

---

## #16b — Function Rest Parameters

| Field | Value |
|-------|-------|
| **Source** | API Design |
| **Deferred from** | N/A — standalone language enhancement |
| **Status** | Open — needs design |

### Details

Parsley supports `...rest` in **array destructuring** (`let [a, ...rest] = arr`) and **dictionary destructuring** (`let {a, ...rest} = dict`), including within function parameters (`fn([first, ...rest]) { rest }`). What's missing is rest at the **function parameter level**: `fn(a, b, ...rest)` where `rest` collects extra positional arguments into an array.

Currently, `FunctionParameter` in the AST only supports three shapes: `Ident`, `ArrayPattern`, and `DictPattern`. The `DOTDOTDOT` token is already lexed (used for destructuring rest). Extra arguments passed to a function are silently dropped.

### Do We Have Something Similar?

The destructuring rest is the closest analog and provides exact patterns to follow in both parsing and evaluation.

### What's Blocking

Nothing technical. The lexer already handles `...`, the destructuring rest provides implementation patterns, and the changes are well-scoped.

### Changes Required

| Area | Change | Lines |
|------|--------|-------|
| **AST** | Add `Rest bool` field to `FunctionParameter`, update `String()` | ~10 |
| **Parser** | Handle `DOTDOTDOT` in `parseFunctionParameter()`, validate rest is last param | ~25 |
| **Evaluator** | Collect remaining args into `*Array` in `extendFunctionEnv()` | ~15 |
| **Arity** | `HasRestParam()` method, audit `ParamCount()` callers | ~10 |
| **Tests** | Happy path, edge cases, error cases (rest not last, multiple rest) | ~80–100 |

### Cost

**~2–3 hours.** Small, self-contained feature. Estimated ~150 lines total. The main risk is ensuring `ParamCount()` consumers handle rest parameters correctly — requires auditing all call sites.

---

## #18 — CSV Upload Merge Mode for /__/db

| Field | Value |
|-------|-------|
| **Source** | FEAT-021 (SQLite Dev Tools) |
| **Deferred from** | FEAT-021 — "Not MVP" |
| **Status** | Open — well-scoped enhancement |

### Details

The current CSV upload (`POST /__/db/upload/{table}`) uses **Replace** mode: it drops the table and recreates it from the CSV. This is destructive — notably, CSV export excludes BLOB columns, so the round-trip (download → edit → re-upload) **permanently loses all BLOB data**.

The proposed **Merge** mode would use SQLite's `INSERT ... ON CONFLICT(pk) DO UPDATE SET` to update existing rows by primary key and insert new ones. Columns not present in the CSV (including BLOB columns) would be preserved.

### Do We Have Something Similar?

The codebase already uses `ON CONFLICT ... DO UPDATE SET` in `server/search/metadata.go` and `INSERT OR REPLACE` in `server/search/document.go`, so the UPSERT pattern is established. The existing `replaceTableFromCSV()` already reads column info including primary key via `PRAGMA table_info`.

### What's Blocking

Nothing. The implementation path is clear.

### Changes Required

| Component | Work |
|-----------|------|
| **`devtools_db.go`** | New `mergeTableFromCSV()` function: read PK columns, validate CSV has PK, generate UPSERT, wrap in transaction |
| **Route + handler** | New `POST /__/db/merge/{table}` endpoint (cleanest option — matches existing one-endpoint-per-action pattern) |
| **UI** | Second button "⬆️ Merge CSV by Primary Key" next to existing Replace button. Conditionally shown when table has a PK. |
| **Edge cases** | Error when table has no PK, handle compound PKs, error when CSV has unknown columns |
| **Tests** | Basic merge, BLOB preservation, no-PK error, compound PK, missing PK column in CSV |

### Cost

**~3–5 hours.** The hardest part is getting the UPSERT generation right for compound primary keys. Everything else is straightforward.

---

## #107 — Overflow Detection for Unit Arithmetic

| Field | Value |
|-------|-------|
| **Source** | FEAT-118 (Measurement Units) |
| **Deferred from** | FEAT-118 Phase 3 — "Partially addressed" |
| **Status** | Partially implemented — gaps in error reporting and test coverage |

### Details

FEAT-118 introduced measurement units with integer-based storage. Phase 3 added a **decimal Scale** system (`true value = Amount × 10^Scale`) so that area values exceeding int64 range can be represented by shifting to coarser scale instead of wrapping.

The overflow detection infrastructure is substantial (~340 lines in `unit_scale.go`): `scaleAdd`, `scaleSub`, `scaleMulScalar`, `scaleDivScalar`, `scaleMulDiv`, `scaleAlign`, `scaleNormalize` all detect overflow and adjust scale. All arithmetic code paths in `eval_unit_infix.go` use these scale-aware helpers.

### Do We Have Something Similar?

Yes — the scale system itself _is_ the overflow handling. The gap is in the edges.

### Remaining Gaps

| Gap | Priority | Description |
|-----|----------|-------------|
| **UNIT-0009 error never emitted** | Medium | Error code exists in catalogue with template "Unit value overflow" but is dead code. When values are truly unrepresentable, `scaleMulScalarOverflow` and `scaleMulDiv` silently return `(0, 0)` instead of erroring. |
| **No `unit_scale_test.go`** | Medium | Scale helpers are only tested indirectly through integration tests. No direct unit tests for edge cases, overflow boundaries, or round-trip preservation. |
| **`math.MinInt64` edge case** | Low | `scaleSubOverflow` negates `bAmount` — if `bAmount == math.MinInt64`, this silently wraps (stays `MinInt64`). |
| **`scaleDivScalar` fallback** | Low | Division overflow path truncates via `float64→int64` conversion, which wraps silently. |
| **Non-area overflow tests** | Low | Length, mass, volume, data families always operate at Scale=0 — no integration tests verify behavior at extreme values. |

### What's Blocking

Nothing. The remaining work is testing and wiring up the existing error code.

### Cost

| Task | Effort |
|------|--------|
| Wire up `UNIT-0009` error at call sites | 2–3 hours |
| Create `unit_scale_test.go` with comprehensive tests | 3–4 hours |
| Fix `MinInt64` edge case | 30 min |
| Fix `scaleDivScalar` fallback | 30 min |
| Non-area overflow integration tests | 1–2 hours |
| **Total** | **~7–10 hours** |

---

## #10 — Session Auth Integration

| Field | Value |
|-------|-------|
| **Source** | FEAT-049 (Sessions and Flash Messages) |
| **Deferred from** | FEAT-049 Phase 3 |
| **Status** | Open — medium complexity, architectural considerations |

### Details

FEAT-049 has three phases:
- **Phase 1 (✅ Done):** Cookie sessions — middleware, encrypt/decrypt, `basil.session` dict, flash support
- **Phase 2 (❌ #9):** SQLite session store — server-side sessions for larger data, cleanup goroutine
- **Phase 3 (❌ #10):** Session auth integration — regenerate session on login/logout, user_id tracking

The goal is **session fixation prevention**: when a user logs in, the Parsley cookie session (`basil.session`) should be regenerated so an attacker who knew the pre-login session ID can't hijack the authenticated session.

### The Architectural Challenge

There are **two completely independent session systems**:

1. **Parsley cookie sessions** (`server/session.go`, `stdlib_session.go`): AES-256-GCM encrypted cookies, used by Parsley scripts via `basil.session`. Cookie: `_basil_session`.

2. **Auth DB sessions** (`server/auth/session.go`, `auth/database.go`): Server-side SQLite storage, used by the WebAuthn auth system. Cookie: `__basil_session`. Random 32-byte token, full CRUD.

These systems don't know about each other. Login/logout happens at the HTTP handler layer via WebAuthn JavaScript flows — there is no `basil.auth.login()` Parsley function. The auth handlers never touch the Parsley cookie session.

A `session.Regenerate()` method exists but is minimal for cookie sessions — it just marks the session dirty and resets expiration. There's no actual session ID to regenerate (the "ID" is the encrypted blob itself).

### Do We Have Something Similar?

The auth system's DB sessions already get a fresh random token on each login — session fixation is handled for the _auth_ session. The gap is that the _Parsley_ session isn't regenerated alongside it.

### What's Blocking

**Architectural plumbing** — the auth handlers need access to the Parsley session middleware to regenerate it on login/logout. Currently these systems operate in separate middleware layers with no bridge.

**Dependency on #9 (SQLite sessions):** Not strictly required — cookie session regeneration works without server-side storage. However, with cookie-only sessions, the old encrypted cookie remains valid until it expires (no server-side invalidation). SQLite sessions (#9) would provide real session fixation protection where the old token is immediately invalidated.

### Cost

| Task | Effort |
|------|--------|
| Bridge auth handlers to Parsley session middleware | Medium (architectural change) |
| Call `Regenerate()` on login/register/recover/logout | Small (few lines per handler) |
| Add `user_id` tracking to Parsley session | Small-Medium |
| Integration tests | Medium |
| **Total** | **~1–2 days** |

The biggest risk is creating tight coupling between the two session systems. Needs careful design.

---

## #19 — HTTP-Only Production Mode (Behind Proxy)

| Field | Value |
|-------|-------|
| **Source** | Discussion |
| **Deferred from** | N/A — originated from deployment discussions |
| **Status** | Open — medium scope, well-understood requirements |

### Details

Currently, the server has a **binary mode split**: dev mode = HTTP, production = HTTPS (required). There is no way to run production features (caching, generic errors, security headers) without TLS. This blocks deployments behind reverse proxies (nginx, Cloudflare, AWS ALB) where the proxy terminates TLS.

Dev mode (`--dev`) is not a workaround because it also disables caching, enables debug tools, disables security headers, shows detailed errors, disables browser caching, and injects live-reload JS. It's fundamentally a different runtime profile.

### Do We Have Something Similar?

**Partially.** A `ProxyConfig` struct already exists with `Trusted bool` and `TrustedIPs []string`. The `proxyAware` middleware extracts real client IP from `X-Forwarded-For`/`X-Real-IP` and validates trusted proxy IPs. However, `proxy.trusted: true` currently only affects header trust — it does **not** allow HTTP-only production mode.

### What's Blocking

Design decision and the number of subsystems that need proxy awareness. The core change is small (skip TLS validation when proxy is trusted), but the behavior must be threaded through:

| Subsystem | Change Needed |
|-----------|---------------|
| **Config validation** (`load.go`) | Allow `proxy.trusted: true` as alternative to HTTPS config |
| **Server startup** (`server.go`) | Third path: production + proxy = HTTP on configurable port |
| **Listen address** | Default to `127.0.0.1:8080` in proxy mode (safe by default) |
| **Cookie Secure flag** | Default `Secure: true` in proxy mode (external connection IS HTTPS) |
| **WebAuthn origin** | Construct `https://` origin from external URL, not internal HTTP |
| **Email base URL** | Same — use external URL for verification links |
| **HSTS** | Works as-is (proxy passes it through) |
| **Warnings** | Warn if `proxy.trusted` without `trusted_ips`, warn if binding to `0.0.0.0` |

### Security Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Untrusted proxy spoofing | HIGH | `trusted_ips` validation (exists but optional — consider requiring it) |
| Cookie Secure flag mismatch | MEDIUM | Default correctly based on proxy mode |
| Binding to 0.0.0.0 without firewall | MEDIUM | Warn when host is `0.0.0.0` in proxy mode |
| WebAuthn origin mismatch | HIGH | Require `proxy.external_url` config for correct URL construction |

### Recommended Design

1. `proxy.trusted: true` enables HTTP production mode
2. Add `proxy.external_url: "https://example.com"` for URL construction (WebAuthn, email links, redirects)
3. Require `trusted_ips` or warn loudly without it
4. Default listen to `127.0.0.1:8080` in proxy mode
5. Default cookie `Secure: true` in proxy mode

### Cost

**~10–13 hours** (~1.5–2 days), broken down:

| Component | Effort |
|-----------|--------|
| Config validation | 1 hour |
| Server startup path | 1–2 hours |
| Cookie/session secure flag | 1 hour |
| Auth origin + email base URL | 1–2 hours |
| Misconfiguration warnings | 1 hour |
| Tests | 3–4 hours |
| Documentation | 1–2 hours |

---

## #17 — Standardize Locale Support Across Stdlib

| Field | Value |
|-------|-------|
| **Source** | FEAT-032 / FEAT-033 |
| **Deferred from** | FEAT-032 (Validation Module) — ad-hoc locale support was sufficient for MVP |
| **Status** | Open — large, cross-cutting effort |

### Details

Locale support is currently ad-hoc across several subsystems, with inconsistent coverage:

### Current State by Subsystem

| Subsystem | Locales | Implementation |
|-----------|---------|----------------|
| **Date/time formatting** | ~25 via `monday` library mapping, but only 7–10 have explicit format strings | `eval_locale.go` `getMondayLocale()` + `getDateFormatForStyle()` |
| **Date/time parsing** | **Only 5** (en-US, en-GB, fr-FR, de-DE, es-ES) | `eval_datetime.go` `localeConfigs` — everything else silently falls back to en-US |
| **Number/currency formatting** | Good — uses `golang.org/x/text` CLDR data, handles any BCP 47 tag | `eval_locale.go` `formatNumberWithLocale()` |
| **Money formatting** | **7 locales** hand-rolled in switch statements | `methods_money.go` `getLocaleSeparators()` — duplicates what `x/text` already does |
| **Currency symbols** | 8 currencies, not locale-sensitive | `methods_money.go` `getCurrencySymbol()` |
| **Currency names** | 4 currencies × 3–5 locales each | `methods_money.go` `getCurrencyName()` — very limited |
| **Postal code validation** | 3 (US, GB, CA) | `stdlib_valid.go` |
| **Date validation** | 3 (ISO, US, GB) | `stdlib_valid.go` |

### The Biggest Inconsistency

There are **two parallel number formatting paths** that don't agree:

1. `formatNumberWithLocale()` — uses `x/text` CLDR data, handles any locale correctly
2. `getLocaleSeparators()` — hand-rolled switch for 7 locales

This means `(42000.5).format({locale: "ja-JP"})` uses CLDR (correct) but `money(4200050, "JPY").format({style: "long"})` uses the hand-rolled path (incorrect for many locales).

### Do We Have Something Similar?

Yes — the `x/text` number/currency formatting is already correct and comprehensive. The problem is that other subsystems (money, dates, validation) don't use it consistently.

### What's Blocking

1. **Design decision** on canonical locale list (proposed: en-US, en-GB, de-DE, fr-FR, es-ES, ja-JP, zh-CN, pt-BR, ru-RU, ar-SA, it-IT, ko-KR, nl-NL, fr-CA — 14 total)
2. **Partial support policy** — what happens when a user passes `"ar-SA"` to `postalCode()`? Error? Fallback? Document per-feature coverage?
3. **CJK complexity** — Japanese/Chinese/Korean date parsing needs character-based month names and different format patterns

### Recommended Approach (Phased)

| Phase | Work | Effort | Impact |
|-------|------|--------|--------|
| **1: Unify money → x/text** | Replace `getLocaleSeparators()`, `symbolAfter()`, `getCurrencyName()` with `x/text/currency` | 2–3 days | Highest ROI — instant CLDR support for all locales, removes most error-prone code |
| **2: Expand date parsing** | Add month name tables for it, pt, nl, ru, ja, zh, ko to `localeConfigs` | 2–3 days | Fixes the worst inconsistency (25 locales format, 5 parse) |
| **3: Expand date format styles** | Add explicit `getDateFormatForStyle()` for canonical locales | 1–2 days | Improves formatting accuracy |
| **4: Expand validation** | Add postal code regexes for DE, FR, JP, IT, ES, etc. | 1–2 days | Mechanical — each is a regex |
| **5: Testing** | Comprehensive tests across locale × feature matrix | 2–3 days | 14 locales × 5 features = 70+ scenarios |

### Cost

**~8–12 developer-days total.** This is the largest item in the report by a significant margin. Phase 1 (unify money) could be done independently as a standalone PR with the highest value-to-effort ratio.

---

## Recommendations

### Immediate Actions

1. **Move #97 and #26 to Completed** — they're already implemented
2. **Fix #15 (CSRF in site mode)** — security gap, trivial fix, ~1–2 hours

### High-Value, Low-Effort

3. **#16b (Function rest parameters)** — clean, well-scoped enhancement, ~2–3 hours
4. **#18 (CSV merge mode)** — practical dev-tool improvement, ~3–5 hours

### Medium Priority

5. **#107 (Unit overflow)** — complete partially-done work, ~7–10 hours
6. **#13 (Site mode caching)** — pick Option C (runtime), ~2–4 hours
7. **#10 (Session auth integration)** — valuable security improvement, ~1–2 days

### Security Hardening

5. **#14 (Auth in site mode)** — design `site.paths` config to cover #14, #13, and #15 together (~5–6 days combined)
6. **#43 (API key expiry)** — ready to implement, schema and validation already done (~0.5–1 day)
7. **#42 (API key scopes)** — defer until there's a concrete use case beyond Git push/pull
8. **#44 (Argon2 hashing)** — defer; bcrypt is fine, prefix-based lookup would be a better first optimization

### Larger Efforts (Plan Separately)

9. **#19 (Proxy mode)** — important for real-world deployment, ~10–13 hours
10. **#17 (Locale standardization)** — large cross-cutting effort, recommend starting with Phase 1 (money → x/text) as a standalone PR

---

## Addendum: API Key Security (#42, #43, #44)

These three items share the same subsystem and are best understood together.

### Current API Key Architecture

The API key system (FEAT-036) uses bcrypt-hashed keys stored in SQLite. Keys are created via `basil apikey create --user <id> --name <label>` and validated on Git-over-HTTPS requests (Basic Auth with the key as the password). The `ValidateAPIKey()` method fetches ALL keys and iterates with bcrypt comparison — O(n) per auth request.

The `APIKey` struct has fields: `id`, `user_id`, `name`, `key_hash` (bcrypt), `key_prefix` (display), `created_at`, `last_used_at`, `expires_at`. Notably: no scope/permission field.

---

## #42 — API Key Scopes

| Field | Value |
|-------|-------|
| **Source** | FEAT-004 (Authentication) |
| **Deferred from** | FEAT-004 — "Not MVP" |
| **Status** | Open — needs design decisions |

### Details

API keys currently inherit the full permissions of their owning user. A key created for an admin user has admin-level access to everything. If a key is leaked, the blast radius is the entire API surface. There's no way to create a read-only key or limit a key to specific routes.

The FEAT-036 spec explicitly notes: *"API Key Scopes (v1: None) — In v1, API keys inherit the user's role — no per-key scopes."*

### Do We Have Something Similar?

The role system (`admin`/`editor`) provides coarse access control at the user level, but keys inherit that fully. There's no per-key restriction mechanism.

### What Would Be Needed

| Component | Change |
|-----------|--------|
| **Schema** | Add `scopes TEXT` column to `api_keys` (JSON or comma-separated) |
| **Struct** | Add `Scopes []string` to `APIKey` |
| **CLI** | `basil apikey create --scopes read-only,git-push` |
| **Validation** | `ValidateAPIKey` must return scope info; callers check scopes |
| **Enforcement** | `git.go` checks `git-read`/`git-push` scope; future HTTP auth checks scopes |
| **Design** | Define scope vocabulary (read-only, git-push, admin, route-specific?) |

### What's Blocking

Design decisions: what scopes to define, whitelist vs. blacklist model, interaction with user roles. Currently only Git uses API keys — scope enforcement has limited surface area.

### Cost

**~2–3 days.** Schema/CRUD is straightforward (~1 day), but scope definitions and enforcement at each call site add complexity. Recommend deferring until there's a concrete use case beyond Git.

---

## #43 — API Key Expiry Flag

| Field | Value |
|-------|-------|
| **Source** | FEAT-036 (API Keys) |
| **Deferred from** | FEAT-036 — "Not MVP" |
| **Status** | Open — **ready to implement** (most infrastructure already exists) |

### Details

The API key system was designed with expiry support but the last mile was never connected:

| Component | Status |
|-----------|--------|
| Schema `expires_at TIMESTAMP` column | ✅ Exists |
| `APIKey.ExpiresAt *time.Time` struct field | ✅ Exists |
| `ValidateAPIKey()` skips expired keys | ✅ Works |
| `scanAPIKeys` reads/populates ExpiresAt | ✅ Works |
| `CreateAPIKey()` accepts expiry parameter | ❌ **Missing** — INSERT omits `expires_at` |
| CLI `--expires` flag | ❌ **Missing** |
| Ability to update expiry on existing keys | ❌ **Missing** |

Keys are effectively immortal unless manually deleted. The validation logic already rejects expired keys — the gap is purely in key creation and management.

### Do We Have Something Similar?

The `expires_at` column and validation check already exist. This is the most "almost done" item in the entire backlog.

### What's Blocking

Nothing. The changes are mechanical.

### Changes Required

1. Add `expiresAt *time.Time` parameter to `CreateAPIKey()` and include `expires_at` in the INSERT
2. Add `--expires` flag to CLI (accept `2025-06-01` or duration like `90d`)
3. Optionally: `basil apikey update --expires` for existing keys
4. Update `apikey list` output to show expiry/expired status

### Cost

**~0.5–1 day.** The hard parts (schema, validation) are already done. This is the most ready-to-implement security item in the backlog.

---

## #44 — Argon2 for API Key Hashing

| Field | Value |
|-------|-------|
| **Source** | FEAT-036 (API Keys) |
| **Deferred from** | FEAT-036 — "Not MVP" |
| **Status** | Open — low priority, bcrypt is still secure |

### Details

The system uses bcrypt at `DefaultCost` (10) for API key hashing, used in 4 places: API key hashing/validation, recovery code hashing/validation, and email verification token hashing/validation. Argon2id is more GPU-resistant and is the current OWASP recommendation for new systems.

### Do We Have Something Similar?

bcrypt is the only hashing algorithm in use. `golang.org/x/crypto` (already a dependency) includes the `argon2` subpackage, so no new module would be needed.

### What Migration Would Involve

| Step | Description |
|------|-------------|
| **New hashing functions** | Argon2id with recommended params (64MB memory, 3 iterations, 4 parallelism), PHC-format encoded |
| **Dual-read migration** | `ValidateAPIKey` detects hash format: `$2a$`/`$2b$` → bcrypt, `$argon2id$` → Argon2. Existing keys keep working without re-hashing |
| **New keys use Argon2** | `GenerateAPIKey()` switches to Argon2 for new keys |
| **4 files affected** | `apikeys.go`, `email_verification.go`, `recovery.go`, and their tests |

### The Real Performance Issue

The bigger problem isn't the hash algorithm — it's the **O(n) scan**. `ValidateAPIKey()` fetches ALL keys and iterates with bcrypt comparison on every auth request. A prefix-based lookup (filter to one candidate by key prefix before the expensive comparison) would be a more impactful performance fix than switching algorithms.

### What's Blocking

Nothing technically, but bcrypt at cost 10 is still considered secure. The O(n) validation scan is the more pressing concern.

### Cost

**~2 days.** The algorithm swap is mechanical, but dual-read migration, parameter encoding, and ensuring existing bcrypt keys still validate requires care. Consider bundling with prefix-based lookup optimization.