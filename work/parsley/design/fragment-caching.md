# Fragment Caching Design

**Status:** Early design / exploration  
**Date:** 2025-12-07

## Overview

Fragment caching allows caching rendered HTML fragments within a page, enabling fine-grained caching of expensive-to-render components while keeping other parts dynamic.

This differs from HTTP response caching (which caches entire pages) and is particularly useful for:
- Pages with expensive queries that don't change often
- Personalized pages with cacheable shared content
- Reducing database load for common UI elements

## Prior Art

| Framework     | Syntax                      | Storage                               |
| ------------- | --------------------------- | ------------------------------------- |
| Rails         | `<% cache(@product) do %>`  | Configurable (Redis, Memcached, file) |
| Django        | `{% cache 500 "sidebar" %}` | Configurable                          |
| Laravel Blade | `@cache('key', 60)`         | Laravel cache driver                  |
| Phoenix       | Fragment caching via ETS    | In-memory                             |

## Proposed API

### Basic Usage

```parsley
<Cache key="sidebar" maxAge={@1h}>
  <Sidebar items={db.query("SELECT * FROM menu_items")}/>
</Cache>
```

- **key**: Unique identifier for this fragment
- **maxAge**: Duration before cache expires
- **Content**: The HTML to cache (evaluated only on cache miss)

### Dynamic Keys

```parsley
// Per-user cache
<Cache key={"user-nav-" + user.id} maxAge={@15m}>
  <UserNavigation user={user}/>
</Cache>

// Per-locale cache
<Cache key={"footer-" + basil.request.locale} maxAge={@24h}>
  <Footer/>
</Cache>
```

### Vary Parameter

Cache different versions based on variables:

```parsley
<Cache key="products" maxAge={@1h} vary={[category, sortOrder]}>
  <ProductList category={category} sort={sortOrder}/>
</Cache>
```

This generates cache keys like `products:electronics:price-asc`.

### Conditional Caching

```parsley
// Only cache for non-admin users
<Cache key="dashboard" maxAge={@5m} enabled={user.role != "admin"}>
  <Dashboard/>
</Cache>
```

## Component Attributes

| Attribute | Type     | Default      | Description                          |
| --------- | -------- | ------------ | ------------------------------------ |
| `key`     | string   | **required** | Cache key identifier                 |
| `maxAge`  | duration | **required** | TTL for cached fragment              |
| `vary`    | array    | `[]`         | Variables to include in cache key    |
| `enabled` | bool     | `true`       | Whether to use cache                 |
| `scope`   | string   | `"global"`   | `"global"`, `"user"`, or `"request"` |

### Scope Options

- **global**: Shared across all users (default)
- **user**: Keyed by `basil.auth.user.id` automatically
- **request**: Only cache within single request (for repeated renders)

## Storage Options

### Option A: In-Memory (LRU)

Pros:
- Fast
- No external dependencies
- Already have `responseCache` infrastructure

Cons:
- Lost on restart
- Not shared across instances
- Memory pressure

### Option B: SQLite Table

```sql
CREATE TABLE _basil_fragment_cache (
  key TEXT PRIMARY KEY,
  content TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL
);
```

Pros:
- Persists across restarts
- Already have SQLite infrastructure
- Can query/inspect cache

Cons:
- Slower than memory
- Disk I/O

### Option C: Hybrid (Recommended)

- In-memory LRU as primary
- Optional SQLite persistence for long-lived fragments
- Configurable per-fragment or globally

```yaml
cache:
  fragment:
    enabled: true
    backend: memory     # memory, sqlite, or hybrid
    max_size: 100MB     # For memory backend
    default_ttl: 1h     # Default if maxAge not specified
```

## Cache Invalidation

### Time-Based (Automatic)

Fragments expire based on `maxAge`. Simplest approach.

### Manual Invalidation

```parsley
// In a form handler after update
basil.cache.invalidate("sidebar")
basil.cache.invalidate("products:*")  // Wildcard
```

### Tag-Based Invalidation

