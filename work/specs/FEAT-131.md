---
id: FEAT-131
title: "Parsley Documentation Overhaul"
status: draft
priority: critical
created: 2026-02-27
author: "@human"
---

# FEAT-131: Parsley Documentation Overhaul

## Summary

The Parsley documentation (manual, reference, CHEATSHEET) contains factual errors, hallucinated methods, missing features, and stale content that will actively mislead humans and AI agents. Before v1.0, every document must be verified against the codebase — the single source of truth — and every code example must be machine-verified by running it through `pars`. This spec also introduces `pars reference --format markdown`, a code-generated API reference that eliminates documentation drift by construction.

These documents are foundational to Parsley's success. They will form the website, the manual, and all instructions for AI agents. **Accuracy is non-negotiable.**

## User Story

As a Parsley developer (human or AI agent), I want documentation that is 100% accurate and verified against the actual codebase so that I can trust what I read and write correct Parsley code without encountering methods, functions, or patterns that don't actually exist.

## Principles

1. **The codebase is the single source of truth.** Documentation must match the code. When in doubt, run `pars -e` or read the evaluator source. Never copy from one document to another — verify against code.
2. **Every code example must be machine-verified.** Run it through `pars -e` or as a `.pars` file. If it doesn't run, it doesn't ship.
3. **Small, focused tasks prevent hallucinations.** Each unit of work targets one document, one section, or one type. Verify before moving on.
4. **Generated > hand-written for API surfaces.** Method tables, builtin lists, and module exports should come from code metadata, not human memory.

## Acceptance Criteria

### Phase 1: CHEATSHEET Critical Fixes
- [ ] 1A: All hallucinated methods removed from CHEATSHEET method tables (§1.5 of audit)
- [ ] 1B: Every method listed in CHEATSHEET verified with `pars -e`
- [ ] 1C: Basil-specific sections removed from Parsley CHEATSHEET
- [ ] 1D: Deprecated imports replaced with current ones
- [ ] 1E: Every code example in CHEATSHEET verified with `pars -e`

### Phase 2: Reference Critical Fixes
- [ ] 2A: `print()`, `println()`, `printf()` removed from reference
- [ ] 2B: `match()` documented correctly as path/URL pattern matcher
- [ ] 2C: Reserved keywords list corrected (add `var`, `not`, `is`, `with`, `computed`, `const`)
- [ ] 2D: Number methods note corrected in Appendix A
- [ ] 2E: `markdown()` builtin signature corrected
- [ ] 2F: Appendix B method counts corrected (strings, numbers, dictionaries)
- [ ] 2G: Every code example in reference verified with `pars -e`

### Phase 3: Missing Documentation
- [ ] 3A: `with` expression documented in manual and reference
- [ ] 3B: `pars` CLI documented (subcommands, flags, security flags, output formats)
- [ ] 3C: REPL documented (commands, modes, history, tab completion)
- [ ] 3D: `var` documented properly in getting-started and variables page
- [ ] 3E: Number methods fully documented in manual

### Phase 4: Stale Content Cleanup
- [ ] 4A: Deprecated module docs removed or clearly marked (`@std/table`, `@std/schema`, `@std/api`, `@std/dev`, `@std/html`)
- [ ] 4B: `@std/session` manual page removed (module does not exist)
- [ ] 4C: Reference regex section corrected (§6.5 — `match()` is path matching, not regex)

### Phase 5: `pars reference` Command
- [ ] 5A: `FormatMarkdown` formatter implemented in `help/format.go`
- [ ] 5B: `pars reference` subcommand added to CLI
- [ ] 5C: Hand-written prose fragments extracted from `reference.md`
- [ ] 5D: Full reference composed from generated + hand-written sections
- [ ] 5E: CI verification step: generate reference, diff against committed copy

### Phase 6: AI Agent Documentation
- [ ] 6A: CHEATSHEET has "AI Quick Start" header pointing agents to `pars describe`
- [ ] 6B: CHEATSHEET method tables replaced with `pars describe` pointers or regenerated from code
- [ ] 6C: Copilot instructions updated to reference `pars describe` and generated reference

### Phase 7: Final Verification Pass
- [ ] 7A: Every code example in every document verified with `pars`
- [ ] 7B: Method/function counts in all documents match code registries
- [ ] 7C: No document references a function, method, or module that does not exist in code

## Design Decisions

