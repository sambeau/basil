---
id: FEAT-041
title: "Public URL for Private Assets"
status: draft
priority: medium
created: 2025-12-07
author: "@human"
---

# FEAT-041: Public URL for Private Assets

## Summary
Add a `publicUrl()` function that makes private files (e.g., component assets in `modules/`) accessible via a public URL. This enables component-local assets (SVGs, icons, CSS) without compromising Basil's simple security model of a single `public/` folder.

## User Story
As a developer building reusable components, I want to keep assets (icons, images) in the same folder as my component code so that components are self-contained and easy to share, while still being able to reference those assets in generated HTML.

## Motivation
JavaScript frameworks (React, Vue, Svelte) commonly co-locate assets with components:
```
components/
├── Button/
│   ├── Button.js
│   └── icon.svg
```

Bundlers like Webpack/Vite handle this by copying assets to the output folder and returning public URLs. Basil doesn't have a build step, so we need a runtime solution that:
1. Keeps the security model simple (one public folder)
2. Allows explicit exceptions via code
3. Doesn't require copying files around

## Acceptance Criteria
- [x] `publicUrl(@./path)` returns a public URL string for the file
- [x] URLs are content-hashed for cache-busting (file changes → URL changes)
- [x] Files are served from original location (no copying)
- [x] Aggressive cache headers: `Cache-Control: public, max-age=31536000, immutable`
- [x] Warning logged for files >10MB
- [x] Error returned for files >100MB (guide users to `public/` folder)
- [x] Registry cleared on server reload (SIGHUP)
- [ ] Works in both `routes:` and `site:` modes (site mode pending FEAT-040)

## Design Decisions

- **Content hash for URLs**: URLs include a hash of file contents (e.g., `/__p/a3f2b1c8.svg`). This provides automatic cache-busting—when a file changes, the hash changes, so browsers fetch the new version. The hash is deterministic, so the same content always produces the same URL.

- **Lazy hashing with cache**: To avoid re-reading large files, we cache the hash along with modTime and size. On subsequent calls, we check if the file has changed before re-hashing. This makes `publicUrl()` fast in steady state.

- **Virtual serving**: Files remain in their original location. Basil maintains an in-memory registry mapping hashes to file paths. No file copying required.

- **Always public (no auth)**: Assets served via `publicUrl()` are always public. This is an explicit choice by the developer. For auth-protected assets, use a handler that checks auth and serves the file.

- **Size limits**: Large files should go in `public/`. We warn at 10MB and error at 100MB to guide developers toward the right pattern.

- **URL prefix `/__p/`**: Short, unlikely to conflict with user routes, clearly identifies public assets.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### API

```parsley
// Basic usage
let iconUrl = publicUrl(@./icon.svg)
<img src={iconUrl}/>
// Output: <img src="/__p/a3f2b1c8.svg"/>

// In a component module (modules/Button.pars)
export Button = fn({label}) {
  let icon = publicUrl(@./button-icon.svg)
  <button>
    <img src={icon} alt=""/>
    {label}
  </button>
}
```

### URL Format
```
/__p/{contentHash}.{extension}

Examples:
/__p/a3f2b1c8.svg
/__p/7d3e9f2a.png
/__p/1b4c8e6d.css
```

### Hash Algorithm
```go
hash := sha256.Sum256(fileContents)
shortHash := hex.EncodeToString(hash[:])[:16]  // First 16 hex chars
```

16 hex chars = 64 bits = effectively collision-free for reasonable asset counts.

### Registry Structure

```go
type assetRegistry struct {
    mu      sync.RWMutex
    byHash  map[string]string      // hash → absolute filepath
    cache   map[string]assetEntry  // filepath → cached hash info
}

type assetEntry struct {
    hash    string
    modTime time.Time
    size    int64
}
```

### Registration Flow

```go
func (r *assetRegistry) Register(filepath string) (string, error) {
    stat, err := os.Stat(filepath)
    if err != nil {
        return "", err
    }
    
    // Size limits
    if stat.Size() > 100*1024*1024 {
        return "", errors.New("file too large for publicUrl() (>100MB) - use public/ folder")
    }
    if stat.Size() > 10*1024*1024 {
        log.Warn("publicUrl(): large file %s (%dMB) - consider public/ folder", 
                 filepath, stat.Size()/1024/1024)
    }
    
    // Check cache
    r.mu.RLock()
    if entry, ok := r.cache[filepath]; ok {
        if entry.modTime.Equal(stat.ModTime()) && entry.size == stat.Size() {
            r.mu.RUnlock()
            return "/__p/" + entry.hash + path.Ext(filepath), nil
        }
    }
    r.mu.RUnlock()
    
    // Read and hash file
    content, err := os.ReadFile(filepath)
    if err != nil {
        return "", err
    }
    hash := sha256Short(content)
    ext := path.Ext(filepath)
    
    // Update registry
    r.mu.Lock()
    r.byHash[hash] = filepath
    r.cache[filepath] = assetEntry{hash, stat.ModTime(), stat.Size()}
    r.mu.Unlock()
    
    return "/__p/" + hash + ext, nil
}
```

