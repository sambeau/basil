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
basil --init myapp --host localhost --admin sam   # Create a new site
basil --dev --site myapp                          # Run it in development mode
```

## Creating a Site

`basil --init FOLDER --host HOSTNAME --admin NAME` creates a **site root**:

```
myapp/
├── site.git/       Bare repository, served at /.git
├── releases/       One directory per deployed commit
│   └── <commit>/     basil.yaml, site/index.pars, public/
├── current ->      The active release
└── data/           Databases, certificates, logs, uploads
    └── uploads/      Durable site writes, served at /__uploads/
```

Both flags are required, and neither is guessed:

- `--host` is written to `server.host`, the site's **public hostname**: the name on
  the certificate, the WebAuthn relying-party id, the address people type. It is
  not a bind address — the listener uses `server.bind` (empty means all
  interfaces), so a site whose hostname points at a load balancer, a NAT, or a
  container host still starts. A public server refuses to start without a host,
  because an empty host tells the certificate manager to attempt issuance for any
  hostname a stranger asks for. Use `localhost` for a site you will only run with
  `--dev`.
- `--admin` names the first Basil account. It is **never** derived from `$USER`
  or `$SUDO_USER` — `--init` usually runs on a server, where the shell is `root`
  or a service account. With a terminal attached, `--init` prompts for it.

`--init` commits the starter site to the release branch and deploys it as release
1, so the server has something to serve from the moment it starts. It creates the
admin account and prints its API key **once**, and installs a pre-commit
formatting hook.

Run under `sudo`, it hands the tree to `$SUDO_USER` and says so; run as `root`
with no `SUDO_USER`, it warns and prints the exact `chown` command. As root it
also insists the folder does not exist yet and that its parent is not writable by
other accounts (unless it is sticky, like `/tmp`): `--init` makes several `git`
calls between checking the folder and its last write, and every write it makes
follows symlinks, so a directory another account prepared could redirect root's
writes. Create the site somewhere only root can write, such as `/srv`. Skipping
the ownership step makes every later write fail — database, logs, certificates and deploys —
with an error that points at the database rather than at ownership.

### Code and state: the two anchors

Everything in the release directory is replaced by the next deploy. Everything in
`data/` survives it. Paths in `basil.yaml` resolve against one or the other and
never against the directory you happen to be standing in — see
[Configuration](../../guide/configuration.md#where-paths-resolve-the-two-anchors).

Site code writes to the data root. `basil.uploads_dir` (`<data_dir>/uploads`) is
always writable and needs no configuration; `basil.data_dir` names the root
itself, and any `security.allow_write` entry resolves there too. Files in the
uploads directory are served at `/__uploads/` and are **public** unless you list
the prefix in `auth.protected_paths`.

### The legacy layout

A plain directory containing `basil.yaml` still works exactly as before, with the
data root defaulting to that directory. `basil --dev` in a working copy needs no
site root and no `current` symlink.

## Starting the Server

```bash
basil                 # Production mode, uses ./basil.yaml (or ./current/basil.yaml)
basil --site PATH     # Serve the active release of a site root
basil --dev           # Development mode (HTTP on localhost)
basil --port 3000     # Override the configured port
basil --quiet         # Suppress request logs
```

Development mode (`--dev`) serves plain HTTP on localhost, disables response caching, injects live-reload into pages (the browser refreshes when you save a handler), shows detailed error pages, and enables the [dev tools](dev-tools.md).

### Config Resolution

Basil finds its configuration in this order:

1. `--config PATH` flag
2. `BASIL_CONFIG` environment variable
3. `./current/basil.yaml` — the active release, if the working directory is a site root
4. `./basil.yaml`
5. `~/.config/basil/basil.yaml`

`--site PATH` names a site root directly and reads the config from its active
release.

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
