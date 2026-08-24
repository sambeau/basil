# Design: Git Deploy — Basil as hub and deployer

**Status:** Final — ready for specification
**Created:** 2026-08-24
**Author:** Discussion between @sambeau and AI; decisions by @sambeau
**Predecessor:** `DESIGN-git-deploy-topology.md` (options, rationale, rejected alternatives)
**Supersedes:** `remote-workflow-design.md` (topology only — its auth model still stands)
**Related:** FEAT-035 (Git over HTTPS — superseded in part), BUG-033 (blocking bug in the
current implementation)

This document records only the agreed design. Where a decision had alternatives worth
knowing about, the reasoning lives in the predecessor document rather than here.

---

## 1. Summary

Basil's production server holds a **bare Git repository** that developers push to, and a
**separate live release** that Basil materialises when the release branch moves.
Publishing is an explicit act (`basil publish`), distinct from sharing work (`git push`).

The server is therefore three things at once, deliberately and separately: the team's
shared Git host, the gate that checks a release before it goes out, and the web server
that serves it. It needs no third-party service, no second machine, no extra port, and no
CI system.

```
git push            →  shared with the team. Not public.
basil publish       →  checked, then live.
```

---

## 2. Decisions

| # | Decision | Note |
| --- | --- | --- |
| D1 | **Server holds a bare repo plus a separate live release.** | Option 3 in the predecessor. |
| D2 | **Push is the transport; the server is not `origin` in doctrine.** | It is a hub that also deploys. |
| D3 | **Publishing is an explicit verb.** `git push` shares; `basil publish` releases. | Default `deploy.branch: live`. `main` supported for push-to-publish. |
| D4 | **A release is validated before it is activated**, and a failed check rejects the push. | Correctness only. On by default. |
| D4a | **A formatting check is available on push, for all branches.** | Style only. Off by default; one line to enable. §6.3. |
| D4b | **The server never rewrites code.** A release is byte-identical to its commit. | Architectural. §6.3. |
| D5 | **Full code/state split, done now.** Releases are directories; activation is a swap. | No installed base to migrate, so no reason to defer. |
| D6 | **Rollback is a first-class operation.** | Previous releases retained. |
| D7 | **Deploys are recorded.** | CLI-visible; feeds a future UI. |
| D8 | **Pull-from-upstream (GitHub etc.) is out of scope.** | Nothing in the current use cases needs it. §10. |
| D9 | **No admin panel work.** | There is no production admin panel to add to. §10. |
| D10 | **Git over SSH is out of scope.** | HTTPS on 443 is the friendliest transport. |

Also settled, and load-bearing for scoping: **there is no installed base.** No live site
uses Git deploy, and the Basil website runs on GitHub Pages. There is no migration path to
build and no compatibility surface to preserve.

---

## 3. Vocabulary

Used consistently below, and in user-facing text.

| Term | Meaning |
| --- | --- |
| **Site root** | The directory an operator points Basil at. Contains everything below. |
| **Repository** | The bare Git repo on the server. One per site. Served at `/.git`. |
| **Release branch** | The branch nominated as "what should be live". Default `live`. |
| **Release** | One deployed commit, materialised as a directory. |
| **Publish** | Move the release branch to a new commit, thereby requesting a deploy. |
| **Deploy** | What the server does in response: check, activate, reload, record. |
| **Data root** | Persistent state. Never touched by a deploy. |

**On "ref":** a ref is a name that points at a commit, and a branch is one — `live` is a
small file containing a commit ID. The implementation speaks in refs, because tags are refs
too and someone will eventually want to release from a tag. **Configuration and
documentation speak in branches.** `deploy.branch: live`, never `deploy.ref:
refs/heads/live`, though the long form is accepted.

---

## 4. On-disk layout

```
/srv/mysite/                      ← the site root; `basil` is pointed here
  site.git/                       bare repository, served at /.git
  releases/
    4f2a1c9…/                     one directory per deployed commit
    a1b2c3d…/
  current -> releases/4f2a1c9…    the live release
  data/                           never touched by a deploy
    data.db                       + -wal / -shm
    .basil-auth.db
    certs/                        Let's Encrypt account key and certificates
    logs/
    cache/images/
    uploads/                      anything the site itself writes
```

### 4.1 What moves, and why it must

`config.BaseDir` currently means "the directory containing `basil.yaml`", and *everything*
resolves against it — code and state alike. That single anchor becomes two:

