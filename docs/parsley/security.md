# Parsley Security Guide

This document provides comprehensive security guidance for Parsley language features, especially for AI-assisted development and code review.

## Table of Contents
- [Security Model Overview](#security-model-overview)
- [Command Execution Security](#command-execution-security)
- [Database Security (SQL Injection Prevention)](#database-security-sql-injection-prevention)
- [File System Security](#file-system-security)
- [Network Security](#network-security)
- [HTML Output (Cross-Site Scripting)](#html-output-cross-site-scripting)
- [Security Policy Configuration](#security-policy-configuration)
- [Safe Patterns](#safe-patterns)
- [Unsafe Patterns](#unsafe-patterns)

---

## Security Model Overview

Parsley has two operational modes:

### Development Mode (env.Security = nil)
- **Full system access** - no restrictions
- File system, network, command execution all permitted
- Suitable for: local development, build scripts, automation tools
- **WARNING**: Never use nil security policy with untrusted input

### Production Mode (env.Security configured)
- **Sandboxed execution** - restricted by security policy
- Only explicitly permitted operations allowed
- Required for: web servers, public APIs, multi-tenant environments
- All security-sensitive operations check `env.Security` first

**Default behavior**: If `env.Security` is nil, all operations are permitted (dev mode).

---

## Command Execution Security

### How Command Execution Works

The `@shell` literal and the execute operator `<=#=>` run external commands using Go's `exec.Command`:

```parsley
let result = @shell("ls", ["-la"]) <=#=> null
// Equivalent to: exec.Command("ls", "-la")
```

### Security Properties

#### ✅ SAFE: No Shell Interpretation
Arguments are passed directly to the binary, **not through a shell**:

```parsley
// This is SAFE - semicolon is literal argument, not command separator
@shell("echo", ["hello; rm -rf /"]) <=#=> null
// Equivalent to: echo "hello; rm -rf /"
// Result: prints "hello; rm -rf /" (semicolon NOT interpreted)
```

#### ⚠️ RISK: Binary Name Can Reference Any Executable
If the binary name is user-controlled:

```parsley
let userInput = "../../usr/bin/dangerous"
@shell(userInput, []) <=#=> null  // Can execute any binary!
```

**Mitigation**: Security policy checks resolved binary path:
```go
if env.Security != nil {
    if err := env.checkPathAccess(resolvedPath, "execute"); err != nil {
        return error  // Blocked by security policy
    }
}
```

#### ⚠️ RISK: PATH Lookup
Simple binary names are resolved via PATH:

```parsley
@shell("python", []) <=#=> null  // Looks up "python" in PATH
```

If an attacker can manipulate PATH (via options.env), they can redirect to malicious binary.

**Mitigation**: Path resolution happens BEFORE custom environment is applied.

### Attack Scenarios & Mitigations

#### Scenario 1: Argument Injection (SAFE)
```parsley
// Attacker tries: @shell("ls", [userInput]) <=#=> null
// where userInput = "-la; rm -rf /"

@shell("ls", ["-la; rm -rf /"]) <=#=> null
// Result: ls receives ONE argument: "-la; rm -rf /"
// Shell metacharacters (;) are literal - NO DANGER
```
✅ **Safe**: Arguments are not shell-interpreted.

#### Scenario 2: Binary Path Traversal (BLOCKED)
```parsley
@shell("../../../usr/bin/evil", []) <=#=> null
// Security policy checks resolved path
// Result: Error if path not in AllowExecute list
```
✅ **Mitigated**: Security policy checks binary path.

#### Scenario 3: Environment Variable Injection (PARTIAL RISK)
```parsley
@shell("gcc", [], {env: {LD_PRELOAD: "/tmp/evil.so"}}) <=#=> null
// Loads malicious shared library into gcc process
```
⚠️ **Risk**: Custom environment can inject dangerous variables.

**Recommendation**: Security policy should block all untrusted command execution, or implement env var allowlist.

#### Scenario 4: Working Directory Escape (BLOCKED)
```parsley
@shell("cat", ["flag.txt"], {dir: path("../../../etc")}) <=#=> null
// Tries to read /etc/flag.txt
```
✅ **Mitigated**: Security policy checks `dir` path.

### Recommendations for Secure Command Execution

1. **Whitelist Permitted Binaries**
   ```go
   AllowExecute: []string{
       "/usr/bin/git",
       "/usr/bin/make",
       "/usr/local/bin/node",
   }
   ```

2. **Never Allow User-Controlled Binary Names**
   ```parsley
   // Basil only — needs the running server
   // UNSAFE:
   let binary = @params.cmd
   @shell(binary, []) <=#=> null  // Arbitrary code execution!

   // SAFE:
   let commands = {
       list: @shell("ls", ["-la"]),
       status: @shell("git", ["status"]),
   }
   let result = commands[@params.cmd] <=#=> null  // Fixed set
   ```

3. **Validate Arguments**
   Even though args are safe from shell injection, validate for application logic:
   ```parsley
   // Basil only — needs the running server
   let branch = @params.branch
   if (!(branch ~ /^[a-zA-Z0-9_-]+$/)) {
       fail("invalid branch name")
   }
   let result = @shell("git", ["checkout", branch]) <=#=> null
   ```

4. **Use Timeouts**
   Prevent indefinite hangs:
   ```parsley
   @shell("slow-command", [], {timeout: @30s}) <=#=> null
   ```

5. **Production Mode: Block All Commands**
   For web servers, consider blocking command execution entirely:
   ```go
   AllowExecute: []string{}  // Empty = no commands allowed
   ```

---

## Database Security (SQL Injection Prevention)

### Automatic SQL Injection Prevention

As of 2026-01-07, Parsley **automatically validates all SQL identifiers** (table names, column names, aliases) to prevent SQL injection.

#### ✅ SAFE: Validated Identifiers
```parsley
@schema users {
    id: integer(auto)
    name: string
    email: string
}

let db = @sqlite("./app.db")
db.createTable(users, "users")
let Users = db.bind(users, "users")

// Column names are validated
Users.insert({name: "Alice", email: "alice@example.com"})

// Projection columns are validated
Users.all({select: ["id", "name"]})
```

All identifiers must match: `^[a-zA-Z_][a-zA-Z0-9_]*$` (alphanumeric, underscore, max 64 chars)

#### ❌ BLOCKED: SQL Injection Attempts
```parsley
// These are BLOCKED with VAL-0003 error:

// Column name injection
Users.where({["name; DROP TABLE users--"]: "evil"})
// Error: invalid SQL identifier

// Projection injection
Users.all({select: ["id", "name' OR '1'='1"]})
// Error: invalid column name in select

// Table name injection (at binding creation)
db.bind(users, "users; DROP TABLE")
// Error: invalid table name
```

### Attack Scenarios & Mitigations

#### Scenario 1: Column Name Injection (BLOCKED)
```parsley
// Basil only — needs the running server
let userFields = {
    email: @params.email,
    ["role; DELETE FROM users--"]: "admin"  // Injection attempt
}
Users.where(userFields)
// Result: VAL-0003 error - invalid SQL identifier
```
✅ **Blocked**: All dictionary keys used as column names are validated.

#### Scenario 2: Dynamic Column Injection (BLOCKED)
```parsley
// Basil only — needs the running server
let sortCol = @params.sort  // User input: "id; DROP TABLE"
Users.all({orderBy: sortCol})
// Result: VAL-0003 error - invalid column name in orderBy
```
✅ **Blocked**: Ordering and projection columns are validated before SQL generation.

#### Scenario 3: Parametrized Values (SAFE)
```parsley
// Values are always parameterized, never interpolated
Users.where({name: userInput})
// Generates: SELECT * FROM users WHERE name = ?
// userInput = "Alice'; DROP TABLE--" is bound as parameter (safe)
```
✅ **Safe**: Values use SQL parameters, not string interpolation.

### Limitations & Edge Cases

**NOT protected** (but also not exposed to user input):
- Schema table names defined in code
- Fixed projection column names in code
- WHERE clause operators (always fixed: =, !=, <, >, IN, etc.)

**Protected**:
- All identifiers from dictionaries
- All identifiers from Query DSL projections
- Table names, aliases, column names

---

## File System Security

### Security Policy Checks

All file operations check security policy:

```parsley
let file <== text(@/etc/passwd)
// If env.Security set: checks AllowRead and DenyRead
```

### Safe Patterns

Restrict to a specific directory tree in the host policy:

```go
AllowRead: []string{"/var/app/data/**"}
DenyRead:  []string{"/var/app/data/secrets/**"}
```

```parsley
// Read allowed
let ok <== text(@/var/app/data/public/file.txt)  // OK

// Read denied
let key <== text(@/var/app/data/secrets/key.pem)  // Error
let pw <== text(@/etc/passwd)  // Error
```

### Unsafe Patterns

```parsley
// Basil only — needs the running server
// UNSAFE: User-controlled path without validation
let filename = @params.file
let data <== text(path("/var/app/data/" + filename))
// Attack: filename = "../../etc/passwd" → reads /etc/passwd

// SAFE: Validate and sanitize
if (filename.includes("..") || filename.includes("/")) {
    fail("invalid filename")
}
let safe <== text(path("/var/app/data/" + filename))  // Now safe
```

---

## Network Security

### HTTP Requests

```parsley
let response <=/= @https://api.example.com
```

**Security considerations**:
- HTTPS vs HTTP (credentials over HTTP logged as security warning)
- SSRF (Server-Side Request Forgery) if URL is user-controlled
- Credentials in URL (logged with per-IP audit trail as of 2026-01-07)

### SSRF Prevention

```parsley
// Basil only — needs the running server
// UNSAFE: User-controlled URL
let target = @params.url
let page <=/= url(target)  // Can request internal services!

// SAFE: Whitelist
let allowedHosts = ["api.example.com", "cdn.example.com"]
let parsedURL = url(target)
if (parsedURL.host not in allowedHosts) {
    fail("invalid host")
}
let safePage <=/= parsedURL
```

---

## HTML Output (Cross-Site Scripting)

### Current Behavior

Tag **content** interpolation is **raw**; **attribute values** are **escaped**.

```parsley
let u = "<script>alert(1)</script>"
<p>u</p>              // <p><script>alert(1)</script></p>  — raw
<p title={u}>""</p>  // <p title="&lt;script&gt;alert(1)&lt;/script&gt;">  — escaped
```

Content is raw by deliberate design: tags evaluate to strings, so a component's
return value can be embedded in another tag — escaping interpolated content
would render nested components as visible text. The cost is that interpolating
*untrusted input* (form submissions, `@params`, query strings, external APIs)
into tag content is a cross-site-scripting (XSS) vector, as in PHP, Perl, and
other raw-by-default templating systems.

Attribute values, by contrast, are HTML-escaped (`& < > "` → entities) so a
value cannot close its quotes and inject new attributes. This was not always
true: before [BUG-052](../../work/bugs/BUG-052.md) the attribute path escaped a
`"` as `\"`, which HTML ignores, and a crafted value could break out into a live
event handler. It is fixed — but escaping is **not context-aware** (see the
`href` caveat below).

The other escaped context is form components: `@field` labels, values, and
error messages are escaped by the renderer.

### What Narrows the Risk

- **Schema validation.** Typed record fields — `email`, `date`, `integer`,
  `enum`, `url`, `phone`, `slug` — cannot carry markup through validation.
  Free-text `string` and `text` fields can (a valid name may contain `<` or
  `&`), so validation reduces but does not remove the exposure.
- **Where values come from.** Markup produced by your own code (tags,
  `.parseMarkdown().html`, `MD()` handles) is trusted by construction. The risk
  is confined to values that originate outside the program.

### Recommendations

- Do not interpolate unprocessed user input into tag **content**. Prefer typed
  schema fields; where free text must be displayed, process it first (there is
  currently **no built-in escape helper** — see below).
- Attribute **values** are escaped against quote-breakout, but escaping is not
  context-aware: an attribute like `href` or `src` given a user-controlled value
  can still carry a `javascript:` URL that entity-escaping does not neutralise.
  Don't route untrusted input into URL attributes.
- Basil's security headers (`X-Content-Type-Options`, `X-Frame-Options`,
  `Referrer-Policy`) reduce the blast radius of an injection but do not
  prevent one.

### Planned Improvement

[FEAT-151](../../work/specs/FEAT-151.md) proposes an `Html` type with
escape-by-default tag interpolation (the Rails/Jinja2 model): plain strings are
escaped when interpolated into a tag, markup values pass through raw, and
`html()` blesses trusted strings. Phase 1 adds a `.escapeHTML()` string method
as the explicit primitive. Until then, the recommendations above are the
mitigation.

## Security Policy Configuration

### Example Production Configuration

```go
env := evaluator.NewEnvironment(nil)
env.Security = &evaluator.SecurityPolicy{
    // File system
    AllowRead: []string{
        "/var/app/data/**",
        "/var/app/templates/**",
    },
    DenyRead: []string{
        "/var/app/data/secrets/**",
    },
    AllowWrite: []string{
        "/var/app/data/uploads/**",
    },
    
    // Network (typically unrestricted for APIs)
    // Consider: implement URL allowlist for SSRF prevention
    
    // Commands (block all in production web server)
    AllowExecute: []string{},  // Empty = no commands
    
    // Future: add AllowBinaries: []string{"git", "make"}
}
```

---

## Safe Patterns

### ✅ Validated User Input
```parsley
// Basil only — needs the running server
let email = @params.email
if (!(email ~ /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/)) {
    fail("invalid email")
}
Users.insert({email: email})
```

### ✅ Whitelisted Operations
```parsley
// Basil only — needs the running server
let operations = {
    list: fn() { @query(Users ??-> *) },
    count: fn() { @query(Users ?-> count) },
}
let op = @params.op
if (!operations.has(op)) {
    fail("invalid operation")
}
operations[op]()
```

### ✅ Parameterized Queries
```parsley
// Values automatically parameterized
@query(Users | email == {userInput} ??-> *)
```

---

## Unsafe Patterns

### ❌ User-Controlled Binary Names
```parsley
// Basil only — needs the running server
let binary = @params.command
@shell(binary, []) <=#=> null  // Arbitrary code execution!
```

### ❌ Path Traversal
```parsley
// Basil only — needs the running server
let filename = @params.file
let data <== text(path("/data/" + filename))  // "../../../etc/passwd"
```

### ❌ SSRF
```parsley
// Basil only — needs the running server
let target = @params.url
let page <=/= url(target)  // Can hit internal services
```

### ❌ Trusting nil Security
```go
// In the Go host that embeds Parsley:
if env.Security == nil {
    // DANGEROUS: Full system access in production!
}
```

---

## AI Maintenance Checklist

When reviewing or writing Parsley code:

- [ ] Is `env.Security` configured for untrusted input?
- [ ] Are user-provided paths validated (no `..`, no absolute paths)?
- [ ] Are user-provided URLs whitelisted or validated?
- [ ] Is command binary name fixed (not user-controlled)?
- [ ] Are command arguments validated (even though shell-safe)?
- [ ] Are file operations within security policy boundaries?
- [ ] Is sensitive data logged? (mask credentials, keys)
- [ ] Is this feature necessary in production mode?
- [ ] Does error message leak sensitive information?
- [ ] Is rate limiting applied to expensive operations?

---

## Security Audit History

| Date | Feature | Security Fix | Severity |
|------|---------|--------------|----------|
| 2026-01-07 | SQL Identifiers | Added automatic validation | CRITICAL |
| 2026-01-07 | Git HTTP Auth | Per-IP insecure request tracking | MEDIUM |

---

## Reporting Security Issues

If you discover a security vulnerability in Parsley:

1. **Do NOT open a public GitHub issue**
2. Email security details to: [security contact]
3. Include: reproduction steps, impact assessment, suggested fix
4. Allow 90 days for patch before public disclosure

---

*Last updated: 2026-01-07*
*Maintainer: AI-assisted (human review required)*
