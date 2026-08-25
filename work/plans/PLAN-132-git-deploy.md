---
id: PLAN-132
feature: FEAT-152, FEAT-153, FEAT-156, FEAT-154, FEAT-155, BUG-033
title: "Implementation Plan for Git Deploy"
status: draft
created: 2026-08-24
updated: 2026-08-25
---

# Implementation Plan: Git Deploy

## Overview

Delivers `DESIGN-git-deploy.md` in five features plus a scoped bug fix. The features are
strictly sequential — each depends on the one before — so this is one plan rather than
five, with a phase per feature.

| Phase | Unit | Depends on | Shape |
| --- | --- | --- | --- |
| 0 | BUG-033 (scoped) | — | Docs + `.gitignore`. Independent, ships immediately |
| 1 | FEAT-152 Site layout | — | Wide blast radius, low conceptual difficulty |
| 2 | FEAT-153 Deploy engine | 1 | The substantial piece. Self-contained, CLI-testable |
| 3 | FEAT-156 Init defaults | 1 | Local-simple init as the default; operator-owned settings. Must land before the hub |
| 4 | FEAT-154 Git hub | 1, 2, 3 | Replaces `server/git.go`. Closes BUG-033 structurally |
| 5 | FEAT-155 `basil publish` | 4 | Developer ergonomics. The optional part is here |

**Why sequential.** `CLAUDE.md` requires one in-flight unit per subsystem, and phases 1–5
all touch `server/config`, `server/server.go` and `cmd/basil`. Running them in parallel
worktrees is exactly the failure mode that rule exists to prevent.

