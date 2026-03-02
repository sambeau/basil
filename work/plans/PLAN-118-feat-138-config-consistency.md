---
id: PLAN-118
feature: FEAT-138
title: "Implementation Plan: Config YAML Consistency for 1.0"
status: draft
created: 2026-03-02
---

# Implementation Plan: FEAT-138 Config YAML Consistency for 1.0

## Overview

Five targeted changes to the `basil.yaml` schema, plus a new configuration reference manual page. This is a clean break — no backward compatibility. All changes are independent of other in-flight features.

**Note on scope:** `maxAge`, `httpOnly`, `sameSite` etc. also appear as Parsley-level cookie property names and `<Cache>` tag attributes throughout `docs/parsley/` and `docs/basil/`. Those are runtime API names, not YAML config keys — they are deliberately **not** changed by this plan. Similarly, `work/design/` documents are historical snapshots and are not updated.

**Spec:** `work/specs/FEAT-138.md`

## Prerequisites

- [ ] Clean working tree (`git status` — no uncommitted changes)
- [ ] All tests pass before starting (`go test ./...`)
- [ ] On `main` branch — create feature branch before starting

## Branch

```
feat/FEAT-138-config-consistency
```

---

## Step 1: `cors.maxAge` → `cors.max_age`

**Scope:** One YAML tag change. Smallest possible change — good warm-up.

### Tasks

- [ ] In `server/config/config.go`: change `CORSConfig.MaxAge` YAML tag from `maxAge` to `max_age`
- [ ] In `server/config/config_test.go`: update any test YAML strings using `maxAge:` and any assertions referencing the default value check comment
- [ ] In `examples/cors/basil.yaml`: rename `maxAge: 86400` → `max_age: 86400`

### Verify

```bash
go test ./server/config/...
```

Check that the CORS test for default `maxAge` value still passes (field name unchanged, only YAML tag changed).

### Commit

```
feat(config)!: rename cors.maxAge to cors.max_age
```

---

## Step 2: `SessionConfig.HttpOnly` → `HTTPOnly` (Go only)

**Scope:** Go struct field rename only. No YAML files change (`http_only` tag is already correct).

### Tasks

- [ ] In `server/config/config.go`:
  - Rename `HttpOnly bool` → `HTTPOnly bool` on `SessionConfig`
  - Update `Defaults()` initialiser: `HttpOnly: true` → `HTTPOnly: true`
- [ ] In `server/session.go`: rename both usages of `s.config.HttpOnly` → `s.config.HTTPOnly`
  - In `Save()` — sets `cookie.HttpOnly`
  - In `Clear()` — sets `cookie.HttpOnly`
- [ ] In `server/session_test.go`: update struct literal `HttpOnly: true` → `HTTPOnly: true` in `testSessionConfig()`
- [ ] In `server/config/config_test.go`: update any struct literals that set this field

### Verify

```bash
go test ./server/...
```

### Commit

```
fix(config): rename SessionConfig.HttpOnly to HTTPOnly per Go conventions
```

---

## Step 3: `sqlite` → `database.path`

**Scope:** New struct, updated loader, seven consumer locations across four files. Includes the `DeveloperConfig.SQLite` half of Change 5 — the two must be done together because `DeveloperDBConfig` is introduced here.

### Tasks

#### 3a. Structs (`server/config/config.go`)

- [ ] Add new structs after `CORSConfig`:
  ```go
  // DatabaseConfig holds database settings
  type DatabaseConfig struct {
      Path string `yaml:"path"` // Path to SQLite database file
  }

  // DeveloperDBConfig holds the database override for a developer profile
  type DeveloperDBConfig struct {
      Path string `yaml:"path"`
  }
  ```
- [ ] Replace `SQLite string \`yaml:"sqlite"\`` on `Config` with `Database DatabaseConfig \`yaml:"database"\``
- [ ] Replace `SQLite string \`yaml:"sqlite"\`` on `DeveloperConfig` with `Database DeveloperDBConfig \`yaml:"database"\``
- [ ] `Defaults()` — remove any SQLite default (there is none; just confirm `Database` zero value is correct)

#### 3b. Loader (`server/config/load.go`)

