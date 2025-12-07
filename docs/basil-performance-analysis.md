# Basil Performance Analysis

## What Basil Does Per Request

Looking at the internals:

1. **AST Caching** (production mode): Scripts are parsed once, AST cached in-memory with `sync.RWMutex`
2. **Response Caching**: Full response caching with configurable TTL
3. **Module Cache**: Cleared per-request (intentional - for fresh `basil.*` context)
4. **Fresh Environment**: New `Environment` created each request
5. **Interpreter Execution**: Walks the AST, evaluates nodes

## The Bottleneck: Interpretation

Basil uses **tree-walking interpretation** - no bytecode compilation, no JIT. Each request:
- Walks the entire AST
- Creates new Parsley objects for every expression
- Allocates memory for intermediate results

This is **inherently slower** than compiled Go code, but:
- AST is cached (no re-parsing)
- Simple operations are fast
- Most handlers are small

## Realistic Throughput Estimates

| Scenario | Estimated RPS | Notes |
|----------|---------------|-------|
| **Cache HIT** | 10,000-50,000+ | Pure Go, no Parsley execution |
| **Simple handler** (return HTML string) | 2,000-5,000 | Minimal AST walking |
| **Typical handler** (template + some logic) | 500-2,000 | Depends on complexity |
| **Complex handler** (loops, DB, validation) | 100-500 | DB is usually the bottleneck |
| **With heavy string ops** | 50-200 | String manipulation in interpreters is slow |

## Comparison to Other Go Servers

| Framework | Typical RPS (simple endpoint) | Why |
|-----------|------------------------------|-----|
| **Pure Go (net/http)** | 100,000+ | Native compilation |
| **Gin/Echo/Fiber** | 50,000-100,000 | Thin wrapper over net/http |
| **Basil (cache hit)** | 10,000-50,000 | Go code, no interpretation |
| **Basil (uncached)** | 500-5,000 | Tree-walking interpreter |
| **Node.js (Express)** | 5,000-15,000 | V8 JIT compiled |
| **Python (Flask)** | 500-2,000 | Interpreted |
| **Ruby (Rails)** | 200-1,000 | Interpreted + framework overhead |

**Basil is comparable to Python/Ruby** for complex handlers, but with caching enabled, it can punch well above its weight.

## Implications for Rate Limiting Defaults

Given this profile:

```javascript
// Conservative but realistic defaults
api.defaults({
    rateLimit: {
        requests: 60,     // 1/sec average - safer for interpreted routes
        window: @1m,
        by: "ip"
    }
})
```

Or tiered by route type:

```javascript
// Cached/static routes - generous
get "/products" {
    cache: {maxAge: @5m},
    rateLimit: {requests: 300, window: @1m}  // Cache will absorb most
}

// Uncached dynamic routes - moderate
get "/dashboard" {
    rateLimit: {requests: 60, window: @1m}
}

// Heavy compute routes - strict
post "/reports/generate" {
    rateLimit: {requests: 10, window: @1m}
}
```

## What Would Make Basil Faster?

If performance became critical:

1. **Bytecode compilation** - compile AST to bytecode, run VM (2-5x faster)
2. **Object pooling** - reuse Integer/String objects (reduce GC pressure)
3. **Response caching** - already exists, use liberally
4. **Pre-compiled templates** - cache template expansion results
5. **Connection pooling** - already exists for DB

## The Honest Answer

For a **typical web app** (blog, dashboard, small API):
- **100-1000 req/sec** is realistic without caching
- **10,000+ req/sec** with aggressive caching
- This is **plenty** for most use cases

For **high-traffic API**:
- Cache everything cacheable
- Keep Parsley handlers thin
- Consider Go handlers for hot paths

**Rate limit default recommendation:** `60/minute` (1/sec) is safe and realistic given interpretation overhead. Users can tune up for cached routes.
