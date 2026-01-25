---
id: PLAN-075
feature: FEAT-102
title: "Implementation Plan: Custom Metadata, Env Vars, and Secrets"
status: completed
created: 2026-01-25
completed: 2026-01-25
---

# Implementation Plan: FEAT-102

## Overview

Implement three configuration capabilities:
1. `meta:` section for custom metadata (accessible as `meta.*` in Parsley)
2. `${VAR}` interpolation for environment variables anywhere in config
3. `!secret` YAML tag to mark sensitive values (hidden in DevTools)

## Prerequisites

- [x] Design finalized in `work/design/metadata-and-secrets-v2.md`
- [x] FEAT-102 spec reviewed

## Tasks

### Task 1: Environment Variable Interpolation
**Files**: `server/config/load.go` (existing)
**Status**: ✅ Complete (already implemented)

Environment variable interpolation with `${VAR}` and `${VAR:-default}` syntax already exists in `server/config/load.go`.

---

### Task 2: Secret YAML Tag
**Files**: `server/config/secret.go`, `server/config/secret_test.go`
**Status**: ✅ Complete

Implemented:
- `SecretString` struct with `value`, `isSecret` fields
- Custom YAML unmarshaler for `!secret` tag
- `SecretTracker` to track paths to secret values
- `ResolveSecretValue()` for runtime resolution
- `GenerateSecureSecret()` for auto-generation

Tests:
- `TestSecretStringUnmarshal`
- `TestSecretStringString`
- `TestSecretStringIsAuto`
- `TestSecretStringMarshal`
- `TestSecretTracker`
- `TestGenerateSecureSecret`
- `TestResolveSecretValue`

---

### Task 3: Config Loader Integration
**Files**: `server/config/config.go`, `server/config/load.go`
**Status**: ✅ Complete

Implemented:
- Added `Meta map[string]interface{}` field to `Config` struct
- Added `Secrets *SecretTracker` field to `Config` struct
- Changed `Session.Secret` from `string` to `SecretString`
- Updated `Defaults()` to initialize `Secrets` and set `Secret: NewSecretString("auto")`
- Tracking of `session.secret` in `SecretTracker` during load

Tests:
- `TestLoadWithMeta`
- `TestLoadWithSecretTag`
- `TestLoadWithSecretAuto`

---

### Task 4: Session Secret Auto-Generation
**Files**: `server/server.go`, `server/config/secret.go`
**Status**: ✅ Complete

Implemented:
- `initSessions()` updated to use `config.ResolveSecretValue()`
- Supports `!secret auto` for auto-generation
- Supports `SESSION_SECRET` env var override
- Supports literal secrets

---

### Task 5: Parsley Integration for Meta
**Files**: `server/handler.go`
**Status**: ✅ Complete

Implemented:
- Meta injection in handler as protected `meta` variable
- Converts `config.Meta` to Parsley object using `parsley.ToParsley()`

Tests:
- `TestMetaInjection`

---

### Task 6: DevTools Secret Hiding
**Files**: `server/devtools.go`, `server/prelude/devtools/env.pars`
**Status**: ✅ Complete

Implemented:
- Expanded `/__/env` page to show more config
- Session secret displayed as `●●●●●●●●` or `(auto-generated)`
- Added meta section display in DevTools
- Updated `TestDevToolsEnvNoSecrets` test

---

### Task 7: Documentation
**Files**: `docs/guide/configuration-example.yaml`, `docs/guide/faq.md`
**Status**: ✅ Complete

Implemented:
- Added meta and session sections to configuration-example.yaml
- Added env vars and secrets section to configuration-example.yaml
- Added FAQ entries for meta, env vars, !secret, and session secret auto

---

### Task 8: Integration Tests
**Status**: ✅ Complete

Added tests:
- `TestLoadWithMeta` — meta section loads correctly
- `TestLoadWithSecretTag` — !secret tag works and tracks secrets
- `TestLoadWithSecretAuto` — !secret auto recognized
- `TestMetaInjection` — meta accessible in Parsley handlers
- `TestDevToolsEnvNoSecrets` — session secret masked in DevTools
- `TestSessionSecretStable` — Same value used throughout server lifetime
- `TestSessionSecretRandom` — Different servers get different secrets

