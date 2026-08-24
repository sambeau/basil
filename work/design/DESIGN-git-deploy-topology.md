# Design Document: Git Deploy Topology

**Status:** Superseded — decisions taken. See `DESIGN-git-deploy.md` for the agreed design;
this document is retained for the options catalogue, the rationale, and the alternatives
that were rejected.
**Created:** 2026-08-24
**Author:** Discussion between @sambeau and AI
**Supersedes in part:** `work/design/remote-workflow-design.md` (topology only; auth model still stands)
**Related:** FEAT-035 (Git over HTTPS), `server/git.go`, `docs/guide/git.md`

---

## 1. The question

Basil's Git deploy currently makes the production server the `origin`. Developers clone
from it, edit, and push back. That works beautifully for one developer and one machine,
and it starts to wobble the moment there are two of either.

Sam's framing: *are we using the wrong mechanism?* Should we switch to, or add, a
pull-based (GitHub-style) workflow?

Short answer: **the mechanism is right, the doctrine is wrong.** Push-over-HTTPS is a
good fit for Basil and we should keep it. What needs to change is what the server's
repository *is*. Today it is simultaneously the team's shared history, the deploy
trigger, and the live filesystem — three different jobs fused into one directory. Almost
every problem in the brief falls out of that fusion.

---

## 2. What we actually have today

Read of `server/git.go`, `server/server.go:560-595`, `cmd/basil/init.go`:

- `githttp.New(siteDir)` where `siteDir = config.BaseDir` — the directory containing
  `basil.yaml`. So the served repository **is the live site directory**, working tree and
  all, including `db/`, `logs/`, `certs/` and the auth database (all gitignored, but
  physically present).
- `go-git-http` shells out to the real `git` binary with `cmd.Dir = siteDir`. This is
  good news for us later (§6.3) — real `git` means real hooks.
- Post-push reload is a library callback that clears three caches. It cannot fail
  visibly, cannot reject a push, and cannot report anything back to the developer.
- Auth is HTTP Basic with an API key as the password, role-gated (`editor`/`admin` to
  push). This part is sound and nothing below proposes changing it.

### 2.1 A gap worth fixing regardless of which option we pick

Git refuses by default to accept a push into the currently checked-out branch of a
non-bare repository. Nothing in Basil sets `receive.denyCurrentBranch`, and
`basil --init` does not `git init` the project at all. Verified locally:

```
remote: error: refusing to update checked out branch: refs/heads/main
 ! [remote rejected] main -> main (branch is currently checked out)
```

So the documented happy path in `docs/guide/git.md` — clone, edit, `git push`, site
updates — only works if the operator has independently run
`git config receive.denyCurrentBranch updateInstead` on the server. That step appears
nowhere in the code, the docs, or `--init`.

And even with `updateInstead` set, git refuses the push whenever the live working tree
has uncommitted changes to *tracked* files — which is exactly the state a long-running
site drifts into if anything ever writes inside a tracked path.

This is a real bug (worth a BUG- entry independently of this discussion), but it is also
a signal: *pushing into a live working tree is something git deliberately makes awkward.*
Every option below that keeps the live tree as the push target has to fight that, and
every option that separates them gets it for free.

---

## 3. The reframe

A git deployment is a **pointer**, not a copy of the truth.

The truth is the commit graph, and git already replicates it to every clone. There is no
"the source of truth" in git — only a *conventional* one the team agrees to meet at.
What a production server uniquely owns is one small fact: **which commit is currently
live**. That is a different job from hosting the team's shared refs, and a different job
again from being a filesystem the web server reads.

Today's design fuses all three. Hence:

| Symptom in the brief | Actual cause |
| --- | --- |
| "Two developers means two sources of truth" | The deploy target is being used as the meeting point |
| "Syncing between my laptop and desktop goes via production" | `push` is the only sync channel, so WIP has to transit prod |
| "Where does each dev's local server push to?" | Local dev doesn't need to push at all — but the model implies it does |
| "Firewalls stop prod pulling from an internal hub" | Only bites if the hub is somewhere other than the box that is already on the internet |

