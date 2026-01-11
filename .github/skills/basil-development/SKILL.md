---
name: basil-development
description: How to develop, configure, and test Basil web applications.
created: 2026-01-11
---

# Basil Development

This skill helps you develop and test web applications using the Basil framework.

## When to use this skill

Use this skill when you need to:
- Set up a new Basil web project
- Configure Basil server (`basil.yaml`)
- Create and test HTTP handlers in Parsley
- Work with sessions, authentication, and databases
- Test handlers with curl during development
- Debug routing, Parts, or server behavior
- Understand Basil-specific features (@basil/http, @basil/auth, etc.)

## What is Basil?

**Basil** is a web framework built on **Parsley** (the language). Think of it as:
- **Parsley** = the language (like JavaScript)
- **Basil** = the web framework (like Express/Rails)

Basil provides:
- HTTP server with routing
- Session management and authentication
- SQLite database integration
- Interactive components (Parts)
- Asset bundling (CSS/JS)
- Development tools and live reload

## Quick Start: Running Basil

### 1. Create a Basic Project

```bash
# Initialize a new project
./basil --init myapp
cd myapp

# Project structure created:
# myapp/
# ├── .gitignore        # Git ignore patterns
# ├── basil.yaml        # Configuration
# ├── site/
# │   └── index.pars    # Homepage
# ├── public/           # Static files
# ├── db/               # SQLite databases
# └── logs/             # Log files
```

### 2. Run in Development Mode

```bash
# Start the server
./basil --dev

# Or with explicit config
./basil --dev --config basil.yaml

# Server starts at http://localhost:8080
```

**Dev mode features:**
- HTTP (not HTTPS) for localhost
- Script caching disabled (edit and refresh)
- Auto-reload on file changes
- Dev tools at `/__dev/log`
- Detailed error messages

### 3. Test with curl

```bash
# Test homepage
curl http://localhost:8080/

# Test with query parameters
curl "http://localhost:8080/search?q=test"

# Test POST with form data
curl -X POST http://localhost:8080/submit \
  -d "name=Alice&email=alice@example.com"

# Test JSON API
curl http://localhost:8080/api/users \
  -H "Content-Type: application/json"

# Test with cookies
curl http://localhost:8080/profile \
  -H "Cookie: session=abc123"
```

## Configuration (basil.yaml)

### Basic Configuration

```yaml
server:
  host: localhost
  port: 8080

# Choose ONE routing strategy:

# Option 1: Filesystem routing (site mode)
site: ./site              # Files in site/ serve at their path
                          # site/about.pars → /about

# Option 2: Explicit routes
routes:
  - path: /
    handler: ./handlers/index.pars
  - path: /api/*
    handler: ./handlers/api.pars
  - path: /users/:id
    handler: ./handlers/user.pars

# Static files
public_dir: ./public      # Serves at web root: public/logo.png → /logo.png

# Database
sqlite: ./data.db         # SQLite database path

# Logging
logging:
  level: info             # debug, info, warn, error
  format: text            # text or json
```

### Session & Authentication

```yaml
session:
  secret: "your-32-char-secret-key-here"  # Required for persistent sessions
  max_age: 24h                             # Session expiry

auth:
  enabled: true
  protected_paths:
    - /dashboard          # Require login for these paths
    - /admin
```

### Security & File Access

```yaml
security:
  allow_write:
    - ./data              # Whitelist directories for file writes
    - ./uploads
```

## Writing Handlers

### Basic Handler (HTML)

```parsley
// handlers/index.pars
let {query} = import @basil/http

<html>
<head>
  <title>"My Basil App"</title>
</head>
<body>
  <h1>"Welcome!"</h1>
  <p>"Query: {query}"</p>
</body>
</html>
```

### Handler with Database

```parsley
// handlers/users.pars
let {db} = import @basil/auth

// Query users from database
let users = db <=??=> "SELECT * FROM users ORDER BY name"

<html>
<body>
  <h1>"Users"</h1>
  <ul>
    for (user in users) {
      <li>"{user.name} ({user.email})"</li>
    }
  </ul>
</body>
</html>
```

### API Handler (JSON)

```parsley
// handlers/api/users.pars
let {method, query} = import @basil/http
let {db} = import @basil/auth

if (method == "GET") {
  // Return all users as JSON
  let users = db <=??=> "SELECT id, name, email FROM users"
  {users: users}
} else if (method == "POST") {
  // Create new user
  let result = db <=!=> "INSERT INTO users (name, email) VALUES (?, ?)" 
    [query.name, query.email]
  {id: result.lastId, success: true}
}
```

### Handler with Form Submission

```parsley
// handlers/contact.pars
let {method, request, response} = import @basil/http
let {session} = import @basil/auth

if (method == "POST") {
  let form = request.form
  // Process form data...
  session.flash("success", "Message sent!")
  response.redirect("/")
} else {
  <html>
  <body>
    <form method="POST">
      <input name="email" type="email" required/>
      <textarea name="message" required/>
      <button type="submit">"Send"</button>
    </form>
  </body>
  </html>
}
```

