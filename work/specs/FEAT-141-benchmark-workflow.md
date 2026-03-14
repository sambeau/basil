---
id: FEAT-141
title: "Repeatable Benchmark Workflow with Regression Guards"
status: draft
priority: medium
created: 2026-03-14
author: "@human"
---

# FEAT-141: Repeatable Benchmark Workflow with Regression Guards

## Summary
Establish a comprehensive, repeatable benchmark workflow that covers all performance-critical code paths and enables regression detection. This addresses the "higher leverage" recommendations from the performance analysis report (BASIL-PERFORMANCE-ANALYSIS-2026-03-10.md) to treat benchmarks as regression guards for future work.

## User Story
As a developer working on Basil, I want a simple benchmark workflow that captures baselines and detects regressions so that I can confidently make changes without accidentally degrading performance.

## Acceptance Criteria
- [ ] `make bench` runs all benchmarks across the project (server, lexer, parser, evaluator)
- [ ] `make bench-save` saves benchmark results to a machine-specific baseline file
- [ ] `make bench-compare` compares current results against the saved baseline using `benchstat`
- [ ] `make bench-diff` compares current branch against main branch (same machine, no stored file)
- [ ] `make bench-history` shows trends across recent benchmark runs
- [ ] `make bench-report` generates a structured performance report (Markdown)
- [ ] Parser benchmarks exist covering representative workloads
- [ ] Evaluator benchmarks exist covering representative workloads
- [ ] Documentation explains how to use the benchmark workflow
- [ ] AI agent instructions updated to run benchmarks after each FEAT implementation

## Design Decisions

- **Use `benchstat` for comparison**: Standard Go tool, already familiar to Go developers, provides statistical comparison with confidence intervals
- **Per-machine baselines**: Benchmark results are machine-specific; storing per-machine baselines allows each developer to track regressions on their own hardware without false positives from machine differences
- **Historical tracking with timestamped files**: Each `bench-save` creates a timestamped file, enabling trend analysis over time while keeping a current baseline for quick comparison
- **Same-machine comparison is primary**: The most valuable comparison is before/after on the same machine; cross-machine comparison of absolute numbers is not meaningful
- **Single `make bench` for all packages**: Consistency and simplicity over granular control; developers can still run `go test -bench` directly for targeted runs
- **Baseline files in `.gitignore`**: Machine-specific baselines are local development artifacts, not shared via git

---

## Machine-Specific Baseline Strategy

### The Problem
Benchmark results vary significantly across machines due to:
- CPU architecture and speed
- Memory bandwidth
- Thermal throttling
- Background processes
- Power settings

Comparing benchmarks across different machines produces misleading results.

### The Solution: Hybrid Approach

| Use Case | Approach |
|----------|----------|
| Local development | Compare before/after on same machine (`make bench-diff`) |
| Personal regression tracking | Per-machine baselines with hostname in filename |
| Historical trends | Timestamped files per machine |
| CI (future) | CI runner maintains its own baseline |

### File Structure

```
work/benchmarks/
├── .gitignore                           # Ignore all baseline files
├── README.md                            # Documents the workflow
└── (generated files, git-ignored):
    ├── baseline-macbook-m2.txt          # Current baseline for this machine
    ├── baseline-ci-runner.txt           # CI's baseline
    ├── history/
    │   ├── macbook-m2/
    │   │   ├── 2026-03-14-abc123.txt    # Historical runs (date-commit)
    │   │   ├── 2026-03-10-def456.txt
    │   │   └── 2026-03-01-789ghi.txt
    │   └── ci-runner/
    │       └── ...
```

### Retention Policy
- Keep all historical benchmark files (files are small, ~10-20KB each)
- Revisit if storage becomes a concern

---

## AI Agent Workflow

### Post-Feature Benchmark Requirement

After completing implementation of any FEAT (feature), AI agents MUST:

1. **Run benchmarks and save**: Execute `make bench-save` to capture post-implementation performance
2. **Compare against baseline**: Execute `make bench-compare` to detect regressions
3. **Report findings**: Include a brief performance summary in the commit message or implementation notes:
   - Any regressions > 5% (flag for human review)
   - Any improvements > 5% (note as wins)
   - If no significant change, state "No performance regression detected"

### When to Skip
- Documentation-only changes
- Test-only changes (unless testing performance-critical code)
- Configuration changes that don't affect runtime

### Instructions to Add to `AGENTS.md`

```markdown
## Benchmark Workflow (Post-Feature)

After implementing any FEAT:
1. Run `make bench-save` to capture performance baseline
2. Run `make bench-compare` to check for regressions
3. Include performance summary in implementation notes:
   - Flag regressions > 5% for human review
   - Note improvements > 5%
   - State "No performance regression" if stable
4. If significant regression detected, investigate before committing
```

