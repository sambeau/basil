---
updated: 2025-12-08
---

# Backlog

Deferred items from implementation, to be picked up in future work.

## High Priority
| Item | Source | Reason Deferred | Notes |
|------|--------|-----------------|-------|
| Remove `basil` global in favor of `std/basil` import | FEAT-019 | Backward compat | Having both `basil` global and `std/basil` import creates two objects with same content. Hard to test properly, confusing API. Deprecate global, require `let {basil} = import("std/basil")`. Breaking change - needs migration path. |
| Complete structured error migration | FEAT-023 | Phase 6+ | Migrate remaining files: `builtins.go`, other `stdlib_*.go` modules (json, http, sftp, etc.). Core evaluator files done. |

## Medium Priority
| Item | Source | Reason Deferred | Notes |
|------|--------|-----------------|-------|
| Per-route caching in site mode | FEAT-040 | Needs design | Site mode has no way to configure cache TTL per index.pars. Routes mode has `cache:` per route. Options: comment directive in index.pars, basil.yaml section per path pattern, or runtime config via `basil.http.response.cache`. |
| Auth integration in site mode | FEAT-040 | Needs design | Site mode has no way to specify auth requirements per handler. Routes mode has `auth:` per route. Options: comment directive in index.pars, basil.yaml section per path pattern, or check `basil.auth.user` in handler and redirect/error manually. |
| Rest operator consistency | API Design | Needs design | **Current state:** Dict rest destructuring works (`let {a, ...rest} = obj`). Array/dict merge handled by `++` operator (`a ++ {z: 3}`, `arr1 ++ arr2`). **Missing:** (1) Array rest destructuring (`let [first, ...rest] = arr`), (2) Function rest parameters (`fn(a, ...rest)`). Note: Spread in literals is NOT needed—use `++` instead. **Cheatsheet showed `fn({title}, ...children)` which doesn't work—fixed.** |
| Standardize locale support across stdlib | FEAT-032/033 | Needs design | Define a standard set of locales (e.g., top 10-15 by usage/currency) and ensure consistent support across: dates (parsing/formatting), times, numbers (decimal/thousands separators), currency formatting, postal codes. Currently ad-hoc (US, GB, ISO). Need to decide: which locales, what coverage each gets, how to handle partial support. Consider: en-US, en-GB, de-DE, fr-FR, es-ES, ja-JP, zh-CN, pt-BR, ru-RU, ar-SA (roughly top 10 traded currencies). |
| CSV upload merge mode for /__/db | FEAT-021 | Not MVP | Current "Replace" overwrites entire table. Add "Merge" option that updates existing rows by primary key and inserts new ones. Use case: download CSV, edit non-BLOB columns, re-upload without losing BLOB data. UI: dropdown or separate button next to "Replace". |
| HTTP-only production mode (behind proxy) | Discussion | Needs design | Allow running without TLS when behind a reverse proxy (nginx, Cloudflare, etc.). Use case: proxy terminates TLS, Basil runs HTTP on localhost/internal IP but with production features (caching, generic errors). Consider: `https.mode: proxy` or `server.tls: false` with warning. Security: must validate proxy is trusted. Options: `--proxy` CLI flag, require `proxy.trusted: true`. |
| Separate dev errors from dev mode | Discussion | Needs design | Allow styled error pages independently of full dev mode. Use case: testing behind proxy with caching enabled but still seeing detailed errors. Options: `--dev-errors` flag, `server.dev_errors: true` config, or make dev mode more granular (`dev.errors: true`, `dev.caching: false`, etc.). |
| Form validation/sanitization | FEAT-002 | Needs design | Options: config-based schemas, Parsley-side validation, or sanitization-only. See spec Phase 2 checklist. |
| OAuth2/OIDC providers | FEAT-004 | Not MVP | Google, GitHub, etc. identity providers. Consider after passkey auth is stable. |
| SMS recovery (Twilio) | FEAT-004 | Not MVP | Recovery via SMS code. Simpler than email (no deliverability issues, just JSON API). Would need Twilio account config. Consider as primary "second factor" option. |
| Email recovery | FEAT-004 | Probably never | Magic link via email. Pain points: deliverability (SPF/DKIM/reputation), styling (1999 CSS), complexity. SMS is easier. |
| Multiple passkeys per user | FEAT-004 | Not MVP | Allow registering phone + laptop + YubiKey. Adds device management UI. |
| Roles/permissions | FEAT-004 | Not MVP | `request.user.role` and role-based route protection |
| Table.groupBy(column) | FEAT-018 | Not MVP | Complex aggregation, needs design for return type |
| Table.join(table, column) | FEAT-018 | Not MVP | SQL joins, needs careful design |
| Table column transforms | FEAT-018 | Not MVP | `transform(col, fn)`, `addColumn(name, fn)` |
| Table.distinct() | FEAT-018 | Not MVP | Deduplication |
| Table.first() / Table.last() | FEAT-018 | Not MVP | Single row access |
| Table.toJSON() | FEAT-018 | Not MVP | JSON output |
| Table.fromCSV(string) | FEAT-018 | Not MVP | CSV parsing into Table |
| Error code documentation/help system | FEAT-023 | Phase 6+ | CLI command or web endpoint to look up error codes with examples/solutions. e.g., `pars error TYPE-0001` or `/__/errors/TYPE-0001`. |

