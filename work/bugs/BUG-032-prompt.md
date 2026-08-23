# Fix BUG-032: enforce arity on user-defined functions

You are a senior Go engineer working on a tree-walking interpreter.

## Vocabulary

arity · call site · callee · parameter binding · positional arguments · rest parameter (`...rest`) · variadic · destructuring parameter · closure environment · `extendFunctionEnv` · `ParamCount()` · callback dispatch · higher-order method · structured error (`perrors.New`) · error class `ClassArity` · error code `ARITY-000x` · error position (`enrichErrorWithPos`, token) · breaking change · regression fixture · golden test · definition of done

## Constraints

- ALWAYS raise the arity error **at the call site**, BECAUSE an error that points inside the callee's body (today's `Identifier not found: a`) misdiagnoses a caller bug as a callee bug.
- ALWAYS make internal callback sites **adapt to `fn.ParamCount()`** rather than pass a fixed count, BECAUSE `for (x in …) fn` and dict iteration already do this ([`eval_control_flow.go:116`](../../pkg/parsley/evaluator/eval_control_flow.go), `:241`) and it is the only way strict arity and `fn()` components can coexist.
- NEVER special-case "too many" as a warning or silently tolerate it for user functions, BECAUSE builtins already error on it and the language must be consistent with itself.
- NEVER add default parameter values or keyword arguments in this unit, BECAUSE they are separate features (BUG-032 "Out of scope") and would widen a breaking change.
- NEVER change how destructuring parameters (`fn({a, b})`, `fn([x, y])`) bind internally, BECAUSE each counts as one positional parameter and its inner leniency is by design.
- ALWAYS follow `CLAUDE.md`'s Definition of Done (build, test, docs, CHANGELOG, merge to `main`, push, clean worktree) BECAUSE "tests pass" is not done here.

## Anti-patterns

- **The Fixed-Count Callback**: an internal site passes N args to a user callback no matter what it declared. Detect: `extendFunctionEnv(fn, []Object{a, b})` with a literal slice. Resolve: branch on `fn.ParamCount()`; raise a `LOOP-0004`-style error for unsupported counts.
- **The Body-Blame Error**: failure reported where the unbound parameter is *used*, not where the call was made. Detect: `Identifier not found` naming a parameter. Resolve: check counts in `extendFunctionEnv`'s callers before evaluating the body, and attach the call expression's token.
- **The Nameless Error**: `Wrong number of arguments to <anonymous>` for a function that was bound with `let add = fn…`. Detect: `Function` struct has no name field ([`evaluator.go:323`](../../pkg/parsley/evaluator/evaluator.go)). Resolve: decide how to name it — callee expression text from the call AST, or a `Name` set at `let`/`export` binding — and be consistent.
- **The Silent Fixture Fix**: a test that relied on dropped arguments is "fixed" by deleting it. Resolve: correct the call in the fixture and keep the assertion.

## Task

Implement the fix described in [`work/bugs/BUG-032.md`](BUG-032.md): user-defined functions called with too many or too few arguments must raise an `ARITY` error at the call site. Read the bug report first — it contains the reproduction, the root cause, the table of every internal call site, the proposed rules, and the language comparison. Do not redo that investigation.

Scope: `pkg/parsley` only, plus its docs and CHANGELOG. No server changes.

Expected effort: 40–80 tool calls. Roughly a third reading (bug report, the call sites listed in it, the error catalogue, existing arity tests), a third implementing, a third testing and verifying with `pars`.

Use: Read, Grep, Edit, Bash (`go build ./...`, `go test ./...`, `./pars -e '…'`), the `done` skill at the end.
Do not use: the browser/preview tools, `verify` (no HTTP surface is affected), Artifact.

## Procedure

