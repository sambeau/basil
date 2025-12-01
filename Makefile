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
