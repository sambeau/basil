---
id: FEAT-153
title: "Deploy engine: releases, validation, activation and rollback"
status: implemented
priority: high
created: 2026-08-24
author: "@sambeau / @claude"
---

# FEAT-153: Deploy engine — releases, validation, activation and rollback

## Summary

Make deployment a thing Basil *does* rather than a side effect of files appearing on disk.
One engine takes a commit and turns it into the live site: resolve, lock, materialise,
validate, activate, record — leaving the previous release live and intact if anything
fails.

Deliberately transport-agnostic. This unit is driven entirely from the CLI and has no Git
transport of its own, which makes it fully testable before FEAT-154 exists.

## User Story

As a Basil operator, I want a broken release to be refused rather than published, and a
bad release to be undone in one command, so that deploying is not an act of faith.

## Motivation

Today "deploy" is three `cache.Clear()` calls in a callback (`server/server.go:579`). There
is no point at which Basil could examine a release and decline it, no record of what was
deployed, and no way back other than deploying again and hoping.

Basil owns the Parsley parser, so it can do something no general-purpose Git host can: read
a release before activating it and refuse one that would break the site. That is a build
check with no build server, and it is the strongest single argument for the whole design.

## Acceptance Criteria

### The pipeline

- [ ] `resolve` — a branch name, tag or SHA resolves to a single commit
- [ ] `lock` — one deploy at a time per site; a concurrent trigger waits or is refused
      cleanly, never interleaved. File lock in the site root
- [ ] `materialise` — the commit is extracted into `releases/<id>/`, byte-identical to the
      commit
- [ ] `validate` — every `.pars` handler, part and layout is parsed; the config is loaded
- [ ] `activate` — `current` is re-pointed and the running server's release path updated
- [ ] `hook` — if `deploy.pars` exists in the release root it runs after activation.
      **Convention, not configuration**, matching `index.pars` / `{folder}.pars` (FEAT-040)
- [ ] `record` — commit, timestamp, duration and outcome are stored, plus **both
      identities**: the Basil account that published (from the API key) and the commit
      author (from `user.name` / `user.email`). They routinely differ — someone merges and
      publishes a colleague's work — and storing only one makes the record useless for the
      question it did not store (`DESIGN-git-deploy.md` §5.2.4)
- [ ] `prune` — releases beyond `deploy.keep` (default 5) are removed, never the active one

### Failure behaviour

- [ ] Any failure before activation leaves the previous release live and untouched
- [ ] A failed release directory is removed, not left half-built
- [ ] Failures are recorded with their reason, not silently discarded
- [ ] Validation errors identify file, line and message
- [ ] A post-deploy hook failure is recorded and reported but does **not** roll back
      automatically (see Design Decisions)

### Activation semantics

- [ ] Requests in flight complete against the release they started on
- [ ] The script, response and fragment caches are cleared on activation
- [ ] Activation is idempotent — deploying the already-active commit is a no-op that
      still records

### CLI

- [ ] `basil deploy <sha|branch|tag>` — run the pipeline for a commit already in the repo
- [ ] `basil rollback [id]` — re-activate the previous release, or a named one
- [ ] `basil releases` — the deploy record: id, commit, when, published-by, authored-by,
      outcome, which is live
- [ ] `basil status` — what is live, and whether the release branch is ahead of it
- [ ] `basil check` — verify bootstrap preconditions and report each plainly: the hostname
      resolves to this machine, port 80 is reachable, a certificate is present or
      obtainable, the repository is not inside a served root, a release is active. This is
      the command to point people at when setup misbehaves
      (`DESIGN-git-deploy.md` §5.1.2)
- [ ] All four exit non-zero on failure and print actionable errors

### Configuration

- [ ] `deploy.keep` (default `5`) — the only setting this feature adds
- [ ] Validation is **always on** for pushes; there is no config key to disable it
- [ ] `basil deploy --no-validate` is the emergency override. It needs shell access, which
      is the right bar for overriding a safety check, and cannot be left switched on
- [ ] No `deploy.validate` and no `deploy.hook` key — see
      `reports/GIT-DEPLOY-DEFAULTS-REVIEW-2026-08-24.md`

## Design Decisions

- **Transport-agnostic by construction.** The engine takes a commit and a trigger label.
  It knows nothing about pushes, hooks or HTTP. This is what makes FEAT-154 small, and it
  means the engine can be fully exercised from the CLI in this unit.

- **Validation is correctness, not style.** Parse and config-load only. Formatting is
  handled by a local pre-commit hook (FEAT-155) and never blocks a release.

- **No config key for validation.** A file-based way to disable the safety check would live
  inside the release being validated, and would persist silently after the emergency that
  justified it. The override is a server-side flag instead.

- **Validation catches broken, not unfinished.** Say so in the docs. It is not a substitute
  for the explicit publish step — the two protect against different mistakes.

- **A release is byte-identical to its commit.** The server never rewrites code. Rollback,
  the deploy record and reproducibility all depend on `releases/<id>` genuinely being that
  commit.

- **Post-deploy hook failure does not auto-roll-back.** A migration that half-ran is not
  made better by reverting the code underneath it. Record it, report it loudly, and let a
  human decide. Revisit if this proves wrong in practice.

- **Rollback re-activates, it does not re-materialise.** The previous release directory is
  still on disk, so rollback is a symlink swap and a cache clear. That is what makes it
  fast enough to be the emergency answer.

