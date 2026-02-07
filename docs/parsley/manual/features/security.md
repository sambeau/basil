---
id: man-pars-security
title: Security Model
system: parsley
type: features
name: security
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - security
  - policy
  - sandbox
  - file access
  - SQL injection
  - command execution
  - PLN
  - permissions
---

# Security Model

Parsley has a configurable security model that controls file system access, command execution, and database safety. In development mode, everything is permitted. In production (Basil server), a security policy restricts what Parsley code can do.

## Operational Modes

### Development Mode

When no security policy is configured (`env.Security = nil`), Parsley has unrestricted access to the file system, network, and command execution. This is the default for the `pars` CLI and local scripts.

### Production Mode

Inside a Basil server, the security policy is configured by the host application. Parsley code runs in a sandbox where only explicitly permitted operations succeed.

## Security Policy

The security policy controls three areas: file reads, file writes, and command execution.

| Setting | Type | Description |
|---|---|---|
| `RestrictRead` | array of paths | Directories denied for reading (blacklist) |
| `NoRead` | boolean | Deny all file reads |
| `RestrictWrite` | array of paths | Directories denied for writing (blacklist) |
| `NoWrite` | boolean | Deny all file writes |
| `AllowWrite` | array of paths | Directories allowed for writing (whitelist) |
| `AllowWriteAll` | boolean | Allow writing to any path |
| `AllowExecute` | array of paths | Directories allowed for command execution (whitelist) |
| `AllowExecuteAll` | boolean | Allow executing any command |

## File System Restrictions

### Read Restrictions

File reads can be restricted by blacklisting directories or by disabling reads entirely:

```parsley
// These reads would be blocked if the path is in RestrictRead
let config <== JSON(@./secrets/keys.json)    // blocked
let public <== text(@./public/readme.txt)    // allowed
```

When `NoRead` is `true`, all file read operations produce an IO-class error.

### Write Restrictions

Writes use a whitelist model — only paths in `AllowWrite` are permitted (unless `AllowWriteAll` is `true`):

```parsley
// Only succeeds if @./uploads is in AllowWrite
data ==> JSON(@./uploads/result.json)
```

Attempting to write outside allowed directories produces a security error.

## SQL Injection Prevention

Parsley prevents SQL injection at two levels:

### Parameterized Values

All values passed through `<SQL>` tag parameters or the Query DSL are bound as SQL parameters, never interpolated into query strings:

```parsley
// SAFE — value is parameterized
let user = db <=?=> <SQL name={userInput}>
    "SELECT * FROM users WHERE name = ?"
</SQL>
```

```parsley
// SAFE — DSL conditions are parameterized
@query(Users | name == {userInput} ?-> *)
```

### Identifier Validation

Table names, column names, and aliases are validated against a strict pattern before being used in SQL. Valid identifiers must:

- Start with a letter or underscore
- Contain only letters, digits, and underscores
- Be at most 64 characters long

Identifiers that fail validation produce an immediate error, blocking SQL injection through identifier manipulation:

```parsley
// Blocked — invalid identifier
db.bind(User, "users; DROP TABLE users--")   // error
```

> ⚠️ Always use `<SQL>` parameters or the Query DSL for user-provided values. Never interpolate user input into raw SQL strings with template literals.

## Command Execution

The `@shell` literal and `<=#=>` operator run external commands. In production mode, only binaries in `AllowExecute` directories can be run.

### No Shell Interpretation

Commands are executed directly via the operating system (not through a shell). Shell metacharacters in arguments are treated as literal characters:

```parsley
let cmd = @shell("echo", ["hello; rm -rf /"])
let result <=#=> cmd
result.stdout                    // "hello; rm -rf /\n"
```

The semicolon is passed as part of the argument — it is **not** interpreted as a command separator.

### Managed Connections

Managed database connections (from `@DB`) cannot be closed by Parsley code. Calling `.close()` on a managed connection raises `DB-0009`. This prevents scripts from disrupting the server's shared database connection.

## PLN Safety

Parsley Literal Notation (PLN) is a data-only serialization format. Deserializing PLN never executes code — it only reconstructs literal values (strings, numbers, arrays, dictionaries, etc.). This makes PLN safe for loading untrusted data files, unlike formats that support code execution during deserialization.

## Error Classes

Security-related errors use the `security` error class and are catchable with `try`:

```parsley
let result = try(fn() {
    let secret <== text(@./secrets/key.pem)
})
if (result.error) {
    log("Access denied: " + result.error)
}
```

## Key Differences from Other Languages

- **Policy, not permissions** — security is configured at the environment level by the host application, not per-file or per-user. Parsley code cannot escalate its own permissions.
- **No shell interpretation** — command execution bypasses the shell entirely, eliminating an entire class of injection attacks.
- **Automatic SQL validation** — identifier validation happens transparently. You don't need to call a sanitize function.
- **Data-only deserialization** — PLN cannot execute code during parsing, unlike Python's `pickle` or Ruby's `Marshal`.

## See Also

- [Database](database.md) — SQL operators and parameterized queries
- [Query DSL](query-dsl.md) — declarative queries with automatic parameterization
- [File I/O](file-io.md) — file read/write operators
- [Shell Commands](commands.md) — command execution
- [Error Handling](../fundamentals/errors.md) — catchable error classes