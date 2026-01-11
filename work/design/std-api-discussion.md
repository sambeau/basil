# std/api Design Discussion

**Status:** Early exploration  
**Date:** December 2025  
**Authors:** Human + AI collaboration

---

## Executive Summary

This document captures the design exploration for a `std/api` module that would provide schema-driven REST API generation for Parsley. The core idea: define a schema once, get validation, sanitization, documentation, and optionally REST endpoints + database binding.

**Key tension identified:** If Basil generates HTML directly from Parsley handlers, and most apps don't expose public APIs, why build special API machinery at all?

---

## The Core Idea

### Schema as Single Source of Truth

```javascript
let api = import("std/api")

let Todo = api.schema("Todo", {
    title: api.string({required: true, min: 1, max: 100}),
    done: api.boolean({default: false})
})
```

The schema defines:
- **Validation rules** (required, min/max, format)
- **Sanitization** (trim, lowercase for emails, etc.)
- **Documentation** (types, examples, descriptions)
- **Database mapping** (column types, indexes)
- **API contract** (request/response shape)

### Type Composition

Types are objects with rich defaults that compose via spread:

```javascript
// api.email internally is:
api.email = {
    type: "string",
    format: "email",
    sanitize: ["trim", "lowercase"],
    validate: fn(v) { valid.email(v) },
    example: "user@example.com",
    description: "Email address"
}

// Use it, override what you need
let field = {...api.email, required: true, description: "Work email"}
```

This achieves the "say email, get everything" ergonomic without method chaining.

---

## The Minimal API

Progressive complexity—start simple, add features as needed:

### Level 0: Schema only (no REST, no DB)

```javascript
let Todo = api.schema("Todo", {
    title: api.string({required: true}),
    done: api.boolean({default: false})
})

// Use manually
let {value, errors} = api.validate(Todo, inputData)
let sanitized = api.sanitize(Todo, inputData)
let docs = api.document(Todo)  // OpenAPI fragment
```

### Level 1: Add database binding

```javascript
let todos = api.bind(Todo, basil.sqlite)

todos.create({title: "New"})
todos.list({where: {done: false}})
todos.update("123", {done: true})
```

### Level 2: Add REST endpoints

```javascript
api.expose(todos, {prefix: "/api/todos"})
// Generates: GET, POST, PUT, DELETE routes
```

### Level 3: Add access control

```javascript
api.expose(todos, {
    prefix: "/api/todos",
    access: "owner"  // Each user sees only their data
})
```

---

## What You Get For Free

| Feature | How |
|---------|-----|
| Validation | Schema types define constraints |
| Sanitization | Schema types define transforms |
| Error messages | Consistent format from schema |
| Documentation | Schema IS the documentation |
| Type coercion | String "true" → boolean true |
| Pagination | Default 20, max 100 |
| Filtering | `?done=false&title_like=foo` |
| Rate limiting | 60/min default |
| Cache headers | Configurable per resource |

---

## Sensible Defaults

### Global defaults (applied unless overridden)

```javascript
api.defaults({
    pagination: {defaultLimit: 20, maxLimit: 100},
    rateLimit: {requests: 60, window: @1m, by: "ip"},
    cache: {maxAge: null, scope: "private"},  // No caching by default (safe)
    access: "public"  // Override for auth-required apps
})
```

### Why 60 requests/minute?

Based on Basil's performance characteristics:
- Tree-walking interpreter: 500-5000 req/sec uncached
- With caching: 10,000-50,000 req/sec
- 60/min = 1/sec average, safe for interpreted routes
- Cached routes can be tuned higher

---

## Objects vs. Chaining

We chose **objects with type functions** over fluent chaining:

| Aspect | Chained | Object + Functions |
|--------|---------|-------------------|
| Readability | Flows like prose | Scannable at a glance |
| Composition | Method inheritance | Object spread |
| Introspection | Hard | Easy—it's just data |
| Performance | Method dispatch overhead | Single merge |
| Parsley fit | Unfamiliar | Matches dictionary patterns |

```javascript
// Instead of:
api.string().required().min(1).max(100)

// We do:
api.string({required: true, min: 1, max: 100})
```

The type functions (`api.string()`, `api.email()`) return objects with rich defaults. You override with options. Composition is object spread.

---

## The Unfair Advantages

Because Basil controls the full stack:

### 1. In-Process SQLite
- No serialization overhead
- No network round-trip
- Instant cache invalidation (same process)

