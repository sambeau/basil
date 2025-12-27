# Frequently Asked Questions

Questions answered during development. AI adds new entries when answering "how do I..." questions.

---

## Process Questions

### How do I start a new feature?
Use `/new-feature` in Copilot Chat with a description of what you want. The AI will guide you through spec creation, planning, and implementation.

*Added: 2025-11-30*

### How do I report a bug?
Use `/fix-bug` in Copilot Chat with a description of the problem. Provide reproduction steps when asked.

*Added: 2025-11-30*

### How do I make a release?
Use `/release` in Copilot Chat. AI will gather changes, determine version bump, and prepare changelog. You approve and execute git commands.

*Added: 2025-11-30*

### Where do deferred items go?
Items that can't be completed in current scope go to `BACKLOG.md`. They're categorized by priority and linked back to their source feature/bug.

*Added: 2025-11-30*

---

## File Questions

### What's the difference between specs and plans?
- **Specs** (`docs/specs/`): What we're building and why. Written mostly by human.
- **Plans** (`docs/plans/`): How we're building it, step by step. Written mostly by AI.

*Added: 2025-11-30*

### Where do I find the template for a new spec?
Templates are in `.github/templates/`. The AI uses these automatically when you use prompt files.

*Added: 2025-11-30*

### What's AGENTS.md for?
It's the first thing AI reads before any task. Contains build commands, project structure, workflow rules, and common pitfalls. Think of it as onboarding documentation for AI.

*Added: 2025-11-30*

---

## Git Questions

### Who does git commits?
AI commits to feature/bug branches, merges to main, and creates release tags. Human reviews and approves.

*Added: 2025-11-30*

### What commit format should I use?
Conventional Commits. See `.github/instructions/commits.instructions.md` or the [Cheatsheet](cheatsheet.md).

*Added: 2025-11-30*

---

## ID Questions

### How do IDs get assigned?
AI reads `ID_COUNTER.md`, takes the next number, and increments the counter. You don't need to do anything.

*Added: 2025-11-30*

### Can I manually assign an ID?
Yes, just edit `ID_COUNTER.md` directly. Make sure to increment the "Next ID" so it's not reused.

*Added: 2025-11-30*

---

## Parsley Questions

### How do I create interactive HTML components?
Use Parts - create a `.part` file that exports view functions. Each view can respond to user interactions via `part-click` or `part-submit` attributes. See `examples/parts/` for a working example.

*Added: 2025-12-10*

### What's the difference between .pars and .part files?
- `.pars` files are normal Parsley modules that can export anything
- `.part` files can only export functions (view functions for interactive components)
- `.part` files need routes in `basil.yaml` to be accessible via HTTP

*Added: 2025-12-10*

### How do Parts update without reloading the page?
When you use a `<Part/>` tag, Basil automatically injects JavaScript that listens for `part-click` and `part-submit` events. When triggered, it fetches the new view from the server and updates just that Part's HTML.

*Added: 2025-12-10*

### How do I auto-refresh a Part?
Add `part-refresh={ms}` to the `<Part/>` tag. The timer resets after interactions and pauses when the tab is hidden.

*Added: 2025-12-10*

### How do I lazy-load a Part?
Use `part-load="view"` (optionally with `part-load-threshold={px}`) on the `<Part/>` tag. Start with a placeholder view and the runtime will load the target view when the Part approaches the viewport.

*Added: 2025-12-10*

---

## Authentication Questions

### How do I protect an entire section of my site?

Use `auth.protected_paths` in your `basil.yaml`:

```yaml
auth:
  enabled: true
  protected_paths:
    - /dashboard
    - /settings
```

All URLs starting with `/dashboard` or `/settings` will require authentication. Works with both site mode and routes mode.

*Added: 2025-12-27*

### How do I make one page public under a protected path?

Use `auth: none` on the specific route:

```yaml
auth:
  enabled: true
  protected_paths:
    - /admin

routes:
  - path: /admin/login
    handler: ./handlers/admin-login.pars
    auth: none    # Public, even though /admin is protected
```

*Added: 2025-12-27*

### How do I restrict a route to admins only?

Three options depending on your setup:

**1. Protected paths with roles:**
```yaml
auth:
  protected_paths:
    - path: /admin
      roles: [admin]
```

**2. Route-level roles:**
```yaml
routes:
  - path: /admin/users
    handler: ./handlers/admin-users.pars
    auth: required
    roles: [admin]
```

**3. API wrapper (for API routes):**
```parsley
let api = import @std/api

export get = api.adminOnly(fn(req) {
    {users: getAllUsers()}
})
```

*Added: 2025-12-27*

### How do I check the user's role in a handler?

Access `basil.auth.user.role`:

```parsley
if (basil.auth.user && basil.auth.user.role == "admin") {
    <AdminPanel/>
} else {
    <p>"Access denied"</p>
}
```

*Added: 2025-12-27*

---

<!-- AI: Add new Q&A entries above this line, with *Added: YYYY-MM-DD* -->