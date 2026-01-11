# Design Document: FEAT-034 Phases 3-6

**Status:** Draft  
**Date:** December 2025  
**Purpose:** Design decisions and options for completing std/api module

---

## Overview

Phases 1-2 (Schema Types, ID Generation) and Phase 5 Auth Wrappers are complete. This document outlines the design challenges and proposed solutions for the remaining work:

- **Phase 3:** Table Binding (schema.table)
- **Phase 4:** API Routes (module export → HTTP mapping)
- **Phase 5b:** Auth Enforcement (server-side checking)
- **Phase 6:** Sensible Defaults (rate limiting, pagination)

---

## Phase 3: Table Binding

### The Problem

`schema.table(schema, db, tableName)` needs to return an object with methods that:
1. Execute SQL queries using the provided database connection
2. Validate/sanitize data against the schema
3. Generate IDs on insert
4. Return results as Parsley objects

### Design Challenges

#### Challenge 3.1: Method Closures

The table methods (`all()`, `find()`, `insert()`, etc.) need access to:
- The database connection
- The schema definition
- The table name

**Option A: Closure via Environment**
Create each method as a `*Function` with a captured environment containing db, schema, table.

```go
// Each method is a Parsley function that closes over the context
allFn := &Function{
    Parameters: []*ast.Identifier{},
    Body: /* generated AST */,
    Env: extendedEnv, // contains __db__, __schema__, __table__
}
```

*Pros:* Pure Parsley, inspectable  
*Cons:* Complex AST generation, slow

**Option B: Go StdlibBuiltin with Context**
Create a new type `TableQueryMethod` that holds context and implements calling.

```go
type TableQueryMethod struct {
    DB        Object      // Database connection
    Schema    *Dictionary // Schema definition
    TableName string
    Method    string      // "all", "find", etc.
}

func (m *TableQueryMethod) Type() ObjectType { return BUILTIN_OBJ }
func (m *TableQueryMethod) Call(args []Object, env *Environment) Object {
    switch m.Method {
    case "all":
        return m.executeAll(env)
    case "find":
        return m.executeFind(args, env)
    // ...
    }
}
```

*Pros:* Simple, performant, full Go access  
*Cons:* Not inspectable as Parsley code

**Recommendation:** Option B - The methods need direct database access which is Go-side anyway.

#### Challenge 3.2: Database Query Execution

Table methods need to execute SQL. We have existing patterns for this.

**Current DB Query Pattern:**
```parsley
basil.sqlite <=??=> "SELECT * FROM todos"
```

This uses the `<=??=>` operator which is handled specially by the evaluator.

**For Table Binding:**
The `TableQueryMethod` needs to call the same underlying database execution code.

**Option A: Reuse evalSQLOperator**
Extract the SQL execution logic into a shared function callable from Go.

```go
func executeSQLQuery(db Object, query string, params []Object, env *Environment) Object {
    // Shared logic from evalSQLOperator
}
```

**Option B: Build Query String and Call Operator**
Construct an AST node and evaluate it.

*Cons:* Roundabout, fragile

**Recommendation:** Option A - Refactor `evalSQLOperator` to expose reusable functions.

#### Challenge 3.3: Schema Validation in Methods

`insert()` and `update()` must validate data against the schema.

**Current State:** `schema.validate()` exists and works.

**Implementation:**
```go
func (m *TableQueryMethod) executeInsert(args []Object, env *Environment) Object {
    data := args[0].(*Dictionary)
    
    // Call schema.validate
    result := schemaValidate([]Object{m.Schema, data}...)
    if validResult, ok := result.(*Dictionary); ok {
        // Check if validation passed
        validExpr := validResult.Pairs["valid"]
        validObj := Eval(validExpr, validResult.Env)
        if b, ok := validObj.(*Boolean); ok && !b.Value {
            // Return validation errors
            return validResult
        }
    }
    
    // Proceed with insert...
}
```

#### Challenge 3.4: ID Generation on Insert

If `id` field is not provided, generate one using the schema's ID type.

