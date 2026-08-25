---
id: FEAT-154
title: "Git hub: bare repository, release branch and receive hooks"
status: implemented
priority: high
created: 2026-08-24
author: "@sambeau / @claude"
---

# FEAT-154: Git hub — bare repository, release branch and receive hooks

## Summary

Turn the server's Git endpoint from "the live site directory, served over HTTP" into a
proper bare repository that a team can share, and wire it to the deploy engine through
real Git hooks so that a push to the release branch is checked before it becomes the live
site — and refused, visibly, if it fails.

This is where BUG-033 is structurally fixed: pushes never contend with a checked-out
branch because there is no longer a checked-out branch.

## User Story

As a developer, I want to push any branch to the server so my colleague or my other
machine can see it, without any of it reaching the public until it is deliberately
released.

## Motivation

`server/git.go` currently serves `config.BaseDir` — a non-bare repository whose working
tree is the live site. Git refuses by default to accept a push into a checked-out branch,
which is BUG-033. Even configured around, a push then fails whenever the live tree has
drifted.

More importantly, the current arrangement gives the server nowhere to *keep* work that
isn't live. A branch pushed to a repository whose working tree is the site is still a
branch in the site's repository, and the only branch anyone can meaningfully use is the one
being served. That is why two developers currently get in each other's way.

The reload path has the same shape of problem: `githttp.EventHandler` fires after the fact.
It cannot refuse a push, cannot report anything to the developer, and races with the files
it is reacting to.

## Acceptance Criteria

### Repository

- [ ] The server holds a **bare** repository at `<site root>/site.git`, **not configurable**
- [ ] Git deploy is active when that repository exists; `git.enabled: false` remains only as
      an off-switch, and nobody ever needs to write `git.enabled: true`
- [ ] Basil **refuses to start** if the repository path resolves inside any served root
      (`public_dir`, `site.path`, `static[].root`)
- [ ] It is served at `/.git` — the clone URL is unchanged from today
- [ ] `git clone https://user@host/.git` succeeds and yields full history
- [ ] Pushing any branch succeeds and stores it; nothing is published
- [ ] The repository lives outside every served root

### Release branch

- [ ] `deploy.branch` (default `live`) names the branch that triggers a deploy
- [ ] A push that moves the release branch triggers the FEAT-153 engine
- [ ] A push that moves any other ref stores it and does nothing else
- [ ] `deploy.branch: main` restores push-to-publish for anyone who wants it — the one
      genuine choice this programme exposes
- [ ] Tags are accepted as a release ref (`refs/tags/...`) even though config speaks in
      branches

### Hooks

- [ ] A `pre-receive` hook runs validation for the release branch **before** any ref moves
- [ ] Validation failure exits non-zero: the push is rejected, the ref does not move, the
      live site is untouched
- [ ] Hook output reaches the developer's terminal as `remote:` lines, including file, line
      and message
- [ ] A `post-receive` hook performs activation via the engine
- [ ] Hooks are installed by Basil, not by the operator, and are re-installed if missing
- [ ] Hooks invoke `basil deploy --from-hook`; no deploy logic lives in shell

### Authentication and transport security

- [ ] Existing model unchanged: HTTP Basic over TLS, API key as password, validated
      against the auth database
- [ ] **Git operations over plain HTTP are refused**, with an error explaining why — not
      merely logged as a warning, which is what happens today (`server/git.go:180`). Basic
      auth puts the API key in an easily-decoded header, so plain HTTP means a plaintext
      credential with push rights
- [ ] The sole exception is a dev-mode localhost bind, decided in code (`isDevLocalhost`),
      never from configuration
- [ ] **The refusal applies to Git endpoints only.** Port 80 must continue to serve ACME
      HTTP-01 challenges at `/.well-known/acme-challenge/` ahead of the HTTPS redirect
      (`server.go:1158`). A blanket plain-HTTP refusal would make the server unable to
      obtain or renew a certificate — tested explicitly, not assumed
- [ ] **There is no `git.require_auth` setting.** Authentication cannot be disabled from a
      config file
- [ ] Basil refuses to serve Git when no auth database exists — promoted from an incidental
      check (`server.go:571`) to a stated guarantee with a test
