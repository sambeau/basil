# New Feature Workflow

Use this prompt when starting a new feature: `/new-feature <description>`

---

## Phase 1: Gather Requirements

Ask the user for:
1. **What** they want to build (one sentence)
2. **Why** they need it (problem it solves)
3. **Who** will use it (if not obvious)

If the description is too vague, ask clarifying questions before proceeding.

---

## Phase 2: Create Specification

1. Read `work/ID_COUNTER.md` and allocate the next FEAT ID
2. Update `work/ID_COUNTER.md` immediately (increment Next ID, update Last Allocated)
3. Create `work/specs/FEAT-XXX.md` using the template from `.github/templates/FEATURE_SPEC.md`
4. Fill in the human-readable sections (Summary, User Story, Acceptance Criteria)
5. Present the spec to the user for review

**Checkpoint**: Wait for user approval before proceeding.

---

## Phase 3: Create Implementation Plan

1. Allocate the next PLAN ID from `work/ID_COUNTER.md`
2. Update `work/ID_COUNTER.md` immediately
3. Create `work/plans/FEAT-XXX-plan.md` using `.github/templates/IMPLEMENTATION_PLAN.md`
4. Break down the work into small, testable tasks
5. Identify files to create/modify
6. Present the plan to the user for review

**Checkpoint**: Wait for user approval before proceeding.

---

## Phase 4: Implementation

1. Create feature branch: `feat/FEAT-XXX-short-description`
2. For each task in the plan:
   a. Implement the change
   b. Run tests: `go test ./...`
   c. Commit with conventional commit message
   d. Update plan progress log
3. Run full validation: `go build -o basil . && go test ./...`

---

## Phase 5: Wrap Up

1. Update the spec with any implementation notes
2. Add any deferred items to `work/BACKLOG.md`
3. Summarize what was done and what's ready for review
4. Remind user to review and merge to main

---

## Quick Reference

| Action | File |
|--------|------|
| Allocate ID | `work/ID_COUNTER.md` |
| Create spec | `work/specs/FEAT-XXX.md` |
| Create plan | `work/plans/FEAT-XXX-plan.md` |
| Track deferrals | `work/BACKLOG.md` |
| Build & test | `go build -o basil . && go test ./...` |