**Implementation:**
```go
func (m *TableQueryMethod) executeInsert(args []Object, env *Environment) Object {
    data := args[0].(*Dictionary)
    
    // Check if ID provided
    if _, hasID := data.Pairs["id"]; !hasID {
        // Get ID type from schema
        idType := getSchemaFieldType(m.Schema, "id")
        newID := generateID(idType) // Uses std/id functions
        data.Pairs["id"] = objectToExpression(newID)
    }
    // ...
}
```

### Proposed Implementation

```go
// New file: pkg/parsley/evaluator/stdlib_table_binding.go

type TableBinding struct {
    DB        Object
    Schema    *Dictionary
    TableName string
}

func (tb *TableBinding) Type() ObjectType { return "TABLE_BINDING" }
func (tb *TableBinding) Inspect() string { return "TableBinding(" + tb.TableName + ")" }

// Make it work with dot notation
func (tb *TableBinding) GetField(name string) Object {
    switch name {
    case "all":
        return &TableQueryMethod{tb, "all"}
    case "find":
        return &TableQueryMethod{tb, "find"}
    // ...
    }
}

type TableQueryMethod struct {
    Binding *TableBinding
    Method  string
}

func (m *TableQueryMethod) Type() ObjectType { return BUILTIN_OBJ }
func (m *TableQueryMethod) Inspect() string { return m.Method }

// Called when method is invoked
func (m *TableQueryMethod) Call(args []Object, env *Environment) Object {
    switch m.Method {
    case "all":
        return m.Binding.executeAll(env)
    case "find":
        return m.Binding.executeFind(args[0], env)
    case "where":
        return m.Binding.executeWhere(args[0], env)
    case "insert":
        return m.Binding.executeInsert(args[0], env)
    case "update":
        return m.Binding.executeUpdate(args[0], args[1], env)
    case "delete":
        return m.Binding.executeDelete(args[0], env)
    default:
        return &Error{Message: "Unknown method: " + m.Method}
    }
}
```

### Open Questions

1. **Should table binding work with non-SQLite databases?**
   - Currently only SQLite is supported
   - Design should be extensible but MVP can be SQLite-only

2. **How to handle transactions?**
   - Could add `Todos.transaction(fn(tx) {...})`
   - Defer to backlog for MVP

3. **Should `where()` support operators beyond `=`?**
   - `where({age: {$gt: 18}})` MongoDB-style?
   - For MVP, equality only. Complex queries use raw SQL.

### Recommended approach and MVP scope

- Use Go-side `TableBinding` + `TableQueryMethod` (StdlibBuiltin) to hold `db`, `schema`, `table` context; no AST generation.
- Refactor SQL execution from `<=??=>` into a reusable helper shared by table methods; keep SQLite-only for MVP but design helper signature to accept driver type.
- Validation/sanitization: call `schema.validate` before `insert`/`update`; on failure return the validation result unchanged (errors array, no side effects).
- ID handling: if `id` missing, generate via schema id format using `std/id`; forbid overwrite on `insert`, allow on `update` only when schema allows (default deny).
- Query surface for MVP: `all(limit/offset)`, `find(id)`, `where(dict equality only)`, `insert(data)`, `update(id, data)`, `delete(id)`; parameterize everything to avoid SQL injection.
- Pagination: defaults `limit=20`, `max=100`, `offset=0`; allow `paginate: false` or `limit: 0` to disable caps when explicitly requested.
- Tests: table-binding unit tests covering happy paths, validation failures, id autogen, pagination, SQL injection guard (malicious values stay parameterized), and update/delete affected rows.

---

## Phase 4: API Routes

### The Problem

When a Parsley module is loaded as an API endpoint, its exports should map to HTTP methods:
- `export get` → GET /path
- `export post` → POST /path
- `export getById` → GET /path/:id

### Design Challenges

#### Challenge 4.1: Detecting API Modules

How does Basil know a module is an API module vs a regular page handler?

**Option A: Path Convention**
Files under `/api/` are automatically API modules.

```yaml
# basil.yaml
routes:
  - path: /api/
    type: api  # implicit for /api/ prefix
```

*Pros:* Simple, conventional  
*Cons:* Less flexible

**Option B: Explicit Configuration**
Declare API routes in config.

