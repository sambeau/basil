---
updated: 2025-12-02
---

# Backlog

Deferred items from implementation, to be picked up in future work.

## High Priority
| Item | Source | Reason Deferred | Notes |
|------|--------|-----------------|-------|
| Resolve paths relative to config file location | FEAT-002 | Phase 1 scope | Handler/static paths should be relative to config file, not CWD |

## Medium Priority
| Item | Source | Reason Deferred | Notes |
|------|--------|-----------------|-------|
| Form validation/sanitization | FEAT-002 | Needs design | Options: config-based schemas, Parsley-side validation, or sanitization-only. See spec Phase 2 checklist. |
| OAuth2/OIDC providers | FEAT-004 | Not MVP | Google, GitHub, etc. identity providers. Consider after passkey auth is stable. |
| SMS recovery (Twilio) | FEAT-004 | Not MVP | Recovery via SMS code. Simpler than email (no deliverability issues, just JSON API). Would need Twilio account config. Consider as primary "second factor" option. |
| Email recovery | FEAT-004 | Probably never | Magic link via email. Pain points: deliverability (SPF/DKIM/reputation), styling (1999 CSS), complexity. SMS is easier. |
| Multiple passkeys per user | FEAT-004 | Not MVP | Allow registering phone + laptop + YubiKey. Adds device management UI. |
| Roles/permissions | FEAT-004 | Not MVP | `request.user.role` and role-based route protection |

## Low Priority / Nice to Have
| Item | Source | Reason Deferred | Notes |
|------|--------|-----------------|-------|
| Admin interface | FEAT-002 | Premature | Needs auth first, unclear requirements. Built with Parsley when needed. |
| Key scopes | FEAT-004 | Not MVP | Limit what API keys can access (read-only, specific routes, etc.) |
| Custom error pages | Dev mode 404 | Polish | Allow users to define custom 404/500 pages via config (e.g., `error_pages: { 404: ./errors/404.pars }`) |

## Completed (Archive)
<!-- Move items here when done, with completion date -->
| Item | Source | Completed | Notes |
|------|--------|-----------|-------|
| — | — | — | — |
