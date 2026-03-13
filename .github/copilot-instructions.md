# Copilot Instructions for Basil

## Overview
Basil is a Go web server for the Parsley programming language.

## Before Any Task
1. **Check `git status` for uncommitted work** — commit or stash before starting new work
2. Read `AGENTS.md` at the repository root — it contains build commands, project structure, and workflow rules
3. Check `work/BACKLOG.md` for related deferred items
4. Use the appropriate prompt file for your task type

## ⚠️ Critical: No Test Failures May Be Ignored

**All tests must pass before any commit. There are no acceptable pre-existing failures.**

- **Do NOT label a failing test "pre-existing" and move on.** If a test fails, fix it.
- **Do NOT assume a failure is unrelated to your change.** Even if it appears unrelated, fix it before committing.
- **Do NOT commit with a known failing test under any circumstances**, even in code you did not touch.
- If you cannot fix a failing test yourself, **stop and tell the user** before committing. Do not work around it or suppress it.

The only acceptable state before a commit is: `ok` for every package in `go test ./...`.

---

## ⚠️ Critical: Preserve Work Through Commits

**Work has been lost due to incomplete commits.** Follow these rules:

### Before Starting New Work
- Run `git status` — if there are uncommitted changes from previous work:
  - If tests pass → commit with an appropriate message
  - If tests fail → stash and inform the user
  - Never start new work on top of uncommitted changes from a different feature

### During Implementation
- Commit at logical checkpoints (new function, passing tests, before risky changes)
- When tests pass after a change, commit

### After Completing a Feature/Fix
1. Run `make test` or `go test ./...`
2. If tests pass → **commit all related changes immediately**
3. A feature isn't done until it's committed

## Writing Parsley Code
Before writing any Parsley code (handlers, tests, examples):
- Read `.github/instructions/parsley.instructions.md` for syntax rules
- Key points: tags don't need quotes, singleton tags MUST be self-closing (`<br/>` not `<br>`), use `{var}` for interpolation (not `${var}`)

### ⚠️ Critical: Parsley Uses Expression-Based Output
**`print()` does NOT exist in Parsley!** Values ARE the output:
```parsley
// ❌ WRONG — These will error!
print("hello")
println("hello")

// ✅ CORRECT — The value IS the output
"hello"
<div>"content"</div>

// ✅ For debugging, use log()
log("debug:", someVar)
```

## Debugging and Testing Parsley Code
Use `pars -e` to quickly test and debug Parsley expressions:
- Outputs PLN (Parsley Literal Notation) format by default, showing structure
- Examples:
  - `pars -e "[1, 2, 3]"` → outputs `[1, 2, 3]`
  - `pars -e '"hello"'` → outputs `"hello"`
  - `pars -e "{a: 1, b: 2}"` → outputs `{a: 1, b: 2}`
- Use `--raw` or `-r` for file-like output (e.g., for HTML rendering)
- Matches REPL behavior for consistency

## Parsley API Reference (Source of Truth)
**Always use `pars describe` to get accurate API information.** Documentation may be outdated; the code is truth.

### Looking Up Methods and Builtins
```bash
# Get all methods for a type
pars describe string
pars describe array
pars describe dict
pars describe integer
pars describe float

# Get info about a specific builtin
pars describe len
pars describe range
pars describe match

# Get complete API as JSON (for programmatic use)
pars describe all --json
```

### Verifying Code Before Writing
Before including Parsley code in documentation or examples:
```bash
# Test that an expression works
pars -e '"hello".toUpper()'

# Test HTML output
pars -r -e '<div class="test">"content"</div>'

# Check if a method exists
pars describe string | grep -i "trim"
```

### Key Principle
If documentation conflicts with `pars describe` output, **trust the command**. Update the documentation to match.

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
- AI merges to main after human approval
- AI can push to remote when delegated by human
- Human creates release tags
- Use Conventional Commits format:
  - `feat(scope): description` — New features
  - `fix(scope): description` — Bug fixes  
  - `test(scope): description` — Test changes
  - `docs(scope): description` — Documentation
  - `chore(scope): description` — Maintenance
  - Add `!` for breaking changes: `feat(parsley)!: description`

## ⚠️ Critical: Features Require Tests

**A feature is not complete until it has tests and the tests pass.**

### Definition of Done
1. Implementation code is written
2. Tests are written that exercise the new functionality
3. All tests pass (`go test ./pkg/parsley/...`)
4. Changes are committed

### Test Requirements
- **New language features**: Add tests to `pkg/parsley/tests/` (integration tests)
- **New methods**: Test both success cases and error cases
- **Bug fixes**: Add regression test that would have caught the bug
- **Edge cases**: Test boundary conditions, empty inputs, error paths

### Running Tests
```bash
# Run all Parsley tests
go test ./pkg/parsley/...

# Run with coverage for evaluator
go test -coverprofile=cov.out -coverpkg=./pkg/parsley/evaluator ./pkg/parsley/...

# Run specific test file
go test ./pkg/parsley/tests -run TestName
```

### What Counts as Tested
- Feature works for typical use cases
- Error conditions return appropriate errors (not panics)
- Edge cases are handled (empty arrays, null values, etc.)

### Do NOT
- Merge or mark complete any feature without tests
- Skip tests "to save time" — untested code causes more work later
- Write tests that only test the happy path

## Documentation
- Update `docs/guide/faq.md` when answering "how do I..." questions
- Add deferred items to `work/BACKLOG.md`

## Parsley Documentation
When documenting Parsley language features:
- When writing manual pages: see `.github/templates/DOC_MAN_BUILTIN.md` and `.github/templates/DDOC_MAN_STD.md`
- `docs/parsley/reference.md` - Comprehensive reference. All features should be documented here with accurate grammar snippets
- `docs/parsley/CHEATSHEET.md` - AI-focused cheatsheet highlighting differences from other languages, ordered by likelihood of being a pitfall
- `docs/parsley/README.md` - Quick guide with examples (may be outdated)

### Documentation Accuracy Rules
1. **Verify examples work** — run `pars -e "code"` before committing doc changes
2. **Check method existence** — use `pars describe <type>` to confirm methods exist
3. **Don't invent builtins** — `print()`, `println()`, `printf()` do NOT exist
4. **Code is truth** — if docs conflict with `pars describe`, fix the docs
