---
id: FEAT-139
title: "Initial Basil Server Benchmark Suite"
status: draft
priority: high
created: 2026-03-10
author: "@ai"
---

# FEAT-139: Initial Basil Server Benchmark Suite

## Summary

Add an initial Go benchmark suite for the Basil server focused on the highest-value hot paths: dynamic page handlers, dynamic API handlers, response caching, fragment caching, and static-file fallback. This feature also adds a simple `make bench` workflow so performance regressions can be measured consistently over time.

The goal is not to build a full profiling framework. The goal is to create a small, reliable benchmark foundation for the Basil server so future optimization work is based on measured data instead of intuition.

## User Story

As a Basil maintainer, I want repeatable benchmarks for the main server request paths so that I can detect regressions, compare implementation changes, and make performance work evidence-driven.

## Motivation

A performance review of the Basil server found that the main costs are architectural rather than low-level:

- dynamic routes execute through the Parsley evaluator
- each request builds a fresh environment and request context
- API handlers may be performing avoidable repeated work
- route-level and fragment caching exist, but their real impact is not measured
- there is currently no benchmark coverage for Basil server hot paths

The repository already has limited benchmark precedent (`pkg/parsley/lexer/lexer_bench_test.go`) and previous search-related work explicitly called out the need for benchmark tests and a `make bench` target. This feature applies that same discipline to the Basil server.

## Non-Goals

- Full repo-wide benchmark conventions for every package
- Production load testing with external traffic generators
- pprof automation or profile artifact generation
- CI pass/fail thresholds for exact benchmark numbers
- Benchmarking every middleware and edge case in the server
- Premature optimization of code paths before baseline measurements exist

## Acceptance Criteria

### Benchmark coverage

- [ ] Add a benchmark for a simple Basil page handler request path
- [ ] Add a benchmark for a page handler with cache hit behavior
- [ ] Add a benchmark for a simple Basil API handler request path
- [ ] Add a benchmark for an API handler returning a small JSON-like response
- [ ] Add a benchmark for `responseCache` get/hit behavior
- [ ] Add a benchmark for `responseCache` set+get behavior
- [ ] Add a benchmark for `fragmentCache` hit behavior
- [ ] Add a benchmark for `fragmentCache` miss behavior
- [ ] Add a benchmark for root/static fallback on static miss followed by route fallback
- [ ] Add a benchmark for static file serving hit path if practical with in-test temp files
- [ ] All benchmarks use `b.ReportAllocs()`
- [ ] Benchmarks are deterministic enough to compare relative changes between commits

### Workflow

- [ ] Add a `make bench` target that runs the initial Basil benchmark suite
- [ ] `make help` documents the new benchmark target
- [ ] Benchmark command avoids running unrelated failing tests by using a benchmark-safe invocation strategy

### Scope and isolation

- [ ] Benchmarks do not require external services
- [ ] Benchmarks do not require network access
- [ ] Benchmarks do not require persistent user-local files
- [ ] Benchmarks clean up temp files/directories they create
- [ ] Benchmarks can run on a developer laptop in a reasonable amount of time

### Documentation

- [ ] Add a short implementation note or report entry summarizing what is benchmarked and how to run it
- [ ] Record any intentionally deferred benchmark scenarios in `work/BACKLOG.md` if discovered during implementation

## Design Decisions

- **Focus on Basil server only**: This feature covers only the Basil server benchmark suite, not a general repository-wide benchmark framework. The problem to solve is the lack of measurement around server hot paths identified in the performance analysis.

- **Start with micro/mid-level benchmarks, not load tests**: Go `Benchmark*` functions are the right first layer because they are easy to run locally, easy to version with the codebase, and good at showing `ns/op`, `B/op`, and `allocs/op`. Full traffic simulation can come later if needed.

- **Benchmark behavior, not internals in isolation only**: Cache-only benchmarks are useful, but the suite should also include request-path benchmarks that exercise real server code through `ServeHTTP` where practical. The suite should reflect actual Basil behavior, not only utility package performance.

- **Keep fixtures minimal**: The benchmark suite should use the smallest realistic handlers/scripts necessary to exercise the target path. The objective is stable comparative measurement, not exhaustive end-to-end realism.

- **Use temp directories and in-memory setup where possible**: The suite should not depend on long-lived on-disk fixtures unless they are checked into the repo for clarity and stability. If temporary files are enough, prefer temporary files.

- **Separate cached vs uncached cases explicitly**: Since Basil’s performance profile changes significantly depending on cache behavior, benchmarks should label cache state clearly rather than mixing both into one number.

