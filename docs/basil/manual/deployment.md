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

Production mode always serves HTTPS. Without an `https:` section, Basil refuses to start with `HTTPS requires either auto: true or cert/key paths`. Development mode (`basil --dev`) serves plain HTTP on localhost and needs no certificates.

You have three ways to get a certificate: let Basil fetch one from Let's Encrypt (the usual choice), supply your own files, or make a self-signed one for local testing.

### Automatic certificates with Let's Encrypt

[Let's Encrypt](https://letsencrypt.org) is a free, automated certificate authority. You do not sign up, pay, or install a client: Basil talks to it directly using the ACME protocol, proves it controls your domain, receives a certificate, and renews it before it expires. All you provide is a domain and an email address.

#### Before you start

1. **A domain name** you control, e.g. `example.com`.
2. **DNS pointing at the server.** Add an `A` record (and an `AAAA` record if you have IPv6) for your domain to the server's public IP. Wait until `dig example.com` or `nslookup example.com` returns that IP from outside the server. Let's Encrypt looks your domain up on the public internet, so private or not-yet-propagated DNS will fail.
3. **Ports 80 and 443 open** to the internet in any firewall or cloud security group. Basil listens on both: 443 for your site, and 80 for a small second server that answers Let's Encrypt's verification requests and redirects everything else to HTTPS. Opening the firewall is your job; Basil only binds the ports. Port 80 is always `:80`, whatever `server.port` is set to. If it is blocked, Let's Encrypt can usually still verify you through port 443, but visitors who type `example.com` without `https://` get a connection error instead of a redirect.
4. **Permission to bind those ports.** On Linux, ports below 1024 need root or the `CAP_NET_BIND_SERVICE` capability. The cleanest option is to grant the capability to the binary once, then run Basil as an ordinary user:

   ```bash
   sudo setcap 'cap_net_bind_service=+ep' /usr/local/bin/basil
   ```

#### Configuration

```yaml
server:
  host: example.com          # the domain the certificate is for
  port: 443
https:
  auto: true
  email: admin@example.com   # expiry and problem notifications from Let's Encrypt
  cache_dir: ./certs         # where certificates are stored, relative to data_dir
```

| Key | Required | What it does |
|---|---|---|
| `server.host` | yes | The domain to request a certificate for. Basil only answers certificate requests for this exact name, and **refuses to start without it** — an empty host would let anyone trigger issuance for any name they put in SNI. `--dev` and a manual `https.cert`/`https.key` are the exceptions. |
| `https.auto` | yes | Turn on Let's Encrypt. |
| `https.email` | recommended | Let's Encrypt emails this address before a certificate expires and if it has to revoke one. Optional, but there is no other way to hear about problems. |
| `https.cache_dir` | no | Directory for the certificate, private key, and Let's Encrypt account key. Relative to the site's data directory; defaults to `<data_dir>/certs`. It never depended on the working directory since FEAT-152 — before that, it did. |

Setting `auto: true` accepts the [Let's Encrypt Subscriber Agreement](https://letsencrypt.org/repository/) on your behalf.

#### What happens on first start

Start Basil in production mode:

```bash
basil
```

The log shows `automatic TLS enabled via Let's Encrypt (cache: …)` and Basil listens on 443 and 80. It then obtains the certificate straight away, without waiting for a visitor:

1. Basil creates a Let's Encrypt account (keyed to your email) and stores the account key in `cache_dir`.
2. Let's Encrypt asks Basil to prove it controls `example.com`. Basil answers the challenge itself — either over port 80 (HTTP-01) or inside the TLS handshake on 443 (TLS-ALPN-01), whichever Let's Encrypt tries first.
3. Let's Encrypt issues a certificate. Basil stores it in `cache_dir` and uses it for every request from then on.

Either `TLS certificate ready for example.com` or a failure naming the two usual
suspects — DNS and port 80 — appears in the log within a few seconds. A failure is
not fatal: the server keeps running and tries again on the first HTTPS request.
Check it yourself with:

```bash
curl -I https://example.com
```

You want `HTTP/2 200` and no certificate warning. If `curl` complains, see [Troubleshooting](#troubleshooting) below.

#### Renewal

Let's Encrypt certificates last 90 days. Basil checks the certificate on each request and renews it automatically when it is within 30 days of expiry, using the same challenge as before. There is no cron job to set up and no restart needed. The one thing renewal needs is that ports 80 and 443 stay reachable from the internet — if you later close port 80 behind a firewall, renewal can still succeed over 443, but keep 80 open to be safe.

#### The `certs` directory

`cache_dir` holds your private key. Treat it accordingly:

- It lives in the site's data directory (`<data_dir>/certs` by default), which is outside the release and outside the repository, so a deploy never replaces it and there is nothing to gitignore.
- Make it readable only by the user Basil runs as: `chmod 700 certs`.
- Keep it between deploys and restarts. If you delete it, Basil requests a fresh certificate on the next start, which counts against Let's Encrypt's rate limits.
- Back it up with the rest of the site if you want restarts on a new machine to be instant, but a lost `certs` directory is not a disaster — Basil simply fetches a new certificate.

#### Rate limits

Let's Encrypt allows [5 certificates per week for the same set of names](https://letsencrypt.org/docs/rate-limits/) and a handful of failed attempts per hour. Normal use never approaches this. You can hit it by repeatedly deleting `certs/` while debugging, or by running several copies of Basil for the same domain, each with its own `cache_dir`. If you do, the error mentions `rateLimited` and you have to wait up to a week. Debug DNS and firewall problems with `curl` *before* pointing Basil at a domain, not after.

#### `www` and other names

Basil requests a certificate for `server.host` and nothing else. A certificate for `example.com` does not cover `www.example.com`. Pick one name as canonical, set it as `server.host`, and send the other to it at the DNS level: a `CNAME` will not help on its own (it still resolves to this server, which will refuse the name), so use your DNS provider's redirect feature, or handle the second name on a reverse proxy in front of Basil.

If `server.host` is empty, Basil will request a certificate for *any* hostname that reaches it. Do not run like that on the internet — anyone pointing a domain at your IP could trigger requests against your rate limits. Always set `server.host` in production.

### Bringing your own certificate

If you already have a certificate — from a corporate CA, a commercial provider, or a tool like `certbot` or `acme.sh` you run for other services — point Basil at the files:

```yaml
server:
  host: example.com
  port: 443
https:
  cert: /etc/ssl/example.com/fullchain.pem
  key: /etc/ssl/example.com/privkey.pem
```

`cert` should be the full chain (leaf plus intermediates), PEM-encoded. When `cert` and `key` are set, `auto` is ignored. Basil reads the files once at startup and does not watch them: after renewing them, restart Basil. (`SIGHUP` reloads scripts, not certificates.)

### Self-signed certificates for local HTTPS

To test production mode on your own machine — for example to check passkeys, which need HTTPS — make a throwaway certificate:

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

Your browser will warn that the certificate is untrusted; click through, or add it to your system's trust store. Never use a self-signed certificate on a public site.

### Behind a reverse proxy

Basil can sit behind Caddy, nginx, or a cloud load balancer that terminates TLS — see [Security](https://herbaceous.net/security.html) for when that makes sense. Two things to know:

- Basil still needs a certificate, because production mode is HTTPS-only. Use a self-signed one (as above) or an origin certificate from your provider; the proxy connects to Basil over HTTPS on an internal port and does not need to trust it.
- Basil reads `X-Forwarded-For` and `X-Real-IP`, so rate limiting and logs see the real client address. Make sure the proxy sets them.

Leave `auto: true` off in this setup: the proxy holds the public certificate, and Let's Encrypt's challenges would never reach Basil.

### Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `HTTPS requires either auto: true or cert/key paths` on start | No `https:` section | Add one, or run with `--dev` for local work |
| `bind: permission denied` on start | Not allowed to open port 80 or 443 | `setcap` as above, or run as root |
| `bind: address already in use` | Another web server (Apache, nginx, a previous Basil) holds the port | Stop it, or put Basil on another port behind it |
| HTTPS works but `http://example.com` does not redirect | Something else holds port 80, so Basil's redirect server failed — this is logged as `HTTP redirect server error` and is not fatal | Stop whatever owns port 80, then restart Basil |
| First request hangs, then fails with a TLS error | Let's Encrypt could not reach the server | Check DNS resolves to this IP from outside; check ports 80 and 443 are open in every firewall |
| Log shows `acme: ... urn:ietf:params:acme:error:dns` | DNS not propagated or pointing elsewhere | Wait for propagation; confirm with `dig` |
| Log shows `urn:ietf:params:acme:error:rateLimited` | Too many certificates requested recently | Stop deleting `certs/`; wait up to a week |
| `curl: (60) SSL certificate problem` | Hostname in the URL does not match `server.host` | Use the exact configured name; add `www` redirect at DNS |
| Works for `example.com`, not `www.example.com` | Certificate is for one name only | See [`www` and other names](#www-and-other-names) |

Basil logs ACME errors to standard error. Run it in the foreground while setting up so you can see them.

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
