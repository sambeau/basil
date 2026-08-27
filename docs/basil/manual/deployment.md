---
id: man-bas-deployment
title: "Deployment"
system: basil
type: feature
name: deployment
created: 2026-07-12
version: 1.0.0-alpha.6
author: "@sam"
keywords:
  - deployment
  - production
  - git deploy
  - publish
  - rollback
  - https
  - tls
  - lets encrypt
  - fly.io
  - security headers
  - compression
  - cors
  - caching
---

# Deployment

Deploying a Basil site has two halves, and they have very different rhythms.

**Setting up a server** happens once. You put the binary on a machine, run one
command, point DNS at it, and start it. Basil gets its own HTTPS certificate.

**Shipping the site** happens all the time, and it is just Git. You `git push`
to share work, and `basil publish` when you want it live. Basil reads the
release before serving it and refuses one that would break the site.

This page covers both, starting with the part you will do every day.

## The shape of a deployed site

`basil --init <dir> --server` builds this on the machine that receives deploys:

```
/srv/mysite/
  site.git/                       the shared Git repository, served at /.git
  releases/
    4f2a1c9…/                     one directory per deployed commit
  current -> releases/4f2a1c9…    the active release
  data/                           databases, certificates, uploads — no deploy touches this
```

Two facts do most of the work here. Each release is a directory named after its
commit and byte-identical to it, so Basil never rewrites your code and the
deploy record can be trusted. And `current` is a symlink, so going live is an
atomic re-point of one link, and rolling back is the same thing in reverse.

Between your laptop and that server sit two verbs, and keeping them apart is the
whole design:

```
git push          →  shared with the team. Stored on the server, published to nobody.
basil publish     →  the release branch moves, the release is checked, and the site goes live.
```

The safe one is the one you get for free. Going public is the one you ask for.

## Two ways to start

There are two ways a site and a server meet, depending on which existed first.
Both end in the same place — a clone on your laptop, a bare repository on the
server, and `basil publish` as the go-live button.

