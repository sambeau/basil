---
id: man-bas-deployment
title: "Deployment"
system: basil
type: feature
name: deployment
created: 2026-07-12
version: 1.0.0-alpha.3
author: "@sam"
keywords:
  - deployment
  - production
  - https
  - tls
  - lets encrypt
  - security headers
  - compression
  - cors
  - caching
---

# Deployment

Basil is a single binary, so deploying is mostly: put the binary and your project on a server, point DNS at it, and turn on HTTPS. Basil handles the certificates.

## HTTPS

Three modes, depending on where you're running:

**Development** — `basil --dev` serves plain HTTP on localhost; no certificates involved.

**Self-signed (local HTTPS testing):**

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem \
  -days 365 -nodes -subj "/CN=localhost"
```

```yaml
server:
  host: localhost
  port: 443
https:
  cert: ./cert.pem
  key: ./key.pem
```

**Production (automatic Let's Encrypt):**

```yaml
server:
  host: example.com          # a real domain pointing at this server
  port: 443
https:
  auto: true
  email: admin@example.com   # Let's Encrypt expiry notifications
  cache_dir: ./certs
```

Requirements: public DNS pointing at the server, port 80 open (ACME challenges + HTTP→HTTPS redirect), port 443 for traffic. Certificates renew themselves; HTTP/2 is on by default.

## Security Headers

Basil sets safe defaults on every response — `Strict-Transport-Security` (1 year), `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin` — customisable under the `security:` config section.

## File-System Security

Handlers can read files but not write, unless you whitelist directories:

```yaml
security:
  allow_write:
    - ./data
    - ./uploads
```

## Compression

gzip is on by default; tune or disable it:

```yaml
compression:
  enabled: true
  level: default     # fastest | default | best | none
  min_size: 1024
```

## Response Caching

Production mode caches responses. In site mode, set a TTL:

```yaml
site:
  path: ./site
  cache: 5m
```

Only successful `GET` responses are cached, keyed on path and query string.
[Parts](parts.md) requests are never cached and never served from the cache,
so interactive Parts keep working inside a cached page — only the page's
initial render is frozen. See [Parts and Response Caching](parts-guide.md#parts-and-response-caching).

## CORS

If browsers on other origins call your API:

```yaml
cors:
  allowed_origins: ["https://app.example.com"]
  allowed_methods: [GET, POST, PUT, DELETE]
  allowed_headers: [Content-Type, Authorization]
  max_age: 1h
```

See the [CORS guide](https://github.com/sambeau/basil/blob/main/docs/guide/cors.md) for preflight details and debugging.

## Updating a Live Site

With the [Git server](git.md) enabled, deploying an update is `git push` — Basil reloads handlers automatically. For binary upgrades, replace the binary and restart; `SIGHUP` reloads scripts without a restart.

## Sessions Across Instances

Running more than one instance behind a load balancer? Give them a shared session secret:

```yaml
session:
  secret: !secret ${SESSION_SECRET}
```

## See Also

- [Configuration](configuration.md) — every section referenced above
- [Git Deploy](git.md) — push-to-deploy
- [Running Basil](running.md) — signals and profiles
