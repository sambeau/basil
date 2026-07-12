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
- [ ] Test install.sh end-to-end on a clean machine (needs a published release)
- [x] Housekeeping: `.kbz/` and `testdata/images/smartcrop/` (855MB) gitignored
- [ ] Human: commit, rename `[Unreleased]` → `[1.0.0-alpha.2]`, tag, push tag
      (pushing the tag publishes the first automated release)

### Phase 2 — Site skeleton
- [ ] `site/` directory: `build.pars`, layout components, vendored Pico CSS
- [ ] GitHub Actions: build `pars` → run `build.pars` → deploy to Pages
- [ ] Wire in highlight.js + Parsley grammar for code blocks
- [ ] DNS: Hover → Pages, HTTPS enabled

### Phase 3 — Story pages *(where the personality lives)*
- [ ] **Home** — one-screen pitch, runnable code example above the fold
      (the CSV → table → HTML example), "no compiler, no npm, no build step",
      download button with live version badge
- [ ] **Why?** — the manifesto: early-2000s story, why the quirks exist
- [ ] **Get Started** — install → hello world → tiny real app in 10 minutes
      (adapt `docs/guide/basil-quick-start.md` + `docs/parsley/manual/getting-started.md`)

### Phase 4 — Port the docs
- [ ] Parsley manual (91 pages; structure already good — mostly mechanical)
- [ ] Basil manual buildout — port guide pages (auth, search, parts, git
      deploy, config, styling) + carve up `docs/basil/reference.md`;
      this closes the biggest docs gap
- [ ] Separate user docs from dev-process docs; fix broken links in
      `docs/guide/README.md` (`quick-start.md`, `cheatsheet.md`, `walkthroughs/`)
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
