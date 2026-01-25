# Design: Site Metadata and Environment Variables (V1)

**Status:** Superseded by [metadata-and-secrets-v2.md](metadata-and-secrets-v2.md)  
**Author:** AI Assistant  
**Date:** 2026-01-25  
**Related:** (no FEAT yet)

---

> **Note:** This document contains exploratory design discussion. See V2 for the final design.

---

## Overview

This document proposes adding two new configuration sections to `basil.yaml`:

1. **`site`** — Custom metadata accessible in Parsley as a read-only object
2. **`env`** — Environment variable mappings with type coercion and defaults

Together, these provide a clean pattern for configuration that varies between environments while guiding developers toward secure practices for secrets.

## Motivation

### Current State

Developers have no standardized way to:
- Store site-specific metadata (site name, contact email, feature flags)
- Access environment variables from Parsley scripts
- Distinguish between "safe to commit" config and "keep secret" config

### Problems This Creates

1. **Ad-hoc solutions** — Developers invent their own patterns, often insecure
2. **Secrets in config** — Without guidance, secrets end up in `basil.yaml`
3. **No environment variation** — Hard to have dev/staging/prod differences
4. **Boilerplate** — Common patterns (site name, analytics ID) require custom code

## Proposed Design

### 1. Site Metadata (`site` section)

A free-form section for non-sensitive, site-specific configuration:

```yaml
# basil.yaml
site:
  name: "My Awesome Blog"
  tagline: "Thoughts on code and coffee"
  contact_email: "hello@example.com"
  analytics_id: "UA-12345678-1"
  
  social:
    twitter: "@myblog"
    github: "myblog"
  
  features:
    comments: true
    newsletter: true
    dark_mode: false
  
  limits:
    posts_per_page: 10
    max_upload_mb: 5
```

**Parsley Access:**

```parsley
<title>site.name - page.title</title>

if site.features.dark_mode {
  <link rel="stylesheet" href="/css/dark.css"/>
}

<footer>
  Contact us: <a href='mailto:@{site.contact_email}'>site.contact_email</a>
</footer>
```

**Characteristics:**
- Read-only in Parsley (never written back to YAML)
- Arbitrary nesting allowed
- All YAML types supported (strings, numbers, booleans, arrays, maps)
- Safe to commit to version control
- Supports env var interpolation (see section 2)
- No automatic secret detection (see "Warnings" section below)

### 2. Environment Variable Interpolation (whole-file)

Environment variables can be interpolated **anywhere** in the config using `${VAR}` syntax:

```yaml
# basil.yaml
port: ${PORT:-8080}                    # with default

database:
  url: ${$DATABASE_URL}                # $ prefix = secret (hidden in devtools)
  max_connections: ${DB_POOL:-10}

auth:
  google_client_id: ${GOOGLE_CLIENT_ID}
  google_client_secret: ${$GOOGLE_CLIENT_SECRET}  # secret

site:
  name: "My Awesome Blog"
  api_url: ${PUBLIC_API_URL:-https://api.example.com}
  contact_email: "hello@example.com"
```

**Syntax:**
- `${VAR}` — substitute env var value (null if unset)
- `${VAR:-default}` — substitute with default if unset
- `${$VAR}` — substitute, and mark as SECRET (hidden in DevTools)
- `${$VAR:-default}` — secret with default

**Type coercion:**
Types are inferred from YAML context. If `port: ${PORT:-8080}`, the `8080` tells YAML it's an integer, so `PORT=9000` becomes integer `9000`.

**Parsley access:**
Config values are available through their normal paths:
- `config.port`
- `config.database.url`
- `site.api_url`

### 3. Named Environment Mappings (`env` section) — Optional

For convenient Parsley access and explicit documentation of required env vars:

