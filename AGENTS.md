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

### Lint
```bash
golangci-lint run
```

### Full Validation (run before committing)
```bash
make check
# Or: go build -o basil ./cmd/basil && go build -o pars ./cmd/pars && go test ./...
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
├── ID_COUNTER.md                # Auto-incrementing ID tracker
├── BACKLOG.md                   # Deferred items
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
└── docs/
    ├── guide/                   # Human-friendly documentation
    │   ├── README.md
    │   ├── quick-start.md
    │   ├── cheatsheet.md
    │   ├── faq.md
    │   └── walkthroughs/
    ├── specs/                   # Feature specifications (FEAT-XXX.md)
    ├── plans/                   # Implementation plans
    ├── bugs/                    # Bug reports (BUG-XXX.md)
    └── decisions/               # Architecture Decision Records
```

## Workflow Rules

### Before Starting Any Task
1. Read the relevant spec/bug report
2. Check BACKLOG.md for related deferred items
3. Use the appropriate prompt file (`/new-feature` or `/fix-bug`)

### During Implementation
- Commit frequently with conventional commit messages
- Update implementation plan progress log
- Run tests after each significant change

### After Implementation
- Update BACKLOG.md with any deferred items
- Update spec/bug with implementation notes
- Ensure all tests pass

## ID Conventions
- Features: `FEAT-001`, `FEAT-002`, ...
- Bugs: `BUG-001`, `BUG-002`, ...
- Plans: `PLAN-001` (linked to FEAT or BUG)
- Decisions: `ADR-001`, `ADR-002`, ...

## ID Allocation Rules
- Always read `ID_COUNTER.md` before creating new specs/bugs
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
2. If not already in `docs/guide/faq.md`, add it
3. If it reveals a gap in walkthroughs, note in BACKLOG.md

## Common Pitfalls
- Always run `go mod tidy` after adding dependencies
- The `internal/` directory is not importable externally
- Tests must not depend on external services

## Trust These Instructions
Follow the instructions in this file and referenced procedure documents. Only search the codebase if instructions are incomplete or incorrect.
