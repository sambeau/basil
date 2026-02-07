---
id: man-pars-commands
title: Shell Commands
system: parsley
type: features
name: commands
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - shell
  - command
  - execute
  - process
  - stdin
  - stdout
  - stderr
  - exit code
---

# Shell Commands

Parsley can execute external commands using the `@shell` literal and the execute operator `<=#=>`. Commands run as child processes — arguments are passed directly to the binary with no shell interpretation, which prevents shell injection attacks by design.

## Creating a Command Handle

`@shell` takes a binary name, an optional arguments array, and an optional options dictionary:

```parsley
let cmd = @shell("ls", ["-la"])
let git = @shell("git", ["status"])
let node = @shell("node", ["--version"])
```

The first argument is the binary name or path. Simple names are resolved via `PATH`; paths containing `/` are used as-is.

The second argument is an array of string arguments:

```parsley
let cmd = @shell("grep", ["-r", "TODO", "./src"])
```

## Executing Commands

The execute operator `<=#=>` runs a command and returns a result dictionary:

```parsley
let result = @shell("echo", ["hello"]) <=#=> null
result.stdout                    // "hello\n"
result.exitCode                  // 0
```

The right side of `<=#=>` is the stdin input. Pass `null` for no input:

```parsley
let result = @shell("echo", ["hello"]) <=#=> null
```

Pass a string to pipe it to the command's stdin:

```parsley
let input = "line1\nline2\nline3"
let result = @shell("wc", ["-l"]) <=#=> input
result.stdout                    // "       3\n"
```

## Result Dictionary

Every command execution returns a dictionary with these keys:

| Key | Type | Description |
|---|---|---|
| `stdout` | string | Standard output |
| `stderr` | string | Standard error |
| `exitCode` | integer | Exit code (0 = success, -1 = failed to run) |
| `error` | string or null | Error message if the command could not be started |

```parsley
let result = @shell("ls", ["nonexistent"]) <=#=> null
result.exitCode                  // non-zero (e.g. 2)
result.stderr                    // "ls: nonexistent: No such file or directory\n"
result.error                     // null (command ran but failed)
```

If the binary is not found:

```parsley
let result = @shell("nonexistent_binary", []) <=#=> null
result.exitCode                  // -1
result.error                     // "command not found: nonexistent_binary"
```

## Options

The third argument to `@shell` is an options dictionary:

```parsley
let cmd = @shell("make", ["build"], {
    env: {PATH: "/usr/local/bin:/usr/bin"},
    dir: @./project,
    timeout: @dur(30, "s")
})
let result = cmd <=#=> null
```

| Option | Type | Description |
|---|---|---|
| `env` | dictionary | Environment variables (replaces inherited env) |
| `dir` | path | Working directory for the command |
| `timeout` | duration | Maximum execution time before the process is killed |

### Environment Variables

Custom environment variables replace the entire inherited environment. Set only the variables the command needs:

```parsley
let result = @shell("env", [], {
    env: {HOME: "/tmp", USER: "test"}
}) <=#=> null
```

### Working Directory

```parsley
let result = @shell("pwd", [], {
    dir: @./subdir
}) <=#=> null
result.stdout                    // path to subdir
```

### Timeouts

Prevent runaway processes with a timeout. The process is killed if it exceeds the duration:

```parsley
let result = @shell("sleep", ["60"], {
    timeout: @dur(5, "s")
}) <=#=> null
// Killed after 5 seconds
```

## Security

Command execution is controlled by the security policy. In production (Basil server) mode, the security policy restricts which binaries can be run.

> ⚠️ Arguments are passed directly to the binary — **not** through a shell. Shell metacharacters like `;`, `|`, `&&`, and backticks are treated as literal characters, not command separators. This prevents shell injection by design.

### Security Policy

| Policy Field | Type | Description |
|---|---|---|
| `AllowExecute` | string array | Allowed binary paths (whitelist) |
| `AllowExecuteAll` | boolean | Allow all binaries (development only) |

In production, set `AllowExecute` to an explicit list of permitted binaries. An empty list blocks all command execution.

### Safe Patterns

```parsley
// Fixed binary name and validated arguments
let branch = userInput
check branch ~ /^[a-zA-Z0-9_-]+$/ else fail("invalid branch name")
let result = @shell("git", ["checkout", branch]) <=#=> null
```

### Unsafe Patterns

```parsley
// NEVER use user input as a binary name
let cmd = userInput
@shell(cmd, []) <=#=> null       // arbitrary code execution!
```

## Key Differences from Other Languages

- **No shell interpretation** — arguments go directly to the binary via `exec`. There is no `sh -c` wrapper, so `|`, `;`, `&&`, and other shell features don't work (and can't be exploited).
- **Operator syntax** — `<=#=>` makes data flow explicit. The left side is the command, the right side is stdin.
- **Structured result** — you get `{stdout, stderr, exitCode, error}` instead of just a string or exit code.
- **Security by default** — in server mode, commands are blocked unless explicitly allowed by the security policy.

## See Also

- [Security Model](security.md) — file, SQL, and command security policies
- [File I/O](file-io.md) — reading and writing files
- [Error Handling](../fundamentals/errors.md) — handling command failures