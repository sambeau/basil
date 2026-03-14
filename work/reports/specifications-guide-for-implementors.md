# Specifications Guide for Implementors
<!-- Intended location: basil/reports/specifications-guide-for-implementors.md -->

## Purpose

This document is an implementor-facing guide to the Basil and Parsley specification set under `work/specs/`.

It is written primarily for:

- AI agents making code or spec changes
- maintainers reviewing proposed changes
- contributors trying to understand **why** the system looks the way it does

This guide complements:

- the codebase's **how**,
- the specs' **what**,
- and the plans/design docs/reports' **why**.

In other words:

- **Code** tells you how something currently works.
- **Specs** tell you what was intended or promised.
- **Plans** tell you how implementation was approached.
- **Design docs / ADRs / reports** tell you why certain tradeoffs were chosen.

Use this file as an **index**, **encyclopedia**, and **change-safety manual** before modifying behavior.

---

# Quick Rules for Implementors

## 1. Do not change behavior from code alone
If you find behavior in code, first ask:

- Is there a matching `FEAT-XXX` spec?
- Is there a matching `PLAN-XXX` implementation plan?
- Is there a design doc in `work/design/`?
- Is there a report in `work/reports/` explaining tradeoffs, audits, or deferred scope?

If yes, read those first.

## 2. Specs are not equal
Some specs are:

- foundational and normative
- partially implemented
- historical
- superseded by later specs
- cleanup/removal specs for pre-1.0 consolidation

Treat each spec as a record in a living history, not as automatically current truth.

## 3. Later specs often revise earlier ones
Important pattern in this repository:

- early specs establish capability
- later specs regularize, unify, or remove earlier APIs
- pre-1.0 cleanup specs intentionally break old assumptions

If two specs conflict, prefer:

1. implemented later spec
2. associated plan and code
3. changelog / readiness / audit reports
4. design docs referenced by the newer spec

## 4. Plans often contain the implementation reality
In this codebase, plans are not just checklists. They often contain:

- exact file targets
- implementation notes
- deviations from the original spec
- phased rollout details
- "done vs deferred" distinctions

When a spec says "should", and the plan says "implemented differently because...", the plan may reflect the true implemented decision.

## 5. Reports often explain the "why" better than specs
For late-cycle work especially, the reasoning often lives in:

- audits
- gap analyses
- performance investigations
- doc verification logs
- readiness reviews

If you need rationale, search `work/reports/` after reading the spec.

---

# Recommended Reading Order

When touching a feature, read in this order:

1. `work/specs/FEAT-XXX.md`
2. matching `work/plans/...`
3. any `design:` path named in spec frontmatter
4. related docs in `work/design/`
5. related audit/report in `work/reports/`
6. actual implementation in code
7. user-facing docs in `docs/guide/` or `docs/parsley/`

This sequence is important because:

- specs explain intent
- plans explain implementation shape
- design docs explain tradeoffs
- code explains current behavior
- user docs explain public contract

---

# How the Specs Are Organized

The specs cover three broad domains:

## A. Basil framework / server specs
These define the Go web framework/runtime around Parsley.

Common topics:

- server boot/config
- routes and handlers
- auth
- devtools
- API behavior
- caching
- git integration
- deployment behavior
- database integration

Typical code areas:
- `cmd/basil/`
- `server/`
- `auth/`
- `config/`

## B. Parsley language specs
These define language syntax, semantics, stdlib, CLI, output behavior, formatting, and tooling.

Typical code areas:
- `cmd/pars/`
- `pkg/parsley/lexer/`
- `pkg/parsley/parser/`
- `pkg/parsley/ast/`
- `pkg/parsley/evaluator/`
- `pkg/parsley/repl/`
- `pkg/parsley/help/`

## C. Meta / process / repository specs
These define workflow, documentation, release shape, and repository structure.

Typical code areas:
- root docs and process files
- `.github/`
- `work/`

---

# Where to Find "Why"

## Primary "why" sources by document type

### Specs
Best for:
- feature intent
- acceptance criteria
- user story
- top-level design decisions

Weak at:
- final implementation detail
- later deviations
- follow-up cleanup

### Plans
Best for:
- implementation phases
- file-by-file execution plan
- what actually got done
- deviations from spec
- deferred scope

### Design docs (`work/design/`)
Best for:
- tradeoff analysis
- alternative approaches considered
- grammar choices
- naming / API philosophy
- architecture decisions before implementation

