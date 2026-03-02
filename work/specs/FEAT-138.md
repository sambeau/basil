---
id: FEAT-138
title: "Config YAML Consistency for 1.0"
status: draft
priority: high
created: 2026-03-02
author: "@human"
---

# FEAT-138: Config YAML Consistency for 1.0

## Summary

Make the `basil.yaml` configuration schema consistent, logical, and future-proof before the 1.0 release. This is a **clean break** — no deprecation period, no backward compatibility with old key names. All changes ship together in one release with clear documentation.

## User Story

As a Basil user, I want the config file to be consistent and predictable so that I can guess key names without checking the docs every time.

## Design Principles

1. **`snake_case` everywhere** — no exceptions
2. **Group by concern** — related settings nest together
3. **Top-level keys = major feature areas** — each is a noun representing a domain
4. **Scalars don't float** — every value lives under its logical parent
5. **Names should be guessable** — if you know the feature, you guess the path
6. **Consistent naming** — the same concept uses the same name everywhere (e.g., `path` for filesystem paths)

## Changes

### Change 1: `cors.maxAge` → `cors.max_age`

The only camelCase YAML key in the entire config. Fix it to match everything else.

**Before:**
```yaml
cors:
  maxAge: 86400
```

**After:**
```yaml
cors:
  max_age: 86400
```

**Go struct:** `CORSConfig.MaxAge` YAML tag changes from `maxAge` to `max_age`.

**Affected files:**
- `server/config/config.go` — YAML tag on `CORSConfig.MaxAge`
- `server/config/config_test.go` — test YAML strings and assertions
- `server/cors.go` — reads `m.config.MaxAge` (Go field unchanged, only YAML tag changes)
- `examples/cors/basil.yaml` — uses `maxAge: 86400`
- `docs/guide/basil-quick-start.md` — CORS section and configuration options table reference `maxAge`
- `docs/guide/cors.md` — references `maxAge` throughout (15+ occurrences)

---

### Change 2: `SessionConfig.HttpOnly` → `HTTPOnly` (Go only)

Go convention: HTTP is an initialism, so the struct field should be `HTTPOnly`. The YAML tag `http_only` is already correct and does **not** change. No `basil.yaml` files are affected.

**Go struct:** `SessionConfig.HttpOnly` → `SessionConfig.HTTPOnly`

**Affected files:**
- `server/config/config.go` — field rename and `Defaults()` initialiser
- `server/config/config_test.go` — struct literals referencing the field
- `server/session.go` — uses `s.config.HttpOnly` in `Save()` and `Clear()`
- `server/session_test.go` — struct literal `HttpOnly: true` in `testSessionConfig()`

---

### Change 3: `sqlite` → `database.path`

Group database config into a section. SQLite is the only supported embedded database and there are no plans to change this. The section exists to logically group database-related config and leave room for future tuning knobs (pragmas, journal mode, busy timeout, etc.) without polluting the top level.

**Before:**
```yaml
sqlite: ./data.db
```

**After:**
```yaml
database:
  path: ./data.db
```

**Go struct:** Replace `SQLite string` field on `Config` with `Database DatabaseConfig`:

```go
type DatabaseConfig struct {
    Path string `yaml:"path"` // Path to SQLite database file
}
```

All references to `cfg.SQLite` become `cfg.Database.Path`.

The `DeveloperConfig.SQLite` field is also replaced as part of this change — see Change 5.

**Affected files:**
- `server/config/config.go` — replace `SQLite string` with `Database DatabaseConfig`, add struct, update `Defaults()`
- `server/config/config_test.go` — update test YAML strings and field assertions
- `server/config/load.go` — path resolution (`cfg.SQLite` → `cfg.Database.Path`)
- `server/config/load_test.go` — all SQLite config tests (`TestLoadSQLiteConfig`, `TestApplyDeveloper` SQLite cases)
- `server/database_test.go` — sets `cfg.SQLite` directly in four test functions
- `server/server.go` — reads `cfg.SQLite` during database initialisation
- `server/devtools.go` — reads `cfg.SQLite` in six places: `openAppDB()`, `serveDB()`, `handleDevDBFileDownload()`, `handleDevDBFileUpload()`, and two locations in `createDevToolsEnv()` (index page `has_db` check and DB settings display)

