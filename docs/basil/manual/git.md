---
id: man-bas-git
title: "Git Deploy"
system: basil
type: feature
name: git
created: 2026-07-12
version: 1.0.0-alpha.3
author: "@sam"
keywords:
  - git
  - deploy
  - push
  - api keys
  - clone
  - https
---

# Git Deploy

Basil can serve your site as a Git repository over HTTPS. Clone it, edit, `git push` — and the running server picks up the changes with no restart. Push-to-deploy with nothing but Git.

> **Basil-only.** The Git endpoint is part of the Basil server.

## Quick Start

**1. Enable it** in `basil.yaml`:

```yaml
git:
  enabled: true
  require_auth: true
```

**2. Create a user and API key** (see [Running Basil](running.md)):

```bash
basil users create              # role: editor or admin
basil apikey create             # → bsl_live_abc123… (save it!)
```

**3. Clone your live site:**

```bash
git clone https://anything:bsl_live_abc123...@yourserver.com/.git mysite
```

The username is ignored — only the API key matters.

**4. Push changes:**

```bash
cd mysite
# edit…
git add . && git commit -m "Update homepage"
git push
```

Basil reloads automatically on push.

## Authentication & Roles

Git access uses API keys via HTTP Basic Auth (key as the password).

| Operation | Required role |
|-----------|---------------|
| Clone / pull | any authenticated user |
| Push | `editor` or `admin` |

Revoke a compromised key with `basil apikey revoke <id>` — no need to touch user accounts.

## See Also

- [Authentication](authentication.md) — users, roles, and API keys
- [Running Basil](running.md) — the `users` and `apikey` CLI
- The [Git guide](https://github.com/sambeau/basil/blob/main/docs/guide/git.md) — hooks, workflows, and troubleshooting
