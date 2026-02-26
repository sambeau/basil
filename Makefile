# Basil Makefile

# Version info from git
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)"

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
	@echo "  make build       - Build basil and pars with version info (default)"
	@echo "  make build-basil - Build basil only"
	@echo "  make build-pars  - Build pars only"
	@echo "  make dev         - Quick build without version injection"
	@echo "  make test        - Run tests"
	@echo "  make check       - Build and test"
	@echo "  make clean       - Remove build artifacts"
	@echo "  make version     - Show version that would be embedded"
	@echo "  make install     - Install to GOPATH/bin"
	@echo "  make docs        - Generate API reference documentation"
	@echo "  make docs-full   - Generate full reference from template"
	@echo "  make verify-docs - Verify generated docs are up to date (for CI)"
