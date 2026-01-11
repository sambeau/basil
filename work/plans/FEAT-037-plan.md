---
id: PLAN-025
feature: FEAT-037
title: "Implementation Plan for Fragment Caching"
status: draft
created: 2025-12-07
---

# Implementation Plan: FEAT-037 Fragment Caching

## Overview
Add `<basil.cache.Cache>` component to cache rendered HTML fragments, reducing database load and improving response times for expensive renders.

## Prerequisites
- [ ] FEAT-038 complete (establishes `basil.*` namespace pattern for components)
- [ ] Design confirmed: in-memory LRU, handler-namespaced keys

## Architecture Decision

Unlike auth components (post-processed via regex), `<basil.cache.Cache>` needs **special evaluation semantics**: it must short-circuit child evaluation on cache hit. This requires evaluator-level handling.

**Approach**: Handle `<basil.cache.Cache>` as a special built-in tag in the evaluator, similar to how `<Fragment>` or other special tags might be handled.

---

## Tasks

### Task 1: Create Fragment Cache Store
**Files**: `server/fragment_cache.go` (new)
**Estimated effort**: Medium

Steps:
1. Create `fragmentCache` struct similar to `responseCache`:
   ```go
   type fragmentCache struct {
       mu      sync.RWMutex
       entries map[string]*fragmentEntry
       maxSize int
       dev     bool // disabled in dev mode
   }
   
   type fragmentEntry struct {
       html      string
       createdAt time.Time
       expiresAt time.Time
   }
   ```
2. Implement methods:
   - `Get(key string) (string, bool)` — returns cached HTML or miss
   - `Set(key string, html string, maxAge time.Duration)`
   - `Invalidate(key string)`
   - `InvalidatePrefix(prefix string)` — for handler invalidation
   - `Clear()` — clear all entries
3. Add LRU eviction when size exceeded

Tests:
- `fragment_cache_test.go`: test get/set/invalidate/expiry

---

### Task 2: Add Fragment Cache to Server
**Files**: `server/server.go`
**Estimated effort**: Small

Steps:
1. Add `fragmentCache *fragmentCache` field to Server struct
2. Initialize in `New()`:
   ```go
   fragmentCache: newFragmentCache(cfg.Server.Dev, 1000), // 1000 entry limit
   ```
3. Clear on file change (in watcher callback, alongside scriptCache)

Tests:
- Verify cache cleared on file change

---

### Task 3: Pass Fragment Cache to Evaluator Context
**Files**: `server/handler.go`
**Estimated effort**: Small

Steps:
1. Add fragment cache reference to `basil` context or environment
2. The evaluator needs access to:
   - Cache store (for get/set)
   - Current handler path (for key namespacing)
   - Dev mode flag (to skip caching)

Option A: Pass via `basil.cache` object in environment
Option B: Pass via evaluator context/options

Recommend Option A for consistency with `basil.*` pattern.

---

### Task 4: Handle `<basil.cache.Cache>` in Evaluator
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Large

This is the core implementation. When evaluator encounters `<basil.cache.Cache>`:

Steps:
1. Detect tag name `basil.cache.Cache` in `evalTagExpression` or similar
2. Extract attributes: `key`, `maxAge`, `enabled`
3. Build full cache key: `{handler_path}:{user_key}`
4. Check cache:
   - **Hit**: Return cached HTML string, skip evaluating children
   - **Miss**: Evaluate children, cache result, return HTML
5. Handle dev mode: log but don't cache

