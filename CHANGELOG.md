# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **`fetch` responses are size-bounded** — HTTP fetches now cap how much of a response body is read into memory (default 100 MB), so a large or hostile upstream can't exhaust memory. Override per request with `{maxSize: bytes}` on the format factory (e.g. `text(url("…"), {maxSize: 5_000_000})`); a non-positive value disables the cap. An over-size body is rejected with a clear error rather than silently truncated.
- **Runaway recursion is now a catchable error, not a crash** — The evaluator bounds function-call depth (`evaluator.MaxCallDepth`, default 5000) and returns a Parsley error (`CALL-0007`, "Maximum call depth exceeded") when a function recurses without a base case. Previously such a script would overflow the Go stack and take down the whole process; now it fails cleanly with a hint, and legitimate deep-but-bounded recursion is unaffected.
- **Panic-recovery middleware** — The Basil HTTP server now wraps every request in an outermost recovery layer, so an unexpected panic in a handler or middleware becomes a logged `500` (with the stack trace on stderr; full detail in the response only in dev mode) instead of a dropped connection. Streaming responses (SSE/live-reload) still work — the wrapper forwards flushing via `http.ResponseController`.
- **Protected paths and role-gated routes** — Sites can require authentication for parts of the app: signed-out visitors are redirected to the login page (`login_path`, default `/login`) and signed-in visitors without the right role get a 403. Works site-wide or per-route via `auth: required`, with `basil.auth.user.role` (`"admin"` / `"editor"`) and `basil.auth.required` available to templates.
- **`part-target` and `part-form` for Parts** — An element *outside* a Part can now update it: give the Part an `id` and trigger it from anywhere with `part-click`/`part-submit` via `part-target="id"`, adding `part-form="formId"` to send a named form's fields when the trigger isn't inside one.
- **`dateAfter` / `dateBefore` search filters** — `search.query()` now accepts inclusive date-range filters (Parsley datetime literals or ISO strings), compared in UTC. Documents without a `date` are excluded whenever a date filter is set, and `total` reflects the filtered match count.
- **`url` on search results** — Query result items now include a `url`, so documents added manually with `search.add()` (which have no source `path`) can still be linked.
- **New documentation** — In-depth Parts guide and Parts JavaScript API pages, plus a rewritten, source-verified search guide.
- **Cheat Sheet and Error Codes now on the docs site** — `docs/parsley/CHEATSHEET.md` and `docs/parsley/error-codes.md` are rendered as standalone pages (with the Parsley sidebar and search indexing), so the manual's long-standing links to them no longer dead-end.

