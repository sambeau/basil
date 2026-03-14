# Basil Makefile

# Version info from git
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)"

# Benchmark configuration
BENCHSTAT := $(shell which benchstat 2>/dev/null || echo "$(shell go env GOPATH)/bin/benchstat")
BENCH_MACHINE := $(shell hostname -s 2>/dev/null | tr '[:upper:]' '[:lower:]' || echo "unknown")
BENCH_DIR := work/benchmarks
BENCH_BASELINE := $(BENCH_DIR)/baseline-$(BENCH_MACHINE).txt
BENCH_HISTORY_DIR := $(BENCH_DIR)/history/$(BENCH_MACHINE)
BENCH_COUNT := 5

# Default target
.PHONY: all
all: build

# Build both CLIs with version info
.PHONY: build
build: build-basil build-pars

.PHONY: build-basil
build-basil:
	go build $(LDFLAGS) -o basil ./cmd/basil

.PHONY: build-pars
build-pars:
	go build $(LDFLAGS) -o pars ./cmd/pars

# Development build (no version injection, faster)
.PHONY: dev
dev:
	go build -o basil ./cmd/basil
	go build -o pars ./cmd/pars

# Run tests
.PHONY: test
test:
	go test ./...

# Run all benchmarks (quick, single iteration)
.PHONY: bench
bench:
	go test -run '^$$' -bench=. -benchmem ./...

# Save benchmark results as new baseline for this machine
.PHONY: bench-save
bench-save:
	@mkdir -p $(BENCH_HISTORY_DIR)
	@echo "Running benchmarks with $(BENCH_COUNT) iterations..."
	@go test -run '^$$' -bench=. -benchmem -count=$(BENCH_COUNT) ./... | tee $(BENCH_BASELINE)
	@cp $(BENCH_BASELINE) "$(BENCH_HISTORY_DIR)/$$(date +%Y-%m-%d)-$(COMMIT).txt"
	@echo ""
	@echo "Saved baseline to $(BENCH_BASELINE)"
	@echo "Saved history to $(BENCH_HISTORY_DIR)/"

# Compare current benchmarks against this machine's baseline
.PHONY: bench-compare
bench-compare:
	@if [ ! -f $(BENCH_BASELINE) ]; then \
		echo "No baseline found for this machine ($(BENCH_MACHINE))."; \
		echo "Run 'make bench-save' first to create a baseline."; \
		exit 1; \
	fi
	@echo "Running benchmarks with $(BENCH_COUNT) iterations..."
	@go test -run '^$$' -bench=. -benchmem -count=$(BENCH_COUNT) ./... > /tmp/bench-current.txt
	@echo ""
	@echo "=== Comparison: baseline vs current ==="
	@$(BENCHSTAT) $(BENCH_BASELINE) /tmp/bench-current.txt

# Compare current branch against main (same machine, no stored file)
.PHONY: bench-diff
bench-diff:
	@echo "Benchmarking current branch..."
	@go test -run '^$$' -bench=. -benchmem -count=$(BENCH_COUNT) ./... > /tmp/bench-current.txt
	@echo "Stashing any uncommitted changes..."
	@git stash --quiet --include-untracked 2>/dev/null || true
	@echo "Checking out main and benchmarking..."
	@git checkout main --quiet
	@go test -run '^$$' -bench=. -benchmem -count=$(BENCH_COUNT) ./... > /tmp/bench-main.txt
	@echo "Restoring original branch..."
	@git checkout - --quiet
	@git stash pop --quiet 2>/dev/null || true
	@echo ""
	@echo "=== Comparison: main vs current branch ==="
	@$(BENCHSTAT) /tmp/bench-main.txt /tmp/bench-current.txt