```yaml
# basil.yaml
env:
  # Maps env var to friendly Parsley name
  # $ prefix marks as secret
  stripe_key: $STRIPE_SECRET_KEY
  database_url: $DATABASE_URL
  site_url: SITE_URL                    # not secret
  
  # With type coercion (for non-string values)
  max_connections: MAX_CONN:int
  debug_mode: DEBUG:bool
  
  # With defaults
  log_level: LOG_LEVEL:string:info
  port: PORT:int:8080
  verbose: VERBOSE:bool:false
  
  # Array from comma-separated env var
  allowed_hosts: ALLOWED_HOSTS:strings
  admin_ids: ADMIN_IDS:ints
```

**Parsley Access:**

```parsley
@if env.debug_mode {
  <div class="debug-panel">...</div>
}

// Use in database connection
@let db = sql.open(env.database_url)
```

**Type Coercion Rules:**

| Type | Env Var Value | Parsley Value |
|------|---------------|---------------|
| `string` (default) | `"hello"` | `"hello"` |
| `int` | `"42"` | `42` |
| `float` | `"3.14"` | `3.14` |
| `bool` | `"true"`, `"1"`, `"yes"` | `true` |
| `bool` | `"false"`, `"0"`, `"no"`, `""` | `false` |
| `strings` | `"a,b,c"` | `["a", "b", "c"]` |
| `ints` | `"1,2,3"` | `[1, 2, 3]` |

**Error Handling:**

- Missing env var without default → Parsley value is `null`
- Type coercion failure → Warning at startup, value is `null`
- Empty string with default → Default is used

### 3. Developer Warnings

To guide developers away from putting secrets in the `site` section, emit warnings at startup:

```
⚠️  site.stripe_key looks like a secret (contains "key").
    Consider using env: mapping instead.
    See: https://basil.dev/guide/secrets

⚠️  site.db_password looks like a secret (contains "password").
    Consider using env: mapping instead.
```

**Trigger patterns** (case-insensitive):
- `secret`, `password`, `passwd`, `pwd`
- `key` (but not `keyboard`, `monkey`)
- `token`, `api_key`, `apikey`
- `credential`, `auth`

**Behavior:**
- Warning only, not an error
- Shown once at startup
- Can be suppressed per-key: `site.stripe_key: !safe "pk_test_..."`

### 4. Startup Validation

At server startup:

1. Parse `site` section → Store as nested map
2. Parse `env` section → For each mapping:
   - Look up environment variable
   - Apply type coercion if specified
   - Apply default if unset and default provided
   - Store result
3. Scan `site` keys for secret-like names → Emit warnings
4. Report missing required env vars (those without defaults that are `null`)

**Startup output example:**

```
Basil v0.9.0
├─ Site: "My Awesome Blog"
├─ Env mappings: 5 configured, 5 resolved
│  └─ ⚠️  STRIPE_SECRET_KEY not set (env.stripe_key will be null)
└─ Listening on :8080
```

## Implementation

### Config Structure Changes

```go
// config/config.go

type Config struct {
    // ... existing fields ...
    
    // Site metadata (arbitrary user data)
    Site map[string]interface{} `yaml:"site"`
    
    // Environment variable mappings
    Env map[string]string `yaml:"env"`
}

// Resolved environment values (after parsing env section)
type ResolvedEnv struct {
    values map[string]interface{}
}

func (c *Config) ResolveEnv() (*ResolvedEnv, []Warning) {
    // Parse env mappings, look up env vars, apply types/defaults
}
```

### Parsley Integration

```go
// evaluator/prelude.go or similar

func (e *Evaluator) setupBasilPrelude(config *Config, resolvedEnv *ResolvedEnv) {
    // site.* - read-only object from config.Site
    e.Set("site", object.FromGo(config.Site))
    
    // env.* - resolved environment values  
    e.Set("env", object.FromGo(resolvedEnv.values))
}
```

### File Changes

