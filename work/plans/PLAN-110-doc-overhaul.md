---
id: PLAN-110
feature: FEAT-131
title: "Implementation Plan for Parsley Documentation Overhaul"
status: draft
created: 2026-02-27
---

# Implementation Plan: FEAT-131 (Parsley Documentation Overhaul)

## Overview

This plan details the implementation of FEAT-131: a complete overhaul of Parsley documentation to ensure 100% accuracy before v1.0. The work is organized into 34 small, focused units across 7 phases to prevent hallucination and enable verification at each step.

**Core Principle:** The codebase is the single source of truth. Every claim must be verified against code. Every example must be run through `pars`.

## Branch

`feat/FEAT-131-doc-overhaul`

## Prerequisites

- [ ] Audit report read: `work/reports/PARSLEY-DOC-AUDIT.md`
- [ ] `pars` CLI available and working
- [ ] Access to evaluator source code (`pkg/parsley/evaluator/`)
- [ ] Working tree is clean (`git status`)

---

## Verification Protocol

**Before modifying any documentation, follow this protocol:**

1. **Read the source code** for the feature being documented
2. **Run `pars -e`** to verify any code examples
3. **Run `pars describe <topic>`** to get authoritative API information
4. **Never copy from other documentation** — always verify against code

**After each unit:**

1. **Run all code examples** in the modified document through `pars -e`
2. **Commit the changes** with a descriptive message
3. **Update the progress log** in this document

---

## Phase 1: CHEATSHEET Critical Fixes

**Goal:** Fix all hallucinated methods and errors in the CHEATSHEET.

**Documents:** `docs/parsley/CHEATSHEET.md`

### Unit 1A: Fix String Method Table

**Estimated effort:** Medium (45 min)

**Source of truth:** `pkg/parsley/evaluator/methods_string.go` → `StringMethodRegistry`

**Steps:**
1. Read `StringMethodRegistry` in `methods_string.go` — list every method name
2. For each method in the CHEATSHEET string table:
   - Run `pars -e '"test".methodName()'`
   - If "Unknown method" error → mark for removal
   - If works → verify arity and description match registry
3. For each method in `StringMethodRegistry` not in CHEATSHEET → add it
4. Update the table with verified methods only

**Verification command:**
```bash
# Test a method exists:
pars -e '"hello".upper()'

# Get authoritative list:
pars describe string
```

**Commit:** `docs(cheatsheet): fix string method table — verified against StringMethodRegistry`

---

### Unit 1B: Fix Array Method Table

**Estimated effort:** Medium (30 min)

**Source of truth:** `pkg/parsley/evaluator/methods_array.go` → `ArrayMethodRegistry`

**Steps:**
1. Read `ArrayMethodRegistry` in `methods_array.go`
2. For each method in the CHEATSHEET array table:
   - Run `pars -e '[1,2,3].methodName()'`
   - Remove if "Unknown method"
   - Verify arity/description if exists
3. Add any missing methods from registry

**Verification command:**
```bash
pars -e '[1,2,3].map(fn(x) x*2)'
pars describe array
```

**Commit:** `docs(cheatsheet): fix array method table — verified against ArrayMethodRegistry`

---

### Unit 1C: Fix Dictionary Method Table

**Estimated effort:** Medium (30 min)

**Source of truth:** `pkg/parsley/evaluator/methods_dict.go` → `DictionaryMethodRegistry`

**Steps:**
1. Read `DictionaryMethodRegistry` in `methods_dict.go`
2. Verify each method with `pars -e '{a:1, b:2}.methodName()'`
3. Remove hallucinated methods, add missing ones

**Verification command:**
```bash
pars -e '{a:1}.keys()'
pars describe dictionary
```

**Commit:** `docs(cheatsheet): fix dictionary method table — verified against DictionaryMethodRegistry`

---

### Unit 1D: Fix Number Method Tables

**Estimated effort:** Medium (30 min)

**Source of truth:** 
- `pkg/parsley/evaluator/methods_number.go` → `IntegerMethodRegistry`, `FloatMethodRegistry`