---

### Change 4: `site` (string) + `site_cache` (duration) → `site` section

`site` and `site_cache` are logically paired — `site_cache` only applies when `site` is set. Making `site` a section groups them and future-proofs for additional site-mode options.

**Before:**
```yaml
site: ./site
site_cache: 5m
```

**After:**
```yaml
site:
  path: ./site
  cache: 5m
```

**Go struct:** Replace `Site string` + `SiteCache time.Duration` fields on `Config` with `Site SiteConfig`:

```go
type SiteConfig struct {
    Path  string        `yaml:"path"`  // Directory for filesystem-based routing
    Cache time.Duration `yaml:"cache"` // Response cache TTL (0 = no cache)
}
```

All references to `cfg.Site` become `cfg.Site.Path`. All references to `cfg.SiteCache` become `cfg.Site.Cache`.

**Affected files:**
- `server/config/config.go` — replace `Site string` + `SiteCache time.Duration` with `Site SiteConfig`, add struct
- `server/config/config_test.go` — update tests
- `server/config/load.go` — path resolution (`cfg.Site` → `cfg.Site.Path`), validation (`cfg.Site != ""` → `cfg.Site.Path != ""`), warnings (same)
- `server/config/load_test.go` — tests referencing site config
- `server/devtools.go` — reads `cfg.Site` and `cfg.SiteCache` in `createDevToolsEnv()`
- `server/site.go` — reads `h.server.config.SiteCache` in `serveWithHandler()`
- `server/site_test.go` — sets `cfg.Site` directly
- `server/server.go` — reads `s.config.Site` in `determineHandlersDir()` and `setupRoutes()`
- `server/watcher.go` — reads `w.server.config.Site` in `collectHandlerDirs()`
- `examples/folder-named-index/basil.yaml` — uses `site: ./site`

---

### Change 5: `DeveloperConfig` field alignment

Fix two naming mismatches in developer profiles so overrides use the same names as what they override.

**Note:** The `DeveloperConfig.SQLite` → `DeveloperConfig.Database` part of this change depends on the `DeveloperDBConfig` struct introduced in Change 3. These must be implemented together in a single step.

| Field | Current YAML | Overrides | Problem | New YAML |
|-------|-------------|-----------|---------|----------|
| `SQLite` | `sqlite` | top-level `sqlite` | Doesn't match new `database.path` | `database: { path: }` |
| `Static` | `static` | `public_dir` | Name doesn't match target; collides with the `static` routes array | `public_dir` |

**Before:**
```yaml
developers:
  sam:
    port: 3001
    sqlite: sam.db
    static: ./sam-public
```

**After:**
```yaml
developers:
  sam:
    port: 3001
    database:
      path: sam.db
    public_dir: ./sam-public
```

**Go struct:**
```go
type DeveloperConfig struct {
    Port      int               `yaml:"port"`
    Database  DeveloperDBConfig `yaml:"database"`
    Handlers  string            `yaml:"handlers"`
    PublicDir string            `yaml:"public_dir"`
    Logging   LoggingConfig     `yaml:"logging"`
}

type DeveloperDBConfig struct {
    Path string `yaml:"path"`
}
```

**Affected files:**
- `server/config/config.go` — replace fields, add `DeveloperDBConfig` struct
- `server/config/load.go` — `ApplyDeveloper()`: `dev.SQLite` → `dev.Database.Path`, `dev.Static` → `dev.PublicDir`
- `server/config/load_test.go` — `TestApplyDeveloper` developer profile tests

---

## What Stays the Same

These top-level keys are already well-structured and require no changes:

| Key | Type | Why it's fine |
|-----|------|--------------|
| `server` | section | Well-structured with `host`, `port`, `https`, `proxy` |
| `auth` | section | Deeply nested but logically organised |
| `session` | section | Good as-is (only Go field name fix, YAML unchanged) |
| `security` | section | Good as-is |
| `cors` | section | Good after `max_age` fix |
| `compression` | section | Good as-is |
| `logging` | section | Good as-is |
| `git` | section | Good as-is |
| `dev` | section | Good as-is |
| `meta` | map | Good as-is |
| `developers` | map | Good after field alignment |
| `routes` | list | Major feature, fine at top level |
| `static` | list | Major feature, fine at top level |
| `public_dir` | string | Cross-cutting (routes + asset pipeline), fine at top level |

