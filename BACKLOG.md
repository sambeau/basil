---
updated: 2025-01-03
---

# Backlog

Deferred items from implementation, to be picked up in future work.

## High Priority
| Item | Source | Reason Deferred | Notes |
|------|--------|-----------------|-------|
| Resolve paths relative to config file location | FEAT-002 | Phase 1 scope | Handler/static paths should be relative to config file, not CWD |
| Add @std/ prefix support to lexer | FEAT-018 | Workaround available | Currently requires `import("std/table")` string syntax. Should support `import(@std/table)` path literal. |

## Medium Priority
| Item | Source | Reason Deferred | Notes |
|------|--------|-----------------|-------|
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

## Low Priority / Nice to Have
| Item | Source | Reason Deferred | Notes |
|------|--------|-----------------|-------|
| Admin interface | FEAT-002 | Premature | Needs auth first, unclear requirements. Built with Parsley when needed. |
| Key scopes | FEAT-004 | Not MVP | Limit what API keys can access (read-only, specific routes, etc.) |
| Custom error pages | Dev mode 404 | Polish | Allow users to define custom 404/500 pages for production via config (e.g., `error_pages: { 404: ./errors/404.pars }`). Dev mode already has styled pages. |
| Better import error messages | BUG-010 | Parser work | When import fails, report which path was tried and from which file. e.g., "Module not found: ./app/pages/components/page.pars (imported from ./app/pages/home.pars line 1)" |

## Completed (Archive)
<!-- Move items here when done, with completion date -->
| Item | Source | Completed | Notes |
|------|--------|-----------|-------|
| — | — | — | — |
