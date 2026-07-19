---
name: verify
description: Build and run a Basil server against a scratch site to observe changes at the real HTTP surface.
---

# Verifying Basil at the HTTP surface

## Build

```bash
go build -o $SCRATCH/basil ./cmd/basil
```

## Minimal scratch site

```
verify-site/
  basil.yaml
  site/home/index.pars     # put pages in subdirs — a root site/index.pars
                           # is a catch-all handler and swallows 404s
```

Minimal page: `<!DOCTYPE html>` then `<html><body><h1>"Text"</h1></body></html>`
(Parsley: text content must be quoted).

## Dev mode (plain HTTP — easiest)

```bash
cd verify-site && $SCRATCH/basil --dev --port 8085
```

Dev mode serves HTTP on localhost; browser tools can open it directly.
Note: handler errors show the detailed *dev* error page, not the production
500 page; 404s show the real production 404 page (no root handler → 404).

## Production mode (needs TLS)

Production requires `https.auto: true` + email, or manual cert/key.
Self-signed works for curl (`-k`); the in-app browser refuses it, so
capture bodies with curl and screenshot them via a static server:

```bash
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 2 -nodes -subj "/CN=localhost"
```

```yaml
server:
  host: localhost
  port: 8443
  https: {auto: false, cert: ./certs/cert.pem, key: ./certs/key.pem}
site:
  path: ./site
```

```bash
curl -ks https://localhost:8443/whatever
```

## Gotchas

- A runtime error for 500-testing: page starting `let x = definitelyNotDefined`.
- Kill servers with `pkill -f "$SCRATCH/basil"` — shell job control (`%1`)
  doesn't survive across Bash tool calls, and a stale server holding the
  port makes the next launch die with "address already in use" while curl
  happily talks to the old binary.
- Server logs go to stderr; redirect to a file and grep for `[WARN]`/`[ERROR]`.
