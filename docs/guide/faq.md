# Frequently Asked Questions

Questions answered during development. AI adds new entries when answering "how do I..." questions.

---

## Process Questions

### How do I start a new feature?
Use `/new-feature` in Copilot Chat with a description of what you want. The AI will guide you through spec creation, planning, and implementation.

*Added: 2025-11-30*

### How do I report a bug?
Use `/fix-bug` in Copilot Chat with a description of the problem. Provide reproduction steps when asked.

*Added: 2025-11-30*

### How do I make a release?
Use `/release` in Copilot Chat. AI will gather changes, determine version bump, and prepare changelog. You approve and execute git commands.

*Added: 2025-11-30*

### Where do deferred items go?
Items that can't be completed in current scope go to `BACKLOG.md`. They're categorized by priority and linked back to their source feature/bug.

*Added: 2025-11-30*

---

## File Questions

### What's the difference between specs and plans?
- **Specs** (`docs/specs/`): What we're building and why. Written mostly by human.
- **Plans** (`docs/plans/`): How we're building it, step by step. Written mostly by AI.

*Added: 2025-11-30*

### Where do I find the template for a new spec?
Templates are in `.github/templates/`. The AI uses these automatically when you use prompt files.

*Added: 2025-11-30*

### What's AGENTS.md for?
It's the first thing AI reads before any task. Contains build commands, project structure, workflow rules, and common pitfalls. Think of it as onboarding documentation for AI.

*Added: 2025-11-30*

---

## Git Questions

### Who does git commits?
AI commits to feature/bug branches. Human merges to main and creates release tags.

*Added: 2025-11-30*

### What commit format should I use?
Conventional Commits. See `.github/instructions/commits.instructions.md` or the [Cheatsheet](cheatsheet.md).

*Added: 2025-11-30*

---

## ID Questions

### How do IDs get assigned?
AI reads `ID_COUNTER.md`, takes the next number, and increments the counter. You don't need to do anything.

*Added: 2025-11-30*

### Can I manually assign an ID?
Yes, just edit `ID_COUNTER.md` directly. Make sure to increment the "Next ID" so it's not reused.

*Added: 2025-11-30*

---

<!-- AI: Add new Q&A entries above this line, with *Added: YYYY-MM-DD* -->