1. Create a worktree branch from current `main` (the repo works in `.claude/worktrees/`; see `CLAUDE.md`). Allocate nothing — the bug already has an ID.
2. Read `work/bugs/BUG-032.md` end to end, then open every file in its call-site table.
3. Read the ARITY entries in [`pkg/parsley/errors/errors.go:369`](../../pkg/parsley/errors/errors.go) and the helpers `newArityError*` in [`eval_errors.go:189`](../../pkg/parsley/evaluator/eval_errors.go). Decide whether an existing code fits a user function or a new one (`ARITY-0007`?) is clearer. IF you add a code THEN add it to `docs/parsley/error-codes.md` and the `errors_test.go` table.
4. Decide where the check lives. `extendFunctionEnv` ([`eval_expressions.go:577`](../../pkg/parsley/evaluator/eval_expressions.go)) returns an `*Environment`, not an `Object`, so it cannot return an error today. Either change its signature or check in its callers (`applyFunction`, `ApplyFunctionWithEnv`, `applyMethodWithThis`). Prefer one place.
5. Decide how the function is named in the message (see *The Nameless Error*). Write the decision as a comment beside the check.
6. Implement strict checking for direct calls: `add(1, 2, 3)` and `one()` both error; the error carries the call site's token.
7. Convert every fixed-count callback site in the bug's table to adapt on `ParamCount()`:
   - tag/component dispatch (`eval_tags.go` ×3): pass props only when `ParamCount() >= 1`
   - `.reduce` (`methods_array.go:217`): 1- or 2-param reducers
   - 1-arg sites (`methods.go`, `stdlib_table.go`, `markdown_helpers.go`): require 1; error otherwise
   IF a site already adapts (`.replace`, loops) THEN leave it alone.
8. Run `go test ./...`. FOR each failure: IF the fixture called with wrong arity THEN fix the call; IF the failure reveals a callback site you missed THEN go back to step 7.
9. Add tests: too many, too few, zero-param component via `<C/>`, 1- and 2-param reducers, and the position of the reported error. Put them beside the existing arity tests (grep `ARITY-0001` in `pkg/parsley/evaluator/*_test.go`).
10. Verify at the CLI with `./pars -e` for each reproduction in the bug report; paste the output into the bug report under a new `## Verification` heading and set `status: fixed`.
11. Update docs: remove the "silently ignored" paragraph and the warning box in [`docs/parsley/manual/fundamentals/functions.md:183-191`](../../docs/parsley/manual/fundamentals/functions.md); change the ARITY row in [`docs/parsley/manual/fundamentals/errors.md:194`](../../docs/parsley/manual/fundamentals/errors.md) from "builtin/method" to "any function"; check `docs/parsley/reference.md` and `CHEATSHEET.md` for the same claim.
12. Add a `## [Unreleased]` entry to `CHANGELOG.md` under a **Breaking** heading — this changes the behaviour of existing programs.
13. Run `/done`.

## Output format

Your final message must contain, in this order:

1. **Decision: error placement** — one sentence (where the check lives and why).
2. **Decision: function naming** — one sentence.
3. **Call sites changed** — a table: file:line, before (fixed N), after (adapts / requires N).
4. **Fixtures corrected** — list of test files whose calls were wrong, with a one-line note each. "None" is an acceptable answer only if `go test ./...` passed on the first run after step 7.
5. **Verification** — the literal `pars -e` output for `add(1,2,3,4)`, `one()`, and `<C/>` with `fn()`.
6. **Done checklist** — the CLAUDE.md boxes, each ticked or explained.

## Examples

### Bad

```
Runtime error in <eval>: line 1, column 19
  Identifier not found: `a`
    let one = fn(a) { a }; one()
                      ^
```
Wrong: blames the body; the caret is on the parameter, not the call.

### Good

```
Runtime error in <eval>: line 1, column 24
  `one` expects 1 argument, got 0
    let one = fn(a) { a }; one()
                           ^
```
Right: names the function, gives both counts, points at the call.

### Bad

```go
// eval_tags.go — still passes props to a fn() component
result := ApplyFunctionWithEnv(val, []Object{props}, env)
```
Wrong: with strict arity this breaks every prop-less component on every site.

### Good

```go
args := []Object{}
if fn, ok := val.(*Function); ok && fn.ParamCount() >= 1 {
    args = []Object{props}
}
result := ApplyFunctionWithEnv(val, args, env)
```
Right: the site adapts, like the for-loop already does.

## Retrieval anchors

Questions this prompt answers:

- Where is the arity gap in the Parsley evaluator, and why has nothing broken yet?
- Which internal call sites pass a fixed number of arguments to user callbacks?
- How should a `fn()` component keep working once arity is strict?
- Which docs and error catalogues must change when a new `ARITY` code is added?
- What does "done" mean for this unit of work?