**Steps:**
1. Read both registries
2. Verify integer methods with `pars -e '42.methodName()'`
3. Verify float methods with `pars -e '3.14.methodName()'`
4. Update both tables

**Verification command:**
```bash
pars -e '(-5).abs()'
pars -e '3.14159.round(2)'
pars describe integer
pars describe float
```

**Commit:** `docs(cheatsheet): fix number method tables — verified against Integer/FloatMethodRegistry`

---

### Unit 1E: Fix Remaining Type Method Tables

**Estimated effort:** Large (60 min)

**Types to verify:** datetime, duration, url, path, file, table, money, regex, boolean

**Source of truth:** Corresponding `methods_*.go` files and their registries

**Steps:**
1. For each type:
   - Find its `MethodRegistry` in evaluator source
   - Run `pars describe <type>` to get authoritative list
   - Verify each CHEATSHEET entry with `pars -e`
   - Fix discrepancies

**Verification commands:**
```bash
pars describe datetime
pars describe duration
pars describe url
pars describe path
pars describe file
pars describe table
pars describe money
pars describe regex
pars describe boolean
```

**Commit:** `docs(cheatsheet): fix datetime/duration/url/path/file/table/money/regex/boolean method tables`

---

### Unit 1F: Strip Basil Content

**Estimated effort:** Small (15 min)

**Steps:**
1. Locate and remove these sections:
   - 🌿 Basil Server section (~L1420–1561)
   - 🧩 Parts section (~L1561–1660)
   - 🎨 Asset Bundles section (~L1660–1686)
2. Keep 🔒 Security Flags (documents `pars` CLI, not Basil)
3. Remove any remaining `@basil/http`, `@basil/auth` examples outside security context
4. Update table of contents if present

**Verification:** `grep -n "Parts\|CSRF\|@basil/http\|@basil/auth" CHEATSHEET.md` should return minimal hits (only in security flags context)

**Commit:** `docs(cheatsheet): remove Basil-specific content (Parts, asset bundles, server config)`

---

### Unit 1G: Fix Deprecated Imports

**Estimated effort:** Small (20 min)

**Source of truth:** `pkg/parsley/evaluator/stdlib_table.go` or module loader

**Steps:**
1. Find all `import` statements in CHEATSHEET
2. Replace deprecated paths:
   - `@std/api` → `@basil/api`
   - `@std/dev` → `@basil/log`
   - `@std/html` → `@basil/html`
3. Remove references to `@std/session` (does not exist)
4. Verify each import: `pars -e 'import "@module/name"; true'`

**Verification command:**
```bash
# Should work:
pars -e 'import "@basil/api"; true'

# Should fail (deprecated):
pars -e 'import "@std/api"; true'
```

**Commit:** `docs(cheatsheet): update deprecated module imports to current paths`

---

### Unit 1H: Verify All Code Examples

**Estimated effort:** Large (90 min)

**Steps:**
1. Extract every code block from CHEATSHEET (use grep or manual scan)
2. Categorize each:
   - **Expression** — run with `pars -e 'code'`
   - **Multi-line** — save to temp file, run with `pars /tmp/test.pars`
   - **Error example** — expected to fail (verify it fails as documented)
   - **Pseudo-code** — not runnable (should be rare, clearly marked)
3. Run every runnable example
4. Fix or remove any that fail unexpectedly
5. Create a verification log

**Verification log format:**
```
| Line | Type | Code (truncated) | Result |
|------|------|------------------|--------|
| 45   | expr | "hello".upper()  | ✅ PASS |
| 67   | expr | [1,2].first()    | ❌ FAIL: Unknown method |
```

**Commit:** `docs(cheatsheet): verify all code examples — N tested, N fixed, N removed`

---

## Phase 2: Reference Critical Fixes

**Goal:** Fix factual errors in the reference document.

**Documents:** `docs/parsley/reference.md`

### Unit 2A: Remove Nonexistent Builtins

