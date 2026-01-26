# Design: Site Metadata, Environment Variables, and Secrets (V2)

**Status:** Draft  
**Author:** AI Assistant  
**Date:** 2026-01-25  
**Supersedes:** [metadata-and-secrets.md](metadata-and-secrets.md) (V1)  
**Spec:** [FEAT-102](../specs/FEAT-102.md)

## Summary

This design adds three capabilities to Basil configuration:

1. **`meta` section** — Custom metadata accessible in Parsley as `meta.*`
2. **`${VAR}` interpolation** — Environment variables usable anywhere in config
3. **`!secret` tag** — Mark values as sensitive (hidden in DevTools)

## Existing Config Fields (Reference)

These top-level YAML keys are already in use and must be avoided:

| Field | Purpose |
|-------|---------|
| `server` | Host, port, HTTPS, proxy settings |
| `security` | HSTS, CSP, frame options, etc. |
| `cors` | Cross-origin resource sharing |
| `compression` | Gzip/zstd settings |
| `auth` | Authentication settings |
| `session` | Session storage settings |
| `git` | Git HTTP server settings |
| `dev` | Development mode settings |
| `sqlite` | Database path |
| `public_dir` | Static files directory |
| `site` | Filesystem-based routing directory |
| `site_cache` | Response cache TTL for site mode |
| `static` | Static route mappings |
| `routes` | Handler route mappings |
| `logging` | Log level, format, output |
| `developers` | Per-developer overrides |

**New fields:** `meta` and `env` (if needed for explicit mappings)

## Syntax

### Complete Grammar

The default value syntax follows **POSIX shell parameter expansion** convention:

- `${VAR:-default}` — Use default if VAR is unset **or empty** (colon-hyphen)
- This matches Bash, Docker Compose, Kubernetes, and most CI systems
- The hyphen distinguishes it from other shell expansions like `${VAR:=default}` (assign)

```yaml
# Literal values
key: value                           # literal string/number/bool
key: !secret value                   # literal, marked as secret

# Environment variable interpolation (POSIX shell syntax)
key: ${VAR}                          # env var (null if unset)
key: ${VAR:-default}                 # env var with default (POSIX syntax)
key: !secret ${VAR}                  # env var, marked as secret
key: !secret ${VAR:-default}         # env var with default, marked as secret

# String concatenation (env vars within strings)
key: "https://${HOST}:${PORT}/api"   # multiple interpolations
```

### Concepts

| Feature | Syntax | Purpose |
|---------|--------|---------|
| Env interpolation | `${VAR}` | Insert environment variable value |
| Default value | `${VAR:-default}` | Fallback if env var unset (POSIX) |
| Secret marker | `!secret` | Hide value in DevTools |

The two features are orthogonal and compose naturally:
- `${VAR}` controls **where** the value comes from
- `!secret` controls **how** the value is treated

## Configuration Sections

### 1. Custom Metadata

Free-form section for non-sensitive, site-specific configuration:

```yaml
meta:
  name: "My Awesome Blog"
  tagline: "Thoughts on code and coffee"
  contact_email: "hello@example.com"
  
  social:
    twitter: "@myblog"
    github: "myblog"
  
  features:
    comments: true
    dark_mode: false
  
  limits:
    posts_per_page: 10
```

**Parsley access:** `meta.name`, `meta.features.dark_mode`, etc.

**Characteristics:**
- Read-only in Parsley
- Arbitrary nesting allowed
- All YAML types supported
- Safe to commit to version control
- Supports `${VAR}` interpolation

### 2. Environment Variables (anywhere)

Environment variables can be interpolated anywhere in the config:

```yaml
port: ${PORT:-8080}
dev: ${DEV:-false}

database:
  url: !secret ${DATABASE_URL}
  pool_size: ${DB_POOL:-10}

meta:
  api_url: ${PUBLIC_API_URL:-https://api.example.com}
```

**Type coercion:** Inferred from YAML context. `port: ${PORT:-8080}` parses as integer because `8080` is an integer in YAML.

### 3. Secrets

The `!secret` YAML tag marks a value as sensitive:

