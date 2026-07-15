# Basil — working notes for Claude

Basil is a web framework written in Go. The Parsley language lives in `pkg/parsley`,
the HTTP server in `server/`, user-facing docs in `docs/`, and process artefacts
(bugs, plans, specs, backlog) in `work/`.

## Build & test

- Build: `go build ./...`
- Test: `go test ./...`

Both must pass before a unit of work is done.

## Workflow: integrate early, keep `main` current

Work happens in per-session git worktrees under `.claude/worktrees/`. The failure mode
to avoid is letting these pile up unmerged: when several worktrees touch the same files
they diverge, and integrating them later is slow and error-prone. (We once accumulated
eight, four of them editing the same search files — don't repeat that.)

Rules:

- **One in-flight unit per subsystem.** Merge to `main` before starting parallel work
  that touches the same area (e.g. don't run two units editing `server/search.go` at once).
- **Rebase on `main` frequently** so integration conflicts stay small.
- **Merge model is direct-to-`main`.** Once a unit meets the Definition of Done, merge it
  straight to `main` and push — no PR required.

## Definition of Done

A unit of work is **not done** until every box is checked. Run `/done` to execute this
checklist mechanically.

- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Behaviour verified in the running app (if it has runtime surface — use `/verify` or `/run`)
- [ ] Docs updated (relevant `docs/basil/manual/` pages) if API or behaviour changed
- [ ] `CHANGELOG.md` has an entry under `## [Unreleased]`
- [ ] Work committed on its branch; working tree clean
- [ ] Branch rebased on the latest `main`, then merged to `main`
- [ ] `main` pushed to `origin`
- [ ] Worktree and branch removed

"Code written and tests pass" is **not** done. **Merged, pushed, and cleaned up is done.**