---

## Performance Reporting

### On-Demand Performance Report

AI agents can generate a comprehensive performance report at any time using `make bench-report`. This produces a structured Markdown report suitable for inclusion in `work/reports/`.

### Report Contents

The performance report should include:

1. **Executive Summary**
   - Overall performance status (healthy/degraded/unknown)
   - Key metrics at a glance
   - Notable changes since last report

2. **Current Performance**
   - Latest benchmark results (tabular)
   - Comparison against baseline
   - Allocation counts for key operations

3. **Trend Analysis**
   - Performance over last N runs (from history)
   - Benchmarks trending worse (> 5% degradation over time)
   - Benchmarks trending better (> 5% improvement over time)
   - Stable benchmarks

4. **Hotspots & Concerns**
   - Highest allocation operations
   - Slowest operations (absolute time)
   - Operations with high variability

5. **Possible Causes** (AI-analyzed)
   - Recent commits that may have affected performance
   - Correlation between code changes and benchmark changes
   - Suggested areas for investigation

6. **Recommendations**
   - Priority items to investigate
   - Quick wins if any are apparent
   - Suggested next steps

### Report Generation

```makefile
# Generate performance report
.PHONY: bench-report
bench-report:
	@echo "Generating performance report..."
	@mkdir -p work/reports
	@$(MAKE) bench-save
	@echo "# Performance Report - $$(date +%Y-%m-%d)" > /tmp/bench-report.md
	@echo "" >> /tmp/bench-report.md
	@echo "Machine: $(BENCH_MACHINE)" >> /tmp/bench-report.md
	@echo "Commit: $$(git rev-parse --short HEAD)" >> /tmp/bench-report.md
	@echo "Branch: $$(git branch --show-current)" >> /tmp/bench-report.md
	@echo "" >> /tmp/bench-report.md
	@echo "## Current Benchmarks" >> /tmp/bench-report.md
	@echo '```' >> /tmp/bench-report.md
	@cat $(BENCH_BASELINE) >> /tmp/bench-report.md
	@echo '```' >> /tmp/bench-report.md
	@echo "" >> /tmp/bench-report.md
	@if [ -d $(BENCH_HISTORY_DIR) ] && [ "$$(ls -1 $(BENCH_HISTORY_DIR)/*.txt 2>/dev/null | wc -l)" -gt 1 ]; then \
		echo "## Trend Analysis (last 5 runs)" >> /tmp/bench-report.md; \
		echo '```' >> /tmp/bench-report.md; \
		ls -1t $(BENCH_HISTORY_DIR)/*.txt | head -5 | xargs benchstat >> /tmp/bench-report.md 2>&1; \
		echo '```' >> /tmp/bench-report.md; \
	else \
		echo "## Trend Analysis" >> /tmp/bench-report.md; \
		echo "Insufficient history for trend analysis. Run bench-save multiple times to build history." >> /tmp/bench-report.md; \
	fi
	@echo "" >> /tmp/bench-report.md
	@echo "---" >> /tmp/bench-report.md
	@echo "*Report generated by \`make bench-report\`. AI agents should analyze and add commentary.*" >> /tmp/bench-report.md
	@cat /tmp/bench-report.md
	@echo ""
	@echo "Report saved to /tmp/bench-report.md"
	@echo "To save permanently: cp /tmp/bench-report.md work/reports/PERF-REPORT-$$(date +%Y-%m-%d).md"
```

### AI Agent Report Workflow

When asked to produce a performance report, AI agents should:

1. Run `make bench-report` to generate the raw data
2. Analyze the output and add:
   - Executive summary with human-readable assessment
   - Interpretation of trends (not just raw numbers)
   - Correlation with recent code changes (check git log)
   - Specific recommendations based on findings
3. Save the enhanced report to `work/reports/PERF-REPORT-YYYY-MM-DD.md`

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Current State
- `make bench` exists but only runs `./server/...` benchmarks
- Server benchmarks are comprehensive: API handlers, caches, Parsley handlers, root handler
- Lexer benchmarks exist in `pkg/parsley/lexer/lexer_bench_test.go`
- No parser benchmarks exist
- No evaluator benchmarks exist
- No baseline comparison workflow exists

### Affected Components
- `Makefile` — Expand `bench` target, add `bench-save`, `bench-compare`, `bench-diff`, `bench-history` targets
- `pkg/parsley/parser/parser_bench_test.go` — New file with parser benchmarks
- `pkg/parsley/evaluator/evaluator_bench_test.go` — New file with evaluator benchmarks
- `work/benchmarks/.gitignore` — Ignore generated baseline files
- `work/benchmarks/README.md` — Document the benchmark workflow
- `AGENTS.md` — Add benchmark workflow to developer docs

### Dependencies
- Depends on: None
- Blocks: None
- Related: BASIL-PERFORMANCE-ANALYSIS-2026-03-10.md recommendations

### Benchmark Coverage Requirements

**Parser benchmarks should cover:**
1. Simple expressions (literals, arithmetic)
2. Medium complexity (function calls, conditionals)
3. Complex programs (nested structures, multiple statements)
4. HTML/tag parsing (representative Parsley workload)

**Evaluator benchmarks should cover:**
1. Arithmetic and basic operations
2. String operations
3. Array/dict operations
4. Function calls (user-defined and builtins)
5. Control flow (conditionals, loops)
6. HTML generation (tag evaluation)

### Makefile Targets

```makefile
# Machine identifier for per-machine baselines
BENCH_MACHINE := $(shell hostname -s | tr '[:upper:]' '[:lower:]')
BENCH_DIR := work/benchmarks
BENCH_BASELINE := $(BENCH_DIR)/baseline-$(BENCH_MACHINE).txt
BENCH_HISTORY_DIR := $(BENCH_DIR)/history/$(BENCH_MACHINE)
BENCH_COUNT := 5

# Run all benchmarks (quick, single iteration)
.PHONY: bench
bench:
	go test -run '^$$' -bench=. -benchmem ./...

# Save benchmark results as new baseline for this machine
.PHONY: bench-save
bench-save:
	@mkdir -p $(BENCH_HISTORY_DIR)
	go test -run '^$$' -bench=. -benchmem -count=$(BENCH_COUNT) ./... | tee $(BENCH_BASELINE)
	@cp $(BENCH_BASELINE) $(BENCH_HISTORY_DIR)/$$(date +%Y-%m-%d)-$$(git rev-parse --short HEAD).txt
	@echo "Saved baseline to $(BENCH_BASELINE)"
	@echo "Saved history to $(BENCH_HISTORY_DIR)/"

# Compare current benchmarks against this machine's baseline
.PHONY: bench-compare
bench-compare:
	@if [ ! -f $(BENCH_BASELINE) ]; then \
		echo "No baseline found for this machine. Run 'make bench-save' first."; \
		exit 1; \
	fi
	go test -run '^$$' -bench=. -benchmem -count=$(BENCH_COUNT) ./... > /tmp/bench-current.txt
	benchstat $(BENCH_BASELINE) /tmp/bench-current.txt

# Compare current branch against main (same machine, no stored file)
.PHONY: bench-diff
bench-diff:
	@echo "Benchmarking current branch..."
	go test -run '^$$' -bench=. -benchmem -count=$(BENCH_COUNT) ./... > /tmp/bench-current.txt
	@echo "Checking out main and benchmarking..."
	git stash --quiet --include-untracked || true
	git checkout main --quiet
	go test -run '^$$' -bench=. -benchmem -count=$(BENCH_COUNT) ./... > /tmp/bench-main.txt
	git checkout - --quiet
	git stash pop --quiet 2>/dev/null || true
	@echo ""
	@echo "=== Comparison (main vs current branch) ==="
	benchstat /tmp/bench-main.txt /tmp/bench-current.txt

# Show trends across recent benchmark runs
.PHONY: bench-history
bench-history:
	@if [ ! -d $(BENCH_HISTORY_DIR) ]; then \
		echo "No history found for this machine. Run 'make bench-save' to start tracking."; \
		exit 1; \
	fi
	@echo "=== Benchmark history for $(BENCH_MACHINE) ==="
	@ls -1t $(BENCH_HISTORY_DIR)/*.txt 2>/dev/null | head -5 | xargs benchstat
```

### Edge Cases & Constraints
1. **Benchmark variability** — Use `-count=5` or higher for statistical significance when comparing
2. **Machine differences** — Baselines are machine-specific; cross-machine comparison is not meaningful
3. **Package test failures** — `go test -bench ./...` will fail if any package has test failures; ensure tests pass first
4. **benchstat availability** — Developers need `go install golang.org/x/perf/cmd/benchstat@latest`
5. **bench-diff with uncommitted changes** — Uses `git stash` to preserve work; may fail if stash conflicts occur
6. **Hostname stability** — Machine identifier uses `hostname -s`; if hostname changes, a new baseline series starts
7. **AI agent benchmark runs** — AI should not block on benchmark failures; report and continue
8. **Report generation** — Raw report from Makefile needs AI analysis to be useful; Makefile provides data, AI provides insight

## Implementation Notes
*Added during/after implementation*

## Related
- Report: `work/reports/BASIL-PERFORMANCE-ANALYSIS-2026-03-10.md` (lines 729-732)
- Plan: `work/plans/FEAT-141-plan.md` (to be created)
- Update required: `AGENTS.md` (add benchmark workflow instructions)
- Update required: `.github/copilot-instructions.md` (add post-FEAT benchmark requirement)