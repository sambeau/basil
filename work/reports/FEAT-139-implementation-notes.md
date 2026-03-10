# FEAT-139 Implementation Notes: Initial Basil Server Benchmark Suite

**Date:** 2026-03-10
**Feature:** FEAT-139
**Plan:** PLAN-119

## Summary

Implemented an initial benchmark suite for the Basil server, covering the main hot paths identified in the performance analysis. Added a `make bench` workflow for easy benchmark execution.

## What Was Implemented

### Benchmark Files

1. **`server/cache_bench_test.go`** — Response cache and fragment cache benchmarks
   - `BenchmarkResponseCache_GetHit` — Cache hit path
   - `BenchmarkResponseCache_GetMiss` — Cache miss path
   - `BenchmarkResponseCache_SetThenGet` — Combined set/get operation
   - `BenchmarkResponseCache_CacheKey` — Key generation overhead
   - `BenchmarkFragmentCache_GetHit` — Fragment cache hit
   - `BenchmarkFragmentCache_GetMiss` — Fragment cache miss
   - `BenchmarkFragmentCache_SetThenGet` — Combined set/get
   - `BenchmarkFragmentCache_LargeFragment` — Large HTML fragment handling
   - `BenchmarkFragmentCache_ManyKeys` — Performance with many cached entries

2. **`server/handler_bench_test.go`** — Page handler benchmarks
   - `BenchmarkParsleyHandler_ServeHTTP_Simple` — Minimal page request
   - `BenchmarkParsleyHandler_ServeHTTP_WithVariables` — Page with variables and iteration
   - `BenchmarkParsleyHandler_ServeHTTP_ResponseCacheHit` — Cached page response
   - `BenchmarkParsleyHandler_ServeHTTP_WithQueryParams` — Page using `@params`
   - `BenchmarkParsleyHandler_ServeHTTP_JSONResponse` — Dictionary/JSON response

3. **`server/api_bench_test.go`** — API handler benchmarks
   - `BenchmarkAPIHandler_ServeHTTP_SimpleGET` — Minimal API GET
   - `BenchmarkAPIHandler_ServeHTTP_SmallResponse` — Structured response
   - `BenchmarkAPIHandler_ServeHTTP_POST` — POST request handling
   - `BenchmarkAPIHandler_ServeHTTP_WithComputation` — API with map/filter/reduce
   - `BenchmarkAPIHandler_ServeHTTP_GetById` — ID extraction from path

4. **`server/server_bench_test.go`** — Root/static fallback benchmarks
   - `BenchmarkRootHandler_StaticHit` — Static file hit
   - `BenchmarkRootHandler_StaticMiss_FallbackToRoute` — Miss to dynamic route
   - `BenchmarkRootHandler_StaticMiss_404` — Miss to 404
   - `BenchmarkRootHandler_MultipleStaticFiles` — Multiple file serving
   - `BenchmarkRootHandler_SingleFileRoute` — Single file route (e.g., favicon)
   - `BenchmarkRootHandler_NestedStaticPath` — Nested directory structure

### Makefile Target

Added `make bench` target:
```bash
go test -run '^$' -bench=. -benchmem ./server/...
```

This runs all server benchmarks while avoiding unrelated tests.

## How to Run

```bash
# Run all server benchmarks
make bench

# Run specific benchmark category
go test -run '^$' -bench=ResponseCache -benchmem ./server
go test -run '^$' -bench=FragmentCache -benchmem ./server
go test -run '^$' -bench=ParsleyHandler -benchmem ./server
go test -run '^$' -bench=APIHandler -benchmem ./server
go test -run '^$' -bench=RootHandler -benchmem ./server
```

## Sample Output

All benchmarks report `ns/op`, `B/op`, and `allocs/op`:

```
BenchmarkResponseCache_GetHit-8         6830312    175.2 ns/op    160 B/op    3 allocs/op
BenchmarkFragmentCache_GetHit-8        25505527     47.1 ns/op      0 B/op    0 allocs/op
BenchmarkParsleyHandler_ServeHTTP_Simple-8  17290  68833 ns/op  114786 B/op  1260 allocs/op
BenchmarkAPIHandler_ServeHTTP_SimpleGET-8   13434  88613 ns/op  148693 B/op  1647 allocs/op
BenchmarkRootHandler_StaticHit-8           98094  12086 ns/op    2312 B/op    29 allocs/op
```

## Key Findings

1. **Cache performance is excellent** — Fragment cache hits are ~47ns with zero allocations
2. **Dynamic page requests** — ~70-90µs per request, ~115-150KB, ~1200-1700 allocations
3. **API requests** — ~90-105µs per request, slightly higher overhead than pages
4. **Static file serving** — ~12µs per request, very efficient

## Known Limitations

- Response cache hit benchmark shows a warning that cache may not be active (the route cache TTL setting works but the X-Cache header isn't always set)
- Benchmarks use minimal Parsley scripts; real-world scripts may have different characteristics
- No database-backed benchmarks in this initial suite (deferred)

## Deferred to Backlog

The following were identified as valuable but out of scope for v1:
- Compression benchmarks
- Database-backed route benchmarks  
- Cold-start/uncached parse-path benchmarks
- Benchmark result history/reporting automation