The current model overloads `git push` with three jobs: **share**, **back up**, and
**release**. Separate them and the multi-developer story stops being hard.

---

## 4. The options

### Option 0 — Fix the mechanics, change nothing else
Set `receive.denyCurrentBranch=updateInstead` (or install a `push-to-checkout` hook) when
Basil serves a repo; teach `basil --init` to `git init`; document it.

**Suits:** nobody as an end state. **Verdict:** necessary either way, not an answer.

### Option 1 — Keep as-is, scope it honestly as "solo mode"
Document the current design as explicitly single-developer, single-machine, and point
teams at something else.

**Cost:** near zero. **Verdict:** the do-nothing baseline. It leaves the actual question
unanswered, but it is not a *wrong* thing to ship while we decide.

### Option 2 — Deploy remote, not origin (doctrine change, small code change)
Stop calling the server `origin`. It becomes a *deploy remote*, exactly like Heroku,
Dokku, or Fly:

```bash
git remote add live https://mysite.example.com/.git
git push live main
```

`origin` lives wherever the team wants. Push-to-deploy scales to teams in the wider world
precisely *because* the deploy remote is not the origin.

**Suits:** teams that already have a shared origin somewhere.
**Doesn't solve:** two developers with no GitHub and no other server — they still have
nowhere to meet.
**Verdict:** correct doctrine, incomplete on its own. Adopt it as framing for Option 3.

### Option 3 — Basil as hub *and* deployer  ← **recommended core**