### ADRs
Best for:
- concise decision record
- "we considered X but chose Y"

### Reports (`work/reports/`)
Best for:
- audits
- gap analysis
- readiness rationale
- cleanup reasoning
- verification evidence
- performance evidence

### CHANGELOG
Best for:
- understanding which later changes are intentionally breaking
- seeing which old features were removed or renamed

---

# High-Value Specs Every Implementor Should Know

These are the specs most likely to affect broad architecture or future changes.

## FEAT-001 — Development Process Framework
**What it is:** repository workflow and AI/human collaboration scaffold.

**Why it matters:**  
This explains why the repository is organized around `specs/`, `plans/`, `bugs/`, `design/`, and `reports/`. It also explains the "newspaper article" pattern used in docs: human summary first, dense implementor detail below the divider.

**Read when:**
- creating new specs/plans
- wondering why the workflow is so document-heavy
- deciding where rationale should live

**Also read:**
- `AGENTS.md`
- `.github/copilot-instructions.md`
- `work/README.md`
- `work/plans/FEAT-001-plan.md`

**Implementation significance:**  
This spec is foundational. It defines how future design reasoning is meant to be captured.

---

## FEAT-002 — Basil Web Server
**What it is:** the foundational Basil server spec.

**Why it matters:**  
This is the original statement of Basil's identity: a focused Go server for Parsley. It sets the framework philosophy:

- HTTPS-first in production
- Parsley-native request handling
- Basil manages resources; Parsley scripts consume them
- auth and infrastructure are server concerns
- response generation is script-driven

**Important design ideas:**
- Basil is not a generic host with a little scripting
- Parsley is central to the request model
- explicit route configuration matters
- server-managed resources protect concurrency and lifecycle

**Read when:**
- changing request/response behavior
- changing handler execution
- changing route config or server startup
- deciding whether something belongs in Basil or Parsley

**Also read:**
- `work/plans/FEAT-002-plan.md`
- `server/`
- `config/`
- `docs/guide/basil-quick-start.md`

**Change hazard:**  
If you modify request injection, response dictionary semantics, auth handoff, or caching strategy, you are likely modifying FEAT-002's contract.

---

## FEAT-004 — Authentication
**What it is:** Basil auth system spec.

**Why it matters:**  
This spec captures one of Basil's strongest opinionated choices:

- passkeys first
- no password-first legacy design
- auth as Parsley components, not Basil-owned pages
- strict air-gap between Parsley and auth database

The most important reasoning here is not "how WebAuthn works" but **where auth authority lives**.

**Core architectural philosophy:**
- Parsley renders/consumes auth UI
- Basil owns auth truth and session state
- Parsley can read identity, not manage credentials directly

**Read when:**
- touching `request.user`
- modifying auth wrappers/components
- changing auth DB exposure
- considering role resolution behavior
- changing auth-related API semantics

**Also read:**
- `work/plans/FEAT-004-plan.md`
- `auth/`
- `docs/guide/authentication.md`
- related API auth specs such as FEAT-034

**Change hazard:**  
Do not let convenience blur the boundary between app code and auth authority without explicit design work.

---

## FEAT-007 — Merge Parsley into Basil Monorepo
**What it is:** historical monorepo merge spec.

**Why it matters:**  
This explains why the repo is shaped the way it is:

- `cmd/basil/`
- `cmd/pars/`
- `pkg/parsley/`
- one module
- unified issue tracking
- one release surface

It also explains the history: Parsley started separately, then merged when the coupling became strong enough to justify a monorepo.

**Read when:**
- reasoning about repo layout
- deciding whether something belongs in Basil or Parsley subtrees
- wondering why import paths and release flow look unified

**Also read:**
- `work/plans/FEAT-007-plan.md`

**Change hazard:**  
Avoid reintroducing conceptual split-brain between Basil and Parsley unless there is a compelling architectural reason.

---

## FEAT-023 — Structured Error Objects
**What it is:** unified structured error model.

**Why it matters:**  
This is one of the most important cross-cutting specs in the repository. It explains the move away from text-only errors toward structured objects with:

- class
- code
- message
- hints
- line/column/file
- data for templating/rendering

This spec's value is mostly philosophical:

- errors should be machine-readable and human-usable
- UI should not regex-parse strings to recover structure
- hints matter
- naming consistency matters
- location matters

