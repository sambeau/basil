---
id: PLAN-121
feature: FEAT-141
title: "Implementation Plan: Repeatable Benchmark Workflow"
status: complete
created: 2026-03-14
---

# Implementation Plan: FEAT-141

## Overview
Implement a comprehensive benchmark workflow with machine-specific baselines, historical tracking, AI agent integration, and performance reporting. This addresses the "higher leverage" recommendations from BASIL-PERFORMANCE-ANALYSIS-2026-03-10.md.

## Prerequisites
- [ ] `benchstat` installed: `go install golang.org/x/perf/cmd/benchstat@latest`
- [ ] Spec reviewed and approved: `work/specs/FEAT-141-benchmark-workflow.md`

## Tasks

### Task 1: Expand `make bench` to run all benchmarks
**Files**: `Makefile`
**Estimated effort**: Small

Steps:
1. Update `bench` target to run `./...` instead of `./server/...`
2. Add machine identifier variable (`BENCH_MACHINE`)
3. Add benchmark directory variables (`BENCH_DIR`, `BENCH_BASELINE`, `BENCH_HISTORY_DIR`)
4. Test that all existing benchmarks run successfully

Tests:
- `make bench` runs lexer benchmarks (previously skipped)
- `make bench` runs server benchmarks (existing)
- Output includes benchmarks from multiple packages

---

### Task 2: Add `make bench-save` with history tracking
**Files**: `Makefile`, `work/benchmarks/.gitignore`, `work/benchmarks/README.md`
**Estimated effort**: Small

Steps:
1. Create `work/benchmarks/` directory structure
2. Create `.gitignore` to ignore all generated benchmark files
3. Create `README.md` documenting the benchmark workflow
4. Implement `bench-save` target that:
   - Runs benchmarks with `-count=5`
   - Saves to machine-specific baseline file
   - Copies to history directory with date-commit filename
5. Test on local machine

Tests:
- `make bench-save` creates `work/benchmarks/baseline-<hostname>.txt`
- `make bench-save` creates `work/benchmarks/history/<hostname>/<date>-<commit>.txt`
- Running twice creates two history files

---

### Task 3: Add `make bench-compare`
**Files**: `Makefile`
**Estimated effort**: Small

Steps:
1. Implement `bench-compare` target that:
   - Checks for existing baseline, exits with message if none
   - Runs current benchmarks with `-count=5`
   - Runs `benchstat` comparing baseline to current
2. Test comparison output

Tests:
- `make bench-compare` fails gracefully if no baseline exists
- `make bench-compare` produces `benchstat` output when baseline exists
- Output shows delta percentages and significance

---

### Task 4: Add `make bench-diff` (branch comparison)
**Files**: `Makefile`
**Estimated effort**: Small

Steps:
1. Implement `bench-diff` target that:
   - Runs benchmarks on current branch
   - Stashes uncommitted changes
   - Checks out main branch
   - Runs benchmarks on main
   - Restores original branch and stash
   - Runs `benchstat` comparing main vs current
2. Test with uncommitted changes
3. Test from feature branch

Tests:
- `make bench-diff` works from feature branch
- `make bench-diff` preserves uncommitted changes
- Output clearly labels "main vs current branch"

---

### Task 5: Add `make bench-history`
**Files**: `Makefile`
**Estimated effort**: Small

Steps:
1. Implement `bench-history` target that:
   - Checks for history directory, exits with message if none
   - Runs `benchstat` across last 5 history files
2. Test with multiple saved runs

Tests:
- `make bench-history` fails gracefully if no history
- `make bench-history` shows trends across multiple runs

---

### Task 6: Add `make bench-report`
**Files**: `Makefile`
**Estimated effort**: Medium

Steps:
1. Implement `bench-report` target that:
   - Runs `bench-save` to capture current state
   - Generates Markdown report with:
     - Header (date, machine, commit, branch)
     - Current benchmark results (code block)
     - Trend analysis from history (if available)
     - Footer noting AI should add analysis
   - Outputs to stdout and saves to `/tmp/bench-report.md`
2. Test report generation

Tests:
- `make bench-report` produces valid Markdown
- Report includes machine identifier
- Report includes trend section (or "insufficient history" message)

---

### Task 7: Add parser benchmarks
**Files**: `pkg/parsley/parser/parser_bench_test.go` (new)
**Estimated effort**: Medium

Steps:
1. Create benchmark file with representative test cases:
   - `BenchmarkParser_SimpleExpression` — literals, arithmetic
   - `BenchmarkParser_MediumComplexity` — function calls, conditionals
   - `BenchmarkParser_ComplexProgram` — nested structures, multiple statements
   - `BenchmarkParser_HTMLTags` — tag parsing (key Parsley workload)
