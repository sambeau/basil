---
id: PLAN-131
title: "Website & Public Launch Plan (herbaceous.net)"
status: in-progress
created: 2026-07-12
---

# PLAN-131: Website & Public Launch

## Overview

Build and launch the public website for Basil & Parsley at **herbaceous.net**,
including release/download infrastructure. The site launch is effectively the
public launch of the project, so the pre-release punch list is folded in.

## Key Decisions

- **Hosting: GitHub Pages.** Free, zero ops, survives a Hacker News spike,
  custom domain + HTTPS built in. A live Basil-served site (or `try.` playground)
  can come later, post-1.0 — the content won't change, just the delivery.
- **Static site generator: Parsley itself.** A `build.pars` script renders the
  markdown docs through Parsley tag components into Pico-styled HTML.
  Dogfooding is the pitch: the site footer links to its own build script.
  Fallback if it fights us: Hugo (single binary, no npm — fits the ethos).
- **Styling: Pico CSS**, vendored (no CDN). Matches the prelude components'
  Pico compatibility and the plain, modern, not-flashy brief (like picocss.com/docs).
- **Syntax highlighting:** the existing `contrib/highlightjs/parsley.js` grammar.
- **Downloads: GitHub Releases + goreleaser.**
  - Permalink to latest: `github.com/sambeau/basil/releases/latest/download/<asset>`
    — download buttons never go stale.
  - Live version badge via `api.github.com/repos/sambeau/basil/releases/latest`.
  - One-line install: `curl -fsSL https://herbaceous.net/install.sh | sh`
    (installer hosted on the site, detects OS/arch, fetches latest release).
  - Homebrew tap via goreleaser (`brew install sambeau/tap/basil`).
  - ⚠️ Verify Windows cross-compile (smartcrop/seamcarve/WASM WebP paths are
    pure Go so it *should* be clean) before promising a Windows binary.
- **Domain:** herbaceous.net, DNS at Hover → GitHub Pages (4 × A records for
  apex + CNAME file in the Pages branch), enforce HTTPS.

## Tone & Messaging

Friendly, fun, humble, personal — a tool for making websites quickly, not a
startup pitch. The story: bringing back the fun of early-2000s web development
— no compiler, no npm, no TypeScript, no build step. The quirks (rich literals
like `@2024-01-15` and `$99.99`, the `<==` slurp operator, first-class HTML)
exist for reasons — sometimes hard-core comp-sci, sometimes playfulness — and
the site should say so, in first person.

The homepage must answer: **What is it? What can it do? How does it do it?
Why did we make it? Why should anyone care?**

Do **not** port dev-process docs (`work/`, the AI-workflow parts of
`docs/guide/README.md`) to the site — contributor docs stay in the repo.

## Phases

### Phase 1 — Release plumbing *(first: everything links to it)*
- [x] goreleaser config (`.goreleaser.yaml`) — `basil` + `pars`, all platforms
      amd64+arm64, CGo-free (verified: modernc SQLite, wasm WebP; Windows and
      Linux cross-compile clean), checksums, both binaries in one archive per
      platform. Homebrew tap section stubbed (needs tap repo + PAT secret).
- [x] GitHub Actions release workflow (`.github/workflows/release.yml`) —
      tests gate the release; goreleaser config validated with
      `goreleaser check`
- [x] CI workflow (`.github/workflows/ci.yml`) — build + test on push/PR
- [x] Fix BUG-005 — new CALL-0006/COMP-0003 errors with destructuring hints;
      fixed dead COMP-0002 wrapper (case-sensitive match); regression tests.
      See `work/bugs/BUG-005.md` resolution.
- [x] Module test fixtures moved into repo (were at an absolute path outside
      the repository — would have failed in CI)
- [x] `install.sh` (`scripts/install.sh`) — OS/arch detection, newest release
      via API (prereleases included), checksum verification
- [x] Test install.sh end-to-end (v1.0.0-alpha.2 published 2026-07-12; installer
      verified: platform detection, checksum, working binaries)
- [x] Housekeeping: `.kbz/` and `testdata/images/smartcrop/` (855MB) gitignored
- [x] Committed, changelog cut, tagged, pushed — v1.0.0-alpha.2 released via
      Actions (authorised by Sam 2026-07-12). First CI run caught and fixed:
      parts tests hardcoding a machine-specific absolute path, and a
      platform-dependent path-traversal security test.

### Phase 2 — Site skeleton
- [x] `site/` directory: `build.pars` (Parsley static generator),
      `components.pars` (Layout, HomePage), vendored Pico CSS v2.1.1
- [x] GitHub Actions (`site.yml`): build `pars` → run `build.pars` → deploy
      to Pages. **Live at https://sambeau.github.io/basil/** (2026-07-12)
