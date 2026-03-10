---
id: PLAN-119
feature: FEAT-139
title: "Implementation Plan for Initial Basil Server Benchmark Suite"
status: draft
created: 2026-03-10
---

# Implementation Plan: FEAT-139

## Overview

Implement an initial Basil server benchmark suite focused on the highest-value hot paths identified in the performance analysis: dynamic page handlers, dynamic API handlers, response cache behavior, fragment cache behavior, and root/static fallback behavior. Add a simple `make bench` workflow that runs the benchmark suite without executing unrelated failing tests.

This plan is intentionally narrow. It does not introduce a repo-wide benchmarking framework or a profiling system. It creates a small, repeatable baseline that future performance work can extend.

## Prerequisites

- [ ] Confirm the benchmark scope remains limited to Basil server paths only
- [ ] Re-read `work/specs/FEAT-139.md` before implementation
- [ ] Re-read `work/reports/BASIL-PERFORMANCE-ANALYSIS-2026-03-10.md` for hotspot context
- [ ] Verify current benchmark invocation works without running normal tests: `go test -run '^$' -bench=. -benchmem ./server/...`
- [ ] Confirm benchmark fixtures can be created with temp directories and no external dependencies

## Tasks

### Task 1: Add benchmark scaffolding and helper fixtures
**Files**: `server/*_bench_test.go` (new files as needed)
**Estimated effort**: Medium

Steps:
1. Create dedicated benchmark files under `server/` to keep benchmark code close to the implementation it measures.
2. Add small benchmark-local helpers for:
   - temp directory setup
   - tiny Parsley page/API fixture creation
   - prewarming script cache or response cache
   - building representative HTTP requests and recorders
3. Keep helper scope minimal and local to benchmark files.
4. Ensure setup work happens outside timed loops where appropriate.
5. Use `b.ReportAllocs()` in every benchmark.

Tests:
- Benchmarks compile successfully
- Running benchmark commands does not require external services
- Fixture setup is deterministic and self-cleaning

---

### Task 2: Implement response cache benchmarks
**Files**: `server/cache_bench_test.go`
**Estimated effort**: Small

Steps:
1. Add `BenchmarkResponseCache_GetHit`.
2. Add `BenchmarkResponseCache_SetThenGet`.
3. Use representative but small request and response values.
4. Prewarm the cache before the timed section for hit-path measurement.
5. Keep benchmark names and setup explicit about warm/hit behavior.

Tests:
- `go test -run '^$' -bench=ResponseCache -benchmem ./server`
- Benchmark output shows `ns/op`, `B/op`, and `allocs/op`

---

### Task 3: Implement fragment cache benchmarks
**Files**: `server/cache_bench_test.go` or `server/fragment_cache_bench_test.go`
**Estimated effort**: Small

Steps:
1. Add `BenchmarkFragmentCache_GetHit`.
2. Add `BenchmarkFragmentCache_GetMiss`.
3. Prepopulate the cache for the hit benchmark.
4. Use realistic but small HTML fragment content.
5. Keep dev-mode disabled in benchmark setup so the cache is active.

Tests:
- `go test -run '^$' -bench=FragmentCache -benchmem ./server`
- Hit and miss paths both execute reliably

---

### Task 4: Implement page handler request-path benchmarks
**Files**: `server/handler_bench_test.go`
**Estimated effort**: Medium

Steps:
1. Create a tiny Parsley page fixture that exercises the normal page handler path.
2. Build a benchmark fixture that creates a handler in production-style cache mode.
3. Add `BenchmarkParsleyHandler_ServeHTTP_Simple`.
4. Add `BenchmarkParsleyHandler_ServeHTTP_ResponseCacheHit`.
5. Ensure AST parsing/prewarming is excluded from the steady-state timed path unless explicitly intended.
6. Keep the page output small and stable to reduce noise.

Tests:
- `go test -run '^$' -bench=ParsleyHandler -benchmem ./server`
- Benchmark runs without dev-mode watcher behavior
- Cache-hit benchmark returns early through the response cache path

---

### Task 5: Implement API handler request-path benchmarks
**Files**: `server/api_bench_test.go`
**Estimated effort**: Medium

Steps:
1. Create a tiny API module fixture that exports a simple `get` handler.
2. Add `BenchmarkAPIHandler_ServeHTTP_SimpleGET`.
3. Add `BenchmarkAPIHandler_ServeHTTP_SmallResponse`.
4. Keep request and response structures minimal but representative.
5. Make sure benchmark setup reflects realistic API handler use without adding unnecessary auth/database complexity.
6. Note any surprising behavior uncovered during benchmark implementation, especially around module cache behavior.

