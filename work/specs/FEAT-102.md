---
id: FEAT-102
title: "Custom Metadata, Environment Variables, and Secrets"
status: draft
priority: high
created: 2026-01-25
author: "@human"
---

# FEAT-102: Custom Metadata, Environment Variables, and Secrets

## Summary

Add three capabilities to Basil configuration: a `meta` section for custom site metadata accessible in Parsley, `${VAR}` interpolation for environment variables anywhere in the config, and a `!secret` YAML tag to mark sensitive values (hidden in DevTools). This provides a clean pattern for environment-specific configuration while guiding developers toward secure practices.

## User Story

As a Basil developer, I want to store custom metadata and use environment variables in my config so that I can have different settings for dev/staging/production without committing secrets to git.

## Acceptance Criteria

- [ ] `meta:` section in basil.yaml accessible as `meta.*` in Parsley
- [ ] `${VAR}` syntax interpolates environment variables anywhere in config
- [ ] `${VAR:-default}` syntax provides fallback when env var is unset (POSIX convention)
- [ ] `!secret` YAML tag marks values as sensitive
- [ ] `!secret` values show `●●●●●●●●` in DevTools `/__/env` page
- [ ] `!secret auto` auto-generates a secure random value (for session secrets)
- [ ] Session secret auto-generated if not provided (remove config field)
- [ ] Type coercion from env vars inferred from YAML context
- [ ] Tests verify secrets are hidden in DevTools output
- [ ] Documentation covers syntax and security best practices

## Design Decisions

- **`meta` not `site`**: The `site` key is already used for filesystem-based routing directory. `meta` is consistent with Parsley's record metadata syntax.

- **`${VAR}` syntax**: Standard shell-like syntax, widely familiar from Docker, Kubernetes, etc. Not `@{VAR}` to avoid suggesting Parsley expressions are supported (deferred).

- **`!secret` as YAML tag**: Leverages YAML's native tag feature. Orthogonal to interpolation — `!secret ${VAR}` composes naturally.

- **Auto-generate session secret**: Removes common security footgun. Breaking change acceptable for pre-alpha. Override via `SESSION_SECRET` env var for multi-instance deployments.

- **No encryption**: `!secret` hides values in DevTools but doesn't encrypt. If server is compromised, secrets are exposed regardless. Documentation emphasizes this.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Syntax Reference

```yaml
# Literal values
key: value                           # literal
key: !secret value                   # literal, marked secret

# Environment variable interpolation
# Uses POSIX shell syntax: ${VAR:-default} (hyphen required)
key: ${VAR}                          # env var (empty string if unset)
key: ${VAR:-default}                 # env var with default (POSIX syntax)
key: !secret ${VAR}                  # env var, marked secret
key: !secret ${VAR:-default}         # env var with default, marked secret

# String concatenation
key: "https://${HOST}:${PORT}/api"   # multiple interpolations
```

### Affected Components

- `server/config/config.go` — Add `Meta map[string]interface{}` field
- `server/config/interpolate.go` — New: `${VAR}` pre-processing
- `server/config/secret.go` — New: `!secret` YAML tag handler, `SecretValue` type
- `server/config/loader.go` — Integrate interpolation + secret tracking
- `server/prelude.go` — Inject `meta.*` into Parsley environment
- `server/devtools.go` — Update `/__/env` to hide `!secret` values
- `server/session.go` — Handle `auto` for session secret generation
- `server/config/sanitize.go` — Update to include meta section, respect secrets

### Processing Order

1. Read `basil.yaml` as raw text
2. Pre-process `${VAR}` and `${VAR:-default}` interpolations (POSIX syntax)
3. Parse resulting YAML with custom `!secret` tag handler
4. Validate structure against Config schema
5. Track which values are marked secret (for DevTools)

### Secret Tracking Data Model

```go
type SecretValue struct {
    Value    interface{}
    IsSecret bool
}

// During config loading, track paths to secret values
type ConfigSecrets struct {
    paths map[string]bool  // e.g., "auth.google.client_secret" -> true
}
```

### Env Var Interpolation

```go
// Match ${VAR} or ${VAR:-default} (POSIX shell syntax)
var envVarRegex = regexp.MustCompile(`\$\{([^}:]+)(?::-([^}]*))?\}`)

func interpolateEnvVars(content string) string {
    return envVarRegex.ReplaceAllStringFunc(content, func(match string) string {
        parts := envVarRegex.FindStringSubmatch(match)
        varName := parts[1]
        defaultVal := parts[2]
        
        if val, ok := os.LookupEnv(varName); ok {
            return val
        }
        return defaultVal
    })
}
```

### Dependencies

- Depends on: None
- Blocks: None

### Edge Cases & Constraints

1. **Unset env var without default** — Returns empty string, YAML parses as empty/null depending on context
2. **Env var value contains YAML special chars** — Could break parsing; document that values should be quoted if containing `:`, `#`, etc.
3. **Nested `${...}` in env var value** — Not recursively expanded (single pass)
4. **`!secret` on non-scalar** — Allowed on maps/arrays, entire subtree marked secret
5. **Type coercion** — `port: ${PORT:-8080}` infers int from default; `port: ${PORT}` with `PORT=8080` infers string unless quoted

### Migration

- **Breaking**: `session.secret` config field behavior changes
- **Action**: Use `!secret auto` or `!secret ${SESSION_SECRET}` 
- **Impact**: Existing sessions invalidated on upgrade (acceptable for pre-alpha)

## Implementation Notes

*To be added during implementation*

## Related

- Plan: `work/plans/PLAN-075-metadata-env-secrets.md`
- Design doc: `work/design/metadata-and-secrets-v2.md`
- Superseded design: `work/design/metadata-and-secrets.md` (V1)