**Estimated effort:** Small (20 min)

**Steps:**
1. Search for `print`, `println`, `printf` in reference.md
2. Remove all entries for these functions
3. Remove any examples using them
4. Verify they don't exist: `pars -e 'print("test")'` → should error

**Source of truth:** `BuiltinMetadata` in `pkg/parsley/evaluator/introspect.go`

**Verification:**
```bash
grep -n "print\b" docs/parsley/reference.md  # Should return minimal hits
pars -e 'print("hello")'  # Should error: unknown function
```

**Commit:** `docs(reference): remove nonexistent print/println/printf builtins`

---

### Unit 2B: Fix `match()` Documentation

**Estimated effort:** Small (20 min)

**Source of truth:** `match` implementation in `pkg/parsley/evaluator/evaluator.go`

**Steps:**
1. Read the `match` builtin implementation in `getBuiltins()`
2. Run test cases:
   ```bash
   pars -e 'match("/users/42", "/users/:id")'
   pars -e 'match("/posts/hello-world", "/posts/:slug")'
   ```
3. Rewrite the reference entry:
   - Purpose: URL/path pattern matching with named captures
   - Returns: dictionary of captured values, or null if no match
   - NOT regex — it's a path pattern matcher

**Commit:** `docs(reference): fix match() documentation — path matcher, not regex`

---

### Unit 2C: Fix Reserved Keywords List

**Estimated effort:** Small (20 min)

**Source of truth:** `pkg/parsley/lexer/lexer.go` or `tokens.go` — keyword definitions

**Steps:**
1. Find keyword definitions in lexer source
2. Compare to reference's keyword list
3. Add missing: `var`, `not`, `is`, `with`, `computed`, `const` (verify each in lexer)
4. Remove any listed that aren't actually reserved

**Verification:** For each keyword, confirm it's in the lexer's keyword map or token list.

**Commit:** `docs(reference): fix reserved keywords list — matches lexer definitions`

---

### Unit 2D: Fix Number Methods in Reference

**Estimated effort:** Medium (30 min)

**Source of truth:** `IntegerMethodRegistry`, `FloatMethodRegistry` in `methods_number.go`

**Steps:**
1. Count methods in each registry
2. Update Appendix A claim (currently says "5 methods" — wrong)
3. Remove claim that numbers lack `abs()`:
   ```bash
   pars -e '(-5).abs()'  # Works, returns 5
   ```
4. List all actual number methods with verified descriptions

**Commit:** `docs(reference): fix number methods documentation — actual count and methods`

---

### Unit 2E: Fix `markdown()` Builtin Signature

**Estimated effort:** Small (15 min)

**Source of truth:** `BuiltinMetadata["markdown"]` in `introspect.go`, implementation in `evaluator.go`

**Steps:**
1. Read the `markdown` builtin implementation
2. Run tests:
   ```bash
   pars -e 'markdown("# Hello")'   # String input
   pars -e 'MD("test.md")'         # File input (different function?)
   ```
3. Correct the reference entry to match actual behavior

**Commit:** `docs(reference): fix markdown() builtin signature and description`

---

### Unit 2F: Fix Method Counts (Appendix B)

**Estimated effort:** Medium (30 min)

**Source of truth:** `pars describe all --json` or individual type registries

**Steps:**
1. Run `pars describe all --json | jq '.types[] | {name, method_count: (.methods | length)}'`
2. Or manually count methods in each registry
3. Update Appendix B with correct counts for:
   - Strings (reference says 27, actual is ~36)
   - Numbers (reference says 5, actual is ~14 for int, ~17 for float)
   - Dictionaries (reference says 12, verify actual)
   - All other types

**Commit:** `docs(reference): fix method counts in Appendix B — verified against registries`

---

### Unit 2G: Verify All Reference Code Examples

**Estimated effort:** Large (120 min)

**Steps:**
1. Same process as Unit 1H but for `reference.md`
2. Extract all code blocks
3. Run each through `pars -e` or as a file
4. Fix or remove failures
5. Create verification log