```yaml
auth:
  session_secret: !secret auto                    # auto-generate
  google_client_id: ${GOOGLE_CLIENT_ID}           # not secret (shown in DevTools)
  google_client_secret: !secret ${GOOGLE_SECRET}  # secret (hidden in DevTools)

stripe:
  publishable_key: ${STRIPE_PK}                   # public, not secret
  secret_key: !secret ${STRIPE_SK}                # secret
  
internal:
  api_key: !secret sk_live_abc123xyz              # literal secret
```

**Behavior:**
- DevTools `/__/env` page shows `●●●●●●●●` instead of value
- Value is still usable in Parsley code
- No actual encryption (see Security section)

### 4. Session Secret Auto-Generation

Special value `!secret auto` triggers automatic secure random generation:

```yaml
auth:
  session_secret: !secret auto
```

**Behavior:**
- Generate 32-byte cryptographically secure random value at startup
- Regenerated each restart (acceptable for sessions)
- Override with `SESSION_SECRET` env var for multi-instance deployments:
  ```yaml
  auth:
    session_secret: !secret ${SESSION_SECRET:-auto}
  ```

The existing `session_secret` config field is removed.

## Full Example

```yaml
# basil.yaml

port: ${PORT:-8080}
dev: ${DEV:-false}
data_dir: ${DATA_DIR:-./data}

database:
  url: !secret ${DATABASE_URL}
  pool_size: ${DB_POOL:-10}

auth:
  enabled: true
  session_secret: !secret ${SESSION_SECRET:-auto}
  session_duration: 24h
  
  google:
    enabled: ${GOOGLE_AUTH_ENABLED:-false}
    client_id: ${GOOGLE_CLIENT_ID}
    client_secret: !secret ${GOOGLE_CLIENT_SECRET}

stripe:
  publishable_key: ${STRIPE_PK}
  secret_key: !secret ${STRIPE_SK}
  webhook_secret: !secret ${STRIPE_WEBHOOK_SECRET}

meta:
  name: "My Blog"
  tagline: "Thoughts on code and coffee"
  contact_email: "hello@example.com"
  base_url: ${SITE_URL:-http://localhost:8080}
  
  social:
    twitter: "@myblog"
    github: "myblog"
  
  features:
    comments: true
    newsletter: ${NEWSLETTER_ENABLED:-false}
    dark_mode: true
  
  limits:
    posts_per_page: 10
    max_upload_mb: ${MAX_UPLOAD:-5}
```

## DevTools Display

The `/__/env` page shows configuration with secrets hidden:

```
Custom Metadata
─────────────────────────────────────────────────────────────
Key                     Value                    Source
meta.name               "My Blog"                literal
meta.base_url           "https://example.com"    SITE_URL
meta.features.comments  true                     literal
meta.features.newsletter false                   NEWSLETTER_ENABLED (default)

Configuration
─────────────────────────────────────────────────────────────
Key                     Value                    Source
port                    8080                     PORT (default)
database.url            ●●●●●●●●                 DATABASE_URL
database.pool_size      10                       DB_POOL (default)
auth.session_secret     (auto-generated)         auto
auth.google.client_id   "123456..."              GOOGLE_CLIENT_ID
auth.google.client_secret [hidden]               GOOGLE_CLIENT_SECRET
stripe.publishable_key  "pk_live_..."            STRIPE_PK
stripe.secret_key       [hidden]                 STRIPE_SK
```

## Parsley Access

### Custom Metadata

```parsley
<title>{meta.name} - {page.title}</title>

@if meta.features.dark_mode {
  <link rel="stylesheet" href="/css/dark.css"/>
}

<p>Contact: {meta.contact_email}</p>
```

### Config Values

Config values (outside of `meta`) are accessible through the existing config prelude or a new `config.*` namespace (implementation detail).

## Implementation

### Processing Order

1. Read `basil.yaml` as raw text
2. Pre-process `${VAR}` and `${VAR:-default}` interpolations
3. Parse resulting YAML with custom `!secret` tag handler
4. Validate structure against Config schema
5. Track which values are marked secret (for DevTools)

### Custom YAML Tag