| | Start here if… | The first publish is… |
|---|---|---|
| **[Scenario 1: server first](#scenario-1-start-on-the-server)** | you are starting something new | ordinary |
| **[Scenario 2: laptop first](#scenario-2-put-a-site-you-already-have-on-a-server)** | you already have a Basil folder you have been working in | a one-time replacement of the starter site |

Everything after the first publish is identical, and is [the daily
loop](#the-daily-loop) below.

---

### Scenario 1: Start on the server

The server is created first and holds the only copy of the history. You clone
it, and that clone is your dev partner.

#### 1. Build the site root — *on the server*

```bash
basil --init /srv/mysite --server --host mysite.example.com --admin sam
```

`--server` is what asks for the deploy topology; without it you get a [plain
local folder](running.md#creating-a-site), which is the right shape for a laptop
and the wrong one for a server. Neither `--host` nor `--admin` is guessed —
`--host` becomes `server.host` (the name on the certificate and the name people
type), and `--admin` names the first Basil account, which is not derived from
`$USER` because this command usually runs as `root` and "root deployed 4f2a1c9"
tells a team nothing.

One run creates the layout above, commits the starter site, deploys it as
release 1, installs the receive hooks, creates the admin account, and prints:

```
  API key: bsl_live_Sl9j5NeWlZR0OV5C915W3G-gGok8Lee2Pn90T5i1BM4
  This is shown once and is not recoverable. Save it now.
```

**Save that key before you close the terminal.** It is the password for both
`git push` and `basil publish`. If you lose it, make another with
`basil apikey create --user usr_…`.

Release 1 exists because a server with no release cannot serve, so it cannot
answer a certificate challenge, so it could never receive the push that would
give it a release.

#### 2. Point DNS at the machine

An `A` record (and `AAAA` if you have IPv6) for `mysite.example.com`, pointing
at the server's public address. Wait until `dig +short mysite.example.com`
returns it from somewhere that is not the server. Let's Encrypt looks your
domain up on the public internet, so DNS that has not propagated will fail.

Ports **80 and 443** need to be open to the internet — see [HTTPS](#https)
below for why 80 matters and what to do about ports under 1024.

#### 3. Start the server

```bash
basil --site /srv/mysite
```

Basil obtains its certificate within a few seconds of starting, without waiting
for a visitor. If anything is wrong, ask:

```bash
basil check --site /srv/mysite
```

which tests the layout, the active release, the repository, `server.host`, DNS,
port 80 and the certificate, and names the fix for each.

#### 4. Clone it — *on your laptop*

The init output ends with this command, with your hostname and account name
already filled in:

```bash
git clone https://sam@mysite.example.com/.git mysite
Password: <paste the API key>       # your OS keychain remembers it
cd mysite
git config core.hooksPath .githooks # opt into the pre-commit formatting hook
```

The `sam@` in the URL only selects which stored credential your machine offers;
Basil ignores it and authenticates on the key alone. Use your real account name
anyway — credential helpers key on *(host, username)*, and two accounts sharing
one entry is annoying to debug later.

#### 5. Work locally

```bash
basil --dev
```

Plain HTTP on `localhost:8080`, live reload, detailed error pages, and every
edit — to a page or to a component several imports below it — picked up on the
next request. The clone is an ordinary Basil folder; nothing about it is
special.

#### 6. Share, then publish

```bash
git add -A
git commit -m "New homepage"
git push          # on the server, published to nobody
basil publish     # review, confirm, live
```

That is the loop from here on.

---

### Scenario 2: Put a site you already have on a server

You have been building in a folder made by `basil --init`, and now it should be
public. Nothing about the folder has to be restructured — the server gets its
own init, your folder gets a remote, and the first publish carries your history
across.

#### 1. Make the config describe the server — *on your laptop*

Open `basil.yaml` and change the top-level `server` block from your laptop to
the machine:

```yaml
server:
  host: mysite.example.com    # was: localhost
  port: 443                   # was: 8080
```

Do this **first**, before anything is pushed. `basil.yaml` ships inside the
release and *becomes* the server's configuration, so a deployed `host:
localhost, port: 8080` would, at the server's next restart, move the public site
to a port nobody is asking for — and take the Git endpoint, your only remote way
back in, with it. Basil warns you at push time if a release moves the listener,
but it is easier not to.

The edit costs you nothing locally: `basil --dev` serves plain HTTP on
`localhost` and turns a production port 443 back into 8080. Same command in the
same folder, before and after. `https.auto` defaults to on, so a config with no
`https:` block still gets its own certificate.

#### 2. Build the site root — *on the server*

```bash
basil --init /srv/mysite --server --host mysite.example.com --admin sam
```

Same command as Scenario 1, same one-time API key, same warning about writing it
down. Then point DNS at the machine and start it:

```bash
basil --site /srv/mysite
```

#### 3. Add the server as `origin` — *on your laptop*

```bash
git remote add origin https://sam@mysite.example.com/.git
git push -u origin main
```

The API key is the password. Your history is now on the server — stored, and
published to nobody. Note that this pushed `main`, which is not the release
branch (`live` is), so nothing has gone live yet.

#### 4. The first publish

```bash
basil publish
```

This one is special, and Basil says so:

```
First publish to "live" on origin.

Your project's history is unrelated to what this server currently publishes,
so publishing will force-replace the release branch with your history. The
server allows this only if it has never had a real release - otherwise it will
refuse, and nothing changes.

2 commits:
  babd5b5 Add an about page
  942a185 Initial site

6 files changed:
  .githooks/pre-commit
  .gitignore
  basil.yaml
  public/.keep
  site/about/about.pars
  site/index.pars

Replace the starter site and publish 2 commits to "live"? [y/N] y

Publishing babd5b5e7b93 to "live" (replacing the starter site)...
remote: replacing the starter site created by 'basil --init' with your first release — this is the one non-fast-forward the release branch allows, and it will not be allowed again
remote: Checking release babd5b5e7b93… ok
remote: Deploying… done (10ms)
```

The server seeded the release branch with a starter commit of its own, so your
history and its history are unrelated: moving `live` onto your commit is a
non-fast-forward, which Git normally refuses to send and Basil normally refuses
to accept.

The server allows it **exactly once** — while the deploy record still shows
nothing but the release `--init` made. From the first real release onward,
force-pushing or deleting the release branch is [refused for
everyone](https://github.com/sambeau/basil/blob/main/docs/guide/git.md#what-is-refused),
because rollback depends on that history staying put. `basil publish` will not
offer the forced path again either.

This is only offered for genuinely unrelated histories. A clone that has merely
fallen behind a shared release branch is an ordinary divergence, and publish
refuses it and tells you to fetch and rebase.

#### 5. Keep working in the same folder

```bash
basil --dev        # exactly as before
```

Your folder is now also a clone. Nothing else changes.

---

## The daily loop

Once either scenario is done, this is all of it:

```bash
# ...edit...
git add -A
git commit -m "New homepage"    # the pre-commit hook formats staged .pars files
git push                        # share. Not public.
basil publish                   # go live.
```

`git push` is the one you run twenty times a day and the harmless one. `basil
publish` runs in your clone, asks the server which branch releases, and shows
you what is about to go out before it moves anything:

```
$ basil publish
Publishing to "live" on origin (96c8953d0186..f7d5d5258029).

1 commit:
  f7d5d52 Add an about page

1 file changed:
  site/about/about.pars

drift: the release branch "live" on origin is 1 commit behind HEAD (this publish closes it).

Publish 1 commit to "live"? [y/N] y

Pushing f7d5d5258029 to "live"...
remote: Checking release f7d5d5258029… ok
remote: Deploying… done (15ms)
To https://mysite.example.com/.git
   96c8953..f7d5d52  HEAD -> live
```

Nothing is sent until you answer the prompt — an empty line, or anything but
`y`, cancels. The `remote:` lines are the server's deploy pipeline running while
the push is still open, streamed to your terminal as it happens.

- **`basil publish --dry-run`** prints the same plan and stops without pushing.
- **`basil publish --yes`** skips the confirmation, for scripts.
- It needs no setup in any clone: it learns the release branch from the server
  and configures the push refspec on first use.

Sharing work *without* publishing it is just a push of a branch that is not the
release branch:

```bash
git checkout -b new-shop
git push -u origin new-shop     # stored on the server, published to nobody
# ...when everyone's happy:
git checkout live && git merge new-shop
basil publish                   # now it goes live
```

`basil publish` is a convenience over a plain push, not a new protocol, so `git
push origin live` still publishes exactly the same way — minus the plan and the
confirmation. Use your editor's Git panel if you prefer.

### Which branch publishes

The release branch is `site.git`'s `HEAD` — Git's own record of the default
branch, and what a fresh clone checks out. `--init --server` points it at
`live`. One command on the server changes it:

```bash
git -C /srv/mysite/site.git symbolic-ref HEAD refs/heads/main
```

Clients need no change; `basil publish` asks the server. It is deliberately a
fact about the *server* rather than a setting in `basil.yaml`, because the
config ships inside the release — a deployed config naming the release branch
could point the protections at some other branch and leave the real one freely
force-pushable.

If you would rather every push go live immediately, point `HEAD` at the branch
you work on. Then `git push` *is* publishing, which is the older push-to-publish
model, and everything else here works unchanged.

## What a publish does

A push that moves the release branch runs the whole pipeline, and so does
`basil deploy` at the server's shell:

1. **Resolve** the branch, tag or SHA to a single commit.
2. **Lock** — one deploy at a time per site.
3. **Materialise** the commit into `releases/<sha>/`, with no `.git` inside.
4. **Validate** — every `.pars` file is parsed and `basil.yaml` is loaded and
   checked. Errors reject the release.
5. **Activate** — `current` is atomically re-pointed.
6. **Hook** — `deploy.pars` in the release root runs, if it exists.
7. **Record** — commit, timestamp, duration, outcome and both identities.
8. **Prune** — releases beyond `deploy.keep` (default 5) are removed, never the
   active one.

Any failure before activation leaves the previous release live and untouched.
The running server notices `current` moving and activates the new release
itself, within about a second — no restart, and requests already in flight
finish on the release they started on.

**What applies live, and what needs a restart.** Routes, handlers, static
mounts, `site.path`, `public_dir`, error pages and caching TTLs are rebuilt on
activation. Listener settings (`server.port`, `server.bind`, `server.host`,
`https.*`), middleware settings (`logging`, `compression`, `security`, `cors`),
subsystem toggles and image limits are bound at startup and take effect on the
next restart. Only the listener changes are warned about, in the terminal you
pushed from.

### When a release is refused

Validation is the gate between materialise and activate. A release that does not
parse never becomes the live site:

```
remote: Checking release 5d56492c6437…
remote: site/broken/broken.pars:1:9: unexpected character 'Unterminated string starting with "broken"'
remote: Release rejected. The live site is unchanged (still 82af36e1c9e0).
remote: error: release 5d56492c6437 refused
 ! [remote rejected] HEAD -> live (pre-receive hook declined)
```

The release branch did not move, the site is untouched, and `basil publish`
exits non-zero. Fix the reported error and publish again.

Validation catches code that is **broken**, not work that is **unfinished** — it
is parse and config-load only. Deciding *when* to publish is still your job. It
is also correctness rather than style: unformatted `.pars` files produce a
warning naming `basil fmt -w`, and never a rejection.

### Rolling back

Rollback re-activates a release that is already on disk. It is a symlink swap
and a cache clear, which is what makes it fast enough to be the emergency
answer:

```bash
basil rollback --site /srv/mysite          # back to the previous release
basil rollback a49d218 --site /srv/mysite  # or name one from basil releases
```

### Looking at what happened

```bash
basil status    --site /srv/mysite   # what is live, and whether the branch is ahead
basil releases  --site /srv/mysite   # the deploy record; * marks the live release
basil check     --site /srv/mysite   # DNS, port 80, certificate, layout, repository
```

Every deploy, rollback, rejection, failure and no-op is recorded in
`<data_dir>/deploy.db`, readable with or without the server running:

```
$ basil releases --site /srv/mysite
  SEQ  RELEASE       WHEN              TRIGGER   PUBLISHER       AUTHOR                OUTCOME
--------------------------------------------------------------------------------------------
* 2    f7d5d5258029  2026-08-27 12:41  push      sam             Sam                   deployed
  1    96c8953d0186  2026-08-27 12:41  init      init            Basil                 deployed

* = the live release
```

Each entry stores two identities, because they routinely differ: the
**publisher** who triggered the deploy (the Basil account behind the API key for
a push, the OS account for a shell deploy) and the **author** who wrote the
commit.

The full pipeline, all five operator commands, the post-deploy hook and the
deploy record are covered in the [Deployment
guide](https://github.com/sambeau/basil/blob/main/docs/guide/deployment.md);
cloning, credentials, roles and what is refused are in the [Git
guide](https://github.com/sambeau/basil/blob/main/docs/guide/git.md).

## Running on Fly.io

If you would rather not keep a VPS, [`contrib/fly/`](https://github.com/sambeau/basil/tree/main/contrib/fly)
has a tested recipe for running Basil on a [Fly Machine](https://fly.io/docs/machines/):
a Dockerfile, an entrypoint, a `fly.toml` and a build script. Roughly:

```sh
fly apps create my-basil-site
fly ips allocate-v4                # a dedicated address; see the README for why
./build.sh && fly deploy           # ships Basil
fly ssh console -C "basil --init /srv/mysite --server --host mysite.example.com --admin sam"
fly machine restart
```

After that it is Scenario 1 or 2 above, unchanged — `fly deploy` ships *Basil*,
`git push` and `basil publish` ship *the site*, and the two lifecycles stay
separate. Basil terminates TLS itself with its own Let's Encrypt certificate, so
port 443 is raw TCP passthrough.

The [Fly.io README](https://github.com/sambeau/basil/blob/main/contrib/fly/README.md)
has the full walkthrough, the reasons behind each setting, and the handful of
platform quirks that are not guessable — read it before you start rather than
after.

## HTTPS

Production mode always serves HTTPS. Without an `https:` section, Basil refuses to start with `HTTPS requires either auto: true or cert/key paths`. Development mode (`basil --dev`) serves plain HTTP on localhost and needs no certificates.

You have three ways to get a certificate: let Basil fetch one from Let's Encrypt (the usual choice), supply your own files, or make a self-signed one for local testing.

### Automatic certificates with Let's Encrypt

[Let's Encrypt](https://letsencrypt.org) is a free, automated certificate authority. You do not sign up, pay, or install a client: Basil talks to it directly using the ACME protocol, proves it controls your domain, receives a certificate, and renews it before it expires. All you provide is a domain and an email address.

#### Before you start

1. **A domain name** you control, e.g. `example.com`.
2. **DNS pointing at the server.** Add an `A` record (and an `AAAA` record if you have IPv6) for your domain to the server's public IP. Wait until `dig example.com` or `nslookup example.com` returns that IP from outside the server. Let's Encrypt looks your domain up on the public internet, so private or not-yet-propagated DNS will fail.
3. **Ports 80 and 443 open** to the internet in any firewall or cloud security group. Basil listens on both: 443 for your site, and 80 for a small second server that answers Let's Encrypt's verification requests and redirects everything else to HTTPS. Opening the firewall is your job; Basil only binds the ports. Port 80 is always `:80`, whatever `server.port` is set to. If it is blocked, Let's Encrypt can usually still verify you through port 443, but visitors who type `example.com` without `https://` get a connection error instead of a redirect.
4. **Permission to bind those ports.** On Linux, ports below 1024 need root or the `CAP_NET_BIND_SERVICE` capability. The cleanest option is to grant the capability to the binary once, then run Basil as an ordinary user:

   ```bash
   sudo setcap 'cap_net_bind_service=+ep' /usr/local/bin/basil
   ```

#### Configuration

```yaml
server:
  host: example.com          # the domain the certificate is for
  port: 443
https:
  auto: true
  email: admin@example.com   # expiry and problem notifications from Let's Encrypt
  cache_dir: ./certs         # where certificates are stored, relative to data_dir
```

| Key | Required | What it does |
|---|---|---|
| `server.host` | yes | The public hostname: the domain to request a certificate for, and the name people type. It is **not** a bind address — the listener uses `server.bind` (empty means all interfaces), so a host behind NAT, a container, or a load balancer still starts. Basil only answers certificate requests for this exact name, and **refuses to start without it** — an empty host would let anyone trigger issuance for any name they put in SNI. `--dev` and a manual `https.cert`/`https.key` are the exceptions. |
| `https.auto` | yes | Turn on Let's Encrypt. |
| `https.email` | recommended | Let's Encrypt emails this address before a certificate expires and if it has to revoke one. Optional, but there is no other way to hear about problems — and Basil warns at every start without it. |
| `https.cache_dir` | no | Directory for the certificate, private key, and Let's Encrypt account key. Relative to the site's data directory; defaults to `<data_dir>/certs`. |

Setting `auto: true` accepts the [Let's Encrypt Subscriber Agreement](https://letsencrypt.org/repository/) on your behalf.

#### What happens on first start

Start Basil in production mode:

```bash
basil --site /srv/mysite
```

The log shows `automatic TLS enabled via Let's Encrypt (cache: …)` and Basil listens on 443 and 80. It then obtains the certificate straight away, without waiting for a visitor:

1. Basil creates a Let's Encrypt account (keyed to your email) and stores the account key in `cache_dir`.
2. Let's Encrypt asks Basil to prove it controls `example.com`. Basil answers the challenge itself — either over port 80 (HTTP-01) or inside the TLS handshake on 443 (TLS-ALPN-01), whichever Let's Encrypt tries first.
3. Let's Encrypt issues a certificate. Basil stores it in `cache_dir` and uses it for every request from then on.

Either `TLS certificate ready for example.com` or a failure naming the two usual
suspects — DNS and port 80 — appears in the log within a few seconds. A failure is
not fatal: the server keeps running and tries again on the first HTTPS request.
Check it yourself with:

```bash
curl -I https://example.com
```

You want `HTTP/2 200` and no certificate warning. If `curl` complains, see [Troubleshooting](#troubleshooting) below.

#### Renewal

Let's Encrypt certificates last 90 days. Basil checks the certificate on each request and renews it automatically when it is within 30 days of expiry, using the same challenge as before. There is no cron job to set up and no restart needed. The one thing renewal needs is that ports 80 and 443 stay reachable from the internet — if you later close port 80 behind a firewall, renewal can still succeed over 443, but keep 80 open to be safe.

#### The `certs` directory

`cache_dir` holds your private key. Treat it accordingly:

- It lives in the site's data directory (`<data_dir>/certs` by default), which is outside the release and outside the repository, so a deploy never replaces it and there is nothing to gitignore.
- Make it readable only by the user Basil runs as: `chmod 700 certs`.
- Keep it between deploys and restarts. If you delete it, Basil requests a fresh certificate on the next start, which counts against Let's Encrypt's rate limits.
- Back it up with the rest of the site if you want restarts on a new machine to be instant, but a lost `certs` directory is not a disaster — Basil simply fetches a new certificate.

#### Rate limits

Let's Encrypt allows [5 certificates per week for the same set of names](https://letsencrypt.org/docs/rate-limits/) and a handful of failed attempts per hour. Normal use never approaches this. You can hit it by repeatedly deleting `certs/` while debugging, or by running several copies of Basil for the same domain, each with its own `cache_dir`. If you do, the error mentions `rateLimited` and you have to wait up to a week. Debug DNS and firewall problems with `curl` *before* pointing Basil at a domain, not after.

#### `www` and other names

Basil requests a certificate for `server.host` and nothing else. A certificate for `example.com` does not cover `www.example.com`. Pick one name as canonical, set it as `server.host`, and send the other to it at the DNS level: a `CNAME` will not help on its own (it still resolves to this server, which will refuse the name), so use your DNS provider's redirect feature, or handle the second name on a reverse proxy in front of Basil.

If `server.host` is empty, Basil will request a certificate for *any* hostname that reaches it. Do not run like that on the internet — anyone pointing a domain at your IP could trigger requests against your rate limits. Always set `server.host` in production.

### Bringing your own certificate

If you already have a certificate — from a corporate CA, a commercial provider, or a tool like `certbot` or `acme.sh` you run for other services — point Basil at the files:

```yaml
server:
  host: example.com
  port: 443
https:
  cert: /etc/ssl/example.com/fullchain.pem
  key: /etc/ssl/example.com/privkey.pem
```

`cert` should be the full chain (leaf plus intermediates), PEM-encoded. When `cert` and `key` are set, `auto` is ignored. Basil reads the files once at startup and does not watch them: after renewing them, restart Basil. (`SIGHUP` re-activates the current release, not the certificates.)

### Self-signed certificates for local HTTPS

To test production mode on your own machine — for example to check passkeys, which need HTTPS — make a throwaway certificate:

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem \
  -days 365 -nodes -subj "/CN=localhost"
```

```yaml
server:
  host: localhost
  port: 443
https:
  cert: ./cert.pem
  key: ./key.pem
```

Your browser will warn that the certificate is untrusted; click through, or add it to your system's trust store. Never use a self-signed certificate on a public site.

### Behind a reverse proxy

Basil can sit behind Caddy, nginx, or a cloud load balancer that terminates TLS — see [Security](https://herbaceous.net/security.html) for when that makes sense. Two things to know:

- **Basil still needs its own certificate**, because production mode is HTTPS-only and there is no setting that relaxes that. Use a self-signed one (as above) or an origin certificate from your provider; the proxy connects to Basil over HTTPS on an internal port and does not need to trust it. A proxy that terminates TLS and talks plain HTTP to Basil will not work — `basil.yaml` is refused at validation with `production mode requires https.auto=true or both https.cert and https.key`.
- Basil reads `X-Forwarded-For` and `X-Real-IP`, so rate limiting and logs see the real client address. Make sure the proxy sets them.

Leave `auto: true` off in this setup: the proxy holds the public certificate, and Let's Encrypt's challenges would never reach Basil.

### Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `HTTPS requires either auto: true or cert/key paths` on start | No `https:` section | Add one, or run with `--dev` for local work |
| `bind: permission denied` on start | Not allowed to open port 80 or 443 | `setcap` as above, or run as root |
| `bind: address already in use` | Another web server (Apache, nginx, a previous Basil) holds the port | Stop it, or put Basil on another port behind it |
| HTTPS works but `http://example.com` does not redirect | Something else holds port 80, so Basil's redirect server failed — this is logged as `HTTP redirect server error` and is not fatal | Stop whatever owns port 80, then restart Basil |
| First request hangs, then fails with a TLS error | Let's Encrypt could not reach the server | Check DNS resolves to this IP from outside; check ports 80 and 443 are open in every firewall |
| Log shows `acme: ... urn:ietf:params:acme:error:dns` | DNS not propagated or pointing elsewhere | Wait for propagation; confirm with `dig` |
| Log shows `urn:ietf:params:acme:error:rateLimited` | Too many certificates requested recently | Stop deleting `certs/`; wait up to a week |
| `curl: (60) SSL certificate problem` | Hostname in the URL does not match `server.host` | Use the exact configured name; add `www` redirect at DNS |
| Works for `example.com`, not `www.example.com` | Certificate is for one name only | See [`www` and other names](#www-and-other-names) |

`basil check --site /srv/mysite` tests most of the above in one go and names the fix for each. Basil also logs ACME errors to standard error — run it in the foreground while setting up so you can see them.

## Security Headers

Basil sets safe defaults on every response — `Strict-Transport-Security` (1 year), `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin` — customisable under the `security:` config section.

## File-System Security

Handlers can read files but not write, unless you whitelist directories:

```yaml
security:
  allow_write:
    - ./data
    - ./uploads
```

Relative paths resolve against the data directory, so a deploy never destroys what is written there. `<data_dir>/uploads` is always writable and is served at `/__uploads/`.

## Compression

gzip is on by default; tune or disable it:

```yaml
compression:
  enabled: true
  level: default     # fastest | default | best | none
  min_size: 1024
```

## Response Caching

Production mode caches responses. In site mode, set a TTL:

```yaml
site:
  path: ./site
  cache: 5m
```

Only successful `GET` responses are cached, keyed on path and query string, and
the response carries `X-Cache: HIT` or `X-Cache: MISS` so you can tell which you
got. [Parts](parts.md) requests are never cached and never served from the
cache, so interactive Parts keep working inside a cached page — only the page's
initial render is frozen. See [Parts and Response
Caching](parts-guide.md#parts-and-response-caching).

`basil --dev` does not cache responses, and picks up an edit to any handler or
component on the next request. `dev.cache: true` opts back into production-like
caching for profiling — see [`dev.cache`](configuration.md#devcache).

## CORS

If browsers on other origins call your API:

```yaml
cors:
  allowed_origins: ["https://app.example.com"]
  allowed_methods: [GET, POST, PUT, DELETE]
  allowed_headers: [Content-Type, Authorization]
  max_age: 1h
```

See the [CORS guide](https://github.com/sambeau/basil/blob/main/docs/guide/cors.md) for preflight details and debugging.

## Upgrading Basil itself

Replace the binary and restart. The site root is untouched, so the repository,
the releases, the accounts and the deploy record all survive. At startup Basil
rewrites the receive hooks if the binary path has moved, so an upgrade does not
silently stop deploying.

`SIGHUP` re-activates whatever release `current` points at, as a manual fallback
when the deploy watcher could not start (the server logs a warning naming SIGHUP
in that case).

**A site that works under `basil --dev` but fails in production is usually two
different versions of Basil.** Your laptop gets a new binary every time you
rebuild; the server only gets one when you put it there. Ask it:

```bash
basil --version
```

## Sessions Across Instances

Running more than one instance behind a load balancer? Give them a shared session secret:

```yaml
session:
  secret: !secret ${SESSION_SECRET}
```

## See Also

- [Git Deploy](git.md) — the endpoint, API keys, roles, and credential storage
- [Configuration](configuration.md) — every section referenced above
- [Running Basil](running.md) — `--init`, flags, profiles, and signals
- [Deployment guide](https://github.com/sambeau/basil/blob/main/docs/guide/deployment.md) — the pipeline, all five operator commands, and the post-deploy hook
- [Fly.io recipe](https://github.com/sambeau/basil/blob/main/contrib/fly/README.md) — a walkthrough for running Basil on a Fly Machine
