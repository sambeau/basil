# Basil Development Guide

Welcome to the human-friendly documentation for the Basil development process.

## Quick Links

| I want to... | Go to... |
|--------------|----------|
| Get started quickly | [Quick Start](quick-start.md) |
| Add authentication | [Authentication](authentication.md) |
| Look up a command/format | [Cheatsheet](cheatsheet.md) |
| Find an answer to a question | [FAQ](faq.md) |
| Create a new feature | [Creating a Feature](walkthroughs/creating-a-feature.md) |
| Fix a bug | [Fixing a Bug](walkthroughs/fixing-a-bug.md) |
| Make a release | [Making a Release](walkthroughs/making-a-release.md) |

## How This Process Works

```
┌─────────────────────────────────────────────────────────────────┐
│                         HUMAN                                    │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐      │
│  │  Idea   │───▶│ Design  │───▶│ Approve │───▶│ Release │      │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘      │
│       │              │              ▲              ▲            │
│       ▼              ▼              │              │            │
├───────────────────────────────────────────────────────────────  │
│                          AI                                      │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐      │
│  │ Assist  │───▶│  Plan   │───▶│Implement│───▶│ Present │      │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘      │
└─────────────────────────────────────────────────────────────────┘
```

**Human responsibilities:**
- Ideas and requirements
- Design decisions
- Reviewing and approving plans
- Merging to main and releasing

**AI responsibilities:**
- Assisting with planning
- Creating detailed implementation plans
- Writing code and tests
- Tracking progress and deferred items
- Presenting work for review

## Key Concepts

### Documents Follow the "Newspaper Pattern"
Every spec and bug report has:
1. **Top section** (above `---`): Human-readable summary, user story, acceptance criteria
2. **Bottom section** (below `---`): AI-dense technical details, implementation notes

You only need to read the top section to understand what's happening.

### IDs Track Everything
- `FEAT-001`: Features
- `BUG-001`: Bugs
- `PLAN-001`: Implementation plans
- `ADR-001`: Architecture decisions

IDs are allocated from `ID_COUNTER.md` — the AI manages this automatically.

### Nothing Gets Lost
- Deferred items go to `BACKLOG.md`
- All changes are tracked in `CHANGELOG.md`
- AI updates the FAQ when answering your questions

## File Locations

| What | Where |
|------|-------|
| AI instructions | `AGENTS.md`, `.github/copilot-instructions.md` |
| ID tracking | `ID_COUNTER.md` |
| Deferred items | `BACKLOG.md` |
| Release history | `CHANGELOG.md` |
| Feature specs | `docs/specs/FEAT-XXX.md` |
| Bug reports | `docs/bugs/BUG-XXX.md` |
| Implementation plans | `docs/plans/` |
| This guide | `docs/guide/` |