## Basil Context (@basil/*)

### HTTP Context (@basil/http)

```parsley
let {request, response, query, route, method} = import @basil/http

// Shortcuts
query                    // URL query params: {id: "123", q: "search"}
route                    // Matched subpath: "users/123" from /api/*
method                   // HTTP method: "GET", "POST", "PUT", "DELETE"

// Request details
request.path             // Full path: "/api/users/123"
request.form             // POST form data: {name: "Alice", email: "..."}
request.cookies          // Cookies: {theme: "dark", session: "..."}
request.headers          // HTTP headers
request.query            // Same as query shortcut

// Response control
response.status = 404
response.headers["X-Custom"] = "value"
response.cookies.theme = "dark"
response.redirect("/dashboard")
```

### Auth & Database (@basil/auth)

```parsley
let {db, session, auth, user} = import @basil/auth

// Database queries
let user = db <=?=> "SELECT * FROM users WHERE id = ?" [123]       // One row
let users = db <=??=> "SELECT * FROM users"                         // All rows
let result = db <=!=> "INSERT INTO users (name) VALUES (?)" ["Alice"]  // Mutation

// Sessions
session.set("cart", ["item1", "item2"])
let cart = session.get("cart", [])      // With default
session.has("user_id")                   // Check key
session.delete("user_id")                // Remove key
session.clear()                          // Clear all

// Flash messages (show once)
session.flash("success", "Profile updated!")
let msg = session.getFlash("success")    // Returns and removes

// Authentication
user                     // Current logged-in user (or null)
auth.isLoggedIn          // Boolean
user.email               // User properties
user.role                // "admin", "user", etc.
```

### Magic Variables

```parsley
// Available in Basil handlers
@params                  // URL/form parameters: @params.id, @params.q
@now                     // Current datetime
@env.HOME                // Environment variables
```

## Parts (Interactive Components)

Parts are server-rendered HTML fragments that update without page reloads.

### Creating a Part (.part file)

```parsley
// parts/counter.part
// IMPORTANT: .part files can ONLY export functions, not variables

export default = fn(props) {
  let count = props.count ?? 0
  <div>
    <p>"Count: {count}"</p>
    <button part-click="increment" part-count={count + 1}>"Increment"</button>
    <button part-click="decrement" part-count={count - 1}>"Decrement"</button>
  </div>
}

export increment = fn(props) {
  let count = props.count
  <div>
    <p>"Count: {count}"</p>
    <button part-click="increment" part-count={count + 1}>"Increment"</button>
    <button part-click="decrement" part-count={count - 1}>"Decrement"</button>
  </div>
}

export decrement = fn(props) {
  // Same as increment but for decrement button
  default(props)
}
```

### Using Parts in Handlers

```parsley
// handlers/index.pars
<html>
<body>
  <h1>"Interactive Counter"</h1>
  
  <!-- Basic usage -->
  <Part src={@~/parts/counter.part} count={0}/>
  
  <!-- With ID for targeting -->
  <Part src={@~/parts/results.part} id="search-results"/>
  
  <!-- Auto-refresh every second -->
  <Part src={@~/parts/clock.part} part-refresh={1000}/>
</body>
</html>
```

### Part Routes in basil.yaml

```yaml
routes:
  - path: /parts/counter.part
    handler: ./parts/counter.part
    # Parts need routes to be accessible via HTTP
```

## Testing Handlers

### Test with curl

```bash
# Test HTML handler
curl http://localhost:8080/

# Test with query parameters
curl "http://localhost:8080/users?role=admin"

# Test POST form
curl -X POST http://localhost:8080/contact \
  -d "name=Alice&email=alice@example.com"

# Test JSON API
curl http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}'

# Test with session cookie
curl http://localhost:8080/dashboard \
  -b "session=your-session-cookie"

# Save cookies from response
curl -c cookies.txt http://localhost:8080/login \
  -d "username=admin&password=secret"

# Use saved cookies
curl -b cookies.txt http://localhost:8080/dashboard
```

### Test in Browser

1. Start dev server: `./basil --dev`
2. Open: http://localhost:8080
3. Check dev tools: http://localhost:8080/__dev/log
4. Use browser DevTools to inspect requests/responses

### Common Development Patterns

```bash
# Run with auto-reload (watches for file changes)
./basil --dev

# Run on different port
./basil --dev --port 3000

# Use specific config
./basil --dev --config myconfig.yaml

# Run with verbose logging
./basil --dev  # Check logs in terminal
```

## Database Operations

### Setting up Database

```yaml
# basil.yaml
sqlite: ./myapp.db
```

### Basic Queries

```parsley
let {db} = import @basil/auth

// Query one row (returns dict or null)
let user = db <=?=> "SELECT * FROM users WHERE id = ?" [userId]

// Query multiple rows (returns array)
let users = db <=??=> "SELECT * FROM users WHERE active = ?" [true]

// Insert/Update/Delete (returns {affected, lastId})
let result = db <=!=> "INSERT INTO users (name, email) VALUES (?, ?)" 
  [name, email]
log("Created user with ID: {result.lastId}")

// Error handling with try
let {result, error} = try (db <=!=> "INSERT INTO users (name) VALUES (?)" [name])
if (error) {
  log("Database error: {error}")
}
```

