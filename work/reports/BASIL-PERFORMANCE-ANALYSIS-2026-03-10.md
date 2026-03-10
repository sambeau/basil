# Basil Performance Analysis — 2026-03-10

## Summary

This report captures a point-in-time performance assessment of the Basil codebase on **2026-03-10**, with special attention to the Basil server request path.

Overall, Basil appears to have **reasonable performance fundamentals** for a programmable web server, but it is **not yet performance-engineered** in the way a mature high-throughput production server would be. The main costs are architectural rather than low-level implementation mistakes. In particular, Basil pays a significant per-request cost for dynamic route execution because requests are routed through the Parsley evaluator, with fresh environment setup and request-context conversion on each request.

### Overall performance assessment

**6.5 / 10**

This score reflects:
- strong use of AST caching in production
- reasonable middleware and cache structure
- conservative and appropriate SQLite configuration
- but also a lack of benchmark coverage, likely allocation-heavy request handling, and at least one server-side behavior that may be defeating reuse on API requests

### Category scores

| Category | Score | Notes |
| --- | --- | --- |
| Dynamic request efficiency | 6 / 10 | Likely CPU- and allocation-heavy due to evaluator-driven execution |
| Static request efficiency | 7 / 10 | Simple and reasonable, but uses synchronous filesystem checks |
| Caching strategy | 7 / 10 | Good basic caches, but some implementations are simplistic |
| Concurrency posture | 7 / 10 | Mostly sound, though optimized more for simplicity than scale |
| Measurement maturity | 4 / 10 | No server benchmarks currently exist |
| Overall | 6.5 / 10 | Acceptable for modest workloads, but under-measured and not yet tuned |

---

## Scope and Method

This analysis is based on:
- inspection of Basil server request handling code
- inspection of server-side cache implementations
- review of compression, watcher, and database setup
- review of current structural hotspots
- inspection of existing benchmark coverage in the repository

This is an architectural performance review, not a full profiling report. It identifies likely hot paths and likely bottlenecks, but it does not claim measured production throughput or latency numbers.

---

## High-Level Performance Model

Basil is not just a thin Go HTTP server. For many routes, it is effectively an **interpreter-backed application server**.

For a dynamic request, the hot path typically includes:

1. route resolution
2. AST lookup from the script cache
3. fresh evaluator environment creation
4. security/session/request context construction
5. conversion of Go request data into Parsley objects
6. script execution via `evaluator.Eval(...)`
7. extraction of response metadata and response writing
8. optional response/fragment cache interaction

This means performance is dominated less by HTTP mux overhead and more by:
- evaluator execution time
- allocation volume
- object conversion costs
- any repeated module setup/import work

That architecture is reasonable for Basil’s goals, but it means dynamic routes will naturally cost more than plain Go handlers or precompiled template handlers.

---

## Main Findings

## 1. Dynamic route execution is likely the dominant cost center

The most important finding is that the expensive part of Basil is almost certainly not the HTTP server itself. It is the per-request dynamic execution path.

### Likely contributors
- `evaluator.Eval(...)`
- environment construction
- request/context map construction
- conversion into Parsley dictionaries and objects
- response metadata extraction after execution

### Implication
For dynamic page or API routes, request cost is likely dominated by:
- CPU in the evaluator
- allocations from setup and conversion
- GC churn under load

### Assessment
This is expected given the product design, but it should be treated as the primary performance domain for any future tuning.

---

## 2. Production AST caching is a meaningful strength

The `scriptCache` used by the server is one of the strongest performance features currently in place.

### Positive characteristics
- production mode caches compiled ASTs
- dev mode disables caching, which is appropriate for hot reload
- `RWMutex` is used appropriately
- repeated lexing/parsing is removed from the steady-state production path

### Impact
This is an important optimization because lexing and parsing are relatively expensive compared with map lookups and mutex reads.

### Caveat
The cache does not currently suppress duplicate concurrent misses for the same script. If many requests hit a cold script path at once, the same script may be parsed multiple times concurrently before one result is cached.

### Assessment
Good implementation for current scale. Potential future improvement would be duplicate-load suppression.

---

## 3. API handlers may be paying a large avoidable penalty

A particularly notable detail in the API request path is that the module cache is cleared at the start of request handling.

### Why this matters
Elsewhere in the codebase, module caching is treated as a performance optimization to preserve reusable loaded state across requests. Clearing that cache per API request may defeat that optimization.

### Likely effects
If the module cache is indeed being invalidated per request, API handlers may incur:
- extra CPU
- extra allocations
- repeated module import/setup work
- reduced throughput
- more variable latency

### Assessment
This is the single most important performance risk found in the Basil server code. It may be intentional for correctness, but if it is not strictly necessary, it is likely one of the highest-value optimization opportunities in the repo.

---

## 4. Per-request environment setup is likely allocation-heavy

Both page and API handlers construct substantial request-specific state on every request.