- **Release directory** — the site's code. Replaced on every deploy. Treat as read-only at
  runtime.
- **Data root** — everything that must survive a deploy. Default `<site root>/data`,
  overridable.

Everything in this table currently lives in the release directory and must resolve against
the data root instead:

| What | Today | Why it matters |
| --- | --- | --- |
| Auth database | `<BaseDir>/.basil-auth.db` | Holds the API keys people deploy *with*. Losing it locks everyone out. |
| Application database | `database.path` | Plus `-wal` / `-shm`. Sessions live here when `session.store: sqlite`. |
| TLS certificates | `https.cache_dir`, default `certs` | Re-issuance is rate-limited by Let's Encrypt, so this is not merely regenerable. **Currently resolves against the process working directory, not `BaseDir`** — fix this inconsistency here. |
| Logs | `logging.output`, `logging.parsley.output`, `dev.log_database` | |
| Image cache | `images.cache_dir`, default `./cache/images` | Not covered by the `.gitignore` that `--init` ships — see BUG-033. |
| Search indexes | Site-chosen path | Rebuildable, at a cost. |
| Site-written files | Wherever the site's code puts them | The hard one — see §4.2. |

### 4.2 Files the site itself writes

Parsley has file I/O, so a site can write uploads and generated files at runtime. Today
these typically land under `public/`, which is inside the release and would be destroyed by
the next deploy.

**Decision:** expose the data root to site code, and serve a configured uploads directory
under a URL prefix so it never needs to sit inside `public/`. Basil already serves
generated assets this way (`/__p/`, `/__img/`), so the pattern exists.

The exact Parsley-facing surface — a `basil.data_dir` value, a path helper, or a restricted
write root — is left to the spec. The requirement is: **a site must have somewhere durable
to write that a deploy will not delete, and it must be easy to find.**

### 4.3 Retention

Keep the last *N* releases (default 5) and prune the oldest. Retention is what makes
rollback instant, so it is not merely housekeeping.

---

## 5. Developer workflow

This section is the source of the spec's acceptance criteria.

### 5.1 One-time, on the server

The only place a shell on the server is required.

```bash
basil --init mysite               # creates site root, repo, data root, config
basil users add sam --role admin
basil apikey create sam           # bsk_… — shown once
# enable git in basil.yaml, start the service
```

`basil --init` must now also create the bare repository and the data root, and must not
leave the repository in the state described by BUG-033.

### 5.2 One-time, per machine

```bash
git clone https://sam@mysite.example.com/.git mysite
Password: <paste the API key>     # OS keychain remembers it
```

Any number of machines and any number of developers do this. Nothing about the second one
is different from the first.

### 5.3 The daily loop

```bash
basil --dev                       # local preview, live reload
# ...edit...
git add -A
git commit -m "New homepage"
git push                          # shared with the team. Not public.
basil publish                     # checked, then live.
```

`basil publish` shows what is about to go out and asks:

```
Publishing 3 commits to mysite.example.com  (a1b2c3d..4f2a1c9)
  site/index.pars, site/about/about.pars, public/style.css
Continue? [y/N] y
remote: Checking release 4f2a1c9…
remote:   14 pages, 3 parts, 1 layout — ok
remote: Deployed 4f2a1c9 (0.4s)
Live: 4f2a1c9
```

Under the hood this is a push to the release branch, so raw Git and editor Git panels
remain a working alternative for anyone who prefers them.

### 5.4 A rejected release

```
remote: Checking release 9e8d7c6…
remote:   site/about/about.pars:14 — unexpected '}'
remote:
Release rejected. The live site is unchanged (still a1b2c3d).
```

The release branch never moved. Nothing was activated. The developer sees the reason in
the terminal they typed into, not in a server log.

### 5.5 Sharing without publishing

```bash
git checkout -b new-shop
git push                          # on the server, published to nobody
# ...from another machine, or a colleague:
git fetch && git checkout new-shop
# ...when everyone's happy:
git merge new-shop
basil publish
```

### 5.6 Drift

Because publishing is deliberate, the live site can fall behind the release branch. The
tooling removes the surprise: `basil publish` and `basil status` report it, and `basil
--dev` mentions it at startup.

```
live is 3 commits behind main
```

### 5.7 Rolling back