- [ ] The URL username is ignored — only the API key authenticates
      (`server/git.go:122-129`). This is existing behaviour; the requirement here is that
      it is **documented explicitly** rather than left to be discovered, along with the
      advice to use the real account name anyway so the credential helper, which keys on
      *(host, username)*, can hold two accounts for one host
- [ ] Any authenticated user may clone and fetch
- [ ] `editor` or `admin` may push, including moving the release branch
- [ ] A rejected role check explains which role is required
- [ ] **The release branch cannot be force-pushed or deleted**, by anyone, with no setting
      to permit it — a rewritable release history makes the deploy record, and therefore
      rollback, unreliable

### Replacing the old path

- [ ] `githttp.EventHandler`-based reload is removed
- [ ] BUG-033 no longer reproduces: a fresh `basil --init` site accepts a first push with
      no manual Git configuration
- [ ] A fresh `basil --init` site can be **cloned** immediately: the repository has the
      starter site on the release branch (FEAT-152), so a clone yields working files rather
      than an empty repository
- [ ] `docs/guide/git.md` is rewritten; the claim that the handler "writes files to the
      site directory" is removed

## Design Decisions

- **Real Git hooks, not the library callback.** `go-git-http` shells out to the real `git`
  binary (`cmd.Dir` is set and `receive-pack` is invoked), so repository hooks fire. This
  buys three things the callback cannot provide: `pre-receive` can *refuse*; hook stdout is
  relayed to the client; and ordering is guaranteed rather than racing.

- **Hooks are thin.** Two-line scripts calling `basil deploy --from-hook`. Shell is a bad
  place for deploy logic and an untestable one.

- **Basil installs the hooks.** An operator who has to install a hook by hand is an
  operator who will one day have a repository that silently stops deploying. Re-install on
  startup if absent.

- **The clone URL does not change.** `/.git` continues to work; only what sits behind it
  changes. Nothing a developer types is different.

- **~~`go-git-http` stays.~~ Reversed during implementation (2026-08-24): `go-git-http`
  was removed.** The premise failed: the library sets only `cmd.Dir` on the `git`
  subprocess and exposes no hook for per-request environment, so the authenticated
  account's name could not be passed to `receive-pack` for the deploy record (D20). Serving
  Smart HTTP directly — ref advertisement plus `git upload-pack`/`receive-pack
  --stateless-rpc` — is ~30 lines per direction, sets `BASIL_PUBLISHER` per request, and
  also drops the library's dumb-protocol raw-file serving of the object store (an
  unnecessary attack surface). One unmaintained dependency gone. See the implementation
  notes below.

## Technical Context

### Files

| File | Change |
| --- | --- |
| `server/git.go` | Point at the bare repo; drop `EventHandler`; add release-ref role check |
| `server/git.go:57-100` | `ServeHTTP` — role check must distinguish the release ref from others |
| `server/server.go:561-593` | `initGit` — bare repo path, hook installation, engine wiring |
| `server/deploy/hooks.go` (new) | Hook templates and installation |
| `cmd/basil/deploy.go` | `--from-hook` mode: reads the ref update lines from stdin |
| `cmd/basil/init.go` | Create the bare repository |
| `docs/guide/git.md` | Rewrite |

### Hook protocol

`pre-receive` receives `<old-sha> <new-sha> <ref-name>` per updated ref on stdin. The
`--from-hook` mode reads the same format, so the shell script is a pass-through. Exit
non-zero from `pre-receive` to reject the entire push.

Note the deletion and creation cases: `<old-sha>` is all-zeroes for a new ref,
`<new-sha>` is all-zeroes for a deletion. Deleting the release branch should be refused.

### Tests

- Fixture repository over a real `httptest` server: clone, push a branch, push the release
  branch, push a broken release branch
- Assert a rejected push leaves the ref unmoved (`git rev-parse` on the server repo)
- Assert hook output appears in the client's stderr
- Role checks: viewer cannot push; editor can push a branch and move the release branch
- Plain-HTTP request to any Git endpoint is refused, and the dev-localhost exception works
- Startup refuses when the repository path resolves inside a served root
- Force-push and deletion of the release branch are both refused
- BUG-033 regression: a freshly initialised site accepts a first push with no manual config
- Force-push and ref-deletion behaviour on the release branch

