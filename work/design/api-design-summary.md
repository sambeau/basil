# Basil API Design Summary

**Status:** Ready for review  
**Date:** December 2025  
**Purpose:** Complete picture of proposed `std/api` design for Basil

---

## Table of Contents

1. [Philosophy](#philosophy)
2. [Schema Types](#schema-types)
3. [Validation](#validation)
4. [Table Binding](#table-binding)
5. [API Routes](#api-routes)
6. [Authentication](#authentication)
7. [Sensible Defaults](#sensible-defaults)
8. [What You Don't Write](#what-you-dont-write)
9. [When You Need More](#when-you-need-more)
10. [Passkey Integration for Web Apps](#passkey-integration-for-web-apps)
11. [Middleware (Or Lack Thereof)](#middleware-or-lack-thereof)
12. [ID Generation](#id-generation-stdid)
13. [Complete Examples](#complete-examples)

---

## Philosophy

### Basil is an HTML Server First

Basil's primary job is generating HTML from Parsley handlers:

```parsley
// This is what Basil is FOR
let todos = basil.sqlite <=??=> "SELECT * FROM todos WHERE done = false"

<html>
<body>
  <h1>My Todos</h1>
  <ul>
  {for (todo in todos) {
    <li>{todo.title}</li>
  }}
  </ul>
</body>
</html>
```

The API is a **side-gig**—useful for mobile apps, SPAs, third-party integrations, and AJAX endpoints, but not the main act.

### Design Principles

1. **Secure by default** — Auth is on. You opt out, not in.
2. **Batteries included** — Validation, rate limiting, pagination come free.
3. **No magic** — Everything is explicit and inspectable.
4. **Composition over inheritance** — Objects and functions, not class hierarchies.
5. **Protect users from themselves** — Sensible limits prevent foot-shooting.

---

## Schema Types

Schemas are the foundation. Define once, get validation, sanitization, and documentation everywhere.

### Basic Definition

```parsley
let schema = import("std/schema")

let Todo = schema.define("Todo", {
    title: schema.string({required: true, min: 1, max: 200}),
    done: schema.boolean({default: false}),
    priority: schema.enum({values: ["low", "medium", "high"], default: "medium"})
})
```

### Types Bring Their Own Behavior

Each type function returns an object with built-in validation, sanitization, and documentation:

```parsley
// schema.email() internally is something like:
{
    type: "string",
    format: "email",
    sanitize: ["trim", "lowercase"],
    validate: fn(v) { valid.email(v) },
    example: "user@example.com",
    description: "Email address"
}

// So when you write:
let User = schema.define("User", {
    email: schema.email({required: true})
})

// "email" gets: trim, lowercase, email format validation, docs
// You didn't ask for any of that—it came with the type
```

### Available Types

| Type | Brings | Options |
|------|--------|---------|
| `schema.string()` | trim | `required`, `min`, `max`, `pattern` |
| `schema.email()` | trim, lowercase, format check | `required` |
| `schema.url()` | trim, format check | `required`, `protocols` |
| `schema.phone()` | format check | `required` |
| `schema.integer()` | type coercion | `required`, `min`, `max` |
| `schema.number()` | type coercion | `required`, `min`, `max` |
| `schema.boolean()` | type coercion | `default` |
| `schema.enum()` | allowed values check | `values`, `default` |
| `schema.date()` | format check | `required`, `min`, `max` |
| `schema.datetime()` | format check | `required` |
| `schema.money()` | precision handling | `required`, `currency` |
| `schema.array()` | nested validation | `of`, `min`, `max` |
| `schema.object()` | nested schema | `properties` |

### Type Composition

Override defaults with the merge operator (`++`):

```parsley
// Start with email type, customize
let workEmail = schema.email() ++ {
    required: true,
    description: "Work email address",
    pattern: "^[^@]+@company\\.com$"
}

let Employee = schema.define("Employee", {
    email: workEmail
})
```

---

## Validation

### Manual Validation

```parsley
let schema = import("std/schema")

let ContactForm = schema.define("ContactForm", {
    name: schema.string({required: true, min: 1, max: 100}),
    email: schema.email({required: true}),
    message: schema.string({required: true, min: 10, max: 5000})
})

// In a handler
let {value, errors} = schema.validate(ContactForm, basil.http.request.form)

if (errors) {
    <div class="errors">
    {for (e in errors) {
        <p class="error">{e.field}: {e.message}</p>
    }}
    </div>
} else {
    // value is sanitized and validated
    sendEmail(value)
    <p class="success">Thanks for your message!</p>
}
```

### Sanitization

Sanitization happens automatically during validation, but you can also call it explicitly:

```parsley
// Sanitize without validating
let clean = schema.sanitize(ContactForm, dirtyData)

// clean.email is now trimmed and lowercase
// clean.name is trimmed
// etc.
```

### Error Format

```parsley
// errors is an array of:
{
    field: "email",
    code: "format",
    message: "Must be a valid email address",
    value: "not-an-email"
}
```

---

## Table Binding

Connect a schema to a database table for query helpers.

### Basic Binding

```parsley
let schema = import("std/schema")

let Todo = schema.define("Todo", {
    title: schema.string({required: true}),
    done: schema.boolean({default: false})
})

// Bind schema to table - returns query helpers
let Todos = schema.table(Todo, basil.sqlite, "todos")
```

### What You Get

```parsley
// Todos is now an object with query methods:

Todos.all()                          // SELECT * FROM todos
Todos.find(id)                       // SELECT ... WHERE id = ?
Todos.where({done: false})           // SELECT ... WHERE done = ?
Todos.insert({title: "New"})         // INSERT INTO todos ...
Todos.update(id, {done: true})       // UPDATE todos SET ... WHERE id = ?
Todos.delete(id)                     // DELETE FROM todos WHERE id = ?
```

### How It Works (No Magic)

`schema.table()` returns a plain object with functions:

```parsley
// It's essentially:
{
    all: fn() {
        basil.sqlite <=??=> "SELECT * FROM todos"
    },
    find: fn(id) {
        basil.sqlite <=?=> "SELECT * FROM todos WHERE id = ?", [id]
    },
    insert: fn(data) {
        let clean = schema.sanitize(Todo, data)
        let {errors} = schema.validate(Todo, clean)
        if (errors) { {error: errors} }
        else {
            let result = basil.sqlite <=!=> "INSERT INTO todos ...", [...]
            {value: {...clean, id: result.lastId}}
        }
    }
    // etc.
}
```

**You can always drop down to raw SQL:**

```parsley
// Complex queries? Just use SQL directly
let results = basil.sqlite <=??=> "
    SELECT t.*, u.name as owner_name
    FROM todos t
    JOIN users u ON t.owner_id = u.id
    WHERE t.done = false
    ORDER BY t.priority DESC
"
```

### Extending with Custom Queries

```parsley
let TodoQueries = Todos ++ {
    pending: fn() {
        Todos.where({done: false})
    },
    
    byOwner: fn(userId) {
        basil.sqlite <=??=> "SELECT * FROM todos WHERE owner_id = ?", [userId]
    },
    
    stats: fn() {
        basil.sqlite <=?=> "
            SELECT 
                COUNT(*) as total,
                SUM(CASE WHEN done THEN 1 ELSE 0 END) as completed
            FROM todos
        "
    }
}

// Use them
let pending = TodoQueries.pending()
let stats = TodoQueries.stats()
```

---

## API Routes

### The Module Pattern

Define API endpoints by exporting handler functions from a module:

**api/todos.pars:**

```parsley
let schema = import("std/schema")

let Todo = schema.define("Todo", {
    title: schema.string({required: true, min: 1, max: 200}),
    done: schema.boolean({default: false})
})

let Todos = schema.table(Todo, basil.sqlite, "todos")

// GET /api/todos
export get = fn(req) {
    Todos.where({owner_id: req.user.id})
}

// POST /api/todos
export post = fn(req) {
    Todos.insert({...req.form, owner_id: req.user.id})
}

// GET /api/todos/:id
export getById = fn(req) {
    Todos.find(req.params.id)
}

// PUT /api/todos/:id
export put = fn(req) {
    Todos.update(req.params.id, req.form)
}

// DELETE /api/todos/:id
export delete = fn(req) {
    Todos.delete(req.params.id)
}
```

### The Routes Object

Compose multiple modules into a single API entry point:

**api/index.pars:**

```parsley
let {public} = import("std/api")

export routes = {
    "/todos": import(@./todos.pars),
    "/users": import(@./users.pars),
    "/health": {
        get: public(fn(req) { {status: "ok", time: now()} })
    }
}
```

**basil.yaml:**

```yaml
routes:
  - path: /api/
    module: ./api/index.pars
```

### Route Mapping

| Export | HTTP Method | Path |
|--------|-------------|------|
| `get` | GET | `/resource` |
| `post` | POST | `/resource` |
| `getById` | GET | `/resource/:id` |
| `put` | PUT | `/resource/:id` |
| `patch` | PATCH | `/resource/:id` |
| `delete` | DELETE | `/resource/:id` |

### Response Format

Return a dictionary for JSON, return HTML for HTML:

```parsley
// JSON response (return dict/array)
export get = fn(req) {
    {data: Todos.all(), count: Todos.count()}
}

// Custom status/headers
export post = fn(req) {
    let result = Todos.insert(req.form)
    if (result.error) {
        {status: 400, body: {errors: result.error}}
    } else {
        {status: 201, body: result.value}
    }
}
```

---

## Authentication

### Three Patterns, One Philosophy

| Use Case | Mechanism | How |
|----------|-----------|-----|
| Web apps (humans) | Passkeys | WebAuthn → session token |
| Machine-to-machine | API Keys | Pre-shared key in header |
| Internal/trusted | `public()` | Opt-out of auth per handler |

### Auth is ON by Default

Every handler requires authentication by default. Use wrapper functions to change this:

```parsley
// api/todos.pars
let {public, adminOnly} = import("std/api")

// Auth required (default) - no wrapper needed
export get = fn(req) {
    // req.user is guaranteed to exist here
    Todos.where({owner_id: req.user.id})
}

// Explicitly public - anyone can call this
export getById = public(fn(req) {
    Todos.find(req.params.id)
})

// Auth required, but only admins
export delete = adminOnly(fn(req) {
    Todos.delete(req.params.id)
})
```

### Auth Wrappers

| Wrapper | Effect |
|---------|--------|
| *(none)* | Auth required (default) |
| `public(fn)` | No auth required |
| `adminOnly(fn)` | Auth required + admin role |
| `roles(["editor", "admin"], fn)` | Auth required + specific roles |

These wrappers return a new function with auth metadata attached. The API runtime reads this metadata to enforce access control.

### Roles Are Roll-Your-Own

Basil's core auth provides identity (`req.user.id`), not roles. Roles are application-specific:

```parsley
// Your schema defines roles
let User = schema.define("User", {
    name: schema.string({required: true}),
    email: schema.email({required: true}),
    role: schema.enum({values: ["user", "editor", "admin"], default: "user"})
})
```

**Option A: Convention-based (simple)**

If your users table has a `role` column, `adminOnly()` just works:

```yaml
# basil.yaml
api:
  auth:
    roleColumn: role  # Default: looks for "role" in users table
```

**Option B: Custom role resolver (flexible)**

For complex scenarios (roles in separate table, multiple roles per user):

```parsley
// config/auth.pars
let Users = schema.table(User, basil.sqlite, "users")
let UserRoles = schema.table(UserRole, basil.sqlite, "user_roles")

export roleResolver = fn(userId) {
    // Return array of role names
    UserRoles.where({user_id: userId}).map(fn(r) { r.role })
}
```

**Option C: Inline role checks (explicit)**

Skip the modifiers entirely and check roles in your handler:

```parsley
export delete = fn(req) {
    let user = Users.find(req.user.id)
    if (user.role != "admin") { forbidden("Admin only") }
    Todos.delete(req.params.id)
}
```

**With options (wrapper style):**

For complex cases, you can use the wrapper function style:

```parsley
let {auth, public} = import("std/api")

// Custom rate limit for this handler
export post = auth({rateLimit: 10}, fn(req) {
    Todos.insert(req.form ++ {owner_id: req.user.id})
})

// Public with custom cache
export get = public({cache: @5m}, fn(req) {
    Todos.all()
})
```

### API Keys (Machine-to-Machine)

For services, scripts, and integrations:

**basil.yaml:**

```yaml
api:
  auth:
    apiKeys: true
```

**Client usage:**

```bash
curl -H "Authorization: Bearer sk_live_abc123..." \
     https://api.example.com/api/todos
```

**Generating keys:**

```bash
basil apikey create --name "CI Pipeline" --scopes "todos:read,todos:write"
# Output: sk_live_abc123def456...
```

### Passkeys (Web Apps)

For humans using browsers:

**basil.yaml:**

```yaml
auth:
  passkey: true
  sessionTTL: 24h
```

The auth flow is automatic when `auth.passkey` is enabled:
1. User registers/logs in via `<PasskeyRegister/>` or `<PasskeyLogin/>`
2. Basil sets a session cookie
3. API requests include the cookie automatically
4. `req.user` is populated in handlers

### The User Object

When authenticated, handlers receive:

```parsley
req.user.id        // "usr_abc123"
req.user.name      // "Sam Phillips"
req.user.email     // "sam@example.com" or null
req.user.created   // @2025-12-01T10:00:00
```

---

## Sensible Defaults

### What's Automatic

| Feature | Default | Why |
|---------|---------|-----|
| Rate limiting | 60 req/min | Protects interpreted routes |
| Pagination | 20 items, max 100 | Prevents accidental full-table dumps |
| Auth | Required | Secure by default |
| Validation | Schema-based | No unvalidated input |
| SQL injection | Parameterized queries | Can't be turned off |
| Error format | Consistent JSON | Predictable for clients |
| Cache headers | `private, no-cache` | Safe default |

### Rate Limiting

```parsley
// Default: 60 requests per minute per IP
// Cached routes can handle more, so tune them:

export rateLimit = {
    requests: 300,
    window: @1m
}

export get = fn(req) { ... }
```

### Pagination

```parsley
// Automatic pagination on list endpoints
// GET /api/todos?limit=50&offset=100

export get = fn(req) {
    // Limit is clamped to 100 max, default 20
    let limit = min(req.query.limit ?? 20, 100)
    let offset = req.query.offset ?? 0
    
    Todos.list({limit: limit, offset: offset})
}
```

---

## What You Don't Write

A minimal API endpoint gets all of this for free:

```parsley
// api/todos.pars

// Define the schema
let Todo = schema.define("Todo", {
    title: schema.string({required: true}),
    done: schema.boolean({default: false})
})

// Bind to database table
let Todos = schema.table(Todo, basil.sqlite, "todos")

export get = fn(req) { Todos.where({owner_id: req.user.id}) }
export post = fn(req) { Todos.insert(req.form ++ {owner_id: req.user.id}) }
```

**What you got without writing:**

| Feature | How |
|---------|-----|
| Authentication | Default on, user available via `req.user` |
| Input validation | Schema validates on insert |
| Input sanitization | Schema sanitizes (trim, lowercase, etc.) |
| SQL injection protection | Parameterized queries always |
| Rate limiting | 60/min default |
| Pagination | Built into list queries |
| Error responses | Consistent format |
| Content-Type | Auto-detected from return value |

---

## When You Need More

### Custom Rate Limits

```parsley
// For expensive operations
export rateLimit = {requests: 10, window: @1m}

// For public cached data
export rateLimit = {requests: 1000, window: @1m}
```

### Custom Validation

```parsley
export post = fn(req) {
    // Additional business logic validation
    if (Todos.count({owner_id: req.user.id}) >= 100) {
        {status: 429, body: {error: "Todo limit reached"}}
    } else {
        Todos.insert({...req.form, owner_id: req.user.id})
    }
}
```

### Complex Queries

```parsley
export get = fn(req) {
    // Drop down to raw SQL for complex needs
    basil.sqlite <=??=> "
        SELECT t.*, 
               COUNT(c.id) as comment_count,
               MAX(c.created_at) as last_comment
        FROM todos t
        LEFT JOIN comments c ON c.todo_id = t.id
        WHERE t.owner_id = ?
        GROUP BY t.id
        ORDER BY t.priority DESC, t.created_at DESC
        LIMIT ? OFFSET ?
    ", [req.user.id, req.query.limit ?? 20, req.query.offset ?? 0]
}
```

### Per-Route Cache Control

```parsley
export cache = {
    maxAge: @5m,
    scope: "private"  // or "public" for CDN caching
}

export get = fn(req) { ... }
```

### Access Control

```parsley
// Owner-only access
export getById = fn(req) {
    let todo = Todos.find(req.params.id)
    if (todo.owner_id != req.user.id) {
        {status: 403, body: {error: "Not your todo"}}
    } else {
        todo
    }
}

// Admin-only endpoint
export delete = fn(req) {
    if (req.user.role != "admin") {
        {status: 403, body: {error: "Admin only"}}
    } else {
        Todos.delete(req.params.id)
    }
}
```

---

## Error Handling with fail()

### The Problem with Return-Value Errors

The examples above use return values for errors:

```parsley
export getById = fn(req) {
    let todo = Todos.find(req.params.id)
    if (!todo || todo.owner_id != req.user.id) {
        {status: 404, body: {error: "Not found"}}  // Awkward
    } else {
        todo
    }
}
```

This is verbose and clutters the happy path. Every handler needs this boilerplate.

### The Solution: fail() with Error Codes

Parsley's `fail()` function creates catchable errors. Currently it only takes a message:

```parsley
fail("Something went wrong")  // Creates error with code USER-0001
```

**Proposed enhancement:** Allow `fail(code, message)`:

```parsley
fail("HTTP-404", "Todo not found")   // Code: HTTP-404
fail("HTTP-403", "Forbidden")        // Code: HTTP-403
fail("VALIDATION", "Email required") // Code: VALIDATION
```

### API Error Helpers

The `std/api` module provides helpers that call `fail` with appropriate HTTP codes:

```parsley
let api = import("std/api")

api.badRequest(msg)    // fail("HTTP-400", msg ?? "Bad request")
api.unauthorized(msg)  // fail("HTTP-401", msg ?? "Unauthorized")
api.forbidden(msg)     // fail("HTTP-403", msg ?? "Forbidden")
api.notFound(msg)      // fail("HTTP-404", msg ?? "Not found")
api.conflict(msg)      // fail("HTTP-409", msg ?? "Conflict")
api.gone(msg)          // fail("HTTP-410", msg ?? "Gone")
api.unprocessable(msg) // fail("HTTP-422", msg ?? "Unprocessable entity")
api.tooMany(msg)       // fail("HTTP-429", msg ?? "Too many requests")
```

### Clean Handler Code

With error helpers, handlers become beautifully simple:

```parsley
let {notFound, forbidden} = import("std/api")

export getById = fn(req) {
    let todo = Todos.find(req.params.id)
    if (!todo) { notFound("Todo not found") }
    if (todo.owner_id != req.user.id) { forbidden() }
    todo
}

export put = fn(req) {
    let todo = Todos.find(req.params.id)
    if (!todo) { notFound() }
    if (todo.owner_id != req.user.id) { forbidden() }
    Todos.update(req.params.id, req.form)
}

export delete = fn(req) {
    let todo = Todos.find(req.params.id)
    if (!todo) { notFound() }
    if (todo.owner_id != req.user.id) { forbidden() }
    Todos.delete(req.params.id)
}
```

### How the API Runtime Handles Errors

The API runtime wraps every handler in error handling:

```
1. Call handler
2. If handler returns normally → 200 OK with result as JSON
3. If handler fails with HTTP-* code → extract status, return error JSON
4. If handler fails with other code → 500 Internal Server Error
```

**Error response format:**

```json
{
    "error": "Todo not found",
    "code": "HTTP-404"
}
```

### Error Code Reference

| Helper | Error Code | HTTP Status | Default Message |
|--------|------------|-------------|-----------------|
| `badRequest()` | `HTTP-400` | 400 | "Bad request" |
| `unauthorized()` | `HTTP-401` | 401 | "Unauthorized" |
| `forbidden()` | `HTTP-403` | 403 | "Forbidden" |
| `notFound()` | `HTTP-404` | 404 | "Not found" |
| `conflict()` | `HTTP-409` | 409 | "Conflict" |
| `gone()` | `HTTP-410` | 410 | "Gone" |
| `unprocessable()` | `HTTP-422` | 422 | "Unprocessable entity" |
| `tooMany()` | `HTTP-429` | 429 | "Too many requests" |

### Custom Error Codes

You can use `fail()` directly for domain-specific errors:

```parsley
// Business logic errors (will become 500s unless caught)
fail("QUOTA-EXCEEDED", "You have reached your todo limit")
fail("DUPLICATE-TITLE", "A todo with this title already exists")

// Or map them to HTTP codes
if (Todos.count({owner_id: req.user.id}) >= 100) {
    api.unprocessable("Todo limit reached (max 100)")
}
```

### Implementation Requirements

To support this pattern:

1. **Parsley change:** Extend `fail()` to accept optional code: `fail(code, message)` or `fail(message)`
2. **Error object:** Expose code when caught by `try` (e.g., `error.code`)
3. **API runtime:** Check for `HTTP-*` prefix and map to HTTP status codes
4. **std/api module:** Provide helper functions that call `fail` with appropriate codes

---

## Passkey Integration for Web Apps

### How It Works Today

Basil already has full passkey support for HTML apps via built-in components:

**Registration:**

```parsley
<PasskeyRegister
  name_placeholder="Your name"
  email_placeholder="you@example.com"
  button_text="Create account"
  redirect="/dashboard"
  class="auth-form"
/>
```

This renders a form with embedded JavaScript that:
1. Collects name/email
2. Calls `POST /__auth/register/begin` to get a WebAuthn challenge
3. Invokes the browser's `navigator.credentials.create()` API
4. Sends the credential to `POST /__auth/register/finish`
5. Receives session cookie and recovery codes
6. Redirects to the specified URL

**Login:**

```parsley
<PasskeyLogin
  button_text="Sign in with passkey"
  redirect="/dashboard"
  class="login-btn"
/>
```

This renders a button with JavaScript that:
1. Calls `POST /__auth/login/begin` to get a challenge
2. Invokes `navigator.credentials.get()` (browser shows passkey picker)
3. Sends the assertion to `POST /__auth/login/finish`
4. Receives session cookie
5. Redirects

### Bridging to API Auth

The session cookie from passkey auth works automatically for API routes:

```parsley
// Web app logs in via <PasskeyLogin/>
// Browser now has session cookie

// AJAX calls to API include cookie automatically:
// fetch('/api/todos')  <-- Cookie sent, user authenticated
```

For SPAs or cases where you need a bearer token instead of cookies:

```parsley
// After passkey login, request a token
// POST /__auth/token
// Response: {token: "eyJ...", expires: "2025-12-02T10:00:00Z"}

// Use token in Authorization header:
// fetch('/api/todos', {headers: {Authorization: 'Bearer eyJ...'}})
```

### The JavaScript Side (Webapp)

Basil's `<PasskeyLogin/>` and `<PasskeyRegister/>` components embed this JavaScript:

```javascript
// Simplified - actual implementation handles errors, loading states, etc.

async function registerPasskey(name, email) {
  // 1. Get challenge from server
  const beginResp = await fetch('/__auth/register/begin', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({name, email})
  });
  const {options, challenge_id} = await beginResp.json();
  
  // 2. Create credential with browser API
  const credential = await navigator.credentials.create({publicKey: options.publicKey});
  
  // 3. Send credential to server
  const finishResp = await fetch('/__auth/register/finish', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({
      challenge_id,
      response: {
        id: credential.id,
        rawId: bufferToBase64(credential.rawId),
        type: credential.type,
        response: {
          clientDataJSON: bufferToBase64(credential.response.clientDataJSON),
          attestationObject: bufferToBase64(credential.response.attestationObject)
        }
      }
    })
  });
  
  // 4. Server sets session cookie, returns user + recovery codes
  const {user, recovery_codes} = await finishResp.json();
  return {user, recovery_codes};
}

async function loginPasskey() {
  // 1. Get challenge
  const beginResp = await fetch('/__auth/login/begin', {method: 'POST'});
  const {options, challenge_id} = await beginResp.json();
  
  // 2. Get credential (browser shows passkey picker)
  const credential = await navigator.credentials.get({publicKey: options.publicKey});
  
  // 3. Send to server
  const finishResp = await fetch('/__auth/login/finish', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({
      challenge_id,
      response: {
        id: credential.id,
        rawId: bufferToBase64(credential.rawId),
        type: credential.type,
        response: {
          clientDataJSON: bufferToBase64(credential.response.clientDataJSON),
          authenticatorData: bufferToBase64(credential.response.authenticatorData),
          signature: bufferToBase64(credential.response.signature),
          userHandle: bufferToBase64(credential.response.userHandle)
        }
      }
    })
  });
  
  // 4. Session cookie is set, user is logged in
  const {user} = await finishResp.json();
  return user;
}
```

### What Basil's Implementation Provides

The existing `auth/` package handles:

| Component | What It Does |
|-----------|--------------|
| `WebAuthnManager` | Challenge generation, credential verification |
| `Handlers` | HTTP endpoints for begin/finish flows |
| `Middleware` | Session validation, user context injection |
| `DB` | User, credential, session, recovery code storage |
| `<Passkey*/>` components | Pre-built HTML + JS for common flows |

**For API auth, we leverage this by:**
1. Using the same session validation for API routes
2. Optionally adding token generation for bearer auth
3. Keeping the same `req.user` interface in handlers

---

## Middleware (Or Lack Thereof)

### The Goal: No Middleware

Traditional middleware chains add complexity and ordering headaches:

```javascript
// Express-style - order matters, hard to reason about
app.use(cors())
app.use(helmet())
app.use(rateLimit())
app.use(auth())
app.use(validate())
// ... your actual handler somewhere down here
```

### Basil's Approach: Built-in Behaviors

Instead of middleware, behaviors are **properties of routes**:

```parsley
let {public} = import("std/api")

// Route-level configuration, not middleware chain
export rateLimit = {requests: 100, window: @1m}
export cache = {maxAge: @5m}

// Auth is per-handler via wrappers
export get = public(fn(req) { ... })   // Anyone can call
export post = fn(req) { ... }          // Auth required (default)
```

### What Middleware Usually Does (And How Basil Handles It)

| Middleware | Basil Equivalent |
|------------|------------------|
| `cors()` | Server-level config in `basil.yaml` |
| `helmet()` | Built-in security headers (configurable) |
| `rateLimit()` | Route-level `export rateLimit = {...}` |
| `auth()` | Default on, wrap handler with `public()` to disable |
| `bodyParser()` | Automatic—JSON parsed into `req.form` |
| `validation()` | Schema validation in handler |
| `logging()` | Server-level, always on |
| `compression()` | Server-level config |

### When You Might Actually Need Middleware

Some cross-cutting concerns are hard to avoid:

**1. Request ID injection:**
Could be server-level (inject `X-Request-ID` on all requests).

**2. Timing/metrics:**
Server-level instrumentation, not per-route.

**3. Custom auth schemes:**
If API keys and passkeys aren't enough (OAuth?), might need a hook.

**4. Request transformation:**
Rare, but sometimes you need to modify requests before handlers see them.

### Possible Future: Hooks, Not Chains

If middleware-like behavior is needed, prefer **hooks** over chains:

```yaml
# basil.yaml
api:
  hooks:
    beforeRequest: ./hooks/before.pars   # Runs before every API request
    afterResponse: ./hooks/after.pars    # Runs after every API response
```

```parsley
// hooks/before.pars
export default = fn(req) {
    // Return modified request, or null to continue unchanged
    // Return {status: 403, body: {...}} to short-circuit
    if (someCondition) {
        {status: 403, body: {error: "Blocked"}}
    }
}
```

This keeps the model simple: hooks are functions, not a chain of opaque middleware.

---

## ID Generation (`std/id`)

### The Problem

IDs are everywhere in APIs, but there's confusion about what to use:
- UUID v4? v7? Which library?
- Sequential integers? (leaks info, guessable)
- NanoID? CUID? ULID?
- How long? What alphabet?

### Sensible Default

```parsley
let id = import("std/id")

id.new()  // Returns a sensible default ID
// "01HQJK8M3V9N2P4R5T6Y7W8X9Z" (ULID-like, sortable, URL-safe)
```

**Why this default?**
- **Sortable by time** — IDs created later sort later (great for "created_at" ordering)
- **URL-safe** — No special characters to escape
- **Compact** — 26 chars vs UUID's 36
- **Unguessable** — Can't iterate to find other records
- **Database-friendly** — Works as primary key in any DB

### When You Need Something Specific

```parsley
let id = import("std/id")

// Standard UUID v4 (random)
id.uuid()     // "f47ac10b-58cc-4372-a567-0e02b2c3d479"
id.uuidv4()   // Same as above (explicit)

// UUID v7 (time-sortable)
id.uuidv7()   // "018d6e8c-8c5b-7000-8000-000000000000"

// NanoID (compact, customizable)
id.nanoid()       // "V1StGXR8_Z5jdHi6B-myT" (21 chars, default)
id.nanoid(10)     // "IRFa-VaY2b" (custom length)

// CUID2 (secure, collision-resistant)
id.cuid()     // "clh3kz7z80000qzrmn831i7rn"

// Sequential (use sparingly—exposes count)
id.sequential()  // Requires DB, returns next int
```

### Integration with Schema

```parsley
let schema = import("std/schema")

let Todo = schema.define("Todo", {
    id: schema.id(),  // Uses std/id.new() by default for generation
    title: schema.string({required: true})
})

// Override the ID format
let LegacyTodo = schema.define("LegacyTodo", {
    id: schema.id({format: "uuid"}),  // Forces UUID v4
    title: schema.string({required: true})
})
```

### Table Binding Handles IDs

When you bind to a table, IDs are generated automatically on insert:

```parsley
let Todos = schema.table(Todo, basil.sqlite, "todos")

// ID is generated automatically
let todo = Todos.insert({title: "Buy milk"})
log(todo.id)  // "01HQJK8M3V9N2P4R5T6Y7W8X9Z"
```

### ID Formats Comparison

| Format | Length | Sortable | URL-safe | Example |
|--------|--------|----------|----------|---------|
| `id.new()` (default) | 26 | ✅ | ✅ | `01HQJK8M3V9N2P4R5T6Y` |
| `id.uuid()` | 36 | ❌ | ⚠️ | `f47ac10b-58cc-4372...` |
| `id.uuidv7()` | 36 | ✅ | ⚠️ | `018d6e8c-8c5b-7000...` |
| `id.nanoid()` | 21 | ❌ | ✅ | `V1StGXR8_Z5jdHi6B-myT` |
| `id.cuid()` | 25 | ✅ | ✅ | `clh3kz7z80000qzrmn...` |

**URL-safe note:** UUIDs need escaping in some contexts due to hyphens.

### API Keys Use std/id

API key generation also uses `std/id` internally:

```parsley
// Under the hood, API keys are:
// "sk_live_" + id.new()
// "sk_test_" + id.new()
```

This ensures consistency across the system.

---

## Complete Examples

### Example 1: Minimal Todo API

**basil.yaml:**

```yaml
server:
  host: localhost
  port: 8080

sqlite: ./data.db

auth:
  passkey: true

routes:
  - path: /
    handler: ./handlers/index.pars
  - path: /api/
    module: ./api/index.pars
```

**api/index.pars:**

```parsley
let {public} = import("std/api")

export routes = {
    "/todos": import(@./todos.pars),
    "/health": {
        get: public(fn(req) { {status: "ok"} })
    }
}
```

**api/todos.pars:**

```parsley
let schema = import("std/schema")
let {notFound, forbidden} = import("std/api")

let Todo = schema.define("Todo", {
    title: schema.string({required: true, min: 1, max: 200}),
    done: schema.boolean({default: false})
})

let Todos = schema.table(Todo, basil.sqlite, "todos")

export get = fn(req) {
    Todos.where({owner_id: req.user.id})
}

export post = fn(req) {
    Todos.insert(req.form ++ {owner_id: req.user.id})
}

export getById = fn(req) {
    let todo = Todos.find(req.params.id)
    if (!todo) { notFound() }
    if (todo.owner_id != req.user.id) { forbidden() }
    todo
}

export put = fn(req) {
    let todo = Todos.find(req.params.id)
    if (!todo) { notFound() }
    if (todo.owner_id != req.user.id) { forbidden() }
    Todos.update(req.params.id, req.form)
}

export delete = fn(req) {
    let todo = Todos.find(req.params.id)
    if (!todo) { notFound() }
    if (todo.owner_id != req.user.id) { forbidden() }
    Todos.delete(req.params.id)
}
```

### Example 2: Machine-to-Machine API

**basil.yaml:**

```yaml
server:
  host: 0.0.0.0
  port: 443

sqlite: ./data.db

api:
  auth:
    apiKeys: true     # Enable API key auth
    passkey: false    # No user login needed

https:
  auto: true
  email: admin@example.com

routes:
  - path: /api/
    module: ./api/index.pars
```

**api/webhooks.pars:**

```parsley
let schema = import("std/schema")

let Event = schema.define("Event", {
    type: schema.string({required: true}),
    payload: schema.object({required: true}),
    timestamp: schema.datetime({default: fn() { now() }})
})

let Events = schema.table(Event, basil.sqlite, "events")

// Receive webhooks from external service
export post = fn(req) {
    let result = Events.insert(req.form)
    if (result.error) {
        {status: 400, body: {errors: result.error}}
    } else {
        {status: 201, body: {id: result.value.id}}
    }
}

// Query events
export get = fn(req) {
    let since = req.query.since ?? "1970-01-01"
    Events.where({timestamp_gte: since})
}
```

### Example 3: Internal Service (No Auth)

**basil.yaml:**

```yaml
server:
  host: 0.0.0.0  # Internal network only
  port: 8080

sqlite: ./metrics.db

routes:
  - path: /api/
    module: ./api/index.pars
```

**api/index.pars:**

```parsley
let {public} = import("std/api")

// Everything is public - behind firewall
export routes = {
    "/metrics": {
        get: public(fn(req) {
            basil.sqlite <=??=> "SELECT * FROM metrics ORDER BY timestamp DESC LIMIT 100"
        }),
        post: public(fn(req) {
            basil.sqlite <=!=> "INSERT INTO metrics (name, value) VALUES (?, ?)",
                [req.form.name, req.form.value]
            {status: 201}
        })
    },
    "/health": {
        get: public(fn(req) { {status: "ok"} })
    }
}
```

### Example 4: Full-Featured User API

**api/users.pars:**

```parsley
let schema = import("std/schema")
let {forbidden} = import("std/api")

let User = schema.define("User", {
    name: schema.string({required: true, min: 1, max: 100}),
    email: schema.email({required: true}),
    bio: schema.string({max: 500}),
    avatar_url: schema.url(),
    role: schema.enum({values: ["user", "admin"], default: "user"})
})

let Users = schema.table(User, basil.sqlite, "users")

// Custom rate limit for user creation
export rateLimit = {requests: 10, window: @1h}

// Get current user profile
export get = fn(req) {
    Users.find(req.user.id)
}

// Update current user profile
export put = fn(req) {
    // Users can only update certain fields
    let allowed = {
        name: req.form.name,
        bio: req.form.bio,
        avatar_url: req.form.avatar_url
    }
    Users.update(req.user.id, allowed)
}

// Admin: list all users
export routes = {
    "/all": {
        get: fn(req) {
            if (req.user.role != "admin") { forbidden("Admin only") }
            Users.all()
        }
    }
}
```

---

## Summary

### The Complete Auth Story

| Scenario | Solution |
|----------|----------|
| Web app users | Passkeys → session cookie → `req.user` |
| Mobile apps | Passkeys → bearer token → `req.user` |
| CI/CD, scripts | API keys → `req.user` (service account) |
| Internal services | `public()` wrapper on all handlers (network-isolated) |

### What You Get For Free

- Schema-based validation and sanitization
- Rate limiting (60/min default)
- Pagination (20 items default, 100 max)
- Auth required by default
- SQL injection protection
- Consistent error format
- Security headers

### What You Configure When Needed

- Custom rate limits
- Custom pagination
- Access control logic
- Complex SQL queries
- Cache headers
- Public route opt-out

### The Design Contract

1. **Explicit is better than implicit** — No hidden magic, just functions and objects
2. **Secure by default** — Opt out of security, don't opt in
3. **Batteries included** — Common needs handled, escape hatches available
4. **Composition over configuration** — Extend with functions, not config files

---

*This document captures design decisions from December 2025. Implementation details may evolve.*
