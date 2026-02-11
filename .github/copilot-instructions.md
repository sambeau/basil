# Copilot Instructions for Basil

## Overview
Basil is a Go web server for the Parsley programming language.

## Before Any Task
1. Read `AGENTS.md` at the repository root — it contains build commands, project structure, and workflow rules
2. Check `work/BACKLOG.md` for related deferred items
3. Use the appropriate prompt file for your task type

## Writing Parsley Code
Before writing any Parsley code (handlers, tests, examples):
- Read `.github/instructions/parsley.instructions.md` for syntax rules
- Key points: tags don't need quotes, singleton tags MUST be self-closing (`<br/>` not `<br>`), use `{var}` for interpolation (not `${var}`)

## Debugging and Testing Parsley Code
Use `pars -e` to quickly test and debug Parsley expressions:
- Outputs PLN (Parsley Literal Notation) format by default, showing structure
- Examples:
  - `pars -e "[1, 2, 3]"` → outputs `[1, 2, 3]`
  - `pars -e '"hello"'` → outputs `"hello"`
  - `pars -e "{a: 1, b: 2}"` → outputs `{a: 1, b: 2}`
- Use `--raw` or `-r` for file-like output (e.g., for HTML rendering)
- Matches REPL behavior for consistency

## Workflow Entry Points
- **New Feature**: Use `/new-feature` prompt
- **Bug Fix**: Use `/fix-bug` prompt  
- **Release**: Use `/release` prompt

## Key Conventions
- Features: `FEAT-XXX` in `work/specs/`
- Bugs: `BUG-XXX` in `work/bugs/`
- Plans: `work/plans/`
- IDs: Managed via `work/ID_COUNTER.md`

## Git Rules
- AI commits to feature/bug branches
- Human merges to main
- Human creates release tags
- Use Conventional Commits format

## Testing
- All code changes must include tests
- Run tests frequently during implementation
- Update test files in `pkg/parsley/tests/` for Parsley language features
- Bug fixes must include regression tests

## Documentation
- Update `docs/guide/faq.md` when answering "how do I..." questions
- Add deferred items to `work/BACKLOG.md`

## Parsley Documentation
When documenting Parsley language features:
- When writing manual pages: see `.github/templates/DOC_MAN_BUILTIN.md` and `.github/templates/DDOC_MAN_STD.md`
- `docs/parsley/reference.md` - Comprehensive reference. All features should be documented here with accurate grammar snippets
- `docs/parsley/CHEATSHEET.md` - AI-focused cheatsheet highlighting differences from other languages, ordered by likelihood of being a pitfall
- `docs/parsley/README.md` - Quick guide with examples (may be outdated)
