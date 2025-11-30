# Bug Fix Workflow

Use this prompt when fixing a bug: `/fix-bug <description>`

---

## Phase 1: Gather Information

Ask the user for:
1. **What** is happening (observed behavior)
2. **What** should happen (expected behavior)
3. **How** to reproduce (step-by-step)
4. **Environment** (OS, Go version, etc.) if relevant

If reproduction steps are unclear, ask for more detail.

---

## Phase 2: Create Bug Report

1. Read `ID_COUNTER.md` and allocate the next BUG ID
2. Update `ID_COUNTER.md` immediately (increment Next ID, update Last Allocated)
3. Create `docs/bugs/BUG-XXX.md` using `.github/templates/BUG_REPORT.md`
4. Document the bug with all gathered information
5. Present the report to the user for confirmation

**Checkpoint**: Wait for user confirmation before proceeding.

---

## Phase 3: Investigate

1. Reproduce the bug locally (if possible)
2. Identify the root cause
3. Document findings in the bug report under "Investigation Notes"
4. Propose a fix approach

**Checkpoint**: Present fix approach for approval.

---

## Phase 4: Fix

1. Create bug fix branch: `fix/BUG-XXX-short-description`
2. Implement the fix
3. Add or update tests to cover the bug
4. Run tests: `go test ./...`
5. Commit with conventional commit message: `fix(<scope>): <description>`
6. Run full validation: `go build -o basil . && go test ./...`

---

## Phase 5: Wrap Up

1. Update bug report with resolution notes
2. Change bug report status to `resolved`
3. Add any deferred items to `BACKLOG.md`
4. Summarize the fix and what's ready for review
5. Remind user to review and merge to main

---

## Quick Reference

| Action | File |
|--------|------|
| Allocate ID | `ID_COUNTER.md` |
| Create bug report | `docs/bugs/BUG-XXX.md` |
| Track deferrals | `BACKLOG.md` |
| Build & test | `go build -o basil . && go test ./...` |
