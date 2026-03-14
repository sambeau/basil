# Workflow Documentation

This directory contains workflow, design, planning, and implementation-reference documentation for Basil and Parsley.

## Structure

- **specs/** — Feature specifications (`FEAT-XXX.md`)
- **plans/** — Implementation plans (`PLAN-XXX.md`, `FEAT-XXX-plan.md`)
- **bugs/** — Bug reports (`BUG-XXX.md`)
- **design/** — Design documents, architecture notes, and decision records
- **reports/** — Audits, investigations, implementation notes, and cross-cutting reference docs
- **docs/** — Workflow process guides and manual testing docs
- **parsley/** — Parsley-specific implementation documentation
  - `design/` — Language design documents
  - `implementation/` — Implementation notes
  - `verification/` — Verification and testing docs
- **ID_COUNTER.md** — Counter for generating unique IDs
- **BACKLOG.md** — Deferred items and future work

## Start Here

### For implementors
If you are making code or spec changes, start with:

1. The relevant spec in `specs/`
2. The matching plan in `plans/`
3. Any related design docs in `design/`
4. Any related investigations or audits in `reports/`

### New: specification guide for implementors
For a high-level map of the spec set — especially focused on **why decisions were made**, where to find the real rationale, and how to safely interpret older vs newer specs — read:

- `reports/specifications-guide-for-implementors.md`

That guide is intended as a reference for maintainers and AI agents. It explains:

- how to read the spec set safely
- which specs are foundational
- where to find decision rationale beyond the spec itself
- how to tell when a spec is historical, partial, or superseded
- what to read before changing Basil/Parsley behavior

### For end users
User-facing documentation is in `docs/` at the repository root:

- `docs/guide/` — Basil framework guides
- `docs/parsley/` — Parsley language reference

## Workflow Process

When working on features or bugs:

1. Read the relevant spec (`work/specs/FEAT-XXX.md`) or bug report (`work/bugs/BUG-XXX.md`)
2. Check `work/BACKLOG.md` for related deferred items
3. Read the matching plan in `work/plans/` if one exists
4. Read related design docs or reports if the change affects behavior, architecture, or semantics
5. Implement changes on a feature branch
6. Update the spec/bug with implementation notes
7. Add any deferred items to `work/BACKLOG.md`

See `AGENTS.md` in the repository root for detailed workflow instructions.