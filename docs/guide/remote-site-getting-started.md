# Getting Started: A Remote Site and Its Local Dev Partner

This guide takes you from nothing to a live website on your own server, with a
copy on your laptop you develop against — the two kept in step through Git.

It is the linear walkthrough. When you want the details behind any step, the
[Deployment](deployment.md) and [Git over HTTPS](git.md) guides are the
reference; this one just gets you there.

## The shape of what you are building

Two copies of one site, with one job each:

- **Your laptop** — the *dev partner*. A plain folder of files you edit and
  preview with `basil --dev`. This is where all the work happens.
- **Your server** — *production*. It holds the shared Git history and serves the
  live site. You never edit files on it directly; you push to it.

```
   laptop (dev)                         server (production)
   ┌───────────────┐                    ┌────────────────────────┐
   │ basil.yaml    │   git push  ───▶   │ site.git   (history)   │
   │ site/         │                    │ releases/  (each deploy)│
   │ public/       │   basil publish ▶  │ current →  (what's live)│
   │ basil --dev   │                    │ data/      (never touched)│
   └───────────────┘                    └────────────────────────┘
```

The laptop folder stays simple — the same folder whether or not it is ever
deployed. All the deployment machinery lives on the server. You move code
between them with two verbs:

- **`git push`** — *share*. Sends your commits to the server. Publishes nothing.
- **`basil publish`** — *go live*. Checks the latest release and, if it is sound,
  makes it the site people see.

Keeping "share" and "go live" separate is deliberate: the safe action is the one
you get for free, and going public is the one you ask for.

## Prerequisites

- **Git** on both machines.
- **A built `basil` binary** on both machines. (Building it: clone the repo and
  `go build -o basil .` — see the [Quick Start](basil-quick-start.md).)
- **A server** you can SSH into, with:
  - a **hostname** pointing at it in DNS (e.g. `mysite.example.com`), and
  - **ports 80 and 443** reachable from the internet. Port 80 answers the
    one-time certificate challenge; 443 serves the site and the Git endpoint.

Basil obtains its own HTTPS certificate from Let's Encrypt on first start, so
there is no certificate to buy, install, or renew.

---

## Part 1 — The site, on your laptop

Create it:

```bash
basil --init mysite
```

```
Created Basil site 'mysite'

  mysite/
  ├── basil.yaml     configuration
  ├── site/          your pages (index.pars is the home page)
  └── public/        static files (CSS, JS, images)

  ✓ git repository on 'main', with the starter site committed
  ✓ pre-commit formatting hook

Start the server:
  cd mysite && basil --dev
```

That is the whole site: a config file, a `site/` folder whose files map to URLs,
and a `public/` folder for static assets. `--init` also made it a Git repository
on `main` with your first commit already in place — which is what lets it
graduate to a server later with no restructuring.

Run it:

```bash
cd mysite
basil --dev
```

Visit **http://localhost:8080** — there is your starter page. Edit
`site/index.pars`, reload, and you will see the change. This is your whole
development loop, and it never needs the server.

Build the site you want here first. When it looks right, move on to Part 2.

> You can develop the whole site this way and deploy only when you are ready.
> Nothing below changes how `basil --dev` works.

---

## Part 2 — The server, once

SSH into your server and run **init in server mode**. This is the same command,
plus `--server` and two required facts about the deployment:

```bash
basil --init /srv/mysite --server --host mysite.example.com --admin sam
```

- **`--host`** is your public hostname. It goes on the certificate and into
  every printed URL. (It is *not* a bind address — a box behind a load balancer
  or NAT still works; see [`server.bind`](configuration.md).)
- **`--admin`** names the first account. It is never guessed from the shell,
  because this usually runs as `root`.

This one command builds the whole server-side layout, commits a placeholder
"release 1" so the server can start and get its certificate immediately,
installs the Git receive hooks, creates your account, and **prints an API key
once**:

```
  API key: bsl_live_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
  This is shown once and is not recoverable. Save it now.
```

**Save that key now** — in your password manager. It is the password for both
pushing and publishing, and it is never shown again. (Lost it? You can always
make another on the box with `basil apikey create`.)

Two notes on running it:

- If you run it with `sudo`, it hands the created tree to your user and tells you
  so. If you run it as bare `root`, it prints the `chown` command to run — do
  that before starting the server, or the server won't be able to write.
- The site root is created at the path you name (`/srv/mysite` above). Pick
  somewhere only you (or root) can write.

Now start the server:

```bash
basil --site /srv/mysite
```

On first start it fetches the certificate. If that fails, the two usual causes
are DNS not yet pointing at this box and port 80 not reachable — run
`basil check --site /srv/mysite`, which tells you which. Once it is up,
`https://mysite.example.com` serves the placeholder site.

> **Keep it running.** For a real deployment you'll want the server under a
> process manager (systemd, etc.) so it restarts on reboot. That is standard
> ops and out of scope here — for a first test, a plain `basil --site …` in a
> terminal (or `tmux`) is fine.

---

## Part 3 — Connect the two

Back **on your laptop**, in your `mysite` folder. Three things: point the config
at the server, add the server as a Git remote, and make the first publish.

### 1. Make the config describe the server

Your local `basil.yaml` says `localhost`. That file ships *inside* the release
and becomes the server's configuration, so before the first push, edit its
top-level `server` block to describe production:

```yaml
server:
  host: mysite.example.com    # was: localhost
  port: 443                   # was: 8080
```

Nothing else needs to change — `https.auto` is on by default, so no `https:`
block is required to get a certificate. And this costs you nothing locally:
`basil --dev` still serves plain HTTP on `localhost:8080` regardless of these
values.

