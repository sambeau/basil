---
id: PLAN-023
feature: FEAT-035
title: "Implementation Plan for Git over HTTPS"
status: in-progress
created: 2025-12-07
---

# Implementation Plan: FEAT-035 Git over HTTPS

## Overview

Implement Git HTTP server allowing developers to clone, pull, and push changes using standard Git commands. Uses `go-git-http` library with HTTP Basic Auth using API keys from FEAT-036.

## Prerequisites

- [x] FEAT-036 implemented (CLI User Management, API Keys)
- [x] API key validation function (`auth.ValidateAPIKey`)
- [x] Role system (admin/editor)
- [ ] `go-git-http` dependency added

## Tasks

### Task 1: Add go-git-http Dependency
**Files**: `go.mod`, `go.sum`
**Estimated effort**: Small

Steps:
1. Run `go get github.com/AaronO/go-git-http`
2. Verify dependency is added to go.mod

Tests:
- Build succeeds

---

### Task 2: Add Git Configuration
**Files**: `config/config.go`
**Estimated effort**: Small

Steps:
1. Add `GitConfig` struct with `Enabled` and `RequireAuth` fields
2. Add `Git GitConfig` field to main `Config` struct
3. Default: `Enabled: false`, `RequireAuth: true`

Tests:
- Config loads with git section
- Config loads without git section (defaults)

---

### Task 3: Create Git HTTP Handler
**Files**: `server/git.go` (new)
**Estimated effort**: Medium

Steps:
1. Create `GitHandler` struct wrapping `githttp.GitHttp`
2. Implement `NewGitHandler(siteDir string, db *auth.DB)` constructor
3. Implement authentication middleware using `auth.ValidateAPIKey`
4. Implement role checking (editor/admin for push, any for clone/pull)
5. Implement post-push event handler to trigger reload

Key implementation:
```go
type GitHandler struct {
    git    *githttp.GitHttp
    authDB *auth.DB
    onPush func() // Callback for post-push reload
}

func (h *GitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request)
func (h *GitHandler) authenticate(w http.ResponseWriter, r *http.Request) (*auth.User, bool)
```

Tests:
- Unit test authentication logic
- Unit test role checking

---

### Task 4: Integrate Git Handler into Server
**Files**: `server/server.go`
**Estimated effort**: Small

Steps:
1. Add `gitHandler *GitHandler` field to `Server` struct
2. Add `initGit()` method to initialize Git handler if enabled
3. Call `initGit()` in `New()` constructor
4. Mount handler at `/.git/` in `setupRoutes()`
5. Wire post-push callback to `ReloadHandlers()`

Tests:
- Server starts with git enabled
- Server starts with git disabled
- `/.git/` route is accessible when enabled

---

### Task 5: Dev Mode Unauthenticated Access
**Files**: `server/git.go`
**Estimated effort**: Small

Steps:
1. Check `config.Server.Dev` in auth middleware
2. If dev mode AND localhost, skip auth
3. Log warning if unauthenticated access enabled

Tests:
- Dev mode allows unauthenticated clone on localhost
- Dev mode still requires auth on non-localhost

---

### Task 6: Security Warnings
**Files**: `server/git.go`, `server/server.go`
**Estimated effort**: Small

Steps:
1. Warn if `git.enabled: true` but `git.require_auth: false` on non-localhost
2. Warn if git enabled without HTTPS (except dev mode)

Tests:
- Warning logged for insecure configurations

---

### Task 7: Integration Testing
**Files**: Manual testing
**Estimated effort**: Medium

Steps:
1. Create test user and API key
2. Test `git clone` with valid API key
3. Test `git clone` with invalid API key (expect 401)
4. Test `git push` as editor (expect success)
5. Test `git push` without proper role (expect 403)
6. Verify live reload after push

Tests:
- Manual test plan execution

---

## Validation Checklist
- [ ] All tests pass: `make check`
- [ ] Build succeeds: `make build`
- [ ] `git clone` works with API key
- [ ] `git push` works and triggers reload
- [ ] Authentication failures return proper errors
- [ ] Role checking works for push operations
- [ ] Dev mode allows unauthenticated localhost access
- [ ] Documentation updated

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-07 | Task 1 | ✅ Complete | go-git-http v0.0.0-20161214145340 |
| 2025-12-07 | Task 2 | ✅ Complete | GitConfig with Enabled/RequireAuth |
| 2025-12-07 | Task 3 | ✅ Complete | server/git.go with auth, role check |
| 2025-12-07 | Task 4 | ✅ Complete | Integrated into server.go |
| 2025-12-07 | Task 5 | ✅ Complete | Dev mode localhost bypass |
| 2025-12-07 | Task 6 | ✅ Complete | Warnings for insecure config |
| 2025-12-07 | Task 7 | 🔄 In Progress | Unit tests pass, manual testing needed |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- Git LFS support — v1 doesn't support large files
- Pre-receive hooks for Parsley syntax validation
- Force push rejection option
- Per-key scopes (git-only keys)
- Push notifications (email/Slack)