```parsley
<Cache key="product-123" maxAge={@1h} tags={["products", "featured"]}>
  <ProductCard product={product}/>
</Cache>

// Later, invalidate all products
basil.cache.invalidateTag("products")
```

## Dev Mode Behavior

In `--dev` mode:
- Cache disabled by default (fresh renders every time)
- DevTools shows what *would* be cached
- Optional `?_cache=1` query param to test caching

```
[cache] HIT  sidebar (age: 45s, expires: 15m)
[cache] MISS user-nav-123 (not found)
[cache] SKIP dashboard (dev mode)
```

## Edge Cases

### Nested Cache Tags

```parsley
<Cache key="outer" maxAge={@1h}>
  <div>
    <Cache key="inner" maxAge={@5m}>  // Allowed? Nested cache
      <ExpensiveComponent/>
    </Cache>
  </div>
</Cache>
```

**Options:**
1. **Forbid nesting** - Simpler, error if detected
2. **Allow, independent TTLs** - Inner updates independently
3. **Allow, outer includes rendered inner** - Outer caches the resolved inner

Recommendation: **Option 1** (forbid) initially, revisit if needed.

### Empty Content

```parsley
<Cache key="maybe-empty" maxAge={@1h}>
  {maybeEmptyList}
</Cache>
```

Should we cache empty results? Probably yes (it's a valid result), but with shorter TTL option?

### Cache Stampede

Multiple requests hitting an expired cache simultaneously. Solutions:
- **Lock-based**: First request regenerates, others wait
- **Stale-while-revalidate**: Serve stale content while regenerating
- **Probabilistic early expiration**: Randomly expire before TTL

For v1: Accept stampede risk, optimize later if needed.

## Implementation Considerations

### Parsley Language Changes

`<Cache>` would be a **built-in component** (like `<PasskeyLogin>`), not a user-defined component. It has special semantics:

1. Evaluate `key`, `maxAge`, etc. attributes
2. Check cache for hit
3. If hit: return cached HTML, skip children evaluation
4. If miss: evaluate children, store result, return

This requires the evaluator to handle `<Cache>` specially—children aren't always evaluated.

### Integration Points

- **scriptCache**: Existing Parsley script cache (parsed AST)
- **responseCache**: Existing HTTP response cache
- **fragmentCache**: New, for rendered HTML fragments

These are related but distinct:
- scriptCache: Parsing optimization
- responseCache: HTTP-level, whole responses
- fragmentCache: Render-level, HTML fragments

## Questions to Resolve

1. **Storage default**: Memory-only for simplicity, or hybrid from start?
	1. Suggest: memory-only for simplicity
2. **Invalidation API**: Just time-based for v1, or include manual?
	1. Suggest: With other caches, i.e on file change
	2. Investigate what manual option would look like: How? Where?
3. **Nesting behavior**: Error, or support with clear semantics?
	1. Suggest for v1: Nothing clever: Option 3: e.g. parent tag stores child tag HTML: both caches exist, parent uses child fragment when it refreshes at its rate, effectively slowing a faster child’s refresh rate.  
4. **Key collisions**: Namespace by handler path, or global?
	1.  Suggest: Namespace by handler path
5. **Serialization**: Just HTML strings, or structured data too?
	1.  Suggest: Both if we can do it simply, else just strings for V1.

## Rough Implementation Order

1. In-memory fragment cache store (similar to responseCache)
2. `<Cache>` component recognition in evaluator
3. Short-circuit evaluation on cache hit
4. Basic DevTools integration (hit/miss logging)
5. (Later) SQLite persistence option
6. (Later) Manual invalidation API
7. (Later) Tag-based invalidation

## Non-Goals (For Now)

- Distributed caching (Redis, Memcached)
- Cache warming/preloading
- Compression of cached fragments
- Async/background refresh

## Related

- FEAT-008: Response caching (HTTP-level)
- `server/cache.go`: Existing cache infrastructure
- Rails fragment caching: https://guides.rubyonrails.org/caching\_with\_rails.html#fragment-caching
