# Basil and Parsley Implementation Summary

## Overview

Basil and Parsley were developed as two closely related projects:

- **Parsley** began life in a separate repository as the programming language and runtime.
- **Basil** was built as the web server and application framework that executes Parsley scripts.
- The two projects were developed **in tandem** for a period, with Basil depending on the external Parsley repository.
- Once the language and server had become tightly interwoven, Parsley was **merged into the Basil monorepo** so they could evolve together.

This document summarizes the implementation history reflected in the planning documents under `work/plans`, with emphasis on chronology, major feature themes, and how the language and framework grew alongside one another.

---

## High-Level Development Story

The implementation history breaks naturally into a few broad phases:

1. **Foundation and workflow setup**
   - Development process, documentation templates, and project structure were established.
   - Basil was created as a Go web server capable of serving Parsley-driven responses.

2. **Early Basil server capabilities**
   - Core server functionality landed: config loading, HTTPS/dev mode, static file serving, request handling, and Parsley execution.
   - Developer ergonomics improved with error pages, live reload, logging, and dev tooling.

3. **Tandem Basil/Parsley evolution**
   - While still separate repos, Parsley gained language features and standard library modules specifically useful for web work.
   - Basil added framework features that depended on Parsley’s runtime model and embedding API.

4. **Monorepo merge**
   - Parsley was imported into Basil under `pkg/parsley/` and `cmd/pars/`.
   - Basil and Parsley then evolved as a single codebase with shared docs, tests, tooling, and release flow.

5. **Expansion toward a full language + framework platform**
   - Parsley grew substantially: syntax, types, standard library, formatting, introspection, PLN, units, error model, and tooling.
   - Basil expanded into auth, devtools, database tooling, git integration, caching, API support, and production/server concerns.

6. **Pre-1.0 consolidation**
   - Later plans focus heavily on consistency, documentation accuracy, toolability, formatting, error quality, and language cleanup.
   - The changelog shows a major Parsley 1.0 alpha milestone with removals, renames, and a more coherent final model.

---

## Chronological Timeline

## Phase 0 — Initial setup and process foundation
**Late 2025**

### FEAT-001 — Development Process Framework
This was the project scaffolding phase. The repository gained:

- `AGENTS.md`
- workflow prompts
- templates for specs, bugs, and plans
- backlog and changelog tracking
- human/AI collaboration process documentation

This matters because the rest of the implementation history is unusually well-documented: many later features have explicit plans, phases, and validation notes.

---

## Phase 1 — Basil MVP established
**2025-11-30**

### FEAT-002 — Basil Web Server (Phase 1 MVP)
This is the real starting point for Basil as a product.

Core capabilities implemented:

- Go server entry point and CLI
- config loading and validation
- HTTP server with graceful shutdown
- Parsley script execution in request handlers
- script caching
- static file serving
- development mode
- basic logging
- example app and quick-start docs

In effect, this delivered the first usable version of Basil: a web server that could serve dynamic responses produced by Parsley.

### What this says about the architecture
At this stage, Parsley was still external, but Basil was already being designed around Parsley as its application language. Basil was never just a generic web server with optional scripting; it was conceived as a Parsley-native runtime.

---

## Phase 2 — Early developer experience and repo convergence
**2025-12-01 to 2025-12-02**

### FEAT-006 — Dev Mode Error Display
Basil quickly gained a stronger developer experience:

- browser-rendered Parsley errors
- syntax highlighting
- source context extraction
- dev-friendly error handling paths

This shows that developer tooling was considered core, not an afterthought.

### FEAT-007 — Parsley Monorepo Merge
This is the pivotal structural event in the history.

Key changes:

- Parsley source copied into `pkg/parsley/`
- Parsley CLI added under `cmd/pars/`
- Basil CLI moved to `cmd/basil/`
- all imports updated from the external Parsley repo
- external dependency removed from `go.mod`
- both binaries built and tested inside one repository

### Significance of the merge
Before this point:

- Parsley and Basil were separate repos
- Basil depended on Parsley externally
- language and framework were advancing together, but with repo boundaries

After this point:

- Basil became the monorepo for both the server and the language
- the `pars` CLI and Basil server shared the same release and test surface
- it became easier to evolve language features specifically needed by the framework
- docs, examples, formatter, tooling, and tests could all stay aligned

This confirms the user’s note: **Parsley started separately, then both projects were developed in tandem until merging became prudent.**

