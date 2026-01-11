# Workflow Documentation

This directory contains workflow and contributor documentation for Basil development.

## Structure

- **specs/** — Feature specifications (FEAT-XXX.md)
- **plans/** — Implementation plans (PLAN-XXX.md)
- **bugs/** — Bug reports (BUG-XXX.md)
- **design/** — Design documents and architecture decisions
- **reports/** — Audits, investigations, and analysis reports
- **docs/** — Workflow process guides and manual testing docs
- **parsley/** — Parsley language implementation documentation
  - design/ — Language design documents
  - implementation/ — Implementation notes
  - verification/ — Verification and testing docs
- **ID_COUNTER.md** — Counter for generating unique IDs
- **BACKLOG.md** — Deferred items and future work

## For End Users

User-facing documentation is in `docs/` at the repository root:
- `docs/guide/` — Basil framework guides
- `docs/parsley/` — Parsley language reference

## Workflow Process

When working on features or bugs:

1. Read the relevant spec (work/specs/FEAT-XXX.md) or bug report (work/bugs/BUG-XXX.md)
2. Check work/BACKLOG.md for related deferred items
3. Create an implementation plan in work/plans/ if needed
4. Implement changes on a feature branch
5. Update the spec/bug with implementation notes
6. Add any deferred items to work/BACKLOG.md

See AGENTS.md in the repository root for detailed workflow instructions.