### 2. Write-Through Cache
```javascript
todos.create({...})
// Internally:
// 1. BEGIN TRANSACTION
// 2. INSERT INTO todos ...
// 3. cache.invalidate("/api/todos")  // Same process, ~1μs
// 4. COMMIT
```
No race conditions, no stale cache.

### 3. Schema-Aware Optimization
- Auto-create indexes on marked fields
- Pre-compile common query patterns
- Validate during JSON parse, not after

### 4. Request = Transaction
- No manual transaction management
- Handler boundary IS the transaction boundary
- Automatic rollback on error

### 5. Live Schema Reload (Dev Mode)
- Change schema → auto-migrate in-memory DB
- No restart, no manual migration

### Performance Potential

| Operation | Typical Framework | Basil Potential |
|-----------|-------------------|-----------------|
| Cache hit | 1-5ms | ~100μs |
| Simple query | 5-20ms | ~200μs |
| Insert + cache invalidate | 10-50ms | ~300μs |

25-250x faster for common operations due to process locality.

---

## Open Questions

### 1. Do we need REST at all?

**The tension:** Basil generates HTML directly from Parsley handlers. The main consumer of data is the web app itself, which can just:

```javascript
// In a Parsley handler
let todos = basil.sqlite.query("SELECT * FROM todos WHERE owner_id = ?", [user.id])

<ul>
    {todos.map(fn(t) {
        <li class={t.done ? "done" : ""}>{t.title}</li>
    })}
</ul>
```

No API needed. Data → HTML directly.

**When you DO need an API:**
- Mobile app consuming your data
- Third-party integrations
- SPA/JavaScript-heavy frontend
- Public API for developers

**Question:** Is the API layer a "nice to have" rather than core? Should it be a separate module (`std/rest`?) rather than bundled with schema/binding?

### 2. What about non-bound APIs?

Sometimes you want REST endpoints without database binding:

```javascript
// Proxy to external service
get "/api/weather/:city" {
    let data = http.get("https://weather.api/{city}")
    {result: data}
}

// Computed/aggregated data
get "/api/dashboard/stats" {
    let users = basil.sqlite.query("SELECT COUNT(*) FROM users")[0].count
    let orders = basil.sqlite.query("SELECT SUM(total) FROM orders")[0].sum
    {result: {users, orders}}
}
```

The schema/validation is still useful here, but there's no "resource" to bind.

**Proposal:** Separate concerns:
- `api.schema()` — define shape, get validation/docs
- `api.bind()` — connect schema to data source (optional)
- `api.expose()` — generate REST routes (optional)

Each layer is independent.

### 3. Complex data transformations

Real APIs often need to:
- Join multiple tables
- Aggregate data
- Transform shapes
- Apply business logic

```javascript
// This is hard to express as a "resource"
get "/api/orders/:id/summary" {
    let order = orders.get(id)
    let items = orderItems.list({where: {orderId: id}})
    let customer = customers.get(order.customerId)
    
    {
        result: {
            orderNumber: order.number,
            customer: {name: customer.name, email: customer.email},
            items: items.map(fn(i) { {name: i.name, qty: i.qty, price: i.price} }),
            subtotal: items.reduce(fn(sum, i) { sum + i.price * i.qty }, 0),
            tax: calculateTax(order),
            total: order.total
        }
    }
}
```

**Question:** Does the resource/binding model help here, or get in the way? 

**Possible answer:** Resources for CRUD, custom handlers for complex stuff. The schema validation/docs are still useful even for custom endpoints.

### 4. The HTML-first nature of Basil

Basil's sweet spot is HTML generation:

```javascript
// This is what Basil is FOR
get "/todos" {
    let todos = basil.sqlite.query("SELECT * FROM todos")
    
    <html>
        <body>
            <ul>
                {todos.map(fn(t) { <li>{t.title}</li> })}
            </ul>
        </body>
    </html>
}
```

Adding a REST layer is almost a different product. It's useful, but is it *core*?

**Counter-argument:** Even HTML apps often have AJAX endpoints:
- Form submissions returning JSON
- Infinite scroll loading more items
- Live search/autocomplete
- Dashboard widgets refreshing

So REST-like endpoints are common even in "traditional" web apps.

---

## Possible Architectures

### Option A: Unified (current design)

```
std/api
├── schema()      — define types, validation, docs
├── bind()        — connect to database
└── expose()      — generate REST routes
```