Tests:
- `go test -run '^$' -bench=APIHandler -benchmem ./server`
- Benchmarks run consistently and expose allocation counts
- API path exercises real `ServeHTTP` behavior, not only helper internals

---

### Task 6: Implement root/static fallback benchmarks
**Files**: `server/server_bench_test.go`
**Estimated effort**: Medium

Steps:
1. Create a temp static root and, if needed, a temp route handler fixture.
2. Add `BenchmarkRootHandler_StaticMiss_FallbackToRoute`.
3. Add `BenchmarkRootHandler_StaticHit` if setup remains simple and stable.
4. Ensure the static-miss benchmark intentionally exercises the `os.Stat` miss plus route fallback path.
5. Keep file sizes small and the filesystem setup minimal.

Tests:
- `go test -run '^$' -bench=RootHandler -benchmem ./server`
- Static-hit benchmark serves an existing temp file
- Static-miss benchmark falls back to route handling as intended

---

### Task 7: Add Makefile workflow for benchmarks
**Files**: `Makefile`
**Estimated effort**: Small

Steps:
1. Add a `bench` target.
2. Use a benchmark-safe command that avoids running unrelated normal tests.
3. Update the help text to document the new target.
4. Keep the command simple and discoverable.

Suggested command:
1. `go test -run '^$' -bench=. -benchmem ./server/...`

Tests:
- `make bench`
- `make help` includes the new target description

---

### Task 8: Document benchmark coverage and usage
**Files**: `work/reports/` or implementation notes, optionally `AGENTS.md` if clearly warranted
**Estimated effort**: Small

Steps:
1. Add a short implementation note or report summarizing:
   - what benchmarks were added
   - how to run them
   - what they are intended to compare
2. Record any deferred benchmark scenarios in `work/BACKLOG.md` if implementation reveals additional worthwhile work that is out of scope.
3. Keep documentation concise and operational.

Tests:
- Documentation accurately matches the actual benchmark command
- Any deferrals are clearly scoped and actionable

---

## Validation Checklist

- [x] Benchmark files compile successfully
- [x] Cache benchmarks run: `go test -run '^$' -bench=Cache -benchmem ./server`
- [x] Page handler benchmarks run: `go test -run '^$' -bench=ParsleyHandler -benchmem ./server`
- [x] API handler benchmarks run: `go test -run '^$' -bench=APIHandler -benchmem ./server`
- [x] Root/static benchmarks run: `go test -run '^$' -bench=RootHandler -benchmem ./server`
- [x] Full benchmark suite runs: `go test -run '^$' -bench=. -benchmem ./server/...`
- [x] Makefile target works: `make bench`
- [x] General tests still pass as far as the repo's current known failures allow
- [x] Linter passes for touched files: `golangci-lint run`
- [x] Documentation updated
- [x] `work/BACKLOG.md` updated with deferrals if any new out-of-scope work is discovered

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-03-10 | Plan created | ✅ Complete | Initial implementation plan for FEAT-139 |
| 2026-03-10 | Benchmark scaffolding | ✅ Complete | Created benchServer helper in handler_bench_test.go |
| 2026-03-10 | Cache benchmarks | ✅ Complete | 9 benchmarks in cache_bench_test.go |
| 2026-03-10 | Page handler benchmarks | ✅ Complete | 5 benchmarks in handler_bench_test.go |
| 2026-03-10 | API handler benchmarks | ✅ Complete | 5 benchmarks in api_bench_test.go |
| 2026-03-10 | Root/static benchmarks | ✅ Complete | 6 benchmarks in server_bench_test.go |
| 2026-03-10 | Makefile workflow | ✅ Complete | Added `make bench` target |
| 2026-03-10 | Documentation | ✅ Complete | Created work/reports/FEAT-139-implementation-notes.md |

## Deferred Items

Items to add to `work/BACKLOG.md` after implementation if they are not completed as part of FEAT-139:

- Compression-specific benchmarks — defer if setup complexity is too high for the initial suite
- Database-backed route benchmarks — defer if they add significant fixture or setup complexity
- Cold-start / uncached parse-path benchmarks — defer if the initial suite focuses only on steady-state request costs
- Repo-wide benchmark conventions — explicitly out of scope for FEAT-139
- Benchmark result history/reporting automation — useful later, but not required for the initial suite