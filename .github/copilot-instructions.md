# Copilot Instructions for Basil

## Overview
Basil is a Go CLI application with a structured Human-AI development workflow.

## Before Any Task
1. Read `AGENTS.md` at the repository root — it contains build commands, project structure, and workflow rules
2. Check `BACKLOG.md` for related deferred items
3. Use the appropriate prompt file for your task type

## Workflow Entry Points
- **New Feature**: Use `/new-feature` prompt
- **Bug Fix**: Use `/fix-bug` prompt  
- **Release**: Use `/release` prompt

## Key Conventions
- Features: `FEAT-XXX` in `docs/specs/`
- Bugs: `BUG-XXX` in `docs/bugs/`
- Plans: `docs/plans/`
- IDs: Managed via `ID_COUNTER.md`

## Git Rules
- AI commits to feature/bug branches
- Human merges to main
- Human creates release tags
- Use Conventional Commits format

## Documentation
- Update `docs/guide/faq.md` when answering "how do I..." questions
- Add deferred items to `BACKLOG.md`