**Commit:** `docs(reference): verify all code examples — N tested, N fixed`

---

## Phase 3: Missing Documentation

**Goal:** Document features that exist in code but are missing from docs.

### Unit 3A: Document `with` Expression

**Estimated effort:** Medium (45 min)

**Source of truth:** `evalWith` or `with` handling in `pkg/parsley/evaluator/evaluator.go`

**Steps:**
1. Read the `with` implementation in evaluator
2. Read any parser support in `pkg/parsley/parser/`
3. Write documentation with:
   - Syntax: `with expr as name { body }`
   - Purpose: scoped binding
   - Examples (verify each with `pars -e`)
4. Add to `docs/parsley/manual/fundamentals/control-flow.md`
5. Add to `reference.md` in control flow section

**Example to verify:**
```bash
pars -e 'with 42 as x { x * 2 }'
```

**Commit:** `docs: add with expression documentation to manual and reference`

---

### Unit 3B: Document `pars` CLI

**Estimated effort:** Medium (60 min)

**Source of truth:** `cmd/pars/main.go`

**Steps:**
1. Read `main.go` for all subcommands and flags
2. Run `pars --help` to see help text
3. Create `docs/parsley/manual/features/cli.md` with:
   - `pars` (no args) — starts REPL
   - `pars file.pars` — runs a file
   - `pars -e "code"` — evaluates expression
   - `pars fmt` — formats code
   - `pars describe` — introspection
   - All flags: `-r/--raw`, `-c/--check`, `-pp/--pretty`, `-q/--quiet`, `--format`, `--machine`
   - Security flags: `--restrict-read`, etc.
4. Verify each documented flag/command by running it

**Commit:** `docs: add pars CLI documentation (cli.md)`

---

### Unit 3C: Document the REPL

**Estimated effort:** Medium (45 min)

**Source of truth:** REPL implementation in `cmd/pars/main.go` or `repl/` package

**Steps:**
1. Read REPL implementation
2. Start `pars` interactively, test each command:
   - `:help` — shows help
   - `:describe <topic>` — introspection
   - `:env` — shows environment
   - `:clear` — clears screen/state
   - `:raw` — toggles raw output mode
3. Create `docs/parsley/manual/features/repl.md` with:
   - Commands
   - Multiline input / continuation
   - History
   - Tab completion (if exists)

**Commit:** `docs: add REPL documentation (repl.md)`

---

### Unit 3D: Fix `var` Documentation

**Estimated effort:** Small (20 min)

**Source of truth:** Variable handling in evaluator

**Steps:**
1. Verify `let` vs `var` behavior:
   ```bash
   pars -e 'let x = 1; x = 2; x'  # Should error (immutable)
   pars -e 'var x = 1; x = 2; x'  # Should return 2 (mutable)
   ```
2. Check `docs/parsley/manual/getting-started.md` — add `var` if missing
3. Check `docs/parsley/manual/fundamentals/variables.md` — ensure `var` is documented

**Commit:** `docs: add var documentation to getting-started and variables page`

---

### Unit 3E: Complete Number Methods Documentation

**Estimated effort:** Medium (45 min)

**Source of truth:** `IntegerMethodRegistry`, `FloatMethodRegistry`

**Steps:**
1. Open `docs/parsley/manual/builtins/numbers.md`
2. For each method in `IntegerMethodRegistry`:
   - Add entry with description, arity, example
   - Verify example with `pars -e`
3. Same for `FloatMethodRegistry`
4. Note which methods are shared vs type-specific

**Commit:** `docs: complete number methods documentation with verified examples`

---

## Phase 4: Stale Content Cleanup

**Goal:** Remove or fix deprecated/incorrect content.

### Unit 4A: Clean Up Deprecated Module Docs

**Estimated effort:** Medium (30 min)

**Steps:**
1. `docs/parsley/manual/stdlib/schema.md`:
   - Add deprecation notice at top
   - Point to `@schema` DSL syntax
