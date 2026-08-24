# Git over HTTPS

Basil can serve your site as a Git repository, allowing you to push changes directly to a running server using standard Git commands.

> **This mechanism is being reworked.** Today Basil serves your live site directory
> directly, which means a push has to contend with a checked-out working tree. That needs
> some manual setup, and it has a failure mode that bites long-running sites — both are
> covered below. The replacement, which pushes to a separate bare repository and deploys
> from it, is described in
> [`work/design/DESIGN-git-deploy.md`](../../work/design/DESIGN-git-deploy.md) and tracked
> as FEAT-152/FEAT-154. Expect the setup steps below to get shorter, not longer.

## Quick Start

### 1. Prepare the repository

Basil serves your project directory as a Git repository, but it doesn't create or
configure that repository for you. `basil --init` writes a `.gitignore`, but it does not
run `git init` or configure anything Git-related, so on a fresh project you have to do two
things by hand on the server, in the project directory:

```bash
cd /path/to/mysite

# Create the repository, if you haven't already
git init -b main .
git add -A
git commit -m "Initial commit"

# Allow pushes into the branch that is checked out
git config receive.denyCurrentBranch updateInstead
```

That second line is the important one. The directory Basil serves is a normal non-bare
repository with a working tree — the live site — and Git refuses by default to accept a
push into the branch that tree has checked out. `updateInstead` tells Git to accept the
push and update the working tree to match, which is exactly what you want here.