### FEAT-008 — Array randomization methods
Soon after the merge, Parsley resumed feature growth inside the monorepo with additive language features like:

- `shuffle()`
- `pick()`
- `take()`

### FEAT-011 — Basil namespace in Parsley environment
Basil’s injected runtime values were reorganized into a proper namespace:

- `basil.http`
- `basil.auth`
- `basil.sqlite`

This is a major integration refinement. It formalized the boundary between language runtime and framework runtime and gave Basil-specific capabilities a structured home inside Parsley code.

### FEAT-014 / FEAT-015 — Optional access ergonomics
Plans for:

- optional indexing `[?n]`
- optional chaining `?.`

reflect a push toward safer, more expressive language ergonomics for real application code.

### FEAT-016 — Per-route `public_dir`
Basil routing and static asset handling became more modular, moving toward route-isolated behavior instead of one global public directory.

---

## Phase 3 — Parsley grows into a richer application language
**Early December 2025**

This phase is dominated by language growth, especially features useful in real apps and data work.

### FEAT-018 — `@std/table`
A major Parsley capability:

- standard library table module
- SQL-like operations on arrays of dictionaries
- filtering, sorting, selecting, limiting, aggregation, HTML rendering

This indicates Parsley was being shaped not just as a templating language, but as a data-oriented scripting language for applications.

### FEAT-020 — Per-developer config overrides
A Basil workflow feature that supported multi-developer local configuration. This reflects the framework maturing for team use.

### FEAT-021 — SQLite dev tools
Basil devtools expanded with database inspection and CSV import/export concepts via `/__/db`.

### FEAT-022 — Block concatenation investigation
An experimental Parsley design investigation into making blocks concatenate expression results. The very existence of this plan is useful historically: the language model was still being actively questioned and refined.

### FEAT-024 — `print()` / `println()` implementation
At this point, Parsley briefly explored explicit print-style output semantics. Later changelog entries show these were eventually removed in favor of expression-based output, which is a good example of the language converging over time toward a cleaner model.

### FEAT-025 — Money type
Parsley gained a proper money/currency type with:

- currency literals
- exact arithmetic
- formatting
- comparisons
- splitting

This is a hallmark of a language moving beyond toy scripting toward application-grade domain types.

### FEAT-027 — Collection insert methods
Arrays, dictionaries, and tables all received richer mutation-like immutable operations.

### FEAT-029 — `try` expression
Parsley got structured, selective error catching with explicit distinction between catchable user errors and programming errors.

### FEAT-031 — `@std/math`
A substantial math standard library module was added.

### FEAT-032 — `@std/valid`
Validation helpers were added for forms and general app input validation.

### FEAT-033 — String sanitizer methods
Methods like whitespace normalization, digit extraction, slug creation, and HTML stripping show an increasingly web-focused standard library.

### FEAT-034 — `@std/api`
This introduced schema validation, ID generation, auth wrappers, and API-oriented helpers. Even where phases were deferred, the direction is clear: Parsley was increasingly being shaped for Basil-backed app development.

---

## Phase 4 — Basil as a fuller web application framework
**December 2025 onward**

This phase is where Basil branches out from “serve Parsley handlers” into a broader framework.

### FEAT-035 — Git over HTTPS
Basil aimed to let developers clone/push site repos directly over HTTPS, with auth integration and live reload implications.

### FEAT-036 / FEAT-035 combined direction
Although not all FEAT-036 details were read directly here, related plans and reports show Basil adding user/API key management and permission-aware infrastructure.

### FEAT-037 — Fragment caching
A plan for `<basil.cache.Cache>` indicates Basil was moving toward component-level performance primitives, not just page-level response caching.

### FEAT-038 onward
The plan list suggests a steady stream of Basil framework features in this era, including:

- caching
- routing refinement
- component loading
- metadata handling
- query DSL work
- auth and security improvements
- environment/secrets consistency
- database evolution
- production behavior cleanup

Even without reading every plan in full, the structure of the plan set shows Basil becoming a serious full-stack framework around the Parsley runtime.

---

## Phase 5 — Language, tooling, and docs acceleration
**Late 2025 into early 2026**

As the project matured, the focus broadened dramatically from core execution to ecosystem quality.

### Parsley tooling and language infrastructure
A large cluster of plans points to this acceleration:

- formatter / pretty-printer
- manual/reference generation
- introspection APIs
- declarative method registry
- help system
- parser and grammar refinements
- tree-sitter grammar and editor tooling
- lint cleanup
- documentation overhaul
- unified error model
- metadata and environment consistency

### Notable feature families in this stage

#### 1. Formatter and source tooling
Multiple plans cover pretty-printing details such as:

- control flow formatting
- method chain formatting
- tag formatting
- query DSL formatting
- table formatting
- comment preservation
- multiline attributes
- CLI formatting support

This culminates in a real AST-based formatter, which is a major milestone for language maturity.

#### 2. Introspection and self-description
Plans around method registries and `pars describe` show Parsley evolving into a self-describing language/toolchain, reducing documentation drift.

#### 3. Error model unification
Later plans emphasize structured errors, catalogued codes, and consistent formatting. This aligns with the changelog’s mention of a unified error model and helps both CLI and Basil server UX.

#### 4. Developer docs and verification
There is strong evidence of a documentation-quality push:
- manual planning
- doc verification reports
- audit reports
- reference generation plans
- consistency cleanups

This is typical of projects approaching a major public milestone.

---

## Phase 6 — Major Parsley expansion
**2026**

By the time the project approaches 1.0 alpha, Parsley has become much more than the small language Basil first embedded.

### Major capabilities visible in plans, backlog, and changelog

#### Language syntax and semantics
- optional access ergonomics
- `try`
- `with` expression
- Unicode identifiers
- SQL tag raw text support
- improved structured errors
- more coherent import/module behavior
- eventual `let` / `var` shift

#### Rich built-in and standard-library types
- money
- tables
- validation
- math
- IDs
- schemas
- data serialization
- units and measurements

#### Data and serialization
- PLN support
- reference and formatter tooling
- SQL/query DSL direction
- remote I/O operators

#### Tooling
- `pars -e`
- `pars --check`
- `pars describe`
- `pars reference`
- formatter
- REPL output improvements
- tree-sitter grammar
- editor integrations

This is the clearest sign that Parsley had grown from “Basil’s scripting language” into a standalone language product, even while remaining embedded in the Basil repo.

---

## Phase 7 — FEAT-118 and the units/measurement era
**2026-02**

The backlog and changelog make FEAT-118 especially prominent.

### FEAT-118 — Measurement units
Implemented in phases, including:

- temperature units
- volume units
- area units
- derived unit arithmetic
- compound display formatting
- decimal scale infrastructure

This appears to have been one of the late major language feature pushes before 1.0 alpha.

Why it matters:

- it required parser, evaluator, formatting, and type-system work
- it added serious domain expressiveness
- it reflects confidence in the language’s core architecture by this point

---

## Phase 8 — Pre-1.0 consolidation and alpha readiness
**2026-02 to 2026-03**

The `CHANGELOG.md` and report names show the project entering a stabilization phase.

### Parsley 1.0 Alpha themes
The changelog for `1.0.0-alpha.1` highlights:

- removal of experimental or redundant APIs
- migration toward expression-based output
- replacement of inconsistent global helpers with method-based APIs
- stronger type and error consistency
- unified formatting model
- introspection and help systems
- more polished docs and editor tooling

### Important cleanup signals
The alpha notes include several breaking changes that show consolidation:

- `print`/`println` removed
- global formatting helpers removed
- table module replaced by `@table` literal syntax
- uppercase form component variants removed
- deprecation scaffolding removed
- `let`/`var` semantics formalized

These are typical “we now know what the language should be” decisions.

### Basil’s concurrent maturation
At the same time, Basil appears to be gaining:

- better devtools
- auth and route protection
- database integration improvements
- performance benchmarking
- git workflows
- configuration consistency
- search and benchmarking
- production polish

So while Parsley is heading toward language alpha, Basil is heading toward framework robustness.

---

## Major Feature Themes

## 1. Basil became a Parsley-native web framework
From the beginning, Basil’s main value proposition was:

- execute Parsley in HTTP handlers
- expose request/response/auth/db features to Parsley
- provide a framework-shaped runtime for Parsley applications

This deepened after the monorepo merge and the `basil.*` namespace work.

## 2. Parsley evolved from embedded DSL to full language
The plans show a clear trajectory:

- early runtime embedding
- then richer syntax and convenience features
- then domain types and stdlib modules
- then formatting, introspection, docs, and editor support
- finally 1.0 alpha cleanup and consolidation

## 3. Basil and Parsley influenced each other continuously
Examples:

- Basil needed better language ergonomics for handlers, forms, APIs, and templates.
- Parsley gained validation, table, schema, ID, money, and sanitization features that are especially useful in web development.
- Basil runtime objects were reorganized to fit naturally into Parsley code (`basil.http`, `basil.auth`, `basil.sqlite`).
- Devtools, error pages, and formatting all depended on close alignment between framework and language internals.

## 4. The monorepo was a turning point
The merge removed friction between:
- runtime development
- CLI tooling
- framework integration
- docs
- tests
- releases

After the merge, the pace and breadth of feature work suggest much tighter iteration.

## 5. Late development prioritized coherence over accumulation
The later plans and alpha changelog show a maturing discipline:
- fewer ad hoc APIs
- more method-based consistency
- verified documentation
- introspection as source of truth
- stronger error taxonomy
- cleanup of legacy or transitional features

---

## Condensed Timeline by Theme

## Foundation
- **FEAT-001** — Development process and project workflow infrastructure

## Basil core server
- **FEAT-002** — Basil MVP: config, server, Parsley handlers, static files, dev mode
- **FEAT-006** — Dev-mode error pages

## Repository convergence
- **FEAT-007** — Parsley merged from separate repo into Basil monorepo
- **FEAT-011** — `basil.*` namespace for framework runtime values

## Early Parsley growth in monorepo
- **FEAT-008** — Array random methods
- **FEAT-014** — Optional indexing
- **FEAT-015** — Optional chaining
- **FEAT-018** — Table module
- **FEAT-024** — Temporary print/println phase
- **FEAT-025** — Money type
- **FEAT-027** — Collection insertion methods
- **FEAT-029** — `try`
- **FEAT-031** — `@std/math`
- **FEAT-032** — `@std/valid`
- **FEAT-033** — String sanitizers
- **FEAT-034** — API/schema/ID/auth support

## Basil framework broadening
- **FEAT-016** — Per-route static directories
- **FEAT-020** — Per-developer config
- **FEAT-021** — SQLite devtools
- **FEAT-035** — Git over HTTPS
- **FEAT-037** — Fragment caching
- plus many later plans around auth, devtools, env/secrets, query DSL, and consistency

## Language/tooling maturity
- formatter work
- help and introspection systems
- declarative method registry
- tree-sitter grammar
- manual/reference generation
- unified error model
- doc verification and cleanup

## Pre-1.0 consolidation
- **FEAT-118** — Units and measurements
- **FEAT-122** and related cleanup — `let`/`var`, output model cleanup, API consolidation
- **1.0.0-alpha.1** — Major alpha milestone for Parsley

---

## Key Milestones

### Milestone 1: Basil becomes usable
The Basil MVP made it possible to run Parsley-backed web apps.

### Milestone 2: Developer ergonomics become first-class
Error pages, live reload, logging, and devtools made Basil feel like a development environment, not just a runtime.

### Milestone 3: Parsley and Basil merge
This is the biggest structural milestone in the whole timeline.

### Milestone 4: Parsley becomes application-grade
Money, tables, validation, math, schemas, IDs, formatting, and structured errors moved Parsley well beyond minimal scripting.

### Milestone 5: Tooling catches up with language scope
Formatter, introspection, `pars describe`, reference generation, tree-sitter, and doc audits signal ecosystem maturity.

### Milestone 6: Coherence and alpha readiness
Breaking cleanups and documentation verification show a shift from experimentation to stabilization.

---

## Overall Assessment

Based on the implementation plans, Basil and Parsley did not evolve as isolated products. They developed as a **paired framework/language system**:

- **Basil** supplied the web runtime, server, and developer tooling.
- **Parsley** supplied the language, expression model, templating, data manipulation, and app-level abstractions.
- Their feature sets repeatedly pulled on each other.
- The eventual monorepo merge was not just a repo-management convenience; it was the natural outcome of a deeply coupled design.

In practical terms, the implementation history shows this progression:

1. build a server,
2. embed a language,
3. improve the developer experience,
4. merge the codebases,
5. expand the language and framework together,
6. then refine everything toward a coherent platform.

That makes the Basil/Parsley story less “framework plus scripting language” and more **a unified full-stack environment whose web server and language were co-designed over time**.

---

## Suggested one-sentence summary

**Parsley began as a separate language project, Basil began as the web server that hosted it, and over time the two evolved in tandem into a single monorepo delivering a tightly integrated language-and-framework platform.**