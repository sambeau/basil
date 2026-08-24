---
id: FEAT-152
title: "Site layout: separate code from persistent state"
status: draft
priority: high
created: 2026-08-24
author: "@sambeau / @claude"
---

# FEAT-152: Site layout — separate code from persistent state

## Summary

Split the single `BaseDir` anchor into two: a **release directory** (the site's code,
replaced on every deploy) and a **data root** (everything that must survive a deploy).
Introduce the site-root layout that Git deploy needs, and fix the paths that currently
resolve against the process working directory instead of the project.

This is foundational. FEAT-153, FEAT-154 and FEAT-155 all depend on it, and it is the
only one of the four with a wide blast radius.

## User Story

As a Basil operator, I want my database, certificates, logs and uploads to be untouched
when I deploy new code, so that publishing a site update can never destroy the data the
site is built on.

## Motivation

`config.BaseDir` currently means "the directory containing `basil.yaml`", and everything
resolves against it — handlers and databases alike. Under `DESIGN-git-deploy.md` the code
directory is replaced wholesale on each deploy, so anything living inside it is destroyed
by the next release.

Two items make this more than housekeeping:

- **The auth database** (`<BaseDir>/.basil-auth.db`) holds the API keys people deploy
  *with*. Losing it locks everyone out of the mechanism that would restore it.
- **TLS certificates** are rate-limited by Let's Encrypt on re-issue, so they are not
  merely regenerable.

Separately, an audit of `server/config/load.go` shows several paths are never resolved
against `BaseDir` at all and therefore land relative to the process working directory —
including `https.cache_dir`, which defaults to `certs`. Where a site's certificates live
today depends on where the operator happened to be standing when they started the server.

## Acceptance Criteria

### Layout

- [ ] A **site root** contains: `site.git/`, `releases/`, `current` (symlink), `data/`
- [ ] `current` points at the active release directory under `releases/`
- [ ] `basil` is pointed at a site root and resolves the active release through `current`
- [ ] The legacy single-directory layout is still accepted, with `data/` defaulting to the
      project directory, so `basil --dev` in a working copy keeps working unchanged

### Public-server requirements

- [ ] A public server **refuses to start** without `server.host` (decided, @sam
      2026-08-24). `hostPolicy()` currently returns `nil` when it is empty
      (`server.go:1168`), telling `autocert` to attempt issuance for any hostname supplied
      in SNI — a way to exhaust the site's Let's Encrypt rate limit from outside
- [ ] `--dev` and a manually configured `tls_cert`/`tls_key` are the exceptions
- [ ] The error names the fix, rather than failing on the first handshake

### Config anchors

- [ ] `config.BaseDir` is replaced by two explicit anchors: `ReleaseDir` and `DataDir`
- [ ] `data_dir` config key, default `<site root>/data` — one of only three settings this
      programme adds (see `reports/GIT-DEPLOY-DEFAULTS-REVIEW-2026-08-24.md`)
- [ ] **Every** persistent path resolves against `DataDir`: `database.path`,
      the auth database, `https.cache_dir`, `logging.output`, `logging.parsley.output`,
      `dev.log_database`, `images.cache_dir`
- [ ] **Every** code path resolves against `ReleaseDir`: `site.path`, `public_dir`,
      `routes[].handler`, `routes[].public_dir`, `static[].root`, `static[].file`,
      `error_pages`, `security.allow_write`
- [ ] No configured path resolves against the process working directory
- [ ] Starting Basil from any directory produces identical path resolution

### Site-written files

- [ ] Site code has a durable location to write to that a deploy will not delete
- [ ] That location is discoverable from Parsley (see Design Decisions)
- [ ] An uploads directory under `DataDir` is servable over HTTP without living inside
      `public_dir`

### `basil --init`

- [ ] Produces the site-root layout, including `data/` and a bare `site.git/`
- [ ] `.gitignore` no longer needs to list runtime state, because state is no longer
      inside the repository at all
- [ ] Creates the first admin user and prints an API key **once**, only on a fresh init,
      never overwriting existing credentials
- [ ] **Commits the starter site to the release branch and deploys it as release 1**, so
      the server can serve, obtain a certificate and be cloned from immediately. Without
      this a fresh server deadlocks — see `DESIGN-git-deploy.md` §5.1.1
- [ ] Accepts `--host <hostname>`, writing it to `server.host`, so no config edit is needed
      between `--init` and a working push
- [ ] Accepts `--admin <name>`. **Required** for a non-interactive init; prompts when a
      terminal is attached. **Never derived from `$USER` or `$SUDO_USER`** — `--init` runs
      on the server, where the shell is usually `root` or a service account, so the
      environment is wrong exactly where it matters. Warn (but accept) if the name given is
      `root`
- [ ] When run as uid 0, hands the created tree to the account that will run the server:
      `chown` to `$SUDO_USER` if set and report it, otherwise warn and print the exact
      `chown` command. Without this, `sudo basil --init` followed by running Basil as an
      ordinary user fails every write — database, logs, certificate cache and every deploy
- [ ] Installs a pre-commit formatting hook in the repository it created
- [ ] Produces a site that deploys with **no configuration step at all** — no `basil.yaml`
      edit is required between `--init` and a working push
- [ ] Prints the layout it created, as it does today
- [ ] Prints the **exact clone command**, with the hostname and account name already in it,
      so nobody has to reason about Git URL usernames on day one

