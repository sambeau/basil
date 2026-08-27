# Configuration Reference

This is the canonical reference for every key in `basil.yaml`. All examples are verified against the 1.0 schema.

## Table of Contents

- [Overview](#overview)
- [Config File Resolution](#config-file-resolution)
- [One File, Many Machines](#one-file-many-machines)
- [Where Paths Resolve: the Two Anchors](#where-paths-resolve-the-two-anchors)
- [Operator-Owned Settings on a Site Root](#operator-owned-settings-on-a-site-root)
- [Environment Variables](#environment-variables)
- [Secrets](#secrets)
- [Complete Annotated Example](#complete-annotated-example)
- [Reference](#reference)
  - [server](#server)
  - [database](#database)
  - [site](#site)
  - [routes](#routes)
  - [static](#static)
  - [public_dir](#public_dir)
  - [auth](#auth)
  - [session](#session)
  - [security](#security)
  - [cors](#cors)
  - [compression](#compression)
  - [logging](#logging)
  - [dev](#dev)
  - [developers](#developers)
  - [meta](#meta)
- [Migrating from Pre-1.0](#migrating-from-pre-10)

---

## Overview

`basil.yaml` is the single configuration file for a Basil application. It controls the server, routing, database, authentication, security headers, CORS, compression, logging, and developer profiles.

Basil uses YAML with a few extensions:

- **Environment variable interpolation**: `${VAR}` and `${VAR:-default}`
- **Secret tagging**: `!secret` marks values as sensitive (hidden in DevTools)
- **Duration literals**: `5m`, `24h`, `30s` (parsed by Go's `time.ParseDuration`)

## Config File Resolution

Basil searches for a config file in this order:

1. **Explicit path** — `basil --config /path/to/basil.yaml`
2. **`BASIL_CONFIG` env var** — `BASIL_CONFIG=./custom.yaml basil`
3. **`./current/basil.yaml`** — the active release, if the working directory is a site root
4. **`./basil.yaml`** — current working directory
5. **`~/.config/basil/basil.yaml`** — XDG config directory

The first file found is used. If none is found, Basil exits with an error.

`basil --site /srv/mysite` points Basil at a site root directly, which is the
usual way to run a deployed site.

## One File, Many Machines

There is one `basil.yaml`, and on a deployed site it travels inside the release —
so the same versioned file serves the production server *and* every developer's
machine. That works, and stays readable, because the file describes **the site,
not the machine it happens to be running on**. Everything machine-shaped lives in
a layer of its own:

| Layer | Mechanism | Versioned? |
| --- | --- | --- |
| **Production truth** | top-level keys: `server.host`, `server.port`, `https`, … | yes |
| **Mode** | `--dev` — HTTP on localhost, live reload, edits picked up on the next request; the production listener is ignored | ambient |
| **Per-person** | `developers.<name>` profiles, applied with `-as <name>` | yes, deliberately |
| **Per-run** | CLI flags — `--port 3000`, `--quiet`, `--config` | no |
| **Secrets** | `!secret` values, resolved from the environment | no |

Read top to bottom, that is the whole discipline. The top level says what the
site is *in production*: `host: mysite.example.com`, port 443, HTTPS on. Nobody
edits those to suit their laptop, because `--dev` already ignores them — it serves
plain HTTP, binds `localhost` when `server.bind` is empty, and turns a production
port 443 into 8080. Somebody who needs a different port puts it under their own
name instead:

```yaml
developers:
  sam:
    port: 3001
    database:
      path: sam.db
    logging:
      level: debug
```

```bash
basil --dev -as sam          # --profile sam is the same flag, spelled out
```

Per-person settings are **versioned on purpose**. They cannot collide — each
person has their own key — and the whole team can see them, which is precisely
what makes "it works on my machine" something you can look up rather than guess
at.

### There is no local override file, on purpose

Basil has no `basil.local.yaml`, no gitignored overlay, no shadow config. It was
considered and turned down. An untracked file that silently changes what the
server does is invisible state — exactly the thing you cannot see when you are
helping someone whose site behaves differently from yours — and it cuts against a
folder you can read top to bottom. Every case it would serve already has a layer
above it: a mode (`--dev`), a person (`developers.<name>`), a run (a flag), or a
secret (`!secret` and an environment variable).

If you hit a per-person need that no `developers.<name>` field covers, the answer
is a new field there, not an escape hatch that overrides anything silently.

### Two backstops

The discipline is social, so sooner or later someone commits a top-level edit
they meant for their own machine. Two mechanisms turn that into a caught mistake
rather than an outage:
[operator-owned settings](#operator-owned-settings-on-a-site-root), which a
release simply cannot change, and
[the listener-change warning](#the-listener-change-warning), which speaks up in
the terminal that pushed it.

## Where Paths Resolve: the Two Anchors

Every path in `basil.yaml` resolves against one of two anchors, and **never**
against the process working directory. Starting Basil from a different
directory does not change where anything lives.

| Anchor | Holds | Keys |
| --- | --- | --- |
| **Release directory** — the site's code, replaced by every deploy | handlers, pages, static assets | `site.path`, `public_dir`, `routes[].handler`, `routes[].public_dir`, `static[].root`, `static[].file`, `error_pages` |
| **Data directory** — persistent state, untouched by a deploy | databases, certificates, logs, caches, uploads | `data_dir`, `database.path`, `https.cache_dir`, `https.cert`, `https.key`, `images.cache_dir`, `dev.log_database`, `logging.output`, `logging.parsley.output`, `security.allow_write` |

Absolute paths are used as given.

### Site-root layout

A deployed site is a **site root**, and `basil.yaml` ships inside the release:

```
/srv/mysite/                      ← point basil here: basil --site /srv/mysite
  site.git/                       bare repository, served at /.git
  releases/
    4f2a1c9…/                     one directory per deployed commit
  current -> releases/4f2a1c9…    the active release (the release directory)
  data/                           the data directory — no deploy touches this
    data.db
    .basil-auth.db
    certs/
    uploads/                      durable site writes, served at /__uploads/
```

`basil --init … --server` creates this layout, on the machine that receives
deploys. The release directory is resolved through `current` once, at startup.

### Legacy layout

A plain project directory with a `basil.yaml` in it still works exactly as
before: the release directory *and* the data directory are both the project
directory, so `basil --dev` in a working copy behaves as it always has. This is
the layout plain `basil --init mysite` writes, and the shape a clone of a
deployed site has.

### `data_dir`

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `data_dir` | string | `<site root>/data`, or the project directory in the legacy layout | Root for everything that must survive a deploy |

A relative `data_dir` resolves against the site root (or the project directory
in the legacy layout).

```yaml
data_dir: ./data
```

### Writing files from site code

`<data_dir>/uploads` is always writable — it is a convention, not a setting, so
site code has a durable place to write with no configuration at all. Anything a
handler writes into the release is destroyed by the next deploy, so any extra
`security.allow_write` entry resolves against the data directory too:

```yaml
security:
  allow_write:
    - ./exports          # <data_dir>/exports
```

Site code finds the durable location through the `basil` context object:

- `basil.data_dir` — the data directory
- `basil.uploads_dir` — `<data_dir>/uploads`
- `basil.uploads_url` — `/__uploads/`, the URL prefix that directory is served
  under (following the existing `/__p/` and `/__img/` pattern)

Files under `<data_dir>/uploads` are served at `/__uploads/` in the site-root
layout, so uploads never have to live inside `public_dir`. Directory listings
are not served, and symlinks that point outside the directory are refused.

**Everything in that directory is world-readable over HTTP**, exactly like
`public_dir`: `/__uploads/<name>` needs no session. Do not write anything there
that should not be public, or protect the prefix:

```yaml
auth:
  protected_paths:
    - path: /__uploads
```

A legacy single-directory project is not affected: `/__uploads/` is only
registered for a site root, so upgrading never publishes an existing `uploads/`
directory.

## Operator-Owned Settings on a Site Root

Because `basil.yaml` ships inside the release, a deploy can change configuration
— with a few exceptions. On a **site root** these belong to the operator:

| Setting | On a site root | Why |
| --- | --- | --- |
| `auth.enabled` | Forced `true` | Pushes are authorised by API keys in the auth database. A release cannot switch that off. |
| `data_dir` | Ignored; stays `<site root>/data` | The auth and deploy databases the deploy path itself runs on live under it, and every persistent path the running server uses was resolved against it. A release that moved it would strand all of them. |

Without the first, an entirely ordinary first push from a site built locally
would brick the server: a local `basil.yaml` has no `auth:` block, because a
laptop needs no accounts, and deploying it would remove the authentication the
next deploy has to pass. Recovery would mean a shell on the box.

Omitting the block is the normal case and is **silent**. Setting it to `false`
*explicitly*, or setting `data_dir`, earns a warning at every start and at every
deploy, naming what was ignored, so the config on disk and the running server
never disagree quietly:

```
warning: auth.enabled: false in this release's basil.yaml is ignored on a site root — pushes are authenticated; a release cannot switch that off
warning: data_dir in this release's basil.yaml is ignored on a site root — the data root is the operator's, and the databases the deploy path runs on live under it; it stays /srv/mysite/data
```

The `data_dir` warning names the key and the decision but never the release's
*value*. `basil.yaml` is environment-interpolated before it is read, so a
`data_dir: ${SOME_SECRET_PATH}` would otherwise print whatever that expanded
to into the server log on every start and every deploy. What an operator needs
is which key was ignored and which root is actually in use; both are there.

Two facts left `basil.yaml` entirely rather than being forced, because forcing a
*value* means nothing: **which branch publishes**, and **whether `/.git` is
served at all**. Both live in the bare repository, where only a shell on the box
can reach them:

```bash
git -C /srv/mysite/site.git symbolic-ref HEAD refs/heads/main   # the release branch
git -C /srv/mysite/site.git config basil.gitEnabled false       # stop serving /.git
```

The keys they replace — `deploy.branch` and `git.enabled` — are gone. A config
that still carries either loads fine and warns, naming the command above; it is
never an error, because a stale key in a file everyone pulls from must not stop
a server (or a `basil publish`) from working. The warning is printed by
`basil publish` too, so a clone does not carry it around unseen.

In the **legacy layout** nothing is forced and `data_dir` works exactly as
documented above: a plain project directory has no bare repository, so it has no
Git endpoint to protect and no accounts to require — a local `basil --dev`
server with neither is exactly right.

This is a narrow fence rather than a general rule about what a release may
change: it covers exactly the settings whose loss removes the way back in.
Hostname, port and TLS all stay deployable — renaming a site over Git is a
legitimate thing to do — and are gated instead.

### The listener-change warning

When a **pushed** release would change `server.host`, `server.port` or the shape
of the `https` block, relative to the release currently live **on a public
server**, the push says so in the terminal you typed it in:

```
remote: warning: this release changes how the live site is served:
remote: warning:   server.host: mysite.example.com → localhost
remote: warning:   server.port: 443 → 8080
remote: The change takes effect when the server restarts, not now. If it was not intended, revert it before then: git revert HEAD && git push.
```

The push still succeeds and the release still goes live — this is a warning, not
a gate. Listener settings bind at startup, so nothing actually moves until the
server restarts, which is exactly why the warning is worth having *now*: at push
time the mistake is one commit deep and one `git revert` from gone.

The check stays quiet where it would be noise:

- **A localhost site says nothing.** If the live release's `server.host` is
  `localhost`, `127.0.0.1`, `::1` or a `.local` name, its listener is a
  developer's own business.
- **`https` is compared by shape, not by field.** `https.auto` defaults to
  `true`, so a release that merely omits the `https:` block obtains its
  certificate exactly as the live one does and is correctly silent. What warns is
  TLS genuinely changing: `auto` turned off, or a manual certificate appearing,
  disappearing or moving.
- **A config that will not load produces no lines at all** — validation already
  owns broken configs, and reporting the same file twice in two voices helps
  nobody.

The commonest cause is graduation: a folder created by local `basil --init` says
`host: localhost`, `port: 8080`, and the sample above is exactly what publishing
it unedited looks like. Both lines are the warning doing its job — the host and
the port each moved. Set **both** top-level values to what the server serves on
(`host: mysite.example.com`, `port: 443`) before the first publish, and the push
is silent: there is nothing left to change. See
[Graduating a local site to a server](deployment.md#graduating-a-local-site-to-a-server).

## Environment Variables

Use `${VAR}` or `${VAR:-default}` anywhere in the config to interpolate environment variables:

```yaml
server:
  port: ${PORT:-8080}

database:
  path: ${DATABASE_PATH:-./data.db}
```

- `${VAR}` — replaced with the value of `VAR`, or empty string if unset
- `${VAR:-default}` — replaced with the value of `VAR`, or `default` if unset

## Secrets

The `!secret` YAML tag marks a value as sensitive. Secret values are hidden in DevTools (shown as `●●●●●●●●`).

```yaml
session:
  secret: !secret ${SESSION_SECRET}      # From env var, marked secret
  secret: !secret auto                   # Auto-generate a secure value
```

The `!secret` tag does **not** encrypt values — it only controls display in DevTools.

For session secrets, `!secret auto` generates a cryptographically secure random value on each server start. This is convenient for development but means sessions are invalidated on restart. For production with multiple instances or persistent sessions, use an explicit secret via an environment variable.

---

## Complete Annotated Example

```yaml
server:
  host: mysite.example.com      # Public hostname (certificate name, links). Required
                                # for a public server; not a bind address.
  bind: ""                      # Listener interface (default: "" = all interfaces)
  port: 8080                    # Listen port (default: 443)
  https:
    auto: true                  # Let's Encrypt auto-certificates
    email: admin@example.com    # ACME notification email (recommended with auto)
    cache_dir: ./certs          # Certificate cache directory
    # cert: ./cert.pem          # Manual certificate path (overrides auto)
    # key: ./key.pem            # Manual key path (overrides auto)
  proxy:
    trusted: false              # Trust X-Forwarded-* headers
    # trusted_ips:              # Restrict to specific proxy IPs
    #   - 10.0.0.0/8

database:
  path: ./data.db               # SQLite database file (relative to config)

site:
  path: ./site                  # Filesystem-based routing directory
  cache: 5m                     # Response cache TTL (0 = no cache)

# routes:                       # Explicit routes (mutually exclusive with site)
#   - path: /
#     handler: ./handlers/index.pars
#     cache: 5m
#   - path: /api/*
#     handler: ./handlers/api.pars
#     type: api

# static:                       # Static file routes
#   - path: /static/
#     root: ./public
#   - path: /favicon.ico
#     file: ./public/favicon.ico

public_dir: ./public            # Static files directory (rewritten to web URLs)

auth:
  enabled: true
  registration: closed          # "open" or "closed"
  session_ttl: 24h
  login_path: /login
  protected_paths:
    - /dashboard
    - path: /admin
      roles: [admin]

session:
  store: cookie                 # "cookie" or "sqlite"
  secret: !secret auto          # Encryption secret
  max_age: 24h                  # Session lifetime
  cookie_name: _basil_session
  # secure: true                # HTTPS only (default: true in production)
  http_only: true               # No JavaScript access
  same_site: Lax                # "Lax", "Strict", or "None"

security:
  hsts:
    enabled: true
    max_age: "31536000"         # 1 year
    include_subdomains: true
    preload: false
  content_type_options: nosniff
  frame_options: DENY
  xss_protection: "1; mode=block"
  referrer_policy: strict-origin-when-cross-origin
  # csp: "default-src 'self'"
  # permissions_policy: "camera=(), microphone=()"
  # allow_write:
  #   - ./data
  #   - ./uploads

cors:
  origins:
    - https://app.example.com
  methods: [GET, HEAD, POST]
  # headers: [Content-Type, Authorization]
  # expose: [X-Total-Count]
  credentials: false
  max_age: 86400                # Preflight cache (seconds)

compression:
  enabled: true
  level: default                # fastest, default, best, none
  min_size: 1024                # Minimum response size (bytes)
  zstd: false                   # Zstd compression

logging:
  level: info                   # debug, info, warn, error
  format: text                  # text or json
  output: stderr                # stderr, stdout, or file path
  # quiet: false                # Suppress request logs
  parsley:
    output: stderr              # Parsley log() output

# No git: section. The release branch and the /.git off-switch are the
# operator's, not the release's - see "Operator-Owned Settings on a Site
# Root" above. Both are set on the server, in the bare repository:
#   git -C /srv/mysite/site.git symbolic-ref HEAD refs/heads/main   # the release branch
#   git -C /srv/mysite/site.git config basil.gitEnabled false       # stop serving /.git

dev:
  # log_database: ./dev_logs.db
  log_max_size: 10MB
  log_truncate_pct: 25
  cache: false                  # Cache like production in dev mode (see the manual)

# Per-person settings, applied with: basil --dev -as sam
developers:
  sam:
    port: 3001
    database:
      path: sam.db
    # handlers: sam-handlers
    public_dir: ./sam-public
    logging:
      level: debug

meta:
  site_name: My App
  version: "1.0"
```

---

## Reference

### `server`

Core server settings.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `host` | string | `""` | Public hostname: the certificate name, the WebAuthn relying-party id, and the address people type. **Required** for a public server — a server with `https.auto: true` and no manual certificate refuses to start without it. Not a bind address. |
| `bind` | string | `""` (all interfaces) | Listener interface. Leave empty unless the server must be reachable on one interface only; `--dev` binds `localhost` when this is empty. |
| `port` | integer | `443` | Listen port (1–65535) |
| `https.auto` | boolean | `true` | Use Let's Encrypt auto-certificates |
| `https.email` | string | | ACME notification email (recommended with `auto: true`, not required — without it Let's Encrypt has no contact address for expiry and revocation notices) |
| `https.cache_dir` | string | `"certs"` | Certificate cache directory (relative to `data_dir`) |
| `https.cert` | string | | Manual certificate path (overrides `auto`) |
| `https.key` | string | | Manual key path (overrides `auto`) |
| `proxy.trusted` | boolean | `false` | Trust `X-Forwarded-*` headers from reverse proxies |
| `proxy.trusted_ips` | string[] | | Restrict proxy trust to specific IPs/CIDRs |

```yaml
server:
  host: localhost
  port: 8080
```

> **Note:** In dev mode (`basil --dev`), HTTPS validation is skipped and the default port is effectively overridden by the `--port` flag.

---

### `database`

SQLite database configuration.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `path` | string | `""` (no database) | Path to SQLite database file |

```yaml
database:
  path: ./data.db
```

The path is resolved relative to the config file directory. If the file doesn't exist, SQLite creates it automatically. An empty path (or omitting the section entirely) means no database is configured — `@DB` will not be available in handlers.

---

### `site`

Filesystem-based routing. When configured, Basil serves requests by finding the nearest `index.pars` (or `{foldername}.pars`) file walking back from the URL path.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `path` | string | `""` (disabled) | Directory for filesystem-based routing |
| `cache` | duration | `0` (no cache) | Response cache TTL for all site handlers |

```yaml
site:
  path: ./site
  cache: 5m
```

> **Important:** `site` and `routes` are mutually exclusive. Use `site` for filesystem routing or `routes` for explicit routing, not both.

---

### `routes`

Explicit route definitions. Each route maps a URL path to a Parsley handler script.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `path` | string | **(required)** | URL path pattern (supports `*` wildcard) |
| `handler` | string | **(required)** | Path to Parsley handler script |
| `auth` | string | `""` | `"required"`, `"optional"`, `"none"`, or empty |
| `roles` | string[] | | Required roles (used with `auth: required`) |
| `cache` | duration | `0` | Response cache TTL |
| `public_dir` | string | | Static files directory for this route |
| `type` | string | | Route type: `"api"` for API modules, empty for pages |

```yaml
routes:
  - path: /
    handler: ./handlers/index.pars
    cache: 5m
  - path: /api/*
    handler: ./handlers/api.pars
    type: api
    auth: optional
```

---

### `static`

Static file routes. Map URL paths to files or directories on disk.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `path` | string | **(required)** | URL path prefix |
| `root` | string | | Directory to serve (for directory serving) |
| `file` | string | | Single file to serve |

```yaml
static:
  - path: /static/
    root: ./public
  - path: /favicon.ico
    file: ./public/favicon.ico
```

Each entry must have either `root` or `file`, but not both.

---

### `public_dir`

Top-level public directory for static files. Files in this directory are served at the web root and are available for `publicUrl()` resolution in handlers.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `public_dir` | string | `"./public"` | Static files directory |

```yaml
public_dir: ./public
```

---

### `auth`

Authentication settings. Basil uses WebAuthn (passkeys) for passwordless authentication.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | boolean | `false` | Enable authentication. On a site root this is [operator-owned](#operator-owned-settings-on-a-site-root) and always on — deploys are authenticated by the API keys in its database. |
| `registration` | string | `"closed"` | `"open"` (anyone) or `"closed"` (invite only) |
| `session_ttl` | duration | `24h` | Auth session duration |
| `login_path` | string | `"/login"` | Redirect path for unauthenticated users |
| `protected_paths` | list | | URL prefixes requiring authentication |
| `email_verification.enabled` | boolean | `false` | Enable email verification |
| `email_verification.provider` | string | | `"mailgun"` or `"resend"` |
| `recovery.codes_enabled` | boolean | `true` | Enable recovery codes |
| `recovery.email_enabled` | boolean | `false` | Enable email recovery |

```yaml
auth:
  enabled: true
  registration: closed
  protected_paths:
    - /dashboard
    - path: /admin
      roles: [admin]
```

Protected paths support both simple strings and objects with role requirements.

---

### `session`

Session storage and cookie settings.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `store` | string | `"cookie"` | `"cookie"` or `"sqlite"` |
| `secret` | string | `!secret auto` | Encryption secret |
| `max_age` | duration | `24h` | Session lifetime |
| `cookie_name` | string | `"_basil_session"` | Session cookie name |
| `secure` | boolean | auto | HTTPS only (`true` in production, `false` in dev) |
| `http_only` | boolean | `true` | Block JavaScript access to cookie |
| `same_site` | string | `"Lax"` | `"Lax"`, `"Strict"`, or `"None"` |
| `table` | string | `"_sessions"` | SQLite table name (when `store: sqlite`) |
| `cleanup` | duration | `1h` | Expired session cleanup interval (when `store: sqlite`) |

```yaml
session:
  store: cookie
  secret: !secret auto
  max_age: 24h
  cookie_name: _basil_session
  http_only: true
  same_site: Lax
```

---

### `security`

Security headers added to every response.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `hsts.enabled` | boolean | `true` | Enable HSTS header |
| `hsts.max_age` | string | `"31536000"` | HSTS max-age in seconds |
| `hsts.include_subdomains` | boolean | `true` | Include subdomains |
| `hsts.preload` | boolean | `false` | Allow HSTS preload list submission |
| `content_type_options` | string | `"nosniff"` | X-Content-Type-Options |
| `frame_options` | string | `"DENY"` | X-Frame-Options |
| `xss_protection` | string | `"1; mode=block"` | X-XSS-Protection |
| `referrer_policy` | string | `"strict-origin-when-cross-origin"` | Referrer-Policy |
| `csp` | string | `""` | Content-Security-Policy |
| `permissions_policy` | string | `""` | Permissions-Policy |
| `allow_write` | string[] | `[]` | Directories where handlers can write files (relative paths resolve against `data_dir`) |

```yaml
security:
  hsts:
    enabled: true
    max_age: "31536000"
  content_type_options: nosniff
  frame_options: DENY
  allow_write:
    - ./data
    - ./uploads
```

---

### `cors`

Cross-Origin Resource Sharing. CORS is disabled by default (no origins configured).

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `origins` | string or string[] | `[]` (disabled) | Allowed origins (`"*"` or specific domains) |
| `methods` | string[] | `[GET, HEAD, POST]` | Allowed HTTP methods |
| `headers` | string[] | `[]` | Allowed request headers |
| `expose` | string[] | `[]` | Response headers exposed to JavaScript |
| `credentials` | boolean | `false` | Allow cookies and auth headers |
| `max_age` | integer | `86400` | Preflight cache duration in seconds |

```yaml
cors:
  origins:
    - https://app.example.com
  methods: [GET, POST, PUT, DELETE]
  headers: [Content-Type, Authorization]
  credentials: true
  max_age: 86400
```

> **Note:** Cannot use `origins: "*"` with `credentials: true` — browsers reject this combination.

See the [CORS Guide](./cors.md) for detailed examples and troubleshooting.

---

### `compression`

HTTP response compression.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | boolean | `true` | Enable gzip compression |
| `level` | string | `"default"` | `"fastest"`, `"default"`, `"best"`, or `"none"` |
| `min_size` | integer | `1024` | Minimum response size to compress (bytes) |
| `zstd` | boolean | `false` | Enable Zstd compression for supporting browsers |

```yaml
compression:
  enabled: true
  level: default
  min_size: 1024
```

---

### `logging`

Log output settings.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `level` | string | `"info"` | `"debug"`, `"info"`, `"warn"`, or `"error"` |
| `format` | string | `"text"` | `"text"` or `"json"` |
| `output` | string | `"stderr"` | `"stderr"`, `"stdout"`, or a file path |
| `quiet` | boolean | `false` | Suppress request logs |
| `parsley.output` | string | `"stderr"` | Output for Parsley `log()` calls |

```yaml
logging:
  level: info
  format: text
  output: stderr
  parsley:
    output: stderr
```

---

### `git`

The Git endpoint at `/.git/`, which serves the site's bare repository
(`<site root>/site.git`).

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | boolean | `true` | Off-switch only. The endpoint is active when the bare repository exists — there is nothing to turn *on* — and on a site root this is [operator-owned](#operator-owned-settings-on-a-site-root) and cannot be turned off from a release's config. A plain project directory has no bare repository and so no endpoint. |

Authentication is not configurable: pushes always require an API key from the
auth database, and Git over plain HTTP is refused outright, with a `--dev` server
answering localhost the only relaxation — decided in code, never from a config
file. (The old `git.require_auth` key is gone; a warning in a log nobody reads
was not a control.) See [Git over HTTPS](git.md#authentication).

---

### `dev`

Development tools settings. Only used when the `--dev` flag is enabled.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `log_database` | string | auto | Path to dev log database file |
| `log_max_size` | string | `"10MB"` | Maximum log database size |
| `log_truncate_pct` | integer | `25` | Percentage to delete when truncating |
| `cache` | boolean | `false` | Cache the way production does in dev mode: response and fragment caching on, imported modules trusted without revalidation. Off by default, so an edit anywhere shows up on the next request. |

```yaml
dev:
  log_max_size: 10MB
  log_truncate_pct: 25
  cache: false
```

---

### `developers`

Named profiles for per-person overrides, applied with `basil --dev -as <name>`
(`--profile <name>` is the same flag). This is where a setting that suits *your
machine* belongs, rather than the top level — see
[One File, Many Machines](#one-file-many-machines).

Each profile can override:

| Key | Type | Description |
|-----|------|-------------|
| `port` | integer | Override `server.port` |
| `database.path` | string | Override `database.path` |
| `handlers` | string | Override handlers directory for all routes |
| `public_dir` | string | Override `public_dir` |
| `logging` | object | Override logging settings |

```yaml
developers:
  sam:
    port: 3001
    database:
      path: sam.db
    public_dir: ./sam-public
    logging:
      level: debug
  alex:
    port: 3002
    database:
      path: alex.db
```

Only non-zero values override the base config. Omitted fields are inherited.

---

### `meta`

Custom metadata accessible in Parsley handlers as `basil.meta.*`. Can contain any YAML structure.

```yaml
meta:
  site_name: My Blog
  tagline: Thoughts on code
  features:
    comments: true
    dark_mode: false
```

Access in handlers:

```parsley
let {basil} = import @std/basil
basil.meta.site_name   // "My Blog"
```

---

## Migrating from Pre-1.0

The 1.0 release makes several breaking changes to the config schema for consistency. All YAML keys now use `snake_case`, and related settings are grouped into sections.

| Pre-1.0 | 1.0 |
|---------|-----|
| `sqlite: ./data.db` | `database:`<br>`  path: ./data.db` |
| `site: ./site` | `site:`<br>`  path: ./site` |
| `site_cache: 5m` | `site:`<br>`  cache: 5m` |
| `cors:`<br>`  maxAge: 86400` | `cors:`<br>`  max_age: 86400` |
| `developers:`<br>`  x:`<br>`    sqlite: x.db` | `developers:`<br>`  x:`<br>`    database:`<br>`      path: x.db` |
| `developers:`<br>`  x:`<br>`    static: ./dir` | `developers:`<br>`  x:`<br>`    public_dir: ./dir` |

**Go struct change (library users only):** `SessionConfig.HttpOnly` has been renamed to `SessionConfig.HTTPOnly` per Go initialism conventions. The YAML key `http_only` is unchanged.

These are clean breaks with no backward compatibility. If your config uses the old key names, YAML parsing will silently ignore them (for top-level scalars like `sqlite`) or fail with a type error (for `site` which changed from string to section). Update your config files to the new schema before upgrading.