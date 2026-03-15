# Backlog Investigation Report

**Date:** 2026-03-16
**Scope:** Items #107, #16b, #18, #10, #13, #15, #19, #26, #97, #17

---

## Executive Summary

Ten backlog items were investigated in depth. Key findings:

| # | Item | Verdict | Effort |
|---|------|---------|--------|
| **#97** | Named capture groups → dictionaries | **Already done** — move to Completed | 0 (bookkeeping) |
| **#26** | Roles/permissions | **Already done** — move to Completed | 0 (bookkeeping) |
| **#15** | CSRF middleware for site mode | **Security gap** — fix ASAP | ~1–2 hours |
| **#16b** | Function rest parameters | Small, well-scoped | ~2–3 hours |
| **#18** | CSV upload merge mode | Small, well-scoped | ~3–5 hours |
| **#107** | Unit arithmetic overflow detection | Partially done, gaps remain | ~7–10 hours |
| **#10** | Session auth integration | Medium, architectural plumbing needed | ~1–2 days |
| **#13** | Per-route caching in site mode | Medium, needs design decision | ~2–6 hours |
| **#19** | HTTP-only production mode (behind proxy) | Medium, touches many subsystems | ~10–13 hours |
| **#17** | Standardize locale support | Large, multi-week effort | ~8–12 days |

Two items (#97 and #26) are already implemented and just need to be moved to Completed. One item (#15) is a security gap with a trivial fix. The rest range from small enhancements to a multi-week standardization effort.

---

## Table of Contents

1. [#97 — Named Capture Groups (ALREADY DONE)](#97--named-capture-groups-should-return-dictionaries)
2. [#26 — Roles/Permissions (ALREADY DONE)](#26--rolespermissions)
3. [#15 — CSRF Middleware for Site Mode](#15--csrf-middleware-for-site-mode)
4. [#13 — Per-Route Caching in Site Mode](#13--per-route-caching-in-site-mode)
5. [#16b — Function Rest Parameters](#16b--function-rest-parameters)
6. [#18 — CSV Upload Merge Mode](#18--csv-upload-merge-mode-for-db)
7. [#107 — Unit Arithmetic Overflow Detection](#107--overflow-detection-for-unit-arithmetic)
8. [#10 — Session Auth Integration](#10--session-auth-integration)
9. [#19 — HTTP-Only Production Mode](#19--http-only-production-mode-behind-proxy)
10. [#17 — Standardize Locale Support](#17--standardize-locale-support-across-stdlib)

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

### Larger Efforts (Plan Separately)

8. **#19 (Proxy mode)** — important for real-world deployment, ~10–13 hours
9. **#17 (Locale standardization)** — large cross-cutting effort, recommend starting with Phase 1 (money → x/text) as a standalone PR