```yaml
routes:
  - path: /api/todos
    module: ./handlers/todos.pars
    type: api
  - path: /api/users
    module: ./handlers/users.pars
    type: api
```

*Pros:* Explicit, flexible  
*Cons:* More verbose

**Option C: Module Self-Declaration**
Module exports a marker.

```parsley
export __api__ = true
export get = fn(req) { ... }
```

*Pros:* Self-contained  
*Cons:* Easy to forget, non-obvious

**Recommendation:** Option A + B hybrid. `/api/` prefix is automatic, but explicit `type: api` available for other paths.

#### Challenge 4.2: Export-to-Route Mapping

**Current Handler Loading:**
Basil loads a `.pars` file, evaluates it, and if the result is a function, calls it with the request.

**New API Handler Loading:**
1. Load and evaluate the module
2. If result has API exports (`get`, `post`, `getById`, etc.), treat as API
3. Match incoming HTTP method + path pattern to export
4. Call the matching export function

```go
// server/api.go

func (h *Handler) handleAPIRequest(w http.ResponseWriter, r *http.Request, module Object) {
    // module is the evaluated .pars file (usually a Dictionary or StdlibModuleDict)
    
    method := r.Method
    hasID := extractIDFromPath(r.URL.Path) != ""
    
    var exportName string
    switch {
    case method == "GET" && !hasID:
        exportName = "get"
    case method == "GET" && hasID:
        exportName = "getById"
    case method == "POST":
        exportName = "post"
    case method == "PUT":
        exportName = "put"
    case method == "PATCH":
        exportName = "patch"
    case method == "DELETE":
        exportName = "delete"
    }
    
    handler := getModuleExport(module, exportName)
    if handler == nil {
        http.Error(w, "Method not allowed", 405)
        return
    }
    
    // Build request object with params
    req := buildAPIRequest(r, hasID)
    
    // Call handler
    result := applyFunctionWithEnv(handler, []Object{req}, env)
    
    // Serialize response
    writeAPIResponse(w, result)
}
```

#### Challenge 4.3: Path Parameter Extraction

For routes like `/api/todos/:id`, extract `id` into `req.params.id`.

**Implementation:**
```go
func extractIDFromPath(path string) string {
    // /api/todos/abc123 → "abc123"
    parts := strings.Split(strings.Trim(path, "/"), "/")
    if len(parts) >= 3 {
        return parts[len(parts)-1]
    }
    return ""
}

func buildAPIRequest(r *http.Request, hasID bool) *Dictionary {
    req := buildBaseRequest(r) // existing request building
    
    if hasID {
        params := &Dictionary{Pairs: map[string]ast.Expression{
            "id": objectToExpression(&String{Value: extractIDFromPath(r.URL.Path)}),
        }}
        req.Pairs["params"] = objectToExpression(params)
    }
    
    return req
}
```

#### Challenge 4.4: Nested Routes via `routes` Export

The `routes` export allows composing multiple API modules:

```parsley
export routes = {
    "/todos": import(@./todos.pars),
    "/users": import(@./users.pars)
}
```

**Implementation:**
```go
func (h *Handler) handleAPIRequest(w http.ResponseWriter, r *http.Request, module Object) {
    // Check for routes export first
    if routes := getModuleExport(module, "routes"); routes != nil {
        // Find matching sub-route
        subPath := extractSubPath(r.URL.Path)
        if subModule := matchRoute(routes, subPath); subModule != nil {
            h.handleAPIRequest(w, r, subModule)
            return
        }
    }
    
    // Otherwise handle as direct API module
    // ...
}
```

#### Challenge 4.5: Response Serialization

| Handler Return | HTTP Response |
|----------------|---------------|
| Dictionary | JSON object |
| Array | JSON array |
| String (looks like HTML) | HTML |
| String (other) | Plain text |
| APIError | JSON error + status code |
| `{status: N, body: X}` | Custom status + body |