---

## Final Top-Level Key Inventory (After All Changes)

| Key | Type | Category |
|-----|------|----------|
| `server` | section | Infrastructure |
| `database` | section | Infrastructure |
| `site` | section | Routing (filesystem-based) |
| `routes` | list | Routing (explicit) |
| `static` | list | Routing (static files) |
| `public_dir` | string | Assets |
| `auth` | section | Auth & Sessions |
| `session` | section | Auth & Sessions |
| `security` | section | Security |
| `cors` | section | Security |
| `compression` | section | Performance |
| `logging` | section | Observability |
| `git` | section | Features |
| `dev` | section | Development |
| `developers` | map | Development |
| `meta` | map | User data |

16 top-level keys. Everything is a noun. Related things are grouped. Nothing is orphaned.

---

## Acceptance Criteria

- [ ] All YAML keys use `snake_case` — no camelCase anywhere
- [ ] `cors.max_age` replaces `cors.maxAge`
- [ ] `database.path` replaces top-level `sqlite`
- [ ] `site.path` and `site.cache` replace top-level `site` and `site_cache`
- [ ] `SessionConfig.HTTPOnly` replaces `HttpOnly` in Go (YAML tag `http_only` unchanged)
- [ ] Developer profile fields match main config names (`public_dir`, `database.path`)
- [ ] All example configs updated (`examples/*/basil.yaml`, `docs/guide/configuration-example.yaml`)
- [ ] All tests pass (`go test ./...`)
- [ ] All existing docs updated (see documentation requirements below)
- [ ] New configuration reference manual page created at `docs/guide/configuration.md`
- [ ] Config loader and validator updated
- [ ] DevTools config display updated

---

## Design Decisions

- **No `driver` field in `database`**: SQLite is the only supported embedded database and there are no plans to change this. Adding a `driver` field would be over-engineering.
- **`site.path` not `site.dir`**: Consistency with `database.path` trumps the pedantic argument that it is always a directory. One naming convention everywhere.
- **`public_dir` stays top-level**: It is cross-cutting — used by both routes and the asset pipeline. Nesting it under `server:` would be wrong. It is also one of the first things set in a new project, so it belongs somewhere easy to find.
- **`routes` and `static` stay top-level**: These are major config sections that can be very long. Nesting them under a `routing:` parent adds indentation for no benefit.
- **Clean break, no deprecation**: This is pre-1.0. We break things now so we never have to break them again.
- **`DeveloperDBConfig` is separate from `DatabaseConfig`**: Developer profiles only override `path`. If `DatabaseConfig` ever grows tuning knobs they should not automatically appear as overridable developer options.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Implementation Order

Each step is independently compilable and testable. Run `go test ./...` after each step before committing.

### Step 1: `cors.maxAge` → `cors.max_age`
- Change YAML tag on `CORSConfig.MaxAge` in `server/config/config.go`
- Update test YAML in `server/config/config_test.go`
- Update `examples/cors/basil.yaml`
- Commit: `feat(config)!: rename cors.maxAge to cors.max_age`

### Step 2: `SessionConfig.HttpOnly` → `HTTPOnly`
- Rename field in `server/config/config.go` and its `Defaults()` initialiser
- Update `server/session.go`: two usages of `s.config.HttpOnly`
- Update `server/session_test.go`: struct literal `HttpOnly: true`
- Update `server/config/config_test.go` if it references the field
- YAML tag `http_only` is unchanged — no `basil.yaml` files affected
- Commit: `fix(config): rename SessionConfig.HttpOnly to HTTPOnly per Go conventions`

