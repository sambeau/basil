---
id: FEAT-152
title: "Site layout: separate code from persistent state"
status: implemented
priority: high
created: 2026-08-24
author: "@sambeau / @claude"
implemented: 2026-08-24
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

### Path audit — VERIFIED 2026-08-24

Every key below was traced from `server/config/config.go`, through the resolution site,
to the syscall that consumes it. Line numbers are as of `c12ffe1`. The earlier draft of
this section was incomplete and wrong in two places; corrections are called out below the
table.

| Config key | Resolved against today | Correct anchor | Resolution (file:line) | Use / syscall (file:line) |
| --- | --- | --- | --- | --- |
| `site.path` | `BaseDir` | `ReleaseDir` | `server/config/load.go:95-97` | `server/server.go:668` (site handler), `server/watcher.go:61` |
| `public_dir` | `BaseDir` | `ReleaseDir` | `server/config/load.go:100-102` | `server/site.go:82` (`filepath.Join` → `ServeFile`), `server/server.go:292` (bundle) |
| `routes[].handler` | `BaseDir` | `ReleaseDir` | `server/config/load.go:70-72` | `server/handler.go:112`, `server/api.go:28` |
| `routes[].public_dir` | `BaseDir` | `ReleaseDir` | `server/config/load.go:73-75`; inherited from `public_dir` at `load.go:79-87` | `server/handler.go:258-268`, `server/api.go:60-70` (becomes `env.RootPath`) |
| `static[].root` | `BaseDir` | `ReleaseDir` | `server/config/load.go:60-62` | `server/server.go:652` (`http.Dir`) |
| `static[].file` | `BaseDir` | `ReleaseDir` | `server/config/load.go:63-65` | `server/server.go:658`, `server/server.go:946` (`http.ServeFile`) |
| `error_pages.<code>` | `BaseDir` | `ReleaseDir` | `server/config/load.go:105-109` | `server/errors.go:376` (parse), `server/server.go:119` (`os.Stat` warning) |
| `security.allow_write` | `BaseDir` | **Undecided — see "Open items"** | `server/config/load.go:112-116` | `server/handler.go:158`, `server/handler.go:278`, `server/api.go:76` → `pkg/parsley/evaluator/eval_helpers.go:442` |
| `developers.<n>.handlers` | `BaseDir` | `ReleaseDir` | `server/config/load.go:424-436` | as `routes[].handler` |
| `developers.<n>.public_dir` | `BaseDir` | `ReleaseDir` | `server/config/load.go:438-451` | as `public_dir` |
| `database.path` | `BaseDir` | `DataDir` | `server/config/load.go:90-92` | `server/server.go:446` (`sql.Open`, + `-wal`/`-shm` sidecars); re-anchor guards at `server/server.go:442`, `server/devtools.go:207,459,523,1073`; `os.Create` on upload at `server/devtools.go:547` |
| `developers.<n>.database.path` | `BaseDir` | `DataDir` | `server/config/load.go:415-421` | as `database.path` |
| *(auth DB — no config key)* | `BaseDir` in the server; `filepath.Dir(configFile)` in the CLI | `DataDir` | none in `load.go`; `server/server.go:499` passes `BaseDir`; `cmd/basil/main.go:356-362` derives it separately | `server/auth/database.go:153,169` → `sql.Open` at `:178` |
| `dev.log_database` | `BaseDir` — **already anchored**, in `devlog.go`, not `load.go` | `DataDir` | `server/devlog.go:56-61` | `server/devlog.go:65` (`os.MkdirAll`), `:70` (`sql.Open`) |
| `https.cache_dir` | **process working directory** (default literal `"certs"`) | `DataDir` | none | `server/server.go:1135-1143` (`autocert.DirCache`) |
| `images.cache_dir` | **process working directory** (default `"./cache/images"`) | `DataDir` | none | `server/server.go:135,145` → `server/images/registry.go:43` → `server/images/cache.go:56,63,80,179,192,207` (`MkdirAll`/`CreateTemp`/`Rename`) |
| `https.cert` | **process working directory** | `DataDir` (operator-owned; must not be replaced by a deploy) | none | `server/server.go:1119` (`ListenAndServeTLS`) |
| `https.key` | **process working directory** | `DataDir` | none | `server/server.go:1119` |
| `logging.output` | **nothing — the key is never read** | `DataDir`, once implemented | none | none |
| `logging.parsley.output` | **nothing — the key is never read** | `DataDir`, once implemented | none | none |

