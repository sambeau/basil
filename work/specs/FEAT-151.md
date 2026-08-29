---
id: FEAT-151
title: "An Html type: escape-by-default tag interpolation"
status: proposed
priority: medium
created: 2026-08-24
author: "@sam / @claude"
related: BUG-032
---

# FEAT-151: An `Html` type — escape-by-default tag interpolation

## Summary

Parsley tags interpolate values raw. `<p>userInput</p>` with
`userInput = "<script>…</script>"` emits the script tag into the page. Escaping
exists only in the form-component path (`@field` labels, values, and errors);
general tag content and attribute values are inserted verbatim, and there is no
built-in way to escape a string.

This feature proposes a marked **`Html`** value type with escape-at-the-tag-boundary
semantics, following the design that Rails (`SafeBuffer`), Jinja2 (`Markup`), and
Phoenix (`{:safe, iodata}`) converged on:

1. Tags evaluate to `Html` instead of plain `string`.
2. Interpolating a plain string into a tag **escapes it**; interpolating `Html`
   passes it through raw. Composition keeps working because components return `Html`.
3. Trusted markup is blessed explicitly with `html(s)`, and sources whose output
   *is* markup (the `MD()` file handle's `.html` field, `.parseMarkdown().html`)
   return `Html` already.
4. Nothing outside tag rendering changes. Strings, operators, CSV, JSON, and the
   database never see the type unless a tag is involved. **Parsley remains a
   general-purpose data language; this proposal must not tax non-web code.**

An interim, independently shippable step — a plain `.escapeHTML()` string
method — is included as Phase 1.

## Motivation

### The gap

```parsley
let u = "<script>alert(1)</script>"
<p>u</p>            // "<p><script>alert(1)</script></p>" — raw, today
```

> **Note (2026-08-29):** attribute *values* are no longer raw. [BUG-052](../bugs/BUG-052.md)
> made `<p title={u}>` HTML-escape its value (`& < > "` → entities), closing an
> attribute-injection hole where the old backslash-escaping let a `"` break out
> into new attributes. So this feature is now specifically about tag **content**
> (`<p>u</p>`) plus the *composition* semantics (a component's `Html` return
> value passing through raw while a plain string is escaped). Attribute escaping
> is done except for context-awareness — a `javascript:` URL in `href={u}` is
> quote-safe but still live, which stays in this feature's Phase 4 (context-aware
> escaping) scope.

Any handler that interpolates request input (`@params`, form fields, query
strings) into a page is an injection vector. No manual page documents this, and
until 2026-08-24 the public site claimed the opposite.

### Why schemas don't close it

Record validation narrows the gap but does not close it. Typed fields
(`email`, `date`, `enum`, `integer`) cannot carry markup, but a *valid* `string`
or `text` field can: a display name of `O'Brien <sales>`, a bio containing
angle brackets. Validation answers "is this a plausible value?"; escaping
answers "how does this value cross into HTML?". They are complementary — and
the Record lifecycle (no value is trusted until validated) is the right mental
model to extend, not replace.

### Honest risk calibration

Raw-by-default is what PHP, Perl, and classic ERB did for decades; Parsley is
not unusually dangerous by that standard. But every current-generation web
system — JSX, Rails 3+, Jinja2, Go `html/template`, Phoenix — escapes by
default. Parsley positions itself as batteries-included for the web; this is
the one battery those stacks have that it lacks.

## Prior art

| System | Mechanism | Key semantics |
|---|---|---|
| Rails | `SafeBuffer` (String subclass), `.html_safe`, `raw()` | ERB escapes plain strings; safe + unsafe concatenation **escapes the unsafe operand** |
| Jinja2 / MarkupSafe | `Markup` (str subclass), `__html__` protocol | Same operand-escaping rule; the most-copied design |
| Phoenix | `{:safe, iodata}` + `Phoenix.HTML.Safe` protocol | Same shape, tuple instead of subclass |
| Go | `html/template` context-aware escaping, `template.HTML` opt-out | Escapes per context (HTML/attr/URL/JS); gold standard, heavyweight |
| React/JSX | Markup is a distinct value type; strings are always text | `dangerouslySetInnerHTML` as the loud opt-out |
| Perl | Taint mode (`-T`): input is tainted, dies at dangerous sinks | The "dirty value" model; viral and coarse, rarely enabled |

Convergent lesson: **a marked type + escaping at the HTML boundary + a named
blessing function**. No successful system relies on per-use-site operators or
sigils — that is manual escaping with shorter syntax, forgotten just as often.

## Design

### The `Html` type

- Go-side: a wrapper around `String` (embed/delegate), so every string method,
  comparison, template interpolation, file write, and JSON/CSV serialisation
  behaves exactly as for a string. Code doing `(<p>"x"</p>).length` or
  `page ==> @./out.html` never notices.
- `.type()` returns `"html"`. `.toString()` returns the raw markup.
- Tags evaluate to `Html`. This is the only new *producer* besides `html()`
  and the markdown surfaces.

### The one rule

At every point where the evaluator interpolates a value into tag **content** or
an **attribute value**:

| Interpolated value | Behaviour |
|---|---|
| `Html` | inserted raw (composition) |
| string | HTML-escaped |
| number, money, date, … | stringified, then escaped (cheap, uniform) |
| `null` | nothing (unchanged) |