### Step 3: `sqlite` → `database.path` (includes DeveloperConfig.SQLite)
- Add `DatabaseConfig` and `DeveloperDBConfig` structs to `server/config/config.go`
- Replace `SQLite string` with `Database DatabaseConfig` on `Config`
- Replace `SQLite string` with `Database DeveloperDBConfig` on `DeveloperConfig`
- Update `Defaults()` — no default database path (empty = no database configured)
- Update `server/config/load.go`:
  - Path resolution block: `cfg.SQLite` → `cfg.Database.Path`
  - `ApplyDeveloper()`: `dev.SQLite` → `dev.Database.Path`
- Update all consumers:
  - `server/server.go`
  - `server/devtools.go` (six occurrences — check all of them)
- Update tests:
  - `server/config/config_test.go`
  - `server/config/load_test.go` (`TestLoadSQLiteConfig`, `TestApplyDeveloper` SQLite cases)
  - `server/database_test.go` (four test functions set `cfg.SQLite` directly)
- Commit: `feat(config)!: move sqlite to database.path section`

### Step 4: `site` + `site_cache` → `site.path` + `site.cache`
- Add `SiteConfig` struct to `server/config/config.go`
- Replace `Site string` + `SiteCache time.Duration` with `Site SiteConfig` on `Config`
- Update `server/config/load.go`:
  - Path resolution: `cfg.Site` → `cfg.Site.Path`
  - `validateBasic()`: `cfg.Site != ""` → `cfg.Site.Path != ""`
  - `Warnings()`: same
- Update all consumers:
  - `server/site.go`: `config.SiteCache` → `config.Site.Cache`
  - `server/server.go`: `s.config.Site` → `s.config.Site.Path` (two locations)
  - `server/watcher.go`: `config.Site` → `config.Site.Path`
  - `server/devtools.go`: `cfg.Site` and `cfg.SiteCache`
- Update tests:
  - `server/site_test.go`: `cfg.Site = siteDir` → `cfg.Site.Path = siteDir`
  - `server/config/load_test.go`: any site-related tests
- Commit: `feat(config)!: move site and site_cache into site section`

### Step 5: `DeveloperConfig.Static` → `DeveloperConfig.PublicDir`
- Rename `Static` field to `PublicDir` in `server/config/config.go`, change YAML tag to `public_dir`
- Update `ApplyDeveloper()` in `server/config/load.go`: `dev.Static` → `dev.PublicDir`
- Update `server/config/load_test.go`: `TestApplyDeveloper` static/public_dir cases
- Commit: `feat(config)!: rename developer profile static to public_dir for consistency`

### Step 6: Documentation sweep + new reference page
- Update all docs referencing changed keys (see table below)
- Create `docs/guide/configuration.md` (see structure below)
- Update `docs/guide/configuration-example.yaml` — full rewrite
- Commit: `docs(config): add configuration reference manual and update all docs for 1.0 schema`

---

## Documentation Requirements

### Files to Update

| File | What to change |
|------|---------------|
| `docs/guide/basil-quick-start.md` | Database section: `sqlite:` → `database:\n  path:`; CORS section and table: `maxAge` → `max_age` |
| `docs/guide/cors.md` | All `maxAge` occurrences → `max_age` (15+ places: config examples, tables, pattern examples, complete example at end) |
| `docs/guide/authentication.md` | Site mode example: `site: ./site` → `site:\n  path: ./site` |
| `docs/guide/configuration-example.yaml` | Full rewrite — `sqlite:` → `database: path:`, `developers` section `sqlite:`/`static:` fields, ensure all keys are present and correct |
| `examples/cors/basil.yaml` | `maxAge: 86400` → `max_age: 86400` |
| `examples/folder-named-index/basil.yaml` | `site: ./site` → `site:\n  path: ./site` |
| `examples/folder-named-index/README.md` | Code block: `site: ./site` → `site:\n  path: ./site` |
| `.github/skills/basil-development/SKILL.md` | Two code examples: `sqlite:` → `database:\n  path:`, `site: ./site` → `site:\n  path: ./site` |
| `.github/skills/basil-development/references/CONFIGURATION.md` | Full update — existing reference has `sqlite:` and `site:` string throughout |
| `.github/skills/basil-development/references/DATABASE.md` | `sqlite: ./myapp.db` example → `database:\n  path: ./myapp.db` |
| `.github/skills/basil-development/references/TESTING.md` | `sqlite: ./test.db` example → `database:\n  path: ./test.db` |