**Read when:**
- adding or changing evaluator errors
- parser error changes
- Basil dev error page changes
- CLI error formatting changes
- implementing new structured error codes

**Also read:**
- `work/plans/PLAN-080-FEAT-105-unified-error-model.md`
- `work/design/unified-error-model.md`
- `pkg/parsley/errors/`
- `server/errors.go`
- later specs FEAT-105 and FEAT-125

**Change hazard:**  
Do not introduce new one-off error formats when the structured model applies.

---

## FEAT-034 — `std/api` / API platform direction
**What it is:** Basil/Parsley API platform spec.

**Why it matters:**  
This captures the project's opinionated API philosophy:

- schema-first
- auth-on-by-default
- secure-by-default wrappers
- helper-heavy for CRUD and validation
- JSON API support inside the same platform as HTML

It also reveals a pattern common in this repo: some named capabilities were implemented under slightly different module shapes (`std/schema`, `std/id`, `std/api` pieces), and some scope was deferred.

**Read when:**
- changing API routing
- changing auth wrappers like `public`, `roles`, `adminOnly`
- changing schema validation for APIs
- altering rate limit/pagination defaults
- touching `server/api.go`

**Also read:**
- `work/plans/FEAT-034-plan.md`
- `work/design/api-design-summary.md`
- `work/design/FEAT-034-phases-3-6-design.md`
- `work/design/std-api-discussion.md`
- `server/api.go`
- `pkg/parsley/evaluator/stdlib_api.go`
- `pkg/parsley/evaluator/stdlib_schema*.go`

**Change hazard:**  
This area mixes framework policy with language-facing ergonomics. Small "cleanup" changes can accidentally weaken secure defaults.

---

## FEAT-100 — Parsley Pretty-Printer
**What it is:** formatter / pretty-printer spec.

**Why it matters:**  
This is more than formatting style. It encodes language maturity decisions:

- canonical style
- parseable output
- REPL usability
- copy-pasteable examples
- `pars fmt` direction

Formatting decisions often become de facto language documentation.

**Read when:**
- changing AST formatting
- changing REPL repr
- changing docs examples generation
- changing `pars fmt`

**Also read:**
- `work/plans/PLAN-072-pretty-printer.md`
- `work/design/DESIGN-PRETTY-PRINTER.md`
- `work/design/FORMATTER_DESIGN.md`
- `pkg/parsley/format/`

**Change hazard:**  
Formatting is a user-facing contract for examples, docs, and tooling; not just cosmetic internals.

---

## FEAT-111 — Declarative Method Registry
**What it is:** method dispatch and introspection source-of-truth refactor.

**Why it matters:**  
This spec explains the move from ad hoc switch-based method dispatch toward registries that power both execution and introspection.

This is crucial for understanding why newer docs/tooling depend on registries.

**Core idea:**
- one method definition should power:
  - dispatch
  - describe/help output
  - future docs generation

**Read when:**
- adding methods to types
- fixing introspection drift
- updating help system
- deciding where method metadata belongs

**Also read:**
- `work/plans/PLAN-117-method-registry-migration.md`
- `pkg/parsley/evaluator/method_registry.go`
- `pkg/parsley/evaluator/introspect.go`

**Change hazard:**  
Do not add methods in ways that bypass the registry unless you also accept breaking help/introspection consistency.

---

## FEAT-112 — Unified Help System
**What it is:** `pars describe` / REPL help system.

**Why it matters:**  
This spec formalizes a very important repository principle:

> Help should come from implementation metadata, not hand-maintained parallel docs.

It is especially important for AI agents because it creates a reliable introspection entry point.

**Read when:**
- changing `pars describe`
- touching help output
- modifying introspection schemas
- changing type/module/builtin discovery surfaces

**Also read:**
- `work/plans/PLAN-088-FEAT-111.md`
- `work/plans/PLAN-089-FEAT-112.md`
- `pkg/parsley/help/`
- `cmd/pars/main.go`
- `pkg/parsley/repl/`

**Change hazard:**  
If you break metadata quality, you break help quality, and then you break AI usability.

---

## FEAT-113 — CLI `-e` outputs PLN by default
**What it is:** CLI behavior correction spec.

**Why it matters:**  
This spec is a good example of the project's philosophy around debugging and developer ergonomics:

- exploration should show structure
- REPL and CLI should align
- raw rendering should be opt-in

It explains **why** `pars -e` is not just "execute this text".

**Read when:**
- changing CLI execution modes
- changing REPL/CLI output parity
- touching PLN vs raw output paths

**Also read:**
- `cmd/pars/main.go`
- `work/plans/PLAN-086.md`
- docs around PLN / CLI

---

## FEAT-118 — Measurement Units
**What it is:** built-in measurement/unit system for Parsley.

**Why it matters:**  
This is a major language-design spec with unusually deep mathematical/representation reasoning. If you touch units, you must read beyond the spec.

Key importance:
- exact arithmetic
- no floating-point drift
- multiple internal representation families
- sigil/literal syntax decisions
- user-facing error philosophy
- explicit explanation of why some intuitive operations are disallowed

This spec is especially valuable because it documents *rationale*, not just API.

**Critical referenced reasoning:**
- `work/design/DESIGN-units-v3.md`
- earlier unit design docs for evolution of decisions

**Read when:**
- touching lexer handling of `#`
- changing unit arithmetic or conversions
- adjusting display/repr/format semantics
- extending unit families
- changing temperature behavior
- changing overflow or rounding semantics

**Also read:**
- `work/plans/FEAT-118-plan.md`
- `work/plans/FEAT-118-phase2-plan.md`
- `work/plans/PLAN-097-FEAT-118-phase3.md`
- `work/plans/PLAN-099-FEAT-118-phase4.md`
- backlog items tied to FEAT-118

**Change hazard:**  
Do not "simplify" unit logic without preserving the exactness and representation rationale. This is a mathematically opinionated subsystem.

---

## FEAT-122 — Swift-style `let` / `var`
**What it is:** variable binding semantics overhaul.

**Why it matters:**  
This spec explains a major breaking language decision: immutable `let`, mutable `var`, and no implicit declaration.

The key reasoning is philosophical and timing-sensitive:

- optional `let` felt unfinished
- pre-1.0 is the right time to break
- immutability should be explicit and meaningful
- Swift-style semantics were considered semantically cleaner than JS-style `let`

**Read when:**
- touching assignment parsing/evaluation
- changing binding mutability rules
- changing exports and destructuring semantics
- changing REPL or migration behavior
- changing keyword/reserved-word behavior

**Also read:**
- `work/reports/LET-CONST-ANALYSIS.md`
- `work/plans/FEAT-122-plan.md`
- `pkg/parsley/parser/`
- `pkg/parsley/evaluator/`

**Change hazard:**  
This is not a syntax-only feature. It is a semantic and pedagogical choice that shapes the language's identity.

---

## FEAT-123 — `with` expression
**What it is:** scoped field-access expression.

**Why it matters:**  
This is a smaller spec, but a good example of how Parsley tends to choose:

- explicit scope
- simple one-purpose syntax
- predictable shadowing
- no hidden rebinding magic

The associated design doc is important because it records the tradeoffs behind the chosen shape.

**Read when:**
- changing `with` semantics
- touching scoping/shadowing behavior
- modifying parser control-flow expression handling

**Also read:**
- `work/design/DESIGN-with-expression.md`
- `work/plans/PLAN-103-with-expression.md`

---

## FEAT-128 — Remove Deprecated Parsley Features for 1.0
**What it is:** pre-1.0 cleanup/removal spec.

**Why it matters:**  
This spec explains why certain things were intentionally removed, hard-errored, or cleaned up before 1.0.

This is essential for avoiding accidental resurrection of deprecated features.

Key philosophy:
- break old paths now, not later
- use hard errors with migration hints
- remove deprecation machinery once removals are complete
- simplify the codebase before 1.0

**Read when:**
- considering re-adding `@std/table`
- handling removed uppercase components
- seeing references to old migration tooling
- deciding whether a compatibility layer should exist

**Also read:**
- `work/plans/PLAN-107.md`
- `work/reports/1.0-READINESS-AUDIT.md`
- `CHANGELOG.md`

**Change hazard:**  
If you restore a removed convenience path "just to help users", you may be undoing deliberate 1.0 surface cleanup.

---

## FEAT-131 — Parsley Documentation Overhaul
**What it is:** docs-trustworthiness and generated-reference spec.

**Why it matters:**  
This may be the most important spec for AI agents specifically.

It states a central project truth:

