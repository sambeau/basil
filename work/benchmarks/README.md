# Benchmark Workflow

This directory contains benchmark baselines and history for performance tracking.

## Quick Reference

| Command | Purpose |
|---------|---------|
| `make bench` | Run all benchmarks (quick, single iteration) |
| `make bench-save` | Save benchmarks as baseline for this machine |
| `make bench-compare` | Compare current performance vs saved baseline |
| `make bench-diff` | Compare current branch vs main branch |
| `make bench-history` | Show trends across recent benchmark runs |
| `make bench-report` | Generate a Markdown performance report |

## How It Works

### Machine-Specific Baselines

Benchmark results vary significantly across machines due to CPU, memory, thermal throttling, and other factors. Cross-machine comparison is not meaningful.

Each machine maintains its own baseline and history:

```
work/benchmarks/
├── baseline-macbook-m2.txt      # Current baseline for this machine
├── baseline-ci-runner.txt       # CI's baseline (if configured)
└── history/
    ├── macbook-m2/
    │   ├── 2026-03-14-abc123.txt
    │   └── 2026-03-10-def456.txt
    └── ci-runner/
        └── ...
```

### Workflow

**During development:**
```bash
# Quick benchmark to see current performance
make bench

# Compare your changes against the baseline
make bench-compare

# Compare your branch against main
make bench-diff
```

**After completing a feature:**
```bash
# Save new baseline and add to history
make bench-save
```

**To analyze trends:**
```bash
# View performance over recent runs
make bench-history

# Generate full report
make bench-report
```

## For AI Agents

After implementing any FEAT:

1. Run `make bench-save` to capture post-implementation performance
2. Run `make bench-compare` to check for regressions
3. Include performance summary in implementation notes:
   - Flag regressions > 5% for human review
   - Note improvements > 5%
   - State "No performance regression" if stable

When asked to produce a performance report:

1. Run `make bench-report` to generate raw data
2. Analyze the output and add:
   - Executive summary with human-readable assessment
   - Interpretation of trends
   - Correlation with recent code changes (check git log)
   - Specific recommendations
3. Save enhanced report to `work/reports/PERF-REPORT-YYYY-MM-DD.md`

## Prerequisites

Install `benchstat` for comparison features:

```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

## File Storage

- All baseline and history files are git-ignored (machine-specific)
- Only this README and `.gitignore` are tracked
- History is kept indefinitely (files are small, ~10-20KB each)