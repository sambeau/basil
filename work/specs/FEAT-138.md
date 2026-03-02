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
- `server/cors.go` — reads `m.config.MaxAge` (Go unchanged, just YAML)
- `examples/cors/basil.yaml` — uses `maxAge: 86400`
- `docs/guide/basil-quick-start.md` — CORS section references `maxAge`
- `docs/guide/cors.md` — references `maxAge` throughout
- `docs/guide/configuration-example.yaml` — not currently shown but may be added

### Change 2: `SessionConfig.HttpOnly` → `HTTPOnly` (Go only)

Go convention: HTTP is an initialism, so the struct field should be `HTTPOnly`. The YAML tag `http_only` is already correct and does **not** change.

**Go struct:** `SessionConfig.HttpOnly` → `SessionConfig.HTTPOnly`

**Affected files:**
- `server/config/config.go` — field rename
- `server/config/config_test.go` — if any tests reference the field by name
- Any Go code referencing `cfg.Session.HttpOnly` → `cfg.Session.HTTPOnly`

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

**Affected files:**
- `server/config/config.go` — replace field, add struct
- `server/config/config_test.go` — update test YAML and assertions
- `server/config/load.go` — path resolution (`cfg.SQLite` → `cfg.Database.Path`)
- `server/config/load_test.go` — all SQLite config tests
- `server/database_test.go` — sets `cfg.SQLite`
- `server/devtools.go` — reads `cfg.SQLite`
- `server/server.go` — reads `cfg.SQLite` for init
- `docs/guide/basil-quick-start.md` — Database Support section
- `docs/guide/configuration-example.yaml` — `sqlite:` comment
- Example configs that reference `sqlite:`

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
- `server/config/config.go` — replace fields, add struct
- `server/config/config_test.go` — update tests
- `server/config/load.go` — path resolution, validation (`cfg.Site` → `cfg.Site.Path`), warnings
- `server/config/load_test.go` — tests referencing site config
- `server/devtools.go` — reads `cfg.Site` and `cfg.SiteCache`
- `server/site.go` — reads `cfg.SiteCache`
- `server/site_test.go` — sets `cfg.Site`
- `examples/folder-named-index/basil.yaml` — uses `site: ./site`
- `docs/guide/basil-quick-start.md` — if site mode mentioned
- `docs/guide/authentication.md` — site mode example

### Change 5: `DeveloperConfig` field alignment

Fix two naming mismatches in developer profiles so overrides use the same names as what they override.

| Field | Current YAML | Overrides | Problem | New YAML |
|-------|-------------|-----------|---------|----------|
| `SQLite` | `sqlite` | top-level `sqlite` | Doesn't match new `database.path` | `database: { path: }` |
| `Static` | `static` | `public_dir` | Name doesn't match target; collides with `static` routes | `public_dir` |

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
    Port      int                `yaml:"port"`
    Database  DeveloperDBConfig  `yaml:"database"`
    Handlers  string             `yaml:"handlers"`
    PublicDir string             `yaml:"public_dir"`
    Logging   LoggingConfig      `yaml:"logging"`
}

