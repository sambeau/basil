# Git Deploy: defaults and minimal-setup review

**Date:** 2026-08-24
**Reviewer:** @claude, at @sambeau's request, before implementation starts
**Covers:** `DESIGN-git-deploy.md`, FEAT-152 … FEAT-155, PLAN-132

The brief: make sure the minimum setup is not just the default but the *recommended* way —
that a developer who does almost nothing gets something that works well and is secure.

## Result

**Ten proposed settings become three.** Seven are deleted outright, four security defaults
are hardened, and first-time setup drops from five steps to two.

| | Before | After |
| --- | --- | --- |
| Settings a user could set | 10 | 3 |
| Settings a normal user *should* set | 0 | 0 |
| Steps from nothing to deployable | 5 | 2 |
| Ways to accidentally disable a safety control | 3 | 0 |

The test applied throughout: **if a setting exists, who sets it, and what happens when they
set it wrong?** A setting whose only purpose is to weaken a guarantee is a setting that
will one day be found in a production config file.

---

## Part 1 — Settings deleted

### 1. `git.require_auth` — delete

An option whose only function is to disable authentication on a public write endpoint.
Today it merely produces a log warning when enabled without TLS.

Authentication is always required. The legitimate case it was serving —
local development — is already handled in code by `isDevLocalhost`
(`server/git.go:161`), which is narrower and cannot be misconfigured from a file.

Anyone on a trusted LAN uses `--dev`. That is what it is for.

### 2. `git.repo` — delete

Always `<site root>/site.git`. Configurable adds a knob nobody needs and one genuine
hazard: a repository path pointed inside `public_dir` would serve the object store —
every version of every file, including anything ever committed by mistake.

Derived, not configured. See also hardening #2.

### 3. `git.fmt_check` — delete, and replace the mechanism

The original design gated formatting server-side, off by default. That failed its own test:
if a team *should* turn it on, then the default is not the recommended way.

Replace with something better on both counts:

- **`basil --init` installs a pre-commit formatting hook** in the repository it just
  created. Formatting is fixed before a commit exists, which is where it belongs.
- **The server warns but never rejects.** Cosmetics must never block a release.

No setting, no decision, and the minimal path produces formatted code automatically.
Installing a hook into a repository Basil itself created is scaffolding, not presumption —
this would be rude only if it reached into an existing repository.

### 4. `deploy.validate` — delete the setting, keep the check

Validation is always on for pushes.

The reason a setting looked necessary is real: if validation ever gets a false positive,
you cannot deploy, and fixing it requires deploying. But a **config-file** escape hatch is
the wrong shape for that — it lives inside the release being validated, and it persists
silently after the emergency has passed.

The override is `basil deploy --no-validate`, on the server. It requires shell access,
which is the correct bar for overriding a safety check, and it cannot be left switched on
by accident.

### 5. `deploy.hook` — delete, use a convention

If `deploy.pars` exists in the release root, run it after activation.

Basil already resolves handlers by convention (`index.pars`, `{folder}.pars`, FEAT-040), so
this matches the house style, and a convention cannot be pointed at the wrong file.

### 6. `deploy.publish_role` — delete

Speculative. Nobody asked to separate "may push" from "may publish", the role model is
coarse (`viewer` / `editor` / `admin`), and it can be added without breaking anything if a
real request appears.

To `work/BACKLOG.md`, with the trigger: *a team asks for a release gate distinct from push
access*.

### 7. `git.enabled` — demote to an off-switch

Git deploy is **on when `<site root>/site.git` exists**, which `basil --init` creates. It is
how a Basil site is deployed; making it opt-in is like making file serving opt-in.

`git.enabled: false` remains, for someone who wants the site root layout without the
endpoint. Nobody needs to write `git.enabled: true` ever again.

Safety check: an enabled endpoint with no accounts refuses everything with 401, and Basil
already declines to start Git without an auth database (`server/server.go:571`). Keep that,
and make it explicit rather than incidental.

---

## Part 2 — Settings kept

| Setting | Default | Why it survives |
| --- | --- | --- |
| `deploy.branch` | `live` | The one genuine choice in the system: whether publishing is deliberate or automatic. Setting it to `main` restores push-to-publish |
| `deploy.keep` | `5` | Disk is finite and full disks bite. Sane default, rarely touched, harmless when it is |
| `data_dir` | `<site root>/data` | Real need: state on a different volume. Derived by default |

No normal user needs to set any of the three.

---

## Part 3 — Security defaults hardened

### 1. Refuse Git over plain HTTP — do not merely warn

**The most important item in this review.**

Git authenticates with HTTP Basic, so the API key crosses the wire in an easily-decoded
header. Over plain HTTP that is a plaintext credential with push rights. The current
implementation logs a warning (`server/git.go:180`, `warnInsecureHTTP`) and proceeds.

A warning in a log nobody reads is not a control. **Refuse the request**, with an error that
says why, except on a dev-mode localhost bind.

Nothing legitimate is lost: Basil obtains its own certificate via Let's Encrypt and already
listens on 443.

### 2. Refuse to start if the repository sits inside a served root

Roughly ten lines, and it forecloses the worst available misconfiguration — serving the
object store as static files. Cheap insurance even though the path is no longer
configurable (#2 above), because `public_dir` and `site.path` still are.

### 3. Refuse force-push and deletion of the release branch

No setting. A release history that can be rewritten makes the deploy record unreliable, and
the record is what rollback depends on.

### 4. Refuse Git when no auth database exists

Already the behaviour; promote it from an incidental check to a stated guarantee with a
test.

---

## Part 4 — The minimal path

Five steps become two.

**Before**

```bash
basil --init mysite
# edit basil.yaml to enable git
basil users add sam --role admin
basil apikey create sam
basil
```

**After**

```bash
basil --init mysite
#   ✓ site root, bare repository, data directory
#   ✓ admin user 'sam'
#   ✓ API key bsk_… (shown once — save it now)
#   ✓ pre-commit formatting hook
basil
```

Then, from any machine:

```bash
git clone https://sam@mysite.example.com/.git
# edit
git push
basil publish
```

`basil --init` creating the first account and key is the single biggest reduction, and it
is no less safe than `apikey create`: generated randomly, hashed at rest, displayed once.
Only on a genuinely fresh init — never overwriting existing credentials.

---

## Part 5 — Observation, not a change

`!secret auto` generates a **fresh random value on every start**
(`server/config/secret.go:133`), so `session.secret: auto` silently invalidates every
session on restart. That is pre-existing and out of scope here.

It becomes relevant to FEAT-152 only in this sense: if `auto` is ever changed to persist,
the generated secret belongs in `DataDir`, not in the release. Worth a line in the spec so
the choice is not made accidentally later.

---

## Applied

`DESIGN-git-deploy.md` §7 and the affected acceptance criteria in FEAT-152 … FEAT-155 have
been updated to match. Deletions are recorded here rather than only in a diff, so the
reasoning survives the next time someone proposes adding one of them back.