### Repeated work observed conceptually
- new evaluator environment
- security policy object
- request context map
- params map
- basil namespace map
- session/auth additions
- conversion of Go values into Parsley objects
- capture logger setup

### Why this matters
Even if each step is individually reasonable, together they likely contribute significant:
- bytes allocated per request
- number of allocations per request
- GC pressure at moderate concurrency

### Assessment
This is likely one of the most important sources of request-time overhead after evaluator execution itself.

---

## 5. Response cache is good enough, but intentionally simple

The route-level response cache has a good, simple correctness model:
- cache key based on method/path/query
- full response body cached
- headers cached
- TTL expiry
- `RWMutex`
- manual prune support

### Strengths
- simple and understandable
- avoids stale complexity
- safe for current use
- likely beneficial for repeated GET requests on cacheable routes

### Weaknesses
- no size bound
- no byte budget
- full body stored in memory
- expired entries are cleaned lazily
- SHA-256 is used for every key, which is robust but not free

### Assessment
This is acceptable for modest workloads. It is not obviously problematic, but it is not tuned for large cache footprints or highly optimized cache-hit paths.

---

## 6. Fragment cache works, but the eviction strategy is weak

The fragment cache is described as LRU-like, but in practice it behaves more like:
- TTL-based storage
- plus opportunistic eviction by soonest expiry time

### Current behavior
When full:
- expired entries are removed first
- if still full, entries are sorted by expiry time
- 10% are evicted

### Concerns
- eviction is not true LRU
- eviction work is heavier than necessary
- the sorting approach is quadratic in structure, not ideal if entry counts grow

### Practical impact
At the current default size of 1000 entries, this is probably acceptable. It is unlikely to be a user-visible issue at current scale unless fragment caching becomes extremely hot.

### Assessment
Not urgent, but not ideal. This cache is serviceable rather than optimized.

---

## 7. Static file fallback is straightforward but incurs filesystem checks

The route and root fallback logic performs filesystem lookups using `os.Stat(...)` before deciding whether to serve a static file or fall back to dynamic handling.

### Positive aspects
- simple
- easy to reason about
- correct behavior
- no extra indexing complexity

### Performance tradeoff
This means static-miss requests perform synchronous filesystem metadata checks.

### Likely effect
At low-to-moderate load this is fine. Under heavier load or on slower filesystems, repeated negative checks can become noticeable.

### Assessment
Acceptable now. If static traffic or path-miss volume increases, this may be worth revisiting.

---

## 8. Compression setup is sensible

Compression is implemented using a mature external library with:
- enable/disable support
- level selection
- minimum size threshold

### Positive aspects
- good library choice
- avoids hand-rolled compression behavior
- configurable enough for practical use

### Important tradeoff
Because Basil already performs nontrivial CPU work for dynamic requests, aggressive compression can stack CPU cost on top of route execution cost.

### Assessment
The implementation is good. Operationally, the best performance profile is likely to come from conservative compression settings rather than maximum compression.

---

## 9. SQLite settings are conservative and appropriate

SQLite is configured with:
- WAL mode
- busy timeout
- a single open connection

### Positive aspects
- good correctness posture
- avoids many SQLite concurrency pitfalls
- appropriate for embedded/app-local use

### Performance implication
The database layer will not scale as a highly concurrent transactional backend, but that is an expected tradeoff for SQLite.

### Assessment
This is a sensible choice for Basil’s current architecture. It is not a performance bug, though it may become a throughput limiter in DB-heavy deployments.

---

## 10. Dev-mode watcher costs are acceptable

The filesystem watcher is dev-only and recursively watches relevant directories.

### Assessment
This is not a production performance concern. Its startup and event handling costs are acceptable for development use.

---

## Performance Strengths

## 1. Good production/development split
The code intentionally disables some caching in dev mode while preserving caching in production. This is the right tradeoff for usability versus speed.

## 2. AST caching avoids repeat parsing
This significantly improves production request performance.

## 3. Cache and middleware code is structurally simple
Simple code is easier to reason about, maintain, and profile.

## 4. SQLite is configured safely
The database setup favors correctness and predictable behavior.

## 5. Compression uses a mature implementation
This reduces the risk of inefficient or buggy compression code.

---

## Main Performance Risks

## 1. Per-request evaluator execution
This is the central cost of Basil and likely the biggest throughput limiter for dynamic routes.

## 2. Request-time allocations
Fresh environment setup and value conversion likely generate substantial allocation pressure.

## 3. Per-request API module cache clearing
This may be causing large avoidable overhead.

## 4. Lack of benchmarks
There is currently no benchmark coverage for server hot paths.

## 5. Cache implementations are more practical than optimized
They are good enough for now, but not designed for very large or very hot workloads.

---

## Measurement Maturity Assessment

Performance engineering maturity is currently limited because the repository does not include a benchmark suite for Basil server hot paths.

### What is missing
There are currently no benchmarks for:
- page handler execution
- API handler execution
- response cache hit performance
- response cache miss performance
- fragment cache hit/miss performance
- root/static fallback behavior
- compression overhead on realistic responses

