---
id: man-bas-running
title: "Running Basil"
system: basil
type: feature
name: running
created: 2026-07-12
version: 1.0.0-alpha.3
author: "@sam"
keywords:
  - init
  - dev
  - cli
  - server
  - port
  - profile
  - signals
  - users
  - apikey
---

# Running Basil

The `basil` command starts the server, scaffolds projects, and manages users and API keys.

```bash
basil --init myapp    # Create a new project
cd myapp
basil --dev           # Run it in development mode
```

## Creating a Project

`basil --init FOLDER` scaffolds a new project:

```
myapp/
├── .gitignore      Git ignore patterns
├── basil.yaml      Configuration
├── site/           Handlers (filesystem routing)
│   └── index.pars  Homepage
├── public/         Static files (CSS, JS, images)
├── db/             SQLite databases
└── logs/           Log files
```

## Starting the Server

```bash
basil                 # Production mode, uses ./basil.yaml
basil --dev           # Development mode (HTTP on localhost)
basil --port 3000     # Override the configured port
basil --quiet         # Suppress request logs
```

Development mode (`--dev`) serves plain HTTP on localhost, disables response caching, injects live-reload into pages (the browser refreshes when you save a handler), shows detailed error pages, and enables the [dev tools](dev-tools.md).

### Config Resolution

Basil finds its configuration in this order:

1. `--config PATH` flag
2. `BASIL_CONFIG` environment variable
3. `./basil.yaml`
4. `~/.config/basil/basil.yaml`

### Developer Profiles

Named profiles in [`basil.yaml`](configuration.md) let each developer override the port, database, or handlers directory:

```bash
basil --dev --profile sam    # or: basil --dev -as sam
```

## CLI Reference

| Flag | Description |
|------|-------------|
| `--config PATH` | Path to config file (default: auto-detect) |
| `--dev` | Development mode (HTTP on localhost) |
| `--quiet` | Suppress request logs (log level: error) |
| `--port PORT` | Override listen port |
| `--profile NAME`, `-as NAME` | Apply a developer profile |
| `--init FOLDER` | Create a new Basil project |
| `--version` | Show version |
| `--help` | Show help |

## User Management

Basil's [authentication](authentication.md) users are managed from the CLI:

```bash
basil users create           # Create a new user
basil users list             # List all users
basil users show <id>        # Show user details
basil users update <id>      # Update name/email
basil users set-role <id>    # Change role
basil users delete <id>      # Delete a user
basil users reset <id>       # Generate new recovery codes
```

## API Key Management

API keys authenticate [Git deploys](git.md) and programmatic access:

```bash
basil apikey create          # Create an API key for a user
basil apikey list            # List a user's API keys
basil apikey revoke <id>     # Revoke an API key
```

## Signals

| Signal | Effect |
|--------|--------|
| `SIGHUP` | Reload scripts (clear cache, re-parse on next request) |

## See Also

- [Configuration](configuration.md) — everything in `basil.yaml`
- [Routing](routing.md) — how requests find handlers
- [Dev Tools](dev-tools.md) — what dev mode gives you
