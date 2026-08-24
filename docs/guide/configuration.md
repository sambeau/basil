# Configuration Reference

This is the canonical reference for every key in `basil.yaml`. All examples are verified against the 1.0 schema.

## Table of Contents

- [Overview](#overview)
- [Config File Resolution](#config-file-resolution)
- [Where Paths Resolve: the Two Anchors](#where-paths-resolve-the-two-anchors)
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
  - [git](#git)
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

`basil --init` creates this layout. The release directory is resolved through
`current` once, at startup.

### Legacy layout

A plain project directory with a `basil.yaml` in it still works exactly as
before: the release directory *and* the data directory are both the project
directory, so `basil --dev` in a working copy behaves as it always has.

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

Anything a handler writes into the release is destroyed by the next deploy, so
`security.allow_write` resolves against the data directory:

```yaml
security:
  allow_write:
    - ./uploads          # <data_dir>/uploads
```

Site code finds the durable location through the `basil` context object:

- `basil.data_dir` — the data directory
- `basil.uploads_dir` — `<data_dir>/uploads`
- `basil.uploads_url` — `/__uploads/`, the URL prefix that directory is served
  under (following the existing `/__p/` and `/__img/` pattern)

Files under `<data_dir>/uploads` are served at `/__uploads/`, so uploads never
have to live inside `public_dir`. Directory listings are not served.

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
  host: localhost               # Bind address (default: "" = all interfaces)
  port: 8080                    # Listen port (default: 443)
  https:
    auto: true                  # Let's Encrypt auto-certificates
    email: admin@example.com    # ACME notification email (required with auto)
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

git:
  enabled: false
  require_auth: true

dev:
  # log_database: ./dev_logs.db
  log_max_size: 10MB
  log_truncate_pct: 25
  cache: false                  # Response caching in dev mode

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
| `host` | string | `""` (all interfaces) | Bind address |
| `port` | integer | `443` | Listen port (1–65535) |
| `https.auto` | boolean | `true` | Use Let's Encrypt auto-certificates |
| `https.email` | string | | ACME notification email (required when `auto: true`) |
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
| `enabled` | boolean | `false` | Enable authentication |
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

Git HTTP server. When enabled, serves the project as a Git repository at `/.git/`.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | boolean | `false` | Enable Git endpoint |
| `require_auth` | boolean | `true` | Require API key authentication |

```yaml
git:
  enabled: true
  require_auth: true
```

---

### `dev`

Development tools settings. Only used when the `--dev` flag is enabled.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `log_database` | string | auto | Path to dev log database file |
| `log_max_size` | string | `"10MB"` | Maximum log database size |
| `log_truncate_pct` | integer | `25` | Percentage to delete when truncating |
| `cache` | boolean | `false` | Enable response caching in dev mode |

```yaml
dev:
  log_max_size: 10MB
  log_truncate_pct: 25
  cache: false
```

---

### `developers`

Named developer profiles for per-developer overrides. Use with `basil --dev --profile <name>`.

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