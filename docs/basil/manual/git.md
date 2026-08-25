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

Basil's server holds a **bare Git repository** you push to, served over HTTPS at `/.git`. Clone it, edit, `git push` to share with the team — and move the **release branch** (`live`) to publish, which checks the release and makes it live. Push-to-deploy with nothing but Git.

> **Basil-only.** The Git endpoint is part of the Basil server. It is on whenever the site has a `site.git` repository, which `basil --init` creates — there is nothing to enable.

## Quick Start

**1. Create a user and API key** (see [Running Basil](running.md)):

```bash
basil users create --name Sam --email sam@example.com --role editor
basil apikey create --user usr_abc123... --name "MacBook Git"   # → bsl_live_abc123… (save it!)
```

**2. Clone your live site** (the key is the password; the username selects a stored credential and is ignored by Basil):

```bash
git clone https://sam@yourserver.com/.git mysite
```

**3. Share work:**

```bash
cd mysite
# edit…
git add . && git commit -m "Update homepage"
git push                        # stored on the server, published to nobody
```

**4. Publish:**

```bash
git push origin live            # the release branch moves; the release is checked, then goes live
```

A push that moves the release branch runs the deploy on the server; its output reaches your terminal as `remote:` lines, and a release that fails validation is rejected with the live site left unchanged.

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
