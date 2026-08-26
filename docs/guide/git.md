# Git over HTTPS

Basil's server holds a **bare Git repository** that your team pushes to. You clone
it, edit locally, `git push` — and none of it reaches the public until you
deliberately release it. The clone URL is `https://user@host/.git`, unchanged from
earlier versions of Basil, but what sits behind it is different: a proper shared Git
host, not the live site directory.

Two verbs do two different things, and keeping them apart is the whole point:

```
git push            →  shared with the team. Stored, published to nobody.
publish             →  the release branch moves, and the site goes live.
```

`git push` is transport. It stores whatever you push — any branch, any tag — and
publishes none of it. **Publishing is moving the release branch** (`live` by
default): a push that moves it triggers a deploy, which checks the release and, only
if it passes, makes it the live site. That pipeline — validate, activate, record,
roll back — is the subject of the [Deployment guide](deployment.md); this page is
about the Git side: cloning, authenticating, pushing, and the credential mechanics
that trip people up.

Publishing has its own verb: **`basil publish`**. It shows you the commits and files
about to go out, reports how far the live site has drifted, and asks before it moves
anything. Underneath it is still a push of the release branch — so raw `git push`
remains a working alternative (see [Publishing with raw Git](#publishing-with-raw-git))
— but the verb is the one to reach for, and the daily loop below is built around it.
None of this needs a single change to `basil.yaml`: a site created by
`basil --init … --server` already has everything the two-verb workflow relies on.

## The endpoint

`basil --init … --server` creates a site with a bare repository at
`<site root>/site.git`, and the server serves it at `/.git` whenever that repository
exists. There is nothing to turn on: the Git endpoint is live because the repository
is there. A push into a bare repository never contends with a checked-out working
tree, so the first push to a fresh site just works — no `receive.denyCurrentBranch`,
no manual repository setup, none of the ceremony older versions of Basil required.

The repository ships with the starter site already committed on the release branch,
so a clone of a freshly initialised server yields working files rather than *"you
appear to have cloned an empty repository"*.

**There is nothing in `basil.yaml` to turn the endpoint on or off.** The config
travels inside the release, so a setting that could switch the endpoint off would
let a release disable the very mechanism it arrived through — leaving no way back
in but a shell on the box. The off-switch is therefore a fact about the server,
recorded in the repository itself:

```bash
git -C /srv/mysite/site.git config basil.gitEnabled false   # then restart
```

With that set, `/.git` is not served at all — clone and push both get a 404 — and
deploys happen at the server shell with `basil deploy`. Unset (the normal case)
or `true` means the endpoint is live whenever `site.git` exists. Changing it
takes effect at the next restart, like the listener settings — a running server
keeps serving whatever it was built with. It only checks the switch again when
something makes it re-read the site (a deploy, or a restart), and if it finds
the change it says so; nothing watches the file, so **restart to apply it**. A
plain local project directory has no bare
repository and so no endpoint at all, which is the correct shape for a laptop and
needs no setting either.

(The old `git.enabled` key is gone. A config that still carries it loads and
warns, naming the command above.)

## Which branch publishes

The release branch is **`site.git`'s `HEAD`** — git's own record of the
repository's default branch, and already what a fresh clone checks out. `basil
--init … --server` points it at `live`. To change it:

```bash
git -C /srv/mysite/site.git symbolic-ref HEAD refs/heads/main
```

That one command is the whole interface. No client changes: `basil publish` asks
the server which branch releases, and a plain `git clone` follows the same `HEAD`.
Nothing in a release can rewrite it — which is the point. When the config named
the branch, a deployed `basil.yaml` could point the protections at some other
branch and leave the real one freely force-pushable.

If you retarget `HEAD` at a branch that does not exist on the server yet, nothing
breaks and nothing deploys: pushes to the old branch are stored and published to
nobody until the new branch arrives (push it once explicitly — `git push origin
HEAD:refs/heads/main`). `basil check` reports that state plainly, as it does a
`HEAD` that names no branch at all.

## Clone your site

```bash
git clone https://sam@mysite.example.com/.git mysite
Password: <paste the API key>     # the OS keychain remembers it
cd mysite
git config core.hooksPath .githooks   # opt into the pre-commit formatting hook
```

The server's own `basil --init … --server` prints this exact command, with the
hostname and account name already filled in, so you rarely type it from memory.
(Coming the other way — a folder you already have locally — is the
[graduation path](deployment.md#graduating-a-local-site-to-a-server), which adds a
remote instead of cloning.)

In dev mode the transport is plain HTTP on localhost:

```bash
basil --dev --site /srv/mysite
git clone http://sam@localhost:8080/.git mysite
```

**Production is always HTTPS.** Basil obtains its own certificate at startup, so the
`https://` URL works without any TLS setup on your part — see the
[Deployment guide](deployment.md#basil-check). Plain HTTP is refused everywhere
except a `--dev` server on localhost (see [Authentication](#authentication)).

## The daily loop

```bash
# ...edit...
git add -A
git commit -m "New homepage"      # the pre-commit hook formats staged .pars files
git push                          # shared with the team. Not public.
basil publish                     # publish: review, confirm, and the site deploys
```

`git push` is the command you run twenty times a day, and it is the harmless one: it
stores your work on the server and shows it to nobody. `basil publish` is the
deliberate one. It runs in your clone, asks the server which branch releases,
and shows you exactly what is about to go out — the commit range, the
files, and the drift it closes — before it moves anything:

```
$ basil publish
Publishing to "live" on origin (38daf0d13ed3..2da6d60b6277).

1 commit:
  2da6d60 New homepage + about page

2 files changed:
  site/about.pars
  site/index.pars

drift: the release branch "live" on origin is 1 commit behind HEAD (this publish closes it).

Publish 1 commit to "live"? [y/N] y

Pushing 2da6d60b6277 to "live"...
remote: Checking release 2da6d60b6277… ok
remote: Deploying… done (4ms)
To https://mysite.example.com/.git
   38daf0d..2da6d60  HEAD -> live
```

The `remote:` lines are the server's deploy pipeline — validate, activate, record —
running while the push is still open and streamed to your terminal as it happens.
Nothing is sent until you answer the prompt: an empty line, or anything but `y`,
cancels and pushes nothing.

- **`basil publish --dry-run`** prints the same plan — commits, files, drift — and
  stops without pushing.
- **`basil publish --yes`** skips the confirmation, for scripts.
- It works from any clone with no prior setup: it learns the release branch from
  the server (one `git ls-remote --symref origin HEAD`) and configures the push
  refspec on first use. No `basil.yaml` changes, no local Git config to remember —
  and an operator who retargets the branch needs no change on any client.

Sharing work *without* publishing it is just a push of a branch that isn't the release
branch — stored on the server, published to nobody:

```bash
git checkout -b new-shop
git push -u origin new-shop       # on the server, published to nobody
# ...a colleague, or you on another machine:
git fetch && git checkout new-shop
# ...when everyone's happy:
git checkout live && git merge new-shop
basil publish                     # now it goes live
```

A release that fails its check is **rejected**: the release branch does not move, the
live site is untouched, and `basil publish` exits non-zero. The reason streams into
the terminal you typed into:

```
$ basil publish --yes
Publishing to "live" on origin (27fbba4810e6..2e7e05cd8b00).

1 commit:
  2e7e05c Broken release
...
Pushing 2e7e05cd8b00 to "live"...
remote: Checking release 2e7e05cd8b00…
remote: site/broken.pars:1:5: unexpected character 'Unterminated string starting with "broken"'
remote: Release rejected. The live site is unchanged (still 27fbba4810e6).
remote: error: release 2e7e05cd8b00 refused
To https://mysite.example.com/.git
 ! [remote rejected] HEAD -> live (pre-receive hook declined)
error: failed to push some refs to 'https://mysite.example.com/.git'
error: publish failed: the push to "live" was rejected (see the messages above)
```

The [Deployment guide](deployment.md#validation) describes what validation catches
(broken code, not unfinished work) and how to override it in an emergency.

## Drift: what is live, and how far behind

Splitting sharing from publishing means the live site can quietly fall behind the
branch. `basil status` tells you when it has — what is live, what the release branch
points at, and the gap:

```
$ basil status --site /srv/mysite
live: 2da6d60b6277  (deploy #2, 2026-08-25 10:33, by push)
the release branch 'live' matches the live release
```

When the branch is ahead, it says by how many commits and prints the command to close
the gap:

```
$ basil status --site /srv/mysite
live: 2da6d60b6277  (deploy #5, 2026-08-25 10:34, by cli:root)
the release branch 'live' is 1 commit ahead of the live release - deploy it with: basil deploy live
```

`basil publish` reports the same drift in its summary before you confirm, so you
rarely have to ask. Status reports rather than insists: a legacy layout or a missing
repository is stated plainly and exits 0.

## Formatting: `basil fmt` and the pre-commit hook

`basil fmt` is the canonical Parsley formatter — the same engine as `pars fmt`, built
into the `basil` binary you already have. It works on whole trees, not just named
files:

- **`basil fmt -w [path]`** rewrites files in place (a directory, or the whole tree
  with no argument).
- **`basil fmt -l`** lists the files whose formatting differs and exits non-zero — a
  CI gate.
- **`basil fmt -d <file>`** shows the diff without touching anything.

```
$ basil fmt -d site/about.pars
diff site/about.pars
-1: let    title="About"
+1: let title = "About"
-2: let items=[1,2,3]
+2: let items = [1, 2, 3]
...
```

You rarely run it by hand, because `basil --init` installs a **pre-commit hook** that
formats staged `.pars` files for you (opt in on a fresh clone with
`git config core.hooksPath .githooks`, which `--init` prints). A messy commit lands
formatted:

```
$ git commit -m "New homepage"
[live 2da6d60] New homepage
$ git show HEAD:site/index.pars
let title = "Home"
<h1>title</h1>
```

The server **warns** about unformatted `.pars` files in a push and **never rejects**
over them — formatting is style, not correctness, and a release is never blocked by
whitespace. The warning names the fix and the push still deploys:

```
$ basil publish --yes
...
remote: Checking release 27fbba4810e6… ok
remote: warning: 1 file(s) are not formatted:
remote:   site/contact.pars
remote: Run 'basil fmt -w' to format them. The push was accepted.
remote: Deploying… done (5ms)
```

There is no setting to turn this into a gate: the hook keeps shared history clean, and
nobody is ever refused by the server over formatting.

## Publishing with raw Git

`basil publish` is a convenience over a plain push, not a new protocol, so raw Git
stays a first-class alternative — use your editor's Git panel or the command line if
you prefer. Publishing is moving the release branch, so a push of it deploys exactly
as `basil publish` does, minus the confirmation and the plan:

```
$ git push origin live
remote: Checking release cd07c0b93bb4… ok
remote: Deploying… done (5ms)
To https://mysite.example.com/.git
   c58530e..cd07c0b  live -> live
```

A push that moves any *other* ref is stored and nothing else happens — no deploy, no
output beyond Git's own:

```
$ git push -u origin new-shop
To https://mysite.example.com/.git
 * [new branch]      new-shop -> new-shop
```

If you would rather a commit go live the moment it is pushed — no separate publish
step — point the release branch at the one you work on, on the server:

```bash
git -C /srv/mysite/site.git symbolic-ref HEAD refs/heads/main
```

Now `git push` (of `main`) *is* publishing, restoring the older push-to-publish
model for teams that want it. This is the one place the two-verb split is a choice
rather than the default; it is the operator's choice, made once on the box, and
everything else on this page works unchanged.

## Authentication

Git access uses **HTTP Basic Auth over TLS**, with your **API key in the password
field**. This is the model every Git client, editor and credential manager already
speaks — the same shape as a GitHub personal access token.

- **Password**: your API key (starts with `bsl_live_`).
- **Username**: selects a stored credential on your machine, and is **ignored by
  Basil** — only the key authenticates. (This matters more than it sounds; see
  [The two things Git calls a username](#the-two-things-git-calls-a-username).)

| Operation | Required role |
|-----------|---------------|
| Clone / fetch | any authenticated user |
| Push (any branch, including the release branch) | `editor` or `admin` |

A push from a `viewer` is refused with a message naming the role required:

```
Forbidden: pushing requires the editor or admin role (Vic has the viewer role)
```

Create users and keys with the CLI (both take the user *ID*, not the name; the first
user created is always an admin):

```bash
basil users create --name Sam --email sam@example.com --role editor
# ✓ Created user usr_7d30113fc69ba1d8...
basil apikey create --user usr_7d30113fc69ba1d8... --name "MacBook Git"
# ✓ Created API key: bsl_live_AcTVMzefTllD875... (save this now — it won't be shown again)
basil apikey list --user usr_7d30113fc69ba1d8...
basil apikey revoke key_42c52e0dc55a5a64...      # takes the key ID, not the key itself
```

### Plain HTTP is refused

HTTP Basic Auth puts the API key in an easily-decoded header, so over plain HTTP it
is a plaintext credential with push rights on the wire. Basil **refuses** Git over
plain HTTP rather than merely warning about it:

```
Git over plain HTTP is refused: HTTP Basic authentication would send the API key
unencrypted. Use https:// (Basil obtains its own certificate), or a --dev server on
localhost.
```

The refusal is scoped to the Git endpoints — the rest of the site, including the
ACME challenge path that lets Basil obtain its certificate, is served over plain HTTP
on port 80 as normal. The **sole exception** is a `--dev` server answering a request
from localhost, and that exception is decided in code, never from a config file.
There is no setting to disable authentication or to allow plain-HTTP Git; a server
with no auth database refuses Git entirely.

Never work around a TLS error with `git -c http.sslVerify=false`: your API key is in
the request, so disabling verification hands push rights to whoever answered the
connection. If the handshake fails, fix the certificate — `basil check` on the server
tells you whether DNS or port 80 is the problem.

## The two things Git calls a username

Git uses the word "username" for two unrelated things, and conflating them is the
likeliest source of confusion here. They never meet:

| | Where it lives | What it does |
|---|---|---|
| **URL username** — the `sam@` in the clone URL | `remote.origin.url` in `.git/config` | Selects which stored credential to send. **Basil ignores it** — only the API key authenticates. Nothing to do with commits. |
| **`user.name` / `user.email`** | `git config`, global or per-repo | The **commit author**, recorded *inside* every commit you make. Nothing to do with authentication. |

A developer whose `user.email` is `sam@personal.example` and whose Basil account is
called `sam` has no conflict to resolve. Authorship is read from the commit; access
is read from the key.

Because Basil ignores the URL username, `https://anything@host/.git` works with a
valid key. **Use your real account name in the URL anyway.** The platform credential
helper keys on *(host, username)*, so two accounts on the same host — you and a
deploy key, say — need distinct URL usernames or they fight over one keychain entry.
It is free hygiene for a case that is annoying to debug later, and the deploy record
still names whoever the key belongs to, not whoever the URL says.

## Remembering your key: credential storage

After the first push, Git offers your key to the platform credential helper, which
stores it so every later `git push` is silent. **What that helper is, and whether it
is secure, depends on your operating system** — and one common Linux fallback stores
the key in plaintext.

| Platform | Helper | Setup |
|---|---|---|
| macOS | Keychain (`osxkeychain`) | None — encrypted, ships configured |
| Windows | Credential Manager (`manager`/`wincred`) | None — encrypted, ships configured |
| Linux | **often none configured** | See below |

On macOS and Windows there is nothing to do. On Linux Git frequently has no
credential helper at all, in which case one of two things happens:

- Git prompts for the key on **every** push, or
- someone reaches for `credential.helper store`, which writes the key **in plaintext**
  to `~/.git-credentials`.

That second option leaves an API key with push rights sitting in a readable file. It
is the most likely way a Basil key leaks in practice. **Prefer `libsecret`,** which
stores the key in your desktop keyring:

```bash
# Debian/Ubuntu: build and enable the libsecret helper
sudo apt install libsecret-1-0 libsecret-1-dev
sudo make -C /usr/share/doc/git/contrib/credential/libsecret
git config --global credential.helper \
  /usr/share/doc/git/contrib/credential/libsecret/git-credential-libsecret
```

Use `credential.helper store` only when you understand that it is plaintext on disk —
a throwaway VM, say — and never on a shared host.

## What is refused

- **Force-pushing the release branch** — refused, for everyone, with no setting to
  permit it. A rewritable release history makes the deploy record, and therefore
  rollback, unreliable.

  ```
  remote: force-pushing the release branch rewrites release history, which the deploy
  remote: record and rollback rely on — it is refused for everyone
  remote: error: release branch force-push refused
   ! [remote rejected] live -> live (pre-receive hook declined)
  ```

  **One exception, once.** A brand-new server's release branch holds only the starter
  commit `--init` made, so the first release pushed from a site you built locally is
  always a non-fast-forward. While the deploy record still shows nothing but that
  `init` release, the push is accepted and announced; from the first real release
  onward the refusal above applies as normal. That is the
  [graduation path](deployment.md#graduating-a-local-site-to-a-server): `basil
  publish` recognises this one state and makes the forced push for you (its
  under-the-hood form is `git push --force origin HEAD:live`, which you can still
  type yourself). It is the only time a Basil server accepts a `--force` at all.

- **Deleting the release branch** — refused, same reasoning.

  ```
  remote: the release branch cannot be deleted
  remote: error: release branch deletion refused
   ! [remote rejected] live (pre-receive hook declined)
  ```

  Force-pushing or deleting any *other* branch is fine — it is your repository as much
  as the server's.

- **Git LFS and submodules are not supported.** The endpoint serves the Smart HTTP
  clone/fetch/push protocol and nothing else; there is no LFS batch API, and a
  submodule pointing back at a Basil server has no repository of its own to resolve
  against. Keep large binaries in the repository directly (or serve them as
  [uploads](deployment.md)) and vendor shared code rather than linking it as a
  submodule.

## Troubleshooting

### A stale cached credential — 401 forever, no prompt

This is the most common failure once things are working. A wrong or revoked key is
cached in your keychain, so pushes fail `401` and re-running the command never prompts
again — Git keeps offering the same bad key. Clear it and Git will prompt afresh on
the next push:

```bash
printf 'protocol=https\nhost=mysite.example.com\n' | git credential reject
```

Then `git push` and paste a current key.

### "Authentication failed" / repeated password prompts

The key is wrong, revoked, or missing. Confirm the key is current
(`basil apikey list --user usr_…` on the server), then either paste it when prompted
or clear a stale cached one as above. The username in the URL is irrelevant to this —
only the key is checked.

### "Forbidden: pushing requires the editor or admin role"

Your key is valid but your account is a `viewer`. Anyone can clone; only `editor` and
`admin` can push. Raise the role on the server:

```bash
basil users show usr_abc123...            # show takes the ID, not the name
basil users set-role usr_abc123... editor
```

### "Git over plain HTTP is refused"

You cloned with an `http://` URL against a production server. Re-point the remote at
`https://`:

```bash
git remote set-url origin https://sam@mysite.example.com/.git
```

Only a `--dev` server on localhost accepts plain-HTTP Git.

### A release push is rejected

The rejection message names the file, line and column
(`site/index.pars:1:5: …`). That is validation refusing broken code before it goes
live — the release branch did not move and the site is unchanged. Fix the reported
error and push again. See the [Deployment guide](deployment.md#validation) for what
validation does and does not catch.

## Security notes

- **HTTPS is not optional in production** — the key is a Basic-auth credential; plain
  HTTP is refused precisely so it is never sent in the clear.
- **Treat the key like a password** — it grants push (and therefore deploy) rights.
  Store it in the OS keychain or `libsecret`, not in `~/.git-credentials`.
- **Give the minimum role** — a machine that only needs to read the site should hold a
  `viewer` key.
- **Rotate on suspicion** — `basil apikey revoke <key_id>` takes effect immediately
  and needs no change to the user account.

## See also

- [Deployment](deployment.md) — the deploy pipeline a release push triggers: validate,
  activate, roll back, and the deploy record.
- [Configuration](configuration.md#site-root-layout) — the site-root layout and the
  two path anchors (release vs data).
- [Authentication](authentication.md) — users, roles and API keys.