## Low Priority / Nice to Have
| Item | Source | Reason Deferred | Notes |
|------|--------|-----------------|-------|
| Fragment cache DevTools integration | FEAT-037 | Not MVP | Add `/__/cache` page showing cache stats (entries, hits, misses, hit rate, size) with clear button. `FragmentCacheStats` and `Stats()` method already exist in `fragment_cache.go`. |
| std/math: Advanced statistics | FEAT-031 | Niche | percentile, quartile, correlation, z-score - add based on demand from data-focused users |
| std/math: Hyperbolic functions | FEAT-031 | Niche | sinh, cosh, tanh - rare use case for most users |
| std/math: Special functions | FEAT-031 | Niche | gamma, factorial - mathematical niche |
| SPREAD-0001 error missing line numbers | Error improvements | Needs refactoring | Error is inside `parseTagProps` string parsing function which doesn't have access to a token. Would need to pass token through or refactor to track position during parsing. |
| Admin interface | FEAT-002 | Premature | Needs auth first, unclear requirements. Built with Parsley when needed. |
| Key scopes | FEAT-004 | Not MVP | Limit what API keys can access (read-only, specific routes, etc.) |
| API key expiry flag | FEAT-036 | Not MVP | Schema has `expires_at` but no CLI flag. Add `--expires` to `basil apikey create`. |
| Argon2 for API key hashing | FEAT-036 | Not MVP | Currently bcrypt. Argon2 is more GPU-resistant. Revisit if key validation perf becomes an issue. |
| Custom error pages | Dev mode 404 | Polish | Allow users to define custom 404/500 pages for production via config (e.g., `error_pages: { 404: ./errors/404.pars }`). Dev mode already has styled pages. |
| Better import error messages | BUG-010 | Parser work | When import fails, report which path was tried and from which file. e.g., "Module not found: ./app/pages/components/page.pars (imported from ./app/pages/home.pars line 1)" |
| Dev logs: JS-based clear without page reload | FEAT-019 | Polish | Currently clear redirects, which re-runs handler dev.log() calls. Use fetch() + DOM update instead. |
| Dev logs: pause/resume toggle | FEAT-019 | Polish | Temporarily stop collecting logs without clearing. Useful when debugging specific requests. |
| Dev logs: `.json` modifier for formatted JSON | FEAT-019 | Not MVP | `dev.log(data, {json: true})` renders value as formatted/syntax-highlighted JSON. |
| Error code validation tests | FEAT-023 | Phase 6+ | Test suite to ensure all error codes in errors.go are actually used, and all newStructuredError calls use valid codes. Prevents drift between defined and used codes. |
| Function methods | API Design | Future exploration | Allow methods on functions for composition, introspection, memoization. Examples: `f.arity`, `f.params`, `f.then(g)`, `f.memoize()`, `f.partial(arg)`. Would enable fluent auth syntax like `fn(req){...}.public()`. Implementation: functions as "callable dictionaries" with `__call__` property. Low priority - wrapper functions work fine. |

## Completed (Archive)
<!-- Move items here when done, with completion date -->
| Item | Source | Completed | Notes |
|------|--------|-----------|-------|
| — | — | — | — |
