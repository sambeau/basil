# Docs audit 2026-08-29 — open decisions and evaluator bugs

An 18-agent sweep ran ~1,465 of the ~1,560 ```parsley blocks across `docs/parsley/`,
`docs/basil/manual/`, and `docs/guide/`, fixing 386 defects (JS/TS/Python-isms, invented
APIs, wrong output claims). The fixes are committed; the distilled patterns are in
[docs/parsley/GOTCHAS.md](../../docs/parsley/GOTCHAS.md). This file records what the sweep
could **not** settle: places where the docs and the evaluator disagree and the evaluator
looks like the wrong one, plus doc-structure questions. In each case the auditor left the
doc describing the *intended* behaviour (or noted where it now describes buggy reality and
should revert once the code is fixed).

## A. Evaluator bugs surfaced by doc claims

1. ~~**Array `.sortBy(fn)` sorts wrongly.**~~ **FIXED.** `[3,1,2].sortBy(fn(x) { x })` → `[1, 3, 2]`;
   `["bbb","a","cc"].sortBy(fn(s) { s.length() })` → `["a","bbb","cc"]`. Negating the key
   sorted correctly, which was the clue: `sortArrayByFunction` sorted the *elements* while the
   comparator indexed a parallel, unsorted `keys` slice, so after the first swap it compared the
   keys of whichever elements had moved into those slots. Keys now travel with their elements
   through `sort.SliceStable`. reference.md's `.sortBy` row described the intended behaviour and
   needed no change. Regression tests: `TestSortByKeyOrdering` in `pkg/parsley/tests/sort_test.go`.
2. ~~**Dict rest-destructuring loses key order.**~~ **FIXED.** `let {id, ...rest} = {id:1, name:"A", active:true}`
   gave `rest.keys()` → `["active","name"]` (sorted), contradicting the documented
   "Ordered key-value pairs" contract. `evalDictDestructuringAssignment` in
   `pkg/parsley/evaluator/eval_expressions.go` built the rest dict from a bare map with no
   `KeyOrder`; it now carries the source's order minus the extracted keys. dictionary.md:301
   and variables.md:146 were already claiming the intent and now run true.
>>>>>>> 3efe1bb (fix(parsley): keep insertion order in dictionary rest-destructuring)
3. **`.fmt(n)` caps at 3 decimal places.** `3.14159.fmt(4)` → `"3.1420"` (rounds at 3, pads
   with zeros). `pkg/parsley/evaluator/methods_numeric.go:417` routes the rounded value
   through a formatter whose `number.Decimal` caps fraction digits at 3. numbers.md:71 left
   claiming `"3.1416"` (the intent).
4. **Styled datetime formatting drops the time.** `@2024-12-25T14:30:00.medium()` renders
   `"Dec 25, 2024"` — identical to a date-only value; `eval_locale.go:251` special-cases the
   two time kinds then falls through to date patterns for datetimes (BUG-045 fixed only the
   time-only case). datetime.md was **changed to match reality** (six sites ~lines 307–344,
   528); if "December 25, 2024 at 2:30 PM" was the design, fix eval_locale.go and revert.
5. **`toBox()` misaligns the title row** when the title is narrower than the table: title row
   wider/narrower than the frame, `┬` instead of `┼` under the title. Affects datetime,
   duration, money, and dictionary `.toBox({title: …})`. Docs now carry the real (broken) art
   in datetime.md ~410, duration.md ~463, money.md:349; dictionary.md:588 left with the clean
   (aspirational) art. Fix the box builder, then re-sync all four.
6. **`return` inside `for` does not exit the function.** The documented "use `return` for
   early exit from loops" (functions.md:78–91) is false: `evalForExpression`
   (eval_control_flow.go:135–142) unwraps the ReturnValue into that iteration's results and
   keeps looping. Fix the evaluator to propagate ReturnValue, or rewrite the subsection.
   Doc left untouched.
7. **Query DSL has no working null test.** `deleted_at is null` fails to parse
   (parseQueryCondition at parser.go:4440 only accepts `is` as an IDENT; it lexes as
   lexer.IS), and the documented-as-wrong `== null` parses but emits `= NULL` (0 rows).
   guide/query-dsl.md L303 left describing the intended `is null`.
8. **Correlated subqueries error.** Two documented forms (guide/query-dsl.md L583–590,
   L686–690) parse but evaluate to "Internal error: unexpected syntax element". Left as
   documented. (Note: the features/query-dsl.md agent ran correlated subqueries successfully
   in its batch — the failing forms are the guide's specific shapes; worth minimizing.)
9. **TableBinding `.insert()` returns null with `id: int(auto)` on SQLite.** The row inserts,
   but generateID() makes a ULID-style id that doesn't match the AUTOINCREMENT rowid, so the
   follow-up find misses and insert() returns NULL; with plain `id: integer` validation fails
   silently (`{valid: false, …}` returned, nothing inserted). guide/query-dsl.md
   Schema-Driven Mutations section blocked on this.
10. **`datetime(auto)` fields are never populated on insert** — createTable emits NOT NULL,
    nothing fills the column, inserts fail; a `datetime = @now` default can't be bound to SQL
    ("unsupported type map[string]interface{}"). Datetime literals can't be SQL params at all
    (same error) — ISO strings work.
11. **TableBinding `.update()` rejects partial payloads** (validates against the full schema),
    so guide/api-table-binding.md's "partial update" PATCH claim is untrue. Decide: skip
    required-checks on update, or stop calling it partial. Both left alone.
12. **`.delete()` returns `{affected: N}`**, not the documented `{deleted: 1}`/`{deleted: N}`
    (features/database.md:346, not yet corrected — flagged cross-batch). Bulk save's real
    shape is `{saved: N, total: M}`.
13. **Tag attribute spread does not override.** dictionary.md:766 claims "later attributes
    override earlier ones"; actually `<button ...base class="x">` emits duplicate `class`
    attributes (plus a stray double space). Fix renderer to dedupe or retract the claim.
14. **`regex.inspect()` is a stub** — returns the receiver (methods_path_url_regex.go:301)
    instead of the `__type` dict its own description promises. regex.md:155 left as intended.
15. **`fromBase64()` on bad input raises an unrelated "Ambiguous date" error** — wrong
    error-catalogue key. strings.md:264 comment describes the intended message; left.
16. **Unit `.short()` uses precision 0** — `#12.3m.short()` → `"12m"`, `#3/8in.short()` →
    `"0in"`, against the code comment "show minimal decimal places"
    (methods_unit.go, formatUnitShort). units.md:426/428/713 now document current behaviour;
    revert when fixed.