- [x] highlight.js + contrib Parsley grammar wired in; github/github-dark
      themes switched by `prefers-color-scheme` (matches Pico)
- [x] Renders all 48 manual pages + homepage + 404; `.md` links rewritten to
      `.html`; `install.sh` served from site root
- [x] DNS + domain live (2026-07-12): Sam added the four apex A records and
      www CNAME at Hover (removed an over-broad wildcard A record); custom
      domain attached via API, Let's Encrypt cert issued in ~2 min, HTTPS
      enforced. **https://herbaceous.net is live** — homepage, manual, and
      `curl -fsSL https://herbaceous.net/install.sh | sh` all verified.
      Note: with workflow-based Pages the CNAME file in dist/ is ignored —
      the domain is set via API/settings.
- [ ] Optional: verify the domain under GitHub Settings → Pages → Verified
      domains (prevents subdomain takeover if DNS ever lapses)

**Parsley gotchas learned writing build.pars** (useful for Phase 3–4):
- `+` concatenates strings; `++` merges/wraps into arrays (README had this wrong)
- regex `.replace(/x/, y)` replaces the *first* match only — use `/x/g`
- `fileList()` returns file handles (`.path` for the path), takes one arg,
  and `**` doesn't match files in the root directory — list both
- raw strings don't span lines; `<script>`/`<style>` contents are raw text
  (don't quote them or the quotes end up in the JS)
- emoji can't appear bare in tag contents — put them in strings
- a line starting with `[` continues the previous expression as indexing

### Phase 3 — Story pages *(where the personality lives)*
- [x] **Home** — pitch, CSV example above the fold (now actually runnable —
      it was missing `let`, in the README too), install one-liner, live
      version badge, links to Why?/Get Started
- [x] **Why?** — first-person manifesto draft at /why.html (early-2000s
      story, quirks & reasons). **Needs Sam's voice pass** — it's my best
      imitation, not the real thing.
- [x] **Get Started** — /get-started.html: install → REPL → first script →
      `basil --init` → hot reload → second page. Every command verified
      against the real binaries.
- [x] Site pages pipeline: `site/pages/*.md` render at the site root
- [x] **v1.0.0-alpha.3 released** (2026-07-12, authorised by Sam) — fixes the
      `basil --init` scaffold. Verified end-to-end from a fresh
      herbaceous.net install: init → dev server → serving.

### Phase 4 — Port and complete the docs

**Basil server manual — the biggest gap.** `docs/basil/manual/` has exactly
one page (`images.md`); `reference.md` is a Parsley-API reference, not a
server manual. Nothing about the server is on the site at all. Write a proper
manual (template exists from the 2026-03-17 commits), sourcing from
`docs/guide/*` and `reference.md`:
- [x] Running Basil — install, `basil --init`, `--dev`, CLI, signals
      (verified against `basil --help` and live runs)
- [x] Configuration — full `basil.yaml` reference (from config.go structs +
      configuration-example.yaml as ground truth)
- [x] Routing — site-mode walk-back (verified against server/site.go),
      folder-named handlers, subpaths, routes/static, asset bundling
- [x] Database — `@DB`, operators, inspector, other engines.
      ⚠️ guide/basil-quick-start.md says `basil.sqlite`; reference.md says
      `@DB` — went with `@DB` (reference is newer). Verify + fix quick-start.
- [x] Authentication — condensed from guide/authentication.md
- [x] Parts — condensed from guide/parts.md
- [x] Full-text search — condensed from guide/search.md
- [x] Git server & push-to-deploy — condensed from guide/git.md
- [x] Images — already existed ✓
- [x] Dev tools — live reload, error pages, dev log, `/__/db` inspector
- [x] Deployment — TLS/Let's Encrypt, headers, compression, caching, CORS
- [x] Basil manual index page + "Basil" in site nav (nav is now
      Get Started / Why? / Parsley / Basil / GitHub); reference.md also
      rendered at /basil/reference.html

**pars CLI & REPL — exists but weak/buried.** `features/cli.md` and
`features/repl.md` are on the site but hidden under the manual's Features
section:
- [x] Audit cli.md against the binary — fixed three fictions: `--machine`
      output shape (`{ok, type, value}` not `{result, type}`), exit codes
      (always 1 on error, not 2 for parse), and `PARS_NO_COLOR` (doesn't
      exist in the code — removed)
- [x] repl.md audited — matches the binary's `:help` exactly; no changes
- [~] pars visibility: Get Started's REPL section is the funnel for now;
      a dedicated nav entry can wait for real user feedback

**General:**
- [x] Parsley manual proofread against the implementation (2026-07-12).
      Method: `pars describe --json` diffed against documented methods per
      builtin page; every ```parsley block executed via harness
      (scratchpad run_blocks.py); flagged claims tested individually.
      Found & fixed: BUG-027 (`~` ignored the g flag — code fix + tests),
      `typeof()` → `.type()`, phantom `while` in the index, regex.md
      split/matchAll/named-group errors, record.md spread/json.encode,
      schema.md toTitleCase, numbers.md toNumber, data-formats.md &
      urls.md `markdown()`/`<==`-over-HTTP staleness, query-dsl.md
      pseudo-code fence. Remaining niggle: `pars describe builtins`
      in-code descriptions advertise `fileList(path, pattern?)` and
      `markdown(path)` file reads that don't match their own arity
      checks — registry metadata fix, tracked below.
- [ ] Fix stale in-code builtin descriptions (`describe builtins`):
      fileList arity, markdown() description
- [x] guide/README.md rewritten as a user-facing index of the actual guide
      files (was a stale dev-process doc with broken links pointing at
      docs/specs//docs/bugs, which moved to work/ long ago)
- [x] file-io.md fixed: `MD()` returns `{html, md, raw}` (not an HTML
      string), removed nonexistent `markdown(@path)` handle, `fileList()`
      one-arg + returns file handles + `**` root caveat documented
- [x] file-io.md `dir()` claim verified against the binary — `f.name` works;
      docs were correct
- [x] guide/basil-quick-start.md: `basil.sqlite` → `@DB` (six occurrences;
      confirmed in code that @DB is current and basil.sqlite is gone)
- [ ] FAQ; tone pass throughout
- [ ] Reconcile manual frontmatter `version: 0.2.0` vs actual release version

### Phase 5 — Polish & launch
- [x] Pico-docs-style three-column layout (2026-07-13): left sidebar = manual
      pages (extracted from each tree's index page **using the `~ /…/g`
      matching fixed in BUG-027**), right sidebar = current page's h2
      headings, active-page highlight, mobile Menu disclosure. Pico nav
      gotchas: negative link margins and horizontal ul flex needed overrides.
- [x] Basil logo SVG (from server/prelude/public/logos/) replaces the emoji
      in the nav brand and serves as the favicon
- [x] OpenGraph tags + canonical URL on every page
- [x] v1.0.0-alpha.4 released (2026-07-13): BUG-027 g-flag fix + manual
      corrections. Verified from a fresh install.
- [x] Layout refinements from Sam's wireframe (2026-07-13): whole docs grid
      capped & centred so the TOC hugs the content (like picocss.com/docs);
      Parsley/Basil switcher at the top of the docs sidebar; top nav is now
      About | Docs | GitHub icon | day/night toggle; About pages (Home /
      Get Started / Why?) share a small docs-style sidebar. Theme toggle
      persists to localStorage and the hljs palette follows data-theme.
      Found BUG-028 en route: the lexer mangles non-ASCII in <script>/<style>
      raw text (workaround: \u escapes; fix tracked in work/bugs/BUG-028.md).
- [x] Client-side search (2026-07-13): build.pars writes search-index.json
      (63 pages; title/headings/body, ~240KB fetched lazily on first focus);
      ~90 lines of vanilla JS score client-side (title 10 / headings 4 /
      body 1, all terms must match, cheap plural stemming) with a dropdown
      in the header. Hidden ≤768px for now — mobile search is a possible
      follow-up. No npm, no external service.
- [x] README rewritten as the public repo front door: install one-liner,
      verified examples, herbaceous.net links throughout, honest alpha note.
      The "don't post to HN" line stays (softened) until Sam calls the launch.
- [ ] Soft launch to friendly devs (Sam, thinking about it)
- [ ] Sam: voice pass on /why.html (in progress)
- [ ] At launch: delete the HN line from README

## Estimates

Phases 1–2: a few focused days. Phase 3: mostly writing (founder voice; drafts
can be AI-assisted). Phase 4: the long tail, parallelisable. Phase 5: small.

## Assessment Snapshot (2026-07-12)

Findings that motivated this plan:

- Full test suite passes across the tree (evaluator, parser, server, auth,
  search, images incl. smartcrop/seamcarve). 1,276 commits.
- Tagged `v1.0.0-alpha.1`; substantial `[Unreleased]` changelog section
  (FEAT-148/149 images, BUG-025 short-circuit ops, breaking config renames)
  awaiting a tag.
- One open bug: BUG-005 (confusing errors importing single-value exports).
- Parsley manual: ~71k words, complete and well-organised. Basil manual:
  one page (`images.md`) — template exists, buildout stalled 2026-03-17.
  `docs/guide/` has good topic guides but its README mixes in dev-process
  content and has broken links.
- No release automation existed (no goreleaser, no `.github/workflows`).
