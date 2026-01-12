# Basil Configuration Reference

> **Note**: For the complete, up-to-date configuration reference with all database operations, file I/O, and server functions, see `docs/basil/reference.md` in the repository root.

This file contains common configuration patterns and examples for `basil.yaml`.

## Basic Configuration

```yaml
server:
  host: localhost
  port: 8080

# Choose ONE routing strategy

# Option 1: Filesystem routing (site mode)
site: ./site              # Files in site/ serve at their path
                          # site/about.pars → /about
                          # site/users/index.pars → /users

# Option 2: Explicit routes
routes:
  - path: /
    handler: ./handlers/index.pars
  - path: /api/*
    handler: ./handlers/api.pars
  - path: /users/:id
    handler: ./handlers/user.pars
  - path: /parts/*.part
    handler: ./parts/{}.part  # Wildcard for all parts
```

## Static Files

```yaml
public_dir: ./public      # Serves at web root
                          # public/logo.png → /logo.png
                          # public/css/style.css → /css/style.css
```

## Database

```yaml
sqlite: ./data.db         # SQLite database path (relative to config file)

# Or for production:
sqlite: /var/lib/myapp/production.db
```

## Sessions & Authentication

```yaml
session:
  secret: "your-32-char-minimum-secret-key-here-12345"  # REQUIRED for persistent sessions
  max_age: 24h              # Session expiry (default: 24h)
  cookie_name: "session"    # Cookie name (default: "session")

auth:
  enabled: true
  protected_paths:
    - /dashboard            # Require login
    - /admin
    - /api/private/*
  public_paths:             # Override protected paths
    - /api/private/health   # Allow without login
```

## Security & File Access

```yaml
security:
  allow_write:              # Whitelist directories for file writes
    - ./data
    - ./uploads
    - ./logs
  allow_read:               # Whitelist directories for file reads (optional)
    - ./data
    - ./config
  allow_execute:            # Whitelist executable paths (for @shell)
    - ./scripts
    - /usr/bin
```

## Logging

```yaml
logging:
  level: info               # debug, info, warn, error
  format: text              # text or json
  output: stdout            # stdout, stderr, or file path
```

## Development vs Production

### Development

```yaml
# basil-dev.yaml
server:
  port: 8080
site: ./site
sqlite: ./dev.db
session:
  secret: "dev-secret-not-for-production"  # Random OK for dev
logging:
  level: debug
```

Run with: `./basil --dev --config basil-dev.yaml`

### Production

```yaml
# basil.yaml
server:
  host: 0.0.0.0           # Listen on all interfaces
  port: 443
  https:
    enabled: true
    cert: /etc/ssl/certs/myapp.crt
    key: /etc/ssl/private/myapp.key
site: ./site
sqlite: /var/lib/myapp/production.db
session:
  secret: "use-strong-32-char-secret-from-env"  # MUST be persistent
  max_age: 168h           # 1 week
security:
  allow_write:
    - /var/lib/myapp/uploads
logging:
  level: warn
  format: json
  output: /var/log/myapp/server.log
```

Run with: `./basil --config basil.yaml`

## Complete Example

```yaml
server:
  host: localhost
  port: 8080

site: ./site

public_dir: ./public

sqlite: ./myapp.db

session:
  secret: "change-this-to-a-real-32-char-secret"
  max_age: 24h

auth:
  enabled: true
  protected_paths:
    - /dashboard
    - /admin
    - /profile

security:
  allow_write:
    - ./data
    - ./uploads

logging:
  level: info
  format: text
```

## See Also

- `docs/basil/reference.md` - Complete Basil API documentation including database operators, file I/O, HTTP context, auth context, and all server functions
- `docs/guide/configuration-example.yaml` - Additional configuration examples
