---
id: FEAT-156
title: "Init defaults: a simple local site by default, server topology opt-in"
status: implemented
priority: high
created: 2026-08-25
implemented: 2026-08-25
author: "@sambeau / @claude"
related: FEAT-152, FEAT-153, FEAT-154, FEAT-155, PLAN-132, BACKLOG #150, BACKLOG #151
---

# FEAT-156: Init defaults — a simple local site by default, server topology opt-in

## Summary

`basil --init` currently produces only the server-side deploy topology from FEAT-152:
bare repository, `releases/`, `current` symlink, `data/`, auth database, one-time API
key — with `--host` and `--admin` mandatory and `git` a hard requirement. That is the
right shape for the machine that *receives* deploys, and the wrong shape for everyone
else. A hobbyist starting a local site gets a miniature deployment server on their
laptop, whether they want one or not.

This feature splits init into two modes:

- **Local (the default).** `basil --init mysite` produces the simple folder Basil has
  always been about: `basil.yaml`, `site/`, `public/`. No flags required, no API key,
  no bare repository. It runs immediately with `basil --dev`.
- **Server (opt-in).** `basil --init mysite --server --host … --admin …` produces
  exactly what today's init produces, unchanged. It is run on the deployment box.

Nothing about the deploy engine changes. The insight this feature encodes is that the
architecture *already* separates simple-local from complicated-remote: the site-root
tree is a server-side artifact, and a developer's copy of a deployed site is just a
clone — `basil.yaml`, `site/`, `public/` — which is the simple folder. FEAT-155's
`basil publish` already assumes this ("works from any clone with no prior setup beyond
the clone"). Local init simply got deleted rather than differentiated; this puts it
back as the default.

It also settles a decision the split exposes: **a release must not be able to disable
the server's deploy mechanism.** `git.*` and `auth.*` become operator-owned on a site
root — a deployed `basil.yaml` that omits or disables them cannot turn off the git
endpoint it arrived through. FEAT-154 has shipped, so this enforcement closes a live
gap rather than gating a future one; it is the reason this feature is the programme's
next unit.

This spec is the design @sambeau committed to propose for **backlog #150**
(simple-by-default init). Its graduation path is the server-side half of the companion
item **#151** (a local project publishing to a remote hub); #151's client ergonomics
(`basil publish --to`, a stored `deploy.remote`) remain backlogged and out of scope
here.

## User Story

As a hobbyist or single developer, I want `basil --init mysite` to give me a plain
folder of files I can understand at a glance and run with `basil --dev`, so that
starting a Basil site feels like the old CGI/PHP days — a folder with files in it —
and not like provisioning infrastructure.

As that same developer later, I want to graduate my site to a real server by running
one init command on the box and pushing, so that going public never requires
restructuring the folder I have been working in.

As a Basil operator, I want the server topology to remain exactly what FEAT-152
built, so that nothing about deploys, releases, or rollback changes underneath me.

## Motivation

From the problem statement (2026-08-25):

> The new git deploy feature is awesome. However, we have inadvertently trampled on
> the basic use case while catering for the more specialised remote deploy use case.
> … The simplicity of a site folder with files in it was a strength of Basil —
> reminiscent of old CGI sites and PHP. I would like us to be able to keep that.

The runtime already keeps it: FEAT-152 explicitly preserved the legacy layout
(`basil --dev` in a plain folder, `DataDir` defaulting to the project directory,
verified byte-identical against the pre-merge binary). And relative persistent paths
(`database.path: ./data.db`) resolve against `DataDir` in both layouts, so the same
`basil.yaml` means the same thing locally and on a server. The simple world survived
everywhere except the one command that creates sites. `cmd/basil/init.go` refuses to
run without `--host`, refuses without `--admin`, refuses without `git`, and its error
text nudges the local user into building server topology ("use `--host localhost` for
a site you will only run with `--dev`"). The design docs (§5.2 of
`DESIGN-git-deploy-topology.md`) show `--init` being run over SSH on the server —
that is the use case it was written for, and now the only one it serves.

## Design

### The two modes

| | Local (default) | Server (`--server`) |
| --- | --- | --- |
| Command | `basil --init mysite` | `basil --init mysite --server --host h --admin a` |
| Runs on | the developer's machine | the deployment box |
| Layout | `basil.yaml`, `site/`, `public/` | `site.git/`, `releases/`, `current`, `data/` |
| `--host` | optional, default `localhost` | required |
| `--admin` | refused (server-only) | required (or prompted interactively) |
| git | optional nicety | hard requirement |
| Auth DB / API key | none | created, key printed once |
| Starts with | `basil --dev` (in the folder) | `basil --site <root>` |
| Config layout | legacy: `DataDir` = project dir | site root: `DataDir` = `<root>/data` |

### Local mode in detail

`basil --init mysite` creates:

```
mysite/
├── basil.yaml          minimal: server.host localhost, site.path, public_dir
├── .gitignore          runtime state patterns (see below)
├── .githooks/
│   └── pre-commit      the FEAT-155 formatting hook (only when git is present)
├── site/
│   └── index.pars      the starter page
└── public/
    └── .keep
```

- **The config is minimal.** `server.host: localhost`, `site.path: ./site`,
  `public_dir: ./public`, basic logging. No `auth:` block, no `git:` block, no
  `https:` block — none of them mean anything until the site has a server, and
  operator-owned enforcement (below) means their absence can never disable anything
  remote. Commented-out `database:` and `data_dir:` stanzas stay as documentation,
  as today, joined by a commented `developers:` example (see "One config file",
  below) so the layering discipline is discoverable from day one.
- **Git is a quiet nicety, not a gate.** When `git` is on the PATH: `git init`
  (normal, not bare, initial branch `main`), set `core.hooksPath .githooks`, make the
  first commit. This costs the hobbyist nothing visible and means the folder is
  already clone-shaped the day they decide to deploy. When `git` is absent: skip all
  of it silently — no warning, no refusal. A `--no-git` flag opts out explicitly.
- **The local `.gitignore` lists runtime state — deliberately unlike the server
  one.** In the legacy layout `DataDir` *is* the project directory, so the auth
  database, dev log database, application database and sidecars, `certs/`, `cache/`,
  `search/` and `uploads/` all land next to the code. The server-mode `.gitignore`
  says "nothing else belongs here"; the local one must guard the future push path
  instead:

  ```gitignore
  .basil-auth.db*
  dev_logs.db*
  *.db
  *.db-wal
  *.db-shm
  certs/
  cache/
  search/
  uploads/
  .DS_Store
  *.swp
  ```

- **The summary output matches the mode.** No API key, no clone command. Print the
  layout, then: `cd mysite && basil --dev`, and one line pointing at the graduation
  path ("when you want to put it on a server: basil --init <dir> --server on the
  box, then connect this folder — see the deploy guide").

### Server mode in detail

Byte-for-byte today's `runInitCommand`, behind the flag. Everything FEAT-152 built —
the deadlock-breaking release 1, config verification before the credential is spent,
root-safety checks, ownership hand-over, the printed clone command — is kept without
modification. `--host` and `--admin` remain required exactly as now; supplying them
without `--server` (and without `--init`) remains an error.

### Graduation: connecting a local site to a new server

The local folder never restructures. The path is:

```bash
# on the server (once)
basil --init /srv/mysite --server --host mysite.example.com --admin sam

# in the local folder (once)
git remote add origin https://sam@mysite.example.com/.git
git push -u origin main
```

then `basil publish` (FEAT-155) — or `git push origin main:live` — to go live.

One mechanical wrinkle: server init creates its own starter commit on the release
branch, so the developer's history and the hub's history are unrelated, and the first
publish is a non-fast-forward update of `live`. The shipped hub refuses release-branch
force-pushes for everyone (`cmd/basil/fromhook.go:163`, FEAT-154) — correctly, since
the deploy record and rollback rely on release history — which means graduation is
currently **impossible** without shell access. This feature amends that check with
exactly one exception: **a non-fast-forward is accepted when `live` still points at
the starter commit created by `--init` and nothing else has ever been deployed** (the
deploy record knows — release 1 is seeded with trigger `init`). After that, the
refusal applies as shipped.

### One config file: it describes the site, not a machine

Once a site syncs between a production server and one or more developer machines,
the versioned `basil.yaml` serves several environments at once. The rule that makes
one file work — rather than splitting into local/remote files or a gitignored
overlay — is a layering discipline Basil already has the machinery for:

| Layer | Mechanism | Versioned? |
| --- | --- | --- |
| Production truth | top-level keys (`host`, `https`, `port`, …) | yes |
| Mode | `--dev` (HTTP, localhost, live reload; ignores the production listener) | ambient |
| Per-person | `developers.<name>` profiles, activated with `-as <name>` | yes, deliberately |
| Per-run | CLI flags (`--port 3000`) | no |
| Secrets | `!secret` (resolved from the environment) | no |

Nobody edits a top-level value to suit their machine: a developer's port lives
under `developers.<name>.port`, where it cannot collide with anyone else's and is
visible to the whole team. A gitignored local overlay file was considered and
rejected: it is invisible state ("works on my machine"), it cuts against the
folder-you-can-read ethos, and every legitimate case it would serve is already
covered by a layer above.

The discipline is social, not enforced — someone will eventually commit a top-level
`port:` edit. Two backstops make that a caught mistake instead of an outage: the
operator-owned settings below, and the FEAT-153 validation gate flagging listener
changes (also below).

### Operator-owned settings: a release cannot disable the deploy mechanism

`basil.yaml` ships inside the release (FEAT-152 design decision), and today's
generated config carries `git.enabled: true` and `auth.enabled: true`. Once local
configs legitimately omit those blocks, the first push from a graduated local site
would deploy a config with git disabled — bricking the deploy mechanism from inside a
deploy, with recovery requiring SSH.

**Decision (proposed): when serving a site root, the server forces `git.enabled`,
`git.require_auth` and `auth.enabled` on, regardless of what the active release's
config says.** A release config that explicitly sets any of them false gets a startup
warning naming the override; omission is silent. In the legacy layout nothing is
forced — a local dev server with no git endpoint is correct.

This is the narrow version of FEAT-152 Open Question 1 (may a release change server
settings?). It does not settle the general question — port, TLS and listener settings
stay deployable — it fences off only the settings whose loss is unrecoverable
without a shell. The hub (FEAT-154) is already live, so this gap is open in shipped
code today; enforcement is the most urgent piece of this feature.

**Decision (@sam, 2026-08-25): listener changes are gated, not forced.** `server.host`
stays deployable — renaming a site over git is legitimate — but the graduation path
makes deploying a dev listener onto a public server an easy accident: a graduated
local config says `host: localhost`, port 8080, no `https:`, and on the next restart
the public site (and its git endpoint) would be gone. So the validation gate
(`server/deploy/validate.go`, built by FEAT-153) must flag a release whose config
changes `server.host`, `server.port` or the `https` block on a public server —
warning by default, in the developer's terminal at push time, where the mistake is
one commit deep and trivially reverted. FEAT-153 shipped before this decision was
taken, so **this feature implements the check** in the pipeline FEAT-153 built;
FEAT-153's spec records the hand-off under Out of Scope.

## Acceptance Criteria

### Local mode (default)

- [ ] `basil --init mysite` succeeds with no other flags and creates
      `basil.yaml`, `site/index.pars`, `public/.keep`, `.gitignore`
- [ ] The folder runs immediately: `basil --dev` inside it serves the starter page
- [ ] No auth database, no API key, no `site.git/`, no `releases/`, no `current`,
      no `data/` is created
- [ ] The generated config contains no `auth:`, `git:` or `https:` blocks and is
      loadable and valid (verified before init reports success, as server mode
      already does)
- [ ] The generated config includes a commented `developers:` example showing a
      per-person port override and naming `-as <name>`
- [ ] `--host` defaults to `localhost`; an explicit `--host` is accepted and
      validated with the existing rules
- [ ] `--admin` with local init is refused with an error naming `--server`
- [ ] With git installed: the folder is a normal repository on branch `main`, with
      one commit, `core.hooksPath` set, and the pre-commit formatting hook in
      `.githooks/`
- [ ] Without git installed: init still succeeds; no repository, no hook, no warning
- [ ] `--no-git` skips repository creation even when git is present
- [ ] The local `.gitignore` covers every runtime file the legacy layout writes into
      the project directory: auth DB and sidecars, dev log DB, `*.db`/`-wal`/`-shm`,
      `certs/`, `cache/`, `search/`, `uploads/`
- [ ] The summary prints the layout and `basil --dev` next steps; it prints no API
      key and no clone command
- [ ] The summary always ends with a one-line graduation pointer (server init on
      the box, connect this folder, doc link) — it is the bridge between the two
      halves of the product, and its tone is a signpost, not a nudge

### Server mode (opt-in)

- [ ] `basil --init mysite --server --host h --admin a` produces exactly today's
      output and layout; the existing init tests pass against it unchanged
- [ ] `--server` without `--host` or (non-interactively) without `--admin` is
      refused with today's error text
- [ ] Cleanup-on-failure, root-safety and ownership hand-over behave as today

### Graduation

- [ ] A folder created by local init (with git) can add the server as `origin` and
      push `main` with no local restructuring
- [ ] The one permitted non-fast-forward on the release branch is implemented in the
      receive-hook path (`cmd/basil/fromhook.go`): accepted only while the deploy
      record shows nothing but the `init`-triggered release 1, refused as shipped
      afterwards — with tests for both sides of the boundary
- [ ] A full graduation exercised end-to-end on localhost: local init → server init →
      remote add → push `main` → publish → the local site is live

### Operator-owned settings

- [ ] When started with `--site <root>`, the server has git and auth enabled even if
      the active release's config omits them
- [ ] A release config explicitly disabling them produces a startup warning naming
      the override; the settings remain on
- [ ] In the legacy layout, nothing is forced: `git.enabled` still defaults to false
- [ ] `docs/guide/configuration.md` documents which settings are operator-owned on a
      site root and why
- [ ] The listener-change warning is implemented in the validation gate
      (`server/deploy/validate.go`): a release whose config changes `server.host`,
      `server.port` or the `https` block relative to the active release on a public
      server warns in the push output; a same-listener release stays silent. The
      decision is recorded in `DESIGN-git-deploy.md` §6.2 and handed off in
      FEAT-153's Out of Scope (both done 2026-08-25, alongside this spec)

### Documentation

- [ ] The getting-started path in the docs leads with local init; the deploy guide
      owns `--server` and graduation
- [ ] `docs/guide/configuration.md` gains a section on the one-file layering
      discipline — "the file describes the site, not your machine" — covering the
      top-level / `--dev` / `developers.<name>` / CLI-flag / `!secret` layers
- [ ] `CHANGELOG.md` entry under `## [Unreleased]`

## Design Decisions

- **Local is the default because the default should match the first-run audience.**
  The person typing `basil --init` for the first time is on their own machine. The
  operator setting up a server is following the deploy guide and will not be
  surprised by one flag. Defaults protect the newcomer; flags serve the specialist.

- **One command, two modes — not two commands.** `--init` is already a flag-style
  command (`basil --init FOLDER`); a sibling flag (`--server`) is the smallest
  coherent surface. A separate `basil init-server` binary-style verb would be the
  first of its kind in the CLI.

- **Local init still creates a git repository (when it can).** The frictionless
  local→remote sync the design promises is git; a folder that is already a
  repository with clean history graduates with two commands. The cost to the
  hobbyist is one hidden directory. But git must not be a *gate* locally — a missing
  git binary silently degrades to a plain folder, because the plain folder is the
  product in this mode.

- **The two `.gitignore`s differ on purpose.** The server-mode ignore file is nearly
  empty because state lives outside the repository; the local ignore file is long
  because in the legacy layout state lives inside the project directory. Writing the
  wrong one in the wrong mode either leaks databases into the future push path or
  teaches falsehoods about where state lives.

- **Operator-owned settings are the narrow fence, not the general answer.** Forcing
  only `git.*`/`auth.*` on site roots fixes the self-bricking hazard without
  deciding whether releases may change ports or TLS (FEAT-152 Open Question 1 stays
  open). The forced settings are exactly those whose loss removes the remote
  recovery path.

- **The starter-commit non-fast-forward exception is stated here, implemented in
  FEAT-154.** It is a property of the graduation story, so it belongs in this spec's
  record; it is enforced in receive hooks, which do not exist until FEAT-154.

- **One versioned config file, no local overlay.** The multi-environment problem
  (prod + N developer machines sharing one `basil.yaml` through git) is solved by
  layering, not by file splitting: top level is production truth, `--dev` makes mode
  ambient, `developers.<name>` namespaces per-person settings where they cannot
  collide, flags cover per-run, `!secret` covers secrets. A gitignored
  `basil.local.yaml` was considered and rejected as invisible state. Revisit only if
  real usage surfaces a per-person need no `DeveloperConfig` field covers — the
  answer then is a new field, not an escape hatch that overrides anything silently.

## Technical Context

### Files

| File | Change |
| --- | --- |
| `cmd/basil/init.go` | Split `runInitCommand` into local and server paths; local starter files, local `.gitignore`, local summary; keep server path unchanged |
| `cmd/basil/init_test.go` | Local-mode tests (with/without git, `--no-git`, refusals); assert server-mode tests unchanged |
| `cmd/basil/main.go` | `--server`, `--no-git` flags; flag-combination validation (`--admin` implies `--server`; `--server` implies `--init`) |
| `server/config/load.go` or `server/server.go` | Operator-owned enforcement when the site-root layout is active (the code already knows which layout it resolved — the same knowledge that picks `DataDir`) |
| `server/deploy/validate.go` | Listener-change warning: compare the candidate release's `server.host`/`server.port`/`https` against the active release's |
| `cmd/basil/fromhook.go` | The starter-commit non-fast-forward exception, decided from the deploy record |
| `work/design/DESIGN-git-deploy.md` | Record the operator-owned decision and the first-publish non-fast-forward rule |
| `docs/guide/` | Getting-started leads with local init; deploy guide owns `--server` and graduation |
| `CHANGELOG.md` | `## [Unreleased]` |

### Notes for implementation

- The mode split falls along existing seams: layout creation, repository creation,
  release 1, config verification, admin creation, and the summary are already
  separate steps in `runInitCommand`. Local mode is a subset with different file
  contents, not a rewrite.
- `writeStarterSite` currently couples config content to `isLocalHost(host)`; in the
  new shape the coupling is to the mode, not the hostname. A server init with an
  internal hostname and a local init with `--host mysite.local` must each get their
  mode's config.
- Operator-owned enforcement belongs where the site-root layout is detected, so
  legacy-layout behaviour is untouched by construction. Add a table test alongside
  `server/config/paths_test.go` in the same spirit: one case per forced key per
  layout.

## Out of Scope

- `basil connect <host>` sugar for the two graduation commands — FEAT-155's orbit.
- The receive-hook enforcement of the first-publish exception — FEAT-154.
- The general "may a release change listener settings?" question — still FEAT-152
  Open Question 1.
- Any change to the deploy engine, release layout, or `basil publish`.
- Migrating a legacy-layout site *in place* to a site root (distinct from
  graduation, which targets a fresh server).

## Dependencies

- FEAT-152–155 (all implemented) — both layouts, the deploy engine, the hub and
  `basil publish` exist. This feature builds on all four: it splits the init that
  FEAT-152 wrote, adds a check to the validation gate FEAT-153 built, amends the
  receive-hook refusal FEAT-154 shipped, and completes the graduation story that
  FEAT-155's `basil publish` assumes.
- **Slotted into PLAN-132 as phase 5** (2026-08-25), the programme's next unit. Two
  of its pieces close gaps in shipped code — the operator-owned enforcement (a
  deployed config can currently disable the hub) and the graduation
  non-fast-forward exception (graduation is currently impossible without shell
  access) — which is why it follows immediately rather than waiting.

## Definition of Done

Project checklist (`CLAUDE.md`) plus:

- [ ] `go build ./...` and `go test ./...` pass
- [ ] `basil --init x` → `cd x && basil --dev` serves the starter page, verified in
      the running app
- [ ] Server-mode init verified unchanged against the FEAT-152 verification steps
- [ ] A local site graduated to a fresh server-mode root end-to-end (clone-free
      path: remote add, push) — on localhost is acceptable
- [ ] Operator-owned enforcement covered by table tests for both layouts
- [ ] Docs and CHANGELOG updated; merged to `main`; worktree and branch removed

## Open Questions

All resolved (@sam, 2026-08-25):

1. ~~Flag name~~ — **`--server`.** The command is typed on the box it configures.
2. ~~Graduation pointer: every time, or doc link only?~~ — **One line, every time.**
   It is the bridge between the two halves of the product; init's summary is the one
   guaranteed touchpoint every local user sees, and the cost is one ignorable line,
   once. Doc-link-only would bury the local→remote story in a page the user has no
   reason to open.
3. ~~Should `--host` on local init switch anything besides `server.host`?~~ — **No.**
   Local mode is one shape; `--dev` owns the dev listener.
4. ~~Does the operator-owned set need `server.host` too?~~ — **No: gate, don't
   force.** Renaming a site over git is legitimate, so forcing the host would
   reintroduce SSH for a routine change. The real hazard — a graduated local config
   deploying `host: localhost` onto a public server, taking down the site *and its
   git endpoint* on the next restart — is caught instead by the FEAT-153 validation
   gate flagging listener changes at push time (see Operator-owned settings).

## Implementation notes (2026-08-25, branch `feat-156-init-defaults`)

Implemented by an orchestrated team (two builders, an integration pass, a docs pass,
three review lenses, two fix rounds), each stage verified by the orchestrator.

### Deviations from the spec as written

- **The operator-owned set is `git.enabled` + `auth.enabled` only** — `git.require_auth`
  no longer exists (removed by FEAT-154 after this spec's operator-owned section was
  drafted).
- **The spec's "legacy layout: `git.enabled` still defaults to false" line was stale.**
  Since FEAT-154 it defaults to true everywhere, with the endpoint active only when
  `site.git` exists — in the legacy layout there is no repository, so the key is inert
  there. Recorded as backlog #154 (the key now has no configuration where it does
  anything).
- **Graduation's first publish is `git push --force origin main:live`,** not
  `basil publish`: the histories are unrelated, and `basil publish` fails opaquely on
  that state. The receive-hook exception works as spec'd; the client ergonomics are
  #151's (noted there). Docs describe the forced push honestly, and `basil publish`
  works normally from the second release on.
- **The listener-change comparison compares https by effective shape**, not raw text:
  `https.auto` defaults on, so a config that merely omits the block serves TLS
  identically and stays silent. What warns: auto turned off, or a manual certificate
  appearing/disappearing/moving.
- The local template's one-shape rule (port 8080 whatever the host — Open Question 3)
  survived review with one consequence: the graduation docs instruct setting
  `server.host` **and** `server.port: 443` before the first publish (`https.auto`
  defaults on; `--dev` maps 443 back to 8080 locally, so the edit costs nothing in dev).

### Security-review hardening (beyond the spec)

- The push-triggered `deploy.pars` sandbox now **denies writes to the deploy record and
  the auth database** (+ sidecars) via the evaluator's `RestrictWrite` — the record is
  an authorization input to the starter-overwrite exception and was forgeable from
  inside a pushed release. CLI deploys keep full power.
- A site root **missing its auth database starts degraded** (warning naming
  `basil users create`; auth + git endpoint off for the run) instead of refusing to
  start — forcing `auth.enabled` had turned a dropped dotfile into an outage.
- `data_dir` is **carried across live swaps** with the standard restart-required
  warning. The `deploy` section is deliberately not carried: nothing in the server
  reads it, and every consumer loads from disk per invocation.
- The listener comparison loads configs with **env interpolation disabled** (no secret
  echo into `remote:` output); an active config with a manual certificate and no host
  counts as public; the starter-exception check runs **under the site deploy lock**.

### Deferred, with owners

- Tree-level sandbox gap (recursive removal of `data_dir` bypasses the file-level
  deny list) → backlog **#152** (evaluator change).
- `deploy.branch` is release-controlled and can un-protect the release branch;
  whether it (and `data_dir` at startup) becomes operator-owned → backlog **#153**
  (@sambeau decision).
- `git.enabled` vestigiality → backlog **#154**.
- First-publish `basil publish` ergonomics → **#151** (noted there).
- Local `git init` inherits the `--initial-branch` git ≥ 2.28 floor from server mode;
  an older git errors rather than degrading silently. Accepted, unprobed.
- Pre-existing `main` failures discovered during integration (not this feature's):
  `TestCheckRepoOutsideServedRoots` (real static-root gap in
  `checkRepoOutsideServedRoots`), `TestCurrentLinkWatcherActivatesRepointedRelease`
  (watcher misses the symlink swap — corroborated by a live repro during docs
  verification), and flaky `TestDeployRaceSerialises` (SQLITE_BUSY on concurrent
  `OpenRecord`). All three handed off as separate task chips with diagnoses.

### Verification (orchestrator, observed directly)

- `go build ./...` clean; `go test ./...`: 19 packages ok; the only failures are the
  two pre-existing `main` issues above, with unchanged signatures.
- `basil --init mysite` → three-entry folder, repo on `main`, hook wired; summary has
  no API key, no clone command, one-line graduation pointer. `basil --dev` inside it
  served the starter page (HTTP 200, observed).
- `TestGitE2E_GraduationFromLocalInitToServer` passes: local init → server init →
  remote add → push `main` → one forced `main:live` (starter-overwrite message) →
  server serves the local site's content → a second forced push is refused.
- Operator-owned table tests (11 cases) pass; explicit-false warns, omission silent,
  legacy untouched.