17. **`.currency(code)` emits a space after the symbol** (`"$ 99.00"`, from x/text
    currency.Symbol). numbers.md was changed to match; CHEATSHEET.md:1587 and
    reference.md:2153–2154 still claim `"$99.00"`. Decide symbol spacing before touching those.
18. **@std/mddoc rendering defects**: `.toMarkdown()` adds a blank line and drops the trailing
    newline; `.text()` double-spaces inside blocks and fuses headings to preceding text
    (making `.wordCount()` miscount); `.map()`/`.filter()` results re-render from `children`
    so `.toMarkdown()` ignores transformed text. mddoc.md/std-mdDoc.md document reality;
    fix markdown_helpers.go (mdDocToMarkdown) and revert.
19. **`@2024-01-31 + @1mo` → `@2024-03-02`** (Go AddDate overflow). If end-of-month clamping
    was the design ("smart month handling"), that's an evaluator change; doc now states the
    overflow.
20. **`pars` exit codes are inconsistent**: `-e` parse error exits 2, file parse error exits 1,
    `-c` exits 1 for syntax but 2 for unreadable file. cli.md now documents the mess; consider
    making the binary consistent and simplifying the table.
21. **Money/number `format()` vs plain rendering disagree** — `$1999.00` vs `$ 1,999.00`; and
    `format("date")` gives long style for datetime values but short for ISO strings.
22. **`record.keys()` sorts alphabetically** while `record.data()` preserves schema order, so
    programmatic form rendering iterating `keys()` is out of declaration order.
