# FEAT-020 Implementation Plan: Per-Developer Config Overrides

## Overview
Enable multiple developers to run isolated Basil instances on the same machine by supporting config overrides via local config file, environment variables, and CLI flags.

## Current State
- Config loaded from `basil.yaml` (or `--config` path)
- Only `--port` CLI override exists (in main.go)
- No local config file support
- No environment variable overrides (except `${VAR}` interpolation in YAML values)

## Implementation Phases

### Phase 1: Local Config File (`basil.local.yaml`)
**Goal**: Auto-load and merge a gitignored local config file

**Changes to `config/load.go`**:

1. Add `loadLocalConfig()` function:
   ```go
   func loadLocalConfig(baseConfigPath string) (*Config, error)
   ```
   - Given `/path/to/basil.yaml`, looks for `/path/to/basil.local.yaml`
   - Returns nil (not error) if file doesn't exist
   - Parses YAML into Config struct

2. Add `mergeConfig()` function:
   ```go
   func mergeConfig(base, override *Config) *Config
   ```
   - Top-level field merge (not deep merge)
   - Zero/empty values in override don't replace base values
   - Need careful handling of:
     - `Server.Port` (0 means "not set", use base)
     - Slice fields like `Routes`, `Static` (override replaces entirely if non-empty)
     - String fields (empty string means "not set")

3. Modify `LoadWithPath()`:
   - After loading base config, call `loadLocalConfig()`
   - If local config exists, merge it

**Tests**:
- `TestLoadLocalConfig` - loads local file when present
- `TestLoadLocalConfigMissing` - silently skips when absent  
- `TestMergeConfig` - various merge scenarios
- `TestLocalConfigOverridesBase` - integration test

**Estimated effort**: 1-2 hours

---

### Phase 2: Environment Variable Overrides
**Goal**: Apply `BASIL_*` env vars after config file loading

**Changes to `config/load.go`**:

1. Add `applyEnvOverrides()` function:
   ```go
   func applyEnvOverrides(cfg *Config, getenv func(string) string)
   ```
   
2. Environment variable mapping:
   | Env Var | Config Field | Parse |
   |---------|--------------|-------|
   | `BASIL_PORT` | `Server.Port` | `strconv.Atoi` |
   | `BASIL_HANDLERS` | First route's `Handler` dir? | string |
   | `BASIL_STATIC` | `PublicDir` | string |
   | `BASIL_DATABASE` | `Auth.Database` or `Database.Path` | string |
   | `BASIL_DEV_DATABASE` | `Dev.LogDatabase` | string |

3. Modify `LoadWithPath()`:
   - After merging local config, call `applyEnvOverrides()`

**Design decision needed**: 
- `BASIL_HANDLERS` - what does this override? We have `Routes[].Handler` (individual files) but no global "handlers directory". 
- Options:
  a) Add `HandlersDir` to Config that gets used as base for relative handler paths
  b) Override just the first route's handler
  c) Skip this env var for now (users use local config for complex changes)

**Recommendation**: Add `HandlersDir string` to Config, use it as prefix for relative handler paths. This matches common pattern.

**Tests**:
- `TestEnvOverridePort`
- `TestEnvOverrideDatabase`
- `TestEnvOverrideEmpty` - empty env var doesn't override
- `TestEnvOverridePriority` - env beats local config

**Estimated effort**: 1 hour

---

### Phase 3: CLI Flags
**Goal**: Add CLI flags for common overrides

**Changes to `cmd/basil/main.go`**:

1. Add new flags:
   ```go
   handlers    = flags.String("handlers", "", "Override handlers directory")
   database    = flags.String("db", "", "Override database path")
   localConfig = flags.String("local-config", "", "Path to local config override file")
   ```

2. Apply after env overrides:
   ```go
   if *handlers != "" {
       cfg.HandlersDir = *handlers
   }
   if *database != "" {
       cfg.Database.Path = *database
   }
   ```

3. If `--local-config` provided, use that instead of auto-detected `basil.local.yaml`

4. Update `printUsage()` with new flags

**Tests**:
- Existing test pattern in `main_test.go`

**Estimated effort**: 30 minutes

---

### Phase 4: Documentation & Cleanup
**Goal**: Document the feature and add gitignore guidance

1. Update `basil.example.yaml` with comments about local overrides
2. Add `basil.local.yaml` to recommended `.gitignore`
3. Update docs (FAQ, quick-start)
4. Update FEAT-020 spec with implementation notes

**Estimated effort**: 30 minutes

---

## Total Estimated Effort
~3-4 hours

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Merge logic complexity | Med | Keep it simple: top-level merge only, document behavior |
| HandlersDir concept unclear | Low | Clear docs, sensible default (config file directory) |
| Breaking existing behavior | Med | All overrides are additive; base config behavior unchanged |

## Testing Strategy
- Unit tests for each new function
- Integration tests for full load → merge → override flow
- Manual testing with example project

## Progress Log
| Phase | Status | Notes |
|-------|--------|-------|
| Phase 1: Local Config | Not started | |
| Phase 2: Env Vars | Not started | |
| Phase 3: CLI Flags | Not started | |
| Phase 4: Docs | Not started | |