- [ ] Path resolution block: `cfg.SQLite` → `cfg.Database.Path` (the `if cfg.SQLite != "" && !filepath.IsAbs(cfg.SQLite)` block)
- [ ] `ApplyDeveloper()`: `dev.SQLite` → `dev.Database.Path` (the `if dev.SQLite != ""` block, including the path join logic)

#### 3c. Server (`server/server.go`)

- [ ] `initDatabase()`: replace `s.config.SQLite == ""` check → `s.config.Database.Path == ""`
- [ ] `initDatabase()`: replace `s.initSQLite(s.config.SQLite)` → `s.initSQLite(s.config.Database.Path)`

#### 3d. DevTools (`server/devtools.go`)

Update all seven occurrences — do not miss any:

- [ ] `openAppDB()`: `h.server.config.SQLite` → `h.server.config.Database.Path`
- [ ] `openAppDB()`: update error message `"no database configured (set sqlite in config)"` → `"no database configured (set database.path in config)"`
- [ ] `serveDB()`: `h.server.config.SQLite` → `h.server.config.Database.Path`
- [ ] `handleDevDBFileDownload()`: `h.server.config.SQLite` → `h.server.config.Database.Path`
- [ ] `handleDevDBFileUpload()`: `h.server.config.SQLite` → `h.server.config.Database.Path`
- [ ] `createDevToolsEnv()` — index page `has_db` check: `h.server.config.SQLite != ""` → `h.server.config.Database.Path != ""`
- [ ] `createDevToolsEnv()` — DB overview page block: `h.server.config.SQLite` → `h.server.config.Database.Path`
- [ ] `createDevToolsEnv()` — DB settings display: `setting("SQLite", cfg.SQLite, ...)` → `setting("Database", cfg.Database.Path, ...)`

#### 3e. Tests

- [ ] `server/config/config_test.go`: update any YAML strings with `sqlite:` and struct literals with `.SQLite`
- [ ] `server/config/load_test.go`:
  - `TestLoadSQLiteConfig`: YAML strings, `cfg.SQLite` assertions → `cfg.Database.Path`
  - `TestApplyDeveloper` SQLite cases: struct literals `{SQLite: "sam.db"}` → `{Database: DeveloperDBConfig{Path: "sam.db"}}`, assertions `cfg.SQLite` → `cfg.Database.Path`
- [ ] `server/database_test.go`: all four test functions that set `cfg.SQLite = ...` → `cfg.Database.Path = ...`

### Verify

```bash
go test ./...
```

### Commit

```
feat(config)!: move sqlite to database.path section
```

---

## Step 4: `site` + `site_cache` → `site.path` + `site.cache`

**Scope:** New struct, updated loader and validator, five consumer files.

### Tasks

#### 4a. Struct (`server/config/config.go`)

- [ ] Add new struct:
  ```go
  // SiteConfig holds filesystem-based routing settings
  type SiteConfig struct {
      Path  string        `yaml:"path"`  // Directory for filesystem-based routing
      Cache time.Duration `yaml:"cache"` // Response cache TTL (0 = no cache)
  }
  ```
- [ ] Replace `Site string \`yaml:"site"\`` and `SiteCache time.Duration \`yaml:"site_cache"\`` on `Config` with `Site SiteConfig \`yaml:"site"\``
- [ ] `Defaults()` — remove any `Site` or `SiteCache` defaults (both were zero values; confirm no change needed)

#### 4b. Loader (`server/config/load.go`)

- [ ] Path resolution block: `cfg.Site` → `cfg.Site.Path` (the `if cfg.Site != "" && !filepath.IsAbs(cfg.Site)` block)
- [ ] `validateBasic()`: `cfg.Site != ""` → `cfg.Site.Path != ""` (the site+routes mutual exclusion check)
- [ ] `Warnings()`: `cfg.Site == ""` → `cfg.Site.Path == ""` (the no-routes-configured warning condition)

#### 4c. Server (`server/server.go`)

- [ ] `determineHandlersDir()`: `s.config.Site != ""` and `s.config.Site` usages → `s.config.Site.Path`
- [ ] `setupRoutes()`: `s.config.Site != ""` and `s.config.Site` usages → `s.config.Site.Path`

#### 4d. Watcher (`server/watcher.go`)