- **Avoid exact numeric promises in the spec**: The suite should establish baseline measurements first. It should not fail implementation just because specific throughput or latency numbers are not met yet.

---

<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Existing Benchmarking State

Current repository state relevant to this feature:

- Benchmarks already exist for the lexer:
  - `pkg/parsley/lexer/lexer_bench_test.go`
- No Basil server benchmarks currently exist
- `Makefile` currently has no `bench` target
- Prior search-related audit/history already establishes benchmark precedent:
  - `work/BACKLOG.md` contains `#37 Search benchmark tests`
  - `work/reports/FEAT-085-audit.md` recommends adding `Benchmark*` tests and a `make bench` target

This feature should follow the repo’s existing lightweight benchmarking style rather than inventing a complex framework.

### Affected Components

- `server/*_test.go` or new `server/*_bench_test.go` files — benchmark implementations
- `server/cache.go` — benchmark target for response cache behavior
- `server/fragment_cache.go` — benchmark target for fragment cache behavior
- `server/handler.go` — benchmark target for page-handler request path
- `server/api.go` — benchmark target for API-handler request path
- `server/server.go` — benchmark target for root/static fallback behavior
- `Makefile` — add `bench` target
- `work/reports/` or implementation notes — optional short benchmark summary after implementation
- `work/BACKLOG.md` — only if new deferred benchmark scenarios emerge during implementation

## Proposed Benchmark Set

The initial suite should stay small and focused. Suggested benchmarks:

### 1. Page Handler Benchmarks

#### `BenchmarkParsleyHandler_ServeHTTP_Simple`
Purpose:
- Measure the baseline dynamic page request path for a small script

Characteristics:
- small Parsley page handler
- no database
- minimal auth/session overhead
- production cache mode for ASTs

Measures:
- request path overhead
- environment construction
- evaluator execution
- response writing
- allocations per request

#### `BenchmarkParsleyHandler_ServeHTTP_ResponseCacheHit`
Purpose:
- Measure the speed of the page path when response cache returns early

Characteristics:
- cache pre-warmed
- GET request
- route configured with cache TTL
- benchmark runs repeated cache-hit requests

Measures:
- cache hit overhead
- response copy/write cost
- allocation reduction relative to uncached path

### 2. API Handler Benchmarks

#### `BenchmarkAPIHandler_ServeHTTP_SimpleGET`
Purpose:
- Measure the API module execution path for a simple request

Characteristics:
- small API module
- minimal request object
- simple return payload
- no auth enforcement complexity unless required for realism

Measures:
- API environment setup
- request object construction
- evaluator execution
- dispatch/write overhead

#### `BenchmarkAPIHandler_ServeHTTP_SmallResponse`
Purpose:
- Measure API response shaping for a small structured response

Characteristics:
- returns a simple dictionary/body/status shape representative of real usage

Measures:
- overhead of API result handling and response generation

### 3. Response Cache Benchmarks

#### `BenchmarkResponseCache_GetHit`
Purpose:
- Isolate the steady-state hit path for `responseCache.Get`

#### `BenchmarkResponseCache_SetThenGet`
Purpose:
- Measure combined set/get path and cloning/storage costs

These benchmarks should use representative HTTP requests and response headers/body sizes, but keep fixtures small.

### 4. Fragment Cache Benchmarks

#### `BenchmarkFragmentCache_GetHit`
Purpose:
- Measure fragment cache hit path

#### `BenchmarkFragmentCache_GetMiss`
Purpose:
- Measure fragment cache miss path

Optional:
- a separate set benchmark if useful, but hit/miss behavior is the higher-value initial coverage

### 5. Root/Static Fallback Benchmarks

#### `BenchmarkRootHandler_StaticMiss_FallbackToRoute`
Purpose:
- Measure the common fallback path when a request is not a static file and falls through to the route handler

Characteristics:
- root handler configured with both static root and route handler
- benchmark path intentionally misses static file resolution

Measures:
- filesystem stat overhead
- fallback logic overhead
- dynamic handler entry cost

#### `BenchmarkRootHandler_StaticHit`
Purpose:
- Measure the static-file hit path if practical using temp files

Characteristics:
- temp static file exists
- request served directly via static path

Measures:
- cost of file existence check plus file serving path

## Benchmark Structure

### File layout

Preferred approach:
- create focused benchmark files under `server/`, such as:
  - `server/handler_bench_test.go`
  - `server/api_bench_test.go`
  - `server/cache_bench_test.go`
  - `server/server_bench_test.go`

