---
id: PLAN-005
feature: FEAT-007
title: "Implementation Plan for Parsley Monorepo Merge"
status: draft
created: 2025-12-01
---

# Implementation Plan: FEAT-007

## Overview
Merge the Parsley language implementation into the Basil repository, updating all imports and removing the external dependency.

## Prerequisites
- [x] Feature spec approved (FEAT-007)
- [ ] Parsley repo is in a clean state (all changes committed)
- [ ] Basil repo is in a clean state

## Tasks

### Task 1: Copy Parsley source into Basil
**Files**: `pkg/parsley/`, `cmd/pars/` (new directories)
**Estimated effort**: Small

Steps:
1. Create `pkg/parsley/` and `cmd/pars/` directories in Basil
2. Copy Parsley's `pkg/*` contents into `pkg/parsley/`
3. Copy Parsley's `std/` into `pkg/parsley/std/`
4. Copy `cmd/pars/` into `cmd/pars/`
5. Copy relevant docs into `docs/parsley/`

Commands:
```bash
mkdir -p pkg/parsley cmd/pars docs/parsley
cp -r ../parsley/pkg/* pkg/parsley/
cp -r ../parsley/std pkg/parsley/
cp -r ../parsley/cmd/pars/* cmd/pars/
cp -r ../parsley/docs/* docs/parsley/
```

---

### Task 1b: Move Basil CLI to cmd/basil/
**Files**: `main.go` → `cmd/basil/main.go`
**Estimated effort**: Small

Steps:
1. Create `cmd/basil/` directory
2. Move `main.go` to `cmd/basil/main.go`
3. Move `main_test.go` to `cmd/basil/main_test.go` (if applicable)

Commands:
```bash
mkdir -p cmd/basil
git mv main.go cmd/basil/
git mv main_test.go cmd/basil/
```

---

### Task 2: Update Parsley internal imports
**Files**: All `.go` files under `pkg/parsley/` and `cmd/pars/`
**Estimated effort**: Medium

Steps:
1. Find all imports of `github.com/sambeau/parsley/pkg/`
2. Replace with `github.com/sambeau/basil/pkg/parsley/`
3. Verify no broken imports

Command:
```bash
find pkg/parsley cmd/pars -name "*.go" -exec sed -i '' 's|github.com/sambeau/parsley/pkg/|github.com/sambeau/basil/pkg/parsley/|g' {} \;
```

---

### Task 3: Update Basil imports
**Files**: `server/handler.go`, any other files importing parsley
**Estimated effort**: Small

Steps:
1. Find all Basil files importing parsley
2. Update import paths
3. Verify compilation

---

### Task 4: Update go.mod
**Files**: `go.mod`
**Estimated effort**: Small

Steps:
1. Remove `replace github.com/sambeau/parsley => ../parsley` directive
2. Remove `github.com/sambeau/parsley` from require block
3. Run `go mod tidy`

---

### Task 5: Build and test
**Files**: N/A
**Estimated effort**: Small

Steps:
1. `go build -o basil ./cmd/basil`
2. `go build -o pars ./cmd/pars`
3. `go test ./...`
4. Manual smoke test of both CLIs

---

### Task 6: Update Makefile
**Files**: `Makefile`
**Estimated effort**: Small

Steps:
1. Update build target to use `./cmd/basil`
2. Add `pars` build target
3. Update any other paths referencing root main.go

---

### Task 7: Clean up
**Files**: Various
**Estimated effort**: Small

Steps:
1. Remove `parsley-src` symlink
2. Update README if needed
3. Update AGENTS.md with new structure
4. Commit everything

---

## Validation Checklist
- [x] `go build -o basil ./cmd/basil` succeeds
- [x] `go build -o pars ./cmd/pars` succeeds
- [x] `go test ./...` passes
- [x] `./basil --version` works
- [x] `./pars --version` works
- [ ] Dev mode error display still works (manual test pending)
- [x] No `replace` directive in go.mod
- [x] Makefile works (`make build`, `make test`)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-01 | Task 1 | ✅ Complete | Copied pkg/* to pkg/parsley/ |
| 2025-12-01 | Task 1b | ✅ Complete | Moved main.go and main_test.go to cmd/basil/ |
| 2025-12-01 | Task 2 | ✅ Complete | Updated all imports via sed |
| 2025-12-01 | Task 3 | ✅ Complete | server/handler.go imports updated |
| 2025-12-01 | Task 4 | ✅ Complete | Removed replace directive, ran go mod tidy |
| 2025-12-01 | Task 5 | ✅ Complete | All builds pass, all tests pass |
| 2025-12-01 | Task 6 | ✅ Complete | Makefile updated with build-basil, build-pars targets |
| 2025-12-01 | Task 7 | ✅ Complete | Removed parsley-src symlink |

## Post-Migration
- Archive `github.com/sambeau/parsley` repo with note pointing to basil
- Update any external references