## Technical Context

### New files

| File | Purpose |
| --- | --- |
| `server/deploy/engine.go` | The pipeline |
| `server/deploy/release.go` | Release directory lifecycle, pruning |
| `server/deploy/record.go` | Deploy record storage |
| `server/deploy/lock.go` | File lock |
| `cmd/basil/deploy.go` | `deploy`, `rollback`, `releases`, `status` subcommands |

### Touched

| File | Change |
| --- | --- |
| `server/server.go:576-593` | `initGit`'s `onPush` callback is replaced by the engine |
| `server/server.go` | Expose an activation entry point that swaps the release path and clears caches |
| `cmd/basil/main.go:35-46` | Subcommand dispatch |
| `server/config/config.go` | `DeployConfig` |

### Materialising

`git archive <sha> | tar -x -C releases/<id>` or `git --work-tree=… checkout-index -a`.
Prefer whichever leaves no `.git` inside the release — the release is code, not a
repository.

### The deploy record

Store where it can be read without the server running, and where a future UI could reach
it. SQLite in `DataDir` is consistent with the rest of Basil; a JSONL file is simpler and
sufficient. Recommend SQLite for queryability.

### Tests

- Pipeline unit tests with a fixture repository: success, validation failure, hook failure
- Concurrent deploys serialise; the second sees a clean refusal or waits
- Failure leaves the previous release active and removes the partial directory
- Rollback restores the previous release and records it
- Pruning keeps `deploy.keep` and never removes the active release
- Activation while serving: a request in flight completes against its original release

## Out of Scope

- Git transport, hooks, the bare repository (FEAT-154)
- `basil publish` and the developer-side workflow (FEAT-155)
- The formatting gate (FEAT-155)
- Zero-downtime listener changes (deferred; see FEAT-152 Open Questions)

## Dependencies

- **FEAT-152** — release directories and `DataDir` must exist first

## Definition of Done

Project checklist (`CLAUDE.md`) plus:

- [ ] `go build ./...` and `go test ./...` pass
- [ ] Engine tested against a fixture repository covering every failure path, not just the
      happy path
- [ ] Verified at the real HTTP surface (`/verify`): deploy a change, observe the served
      output change; deploy a broken release, observe it refused and the old output still
      served
- [ ] Rollback verified the same way
- [ ] Concurrency test demonstrates serialisation
- [ ] `docs/guide/` gains a deployment page covering the CLI and the pipeline
- [ ] `CHANGELOG.md` entry under `## [Unreleased]`
- [ ] Merged to `main` and pushed; worktree and branch removed

## Open Questions — resolved

1. Release id: **full SHA directories** (matching `--init`); `basil releases` shows the
   record's sequence number plus a 12-char SHA, giving sequence-plus-SHA where it is read.
2. Deploy record: **SQLite** at `<data_dir>/deploy.db`, WAL, single connection, pragmas in
   the DSN.
3. Eager certificate issuance: **already implemented in FEAT-152** (obtained at startup,
   outcome logged naming DNS/port 80); nothing left for this feature.
4. `basil deploy` with the server stopped: **yes** — it activates the release for the next
   start; a running server picks activation up through its own watcher on `current`.

## Implementation Notes (2026-08-24)

Implemented on `claude/basil-git-deploy-0ad5un` as `server/deploy/` (lock, record,
release, validate, engine), `server/activate.go` (+ pinned-handler changes in server.go,
site.go, handler.go, api.go, errors.go), and `cmd/basil/deploy.go`. Three-lens review
(spec, craft, concurrency) yielded 2 proven blockers and 5 majors, all fixed with
fail-before/pass-after regression tests, plus 16 minors — see `git log` and
`DESIGN-git-deploy.md` §6.6 for what the phase taught.

Deviations and decisions of note:

- **Activation mechanism**: one atomic `serveState` publish (mux + config + asset bundle);
  handlers pin config/bundle/cache-generation at construction. The in-flight guarantee is
  race-detector-tested, not incidental.
- **Response/fragment caches are generation-salted** so an in-flight old-release request
  cannot poison the new release's cache (was deterministically reproducible before).
- **Restart-required config**: sections whose subsystems are not rebuilt live are carried
  from the running config with a per-section warning (design doc §7, decided).
- **`deploy.keep` is clamped to ≥ 2** and pruning also protects the previously-activated
  release — another process may still be serving it.
- **Hook (`deploy.pars`)**: runs post-activation with an explicit CLI-trust security
  policy (FEAT-154's push trigger must revisit); relative writes land in the release root
  (script-relative), so docs direct hooks at absolute/data-dir paths; a hook failure exits
  3 with the release live and the reason in the record and `basil releases`.
- **`--init` seeds the deploy record** with release 1 (trigger `init`), so the first
  rollback after the first real deploy works.
- **`basil check`** is honest about local limits: port-80 reachability from outside and
  certificate obtainability are reported as notes, not proven — weaker than this spec's
  "port 80 is reachable" wording, deliberately.
- **Windows**: the file lock has unix (flock) and Windows (LockFileEx) implementations;
  `GOOS=windows` builds.
- **Not done here**: `initGit`'s legacy `onPush` reload still exists — its removal is
  FEAT-154's scope. Crash window between `SetCurrent` and the record write is documented,
  not closed (backlog #138). Deferred items live in `work/BACKLOG.md` #136–#143.