type DeveloperDBConfig struct {
    Path string `yaml:"path"`
}
```

**Affected files:**
- `server/config/config.go` — replace fields, add struct
- `server/config/load.go` — `ApplyDeveloper()` logic
- `server/config/load_test.go` — developer profile tests
- `docs/guide/configuration-example.yaml` — developer profiles section

## What Stays the Same

These top-level keys are already well-structured and require no changes:

| Key | Type | Why it's fine |
|-----|------|--------------|
| `server` | section | Well-structured with `host`, `port`, `https`, `proxy` |
| `auth` | section | Deeply nested but logically organized |
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

## Acceptance Criteria

- [ ] All YAML keys use `snake_case` (no camelCase anywhere)
- [ ] `cors.max_age` replaces `cors.maxAge`
- [ ] `database.path` replaces top-level `sqlite`
- [ ] `site.path` and `site.cache` replace top-level `site` and `site_cache`
- [ ] `SessionConfig.HTTPOnly` replaces `HttpOnly` in Go (YAML unchanged)
- [ ] Developer profile fields match main config names (`public_dir`, `database.path`)
- [ ] All example configs updated (`examples/*/basil.yaml`, `docs/guide/configuration-example.yaml`)
- [ ] All tests pass (`go test ./...`)
- [ ] All existing docs updated (quick-start, CORS guide, auth guide, etc.)
- [ ] New configuration reference manual page created at `docs/guide/configuration.md`
- [ ] Config loader and validator updated
- [ ] DevTools config display updated

## Design Decisions

- **No `driver` field in `database`**: SQLite is the only supported embedded database and there are no plans to change this. Adding a `driver` field would be over-engineering.
- **`site.path` not `site.dir`**: Consistency with `database.path` trumps the pedantic argument that it's always a directory. One naming convention everywhere.
- **`public_dir` stays top-level**: It's cross-cutting (used by routes and the asset pipeline). Nesting it under `server:` would be wrong. It's also one of the first things you set in a new project.
- **`routes` and `static` stay top-level**: They're major config sections that can be very long. Nesting them under a `routing:` parent adds indentation for no benefit.
- **Clean break, no deprecation**: This is pre-1.0. We break things now so we never have to break them again.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Implementation Order

Each step is independently compilable and testable. Run `go test ./...` after each.

### Step 1: `cors.maxAge` → `cors.max_age`
- Change YAML tag on `CORSConfig.MaxAge` in `server/config/config.go`
- Update test YAML in `server/config/config_test.go`
- Update `examples/cors/basil.yaml`
- Commit: `feat(config)!: rename cors.maxAge to cors.max_age`

### Step 2: `SessionConfig.HttpOnly` → `HTTPOnly`
- Rename Go struct field in `server/config/config.go`
- Find and update all Go references (`grep -r 'HttpOnly' server/`)
- YAML tag `http_only` is unchanged — no config file impact
- Commit: `fix(config): rename SessionConfig.HttpOnly to HTTPOnly per Go conventions`

### Step 3: `sqlite` → `database.path`
- Create `DatabaseConfig` struct in `server/config/config.go`
- Replace `SQLite string` with `Database DatabaseConfig` on `Config`
- Create `DeveloperDBConfig` struct, replace `SQLite string` on `DeveloperConfig`
- Update `Defaults()` — no default database path (empty = no database)
- Update `server/config/load.go`:
  - Path resolution: `cfg.SQLite` → `cfg.Database.Path`
  - Secret tracking if needed
- Update `ApplyDeveloper()` in `server/config/load.go`
- Update all consumers: `server/server.go`, `server/devtools.go`, `server/database_test.go`
- Update all tests in `server/config/load_test.go`
- Update `examples/auth/basil.yaml` if it uses sqlite
- Update `docs/guide/basil-quick-start.md` Database Support section
- Update `docs/guide/configuration-example.yaml`
- Commit: `feat(config)!: move sqlite to database.path section`

### Step 4: `site` + `site_cache` → `site.path` + `site.cache`
- Create `SiteConfig` struct in `server/config/config.go`
- Replace `Site string` + `SiteCache time.Duration` with `Site SiteConfig` on `Config`
- Update `server/config/load.go`:
  - Path resolution: `cfg.Site` → `cfg.Site.Path`
  - Validation: `cfg.Site != ""` → `cfg.Site.Path != ""`
  - Warnings: same change
- Update consumers: `server/site.go`, `server/devtools.go`
- Update tests: `server/site_test.go`, `server/config/load_test.go`
- Update `examples/folder-named-index/basil.yaml`
- Update `docs/guide/authentication.md` site mode example
- Commit: `feat(config)!: move site and site_cache into site section`

### Step 5: `DeveloperConfig.Static` → `DeveloperConfig.PublicDir`
- Rename field in `server/config/config.go`, change YAML tag to `public_dir`
- Update `ApplyDeveloper()` in `server/config/load.go`
- Update tests in `server/config/load_test.go`
- Update `docs/guide/configuration-example.yaml` developer profiles
- Commit: `feat(config)!: rename developer static to public_dir for consistency`

### Step 6: Documentation
- Update all docs that reference changed config keys (see docs list below)
- Create `docs/guide/configuration.md` — full configuration reference manual page
- Update `docs/guide/configuration-example.yaml` to reflect all changes
- Commit: `docs(config): add configuration reference and update all docs for new schema`

## Documentation Requirements

### Docs to Update

| File | What to change |
|------|---------------|
| `docs/guide/basil-quick-start.md` | `sqlite:` → `database: path:`, `maxAge` → `max_age`, CORS table |
| `docs/guide/cors.md` | All `maxAge` references → `max_age` (appears 15+ times) |
| `docs/guide/authentication.md` | Site mode example: `site:` string → `site: path:` |
| `docs/guide/configuration-example.yaml` | Full rewrite to match new schema |
| `examples/cors/basil.yaml` | `maxAge` → `max_age` |
| `examples/folder-named-index/basil.yaml` | `site: ./site` → `site: path: ./site` |

### New Doc: `docs/guide/configuration.md`

Create a comprehensive configuration reference manual page. This is the single source of truth for every config key. Structure:

1. **Overview** — what `basil.yaml` is, where it's loaded from, env var interpolation
2. **Complete Example** — full annotated config showing every key with defaults
3. **Reference** — every section documented with:
   - Key name and type
   - Default value
   - Description
   - Example
4. **Sections** (one heading per top-level key):
   - `server` — host, port, https, proxy
   - `database` — path
   - `site` — path, cache
   - `routes` — path, handler, auth, roles, cache, public_dir, type
   - `static` — path, root, file
   - `public_dir`
   - `auth` — enabled, registration, session_ttl, login_path, protected_paths, email_verification, recovery
   - `session` — store, secret, max_age, cookie_name, secure, http_only, same_site, table, cleanup
   - `security` — hsts, content_type_options, frame_options, xss_protection, referrer_policy, csp, permissions_policy, allow_write
   - `cors` — origins, methods, headers, expose, credentials, max_age
   - `compression` — enabled, level, min_size, zstd
   - `logging` — level, format, output, quiet, parsley.output
   - `git` — enabled, require_auth
   - `dev` — log_database, log_max_size, log_truncate_pct, cache
   - `developers` — port, database.path, handlers, public_dir, logging
   - `meta`
5. **Environment Variables** — `${VAR}` and `${VAR:-default}` syntax
6. **Secrets** — `!secret` tag usage
7. **Config File Resolution** — search order (explicit path → `BASIL_CONFIG` → `./basil.yaml` → `~/.config/basil/basil.yaml`)
8. **Migration from Pre-1.0** — table of old key → new key for users upgrading

### Edge Cases & Constraints

1. `site.path` and `routes` remain mutually exclusive — validation logic moves from `cfg.Site != ""` to `cfg.Site.Path != ""`
2. `database.path` empty string means no database — same semantics as old `sqlite: ""`
3. Developer profile `database.path` resolution must join with `cfg.BaseDir` just like the main `database.path`
4. The `DeveloperDBConfig` struct is separate from `DatabaseConfig` in case they diverge in the future (developer profiles only override `path`, not hypothetical future tuning knobs)

## Related

- Ship review: `work/reports/1.0-SHIP-REVIEW.md` § 3
- Plan: `work/plans/PLAN-FEAT-138.md` (to be created)