Backlog item #114 (`testenv.WithGit()`) becomes relevant here — this unit is the reason to
build it.

## Out of Scope

- `basil publish` and drift reporting (FEAT-155)
- The formatting gate (FEAT-155)
- Pull from an external upstream, push from CI, Git over SSH — see
  `DESIGN-git-deploy.md` §10
- Git LFS and submodules — unsupported; document rather than implement

## Dependencies

- **FEAT-152** — the layout
- **FEAT-153** — the engine the hooks call

## Definition of Done

Project checklist (`CLAUDE.md`) plus:

- [ ] `go build ./...` and `go test ./...` pass
- [ ] End-to-end verified by hand against a running server: clone, push a branch (nothing
      published), push the release branch (site updates), push a broken release (rejected,
      site unchanged, error visible in the terminal)
- [ ] BUG-033 regression test present and passing; BUG-033 closed with a note pointing here
- [ ] Role matrix tested
- [ ] **Plain HTTP verified refused** against a real non-TLS listener, and the dev-localhost
      exception verified to still work
- [ ] **Certificate issuance verified unaffected** — a server that has never held a
      certificate obtains one and is then clonable, end to end
- [ ] Force-push and release-branch deletion verified refused
- [ ] `docs/guide/git.md` rewritten and accurate — every command in it actually run
- [ ] Docs distinguish the **URL username** (selects a stored credential) from
      `user.name`/`user.email` (commit authorship). These are unrelated and conflating them
      is the likeliest confusion in the design
- [ ] Docs cover **credential storage per platform**, and state that
      `credential.helper store` writes the API key in plaintext to `~/.git-credentials`.
      Linux frequently has no helper configured, so this is the most likely way a key leaks
      in practice — recommend `libsecret`
- [ ] Troubleshooting covers a stale cached credential (`git credential reject`), which
      otherwise fails 401 forever and never re-prompts
- [ ] `CHANGELOG.md` entry under `## [Unreleased]`
- [ ] FEAT-035 marked superseded, with a pointer to this spec
- [ ] Merged to `main` and pushed; worktree and branch removed

## Implementation Notes (2026-08-24)

Implemented on `claude/basil-git-deploy-0ad5un` (G1 `f1a336d` hooks/`--from-hook`/
`deploy.branch`/`Engine.Prepare`; G2 `cec735c` transport + hardening; G3 `02c9e15` docs +
BUG-033/FEAT-035 closure; fix round `11230ab`). All acceptance criteria met with code and a
proving test; verified end to end by the orchestrator with a real `git` client (clone,
store-only branch push, publishing push with the pusher recorded, broken push rejected with
the ref unmoved and the live site unchanged, force-push and deletion refused).

Reviewed by three lenses (spec, craft, security). The security lens **refuted** command
injection, path traversal, raw-file serving, force-push/deletion bypass, and ACME-scoping
regressions with file:line. Findings fixed in `11230ab`.

Decisions and deviations:

- **`go-git-http` removed; Smart HTTP served directly** (see the reversed Design Decision
  above) — needed for per-request `BASIL_PUBLISHER`, and it dropped the dumb-protocol
  raw-file surface.
- **`deploy.pars` runs sandboxed on a push** — `@exec`/`@shell` denied, writes scoped to
  the data root; full power only via `basil deploy` at the server shell. Decision by
  @sambeau; rationale in `DESIGN-git-deploy.md` §6.7. The escape hatch is named in the
  refusal the developer sees.
- **The receive-pack role gate reads the decoded git service once** — a percent-encoded
  service name (`git-receive%2dpack`) can no longer skip the editor/admin gate on the ref
  advertisement.
- **`viewer` role** added to hold the fetch-only side of the matrix; `SetUserRole` accepts
  it; account names reject control characters.
- **The RPC gzip body is bounded** (1 GiB decompressed) against a decompression bomb; a
  full total-push-size limit is deferred — Open Question 2, backlog #144.

Open Questions resolved: (1) non-release branches are **not** auto-pruned — the repository
is the developer's too. (2) A push-size limit is **partially** addressed (gzip bound); the
overall cap is deferred to backlog #144.