Without it your first push is rejected. See
[Troubleshooting](#troubleshooting) for what that looks like.

### 2. Enable Git

In your `basil.yaml`:

```yaml
git:
  enabled: true
  require_auth: true    # Require API key for access

auth:
  enabled: true         # Required when require_auth is true
```

`require_auth: true` needs `auth.enabled: true` as well. Without it the server does not
start at all — it exits with `git server requires auth but auth is not enabled`.

### 3. Create a User and API Key

```bash
# Create a user. This prints the user ID you need in the next step.
basil users create --name Alice --email alice@example.com --role editor
# ✓ Created user usr_7d30113fc69ba1d8...

# Generate an API key for that user
basil apikey create --user usr_7d30113fc69ba1d8... --name "MacBook Git"
# ✓ Created API key: bsl_live_AcTVMzefTllD875... (save this now — it won't be shown again)
#   Key ID: key_42c52e0dc55a5a64...
```

Both commands take the user *ID*, not the name. Note that the first user you create is
always an admin regardless of `--role`; the CLI says so when it ignores the flag.

### 4. Clone Your Site

```bash
# Clone from the running server
git clone https://username:bsl_live_AcTVMzefTllD875...@yourserver.com/.git mysite
```

The username can be anything—only the API key matters.

### 5. Push Changes

```bash
cd mysite
# Make changes...
git add .
git commit -m "Update homepage"
git push
```

Basil automatically reloads when you push—no server restart needed.

## Authentication

Git authentication uses **API keys** via HTTP Basic Auth:

- **Username**: Can be anything (ignored)
- **Password**: Your API key (starts with `bsl_live_`)

### Role Requirements

| Operation | Required Role |
|-----------|---------------|
| Clone/Pull | Any authenticated user |
| Push | `editor` or `admin` |

Users with the `viewer` role can clone but cannot push.

### Creating API Keys

```bash
# Create an API key for a user (--user and --name are both required)
basil apikey create --user usr_7d30113fc69ba1d8... --name "MacBook Git"
# ✓ Created API key: bsl_live_AcTVMzefTllD875...
#   Key ID: key_42c52e0dc55a5a64...

# List API keys for a user
basil apikey list --user usr_7d30113fc69ba1d8...

# Revoke an API key — this takes the key ID, not the key itself
basil apikey revoke key_42c52e0dc55a5a64...
```

## Configuration Reference

```yaml
git:
  enabled: false       # Enable Git server (default: false)
  require_auth: true   # Require authentication (default: true)

auth:
  enabled: true        # Must be true when git.require_auth is true
```

### Options

- **`enabled`**: Set to `true` to enable the Git HTTP endpoint at `/.git/`
- **`require_auth`**: When `true`, all Git operations require a valid API key. This
  requires `auth.enabled: true`; if auth is off the server exits at startup with
  `git server requires auth but auth is not enabled`

### Security Warning

If you set `require_auth: false`, Basil will log a warning:

```
⚠ Git enabled without authentication - anyone can push
```

This is dangerous in production—anyone could push malicious code to your server.

## Dev Mode

In dev mode (`--dev`), requests from localhost bypass authentication:

```bash
# Dev mode - no API key needed from localhost
basil --dev
git clone http://localhost:8080/.git mysite
```

This makes local development easier while keeping production secure.

## How Push Reload Works

When you `git push`:

1. Basil receives the push at `/.git/` and checks your API key and role
2. It hands the request to the `go-git-http` handler, which runs `git receive-pack` in
   your project directory
3. `receive-pack` stores the pushed objects and updates the refs — it does **not** write
   your files into the site directory by itself
4. Because you set `receive.denyCurrentBranch=updateInstead`, Git then updates the
   checked-out working tree to match the ref it just moved. This is the step that makes
   the new files appear on disk, and it is Git doing it, not Basil
5. Basil clears its script, response and fragment caches
6. The next request loads the updated files

So the file update depends entirely on receive behaviour being configured. If it isn't,
the push is refused (step 4 never happens) and nothing changes on disk. Basil's cache
clear assumes the files are already there; it does not fetch or check out anything itself.

This happens automatically—no webhook or restart needed.

### The drifted working tree

`updateInstead` is deliberately cautious: it refuses to update the working tree if that
tree has **uncommitted changes to tracked files**, because updating would throw those
changes away. On a long-running site this is the failure you are most likely to hit — a
deploy that worked yesterday starts getting rejected, with no obvious cause.

Anything that writes inside a tracked path can cause it: a generated file, a cache
directory that isn't ignored, an edit made on the server, a stray `.DS_Store` that got
committed once and now changes.

Untracked and ignored files are fine. `logs/`, `db/`, `certs/` and `cache/` are in the
`.gitignore` that `basil --init` writes, so the log files, the database and the
transformed-image cache do not trigger this. If you keep state somewhere else inside the
project, add it to `.gitignore` too.

To fix a drifted tree, go to the server and either discard or commit the local changes:

```bash
cd /path/to/mysite
git status                # see what has drifted
git checkout -- .         # discard the local changes...
# ...or keep them:
git stash
```

Then push again.

## Example Workflow

### Initial Setup (Server)

```yaml
# basil.yaml
server:
  host: 0.0.0.0
  port: 443
  tls_cert: /etc/ssl/cert.pem
  tls_key: /etc/ssl/key.pem

git:
  enabled: true
  require_auth: true

auth:
  enabled: true
```

```bash
# Prepare the repository (once)
cd /path/to/mysite
git init -b main .
git add -A && git commit -m "Initial commit"
git config receive.denyCurrentBranch updateInstead

# Create a deployment user and note the usr_ ID it prints
basil users create --name Deploy --email deploy@example.com --role editor
# ✓ Created user usr_deploy123...

basil apikey create --user usr_deploy123... --name "Deploy key"
# ✓ Created API key: bsl_live_deploy123...
```

### Developer Workflow

```bash
# Clone the site
git clone https://deploy:bsl_live_deploy123...@mysite.com/.git mysite
cd mysite

# Make changes
vim handlers/index.pars

# Deploy
git add .
git commit -m "Update homepage"
git push   # Site updates as soon as the push lands
```

## Troubleshooting

### "branch is currently checked out"

```
remote: error: refusing to update checked out branch: refs/heads/main
remote: error: By default, updating the current branch in a non-bare repository
remote: is denied, because it will make the index and work tree inconsistent
remote: with what you pushed, and will require 'git reset --hard' to match
remote: the work tree to HEAD.
 ! [remote rejected] main -> main (branch is currently checked out)
error: failed to push some refs to 'https://mysite.com/.git'
```

Receive behaviour hasn't been configured. On the server, in the project directory:

```bash
git config receive.denyCurrentBranch updateInstead
```

Then push again. Git's own message suggests `ignore` or `warn` as well — don't use those
here. They accept the push but leave the working tree untouched, so your site would keep
serving the old files while the repository claims to be up to date.

### "Working directory has unstaged changes"

```
 ! [remote rejected] main -> main (Working directory has unstaged changes)
error: failed to push some refs to 'https://mysite.com/.git'
```

The live working tree has uncommitted changes to tracked files, so Git won't overwrite it.
See [The drifted working tree](#the-drifted-working-tree) above for why this happens and
how to clear it.

### "Authentication required"

You need to include your API key in the Git URL:

```bash
git clone https://user:bsl_live_yourkey@server/.git
```

Or configure Git credentials:

```bash
git config credential.helper store
# Then enter credentials when prompted
```

### "Forbidden: editor or admin role required"

Your user doesn't have permission to push. Check their role:

```bash
basil users list                    # find the usr_ ID
basil users show usr_abc123...      # show takes the ID, not the name
```

Upgrade to editor if needed:

```bash
basil users set-role usr_abc123... editor
```

### Push succeeds but site doesn't update

If `receive.denyCurrentBranch` is set to `ignore` or `warn` rather than `updateInstead`,
Git accepts the push and updates the refs but never touches the working tree, so the files
Basil serves are unchanged. Check it on the server:

```bash
git config receive.denyCurrentBranch
```

Otherwise, check the server logs. The cache clear might have failed, or there might be a
parsing error in your Parsley files.

### Clone works but push fails

Check the rejection message. `(branch is currently checked out)` and
`(Working directory has unstaged changes)` are the two repository-side failures above —
they have nothing to do with your key. A `403` with `editor or admin role required` means
your API key is valid but your role is insufficient; only `editor` and `admin` can push.

## Security Notes

- **Always use HTTPS in production**: API keys are sent in plain text with HTTP Basic Auth
- **Keep API keys secret**: They grant full access to your site
- **Use strong roles**: Give users the minimum role they need
- **Audit API keys**: Use `basil apikey list --user <usr_id>` to see who has access
- **Revoke unused keys**: `basil apikey revoke <key_id>` when no longer needed

## Comparison with Other Deployment Methods

| Method | Pros | Cons |
|--------|------|------|
| Git push | Familiar workflow, instant updates | Requires API key and repository setup |
| SCP/rsync | Simple, no setup | Manual, no versioning |
| CI/CD pipeline | Automated, auditable | Complex setup |

Git push is ideal for small teams who want quick deploys with version history.
