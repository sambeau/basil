---
id: FEAT-150
title: "Consistent Null Propagation for Index Access"
status: implemented
priority: high
created: 2026-07-15
author: "@human / @ai"
related: FEAT-014, FEAT-105
---

# FEAT-150: Consistent Null Propagation for Index Access

## Summary

Parsley's two access forms disagree about `null`. **Dot access propagates `null`**
(`null.x` → `null`; `user.profile.avatar` walks through a missing `profile` and yields
`null`), but **bracket access does not** (`null["x"]` → *error*), and the optional-index
operator `[?]` does not close the gap (`null[?"x"]` also errors). This makes
`user.profile.avatar` succeed while the equivalent `user["profile"]["avatar"]` throws.

This feature makes bracket access propagate a `null` receiver exactly as dot access
already does, so the two forms behave identically. It establishes one clear, teachable
rule for the whole access surface — **absence yields `null`; an out-of-range position is
an error** — and records the design rationale that was previously undocumented.

Out-of-bounds array/string indexing stays strict (an error), with `[?]` retained as the
opt-in for safe positional access. No new syntax is added.

## User Story

As a Parsley developer walking possibly-incomplete data (config, JSON responses, template
contexts), I want `data["a"]["b"]["c"]` to yield `null` when an intermediate step is
missing — the same way `data.a.b.c` already does — so that bracket access and dot access
tell one consistent story and I don't have to remember which one crashes.

As a developer who wants a missing value to fail loudly at a specific point, I want a clear
idiom (`data["a"] ?? fail("a is required")`) to assert presence mid-chain, so that
forgiving-by-default access never traps me into silent nulls when I actually want an error.

## Motivation

The immediate trigger was noticing that `[?]` is a no-op on dictionaries (documented in the
[dictionary manual](../../docs/parsley/manual/builtins/dictionary.md) after FEAT-014). But
investigation (see Design Decisions) showed the arrays-vs-dictionaries wart was a shallow
symptom. The real inconsistency is **dot vs bracket**:

| Access | plain | optional `[?]` |
|---|---|---|
| `d["missingKey"]` (dict) | `null` | `null` (identical) |
| `a[99]` (array out-of-bounds) | **error** | `null` |
| `null["x"]` (null receiver, bracket) | **error** | **error** ← `?` does not help |
| `null.x` (null receiver, dot) | **`null`** | — |
| `u.profile.avatar` (missing mid-chain, dot) | **`null`** | — |
| `u["profile"]["avatar"]` (missing mid-chain, bracket) | **error** | **error** ← `?` does not help |

Parsley has *already* committed to forgiving, null-propagating access — through dot. Dot
access explicitly propagates `null` ([`evalDotExpression`, evaluator.go:5604](../../pkg/parsley/evaluator/evaluator.go)):
`// Null propagation: property access on null returns null`. Bracket access simply never
got the mirror. The fix is to give it one.

### Design Principles

- **One rule for the whole access surface.** *Absence yields `null`; an out-of-range
  position is an error.* Dot and bracket must agree.
- **Consistency with what already ships.** Dot access is the anchor — it is forgiving and
  cannot be made strict without breaking every optional-chaining template. Bracket access
  moves to match it, not the reverse.
- **Keep the bug-catch that earns its keep.** Out-of-bounds array/string indexing is almost
  always a logic error; it stays strict. `[?]` remains the deliberate opt-in.
- **No new syntax.** Strictness mid-chain is already expressible (`?? fail("msg")`); the
  language does not need a non-null-assertion operator.
- **The Parsley aesthetic:** simple, minimal, complete, composable.

## Acceptance Criteria

### Phase 1: Null-receiver propagation for bracket access

- [x] `null["x"]` returns `null` (was: index type error) — mirrors `null.x`
- [x] `null[0]` returns `null` (any index type on a `null` receiver → `null`)
- [x] `null[?"x"]` returns `null` (the `?` is irrelevant when the receiver is `null`)
- [x] Chained bracket access propagates: with `let u = {name: "a"}`,
      `u["profile"]["avatar"]` returns `null` (was: error at the `null["avatar"]` hop)
- [x] Mixed chains agree: `u["profile"].avatar` and `u.profile["avatar"]` both return `null`
- [x] The `null` receiver short-circuits: the index sub-expression is **not** evaluated when
      the receiver is `null` (matches dot access, which never evaluates the key on a `null`
      receiver). e.g. `null[sideEffect()]` does not call `sideEffect`.
- [x] Slice access mirrors the same rule: `null[1:2]` returns `null`

### Phase 2: Everything else stays exactly as it is (regression guards)