### New Doc: `docs/guide/configuration.md`

This is the single source of truth for every config key. Structure:

1. **Overview** — what `basil.yaml` is, where Basil looks for it, env var interpolation syntax
2. **Complete annotated example** — every key shown with its default, commented out where optional
3. **Reference** — one H2 section per top-level key, each containing:
   - Brief description of the section's purpose
   - Table: key name | type | default | description
   - Minimal usage example
4. **Sections in order:**
   - `server` — `host`, `port`, `https` (sub-keys: `auto`, `email`, `cache_dir`, `cert`, `key`), `proxy` (sub-keys: `trusted`, `trusted_ips`)
   - `database` — `path`
   - `site` — `path`, `cache`
   - `routes` — `path`, `handler`, `auth`, `roles`, `cache`, `public_dir`, `type`
   - `static` — `path`, `root`, `file`
   - `public_dir`
   - `auth` — `enabled`, `registration`, `session_ttl`, `login_path`, `protected_paths`, `email_verification` (all sub-keys), `recovery`
   - `session` — `store`, `secret`, `max_age`, `cookie_name`, `secure`, `http_only`, `same_site`, `table`, `cleanup`
   - `security` — `hsts` (sub-keys), `content_type_options`, `frame_options`, `xss_protection`, `referrer_policy`, `csp`, `permissions_policy`, `allow_write`
   - `cors` — `origins`, `methods`, `headers`, `expose`, `credentials`, `max_age`
   - `compression` — `enabled`, `level`, `min_size`, `zstd`
   - `logging` — `level`, `format`, `output`, `quiet`, `parsley.output`
   - `git` — `enabled`, `require_auth`
   - `dev` — `log_database`, `log_max_size`, `log_truncate_pct`, `cache`
   - `developers` — `port`, `database.path`, `handlers`, `public_dir`, `logging`
   - `meta`
5. **Environment variables** — `${VAR}` and `${VAR:-default}` syntax with examples
6. **Secrets** — `!secret` tag: what it does, `!secret auto`, `!secret ${ENV_VAR}`
7. **Config file resolution** — search order: explicit path → `BASIL_CONFIG` env var → `./basil.yaml` → `~/.config/basil/basil.yaml`
8. **Migrating from pre-1.0** — table of every changed key:

   | Pre-1.0 | 1.0 |
   |---------|-----|
   | `sqlite: ./data.db` | `database:\n  path: ./data.db` |
   | `site: ./site` | `site:\n  path: ./site` |
   | `site_cache: 5m` | `site:\n  cache: 5m` |
   | `cors:\n  maxAge: 86400` | `cors:\n  max_age: 86400` |
   | `developers:\n  x:\n    sqlite: x.db` | `developers:\n  x:\n    database:\n      path: x.db` |
   | `developers:\n  x:\n    static: ./dir` | `developers:\n  x:\n    public_dir: ./dir` |

---

## Edge Cases & Constraints

1. **`site.path` and `routes` are mutually exclusive** — validation logic in `validateBasic()` moves from `cfg.Site != ""` to `cfg.Site.Path != ""`. The error message should remain unchanged.
2. **`database.path` empty = no database** — same semantics as old `sqlite: ""`. No default path is set in `Defaults()`.
3. **Developer `database.path` path resolution** — `ApplyDeveloper()` must join with `cfg.BaseDir` just as it does for the main `database.path`, and must handle both relative and absolute paths.
4. **`site` being a section breaks configs that set `site: ./site`** — YAML will reject this as a type mismatch (string vs mapping). The error message from the YAML parser may be confusing; the migration doc must be clear.
5. **`Warnings()` no-routes check** — currently `cfg.Site == ""` is part of the condition that warns about no routes. This must change to `cfg.Site.Path == ""` or the warning will fire incorrectly in site mode after the change.

## Related

- Ship review: `work/reports/1.0-SHIP-REVIEW.md` § 3
- Plan: `work/plans/PLAN-118-feat-138-config-consistency.md`