```go
type SecretValue struct {
    Value    interface{}
    IsSecret bool
}

// Register !secret tag with YAML parser
yaml.RegisterTagHandler("!secret", func(node *yaml.Node) (interface{}, error) {
    var value interface{}
    if err := node.Decode(&value); err != nil {
        return nil, err
    }
    return SecretValue{Value: value, IsSecret: true}, nil
})
```

### Env Var Interpolation

```go
func interpolateEnvVars(content string) string {
    // Match ${VAR} or ${VAR:-default}
    re := regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)(?::([^}]*))?\}`)
    return re.ReplaceAllStringFunc(content, func(match string) string {
        parts := re.FindStringSubmatch(match)
        varName := parts[1]
        defaultVal := parts[2]
        
        if val, ok := os.LookupEnv(varName); ok {
            return val
        }
        return defaultVal // empty string if no default
    })
}
```

### File Changes

| File | Change |
|------|--------|
| `server/config/config.go` | Add `Meta` field, remove `SessionSecret` |
| `server/config/interpolate.go` | New: env var interpolation |
| `server/config/secret.go` | New: `!secret` tag, `SecretValue` type |
| `server/config/loader.go` | Integrate interpolation + secret tracking |
| `server/prelude.go` | Inject `meta.*` into Parsley |
| `server/devtools.go` | Update `/__/env` to hide secrets |
| `server/session.go` | Handle `auto` for session secret |

## Security Considerations

### What `!secret` Does

- Hides value in DevTools `/__/env` page
- Signals to developers "this is sensitive"
- Allows tooling to identify secrets (linting, auditing)

### What `!secret` Does NOT Do

- **Not encryption** — Value exists in plaintext in memory
- **Not access control** — Parsley code can still read and log the value
- **Not runtime protection** — Compromised server can read secrets

### Threat Model

| Threat | Protected? |
|--------|------------|
| Secrets in git | ✅ Use `${VAR}`, only env var names committed |
| Secrets in DevTools | ✅ `!secret` values show `[hidden]` |
| Secrets in backups | ✅ Only env var names in config file |
| Accidental logging | ⚠️ Developer could still `@log` a secret |
| Server compromise | ❌ Attacker can read memory/env vars |

### Best Practices

1. **Never commit secrets** — Always use `${VAR}` for real secrets
2. **Use `.env` for local dev** — Add to `.gitignore`
3. **Mark with `!secret`** — Even for env vars, for documentation
4. **Production** — Set env vars via platform (Heroku, Docker, systemd, etc.)

## Testing

### Unit Tests

- `TestEnvInterpolation` — `${VAR}` replacement
- `TestEnvInterpolationDefault` — `${VAR:-default}` behavior
- `TestEnvInterpolationMissing` — Unset var without default
- `TestSecretTag` — `!secret` parsing
- `TestSecretTagWithEnv` — `!secret ${VAR}` combination
- `TestSessionSecretAuto` — Auto-generation

### Integration Tests

- `TestMetaInParsley` — Access `meta.*` in templates
- `TestDevToolsSecretsHidden` — Secrets show `[hidden]`
- `TestDevToolsNonSecretsShown` — Non-secrets show values

## Migration

- **Breaking change:** `session_secret` config field removed
- **Action required:** Use `!secret auto` or `!secret ${SESSION_SECRET}`
- **Acceptable:** Pre-alpha, breaking changes expected

## Future Considerations

### Parsley Interpolation (deferred)

Could use Parsley's existing interpolator instead of simple `${VAR}`:

```yaml
uploads_dir: @{path.join(data_dir, "uploads")}
```

Deferred: Simple `${VAR}` covers 95% of use cases. Revisit if needed.

### Named Env Mappings (deferred)

Optional `env:` section for explicit env var documentation:

```yaml
env:
  stripe_key: STRIPE_SECRET_KEY
  db_url: DATABASE_URL:string
```

Deferred: Direct `${VAR}` interpolation is simpler.

### Secret Warnings (deferred)

Warn if `meta` section contains secret-like keys without `!secret`:

```
⚠️ meta.api_key looks like a secret. Consider using !secret.
```

Deferred: Can add later without breaking changes.
