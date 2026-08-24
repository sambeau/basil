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
| D4a | **Formatting is fixed locally by a pre-commit hook `--init` installs.** The server warns, never rejects. | No setting. §6.3. |
| D4b | **The server never rewrites code.** A release is byte-identical to its commit. | Architectural. §6.3. |
| D12 | **Git over plain HTTP is refused, not warned about.** | The API key is a plaintext credential over Basic auth. §9. |
| D13 | **The default configuration is the recommended configuration.** Three keys, none of which a normal site sets. | §7. |
| D14 | **`--init` commits and deploys an initial release**, so a fresh server is never in a state where it cannot serve, cannot get a certificate, and cannot be pushed to. | §5.1.1. |
| D15 | **The certificate is obtained at startup, not on first use**, and `basil check` verifies the preconditions. | §5.1.2. |
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

### 5.1 Bootstrapping a server from clean

The order matters here, and there is one trap worth naming up front.

**Prerequisites** — not Basil's job, but the setup fails confusingly without them:

- a server with a public IP, ports **80 and 443** reachable from the internet
- a DNS `A`/`AAAA` record for the site's hostname pointing at it
- the `basil` binary installed

**Two commands:**

```bash
basil --init mysite --host mysite.example.com
#   ✓ site root, bare repository, data directory
#   ✓ starter site committed and deployed as release 1
#   ✓ admin user 'sam'
#   ✓ API key bsk_…  (shown once — save it now)
#   ✓ pre-commit formatting hook
basil
```

`--init` creating and **deploying an initial release** is not cosmetic — see §5.1.1.

On startup Basil obtains its certificate, logs the result, and serves the starter site. The
developer can then clone.

#### 5.1.1 The cold-start deadlock, and why `--init` deploys

There is no chicken-and-egg problem with the certificate itself. Certificate issuance is
independent of site content: `autocert` obtains one during the first TLS handshake, and the
ACME HTTP-01 challenge is answered over **plain HTTP on port 80** at
`/.well-known/acme-challenge/`, ahead of the HTTPS redirect
(`server.go:1158`, `runHTTPRedirect`).

**But the empty-site state is a genuine deadlock, and this design would create it.** If
`--init` left a bare repository with no commits and no releases, `current` would point at
nothing, the server would have nothing to serve, and:

> no release → nothing to serve → no server → no ACME response → no certificate →
> no HTTPS clone → no push → no release

**`basil --init` therefore commits the starter site to the release branch and deploys it as
release 1.** The server always has something to serve, and `git clone` yields a working
starter site rather than *"you appear to have cloned an empty repository"*.

#### 5.1.2 Issue the certificate eagerly, not on first use

`autocert` issues lazily, on the first TLS handshake for a hostname. Left alone, that makes
**the developer's first `git clone` the request that triggers issuance** — and if ACME fails
(DNS not yet propagated, port 80 blocked by a cloud firewall), the handshake fails and Git
reports an opaque TLS error. The operator cannot tell whether the problem is Basil, DNS, or
their own firewall.

Two requirements follow:

- **Obtain the certificate at startup** for the configured hostname, and log the outcome
  plainly — including which of DNS or port 80 looks wrong when it fails.
- **`basil check`** verifies the preconditions on demand: the hostname resolves to this
  machine, port 80 is reachable from outside, a certificate is present or obtainable, and
  the repository is not inside a served root. This is the command to point people at when
  setup misbehaves.

Requiring a hostname is also a small security fix. `hostPolicy()` returns `nil` when
`server.host` is empty (`server.go:1168`), which tells `autocert` to attempt issuance for
**any** hostname a stranger puts in SNI — a way to burn a site's Let's Encrypt rate limit
from outside. `--host` should be required for a public server.

#### 5.1.3 Without a public hostname

Three supported paths, none of which need ACME:

| Situation | Approach |
| --- | --- |
| Local development | `basil --dev` — HTTP on localhost, Git auth relaxed for localhost only |
| Private network or an existing certificate | `server.https.tls_cert` / `tls_key` |
| Public server, DNS not ready yet | Set it up, then `basil check` before pointing developers at it |

Never work around a failing handshake with `git -c http.sslVerify=false`: the API key is
sent in the request, so disabling verification hands push rights to whoever answered.

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
server, and it is the main thing a general-purpose Git host cannot offer. **Always on** —
there is no config key to disable it; the emergency override is `basil deploy
--no-validate` on the server (§7).

Be clear about the limits, in the docs as well as here: **validation catches code that is
broken, not work that is unfinished.** It is not a substitute for the explicit publish step
(D3) — the two protect against different mistakes.

### 6.3 Formatting — fixed locally, never a blocker