### Serving Flow

```go
// Handler for /__p/* routes
func (r *assetRegistry) ServeAsset(w http.ResponseWriter, req *http.Request) {
    // Extract hash from URL: /__p/{hash}.{ext}
    hashWithExt := strings.TrimPrefix(req.URL.Path, "/__p/")
    hash := strings.TrimSuffix(hashWithExt, path.Ext(hashWithExt))
    
    r.mu.RLock()
    filepath, ok := r.byHash[hash]
    r.mu.RUnlock()
    
    if !ok {
        http.NotFound(w, req)
        return
    }
    
    // Set aggressive cache headers (content-addressed = immutable)
    w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
    
    // Serve file
    http.ServeFile(w, req, filepath)
}
```

### Affected Components
- `server/server.go` — Add asset registry, wire up `/__p/` route
- `server/assets.go` — New file: asset registry implementation
- `pkg/parsley/evaluator/builtins.go` — Add `publicUrl()` builtin
- `pkg/parsley/evaluator/evaluator.go` — Wire builtin to server registry

### Integration with Server

The asset registry lives on the Server struct:
```go
type Server struct {
    // ...existing fields...
    assetRegistry *assetRegistry
}
```

The `publicUrl()` builtin needs access to the registry. Options:
1. Pass registry via environment (like `basil` context)
2. Global registry (simpler but less clean)

Recommend: Add to `basil` context as `basil.publicUrl()` or pass registry via environment.

### Edge Cases & Constraints

1. **File not found** — Return error with clear message including resolved path
2. **File outside handler root** — Security check, reject with error
3. **Symlinks** — Follow symlinks but verify resolved path is within allowed directories
4. **Dev mode** — Registry still works but could skip caching for instant refresh
5. **Concurrent access** — Registry is thread-safe with RWMutex
6. **Server reload** — Clear registry on SIGHUP, rebuild lazily

### Example Usage

**modules/Avatar.pars:**
```parsley
let defaultAvatar = publicUrl(@./default-avatar.svg)

export Avatar = fn({src, alt}) {
  let imgSrc = src ?? defaultAvatar
  <img class="avatar" src={imgSrc} alt={alt ?? ""}/>
}
```

**handlers/profile.pars:**
```parsley
{Avatar} = import(@~/modules/Avatar.pars)

let user = basil.sqlite <=?=> "SELECT * FROM users WHERE id = 1"

<html>
<body>
  <Avatar src={user.avatarUrl} alt={user.name}/>
  <h1>{user.name}</h1>
</body>
</html>
```

## Implementation Notes
*Added during/after implementation*

### Implementation Date: 2025-12-07

**Files Changed:**
- `server/assets.go` (new) — Asset registry with content hashing, HTTP handler
- `server/assets_test.go` (new) — Registry and handler tests
- `server/server.go` — Added `assetRegistry` field, initialization, route setup
- `server/handler.go` — Pass asset registry to environment, inject `publicUrl` builtin
- `server/api.go` — Same changes for API handlers
- `pkg/parsley/evaluator/evaluator.go` — Added `AssetRegistrar` interface, `AssetRegistry` field on `Environment`
- `pkg/parsley/evaluator/public_url.go` (new) — `publicUrl()` builtin implementation
- `pkg/parsley/tests/public_url_test.go` (new) — Builtin tests

**Design Decisions:**
- `publicUrl` is a top-level function (not `basil.publicUrl()`) for simplicity
- Implemented as `StdlibBuiltin` with environment access
- Hash uses first 16 hex chars of SHA256 (64 bits)
- Path security check uses RootPath from handler environment
- Errors use existing error classes (state, security, type, arity)

## Related
- Design doc: `docs/parsley/design/Public files.md`
- FEAT-040: Filesystem-Based Routing (different approach to same problem space)
- FEAT-037: Fragment Caching (similar caching patterns)