```go
// Pseudo-code
func evalCacheTag(tag *ast.TagExpression, env *Environment) Object {
    key := evalAttribute(tag, "key", env)
    maxAge := evalAttribute(tag, "maxAge", env)
    enabled := evalAttribute(tag, "enabled", env) // default true
    
    if !enabled || env.DevMode {
        // Log skip, evaluate children normally
        return evalChildren(tag.Children, env)
    }
    
    fullKey := env.HandlerPath + ":" + key
    
    if cached, ok := env.FragmentCache.Get(fullKey); ok {
        // Cache hit
        logCacheHit(fullKey)
        return &String{Value: cached}
    }
    
    // Cache miss - evaluate children
    result := evalChildren(tag.Children, env)
    html := objectToString(result)
    
    env.FragmentCache.Set(fullKey, html, maxAge)
    logCacheMiss(fullKey)
    
    return result
}
```

Challenge: The evaluator needs to recognize dotted tag names (`basil.cache.Cache`) and route to special handling. Current tag evaluation likely assumes simple names.

Tests:
- Cache hit returns cached content without re-evaluation
- Cache miss evaluates and stores
- Expiry works (set short TTL, wait, verify miss)
- Different handlers have different namespaces

---

### Task 5: Add `basil.cache.invalidate()` Function
**Files**: `pkg/parsley/evaluator/evaluator.go` or `server/handler.go`
**Estimated effort**: Medium

Steps:
1. Add `invalidate` function to `basil.cache` object
2. Implementation:
   ```go
   func cacheInvalidate(args []Object, env *Environment) Object {
       key := args[0].(*String).Value
       if strings.Contains(key, ":") {
           // Full path provided
           env.FragmentCache.Invalidate(key)
       } else {
           // Relative to current handler
           fullKey := env.HandlerPath + ":" + key
           env.FragmentCache.Invalidate(fullKey)
       }
       return NULL
   }
   ```

Tests:
- Invalidate by key clears that entry
- Invalidate with full path works cross-handler

---

### Task 6: DevTools Integration
**Files**: `server/devtools.go`
**Estimated effort**: Small

Steps:
1. Add fragment cache stats to devtools output:
   - Total entries
   - Hit/miss counts
   - Size estimate
2. Add endpoint to view/clear fragment cache

Tests:
- DevTools shows fragment cache stats

---

### Task 7: Handler File Change Invalidation
**Files**: `server/watcher.go` or `server/handler.go`
**Estimated effort**: Small

Steps:
1. When a handler file changes, invalidate all its fragments:
   ```go
   fragmentCache.InvalidatePrefix(handlerPath + ":")
   ```
2. This is in addition to clearing the script cache

Tests:
- Change handler file, verify its fragments cleared

---

## Key Technical Challenges

### 1. Dotted Tag Name Resolution
The evaluator needs to handle `<basil.cache.Cache>` where `basil.cache.Cache` is a dotted path, not a simple identifier.

Options:
- A) Parse tag name, check if starts with `basil.`, route to built-in handler
- B) Look up `basil` → `cache` → `Cache` as nested dictionary access
- C) Register `basil.cache.Cache` as a string key in a builtins map

Recommend: Option A for simplicity. Check tag name prefix.

### 2. Short-Circuit Evaluation
Cache hits must NOT evaluate children. This is different from normal tag evaluation.

The evaluator must check cache BEFORE evaluating `tag.Children`.

### 3. Environment Threading
Fragment cache instance must be accessible from evaluator. Options:
- Pass in environment (`env.FragmentCache`)
- Pass in evaluation context
- Global (not recommended)

---

## Validation Checklist
- [ ] All tests pass: `make check`
- [ ] New fragment cache tests pass
- [ ] Manual test: cached fragment returns faster, invalidation works
- [ ] Dev mode: caching disabled, logs shown
- [ ] DevTools shows cache stats

## Dependencies
- FEAT-038 should be done first (establishes pattern, may surface dotted-name issues)

## Risks
- Evaluator changes are complex, may have edge cases
- Memory usage if many fragments cached (mitigate with LRU)
- Cache key collisions (mitigate with handler namespacing)

## Future Enhancements (Not in v1)
- SQLite persistence for long-lived fragments
- Wildcard invalidation: `basil.cache.invalidate("user-*")`
- Tag-based invalidation
- Cache warming