Because tags are eager, this happens at tag *construction* — no lazy render
tree, no taint tracking through the language. The decision lives in one place:
the tag-interpolation sites in `eval_tags.go` (the same seam where `@field`
escaping already lives).

### Blessing and unblessing

- `html(s)` — mark a string as trusted markup. Greppable, auditable.
- `.escapeHTML()` — explicit escape, returning `Html` (already-escaped content
  is by definition safe to insert).
- `MD()` handles and `.parseMarkdown()` return `.html` as `Html` — trusted
  output of our own renderer. This removes the most common legitimate need for
  raw interpolation before anyone types `html()`.

### Operators

- `Html + Html → Html` (concatenation of markup).
- `Html + string → Html`, **escaping the string operand** (the MarkupSafe/Rails
  rule). Symmetrically for `string + Html`.
- Everything else — `string + string`, comparisons, methods — unchanged. The
  type is inert outside tag-land.

### Schema and Record integration

Add an `html` field type:

| Field type | Storage | Rendered in a tag |
|---|---|---|
| `string`, `text` | TEXT | escaped |
| `html` (new) | TEXT | raw (value is `Html` on read) |

A CMS body field is `html`; a display name is `string`. The schema remains the
single source of truth — now for *trust* as well as shape, validation, storage,
and form rendering. `@field` rendering of an `html` field in a form edits the
raw markup (textarea content still escaped in transit, as today).

### Out of scope (v1)

- **Context-aware escaping** (URL, `javascript:`, CSS contexts — the Go
  `html/template` level). Documented as a known limitation: `href={userInput}`
  is escaped as an attribute but a `javascript:` URL survives escaping.
- Sanitisation (allowlisting tags in untrusted markup). `html()` is a trust
  assertion, not a cleaner.
- `<script>`/`<style>`/`<SQL>` raw-text bodies — already their own parsing
  context, unchanged.

## Alternatives considered

1. **Do nothing (documented status quo).** Zero breakage; the gap stays open
   and the docs carry the burden forever. Rejected as the end state, acceptable
   as the interim (Phase 1 docs are needed regardless).
2. **Explicit escaping only** (`.escapeHTML()`, no auto behaviour). What PHP
   offers. Opt-in security is forgotten at exactly the call sites that matter.
   Kept as Phase 1 because it is needed as a primitive anyway.
3. **A render-time sigil/operator** (e.g. `<p>!u</p>`). Same per-site memory
   burden as (2) with new syntax; no composition story. Rejected.
4. **Perl-style taint tracking** on values from `@params`/files/network.
   Closest to the Record dirty/clean instinct, but viral through every
   operation, expensive, and wrong for a general-purpose data language where
   most tainted values never go near HTML. Rejected — the *boundary* model
   (escape where values enter markup) gives the same protection at a fraction
   of the surface.
5. **Go-style context-aware escaping.** Strictly better security; requires the
   tag renderer to track lexical context (attribute vs text vs URL). Deferred —
   the `Html` type is a prerequisite for it anyway, so v1 does not foreclose it.

## Breaking changes and migration

- Handlers that deliberately interpolate markup held in a plain string break
  **visibly**: the page shows `&lt;b&gt;…` instead of rendering. The fix is one
  `html()` call (or moving the source to `MD()`/`html` schema fields). Failing
  toward over-escaping is the safe direction — compare silent injection.
- `(<p>"x"</p>).type()` changes from `"string"` to `"html"`. Code switching on
  `.type()` may need a case.
- The test suite is the canary: fixtures that assert raw interpolation of
  markup-bearing strings need `html()` added, and each such site is exactly the
  audit this feature exists to force.
- CHANGELOG entry under **Breaking**, with the migration recipe.

## Phases

1. **`.escapeHTML()` + docs** (independent; no breaking change). Add the string
   method; land the manual updates (done alongside this spec: `security.md`
   gains an XSS section, `tags.md` a warning). Ship in the next release.
2. **The `Html` type** behind the design above: type, tag interpolation rule,
   `html()`, operator semantics, markdown surfaces.
3. **Schema `html` field type** and Record integration.
4. *(Optional, later)* context-aware attribute/URL escaping on top of the type.

## Acceptance criteria (Phase 2+)

- [ ] `<p>u</p>` with `u = "<script>x</script>"` renders `&lt;script&gt;x&lt;/script&gt;`
- [x] `<p class={u}>` escapes the attribute value — done ahead of this feature by [BUG-052](../bugs/BUG-052.md)
- [ ] `<div>Header()</div>` where `Header` returns a tag composes unescaped
- [ ] `html(s)` and `MD(...).html` interpolate raw
- [ ] `Html + string` escapes the string operand; `string + string` untouched
- [ ] `(<p>"x"</p>)` still supports all string methods; writing it to a file emits raw markup
- [ ] Schema `html` fields round-trip raw through DB and forms; `string`/`text` render escaped
- [ ] No behaviour change in any program that never constructs a tag
- [ ] `go build ./...` and `go test ./...` pass; docs and CHANGELOG updated

## Open questions

- Name of the blessing function: `html()` vs `.rawHTML()` vs both. (Rails has
  both `raw()` and `.html_safe`; two spellings may be one too many.)
- Should template strings (`"...{expr}..."`) ever escape? Proposal: no — they
  are general-purpose strings; only tags escape. A template string interpolating
  `Html` stringifies it raw.
- Does `@field` on an `html` schema field need a richer editor story, or is a
  textarea enough for v1?
