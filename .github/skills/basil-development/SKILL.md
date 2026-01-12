---
name: basil-development
description: Develop and test Basil web applications with HTTP handlers, routing, sessions, authentication, databases, and interactive components (Parts). Use when building Basil web apps, configuring basil.yaml, testing handlers with curl, working with @basil/http or @basil/auth context, or debugging routing and Parts.
---

# Basil Development

## What is Basil?

**Basil** is a web framework for **Parsley** (the language):
- **Parsley** = the language (like JavaScript)
- **Basil** = the web framework (like Express/Rails)

Basil provides HTTP server, routing, sessions, auth, database, interactive components (Parts), and asset bundling.

## Quick Start

```bash
# Initialize project
./basil --init myapp && cd myapp

# Run dev server (auto-reload, detailed errors, dev tools at /__dev/log)
./basil --dev

# Test
curl http://localhost:8080/
```

## Core Concepts

### 1. Handlers
Parsley files that handle HTTP requests.

```parsley
// site/index.pars
<html>
<body>
  <h1>"Welcome!"</h1>
  <p>"Search: {@params.q ?? 'none'}"</p>
</body>
</html>
```

### 2. Configuration (basil.yaml)

```yaml
server:
  port: 8080
site: ./site           # Filesystem routing: site/about.pars → /about
sqlite: ./data.db      # Database
session:
  secret: "your-32-char-secret"  # Required for sessions
```

### 3. Parts (Interactive Components)
`.part` files with functions that return HTML. Update without page reload.

```parsley
// parts/counter.part
export default = fn(props) {
  let count = props.count ?? 0
  <div>
    <p>"Count: {count}"</p>
    <button part-click="increment" part-count={count + 1}>"++"</button>
  </div>
}

export increment = fn(props) {
  default(props)  // Re-render with new count
}
```

**Usage:**
```parsley
<Part src={@~/parts/counter.part} count={0}/>
```

**Important**: Parts need routes in basil.yaml:
```yaml
routes:
  - path: /parts/counter.part
    handler: ./parts/counter.part
```

## Essential Imports

### HTTP Context (@basil/http)

```parsley
let {request, response, method} = import @basil/http

@params         // URL/form params: {id: "123"}
method          // "GET", "POST", etc.
request.form    // POST form data
request.path    // "/api/users/123"
response.status = 404
response.redirect("/dashboard")
```

### Database & Auth (@basil/auth)

```parsley
let {db, session, user} = import @basil/auth

// Database (see docs/basil/reference.md for complete API)
let user = db <=?=> "SELECT * FROM users WHERE id = ?" [123]  // One row
let users = db <=??=> "SELECT * FROM users"                    // All rows
let result = db <=!=> "INSERT INTO users (name) VALUES (?)" ["Alice"]  // Mutation

// Sessions
session.set("cart", ["item1", "item2"])
let cart = session.get("cart", [])
session.flash("success", "Saved!")  // Show-once message

// Current user (null if not logged in)
user.email
user.role  // "admin", "user", etc.
```

### Magic Variables

```parsley
@params         // URL/form parameters
@now            // Current datetime
@env.HOME       // Environment variables
```

## Common Handler Patterns

### Database Query Handler

```parsley
let {db} = import @basil/auth

let users = db <=??=> "SELECT * FROM users ORDER BY name"

<html>
<body>
  <ul>
    for (user in users) {
      <li>"{user.name}"</li>
    }
  </ul>
</body>
</html>
```

### JSON API Handler

```parsley
let {method} = import @basil/http
let {db} = import @basil/auth

if (method == "GET") {
  let users = db <=??=> "SELECT id, name FROM users"
  {users: users}
} else if (method == "POST") {
  let result = db <=!=> "INSERT INTO users (name) VALUES (?)" [@params.name]
  {id: result.lastId}
}
```

### Form Handler

```parsley
let {method, request} = import @basil/http
let {session} = import @basil/auth
let {redirect} = import @std/api

if (method == "POST") {
  let form = request.form
  // Process form...
  session.flash("success", "Saved!")
  redirect("/")
} else {
  <form method="POST">
    <input name="email" required/>
    <button>"Submit"</button>
  </form>
}
```

## Testing with curl

```bash
# GET
curl http://localhost:8080/

# With query params
curl "http://localhost:8080/search?q=test"

# POST form
curl -X POST http://localhost:8080/contact -d "name=Alice&email=alice@example.com"

# JSON API
curl http://localhost:8080/api/users -H "Content-Type: application/json" -d '{"name":"Alice"}'

# With cookies
curl -c cookies.txt http://localhost:8080/login -d "user=admin&pass=secret"
curl -b cookies.txt http://localhost:8080/dashboard
```

## Common Pitfalls

1. **Parts need routes** - Add to basil.yaml `routes:`
2. **Parts export functions only** - No variables in .part files
3. **Session secret required** - Set in basil.yaml for persistent sessions
4. **Database path relative to config** - Not to handler file
5. **File writes need whitelist** - Use `security.allow_write` in config

## Quick Reference

### Commands

```bash
./basil --init myapp        # Create project
./basil --dev               # Run dev server
./basil --dev --port 3000   # Custom port
```

### Config Essentials

```yaml
server:
  port: 8080
site: ./site              # Filesystem routing
sqlite: ./data.db         # Database
session:
  secret: "32-char-secret"
auth:
  enabled: true
  protected_paths: ["/admin"]
```

### Import Essentials

```parsley
let {method, request, response} = import @basil/http
let {db, session, user} = import @basil/auth
let {redirect, notFound} = import @std/api
```

## Detailed Documentation

For comprehensive guides, see:

- **[references/CONFIGURATION.md](references/CONFIGURATION.md)** - Complete basil.yaml reference
- **[references/PARTS.md](references/PARTS.md)** - Interactive components guide
- **[references/DATABASE.md](references/DATABASE.md)** - Database operations
- **[references/TESTING.md](references/TESTING.md)** - Testing strategies
- **Basil API**: `docs/basil/reference.md` - Full language reference with all operators and builtins
- **Parsley**: `docs/parsley/CHEATSHEET.md` and `docs/parsley/reference.md`

## Best Practices

1. Use `./basil --dev` for auto-reload and detailed errors
2. Test with curl before browser testing
3. Check `http://localhost:8080/__dev/log` for debugging
4. Validate Parsley with `./pars` before using in handlers
5. Use `{data, error}` pattern for error handling
6. Set session secret for persistent sessions
7. Whitelist directories for file operations