- **Code is truth, not docs.** When a document disagrees with the code, the document is wrong. Do not "fix" code to match docs. Fix docs to match code.
- **Verify with `pars`, not by reading other docs.** The audit found errors that propagated from one document to another. Always verify against the running interpreter.
- **Strip Basil from Parsley CHEATSHEET.** ~280 lines of Basil-specific content (Parts, CSRF, asset bundles) removed. Security flags stay (they're `pars` CLI). A Basil cheatsheet is a separate future effort.
- **Generate API surfaces, hand-write prose.** `pars reference --format markdown` generates builtin tables, type method tables, operator lists, and module exports from code metadata. Grammar, control flow, tags, and tutorials remain hand-written.
- **Small tasks, verified incrementally.** Each unit below is scoped to prevent context overload and hallucination. Complete and verify each before moving to the next.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Source of Truth: Code Metadata

The evaluator already exposes structured metadata that covers the entire API surface:

| Source | File | What It Covers |
|--------|------|----------------|
| `BuiltinMetadata` | `evaluator/introspect.go` | ~45 builtins: name, arity, params, description, category |
| `OperatorMetadata` | `evaluator/introspect.go` | ~26 operators: symbol, name, description, category, example |
| `MethodRegistry` (per type) | `evaluator/method_registry.go` | Every method on every type: name, arity, description |
| Property registries | `evaluator/introspect.go` | Type properties: name, type, description |
| Module metadata | `evaluator/introspect.go` | Module descriptions and export lists |
| `TopicResult` / `TypeSchema` | `help/help.go` | Composed structures for all of the above |
| `FormatText` / `FormatJSON` | `help/format.go` | Existing formatters (markdown would be a third) |
| `pars describe all --json` | `cmd/pars/main.go` | Complete API schema dump — already works |

### Verification Method

For every code example in every document, the verification process is:

```bash
# For expression examples:
pars -e '<example code>'

# For multi-line examples saved as files:
pars /tmp/verify_example.pars

# For examples that produce HTML output:
pars -e --raw '<example code>'
```

If the example errors, it is wrong and must be fixed or removed. No exceptions.

### Documents to Modify

| Document | Path | Role |
|----------|------|------|
| CHEATSHEET | `docs/parsley/CHEATSHEET.md` | AI agent primary reference, patterns & pitfalls |
| Reference | `docs/parsley/reference.md` | Complete API reference (will become generated) |
| Getting Started | `docs/parsley/manual/getting-started.md` | Tutorial entry point |
| Variables | `docs/parsley/manual/fundamentals/variables.md` | `let`/`var` docs |
| Control Flow | `docs/parsley/manual/fundamentals/control-flow.md` | Needs `with` |
| Numbers | `docs/parsley/manual/builtins/numbers.md` | Missing methods |
| Manual Index | `docs/parsley/manual/index.md` | May reference stale modules |
| Stdlib pages | `docs/parsley/manual/stdlib/*.md` | Deprecated modules to remove/mark |
| Copilot Instructions | `.github/copilot-instructions.md` | AI agent setup instructions |

### New Files to Create

| File | Purpose |
|------|---------|
| `pkg/parsley/help/format_markdown.go` | Markdown formatter for `pars reference` |
| `docs/parsley/manual/features/cli.md` | `pars` CLI documentation |
| `docs/parsley/manual/features/repl.md` | REPL documentation |

### Dependencies

- Depends on: Audit report (`work/reports/PARSLEY-DOC-AUDIT.md`) for the complete error list
- Blocks: Parsley v1.0 release, AI SKILL files, website launch

---

## Implementation Units

Each unit is designed to be completable in a single focused session. **Do not combine units.** Complete and verify each before starting the next.

### ⚠️ Rules for Every Unit

1. **Read the relevant source code first.** Do not trust other documentation.
2. **Run `pars -e` to verify every claim.** If you write that a method exists, prove it. If you write an example, run it.
3. **Do not invent methods, parameters, or behaviours.** If you're unsure whether something exists, check the code or run `pars -e "value.method()"`. An "Unknown method" error means it doesn't exist.
4. **Commit after each unit passes verification.**

---

### Phase 1: CHEATSHEET Critical Fixes

#### Unit 1A: Fix CHEATSHEET String Method Table

**Scope:** The string method reference table in the CHEATSHEET.

**Process:**
1. Read `pkg/parsley/evaluator/methods_string.go` — specifically `StringMethodRegistry`
2. Run `pars -e '"hello".methodName()'` for every method listed in the CHEATSHEET
3. Remove any method that returns "Unknown method"
4. Add any method that exists in the registry but is missing from the table
5. Verify the arity and description of each method matches the registry

**Verification:** Every method in the updated table runs without error in `pars -e`.

#### Unit 1B: Fix CHEATSHEET Array Method Table

**Scope:** The array method reference table in the CHEATSHEET.

**Process:**
1. Read `pkg/parsley/evaluator/methods_array.go` — specifically `ArrayMethodRegistry`
2. Run `pars -e '[1,2,3].methodName()'` for every method listed
3. Remove hallucinated methods, add missing ones
4. Verify arity and description

**Verification:** Every method in the updated table runs without error in `pars -e`.

#### Unit 1C: Fix CHEATSHEET Dictionary Method Table

**Scope:** The dictionary method reference table in the CHEATSHEET.

**Process:**
1. Read `pkg/parsley/evaluator/methods_dict.go` — specifically `DictionaryMethodRegistry`
2. Run `pars -e '{a:1}.methodName()'` for every method listed
3. Remove hallucinated methods, add missing ones

**Verification:** Every method in the updated table runs without error in `pars -e`.

#### Unit 1D: Fix CHEATSHEET Number Method Tables

**Scope:** The integer and float method reference tables in the CHEATSHEET.

**Process:**
1. Read `pkg/parsley/evaluator/methods_number.go` — `IntegerMethodRegistry` and `FloatMethodRegistry`
2. Verify each listed method with `pars -e '42.methodName()'` and `pars -e '3.14.methodName()'`
3. Remove hallucinated methods, add missing ones

**Verification:** Every method runs without error in `pars -e`.

#### Unit 1E: Fix CHEATSHEET Remaining Type Method Tables

**Scope:** Method tables for datetime, duration, url, path, file, table, money, regex, boolean, and any other types in the CHEATSHEET.

**Process:**
1. For each type, read the corresponding `methods_*.go` and its `MethodRegistry`
2. Verify each listed method with `pars -e`
3. Remove hallucinated methods, add missing ones

**Verification:** Every method runs without error in `pars -e`.

#### Unit 1F: Strip Basil Content from CHEATSHEET

**Scope:** Remove Basil-specific sections from the Parsley CHEATSHEET.

**Sections to remove:**
- 🌿 Basil Server (~L1420–1561)
- 🧩 Parts (~L1561–1660)
- 🎨 Asset Bundles (~L1660–1686)

**Sections to keep:**
- 🔒 Security Flags (documents `pars` CLI behaviour, not Basil server)

**Verification:** The CHEATSHEET contains no references to `@basil/http`, `@basil/auth`, Parts, CSRF tokens, asset bundles, or Basil server configuration (except in the context of security flags).

#### Unit 1G: Fix CHEATSHEET Deprecated Imports and Module References

**Scope:** Module Quick Reference section and any import examples.

**Process:**
1. Read `pkg/parsley/evaluator/stdlib_table.go` (or equivalent module loader) to get the current module names
2. Replace all deprecated imports (`@std/api` → `@basil/api`, `@std/dev` → `@basil/log`, `@std/html` → `@basil/html`)
3. Remove references to modules that don't exist (`@std/session`)

**Verification:** Every `import` example in the CHEATSHEET uses a current, non-deprecated module path.

#### Unit 1H: Verify All CHEATSHEET Code Examples

**Scope:** Every code block in the CHEATSHEET that contains runnable Parsley code.

**Process:**
1. Extract every code block from the CHEATSHEET
2. Run each through `pars -e` (for expressions) or as a `.pars` file (for multi-line)
3. Fix or remove any example that errors
4. Note: some examples are intentionally showing error cases — these are fine if clearly marked

**Verification:** A log of every example tested and its result. Zero unexpected failures.

---

### Phase 2: Reference Critical Fixes

#### Unit 2A: Remove Nonexistent Builtins from Reference

**Scope:** Remove `print()`, `println()`, `printf()` from the reference.

**Process:**
1. Search `reference.md` for all mentions of `print`, `println`, `printf`
2. Remove the function entries and any examples that use them
3. Verify: `pars -e 'print("hello")'` should error, confirming these don't exist
4. Ensure the reference's builtin list matches `BuiltinMetadata` in `evaluator/introspect.go`

**Verification:** `grep -i 'print\b' reference.md` returns zero hits for function calls (may appear in prose explaining that print doesn't exist).

#### Unit 2B: Fix `match()` Documentation

**Scope:** The `match()` builtin entry in the reference.

**Process:**
1. Read `match()` implementation in `evaluator/evaluator.go` (search `getBuiltins` for "match")
2. Run `pars -e 'match("/users/42", "/users/:id")'` to see actual behaviour
3. Rewrite the reference entry to describe path/URL pattern matching with captures
4. Remove any regex-related description

**Verification:** The example in the rewritten entry runs correctly in `pars -e`.

#### Unit 2C: Fix Reserved Keywords List

**Scope:** The reserved keywords section in the reference.

**Process:**
1. Read `pkg/parsley/lexer/` to find the actual keyword list (look for token definitions or keyword maps)
2. Compare against what the reference lists
3. Add missing keywords: `var`, `not`, `is`, `with`, `computed`, `const` (verify each against lexer)
4. Remove any listed keywords that aren't actually reserved

**Verification:** The keyword list in the reference matches the lexer's keyword definitions exactly.

#### Unit 2D: Fix Number Methods in Reference

**Scope:** The number methods section and Appendix A of the reference.

**Process:**
1. Read `IntegerMethodRegistry` and `FloatMethodRegistry` in `methods_number.go`
2. Count actual methods for each type
3. Correct the "5 methods" claim — actual count from registries
4. Remove the claim that numbers lack `abs()` — verify with `pars -e '(-5).abs()'`
5. List all actual number methods with correct descriptions

**Verification:** Every number method listed runs in `pars -e`. Method count matches registry size.

#### Unit 2E: Fix `markdown()` Builtin Signature

**Scope:** The `markdown()` entry in the reference.

**Process:**
1. Read the `markdown` builtin implementation in `evaluator/evaluator.go`
2. Read `BuiltinMetadata["markdown"]` in `introspect.go`
3. Determine: does it take a string of markdown content, or a file path?
4. Run `pars -e 'markdown("# Hello")'` and `pars -e 'MD("test.md")'` to see the difference
5. Correct the reference entry

**Verification:** The documented signature matches what `pars -e` accepts.

#### Unit 2F: Fix Reference Method Counts (Appendix B)

**Scope:** Method count summaries in Appendix B of the reference.

**Process:**
1. For each type, count methods from its `MethodRegistry` in source
2. Run `pars -e 'describe("string")'` (or `pars describe string`) to cross-check
3. Update all counts in Appendix B

**Verification:** Every count matches the registry. Run `pars describe <type>` for each to confirm.

#### Unit 2G: Verify All Reference Code Examples

**Scope:** Every code block in the reference.

**Process:** Same as Unit 1H but for `reference.md`.

**Verification:** A log of every example tested. Zero unexpected failures.

---

### Phase 3: Missing Documentation

#### Unit 3A: Document `with` Expression

**Scope:** Add `with` documentation to the manual and reference.

**Process:**
1. Read the `with` implementation in `evaluator/evaluator.go` (search for `evalWith` or `with` in the evaluator)
2. Read the parser support in `pkg/parsley/parser/`
3. Write examples and verify each with `pars -e`
4. Add to `docs/parsley/manual/fundamentals/control-flow.md`
5. Add to `reference.md` in the control flow section

**Verification:** Every `with` example runs correctly in `pars -e`.

#### Unit 3B: Document `pars` CLI

**Scope:** Create `docs/parsley/manual/features/cli.md`.

**Process:**
1. Read `cmd/pars/main.go` for all subcommands and flags
2. Run `pars --help` to see the actual help text
3. Run each subcommand (`pars fmt --help`, `pars describe --help`, etc.)
4. Document: `pars` (REPL), `pars file.pars`, `pars -e`, `pars fmt`, `pars describe`, all flags
5. Include examples, verify each with `pars`

**Verification:** Every documented flag and subcommand works as described when run.

#### Unit 3C: Document the REPL

**Scope:** Create `docs/parsley/manual/features/repl.md`.

**Process:**
1. Read the REPL implementation (search for REPL in `cmd/pars/main.go` or a `repl/` package)
2. Document commands: `:help`, `:describe`, `:env`, `:clear`, `:raw`
3. Document: history, continuation detection, multiline input, prompt format
4. Test each command by running `pars` interactively

**Verification:** Every documented REPL command works as described.

#### Unit 3D: Fix `var` Documentation

**Scope:** Ensure `var` is properly documented in getting-started and variables page.

**Process:**
1. Read how `var` works in the evaluator (search for variable assignment / mutation)
2. Check `docs/parsley/manual/getting-started.md` — does it mention `var`?
3. Check `docs/parsley/manual/fundamentals/variables.md` — is `var` documented?
4. Add/fix documentation with verified examples

**Verification:** Examples of `let` (immutable) and `var` (mutable) both run correctly in `pars -e`.

#### Unit 3E: Complete Number Methods Documentation

**Scope:** `docs/parsley/manual/builtins/numbers.md`.

**Process:**
1. Read `IntegerMethodRegistry` and `FloatMethodRegistry`
2. List every method with description, arity, and a verified example
3. Run every example through `pars -e`

**Verification:** Every method listed runs correctly. No method in the registry is missing from the docs.

---

### Phase 4: Stale Content Cleanup

#### Unit 4A: Clean Up Deprecated Module Docs

**Scope:** Manual pages for deprecated/moved modules.

**Process:**
1. `docs/parsley/manual/stdlib/schema.md` — Mark as deprecated, point to `@schema` DSL
2. `docs/parsley/manual/stdlib/table.md` — Mark as deprecated if module is deprecated
3. `docs/parsley/manual/stdlib/dev.md` — Update: `@std/dev` → `@basil/log`
4. `docs/parsley/manual/stdlib/api.md` — Update: `@std/api` → `@basil/api`
5. `docs/parsley/manual/stdlib/html.md` — Update: `@std/html` → `@basil/html`
6. Verify each module's current status in the evaluator source

**Verification:** Every import path in stdlib docs matches the current, non-deprecated module name.

#### Unit 4B: Remove `@std/session` Manual Page

**Scope:** `docs/parsley/manual/stdlib/session.md`.

**Process:**
1. Confirm `@std/session` does not exist: `pars -e 'import "@std/session"'` should error
2. Remove or replace the page with a note that sessions are handled by `@basil/auth`
3. Remove from manual index if listed

**Verification:** No document references `@std/session` as an available module.

#### Unit 4C: Fix Reference Regex/Match Section

**Scope:** Reference §6.5 or wherever `match()` is categorised under regex.

**Process:**
1. Ensure `match()` is not in a "regex" section — it's a path matcher
2. Move or re-categorise it appropriately
3. Ensure `regex()` builtin is correctly documented separately

**Verification:** `match()` and `regex()` are in separate, correctly-labelled sections.

---

### Phase 5: `pars reference` Command

#### Unit 5A: Implement `FormatMarkdown` Formatter

**Scope:** `pkg/parsley/help/format_markdown.go` (new file).

**Process:**
1. Follow the pattern of `FormatText` in `help/format.go`
2. Implement `FormatMarkdown(result *TopicResult) string`
3. Generate markdown tables for: builtins (grouped by category), type methods, type properties, operators (grouped by category), module exports
4. Include a YAML frontmatter header with generation timestamp
5. Write tests

**Verification:** `FormatMarkdown(describeAll())` produces valid, well-structured markdown. Every method and builtin in the output exists in the code.

#### Unit 5B: Add `pars reference` Subcommand

**Scope:** `cmd/pars/main.go`.

**Process:**
1. Add `reference` as a recognised subcommand
2. Support `--format markdown|json|text` (default: markdown)
3. Support `--api-only` flag (generated sections only, no hand-written fragments)
4. Call `describeAll()` and format with the appropriate formatter
5. Write to stdout

**Verification:** `pars reference --format markdown` produces valid markdown output. `pars reference --format json` produces valid JSON.

#### Unit 5C: Extract Hand-Written Prose Fragments

**Scope:** Split `reference.md` into reusable fragments.

**Process:**
1. Identify sections of `reference.md` that are hand-written prose (literals, grammar, control flow, tags, examples)
2. Extract each into a separate file under `docs/parsley/reference-fragments/` (or similar)
3. Use a naming convention that defines ordering (e.g., `01-introduction.md`, `02-literals.md`)
4. Mark where generated sections should be inserted (e.g., `<!-- GENERATED: builtins -->`)

**Verification:** The fragments, when manually concatenated, cover all non-API prose from the original reference.

#### Unit 5D: Compose Full Reference

**Scope:** Wire up fragment stitching in the `pars reference` command.

**Process:**
1. When `--format markdown` is used without `--api-only`, read the hand-written fragments
2. Insert generated sections at the marked insertion points
3. Produce a single complete `reference.md`

**Verification:** `pars reference --format markdown` produces a complete reference that includes both hand-written prose and generated API tables. Every code example in it runs in `pars -e`.

#### Unit 5E: CI Verification

**Scope:** Add a CI step or Makefile target.

**Process:**
1. Add `make verify-docs` (or similar) that runs `pars reference --format markdown` and diffs against the committed `reference.md`
2. Fail if they differ — forces regeneration after any API change

**Verification:** Deliberately change a method description in a registry, run verify-docs, confirm it fails.

---

### Phase 6: AI Agent Documentation

#### Unit 6A: Add AI Quick Start to CHEATSHEET

**Scope:** Add a header section to the CHEATSHEET for AI agents.

**Content should include:**
- Use `pars describe <topic>` for API lookups (always accurate, from code)
- Use `pars describe all --json` for complete machine-readable API schema
- Use the CHEATSHEET for patterns, pitfalls, and idioms (not API details)
- Use `pars -e` to verify any code before outputting it

**Verification:** An AI agent reading only this header would know where to find accurate API information.

#### Unit 6B: Replace CHEATSHEET API Tables with Generated Content or Pointers

**Scope:** The method reference tables in the CHEATSHEET.

**Decision (choose one during implementation):**
- **Option A:** Replace method tables with output from `pars describe <type>` — regenerate on each docs build
- **Option B:** Replace method tables with a note: "Run `pars describe string` for the complete, up-to-date method list"
- **Option C:** Keep the tables but generate them from code metadata (a smaller version of `FormatMarkdown`)

**Verification:** No hand-maintained method table remains in the CHEATSHEET that could drift from code.

#### Unit 6C: Update Copilot Instructions

**Scope:** `.github/copilot-instructions.md`.

**Process:**
1. Add guidance to use `pars describe <topic>` for API information
2. Add guidance to use `pars -e` to verify code examples before writing them in docs
3. Reference the generated reference as the authoritative API source (once Phase 5 ships)

**Verification:** The copilot instructions accurately describe the available tools.

---

### Phase 7: Final Verification Pass

#### Unit 7A: Machine-Verify All Code Examples

**Scope:** Every code block in every Parsley document.

**Process:**
1. Write a script (or use `pars` directly) to extract and run every code example
2. Categorise: expression examples (run with `pars -e`), file examples (run as `.pars`), error examples (expected to fail), pseudo-code (not runnable — should be rare and clearly marked)
3. Run all runnable examples
4. Fix any failures

**Deliverable:** A verification log showing every example and its pass/fail status. Target: 100% pass rate for runnable examples.

#### Unit 7B: Verify Method and Function Counts

**Scope:** Every document that states a count of methods, builtins, operators, or modules.

**Process:**
1. Run `pars describe all --json` to get authoritative counts
2. Search all documents for count claims
3. Fix any mismatches

**Verification:** Every count in every document matches `pars describe all --json`.

#### Unit 7C: Verify No Phantom References

**Scope:** Every document.

**Process:**
1. Extract every function name, method name, and module path mentioned in docs
2. Verify each exists in code (via `pars -e`, `pars describe`, or source grep)
3. Remove or fix any reference to something that doesn't exist

**Verification:** Zero references to nonexistent functions, methods, or modules in any document.

---

## Estimated Effort

| Phase | Units | Est. Time | Notes |
|-------|-------|-----------|-------|
| Phase 1: CHEATSHEET fixes | 8 units | 6-8h | Mostly verification + deletion |
| Phase 2: Reference fixes | 7 units | 4-6h | Targeted corrections |
| Phase 3: Missing docs | 5 units | 6-8h | New content, requires code reading |
| Phase 4: Stale cleanup | 3 units | 2-3h | Mostly deletion |
| Phase 5: `pars reference` | 5 units | 8-12h | New Go code + fragment extraction |
| Phase 6: AI agent docs | 3 units | 2-3h | Small additions |
| Phase 7: Final verification | 3 units | 4-6h | Comprehensive sweep |
| **Total** | **34 units** | **~32-46h** | |

### Suggested Priority Order

**Must-have for v1.0:** Phases 1, 2, 3, 4, 7
**Should-have for v1.0:** Phase 6
**Target v1.0 if schedule permits, otherwise v1.1:** Phase 5

---

## Related

- Audit: `work/reports/PARSLEY-DOC-AUDIT.md` (complete error catalogue and Appendix B feasibility analysis)
- CHEATSHEET: `docs/parsley/CHEATSHEET.md`
- Reference: `docs/parsley/reference.md`
- Help package: `pkg/parsley/help/`
- Method registries: `pkg/parsley/evaluator/method_registry.go`, `methods_*.go`
- Builtin metadata: `pkg/parsley/evaluator/introspect.go`