Consistent formatting matters for the reason it is wanted: unformatted code makes diffs
noisy and `git blame` useless. That cost lands on shared history, not on the running site.

**The server never rewrites code.** A release directory must be byte-identical to the commit
it claims to be, or rollback, the deploy record and reproducibility all become lies.
Format-on-receive is ruled out architecturally, not merely declined.

That leaves the question of where the check goes, and the answer is: **not on the server at
all, in the normal case.**

- **`basil --init` installs a pre-commit hook** in the repository it creates. Formatting is
  fixed before a commit exists — the earliest, cheapest point, and one where the developer
  is already looking at the code.
- **The server warns and never rejects.** A push carrying unformatted files reports them and
  proceeds. Publishing must never fail for a cosmetic reason; blocking a hotfix over
  whitespace would be a category error.

An earlier draft made this a server-side gate with a `git.fmt_check` setting, off by
default. That failed its own test: if a team *should* switch it on, the default was not the
recommendation. The hook gets the same outcome with no setting and no way to be refused by
a server over whitespace.

The mechanism exists — `pars fmt -l` lists files whose formatting differs
(`cmd/pars/main.go:863`), the `gofmt -l` idiom. `basil fmt` shares that implementation.

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

**A working site needs none of this.** Every value below has a default that is also the
recommendation. See `reports/GIT-DEPLOY-DEFAULTS-REVIEW-2026-08-24.md` for the settings
that were considered and deliberately not added.

```yaml
deploy:
  branch: live       # what goes live. Set to `main` for push-to-publish.
  keep: 5            # releases retained for rollback

data_dir: ./data     # persistent state; defaults to <site root>/data
```

That is the whole surface. Three keys, none of which a normal site sets.

**Not settings, by decision:**

| Behaviour | How it is decided |
| --- | --- |
| Git deploy on/off | On when `<site root>/site.git` exists, which `--init` creates. `git.enabled: false` remains as an off-switch |
| Repository location | Always `<site root>/site.git`. Never configurable — a repository inside a served root would expose every version of every file |
| Authentication | Always required. Never disableable from config; `--dev` on localhost is the only relaxation, and it is decided in code |
| Validation | Always on for pushes. Emergency override is `basil deploy --no-validate` on the server, which needs shell access and cannot be left on by accident |
| Formatting | `--init` installs a pre-commit hook; the server warns and never rejects. Cosmetics must not block a release |
| Post-deploy script | Convention: `deploy.pars` in the release root, if present |
| Force-push / deleting the release branch | Always refused |

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
| `basil check` | server | Verify bootstrap preconditions: DNS, port 80, certificate, repo placement |
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

Four guarantees, none of them configurable:

1. **Plain HTTP is refused.** Basic auth puts the API key in an easily-decoded header, so
   over plain HTTP it is a plaintext credential with push rights. The current
   implementation logs a warning and proceeds (`server/git.go:180`); a warning in a log
   nobody reads is not a control. Refuse the request, with an error saying why. The only
   exception is a dev-mode localhost bind. Nothing is lost — Basil obtains its own
   certificate and already listens on 443.
2. **No auth database, no Git.** Already the behaviour (`server.go:571`); promoted to a
   stated guarantee with a test.
3. **The repository may not sit inside a served root.** Refuse to start otherwise. The
   repository path is not configurable, but `public_dir` and `site.path` are.
4. **The release branch cannot be force-pushed or deleted.** A rewritable release history
   makes the deploy record — and therefore rollback — unreliable.

Separating "may push" from "may publish" was considered and deferred: nobody has asked for
it, and it can be added without breaking anything. See the backlog.

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
| A separate role for publishing vs pushing | Speculative; the role model is coarse and this can be added later without breaking anything | A team asking for a release gate distinct from push access |

---

## 11. Left to the spec

Genuinely implementation-level; none of these change the shape above.

1. **The Parsley-facing data path** (§4.2) — value, helper, or restricted write root.
2. **Uploads URL prefix** — reuse the `/__p/` pattern or introduce a configured one.
3. **Whether publishing requires a higher role than pushing** (§9).
4. **Whether a config change to server settings requires a restart** (§7).
5. **Release directory naming** — full SHA, short SHA, or sequence plus SHA.
6. **What `basil --init` produces** now that the layout has four top-level entries.
7. **Whether `basil fmt` wraps `pars fmt` or shares its implementation** (§6.3). Sharing
   is preferred, so a Basil install does not require the `pars` binary.

---

## 12. Before any of this: BUG-033

The current implementation does not work as documented — the repository is never created
and receive behaviour is never configured, so a first push is rejected. That fix is small,
independent, and should not wait for this design. It also does not conflict with it: the
layout here removes the whole class of problem by never pushing into a checked-out tree.