- **From any machine:** revert the commit and `basil publish` again. Goes through the same
  checks as anything else. This is the normal path.
- **On the server:** `basil rollback` re-points `current` at the previous release. For when
  the site is broken and the answer needs to take two seconds.

---

## 6. The deploy engine

One engine; triggers plug into the front of it. This separation is the point — it is what
made adding other triggers cheap in the discussion, and it keeps the logic in one place.

```
trigger      release branch moved | basil deploy <sha> | basil rollback
   ↓
resolve      branch or ref → commit
   ↓
lock         one deploy at a time per site
   ↓
materialise  extract the commit into releases/<sha>/
   ↓
validate     parse every .pars; load the config; optional site gate
   ↓
activate     re-point `current`; update the running root; clear caches
   ↓
hook         optional post-deploy script (migrations, cache warm)
   ↓
record       sha, author, timestamp, duration, outcome
   ↓
failure      previous release stays live; reasons returned to the trigger
```

### 6.1 The trigger

**The trigger is the release branch moving to a new commit.** A push that does not move it
changes nothing — feature branches are stored and published to nobody.

Implemented with real Git hooks, not the `go-git-http` `EventHandler`. `go-git-http` shells
out to the real `git` binary, so repository hooks fire, which buys three things the current
callback cannot provide:

- **`pre-receive` can refuse.** It receives `<old-sha> <new-sha> <ref-name>` per ref on
  stdin. If the ref is the release branch, validation runs here; a non-zero exit rejects
  the push, and the branch never moves.
- **Hook output reaches the developer.** Git relays it as `remote:` lines.
- **Ordering is guaranteed.** Activation happens in `post-receive`, not racing alongside.

Both hooks are thin scripts calling `basil deploy --from-hook`. The binary is already on
the box.

### 6.2 Validation — correctness, at publish

Basil owns the Parsley parser, so a release can be checked before it is activated: parse
every handler, part and layout, and load the config. This is a build check with no build
server, and it is the main thing a general-purpose Git host cannot offer. **On by default**
(`deploy.validate`).

Be clear about the limits, in the docs as well as here: **validation catches code that is
broken, not work that is unfinished.** It is not a substitute for the explicit publish step
(D3) — the two protect against different mistakes.

### 6.3 Formatting — style, at push

Consistent formatting matters most for exactly the reason it is wanted here: unformatted
code makes diffs noisy and `git blame` useless, and that cost lands on shared history, not
on the running site. So the formatting gate belongs at a different point from validation.

**The server never rewrites code.** A release directory must be byte-identical to the
commit it claims to be, or rollback, the deploy record and reproducibility all become
lies. Format-on-receive is therefore ruled out architecturally, not merely declined.

That leaves the right shape:

| Gate | Runs on | Checks | Default |
| --- | --- | --- | --- |
| Formatting | **every push, any branch** | style | **off** |
| Validation | **release branch only** | correctness | **on** |

Checking format at push rather than at publish is deliberate, and it works out neatly:

- A commit enters shared history at the moment it is pushed. That is exactly when style
  matters and exactly when it is cheapest to fix.
- Publishing never fails for a cosmetic reason. Blocking a production release — possibly a
  hotfix — because of whitespace would be a category error.
- Because everything in the repository has already passed the check, publish never
  encounters unformatted code anyway. The two gates do not overlap.

The mechanism exists: `pars fmt -l` already lists files whose formatting differs
(`cmd/pars/main.go:863`), which is the `gofmt -l` idiom. The check examines **only the
`.pars` files touched by the pushed commits**, so enabling it on an existing repository
does not reject history nobody has looked at.

**Off by default** because a solo developer on their own site does not need it, and being
refused by a server over whitespace is obnoxious when nobody else is reading the diffs. One
line turns it on, and a team should.

The server check is a backstop, not the primary mechanism. Formatting belongs on the
developer's machine, before the commit exists: ship `basil fmt`, and have `basil --init`
offer to install a pre-commit hook. A rejection message must name the fix:

```
remote: 2 files are not formatted:
remote:   site/index.pars
remote:   site/about/about.pars
remote: Run `basil fmt -w` and amend, or set git.fmt_check: false
```

### 6.4 Activation

Activation re-points `current` at the new release directory and updates the running
server's root path, then clears the script, response and fragment caches. Requests already
in flight have resolved their paths and complete against the release they started on.

### 6.5 Locking and failure

