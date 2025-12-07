# Design Document: std/web Module

**Status:** Draft  
**Date:** December 2025  
**Related:** FEAT-034 (std/api)

---

## Overview

This document explores applying patterns learned from FEAT-034 (std/api) to web/HTML output. The goal is to identify a minimal, composable subset of features that would benefit web views without duplicating existing functionality.

---

## What FEAT-034 Teaches Us

### Patterns That Work Well

1. **Schema as Foundation** - Define once, use everywhere (validation, table binding, documentation)
2. **Sensible Defaults** - Auth-on, pagination, rate limiting with easy opt-out
3. **Wrapper Functions** - `public(fn)` style is clean and composable
4. **Module-as-Resource** - Exports map to capabilities (`get`, `post`, `routes`)

### What Could Transfer to Web/HTML

| API Pattern | Web Equivalent | Notes |
|-------------|----------------|-------|
| `schema.validate(schema, data)` | Form validation | Already usable! |
| `schema.table(schema, db, "users")` | Same - data access | Works in any handler |
| Auth wrappers (`public`, `adminOnly`) | Page protection | Already works via route config |
| Rate limiting | Less critical | Mostly for APIs, bots |
| `routes` export for composition | Component routing? | Interesting idea |

---

## Potential New Patterns for Web

### 1. Schema-Driven Forms

The schema could generate forms automatically:

```parsley
let User = schema.define("User", {
  name: schema.string({required: true, min: 2}),
  email: schema.email({required: true}),
  role: schema.enum({values: ["user", "admin"]})
})

// Generate a form from schema
<schema.form schema={User} action="/users" method="POST">
  <button>Create User</button>
</schema.form>

// Or validate form submission
let result = schema.validate(User, basil.http.request.form)
if result.errors {
  <schema.form schema={User} errors={result.errors} values={result.value}/>
} else {
  // Success
}
```

### 2. Table Binding for Web Views

Already works! But could add helpers:

```parsley
let Users = schema.table(User, basil.sqlite, "users")

// Pagination is automatic from query params
let users = Users.all()  // respects ?limit=&offset=

// Could add: render helpers
<table>
  <schema.thead schema={User}/>  // Generate headers from schema
  {for user in users {
    <schema.row data={user} schema={User}/>
  }}
</table>
```

### 3. Route Composition via Tags/Components

What if page handlers could compose like API modules?

```parsley
// layouts/App.pars
export App = fn({children, title}) {
  <html>
    <head><title>{title}</title></head>
    <body>
      <nav>...</nav>
      {children}
    </body>
  </html>
}

// pages/users.pars - export structure similar to API
export default = fn(req) {
  <App title="Users">
    <UserList/>
  </App>
}

// Optional: nested routes like API
export routes = {
  "/:id": import(@./user-detail.pars),
  "/new": import(@./user-new.pars)
}
```

### 4. Auth for Web Pages

The existing route-level auth works, but could add component-level:

```parsley
// Only render if user has role
<auth.guard roles={["admin"]}>
  <AdminPanel/>
</auth.guard>

// Or conditional rendering
if basil.http.user?.role == "admin" {
  <AdminPanel/>
}
```

---

## Feature Analysis: Do We Need These for Web?

| Feature | API Need | Web Need | Notes |
|---------|----------|----------|-------|
| Schema validation | ✅ Critical | ✅ Useful | Forms need validation |
| Table binding | ✅ Critical | ✅ Useful | Same CRUD needs |
| Auth wrappers | ✅ Critical | ⚠️ Route-level OK | Page auth via config works |
| Rate limiting | ✅ Critical | ⚠️ Edge cases | Only for login, search |
| Nested routes | ✅ Useful | ⚠️ Maybe | File-based routing is simpler |
| Error helpers | ✅ Critical | ⚠️ Different | Pages use error pages, not JSON |

---

## Proposed: `std/web` Module

A minimal, composable module for web views:

```parsley
let web = import(@std/web)

// Form helpers
web.form(schema, {action, method, values?, errors?})
web.input(field, {value?, error?})

// Table helpers  
web.table(data, schema, {class?})
web.pagination(total, {limit, offset})

// Auth guards (component-level)
web.guard({roles?, user?}, children)
web.loginForm({action, redirect?})

// Flash messages
web.flash()  // Render any flash messages
web.setFlash(type, message)  // Set for next request
```

### Example Usage

```parsley
let {form, table, pagination, guard} = import(@std/web)
let schema = import(@std/schema)

let User = schema.define("User", {
  name: schema.string({required: true}),
  email: schema.email({required: true})
})
let Users = schema.table(User, basil.sqlite, "users")

// In handler
let users = Users.all()

<html>
<body>
  <guard roles={["admin"]}>
    <a href="/users/new">Add User</a>
  </guard>
  
  <table schema={User} data={users}/>
  <pagination total={Users.count()} limit={20}/>
</body>
</html>
```

---

## What's Actually New vs Already Possible?

### Already works today
- `schema.validate()` for form validation
- `schema.table()` for CRUD
- Route-level auth via config
- Manual form building

### Would need implementation
- `std/web` module with form/table generators
- Component-level auth guards
- Flash messages
- Pagination component
- Schema-to-HTML field type mapping

---

## Recommendation

1. **Don't duplicate** - `std/schema` and table binding work for web already
2. **Add `std/web`** - Light helpers for forms, tables, pagination
3. **Skip rate limiting** - Not needed for typical web pages
4. **Consider nested routes** - But low priority, file-based works

### Minimal Valuable Addition

The minimal valuable addition would be:

| Feature | Priority | Rationale |
|---------|----------|-----------|
| **Form generation** from schema | High | Biggest win - eliminates boilerplate |
| **Flash messages** | Medium | Session-based, common need |
| **Pagination component** | Medium | Repetitive to hand-write |
| Auth guards (component) | Low | Route-level works |
| Table rendering | Low | Easy to hand-write |

---

## Open Questions

1. **Should form generation be in `std/schema` or `std/web`?**
   - Pro `std/schema`: keeps schema-related logic together
   - Pro `std/web`: separates data from presentation

2. **How should flash messages persist?**
   - Session-based (requires auth/session system)
   - Cookie-based (simpler, size-limited)

3. **Should pagination be automatic or explicit?**
   - `Users.all()` already respects query params
   - Component just renders the UI controls

4. **Tag-based vs function-based helpers?**
   - `<web.form schema={User}/>` vs `web.form(User, {})`
   - Tags feel more natural in Parsley HTML context

---

## Next Steps

If this design is approved:

1. Create FEAT spec for `std/web`
2. Start with form generation (highest value)
3. Add flash messages
4. Add pagination component
5. Evaluate auth guards based on usage