- [x] `dict["missingKey"]` → `null` (unchanged; absence on a present dict)
- [x] `arr[99]` → **error** `INDEX-0001` (unchanged; out-of-range on a present array)
- [x] `arr[?99]` → `null` (unchanged; opt-in positional safety)
- [x] `"hello"[99]` → **error**; `"hello"[?99]` → `null` (unchanged)
- [x] Type mismatch stays an error: `arr["x"]`, `5[0]`, `true[0]` → error. Leniency applies
      **only** to a `null` receiver, never to a type mismatch, and `[?]` does not swallow
      type errors.
- [x] `[?]` on a dictionary remains a no-op (a dictionary has no positional range) — now
      documented as a consequence of the rule, not an accident.

### Phase 3: Documentation & the strict-assertion idiom

- [x] Update the access rule everywhere it appears, stating it once, consistently:
  - [x] `docs/parsley/manual/builtins/dictionary.md`
  - [x] `docs/parsley/manual/fundamentals/operators.md`
  - [x] `docs/parsley/manual/fundamentals/errors.md`
  - [x] `docs/parsley/reference.md` and the `docs/parsley/reference-fragments/` sources
        (`02-operators.md`, `10-error-handling.md`)
- [x] Document the null-propagation triad, keyed off `??`:
  - `a["b"]["c"]` — **propagate** (null flows through; the common case)
  - `a["b"] ?? "fallback"` — **recover** (supply a value at this hop)
  - `a["b"] ?? fail("b is required")` — **assert** (fail loudly, here, with a reason)
- [x] `CHANGELOG.md` entry under `## [Unreleased]` (Changed)

### Non-Goals

- No dedicated non-null-assertion operator (e.g. a postfix `!`). Deferred; `?? fail()`
  covers the need. Revisit only if real usage shows it is too noisy in hot paths — it is a
  non-breaking addition later.
- No change to dot access (already correct).
- No change to out-of-bounds or type-mismatch behavior.
- Not making `[?]` an error on dictionaries. It stays a tolerated, now-explained no-op.

## Design Decisions

> This section is the record the codebase previously lacked. Before this feature there was
> **no** written rationale anywhere in `work/` for why dictionaries returned `null` while
> arrays errored; FEAT-014 treated dict-returns-`null` as a pre-existing fact and added
> `[?]` for arrays/strings only, with a one-line "accepted for API consistency" comment on
> the dictionary path. This settles the question deliberately.

### The unifying principle: absence vs. out-of-range

**Decision**: Access has two distinct failure modes and they get different answers.

- **Absence** — reaching *into* a `null` receiver, or asking a dictionary for a key it does
  not have. Absence is a *normal query result* → `null`.
- **Out-of-range** — naming a position outside a container that *exists* (`arr[99]`). This
  is *almost always a logic error* (off-by-one, wrong length assumption) → error, opt out
  with `[?]`.

**Rationale**: These are genuinely different questions, so treating them differently is
principled, not inconsistent. Absence is "is there anything here?" — `null` is a useful,
truthy-testable answer. Out-of-range is "give me slot 99 of these 3" — surfacing that at
the fault site with `index 99 out of range (length 3)` is far more useful than a `null`
that drifts three functions downstream and manifests as garbage (the JavaScript
`arr[99] === undefined` footgun). This split is not novel: Go reads a missing/nil map key
as the zero value (absence is lenient) but panics on slice out-of-bounds (positional is
strict).

### Unify toward forgiving (match dot), not toward strict

**Decision**: Make bracket access propagate a `null` receiver, rather than making dot
access (and dictionaries) strict.

**Rationale**: "Make it consistent" has two solutions pointing in opposite directions:
(a) forgiving everywhere, or (b) strict everywhere. Direction (b) would require dot access
to start erroring on missing keys — breaking every `user.maybeMissing` chain in existing
templates and destroying the optional-chaining ergonomics the language relies on. Dot
access is therefore an anchor that cannot practically move. The only coherent option is to
bring bracket access to it. This also matches Parsley's revealed philosophy: null-safe
`in`/`not in`, `??`, and null-propagating `.` are all already forgiving.

### Cross-language grounding

**Decision**: Adopt forgiving-by-default receiver access with no operator required, as in
the data-transformation languages Parsley most resembles.

**Rationale**: Languages split into two camps on a `null`/nil receiver:

- *Forgiving by default (no operator):* Objective-C (nil-messaging), **Clojure**
  (`(get nil k)`, `(get-in nil ks)` → `nil`), **Elixir** (`get_in`/Access propagate `nil`),
  Perl, PHP (with a warning), Go (nil-map reads).
- *Strict, opt-in via `?.`/`&.`:* JavaScript/TS, C#, Kotlin, Swift, Ruby, Groovy — but the
  statically-typed ones lean on the **type system** to force `?.` where needed.
- *Strict, no opt-in:* Python, Lua.

