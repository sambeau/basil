# Deployment

Deploying a Basil site is one command:

```bash
basil deploy live --site /srv/mysite
```

`basil deploy` takes a commit that is already in the site's repository and turns
it into the live site — and it reads the release before activating it, refusing
one that would break the site. A rejected release **leaves the live site
unchanged**. A bad one that got through is undone with `basil rollback`.

This page covers the deploy pipeline and the five operator commands: `deploy`,
`rollback`, `releases`, `status` and `check`. All five accept `--site PATH`
(point at the site root) or `--config PATH` (name the config file directly) —
the two are alternatives — and all exit non-zero on failure (2 for a usage
error, 1 for everything else).

## Setting up the server

Deploying needs a machine set up to *receive* deploys, and one command on that
box builds it:

```bash
basil --init /srv/mysite --server --host mysite.example.com --admin sam
```

`--server` is the flag that asks for the deploy topology. Without it,
`basil --init` makes the [plain local folder](basil-quick-start.md) — the right
shape for a laptop, and the wrong one for a server. Both flags are required in
this mode, and neither is guessed:

- **`--host`** is written to `server.host`: the site's public hostname, the name
  on the certificate, the WebAuthn relying-party id, the address people type. It
  is not a bind address — that is `server.bind` — so a site whose hostname points
  at a load balancer, a NAT or a container host still starts.
- **`--admin`** names the first Basil account. It is never derived from `$USER`
  or `$SUDO_USER`, because this command usually runs as `root` and "root deployed
  4f2a1c9" tells a team nothing. With a terminal attached, it is prompted for.

One run creates the site root described below, commits the starter site to the
release branch and deploys it as **release 1**, installs the receive hooks,
creates the admin account and prints its **API key once**, and ends with the
exact `git clone` command for the site, hostname and account name already filled
in. Release 1 exists because a server with no release cannot serve, so it cannot
answer an ACME challenge, cannot obtain a certificate, and could never receive
the push that would give it a release.

Run under `sudo` it hands the tree over to `$SUDO_USER` and says so;
run as `root` with no `SUDO_USER` it prints the `chown` to run. As root it also
insists the folder does not exist yet and that its parent is not writable by
other accounts, because every write it makes follows symlinks.

Then start the server:

```bash
basil --site /srv/mysite
```

