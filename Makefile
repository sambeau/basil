# Basil Makefile

# Version info from git
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)"

# Default target
.PHONY: all
all: build

# Build with version info
.PHONY: build
build:
	go build $(LDFLAGS) -o basil .

# Development build (no version injection, faster)
.PHONY: dev
dev:
	go build -o basil .

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
	rm -f basil

# Show version that would be embedded
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"

# Install to GOPATH/bin
.PHONY: install
install:
	go install $(LDFLAGS) .

.PHONY: help
help:
	@echo "Basil build targets:"
	@echo "  make build   - Build with version info (default)"
	@echo "  make dev     - Quick build without version injection"
	@echo "  make test    - Run tests"
	@echo "  make check   - Build and test"
	@echo "  make clean   - Remove build artifacts"
	@echo "  make version - Show version that would be embedded"
	@echo "  make install - Install to GOPATH/bin"
