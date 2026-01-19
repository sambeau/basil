# CLI Reorganization Design

**Status:** Proposal  
**Date:** 2026-01-19  
**Author:** AI Assistant with human review

## Overview

This document proposes a reorganization of the `basil` and `pars` CLI tools to improve consistency, discoverability, and prepare for new features like `pars doc`.

## Current State

### `basil` CLI

```
basil                              # Run server (default)
basil --init <folder>              # Create new project
basil --dev, --port, etc.          # Server flags

basil users <cmd>                  # User management
  create, list, show, update, set-role, delete, reset

basil apikey <cmd>                 # API key management
  create, list, revoke

basil auth <cmd>                   # Email verification
  verify-email, resend-verification, reset-verification, status, email-logs
```

**Issues:**
1. `users`, `apikey`, and `auth` are all authentication-related but separate
2. `apikey` is singular, inconsistent with `users`
3. `auth` commands mostly operate on users but are separate from `users`
4. `--init` is a flag but behaves like a subcommand

### `pars` CLI

```
pars                               # REPL (default)
pars <file>                        # Execute file
pars --pp, --pretty                # Pretty-print output
pars --restrict-read, etc.         # Security flags
```

**Issues:**
1. No subcommand structure for future tools (doc, fmt, check)
2. Adding `pars doc` would be inconsistent with current flag-only design

## Proposed Design

### Principle: Subcommand-First

Both tools adopt a subcommand pattern similar to `go`, `git`, and `docker`:
- Default behavior preserved for backwards compatibility
- Explicit subcommands for clarity
- Related commands grouped under namespaces

---

## `pars` Reorganization

### Proposed Structure

```
pars                               # REPL (default, unchanged)
pars <file>                        # Execute file (unchanged shorthand)
pars run <file>                    # Execute file (explicit)
pars doc <file|dir>                # Generate documentation
pars fmt <file>                    # Format code (future)
pars check <file>                  # Lint/validate (future)
```

### Subcommand Details

#### `pars run`
Execute a Parsley script. Current behavior, made explicit.

```
pars run script.pars               # Execute script
pars run --pretty script.pars      # Pretty-print output
pars run --no-write script.pars    # Security: deny writes
```

**Migration:** `pars <file>` continues to work as shorthand for `pars run <file>`.

#### `pars doc`
Generate documentation from Parsley source files.

```
pars doc handlers/                 # Generate docs for directory
pars doc api.pars                  # Generate docs for single file
pars doc --format=html .           # Output as HTML
pars doc --format=md .             # Output as Markdown (default)
pars doc --out=docs/ handlers/     # Write to directory
```

**See:** Separate design doc for `@doc` syntax and doc generation.

#### `pars fmt` (Future)
Format Parsley source code.

```
pars fmt script.pars               # Format in place
pars fmt --check script.pars       # Check if formatted (exit code)
pars fmt --diff script.pars        # Show diff
```

#### `pars check` (Future)
Validate Parsley source without executing.

```
pars check script.pars             # Parse and validate
pars check --strict script.pars    # Stricter validation
```

### Backwards Compatibility

| Old Command | New Command | Behavior |
|-------------|-------------|----------|
| `pars` | `pars` | REPL (unchanged) |
| `pars script.pars` | `pars run script.pars` | Execute (both work) |
| `pars --pretty script.pars` | `pars run --pretty script.pars` | Both work |

---

## `basil` Reorganization

### Option A: Nested Auth Namespace (Recommended)

Group all authentication-related commands under `basil auth`:

```
basil                              # Run server (default)
basil serve                        # Run server (explicit)
basil init <folder>                # Create new project (was --init)

basil auth users <cmd>             # User management
  create, list, show, update, set-role, delete, reset

basil auth keys <cmd>              # API key management (renamed from apikey)
  create, list, revoke

basil auth email <cmd>             # Email verification (was auth)
  verify, resend, reset, status, logs
```

**Pros:**
- Clear grouping of related functionality
- Matches mental model: "auth stuff"
- Room for future auth features (oauth, sessions, etc.)

**Cons:**
- Breaking change for existing scripts
- More typing for common operations

### Option B: Flat with Consistent Naming

Keep flat structure but improve naming:

```
basil                              # Run server
basil serve                        # Run server (explicit)
basil init <folder>                # Create new project

basil users <cmd>                  # User management (unchanged)
basil keys <cmd>                   # API keys (renamed from apikey)
basil email <cmd>                  # Email verification (renamed from auth)
```