- [ ] `collectHandlerDirs()`: `w.server.config.Site != ""` and `w.server.config.Site` → `w.server.config.Site.Path`

#### 4e. Site handler (`server/site.go`)

- [ ] `serveWithHandler()`: `h.server.config.SiteCache` → `h.server.config.Site.Cache`

#### 4f. DevTools (`server/devtools.go`)

- [ ] `createDevToolsEnv()`: `cfg.Site != ""` → `cfg.Site.Path != ""`, `cfg.Site` display → `cfg.Site.Path`, `cfg.SiteCache` → `cfg.Site.Cache`

#### 4g. Tests

- [ ] `server/site_test.go`: `cfg.Site = siteDir` → `cfg.Site.Path = siteDir`
- [ ] `server/config/load_test.go`: any site config tests — update YAML strings and field assertions

### Verify

```bash
go test ./...
```

### Commit

```
feat(config)!: move site and site_cache into site section
```

---

## Step 5: `DeveloperConfig.Static` → `DeveloperConfig.PublicDir`

**Scope:** Field rename only. `DeveloperConfig.SQLite` was already migrated in Step 3.

### Tasks

#### 5a. Struct (`server/config/config.go`)

- [ ] Rename `Static string \`yaml:"static"\`` → `PublicDir string \`yaml:"public_dir"\`` on `DeveloperConfig`

#### 5b. Loader (`server/config/load.go`)

- [ ] `ApplyDeveloper()`: rename `dev.Static` → `dev.PublicDir` in the static override block (the `if dev.Static != ""` block and all references within it)

#### 5c. Tests (`server/config/load_test.go`)

- [ ] Update `TestApplyDeveloper` static/public_dir cases:
  - Struct literals `{Static: "sam-public"}` → `{PublicDir: "sam-public"}`
  - Any assertion comments referencing the field name

### Verify

```bash
go test ./...
```

### Commit

```
feat(config)!: rename developer profile static to public_dir for consistency
```

---

## Step 6: Documentation sweep + new reference page

**Scope:** Update every file referencing changed keys. Create `docs/guide/configuration.md`.

### Tasks

#### 6a. Update existing docs

- [ ] **`examples/cors/basil.yaml`**: `maxAge: 86400` → `max_age: 86400`

- [ ] **`examples/cors/README.md`**: three occurrences of `maxAge` — YAML example, section heading, and inline example

- [ ] **`examples/folder-named-index/basil.yaml`**: `site: ./site` → `site:\n  path: ./site`

- [ ] **`examples/folder-named-index/README.md`**: code block `site: ./site` → `site:\n  path: ./site`

- [ ] **`docs/guide/configuration-example.yaml`**: full rewrite
  - `sqlite: ./db/data.db` comment → `database:\n  path: ./db/data.db`
  - `developers` section: `sqlite: sam.db` → `database:\n      path: sam.db`, `# static: sam-public` → `# public_dir: ./sam-public`
  - Confirm no `maxAge`, `site_cache`, or top-level `site: string` remain

- [ ] **`docs/guide/basil-quick-start.md`**:
  - Database Support section: `sqlite: ./data.db` → `database:\n  path: ./data.db`
  - CORS section code examples: `maxAge:` → `max_age:`
  - CORS configuration options table: `maxAge` column header and description → `max_age`

- [ ] **`docs/guide/cors.md`**:
  - All `maxAge:` occurrences in YAML examples → `max_age:`
  - Configuration reference table entry for `maxAge` → `max_age`
  - All prose references to `` `maxAge` `` → `` `max_age` ``
  - (15+ occurrences total — grep for `maxAge` in this file to find all)

- [ ] **`docs/guide/authentication.md`**:
  - Site mode example: `site: ./site` → `site:\n  path: ./site`

- [ ] **`.github/skills/basil-development/SKILL.md`**:
  - Both code examples: `sqlite: ./data.db` → `database:\n  path: ./data.db`, `site: ./site` → `site:\n  path: ./site`

- [ ] **`.github/skills/basil-development/references/CONFIGURATION.md`**:
  - Full update — this file has `sqlite:` and `site:` as string throughout
  - All `sqlite:` occurrences → `database:\n  path:`
  - All `site: ./site` occurrences → `site:\n  path: ./site`

