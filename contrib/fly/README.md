# Basil on Fly.io

Run a Basil site on a [Fly Machine](https://fly.io/docs/machines/), with its own
domain and its own HTTPS certificate, for a few dollars a month.

This directory is a working recipe: a `Dockerfile`, an `entrypoint.sh`, a
`fly.toml` and a `build.sh`. Copy them into your own app directory, change two
names in `fly.toml`, and follow the walkthrough below. It has been run for real
against a live domain.

It is a starting point rather than a product. The files are commented, and the
comments explain each choice.

---

## Before you start

You need:

- **A Fly account** and the [`flyctl` CLI](https://fly.io/docs/flyctl/install/),
  signed in (`fly auth login`).
- **A domain name** you control, and access to its DNS.
- **A Go toolchain**, and a clone of the Basil repository. `build.sh`
  cross-compiles the binary on your machine; there is no build stage in the
  image.
- **About $5/month.** A `shared-cpu-1x` Machine with 1GB of RAM, a 3GB volume,
  and a **dedicated IPv4 address**. The free shared address will not work here
  — see [Why the dedicated address](#why-the-dedicated-address).

Everything below assumes you are in `contrib/fly/` (or your own copy of it).

---

## The walkthrough

### 1. Name your app

Open `fly.toml` and change the two placeholders at the top:

```toml
app = "my-basil-site"      # was: CHANGE-ME
primary_region = "lhr"     # pick one near your visitors
```

`fly platform regions` lists them.

### 2. Create the app and give it an address

```sh
fly apps create my-basil-site
fly ips allocate-v4
```

The second command prints the address; the next step needs it. The dedicated
address costs a couple of dollars a month and cannot be skipped.

### 3. Point your domain at it

Add an `A` record for the hostname you want (say `mysite.example.com`) pointing
at the address from step 2. Then wait for it, and check from your own machine
rather than from Fly:

```sh
dig +short mysite.example.com
```

Do not move on until that prints the right address. Let's Encrypt looks your
domain up on the public internet, and a certificate request against DNS that has
not propagated simply fails.

### 4. Build the binary and ship the image

```sh
./build.sh
fly deploy
```

`build.sh` cross-compiles Basil for Linux, stamps the version and commit into
it, drops it next to the `Dockerfile`, and warns you if the result is not
statically linked. `fly deploy` builds the image around it and boots a Machine.

If you asked for an ARM Machine, build for it: `GOARCH=arm64 ./build.sh`.

### 5. Check the logs

```sh
fly logs
```

The first boot has nothing to serve, because the site root lives on the volume
and the volume was empty until a moment ago. The Machine says so and holds
itself open:

```
basil: /srv/mysite has no site.git — the site root has not been initialised.
basil:
basil: Run this once, from your laptop:
basil:
basil:   fly ssh console -C "basil --init /srv/mysite --server --host YOUR.HOSTNAME --admin YOUR-NAME"
basil:
basil: Write down the API key it prints — it is shown once. Then:
basil:
basil:   fly machine restart
```

The entrypoint does not run the init itself: the init prints an API key
exactly once, and `fly logs` is the wrong place for a credential.

### 6. Initialise the site root

Run it with your own hostname and account name:

```sh
fly ssh console -C "basil --init /srv/mysite --server --host mysite.example.com --admin sam"
```

This creates the repository, commits and deploys a starter site, installs the
receive hooks, creates your admin account, and prints:

```
  API key: bsl_live_Sl9j5NeWlZR0OV5C915W3G-gGok8Lee2Pn90T5i1BM4
  This is shown once and is not recoverable. Save it now.
```

**Copy that key somewhere safe before you close the terminal.** It is the
password for `git push` and `basil publish` alike. (If you lose it, you can make
another over `fly ssh console` with `basil apikey create`.)

### 7. Start serving

```sh
fly machine restart
```

Basil now boots properly, and requests its certificate within a few seconds
without waiting for a visitor. Check it from outside:

```sh
curl -I https://mysite.example.com
```

You want `HTTP/2 200` and no certificate warning. If something is off, ask the
box:

```sh
fly ssh console -C "basil check --site /srv/mysite"
```

which tests DNS, port 80, the certificate, the repository and the active release,
and names the fix for each.

### 8. Put your site on it

From here it is ordinary Basil deployment and has nothing to do with Fly. Either
clone the server and work in the clone:

```sh
git clone https://sam@mysite.example.com/.git mysite
cd mysite
git config core.hooksPath .githooks
```

…or connect a folder you already have:

```sh
cd mysite
git remote add origin https://sam@mysite.example.com/.git
git push -u origin main
basil publish            # the first publish replaces the starter site
```

Either way the API key from step 6 is the password. Both paths are written out
step by step in [Deployment → Two ways to
start](../../docs/basil/manual/deployment.md#two-ways-to-start).

**While you have `basil.yaml` open, add your email address.** The starter config
omits `https.email`, so Basil warns at every start and Let's Encrypt has no way
to tell you about an expiry or a revocation:

```yaml
https:
  auto: true
  email: you@example.com
```

It reaches the server the same way everything else does — commit it and
`basil publish`.

---

## After the first deploy, stop deploying

The two lifecycles are separate:

```
fly deploy                       ships Basil
git push / basil publish         ships the site
```

Once the Machine is up you never touch Fly to change a page. You push, the
receive hook validates the release, and the running server picks up the new
release within about a second.

To upgrade Basil itself:

```sh
./build.sh && fly deploy
```

The volume is untouched, so the repository, the releases, the accounts and the
deploy record all survive. Basil rewrites the receive hooks at startup if the
binary path has drifted, so an upgrade does not silently stop deploying.

**A site that works under `basil --dev` but fails in production usually means
the two Basils are different versions.** Your laptop gets a new binary every
time you rebuild; the Machine only gets one when you `fly deploy`. Ask it:

```sh
fly ssh console -C "basil --version"
```

`build.sh` stamps the version and commit into the binary. Without that, the
Machine reports `dev (unknown)` and the question has no useful answer.

---

## How this is put together

### The mental model

The thing that trips people up on Fly is that you do not *install* software onto
a Machine. A Machine is a microVM booted from an OCI image, and the image *is*
the filesystem. When you `fly ssh console` into a fresh one and find no `curl`
and no `git`, nothing is missing — you simply have not put them there yet. The
`Dockerfile` is where you say what the box contains.

That suits Basil well, because Basil is one binary.

### Why a Machine rather than a Sprite

[Sprites](https://fly.io/sprites/) are more durable than they first look: they
sleep rather than die, they checkpoint their filesystem, and they come back in
about a second with everything intact. They are a pleasant place to *develop*.

But a Basil server wants to be awake. It renews its own certificate, it watches
for a deploy to activate, and it answers on a domain you chose. Sprites are
shaped for workloads that idle cheaply and wake on demand; a Machine with a
Volume is shaped for a long-lived site.

### What is in the image

Four things:

- **The Basil binary**, built static — see the [build note](#the-build-trap).
- **`git`.** A real runtime dependency, not a convenience: Basil runs it to
  serve the Git endpoint, to materialise a release, and to read which branch
  publishes.
- **`/bin/sh`.** The receive hooks Basil installs are two-line shell scripts. So
  `scratch` and distroless images are out; Alpine is fine.
- **CA certificates**, for ACME and for anything a handler calls out to, and
  **tzdata** for date formatting.

Nothing needs `curl`. Nothing fetches anything at boot. No Go toolchain in the
image, no build stage, no C compiler.

### Why the volume is mounted at `/srv`

A `fly deploy` replaces the Machine's root filesystem, so everything that must
survive one — `site.git`, the releases, the `current` symlink, the deploy
record, the auth database and the certificate cache — has to be on the Volume.

It mounts at `/srv` with the site root a directory *inside* it, rather than
straight onto `/srv/mysite`, because **a freshly formatted volume is not
empty**: ext4 puts a `lost+found` in it, and `basil --init` rightly refuses a
folder that is not empty when it is about to take ownership of the tree.
Mounting one level up leaves `lost+found` where the filesystem wants it and
hands init a clean directory, without deleting anything.

### Why the dedicated address

`fly.toml` puts port 443 on **raw TCP passthrough** (`handlers = []`): the Fly
proxy moves bytes, and Basil terminates TLS itself with its own Let's Encrypt
certificate, exactly as it would on a VPS. That is the topology `basil --init
--server` writes a config for, so nothing has to be edited to match the
platform, and it exercises the same certificate path a real Basil deployment
takes.

Fly's *shared* IPv4 only carries the `http` and `tls` handlers, so passthrough
needs `fly ips allocate-v4`.

The obvious way to dodge that cost — let the Fly proxy terminate TLS on the
shared address and have Basil serve plain HTTP behind it — **does not work
today**, and it fails late, at deploy time:

```
remote: basil.yaml: configuration errors:
remote:   - production mode requires https.auto=true or both https.cert and https.key
remote: Release rejected. The live site is unchanged (still 3f499cd1a6fb).
```

Production mode insists on HTTPS at Basil's own listener, and there is no
setting that relaxes it. `server.proxy.trusted` handles `X-Forwarded-*` headers
but buys no exemption from that rule, and `--dev` — the one mode that skips it —
is not something to point at the internet. In principle you could satisfy the
check with a self-signed certificate and have the proxy re-encrypt to the
backend, but that is more moving parts than a dedicated address is worth.

### Why port 80 is different

Port 80 is where the ACME challenge is answered, so it cannot simply be dropped
— and it is the one port that **cannot** be passthrough. With `handlers = []`
the Fly proxy accepts the TCP connection and then delivers nothing at all: a raw
request to port 80 through the edge comes back with zero bytes, while the
identical request to `127.0.0.1:80` inside the Machine gets Basil's `301`
immediately. Port 443 passthrough is fine throughout; only port 80 behaves this
way.

It needs `handlers = ["http"]`, which costs nothing: that port carries the
HTTP-01 challenge and a redirect to HTTPS, and no site traffic.

The failure mode if you get this wrong is silent — TCP connects, nothing comes
back, and certificate issuance fails with `no viable challenge type found`.

### The build trap

`CGO_ENABLED=0` on its own does **not** give you a static binary here. Basil's
image support can `dlopen` a system libwebp when it finds one, and that leaves
the binary asking for glibc's loader even with cgo off. On Alpine, whose loader
is musl's, the symptom is a thoroughly confusing:

```
sh: basil: not found
```

…for a binary that is plainly sitting right there, executable, in
`/usr/local/bin`.

`build.sh` passes `-tags nodynamic`, which drops that path and uses the embedded
WASM decoder only. That costs nothing here, since a container with no libwebp
installed would have fallen back to WASM at runtime anyway. The script checks
the result and warns if the binary comes out dynamic; if it ever does, the
escape hatch is to base the image on `debian:bookworm-slim` instead of Alpine
and accept the glibc loader.

---

## Things that will bite

- **Do not let the Machine auto-stop.** A stopped Basil renews no certificate
  and activates no deploy. `auto_stop_machines = "off"`, which is what the
  `fly.toml` here sets.
- **One Machine.** A Volume attaches to exactly one, and the deploy lock only
  means anything on a local filesystem. Do not scale this out.
- **Keep the binary at `/usr/local/bin/basil`.** The receive hooks bake in an
  absolute path. Basil self-heals a drifted path at startup, but not moving is
  better than healing.
- **Port 80 needs `handlers = ["http"]`**, not passthrough. See
  [above](#why-port-80-is-different) — the failure is silent.
- **A fresh volume is not empty.** Hence mounting at `/srv` rather than straight
  onto the site root.
- **Put `https.email` in your `basil.yaml`.** Otherwise Basil warns at every
  start and Let's Encrypt has no contact address for expiry and revocation
  notices.
- **1GB of RAM, not 512MB.** Basil itself is content with very little, but `git`
  repacking a push is the memory-hungry moment.
- **`basil check` is the first thing to run when something is odd.** It tests
  DNS, port 80, the certificate, the repository placement and the active
  release, and names the fix for each:

  ```sh
  fly ssh console -C "basil check --site /srv/mysite"
  ```

---

## Where this has been run

The recipe has been run end to end on a Machine in `lhr` with a dedicated IPv4
and a real domain pointed at it. Confirmed from outside the network: Basil
obtained its own Let's Encrypt certificate through the passthrough, served the
site over HTTP/2 with its security headers intact, redirected port 80 to HTTPS,
and answered `/.git` with a Basic-auth challenge. On the box, `basil check`
passed everything with the certificate cached.

The two awkward findings above — the `lost+found` on a fresh volume, and port 80
needing the `http` handler — came out of that run, not out of reasoning.

Note that nothing in `contrib/` is built or tested by Basil's CI.

---

## See also

- [Deployment](../../docs/basil/manual/deployment.md) — HTTPS, the two Git
  workflows, rolling back, and the operator commands
- [Git Deploy](../../docs/basil/manual/git.md) — API keys, roles, and what the
  endpoint refuses
- [Deployment guide](../../docs/guide/deployment.md) — the full pipeline and the
  post-deploy hook
- [Configuration](../../docs/guide/configuration.md) — the site-root layout, and
  one config file across many machines