| File | Change |
|------|--------|
| `server/config/config.go` | Add `Site` and `Env` fields |
| `server/config/env.go` | New file: env parsing logic |
| `server/config/env_test.go` | Tests for env parsing |
| `server/config/warnings.go` | Secret detection warnings |
| `server/prelude.go` | Inject `site` and `env` into Parsley |
| `server/server.go` | Call env resolution at startup |
| `docs/guide/configuration.md` | Document new sections |
| `docs/guide/secrets.md` | New guide on secrets best practices |

### Env Parsing Grammar

```
env_mapping = env_var_name [ ":" type [ ":" default ] ]
env_var_name = [A-Z][A-Z0-9_]*
type = "string" | "int" | "float" | "bool" | "strings" | "ints"
default = <any value appropriate for type>
```

Examples:
- `DATABASE_URL` → string, no default
- `PORT:int` → int, no default
- `DEBUG:bool:false` → bool, default false
- `HOSTS:strings` → string array from CSV, no default

## Alternatives Considered

### Alternative A: Encrypted Secrets File

```yaml
secrets_file: secrets.yaml.enc
```

With CLI: `basil secrets edit`, `basil secrets set KEY VALUE`

**Rejected because:**
- Adds significant complexity (encryption, key management, CLI tooling)
- False sense of security (if server is compromised, secrets are exposed)
- Doesn't follow industry standard practices
- Key management becomes the user's problem

**May revisit** if there's clear demand, but env vars are the better default.

### Alternative B: Secrets Section in Config

```yaml
secrets:
  stripe_key: sk_live_...
```

**Rejected because:**
- Encourages committing secrets to git
- No better than env vars security-wise
- Harder to vary between environments
- Against 12-factor app principles

### Alternative C: Only Site Metadata (No Env Mapping)

Just add `site` section, tell users to use `os.getenv()` in Parsley.

**Rejected because:**
- Requires adding `os.getenv()` to Parsley (maybe fine)
- No type coercion or defaults
- Less guidance toward secure practices
- Misses opportunity to make env vars ergonomic

### Alternative D: Combined Section

```yaml
config:
  site_name: "My Blog"           # literal value
  stripe_key: ${STRIPE_KEY}      # env var interpolation
  port: ${PORT:-8080}            # with default
```

**Rejected because:**
- Mixes secrets and non-secrets in same section
- Shell-like syntax feels foreign in YAML
- Harder to scan for "what needs env vars set"
- Less clear separation of concerns

### Alternative E: Whole-File Env Interpolation

Instead of a dedicated `env:` section, interpolate `${VAR}` anywhere in the config:

```yaml
port: ${PORT:-8080}
database:
  url: ${DATABASE_URL}
  pool_size: ${DB_POOL:-10}
  
auth:
  google_client_id: ${GOOGLE_CLIENT_ID}
  
site:
  api_endpoint: ${API_URL:-https://api.example.com}
```

**Analysis:**

This is a common pattern (docker-compose, Kubernetes, many tools). Implementation options:

1. **Pre-parse interpolation** — Run `envsubst`-style replacement before YAML parsing
   - Simple to implement
   - Familiar syntax
   - But: can break YAML if env var contains special chars
   
2. **Post-parse interpolation** — Parse YAML, then walk tree replacing `${...}` strings
   - Safer (YAML structure preserved)
   - Can do type coercion based on context
   - More complex to implement

**Advantages:**
- No special section needed
- Env vars usable anywhere (port, database settings, site metadata)
- Familiar to developers from other tools
- More flexible than dedicated section

**Disadvantages:**
- Loses explicit "what env vars does this app need" declaration
- Harder to report "missing env vars needed" at startup
- Type coercion less explicit (infer from YAML context?)
- How to mark secrets vs non-secrets for DevTools?

**Hybrid approach (RECOMMENDED):**

Support both! Whole-file interpolation for flexibility, plus the secret marker:

```yaml
port: ${PORT:-8080}
database:
  url: ${$DATABASE_URL}           # $ after ${ = secret
  
site:
  name: "My Blog"
  api_url: ${PUBLIC_API_URL}      # no $ = not secret
  
# Optional: explicit env section for discoverability
env:
  stripe_key: $STRIPE_KEY         # still useful for Parsley access as env.*
```

The `${$VAR}` syntax (or `$${VAR}`) marks a secret. Bit ugly, but unambiguous.

**Alternative secret syntax options:**

| Syntax | Example | Notes |
|--------|---------|-------|
| `${$VAR}` | `${$DATABASE_URL}` | Dollar inside braces = secret |
| `$${VAR}` | `$${DATABASE_URL}` | Double dollar = secret |
| `${VAR!}` | `${DATABASE_URL!}` | Trailing bang = secret |
| `${secret:VAR}` | `${secret:DATABASE_URL}` | Explicit prefix |
| `$[VAR]` | `$[DATABASE_URL]` | Brackets = secret |

**Recommendation:** `${$VAR}` — the `$` inside braces echoes the `$` prefix in the env section, maintaining consistency.

## Security Considerations

### What This Design Protects Against

| Threat | Protection |
|--------|------------|
| Secrets committed to git | ✅ Env vars never in config file |
| Secrets in backups | ✅ Only env var names stored |
| Accidental logging | ⚠️ Parsley could still log `env.stripe_key` |
| Compromised server | ❌ Env vars readable by attacker |

### What This Design Does NOT Protect Against

- **Server compromise** — If attacker has shell access, they can read env vars
- **Memory dumps** — Secrets exist in process memory
- **Parsley logging** — Developer could `@log env.stripe_key`

### Guidance for Developers

The `docs/guide/secrets.md` should cover:

1. **Never commit secrets** — Use env vars or secrets managers
2. **Use `.env` files for local dev** — But `.gitignore` them
3. **Production deployment** — Set env vars via your platform (Heroku, Railway, systemd, Docker, etc.)
4. **Rotation** — Env vars make rotation easy (just restart with new value)
5. **When to use secrets managers** — For larger deployments, consider Vault, AWS Secrets Manager, etc.

## DevTools Integration

### `/__/env` Page Updates

Add sections for site metadata and env mappings. Secrets (marked with `$`) show `[hidden]` instead of the actual value:

```
Site Metadata
─────────────────────────────────────────────────
name              "My Awesome Blog"
contact_email     "hello@example.com"
api_url           "https://api.example.com"      (from PUBLIC_API_URL)
features.comments true
features.dark_mode false

Environment Variables (interpolated)
─────────────────────────────────────────────────
Setting              Env Var              Value
port                 PORT                 8080 (default)
database.url         $DATABASE_URL        [hidden]           ← secret
database.pool        DB_POOL              10 (default)
auth.client_id       GOOGLE_CLIENT_ID     "abc123..."
auth.client_secret   $GOOGLE_CLIENT_SECRET [hidden]          ← secret

Named Env Mappings (env.*)
─────────────────────────────────────────────────
Parsley Name    Env Var              Secret?   Value
stripe_key      $STRIPE_SECRET_KEY   yes       [hidden]
database_url    $DATABASE_URL        yes       [hidden]  
site_url        SITE_URL             no        "https://example.com"
debug_mode      DEBUG:bool           no        false (default)
```

**Rules:**
- Values from env vars marked with `$` prefix show `[hidden]`
- Non-secret env var values are shown (truncated if long)
- Show whether value is from env var or default
- Show type coercion where applicable

## Migration / Backwards Compatibility

- **No breaking changes** — Both sections are optional
- **Existing configs** — Continue to work unchanged  
- **Gradual adoption** — Developers can add sections as needed
- **Session secret** — Will be auto-generated (may invalidate existing sessions, acceptable for pre-alpha)

## Testing Strategy

### Unit Tests

- `TestEnvParsing` — Type coercion for all types
- `TestEnvDefaults` — Default value handling
- `TestEnvMissing` — Behavior when env var unset
- `TestSecretWarnings` — Warning detection patterns
- `TestSiteMetadata` — Arbitrary nesting, all types

