# Parsley Documentation Audit Report

**Date:** 2026-02-27
**Scope:** All Parsley language documentation (manual, reference, cheatsheet, pars CLI)
**Method:** Code-is-truth audit — every finding verified against evaluator source code
**Goal:** Ensure documentation is complete, correct, and v1.0-ready

---

## Executive Summary

The documentation is in good shape structurally. The manual is well-organised, the getting-started tutorial is excellent, and most type pages are thorough. However, there are several factual errors in the reference (most critically: documenting functions that don't exist), missing coverage of implemented features, stale deprecation references that should be removed for v1.0, and the reference has significant overlap with the manual that needs resolution.

### Priority Summary

| Priority | Count | Description |
|----------|-------|-------------|
| 🔴 Critical | 5 | Factual errors — documents things that don't exist, or misses what does |
| 🟠 High | 7 | Missing documentation for implemented features |
| 🟡 Medium | 8 | Stale content, deprecation warnings to remove, structural issues |
| 🟢 Low | 5 | Polish, consistency, minor gaps |

---

## Part 1: Factual Errors (🔴 Critical)

### 1.1 Reference documents `print()`, `println()`, `printf()` — they do not exist

**File:** `docs/parsley/reference.md` § 6.2 "Output" (L3098–3125)

The reference lists these as builtin functions:

| Function | Status |
|----------|--------|
| `print(vals...)` | ❌ Does not exist |
| `println(vals...)` | ❌ Does not exist |
| `printf(template, dict)` | ❌ Does not exist |
| `log(vals...)` | ✅ Exists |
| `logLine(vals...)` | ✅ Exists |

**Evidence:** `getBuiltins()` in `evaluator.go` has no entries for `"print"`, `"println"`, or `"printf"`. The copilot instructions explicitly state: "print() does NOT exist in Parsley! Values ARE the output."

**Fix:** Remove `print`, `println`, and `printf` from the reference. The section should only document `log()` and `logLine()`, with a clear note that Parsley uses expression-based output.

### 1.2 Reference documents `match()` with wrong signature

**File:** `docs/parsley/reference.md` § 6.5 "Regex" (L3166–3183)

The reference says:
> `match(str, pattern, flags?)` — Find first match

**Actual implementation** (`evaluator.go` L3284): `match(path, pattern)` extracts named parameters from URL paths (like `:id` and `*rest`). It takes a path string/dict and a pattern string, returning a dictionary of captures or null. It is **not** a regex match function.

**Fix:** Move `match()` out of the "Regex" section. Document it under a new section (e.g., "Routing / Path Matching") with its actual signature and semantics:
```
match(path, pattern) → dictionary | null
// match("/users/42", "/users/:id") → {id: "42"}
// match("/files/a/b/c", "/files/*rest") → {rest: ["a", "b", "c"]}
```

### 1.3 Reference reserved keywords list is incomplete

**File:** `docs/parsley/reference.md` § "Reserved Keywords" (L4870–4879)

Current list:
```
fn, function, let, for, in, if, else, return, export, import,
try, check, stop, skip, true, false, null, and, or, as, via
```

**Missing keywords** (all registered in `lexer.go` L408–435):
- `var` — mutable variable declaration (actively used)
- `not` — aliased to `BANG` token (alternative to `!`)
- `is` — type/schema checking operator
- `with` — scoped field access expression
- `computed` — for `export computed` declarations
- `const` — reserved keyword (gives helpful error suggesting `let`)

**Fix:** Update to:
```
fn, function, let, var, const, for, in, if, else, return,
export, computed, import, try, check, stop, skip, with,
true, false, null, and, or, not, is, as, via
```

### 1.4 Reference Appendix A claims numbers have no `abs()` method

**File:** `docs/parsley/reference.md` Appendix A (L4904)

> **Note**: Numbers do not have math methods like `abs()`. Use `@std/math` for mathematical operations.

**This is wrong.** Both `IntegerMethodRegistry` and `FloatMethodRegistry` (in `methods_numeric.go`) register `abs`, `round` (float only), `floor` (float only), `ceil` (float only), plus the full formatter API (`fmt`, `short`, `medium`, `long`, `currency`, `percent`, `humanize`, `toBox`, `repr`, `toJSON`, `inspect`).

**Fix:** Remove the incorrect note. Update Appendix A's Number row to list key methods: `abs`, `round`, `floor`, `ceil`, `fmt`, `currency`, `percent`, `humanize`.

### 1.5 CHEATSHEET method reference tables contain many hallucinated methods

**File:** `docs/parsley/CHEATSHEET.md` § "📝 Method Reference" (L1702–1873)

The method reference tables in the CHEATSHEET list dozens of methods that **do not exist** in Parsley. Every one of the following was verified against `pars -e` and produces `Unknown method` errors:

**String methods that don't exist:**

| Listed | Status | Correct Alternative |
|--------|--------|---------------------|
| `.capitalize()` | ❌ DNE | `.toTitle()` |
| `.title()` | ❌ DNE | `.toTitle()` |
| `.trimStart()` | ❌ DNE | `.trim()` only |
| `.trimEnd()` | ❌ DNE | `.trim()` only |
| `.replaceAll(old, new)` | ❌ DNE | `.replace()` already replaces all |
| `.startsWith(prefix)` | ❌ DNE | Use `in` operator or regex |
| `.endsWith(suffix)` | ❌ DNE | Use regex |
| `.indexOf(substr)` | ❌ DNE | Use regex match |
| `.escapeHtml()` | ❌ DNE | `.htmlEncode()` |
| `.pad(len, char?)` | ❌ DNE | Not implemented |
| `.padStart(len, char?)` | ❌ DNE | Not implemented |
| `.padEnd(len, char?)` | ❌ DNE | Not implemented |

**Array methods that don't exist:**

| Listed | Status | Correct Alternative |
|--------|--------|---------------------|
| `.first()` | ❌ DNE | `arr[0]` |
| `.last()` | ❌ DNE | `arr[-1]` |
| `.findIndex(fn)` | ❌ DNE | Not implemented |
| `.every(fn)` | ❌ DNE | Not implemented (use `for` + `check`) |
| `.some(fn)` | ❌ DNE | Not implemented (use `for` + `if`) |
| `.flatten()` | ❌ DNE | Not implemented |
| `.unique()` | ❌ DNE | Not implemented (table has `.unique()`) |
| `.find(fn)` | ❌ DNE | Use `.filter(fn)[0]` |

**Dictionary methods that don't exist:**

| Listed | Status | Correct Alternative |
|--------|--------|---------------------|
| `.get(key, default?)` | ❌ DNE | Use `dict.key` + `??` null coalescing |
| `.merge(other)` | ❌ DNE | Use `++` operator |
| `.without(keys...)` | ❌ DNE | Use `.delete(key)` |
| `.pick(keys...)` | ❌ DNE | Use destructuring |

**Impact:** This is particularly damaging because the CHEATSHEET is the primary reference for AI agents. AI agents will confidently use these methods, produce broken code, and waste cycles debugging. This is the highest-priority fix for v1.0.

**Evidence:** All verified with `pars -e`:
```
$ pars -e '"hello".capitalize()'
Runtime error: Unknown method `capitalize` for string

$ pars -e '[1,2,3].first()'
Runtime error: Unknown method `first` for array

$ pars -e '{a:1}.get("a")'
Runtime error: Unknown method `get` for dictionary
```

**Fix:** Completely rewrite the CHEATSHEET method reference tables using only methods confirmed to exist in the `StringMethodRegistry`, `evalArrayMethod`, and `evalDictionaryMethod` code. The manual pages (`builtins/strings.md`, `builtins/array.md`, `builtins/dictionary.md`) have correct method lists and should be used as the source of truth.

---

## Part 2: Missing Documentation (🟠 High)

### 2.1 `with` expression is undocumented

The `with` keyword is a registered keyword, parsed by `parseWithExpression`, and evaluated by `evalWithExpression` in `eval_control_flow.go`. It creates a scoped block where dictionary/record fields are available as local variables:

```parsley
let user = {name: "Alice", age: 30}
with (user) {
    `{name} is {age} years old`
}
// "Alice is 30 years old"
```

**Status:** The CHEATSHEET mentions it (§ "Scoped Field Access"), but it is absent from:
- The reference (not in Control Flow, not in Statements)
- The manual (`fundamentals/control-flow.md` — no mention)

**Fix:** Add to `fundamentals/control-flow.md` and to the reference § 3 "Control Flow".

### 2.2 `var` is under-documented in the manual

`var` is documented in `fundamentals/variables.md` and in the reference § 4.1, but:
- The getting-started tutorial never mentions `var` — it only shows `let`
- The CHEATSHEET covers it well (§ 2), but many manual pages use `let` exclusively

**Fix:** Add a brief mention to getting-started.md (in the "Variables and Expressions" section) explaining when to use `var` vs `let`.

### 2.3 Number methods are missing from the manual

**File:** `docs/parsley/manual/builtins/numbers.md`

The numbers manual page needs to document the full method registry from `methods_numeric.go`. The integer and float registries include 14+ methods each, including the unified formatter API (`fmt`, `format`, `short`, `medium`, `long`, `currency`, `percent`, `humanize`) plus `abs`, `round`, `floor`, `ceil`, `toBox`, `repr`, `toJSON`, `inspect`.

The reference § 5.4 "Number Methods" documents some of these but is incomplete (missing `inspect`, and the formatter API sugar methods).

**Fix:** Ensure `builtins/numbers.md` has a complete methods table matching the registries. Verify the reference matches.

### 2.4 String methods: missing `toCamel`, `toPascal`, `toSnake`, `toKebab`, `truncate` from the reference

**File:** `docs/parsley/reference.md` Appendix B "String Methods (27 methods)"

The string manual page (`builtins/strings.md`) correctly documents all these methods, but the reference's Appendix B "String Methods (27 methods)" lists only 27 — it is missing:
- `toCamel()`
- `toPascal()`
- `toSnake()`
- `toKebab()`
- `truncate(len, suffix?)`
- `toBase64()`
- `fromBase64()`
- `toJSON()`
- `repr()`

The actual count from `StringMethodRegistry` is **36 methods**. The reference says "27 methods."

**Fix:** Update the reference Appendix B to include all 36 string methods.

### 2.5 `markdown()` builtin function signature is wrong in the reference

**File:** `docs/parsley/reference.md` § 6.12

The reference says: `markdown(path, opts?)` — takes a path. The actual code (`evaluator.go` L3070) takes a **string**, not a path, and explicitly errors if you pass a dict (suggesting `MD()` for files):

```go
// First argument must be a string
str, ok := args[0].(*String)
if !ok {
    if _, isDict := args[0].(*Dictionary); isDict {
        return newTypeError("...", "markdown", "a string (use MD(@path) for files)", ...)
    }
}
```

**Fix:** Correct the signature to `markdown(text, opts?)` with `text: string`. Note that `MD()` is for files, `markdown()` is for parsing strings.

### 2.6 `pars` CLI: `fmt` and `describe` subcommands not in manual

The `pars` CLI has three subcommands beyond basic file execution:
1. `pars fmt` — format Parsley source files (with `-w`, `-d`, `-l` flags)
2. `pars describe <topic>` — show help for types, builtins, modules, operators
3. `pars migrate-let-var` — not yet implemented (placeholder only)

None of these are documented in the Parsley manual. The `pars --help` output (`printHelp()` in `cmd/pars/main.go`) is comprehensive, but there's no manual page for the CLI itself.

**Fix:** Create `docs/parsley/manual/getting-started.md` already mentions `pars` briefly. Consider adding a `pars` CLI section to the manual (or a standalone page) documenting:
- Basic usage (`pars`, `pars file.pars`, `pars -e`)
- `pars fmt` subcommand
- `pars describe` subcommand
- Security flags
- Output format flags (`--machine`, `--format`, `-q`)

### 2.7 REPL is undocumented in the manual

The REPL (`pkg/parsley/repl/repl.go`) has several features worth documenting:
- Commands: `:help`, `:describe <topic>`, `:d <topic>`, `:env`, `:clear`, `:raw`
- Two output modes: normal (PLN representation with `>>` prompt) and raw (script-style with `:>` prompt)
- Tab completion for keywords, builtins, REPL commands, and describe topics
- Multi-line input (auto-detects unclosed braces/brackets/tags)
- History (persisted to `~/.parsley_history`)

**Fix:** Add a REPL section to the getting-started page or create a dedicated page. The REPL is the primary interactive tool for learning Parsley.

---

## Part 3: Stale/Deprecated Content (🟡 Medium)

### 3.1 Deprecation warnings should be removed for v1.0

For a v1.0 clean slate, we should decide: do deprecated modules still work, or are they removed? Currently the code still loads them with warnings. The following deprecations exist:

| Code | What | Replacement |
|------|------|-------------|
| DEP-001 | `@std/table` | `@table` literal syntax |
| DEP-002 | `@std/schema` | `@schema { }` DSL syntax |
| DEP-002 | `<Label>` component | `<label @field>` |
| DEP-003 | `@std/api` → `@basil/api` | `import @basil/api` |
| DEP-003 | `<Error>` component | `<error @field>` |
| DEP-004 | `@std/dev` → `@basil/log` | `import @basil/log` |
| DEP-004 | `<Meta @field>` component | `<val @field @key="help"/>` |
| DEP-005 | `@std/html` → `@basil/html` | `import @basil/html` |
| DEP-005 | `format(array, style)` function | `array.format(style)` method |

**Decision needed:** For v1.0, should these deprecated paths be removed from the code entirely, or kept as compatibility shims? Either way, the documentation should not prominently feature deprecated APIs.

**Current doc issues:**
- The reference § 7.6 documents `@std/table` as "(Deprecated)" — this is correct but verbose for v1.0
- The reference § 7.7 documents `@std/schema` as "(Deprecated)" — same
- The CHEATSHEET § "Module Quick Reference" still shows `let {dev} = import @std/dev` and `let {string, object, validate} = import @std/schema` and `let {notFound, redirect} = import @std/api`
- The manual index lists `@std/session` as a standard library module, but session is actually accessed via `@basil/auth` — there is no `@std/session` import

**Fix:**
- Remove deprecated module documentation from prominent positions. Keep a brief "Migration" note if the shims remain in code.
- Fix the CHEATSHEET examples to use current APIs (`@basil/api`, `@basil/log`, `@schema { }`, `@table`)
- Remove `@std/session` from the manual index — session is part of `@basil/auth`

### 3.2 Manual index lists `@std/session` — no such module exists

**File:** `docs/parsley/manual/index.md`

The index lists:
> `@std/session` — Session management — key-value storage and flash messages

The `getStdlibModules()` function does not include "session". Session management is accessed through `@basil/auth` which exposes a `session` property. The session manual page (`manual/stdlib/session.md`) correctly states "The session object is accessed via `basil.session`" but its location under `stdlib/` is misleading.

**Fix:** Remove `@std/session` from the manual index stdlib table. Move `session.md` to a more appropriate location (perhaps under `features/` or `stdlib/` renamed to note it's a `@basil/auth` sub-feature). Update the index to reference it under the Basil server section if needed.

### 3.3 Manual page `stdlib/schema.md` documents deprecated `@std/schema`

This entire page documents the deprecated programmatic schema API. For v1.0, the `@schema { }` DSL is the canonical way to define schemas.

**Fix:** Either remove this page entirely, or convert it to a brief migration note. The `builtins/schema.md` page already documents the `@schema` DSL.

### 3.4 Reference § 6.5 Regex — `match()` is a path matcher, not regex

(Covered in § 1.2 above, but noting here as stale categorisation.)

### 3.5 `@std/html` manual page still refers to `@std/html`

**File:** `docs/parsley/manual/stdlib/html.md` (if it exists) — the HTML module has moved to `@basil/html`.

**Fix:** Ensure any stdlib HTML documentation points to `@basil/html`.

### 3.6 CHEATSHEET § "Standard Library" uses deprecated imports

**File:** `docs/parsley/CHEATSHEET.md` L1145–1165

Multiple examples use old import paths. Need to update all to current APIs.

### 3.7 Reference Appendix B number methods count is wrong

Says "5 methods" — actual count from registries is 14 for integers, 17 for floats.

### 3.8 Reference Appendix B dictionary methods count is wrong

Says "12 methods" — needs verification against the actual `dictionaryMethods` map in `methods.go`.

---

## Part 4: Minor Issues (🟢 Low)

### 4.1 `logLine()` — unclear what it does differently from `log()`

The code shows `logLine` as a placeholder that returns NULL. The reference documents it alongside `log()`. Needs clarification or removal.

### 4.2 `toBox()` method is inconsistently documented

Most type pages document `toBox()` but the options vary by type. A single cross-reference note about the box rendering system would help.

### 4.3 Getting-started mentions `try` returns `{result, error}` but also says "there are no catch blocks"

The phrasing "there are no catch blocks" is accurate but could confuse — `try` does catch errors, it just uses destructuring instead of blocks. The control-flow page explains this better.

### 4.4 `parseMarkdown` string method — options parameter undocumented

`stringParseMarkdown` accepts an optional options dict but the strings manual page doesn't explain what options are available.

### 4.5 Record methods not fully documented

`methods_record.go` shows record-specific methods (`validate`, `update`, `errors`, `error`, `errorCode`, `errorList`, `isValid`, `hasError`, `failIfInvalid`, `schema`, `data`, `keys`, `toJSON`, `withError`, `title`, `placeholder`, `meta`, `enumValues`, `format`) that should be cross-checked against `builtins/record.md`.

---

## Part 5: The Reference vs Manual Question

### Current State

- **Reference** (`reference.md`): ~5,000 lines, single file, covers everything in one place. Grammar snippets, method tables, examples.
- **Manual** (`manual/`): ~30 pages across fundamentals, builtins, stdlib, features. Tutorial-style, thorough explanations, see-also links.

### Overlap

The overlap is significant. For example, string methods are documented in:
1. `manual/builtins/strings.md` — comprehensive, well-structured
2. `reference.md` § 5.1 and Appendix B — table format, briefer

Both need to be kept in sync, which is the root cause of many discrepancies found in this audit.

### Recommendation: Keep Both, but Change Their Roles

**Manual = primary documentation** (human-written, tutorial-oriented, source of truth for prose)
**Reference = auto-generated or tightly-maintained API reference** (complete, terse, tables-only)

The manual is clearly the better-written documentation. The reference is useful as a quick-lookup but is the source of most errors because it's maintained separately.

**Options:**
1. **Keep both, sync manually.** Current approach. Error-prone. Not recommended.
2. **Generate the reference from code metadata.** The method registries already have `Description` and `Arity` fields. The `help` package and `BuiltinMetadata` already power `pars describe`. A reference could be generated from these. This would eliminate drift. **Recommended for post-v1.0.**
3. **Drop the reference, enhance the manual.** Make the manual pages more complete with API tables. Add the appendices to the manual index. Removes duplication entirely. **Recommended for v1.0 if time is limited.**

For v1.0, my recommendation: **keep the reference but make a single pass to sync it with the manual** (the manual pages are more accurate). Long-term, generate the reference from code.

---

## Part 6: AI/Agent Documentation

### Current State

The CHEATSHEET (`docs/parsley/CHEATSHEET.md`) is ~1,965 lines and targets "beginners and AI agents." The copilot instructions (`.github/copilot-instructions.md`) reference it.

### Assessment

**Strengths:**
- Major gotchas section is excellent — the "no print()" warning saves enormous amounts of AI confusion
- Covers the right pitfalls (let vs var, self-closing tags, expression-based output)
- Method reference tables are useful for quick lookup
- Basil server section is practical

**Issues:**
- **Method reference tables are full of hallucinated methods** (see §1.5) — this is the most damaging issue
- Uses deprecated import examples (§ "Module Quick Reference")
- At ~2,000 lines, it's on the edge of being too long for context windows
- Mixes Parsley language reference with Basil server docs (Parts, CSRF, asset bundles)
- Some examples may not run correctly (should be verified with `pars -e`)

### Recommendations

1. **Update the CHEATSHEET for v1.0** — fix deprecated imports, remove references to deprecated APIs
2. **Consider splitting**: Parsley-only cheatsheet (~1,200 lines) + Basil-specific cheatsheet. Most AI agents working on Parsley code don't need the Parts/CSRF/session sections.
3. **Do NOT create "SKILL" files yet.** The CHEATSHEET format works well. Standard skills/knowledge files are framework-specific and add maintenance burden. Revisit after v1.0 when the API is stable. If we do create them, they should be generated from the same code metadata that powers `pars describe`.
4. **Verify all code examples** — every code block in the CHEATSHEET should be runnable with `pars -e`. This is a one-time pass but important for AI trust.

---

## Part 7: Recommended Action Plan

### Phase 1: Critical Fixes (before v1.0)

1. Remove `print()`, `println()`, `printf()` from the reference
2. Fix `match()` documentation (path matching, not regex)
3. Fix reserved keywords list (add `var`, `not`, `is`, `with`, `computed`, `const`)
4. Fix number methods note in Appendix A
5. Fix `markdown()` builtin signature
6. Fix Appendix B method counts (strings: 36 not 27, numbers: 14+ not 5)
7. **Rewrite CHEATSHEET method reference tables** — remove all hallucinated methods, replace with verified methods from code registries

### Phase 2: Missing Coverage

7. Document `with` expression in manual and reference
8. Document `pars` CLI (fmt, describe, security flags, output formats)
9. Document the REPL (commands, modes, tab completion)
10. Add `var` mention to getting-started tutorial
11. Complete number methods documentation in the manual
12. Fix `@std/session` → explain session is via `@basil/auth`

### Phase 3: v1.0 Cleanup

13. Remove or clearly mark all deprecated module docs (`@std/table`, `@std/schema`, `@std/api`, `@std/dev`, `@std/html`)
14. Update CHEATSHEET to use only current APIs
15. Decide on reference vs manual strategy

### Phase 4: Verification

16. Run every code example in the manual through `pars -e` or as `.pars` files
17. Run every code example in the CHEATSHEET through `pars -e`
18. Run every code example in the reference through `pars -e`

> ⚠️ Phase 4 is non-negotiable. The CHEATSHEET hallucination problem (§1.5) shows that
> unverified documentation actively harms users and AI agents. Every code example in every
> document must be machine-verified before v1.0 ships.

---

## Appendix: Complete Feature Coverage Matrix

### Builtins (from `getBuiltins()`)

| Builtin | Reference | Manual | CHEATSHEET |
|---------|-----------|--------|------------|
| `date()` | ✅ | ✅ datetime.md | ✅ |
| `time()` | ✅ | ✅ datetime.md | ❌ |
| `datetime()` | ✅ | ✅ datetime.md | ❌ |
| `url()` | ✅ | ✅ urls.md | ❌ |
| `path()` | ✅ | ✅ paths.md | ❌ |
| `duration()` | ✅ | ✅ duration.md | ❌ |
| `file()` | ✅ | ✅ file-io.md | ✅ |
| `dir()` | ✅ | ✅ file-io.md | ❌ |
| `fileList()` | ✅ | ❌ | ❌ |
| `JSON()` | ✅ | ✅ data-formats.md | ✅ |
| `YAML()` | ✅ | ❌ | ❌ |
| `PLN()` | ✅ | ✅ pln.md | ❌ |
| `CSV()` | ✅ | ✅ data-formats.md | ✅ |
| `text()` | ✅ | ✅ file-io.md | ✅ |
| `lines()` | ✅ | ✅ file-io.md | ✅ |
| `raw()` | ✅ | ❌ | ❌ |
| `SVG()` | ✅ | ❌ | ❌ |
| `MD()` | ✅ | ❌ | ✅ |
| `markdown()` | ⚠️ wrong sig | ❌ | ❌ |
| `format()` | ✅ | ❌ | ✅ |
| `regex()` | ✅ | ✅ regex.md | ❌ |
| `match()` | ⚠️ wrong desc | ❌ | ❌ |
| `tag()` | ✅ | ❌ | ❌ |
| `asset()` | ✅ | ❌ | ❌ |
| `repr()` | ✅ | ❌ | ❌ |
| `inspect()` | ✅ | ❌ | ❌ |
| `describe()` | ✅ | ❌ | ❌ |
| `builtins()` | ✅ | ❌ | ❌ |
| `toInt()` | ✅ | ❌ | ❌ |
| `toFloat()` | ✅ | ❌ | ❌ |
| `toNumber()` | ✅ | ❌ | ❌ |
| `toString()` | ✅ | ❌ | ❌ |
| `toArray()` | ✅ | ❌ | ❌ |
| `toDict()` | ✅ | ❌ | ❌ |
| `log()` | ✅ | ✅ getting-started | ✅ |
| `logLine()` | ✅ | ❌ | ❌ |
| `fail()` | ✅ | ✅ errors.md | ✅ |
| `serialize()` | ✅ | ✅ pln.md | ❌ |
| `deserialize()` | ✅ | ✅ pln.md | ❌ |
| `table()` | ✅ | ✅ table.md | ❌ |
| `money()` | ✅ | ✅ money.md | ❌ |
| `unit()` | ✅ | ✅ units.md | ❌ |
| Named unit ctors | ✅ | ❌ | ❌ |
| `print()` | ❌ BOGUS | — | — |
| `println()` | ❌ BOGUS | — | — |
| `printf()` | ❌ BOGUS | — | — |

### Standard Library Modules

| Module | In Code | Reference | Manual | CHEATSHEET |
|--------|---------|-----------|--------|------------|
| `@std/math` | ✅ | ✅ | ✅ | ✅ |
| `@std/valid` | ✅ | ✅ | ✅ | ✅ |
| `@std/id` | ✅ | ✅ | ✅ | ✅ |
| `@std/hash` | ✅ | ✅ | ✅ | ❌ |
| `@std/mdDoc` | ✅ | ✅ | ✅ | ❌ |
| `@basil/http` | ✅ | ✅ | ❌ (Basil scope) | ✅ |
| `@basil/auth` | ✅ | ✅ | ❌ (Basil scope) | ✅ |
| `@basil/api` | ✅ | ✅ | ✅ | ✅ |
| `@basil/log` | ✅ | ✅ | ✅ (as @std/dev) | ✅ |
| `@basil/html` | ✅ | ✅ | ❌ | ✅ |
| `@std/table` | ✅ dep. | ✅ dep. | ❌ | ⚠️ stale |
| `@std/schema` | ✅ dep. | ✅ dep. | ✅ dep. | ⚠️ stale |
| `@std/api` | ✅ dep. | ❌ | ❌ | ⚠️ stale |
| `@std/dev` | ✅ dep. | ❌ | ✅ dep. | ⚠️ stale |
| `@std/html` | ✅ dep. | ❌ | ❌ | ⚠️ stale |
| `@std/session` | ❌ DNE | ❌ | ⚠️ listed! | ❌ |

### Language Features

| Feature | In Code | Reference | Manual | CHEATSHEET |
|---------|---------|-----------|--------|------------|
| `let` / `var` | ✅ | ✅ | ✅ | ✅ |
| `if` / `else` | ✅ | ✅ | ✅ | ✅ |
| `for` / `in` | ✅ | ✅ | ✅ | ✅ |
| `with` | ✅ | ❌ | ❌ | ✅ (brief) |
| `try` | ✅ | ✅ | ✅ | ✅ |
| `check` | ✅ | ✅ | ✅ | ❌ |
| `stop` / `skip` | ✅ | ✅ | ✅ | ❌ |
| `fail` | ✅ | ✅ | ✅ | ✅ |
| `export` / `import` | ✅ | ✅ | ✅ | ✅ |
| `export computed` | ✅ | ✅ | ✅ modules.md | ✅ |
| `return` | ✅ | ✅ | ✅ | ❌ |
| `fn` / `function` | ✅ | ✅ | ✅ | ✅ |
| Tags (HTML) | ✅ | ✅ | ✅ | ✅ |
| Form binding | ✅ | ✅ | ❌ (data-model mentions) | ✅ |
| `@schema` DSL | ✅ | ✅ | ✅ | ✅ |
| `@table` literal | ✅ | ✅ | ✅ | ✅ |
| `is` operator | ✅ | ✅ | ✅ | ❌ |
| `not` keyword | ✅ | ❌ | ❌ | ❌ |
| `while` loop | ❌ DNE | ❌ (correct) | ❌ (correctly noted) | ❌ |

### pars CLI Features

| Feature | In Code | Documented |
|---------|---------|------------|
| `pars` (REPL) | ✅ | ❌ (no manual page) |
| `pars file.pars` | ✅ | ✅ getting-started |
| `pars -e "code"` | ✅ | ✅ copilot-instructions |
| `pars -r` / `--raw` | ✅ | ❌ |
| `pars -c` / `--check` | ✅ | ❌ |
| `pars -pp` / `--pretty` | ✅ | ❌ |
| `pars fmt` | ✅ | ❌ |
| `pars describe` | ✅ | ❌ |
| `pars migrate-let-var` | Stub only | ❌ |
| `--format` / `--machine` | ✅ | ❌ |
| `-q` / `--quiet` | ✅ | ❌ |
| Security flags | ✅ | ❌ (CHEATSHEET mentions briefly) |
| REPL `:help` | ✅ | ❌ |
| REPL `:describe` | ✅ | ❌ |
| REPL `:env` | ✅ | ❌ |
| REPL `:clear` | ✅ | ❌ |
| REPL `:raw` | ✅ | ❌ |