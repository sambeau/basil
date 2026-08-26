# Basil on Fly.io

A working recipe for running a Basil server on a [Fly Machine](https://fly.io/docs/machines/):
a Dockerfile, an entrypoint, a `fly.toml`, and a script to build the binary.

These are a starting point, not a product. Copy them into your own app
directory, change the two names at the top of `fly.toml`, and read the
comments — they try to explain the choices rather than just make them.

## The mental model

The thing that trips people up on Fly is that you do not *install* software
onto a Machine. A Machine is a microVM booted from an OCI image, and the image
*is* the filesystem. When you `fly ssh console` into a fresh one and find no
`curl` and no `git`, nothing is missing — you simply have not put them there
yet. The Dockerfile is where you say what the box contains.

That suits Basil well, because Basil is one binary.

## Why a Machine rather than a Sprite

[Sprites](https://fly.io/sprites/) are more durable than they first look: they
sleep rather than die, they checkpoint their filesystem, and they come back in
about a second with everything intact. They are a pleasant place to *develop*.

But a Basil server wants to be awake. It renews its own certificate, it
watches the `current` symlink for a deploy to activate, and it answers on a
domain you chose. Sprites are shaped for agent workloads that idle cheaply and
wake on demand; a Machine with a Volume is shaped for a long-lived site. This
directory does the latter.

## What Basil needs on the machine

Less than you would think, and one thing more than you might expect:

- **The binary**, built static — see the note on `-tags nodynamic` below,
  because this is the step with a trap in it.
- **`git`.** A real runtime dependency, not a convenience. Basil shells out to
  it for the Git endpoint (`git-upload-pack`, `git-receive-pack`), to
  materialise a release (`git archive`), and to read the release branch
  (`git symbolic-ref`).
- **`/bin/sh`.** The receive hooks Basil installs are two-line shell scripts.
  So `scratch` and distroless images are out; Alpine is fine.
- **CA certificates**, for ACME and for anything a handler calls out to.
- **A persistent site root.** A `fly deploy` replaces the root filesystem, so
  `site.git`, the releases, `current`, `data/deploy.db`, the auth database and
  the certificate cache all have to be on a Volume.

Nothing needs `curl`. Nothing fetches anything at boot. No Go toolchain, no
build stage, no C compiler.

## The trap: `CGO_ENABLED=0` is not enough

Basil looks like it should cross-compile to a static binary and mostly does —
SQLite is `modernc.org/sqlite`, WebP decodes through wazero, nothing wants a C
toolchain. But `server/images` imports `github.com/gen2brain/webp`, which
pulls in `ebitengine/purego` to `dlopen()` a system libwebp when it can find
one. That leaves the binary asking for glibc's loader at
`/lib/ld-linux-*.so.1` even with cgo disabled, and on Alpine — whose loader is
musl's — you get a thoroughly confusing `not found` for a binary that is
plainly sitting right there, executable, in `/usr/local/bin`.

`build.sh` passes `-tags nodynamic`, which drops that path and uses the
embedded WASM decoder only. That costs nothing here: a container with no
libwebp installed would have fallen back to WASM at runtime anyway.
`go test -tags nodynamic ./server/images/...` passes.

The script checks the result and warns if the binary comes out dynamic. If it
ever does, the escape hatch is to base the image on `debian:bookworm-slim`
instead of Alpine and accept the glibc loader.

## The recipe

```sh
# 1. an app and a dedicated IPv4 (see "TLS" below for why dedicated)
fly apps create my-basil-site
fly ips allocate-v4

# 2. point your domain's A record at that address, and wait for it
dig +short mysite.example.com

# 3. build the binary and ship the image
./build.sh
fly deploy

# 4. initialise the site root, once, on the volume
fly ssh console -C "basil --init /srv/mysite --server --host mysite.example.com --admin sam"
```

Step 4 prints an API key **once**. Write it down before you close the
terminal; it is not recoverable, and it is the password for both `git push`
and `basil publish`.

The entrypoint deliberately refuses to run that init for you. It could — but
the key would land in `fly logs`, which is the wrong home for a credential.
Until the site root exists the entrypoint holds the Machine open and prints
what to run, so there is always something to `fly ssh console` into.

```sh
# 5. start serving
fly machine restart
```

Then from your laptop, in the site folder you want to publish:

```sh
git remote add origin https://sam@mysite.example.com/.git
git push -u origin main
basil publish
```

The first publish is the special one — the server is still carrying the
starter site `--init` made, so Basil asks before replacing it. After that it is
the ordinary loop. See the [deployment guide](../../docs/guide/deployment.md).

## TLS, and why the dedicated IPv4 is not optional

`fly.toml` here puts port 443 on raw TCP passthrough (`handlers = []`): the
Fly proxy moves bytes, and Basil terminates TLS itself with its own Let's
Encrypt certificate, exactly as it would on a VPS.

Fly's *shared* IPv4 only carries the `http` and `tls` handlers, so passthrough
needs `fly ips allocate-v4`. A couple of dollars a month.

**Port 80 is the exception, and it must be.** It is where the HTTP-01 challenge
is answered, so it cannot simply be dropped — but with `handlers = []` the Fly
proxy accepts the TCP connection and then delivers nothing at all. A raw
request to port 80 through the edge came back with zero bytes while the
identical request to `127.0.0.1:80` inside the Machine got Basil's `301`
immediately. Port 443 passthrough was fine throughout; only port 80 behaves
this way. It needs `handlers = ["http"]`, which costs nothing: that port
carries the ACME challenge and a redirect to HTTPS, and no site traffic.

The obvious way to dodge that cost — let the Fly proxy terminate TLS on the
shared IP and have Basil serve plain HTTP behind it — **does not currently
work**, and fails in a way worth knowing about in advance. A release whose
config says:

```yaml
server:
  https:
    auto: false
  proxy:
    trusted: true
```

is refused by config validation before it ever goes live:

```
remote: basil.yaml: configuration errors:
remote:   - production mode requires https.auto=true or both https.cert and https.key
remote: Release rejected. The live site is unchanged (still 3f499cd1a6fb).
```

Production mode insists on HTTPS at the Basil listener (see
`validateHTTPS` in `server/config/load.go`). `server.proxy.trusted` exists for
`X-Forwarded-*`, but it does not buy an exemption from that rule, and `--dev`
— the one mode that skips it — is not what you want facing the internet. In
principle you could satisfy the check with a self-signed `https.cert` /
`https.key` pair and have the proxy re-encrypt to the backend, but that is
more moving parts than the dedicated address is worth.

Passthrough is also the more honest test: it exercises the same certificate
path a real Basil deployment takes.

## After the first deploy, stop deploying

This is the part worth internalising, and the pleasant part.

`fly deploy` ships **Basil**. `git push` and `basil publish` ship **the site**.
Once the Machine is up you do not touch Fly again to change a page — you push,
the receive hook validates the release, and the running server picks up the new
`current` within about a second. The two lifecycles are genuinely separate.

To upgrade Basil itself: `./build.sh && fly deploy`. The volume is untouched,
so the repository, the releases, the accounts and the deploy record all
survive. At startup Basil rewrites the receive hooks if the binary path has
drifted, so an upgrade does not silently stop deploying.

## Things that will bite

- **Do not let the Machine auto-stop.** A stopped Basil renews no certificate
  and activates no deploy. `auto_stop_machines = "off"`.
- **One Machine.** A Volume attaches to exactly one, and the deploy lock is a
  kernel file lock that only means anything on a local filesystem. Do not
  scale this out.
- **Keep the binary at `/usr/local/bin/basil`.** The hooks bake in an absolute
  path.
- **Port 80 needs `handlers = ["http"]`**, not passthrough — see TLS above. The
  failure mode is a silent one: TCP connects, nothing comes back, and ACME
  fails with `no viable challenge type found`.
- **A fresh volume is not empty.** ext4 puts a `lost+found` in it, and
  `basil --init` refuses a non-empty folder. Hence mounting at `/srv` with the
  site root a directory inside it, rather than mounting straight onto
  `/srv/mysite`.
- **Put `https.email` in your `basil.yaml`.** The starter config omits it and
  Basil warns at every start; without it Let's Encrypt has no contact address
  for expiry and revocation notices.
- **`basil check --site /srv/mysite` is the first thing to run when something
  is odd.** It tests DNS, port 80, the certificate, the repository placement
  and the active release, and names the fix for each.

## What has actually been tested

This recipe has been run for real, on a Machine in `lhr` with a dedicated IPv4
and a domain pointed at it. Confirmed from outside the network: Basil obtained
its own Let's Encrypt certificate through the passthrough, served the starter
site over HTTP/2 with its security headers intact, redirected port 80 to HTTPS,
and answered `/.git/info/refs` with `401 WWW-Authenticate: Basic realm="Basil
Git"`. On the box, `basil check` passed every check with the certificate
cached.

Both of the awkward findings above — the `lost+found` on a fresh volume, and
port 80 needing the `http` handler — came out of that run rather than out of
reasoning, which is the usual way.

The deploy pipeline itself was exercised separately under Docker: a `git push`
to the release branch runs pre-receive validation and a post-receive deploy end
to end, and a release with a bad config is rejected with the live site left
alone. The rejected-config transcript above is from that run.

## See also

- [Deployment](../../docs/guide/deployment.md) — the deploy pipeline and the
  operator commands
- [Getting started with a remote site](../../docs/guide/remote-site-getting-started.md)
- [Git over HTTPS](../../docs/guide/git.md) — cloning, pushing, what is refused
- [Configuration](../../docs/guide/configuration.md) — the site-root layout,
  and one file across many machines