### Using Query DSL (Advanced)

The Query DSL provides a declarative way to query databases with schema bindings.

```parsley
let {db} = import @basil/auth

// Define schema with @schema
let Users = @schema db.users {
  id: string,
  name: string,
  email: string,
  active: boolean
}

// Query all active users
let users = @query(Users | active == true ??-> *)

// Query with ordering and limit
let topUsers = @query(Users | active == true | order name | limit 10 ??-> *)

// Query single row
let user = @query(Users | id == userId ?-> *)
```

**DSL Operators:**
- `??->` - Return many rows (array)
- `?->` - Return one row (or null)
- `|` - Pipe conditions
- `*` - Select all columns

See `docs/parsley/reference.md` for complete DSL documentation.

## File I/O in Handlers

### Reading Files

```parsley
// Read JSON
let data <== JSON(@./data/users.json)

// Read CSV
let data <== CSV(@./data/users.csv)

// Read text
let content <== TEXT(@./templates/welcome.txt)

// Error handling
let {data, error} <== JSON(@./config.json)
if (error) {
  log("Failed to read file: {error}")
}
```

### Writing Files (Requires Configuration)

```yaml
# basil.yaml
security:
  allow_write:
    - ./data
    - ./uploads
```

```parsley
// Write JSON
{name: "Alice", age: 30} ==> JSON(@./data/user.json)

// Write CSV
[[1,2,3], [4,5,6]] ==> CSV(@./data/output.csv)
```

## Asset Bundling

Basil automatically bundles CSS and JavaScript from your handlers directory.

```parsley
// In handlers/index.pars
<html>
<head>
  <CSS/>    <!-- Auto-generated: /__site.css?v=hash -->
</head>
<body>
  <h1>"Hello"</h1>
  <Javascript/>  <!-- Auto-generated: /__site.js?v=hash -->
</body>
</html>
```

All `.css` and `.js` files in your handlers directory are automatically:
- Concatenated in alphabetical order
- Cache-busted with content hash
- Served at `/__site.css` and `/__site.js`

## Common Pitfalls

### 1. Parts Need Routes

```yaml
# ❌ WRONG - Part not routed
<Part src={@~/parts/counter.part}/>

# ✅ CORRECT - Add route in basil.yaml
routes:
  - path: /parts/counter.part
    handler: ./parts/counter.part
```

### 2. Parts Can Only Export Functions

```parsley
// ❌ WRONG
export counter = 0        // Variables not allowed in .part files

// ✅ CORRECT
export default = fn(props) {
  let count = props.count
  // ... return HTML
}
```

### 3. Database Path is Relative to Config

```yaml
# If basil.yaml is in project root:
sqlite: ./data.db          # ✅ ./data.db
sqlite: data.db            # ✅ Same as ./data.db

# Not relative to handler file
```

### 4. Sessions Need Secret in Production

```yaml
# ❌ Dev mode uses random secret (sessions lost on restart)
# ✅ Set persistent secret
session:
  secret: "your-32-char-minimum-secret-key"
```

## Where to Learn More

- **Basil Framework**: See `docs/guide/basil-quick-start.md`
- **Parsley Language**: See `docs/parsley/CHEATSHEET.md`
- **Configuration**: See `docs/guide/configuration-example.yaml`
- **Authentication**: See `docs/guide/authentication.md`
- **Parts**: See `docs/guide/parts.md`
- **FAQ**: See `docs/guide/faq.md`
- **API Reference**: See `docs/parsley/reference.md`

## Quick Reference

### Essential Imports

```parsley
// HTTP basics
let {query, method, request, response} = import @basil/http

// Database and sessions
let {db, session, user} = import @basil/auth

// Standard library
let {floor, ceil} = import @std/math
let {email, url} = import @std/valid
let {redirect, notFound} = import @std/api
```

### Essential Commands

```bash
# Development
./basil --init myapp          # Create new project
./basil --dev                 # Run dev server
./basil --dev --port 3000     # Custom port

# Production
./basil --config basil.yaml   # Run production server

# Testing
curl http://localhost:8080/   # Test endpoint
```

### Essential Config

```yaml
server:
  port: 8080
site: ./site                   # Filesystem routing
sqlite: ./data.db              # Database
session:
  secret: "32-char-secret"     # Session key
```

## Best Practices

1. **Use dev mode during development** - `./basil --dev`
2. **Test with curl** - Verify endpoints work before browser testing
3. **Check dev logs** - Visit `http://localhost:8080/__dev/log`
4. **Use .part files for interactivity** - Not regular handlers
5. **Validate with pars first** - Test Parsley code with `./pars` before using in handlers
6. **Handle errors** - Use `{data, error}` pattern for file/database ops
7. **Set session secret** - Required for persistent sessions
8. **Whitelist write directories** - Use `security.allow_write` in config
