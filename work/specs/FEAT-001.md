---
id: FEAT-001
title: "Development Process Framework"
status: complete
priority: high
created: 2025-11-30
author: "@human"
---

# FEAT-001: Development Process Framework

## Summary
Define a comprehensive Human-AI development workflow that manages features from idea to release and bugs from report to resolution. The process should be human-friendly for design/planning phases and AI-optimized for implementation, with clear handoff points and a feedback loop for deferred items.

## User Story
As a developer working with AI assistance, I want a structured development process so that features and bugs are tracked consistently, nothing falls through the cracks, and both human and AI can work efficiently in their respective strengths.

## Acceptance Criteria
- [x] Process infrastructure files exist (AGENTS.md, ID_COUNTER.md, BACKLOG.md, CHANGELOG.md)
- [x] Instruction files define code standards and commit conventions
- [x] Prompt files enforce step-by-step workflows for features, bugs, and releases
- [x] Templates use "newspaper article" pattern (human summary → AI details)
- [x] Human-friendly guide documentation exists with cheatsheet and FAQ
- [x] AI can allocate IDs without human intervention
- [x] Deferred items have clear path back to BACKLOG.md
- [x] Git workflow is documented (AI commits to branches, human merges to main)

## Design Decisions
- **Markdown everywhere**: Universal format for human and AI
- **YAML frontmatter**: Machine-parseable metadata
- **Newspaper article pattern**: Human-readable summary at top, AI-dense details below horizontal rule
- **AI-managed ID counter**: No scripts needed, AI reads/updates ID_COUNTER.md
- **Modular prompt files**: Separate workflows vs. one monolithic document
- **Guide documentation**: Reduces repeat questions, AI updates FAQ when answering

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `.github/copilot-instructions.md` — Update with workflow overview
- `.github/instructions/` — New folder for always-on rules
- `.github/prompts/` — New folder for workflow prompts  
- `.github/templates/` — New folder for document scaffolds
- `docs/guide/` — New folder for human documentation
- `docs/specs/` — New folder for feature specifications
- `docs/plans/` — New folder for implementation plans
- `docs/bugs/` — New folder for bug reports
- `docs/decisions/` — New folder for ADRs
- `AGENTS.md` — New file at root
- `ID_COUNTER.md` — New file at root
- `BACKLOG.md` — New file at root
- `CHANGELOG.md` — New file at root

### Dependencies
- Depends on: None (foundational feature)
- Blocks: All future features and bugs (they use this process)

### File Structure
```
basil/
├── AGENTS.md
├── ID_COUNTER.md
├── BACKLOG.md
├── CHANGELOG.md
├── .github/
│   ├── copilot-instructions.md
│   ├── instructions/
│   │   ├── code.instructions.md
│   │   └── commits.instructions.md
│   ├── prompts/
│   │   ├── new-feature.prompt.md
│   │   ├── fix-bug.prompt.md
│   │   └── release.prompt.md
│   └── templates/
│       ├── FEATURE_SPEC.md
│       ├── BUG_REPORT.md
│       └── IMPLEMENTATION_PLAN.md
└── docs/
    ├── guide/
    │   ├── README.md
    │   ├── quick-start.md
    │   ├── cheatsheet.md
    │   ├── faq.md
    │   └── walkthroughs/
    │       ├── creating-a-feature.md
    │       ├── fixing-a-bug.md
    │       └── making-a-release.md
    ├── specs/
    │   └── FEAT-001.md
    ├── plans/
    │   └── FEAT-001-plan.md
    ├── bugs/
    └── decisions/
```

### Edge Cases & Constraints
1. ID counter reaching 999 — Alert human, unlikely but documented
2. Conflicting instructions — Repository-wide takes precedence, avoid conflicts
3. Large context windows — Templates designed to be standalone, not require full codebase

### Out of Scope
- GitHub Issues integration (future enhancement)
- Automated changelog generation (manual for now)
- CI/CD pipeline integration

## Implementation Notes

### Actual Changes Made
- Created AGENTS.md with operational context and workflow rules
- Created ID_COUNTER.md with counters for FEAT, BUG, PLAN, ADR
- Created BACKLOG.md with priority sections
- Created CHANGELOG.md with initial structure
- Created code.instructions.md for Go standards
- Created commits.instructions.md for Conventional Commits
- Updated copilot-instructions.md with workflow overview
- Created FEATURE_SPEC.md template with newspaper pattern
- Created BUG_REPORT.md template with newspaper pattern
- Created IMPLEMENTATION_PLAN.md template
- Created new-feature.prompt.md workflow
- Created fix-bug.prompt.md workflow
- Created release.prompt.md workflow
- (In progress) Guide documentation

### Deferred Items
<!-- To be filled after implementation complete -->
