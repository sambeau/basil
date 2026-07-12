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
- [ ] **Cut v1.0.0-alpha.3** — while verifying the tutorial we found
      `basil --init` in alpha.2 generates a basil.yaml the server rejects
      (`site:` string vs struct; stale `sqlite:` hint). Fixed on main
      (03dcb28) with a config.Load regression test, but the live tutorial's
      step 4 fails on alpha.2 binaries until a new release ships.

### Phase 4 — Port and complete the docs

**Basil server manual — the biggest gap.** `docs/basil/manual/` has exactly
one page (`images.md`); `reference.md` is a Parsley-API reference, not a
server manual. Nothing about the server is on the site at all. Write a proper
manual (template exists from the 2026-03-17 commits), sourcing from
`docs/guide/*` and `reference.md`:
- [ ] Running Basil — install, `basil --init`, `--dev`, `--config`, CLI flags
- [ ] Configuration — `basil.yaml` reference (adapt guide/configuration.md)
- [ ] Routing — file-based routes, site mode, handlers
- [ ] Database — built-in SQLite, `/__/db` inspector, bindings
- [ ] Authentication — sessions, users, roles, API keys, passkeys
      (adapt guide/authentication.md)
- [ ] Parts — reloadable components (adapt guide/parts.md)
- [ ] Full-text search (adapt guide/search.md)
- [ ] Git server & push-to-deploy (adapt guide/git.md)
- [ ] Images — exists (`images.md`) ✓
- [ ] Dev tools — error pages, request logging, hot reload
- [ ] Deployment — TLS, production mode, CORS (adapt guide/cors.md)
- [ ] Basil manual index page + **Basil section in the site nav**

**pars CLI & REPL — exists but weak/buried.** `features/cli.md` and
`features/repl.md` are on the site but hidden under the manual's Features
section:
- [ ] Audit both pages against the actual binary (`pars --help`,
      `pars describe`, `--check`, security flags) — the manual has already
      been caught documenting things the code doesn't do (see gotchas below)
- [ ] Beef up the REPL page: worked examples, `.describe`, output modes,
      the "try Parsley in 60 seconds" angle — the REPL is the funnel from
      the homepage's "Try it" section
- [ ] Give pars a visible home: "The pars command" nav entry or a
      prominent path from Get Started, not just a Features subpage

**General:**
- [ ] Parsley manual (48 pages; structure already good — rendering works,
      needs a proofread pass against implementation)
- [ ] Separate user docs from dev-process docs; fix broken links in
      `docs/guide/README.md` (`quick-start.md`, `cheatsheet.md`, `walkthroughs/`)
- [ ] Fix stale manual pages found while dogfooding: file-io.md documents
      `markdown(@path)`, 2-arg `fileList()`, and `dir()` handles yielding
      `.name` — none match the implementation
- [ ] FAQ; tone pass throughout
- [ ] Reconcile manual frontmatter `version: 0.2.0` vs actual release version

### Phase 5 — Polish & launch
- [ ] Client-side search (Pagefind, or Parsley-generated JSON index + small JS)
- [ ] OpenGraph tags, favicon
- [ ] Soft launch to friendly devs
- [ ] Update README (remove "don't post to HN"), launch on our own terms

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