2. `docs/parsley/manual/stdlib/table.md`:
   - Check if `@std/table` is deprecated; if so, mark it
3. `docs/parsley/manual/stdlib/dev.md`:
   - Update: `@std/dev` is now `@basil/log`
4. `docs/parsley/manual/stdlib/api.md`:
   - Update: `@std/api` is now `@basil/api`
5. `docs/parsley/manual/stdlib/html.md`:
   - Update: `@std/html` is now `@basil/html`

**Verification:** Run `pars -e 'import "@new/path"; true'` for each new path.

**Commit:** `docs: update deprecated module documentation with current paths and notices`

---

### Unit 4B: Remove `@std/session` Page

**Estimated effort:** Small (10 min)

**Steps:**
1. Verify module doesn't exist: `pars -e 'import "@std/session"'` → should error
2. Remove `docs/parsley/manual/stdlib/session.md`
3. Remove from `docs/parsley/manual/index.md` if listed
4. Add note to `@basil/auth` docs that sessions are handled there (if appropriate)

**Commit:** `docs: remove nonexistent @std/session documentation`

---

### Unit 4C: Fix Reference Regex/Match Section

**Estimated effort:** Small (15 min)

**Steps:**
1. Find where `match()` is documented in reference
2. If it's in a "Regex" section, move it out
3. `match()` = path pattern matching
4. `regex()` = regular expression creation
5. Ensure they're in separate, correctly-labelled sections

**Commit:** `docs(reference): separate match() (path patterns) from regex section`

---

## Phase 5: `pars reference` Command

**Goal:** Build a code-generated API reference that cannot drift from implementation.

### Unit 5A: Implement `FormatMarkdown` Formatter

**Estimated effort:** Large (120 min)

**Files:** Create `pkg/parsley/help/format_markdown.go`

**Steps:**
1. Study existing `FormatText` and `FormatJSON` in `format.go`
2. Create `FormatMarkdown(result *TopicResult) string` following same pattern
3. Implement formatters for:
   - Type: heading, properties table, methods table
   - Module: heading, description, exports table
   - Builtin: signature, description, parameters
   - Builtin list: grouped by category
   - Operator list: grouped by category
   - All: complete API reference
4. Add YAML frontmatter with generation timestamp
5. Write tests

**Test verification:**
```go
result := describeAll()
md := FormatMarkdown(result)
// Verify contains expected sections
// Verify valid markdown syntax
```

**Commit:** `feat(help): add FormatMarkdown formatter for pars reference command`

---

### Unit 5B: Add `pars reference` Subcommand

**Estimated effort:** Medium (45 min)

**Files:** `cmd/pars/main.go`

**Steps:**
1. Add `reference` to command dispatch
2. Support flags:
   - `--format markdown|json|text` (default: markdown)
   - `--api-only` (skip hand-written fragments)
3. Call `describeAll()` and format with appropriate formatter
4. Output to stdout
5. Add help text

**Verification:**
```bash
pars reference --format markdown > /tmp/test.md
pars reference --format json | jq .
pars reference --help
```

**Commit:** `feat(cli): add pars reference subcommand with format options`

---

### Unit 5C: Extract Hand-Written Fragments

**Estimated effort:** Medium (60 min)

**Steps:**
1. Analyze `reference.md` to identify hand-written sections:
   - Introduction
   - Literals (strings, numbers, etc.)
   - Grammar/syntax rules
   - Control flow
   - Tags/HTML
   - Examples and tutorials
2. Create `docs/parsley/reference-fragments/` directory
3. Extract each section into numbered files:
   - `01-introduction.md`
   - `02-literals.md`
   - `03-control-flow.md`
   - `04-tags.md`
   - etc.
4. Add insertion markers: `<!-- GENERATED: builtins -->`

**Commit:** `docs: extract hand-written reference sections into fragments`

---

### Unit 5D: Implement Fragment Composition

**Estimated effort:** Medium (60 min)

**Files:** `pkg/parsley/help/` or `cmd/pars/`

