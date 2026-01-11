---
id: FEAT-010
title: "Log version on startup"
status: complete
priority: low
created: 2025-12-01
author: "@human"
---

# FEAT-010: Log version on startup

## Summary
Display the Basil version in the startup logs so operators can see what version is running without needing to restart with `--version`.

## User Story
As a developer or operator, I want to see the Basil version in the startup logs so that I can verify which version is running without restarting the server.

## Acceptance Criteria
- [ ] Version string appears in server startup output
- [ ] Format matches `--version` output style: `basil version X.X.X (commit)`
- [ ] Works in both dev and production modes

## Design Decisions
- **Log location**: Print version immediately before "Starting Basil..." message for consistency
- **Pass version to server**: Add `Version` and `Commit` fields to server config or as parameters to `New()`

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `cmd/basil/main.go` — Pass version to server
- `server/server.go` — Store and log version on startup

### Dependencies
- None

### Edge Cases & Constraints
1. Version may be "dev" in local builds — still display it
2. Commit may be "unknown" in some builds — still display it

## Implementation Notes
*Added during/after implementation*

## Related
- Plan: `work/plans/FEAT-010-plan.md` (if needed)