> documentation is not authoritative unless verified against code

It also formalizes:
- `pars describe` as trusted source
- code-generated reference as anti-drift mechanism
- example verification via running code
- removal of hallucinated or stale docs

**Read when:**
- editing docs
- generating reference material
- using docs to justify a behavior change
- deciding whether to trust a manual page or the code

**Also read:**
- `work/reports/PARSLEY-DOC-AUDIT.md`
- `work/reports/DOC-VERIFICATION-LOG.md`
- `work/plans/PLAN-110-doc-overhaul.md`
- `docs/parsley/`
- `pkg/parsley/help/`

**Change hazard:**  
Never update code to match stale docs without verifying the intended truth source first.

---

## FEAT-140 — Remove API handler module-cache clearing
**What it is:** targeted cleanup/performance/correctness spec.

**Why it matters:**  
This spec is a very good example of repository reasoning style in mature phases:

- identify suspicious behavior
- investigate with tests and reports
- prove why it is unnecessary
- remove it only after confirming architectural safety

The important "why" here is the explanation of `DynamicAccessor` and environment propagation.

**Read when:**
- touching module cache behavior
- changing request-scoped API state
- changing environment propagation for API handlers
- touching `server/api.go`

**Also read:**
- `work/reports/BASIL-PERFORMANCE-ANALYSIS-2026-03-10.md`
- related tests named in the spec
- relevant evaluator environment code

**Change hazard:**  
This area is subtle. If you change caching or request-scoped module behavior, confirm you still preserve freshness guarantees.

---

# Specification Families and How to Use Them

Below is a practical map of the spec collection by theme.

---

# 1. Repository / Process / Meta Specs

## Core specs
- `FEAT-001` — workflow/process framework
- `FEAT-005` — semantic versioning
- `FEAT-007` — monorepo merge

## Why these exist
These specs explain:

- how the project is operated
- how releases are conceptualized
- why Basil and Parsley live together
- why process artifacts are first-class

## Read these before
- creating new process docs
- changing repo structure
- changing release semantics
- adding AI workflow changes

---

# 2. Basil Server / Framework Specs

Representative specs include:
- `FEAT-002` — Basil core server
- `FEAT-004` — authentication
- `FEAT-006` — browser error display
- `FEAT-016` — per-route `public_dir`
- `FEAT-020` — developer config overrides
- `FEAT-021` — SQLite dev tools
- `FEAT-035` — Git over HTTPS
- `FEAT-037` — fragment caching
- `FEAT-040+` — route/site/runtime refinements
- `FEAT-084+` and later framework integration features
- `FEAT-132`, `FEAT-139`, `FEAT-140` — testing/perf/runtime cleanup

## Recurring design themes
- secure-by-default
- explicit route behavior
- dev-mode ergonomics
- framework runtime injected into Parsley
- server owns privileged resources
- app logic should remain scriptable, but infrastructure is Go-owned

## Where the reasoning often lives
- the spec itself for first-order philosophy
- plans for route and handler wiring details
- `work/design/security-features.md`
- `work/design/rails-inspired-ux.md`
- `work/design/sessions-state.md`
- performance and readiness reports

## Warning for implementors
Framework changes often have language-facing consequences because Basil injects runtime structures into Parsley. Check both sides before changing interfaces.

---

# 3. Parsley Language Core Specs

Representative examples:
- syntax / semantics: `FEAT-008`, `009`, `014`, `015`, `022`, `029`, `122`, `123`
- typing / runtime objects: `025`, `087`, `090`, `091`, `118`, `119`, `121`
- error model: `023`, `105`, `125`
- output / formatting semantics: `024`, `100`, `113`, `120`

## Recurring design themes
- clarity over maximal cleverness
- readable, explicit syntax
- strong emphasis on helpful error messages
- expression-oriented output model
- gradual move toward coherent, method-based APIs
- preference for exactness where domain types justify it

## Where the reasoning often lives
- spec design decisions
- dedicated design docs for grammar-heavy or representation-heavy features
- plans for exact implementation shape
- changelog for later cleanup that supersedes earlier direction

## Warning for implementors
Many earlier exploratory features were later cleaned up or removed. Always check whether a later cleanup spec changed the intended long-term contract.

---

# 4. Parsley Standard Library / Modules Specs