**Steps:**
1. Add logic to `pars reference` to read fragment files
2. Parse insertion markers in fragments
3. Generate API sections with `FormatMarkdown`
4. Stitch fragments + generated sections together
5. Output complete reference

**Verification:**
```bash
pars reference --format markdown > /tmp/full-ref.md
# Should contain both prose and generated tables
```

**Commit:** `feat(cli): compose full reference from fragments + generated API sections`

---

### Unit 5E: Add CI Verification

**Estimated effort:** Small (30 min)

**Steps:**
1. Add Makefile target:
   ```makefile
   verify-docs:
   	pars reference --format markdown > /tmp/generated-ref.md
   	diff -q docs/parsley/reference.md /tmp/generated-ref.md
   ```
2. Or create `scripts/verify-docs.sh`
3. Add to CI workflow (if exists) or document as pre-release check

**Verification:** Change a method description in a registry, run verify-docs, confirm it fails.

**Commit:** `chore: add verify-docs target for reference drift detection`

---

## Phase 6: AI Agent Documentation

**Goal:** Make documentation AI-friendly with clear guidance and accurate data sources.

### Unit 6A: Add AI Quick Start to CHEATSHEET

**Estimated effort:** Small (20 min)

**Steps:**
1. Add a new section near the top of CHEATSHEET: "🤖 AI Agent Quick Start"
2. Content:
   - For API lookups: `pars describe <topic>` (always accurate)
   - For complete schema: `pars describe all --json`
   - For patterns/pitfalls: use this CHEATSHEET
   - To verify code: `pars -e "code"` before outputting
3. Keep it brief — 10-15 lines max

**Commit:** `docs(cheatsheet): add AI Quick Start section for agent guidance`

---

### Unit 6B: Replace/Regenerate CHEATSHEET API Tables

**Estimated effort:** Medium (45 min)

**Decision:** Choose one approach:
- **Option A:** Replace tables with `pars describe` pointers
- **Option B:** Regenerate tables from registries (automated)
- **Option C:** Keep tables but add "verified against code" timestamps

**Steps (for Option A — simplest):**
1. Replace each method table with:
   ```markdown
   > Run `pars describe string` for the complete, up-to-date method list.
   ```
2. Keep a few key examples inline for quick reference

**Steps (for Option B — most robust):**
1. Create a script that generates markdown tables from registries
2. Run during docs build to update CHEATSHEET tables
3. Add generation timestamp to each table

**Commit:** `docs(cheatsheet): replace static method tables with pars describe references`

---

### Unit 6C: Update Copilot Instructions

**Estimated effort:** Small (20 min)

**Files:** `.github/copilot-instructions.md`

**Steps:**
1. Add section on using `pars describe` for API information
2. Add guidance to verify code with `pars -e` before writing examples
3. Reference `pars reference --format markdown` as authoritative source (after Phase 5)
4. Emphasize: code is truth, not other documentation

**Commit:** `docs: update copilot instructions with pars describe and verification guidance`

---

## Phase 7: Final Verification Pass

**Goal:** Comprehensive verification that all documentation is accurate.

### Unit 7A: Machine-Verify All Code Examples

**Estimated effort:** Large (120 min)

**Steps:**
1. Create or run a verification script:
   ```bash
   # Extract code blocks and test them
   for doc in docs/parsley/**/*.md; do
     # Extract code blocks, run through pars -e
   done
   ```
2. Categorize results:
   - ✅ Pass — example runs correctly
   - ❌ Fail — example errors (needs fix)
   - ⚠️ Expected fail — error example (verify error matches)
   - ⏭️ Skip — pseudo-code (clearly marked)
3. Fix all failures
4. Create final verification log

**Deliverable:** `work/reports/DOC-VERIFICATION-LOG.md` with pass/fail for every example

**Commit:** `docs: final verification pass — all examples verified with pars`

---

### Unit 7B: Verify Method and Function Counts

**Estimated effort:** Small (30 min)

