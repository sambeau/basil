---
title: Security
---

# Security

Can you put Basil on the internet? Yes. Here is what protects you, and what is still your job.

## How it works

Security works in layers. Parsley runs in a sandbox. Basil adds the web-facing defences on top. Neither asks you to remember a sanitising function.

## The Parsley sandbox

Inside a Basil server, Parsley code runs under a [security policy](manual/features/security.html) that the server sets, not the script. A script cannot raise its own permissions.

- **File access** — handlers can read the project directory. They cannot write anywhere until you whitelist a folder in `basil.yaml`.
- **SQL** — the `<SQL>` tag and the query operators parameterise every query and validate column and table names. You never build SQL strings by hand.
- **Commands** — external commands run without a shell, so shell injection cannot happen. A whitelist decides which binaries may run.
- **Data files** — PLN, Parsley's own data format, holds data only. Parsing it cannot run code, unlike `pickle` or `Marshal`.

## What Basil adds

- **HTTPS by default** — point a domain at the server and Basil fetches and renews Let's Encrypt certificates itself, and redirects HTTP to HTTPS. See [Deployment](basil/deployment.html).
- **Security headers** — Basil sets HSTS, `nosniff`, `X-Frame-Options: DENY`, and a strict referrer policy on every response.
- **Passkeys, not passwords** — login uses WebAuthn, so there are no password hashes to leak. Recovery codes cover lost devices. See [Authentication](basil/authentication.html).
- **CSRF protection** — Basil checks a per-session token on every mutating request to an authenticated route.
- **Rate limiting** — a token bucket per user or IP on API routes, plus server-side limits on things like verification emails.
- **Sessions** — signed cookies, with a server secret Basil generates and stores for you.
- **Secrets** — `basil.yaml` pulls secrets from environment variables with `!secret`, so nothing sensitive lives in the file.

## Caveats

Basil is young. It has had internal security review and fixes (the [Parsley security guide](https://github.com/sambeau/basil/blob/main/docs/parsley/security.md) lists them) but no independent audit. The defaults are safe, but if your handler takes user input and uses it to pick a file or a command, that is on you — as in any language. Keep the binary up to date.

## Should you put it behind a reverse proxy?

You do not have to. Basil faces the internet directly: it handles TLS, HTTP/2, compression, and redirects itself, and it reads `X-Forwarded-For` if a proxy sits in front.

Put it behind Caddy, nginx, or a cloud load balancer when:

- You already run one and want every site on the box managed the same way.
- You host several apps on one IP address.
- You want a WAF, DDoS protection, or edge caching from a provider.
- You run more than one Basil instance and need to balance between them.

Otherwise, one binary on one port is the simpler setup, and simpler is usually safer.