- [ ] **`.github/skills/basil-development/references/DATABASE.md`**:
  - `sqlite: ./myapp.db` → `database:\n  path: ./myapp.db`

- [ ] **`.github/skills/basil-development/references/TESTING.md`**:
  - `sqlite: ./test.db` → `database:\n  path: ./test.db`

#### 6b. Create `docs/guide/configuration.md`

Create the comprehensive configuration reference manual page. This is the permanent single source of truth — write it as if it will be read thousands of times.

**Structure:**

```
# Configuration Reference

## Overview
- What basil.yaml is
- Config file resolution order (explicit path → BASIL_CONFIG → ./basil.yaml → ~/.config/basil/basil.yaml)
- Environment variable interpolation: ${VAR} and ${VAR:-default}
- The !secret tag

## Complete Example
Full annotated basil.yaml showing every key, with defaults indicated,
optional sections commented out

## Reference

### server
Table: host, port + sub-sections https and proxy (each as sub-table)

### database
Table: path

### site
Table: path, cache
Note: mutually exclusive with routes

### routes
Table: path, handler, auth, roles, cache, public_dir, type
Note: mutually exclusive with site

### static
Table: path, root, file

### public_dir
Inline description (single string key)

### auth
Table: enabled, registration, session_ttl, login_path, protected_paths
Sub-section: email_verification (full table of all sub-keys)
Sub-section: recovery

### session
Table: store, secret, max_age, cookie_name, secure, http_only, same_site
Sub-section: SQLite session store options (table, cleanup)

### security
Sub-section: hsts (table of sub-keys)
Table: content_type_options, frame_options, xss_protection,
       referrer_policy, csp, permissions_policy, allow_write

### cors
Table: origins, methods, headers, expose, credentials, max_age

### compression
Table: enabled, level, min_size, zstd

### logging
Table: level, format, output, quiet
Sub-section: parsley (table: output)

### git
Table: enabled, require_auth

### dev
Table: log_database, log_max_size, log_truncate_pct, cache

### developers
Structure explanation + table of override fields:
port, database.path, handlers, public_dir, logging

### meta
Explanation of arbitrary key/value map, example

## Migrating from pre-1.0
Table of every changed key: old → new
```

### Verify

Re-run grep to confirm no stale key names remain in docs or examples:

```bash
# Config YAML keys that should no longer exist
grep -rn 'maxAge\|site_cache' docs/guide/ examples/ .github/skills/

# sqlite as a YAML key (not prose mentions of SQLite the database)
grep -rn '^\s*sqlite:' docs/guide/ examples/ .github/skills/

# Old developer profile keys
grep -rn 'static:.*public\|sqlite:.*\.db' docs/guide/configuration-example.yaml
```

Each command should return zero matches. Note: `maxAge` will still appear in `docs/parsley/`, `docs/basil/reference.md`, and Parsley test files — those are runtime API names and are correct.

### Commit

```
docs(config): add configuration reference manual and update all docs for 1.0 schema
```

---

## Final Verification

Run the full test suite one last time from the repo root:

```bash
go test ./...
```

Then build to confirm no compilation errors:

```bash
make build
```

Then confirm no stale YAML config keys remain in user-facing docs:

```bash
grep -rn 'maxAge\|site_cache' docs/guide/ examples/ .github/skills/
grep -rn '^\s*sqlite:' docs/guide/ examples/ .github/skills/
```

Both should return zero matches. (`maxAge` in `docs/parsley/` and `docs/basil/` is intentionally unchanged — those are Parsley runtime property names.)

---

## Progress Log

| Step | Status | Commit | Notes |
|------|--------|--------|-------|
| 1. `cors.max_age` | ⬜ not started | — | — |
| 2. `HTTPOnly` | ⬜ not started | — | — |
| 3. `database.path` | ⬜ not started | — | — |
| 4. `site` section | ⬜ not started | — | — |
| 5. `public_dir` | ⬜ not started | — | — |
| 6. Docs | ⬜ not started | — | — |

## Related

- Spec: `work/specs/FEAT-138.md`
- Ship review: `work/reports/1.0-SHIP-REVIEW.md` § 3