- One deploy at a time per site. A concurrent trigger waits or is refused cleanly; it never
  interleaves. A file lock in the site root is sufficient at this scale.
- Any failure before activation leaves the previous release live and untouched.
- A failed release directory is removed, not left half-built.

---

## 7. Configuration

```yaml
git:
  enabled: true
  require_auth: true
  repo: ./site.git              # bare repository (default: <site root>/site.git)
  fmt_check: false              # reject pushes containing unformatted .pars files

deploy:
  branch: live                  # what goes live. Set to `main` for push-to-publish.
  validate: true                # parse-check the release before activating
  keep: 5                       # releases retained for rollback
  hook: ./deploy.pars           # optional post-deploy script

data_dir: ./data                # persistent state; never replaced by a deploy
```

`basil.yaml` ships inside the release, so config changes are versioned and roll back with
everything else. Everything it points at for persistent state resolves against `data_dir`,
which is outside the release. Secrets continue to use the existing `!secret` mechanism and
must not be committed.

**Implementation note for the spec:** because server settings (port, TLS) live in the
versioned config, a deploy can in principle change the listener. Decide whether such a
change requires a restart or is applied live, and say so.

---

## 8. CLI surface

Basil already has `users`, `apikey` and `auth` subcommands, so these fit the existing
pattern.

| Command | Runs on | Purpose |
| --- | --- | --- |
| `basil publish` | developer machine | Push the current commit to the release branch and report the result |
| `basil status` | developer machine | What's live, what's on the release branch, how far behind |
| `basil deploy <sha\|branch>` | server | First deploy; deploy a commit already in the repo |
| `basil rollback [sha]` | server | Re-activate the previous (or a named) release |
| `basil releases` | server | The deploy record |
| `basil fmt` | developer machine | Format `.pars` files; `-w` in place, `-l` to list |
| `basil deploy --from-hook` | server | Internal; invoked by the Git hooks |

---

## 9. Authentication

Unchanged from FEAT-035, which works and is not in question: HTTP Basic over TLS, API key
in the password field, validated against the auth database. `editor` or `admin` to push,
any authenticated user to clone.

One addition: **publishing may warrant a higher bar than pushing.** Under D3 these are now
different acts, so the roles can differ — for example `editor` to push a branch, `admin` to
move the release branch. Left to the spec; the mechanism (checking the ref name in
`pre-receive`) is already required by §6.1.

---

## 10. Out of scope

Recorded so the spec does not have to re-litigate them.

| Not doing | Why | What would bring it back |
| --- | --- | --- |
| Pull from GitHub/GitLab (webhook or poll) | Nothing in the current use cases needs it, and it introduces a third-party dependency | Deploying one site to several servers, which push handles badly |
| Push from CI | Same | An external check that must gate releases |
| Admin panel deploy view | **There is no production admin panel.** `/__/` dev tools are mounted only when `Server.Dev` is true (`server.go:618`), so this would mean building an authenticated production UI first | A production admin panel existing for other reasons. The deploy record (D7) is designed to feed one |
| Git over SSH | Extra daemon, port and key management; 443 is the friendliest transport | Someone asking for key-based auth specifically |
| Branch → environment mapping (staging) | Two Basil instances already achieve it, and `basil publish` against a second server covers promotion | Demand for many environments per host |
| Git LFS, submodules | Not supported. Say so in the docs rather than letting people discover it | |

---

## 11. Left to the spec

Genuinely implementation-level; none of these change the shape above.

1. **The Parsley-facing data path** (§4.2) — value, helper, or restricted write root.
2. **Uploads URL prefix** — reuse the `/__p/` pattern or introduce a configured one.
3. **Whether publishing requires a higher role than pushing** (§9).
4. **Whether a config change to server settings requires a restart** (§7).
5. **Release directory naming** — full SHA, short SHA, or sequence plus SHA.
6. **What `basil --init` produces** now that the layout has four top-level entries.
7. **Whether `basil fmt` wraps `pars fmt` or shares its implementation** (§6.3), and
   whether `basil --init` installs a pre-commit hook by default or offers to.

---

## 12. Before any of this: BUG-033

The current implementation does not work as documented — the repository is never created
and receive behaviour is never configured, so a first push is rejected. That fix is small,
independent, and should not wait for this design. It also does not conflict with it: the
layout here removes the whole class of problem by never pushing into a checked-out tree.