```go
func writeAPIResponse(w http.ResponseWriter, result Object) {
    switch r := result.(type) {
    case *APIError:
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(r.Status)
        json.NewEncoder(w).Encode(r.ToDict())
        
    case *Dictionary:
        // Check for custom status/body
        if statusExpr, ok := r.Pairs["status"]; ok {
            status := evalToInt(statusExpr)
            body := r.Pairs["body"]
            w.WriteHeader(status)
            writeJSONBody(w, body)
        } else {
            w.Header().Set("Content-Type", "application/json")
            writeJSONBody(w, r)
        }
        
    case *Array:
        w.Header().Set("Content-Type", "application/json")
        writeJSONBody(w, r)
        
    case *String:
        if looksLikeHTML(r.Value) {
            w.Header().Set("Content-Type", "text/html")
        } else {
            w.Header().Set("Content-Type", "text/plain")
        }
        w.Write([]byte(r.Value))
    }
}
```

### Open Questions

1. **How to handle file uploads in API routes?**
   - Existing `req.files` should work
   - May need size limits in API config

2. **Should API routes support middleware?**
   - For MVP, auth wrappers cover the main use case
   - Generic middleware deferred to backlog

3. **CORS handling?**
   - Should be configurable in basil.yaml
   - Default: same-origin only for security

### Recommended routing strategy

- Detect API handlers by `/api/` prefix OR explicit `type: api` in config; allow opt-in for non-prefix routes.
- Export mapping: `get/post/put/patch/delete/getById` → HTTP verbs; `routes` export enables nested routing (e.g., `/api` dispatches to `/todos`, `/users`).
- Request building: set `req.params.id` from trailing path segment when `:id` present; keep `req.query`, `req.body`, `req.headers`, `req.files` consistent with current handler shape.
- Response serialization: dictionaries/arrays → JSON with `Content-Type: application/json`; strings detected as HTML → text/html; otherwise text/plain; `{status, body, headers}` respected; `APIError` mapped to HTTP status + JSON body.
- Method fallback: if export missing, return 405 with `Allow` header populated from available exports.
- Tests: route detection (prefix + explicit), export mapping, params extraction, nested `routes`, content-type selection, `{status, body}` handling, and 405 behavior.

---

## Phase 5b: Auth Enforcement

### The Problem

Auth wrappers (`public()`, `adminOnly()`, etc.) set metadata on functions. The server needs to:
1. Read the auth metadata before calling the handler
2. Enforce authentication/authorization
3. Return appropriate HTTP errors

### Design Challenges

#### Challenge 5b.1: Reading Auth Metadata

`AuthWrappedFunction` has `GetAuthMetadata()` method. Need to check this before calling.

```go
func (h *Handler) handleAPIRequest(w http.ResponseWriter, r *http.Request, ...) {
    handler := getModuleExport(module, exportName)
    
    // Check auth metadata
    authMeta := getAuthMetadata(handler)
    
    if authMeta.AuthType != "public" {
        // Require authentication
        user, err := h.authenticateRequest(r)
        if err != nil {
            writeAPIResponse(w, &APIError{Status: 401, ...})
            return
        }
        
        // Check roles if needed
        if authMeta.AuthType == "admin" {
            if user.Role != "admin" {
                writeAPIResponse(w, &APIError{Status: 403, ...})
                return
            }
        } else if authMeta.AuthType == "roles" {
            if !hasAnyRole(user, authMeta.Roles) {
                writeAPIResponse(w, &APIError{Status: 403, ...})
                return
            }
        }
        
        // Add user to request
        req.Pairs["user"] = objectToExpression(userToDict(user))
    }
    
    // Call handler
    result := applyFunctionWithEnv(handler, []Object{req}, env)
    // ...
}

func getAuthMetadata(handler Object) *AuthMetadata {
    if wrapped, ok := handler.(*AuthWrappedFunction); ok {
        return &AuthMetadata{
            AuthType: wrapped.AuthType,
            Roles:    wrapped.Roles,
        }
    }
    // Default: require auth
    return &AuthMetadata{AuthType: "auth"}
}
```

#### Challenge 5b.2: Role Resolution

How to get user's role for authorization checks?

**Option A: Column Convention**
Look up role from users table based on configured column.

```yaml
# basil.yaml
api:
  auth:
    roleColumn: role  # default
```

