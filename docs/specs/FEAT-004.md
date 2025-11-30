---
id: FEAT-004
title: "Authentication"
status: planning
priority: medium
created: 2025-11-30
author: "@sambeau"
---

# FEAT-004: Authentication

## Summary
Add authentication support to Basil, allowing routes to be protected and user identity to be passed to Parsley handlers.

## Status: Planning Discussion

Before implementation, we need to decide on the approach. This document captures the options and trade-offs.

---

## Key Questions

### 1. What's the minimum viable auth?

**Option A: Single admin password**
- Simplest possible: one password in config (hashed)
- Protects routes with basic auth or session cookie
- No user management, just "authenticated or not"
- Good for: personal projects, internal tools

**Option B: Multi-user with local accounts**
- Users stored in SQLite (already integrated)
- Registration, login, password reset
- User identity passed to handlers: `{id, email, role}`
- Good for: apps with multiple users

**Option C: External identity provider**
- OAuth2/OIDC with Google, GitHub, etc.
- Basil doesn't manage passwords
- Good for: public apps, enterprise

### 2. Session management

**Cookie-based sessions** (traditional)
- Server stores session data
- Works well with browser-based apps
- Needs CSRF protection

**JWT tokens** (stateless)
- No server-side session storage
- Good for APIs
- Harder to revoke

### 3. Future-proofing for Passkeys/WebAuthn

WebAuthn is the modern passwordless standard. Should we:
- Start with passwords, add WebAuthn later?
- Go WebAuthn-first? (steeper learning curve)
- Support both from the start?

---

## Proposed Approach

**Recommendation: Start simple, build incrementally**

### Phase 1: Basic Session Auth
- Session cookies with secure defaults
- Password hashing (bcrypt)
- Single-user mode (admin password in config)
- Route protection via config:
  ```yaml
  routes:
    - path: /admin/
      handler: ./handlers/admin.pars
      auth: required
  ```

### Phase 2: Multi-user
- User table in SQLite
- Registration/login handlers (built-in or Parsley-based?)
- User object passed to handlers

### Phase 3: WebAuthn/Passkeys
- Add as alternative to passwords
- Modern browsers only

---

## Discussion Points

1. **Is single-user mode enough for v1?**
   - Covers personal blogs, admin dashboards, internal tools
   - Multi-user could come later

2. **Should Basil provide login UI or just the mechanism?**
   - Option A: Basil serves a built-in login page
   - Option B: Basil provides auth primitives, user builds UI in Parsley
   - Option C: Both (default UI that can be overridden)

3. **Where does session data live?**
   - In-memory (simple, lost on restart)
   - SQLite (persistent, already have it)
   - Encrypted cookie (stateless)

4. **How does the handler know who's logged in?**
   ```parsley
   // Option A: request.user object
   if request.user {
     <p>Hello, {request.user.email}!</p>
   }
   
   // Option B: separate 'user' variable
   if user {
     <p>Hello, {user.email}!</p>
   }
   ```

---

## Next Steps

- [ ] Discuss and decide on Phase 1 scope
- [ ] Create implementation plan
- [ ] Implement

---

## References

- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [WebAuthn Guide](https://webauthn.guide/)
- [Go bcrypt package](https://pkg.go.dev/golang.org/x/crypto/bcrypt)
