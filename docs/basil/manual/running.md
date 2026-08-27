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
basil --init myapp        # Create a new site folder
cd myapp && basil --dev   # Run it in development mode
```

## Creating a Site

`basil --init FOLDER` creates the plain folder Basil has always been about:

```
myapp/
├── basil.yaml      Configuration
├── site/           Your pages (index.pars is the home page)
│   └── index.pars
└── public/         Static files (CSS, JS, images)
```

No flags are required. `--host` defaults to `localhost` (an explicit one is
accepted and validated); `--admin` is refused here, because a local folder has no
accounts. The folder runs immediately with `basil --dev` inside it, and
everything Basil writes as it runs — databases, caches, certificates, uploads —
lands beside your code, covered by the generated `.gitignore`.

If `git` is on the PATH, `--init` also makes the folder a repository on `main`
with the starter site committed and `core.hooksPath` pointed at the pre-commit
formatting hook. It is a quiet nicety, not a gate: `--no-git` opts out, and a
machine without `git` simply gets the plain folder, with no warning. The point is
that the folder is already clone-shaped on the day you decide to deploy it — see
[Graduating a local site to a server](https://github.com/sambeau/basil/blob/main/docs/guide/deployment.md#graduating-a-local-site-to-a-server).

### Creating a deployment server: `--server`

`basil --init FOLDER --server --host HOSTNAME --admin NAME` is the other half,
run on the machine that will *receive* deploys. It creates a **site root**:

```
myapp/
├── site.git/       Bare repository, served at /.git
├── releases/       One directory per deployed commit
│   └── <commit>/     basil.yaml, site/index.pars, public/
├── current ->      The active release
└── data/           Databases, certificates, logs, uploads
    └── uploads/      Durable site writes, served at /__uploads/
```

Both flags are required in this mode, and neither is guessed:

- `--host` is written to `server.host`, the site's **public hostname**: the name on
  the certificate, the WebAuthn relying-party id, the address people type. It is
  not a bind address — the listener uses `server.bind` (empty means all
  interfaces), so a site whose hostname points at a load balancer, a NAT, or a
  container host still starts. A public server refuses to start without a host,
  because an empty host tells the certificate manager to attempt issuance for any
  hostname a stranger asks for. Use `localhost` for a site you will only run with
  `--dev`.
- `--admin` names the first Basil account. It is **never** derived from `$USER`
  or `$SUDO_USER` — this command usually runs on a server, where the shell is
  `root` or a service account. With a terminal attached, `--init` prompts for it.

Server init commits the starter site to the release branch and deploys it as
release 1, so the server has something to serve from the moment it starts. It
creates the admin account and prints its API key **once**, installs a pre-commit
formatting hook and the receive hooks, and prints the `git clone` command for the
site. `--no-git` is refused in this mode: a machine that receives pushes cannot be
built without git.

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
[Configuration](https://github.com/sambeau/basil/blob/main/docs/guide/configuration.md#where-paths-resolve-the-two-anchors).

Site code writes to the data root. `basil.uploads_dir` (`<data_dir>/uploads`) is
always writable and needs no configuration; `basil.data_dir` names the root
itself, and any `security.allow_write` entry resolves there too. Files in the
uploads directory are served at `/__uploads/` and are **public** unless you list
the prefix in `auth.protected_paths`.

### The legacy layout

A plain directory containing `basil.yaml` still works exactly as before, with the
data root defaulting to that directory. `basil --dev` in a working copy needs no
site root and no `current` symlink. This is the layout plain `basil --init` writes
and the shape a clone of a deployed site has, which is why a local folder can
graduate to a server without being restructured.

## Starting the Server

```bash
basil                 # Production mode, uses ./basil.yaml (or ./current/basil.yaml)
basil --site PATH     # Serve the active release of a site root
basil --dev           # Development mode (HTTP on localhost)
basil --port 3000     # Override the configured port
basil --quiet         # Suppress request logs
```

Development mode (`--dev`) serves plain HTTP on localhost, disables response caching, injects live-reload into pages (the browser refreshes when you save a handler), shows detailed error pages, and enables the [dev tools](dev-tools.md).

Any edit — to the page you are looking at, or to a component five imports below it — shows up on the next request. Imported files are still cached, because importing is expensive and a page is mostly components, but each cached module is checked against the files it was built from and re-read when any of them changes. Nothing has to notice the edit for this to work; live reload just saves you pressing the refresh key. See [`dev.cache`](configuration.md#devcache) for the opt-out.

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
| `--init FOLDER` | Create a new Basil site folder |
| `--server` | With `--init`: build the server deploy topology (on the box) |
| `--no-git` | With `--init`: do not create a git repository |
| `--host HOSTNAME` | With `--init`: the site's public hostname (default `localhost`; required with `--server`) |
| `--admin NAME` | With `--init --server`: the first account's name |
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
| `SIGHUP` | Activate the release `current` points at — a full route/cache rebuild (in the legacy single-directory layout: reload scripts, clear cache, re-parse on next request) |

## See Also

- [Configuration](configuration.md) — everything in `basil.yaml`
- [Routing](routing.md) — how requests find handlers
- [Dev Tools](dev-tools.md) — what dev mode gives you