# Show trends across recent benchmark runs
.PHONY: bench-history
bench-history:
	@if [ ! -d $(BENCH_HISTORY_DIR) ]; then \
		echo "No history found for this machine ($(BENCH_MACHINE))."; \
		echo "Run 'make bench-save' to start tracking."; \
		exit 1; \
	fi
	@HIST_COUNT=$$(ls -1 $(BENCH_HISTORY_DIR)/*.txt 2>/dev/null | wc -l | tr -d ' '); \
	if [ "$$HIST_COUNT" -eq 0 ]; then \
		echo "No history files found. Run 'make bench-save' to start tracking."; \
		exit 1; \
	fi
	@echo "=== Benchmark history for $(BENCH_MACHINE) (last 5 runs) ==="
	@ls -1t $(BENCH_HISTORY_DIR)/*.txt | head -5 | xargs $(BENCHSTAT)

# Generate performance report
.PHONY: bench-report
bench-report:
	@echo "Generating performance report..."
	@mkdir -p $(BENCH_DIR)
	@$(MAKE) --no-print-directory bench-save
	@echo ""
	@echo "# Performance Report - $$(date +%Y-%m-%d)" > /tmp/bench-report.md
	@echo "" >> /tmp/bench-report.md
	@echo "**Machine:** $(BENCH_MACHINE)" >> /tmp/bench-report.md
	@echo "**Commit:** $(COMMIT)" >> /tmp/bench-report.md
	@echo "**Branch:** $$(git branch --show-current)" >> /tmp/bench-report.md
	@echo "" >> /tmp/bench-report.md
	@echo "## Current Benchmarks" >> /tmp/bench-report.md
	@echo "" >> /tmp/bench-report.md
	@echo '```' >> /tmp/bench-report.md
	@cat $(BENCH_BASELINE) >> /tmp/bench-report.md
	@echo '```' >> /tmp/bench-report.md
	@echo "" >> /tmp/bench-report.md
	@HIST_COUNT=$$(ls -1 $(BENCH_HISTORY_DIR)/*.txt 2>/dev/null | wc -l | tr -d ' '); \
	if [ "$$HIST_COUNT" -gt 1 ]; then \
		echo "## Trend Analysis (last 5 runs)" >> /tmp/bench-report.md; \
		echo "" >> /tmp/bench-report.md; \
		echo '```' >> /tmp/bench-report.md; \
		ls -1t $(BENCH_HISTORY_DIR)/*.txt | head -5 | xargs $(BENCHSTAT) >> /tmp/bench-report.md 2>&1; \
		echo '```' >> /tmp/bench-report.md; \
	else \
		echo "## Trend Analysis" >> /tmp/bench-report.md; \
		echo "" >> /tmp/bench-report.md; \
		echo "Insufficient history for trend analysis. Run \`make bench-save\` multiple times to build history." >> /tmp/bench-report.md; \
	fi
	@echo "" >> /tmp/bench-report.md
	@echo "---" >> /tmp/bench-report.md
	@echo "" >> /tmp/bench-report.md
	@echo "*Report generated by \`make bench-report\`. AI agents should analyze and add commentary.*" >> /tmp/bench-report.md
	@echo ""
	@echo "=========================================="
	@cat /tmp/bench-report.md
	@echo "=========================================="
	@echo ""
	@echo "Report saved to /tmp/bench-report.md"
	@echo "To save permanently: cp /tmp/bench-report.md work/reports/PERF-REPORT-$$(date +%Y-%m-%d).md"

# Build and test (full validation)
.PHONY: check
check: build test

# Clean build artifacts
.PHONY: clean
clean:
	rm -f basil pars

# Show version that would be embedded
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"

# Install to GOPATH/bin
.PHONY: install
install:
	go install $(LDFLAGS) ./cmd/basil
	go install $(LDFLAGS) ./cmd/pars

# Verify generated docs are up to date
.PHONY: verify-docs
verify-docs: build-pars
	@echo "Verifying API reference is up to date..."
	@./pars reference --verify docs/parsley/api-reference.md || \
		(echo "Run 'make docs' to regenerate" && exit 1)

# Generate API reference documentation (API-only)
.PHONY: docs
docs: build-pars
	@echo "Generating API reference..."
	@./pars reference > docs/parsley/api-reference.md
	@echo "Generated docs/parsley/api-reference.md"

# Generate full reference from template (fragments + generated)
.PHONY: docs-full
docs-full: build-pars
	@echo "Generating full reference from template..."
	@./pars reference --template docs/parsley/reference.tmpl.md > docs/parsley/generated-reference.md
	@echo "Generated docs/parsley/generated-reference.md"

.PHONY: help
help:
	@echo "Basil build targets:"
	@echo "  make build         - Build basil and pars with version info (default)"
	@echo "  make build-basil   - Build basil only"
	@echo "  make build-pars    - Build pars only"
	@echo "  make dev           - Quick build without version injection"
	@echo "  make test          - Run tests"
	@echo "  make check         - Build and test"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make version       - Show version that would be embedded"
	@echo "  make install       - Install to GOPATH/bin"
	@echo "  make docs          - Generate API reference documentation"
	@echo "  make docs-full     - Generate full reference from template"
	@echo "  make verify-docs   - Verify generated docs are up to date (for CI)"
	@echo ""
	@echo "Benchmark targets:"
	@echo "  make bench         - Run all benchmarks (quick, single iteration)"
	@echo "  make bench-save    - Save benchmarks as baseline for this machine"
	@echo "  make bench-compare - Compare current vs saved baseline"
	@echo "  make bench-diff    - Compare current branch vs main"
	@echo "  make bench-history - Show trends across recent runs"
	@echo "  make bench-report  - Generate performance report (Markdown)"