### Integration Tests

- `TestSiteInParsley` — Access `site.*` in templates
- `TestEnvInParsley` — Access `env.*` in templates
- `TestStartupWarnings` — Warning output format
- `TestDevToolsEnvPage` — New sections appear correctly

## Open Questions

### Q1: Should `env` values be available in handlers vs templates?

**Proposal:** Available everywhere Parsley runs (handlers, templates, parts).

### Q2: Should we support nested env mappings?

```yaml
env:
  database:
    url: DATABASE_URL
    pool_size: DB_POOL:int:10
```

**Proposal:** No, keep it flat. Nesting adds complexity without clear benefit. Use naming conventions: `db_url`, `db_pool_size`.

### Q3: Should missing env vars (without defaults) be errors or warnings?

**Proposal:** Warnings, with value set to `null`. Let Parsley code handle the null case. Errors would break startup, which is too strict for optional config.

### Q4: What about the existing `session_secret` in config?

**Decision:** Auto-generate secure random. Generate a cryptographically secure random secret at startup if none is provided. Store in memory only (regenerated each restart, which is fine for sessions). Accept `SESSION_SECRET` env var as override for multi-instance deployments. Remove the config field entirely.

This may invalidate existing sessions on upgrade—acceptable for pre-alpha.

### Q5: Marking secrets in env mappings

**Decision:** Use `$` prefix on env var name to mark as secret:

```yaml
env:
  stripe_key: $STRIPE_SECRET_KEY   # $ prefix = secret, hidden in devtools
  database_url: $DATABASE_URL      # secret
  site_url: SITE_URL               # no $ = not secret, shown in devtools
  debug: DEBUG:bool:false          # not secret
```

This is elegant:
- `$` visually signals "this is sensitive" (familiar from shell)
- Machine-readable for DevTools (obscure values with `$` prefix)
- No YAML ambiguity (`$FOO` is a valid string in YAML)
- Complements rather than replaces the `!safe` tag for site metadata

## Implementation Phases

### Phase 1: Whole-File Env Interpolation + Session Secret (FEAT-102?)
- Pre-parse `${VAR}` and `${VAR:-default}` substitution
- Track `${$VAR}` as secrets (for DevTools)
- Auto-generate session secret (remove config field)
- Basic tests
- **This unlocks env vars anywhere in config immediately**

### Phase 2: Site Metadata (FEAT-103?)
- Add `Site` field to config
- Inject into Parsley as `site.*`
- Tests and documentation

### Phase 3: Named Env Mappings (FEAT-104?)
- Add `Env` field to config  
- Implement parsing with types/defaults and `$` secret marker
- Inject into Parsley as `env.*`
- Startup reporting of missing/set vars
- Tests

### Phase 4: Warnings & DevTools Polish (FEAT-105?)
- Secret-like key detection in `site` section
- `!safe` YAML tag
- DevTools `/__/env` page updates (site, interpolated vars, env mappings)
- Secrets guide documentation

## Success Criteria

1. Developer can use `${VAR}` anywhere in basil.yaml
2. Developer can add `site:` section and access values as `site.*` in Parsley
3. Developer can add `env:` mappings and access as `env.*` in Parsley with type safety
4. Secrets marked with `$` are hidden in DevTools
5. Warnings guide developers away from putting secrets in `site` section
6. Session secret is auto-generated (no config needed)
7. Documentation clearly explains the pattern and security considerations

## References

- [12-Factor App: Config](https://12factor.net/config)
- [OWASP: Secrets Management](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html)
- [Rails credentials](https://guides.rubyonrails.org/security.html#custom-credentials)
- [Django settings](https://docs.djangoproject.com/en/4.2/topics/settings/)
- [Next.js Environment Variables](https://nextjs.org/docs/basic-features/environment-variables)