```go
func (h *Handler) getUserRole(userID string) string {
    row := h.db.QueryRow("SELECT " + h.config.RoleColumn + " FROM users WHERE id = ?", userID)
    var role string
    row.Scan(&role)
    return role
}
```

*Pros:* Simple, works for most cases  
*Cons:* Inflexible for complex role systems

**Option B: Role in Session**
Store role in session when user logs in.

```go
func (h *Handler) getUserFromSession(r *http.Request) *User {
    session := h.getSession(r)
    return &User{
        ID:   session.UserID,
        Role: session.Role, // Cached at login
    }
}
```

*Pros:* Fast, no DB lookup  
*Cons:* Role changes require re-login

**Option C: Custom Resolver Function**
Allow Parsley function for complex role resolution.

```yaml
# basil.yaml
api:
  auth:
    roleResolver: ./auth/roles.pars
```

```parsley
// auth/roles.pars
export resolve = fn(user) {
    let permissions = basil.sqlite <=??=> "
        SELECT p.name FROM permissions p
        JOIN user_permissions up ON up.permission_id = p.id
        WHERE up.user_id = ?
    ", [user.id]
    
    permissions.map(fn(p) { p.name })
}
```

*Pros:* Flexible  
*Cons:* Complex, performance concerns

**Recommendation:** Option B (session-cached role) for MVP, with Option A as fallback. Option C deferred.

### Open Questions

1. **API key authentication?**
   - Different from session auth
   - Could support `Authorization: Bearer <key>` header
   - Defer to backlog for MVP

2. **How to invalidate cached roles?**
   - Admin changes user role → user's session still has old role
   - Could add role version to session, check on each request
   - Accept this limitation for MVP

### Recommended auth enforcement

- Default stance: require auth unless wrapped with `public()`. If function is `AuthWrappedFunction`, read metadata to decide `public`, `auth`, `admin`, or `roles` enforcement.
- Identity: reuse existing session-based auth; attach `req.user` (id, email, role) when authenticated; if missing and not public → 401 `APIError`.
- Roles: prefer session-cached role; fallback to configurable role column lookup (`api.auth.roleColumn`, default `role`). Custom resolver deferred.
- Errors: use `APIError` for 401/403; do not panic/abort evaluator. Wrong role → 403 with code `HTTP-403`.
- Tests: public route works unauthenticated, auth-required route returns 401 when absent, admin-only rejects non-admin, roles wrapper enforces allowed set, and `req.user` is present when authenticated.

---

## Phase 6: Sensible Defaults

### Challenge 6.1: Rate Limiting

**Requirements:**
- Default: 60 requests per minute per IP
- For authenticated users: per user ID
- Per-route override via module export
- Return 429 when exceeded

**Implementation Options:**

**Option A: In-Memory Token Bucket**
Simple, no dependencies, but doesn't scale across instances.

```go
type RateLimiter struct {
    buckets sync.Map // IP/UserID → *tokenBucket
}

type tokenBucket struct {
    tokens    int
    lastRefill time.Time
    mu        sync.Mutex
}

func (rl *RateLimiter) Allow(key string, limit int, window time.Duration) bool {
    bucket := rl.getOrCreate(key)
    bucket.mu.Lock()
    defer bucket.mu.Unlock()
    
    // Refill tokens
    elapsed := time.Since(bucket.lastRefill)
    refill := int(elapsed / window * time.Duration(limit))
    bucket.tokens = min(limit, bucket.tokens + refill)
    bucket.lastRefill = time.Now()
    
    if bucket.tokens > 0 {
        bucket.tokens--
        return true
    }
    return false
}
```

**Option B: SQLite-Based**
Scales across restarts, simple persistence.

```sql
CREATE TABLE rate_limits (
    key TEXT PRIMARY KEY,
    tokens INTEGER,
    last_refill INTEGER
);
```

*Cons:* DB overhead per request

**Recommendation:** Option A for MVP. Single-instance is fine for target use case.

**Per-Route Override:**
```parsley
// api/heavy-endpoint.pars
export rateLimit = {requests: 10, window: @1m}

export get = fn(req) { ... }
```