Everything in one module. Simple to discover, but tightly coupled.

### Option B: Separated

```
std/schema        — types, validation, docs
std/data          — database binding, CRUD
std/rest          — REST route generation
```

More modular, but more to learn.

### Option C: Schema core, REST optional

```
std/schema        — types, validation, docs
std/api           — REST generation (imports std/schema)
```

Schema is foundational, REST is built on top.

---

## Comparison to Alternatives

| Framework | Schema Location | REST | Binding | Our Advantage |
|-----------|----------------|------|---------|---------------|
| Strapi | JSON | ✅ | ✅ | Lighter, same language |
| Django RF | Python classes | ✅ | ✅ | Less verbose |
| Prisma | .prisma file | ❌ | ✅ | REST included |
| Rails | Ruby DSL | ✅ | ✅ | Single binary |
| PostgREST | SQL | ✅ | ❌ | Customizable |

Our unique position: **schema = code = validation = docs** in one language, with in-process performance advantages.

---

## What Needs More Thought

1. **Relationship handling** — How do foreign keys, joins, nested resources work?

2. **Migrations** — Dev auto-sync is easy, but production needs proper migrations.

3. **The REST vs HTML tension** — Is REST a distraction from Basil's core value prop?

4. **Complex queries** — When does the binding model break down?

5. **Multiple data sources** — What if you need SQLite + external API?

6. **Caching granularity** — Per-resource? Per-query? Per-user?

7. **Error format** — Should API errors match Parsley's `{value, error}` or use HTTP conventions?

---

## Tentative Recommendation

### Phase 1: Schema Only
Build `std/schema` with:
- Type definitions (`api.string()`, `api.email()`, etc.)
- Validation (`schema.validate(data)`)
- Sanitization (`schema.sanitize(data)`)
- Documentation (`schema.toOpenAPI()`)

This is useful immediately for form validation, input handling, etc.

### Phase 2: Data Binding
Add `api.bind(Schema, source)` with:
- CRUD operations
- Simple where clauses
- Pagination
- Auto table creation (dev mode)

This is useful for HTML apps that want cleaner data access.

### Phase 3: REST Generation
Add `api.expose(resource, options)` with:
- Route generation
- Access control
- Rate limiting
- Cache headers

This is useful for apps that need APIs (mobile, SPAs, integrations).

Each phase delivers value independently. You can use schema without binding, binding without REST.

---

## Next Steps

1. **Decide:** Is REST generation core or optional?
2. **Prototype:** Build `std/schema` types, test with `std/valid`
3. **Evaluate:** How much does binding actually help vs. raw SQL?
4. **Benchmark:** Measure the performance advantages with real queries

---

## Appendix: Example Code at Various Levels

### Just validation (no binding, no REST)

```javascript
let schema = import("std/schema")

let ContactForm = schema.define({
    name: schema.string({required: true, min: 1, max: 100}),
    email: schema.email({required: true}),
    message: schema.string({required: true, min: 10, max: 5000})
})

post "/contact" {
    let {value, errors} = schema.validate(ContactForm, body)
    
    if errors {
        <div class=error>
            {errors.map(fn(e) { <p>{e.field}: {e.message}</p> })}
        </div>
    } else {
        sendEmail(value)
        <div class=success>Thanks for your message!</div>
    }
}
```

### With binding (no REST)

```javascript
let api = import("std/api")

let Todo = api.schema("Todo", {
    title: api.string({required: true}),
    done: api.boolean({default: false})
})

let todos = api.bind(Todo, basil.sqlite)

get "/todos" {
    let items = todos.list({where: {done: false}})
    
    <ul>
        {items.map(fn(t) { <li>{t.title}</li> })}
    </ul>
}

post "/todos" {
    let {value, errors} = api.validate(Todo, body)
    if errors { return <div class=error>Invalid</div> }
    
    todos.create(value)
    redirect("/todos")
}
```

### Full REST API

```javascript
let api = import("std/api")

let Todo = api.schema("Todo", {
    title: api.string({required: true}),
    done: api.boolean({default: false})
})

let todos = api.bind(Todo, basil.sqlite)

api.expose(todos, {
    prefix: "/api/todos",
    access: "authenticated"
})

// That's it. Full CRUD API with auth, validation, pagination, rate limiting.
```

---

*This document captures a design discussion. Nothing here is committed to implementation. The goal is to explore the design space and identify the right scope for a potential std/api module.*
