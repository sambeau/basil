---
id: FEAT-020
title: "Per-Developer Config Overrides"
status: complete
priority: medium
created: 2025-12-03
author: "@human"
---

# FEAT-020: Per-Developer Config Overrides

## Summary
Allow a single config file to define multiple developer instances, each with their own port, database, handlers, and static paths. Admin controls all configurations centrally; developers select their instance via CLI flag.

## User Story
As an **admin**, I want **to define developer instances in a single config file** so that **I control what configurations are available and can manage them centrally**.

As a **developer on a team**, I want **to run my own Basil instance by name** so that **I can develop on my branch without conflicting with other developers**.

## Acceptance Criteria

- [x] Config supports `developers` section with named developer configs
- [x] Each developer config can override: `port`, `database`, `handlers`, `static`, `logging`
- [x] `basil --dev --profile alice` or `basil --dev -as alice` runs using developer "alice" config
- [x] Developer configs inherit base config values if not specified
- [x] Production config (server section) remains unchanged

## Config Example

```yaml
server:
  host: example.com
  port: 443

static:
  - path: /static/
    root: ./public

database:
  path: ./data/production.db

routes:
  - path: /
    handler: ./handlers/index.pars

# Developer instances - each runs on their own port
developers:
  alice:
    port: 3001
    database: ./data/alice.db
    # handlers and static inherited from base
    
  bob:
    port: 3002
    database: ./data/bob.db
    handlers: ./handlers-experimental  # override
    
  shared:
    port: 3000
    # Everything else inherited - for quick testing
```

## Usage

```bash
# Run as developer "alice"
basil --dev -as alice

# Run as developer "bob" 
basil --dev --profile bob

# Development mode (base config)
basil --dev

# Production (no --dev flag)
basil
```

## Design Decisions

- **Single config file**: Admin controls all configurations; no local files to manage
- **Named developers**: Clear, explicit; avoid magic port assignment
- **Inheritance**: Developer configs only need to specify overrides

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `config/config.go` — Add `DeveloperConfig` struct and `Developers` map to Config
- `config/load.go` — Add merge logic for developer config
- `cmd/basil/main.go` — Modify `--dev` flag to accept optional name

### Dependencies
- Depends on: None
- Blocks: Future multi-hostname/multi-site features

### Edge Cases & Constraints
1. **No developers defined + `--dev name`** — Error: "developer 'name' not found in config"
2. **`--dev` without name, no developers** — Current behavior (dev mode, base config)
3. **`--dev` without name, developers exist** — Use first developer? Or require name? (TBD)
4. **Developer overrides production port** — Allowed (admin's choice)
5. **Relative paths in developer config** — Resolve relative to config file (same as base)

### Developer Config Fields
| Field | Type | Overrides |
|-------|------|-----------|
| `port` | int | `server.port` |
| `database` | string | `database.path` |
| `handlers` | string | `handlers.root` (new field) |
| `static` | string | `public_dir` |
| `logging` | LoggingConfig | `logging.*` |

### Implementation Notes
*Added during/after implementation*

**Implemented 2025-01-09:**
- Added `DeveloperConfig` struct to `config/config.go`
- Added `Developers map[string]DeveloperConfig` to `Config` struct
- Added `ApplyDeveloper(cfg *Config, profileName string)` to `config/load.go`
- Added `--profile` flag with `-as` alias to `cmd/basil/main.go`
- Profile is applied after config load but before other CLI overrides (port, quiet)
- Tests added to `config/load_test.go` covering all override scenarios
- Example added to `basil.example.yaml`

**Design change:** Used separate `--profile`/`-as` flags rather than extending `--dev` to take an optional argument, as Go's flag package doesn't support optional arguments cleanly.

## Related
- Plan: [FEAT-020-plan.md](../plans/FEAT-020-plan.md)