#### Corrections to the initial audit

1. **`dev.log_database` is already anchored.** The initial audit listed it as unresolved.
   It is resolved — just not in `load.go`. `initDevTools` passes `config.BaseDir` to
   `NewDevLog`, which joins a relative path at `server/devlog.go:60` and supplies the
   default filename `dev_logs.db` at `:58`. Note the fallback at `server/server.go:265-268`:
   if `BaseDir` does not exist, the dev log silently relocates to `os.TempDir()`.
2. **`logging.output` and `logging.parsley.output` are dead keys.** Nothing in the tree
   reads either field (the only references are the developer-profile override at
   `server/config/load.go:460-462` and the struct definitions). They cannot "land relative
   to the process working directory" because no file is ever opened for them. Yet
   `basil --init` writes both into the generated `basil.yaml`
   (`cmd/basil/init.go:52-57`) and creates a `logs/` directory for them
   (`cmd/basil/init.go:105-108`), and `docs/guide/configuration-example.yaml:31,33`
   documents them. Anchoring them is not the fix; implementing or removing them is.
3. **`https.cert` and `https.key` were missing from the audit entirely.** They are passed
   raw to `ListenAndServeTLS` at `server/server.go:1119` and resolve against the process
   working directory — the same defect as `https.cache_dir`, and on the same code path
   that FEAT-152 names as the exception permitting a public server to start without
   `server.host`.
4. **The three `developers.<n>.*` path overrides were missing.** They re-anchor to
   `cfg.BaseDir` at `server/config/load.go:418,427,442` and must be split by anchor
   alongside their non-profile equivalents.

#### Unanchored (reach a syscall without ever meeting an anchor)

- `https.cache_dir` → `autocert.DirCache` (`server/server.go:1143`)
- `https.cert`, `https.key` → `ListenAndServeTLS` (`server/server.go:1119`)
- `images.cache_dir` → `os.MkdirAll` / `os.CreateTemp` (`server/images/cache.go:56,63`)
- Parsley `write()` targets. `checkPathAccess` resolves the *written* path with
  `filepath.Abs` — i.e. against the process working directory
  (`pkg/parsley/evaluator/eval_helpers.go:396`) — while the `allow_write` whitelist it is
  compared against was resolved against `BaseDir`. The two agree only when the server was
  started from the project directory. This is the mechanism behind the "start from any
  directory" acceptance criterion and it is not a `load.go` fix.

#### Writes at runtime with no config key at all

- **Auth database** `<BaseDir>/.basil-auth.db` plus its `-wal`/`-shm` sidecars —
  `server/auth/database.go:153,169`. Two independent derivations (server vs. CLI) that
  must be kept in agreement.
- **Dev log database** default `<BaseDir>/dev_logs.db` and sidecars — `server/devlog.go:58`.
- **Database backups**: `<database.path>.<timestamp>.backup` written next to the database
  on DevTools upload — `server/devtools.go:528-530`, `:597`.
- **Database replacement**: DevTools overwrites `database.path` in place via `os.Create` —
  `server/devtools.go:547`.
- **Search index**: `@SEARCH` opens a SQLite index and `MkdirAll`s its directory
  (`server/search.go:351,357`). A relative `path` resolves against `env.RootPath`
  (`server/search.go:344-346`), which is the handler root or `public_dir` — i.e. inside the
  release. With no `path` given, the index is auto-named next to the first `watch` path
  (`server/search.go:234-238`), which is site content — also inside the release. Every
  `@SEARCH` index today lands in code that a deploy replaces.
- **Image cache**: `os.MkdirAll` + `CreateTemp` + `Rename` under `images.cache_dir`
  (`server/images/cache.go:56-80`, `:179-207`).
- **Git repository**: `NewGitHandler` is handed `config.BaseDir` as the repository root
  (`server/server.go:576` → `server/git.go:31`), so pushes write into the code directory.
  FEAT-154 owns the move to `site.git/`, but the anchor decision belongs here.
