# Plan: Per-Route public_dir

**Related to:** FEAT-016 (refinement)

## Overview

Move `public_dir` from global config to per-route config, allowing each route to have its own static files directory.

## Current State

```yaml
# Global public_dir (current)
public_dir: ./public

routes:
  - path: /
    handler: ./app/app.pars
  - path: /admin
    handler: ./admin/admin.pars
```

All routes share a single `./public` directory.

## Target State

```yaml
routes:
  - path: /
    handler: ./app/app.pars
    public_dir: ./app/public      # /styles.css → ./app/public/styles.css

  - path: /admin
    handler: ./admin/admin.pars
    public_dir: ./admin/public    # /admin/styles.css → ./admin/public/styles.css
```

Each route can have its own `public_dir`, or none at all.

## Implementation Steps

### 1. Update Config Types
- Add `PublicDir string` field to `Route` struct
- Keep global `PublicDir` for backward compatibility (applies to "/" route if not specified)
- Update `config/load.go` to resolve per-route paths

### 2. Update Route Handler Setup
- `server/server.go`: Each route with `public_dir` needs static file fallback
- Remove global root handler logic for public_dir
- Each non-root route with `public_dir` gets its own fallback handler

### 3. Update Basil Context Injection
- `server/handler.go`: `buildBasilContext` receives route's `public_dir` instead of global
- `parsleyHandler` already has access to its route

### 4. Static File Serving Logic

For route `/admin` with `public_dir: ./admin/public`:
- Request `/admin/styles.css` → try handler → fallback to `./admin/public/styles.css`
- The route prefix is stripped when looking up static files

For root route `/` with `public_dir: ./app/public`:
- Request `/styles.css` → try handler → fallback to `./app/public/styles.css`

### 5. Backward Compatibility
- Global `public_dir` still works - applies to "/" route as default
- If route has `public_dir`, it overrides global for that route
- Routes without `public_dir` don't serve static files

## Files to Modify

1. `config/config.go` - Add `PublicDir` to Route struct
2. `config/load.go` - Resolve per-route public_dir paths
3. `server/server.go` - Update route registration with per-route static fallback
4. `server/handler.go` - Use route's public_dir in basil context

## Testing

1. Route with public_dir serves its own static files
2. Route without public_dir returns 404 for static paths
3. Multiple routes with different public_dirs work correctly
4. `asset()` function uses correct public_dir per route
5. Backward compat: global public_dir works for "/" route

## Progress Log

- [x] Step 1: Update config types - Added `PublicDir` to Route struct
- [x] Step 2: Update config loading - Resolve per-route paths, apply global as default for "/"
- [x] Step 3: Update server route setup - Added `createRouteWithStaticFallback`
- [x] Step 4: Update handler basil context - Use route's public_dir
- [x] Step 5: Test - All tests pass
