---
id: FEAT-020
title: "Per-Developer Config Overrides"
status: draft
priority: medium
created: 2025-12-03
author: "@human"
---

# FEAT-020: Per-Developer Config Overrides

## Summary
Allow developers to override config values (port, database, handlers directory) without modifying the shared `basil.yaml`. This enables multiple developers to run isolated instances on the same machine, each with their own port and data.

## User Story
As a **developer on a team**, I want **to run my own Basil instance with custom port and database** so that **I can develop on my branch without conflicting with other developers**.

As a **solo developer**, I want **to test different configurations quickly** so that **I can experiment without editing my main config file**.

## Acceptance Criteria

### Phase 1: Local Config File
- [ ] Support `basil.local.yaml` that merges on top of `basil.yaml`
- [ ] `basil.local.yaml` is auto-loaded if present (same directory as main config)
- [ ] Local config only needs to specify overrides, not full config
- [ ] Document that `basil.local.yaml` should be gitignored

### Phase 2: Environment Variables
- [ ] `BASIL_PORT` overrides `server.port`
- [ ] `BASIL_HANDLERS` overrides `handlers.root`
- [ ] `BASIL_STATIC` overrides `static.root`
- [ ] `BASIL_DATABASE` overrides `auth.database`
- [ ] `BASIL_DEV_DATABASE` overrides `dev.log_database`
- [ ] Environment variables take precedence over config files

### Phase 3: CLI Flags
- [ ] `--port` flag overrides port
- [ ] `--handlers` flag overrides handlers root
- [ ] `--db` flag overrides auth database
- [ ] CLI flags take highest precedence
- [ ] `--local-config` flag to specify alternate local config file

## Design Decisions

- **Priority order**: CLI > env vars > basil.local.yaml > basil.yaml (standard override pattern)
- **Naming**: `basil.local.yaml` matches patterns like `.env.local`, `docker-compose.override.yml`
- **Merge strategy**: Simple top-level merge, not deep merge (keeps it predictable)
- **Env var prefix**: `BASIL_` namespace avoids conflicts
- **Minimal CLI flags**: Only the most common overrides; env vars cover the rest

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `config/load.go` — Add local config loading, env var parsing, merge logic
- `config/config.go` — May need helper methods for merging
- `cmd/basil/main.go` — Add CLI flags, pass to config loader

### Dependencies
- Depends on: None
- Blocks: Future multi-hostname/multi-site features (B use case)

### Edge Cases & Constraints
1. **Missing local config** — Silently skip (not an error)
2. **Invalid local config** — Error with clear message mentioning it's the local file
3. **Relative paths in local config** — Resolve relative to config file location (existing behavior)
4. **Partial overrides** — Only specified fields override; unspecified fields keep base values
5. **Empty env vars** — Treat empty string as "not set" (don't override with empty)

### Config Loading Algorithm
```
1. Determine base config path (--config flag or default basil.yaml)
2. Load base config
3. Check for basil.local.yaml in same directory
4. If exists, load and merge (local values win)
5. Apply BASIL_* environment variables (if set and non-empty)
6. Apply CLI flags (if provided)
7. Validate final merged config
```

### Environment Variable Mapping
| Env Var | Config Path | Type |
|---------|-------------|------|
| BASIL_PORT | server.port | int |
| BASIL_HANDLERS | handlers.root | string |
| BASIL_STATIC | static.root | string |
| BASIL_DATABASE | auth.database | string |
| BASIL_DEV_DATABASE | dev.log_database | string |

### CLI Flag Mapping
| Flag | Config Path | Notes |
|------|-------------|-------|
| --port | server.port | Short: -p |
| --handlers | handlers.root | |
| --db | auth.database | |
| --local-config | (special) | Path to alternate local config |

## Implementation Notes
*Added during/after implementation*

## Related
- Backlog item: "HTTP-only production mode (behind proxy)" (related to future use case B)