The server keeps a **bare** repository (the team's shared refs) and, separately, a
**deployed working tree** that Basil materialises when a nominated ref moves.

```
/srv/mysite/
  site.git/          bare — the hub. Served at https://mysite.example.com/.git
  live/              working tree — what the server actually serves
  db/  logs/  certs/ persistent state, never touched by a deploy
```

- **Solo developer:** `git clone https://site/.git`, edit, `git push`. Byte-for-byte the
  same UX as today — same URL, same commands — but the push now lands in a bare repo and
  Basil drives the checkout, so §2.1 evaporates.
- **Two developers:** both clone the hub. Feature branches push freely and are *stored,
  not deployed*. Merge to `main` deploys. No GitHub, no second server, no extra port, no
  extra daemon.
- **One developer, two machines:** push WIP branches from either machine to the hub.
  Only the deploy ref goes live. WIP never transits production.
- **Explicit release gate (optional):** set `deploy.ref: refs/heads/live` and promote with
  `git push origin main:live`.

This *is* the "a GitHub on a server exposed to the internet, with a trigger" that the
brief asks about — with the pleasing twist that the server Basil already runs, already on
port 443, already holding a Let's Encrypt certificate, already carrying an auth database
and a role model, **is** that server. No new infrastructure.

**Costs:** repo layout change and a migration path for existing sites; Basil now owns a
checkout it must keep consistent; disk roughly doubles (irrelevant at Basil site sizes);
we are, unavoidably, now running a small git host — with the backup and corruption
duties that implies.

### Option 4 — Pull deploy from an external upstream (additive, not a replacement)

```yaml
deploy:
  mode: pull
  source:
    url: https://github.com/me/site.git
    ref: main
    credential: !secret DEPLOY_TOKEN
  trigger:
    webhook: { path: /__deploy, secret: !secret WEBHOOK_SECRET }
    poll: 5m        # fallback for hosts with no webhooks
```

The server becomes a mirror of an upstream it fetches from. Basil is already reachable on
443 with a valid certificate, so webhook delivery is trivial; `poll` covers anything that
can't call in.

**Suits:** teams already on GitHub/GitLab/Codeberg; anyone wanting a CI gate before
deploy; and — the case push handles badly — **one repo fanning out to several servers**.
**New surface:** storing a credential for private fetches; verifying webhook signatures;
poll scheduling.
**On the "not everyone pays for GitHub" objection:** worth noting that private repos with
unlimited collaborators are free on GitHub now — the paid tier mostly buys CI minutes and
org features. The stronger argument for not mandating it isn't cost, it's *principle*:
Basil should not require a third party to deploy itself. Option 3 honours that; Option 4
supports the people who've already chosen one.

### Option 5 — Push from a dev Basil to a prod Basil
Sam's question: *could we push from a development server to a production server?* Yes —
and once Option 3 exists it costs almost nothing. The dev box's repo carries a remote
pointing at prod's `/.git`, and `basil promote` is `git push prod <sha>:main`. That gives
staging → production promotion of an exact commit, with no CI system, which is a very
"small operation" thing to want.

### Option 6 — Git over SSH
Real benefits (key auth, no password-in-URL), real costs (daemon, port, key management,
firewall-hostile). Port 443 is the most firewall-friendly thing we have.
**Verdict:** defer; revisit only if asked for.

### Fit matrix

| Scenario | 1 (today) | 3 (hub) | 4 (pull) |
| --- | :-: | :-: | :-: |
| One dev, one machine | ✅ | ✅ | ⚠️ needs a third party |
| One dev, two machines | ❌ WIP via prod | ✅ | ✅ |
| Two-plus devs, no third party | ❌ | ✅ | ❌ |
| Two-plus devs, already on GitHub | ⚠️ | ✅ | ✅ |
| Feature branches / review before release | ❌ | ✅ | ✅ |
| CI gate before deploy | ❌ | ⚠️ local only | ✅ |
| One repo → several servers | ❌ | ❌ | ✅ |
| Works behind a corporate firewall | ✅ | ✅ | ✅ (outbound) |

---

## 5. The thing all of these actually share

Whichever transport we choose, there is one object underneath, and it is the real design
work. Today it doesn't exist — its entire implementation is three `cache.Clear()` calls.

```
trigger        push received | webhook | poll | CLI | promote
  ↓
resolve        ref → commit sha
  ↓
lock           one deploy at a time
  ↓
materialise    check the tree out somewhere safe
  ↓
validate       parse every .pars; load the config; run an optional user gate
  ↓
activate       make it live
  ↓
reload         clear caches
  ↓
record         sha, author, timestamp, outcome → deploy log
  ↓
on failure     leave the previous version live; report back to whoever triggered it
```

**Transport is a plug-in on the front of this. The engine is the product.** Choosing
Option 3 and Option 4 both, later, costs far less if we build the engine first.

Two parts of that pipeline deserve highlighting because they are things a plain git host
*cannot* do and Basil trivially can:

- **`validate`** — Basil owns the Parsley parser. It can parse-check every handler before
  activating and refuse a deploy that would break the site. That is a CI gate with no CI.
- **`record` / rollback** — `basil deploy log`, `basil rollback`, and a deploy history in
  the admin panel are nearly free once deploys are a modelled event rather than a side
  effect.

---

### 5.1 What actually triggers a deploy

Worth being precise, because "a push" is the wrong answer. The trigger is **a nominated
ref moving to a new commit** in the server's repository. A push that does not move that
ref changes nothing.

The chain, under Option 3:

1. Developer runs `git push`. Basil authenticates (API key, `editor`/`admin` role) and
   hands off to `git receive-pack` in the bare repo.
2. Git runs the repo's **`pre-receive`** hook *before* updating any ref. The hook
   receives one line per ref on stdin — `<old-sha> <new-sha> <ref-name>`. Basil decides:
   - **Not the deploy ref?** Accept and stop. The branch is stored, nothing is published.
   - **The deploy ref?** Materialise the new commit somewhere private, parse every
     `.pars` file, load the config. On failure, exit non-zero: **the push is rejected**,
     the ref never moves, the live site is untouched, and the errors appear in the
     developer's terminal as `remote:` lines.
3. On success git updates the ref and runs **`post-receive`**, which activates: update
   the live tree, run any post-deploy hook (migrations, cache warm), clear caches,
   record the deploy.

Both hooks are two-line scripts calling `basil deploy --from-hook`; the binary is
already on the box. This is the concrete reason §6.3 argues for real git hooks over the
library's `EventHandler` — only `pre-receive` can *refuse*, and only hook output reaches
the developer.

**The same engine, other triggers:**

| Trigger | Used for |
| --- | --- |
| Deploy ref moves during a push | The normal path |
| `basil deploy <ref-or-sha>` on the server | First deploy, and deploying a commit already in the repo |
| `basil rollback` | Re-activating the previous recorded release |
| `git push origin main:live` | If `deploy.ref` is a separate branch, `main` shares and `live` publishes |
| Webhook / poll (Option 4) | Fetch from upstream, then hand the commit to the same engine |

Two things that are explicitly **not** triggers: a file watcher (that is dev mode, and
should stay there), and "any push" (feature branches are stored, not published).

Deploys must serialise — one at a time, with a second concurrent push either waiting or
being refused cleanly rather than interleaving. A file lock in the repo directory is
sufficient at this scale.

#### On the word "ref"

Worth defining, because it is jargon we should keep out of user-facing text. A **ref** is
just a name that points at a commit. A branch *is* a ref: `main` is literally a small file
in the repository containing one commit ID. Committing rewrites that file with the new ID.
That is the whole mechanism — a branch is a sticky note with a commit ID on it.

So "the deploy ref moved" means "the branch we nominated as the live one now points at a
different commit". The implementation should speak in refs because tags are refs too and
someone will eventually want to deploy on a tag. **The documentation and config should
speak in branches** — `deploy.branch: main` reads better than `deploy.ref:
refs/heads/main` for the 95% case, with the long form accepted for the rest.

### 5.2 The workflow, end to end

The question this has to answer: *does deploying require SSH-ing into the server?* **No.**
The push is the deploy. SSH is for first-time installation and for emergencies.

The reason is worth stating plainly because it is the mental unlock: **`git push` is not a
file copy — it is a conversation with a program running on the server.** That program is
already Basil, and today it already runs code on push (it clears three caches). We are not
adding remote execution; we are making the remote execution do something useful.

#### One-time, on the server

Unchanged from today, and the only time SSH appears:

```bash
ssh you@mysite.example.com
basil --init mysite
basil users add sam --role admin
basil apikey create sam          # bsk_… — save it
# set git.enabled: true in basil.yaml, then start the service
```

#### One-time, on each machine you work from

```bash
git clone https://sam@mysite.example.com/.git mysite
Password: <paste the API key>     # the OS keychain remembers it
```

#### The daily loop

```bash
basil --dev                       # local preview at :8080, live reload
# ...edit...
git add -A
git commit -m "New homepage"
git push                          # shares. Publishes nothing.
basil publish                     # goes live.
```

`basil publish` output:

```
Publishing 3 commits to mysite.example.com  (a1b2c3d..4f2a1c9)
  site/index.pars, site/about/about.pars, public/style.css
Continue? [y/N] y
remote: Checking release 4f2a1c9…
remote:   14 pages, 3 parts, 1 layout — ok
remote: Deployed 4f2a1c9 (0.4s)
Live: 4f2a1c9
```

See §5.3 for why publishing is a separate verb rather than a side effect of `git push`.

#### When a release doesn't hold up

```
remote: Checking release 9e8d7c6…
remote:   site/about/about.pars:14 — unexpected '}'
remote:
remote: Release rejected. The live site is unchanged (still a1b2c3d).
 ! [remote rejected] main -> main (failed pre-deploy check)
```

Fix, commit, push again. Production never wobbled, and the developer never had to go
looking for a log.

#### Sharing work without publishing it

This is the part that is impossible today:

```bash
git checkout -b new-shop
git push -u origin new-shop       # stored on the server. Nothing is published.
```

From the other machine, or the other person:

```bash
git fetch
git checkout new-shop
```

When it's ready:

```bash
git checkout main
git merge new-shop
git push                          # now it publishes
```

#### Rolling back

Two routes, deliberately:

- **From anywhere, no SSH:** `git revert <sha> && git push`. Clean history, goes through
  the same checks, needs a machine with the clone on it.
- **On the server:** `basil rollback`. For when the site is broken and the answer needs to
  take two seconds. This is the emergency door, not the normal path.

If we decide the admin panel gets a deploy view (§8 Q6), a rollback button there removes
the last routine reason to hold a shell on the box.

### 5.3 Why publishing is a separate verb

Raised by @sam, 2026-08-24, and it resolves §8 Q2. The earlier sketch had the dangerous
action as the default one: bare `git push` published to the world, while the *safe* action
— sharing work without publishing it — needed a longer, more deliberate command. That is
backwards. **The safe action should be the one you get for free.**

First, a framing correction that matters for understanding the fix: there is only ever
**one repository** on the server. `git push` and `git push -u origin new-shop` both talk to
the same hub. What differs is *which branch* they update, and whether that branch happens
to be the one nominated for release. So the question is not "which repository is the
default" — it is **"is the branch I work on every day a branch that publishes?"**

#### The two arrangements

| | `deploy.branch: main` | `deploy.branch: live` (proposed default) |
| --- | --- | --- |
| `git push` on main | **publishes** | shares only |
| Publishing | automatic | `basil publish` |
| Invariant | `main` is always what's live | live is some ancestor of `main` |
| Failure mode | publishing something you didn't mean to | site quietly falls behind |
| Steps per release | one | two |

The failure modes are **asymmetric**, and that decides it. A site running a few hours
behind is visible, private, and fixed in one command. Something published that shouldn't
have been is out in public and cannot be recalled. The validation gate does not help here
either: it catches code that is *broken*, not work that is merely *unfinished*. A
half-finished redesign parses perfectly.

The obvious objection — that the site can silently drift behind — is a tooling problem, not
a design problem. `basil --dev` and `basil publish` both know both commits, so either can
say `live is 3 commits behind main` and remove the surprise. That is the whole cost of the
safe default, and it is cheap to buy off.

#### Why a verb, not a refspec

The mechanism could be plain git — `git push origin main:live` does exactly the right
thing. It is a bad interface anyway:

- Refspec syntax is obscure. It is the first thing a developer would have to learn that
  they did not already know, for the single most consequential command in the system.
- It cannot show you what you are about to do. `basil publish` can list the commits and
  changed files, and ask.
- It cannot report back well. A verb owns its own output.

`basil publish` is one word, says what it does, and is available already — the developer
has the binary because they run `basil --dev`. Under the hood it is still just a push to
the release branch, so raw git and editor Git panels keep working for anyone who prefers
them; `basil publish` can add the remote refspec on first use so it works without setup.

#### Keeping the one-step mode

`deploy.branch: main` restores push-to-publish for anyone who wants it — a solo developer
on a personal site may reasonably prefer one step and the simpler invariant. It stays a
supported configuration and a one-line change. **The argument here is only about which way
round the default should be**, and the default should protect the case where getting it
wrong is public.


---

## 6. Implementation notes for whichever way we go

### 6.1 State must not live inside the deployable unit

`config.BaseDir` is currently the project root *and* the repo *and* the home of every
piece of persistent state. Under Option 3 the deployable unit is the checked-out tree,
and state has to sit beside it rather than inside it.

**Full inventory of what is not code**, in decreasing order of how much it hurts to lose:

| What | Where today | Notes |
| --- | --- | --- |
| Auth database | `<BaseDir>/.basil-auth.db` (`auth/database.go:153`) | Users, roles, API keys, passkeys, recovery codes. Losing it destroys the credentials needed to deploy in the first place. |
| Application database | `database.path`, resolved against `BaseDir` (`server.go:442`) | Plus `-wal` / `-shm` sidecars. Sessions live here too when `session.store: sqlite`. |
| TLS certificates | `https.cache_dir`, default `certs` (`server.go:1134`) | Let's Encrypt account key and issued certs. Re-issuing is rate-limited, so this is not merely regenerable. **Note the path is resolved against the process working directory, not `BaseDir`** — an inconsistency worth fixing in the same change. |
| Logs | `logging.output`, `logging.parsley.output`, `dev.log_database` | `./logs/` by convention. |
| Image cache | `images.cache_dir`, default `./cache/images` (`config.go:350`) | Regenerable but expensive. **Not covered by the `.gitignore` that `--init` ships** — see BUG-033. |
| Search indexes | Site-chosen path, resolved against the site root (`search.go:345`) | Rebuildable from source content, at a cost. |
| Whatever the site itself writes | Anywhere the site's own code puts it | Parsley has file I/O (`eval_file_io.go:591`, `methods_file_http.go:148`). Uploads and generated files typically land under `public/`, which *is* a deployed directory. |

The first six are config values and move mechanically. The last row is the genuinely
hard one: there is no config key for it because it is site code, not framework config.
Two candidate answers, not yet chosen — give sites a data directory they are expected to
write into and expose it to Parsley, or serve a configured upload directory under a URL
prefix so it never needs to live inside `public/` at all. Basil already serves generated
assets this way (`/__p/`, `/__img/`), so there is precedent for the second.

**How much of this is actually required, and when.** It depends on the activation
strategy, and the distinction matters for scoping:

- **In-place checkout** (git updates the live directory to match the release): ignored
  and untracked files are left alone, so `db/`, `logs/`, `certs/` and uploads survive
  untouched. Almost nothing has to move. What *is* needed is tightening the ignore rules
  so nothing persistent sits in a tracked path — otherwise it collides with a checkout
  and, per BUG-033, can block deploys outright.
- **Replaceable release directories** (§6.2): the live directory is swapped wholesale,
  so anything inside it is gone on the next deploy. Here the split is mandatory.

So the split is a **prerequisite for atomic releases and instant rollback, not for
Option 3 as scoped**. Recommendation: do the ignore-rule tightening and the `certs`
path fix now, design the layout so the full split stays available, and only do the split
when we go after atomic releases.

### 6.2 Atomic releases — reconsidered now that there is no installed base
Capistrano-style `releases/<sha>/` plus a `current` symlink gives atomic swaps and instant
rollback. It requires splitting `BaseDir` into code and state (§6.1).

An earlier draft of this document recommended deferring it, on the grounds that the split
is a bigger change than it looks. **That reasoning was mostly about migration cost, and
there is no installed base to migrate** (@sam, 2026-08-24: no live sites use Git deploy;
the Basil website itself is on GitHub Pages). That changes the calculus:

- Retrofitting a code/state split onto deployed sites is the expensive version. Choosing
  the layout correctly at the outset is nearly free.
- Rollback is the feature that makes push-to-deploy feel safe, and it falls straight out
  of keeping the previous release directory around.
- The alternative — shipping in-place checkout now — means doing the layout work anyway,
  later, with sites in the field.

**Revised recommendation:** do the split as part of the core work, and treat release
directories as the default activation strategy rather than a later enhancement. The
validate-before-activate gate is still the more important half; atomic swap is the cheap
half once state has somewhere else to live.

### 6.3 Use real git hooks, not the library callback
`go-git-http` shells out to the real `git` binary, so real `pre-receive` / `post-receive`
hooks fire. That buys three things the current `EventHandler` cannot provide:

- **Rejection.** A `pre-receive` hook exiting non-zero rejects the push. A deploy that
  fails validation should fail the developer's `git push`, not silently half-apply.
- **Feedback.** Hook stdout is relayed to the client as `remote: …` lines. Validation
  errors can appear in the developer's terminal, Heroku-style.
- **Ordering.** The reload happens as part of receive, not racing alongside it.

The hooks themselves can be two-line scripts calling `basil deploy --from-hook`; the
binary is already on the box.

### 6.4 Config sketch (illustrative, not settled)

```yaml
git:
  enabled: true
  require_auth: true
  repo: ./site.git            # bare hub (default: <basedir>/site.git)

deploy:
  mode: push                  # push | pull
  branch: live                # what goes live. Set to `main` for push-to-publish (§5.3)
  tree: ./live                # where it's materialised
  validate: true              # parse-check before activating
  hook: ./deploy.pars         # optional post-deploy script (migrations, cache warm)
  mirror: git@github.com:me/site.git   # optional: push a copy after each deploy
```

`mirror` is a small feature with a good story: it answers "the server is now the only
shared copy" for anyone who *does* have a GitHub account, without requiring one.

### 6.5 Migration — not a constraint
There is no installed base. No live site currently uses Git deploy, and the Basil website
runs on GitHub Pages (@sam, 2026-08-24). So there is no compatibility surface to preserve
and no migration tool to write.

This is worth stating explicitly because it removes the usual reason to make a design
worse: we can choose the on-disk layout, the config keys and the activation strategy on
their merits alone. See §6.2, where it changes the recommendation outright.

If a `basil git migrate` is ever wanted — for someone who set up the current arrangement
by hand — it stays straightforward: the public URL is just a route, so existing clones
would keep working untouched. But it is not gating anything.

### 6.6 Assorted
- **Locking:** two simultaneous pushes must serialise. A file lock in the repo dir is
  enough at this scale.
- **Secrets:** `basil.yaml` is in the repo. Confirm the `!secret` story holds up when the
  config is pushed through a hub that other developers can clone.
- **Repo location:** a bare repo outside every served root is structurally safer than the
  current arrangement, where `.git` sits inside `BaseDir` and safety depends on the served
  roots (`public_dir`, `site.path`) staying disjoint from it.
- **Not supported, and fine:** Git LFS, submodules, shallow clones as a deploy source.
  Say so in the docs rather than discovering it in the wild.

---

## 7. Recommendation

**Keep push. Split the repo from the working tree. Build the deploy engine. Add pull later.**

1. **Now (small, independent of everything else)**
   - Fix §2.1: `--init` creates the repo; Basil configures receive behaviour; docs match
     reality. File as a bug.
   - Move post-push reload onto a real `post-receive` hook.

2. **Core proposal — Option 3 + §5 engine**
   Bare hub + Basil-managed checkout + validate → activate → reload → record, with a
   deploy log and `basil rollback`. Unlocks teams and multi-machine work with no third
   party and no new infrastructure. Publishing is an explicit `basil publish` rather than
   a side effect of `git push` (§5.3), and because there is no installed base the
   code/state split happens here rather than later (§6.2). This is the piece worth
   writing a FEAT for.

3. **Additive — Option 4**
   Pull mode with webhook + poll, once the engine exists. For GitHub users, CI gates, and
   multi-server fan-out. Option 5 (`basil promote`) falls out of 2 and 3 nearly free.

4. **Defer** — SSH transport; atomic release directories; branch → environment mapping
   (staging on a second host or path).

The one-line pitch, which is worth stating because it is genuinely unusual: **Basil can be
the host, the origin, the CI gate, and the deployer, on one box, over one port, with a
certificate it obtained itself.** That is a real differentiator against Pages/Netlify/
Vercel, and it is entirely in keeping with a framework that already refuses to make you
assemble five services to serve a website.

---

## 8. Decisions needed before this becomes a spec

1. **Is Option 3 the direction?** Everything else here is sequencing.
2. ~~**Default deploy ref**~~ — **proposed resolved** (@sam, 2026-08-24). Default
   `deploy.branch: live`, so `git push` shares and `basil publish` releases. `main`
   remains a supported one-liner for anyone wanting push-to-publish. Reasoning in §5.3;
   still Sam's call to confirm.
3. **Is the validation gate on by default?** A parse error rejecting the push is a great
   default and a surprising one. Recommendation: on, with `deploy.validate: false`.
4. **Do we ship pull mode at all**, or say "use Option 3, or push from your CI"?
5. **How far do we take the BaseDir code/state split** now, given it gates atomic releases
   later?
6. **Does the admin panel get a deploy view** (history, current sha, rollback button), or
   is CLI enough for v1?

---

## Appendix: what the brief got right

The five advantages listed for the current design all survive this proposal intact:

1. *The server always runs the latest code* — still true; the deploy ref is authoritative.
2. *Simple to understand and operate* — arguably simpler, since "the repo" and "the live
   files" stop being the same confusing object.
3. *Works well for a single developer* — unchanged UX, same URL, same commands.
4. *Local dev, then push to production* — unchanged, and now WIP never has to transit prod.
5. *Works with firewalls and LANs* — unchanged, and the reason to prefer push over pull as
   the default. Outbound-only pull is the fallback, not the primary.

Nothing here asks Basil to give up what makes its deploy story good. It asks it to stop
making one directory do three jobs.
