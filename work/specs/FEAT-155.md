---
id: FEAT-155
title: "basil publish: explicit releases, drift reporting and the formatting gate"
status: draft
priority: high
created: 2026-08-24
author: "@sambeau / @claude"
---

# FEAT-155: `basil publish` — explicit releases, drift reporting and the formatting gate

## Summary

Give publishing its own verb. `git push` shares work with the team; `basil publish` puts it
in front of the public. Plus the two things that make that split comfortable to live with:
telling developers when the live site has fallen behind, and an optional formatting gate
that keeps shared history clean without ever blocking a release.

## User Story

As a developer, I want the command I type twenty times a day to be the harmless one, and
publishing to the world to be something I choose, see, and confirm.

## Motivation

If `main` publishes, then the default action — the short one, the one in muscle memory,
the one an editor's Git button runs — is the one that puts work in front of the public.
That is backwards. The safe action should be the one you get for free.

The failure modes are asymmetric and that is what decides it. A site running a few hours
behind is visible, private, and fixed with one command. Something published that shouldn't
have been is out in public and cannot be recalled. The validation gate does not help here:
it catches code that is *broken*, not work that is merely *unfinished*. A half-finished
redesign parses perfectly.

The obvious cost — the site can silently fall behind — is a tooling problem, not a design
problem, and this unit buys it off.

## Acceptance Criteria

### `basil publish`

- [ ] Pushes the current commit to the release branch on the configured server
- [ ] Shows what is about to be published before doing it: commit range and changed files
- [ ] Prompts for confirmation; `--yes` skips it for scripted use
- [ ] Streams the server's validation output as it arrives
- [ ] Reports the deployed commit on success and the reason on failure
- [ ] Exits non-zero on rejection
- [ ] Works from any clone of the site, with no prior setup beyond the clone — it
      configures the release refspec on first use
- [ ] `--dry-run` shows what would be published and stops

### Drift reporting

- [ ] `basil status` reports what is live, what the release branch points at, and how far
      apart they are
- [ ] `basil publish` reports drift as part of its summary
- [ ] `basil --dev` mentions at startup when the live site is behind the local branch
- [ ] Drift reporting degrades gracefully when the server is unreachable — a warning, not a
      failure

### `basil fmt`

- [ ] Formats `.pars` files: `-w` in place, `-l` to list unformatted, `-d` to diff
- [ ] Matches `pars fmt` behaviour exactly (`cmd/pars/main.go:863`)
- [ ] Operates on a directory tree, not just named files, so `basil fmt -w` is useful

### Formatting

- [ ] `basil --init` installs a **pre-commit hook** in the repository it creates, so
      committed code is formatted without anybody deciding anything
- [ ] The server **warns** about unformatted `.pars` files in a push and **never rejects**
- [ ] The warning names the fix: `Run 'basil fmt -w'`
- [ ] The server never rewrites code — it reports, it does not fix
- [ ] **There is no `git.fmt_check` setting.** A gate a team "should turn on" is a default
      that is not the recommendation

### Documentation

- [ ] The two-verb workflow is the documented default
- [ ] `deploy.branch: main` is documented as the supported push-to-publish alternative
- [ ] The docs show a working setup with **no `basil.yaml` changes at all**

## Design Decisions

- **A verb, not a refspec.** `git push origin main:live` does exactly the right thing and
  is a bad interface: refspec syntax would be the first genuinely new thing a developer has
  to learn, for the most consequential command in the system. It also cannot show what is
  about to happen, and cannot report back well. `basil publish` is one word, and the binary
  is already on the machine because they run `basil --dev`.

- **Underneath it is still a push.** Raw Git and editor Git panels stay a working
  alternative for anyone who prefers them. The verb is ergonomics, not a new protocol.

- **Formatting is fixed locally, before the commit exists.** An earlier draft made this a
  server-side gate with a `git.fmt_check` setting, off by default — which failed its own
  test: if a team *should* switch it on, the default was not the recommendation. A
  pre-commit hook installed by `--init` gets the same outcome with no setting, and nobody is
  ever refused by a server over whitespace.

- **Installing the hook is scaffolding, not presumption.** `--init` creates the repository;
  putting a hook in it is part of creating it. This would be rude only if it reached into a
  repository someone else had made.

- **Confirmation is on by default.** The whole point of the unit is that publishing is
  deliberate. `--yes` exists for scripts.

## Technical Context

### Files

| File | Change |
| --- | --- |
| `cmd/basil/publish.go` (new) | `publish`, client-side `status` |
| `cmd/basil/fmt.go` (new) | `basil fmt`; share the implementation with `cmd/pars` |
| `pkg/parsley/formatter/` | Existing formatter; extract a shared entry point if needed |
| `server/deploy/hooks.go` | Pre-commit hook template; `pre-receive` warns on unformatted files |
| `cmd/basil/init.go` | Install the pre-commit hook |
| `cmd/basil/main.go` | Subcommand dispatch |
| `docs/guide/git.md` | The two-verb workflow |

### Where the remote comes from

`basil publish` runs in a clone, so `origin` already points at the server. Read the release
branch from the server rather than requiring local config — one round trip, no state to
drift. Cache it if that proves slow.

### Sharing the formatter

`pars fmt` already exists with `-w`, `-d`, `-l`. Prefer extracting a shared package over
shelling out from `basil` to `pars`, so a Basil install does not require the `pars` binary.

### Tests

- `publish` against a fixture server: success, rejection, dry-run, `--yes`
- Confirmation prompt is not skippable without `--yes`
- Drift reporting: ahead, behind, level, and server unreachable
- `--init` installs a working pre-commit hook that formats on commit
- An unformatted push is warned about and **succeeds**
- The warning names the fix; non-`.pars` files are ignored

## Out of Scope

- A `git-publish` shim on `PATH` — `basil publish` is canonical (decided 2026-08-24)
- An admin panel view — there is no production admin panel; see `DESIGN-git-deploy.md` §10
- Auto-formatting on the server — ruled out architecturally (D4b)

## Dependencies

- **FEAT-154** — the release branch and the hooks it extends

## Definition of Done

Project checklist (`CLAUDE.md`) plus:

- [ ] `go build ./...` and `go test ./...` pass
- [ ] Full manual round trip performed against a running server: `git push` shares and
      publishes nothing; `basil publish` releases; a broken release is rejected with the
      reason visible; drift is reported accurately
- [ ] Pre-commit hook verified to format on commit in a freshly initialised site
- [ ] An unformatted push verified to warn and succeed, never to fail
- [ ] `docs/guide/git.md` rewritten around the two verbs, and every command in it run
- [ ] `basil --help` output covers the new subcommands
- [ ] `CHANGELOG.md` entry under `## [Unreleased]`
- [ ] Merged to `main` and pushed; worktree and branch removed

## Open Questions

1. Should `basil publish` refuse when the working tree is dirty? Recommend warning, not
   refusing: publishing a committed state with uncommitted local edits is legitimate, but
   surprising often enough to mention.
2. Does `basil status` need to work without network access? Recommend yes, reporting local
   state and noting the server was unreachable.