- **`basil --init`** creates `site/`, `public/`, `logs/`, `db/` under the project directory
  (`cmd/basil/init.go:90-115`) with no data/release distinction.

#### Not filesystem paths (checked, no action)

`auth.login_path` and `auth.protected_paths[].path` are URL paths. `session.table` is a
SQL identifier. `session.store: sqlite` is accepted by the schema but unimplemented —
`initSessions` only ever builds a cookie store (`server/server.go:415-417`), so it opens
no file today; if it is implemented, its database belongs in `DataDir`.

#### Open items this audit could not settle

1. **`security.allow_write` has no single correct anchor.** The spec's acceptance criteria
   place it under `ReleaseDir`, but it is the only gate on site writes, and the shipped
   example whitelists `./data` and `./uploads`
   (`docs/guide/configuration-example.yaml:84-85`) — both of which are, by the same
   spec's "Site-written files" criteria, `DataDir` content. Anchoring it to `ReleaseDir`
   would make the durable write location unreachable. Needs a decision: resolve each entry
   against `DataDir`, accept both anchors, or introduce an explicit prefix syntax.
2. **Search index anchor** (already Open Question 2). Confirmed that today it is
   release-relative in both the explicit and auto-named cases, so "leave it site-relative"
   is not a no-op — it is a decision to keep losing the index on every deploy.
3. **Whether `https.cert`/`https.key` belong in `DataDir` or in an operator-owned location
   outside both anchors.** This depends on Open Question 1 (may a release change listener
   settings?), which is unresolved.
4. **`config.BaseDir` has non-path consumers.** `server/errors.go:169-170` uses it as a
   string prefix to shorten filenames in error output, and `server/devtools.go:1215`
   displays it. Splitting the field forces a choice at both; neither is a resolution site.

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

## Implementation notes (2026-08-24, branch `feat-152-site-layout`)

### How the open items were settled

1. **`security.allow_write` resolves against `DataDir`.** The acceptance criteria placed it
   under `ReleaseDir`, but that is the only gate on site writes and the durable write
   location is in the data root, so a release anchor would make it unreachable. Relative
   entries now resolve against `DataDir` (`./uploads` → `<data root>/uploads`); absolute
   entries are untouched. In the legacy layout `DataDir` is the project directory, so
   nothing changes there. Tested both ways: a handler writing under the data root
   succeeds, a handler writing into the release is refused
   (`server/datadir_test.go:TestHandlerWritesLandOutsideTheRelease`).
2. **Search indexes resolve against `DataDir`** (Open Question 2, recommendation taken).
   `Environment.DataPath` carries the data root into the evaluator; a relative `path`
   lands in `<data root>/search/`, and an auto-named index keeps its filename but moves
   out of the watched content. `pars` sets no `DataPath` and keeps today's
   script-relative behaviour.
3. **`https.cert` / `https.key` resolve against `DataDir`.** They are operator-owned and
   must not be replaced by a deploy; putting them in the data root is the smallest thing
   that achieves that. Open Question 1 (may a release change listener settings?) is
   unresolved and does not block this: an absolute path still works for anyone who wants
   the certificate outside both anchors.
4. **`BaseDir`'s non-path consumers**: `server/errors.go` trims `ReleaseDir` from error
   filenames (they are always code), and DevTools now shows both anchors as separate
   settings rather than one "Base Dir".
5. **`logging.output` / `logging.parsley.output` are still dead keys.** They are now
   anchored to `DataDir` so that implementing them cannot accidentally reintroduce a
   working-directory path, but nothing opens them yet. `--init` no longer writes them
   into the generated config, and no longer creates `logs/`. Implementing or removing
   them remains open, and is not FEAT-152's business.

### Decisions taken during implementation

- **`https.email` moved from an error to a warning.** `https.auto` required it, so a
  public server generated by `--init` could not start without a config edit — which
  contradicts "no configuration step at all". Let's Encrypt does not require a contact
  address. A missing one now warns about expiry and revocation notices.