> **Why this matters.** If you leave `localhost`/`8080` in and publish, the
> server would — at its next restart — move the public site to port 8080 and
> take its Git endpoint (your way back in) with it. Basil warns you at push time
> if you do, but it is easiest to just set the two values now. Commit the edit:
>
> ```bash
> git add basil.yaml && git commit -m "Point config at the server"
> ```

### 2. Add the server as `origin`

```bash
git remote add origin https://sam@mysite.example.com/.git
```

The `sam@` in the URL only picks which saved credential your machine offers —
Basil authenticates on the **key**, not the name. Use your real account name
anyway; it keeps credentials tidy if you ever have more than one. (The full
story: [the two things Git calls a username](git.md#the-two-things-git-calls-a-username).)

### 3. Push your history, then publish

```bash
git push -u origin main
```

Git will ask for a password: **paste your API key** (the `bsl_live_…` from
Part 2). On macOS and Windows your OS keychain remembers it and you won't be
asked again. (On Linux, Git often has no credential helper configured — see
[remembering your key](git.md#remembering-your-key-credential-storage) before
you reach for the plaintext option.)

Your commits are now on the server, but nothing is live yet — the placeholder is
still serving. Make it live:

```bash
basil publish
```

The very first publish is special. Your history and the server's placeholder are
unrelated, so replacing the placeholder is a one-time forced move that Git
normally refuses. `basil publish` recognises exactly this state, explains it, and
does it for you once you confirm:

```
First publish to "live" on origin.

This server has only its initial placeholder site. Publishing will replace it
with your project's history. This is a one-time replacement; afterwards the
release branch is protected normally.

1 commit:
  36f38ce my first page

Replace the starter site and publish 1 commit to "live"? [y/N] y

remote: replacing the starter site created by 'basil --init' with your first release …
remote: Checking release 36f38ce44dc4… ok
remote: Deploying… done (16ms)
```

Reload `https://mysite.example.com` — your site is live.

After this, the release branch is protected like any other: that forced move is
allowed **exactly once**, and every publish from here on is an ordinary
fast-forward. `basil publish` will never offer a force again unless it sees this
same never-yet-deployed state.

---

## Part 4 — The daily loop

From now on, it is the same two verbs, and you never touch the server:

```bash
# in your mysite folder, on your laptop
basil --dev                       # preview at localhost:8080 while you work
# ... edit site/… , public/… ...

git add -A
git commit -m "New homepage"
git push                          # shares with the server. Still nothing live.

basil publish                     # checks the release and makes it live
```

`basil publish` shows you what you're about to publish and asks before it does
anything:

```
$ basil publish
Publishing 2 commits to "live" (a1b2c3d..4f2a1c9):
  site/index.pars
  public/style.css
Continue? [y/N] y
remote: Checking release 4f2a1c9… ok
remote: Deploying… done (0.4s)
```

If a release has a broken `.pars` file or an invalid config, the check **rejects
it and the live site does not change** — you fix it, commit, and publish again.
Production never wobbles.

### Handy commands

All of these run on the server (or with `--site /srv/mysite` from anywhere on
the box):

```bash
basil status   --site /srv/mysite   # what's live, and whether it's behind the branch
basil releases --site /srv/mysite   # the deploy history: sha, who, when, outcome
basil rollback --site /srv/mysite   # re-activate the previous release, instantly
basil check    --site /srv/mysite   # verify hostname, ports, certificate, layout
```

`basil --dev` on your laptop also tells you at startup when the live site is
behind your local branch, so "did I publish that?" has an answer without SSHing
anywhere.

### Rolling back

Two ways, on purpose:

- **From your laptop, no SSH:** `git revert <sha> && git push && basil publish` —
  clean history, goes through the same checks.
- **On the server, right now:** `basil rollback --site /srv/mysite` — for when
  the site is broken and the fix needs to take two seconds. This is the
  emergency door, not the daily path.

---

## A second machine, or a second person

The model extends without any new setup. Anyone with an account and a key clones
the same server:

```bash
git clone https://sam@mysite.example.com/.git mysite
cd mysite && git config core.hooksPath .githooks   # enable the format-on-commit hook
```

Now they have their own dev partner. Feature branches push freely and are
**stored, not published** (only the release branch, `live`, goes live). Merge to
`main` and `basil publish` when it's ready. Your work-in-progress never has to
pass through production to reach your other laptop.

---

## When something goes wrong the first time

| Symptom | Most likely cause |
|---|---|
| Server won't start; certificate error | DNS not pointing at the box yet, or port 80 blocked. Run `basil check`. |
| `git push` rejected: *"branch is currently checked out"* | You're pushing to an old-style repo, not a `--server` site root. Re-check Part 2. |
| Push asks for the password every time (Linux) | No credential helper configured — see [credential storage](git.md#remembering-your-key-credential-storage). |
| *"Forbidden: pushing requires the editor or admin role"* | Your account is a `viewer`. The first account is admin; others need `editor`+. |
| *"Git over plain HTTP is refused"* | You cloned with `http://`. Use `https://`. |
| `basil publish` says nothing to publish, but you expected a first publish | You haven't `git push`ed your branch yet, or the histories aren't actually unrelated. |
| Every write on the server fails after `sudo … --init` | The tree is root-owned. Run the `chown` the init printed. |

The [Git guide's troubleshooting section](git.md#troubleshooting) covers each of
these in depth.

---

## See also

- **[Deployment](deployment.md)** — the deploy pipeline, all five operator
  commands, and the site-root layout in full.
- **[Git over HTTPS](git.md)** — cloning, pushing, publishing, authentication,
  credential storage, and what is refused.
- **[Configuration](configuration.md)** — the one config file across machines,
  and which settings are operator-owned on a site root.
- **[Quick Start](basil-quick-start.md)** — the local-only path, for when you
  just want to build.