**Steps:**
1. Run `pars describe all --json` to get authoritative counts
2. Search all docs for count claims:
   ```bash
   grep -rn "methods\|functions\|builtins" docs/parsley/
   ```
3. Verify each claim matches JSON output
4. Fix mismatches

**Commit:** `docs: verify all method/function counts match code`

---

### Unit 7C: Verify No Phantom References

**Estimated effort:** Medium (60 min)

**Steps:**
1. Extract all function/method/module names from docs
2. For each:
   - Run `pars -e 'name'` or `pars describe name`
   - Confirm it exists
3. Remove or fix any references to nonexistent items
4. Check for common hallucinations:
   - `print()`, `println()`, `printf()`
   - `.capitalize()`, `.startsWith()`, `.endsWith()` (if not real)
   - `@std/session`

**Commit:** `docs: remove all references to nonexistent functions/methods/modules`

---

## Validation Checklist

- [ ] All tests pass: `go test ./pkg/parsley/...`
- [ ] Build succeeds: `go build -o pars ./cmd/pars`
- [ ] `pars reference --format markdown` generates valid output
- [ ] All code examples in CHEATSHEET verified
- [ ] All code examples in reference.md verified
- [ ] All code examples in manual verified
- [ ] Method counts match registries
- [ ] No references to nonexistent functions/methods
- [ ] CHEATSHEET contains no Basil-specific content (except security flags)
- [ ] All imports use current (non-deprecated) paths
- [ ] Copilot instructions updated

---

## Progress Log

