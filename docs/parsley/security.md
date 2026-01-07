# Parsley Security Guide

This document provides comprehensive security guidance for Parsley language features, especially for AI-assisted development and code review.

## Table of Contents
- [Security Model Overview](#security-model-overview)
- [Command Execution Security](#command-execution-security)
- [Database Security (SQL Injection Prevention)](#database-security-sql-injection-prevention)
- [File System Security](#file-system-security)
- [Network Security](#network-security)
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

The `execute()` function runs external commands using Go's `exec.Command`:

```parsley
let result = execute(cmd("ls", "-la"))
// Equivalent to: exec.Command("ls", "-la")
```

### Security Properties

#### ✅ SAFE: No Shell Interpretation
Arguments are passed directly to the binary, **not through a shell**:

```parsley
// This is SAFE - semicolon is literal argument, not command separator
execute(cmd("echo", "hello; rm -rf /"))
// Equivalent to: echo "hello; rm -rf /"
// Result: prints "hello; rm -rf /" (semicolon NOT interpreted)
```

#### ⚠️ RISK: Binary Name Can Reference Any Executable
If the binary name is user-controlled:

```parsley
let userInput = "../../usr/bin/dangerous"
execute(cmd(userInput))  // Can execute any binary!
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
execute(cmd("python"))  // Looks up "python" in PATH
```

If an attacker can manipulate PATH (via options.env), they can redirect to malicious binary.

**Mitigation**: Path resolution happens BEFORE custom environment is applied.

### Attack Scenarios & Mitigations

#### Scenario 1: Argument Injection (SAFE)
```parsley
// Attacker tries: execute(cmd("ls", userInput))
// where userInput = "-la; rm -rf /"

execute(cmd("ls", "-la; rm -rf /"))
// Result: ls receives ONE argument: "-la; rm -rf /"
// Shell metacharacters (;) are literal - NO DANGER
```
✅ **Safe**: Arguments are not shell-interpreted.

#### Scenario 2: Binary Path Traversal (BLOCKED)
```parsley
execute(cmd("../../../usr/bin/evil"))
// Security policy checks resolved path
// Result: Error if path not in AllowExecute list
```
✅ **Mitigated**: Security policy checks binary path.

#### Scenario 3: Environment Variable Injection (PARTIAL RISK)
```parsley
execute(cmd("gcc"), {env: {("LD_PRELOAD"): "/tmp/evil.so"}})
// Loads malicious shared library into gcc process
```
⚠️ **Risk**: Custom environment can inject dangerous variables.

**Recommendation**: Security policy should block all untrusted command execution, or implement env var allowlist.

#### Scenario 4: Working Directory Escape (BLOCKED)
```parsley
execute(cmd("cat", "flag.txt"), {dir: path("../../../etc")})
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
   // UNSAFE:
   let binary = request.params.get("cmd")
   execute(cmd(binary))  // Arbitrary code execution!
   
   // SAFE:
   let commands = {
       ("list"): cmd("ls", "-la"),
       ("status"): cmd("git", "status"),
   }
   execute(commands.get(request.params.get("cmd")))  // Fixed set
   ```

3. **Validate Arguments**
   Even though args are safe from shell injection, validate for application logic:
   ```parsley
   let branch = request.params.get("branch")
   if (!branch.match(/^[a-zA-Z0-9_-]+$/)) {
       fail("invalid branch name")
   }
   execute(cmd("git", "checkout", branch))
   ```

4. **Use Timeouts**
   Prevent indefinite hangs:
   ```parsley
   execute(cmd("slow-command"), {timeout: dur(30, "s")})
   ```

5. **Production Mode: Block All Commands**
   For web servers, consider blocking execute() entirely:
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
    id: integer primary,
    name: string,
    email: string
}

let db = database("sqlite:./app.db")
let Users = db.table("users", schema: users)

// Column names are validated
Users.insert({name: "Alice", email: "alice@example.com"})

// Projection columns are validated  
@query(Users) ?-> ["id", "name"]
```

All identifiers must match: `^[a-zA-Z_][a-zA-Z0-9_]*$` (alphanumeric, underscore, max 64 chars)

#### ❌ BLOCKED: SQL Injection Attempts
```parsley
// These are BLOCKED with VAL-0003 error:

// Column name injection
Users.insert({("name; DROP TABLE users--"): "evil"})
// Error: invalid SQL identifier

// Projection injection  
@query(Users) ?-> ["id", "name' OR '1'='1"]
// Error: invalid column name in projection

// Table name injection (at binding creation)
db.table("users; DROP TABLE", schema: users)
// Error: invalid table name
```

### Attack Scenarios & Mitigations

#### Scenario 1: Column Name Injection (BLOCKED)
```parsley
let userFields = {
    ("email"): request.params.get("email"),
    ("role; DELETE FROM users--"): "admin"  // Injection attempt
}
Users.insert(userFields)
// Result: VAL-0003 error - invalid SQL identifier
```
✅ **Blocked**: All dictionary keys used as column names are validated.

#### Scenario 2: Query DSL Injection (BLOCKED)
```parsley
let sortCol = request.params.get("sort")  // User input: "id; DROP TABLE"
@query(Users) ?-> [sortCol]
// Result: VAL-0003 error - invalid column name in projection
```
✅ **Blocked**: Projection columns validated before SQL generation.

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
let file = fs.read("/etc/passwd")
// If env.Security set: checks AllowRead and DenyRead
```

### Safe Patterns

```parsley
// Restrict to specific directory tree
AllowRead: []string{"/var/app/data/**"}
DenyRead: []string{"/var/app/data/secrets/**"}

// Read allowed
fs.read("/var/app/data/public/file.txt")  // OK

// Read denied
fs.read("/var/app/data/secrets/key.pem")  // Error
fs.read("/etc/passwd")  // Error
```

### Unsafe Patterns

```parsley
// UNSAFE: User-controlled path without validation
let filename = request.params.get("file")
fs.read("/var/app/data/" + filename)
// Attack: filename = "../../etc/passwd" → reads /etc/passwd

// SAFE: Validate and sanitize
let filename = request.params.get("file")
if (filename.contains("..") || filename.contains("/")) {
    fail("invalid filename")
}
fs.read("/var/app/data/" + filename)  // Now safe
```

---

## Network Security

### HTTP Requests

```parsley
let response = http.get("https://api.example.com")
```

**Security considerations**:
- HTTPS vs HTTP (credentials over HTTP logged as security warning)
- SSRF (Server-Side Request Forgery) if URL is user-controlled
- Credentials in URL (logged with per-IP audit trail as of 2026-01-07)

### SSRF Prevention

```parsley
// UNSAFE: User-controlled URL
let url = request.params.get("url")
http.get(url)  // Can request internal services!

// SAFE: Whitelist
let allowedHosts = ["api.example.com", "cdn.example.com"]
let url = request.params.get("url")
let parsedURL = url(url)
if (!allowedHosts.contains(parsedURL.host)) {
    fail("invalid host")
}
http.get(url)
```

---

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
let email = request.params.get("email")
if (!email.match(/^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/)) {
    fail("invalid email")
}
Users.insert({email: email})
```

### ✅ Whitelisted Operations
```parsley
let operations = {
    ("list"): fn() { @query(Users) ?-> * },
    ("count"): fn() { @query(Users) ?-> .count },
}
let op = request.params.get("op")
if (!operations.has(op)) {
    fail("invalid operation")
}
operations.get(op)()
```

### ✅ Parameterized Queries
```parsley
// Values automatically parameterized
@query(Users).where({email: userInput}) ?-> *
```

---

## Unsafe Patterns

### ❌ User-Controlled Binary Names
```parsley
let cmd = request.params.get("command")
execute(cmd(cmd))  // Arbitrary code execution!
```

### ❌ Path Traversal
```parsley
let filename = request.params.get("file")
fs.read("/data/" + filename)  // "../../../etc/passwd"
```

### ❌ SSRF
```parsley
let url = request.params.get("url")
http.get(url)  // Can hit internal services
```

### ❌ Trusting nil Security
```parsley
// In web server handler:
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
