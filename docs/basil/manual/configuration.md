---
id: man-bas-configuration
title: "Configuration"
system: basil
type: feature
name: configuration
created: 2026-07-12
version: 1.0.0-alpha.3
author: "@sam"
keywords:
  - basil.yaml
  - config
  - server
  - session
  - database
  - logging
  - compression
  - meta
  - developers
  - secrets
---

# Configuration

Basil is configured with a single `basil.yaml` file. Everything is optional — a new project runs on defaults — and relative paths are resolved from the directory containing the config file.

```yaml
server:
  host: localhost
  port: 8080

site:
  path: ./site

public_dir: ./public
```

## server

```yaml
server:
  host: localhost
  port: 8080
```

Use `--port` to override the port at startup, or a [developer profile](#developers) for per-person overrides.

## site

Filesystem-based routing: files under `path` are served at their URL path. See [Routing](routing.md).

```yaml
site:
  path: ./site
  cache: 5m        # Response cache TTL (0 = no cache)
```

## routes & static

Explicit routing, as an alternative (or addition) to site mode:

```yaml
routes:
  - path: /
    handler: ./handlers/index.pars
  - path: /dashboard
    handler: ./handlers/dashboard.pars
    auth: required

static:
  - path: /static/
    root: ./public
```

## error_pages

Custom error pages, replacing Basil's built-in ones:

```yaml
error_pages:
  404: ./errors/404.pars
  403: ./errors/403.pars
  500: ./errors/500.pars
```

Each entry maps an HTTP error status code (400–599) to a Parsley template.
The template receives an `error` dictionary (`.code`, `.message`) and the
usual `basil` dictionary (`.version`, `.dev`), and should return a full HTML
page. If a custom page is missing or fails to render, Basil logs the problem
and serves its built-in page instead — a broken error page never takes down
error handling.

Basil ships built-in pages for `404`, `403`, and `500`; any other code falls
back to the 500 page unless you supply your own. The `403` page covers both
role failures on a protected path and CSRF validation failures.

In dev mode, a failing handler still shows the detailed [dev error
page](dev-tools.md); custom pages replace the generic production pages (and
the 404/403 pages, which are the same in both modes).

**API requests get JSON, not HTML.** A request that looks like an API call —
`Accept: application/json`, a JSON `Content-Type`, or a path under `/api/` —
receives a JSON error body instead of any error page:

```json
{ "error": { "code": "HTTP-404", "message": "Not Found" } }
```

In dev mode a `500` also carries a `details` field with the underlying error;
in production the details go to the logs only. Content negotiation happens
before error pages are consulted, so a custom `error_pages` template never
gets served to a JSON client.

## public_dir

```yaml
public_dir: ./public
```

Static files, served at the web root. Parsley paths under this directory are automatically rewritten to URLs in HTML output: `@./public/images/foo.png` becomes `/images/foo.png`.

## database

```yaml
database:
  path: ./db/data.db
```

The built-in SQLite database, available in handlers as `@DB`. See [Database](database.md).

## session

```yaml
session:
  store: cookie              # "cookie" (default) or "sqlite"
  secret: !secret auto       # Auto-generate (recommended)
  max_age: 24h
  cookie_name: "_basil_session"
```

Use `secret: !secret ${SESSION_SECRET}` to read the secret from an environment variable — needed when running multiple instances behind a load balancer.

## auth

```yaml
auth:
  enabled: true
  registration: open         # "open" or "closed"
  session_ttl: 24h
  login_path: /login
  protected_paths:
    - /dashboard
    - path: /admin
      roles: [admin]
```

Passkey authentication. See [Authentication](authentication.md).

## security

```yaml
security:
  allow_write:
    - ./data
    - ./uploads
```

By default handlers can read files but not write. `allow_write` whitelists directories handlers may write to.

## git

```yaml
# git:
#   enabled: false     # off-switch only; ignored on a site root
```

The Git endpoint at `/.git/` is live whenever the site has a bare repository, so
there is nothing to turn on. `enabled` is an off-switch only, and on a site root
it is operator-owned: a release cannot disable the endpoint it arrived through.
Authentication is not configurable — pushes always need an API key. See
[Git Deploy](git.md) and
[Operator-owned settings](../../guide/configuration.md#operator-owned-settings-on-a-site-root).

## images

```yaml
images:
  cache_dir: ./cache/images
  max_width: 4096
  max_height: 4096
```

Image transformation cache and limits. See [Images](images.md).

## compression

```yaml
compression:
  enabled: true      # gzip (default: true)
  level: default     # fastest | default | best | none
  min_size: 1024     # Minimum response size to compress (bytes)
  zstd: false        # Zstd support (experimental)
```

## cors

```yaml
cors:
  allowed_origins: ["https://app.example.com"]
  allowed_methods: [GET, POST]
  max_age: 1h
```

Cross-origin resource sharing for APIs. See [Deployment](deployment.md).

## logging

```yaml
logging:
  level: info                    # debug | info | warn | error
  format: text                   # text | json
  output: ./logs/basil.log       # or "stderr"
  parsley:
    output: ./logs/parsley.log   # log() output from handlers
```

## dev

```yaml
dev:
  log_database: ./logs/dev_logs.db   # Dev log panel storage
  log_max_size: 10MB
  log_truncate_pct: 25
  cache: false                       # Response caching in dev mode
```

See [Dev Tools](dev-tools.md).

## meta

Free-form site metadata, available in every handler as `meta.*`:

```yaml
meta:
  name: "My Awesome Blog"
  tagline: "Thoughts on code and coffee"
  social:
    github: "myblog"
```

```parsley
<title>meta.name</title>
<p>meta.tagline</p>
```

## developers

Per-developer overrides, applied with `basil --dev --profile NAME`:

```yaml
developers:
  sam:
    port: 3001
    database:
      path: sam.db
```

## Secrets

The `!secret` YAML tag marks values as sensitive — they're masked in the dev tools:

```yaml
session:
  secret: !secret auto                  # Generate and persist automatically
  # secret: !secret ${SESSION_SECRET}   # From an environment variable
```

## See Also

- [Running Basil](running.md) — config resolution order
- The [example configuration](https://github.com/sambeau/basil/blob/main/docs/guide/configuration-example.yaml) with every option commented