### Changed
- **Documented the public vs internal API boundary** — The README and `pkg/parsley/README.md` now state that `pkg/parsley/parsley` is the one supported embedding API (stable across 1.x) and that the other `pkg/parsley/*` packages (evaluator, lexer, parser, ast, format, …) are implementation detail with no compatibility guarantee. Also corrected the embedding example (the old `parsley.Env{}` form and bare `pkg/parsley` import never compiled — it's `parsley.Eval(src, parsley.WithVar(...))` from `pkg/parsley/parsley`) and fixed stale `github.com/sambeau/parsley/...` import paths left over from the pre-Basil module layout.
- **Toolchain and dependencies updated to clear all known vulnerabilities** — Pinned the build toolchain to go1.26.5 and updated `golang.org/x/net`, `golang.org/x/crypto`, `golang.org/x/image`, and `github.com/yuin/goldmark` (plus their transitive `golang.org/x/*` deps). `govulncheck ./...` now reports zero affected vulnerabilities, down from 36.
- **Session tokens are hashed at rest** — The server now stores the SHA-256 of each session token (not the token itself), so a read of the `sessions` table no longer yields usable tokens. The raw token still lives only in the user's cookie and validates as before. One-time effect: existing sessions are invalidated on upgrade (users sign in again); no data migration is needed.
- **Hardened two crypto details** — Recovery-code generation now uses rejection-sampled `crypto/rand.Int` (removing a slight modulo bias toward the first characters of the alphabet) and propagates RNG errors instead of ignoring them; CSRF token comparison now uses `crypto/subtle.ConstantTimeCompare` instead of a hand-rolled loop.
- **Per-call regex compilation hoisted off hot paths** — Several `regexp.MustCompile` calls that ran on every invocation now compile once at package load: the form/tag attribute strippers (`@record`/`@field`/`@tag`/`@key`) and the datetime detection patterns. Localized month-name patterns are memoised per name. No behaviour change; less work per render and per date parse.
- **Codebase-wide formatting, lint, and CI gate** — Ran `gofmt` and applied `golangci-lint`/`modernize` autofixes across the tree (idiomatic `min`/`max`, `maps.Copy`, `slices.Contains`, `range`-over-int, `http.NoBody`, etc.), renamed a local that shadowed the `max` builtin, and modernised octal literals. Scan errors in the search indexer are no longer silently swallowed — they're reported via an optional logger (falling back to stderr). CI now enforces `gofmt` and `go vet` in addition to build and tests.
- **API-key and email-verification lookups are now O(1), not O(n)** — Validating an API key (e.g. on every Git-over-HTTPS request) and looking up an email-verification token previously loaded *every* stored key/token and ran a bcrypt comparison against each — an attacker could force a full bcrypt scan per request. Both now find the single candidate via an indexed lookup (API keys by their deterministic prefix; verification tokens by an added SHA-256 `token_lookup` column) and run at most one bcrypt comparison. High-entropy secrets don't need bcrypt's brute-force resistance for the *lookup*, only for the confirming compare. Fully backward-compatible: existing keys validate unchanged, and pre-existing verification tokens fall back to a bounded, self-clearing legacy scan. Adds auth-DB migrations 15–17 (idempotent).
- **Geomini display font for headings and logo** — Headings (the homepage hero, section titles, docs content headings) and the "Basil & Parsley" logo wordmark now use Geomini, a variable geometric sans (SIL Open Font License, weight axis 200–800), inlined as a woff2 data URI so it always loads with no external request and no fallback flash. The big hero headline and the logo take the heaviest weight (800), the logo with tightened tracking so it reads as a wordmark; smaller headings sit at 700. Body text stays on the system font.
- **Site-wide basil-green accent** — Retinted Pico's primary colour ramp — links, buttons, focus rings, active nav and sidebar states, the Parsley/Basil docs switcher — from the default azure to a green that matches the logo, in both light and dark themes. The homepage hero draws its accent from the same primary, so the brand colour has a single source of truth.
- **Homepage redesign** — The herbaceous.net front page moves from a plain single-column stack to a two-column hero: the pitch and a row of no-toolchain chips on the left, the `csv-page.pars` example framed in a little editor window on the right, with a basil-green accent (tied to the logo, distinct from Pico's azure links) on the kicker, the word "fun", and the install button. The literal types (`@date`, `$money`, `@path`, URLs) now render as a small card grid rather than an inline sentence. Fully theme-aware (light and dark) and built entirely on the site's existing Pico styles and highlight.js grammar — no new dependencies, no build-step changes.
- **Site source reorganised so the homepage is easy to find and edit** — `site/components.pars` had grown into a 337-line grab-bag holding the page layout, the nav components, ~150 lines of inline JavaScript, *and* the homepage copy. It's now split by concern: the homepage content lives in its own [`site/pages/home.pars`](site/pages/home.pars), the HTML shell and navigation in [`site/layout.pars`](site/layout.pars), and the build machinery (renderTree, link rewriting, search indexing) in [`site/lib.pars`](site/lib.pars), leaving `build.pars` as a linear recipe of what gets built. The theme-toggle, search, and latest-release scripts moved out of Parsley string literals into a real [`site/assets/site.js`](site/assets/site.js) (only the pre-paint theme bootstrap stays inline), which also un-blocks editing that JS normally and lets the moon/sun glyphs be literal characters. A new [`site/README.md`](site/README.md) maps "I want to edit X" to the right file. The last content trapped in build code moved out too: the 404 page and the homepage's browser title now live in [`site/pages/404.pars`](site/pages/404.pars) and `home.pars`, so every page's words are in `site/pages/` or `docs/` and the build files never need opening to edit content. Generated output is unchanged apart from the script relocation — all 71 pages and the search index were diffed against a pre-refactor build to confirm.
- **Consistent null propagation for index access (FEAT-150)** — Indexing or slicing into `null` now returns `null` instead of erroring, matching dot access (`null.x` was already `null`). So bracket chains survive a missing intermediate the way dot chains do: `user["profile"]["avatar"]` yields `null` when `profile` is missing, just like `user.profile.avatar`. The `null` receiver short-circuits, so the index sub-expression isn't evaluated. Unchanged: out-of-range indexing on a *present* array/string is still an error (opt into `null` with `[?n]`), missing dictionary keys are still `null`, and type mismatches still error. The access rule is now one line — **absence yields `null`; an out-of-range position is an error** — with `?? fail("…")` documented as the idiom for asserting presence mid-chain.
- **API docs reorganised — server modules now live in the Basil manual** — `@basil/api`, `@basil/log`, `@basil/html`, and Session Management moved out of the Parsley Standard Library into a new "Handler API" section of the Basil manual, where they belong (they only run inside Basil handlers). Each page is retitled to its live `@basil/*` name with a short "Import path" note replacing the misleading "Deprecated" banner — the pages document current functionality, not deprecated features. The `@std/dev` → `@basil/log` page now imports and uses the module consistently (`let {dev} = import @basil/log`). Parsley-manual sidebar no longer duplicates the `@basil/*` entries, and `@std/mdDoc` is corrected to `@std/mddoc`. The site generator gained a symmetric `fixParsleyLinks` so Parsley→Basil cross-links resolve under the flattened site layout.
- **Docs sidebar menus simplified** — Dropped the "Quick Reference by Topic" section from the Parsley manual index (every entry was a duplicate of a page already listed under Fundamentals / Built-in Types / Features / Standard Library, so nothing was orphaned). In the Basil manual, the three in-depth guides (Authentication, Parts, Search) are pulled out of "Features" into their own "Guides" section, and "Session Management" moves from "Handler API" into "Features". Menus are generated from each manual's `index.md`, so these are index-only edits.
- **`@env` and `@args` moved to the Parsley manual** — Both are Parsley-level globals injected by the evaluator (available in the `pars` CLI, the REPL, and Basil handlers alike), but the only prose home for `@env` was the Basil "Server Globals" page, which mis-filed it as server-specific. Added a new **[Globals](docs/parsley/manual/fundamentals/globals.md)** page under Parsley Fundamentals covering `@env` and `@args`, listed it in the Parsley manual index, and cross-linked it from the CLI page. The Basil Server Globals page is now scoped to the genuinely server-injected globals (`@params`, `publicUrl()`, CSRF), points to the Parsley page for `@env`/`@args`, and gained an "Other Server Globals" note for `@DB` and `@SEARCH` (documented on their own pages).
- **Basil reference broken up into pages** — The monolithic `basil/reference.html` had two natures mashed together: core Parsley features (already documented in the Parsley manual) and Basil-server-only extensions, several of which were buried with no home. It's now a slim **Server Reference** map — a link index plus the feature-availability, type, and format-factory tables — and its content is distributed into proper Handler API pages. Three new pages: **Server Globals** (`@params`, `@env`, `@args`, `publicUrl()`, CSRF — previously only in the reference), **@basil/http** (`request`/`response`/`route`/`method` — previously had no manual page at all), and **@basil/auth** (`session`/`auth`/`user`). Session methods, API helpers, dev-logging, and image functions already had manual pages, so those reference sections were dropped in favour of them; the Parsley-language sections now link to the Parsley manual. The site generator collapses `manual/` links and remaps Parsley cross-links for the flattened reference page.

### Removed
- **Dead internal functions** — Removed ~12 unreachable functions flagged by `deadcode`/`staticcheck` (e.g. `evaluator.TableFromDict`, `Lexer.EnterTagContentMode`, `auth.DefaultConfig`, `auth.GetUserFromContext`, `FTS5Index.Weights`, `pln.SerializeWithEnv`, and two unused test helpers). No behaviour change; public embedding APIs and interface-required methods were kept.
- **Deprecated stdlib module pages** — Deleted the `@std/table` and `@std/schema` reference pages (and their inbound links) ahead of v1.0. Both modules were already deprecated in the evaluator; use the built-in `table` type and the `@schema { … }` DSL respectively.
- **Redundant `docs/guide/parts.md`** — The older standalone Parts guide (GitHub-only, not built into the site) was fully superseded by the maintained manual trio — [Parts](docs/basil/manual/parts.md) (reference), [Parts guide](docs/basil/manual/parts-guide.md) (in-depth), and [Parts JavaScript API](docs/basil/manual/parts-js.md). Verified the manual already covers everything it held (prop coercion, HMAC-signed complex props, `.part` routing, and the `part-refresh`/`part-load`/`part-lazy`/`part-target` patterns), then deleted it and repointed its two inbound links (`docs/guide/README.md`, `docs/guide/search.md`) at the manual.

### Fixed
- **SFTP `rmdir({recursive: true})` now actually removes the tree** — The `recursive` option was parsed but silently ignored: `RemoveDirectory` only removes empty directories, so a recursive call on a non-empty directory failed (or misled). It now walks the remote tree depth-first, deleting contents before their parent directories, matching `os.RemoveAll` semantics. A plain non-recursive `rmdir()` on a non-empty directory still errors, as before.
- **String `[i]` indexing and `[a:b]` slicing are now rune-based, not byte-based (BUG-029)** — `.length()` and `for … in` iteration have always counted Unicode codepoints, but indexing and slicing counted bytes, so on non-ASCII text they could land in the middle of a multi-byte character and return an invalid partial byte (`"café"[3]` → `"Ã"` instead of `"é"`; `"café"[0:4]` split the `é`). Both paths in `evalStringIndexExpression`/`evalStringSliceExpression` now convert to `[]rune` first, so index/slice positions, negative indices, out-of-range (`INDEX-0001`) detection, optional access (`[?n]`), and beyond-length clamping are all measured in characters, consistent with `.length()`. **Behaviour change:** any code relying on byte indices into non-ASCII strings will now get character positions instead (`"café"[3]` → `"é"`, `"a中😀"[1]` → `"中"`); pure-ASCII strings are unaffected. The manual now documents the model too: a new **Text & Unicode** section on the Strings page states that `.length()`, `for … in` iteration, and `[i]`/`[a:b]` indexing/slicing are all character- (rune-) based, and the Control Flow page's string-iteration example shows a multi-byte case.
- **Hallucinated/invalid syntax in the Parsley and Basil manuals** — Worked through the code-error audit ([work/reports/PARSLEY-MANUAL-CODE-ERRORS-2026-07-16.md](work/reports/PARSLEY-MANUAL-CODE-ERRORS-2026-07-16.md)), verifying each finding against the parser/evaluator source and `pars` before rewriting. Fixes: `enum(...)`/`string("a","b")` call-form invented for enums → real `enum[...]` bracket form (schema.md, query-dsl.md); `@dur(30, "s")` → real duration literals `@30s`/`@5s` (commands.md); named `fn getUser(req) { … }` declarations (which don't exist) → `let getUser = fn(req) { … }` across api.md, log.md, session.md; ES6 dict shorthand `{name, age}` → explicit `{name: name, age: age}` (dictionary.md); `@insert(Users |< …form .)` spread and `|< name: {userName}` brace-value → real bare-value writes (record.md, query-dsl.md); `try(fn() { … })` → the documented `try fn() { … }()` call form, with the inner `<=?=>` query assigned to a binding since the operator only parses in assignment position (security.md, database.md); `@table` literals whose second row added a column missing from the first (schema.md); `@field phone: Contact.phone` → the real `<form @record>`/`<input @field>` binding (schema.md); `toInt(@params.page ?? "1")` → `toInt((@params.page ?? "1"))` since `??` needs its own parens inside a call arg (search-guide.md); and a comment carrying an apostrophe on a `<Part>` attribute line, which the tag lexer read as an unterminated string and ran to EOF (parts-guide.md). Also corrected two operator false-starts the report flagged — `<=?=>`/`<=#=>` **do** exist, but were shown without their left-hand connection/command operand (`let results <=?=> …` → `let results = db <=??=> …` in table.md; `let result <=#=> cmd` → `let result = cmd <=#=> null` in security.md; the `<=#=>` example in operators.md) — and moved mis-fenced content (the native-HTML tag list in html.md and the `fn(x) → type` signature blocks in images.md) out of `parsley` fences into plain `text`. A follow-up sweep — parse-checking every `parsley` block after the fixes, both whole and line-by-line — turned up three more: a bare `weights: { … }` fragment fenced as runnable code (→ `let weights = { … }`, search-guide.md), a `${input}` JS-ism in the "unsafe SQL" demo that Parsley reads as a literal `$` and which contradicts the manual's own `{expr}`-not-`${expr}` guidance (→ `{input}`, database.md), and a stray `@std/mdDoc` casing in the module table (→ `@std/mddoc`, modules.md). Every rewritten snippet re-checked with `pars`; the only remaining parse failures across the manual are deliberate "❌ error" demos and the expression-listing convention (each line valid on its own).
- **Modal-editor Parts example was incomplete and used invalid syntax** — The "A Modal Editor" example in the Parts guide submitted to a `save` view that was never defined, its edit form dropped the row `id` (so edits couldn't target a row), and it used a `?.` optional-chaining operator that Parsley doesn't have. Added the missing `save` view (validates, then inserts or updates by hidden `id`), added the hidden `id` field to the form, and replaced `person?.name` with plain `person.name` — Parsley's `.` already null-propagates, so `?.` isn't needed (and doesn't parse). Also corrected a stray `maybeNull?.foo()` in the booleans page for the same reason. The full example now passes `pars -c`.
- **Invalid SQL parameter syntax in the docs** — The Parts Guide, Basil Database page, and the older Parts guide all showed parameterised queries as a plain string with `?` placeholders bound by an array — `@DB <=!=> "UPDATE todos SET text = ? WHERE id = ?" <- [text, id]`. That doesn't parse (`unexpected '<-'`, verified with `pars`): a bare string query takes no parameters, and there is no array-binding operator. Rewrote all 13 examples across the three pages to the real form — a `<SQL>` tag with named `:params` supplied as attributes, e.g. `@DB <=!=> <SQL text={text} id={id}>UPDATE todos SET text = :text WHERE id = :id</SQL>` — and confirmed each corrected snippet runs against an in-memory SQLite.
- **`publicUrl()` docs corrected and reframed** — The Server Globals page claimed `publicUrl()` "copies the file into the public assets directory" and returned URLs like `/assets/logo-a1b2c3d4.svg`. Neither is true: it registers a content-hash→path mapping in an in-memory registry and serves the file *in place* from disk at `/__p/{hash}.{ext}` (verified against `server/assets.go`). Rewrote the section to lead with its actual purpose — publishing a private on-disk file as a URL without moving it — with cache versioning as the secondary benefit (the newest version is cached aggressively rather than nothing being cached), plus an explanation of the public directory (`public_dir`, set in `basil.yaml`), where it differs from `publicUrl()`, and the 100 MB size ceiling.
- **Site generator now wipes `dist/` before rebuilding** — `site/build.pars` regenerated pages in place without clearing the output directory first, so pages deleted or renamed at source left orphaned HTML behind (e.g. stale `manual/stdlib/{schema,table,api,html,log,session}.html` from removed/moved reference pages, one of which had dead links). The build now removes `dist/` at the start (`dir(distDir).rmdir({recursive: true})`, a no-op on a first build) and regenerates everything, including CNAME/install.sh/assets. Deployed CI builds were unaffected (they start from an empty checkout); this fixes local staleness and the robustness gap.
- **Wrong `@basil/log` usage in the cheat sheet and reference** — `CHEATSHEET.md` showed `let {log, warn, error} = import @basil/log` / `log.log(…)` and `reference.md` bound the module to `log` while calling `dev.log(…)`; the module exports a single `dev` object (verified against `stdlib_dev.go`), so both were wrong. Corrected to `let {dev} = import @basil/log` / `dev.log(…)`, fixed the reference's `logPage`/`clearLogPage` signatures to match the code, fixed the cheat sheet's `@basil/http` destructure (`params`, not `query`), and corrected remaining `@std/mdDoc` → `@std/mddoc` casing in the reference.
- **Broken internal links in the built docs site** — The generated site had four dead links: `reference.html` → a non-flattened `manual/images.html`, and the Parsley manual's links to the Cheat Sheet, Error Codes, and "Basil" (which pointed at the unrendered `guide/README`). The first is remapped for the flattened layout, the Cheat Sheet and Error Codes pages are now rendered (see Added), and the "Basil" link points at the Basil manual index.
- **Comments docs omitted the shebang exception** — `comments.md` stated flatly that `#` "will cause a parse error", but the lexer has always skipped a `#!` shebang on the first line (for executable CLI scripts). The page now documents shebang lines and scopes the `#`-is-not-a-comment rule accordingly. No code change — the behaviour was already implemented and tested.
- **Multi-byte UTF-8 characters survived only as their first byte inside `<script>`/`<style>` raw text (BUG-028)** — the lexer's raw-text scanner (and the tag-text and CDATA scanners, which shared the pattern) appended one byte per rune while advancing a full rune per step, so `"☾"` in an inline script came out as the invalid byte `e2`. All three scanners now copy the full UTF-8 sequence; `@{}` interpolation in raw text is unaffected.
- **`@SEARCH` now honours its documented options** — The index file option is `path`, as the docs have always said (the code previously only accepted an undocumented `backend` key, so `path: "search.db"` was silently ignored and a default `<watch-dir>_search.db` file appeared instead). `snippetLength` (approximate characters, default 200, capped at FTS5's 64-word limit) and `highlightTag` (default `"mark"`) now work instead of being silently ignored. Unrecognised `@SEARCH` option keys — including `weights` sub-keys — are now an error rather than falling back to silent defaults, and `backend` gets a "did you mean `path`?" hint.
- **Optional access `[?key]` on dictionaries oversold in the docs** — The dictionary manual, reference, and reference-fragments framed `d[?"x"]` as a distinct "safe access" feature, but dictionary access has always returned `null` for missing keys, so `[?key]` is an accepted-but-redundant no-op there (documented as such since it was added in FEAT-014, whose real purpose was out-of-range array/string indexing). The docs now say so plainly and drop the misleading dictionary examples. No code change.
- **Variables manual wrongly claimed dictionary destructuring can't rename** — `variables.md` stated there was "no renaming syntax" and that `let {name: alias}` should be replaced by a separate assignment, but Parsley has supported `as` renaming (`let {a as x} = {a: 5}`, including nested patterns and function params) since it was implemented — as the dictionary manual already documented. The page now shows `as` and reframes the colon caveat accurately (the colon is reserved for *nested* destructuring, so the JS-style `{name: alias}` form is the parse error). No code change.
- **`asset()` docs wrongly claimed it does cache-busting** — The Parsley "Assets" section (`file-io.md`), reference, and API reference all described the core `asset()` builtin as producing content-hashed, cache-busting URLs like `/assets/logo-a1b2c3d4.png` — but that is Basil's server-only `publicUrl()`. `asset()` merely strips the `public_dir` prefix to map a path to its web-root-relative URL (no hashing, and it also accepts `fileList()` dicts). Corrected all three descriptions and examples, and added a mutual cross-reference between `asset()` (Parsley reference) and `publicUrl()` (Basil reference) clarifying which does what. No code change.

## [1.0.0-alpha.4] - 2026-07-13

### Fixed
- **`~` now honours the `g` flag (BUG-027)** — `text ~ /pattern/g` returns every match as documented (array of strings; with capture groups, one `[full, groups…]` array per match; with named groups, one dictionary per match). Previously the flag was ignored and only the first match returned.
- **Manual corrections from the implementation proofread** — regex.md (`.split()` doesn't accept regex; named-group results are dictionaries, not indexed arrays; wrong expected output in a global-replace example; no `.matchAll()`), numbers.md (string parsing is via the `toNumber()`/`toInt()`/`toFloat()` builtins, not a string method), record.md (dictionary spread doesn't exist — use `.data() ++ {…}`; `user.toJSON()` not `json.encode()`; response `.data` not `.parse()`), schema.md (`.toTitle()` not `.toTitleCase()`), types.md (`typeof()` doesn't exist — it's the `.type()` method), index.md (no `while` loop), data-formats.md and urls.md (`MD()` returns `{html, md, raw}`; the removed `markdown()` handle; network reads use `<=/=`, not `<==`).

## [1.0.0-alpha.3] - 2026-07-12

### Fixed
- **`basil --init` generated a config the server rejects** — The scaffold still used the pre-alpha.2 schema: `site: ./site` failed to parse (`site` is now a mapping with a `path` key), and the commented database hint pointed at the removed top-level `sqlite:` key (now `database.path`). Following the scaffold's own printed instructions (`basil --init myapp && cd myapp && basil`) errored out of the box. The init test now loads the generated YAML through the real config parser.
- **Homepage/README code example didn't run** — The CSV example was missing `let` before `emailList <== CSV(...)`.

## [1.0.0-alpha.2] - 2026-07-12

### Added

#### Release & CI Infrastructure
- **goreleaser configuration** — Push a version tag and GitHub Actions builds cross-compiled `basil` + `pars` binaries (macOS/Linux/Windows, amd64 + arm64, CGo-free), packages them with checksums, and publishes a GitHub Release.
- **CI workflow** — Build and test on every push and pull request.
- **`scripts/install.sh`** — One-line installer: detects OS/arch, fetches the newest release (prereleases included), verifies the SHA-256 checksum, and installs both binaries.

#### Image Transformation (FEAT-148)
- **`image()` builtin** — Transform images and serve them at content-hashed, immutable URLs. Auto-rotates via EXIF, strips metadata, resizes (width/height/fit or center-crop), converts format, and caches results to disk. No upscaling; concurrent transforms are deduplicated via singleflight.
- **`imageInfo()` builtin** — Returns `{width, height, format, orientation}` for an image without transforming it. Results are cached in memory (keyed by path + modification time) for efficient gallery-page use.
- **`imageBlur()` builtin** — Generates a Low Quality Image Placeholder (LQIP) as an inline `data:image/jpeg;base64,...` URI (~600 bytes). Use as an instant CSS background while the full image loads.
- **`imageSrcset()` builtin** — Generates multiple resized variants and returns `{src, srcset, width, height}` for responsive `<img>` tags. Supports both width descriptor mode (`[400, 800, 1200]`) and density descriptor mode (`[1, 2, 3], "x"`).
- **WebP output encoding** — All four builtins support WebP output via `gen2brain/webp` (pure-Go, no CGo). Uses native `libwebp` when available on the host, with a WASM fallback (`wazero`) otherwise. WebP typically produces 67–72% smaller files than JPEG for photos.
- **Automatic sharpening on downscale** — A subtle unsharp mask (σ=0.5) is applied after resize to recover edge detail. Disable with `{sharpen: false}` or set an explicit sigma with `{sharpen: 1.2}`.
- **Disk cache at `/__img/`** — Transformed images are served at `/__img/{hash}.{ext}` with immutable `Cache-Control` headers. Cache is stored at `./cache/images/` by default (configurable via `images.cache_dir`).
- **Binary size note** — The `basil` binary grows by ~4 MB (35 MB → 39 MB) due to the bundled WASM runtime. Install `libwebp` on your production host (`brew install webp` / `apt install libwebp-dev`) to use the faster native encoder and confirm the runtime path at startup.

#### Content-Aware Image Operations (FEAT-149)
- **Smart crop** (`{crop: "smart"}`) — Analyses images for faces and high-interest regions (edges, saturation, skin tones, composition) and crops to preserve them. Uses a two-pass architecture: PICO face detection at 640px followed by heuristic scoring at 256px. Pure Go, no new dependencies — the PICO cascade file (234 KB, MIT-licensed) is embedded via `go:embed`.
- **Focal point option** (`{focal: {x, y}}`) — Guide smart crop toward a specific point or region using normalised (0–1) coordinates. Supports both point `{x, y}` and rectangle `{x, y, w, h}` forms.
- **Seam carving** (`{scale: "smart"}`) — Content-aware resizing that removes or inserts low-energy pixel paths (seams) instead of uniformly scaling. Preserves faces, text, and important content when changing aspect ratio. Supports both reduction and enlargement. Based on Avidan & Shamir, SIGGRAPH 2007.
- **Validation** — `crop: "smart"` requires both width and height; `focal` requires `crop: "smart"`; `crop` and `scale` are mutually exclusive. Changes beyond 30% via seam carving log a warning.

#### Language Features
- **Short-circuit evaluation for `&&` and `||`** (BUG-025) — Logical operators now short-circuit correctly: `&&` skips the right operand when the left is falsy, `||` skips when the left is truthy. Collection overloads (intersection/union) are preserved for non-boolean operands.

#### Standard Library
- **Consistent string coercion for typed values** (FEAT-146) — DateTime, Duration, and Unit types now render consistently across tables, templates, and `toString()`:
  - Tables: Duration renders as "2 hours 30 minutes" (was raw dict), Unit renders as "5km" (was `<UNIT>` inspect format)
  - Templates: DateTime uses `.medium()` for human-friendly output ("Jun 15, 2025")
  - `toString()`: DateTime remains ISO format for programmatic use
- **`<field/>` tag and `record.fieldProps()`** (FEAT-145) — Declarative form field rendering with automatic label, type, validation, and accessibility attributes derived from schema metadata.
- **DataTable redesign** (FEAT-144) — Rebuilt DataTable component with sortable columns, pagination, row selection, footer aggregates, empty state, and responsive layout.
- **Meta component and Page restructure** (FEAT-142) — Extracted `<Meta/>` from `<Page/>` for flexible `<head>` management; Page now supports `dir` prop for RTL layouts.
- **Accessibility components** (FEAT-143) — New `<SkipLink/>`, `<VisuallyHidden/>`, `<LiveRegion/>`, `<FocusTrap/>` components; plus `<Details/>`, `<Accordion/>`, `<Dialog/>`, `<Toast/>`, `<Toasts/>`, `<Pagination/>`, `<ErrorSummary/>` with Pico CSS compatibility.
- **Prelude smoke test** — Automated rendering test for all 33 prelude components to catch syntax and runtime errors.
- **Money formatting in tables** — Money values in table cells now render with `.medium()` formatting.

#### Namespace & Organisation
- **`@basil/` namespace for server modules** — `@std/dev` → `@basil/log`, `@std/html` → `@basil/html`, `@std/api` → `@basil/api`. Old `@std/` names kept as deprecated aliases (removal tracked in backlog #121).
- **`@std/mddoc`** — Renamed from `@std/mdDoc` for naming consistency. Old camelCase name kept as deprecated alias.
- **Method registry migration complete** (FEAT-137) — All ~20 remaining types migrated to declarative `MethodRegistry`. `pars describe` now works correctly for all types including array, dictionary, datetime, duration, path, url, regex, file, dir, boolean, null, request, response, DBConnection, SFTP, session, record, table, and more.

### Changed
- **Config YAML consistency** — `cors.maxAge` renamed to `cors.max_age`; top-level `sqlite` key moved to `database.path`; `SessionConfig.HttpOnly` renamed to `HTTPOnly` (YAML tag `http_only`). These are breaking changes to config files from pre-alpha formats.
- **`@std/schema` docs** — Slimmed to deprecation notice pointing to `@schema { ... }` DSL syntax.

### Fixed
- **Helpful errors for non-destructured imports (BUG-005)** — Using a module dictionary as a function (`let Logo = import @./logo.pars` then `Logo()` or `<Logo/>`) now explains that `import` returns a dictionary of exports and shows the destructuring fix (`let { Logo } = import ...`). New error codes CALL-0006 and COMP-0003.
- **Component error wrapper never fired** — A case-sensitive string match meant "Cannot use '<X/>' because 'X' is not a function" (COMP-0002) was unreachable; users saw the raw "Cannot call dictionary as a function" instead. Now matched by error code.
- **Module tests now run outside this machine** — Test fixtures for module imports lived at an absolute path outside the repository; moved to `pkg/parsley/tests/test_fixtures/modules/` so CI can run them.
- **9 prelude component bugs** discovered by smoke test:
  - `.length` used as property instead of `.length()` on arrays (Checkbox, CheckboxGroup, RadioGroup, Pagination)
  - `.format("iso")` vs `.iso` differences in time-related components
  - `.floor()` called on integer division results (unnecessary)
- **DataTable footer null guard** — Simplified using short-circuit `&&` (enabled by BUG-025).
- **`array.join()` skips null and empty strings** — No longer inserts extra separators for nil/empty elements.
- **Search debug output removed** — Removed `fmt.Printf("[DEBUG]...")` statements from search option parsing that were printing to stdout in production.
- **Search text file size limit** — Added 50MB size guard for Markdown/HTML files in search indexer, matching existing PDF/DOCX limits, preventing OOM on oversized files.
- **Git handler path traversal guard** — Added explicit `..` check in `GitHandler.ServeHTTP` before delegating to `go-git-http`, returning 400 Bad Request for traversal attempts.

## [1.0.0-alpha.1] - 2026-02-26

**Parsley 1.0 Alpha** — The first alpha release of the Parsley programming language.

### Breaking Changes
- **`let`/`var` variable declarations** (FEAT-122) — Swift-style immutable/mutable semantics replace bare assignment. `let` for immutable bindings, `var` for mutable.
- **`print`/`println`/`printf` removed** (FEAT-120) — Parsley uses expression-based output; values ARE the output. Use `log()` for debugging.
- **`bytes()` renamed to `raw()`** (FEAT-119) — `bytes()` is now a unit constructor for data sizes.
- **Area constructors renamed** (FEAT-118) — Renamed to match unit suffixes.
- **Global formatting/serialization functions removed** — Replaced with unified `.fmt()` and style methods (FEAT-121).
- **`==>` and `==>>` no longer accept network targets** — Must use dedicated `=/=>` and `=/=>>` for network I/O, enforcing a visible distinction between local and remote operations.
- **`@std/table` module removed** (FEAT-128) — Use `@table` literal syntax instead: `@table [{name: "Alice", age: 30}]`.
- **`format(array, style)` removed** (FEAT-128) — Use the method form: `["a", "b"].format("and")`.
- **Uppercase form components `<Label>`, `<Error>`, `<Meta>` removed** (FEAT-128) — Use lowercase `<label>`, `<error>`, `<val>`.
- **`migrate-let-var` CLI command removed** (FEAT-128).
- **Deprecation warning infrastructure removed** (FEAT-128) — No longer needed for 1.0.

### Added

#### Language Features
- **Measurement units** (FEAT-118) — Length, weight, temperature, volume, area, and data size units with cross-system conversion and derived unit arithmetic (e.g., length × length → area).
- **`with` expression** (FEAT-123) — Scoped field access for cleaner dictionary/record manipulation.
- **Unified formatter API** (FEAT-121) — `.fmt()` method and style methods for all types.
- **Remote write operators `=/=>` and `=/=>>`** — Send data to HTTP endpoints or SFTP servers. Counterpart to the fetch operator `<=/=`.
- **PLN (Parsley Literal Notation)** — Native serialization format with full read/write support (FEAT-115), including Money and DateTime literals (FEAT-116).
- **Named capture groups** (FEAT-127) — Regex named captures return dictionaries.
- **Custom `failIfInvalid` messages** (FEAT-127) — Optional string parameter for record validation.
- **Unicode identifiers and HTML tags** (FEAT-103) — Full Unicode support in variable names and tag names.
- **SQL tag raw text content** (FEAT-117) — Direct attribute syntax for SQL tags.
- **`string.title()` Unicode-aware** — Now uses `golang.org/x/text/cases` for proper title casing.

#### Standard Library
- **`@std/hash` module** — Hashing functions for strings and data.
- **`@std/valid` refactored** — Cleaner validation API.
- **`@basil/` namespace** — Server-specific modules moved to `@basil/http` and `@basil/auth`.
- **`@std/schema` deprecated** in favor of `@schema { ... }` DSL syntax (still works for compatibility).
- **`.reorder()` method** — For dictionaries and arrays.

#### CLI & Tooling
- **`pars -e` expression evaluation** (FEAT-113) — Quick expression testing with PLN output.
- **`pars --raw` / `-r` flag** — File-like output mode for scripts.
- **`pars describe`** — Introspection API for types, methods, builtins, and modules.
- **`pars reference`** — Generate reference documentation in multiple formats.
- **`pars --check` flag** — Syntax checking without execution.
- **Help system** (FEAT-112) — Self-describing modules with unified help infrastructure.
- **Declarative method registry** (FEAT-111) — Methods described via metadata for introspection.

#### Code Formatter
- **AST-based formatter** — Full Parsley code formatter with intelligent line breaking, comment preservation, method chain formatting, multiline tag attributes, and tab indentation.

#### DevTools
- **Browser error pages** — Parse and runtime errors displayed with source line context.
- **Live reload** — Development mode with live browser refresh for templates and CSS.
- **Logs page** — Real-time log display with copy button.
- **Environment page** — Configuration display with secret masking.
- **Component loading infrastructure** — Shared CSS and component system.

#### REPL
- **Raw output mode** — Script-style output for piping.
- **AST formatter for function output** — Formatted function definitions.
- **Table repr** — Formatted table display.
- **Improved indentation** — Correct nesting for functions in arrays/dicts.

#### Database
- **PostgreSQL and MySQL support** (FEAT-107) — In addition to existing SQLite support.

#### Editor Support
- **Tree-sitter grammar** — Full Parsley grammar for syntax highlighting (FEAT-109, FEAT-114).
- **Zed Editor extension** — Language support for Zed.
- **VS Code syntax updates** — Updated for all 1.0 language features.
- **Highlight.js grammar** — Updated for web-based highlighting.

### Changed
- `string.title()` uses Unicode-aware title casing (minor behavior change with apostrophes)
- Structured error model with error codes, hints, and consistent formatting (FEAT-105, FEAT-125)
- Comprehensive documentation overhaul (FEAT-131) — Reference, cheatsheet, manual, and API docs verified against `pars describe`

### Fixed
- Duration multiplication is now commutative (BUG-021)
- Module imports no longer require execute permission (BUG-022)
- Phantom operators and wrong metadata in introspection (BUG-023)
- Dictionary insertion order preserved in `toCSV` (BUG-019)
- ISO date format accepts single-digit months and days
- Record repr uses valid Parsley syntax
- Updated deprecated mailgun API calls
- Updated deprecated goldmark `Text` property usage
- Fixed unchecked error returns on database close in CLI commands
- Acronym handling in case conversion methods

### Removed
- `print`/`println`/`printf` builtins (FEAT-120)
- `@std/table` module (FEAT-128) — use `@table` literal
- `format(array, style)` global function (FEAT-128) — use `array.format(style)`
- `<Label>`, `<Error>`, `<Meta>` uppercase components (FEAT-128)
- `migrate-let-var` CLI command (FEAT-128)
- Deprecation warning infrastructure (FEAT-128)
- 12 unused internal functions (no public API changes)

### Security
- Upgraded to Go 1.26.0 to fix TLS and URL parsing vulnerabilities (GO-2026-4337, GO-2026-4340, GO-2026-4341)
- Updated chi router dependency to v5.2.2 to fix host header injection vulnerability (GO-2025-3770)

## [0.2.0] - 2025-06-20

### Added
- **Basil Web Server** (FEAT-002)
  - Full HTTP/HTTPS server with route-based request handling
  - HTTPS with automatic Let's Encrypt certificates (autocert)
  - Development mode (`--dev`) with HTTP and live browser refresh
  - SQLite database support with Parsley operators (`<=?=>`, `<=??=>`, `<=!=>`)
  - Request logging (text and JSON formats)
  - Form parsing: URL-encoded, multipart/form-data, JSON body
  - Security headers (HSTS, CSP, X-Frame-Options, etc.)
  - Reverse proxy support (X-Forwarded-For, X-Real-IP)
  - AST caching for production performance
  - SIGHUP handler for production script/cache reload
  - Route-based response caching with configurable TTL
- **Semantic Versioning** (FEAT-005)
  - Git tag-based versioning with build-time injection
  - Makefile with `build`, `dev`, `test`, `clean`, `check` targets
  - `--version` flag shows version and commit hash

### Fixed
- Live reload no longer creates excessive log noise (BUG-004)

## [0.1.0] - 2025-11-30

### Added
- Development Process Framework (FEAT-001)
  - `AGENTS.md` for AI operational context
  - `ID_COUNTER.md` for automated ID allocation
  - `BACKLOG.md` for deferred items tracking
  - Instruction files for code standards and commit conventions
  - Prompt files for feature, bug, and release workflows
  - Document templates for specs, bugs, and implementation plans
  - Human-friendly guide documentation (quick-start, cheatsheet, FAQ, walkthroughs)

### Changed
- Updated `.github/copilot-instructions.md` with project workflow

## [0.0.0] - 2025-11-30

### Added
- Initial project setup
- Basic Go module structure
- VS Code debug configuration
