---
id: FEAT-157
title: "The release branch lives in site.git/HEAD; retire deploy.branch and git.enabled"
status: proposed
priority: high
created: 2026-08-25
author: "@sambeau / @claude"
related: FEAT-154, FEAT-155, FEAT-156, BACKLOG #153, BACKLOG #154, BACKLOG #151
---

# FEAT-157: The release branch lives in `site.git/HEAD` — retire `deploy.branch` and `git.enabled`

## Summary

The branch whose movement publishes a release is currently named by `deploy.branch`
in `basil.yaml` — which ships inside the release. A deployed release that sets
`deploy.branch: shipping` therefore un-protects `live`: the force-push and deletion
refusals guard whatever the *active release's config* says, so the history-protection
invariant is owned by the thing it protects (backlog #153).

The fix is not a new setting but the removal of a duplicated one. The bare
repository already records the release branch: `--init` runs
`git symbolic-ref HEAD refs/heads/live` on `site.git` so clones check out the right
branch (`cmd/basil/init.go:772`). `site.git/HEAD` and `deploy.branch` are two records
of the same decision, kept in agreement today only because init writes both from one
constant. This feature makes **`site.git/HEAD` the single source of truth**:

1. Every server-side consumer of the release branch reads it from `site.git/HEAD`.
2. `basil publish` learns it from the server (`git ls-remote --symref origin HEAD`)
   instead of from its local clone's committed config.
3. The `deploy.branch` config key is removed; a config still carrying it gets a
   load-time warning naming the replacement, never an error.
4. `git.enabled` — inert since FEAT-156 made it operator-owned (backlog #154) — is
   removed from `basil.yaml` too. The one configuration it used to express (serve the
   site, but do not expose `/.git`; deploys happen at the shell) moves to the same
   operator-owned home: `git config basil.gitEnabled false` inside `site.git`.
5. Riders from #153's investigation: `data_dir` in a release config is ignored with a
   warning on site roots (the data root is the operator's ground truth), and the
   `dev` section joins `carryRestartRequiredSettings`.

An operator changes the release branch with one git command on the box:

```
git -C /srv/mysite/site.git symbolic-ref HEAD refs/heads/main
```

— which is also how push-to-publish mode (design decision D3's "`main` remains a
supported one-liner") is now expressed.

## User Story

As a Basil operator, I want the branch that publishes — and whether the git endpoint
is exposed at all — to be facts about my server that no deploy can rewrite, so that
the protections built in FEAT-154/156 cannot be steered by the content they protect.

As a developer running `basil publish`, I want the client to ask the server which
branch releases, so that a clone (or, later, a graduated local project — #151) needs
no committed config naming it.

## Motivation

FEAT-156's security review (2026-08-25) showed the concrete attack: deploy a release
whose config says `deploy.branch: shipping`. From then on `refs/heads/live` is "any
other ref" to the hub — freely force-pushable and deletable via store-and-stop
(`cmd/basil/fromhook.go:143-147`) — and the first push of `shipping` takes the
ref-creation path that skips both the ancestry check and the starter-overwrite record
check. An authenticated editor is required, so this is privilege misuse rather than
escalation, but the invariant "release history cannot be rewritten" must not be
ownable by a release.

FEAT-156 fenced `git.enabled`/`auth.enabled` by *forcing* them — correct for
booleans with one sane server value. The release branch is a *value*, so forcing is
meaningless; it needs an operator-owned home. `site.git/HEAD` already is one:
server-side, deploy-proof, git-native, and already required to agree with the
setting it replaces.

### Why HEAD and not a `basil.*` git-config key for the branch

`HEAD` also controls what a fresh clone checks out. Keeping branch-that-publishes
and branch-clones-get as one fact is a feature: today they can only drift apart by
operator error, and after this change they cannot drift at all. (Decided, @sam
2026-08-25: use `site.git/HEAD`.) The `basil.*` git-config namespace is still
introduced — for `gitEnabled`, where no git-native fact exists.

## Design

### The single source and its readers

A new helper — `deploy.ReleaseBranch(repoDir string) (string, error)` — shells
`git symbolic-ref HEAD` in the bare repository and returns the branch name
(`refs/heads/` stripped). Its consumers:

| Consumer | Today | After |
| --- | --- | --- |
| Receive hooks (`cmd/basil/fromhook.go:129`) | `cfg.Deploy.ReleaseRef()` from the active release | `deploy.ReleaseBranch(cfg.BareRepoPath())` |
| Server CLI hints (`cmd/basil/deploy.go:131`, `basil status`) | `cfg.Deploy.Branch` | same helper, when a site root is active |
| `basil publish` (`cmd/basil/publish.go:73`) | local clone's committed `deploy.branch` | `git ls-remote --symref origin HEAD` — one round trip it already makes (`publish.go:97`) |
| `--init --server` | writes the constant into both `basil.yaml` and HEAD | writes HEAD only |
| Fresh clones | follow HEAD (unchanged) | follow HEAD (unchanged) |

`pars` and the legacy layout have no repository and publish nothing; no consumer
exists there.

### Failure modes, decided

- **HEAD names a branch that does not exist yet** (operator retargeted before pushing
  it): the hub's protections attach to the named branch — pushes to the *old* release
  branch are stored, not published, and nothing deploys until the new branch arrives.
  `basil check` reports it (below).
- **HEAD unreadable / repository missing**: the hook refuses the push with an error
  naming the file, and `basil publish` degrades exactly as it does today when
  `ls-remote` fails (`publish.go:89-97` already treats that as "server unreachable").
- **Detached HEAD in site.git** (operator ran something odd): treated as unreadable —
  refuse with a message naming the fix (`git symbolic-ref HEAD refs/heads/<branch>`).

### Retired keys

- **`deploy.branch`** is removed from `DeployConfig`. A config that still contains it
  loads fine and warns: *"deploy.branch is no longer read — the release branch is
  site.git's HEAD; change it with: git -C <site root>/site.git symbolic-ref HEAD
  refs/heads/<branch>"*. Detection uses the raw-YAML probe pattern from
  `server/config/operator.go` (the struct field is gone, so the probe is the only
  reader). Never an error: a stale key must not block a load.
- **`git.enabled`** is removed the same way, warning that the endpoint is controlled
  by `git config basil.gitEnabled` in `site.git`. `enforceOperatorOwned` drops its
  git half (auth stays); the FEAT-156 operator-owned docs section shrinks
  accordingly.
- `deploy.keep` is untouched — it is retention tuning, not an invariant.

### The operator off-switch: `basil.gitEnabled`

`git config basil.gitEnabled false` in `site.git` disables serving `/.git` (clone
and push both). Default absent = enabled, preserving today's behaviour. Read at
startup and on `SwapRelease` alongside the config load; a change requires the same
restart the listener settings do. This restores the CLI-only-deploys configuration
that FEAT-156 made inexpressible — as an operator fact a release cannot touch.
FEAT-154's "no auth database, no Git" and FEAT-156's missing-DB degrade are
unchanged and compose with it (the endpoint serves only when the switch is on AND
the auth database exists).

### Riders

- **`data_dir` on site roots**: a release config that sets it is ignored with the
  operator-owned warning; the running data root stays the startup value
  (`<site root>/data` by convention). FEAT-156 already carries it across live swaps;
  this closes the restart path. The legacy layout keeps the key exactly as FEAT-152
  defined it — there it is the operator speaking.
- **The `dev` section** joins `carryRestartRequiredSettings`
  (`server/activate.go`), closing the `dev.log_database` loose end from FEAT-156's
  review.

## Acceptance Criteria

### Source of truth

- [ ] The receive hooks read the release branch from `site.git/HEAD`; a deployed
      config naming a different `deploy.branch` changes nothing (regression test:
      the #153 attack — deploy such a release, then attempt a force-push of `live` —
      is refused)
- [ ] `basil deploy`/`basil status` on a site root report the branch from HEAD
- [ ] `basil publish` learns the branch via `ls-remote --symref origin HEAD` and
      needs no `deploy.branch` anywhere; publishing to a hub whose HEAD was
      retargeted publishes to the new branch with no client change
- [ ] `--init --server` no longer writes `deploy.branch` into the generated config
- [ ] Retargeting HEAD to a not-yet-pushed branch: pushes to the old branch are
      stored-not-published; `basil check` reports the state plainly
- [ ] `basil check` verifies `site.git`'s HEAD names a branch that exists, warning
      when the operator retargeted ahead of the first push (one `show-ref` call)
- [ ] Detached or unreadable HEAD: push refused with the symbolic-ref fix named

### Retired keys

- [ ] `deploy.branch` and `git.enabled` are gone from the config structs, the
      generated configs, `configuration-example.yaml`, and the docs
- [ ] A config carrying either key loads with a warning naming the replacement;
      never an error
- [ ] The retired-key warning also surfaces where a clone would otherwise never see
      it: at `basil --dev` startup and in `basil publish` output — a stale committed
      key must not linger unseen in the repository everyone pulls from
- [ ] `git config basil.gitEnabled false` in `site.git` disables `/.git` (clone and
      push both 404/refused); absent or `true` serves as today; tested both ways
- [ ] `enforceOperatorOwned` forces `auth.enabled` only; its table test updated

### Riders

- [ ] A release setting `data_dir` on a site root: ignored, warned, at startup and
      across swaps; legacy layout unchanged (table-test rows)
- [ ] `dev` section carried across `SwapRelease` with the standard warning

### Documentation

- [ ] `docs/guide/git.md` + `deployment.md`: one section on retargeting the release
      branch (the symbolic-ref command), push-to-publish mode expressed this way,
      and the `basil.gitEnabled` switch
- [ ] `docs/guide/configuration.md`: operator-owned section updated (auth forced;
      branch and git switch live in `site.git`; `data_dir` ignored on site roots)
- [ ] `DESIGN-git-deploy.md`: D3's "set `deploy.branch: main`" note updated to the
      symbolic-ref command; §7's three-key config table updated; the #153 attack and
      this resolution recorded
- [ ] `CHANGELOG.md` under `## [Unreleased]` (Breaking: two keys removed)

## Out of Scope

- `basil publish --to` / adopting a remote from an un-cloned project — #151.
- A `basil` verb wrapping the symbolic-ref command (revisit if the raw command
  proves unfriendly in practice; it is one documented line).
- The recursive-removal sandbox fix — #152, its own unit.
- Multi-branch → multi-environment mapping (deferred since PLAN-132).

## Dependencies

- FEAT-154/155/156 (all implemented). No migration: no installed base.
- Sequencing note: touches `cmd/basil/fromhook.go` and `server/config` — merge any
  in-flight work on those files first (one in-flight unit per subsystem).

## Definition of Done

Project checklist (`CLAUDE.md`) plus:

- [ ] The #153 attack regression test exists and fails against pre-FEAT-157 code
- [ ] `basil publish` exercised against a live hub after a HEAD retarget
- [ ] Both states of `basil.gitEnabled` exercised at the real HTTP surface
- [ ] Backlog #153 and #154 closed with pointers here

## Open Questions

All resolved (@sam, 2026-08-25):

1. ~~Should `basil check` verify HEAD names an existing branch?~~ — **Yes.** One
   `show-ref` call, and `check` is the diagnosis surface. In the acceptance
   criteria.
2. ~~Retired-key warnings in a clone?~~ — **Yes, at `basil --dev` startup and in
   `basil publish` output.** The clone is where a stale committed key would
   otherwise linger unseen. In the acceptance criteria.