- **The `basil` context object is bound in handler environments.** The spec asks for
  `basil.data_dir`, but the context dictionary was only reachable through
  `@basil/...` module imports. `env.Set("basil", …)` — the pattern DevTools and the error
  pages already used — makes `basil.data_dir`, `basil.uploads_dir` and
  `basil.uploads_url` readable directly. No new builtin, no new module.
- **Uploads are a convention, not a setting.** `<data root>/uploads` is served at
  `/__uploads/` with no configuration key, keeping the programme's three-setting budget
  (`reports/GIT-DEPLOY-DEFAULTS-REVIEW-2026-08-24.md`). Directory listings are refused.
- **`--init` requires git.** It creates the repository and deploys release 1 from it, so
  a missing `git` is a clear refusal rather than a half-built site.
- **The pre-commit hook ships as `.githooks/pre-commit` in the starter site**, and the
  printed clone command includes `git config core.hooksPath .githooks`. A hook cannot be
  pushed into someone else's clone; FEAT-155 owns making this automatic.
- **`initGit` points at `<site root>/site.git` when it exists**, falling back to the
  release directory in the legacy layout. This is the anchor decision only — the hub
  itself is FEAT-154, and `git.enabled` still defaults to false.
- **Release ids are full commit sha1s** (`releases/<sha>`). FEAT-153 owns release
  identity and may change this.

### Not done here

- No deploy engine, no Git hub, no `basil publish` (FEAT-153/154/155).
- No `basil check` (DESIGN-git-deploy §5.1.2); the eager certificate fetch logs the same
  two suspects — DNS and port 80 — when it fails.
- The eager fetch asks as a modern TLS 1.3 client so autocert issues the ECDSA
  certificate a browser would trigger; asking as a different client would issue a second
  certificate later and spend the rate limit twice.
- Clean-install bootstrap on a real host with real DNS (PLAN-132 Definition of Done) has
  not been performed: no such host was available. Verified locally instead — `--init`,
  start, serve, uploads, legacy layout, and a clone of the created repository.

### Verification (2026-08-24, release engineering pass)

Landed on `main` as `9cf4e04` + `ddd6e24` (fast-forward; the branch was already on top
of `e55a2dd`, so the rebase was a no-op). Branch `feat-152-site-layout` deleted.

Observed directly, not taken on report:

- `go build ./...` and `go test ./...` both pass (`-count=1`, no cached results): every
  package `ok`, including the new `server/config` and `server` suites.
- **Path-anchor table test** — `server/config/paths_test.go` carries the audit table as
  executable cases, with separate tests for `routes[]`, developer profiles and the legacy
  layout.
- **Three working directories** — the same site root started from `/`, `$HOME` and a
  directory inside the site root itself. All three served HTTP 200, and the resolved
  absolute paths (config, handler root, `site.path`, dev log database) were byte-identical
  across all three. No `certs/` or `cache/` directory appeared in any working directory.
- **Legacy layout** — a plain project directory (`basil.yaml` + `site/`) run with
  `basil --dev` was compared against a binary built from `main` before the merge. Response
  body identical; startup logs identical apart from timestamps and request durations.
- **`basil --init`** — produces `site.git/`, `releases/<sha>/`, `current` → the release,
  and `data/` (0700) holding `.basil-auth.db` and `uploads/`. Prints the layout, the API
  key once, and a clone command with hostname and account name in it. Refuses a non-empty
  directory, refuses a missing `--host`, and refuses a missing `--admin` non-interactively.
- **Public server without `server.host`** refuses at config validation with an error naming
  all three fixes (set the host, `--dev`, or configure `https.cert`/`https.key`).
- **Uploads** — `<data root>/uploads/note.txt` served at `/__uploads/note.txt` from outside
  `public_dir`; a symlink planted in the uploads directory, a `..` traversal, and a
  directory listing all returned 404.

Not verified, and still open:

- **Clean-install bootstrap on a real host with real DNS** (PLAN-132 Definition of Done).
  No such host was available. Nothing on the live ACME path — certificate issuance, the
  eager fetch, the backoff marker, the port-80 challenge — has been exercised against
  Let's Encrypt; it is covered by unit tests only.
- **Not pushed to `origin`.** `main` is two commits ahead and waiting for the development
  lead to review and push.