### Why this matters
Without benchmarks:
- performance regressions are easier to introduce
- optimization work is harder to prioritize
- architectural concerns cannot be converted into measured evidence
- realistic capacity planning is not possible

### Assessment
This is the biggest process gap in the repository’s current performance posture.

---

## Most Likely Bottlenecks

If prioritized by likely practical importance, the most probable server bottlenecks are:

1. `evaluator.Eval(...)` on dynamic requests
2. environment and request-context construction
3. conversion between Go values and Parsley objects
4. per-request module cache clearing in API routes
5. SQLite serialization on DB-backed routes

Secondary bottlenecks or scaling concerns:
- static fallback filesystem checks
- fragment cache eviction cost under pressure
- compression CPU on large dynamic responses

---

## Performance by Area

## Dynamic page handlers
### Verdict
**Potentially expensive**

### Reasons
- fresh evaluator environment
- request/session/context injection
- full script evaluation
- result transformation and metadata extraction

---

## API handlers
### Verdict
**Potentially more expensive than necessary**

### Reasons
- same evaluator-driven cost model as page handlers
- module cache clearing may defeat reuse
- request object construction and handler dispatch add additional work

---

## Static file serving
### Verdict
**Good enough**

### Reasons
- simple path handling
- straightforward serving behavior
- fallback path includes filesystem checks, but not excessive complexity

---

## Route and fragment caching
### Verdict
**Useful, but basic**

### Reasons
- caches are easy to understand and probably beneficial
- implementations are not highly optimized for large-scale pressure

---

## Compression
### Verdict
**Good implementation, operationally sensitive**

### Reasons
- good library choice
- likely fine in normal use
- may become a meaningful CPU multiplier on already-heavy dynamic routes

---

## Database-backed requests
### Verdict
**Correctness-first, not throughput-first**

### Reasons
- SQLite configuration is sensible
- single connection limits parallel DB throughput
- acceptable for current architectural target

---

## Recommended Priorities

## Priority 1: Add a Basil server benchmark suite

This is the most important next step.

### Suggested benchmarks
- page handler render benchmark
- API handler simple GET benchmark
- API handler JSON response benchmark
- response cache hit benchmark
- response cache set/get benchmark
- fragment cache hit benchmark
- fragment cache miss benchmark
- root handler static-miss fallback benchmark
- compression overhead benchmark

### Why this matters
This turns the current analysis from informed inference into measured evidence.

---

## Priority 2: Review API module cache clearing

Investigate whether clearing the module cache on every API request is necessary.

### Why this matters
If it is not necessary, removing it may yield one of the highest-impact performance improvements in the server path.

---

## Priority 3: Measure allocations in dynamic route execution

Focus profiling on:
- request context construction
- basil context construction
- params construction
- Go-to-Parsley conversion
- response metadata extraction

### Why this matters
Reducing allocations may improve latency consistency and throughput more than micro-optimizing low-level code.

---

## Priority 4: Benchmark compression settings

Evaluate realistic dynamic responses with:
- compression disabled
- default compression
- fastest compression
- best compression

### Why this matters
This will show whether compression is a worthwhile tradeoff for your actual workloads.

---

## Priority 5: Revisit fragment cache eviction if it becomes important

The current fragment cache implementation is serviceable, but should not be treated as a finished scaling design.

---

## Suggested Near-Term Work Items

### Small / focused
- inspect the API request path for avoidable repeated work
- add a first benchmark for page and API handlers
- measure allocation counts for dynamic route execution

### Medium effort
- add cache benchmarks
- measure impact of SQLite-backed routes
- benchmark realistic compression costs

### Higher leverage
- establish a repeatable `make bench` or equivalent benchmark workflow
- treat server benchmarks as regression guards for future work

---

## Final Verdict

Basil’s performance story is **promising but immature**.

The codebase has several good fundamentals:
- production AST caching
- clear dev/prod behavior split
- simple cache designs
- conservative database setup
- sensible middleware structure

However, it is not yet in a state where strong performance claims can be made, because:
- dynamic route execution is inherently heavy
- API handlers may be performing avoidable repeated work
- there is no benchmark coverage for server hot paths
- allocation behavior has not yet been quantified

### Bottom line

**Basil server performance is probably acceptable for modest workloads, but it is currently under-measured and only lightly optimized.**  
The best next step is not broad refactoring. It is to:
1. add benchmarks
2. validate the API module-cache behavior
3. profile allocations and evaluator-heavy request paths

If those are addressed, the codebase could move toward a much stronger performance posture without requiring large architectural change.

---

## Appendix: Snapshot Conclusions

- Overall performance posture: **Reasonable, but under-measured**
- Strongest current optimization: **production AST caching**
- Most important likely bottleneck: **dynamic request execution via `evaluator.Eval(...)`**
- Most concerning server-specific behavior: **per-request module cache clearing in API handlers**
- Main process gap: **no server benchmark suite**
- Best next step: **add benchmarks before optimizing**