**Why FEAT-156 sits ahead of the hub.** Its operator-owned settings decision (a release
cannot disable `git.*`/`auth.*` on a site root) must be enforced before FEAT-154 makes
the hub load-bearing — otherwise the first graduated local site to push a minimal config
bricks the deploy mechanism from inside a deploy. Slotting it after FEAT-153 also means
its listener-change criterion (recorded in FEAT-153's spec) is already implemented by
the time init starts generating configs without `https:` blocks.

**Where it could stop.** After phase 4 the design is functionally complete: teams can
share work and releases are validated. Phase 5 is what makes it pleasant.

**Defaults.** `reports/GIT-DEPLOY-DEFAULTS-REVIEW-2026-08-24.md` reduced the programme's
configuration surface from ten settings to three and hardened four security defaults.
Phases 1–4 implement that review; do not reintroduce a deleted setting without reading the
reasoning there first.

## Prerequisites

- [x] `DESIGN-git-deploy.md` agreed and merged
- [x] BUG-033 filed and scoped
- [ ] Specs FEAT-152 … FEAT-155 reviewed by @sambeau
- [x] FEAT-156 spec decided (@sam, 2026-08-25; all open questions resolved)
- [ ] Working tree clean, `main` current

---

## Phase 0 — BUG-033 (scoped)

**Files**: `docs/guide/git.md`, `cmd/basil/init.go`
**Effort**: Small. Independent of everything below — do it first and merge it

1. Rewrite the "How Push Reload Works" section of `docs/guide/git.md`. Remove the claim
   that the handler "writes files to the site directory"; it runs `git receive-pack`,
   which updates refs.
2. Document the manual `git config receive.denyCurrentBranch updateInstead` step required
   to use the feature as it stands, and the drifted-tree failure that follows it.
3. Add a note that the mechanism is being reworked, pointing at `DESIGN-git-deploy.md`.
4. Add `cache/` to `gitignoreContent` in `cmd/basil/init.go:13`.
5. Optional (~10 lines): in `initGit`, verify the directory is a repository and log a
   clear error if not.

**Tests**: `.gitignore` template test if one exists; otherwise documentation only.

**Done when**: docs describe something that actually works, and BUG-033 records that the
structural fix is owned by FEAT-154.

---

## Phase 1 — FEAT-152: Site layout

**Effort**: Medium. Mechanically simple, touches many call sites

### Task 1.1 — Audit every configured path
`server/config/load.go`. Produce a table of every path key and which anchor it should
resolve against. The initial audit found these resolved against `BaseDir`: `static[].root`,
`static[].file`, `routes[].handler`, `routes[].public_dir`, `database.path`, `site.path`,
`public_dir`, `error_pages`, `security.allow_write`. And these **not resolved at all** —
they land relative to the process working directory: `https.cache_dir`, `logging.output`,
`logging.parsley.output`, `dev.log_database`, `images.cache_dir`.

Treat that list as a starting point, not as complete. Write the table into FEAT-152 as
implementation notes.

### Task 1.2 — Split the anchor
Replace `Config.BaseDir` with `ReleaseDir` and `DataDir`. Add the `data_dir` key. Resolve
each path against its anchor per the table. Compiler errors are the checklist here — that
is why the field is renamed rather than kept alongside.

### Task 1.3 — Fix the unresolved paths
Anchor `https.cache_dir` and the logging paths to `DataDir`. This changes where an existing
server looks for its certificates, which is a behaviour change worth its own CHANGELOG
line even with no installed base.

### Task 1.4 — Site root and `current`
Resolve the active release through `current`. Support the legacy layout (no site root, no
symlink) so `basil --dev` in a working copy is unchanged.

### Task 1.5 — Parsley data path and uploads
Expose the data root as `basil.data_dir`. Serve a configured uploads directory under a URL
prefix, following `/__p/` and `/__img/`.

### Task 1.6 — `basil --init`
Produce the site-root layout including `data/` and a bare `site.git/`. Accept `--host` and
`--admin`.

**Never infer the admin name from the environment.** `--init` runs on the server, where the
shell is usually `root` (directly, or via `sudo` while granting the port capability), so
`$USER` is wrong exactly where it matters. Required flag; prompt when a terminal is
attached.

**Hand the tree to the right owner.** When `--init` runs as uid 0, `chown` the created tree
to `$SUDO_USER` and say so, or warn and print the command. `sudo basil --init` followed by
running Basil as an ordinary user otherwise fails every write, and the error points at the
database rather than at ownership.

**Commit the starter site to the release branch and deploy it as release 1.** This is not
polish: without an initial release a fresh server has nothing to serve, so it cannot answer
an ACME challenge, cannot obtain a certificate, cannot be cloned over HTTPS, cannot be
pushed to, and cannot get a release. See `DESIGN-git-deploy.md` §5.1.1.

### Task 1.7 — Certificate bootstrap
Obtain the certificate at startup rather than on the first TLS handshake, and log the
outcome plainly — naming DNS or port 80 when it fails.

Refuse to start a public server without `server.host` (decided, @sam 2026-08-24).
`hostPolicy()` returns `nil` when empty (`server.go:1168`), letting `autocert` attempt
issuance for any hostname in SNI. `--dev` and a manual `tls_cert` are the exceptions.

**Tests**: table test over every path key and its anchor; identical resolution from three
different working directories; legacy layout preserved; no write lands inside `ReleaseDir`;
`--init` produces a site with an active release and a clonable repository.

**Risk**: highest-blast-radius phase. Mitigation: rename rather than add the field, so
every call site must be visited.

---

## Phase 2 — FEAT-153: Deploy engine

**Effort**: Large. The substantial piece

### Task 2.1 — Package skeleton
`server/deploy/`: `engine.go`, `release.go`, `record.go`, `lock.go`. Engine takes a commit
and a trigger label and knows nothing about HTTP or Git.

### Task 2.2 — Materialise
Extract a commit into `releases/<id>/` with no `.git` inside it. `git archive` piped to
`tar`, or `checkout-index`.

### Task 2.3 — Validate
Parse every `.pars` handler, part and layout; load the config. Return structured errors
with file, line and message — the hook in phase 4 relays these verbatim.

Includes the listener-change check (FEAT-156, decided @sam 2026-08-25): warn at push
time when a release's config changes `server.host`, `server.port` or the `https` block
relative to the active release on a public server. See FEAT-153's acceptance criteria
and `DESIGN-git-deploy.md` §6.2.

### Task 2.4 — Activate
Re-point `current`, update the running server's release path, clear the script, response
and fragment caches. Requests in flight complete against their original release.

### Task 2.5 — Record and prune
Deploy record (recommend SQLite in `DataDir`), capturing **both** the publishing Basil
account (from the API key) and the commit author (from the commit). They routinely differ.
Prune beyond `deploy.keep`, never the active release.

### Task 2.6 — Lock
File lock in the site root. Concurrent trigger waits or refuses cleanly.

### Task 2.7 — CLI
`basil deploy`, `rollback`, `releases`, `status`. Non-zero exit and actionable errors.

**Tests**: fixture repository through every failure path, not just the happy path;
concurrency; failure leaves the previous release live and removes the partial directory;
rollback; pruning; in-flight request during activation.

**Verify**: `/verify` — deploy a change and watch the served output change; deploy a broken
release and watch it refused with the old output still served.

**Note**: this phase is fully exercisable with no Git transport. Do not let phase 4 start
before its tests are green.

---

## Phase 3 — FEAT-156: Init defaults

**Effort**: Small-to-medium. One command split along existing seams, plus a config
enforcement point. See `work/specs/FEAT-156.md` — all decisions are settled

### Task 3.1 — Split `--init` into local and server modes
`cmd/basil/init.go`, `cmd/basil/main.go`. Local is the default: `basil --init mysite`
with no other flags produces `basil.yaml`, `site/index.pars`, `public/.keep`,
`.gitignore` and runs immediately with `basil --dev`. `--server` carries today's
behaviour byte-for-byte — existing init tests must pass against it unchanged. Flag
validation: `--admin` implies `--server`; `--server` implies `--init`.

### Task 3.2 — Local starter content
Minimal config (no `auth:`/`git:`/`https:` blocks; commented `database:`, `data_dir:`
and `developers:` examples), the runtime-state `.gitignore` (auth DB, `*.db` and
sidecars, `certs/`, `cache/`, `search/`, `uploads/`), and a mode-appropriate summary:
no API key, no clone command, always ending with the one-line graduation pointer.

### Task 3.3 — Git as a quiet nicety
When `git` is on the PATH: normal repository on `main`, `core.hooksPath .githooks`,
the formatting hook, one initial commit. When absent: skip silently. `--no-git` opts
out explicitly. Git is never a gate in local mode.

### Task 3.4 — Operator-owned settings
Where the site-root layout is detected (the same knowledge that picks `DataDir`):
force `git.enabled`, `git.require_auth` and `auth.enabled` on, warning when a release
config explicitly disables them, silent on omission. Legacy layout forces nothing —
`git.enabled` still defaults to false there.

### Task 3.5 — Record for the hub
Write the first-publish non-fast-forward rule into `DESIGN-git-deploy.md` for phase 4:
the hub accepts a non-fast-forward on the release branch only while `live` still points
at the `--init` starter commit and nothing has ever been deployed.

### Task 3.6 — Documentation
Getting-started leads with local init; the deploy guide owns `--server` and
graduation; `docs/guide/configuration.md` gains the one-file layering section
("the file describes the site, not your machine") and the operator-owned list.

**Tests**: local init with/without git and with `--no-git`; refusals (`--admin`
without `--server`); generated config loadable, valid, and containing no server-only
blocks; operator-owned table test per forced key per layout; server-mode tests
unchanged.

**Verify**: `basil --init x && cd x && basil --dev` serves the starter page; a local
site graduated to a fresh `--server` root end-to-end on localhost (remote add, push).

**Why this phase must precede the hub**: the operator-owned enforcement is what makes
a graduated local config safe to deploy. Once phase 4 makes pushes load-bearing, a
minimal config arriving before this enforcement exists would disable the git endpoint
it arrived through.

---

## Phase 4 — FEAT-154: Git hub

**Effort**: Medium

### Task 3.1 — Bare repository
`initGit` points at `<site root>/site.git`. Served at `/.git` — the clone URL does not
change. `basil --init` creates it.

### Task 3.2 — Hook installation
`server/deploy/hooks.go`. Write `pre-receive` and `post-receive` as thin scripts calling
`basil deploy --from-hook`. Install on startup; re-install if missing.

### Task 3.3 — `--from-hook`
Read `<old-sha> <new-sha> <ref-name>` lines from stdin. Handle the all-zeroes cases:
creation and deletion. Refuse deletion of the release branch.

### Task 3.4 — Release branch semantics
`deploy.branch` (default `live`). Release ref moves → validate in `pre-receive`, activate in
`post-receive`. Any other ref → store and stop.

### Task 3.5 — Security hardening
Refuse Git over plain HTTP (currently only warned about, `server/git.go:180`), with the
dev-localhost exception decided in code. **Scope the refusal to Git endpoints** — port 80
must keep serving ACME challenges at `/.well-known/acme-challenge/`, or the server can
never obtain or renew a certificate. Test it rather than assuming it. Remove `git.require_auth` entirely. Refuse to start
if the repository path resolves inside a served root. Refuse force-push and deletion of the
release branch. Promote "no auth database, no Git" to a tested guarantee.

### Task 3.6 — Remove the old path
Delete the `githttp.EventHandler` reload. Rewrite `docs/guide/git.md`.

**Tests**: fixture repository over `httptest` — clone, push a branch, push the release
branch, push a broken release. Assert a rejected push leaves the ref unmoved. Assert hook
output reaches the client's stderr. Role matrix. **BUG-033 regression: a freshly
initialised site accepts a first push with no manual configuration.**

Backlog #114 (`testenv.WithGit()`) is worth building here — this is the phase that needs it.

**Verify**: by hand against a running server, end to end.

**Done when**: BUG-033 closes with a pointer to this phase, and FEAT-035 is marked
superseded.

---

## Phase 5 — FEAT-155: `basil publish`

**Effort**: Medium

### Task 4.1 — `basil publish`
Push the current commit to the release branch. Show the commit range and changed files,
confirm, stream the server's validation output, report the result. `--yes`, `--dry-run`.
Configure the refspec on first use so a plain clone is enough.

### Task 4.2 — Drift reporting
`basil status`; drift in the `publish` summary; a note at `basil --dev` startup. Degrade to
a warning when the server is unreachable.

### Task 4.3 — `basil fmt`
Extract a shared package from `cmd/pars/main.go:863` rather than shelling out, so a Basil
install does not require the `pars` binary. Add directory-tree operation.

### Task 4.4 — Formatting
Install a pre-commit hook from `basil --init`. The server warns on unformatted `.pars` files
in a push and never rejects. No config key. The server never rewrites code.

### Task 4.5 — Documentation
Rewrite `docs/guide/git.md` around the two verbs. Run every command in it.

Three things that must be covered because they are the predictable confusions, not
optional detail (`DESIGN-git-deploy.md` §5.2):

- The **URL username** (selects a stored credential, ignored by Basil) versus
  `user.name`/`user.email` (commit authorship). Unrelated; explain them together.
- **Credential storage per platform**, including that `credential.helper store` writes the
  API key in plaintext. Linux often has no helper configured, making this the most likely
  real-world key leak.
- **Clearing a stale cached credential**, which otherwise fails 401 forever without ever
  re-prompting.

**Tests**: publish success, rejection, dry-run, `--yes`; confirmation not skippable without
`--yes`; drift in all four states; pre-commit hook formats on commit; an unformatted push
warns and succeeds.

---

## Definition of Done

Every phase must satisfy the project checklist in `CLAUDE.md`:

- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Behaviour verified in the running app (`/verify` or `/run`)
- [ ] Docs updated if API or behaviour changed
- [ ] `CHANGELOG.md` entry under `## [Unreleased]`
- [ ] Work committed on its branch; working tree clean
- [ ] Branch rebased on the latest `main`, then merged to `main`
- [ ] `main` pushed to `origin`
- [ ] Worktree and branch removed

Plus, for this programme specifically:

- [ ] **No happy-path-only tests.** Every phase has an explicit failure-path test. This is
      a deployment system; the failure paths are the product
- [ ] **Failure leaves the previous release serving.** Asserted in phases 2, 4 and 5
- [ ] **Manual end-to-end round trip** performed at the end of phases 4 and 5 against a
      real server over HTTPS, not just `httptest`
- [ ] **A clean-install bootstrap is performed at least once** on a fresh host with real
      DNS: `--init`, start, clone, edit, publish — with no manual certificate or Git setup
      at any point. Run it **via `sudo` as an ordinary user**, not as root, so the
      ownership path is genuinely exercised
- [ ] **The docs are executed, not written.** Every command in `docs/guide/git.md` is run
      as written before the phase is called done
- [ ] **The spec is updated with implementation notes**, and any deferral goes to
      `work/BACKLOG.md` with a trigger for revisiting
- [ ] **BUG-033 closed** at the end of phase 4, with its regression test in place
- [ ] **FEAT-035 marked superseded** at the end of phase 4

## Risks

| Risk | Likelihood | Mitigation |
| --- | --- | --- |
| Phase 1 misses a path and state ends up inside a release | Medium | Rename the field so every call site must be visited; table test over every key; assert no write lands in `ReleaseDir` |
| Activation races with in-flight requests | Medium | Requests resolve their path at start; test explicitly with a request open across an activation |
| A half-run post-deploy hook leaves inconsistent state | Medium | Do not auto-roll-back; record and report loudly. Documented decision, not an oversight |
| `go-git-http` is unmaintained | Low | It does very little; vendoring is a contained change if needed |
| Hooks silently absent, so pushes stop deploying | Low | Basil installs them and re-installs on startup if missing |
| Cold-start deadlock: no release → no cert → no clone → no release | **High if missed** | `--init` deploys release 1 (Task 1.6). Covered by the clean-install bootstrap in the DoD |
| `sudo basil --init` leaves a root-owned tree the server cannot write to | **High** | Task 1.6 chowns to `SUDO_USER`. The clean-install bootstrap must be done via sudo, not as root, so this path is actually exercised |
| Plain-HTTP refusal breaks ACME renewal | Medium | Scope the refusal to Git endpoints; explicit test that a challenge path still answers |
| A graduated local config disables git/auth or deploys a dev listener onto a public server | Medium | Phase 3 forces `git.*`/`auth.*` on site roots; phase 2's validation gate warns on listener changes. Phase 3 must merge before phase 4 makes pushes load-bearing |
| Phases run in parallel and conflict | Medium | Strictly sequential; one in-flight unit per subsystem |

## Deferred

Recorded in `DESIGN-git-deploy.md` §10 and to be added to `work/BACKLOG.md` when phase 5
merges: pull from an external upstream, push from CI, Git over SSH, an admin panel deploy
view, branch → environment mapping, Git LFS and submodules.