This keeps benchmarks close to the code they measure and matches normal Go test organization.

### Helper strategy

Small benchmark-only helpers are acceptable if needed, for example:
- create temporary handler roots
- build a minimal `Server` or handler fixture
- prewarm AST cache or response cache
- build representative requests
- write tiny Parsley handler files for execution

Do not create a large “benchmark framework” abstraction. A few local helpers are enough.

### Invocation strategy

Because normal `go test ./...` currently includes unrelated failing tests in the repo, the benchmark workflow should use a benchmark-safe command.

Suggested pattern:
- `go test -run '^$$' -bench=. -benchmem ./server/...`

This avoids running regular tests while still running benchmark functions.

The `Makefile` target should use this pattern, or a narrower one if the final suite ends up intentionally scoped to specific benchmark files/packages.

## Data and Fixture Design

### Parsley scripts
Use tiny scripts/modules that are:
- representative enough to exercise real Basil behavior
- small enough to keep benchmark results stable
- easy to read in the benchmark source

Examples of benchmark fixture intent:
- a minimal page script returning a short HTML structure
- a minimal API module exporting `get`
- a route configuration with caching enabled
- a root handler with a temporary public directory

### Filesystem usage
If a benchmark needs actual files:
- create them in `b.TempDir()`
- write only the minimal files required
- avoid checked-in fixture directories unless implementation clarity strongly benefits from them

### DB usage
Do not include database-backed route benchmarks in the initial suite unless they can be added without significant setup complexity. DB-backed request benchmarks are valuable, but they are lower priority than the core dynamic server path and may be added later once the basic suite exists.

## Constraints and Edge Cases

1. **Benchmarks must avoid unrelated repo test failures**
   - Use a benchmark-specific command that does not run standard tests.

2. **Benchmarks must not accidentally measure setup inside the timed loop**
   - Expensive fixture creation should happen before `b.ResetTimer()` where appropriate.

3. **Benchmarks should report allocations**
   - `b.ReportAllocs()` should be used consistently.

4. **Benchmarks should clearly distinguish cold vs warm paths**
   - For example, AST parse/setup should not be accidentally mixed into a “steady-state request” benchmark unless the benchmark name says it is a cold-path benchmark.

5. **Benchmarks should not depend on dev-mode watcher behavior**
   - Dev-mode file watching is not part of the target measurement for this suite.

6. **Compression benchmarking is optional in v1 of the suite**
   - If implementing compression benchmarks adds too much setup complexity, it may be deferred to backlog. The initial spec’s main target is baseline server request measurement.

7. **Avoid brittle assertions on exact timing**
   - Benchmarks should exist to produce numbers, not to encode fragile performance thresholds.

## Dependencies

- Depends on: none
- Informed by:
  - `work/reports/BASIL-PERFORMANCE-ANALYSIS-2026-03-10.md`
  - `work/BACKLOG.md` items `#115` and `#116`
  - existing lexer benchmark precedent
  - search benchmark precedent in backlog/audit notes

## Implementation Notes

Implementation should proceed in this order:

1. Add cache benchmarks first
   - easiest to implement
   - lowest fixture complexity
   - immediate measurement value

2. Add page handler benchmark
   - most important dynamic-route baseline

3. Add API handler benchmark
   - likely the most revealing server benchmark because of module/environment behavior

4. Add root/static fallback benchmark
   - useful for static-vs-dynamic dispatch cost understanding

5. Add `make bench`
   - only after benchmark scope is known and stable

This sequencing reduces implementation risk and keeps the feature shippable in partial stages if needed.

## Suggested Benchmark Command

Recommended Makefile target command:

`go test -run '^$$' -bench=. -benchmem ./server/...`

Possible future variants:
- `go test -run '^$$' -bench=BenchmarkParsleyHandler -benchmem ./server`
- `go test -run '^$$' -bench=BenchmarkAPIHandler -benchmem ./server`

But the initial developer workflow should prefer one simple command.

## Success Criteria After Implementation

After implementation, maintainers should be able to:

- run `make bench`
- see benchmark output for core Basil server paths
- compare `ns/op`, `B/op`, and `allocs/op` across commits
- use the results to validate future server performance work
- extend the benchmark suite incrementally without redesigning it

## Related

- Report: `work/reports/BASIL-PERFORMANCE-ANALYSIS-2026-03-10.md`
- Backlog: `work/BACKLOG.md` (`#115`, `#116`)
- Related precedent: `work/reports/FEAT-085-audit.md`
- Existing benchmark example: `pkg/parsley/lexer/lexer_bench_test.go`