Representative examples:
- `FEAT-018` — table module
- `FEAT-031` — std/math
- `FEAT-032` — std/valid
- `FEAT-033` — sanitizers
- `FEAT-034` — schema/API/id/auth direction
- `FEAT-087+` — table/builtin evolution
- `FEAT-096+` — computed exports, metadata, etc.

## Recurring design themes
- built-ins or stdlib should support real app work
- immutable / chainable APIs where practical
- module exports should be introspectable
- avoid parallel undocumented special cases
- move toward simpler and more coherent surfaces over time

## Where the reasoning often lives
- spec summary and design decisions
- design docs for schema/table/api discussions
- plans for deferred vs implemented split
- later cleanup specs if module forms changed

## Warning for implementors
Some stdlib surfaces were transitional. In particular, table/schema/API areas evolved significantly; check later specs and cleanup work before extending older shapes.

---

# 5. CLI / REPL / Tooling / Documentation Specs

Representative examples:
- `FEAT-100` — pretty-printer
- `FEAT-109`, `114` — editor/tree-sitter work
- `FEAT-111` — method registry
- `FEAT-112` — help system
- `FEAT-113` — CLI `-e` output
- `FEAT-131` — documentation overhaul

## Recurring design themes
- tooling should reduce hallucination/drift
- discoverability matters
- REPL, CLI, and docs should align
- metadata should generate reference material where possible
- examples must be executable/verifiable

## Where the reasoning often lives
- spec + plan
- doc audit reports
- help/introspection code
- formatter design docs

## Warning for implementors
Tooling changes can affect:
- user docs
- AI guidance
- reference generation
- tests that assert textual output

Treat them as interface changes, not just internal refactors.

---

# A Practical Decision Tree for Future Changes

## If you are changing parser or syntax
Read:
- the target feature spec
- any referenced design doc
- related grammar specs nearby in number/time
- formatter spec if syntax affects pretty-printing
- help/docs specs if syntax appears in reference material

Also inspect:
- lexer
- parser
- AST
- formatter
- docs examples

## If you are changing evaluator/runtime semantics
Read:
- target feature spec
- error model specs (`FEAT-023`, `FEAT-105`, `FEAT-125`)
- any cleanup/deprecation spec that may supersede older semantics
- matching plan

Also inspect:
- object model
- help/introspection if surface area changes
- docs and tests

## If you are changing Basil request/auth/API behavior
Read:
- `FEAT-002`
- `FEAT-004`
- `FEAT-034`
- any later Basil framework spec touching the same area
- relevant performance or investigation report

Also inspect:
- `server/`
- `auth/`
- evaluator glue code for Basil context injection

## If you are changing docs or examples
Read:
- `FEAT-131`
- relevant feature specs
- `PARSLEY-DOC-AUDIT.md`
- `DOC-VERIFICATION-LOG.md`

Rule:
- verify against implementation, not against another doc

## If you are removing or reviving old behavior
Read:
- `FEAT-128`
- changelog
- readiness audit reports
- any deprecation or cleanup plan

Rule:
- assume removals were intentional until proven otherwise

---

# How to Recognize a Spec That Is Mostly Historical

A spec may be historical/transitional if:

- later specs explicitly replace its API
- changelog lists breaking cleanup/removal
- implementation notes or plans say it was superseded
- acceptance criteria differ from actual code and later docs
- reports describe it as deprecated or transitional

Examples of likely caution zones:
- early table/module forms before later cleanup
- transitional print/output semantics
- migration-only CLI tooling
- deprecated uppercase components
- old documentation assumptions contradicted by FEAT-131 audit work

When you find one:
- do not ignore it
- use it to understand evolution
- but confirm final behavior from later specs/plans/code

---

# Suggested Mental Model of the Project

## Basil
Basil is the server/runtime/framework layer.

It is responsible for:
- process boundaries
- resource ownership
- auth authority
- request/response orchestration
- devtools and hosting behavior
- policy defaults (security, rate limits, etc.)

## Parsley
Parsley is the language/runtime/tooling layer.

It is responsible for:
- syntax
- evaluation
- representation
- templates/output
- stdlib / builtins / methods
- formatting / help / CLI / REPL
- language ergonomics

## The integration layer
The most important boundary in the codebase is not Go vs Parsley.  
It is:

- what Basil owns and injects,
- versus what Parsley code may express and manipulate.

Many specs are really about policing or refining that boundary.

---

# Where to Look for Actual Implementations