```go
func getRateLimit(module Object) (int, time.Duration) {
    if rl := getModuleExport(module, "rateLimit"); rl != nil {
        // Parse requests and window
        return parseRateLimit(rl)
    }
    return 60, time.Minute // default
}
```

### Challenge 6.2: Pagination

**Requirements:**
- Default limit: 20 items
- Max limit: 100 items
- Accept `?limit=N&offset=M` query params
- Apply to `all()` and `where()` methods

**Implementation:**

```go
func (tb *TableBinding) executeAll(env *Environment) Object {
    // Get pagination from request context
    limit, offset := getPagination(env)
    
    query := fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?", tb.TableName)
    return executeQuery(tb.DB, query, []any{limit, offset}, env)
}

func getPagination(env *Environment) (int, int) {
    req := env.Get("req")
    query := getQueryParams(req)
    
    limit := 20 // default
    if l, ok := query["limit"]; ok {
        limit = min(100, max(1, parseInt(l))) // clamp 1-100
    }
    
    offset := 0
    if o, ok := query["offset"]; ok {
        offset = max(0, parseInt(o))
    }
    
    return limit, offset
}
```

**Open Question:** Should pagination be opt-in or opt-out?
- Opt-in: `Todos.all({paginate: true})` 
- Opt-out: Always paginated, `Todos.all({paginate: false})` for full list

**Recommendation:** Always paginate by default. Explicit `limit: 0` or `paginate: false` to disable.

### Challenge 6.3: Error Response Format

Ensure consistent error format across all API responses.

**Standard Format:**
```json
{
  "error": {
    "code": "HTTP-404",
    "message": "Todo not found",
    "field": "id"  // optional, for validation errors
  }
}
```

**Implementation:** Already done in Phase 5 with `APIError.ToDict()`.

### Recommended defaults

- Rate limiting: in-memory token bucket keyed by user id when authenticated else IP; default 60 req/min; per-module override via `export rateLimit = {requests, window}`; return 429 with JSON error.
- Pagination: defaults `limit=20`, `max=100`, `offset=0`; apply in table `all/where`; allow override via query params; `paginate: false` or `limit: 0` to disable caps intentionally.
- Error envelope: always `APIError.ToDict()` shape; for validation failures from table binding, return `{errors: [...]}` with HTTP 400; for missing resources, use `api.notFound` → HTTP 404.
- Telemetry (optional, deferred if time): log request path, status, duration for API routes; minimal stdout logging acceptable for MVP.

---

## Implementation Order

Recommended order based on dependencies:

1. **Phase 4.1:** Basic API route detection and export mapping
   - Allows testing API handlers manually

2. **Phase 5b:** Auth enforcement
   - Critical for security, should come early

3. **Phase 4.2:** JSON response serialization
   - Makes API handlers useful

4. **Phase 3:** Table binding
   - Can be tested via API routes

5. **Phase 6.1:** Rate limiting
   - Security feature, but not blocking

6. **Phase 6.2:** Pagination
   - Enhancement, can come last

---

## Summary of Decisions Needed

| Area | Decision | Options | Recommendation |
|------|----------|---------|----------------|
| Table methods | Closure style | Go type vs Parsley function | Go type (Option B) |
| API detection | Path convention | /api/ prefix vs explicit | Hybrid |
| Role resolution | Where to get role | DB lookup vs session cache | Session cache |
| Rate limiting | Storage | In-memory vs SQLite | In-memory |
| Pagination | Default behavior | Opt-in vs always-on | Always-on |

---

## Risks and Mitigations

1. **Complexity creep**
   - Risk: Too many features, hard to maintain
   - Mitigation: Strict MVP scope, defer non-essential features

2. **Performance concerns with table binding**
   - Risk: Slow queries, N+1 problems
   - Mitigation: Encourage raw SQL for complex queries, add logging

3. **Security gaps in auth enforcement**
   - Risk: Bypasses, role escalation
   - Mitigation: Auth required by default, extensive testing

4. **Breaking changes to existing handler behavior**
   - Risk: API route handling breaks HTML routes
   - Mitigation: Clear separation, `/api/` prefix convention