The strict-with-`?.` mainstream depends on a compiler nagging you into the operator. Parsley
is dynamically typed and template-oriented; it has no such safety net, so strict-by-default
just means runtime crashes on ordinary missing data. The forgiving camp — Clojure and
Elixir especially, both built around walking maybe-present nested data — is the right
reference class, and Parsley already sits in it via dot access. Go additionally validates
the *specific* combination adopted here: absence-lenient, out-of-bounds-strict.

### Keep `[?]`; give it one honest job

**Decision**: Retain `[?]` and define it crisply as *"tolerate an out-of-range positional
index."*

**Rationale**: Once absence (including `null` receivers) is handled by default, `[?]`'s
only remaining job is opt-in positional safety on arrays/strings — `arr[?0] ?? default`,
its original FEAT-014 purpose. That is a real, non-redundant job. On dictionaries it stays a
no-op, but now for a stated reason (*a dictionary has no positional range, so there is
nothing to tolerate*), which is honest and one line to document — not the accident it looked
like before. Removing `[?]` was considered and rejected: it would delete a genuinely useful
array/string affordance to fix a cosmetic dictionary wart.

### No non-null-assertion operator — use `?? fail()`

**Decision**: Do not add a postfix strict-access operator. Standardise the strict-assertion
idiom on the existing `x ?? fail("msg")`.