**Pros:**
- Minimal change
- Shorter commands

**Cons:**
- Less organized as features grow
- `email` is vague (could be sending emails, not just verification)

### Option C: User-Centric Organization

Organize commands around users as the primary entity:

```
basil users create                 # Create user
basil users list                   # List users  
basil users show <id>              # Show user details
basil users update <id>            # Update user
basil users delete <id>            # Delete user
basil users verify <id>            # Verify email (moved from auth)
basil users reset-codes <id>       # Reset recovery codes
basil users keys <id>              # List user's API keys
basil users keys create <id>       # Create key for user

basil keys revoke <key_id>         # Revoke key (standalone, key-centric)
basil email-logs [user_id]         # View email logs (standalone)
```

**Pros:**
- User is the natural entity to operate on
- Combines related operations

**Cons:**
- Inconsistent with key-centric operations
- `basil users keys create` is wordy

### Recommendation

**Option A (Nested Auth)** provides the cleanest long-term structure while allowing deprecation of old commands.

### Migration Strategy

1. **Phase 1:** Add new commands alongside old ones
   - `basil auth users` works alongside `basil users`
   - Print deprecation warning for old commands
   
2. **Phase 2:** Update documentation to use new commands
   - All examples use `basil auth users`
   - Old commands still work
   
3. **Phase 3:** Remove old commands in next major version
   - Clear error message pointing to new command

### Detailed Command Mapping (Option A)

| Old Command | New Command |
|-------------|-------------|
| `basil --init foo` | `basil init foo` |
| `basil users create` | `basil auth users create` |
| `basil users list` | `basil auth users list` |
| `basil users show <id>` | `basil auth users show <id>` |
| `basil users update <id>` | `basil auth users update <id>` |
| `basil users set-role <id>` | `basil auth users set-role <id>` |
| `basil users delete <id>` | `basil auth users delete <id>` |
| `basil users reset <id>` | `basil auth users reset <id>` |
| `basil apikey create` | `basil auth keys create` |
| `basil apikey list` | `basil auth keys list` |
| `basil apikey revoke <id>` | `basil auth keys revoke <id>` |
| `basil auth verify-email <id>` | `basil auth email verify <id>` |
| `basil auth resend-verification <id>` | `basil auth email resend <id>` |
| `basil auth reset-verification <id>` | `basil auth email reset <id>` |
| `basil auth status <id>` | `basil auth email status <id>` |
| `basil auth email-logs` | `basil auth email logs` |

---

## Help System

### `pars help`

```
$ pars --help
pars - Parsley language interpreter

Usage:
  pars [command]

Commands:
  run         Execute a Parsley script
  doc         Generate documentation
  fmt         Format source code (coming soon)
  check       Validate source code (coming soon)

Run 'pars <command> --help' for more information on a command.

Without a command, pars starts an interactive REPL.
Running 'pars <file>' is shorthand for 'pars run <file>'.
```

### `basil help`

```
$ basil --help
basil - Web server for Parsley

Usage:
  basil [command]

Server Commands:
  serve       Start the web server (default)
  init        Create a new Basil project

Auth Commands:
  auth        Manage authentication
    users     User management
    keys      API key management
    email     Email verification

Run 'basil <command> --help' for more information on a command.

Without a command, basil starts the web server.
```

---

## Implementation Plan

### Phase 1: pars doc (Independent)
1. Add subcommand infrastructure to `pars`
2. Implement `pars doc` command
3. Keep `pars <file>` working as before

### Phase 2: basil reorganization
1. Add `basil serve` as explicit alias
2. Add `basil init` (deprecate `--init`)
3. Add `basil auth users|keys|email` namespace
4. Deprecation warnings on old commands

### Phase 3: Cleanup (Major Version)
1. Remove deprecated commands
2. Update all documentation

---

## Open Questions

1. **Should `pars run` be required or remain optional?**
   - Recommendation: Keep optional for ergonomics

2. **How long should deprecation period be?**
   - Recommendation: One minor version with warnings, remove in next major

3. **Should we add `basil auth sessions` for session management?**
   - Recommendation: Yes, when session management CLI is needed

4. **Should `pars doc` be a separate binary?**
   - Recommendation: No, keep in `pars` for discoverability

---

## References

- Go CLI: `go build`, `go test`, `go doc`, `go fmt`
- Docker CLI: `docker container`, `docker image`, `docker network`
- Git CLI: `git branch`, `git remote`, `git stash`