---

### Task 5: Parsley Integration for Meta
**Files**: `server/prelude.go`
**Estimated effort**: Small

Inject `meta.*` values into Parsley environment.

Steps:
1. In prelude setup, check if `config.Meta` is non-nil
2. Convert `Meta` map to Parsley object
3. Bind as `meta` in Parsley environment
4. Ensure read-only (no write-back needed)

Tests:
- `TestMetaInParsley` — Access `meta.name` in Parsley
- `TestMetaNestedInParsley` — Access `meta.features.dark_mode`
- `TestMetaWithEnvVar` — `meta` value from `${VAR}` accessible
- `TestMetaMissing` — No error when `meta:` section absent

---

### Task 6: DevTools Secret Hiding
**Files**: `server/devtools.go`, `server/config/sanitize.go`
**Estimated effort**: Medium

Update `/__/env` page to show `[hidden]` for secret values.

Steps:
1. Add `meta` section to `SanitizedConfig()` output
2. Check secret paths when building sanitized output
3. Replace secret values with `[hidden]` string
4. Show env var name and source for interpolated values
5. Update env.pars template to display meta section

Tests:
- `TestDevToolsSecretsHidden` — `!secret` values show `[hidden]`
- `TestDevToolsNonSecretsShown` — Non-secret values visible
- `TestDevToolsMetaSection` — Meta section appears in output
- `TestDevToolsEnvVarSource` — Shows env var name as source

---

### Task 7: Documentation
**Files**: `docs/guide/configuration.md`, `docs/guide/secrets.md`
**Estimated effort**: Medium

Document the new configuration features.

Steps:
1. Update `docs/guide/configuration.md` with `meta:` section docs
2. Update with `${VAR}` interpolation syntax
3. Document `!secret` tag usage
4. Create `docs/guide/secrets.md` with security best practices
5. Add examples for common patterns (dev/prod configs, .env files)

Documentation sections:
- Meta section syntax and Parsley access
- Environment variable interpolation
- Default values
- Secret marking
- Session secret auto-generation
- Security considerations and best practices

---

### Task 8: Integration Tests
**Files**: `server/config/integration_test.go`
**Estimated effort**: Small

End-to-end tests for the complete feature.

Steps:
1. Create test config file with all features
2. Test full load → Parsley access → DevTools display flow
3. Verify secrets hidden end-to-end
4. Test with various env var combinations

Tests:
- `TestFullConfigFlow` — Load config, access in Parsley, check DevTools
- `TestConfigDevProdPattern` — Typical dev/prod config pattern works
- `TestConfigNoSecretLeak` — No secrets in any output

---

## Validation Checklist

- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] `docs/guide/configuration.md` updated
- [ ] `docs/guide/secrets.md` created
- [ ] Example configs updated in `examples/`
- [ ] work/BACKLOG.md updated with deferrals

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| | Task 1: Env interpolation | ⬜ Not started | — |
| | Task 2: Secret tag | ⬜ Not started | — |
| | Task 3: Loader integration | ⬜ Not started | — |
| | Task 4: Session secret auto | ⬜ Not started | — |
| | Task 5: Parsley meta | ⬜ Not started | — |
| | Task 6: DevTools secrets | ⬜ Not started | — |
| | Task 7: Documentation | ⬜ Not started | — |
| | Task 8: Integration tests | ⬜ Not started | — |

## Deferred Items

Items to add to work/BACKLOG.md after implementation:

- **Parsley interpolation in config** — Use `@{expr}` instead of `${VAR}` for full Parsley expression support. Deferred: `${VAR}` covers 95% of use cases.
- **Named env mappings (`env:` section)** — Explicit env var documentation section. Deferred: Direct interpolation is simpler.
- **Secret warnings** — Warn if `meta` section contains secret-like keys. Deferred: Can add without breaking changes.
- **Recursive interpolation** — Expand `${VAR}` in env var values. Deferred: Single pass is safer and predictable.
