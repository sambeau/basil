# Cheatsheet

One-page reference for the Basil development process.

## Workflow Overview

```
FEATURE                              BUG
───────                              ───
/new-feature                         /fix-bug
    │                                    │
    ▼                                    ▼
┌─────────┐                         ┌─────────┐
│  Spec   │ ◀── Human reviews       │ Report  │ ◀── Human provides details
└────┬────┘                         └────┬────┘
     │                                   │
     ▼                                   ▼
┌─────────┐                         ┌─────────┐
│  Plan   │ ◀── Human approves      │Investigate│
└────┬────┘                         └────┬────┘
     │                                   │
     ▼                                   ▼
┌─────────┐                         ┌─────────┐
│Implement│                         │  Fix    │
└────┬────┘                         └────┬────┘
     │                                   │
     ▼                                   ▼
┌─────────┐                         ┌─────────┐
│ Review  │ ◀── Human merges        │ Review  │ ◀── Human merges
└─────────┘                         └─────────┘
```

## ID Format

| Type | Format | Example |
|------|--------|---------|
| Feature | `FEAT-XXX` | FEAT-001 |
| Bug | `BUG-XXX` | BUG-001 |
| Plan | `PLAN-XXX` | PLAN-001 |

## Git Branches

| Type | Pattern | Example |
|------|---------|---------|
| Feature | `feat/FEAT-XXX-desc` | `feat/FEAT-001-user-auth` |
| Bug fix | `fix/BUG-XXX-desc` | `fix/BUG-003-null-pointer` |

## Commit Messages

```
<type>(<scope>): <description>

Types: feat, fix, docs, refactor, test, chore, perf
Scope: FEAT-XXX or BUG-XXX when applicable
```

**Examples:**
```
feat(FEAT-001): add user authentication
fix(BUG-003): prevent null pointer in config
chore: update dependencies
```

## File Locations

```
basil/
├── AGENTS.md          # AI reads this first
├── ID_COUNTER.md      # Get next ID here
├── BACKLOG.md         # Deferred items
├── CHANGELOG.md       # Release history
├── .github/
│   ├── prompts/       # /new-feature, /fix-bug, /release
│   ├── templates/     # Spec, bug, plan templates
│   └── instructions/  # Code standards, commit rules
└── docs/
    ├── guide/         # You are here
    ├── specs/         # FEAT-XXX.md files
    ├── plans/         # Implementation plans
    └── bugs/          # BUG-XXX.md files
```

## Status Values

### Specs (FEAT-XXX)
`draft` → `approved` → `in-progress` → `done` | `deferred`

### Bugs (BUG-XXX)
`reported` → `investigating` → `fix-in-progress` → `resolved` | `wont-fix`

### Plans (PLAN-XXX)
`draft` → `approved` → `in-progress` → `complete` | `abandoned`

## Commands

| Action | Command |
|--------|--------|
| Build | `go build -o basil .` |
| Test | `go test ./...` |
| Run | `go run .` |
| Lint | `golangci-lint run` |
| Full check | `go build -o basil . && go test ./...` |

## Prompt Files

| Prompt | Use for |
|--------|---------|
| `/new-feature` | Starting a new feature |
| `/fix-bug` | Fixing a bug |
| `/release` | Preparing a release |

## Human Checkpoints

AI **stops and waits** for human at:
1. Spec review (before planning)
2. Plan approval (before implementing)
3. Final review (before merge)
4. Release approval (before tagging)