23. **`toDict()`/`inspect()` on datetime/duration leak derived fields** (weekday, unix, iso,
    totalSeconds) into the "clean reconstruction dictionary". Docs now list what's returned.

## B. Language/design questions

1. **Double-quoted tag *attributes* interpolate `{expr}`** — verified (`class="user-{id}"`
   works; an inline JS `onclick="… {method: 'POST'} …"` is a parse error). This contradicts
   the string rule ("double quotes never interpolate") and is documented nowhere in tags.md;
   the old cheatsheet shows it as pre-v1 behaviour. Decide: intended (document it in
   tags.md) or leftover (fix evaluator, revert CHEATSHEET §2/§10 edits). GOTCHAS.md #14
   currently documents it as real.
2. **Fields are required by default** (`Required = !Nullable` unless `auto`) — record.md's
   constraints table reads as if `required` were opt-in. Wording pass wanted.
3. **`@std/api` vs `@basil/api`** (aliases, both silent) and **`basil.auth.user` vs
   `@basil/auth`** — two idioms coexist across the manual; pick one project-wide.
4. **paths `.match()`** requires patterns to include the `@./` leading `.` segment; prose
   calls them "route-style patterns", which argues the evaluator should normalize.
5. **Semicolon statement terminators** work but are documented nowhere; they'd let bare
   expression sequences avoid the `(`/`!`/`not` line-start traps.
6. **http.md says request header keys are lowercase**; server/handler.go:737 copies them
   Go-canonical ("User-Agent"). One is wrong (quick-start now uses canonical case).
7. **parts-js.md `part-form` caveat deleted** — server/handler.go:1638 ships it; if it was
   deliberately soft-launched, restore the caveat and fix parts.md instead.
8. **search-guide.md:204 "Heads-up" corrected** to say results do echo `url` — source-backed
   (server/search.go:813) but not run end-to-end; maintainer eye wanted.
9. **guide/search.md `.update(url, fields)` "partial update"** contradicts
   search-guide.md's wholesale-replace description. Reconcile.

## C. Doc-structure follow-ups

1. **reference.md regeneration risk**: reference.tmpl.md declares sections 5–7 as
   `{{generate}}` output, but the committed reference.md is richer than what
   pkg/parsley/help/compose.go emits — regenerating would wipe the audit's fixes there.
   Decide: hand-maintained (retire template) or move content into fragments/Go metadata.
2. **table.md has two `# Tables` H1s** (~L22 and ~L475) — a visible merge artifact with
   duplicated examples and two columnProps sections.
3. **docs/parsley/security.md** is a near-duplicate of manual/features/security.md and was
   written against an API that never existed (now repaired) — candidate for
   delete-and-redirect.
4. **docs/parsley/std-mdDoc.md** is a stale near-duplicate of manual/stdlib/mddoc.md
   (v0.15.3 frontmatter, deprecated import path) — consolidate or mark superseded.
5. **docs/guide/search.md** carries a Superseded banner; consider redirecting instead of
   maintaining a second copy.
6. **@std/table is removed, not deprecated** — reference.md §7.6 still documents it as a
   module (callout corrected); the section duplicates §5.12 Table and may want deleting.
   Stale "deprecated" wording also at api-reference.md:819, modules.md:217.
7. **CHEATSHEET unclosed-tag fragments** (§2/§9/§10 attribute examples are bare opening
   tags, hard parse errors if pasted) and **styling.md L247/L253** (same) — close them or
   re-fence as ```html.
8. **query-dsl.md L284** is a catalogue of bare `|` clauses inside a ```parsley fence —
   re-fence as plain ``` so doc-test harnesses stop tripping on it.
9. **Two illustrative-seed mismatches in guide/query-dsl.md** result comments (counts assume
   seed rows the Setup block doesn't create) — left as illustrative.
10. **Lint idea from the audit**: inside a ```parsley fence, flag any non-first line starting
    with `(`, `!`, or `not` (the silent line-continuation traps — 15 hits between them), and
    flag `"…{ident}…"` in double quotes outside tag attributes.
