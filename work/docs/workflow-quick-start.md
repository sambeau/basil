# Workflow Quick Start Guide

Get up and running with the Basil development workflow process in 5 minutes.

For getting started with **using Basil as a web framework**, see [Basil Quick Start](../../docs/guide/basil-quick-start.md).

## Prerequisites
- VS Code with GitHub Copilot extension
- Go 1.21+ installed
- Git configured

## Your First Feature

### 1. Start a Conversation
Open Copilot Chat and type:
```
/new-feature I want to add [your feature description]
```

### 2. Review the Spec
The AI will create a spec file at `work/specs/FEAT-XXX.md`. 

**Your job:** Read the top section (Summary, User Story, Acceptance Criteria) and either:
- ✅ Approve to continue
- ✏️ Request changes

### 3. Review the Plan
The AI creates an implementation plan at `work/plans/FEAT-XXX-plan.md`.

**Your job:** Check the Checklist section and approve or request changes.

### 4. Let AI Implement
Once approved, the AI will:
- Create a feature branch
- Write code and tests
- Commit with proper messages
- Update progress in the plan

### 5. Review and Merge
When AI presents completed work:
- Review the changes
- Check tests pass: `go test ./...`
- Merge the feature branch to main

## Your First Bug Fix

### 1. Start a Conversation
```
/fix-bug [description of the bug]
```

### 2. Provide Details
Fill in reproduction steps if the AI asks.

### 3. Review Fix Strategy
AI will investigate and propose a fix. Approve or suggest alternatives.

### 4. Review and Merge
Same as features — review, test, merge.

## Common Commands

| Task | Command |
|------|--------|
| Build | `go build -o basil .` |
| Test | `go test ./...` |
| Run | `go run .` |

## Where to Find Things

| Looking for... | Check... |
|----------------|----------|
| What's being worked on | `work/plans/` |
| What's been deferred | `work/BACKLOG.md` |
| What's been released | `CHANGELOG.md` |
| How workflow works | `work/docs/workflow-faq.md` |
| How to use Basil | `docs/guide/faq.md` |

## Need Help?
- **Workflow questions?** Check the [Workflow FAQ](workflow-faq.md)
- **Basil/Parsley questions?** Check the [Basil FAQ](../../docs/guide/faq.md)
- Ask the AI — it'll answer and add to the appropriate FAQ