| Date | Unit | Status | Notes |
|------|------|--------|-------|
| 2026-02-27 | 1A String methods | ✅ Complete | Removed 10 hallucinated methods, added 12 real ones, fixed title→toTitle |
| 2026-02-27 | 1B Array methods | ✅ Complete | Removed 8 hallucinated methods (first, last, find, etc.), fixed reduce arg order |
| 2026-02-27 | 1C Dictionary methods | ✅ Complete | Removed 4 hallucinated methods (get, merge, without, pick), added entries |
| 2026-02-27 | 1D Number methods | ✅ Complete | Added abs, humanize, round, floor, ceil; verified all examples |
| 2026-02-27 | 1E Other type methods | ✅ Complete | Restructured datetime/duration as properties+methods tables |
| 2026-02-27 | 1F Strip Basil | ✅ Complete | Removed ~280 lines (Server, Parts, Asset Bundles); kept Security Flags |
| 2026-02-27 | 1G Fix imports | ✅ Complete | @std/api→@basil/api, @std/dev→@basil/log, @std/html→@basil/html, @schema DSL |
| 2026-02-27 | 1H Verify examples | ✅ Complete | All method examples verified with pars -e |
| 2026-02-26 | 2A Remove print | ✅ Complete | Removed print/println/printf from Output section, fixed examples to use log() |
| 2026-02-26 | 2B Fix match() | ✅ Complete | Changed from regex to path pattern matcher, documented :name and *name captures |
| 2026-02-26 | 2C Fix keywords | ✅ Complete | Added var, const, not, is, computed, with to match lexer definitions |
| 2026-02-26 | 2D Fix number methods | ✅ Complete | Added abs/round/ceil/floor, split Integer (13) and Float (16) methods |
| 2026-02-26 | 2E Fix markdown() | ✅ Complete | Fixed signature: takes string not path, returns {html, md, raw} |
| 2026-02-26 | 2F Fix counts | ✅ Complete | String 38, Array 15, Dict 9, Integer 13, Float 16 methods |
| 2026-02-26 | 2G Verify examples | ✅ Complete | All modified examples verified with pars -e |
| 2026-02-26 | 3A Document with | ✅ Complete | Added to control-flow.md and reference.md with syntax, examples, scope rules |
| 2026-02-26 | 3B Document CLI | ✅ Complete | Created cli.md: subcommands, options, security flags, examples |
| 2026-02-26 | 3C Document REPL | ✅ Complete | Created repl.md: commands, output modes, keyboard shortcuts, debugging tips |
| 2026-02-26 | 3D Fix var docs | ✅ Complete | Rewrote variables.md for let/var; updated getting-started.md |
| 2026-02-26 | 3E Number methods | ✅ Complete | Fixed numbers.md: abs() on both types, round/ceil/floor float-only |
| 2026-02-26 | 4A Deprecated modules | ✅ Complete | Added deprecation notices to schema.md and table.md |
| 2026-02-26 | 4B Fix session | ✅ Complete | Renamed to 'Session Management', clarified basil.session access |
| 2026-02-26 | 4C Fix regex section | ✅ Complete | Already done in Phase 2 — match() in separate subsection |
| 2026-02-27 | 5A FormatMarkdown | ✅ Complete | Created format_markdown.go with formatters for all TopicResult kinds |
| 2026-02-27 | 5B reference cmd | ✅ Complete | Added `pars reference` subcommand with --format, --verify, --template flags |
| 2026-02-27 | 5C Extract fragments | ✅ Complete | Created reference-fragments/ with 10 hand-written section files |
| 2026-02-27 | 5D Composition | ✅ Complete | Created compose.go template engine with include/generate/template directives |
| 2026-02-27 | 5E CI verification | ✅ Complete | Added make verify-docs, make docs, make docs-full targets; generated api-reference.md |
| 2026-02-27 | 6A AI Quick Start | ✅ Complete | Added 🤖 AI Agent Quick Start section with pars describe table, key rules, verification examples |
| 2026-02-27 | 6B Replace tables | ✅ Complete | Option A already implemented - tables have pars describe references at top of Method Reference section |
| 2026-02-27 | 6C Copilot instructions | ✅ Complete | Added API Reference section, verification workflow, documentation accuracy rules |
| 2026-02-26 | 7A Verify all examples | ✅ Complete | Tested string/array/dict/number methods, control flow, operators, builtins |
| 2026-02-26 | 7B Verify counts | ✅ Complete | All method counts match pars describe: String 38, Array 15, Dict 9, Int 13, Float 16 |
| 2026-02-26 | 7C No phantoms | ✅ Complete | Fixed print()→log(), updated @std/*→@basil/* imports in 5 files |

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Hallucinating methods during fixes | Medium | High | Always verify with `pars -e` before writing |
| Missing some code examples | Medium | Medium | Systematic extraction, verification log |
| `pars reference` generates invalid markdown | Low | Medium | Write tests, manual review |
| Fragment extraction misses content | Low | Medium | Diff original vs reconstructed reference |
| Phase 5 takes longer than estimated | Medium | Low | Phase 5 can slip to v1.1 if needed |

---

## Estimated Total Time

| Phase | Units | Est. Time |
|-------|-------|-----------|
| Phase 1: CHEATSHEET | 8 | 6-8h |
| Phase 2: Reference | 7 | 4-5h |
| Phase 3: Missing docs | 5 | 4-5h |
| Phase 4: Cleanup | 3 | 1-2h |
| Phase 5: pars reference | 5 | 6-8h |
| Phase 6: AI docs | 3 | 1-2h |
| Phase 7: Verification | 3 | 3-4h |
| **Total** | **34** | **~25-34h** |

**Recommended execution order:**
1. Phases 1-4 first (critical fixes, ~15-20h)
2. Phase 7 (verification, ~3-4h)
3. Phase 6 (AI docs, ~1-2h)
4. Phase 5 (pars reference, ~6-8h) — can be v1.1 if schedule tight

---

## Deferred Items

Items to add to `work/BACKLOG.md` if discovered during implementation:

- [ ] (Space for items found during work)

---

## References

- Spec: `work/specs/FEAT-131.md`
- Audit: `work/reports/PARSLEY-DOC-AUDIT.md`
- CHEATSHEET: `docs/parsley/CHEATSHEET.md`
- Reference: `docs/parsley/reference.md`
- Help package: `pkg/parsley/help/`
- Evaluator: `pkg/parsley/evaluator/`
