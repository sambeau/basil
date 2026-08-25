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

> A friendlier `basil publish` command — one that shows you what is about to go out
> and asks before it does — is coming (FEAT-155). Until it lands, publishing is
> `git push` of the release branch, exactly as below.

## The endpoint

`basil --init` creates a site with a bare repository at `<site root>/site.git`, and
the server serves it at `/.git` whenever that repository exists. There is nothing to
turn on: the Git endpoint is live because the repository is there. A push into a bare
repository never contends with a checked-out working tree, so the first push to a
fresh site just works — no `receive.denyCurrentBranch`, no manual repository setup,
none of the ceremony older versions of Basil required.

The repository ships with the starter site already committed on the release branch,
so a clone of a freshly initialised server yields working files rather than *"you
appear to have cloned an empty repository"*.

To turn the endpoint off entirely — an unusual thing to want — set `git.enabled:
false`. There is no `git.enabled: true`: the endpoint is on when the repository
exists.

## Clone your site

```bash
git clone https://sam@mysite.example.com/.git mysite
Password: <paste the API key>     # the OS keychain remembers it
cd mysite
git config core.hooksPath .githooks   # opt into the pre-commit formatting hook
```

`basil --init` prints this exact command, with your hostname and account name
already filled in, so you rarely type it from memory.

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
git commit -m "New homepage"
git push                          # shared with the team. Not public.
git push origin live              # publish: the release branch moves, the site deploys
```

A push that moves the release branch runs the deploy pipeline on the server, and its
output reaches your terminal as `remote:` lines while the push is still running:

```
$ git push origin live
remote: Checking release cd07c0b93bb4… ok
remote: deploying cd07c0b93bb4
remote: deployed cd07c0b93bb4 in 5ms
remote: Deployed cd07c0b93bb4 (5ms)
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

So sharing work without publishing it is just a push of a branch that isn't the
release branch:

```bash
git checkout -b new-shop
git push -u origin new-shop       # on the server, published to nobody
# ...a colleague, or you on another machine:
git fetch && git checkout new-shop
# ...when everyone's happy:
git checkout live && git merge new-shop
git push origin live              # now it goes live
```

A release that fails its check is **rejected**, the release branch does not move, and
the live site is untouched — you see the reason in the terminal you typed into:

```
$ git push origin live
remote: Checking release 3c2ffbf198b3…
remote: site/index.pars:1:5: unexpected character 'Unterminated string starting with "broken"'
remote: Release rejected. The live site is unchanged (still cd07c0b93bb4).
remote: error: release 3c2ffbf198b3 refused
To https://mysite.example.com/.git
 ! [remote rejected] live -> live (pre-receive hook declined)
error: failed to push some refs to 'https://mysite.example.com/.git'
```

The [Deployment guide](deployment.md#validation) describes what validation catches
(broken code, not unfinished work) and how to override it in an emergency.

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