`--no-git` belongs to local mode and is refused here: a machine that receives
pushes cannot be built without git. Already have a site on your laptop? See
[Graduating a local site to a server](#graduating-a-local-site-to-a-server).

## The site root

Deployment works on the site-root layout that `basil --init … --server` creates:

```
/srv/mysite/
  site.git/                       bare repository
  releases/
    4f2a1c9…/                     one directory per deployed commit
  current -> releases/4f2a1c9…    the active release
  data/                           persistent state — no deploy touches this
```

Each release is a directory named by its full commit SHA, byte-identical to
that commit — the server never rewrites code, which is what makes rollback and
the deploy record trustworthy. `current` is a symlink naming the active
release; activation is an atomic re-point of that link. The
[configuration guide](configuration.md#site-root-layout) describes the layout
and the two path anchors (release vs data) in full.

The legacy single-directory layout has no releases, so the deploy commands
refuse it and say so.

## Graduating a local site to a server

A folder created by `basil --init` never has to be restructured to go public.
The server gets its own init, your folder gets a Git remote, and the first push
carries your site across.

Before you push, open your local `basil.yaml` and make its top-level
`server` block describe the *server*, not your laptop — the hostname the site
will answer on, and the port a public site is reached at:

```yaml
server:
  host: mysite.example.com    # was: localhost
  port: 443                   # was: 8080
```

Nothing else has to change. `https.auto` defaults to on, so a config with no
`https:` block still obtains its own certificate. And locally the edit costs you
nothing, because `--dev` serves plain HTTP on `localhost` and turns a production
port 443 back into 8080 — the same `basil --dev` in the same folder, before and
after.

Leaving them at their local defaults is the one accident graduation makes easy:
`basil.yaml` ships inside the release and *becomes* the server's configuration,
so a deployed `host: localhost`, `port: 8080` would, at the server's next
restart, move the public site to a port nobody is asking for and take its Git
endpoint — your only remote way back in — with it. Pushing that release warns you at push time (see
[the listener-change warning](configuration.md#the-listener-change-warning)); the
[configuration guide](configuration.md#one-file-many-machines) explains the
layering that makes one file work everywhere.

```bash
# on the server, once
basil --init /srv/mysite --server --host mysite.example.com --admin sam

# in your local folder, once
git remote add origin https://sam@mysite.example.com/.git
git push -u origin main                 # your history is now on the server
git push --force origin main:live       # the first publish — see below
```

The API key printed by the server's init is the password for both pushes; the
username in the URL only selects which stored credential your machine offers
(see [Git over HTTPS](git.md#authentication)). After that it is the ordinary
loop — `git push` to share, `basil publish` to go live.

### Why the first publish needs `--force`

Server init seeds the release branch with a starter commit of its own, so at the
moment you connect the two, your history and the server's are unrelated: moving
`live` onto your commit is a non-fast-forward, and Git will not send one without
`--force`.

The hub accepts that **exactly once**. While the deploy record still shows
nothing but the release `--init` created, a non-fast-forward on the release
branch is allowed, and the server says so as it happens:

```
$ git push --force origin main:live
remote: replacing the starter site created by 'basil --init' with your first release — this is the one non-fast-forward the release branch allows, and it will not be allowed again
remote: Checking release 36f38ce44dc4… ok
remote: deploying 36f38ce44dc4
remote: deployed 36f38ce44dc4 in 16ms
remote: Deployed 36f38ce44dc4 (16ms)
To https://mysite.example.com/.git
 + 759fc02...36f38ce main -> live (forced update)
```

Once one real release exists, release history is protected exactly as before:
force-pushing or deleting the release branch is
[refused for everyone](git.md#what-is-refused), because the deploy record — and
therefore rollback — depends on it.

`basil publish` is the command for every publish *after* that one. It never
force-pushes, so it cannot make this one crossing; run the `--force` push above
once, and use `basil publish` from then on.

## `basil deploy <sha|branch|tag>`

Name the commit that should go live — a branch (typically `live`, the release
branch), a tag, or a full or abbreviated SHA. The commit must already be in the
site's repository (`site.git`); `deploy` does not fetch anything.

```
$ basil deploy live --site /srv/mysite
deploying a49d218cf47e
deployed a49d218cf47e in 14ms
Live: a49d218cf47e
```

One deploy runs the whole pipeline:

1. **Resolve** — the branch, tag or SHA resolves to a single commit.
2. **Lock** — one deploy at a time per site. A concurrent deploy waits up to
   30 seconds for the lock, then is refused cleanly. The lock is a kernel
   file lock, which is only reliable on a local filesystem: keep the site
   root on local disk, not on an NFS or SMB mount.
3. **Materialise** — the commit is extracted into `releases/<sha>/`, with no
   `.git` inside. A directory that already exists for that commit is reused
   as-is.
4. **Validate** — every `.pars` file is parsed and the release's `basil.yaml`
   is loaded and validated. Errors reject the release (see below).
5. **Activate** — `current` is atomically re-pointed at the new release.
6. **Hook** — if `deploy.pars` exists in the release root, it runs now (see
   [The post-deploy hook](#the-post-deploy-hook-deploypars)).
7. **Record** — commit, timestamp, duration, outcome and both identities are
   written to the deploy record.
8. **Prune** — old releases beyond `deploy.keep` are removed, never the active
   one.

Any failure before activation leaves the previous release live and untouched,
and a release directory this deploy created is removed rather than left
half-built. Every outcome — success, rejection, failure, even deploying a typo
that resolves to nothing — is recorded.

Deploying the commit that is already live is a no-op that still records:

```
$ basil deploy 8b3e40a --site /srv/mysite
deploying 8b3e40a71ca2
8b3e40a71ca2 is already live — nothing to do
```

### A running server picks the deploy up on its own

The server watches the site root for `current` being re-pointed and activates
the new release itself: it loads the new release's config, rebuilds its routes
against the new code, clears the script, response and fragment caches, and
starts serving the new release — while requests already in flight complete
against the release they started on. The CLI does not need to signal anything.
Activation lands within about a second of the re-point: the server notices the
change from a filesystem event where the platform reports one, and from a
once-a-second read of the link where it does not.

Deploying with the server **stopped** also works: the release is activated on
disk and the server serves it on its next start. `kill -HUP <pid>` re-activates
whatever `current` points at, as a manual fallback when the watcher could not
start (the server logs a warning naming SIGHUP in that case).

### What applies live, and what needs a restart

Because `basil.yaml` ships inside the release, a deploy can change
configuration too. Everything route- and site-level — routes, handlers, static
mounts, `site.path`, `public_dir`, error pages, caching TTLs — is rebuilt on
activation and applies immediately. Settings that are bound at startup do
**not** apply live; they take effect on the next restart:

- **Listener settings** — `server.port`, `server.bind`, `server.host`,
  `https.*`. The server keeps the running values and logs a warning naming
  each one the new release tried to change.
- **Middleware settings** — `logging`, `compression`, `security` headers,
  `cors`.
- **Subsystem toggles** — `auth`, `git`, `database.path`, `session`.
- **Image processing limits** — `images.*`.

Only the listener settings are warned about; treat a change to any of the
others as "deploy, then restart when convenient".

A release that arrives by **push** is warned about earlier still — in the
terminal you pushed from, before you have walked away — whenever it would move
the listener of a public site:

```
remote: warning: this release changes how the live site is served:
remote: warning:   server.host: mysite.example.com → localhost
remote: warning:   server.port: 443 → 8080
remote: The change takes effect when the server restarts, not now. If it was not intended, revert it before then: git revert HEAD && git push.
```

The push still succeeds — renaming a site or moving its port over Git is
legitimate — but the mistake is one commit deep at that point, and trivially
reverted. See
[the listener-change warning](configuration.md#the-listener-change-warning).

## Validation

Validation is the gate between materialise and activate: every `.pars` file in
the release — handlers, parts, layouts, the hook — is parsed, and the release's
`basil.yaml` is loaded and validated exactly as the server would at startup.
All errors are collected and reported, one grep-able `file:line:col: message`
per line, so a rejected deploy names everything wrong at once:

```
$ basil deploy live --site /srv/mysite
deploying f84bad0feca0
site/broken.pars:1:13: expected ')', got '{'
Release rejected. The live site is unchanged.
error: release f84bad0feca0 failed validation with 1 error(s)
```

The rejected release's directory is removed, the rejection is recorded, and the
command exits 1.

Validation catches code that is **broken**, not work that is **unfinished**: it
is parse and config-load only, and it cannot know that a page is half-written
or a feature incomplete. Deciding *when* to publish is still your job —
validation and the explicit deploy step protect against different mistakes. It
is also correctness, not style: formatting never blocks a release.

There is deliberately **no config key** to turn validation off — a switch
inside the release being validated would outlive the emergency that justified
it. The emergency override is a flag:

```
$ basil deploy f84bad0feca0 --site /srv/mysite --no-validate
deploying f84bad0feca0
WARNING: validation skipped (--no-validate)
deployed f84bad0feca0 in 9ms
Live: f84bad0feca0
```

`--no-validate` needs shell access on the server, which is the right bar for
overriding a safety check, and it cannot be left switched on.

## `basil rollback [id]`

Rollback re-activates a release that is already on disk — it is a symlink swap
and a cache clear, which is what makes it fast enough to be the emergency
answer. It never re-materialises anything.

Bare `basil rollback` returns to the previous release (the most recently
activated commit that is not the one currently live):

```
$ basil rollback --site /srv/mysite
rolled back to a49d218cf47e
Live: a49d218cf47e
```

Running it again rolls *forward* again, by the same rule. To name a target,
pass a sequence number or a SHA prefix from `basil releases`:

```bash
basil rollback 3
basil rollback a49d218
```

An ambiguous prefix is refused, not guessed. A release that pruning has removed
from disk cannot be rolled back to — deploy its commit instead, which
re-materialises it from the repository:

```
$ basil rollback 3 --site /srv/mysite
error: release f84bad0feca0 is no longer on disk (pruned?) — rollback never re-materialises; deploy it instead: basil deploy f84bad0feca0
```

## `basil releases` — the deploy record

Every deploy, rollback, rejection, failure and no-op is recorded in a SQLite
database at `<data_dir>/deploy.db`, readable with or without the server
running. `basil releases` shows it, newest first:

```
$ basil releases --site /srv/mysite
  SEQ  RELEASE       WHEN              TRIGGER   PUBLISHER       AUTHOR                OUTCOME
--------------------------------------------------------------------------------------------
  3    f84bad0feca0  2026-08-24 17:01  cli       cli:root        Ada Author            rejected
       site/broken.pars:1:13: expected ')', got '{'
* 2    8b3e40a71ca2  2026-08-24 17:00  cli       cli:root        Ada Author            deployed
  1    a49d218cf47e  2026-08-24 17:00  cli       cli:root        Ada Author            deployed

* = the live release
```

`*` marks the live release, and rejected or failed entries show their reason on
a second line. Outcomes are `deployed`, `rejected` (validation refused it),
`failed` (the pipeline could not finish), `no-op` (already live) and
`rolled-back`.

Each entry stores **two identities**, because they routinely differ:

- **Publisher** — who triggered the deploy. A CLI deploy has shell access and
  no Basil account in hand, so it records the operating system account
  (`cli:sam`). A `git push` (or `basil publish`) authenticated with an API key
  records the Basil account behind that key. A push over a `--dev` localhost
  server, where authentication is relaxed, records `push`.
- **Author** — who wrote the commit, read from the commit itself
  (`user.name` / `user.email`).

Someone merging and publishing a colleague's work is normal; a record that
stored only one identity could not answer the question it did not store.

## `basil status`

What is live, and whether the release branch has commits the live release does
not:

```
$ basil status --site /srv/mysite
live: a49d218cf47e  (deploy #1, 2026-08-24 17:00, by cli:root)
the release branch 'live' matches the live release
```

When the branch is ahead, status says by how many commits and prints the
`basil deploy` to run. It names the site's **actual** release branch throughout —
whatever `site.git`'s `HEAD` points at, `live` by default — not the literal word
"live". Status reports rather than insists: a legacy layout or a missing
repository is stated plainly and exits 0.

### Changing the release branch

The branch whose movement publishes is `site.git`'s `HEAD`, and one git command
changes it:

```bash
git -C /srv/mysite/site.git symbolic-ref HEAD refs/heads/main
```

It is deliberately a fact about the server rather than a setting in `basil.yaml`:
the config ships inside the release, so a deployed config naming the release
branch would let a release un-protect the branch whose history the hub refuses to
rewrite. The same reasoning puts the endpoint's off-switch there —
`git -C /srv/mysite/site.git config basil.gitEnabled false`, which stops `/.git`
being served at all and leaves deploys to `basil deploy` at the shell. Both are
covered in the [Git guide](git.md#which-branch-publishes).

Clients need no change: `basil publish` asks the server which branch releases,
and a clone follows the same `HEAD`. If the new branch does not exist on the
server yet, pushes to the old one are stored and publish nothing until it
arrives; `basil check` says so.

## `basil check`

`check` verifies the bootstrap preconditions and reports each one plainly with
a fix hint — this is the command to reach for when setup misbehaves:

```
$ basil check --site /srv/mysite
ok    config: loads
ok    site root: /srv/mysite
ok    release: a49d218cf47e is active
ok    repository: /srv/mysite/site.git
ok    repository placement: not inside any served root
ok    server.host: example.com
FAIL  dns: example.com does not resolve (lookup example.com: no such host) - create an A/AAAA record pointing it at this server
note  port 80: free - nothing is listening, so the server (which answers ACME challenges there) is probably not running; reachability from outside cannot be verified from here
note  certificate: none cached for example.com yet - the server obtains one at startup (fix DNS and port 80 first if that fails)
error: 1 check(s) failed
```

Hard failures — layout, active release, repository (including a repository
that resolves inside a served root, which would hand the site's history to
anyone with a browser), `server.host`, DNS — make the command exit non-zero.
Things a local process cannot prove either way, like port-80 reachability from
outside or certificate issuance, are reported as `note` and never fail the run.

## Configuration: `deploy.keep`

Deployment adds exactly one setting:

```yaml
deploy:
  keep: 5
```

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `deploy.keep` | int | `5` | How many release directories to retain. After each successful deploy, older releases beyond this count are removed — the active release is never pruned. A value of `0` or less disables pruning. |

Pruning removes directories only; the deploy record keeps every entry. There is
no `deploy.validate` (validation is always on; the override is the
`--no-validate` flag) and no `deploy.hook` (the hook is a convention, below).

## The post-deploy hook: `deploy.pars`

If a file called `deploy.pars` exists in the **release root** (next to
`basil.yaml`, not inside `site/`), it runs after activation, on every deploy of
a release that contains it. Its presence is the whole configuration —
convention, like `index.pars` — and it is an ordinary Parsley script, run the
way the `pars` CLI runs one, with `@env` and `@args` available. Use it for the
things a release needs done once it is live: a schema migration, a cache warm,
a notification.

A hook failure is recorded and reported loudly, but the deploy still succeeds
and is **never rolled back automatically**:

```
$ basil deploy live --site /srv/mysite
deploying 1446a5e3e224
DEPLOY WARNING: post-deploy hook deploy.pars failed: line 1: Identifier not found: `doesNotExist`
The release is live. Inspect the hook's work and roll back deliberately if needed: basil rollback
deployed 1446a5e3e224 in 13ms
Live: 1446a5e3e224
```

A migration that half-ran is not made better by reverting the code underneath
it — inspect, then decide.

**Use absolute paths in a hook** — ideally under the data directory. A relative
`@./…` path resolves against the hook's own directory, which is the release
itself: anything written there is destroyed when the release is pruned, and it
makes the release no longer byte-identical to its commit. The hook runs in the
deploying process, so nothing about its working directory is stable either.
The data directory (`/srv/mysite/data` by default) is the durable place; pass
its location in via an environment variable or write it into the hook.

Rollback does not re-run the hook: it re-activates code that already ran its
hook when it was first deployed.

## See also

- [Configuration](configuration.md) — the site-root layout and the two path
  anchors
- [Git over HTTPS](git.md) — cloning from and pushing to `site.git`
- [Running Basil](../basil/manual/running.md) — signals, profiles, the layouts