### Migration

- [ ] None required. There is no installed base (see `DESIGN-git-deploy.md` §6.5)

## Design Decisions

- **Two anchors, not a prefix scheme.** `ReleaseDir` and `DataDir` are separate fields on
  `Config`, so every call site has to say which it means. A single "root" with
  conventional subpaths would let state leak back into the release by accident.

- **`basil.yaml` ships inside the release.** Config changes are versioned and roll back
  with the code. The one consequence to handle: server settings (port, TLS) become
  deployable, so a release can in principle change the listener — see Open Questions.

- **Legacy layout stays supported.** `basil --dev` run inside a git working copy must
  keep working with no site root and no `current` symlink; otherwise local development
  becomes ceremony. In that mode `DataDir` defaults to the project directory, which is
  exactly today's behaviour.

- **The Parsley-facing write path is a value, not a new builtin.** Expose the data root
  through the existing `basil` context object (`basil.data_dir`) rather than adding
  language surface. Writes remain governed by `security.allow_write`.

- **Uploads are served, not linked.** A configured uploads directory is served under a
  URL prefix, reusing the pattern already established by `/__p/` and `/__img/`. Symlinking
  `public/uploads` into the data root would work but reintroduces a path inside the
  release that must not be replaced.

## Technical Context

### Files

| File | Change |
| --- | --- |
| `server/config/config.go` | `BaseDir` → `ReleaseDir` + `DataDir`; add `data_dir` key |
| `server/config/load.go:56-114` | The single choke point for path resolution. Split the list by anchor |
| `server/config/load.go:418-442` | Secondary resolution for handlers/static — same split |
| `server/server.go:265,442,499,576` | `BaseDir` call sites |
| `server/server.go:1131-1145` | `autocert.DirCache(cacheDir)` — currently unresolved; anchor to `DataDir` |
| `server/auth/database.go:146-172` | `OpenDB(basePath)` — callers pass `DataDir` |
| `server/devlog.go:65` | Dev log database path |
| `server/images/cache.go` | Cache dir supplied already resolved |
| `server/search.go:341-357` | Search index resolves against `env.RootPath`; decide anchor |
| `cmd/basil/init.go` | New layout |
| `cmd/basil/main.go` | `--site`, `--host`, `--admin` flags |
| `server/server.go:1168` | `hostPolicy()` — refuse an empty host on a public server |

### Path audit

Currently resolved against `BaseDir` (`load.go`): `static[].root`, `static[].file`,
`routes[].handler`, `routes[].public_dir`, `database.path`, `site.path`, `public_dir`,
`error_pages`, `security.allow_write`.

**Not resolved at all** — these land relative to the process working directory and must be
fixed: `https.cache_dir`, `logging.output`, `logging.parsley.output`, `dev.log_database`,
`images.cache_dir`. Confirm the full set during implementation; the list above is what an
initial audit found, not a guarantee of completeness.

### Tests

- Table test over every config path key asserting which anchor it resolves against
- Start from a different working directory; assert identical resolution
- Legacy layout: no site root, no `current` — assert today's behaviour is preserved
- Assert no write occurs inside `ReleaseDir` during a request that writes a file

## Out of Scope

- Release directories and activation (FEAT-153)
- Anything Git (FEAT-154)
- Multi-site / virtual hosts — still one site per server

## Dependencies

None. This is the first unit.

## Definition of Done

Project checklist (`CLAUDE.md`) plus:

- [ ] `go build ./...` and `go test ./...` pass
- [ ] Path-anchor table test covers every configured path key
- [ ] A site started from three different working directories resolves paths identically
- [ ] `basil --dev` in a plain project directory behaves exactly as before
- [ ] `basil --init` produces the new layout and the output is verified by running it
- [ ] `docs/guide/configuration.md` documents `data_dir` and the two anchors
- [ ] `docs/guide/configuration-example.yaml` updated
- [ ] `CHANGELOG.md` entry under `## [Unreleased]`
- [ ] Merged to `main` and pushed; worktree and branch removed

## Notes

`!secret auto` currently generates a fresh value on every start
(`server/config/secret.go:133`), so `session.secret: auto` invalidates sessions on restart.
That is pre-existing and out of scope. It matters here only in one respect: **if `auto` is
ever changed to persist, the generated secret belongs in `DataDir`**, not in the release.
Recorded so the choice is not made accidentally later.

## Open Questions

1. Should a release be allowed to change server settings (port, TLS), or should those be
   read from an operator-owned file outside the release? Simplest answer: allow it, and
   require a restart for listener changes.
2. Does the search index anchor to `DataDir` (it is derived data) or stay site-relative?
   Recommend `DataDir`.
3. `--site <path>` flag, or infer the site root from the working directory? Recommend the
   flag, defaulting to the working directory.
4. ~~Should a public server refuse to start without `server.host`?~~ **Decided** (@sam,
   2026-08-24): yes, with `--dev` and a manual `tls_cert` as the exceptions.
5. ~~Should `--init` derive the admin name from `$USER`?~~ **Decided**: no. Required flag,
   or a prompt when interactive. The earlier recommendation to derive it was wrong —
   `--init` runs on a server, where `$USER` is usually `root`.
6. Should `--init` refuse outright when run as root with no `SUDO_USER`, rather than
   warning? Recommend warning: an operator who genuinely intends to run Basil as root has
   a working setup, and refusing would break it.
