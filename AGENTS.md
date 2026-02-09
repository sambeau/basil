# Agent Instructions for Basil

## Project Overview
Basil is a Go web server for the Parsley programming language.

**Repository size:** Medium (monorepo)
**Primary language:** Go 1.24+
**Architecture:** Single module, monorepo with Parsley

## Build & Validation Commands

### Build (with version)
```bash
make build
# Or manually:
# go build -ldflags "-X main.Version=$(git describe --tags --always) -X main.Commit=$(git rev-parse --short HEAD)" -o basil ./cmd/basil
# go build -ldflags "-X main.Version=$(git describe --tags --always)" -o pars ./cmd/pars
```

### Quick Build (development)
```bash
make dev
# Or: go build -o basil ./cmd/basil && go build -o pars ./cmd/pars
```

### Test
```bash
make test
# Or: go test ./...
```

### Lint (new issues only)
```bash
golangci-lint run
```
This uses the baseline in `.golangci.yml` to only report issues in new/changed code.

### Lint (all issues)
```bash
golangci-lint run --new-from-rev=""
```

### Full Validation (run before committing)
```bash
make check
# Or: go build -o basil ./cmd/basil && go build -o pars ./cmd/pars && go test ./...
```

## Linting Guidelines

**Run the linter regularly** — at minimum before each commit and after significant changes.

### Incremental Linting (Default)
The project uses a baseline commit in `.golangci.yml` (`new-from-rev`). By default, `golangci-lint run` only reports issues introduced after the baseline. This allows gradual improvement without requiring all legacy issues to be fixed.

### What to Fix
- **Always fix**: Issues in code you wrote or modified
- **Encouraged**: Fixing nearby legacy issues while you're in a file
- **Don't block on**: Legacy issues in unrelated code

### Enabled Linters
- `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused` (defaults)
- `modernize` — Modern Go idioms (any, min/max, range int, slices/maps)
- `gocritic` — Style and diagnostic checks

### Updating the Baseline
After a bulk cleanup of legacy issues, update the baseline in `.golangci.yml`:
```yaml
issues:
  new-from-rev: <new-commit-sha>
```

## Project Structure
```
basil/
├── cmd/
│   ├── basil/                   # Basil server CLI entry point
│   │   └── main.go
│   └── pars/                    # Parsley CLI entry point
│       └── main.go
├── pkg/
│   └── parsley/                 # Parsley language implementation
│       ├── ast/                 # Abstract syntax tree
│       ├── evaluator/           # Interpreter
│       ├── lexer/               # Tokenizer
│       ├── parser/              # Parser
│       ├── parsley/             # High-level API
│       └── ...                  # Other packages
├── server/                      # Basil server logic
├── auth/                        # Basil authentication
├── config/                      # Basil configuration
├── AGENTS.md                    # This file - agent operational context
├── CHANGELOG.md                 # Release history
├── .github/
│   ├── copilot-instructions.md  # Repository-wide AI instructions
│   ├── instructions/            # Always-on AI rules
│   │   ├── code.instructions.md
│   │   └── commits.instructions.md
│   ├── prompts/                 # Workflow prompts
│   │   ├── new-feature.prompt.md
│   │   ├── fix-bug.prompt.md
│   │   └── release.prompt.md
│   └── templates/               # Document templates
│       ├── FEATURE_SPEC.md
│       ├── BUG_REPORT.md
│       └── IMPLEMENTATION_PLAN.md
├── docs/
│   ├── guide/                   # User guides for Basil framework
│   │   ├── README.md
│   │   ├── basil-quick-start.md
│   │   ├── cheatsheet.md
│   │   └── faq.md
│   └── parsley/                 # Parsley language reference
│       ├── README.md
│       └── manual/              # Builtins and stdlib docs
└── work/
    ├── docs/                    # Workflow process guides    │   ├── workflow-quick-start.md
    │   └── workflow-faq.md    ├── specs/                   # Feature specifications (FEAT-XXX.md)
    ├── plans/                   # Implementation plans (PLAN-XXX.md)
    ├── bugs/                    # Bug reports (BUG-XXX.md)
    ├── design/                  # Design documents
    ├── reports/                 # Audits and investigations
    ├── parsley/                 # Parsley implementation docs
    ├── ID_COUNTER.md            # Auto-incrementing ID tracker
    └── BACKLOG.md               # Deferred items
```

## Workflow Rules

### Before Starting Any Task
1. Read the relevant spec/bug report from work/
2. Check work/BACKLOG.md for related deferred items
3. Use the appropriate prompt file (`/new-feature` or `/fix-bug`)

### During Implementation
- Commit frequently with conventional commit messages
- Update implementation plan progress log
- Run tests after each significant change

### After Implementation
- Update work/BACKLOG.md with any deferred items
- Update spec/bug with implementation notes
- Ensure all tests pass

## ID Conventions
- Features: `FEAT-001`, `FEAT-002`, ...
- Bugs: `BUG-001`, `BUG-002`, ...
- Plans: `PLAN-001` (linked to FEAT or BUG)

## ID Allocation Rules
- Always read `work/ID_COUNTER.md` before creating new specs/bugs
- Always update the counter immediately after creating the document
- Use zero-padded 3-digit format: `001`, `002`, ... `999`
- If counter shows `999`, alert human (unlikely but handles edge case)
- Never reuse IDs, even for deleted/abandoned items

## Git Workflow
- Feature branches: `feat/FEAT-XXX-short-description`
- Bug fix branches: `fix/BUG-XXX-short-description`
- AI commits to feature branches
- AI merges to main after human approval
- AI creates release tags

## Documentation Updates
When answering a "how do I..." question from human:
1. Answer the question
2. Add to appropriate FAQ:
   - **Workflow/process questions**: `work/docs/workflow-faq.md`
   - **Basil/Parsley usage**: `docs/guide/faq.md`
3. If it reveals a gap in walkthroughs, note in work/BACKLOG.md

## Common Pitfalls
- Always run `go mod tidy` after adding dependencies
- The `internal/` directory is not importable externally
- Tests must not depend on external services
- Run `golangci-lint run` before committing to catch issues early

## Trust These Instructions
Follow the instructions in this file and referenced procedure documents. Only search the codebase if instructions are incomplete or incorrect.