2. Ensure benchmarks use `b.ReportAllocs()`
3. Verify benchmarks run successfully

Tests:
- All parser benchmarks pass
- Benchmarks appear in `make bench` output
- Results are reasonable (not obviously broken)

---

### Task 8: Add evaluator benchmarks
**Files**: `pkg/parsley/evaluator/evaluator_bench_test.go` (new)
**Estimated effort**: Medium

Steps:
1. Create benchmark file with representative test cases:
   - `BenchmarkEvaluator_Arithmetic` — basic math operations
   - `BenchmarkEvaluator_Strings` — string operations
   - `BenchmarkEvaluator_Arrays` — array creation, access, methods
   - `BenchmarkEvaluator_Dicts` — dict creation, access, methods
   - `BenchmarkEvaluator_Functions` — user-defined function calls
   - `BenchmarkEvaluator_Builtins` — builtin function calls
   - `BenchmarkEvaluator_ControlFlow` — conditionals, loops
   - `BenchmarkEvaluator_HTMLGeneration` — tag evaluation
2. Ensure benchmarks use `b.ReportAllocs()`
3. Verify benchmarks run successfully

Tests:
- All evaluator benchmarks pass
- Benchmarks appear in `make bench` output
- Results are reasonable (not obviously broken)

---

### Task 9: Update AGENTS.md with benchmark workflow
**Files**: `AGENTS.md`
**Estimated effort**: Small

Steps:
1. Add "Benchmark Workflow" section with:
   - Available `make` targets and their purposes
   - Post-feature benchmark requirement
   - How to interpret results
   - When to skip benchmarks
2. Add to "After Implementation" checklist

Tests:
- Documentation is clear and accurate
- Instructions match implemented behavior

---

### Task 10: Update copilot-instructions.md with post-FEAT requirement
**Files**: `.github/copilot-instructions.md`
**Estimated effort**: Small

Steps:
1. Add benchmark requirement to "After Implementation" section
2. Specify regression threshold (5%) for flagging
3. Add to Definition of Done

Tests:
- Instructions are clear
- Matches AGENTS.md guidance

---

### Task 11: Save initial baseline and validate workflow
**Files**: `work/benchmarks/history/...` (generated, gitignored)
**Estimated effort**: Small

Steps:
1. Run `make bench-save` to create initial baseline
2. Run `make bench-compare` to verify comparison works
3. Run `make bench-report` to verify report generation
4. Run `make bench-history` (will show single entry)
5. Document any issues encountered

Tests:
- Full workflow runs without errors
- All targets behave as documented

---

## Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] `make bench` runs all benchmarks (server, lexer, parser, evaluator)
- [ ] `make bench-save` creates baseline and history files
- [ ] `make bench-compare` compares against baseline
- [ ] `make bench-diff` compares current branch vs main
- [ ] `make bench-history` shows trends
- [ ] `make bench-report` generates Markdown report
- [ ] AGENTS.md updated with benchmark workflow
- [ ] copilot-instructions.md updated with post-FEAT requirement
- [ ] work/BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-03-14 | Task 1: Expand make bench | ✅ Complete | Runs ./... instead of ./server/... |
| 2026-03-14 | Task 2: Add bench-save | ✅ Complete | Machine-specific baselines + history |
| 2026-03-14 | Task 3: Add bench-compare | ✅ Complete | Added BENCHSTAT variable for PATH issues |
| 2026-03-14 | Task 4: Add bench-diff | ✅ Complete | Compares current branch vs main |
| 2026-03-14 | Task 5: Add bench-history | ✅ Complete | Shows trends across recent runs |
| 2026-03-14 | Task 6: Add bench-report | ✅ Complete | Generates Markdown report |
| 2026-03-14 | Task 7: Parser benchmarks | ✅ Complete | 6 benchmarks in parser_bench_test.go |
| 2026-03-14 | Task 8: Evaluator benchmarks | ✅ Complete | 11 benchmarks in evaluator_bench_test.go |
| 2026-03-14 | Task 9: Update AGENTS.md | ✅ Complete | Added benchmark workflow section |
| 2026-03-14 | Task 10: Update copilot-instructions | ✅ Complete | Added post-FEAT benchmark requirement |
| 2026-03-14 | Task 11: Initial baseline | ✅ Complete | Validated full workflow on sams-macbook-air |

## Deferred Items
Items to add to work/BACKLOG.md after implementation:
- CI integration for benchmarks — Run benchmarks on PRs with CI-specific baseline
- Benchmark visualization — Generate charts from historical data
- Auto-pruning of old history — If storage becomes a concern