---
id: FEAT-005
title: "Semantic Versioning"
status: complete
priority: low
created: 2025-11-30
completed: 2025-11-30
author: "@sambeau"
---

# FEAT-005: Semantic Versioning

## Summary
Add proper semantic versioning to Basil with build-time injection from git tags.

## User Story
As a user, I want to see the version of Basil I'm running so I can report bugs accurately and know if I need to update.

## Acceptance Criteria
- [x] `basil --version` shows version and commit hash: `basil version 0.2.0 (abc1234)`
- [x] Version defaults to "dev" for untagged builds
- [x] Version injected at build time via `-ldflags`
- [x] Makefile documents the build command
- [x] AGENTS.md updated with versioned build command

## Technical Approach

### Variables (in main.go)
```go
var (
    Version = "dev"      // Set via -ldflags
    Commit  = "unknown"  // Set via -ldflags
)
```

### Build Command
```bash
go build -ldflags "-X main.Version=$(git describe --tags --always) -X main.Commit=$(git rev-parse --short HEAD)" -o basil .
```

### Output Format
```
basil version 0.2.0 (abc1234)
```

For untagged builds:
```
basil version dev (abc1234)
```

## Implementation Notes
<!-- Filled after implementation -->