Below is a useful starting map.

## Basil implementation hotspots
- `cmd/basil/main.go`
- `server/server.go`
- `server/handler.go`
- `server/api.go`
- `server/errors.go`
- `server/devtools*.go`
- `server/ratelimit.go`
- `auth/`
- `config/`

## Parsley implementation hotspots
- `cmd/pars/main.go`
- `pkg/parsley/lexer/`
- `pkg/parsley/parser/`
- `pkg/parsley/ast/`
- `pkg/parsley/evaluator/`
- `pkg/parsley/format/`
- `pkg/parsley/help/`
- `pkg/parsley/repl/`

## Spec reasoning hotspots outside `work/specs/`
- `work/plans/`
- `work/design/`
- `work/reports/`
- `CHANGELOG.md`
- `docs/parsley/`
- `docs/guide/`

---

# AI-Agent Usage Notes

## Before making a change
For any non-trivial change, collect:

- target spec
- related plan
- related design/report
- related code file
- user-facing doc or changelog reference if public behavior changes

## When two sources disagree
Use this order:

1. implemented code
2. matching plan implementation notes
3. later spec or cleanup spec
4. design doc
5. older spec
6. older user docs

But do not stop at "code wins".  
If code disagrees with a current spec, note it explicitly and decide whether this is:
- drift
- partial implementation
- intentional divergence
- outdated spec

## When writing new specs
Prefer the existing repository pattern:

- human summary at top
- explicit acceptance criteria
- design decisions section
- AI-focused implementation details below divider
- references to dependent features/design docs
- note what is intentionally deferred

## When changing old features
Check for:
- backlog follow-ups
- reports mentioning audits/gaps
- readiness docs that may treat the current behavior as intentional
- changelog entries indicating public promise

---

# High-Value Companion Documents

These are often worth reading before major work.

## Workflow / governance
- `AGENTS.md`
- `work/README.md`
- `work/BACKLOG.md`
- `CHANGELOG.md`

## Design documents
- `work/design/api-design-summary.md`
- `work/design/FEAT-034-phases-3-6-design.md`
- `work/design/unified-error-model.md`
- `work/design/DESIGN-units-v3.md`
- `work/design/DESIGN-with-expression.md`
- `work/design/DESIGN-PRETTY-PRINTER.md`
- `work/design/FORMATTER_DESIGN.md`
- `work/design/schema-table-binding.md`
- `work/design/std-api-discussion.md`

## Reports / audits
- `work/reports/PARSLEY-DOC-AUDIT.md`
- `work/reports/DOC-VERIFICATION-LOG.md`
- `work/reports/1.0-READINESS-AUDIT.md`
- `work/reports/PARSLEY-1.0-ALPHA-READINESS.md`
- `work/reports/BASIL-PERFORMANCE-ANALYSIS-2026-03-10.md`
- `work/reports/CODE-QUALITY-ASSESSMENT-2026-03-10.md`

---

# Change-Safety Checklist

Before merging a behavioral change, answer:

- Which spec owns this behavior?
- Is there a later cleanup or replacement spec?
- Is the feature fully implemented, partially implemented, or intentionally deferred?
- Does a plan document a different implementation from the original spec?
- Is there a design doc that explains a non-obvious tradeoff?
- Does the changelog imply this is part of the public contract?
- Do docs need updating?
- Does help/introspection need updating?
- Does formatter/REPL/CLI output need updating?
- Does error taxonomy need updating?
- Does this change cross the Basil/Parsley boundary?

If you cannot answer these, you probably need to read more before changing code.

---

# Final Guidance

The specification set in this repository is best understood as a **layered decision history**.

Do not read `work/specs/` as a flat list of isolated feature requests.

Read it as:

1. a history of intended capabilities,
2. a map of architectural boundaries,
3. a set of design arguments,
4. and a warning system for where future changes can accidentally violate project philosophy.

If you remember only a few rules, remember these:

- Read the spec before the code.
- Read the plan before assuming the spec was implemented literally.
- Read the design doc when the tradeoff matters.
- Read the report when you need the deeper rationale.
- Trust verified implementation metadata over stale prose.
- Treat later cleanup specs as intentional narrowing, not inconvenience.
- Preserve the Basil/Parsley boundary unless you are explicitly redesigning it.

That is the "why" layer this repository expects implementors to consult before changing the "how".