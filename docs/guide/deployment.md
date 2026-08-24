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

## The site root

Deployment works on the site-root layout that `basil --init` creates:

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
  (`cli:sam`). When push-to-deploy arrives, a push will record the Basil
  account behind the API key here instead.
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
`basil deploy live` to run. Status reports rather than insists: a legacy
layout or a missing repository is stated plainly and exits 0.

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
