# Git over HTTPS

Basil can serve your site as a Git repository, allowing you to push changes directly to a running server using standard Git commands.

## Quick Start

### 1. Enable Git

In your `basil.yaml`:

```yaml
git:
  enabled: true
  require_auth: true    # Require API key for access
```

### 2. Create a User and API Key

```bash
# Create a user with editor or admin role
basil users add alice --role editor

# Generate an API key
basil apikey create alice
# Output: bsk_abc123... (save this!)
```

### 3. Clone Your Site

```bash
# Clone from the running server
git clone https://username:bsk_abc123...@yourserver.com/.git mysite
```

The username can be anything—only the API key matters.

### 4. Push Changes

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
- **Password**: Your API key (starts with `bsk_`)

### Role Requirements

| Operation | Required Role |
|-----------|---------------|
| Clone/Pull | Any authenticated user |
| Push | `editor` or `admin` |

Users with the `viewer` role can clone but cannot push.

### Creating API Keys

```bash
# Create an API key for a user
basil apikey create alice
# bsk_abc123...

# List API keys for a user
basil apikey list alice

# Revoke an API key
basil apikey revoke bsk_abc123
```

## Configuration Reference

```yaml
git:
  enabled: false       # Enable Git server (default: false)
  require_auth: true   # Require authentication (default: true)
```

### Options

- **`enabled`**: Set to `true` to enable the Git HTTP endpoint at `/.git/`
- **`require_auth`**: When `true`, all Git operations require a valid API key

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

1. Basil receives the push via the Git HTTP protocol
2. The `go-git-http` handler writes files to the site directory
3. Basil clears its script and response caches
4. Next request loads the updated files

This happens automatically—no webhook or restart needed.

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
# Create a deployment user
basil users add deploy --role editor
basil apikey create deploy
# bsk_deploy123...
```

### Developer Workflow

```bash
# Clone the site
git clone https://deploy:bsk_deploy123...@mysite.com/.git mysite
cd mysite

# Make changes
vim handlers/index.pars

# Deploy
git add .
git commit -m "Update homepage"
git push   # Site updates instantly!
```

## Troubleshooting

### "Authentication required"

You need to include your API key in the Git URL:

```bash
git clone https://user:bsk_yourkey@server/.git
```

Or configure Git credentials:

```bash
git config credential.helper store
# Then enter credentials when prompted
```

### "Forbidden: editor or admin role required"

Your user doesn't have permission to push. Check their role:

```bash
basil users show username
```

Upgrade to editor if needed:

```bash
basil users add username --role editor
# Or delete and recreate the user
```

### Push succeeds but site doesn't update

Check the server logs for errors. The cache clear might have failed, or there might be a parsing error in your Parsley files.

### Clone works but push fails

This usually means your API key is valid but your role is insufficient. Only `editor` and `admin` roles can push.

## Security Notes

- **Always use HTTPS in production**: API keys are sent in plain text with HTTP Basic Auth
- **Keep API keys secret**: They grant full access to your site
- **Use strong roles**: Give users the minimum role they need
- **Audit API keys**: Use `basil apikey list` to see who has access
- **Revoke unused keys**: `basil apikey revoke` when no longer needed

## Comparison with Other Deployment Methods

| Method | Pros | Cons |
|--------|------|------|
| Git push | Familiar workflow, instant updates | Requires API key setup |
| SCP/rsync | Simple, no setup | Manual, no versioning |
| CI/CD pipeline | Automated, auditable | Complex setup |

Git push is ideal for small teams who want quick deploys with version history.