**Rationale**: Forgiving-by-default has a real cost — a `null` from a genuine mistake
propagates silently. The mirror of safe-navigation is a non-null assertion (Kotlin `!!`,
Swift `!`, Rust `.expect("msg")`). Parsley already expresses this *inline and per-hop*
without new syntax: `??` short-circuits (verified: `1 ?? fail("boom")` → `1`) and `fail()`
(from FEAT-105's unified error model, catchable by `try`) produces a located, custom error.
So `(a["b"] ?? fail("expected b"))["c"]` asserts presence exactly where the invariant lives
and reports it with a human-readable reason pointed at that hop — equivalent to Rust's
`.expect()`, the gold-standard form, and *better* than a bare `!!` that emits a generic
error. This yields a complete, symmetric system off one operator — **propagate / recover /
assert** — so going forgiving-by-default surrenders nothing on the strict side. A dedicated
operator is a non-breaking future addition if evidence ever warrants it; adding punctuation
pre-emptively is the harder-to-reverse move.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### The one behavioural change

`null` receiver on index/slice access must yield `null`, short-circuiting before the index
is evaluated — exactly mirroring `evalDotExpression`.

Precedent to mirror, [`evaluator.go:5604`](../../pkg/parsley/evaluator/evaluator.go):

```go
// Null propagation: property access on null returns null
if left == NULL || left == nil {
    return NULL
}
```

Primary change site — the `*ast.IndexExpression` dispatch in `Eval`
([`evaluator.go:5036`](../../pkg/parsley/evaluator/evaluator.go)). Add the guard *after*
evaluating `left` and *before* evaluating the index, so a `null` receiver short-circuits
the index sub-expression (matching dot, which never evaluates the key on `null`):

```go
case *ast.IndexExpression:
    left := Eval(node.Left, env)
    if isError(left) {
        return left
    }
    // Null propagation: indexing into null returns null (mirrors evalDotExpression)
    if left == NULL || left == nil {
        return NULL
    }
    index := Eval(node.Index, env)
    if isError(index) {
        return index
    }
    return evalIndexExpression(node.Token, left, index, node.Optional)
```

Apply the same guard to the `*ast.SliceExpression` dispatch
([`evaluator.go:5047`](../../pkg/parsley/evaluator/evaluator.go)) so `null[1:2]` → `null`.

A defensive mirror may also be added at the top of `evalIndexExpression`
([`eval_operators.go:121`](../../pkg/parsley/evaluator/eval_operators.go)) — return `NULL`
when `left.Type() == NULL_OBJ` — so any other caller of that function inherits the rule.
This second guard cannot short-circuit the index (already evaluated by then) but keeps the
invariant local to the function.

Everything else in `evalIndexExpression` — the per-type handlers, the `optional` flag, the
out-of-bounds `INDEX-0001` error, the type-mismatch error — is unchanged.

### Behaviour matrix (target)

| Expression | Receiver | Today | Target | Changed? |
|---|---|---|---|---|
| `null["x"]` / `null[0]` | null | error | `null` | **yes** |
| `null[?"x"]` | null | error | `null` | **yes** |
| `null[1:2]` | null | error | `null` | **yes** |
| `u["profile"]["avatar"]` (profile missing) | null mid-chain | error | `null` | **yes** |
| `d["missingKey"]` | present dict | `null` | `null` | no |
| `d[?"missingKey"]` | present dict | `null` | `null` | no |
| `arr[99]` | present array | error `INDEX-0001` | error `INDEX-0001` | no |
| `arr[?99]` | present array | `null` | `null` | no |
| `"hi"[99]` / `"hi"[?99]` | present string | error / `null` | error / `null` | no |
| `arr["x"]`, `5[0]` | type mismatch | error | error | no |
| `null.x` | null | `null` | `null` | no (dot; already correct) |

### Affected Components

| File | Change | Description |
|------|--------|-------------|
| `pkg/parsley/evaluator/evaluator.go` | **MODIFY** | `null`-receiver guard in the `*ast.IndexExpression` and `*ast.SliceExpression` dispatch cases (short-circuit before index eval) |
| `pkg/parsley/evaluator/eval_operators.go` | **MODIFY** | Optional defensive `NULL_OBJ` guard at the top of `evalIndexExpression`; update the leading doc comments to state the absence-vs-out-of-range rule |
| `pkg/parsley/tests/optional_index_test.go` | **MODIFY** | Add regression cases confirming out-of-bounds and type-mismatch behaviour is unchanged |
| `pkg/parsley/tests/null_propagation_test.go` | **NEW** | `null`-receiver propagation for bracket + slice; chained/mixed dot+bracket chains; index short-circuit (no side effect on `null` receiver); the `?? fail()` assertion idiom |
| `docs/parsley/manual/builtins/dictionary.md` | **MODIFY** | Restate the rule; keep the `[?]`-is-a-no-op note, now framed as a consequence |
| `docs/parsley/manual/fundamentals/operators.md` | **MODIFY** | State the one access rule; document the propagate/recover/assert triad |
| `docs/parsley/manual/fundamentals/errors.md` | **MODIFY** | Optional access + `?? fail()` assertion idiom |
| `docs/parsley/reference.md` | **MODIFY** | Regenerated from fragments |
| `docs/parsley/reference-fragments/02-operators.md` | **MODIFY** | Access rule + triad (source of truth) |
| `docs/parsley/reference-fragments/10-error-handling.md` | **MODIFY** | Optional access + assertion idiom (source of truth) |
| `CHANGELOG.md` | **MODIFY** | `## [Unreleased]` → Changed |

### Edge Cases & Constraints

1. **Index side effects on a `null` receiver** — must *not* run. The guard sits before the
   index `Eval`, matching dot access. Test: `null[sideEffect()]` leaves `sideEffect`
   uncalled.
2. **`null` vs. type mismatch** — leniency is *only* for a `null` receiver. `5[0]`,
   `true["x"]`, `arr["x"]` remain type errors; `[?]` does not swallow them.
3. **`response`-dict unwrapping** — `evalIndexExpression` first unwraps a `response` dict's
   `__data`. If `__data` evaluates to `null`, indexing it now yields `null` (consistent).
4. **Nested `null` from out-of-bounds is still not silent** — `arr[99]["x"]` errors at
   `arr[99]` (out-of-range, strict), so the chain still fails loudly where the *positional*
   mistake is. Only *absence* propagates.
5. **Assignment targets** — this feature concerns read access only. `null["x"] = 1` (index
   assignment to a `null` target) is out of scope and keeps its current behaviour.
6. **Truthiness unaffected** — `null` remains falsy, so `if (a["b"]["c"]) { … }` and
   `a["b"] ?? …` continue to work as before.

### Testing

- [x] `null["x"]`, `null[0]`, `null[?"x"]`, `null[1:2]` → `null`
- [x] `let u = {name:"a"}`: `u["profile"]["avatar"]` → `null`; `u["profile"].avatar` →
      `null`; `u.profile["avatar"]` → `null`
- [x] Short-circuit: index expression with a side effect is not evaluated on a `null`
      receiver
- [x] Regression: `arr[99]` errors `INDEX-0001`; `arr[?99]` → `null`; `"hi"[99]` errors;
      `arr["x"]`, `5[0]` error
- [x] Regression: `dict["missing"]` → `null`; `dict[?"missing"]` → `null`
- [x] Idiom: `(a["b"] ?? fail("no b"))["c"]` → value when present, located error when
      missing; `try` catches the error
- [x] `go build ./...` and `go test ./...` pass

## Related

- Spec: `work/specs/FEAT-014.md` — Optional Indexing `[?n]` (introduced `[?]`; the origin
  of the arrays-vs-dictionaries asymmetry)
- Spec: `work/specs/FEAT-105.md` — Unified Error Model (`fail()`, the assertion primitive)
- Precedent in code: `evalDotExpression` null propagation,
  `pkg/parsley/evaluator/evaluator.go` — the behaviour bracket access is being aligned to
- Prior doc fix: CHANGELOG `## [Unreleased]` — "Optional access `[?key]` on dictionaries
  oversold in the docs" (corrected the docs; this feature corrects the semantics)
