---
id: FEAT-037
title: "Fragment Caching"
status: draft
priority: medium
created: 2025-12-07
author: "@human"
---

# FEAT-037: Fragment Caching

## Summary
Add a `<basil.cache.Cache>` component that caches rendered HTML fragments, enabling fine-grained caching of expensive-to-render components while keeping other parts of the page dynamic. This reduces database load and improves response times for pages with expensive queries or shared content.

## User Story
As a developer, I want to cache expensive HTML fragments so that I can build personalized pages with good performance by caching the parts that don't change often.

## Acceptance Criteria
- [ ] `<basil.cache.Cache key="..." maxAge={@duration}>` component caches its children's rendered HTML
- [ ] Cache hit skips evaluation of children entirely (performance benefit)
- [ ] Cache miss evaluates children, stores result, returns HTML
- [ ] Dynamic keys work: `key={"prefix-" + variable}`
- [ ] Keys are namespaced by handler path (no collisions between handlers)
- [ ] Cache invalidated on file change (like other caches)
- [ ] `basil.cache.invalidate(key)` clears a specific cache entry
- [ ] Dev mode: cache disabled by default, logs what would be cached
- [ ] DevTools shows cache hit/miss information

## Design Decisions
- **Lives in `basil.cache` namespace**: Consistent with Basil namespace structure (see FEAT-038). Future `basil.cache.invalidate()` will live alongside.
- **In-memory LRU storage for v1**: Simplest approach, matches existing caches. SQLite persistence deferred.
- **Handler-namespaced keys**: `key="sidebar"` in `/dashboard.pars` becomes `/dashboard:sidebar` internally. Prevents collisions.
- **Nested caches allowed (Option 3)**: Parent stores child's rendered HTML. Both caches exist independently. Parent refreshes at its rate, using child's cached fragment.
- **Invalidation on file change**: When handler file changes, its fragments are cleared. Manual invalidation API deferred.
- **HTML strings for v1**: Cache stores rendered HTML strings. Structured data caching deferred.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/evaluator/` — Handle `<Cache>` component with short-circuit evaluation
- `server/cache.go` — Add fragment cache store (similar to responseCache)
- `server/handler.go` — Pass fragment cache to evaluator context
- `server/devtools.go` — Add fragment cache statistics

### Dependencies
- Depends on: None
- Blocks: None

### API Reference

#### Basic Usage
```parsley
<basil.cache.Cache key="sidebar" maxAge={@1h}>
  <Sidebar items={db.query("SELECT * FROM menu_items")}/>
</basil.cache.Cache>
```

#### Dynamic Keys
```parsley
<basil.cache.Cache key={"user-nav-" + user.id} maxAge={@15m}>
  <UserNavigation user={user}/>
</basil.cache.Cache>
```

#### Attributes

| Attribute | Type     | Default      | Description             |
| --------- | -------- | ------------ | ----------------------- |
| `key`     | string   | **required** | Cache key identifier    |
| `maxAge`  | duration | **required** | TTL for cached fragment |
| `enabled` | bool     | `true`       | Whether to use cache    |

Future attributes (deferred):
- `vary`: Variables to include in cache key
- `scope`: `"global"`, `"user"`, or `"request"`
- `tags`: For tag-based invalidation

### Edge Cases & Constraints

1. **Empty content** — Cache empty results (valid result). No special handling.
2. **Nested caches** — Parent caches child's rendered output. Both exist in cache. Child updates independently, parent uses child's cached fragment when parent refreshes.
3. **Cache stampede** — Accept risk for v1. First request regenerates, concurrent requests may also regenerate. Optimize later if needed.
4. **Dev mode** — Cache operations logged but not executed. Shows `[cache] SKIP key (dev mode)`.
5. **Missing key attribute** — Runtime error: "Cache component requires 'key' attribute"
6. **Missing maxAge attribute** — Runtime error: "Cache component requires 'maxAge' attribute"

### Cache Key Format

Internal key format: `{handler_path}:{user_key}`

Examples:
- `key="sidebar"` in `/handlers/dashboard.pars` → `/handlers/dashboard:sidebar`
- `key={"user-" + id}` with id=123 → `/handlers/profile:user-123`

### Invalidation Strategy

For v1, fragments are invalidated:
1. **On expiry**: When `maxAge` duration passes
2. **On file change**: When the handler file is modified (watcher triggers clear)
3. **On server restart**: In-memory cache is lost

### Manual Invalidation: `basil.cache.invalidate()`

**Status**: Include in v1 if straightforward, otherwise defer.

Manual invalidation is useful when data changes outside of file edits—e.g., after a form submission updates the database.

#### Basic API
```parsley
// Invalidate a specific key (current handler's namespace)
basil.cache.invalidate("sidebar")

// Invalidate with explicit handler path
basil.cache.invalidate("/handlers/dashboard:sidebar")

// Invalidate all keys matching a prefix (wildcard)
basil.cache.invalidate("user-*")  // All user-* keys in current handler
basil.cache.invalidate("/handlers/dashboard:*")  // All keys in dashboard handler
```

#### Use Case: Form Handler
```parsley
// In handlers/edit-menu.pars (POST handler)
let item = basil.http.request.form

// Update database
basil.sqlite.exec("UPDATE menu_items SET name = ? WHERE id = ?", [item.name, item.id])

// Invalidate cached sidebar (which shows menu items)
basil.cache.invalidate("/handlers/layout:sidebar")

// Redirect back
basil.http.response.status = 303
basil.http.response.headers["Location"] = "/admin/menu"
```

#### Design Questions

1. **Key scoping**: Should `invalidate("sidebar")` only affect current handler, or search all handlers?
   - **Suggest**: Current handler by default, full path for cross-handler

2. **Wildcards**: Support `*` patterns?
   - **Suggest**: Yes, useful for `user-*` to clear all user-specific fragments
   - Simple glob, not regex

3. **Return value**: Void, or count of invalidated entries?
   - **Suggest**: Void for simplicity (or bool: true if anything was invalidated)

4. **Non-existent keys**: Error or silent no-op?
   - **Suggest**: Silent no-op (cache may have expired already)

5. **Cross-handler invalidation**: Allow invalidating another handler's cache?
   - **Suggest**: Yes, via full path `/handlers/other:key`
   - Useful for shared fragments (e.g., nav bar cached in layout, invalidated by settings page)

#### Implementation Complexity

**Simple case** (include in v1):
```parsley
basil.cache.invalidate("sidebar")  // Exact key, current handler
```
This is just a map delete—trivial to implement.

**Complex case** (maybe defer):
```parsley
basil.cache.invalidate("user-*")  // Wildcard requires iteration
basil.cache.invalidate("/handlers/other:*")  // Cross-handler + wildcard
```
Wildcards require iterating cache keys, which is O(n) but probably fine for typical cache sizes.

#### Recommendation

Include **basic invalidation** in FEAT-037:
- `basil.cache.invalidate(key)` — exact key, current handler namespace
- `basil.cache.invalidate(fullPath)` — exact key with explicit handler path

Defer **wildcards** to future enhancement if needed:
- `basil.cache.invalidate("prefix-*")` — pattern matching

## Implementation Notes
*Added during/after implementation*

## Related
- Design doc: `docs/parsley/design/fragment-caching.md`
- FEAT-038: Basil Namespace Cleanup (establishes `basil.cache.*` pattern)
- FEAT-039: Enhanced Import Syntax (allows `{Cache} = import @basil/cache`)
- FEAT-008: Response caching (HTTP-level, different layer)
- `server/cache.go`: Existing cache infrastructure
