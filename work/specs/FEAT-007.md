---
id: FEAT-007
title: "Merge Parsley into Basil Monorepo"
status: implemented
priority: medium
created: 2025-12-01
author: "@human"
---

# FEAT-007: Merge Parsley into Basil Monorepo

## Summary
Bring the Parsley language implementation into the Basil repository as a subtree/subdirectory, eliminating the need for separate repo management and `go.mod` replace directives.

## User Story
As a maintainer of both Basil and Parsley, I want a single repository so that I can make atomic changes across both codebases, avoid version sync issues, and simplify the development workflow.

## Acceptance Criteria
- [ ] Parsley source code lives in `pkg/parsley/` within the Basil repo
- [ ] Basil CLI moved to `cmd/basil/` (standard Go project layout)
- [ ] Parsley CLI lives in `cmd/pars/`
- [ ] All Basil imports updated to use `github.com/sambeau/basil/pkg/parsley/...`
- [ ] `go.mod` no longer needs `replace` directive for Parsley
- [ ] `go build ./cmd/basil` and `go build ./cmd/pars` both work
- [ ] `go test ./...` passes
- [ ] Git history preserved (nice to have, not required)
- [ ] CI/tests pass
- [ ] Documentation updated
- [ ] Makefile updated with new build paths

## Design Decisions
- **Location: `pkg/parsley/`** — Public package, allows potential external use
- **Keep Parsley CLI** — Build `pars` binary from `cmd/pars/` for standalone use
- **One go.mod** — Single module for the entire repo
- **Archive separate repo** — Mark github.com/sambeau/parsley as archived, pointing to basil
- **Copy files only (no git history merge)** — Simpler migration; original Parsley repo preserved on GitHub as historical reference
- **Unified issue tracking** — Parsley issues now tracked via Basil's FEAT-XXX/BUG-XXX system

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Current Structure (Parsley)
```
parsley/
├── cmd/pars/          # CLI entry point
├── pkg/
│   ├── ast/           # Abstract syntax tree
│   ├── evaluator/     # Interpreter
│   ├── lexer/         # Tokenizer
│   ├── parser/        # Parser
│   └── parsley/       # High-level API
├── std/               # Standard library
├── docs/
├── examples/
└── tests/
```

### Proposed Structure (Basil after merge)
```
basil/
├── cmd/
│   ├── basil/         # Basil server CLI (moved from root main.go)
│   └── pars/          # Parsley CLI (moved from parsley repo)
├── pkg/
│   └── parsley/       # All Parsley packages
│       ├── ast/
│       ├── evaluator/
│       ├── lexer/
│       ├── parser/
│       ├── parsley/   # High-level API
│       └── std/       # Standard library
├── server/            # Basil server logic
├── auth/              # Basil auth
├── config/            # Basil config
├── docs/
│   └── parsley/       # Parsley documentation (moved)
└── Makefile           # Updated build targets
```

### Import Changes
| Before | After |
|--------|-------|
| `github.com/sambeau/parsley/pkg/ast` | `github.com/sambeau/basil/pkg/parsley/ast` |
| `github.com/sambeau/parsley/pkg/evaluator` | `github.com/sambeau/basil/pkg/parsley/evaluator` |
| `github.com/sambeau/parsley/pkg/lexer` | `github.com/sambeau/basil/pkg/parsley/lexer` |
| `github.com/sambeau/parsley/pkg/parser` | `github.com/sambeau/basil/pkg/parsley/parser` |
| `github.com/sambeau/parsley/pkg/parsley` | `github.com/sambeau/basil/pkg/parsley/parsley` |

### Files to Update in Basil
- `go.mod` — Remove replace directive, remove parsley dependency
- `server/handler.go` — Update imports
- Any other files importing parsley packages

### Files to Update in Parsley (after move)
- All internal imports need `basil/pkg/parsley/` prefix
- `cmd/pars/main.go` — Update imports

### Dependencies
- Depends on: Nothing
- Blocks: Nothing (improves DX going forward)

### Edge Cases & Constraints
1. **External Parsley users** — If anyone uses Parsley standalone, they'd need to switch to the basil import path or we maintain a thin wrapper repo
2. **Parsley releases** — No more separate Parsley versions; it's part of Basil releases
3. **CI** — Single CI pipeline covers both

## Implementation Notes
*Added during/after implementation*

## Related
- Plan: `work/plans/FEAT-007-plan.md`
- This is a one-time migration task

