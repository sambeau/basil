---
name: done
description: Run the project Definition of Done for the current unit of work and integrate it — build, test, verify, docs/changelog, commit, rebase, merge to main, push, and remove the worktree. Use when a piece of work is finished and should land on main. Merge model is direct-to-main (no PR).
---

# /done — finish and integrate the current unit of work

Take the work in the current worktree from "code written" to **merged, pushed, and cleaned
up** on `main`. Merge model is **direct-to-main** — no pull request.

Work through the steps in order. **Stop and ask the user** if a gate fails and the fix
isn't obvious, if a merge conflict needs a judgement call, or if the push is blocked.
Never work around a safety block.

## 0. Orient

- `git rev-parse --abbrev-ref HEAD` — the feature branch (call it `$BRANCH`).
- `git worktree list` — find the worktree checked out on `[main]`; call its path `$MAIN`.
  All merge/push commands run there via `git -C "$MAIN" …`.
- `git status --porcelain` — see what's still uncommitted here.

## 1. Build & test (in the current worktree)

- `go build ./...` — if it fails, stop and fix before continuing.
- `go test ./...` — if it fails, stop and fix.

## 2. Verify behaviour

If the change has runtime surface (anything the server renders, serves, or logs), verify
it in the running app — invoke the `verify` or `run` skill and observe it working. Skip
only for pure docs / test / tooling changes with nothing to exercise.

## 3. Docs & changelog

- If the change adds or alters user-facing behaviour or API, update the relevant
  `docs/basil/manual/` page(s).
- Ensure `CHANGELOG.md` has an entry under `## [Unreleased]` (Added / Changed / Fixed as
  appropriate). Add one if it's missing.

## 4. Commit

If the working tree is dirty, commit everything on `$BRANCH` with a clear conventional
message (`feat(...)`, `fix(...)`, `docs:`), ending with:

```
Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
```

The working tree must be clean before merging.

## 5. Integrate into `main`

Run against the main worktree unless noted:

- `git -C "$MAIN" fetch origin`
- `git -C "$MAIN" merge --ff-only origin/main` — bring `main` current (usually a no-op).
- Rebase the feature branch on the freshened `main` so conflicts surface now and history
  stays clean, from the feature worktree: `git rebase main`
  - Resolve any conflicts thoughtfully (keep every branch's real work), then re-run
    `go test ./...`.
- Merge: `git -C "$MAIN" merge --no-ff "$BRANCH" -m "Merge branch '$BRANCH' (<one-line summary>)"`
  (append the co-author trailer).
- Confirm: no conflict markers anywhere (`grep -rn '^<<<<<<<' .`), and
  `git -C "$MAIN" status --porcelain` is empty.

## 6. Push

- `git -C "$MAIN" push origin main`.
- If the push is blocked by the auto-mode safety classifier, **stop and ask the user** to
  approve it or run it themselves. Do not attempt to route around the block.

## 7. Clean up

- Remove the worktree and delete the merged branch:
  `git -C "$MAIN" worktree remove <feature-worktree-path>` then
  `git -C "$MAIN" branch -d "$BRANCH"`.
- **If this session is running inside the worktree being removed**, do this last and tell
  the user their session's worktree has been removed (the shell's directory is now gone) —
  or leave removal to them and just report the branch is safe to delete.

## Report

Summarise: what merged, the new `main` tip (`git -C "$MAIN" log --oneline -1`), the test
result, and anything skipped or left as follow-up.
