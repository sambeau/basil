# Basil Parsley Language Reference

> Basil web framework extensions to the Parsley language  
> All syntax verified against `pars` v0.2.0 and `basil` (January 2026)

## Table of Contents

1. [Connection Literals](#1-connection-literals)
2. [Database Operations](#2-database-operations)
3. [File I/O Operations](#3-file-io-operations)
4. [Format Factories](#4-format-factories)
5. [HTTP Context (@basil/http)](#5-http-context-basilhttp)
6. [Auth Context (@basil/auth)](#6-auth-context-basilauth)
7. [Session Methods](#7-session-methods)
8. [API Helpers (@std/api)](#8-api-helpers-stdapi)
9. [Dev Tools (@std/dev)](#9-dev-tools-stddev)
10. [Server Globals](#10-server-globals)
11. [Server Functions](#11-server-functions)
12. [Error Handling](#12-error-handling)

---

## 1. Connection Literals

Connection literals create database and service connections.

### 1.1 SQLite

```parsley
let db = @sqlite("./database.db")
let db = @sqlite(":memory:")
```

**Arguments:**
- `path` (string): File path to the database, or `":memory:"` for an in-memory database

**Returns:** `DBConnection` object with the following properties:
- `driver` — `"sqlite"`
- `inTransaction` — `true` if currently in a transaction
- `lastError` — Error message from last failed operation, or empty string
- `sqliteVersion` — SQLite version string (e.g., `"3.45.0"`)

### 1.2 PostgreSQL

```parsley
let db = @postgres("postgres://user:pass@host:5432/dbname")
```

**Arguments:**
- `dsn` (string): PostgreSQL connection string

**Returns:** `DBConnection` object with:
- `driver` — `"postgres"`
- `inTransaction` — `true` if currently in a transaction
- `lastError` — Error message from last failed operation

### 1.3 MySQL

```parsley
let db = @mysql("user:pass@tcp(host:3306)/dbname")
```

**Arguments:**
- `dsn` (string): MySQL connection string in Go database/sql format

**Returns:** `DBConnection` object with:
- `driver` — `"mysql"`
- `inTransaction` — `true` if currently in a transaction
- `lastError` — Error message from last failed operation

### 1.4 SFTP

```parsley
let sftp = @sftp("user@host:22")
```

**Arguments:**
- `address` (string): SSH address in `user@host:port` format
- `options` (Dictionary, optional):
  - `knownHostsFile` (path|string) — Path to known_hosts file for host key verification
  - `timeout` (duration) — Connection timeout

**Returns:** `SFTPConnection` object with properties:
- `host` (String) — Remote hostname
- `port` (Integer) — Port number (default: 22)
- `user` (String) — Username
- `connected` (Boolean) — `true` if connection is active
- `lastError` (String) — Error message from last failed operation

**Authentication:** Uses SSH agent for key-based authentication.

#### SFTP Connection Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.close()` | none | `null` | Close the connection |

#### SFTP File Operations

Access remote files by indexing the connection:

```parsley
let sftp = @sftp("user@host:22")
let remoteFile = sftp[@./path/to/file.txt]
```

This creates an `SFTPFileHandle` with methods:

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.mkdir(options?)` | `{parents?: boolean}` | `null` | Create directory (`parents: true` for recursive) |
| `.rmdir(options?)` | `{recursive?: boolean}` | `null` | Remove directory |
| `.remove()` | none | `null` | Remove file |

**Reading/Writing Remote Files:**

```parsley
let sftp = @sftp("user@host:22")

// Read remote file
let content <== text(sftp[@./remote/file.txt])

// Write to remote file
"data" ==> text(sftp[@./remote/file.txt])

// Create remote directory
sftp[@./remote/newdir].mkdir({parents: true})
```

**Errors:** `NET-0003` (SSH connection failed), `NET-0008` (host key verification failed), `NET-0009` (SFTP client failed)

### 1.5 Shell

Creates command handles for executing external programs.

```parsley
@shell("binary")
@shell("binary", ["arg1", "arg2"])
@shell("binary", ["args"], {env: {...}, dir: @./path, timeout: @30s})
```

**Arguments:**
- `binary` (String, required) — Command name or path to executable
- `args` (Array of String, optional) — Command-line arguments
- `options` (Dictionary, optional):
  - `env` (Dictionary) — Environment variables (replaces inherited env if set)
  - `dir` (path) — Working directory
  - `timeout` (duration) — Execution timeout

**Returns:** `Command` handle dictionary with:
- `__type` — `"command"`
- `binary` (String) — The binary name/path
- `args` (Array) — The arguments
- `options` (Dictionary) — The options

**Note:** `@shell(...)` creates a command handle but does not execute it. Use `<=#=>` to execute.

#### Execute Operator (`<=#=>`)

Executes a command handle with optional stdin input.

```parsley
let result = @shell("echo", ["hello"]) <=#=> null
let result = @shell("cat") <=#=> "stdin input"
```

**Arguments:**
- Left: Command handle from `@shell(...)`
- Right: Stdin input (`null` for no input, or string)

**Returns:** Result dictionary with:
- `stdout` (String) — Standard output
- `stderr` (String) — Standard error
- `exitCode` (Integer) — Exit code (0 = success, -1 = execution failed)
- `error` (String|null) — Error message if execution failed, `null` otherwise

**Examples:**

```parsley
// Simple command
let result = @shell("echo", ["Hello, World!"]) <=#=> null
result.stdout                   // "Hello, World!\n"
result.exitCode                 // 0

// Command with environment
let result = @shell("printenv", ["MY_VAR"], {
    env: {MY_VAR: "test"}
}) <=#=> null
result.stdout                   // "test\n"

// Command with stdin
let result = @shell("cat") <=#=> "piped input"
result.stdout                   // "piped input"

// Command with timeout
let result = @shell("sleep", ["10"], {timeout: @2s}) <=#=> null
// Kills process after 2 seconds

// Check for errors
if (result.exitCode != 0) {
    `Command failed: {result.stderr}`
}
```

**Security:**
- Commands are NOT passed through a shell—arguments are passed directly to `exec`
- Shell metacharacters in arguments are treated as literals (safe from injection)
- Security policy can restrict executable access in production mode

**Errors:** `CMD-0001` (missing field), `CMD-0002` (wrong type), `CMD-0003` (invalid args), `CMD-0004` (invalid stdin type)

### 1.6 Server Database (@DB)

In Basil handlers, `@DB` refers to the configured server database:

```parsley
// In basil.yaml: database.path: "./app.db"
let users = @DB <=??=> "SELECT * FROM users"
```

**Note:** `@DB` is a managed connection—scripts cannot close it.

---

## 2. Database Operations

All database operators require a `DBConnection` on the left-hand side.

### 2.1 Query One Row (`<=?=>`)

Returns a single row as a dictionary, or `null` if not found.

```parsley
let db = @sqlite("./test.db")
let user = db <=?=> "SELECT * FROM users WHERE id = 1"
user.name                       // "Alice"
```

**Arguments:**
- Left: `DBConnection` object
- Right: SQL string or `<SQL>` tag expression

**Returns:** 
- `Dictionary` — Row data with column names as keys, values auto-converted:
  - INTEGER → `Integer`
  - REAL/FLOAT → `Float`
  - TEXT/VARCHAR → `String`
  - BLOB → `Array` of integers (bytes)
  - NULL → `null`
- `null` — If no matching row found

**Errors:** `DB-0002` (query failed), `DB-0004` (scan failed), `DB-0008` (column error)

### 2.2 Query Many Rows (`<=??=>`)

Returns an array of dictionaries.

```parsley
let users = db <=??=> "SELECT * FROM users ORDER BY id"
users.length()                  // 2
users[0].name                   // "Alice"
```

**Arguments:**
- Left: `DBConnection` object
- Right: SQL string or `<SQL>` tag expression

**Returns:** `Array` of `Dictionary` objects, each representing a row. Empty array if no rows match.

**Errors:** `DB-0002` (query failed), `DB-0004` (scan failed), `DB-0008` (column error)

### 2.3 Execute Mutation (`<=!=>`)

Executes INSERT, UPDATE, DELETE, or DDL statements.

```parsley
let result = db <=!=> "INSERT INTO users (name, age) VALUES ('Bob', 25)"
result.affected                 // 1
result.lastId                   // 3
```

**Arguments:**
- Left: `DBConnection` object
- Right: SQL string or `<SQL>` tag expression

**Returns:** `Dictionary` with:
- `affected` (Integer) — Number of rows affected
- `lastId` (Integer) — Last inserted row ID (for INSERT statements)

**Important:** Execute statements must be assigned to a variable (even `_`):

```parsley
let _ = db <=!=> "DELETE FROM users WHERE id = 5"
```

**Errors:** `DB-0011` (execute failed)

### 2.4 Parameterized Queries

Use template strings for safe parameterization:

```parsley
let name = "Alice"
let user = db <=?=> `SELECT * FROM users WHERE name = '{name}'`
```

Or use the `<SQL>` tag for complex queries with positional parameters:

```parsley
let SearchUsers = fn(term) {
    <SQL params={name: term}>
        "SELECT * FROM users WHERE name LIKE ?"
    </SQL>
}
let users = db <=??=> <SearchUsers term="Ali%"/>
```

**`<SQL>` Tag Attributes:**
- `params` (Dictionary) — Named parameters, passed as positional `?` placeholders in sorted key order

**Note:** Template strings are convenient but the `<SQL>` tag provides explicit parameter binding when needed.

### 2.5 Database Connection Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.begin()` | none | `boolean` | Begin a transaction |
| `.commit()` | none | `boolean` | Commit the current transaction |
| `.rollback()` | none | `boolean` | Rollback the current transaction |
| `.close()` | none | `null` | Close the connection (not allowed on managed connections) |
| `.ping()` | none | `boolean` | Test if connection is alive |
| `.lastInsertId()` | none | `integer` | Get last inserted row ID (SQLite only) |
| `.createTable(schema, name?)` | `schema: Schema`, `name?: string` | `boolean` | Create table from schema if not exists |
| `.bind(schema, name, opts?)` | `schema: Schema`, `name: string`, `opts?: dict` | `TableBinding` | Bind schema to table |

#### Transaction Example

```parsley
let db = @sqlite("./app.db")

db.begin()
let _ = db <=!=> "INSERT INTO users (name) VALUES ('Alice')"
let _ = db <=!=> "INSERT INTO orders (user_id) VALUES (1)"
db.commit()
// Or: db.rollback() to cancel
```

**Note:** Managed connections (`@DB`) cannot be closed by scripts.

**Errors:**
- `DB-0006` — No transaction in progress (for commit/rollback)
- `DB-0007` — Already in transaction (for begin)
- `DB-0009` — Cannot close managed connection

#### Schema Binding Example

```parsley
let UserSchema = schema {
    name: string
    email: string
    age: integer?
}

// Bind schema to table for CRUD operations
let Users = db.bind(UserSchema, "users")

// Use table binding (see Schema-Table Binding documentation)
let user = Users.find(1)
```

---

## 3. File I/O Operations

File I/O operators work with file handles created by format factories.

### 3.1 Read File (`<==`)

Reads file content based on format factory.

```parsley
// Read as text
let content <== text(@./file.txt)

// Read as JSON
let data <== JSON(@./config.json)

// Read as lines array
let lines <== lines(@./log.txt)

// Read directory listing
let entries <== dir(@./data)
```

**Arguments:**
- Left: Variable name(s) for assignment
- Right: File handle from a format factory

**Returns:** Content type depends on the format factory used (see Section 4).

**Errors:** `IO-0003` (read failed), `FILEOP-0007` (invalid handle)

### 3.2 Write File (`==>`)

Writes content to file, overwriting existing content.

```parsley
"Hello, World!" ==> text(@./output.txt)
{name: "Alice"} ==> JSON(@./data.json)
```

**Arguments:**
- Left: Value to write (type must match format expectations)
- Right: File handle from a format factory

**Returns:** `null` on success

**Errors:** `IO-0004` (write failed)

### 3.3 Append to File (`==>>`)

Appends content to existing file.

```parsley
"Log entry\n" ==>> text(@./app.log)
```

**Arguments:**
- Left: Value to append
- Right: File handle (typically `text` or `lines` format)

**Returns:** `null` on success

### 3.4 Fetch URL (`<=/=`)

Fetches content from HTTP/HTTPS URLs.

```parsley
let data <=/= JSON(@https://api.example.com/users)
```

**Arguments:**
- Left: Variable name(s) for assignment
- Right: URL path literal with format factory

**Returns:** Parsed response body (type depends on format factory)

**Errors:** `NET-0002` (request failed), `NET-0004` (non-2xx status)

### 3.5 Error Capture Pattern

Use `{data, error}` destructuring to capture errors instead of failing:

```parsley
let {data, error} <== JSON(@./maybe-missing.json)
if (error) {
    "File not found or invalid JSON"
} else {
    data.value
}
```

**With error capture:**
- `data` — The successfully read content, or `null` on error
- `error` — Error message string, or `null` on success

This pattern prevents script termination on I/O errors, allowing graceful handling.

---

## 4. Format Factories

Format factories create file handles with specific read/write formats. All format factories accept a path literal as the first argument and an optional options dictionary as the second argument.

### 4.1 JSON

```parsley
let handle = JSON(@./data.json)
let data <== handle
```

**Arguments:**
- `path` (path literal or string) — File path
- `options` (Dictionary, optional) — Reserved for future use

**Read Returns:** Parsed JSON value (Dictionary, Array, String, Integer, Float, Boolean, or null)

**Write Accepts:** Any JSON-serializable value

### 4.2 CSV

```parsley
let handle = CSV(@./data.csv)
let rows <== handle             // Array of dictionaries
rows[0].name                    // First row, "name" column
```

**Arguments:**
- `path` (path literal or string) — File path
- `options` (Dictionary, optional):
  - `header` (Boolean) — If `true` (default), first row is treated as headers and rows are returned as dictionaries. If `false`, all rows are returned as arrays.

**Read Returns:**
- With headers: `Array` of `Dictionary` (column names as keys)
- Without headers: `Array` of `Array`

**Value Auto-Conversion:** CSV values are automatically converted:
- Integers (e.g., `"42"`) → `Integer`
- Floats (e.g., `"3.14"`) → `Float`
- `"true"`/`"false"` → `Boolean`
- All others → `String`

**Write Accepts:** `Array` of `Dictionary` or `Array` of `Array`

### 4.3 YAML

```parsley
let handle = YAML(@./config.yaml)
let config <== handle
```

**Arguments:**
- `path` (path literal or string) — File path
- `options` (Dictionary, optional) — Reserved for future use

**Read Returns:** Parsed YAML value (same types as JSON)

**Write Accepts:** Any YAML-serializable value

### 4.4 text

```parsley
let handle = text(@./readme.txt)
let content <== handle          // String
```

**Arguments:**
- `path` (path literal or string) — File path
- `options` (Dictionary, optional) — Reserved for future use (e.g., encoding)

**Read Returns:** `String` — Entire file content

**Write Accepts:** `String`

### 4.5 lines

```parsley
let handle = lines(@./file.txt)
let arr <== handle              // Array of strings
```

**Arguments:**
- `path` (path literal or string) — File path
- `options` (Dictionary, optional) — Reserved for future use

**Read Returns:** `Array` of `String` — One string per line (trailing empty line removed)

**Write Accepts:** `Array` of `String` (joined with newlines)

### 4.6 bytes

```parsley
let handle = bytes(@./binary.dat)
let data <== handle             // Array of integers (0-255)
```

**Arguments:**
- `path` (path literal or string) — File path
- `options` (Dictionary, optional) — Reserved for future use

**Read Returns:** `Array` of `Integer` — Each byte as an integer (0–255)

**Write Accepts:** `Array` of `Integer` (values 0–255)

### 4.7 SVG

Reads SVG files, stripping XML prolog for direct HTML embedding:

```parsley
let icon <== SVG(@./icon.svg)
<div class="icon">{icon}</div>
```

**Arguments:**
- `path` (path literal or string) — SVG file path
- `options` (Dictionary, optional) — Reserved for future use

**Read Returns:** `String` — SVG content without XML declaration, ready for HTML embedding

### 4.8 MD (Markdown)

Reads markdown files with optional YAML frontmatter:

```parsley
let doc <== MD(@./post.md)
doc.frontmatter.title           // YAML frontmatter as dictionary
doc.content                     // HTML content (rendered from markdown)
```

**Arguments:**
- `path` (path literal or string) — Markdown file path
- `options` (Dictionary, optional) — Reserved for future use

**Read Returns:** `Dictionary` with:
- `frontmatter` (Dictionary) — Parsed YAML frontmatter, or empty dictionary if none
- `content` (String) — Markdown rendered to HTML

### 4.9 dir (Directory)

Creates directory handle for listing contents:

```parsley
let d = dir(@./data)
let entries <== d
for (entry in entries) {
    entry.name
}
```

**Arguments:**
- `path` (path literal or string) — Directory path

**Read Returns:** `Array` of file/directory handles, each with:
- `name` (String) — Entry name
- `isDir` (Boolean) — `true` if directory
- `isFile` (Boolean) — `true` if file
- Format auto-detected from extension (for files)

### 4.10 file (Auto-detect)

Auto-detects format from file extension:

```parsley
let data <== file(@./config.json)   // Detected as JSON
let text <== file(@./readme.txt)    // Detected as text
```

**Arguments:**
- `path` (path literal or string) — File path

**Auto-detection mapping:**
- `.json` → JSON
- `.csv` → CSV
- `.txt`, `.md`, `.html`, `.xml`, `.pars` → text
- `.log` → lines
- All others → text

---

## 5. HTTP Context (@basil/http)

**Server-only**: Available in Basil request handlers.

```parsley
let {request, response, route, method} = import @basil/http
```

All exports are dynamic accessors that always return the current request's values, even when imported at module scope.

### 5.1 request

Access to the current HTTP request.

**Type:** `Dictionary`

**Properties:**
- `method` (String) — HTTP method: `"GET"`, `"POST"`, `"PUT"`, `"DELETE"`, etc.
- `path` (String) — Full request path (e.g., `"/users/123"`)
- `route` (String) — The matched route portion after the handler mount point
- `query` (Dictionary) — Query string parameters (e.g., `?name=foo` → `{name: "foo"}`)
- `headers` (Dictionary) — Request headers (lowercase keys)
- `body` (String|Dictionary) — Request body (for POST/PUT). JSON bodies are auto-parsed.

**Example:**
```parsley
let {request} = import @basil/http
request.method                  // "GET"
request.query.page ?? 1         // Query param with default
```

### 5.2 response

Set response headers, status, and cookies.

**Type:** `Dictionary` (mutable)

**Properties:**
- `status` (Integer) — HTTP status code (default: 200)
- `headers` (Dictionary) — Response headers to set
- `cookies` (Array) — Cookies to set (array of cookie dictionaries)

**Example:**
```parsley
let {response} = import @basil/http
response.status = 201
response.headers["X-Custom"] = "value"
response.cookies = [{name: "session", value: "abc123", httpOnly: true}]
```

**Cookie Dictionary Properties:**
- `name` (String, required) — Cookie name
- `value` (String, required) — Cookie value
- `path` (String) — Cookie path (default: "/")
- `domain` (String) — Cookie domain
- `maxAge` (Integer) — Max age in seconds
- `expires` (String) — Expiry date
- `httpOnly` (Boolean) — HTTP-only flag
- `secure` (Boolean) — Secure flag
- `sameSite` (String) — SameSite policy: `"Strict"`, `"Lax"`, `"None"`

### 5.3 route

The matched route path (portion after handler mount point).

**Type:** `String`

**Example:**
```parsley
let {route} = import @basil/http
// Handler mounted at /users, request to /users/123
route                           // "123"
```

### 5.4 method

Current HTTP method (shortcut for `request.method`).

**Type:** `String`

**Example:**
```parsley
let {method} = import @basil/http
if (method == "POST") {
    // Handle form submission
}
```

---

## 6. Auth Context (@basil/auth)

**Server-only**: Available when auth is configured in `basil.yaml`.

```parsley
let {db, session, auth, user} = import @basil/auth
```

All exports are dynamic accessors for per-request freshness.

### 6.1 db

Server-configured database connection.

**Type:** `DBConnection` (managed—cannot be closed by scripts)

**Example:**
```parsley
let {db} = import @basil/auth
let users = db <=??=> "SELECT * FROM users"
```

### 6.2 session

Current session module (see Section 7 for methods).

**Type:** `SessionModule`

### 6.3 auth

Authentication context dictionary.

**Type:** `Dictionary`

**Properties:**
- `required` (Boolean) — `true` if route requires authentication
- `user` (Dictionary|null) — Authenticated user, or `null` if not logged in

**Example:**
```parsley
let {auth} = import @basil/auth
if (auth.required && !auth.user) {
    redirect("/login")
}
```

### 6.4 user

Shortcut to `auth.user`—the currently authenticated user.

**Type:** `Dictionary` or `null`

**User Dictionary Properties (when authenticated):**
- `id` (Integer) — User ID
- `name` (String) — Display name
- `email` (String) — Email address
- `role` (String) — User role (e.g., `"admin"`, `"user"`)
- `created` (String) — Account creation timestamp
- `email_verified_at` (String|null) — Email verification timestamp
- `email_verification_pending` (Boolean) — `true` if email not yet verified

**Example:**
```parsley
let {user} = import @basil/auth
if (user) {
    `Welcome, {user.name}!`
} else {
    "Please log in"
}
```

---

## 7. Session Methods

**Server-only**: Available via `@basil/auth` session.

```parsley
let {session} = import @basil/auth
```

Session data is automatically persisted between requests (stored in encrypted cookies by default).

### 7.1 get(key, default?)

Retrieve a value from the session.

**Arguments:**
- `key` (String) — The session key to retrieve
- `default` (any, optional) — Value to return if key doesn't exist

**Returns:** Stored value, `default` if provided and key missing, or `null`

```parsley
session.get("userId")           // 123 or null
session.get("theme", "light")   // "light" if not set
```

### 7.2 set(key, value)

Store a value in the session.

**Arguments:**
- `key` (String) — The session key
- `value` (any) — Value to store (must be JSON-serializable)

**Returns:** `null`

```parsley
session.set("userId", 123)
session.set("cart", [{id: 1, qty: 2}])
```

### 7.3 has(key)

Check if a key exists in the session.

**Arguments:**
- `key` (String) — The session key to check

**Returns:** `Boolean`

```parsley
session.has("userId")           // true or false
```

### 7.4 delete(key)

Remove a value from the session.

**Arguments:**
- `key` (String) — The session key to remove

**Returns:** `null`

```parsley
session.delete("userId")
```

### 7.5 clear()

Remove all session data (including flash messages).

**Arguments:** None

**Returns:** `null`

```parsley
session.clear()
```

### 7.6 all()

Get all session data as a dictionary.

**Arguments:** None

**Returns:** `Dictionary` — All key-value pairs in the session

```parsley
let data = session.all()        // {userId: 123, theme: "dark", ...}
```

### 7.7 flash(key, message)

Set a flash message (one-time message, cleared after retrieval).

**Arguments:**
- `key` (String) — Flash message key (e.g., `"success"`, `"error"`)
- `message` (String) — The message content

**Returns:** `null`

```parsley
session.flash("success", "Item saved!")
session.flash("error", "Validation failed")
```

### 7.8 getFlash(key)

Retrieve and remove a flash message.

**Arguments:**
- `key` (String) — Flash message key

**Returns:** `String` (the message) or `null` if not set

```parsley
let msg = session.getFlash("success")  // "Item saved!" (then cleared)
```

### 7.9 hasFlash()

Check if any flash messages exist.

**Arguments:** None

**Returns:** `Boolean`

```parsley
if (session.hasFlash()) {
    // Display flash messages
}
```

### 7.10 getAllFlash()

Retrieve and remove all flash messages.

**Arguments:** None

**Returns:** `Dictionary` — All flash messages (then cleared)

```parsley
let flashes = session.getAllFlash()    // {success: "...", error: "..."}
```

### 7.11 regenerate()

Regenerate session ID (recommended after login for security).

**Arguments:** None

**Returns:** `null`

```parsley
// After successful login
session.set("userId", user.id)
session.regenerate()            // Prevent session fixation attacks
```

---

## 8. API Helpers (@std/api)

```parsley
import @std/api
// or
let {redirect, notFound, forbidden} = import @std/api
```

### 8.1 redirect(url, status?)

Create an HTTP redirect response.

**Arguments:**
- `url` (String or path) — Redirect destination URL
- `status` (Integer, optional) — HTTP status code (default: 302)

**Returns:** `Redirect` object (handled specially by Basil server)

**Valid status codes:** 301, 302, 303, 307, 308

**Errors:** `VALUE-0001` (empty URL), `VALUE-0002` (invalid status code)

```parsley
redirect("/dashboard")          // 302 Found (default)
redirect("/dashboard", 301)     // 301 Permanent Redirect
redirect("/other", 303)         // 303 See Other
redirect("/temp", 307)          // 307 Temporary Redirect
redirect("/perm", 308)          // 308 Permanent Redirect
```

### 8.2 Error Responses

All error functions create `APIError` objects that Basil converts to appropriate HTTP responses.

**Common Signature:**
- `message` (String, optional) — Custom error message

**Returns:** `APIError` object with `code`, `message`, and `status` properties

#### notFound(message?)

```parsley
notFound()                      // 404, "Not found"
notFound("User not found")      // 404, custom message
```

#### forbidden(message?)

```parsley
forbidden()                     // 403, "Forbidden"
forbidden("Access denied")
```

#### badRequest(message?)

```parsley
badRequest()                    // 400, "Bad request"
badRequest("Invalid input")
```

#### unauthorized(message?)

```parsley
unauthorized()                  // 401, "Unauthorized"
unauthorized("Please log in")
```

#### conflict(message?)

```parsley
conflict()                      // 409, "Conflict"
conflict("Resource already exists")
```

#### serverError(message?)

```parsley
serverError()                   // 500, "Internal server error"
serverError("Something went wrong")
```

### 8.3 Auth Wrappers

Wrap handler functions with auth requirements. These decorators add metadata that Basil checks before invoking the handler.

#### public(fn) / public(options, fn)

Mark a handler as publicly accessible (no authentication required).

**Arguments:**
- `options` (Dictionary, optional) — Additional options
- `fn` (Function) — The handler function to wrap

**Returns:** `AuthWrappedFunction`

```parsley
let api = import @std/api

export listProducts = api.public(fn() {
    // Anyone can access
})
```

#### adminOnly(fn)

Require admin role to access the handler.

**Arguments:**
- `fn` (Function) — The handler function to wrap

**Returns:** `AuthWrappedFunction`

```parsley
export deleteUser = api.adminOnly(fn(id) {
    // Only admin users can access
})
```

#### roles(roleList, fn)

Require specific roles to access the handler.

**Arguments:**
- `roleList` (Array of String) — List of allowed roles
- `fn` (Function) — The handler function to wrap

**Returns:** `AuthWrappedFunction`

```parsley
export editContent = api.roles(["editor", "admin"], fn() {
    // Only editors or admins can access
})
```

#### auth(fn) / auth(options, fn)

Require authentication (any logged-in user).

**Arguments:**
- `options` (Dictionary, optional) — Additional auth options
- `fn` (Function) — The handler function to wrap

**Returns:** `AuthWrappedFunction`

```parsley
export profile = api.auth(fn() {
    // Any authenticated user can access
})
```

---

## 9. Dev Tools (@std/dev)

```parsley
let {dev} = import @std/dev
```

Dev tools are for development debugging. All functions are **no-ops in production mode** (when `dev: false` in `basil.yaml`), so they can safely remain in production code.

### 9.1 dev.log(value) / dev.log(label, value) / dev.log(label, value, options)

Log values to Basil's dev panel.

**Arguments:**
- `value` (any) — Value to log
- `label` (String, optional) — Label for the log entry
- `options` (Dictionary, optional):
  - `level` (String) — Log level: `"info"` (default), `"warn"`, `"error"`

**Returns:** `null`

```parsley
dev.log(someValue)                      // Log value
dev.log("user", currentUser)            // Log with label
dev.log("warning", data, {level: "warn"}) // Log as warning
dev.log("error!", err, {level: "error"})  // Log as error
```

### 9.2 dev.clearLog()

Clear all dev log entries for the current route.

**Arguments:** None

**Returns:** `null`

```parsley
dev.clearLog()
```

### 9.3 dev.logPage(route, value) / dev.logPage(route, label, value, options?)

Log to a specific route's dev panel.

**Arguments:**
- `route` (String) — Route path (must start with `/`)
- `value` (any) — Value to log
- `label` (String, optional) — Label for the log entry
- `options` (Dictionary, optional):
  - `level` (String) — Log level: `"info"`, `"warn"`, `"error"`

**Returns:** `null`

**Errors:** `VAL-0009` (invalid route format)

```parsley
dev.logPage("/admin", "debug info")
dev.logPage("/admin", "users", userList)
dev.logPage("/admin", "error", err, {level: "error"})
```

### 9.4 dev.setLogRoute(route)

Set default route for subsequent `dev.log()` calls.

**Arguments:**
- `route` (String) — Route path (must start with `/`), or empty string to reset

**Returns:** `null`

**Errors:** `VAL-0009` (invalid route format)

```parsley
dev.setLogRoute("/dashboard")
dev.log("now goes to /dashboard")
dev.setLogRoute("")              // Reset to current route
```

### 9.5 dev.clearLogPage(route)

Clear logs for a specific route.

**Arguments:**
- `route` (String) — Route path to clear logs for

**Returns:** `null`

**Errors:** `VAL-0009` (invalid route format)

```parsley
dev.clearLogPage("/admin")
```

---

## 10. Server Globals

### 10.1 @params

**Server-only**: Merged URL query parameters and form data.

**Type:** `Dictionary`

Query parameters and POST form data are merged, with form data taking precedence on conflicts.

```parsley
// URL: /search?q=hello
// Or POST form: q=hello
@params.q                       // "hello"
@params["page"] ?? 1            // Default to 1 if missing
```

**Note:** Use `@params` instead of `request.query` for unified access to both query strings and form submissions.

### 10.2 @env

Environment variables dictionary. Works in both `pars` CLI and Basil server.

**Type:** `Dictionary`

```parsley
@env.HOME                       // "/Users/alice"
@env["DATABASE_URL"]            // Connection string
@env.PATH                       // System PATH
```

**Note:** Environment variables are read at startup and are read-only.

### 10.3 @args

Command-line arguments array. Primarily useful in `pars` CLI scripts.

**Type:** `Array` of `String`

```parsley
// pars script.pars arg1 arg2
@args[0]                        // "arg1"
@args[1]                        // "arg2"
@args.length()                  // 2
```

**Note:** In Basil server context, `@args` is typically empty.

---

## 11. Server Functions

### 11.1 publicUrl(path)

**Server-only**: Register a file and get its content-hashed public URL.

**Arguments:**
- `path` (path literal or String) — Path to the file to register

**Returns:** `String` — Public URL with content hash for cache-busting

**Errors:**
- `state` error — If called outside Basil server context
- `security` error — If path is outside handler directory
- `IO-0001` — If file cannot be read

```parsley
let logoUrl = publicUrl(@./assets/logo.svg)
<img src={logoUrl} alt="Logo"/>
// Produces: /assets/logo-a1b2c3d4.svg
```

**How it works:**
1. File content is hashed
2. File is copied to public assets with hash in filename
3. URL with hash is returned for cache-busting

**Security:** The path must be within the handler's root directory. Path traversal attempts are rejected.

### 11.2 CSRF Token

Access CSRF token via the basil context for form protection.

**Available via:** `basil.csrf.token` in the request context

```parsley
<form method="post">
    <input type="hidden" name="_csrf" value={basil.csrf.token}/>
    // form fields...
    <button type="submit">Submit</button>
</form>
```

**How it works:**
1. Basil generates a unique CSRF token per session
2. Token is stored in a cookie and available via `basil.csrf.token`
3. POST/PUT/DELETE requests must include the token as `_csrf` parameter
4. Basil validates the token and rejects mismatched requests

**Note:** CSRF protection is automatic for state-changing HTTP methods. Always include the token in forms.

---

## 12. Error Handling

### 12.1 Error Object Structure

When operations fail, Basil returns an `Error` object with:

- `class` — Error category (e.g., `"database"`, `"io"`, `"security"`)
- `code` — Specific error code (e.g., `"DB-0002"`, `"IO-0003"`)
- `message` — Human-readable error description
- `hints` — Array of suggestions for fixing the error

### 12.2 Error Classes

| Class | Description |
|-------|-------------|
| `database` | Database connection/query errors |
| `io` | File I/O errors |
| `network` | HTTP/SFTP network errors |
| `security` | Permission/access denied errors |
| `type` | Type mismatch errors |
| `value` | Invalid value errors |
| `state` | Invalid state errors |

### 12.3 Common Error Codes

#### Database Errors (DB-0xxx)

| Code | Description |
|------|-------------|
| `DB-0002` | Query execution failed |
| `DB-0003` | Failed to open database |
| `DB-0004` | Failed to scan row |
| `DB-0006` | No transaction in progress |
| `DB-0007` | Already in transaction |
| `DB-0009` | Cannot close managed connection |
| `DB-0011` | Execute statement failed |
| `DB-0012` | Invalid operand type for database operator |

#### I/O Errors (IO-0xxx)

| Code | Description |
|------|-------------|
| `IO-0001` | General file operation failed |
| `IO-0002` | Module not found |
| `IO-0003` | Failed to read file |
| `IO-0004` | Failed to write file |
| `IO-0005` | Failed to delete file |
| `IO-0006` | Failed to create directory |

#### Network Errors (NET-0xxx)

| Code | Description |
|------|-------------|
| `NET-0001` | Network operation failed |
| `NET-0002` | HTTP request failed |
| `NET-0003` | SSH connection failed |
| `NET-0004` | Non-2xx HTTP status returned |

#### Security Errors (SEC-0xxx)

| Code | Description |
|------|-------------|
| `SEC-0001` | General access denied |
| `SEC-0002` | Read access denied |
| `SEC-0003` | Write access denied |
| `SEC-0004` | Execute access denied |
| `SEC-0005` | Network access denied |

### 12.4 Error Handling Patterns

#### Pattern 1: Error Capture with File I/O

Use destructuring to capture errors without script termination:

```parsley
let {data, error} <== JSON(@./config.json)
if (error) {
    // Handle gracefully
    let config = {defaults: true}
} else {
    let config = data
}
```

#### Pattern 2: Null Checks for Database Queries

Single-row queries return `null` when no match is found:

```parsley
let user = db <=?=> `SELECT * FROM users WHERE id = {id}`
if (!user) {
    notFound("User not found")
} else {
    <UserProfile user={user}/>
}
```

#### Pattern 3: Empty Array for Multi-Row Queries

Multi-row queries return an empty array when no matches:

```parsley
let posts = db <=??=> "SELECT * FROM posts WHERE published = true"
if (posts.length() == 0) {
    <p>No posts yet.</p>
} else {
    for (post in posts) {
        <PostCard post={post}/>
    }
}
```

#### Pattern 4: API Error Responses

Use `@std/api` helpers to return appropriate HTTP errors:

```parsley
import @std/api

let user = db <=?=> `SELECT * FROM users WHERE id = {id}`
if (!user) {
    notFound("User not found")
}

if (user.role != "admin") {
    forbidden("Admin access required")
}
```

---

## Appendix A: Feature Availability

| Feature | pars CLI | Basil Server | Notes |
|---------|----------|--------------|-------|
| File I/O (`<==`, `==>`, `==>>`) | ✓ | ✓ | |
| URL Fetch (`<=/=`) | ✓ | ✓ | Requires `--allow-net` in pars |
| Database operators | ✓ | ✓ | |
| Format factories | ✓ | ✓ | |
| @env | ✓ | ✓ | |
| @args | ✓ | — | Empty in server context |
| @std/api | ✓ | ✓ | Redirect/errors are no-ops in pars |
| @std/dev | no-op | ✓ | Dev panel requires server |
| @basil/http | — | ✓ | Request context only |
| @basil/auth | — | ✓ | Requires auth config |
| @params | — | ✓ | Query + form data |
| @DB | — | ✓ | Server-configured database |
| Session methods | — | ✓ | Cookie-based sessions |
| publicUrl() | — | ✓ | Asset registration |
| CSRF token | — | ✓ | Form protection |

## Appendix B: Type Summary

| Type | Description | Example |
|------|-------------|---------|
| `DBConnection` | Database connection handle | `@sqlite("./db.sqlite")` |
| `SFTPConnection` | SFTP connection handle | `@sftp("user@host:22")` |
| `SessionModule` | Session data wrapper | `import @basil/auth` |
| `Redirect` | HTTP redirect response | `redirect("/path")` |
| `APIError` | HTTP error response | `notFound()` |
| `AuthWrappedFunction` | Auth-decorated function | `api.public(fn() {...})` |
| File Handle | File read/write handle | `JSON(@./data.json)` |
| Directory Handle | Directory listing handle | `dir(@./data)` |

## Appendix C: Format Factory Summary

| Factory | Read Type | Write Type | Options |
|---------|-----------|------------|---------|
| `JSON` | Dictionary/Array/... | any JSON-serializable | — |
| `CSV` | Array of Dict/Array | Array of Dict/Array | `{header: bool}` |
| `YAML` | Dictionary/Array/... | any YAML-serializable | — |
| `text` | String | String | — |
| `lines` | Array of String | Array of String | — |
| `bytes` | Array of Integer | Array of Integer | — |
| `SVG` | String (no prolog) | — | — |
| `MD` | Dict (frontmatter, content) | — | — |
| `dir` | Array of handles | — | — |
| `file` | auto-detected | auto-detected | — |
