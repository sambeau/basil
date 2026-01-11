---
id: PLAN-059
feature: N/A (Infrastructure)
title: "Documentation Reorganization: docs/ vs work/ Split"
status: draft
created: 2026-01-11
---

# Implementation Plan: Documentation Reorganization

## Overview

Reorganize repository documentation by audience:
- **docs/** — User-facing documentation (guides, reference, examples)
- **work/** — Workflow/contributor documentation (specs, plans, bugs, design, reports)

This addresses the current issue where docs/ mixes user documentation with workflow documentation, making it unclear what's for end users vs contributors.

**Note on decisions/**: Architecture decisions are now captured iteratively in design documents and formalized in specs, rather than as separate ADRs. The existing ADR-001 will move to work/design/.

**Note on root files**: AGENTS.md stays in root (best practice for AI tooling discoverability). ID_COUNTER.md and BACKLOG.md move to work/ (workflow-only, all .github/ references will be updated).

**Branch:** `feat/docs-reorganization`  
**Estimated Total Time:** 10-12 hours  
**Files Affected:** ~249 documentation files + 3 package moves (auth, config, search) + ~20 .github files

## Prerequisites

- [x] Research completed (investigation report)
- [x] Phased plan approved by human
- [x] Current work committed (dictionary iteration, type tests)
- [ ] Clean working directory before starting

## Tasks

### Phase 1: Branch Creation & Structure Setup
**Estimated effort:** Small (30 minutes)  
**Risk:** Low  
**Blocking:** No

**Steps:**
1. Create feature branch: `git checkout -b feat/docs-reorganization`
2. Create work/ directory structure:
   ```bash
   mkdir -p work/{docs,specs,plans,bugs,design,reports}
   mkdir -p work/parsley/{design,implementation,verification}
   ```
3. Create placeholder README files:
   - `work/README.md` — Workflow documentation index
   - `work/docs/README.md` — Workflow process guides
   - `work/parsley/README.md` — Parsley implementation notes
4. Add .gitkeep files where needed

**Validation:**
```bash
ls -la work/
tree work/ -L 2
```

**Commit message:**
```
feat: create work/ directory structure for workflow docs

Established new work/ directory for workflow/contributor documentation
separate from user-facing docs/.

Structure:
- work/docs/ — Workflow process guides
- work/specs/ — Feature specifications
- work/plans/ — Implementation plans
- work/bugs/ — Bug reports
- work/design/ — Design documents
- work/reports/ — Audits, investigations, and analysis reports
- work/parsley/ — Parsley implementation docs
```

---

### Phase 2: Move Workflow Documentation (Bulk)
**Estimated effort:** Medium (1.5 hours)  
**Risk:** Low (git mv preserves history)  
**Blocking:** No

**Steps:**
1. Move specs/: `git mv work/specs/* work/specs/`
2. Move plans/: `git mv work/plans/* work/plans/`
3. Move bugs/: `git mv work/bugs/* work/bugs/`
4. Move design/: `git mv work/design/* work/design/`
6. Move Parsley workflow docs:
   ```bash
   git mv docs/parsley/design work/parsley/
   git mv docs/parsley/implementation work/parsley/
   git mv docs/parsley/verification work/parsley/
   ```
7. Move top-level workflow docs:
   ```bash
   git mv docs/FEAT-085-AUDIT.md work/reports/
   git mv docs/basil-performance-analysis.md work/reports/
   git mv docs/MANUAL_TESTING.md work/docs/
   ```
8. Move walkthrough guides:
   ```bash
   git mv work/docs/* work/docs/
   rmdir docs/guide/walkthroughs
   ```
8. Move decisions/ content to design/:
   ```bash
   git mv docs/decisions/ADR-001-notification-api-defer.md work/design/
   ```
9. Move ID_COUNTER.md to work/:
   ```bash
   git mv ID_COUNTER.md work/ID_COUNTER.md
   ```
10. Move BACKLOG.md to work/:
   ```bash
   git mv BACKLOG.md work/BACKLOG.md
   ```
11. Remove empty directories:
   ```bash
   rmdir docs/specs docs/plans docs/bugs docs/decisions docs/design
   ```

**Validation:**
```bash
# Check file counts
ls work/specs/*.md | wc -l      # Should be ~86
ls work/plans/*.md | wc -l      # Should be ~72
ls work/bugs/*.md | wc -l       # Should be ~15
ls work/design/*.md | wc -l     # Should be ~25 (includes ADR-001)
ls work/reports/*.md | wc -l    # Should be ~2
ls work/parsley/design/*.md | wc -l  # Should be ~40

# Verify git tracked moves (not deletions)
git status | grep renamed
```

**Commit message:**
```
refactor: move workflow docs from docs/ to work/

Reorganize documentation by audience:
- docs/ = user-facing documentation
- work/ = workflow/contributor documentation

Moved 249 files:
- specs/ (86 files) → work/specs/
- plans/ (72 files) → work/plans/
- bugs/ (15 files) → work/bugs/
- design/ (25 files) → work/design/ (includes ADR-001)
- parsley/design/ (40 files) → work/parsley/design/
- parsley/implementation/ (1 file) → work/parsley/implementation/
- parsley/verification/ (1 file) → work/parsley/verification/
- walkthroughs/ (3 files) → work/docs/
- audits and reports → work/reports/
- ID_COUNTER.md → work/ID_COUNTER.md (workflow tracking)
- BACKLOG.md → work/BACKLOG.md (deferred items tracking)

Git mv used to preserve history.
```

---

### Phase 3: Update Critical Workflow Files
**Estimated effort:** Large (2 hours)  
**Risk:** HIGH (breaks AI workflows if wrong)  
**Blocking:** YES - MUST TEST before continuing

**Files to update:**
1. `AGENTS.md` — Rewrite project structure section (lines 55-67), update all path references, keep in root
2. `.github/copilot-instructions.md` — Update all docs/ path references (6 lines) + ID_COUNTER.md + BACKLOG.md paths
3. `.github/instructions/code.instructions.md` — Update 3 path references
4. `.github/prompts/new-feature.prompt.md` — Update spec/plan paths (4 references) + ID_COUNTER.md path
5. `.github/prompts/fix-bug.prompt.md` — Update bug report paths (2 references) + ID_COUNTER.md path
6. `.github/prompts/release.prompt.md` — Update changelog/docs paths (1 reference) + BACKLOG.md path if referenced
7. `.github/templates/IMPLEMENTATION_PLAN.md` — Update file location table + BACKLOG.md path
8. `.github/templates/FEATURE_SPEC.md` — Update paths if referenced + ID_COUNTER.md path
9. `.github/templates/BUG_REPORT.md` — Update paths if referenced + ID_COUNTER.md path

**Validation (CRITICAL):**
```bash
# Test AI workflow commands in GitHub Copilot Chat:
# 1. Try: /new-feature [describe something simple]
# 2. Try: /fix-bug [describe something]
# 3. Verify specs get created in work/specs/ not work/specs/
# 4. Check that templates reference correct paths

# Manual checks - should return NO results:
grep -r "docs/specs" .github/
grep -r "docs/plans" .github/
grep -r "docs/bugs" .github/
grep -r "docs/design" .github/
grep -r "docs/decisions" .github/

# Check ID_COUNTER.md references are updated (should all point to work/)
grep -r "ID_COUNTER.md" .github/ AGENTS.md
# All references should be work/ID_COUNTER.md, not root ID_COUNTER.md

# Check BACKLOG.md references are updated (should all point to work/)
grep -r "BACKLOG.md" .github/ AGENTS.md
# All references should be work/BACKLOG.md, not root BACKLOG.md
```

**⚠️ CHECKPOINT:** If AI workflows don't work correctly, FIX before continuing to Phase 4

**⚠️ CRITICAL:** This reorganization MUST work for AIs as well as (if not better than) humans. All path changes must be accurately reflected in .github/ instructions so AI agents can discover and use workflow files.

**Commit message:**
```
refactor: update workflow files for work/ structure

Updated all .github/ files and AGENTS.md to reference new work/
directory structure:
- work/specs/ → work/specs/
- work/plans/ → work/plans/
- work/bugs/ → work/bugs/
- work/design/ → work/design/
- docs/decisions/ → work/design/ (ADRs merged into design)
- work/docs/ → work/docs/

Files updated:
- AGENTS.md (project structure, workflow rules, kept in root)
- .github/copilot-instructions.md (docs/ paths + ID_COUNTER.md + BACKLOG.md)
- .github/instructions/code.instructions.md
- .github/prompts/*.prompt.md (3 files, updated ID_COUNTER.md/BACKLOG.md paths)
- .github/templates/*.md (3 files, updated ID_COUNTER.md/BACKLOG.md paths)

AI workflows now create specs/plans/bugs in work/ directory.
All AI instructions updated for work/ID_COUNTER.md and work/BACKLOG.md locations.
```

---

### Phase 4: Update Internal Cross-References
**Estimated effort:** Large (3 hours)  
**Risk:** Medium (breaks internal doc links)  
**Blocking:** No (can fix later if needed)

**Steps:**
1. Batch find-replace in work/ files:
   ```bash
   # Update spec references
   find work/specs work/plans work/bugs work/design -type f -name "*.md" -exec sed -i '' 's|/work/specs/|/work/specs/|g' {} +
   find work/specs work/plans work/bugs work/design -type f -name "*.md" -exec sed -i '' 's|work/specs/|work/specs/|g' {} +
   
   # Update plan references
   find work/specs work/plans work/bugs work/design -type f -name "*.md" -exec sed -i '' 's|/work/plans/|/work/plans/|g' {} +
   find work/specs work/plans work/bugs work/design -type f -name "*.md" -exec sed -i '' 's|work/plans/|work/plans/|g' {} +
   
   # Update bug references
   find work/specs work/plans work/bugs work/design -type f -name "*.md" -exec sed -i '' 's|/work/bugs/|/work/bugs/|g' {} +
   find work/specs work/plans work/bugs work/design -type f -name "*.md" -exec sed -i '' 's|work/bugs/|work/bugs/|g' {} +
   
   # Update design references
   find work/specs work/plans work/bugs work/design -type f -name "*.md" -exec sed -i '' 's|/work/design/|/work/design/|g' {} +
   find work/specs work/plans work/bugs work/design -type f -name "*.md" -exec sed -i '' 's|work/design/|work/design/|g' {} +
   
   # Update parsley design references
   find work/specs work/plans work/bugs work/design -type f -name "*.md" -exec sed -i '' 's|/work/parsley/design/|/work/parsley/design/|g' {} +
   find work/specs work/plans work/bugs work/design -type f -name "*.md" -exec sed -i '' 's|work/parsley/design/|work/parsley/design/|g' {} +
   
   # Update walkthrough references
   find work/specs work/plans work/bugs work/design -type f -name "*.md" -exec sed -i '' 's|work/docs/|work/docs/|g' {} +
   ```

2. Manual review of key files:
   - Check work/specs/FEAT-001.md through FEAT-010 for remaining old paths
   - Check recent specs (FEAT-080+) which likely have more cross-references
   - Check work/design/ files
   - Check work/docs/ walkthrough files

3. Update work/BACKLOG.md if it references old docs/ structure

**Validation:**
```bash
# Search for remaining old paths (should be minimal/none)
grep -r "work/specs/" work/
grep -r "work/plans/" work/
grep -r "work/bugs/" work/
grep -r "work/design/" work/
grep -r "docs/guide/walkthroughs" work/

# Check BACKLOG.md
grep -n "docs/" work/BACKLOG.md
```

**Commit message:**
```
refactor: update internal cross-references in work/

Batch updated ~100+ cross-references in workflow documents:
- work/specs/ → work/specs/
- work/plans/ → work/plans/
- work/bugs/ → work/bugs/
- work/design/ → work/design/
- work/parsley/design/ → work/parsley/design/
- work/docs/ → work/docs/

Manual review of key specs and design docs completed.
```

---

### Phase 5: Reorganize User Documentation
**Estimated effort:** Small (1 hour)  
**Risk:** Low  
**Blocking:** No

**Steps:**
1. Merge manual/ into docs/parsley/manual/:
   ```bash
   mkdir -p docs/parsley/manual
   git mv docs/manual/builtins docs/parsley/manual/
   git mv docs/manual/stdlib docs/parsley/manual/
   rmdir docs/manual
   ```

2. Clean up metadata files:
   ```bash
   find docs -name ".Ulysses-*.plist" -delete
   find docs -name ".DS_Store" -delete
   git add -u  # Stage deletions
   ```

3. Create/update README files:
   - Update `docs/README.md` as top-level docs index
   - Update `docs/guide/README.md` to clarify it's for Basil users
   - Update `docs/parsley/README.md` to clarify it's language reference

4. Handle archive files (defer to human for 1.0 cleanup):
   - Leave `docs/parsley/CHEATSHEET.old.md` for human review
   - Leave `docs/parsley/Parts.txt` for human review
   - Leave `docs/parsley/archive/` for human review
   - Leave `docs/parsley/TODO.md` for human review

**Validation:**
```bash
# Check docs/ structure is clean
tree docs/ -L 2

# Should show:
# docs/
# ├── README.md
# ├── guide/          (Basil user guides)
# └── parsley/        (Parsley language reference)
#     └── manual/
#         ├── builtins/
#         └── stdlib/
```

**Commit message:**
```
refactor: reorganize user documentation in docs/

- Merged docs/manual/ into docs/parsley/manual/
- Removed editor metadata (.Ulysses-*.plist, .DS_Store)
- Updated README files for clarity
- docs/ now contains ONLY user-facing documentation:
  - docs/guide/ = Basil framework guides
  - docs/parsley/ = Parsley language reference

Archive files (CHEATSHEET.old.md, etc.) deferred to human review before 1.0.
```

---

### Phase 6: Breaking Changes (auth/, config/ moves)
**Estimated effort:** Large (2 hours)  
**Risk:** HIGH (breaks imports, requires testing)  
**Blocking:** YES - MUST TEST thoroughly

**Steps:**
1. Move auth/ to server/auth/:
   ```bash
   git mv auth server/auth
   ```

2. Move config/ to server/config/:
   ```bash
   git mv config server/config
   ```

3. Move pkg/search/ to server/search/:
   ```bash
   git mv pkg/search server/search
   ```

4. Update imports in all Go files:
   ```bash
   # Update auth imports
   find . -name "*.go" -exec sed -i '' 's|"github.com/sambeau/basil/auth"|"github.com/sambeau/basil/server/auth"|g' {} +
   
   # Update config imports
   find . -name "*.go" -exec sed -i '' 's|"github.com/sambeau/basil/config"|"github.com/sambeau/basil/server/config"|g' {} +
   
   # Update search imports
   find . -name "*.go" -exec sed -i '' 's|"github.com/sambeau/basil/pkg/search"|"github.com/sambeau/basil/server/search"|g' {} +
   ```

4. Run go mod tidy:
   ```bash
   go mod tidy
   ```

**Validation (COMPREHENSIVE):**
```bash
# 1. Build check
make dev
# Should complete without errors

# 2. Run all tests
make test
# All tests should pass

# 3. Full validation
make check
# Should pass (build + test)

# 4. Manual smoke tests
cd examples/hello && ../../basil &
# Visit http://localhost:8080/ - should work

cd examples/auth && ../../basil &
# Visit http://localhost:8080/ - auth should work

# 5. Check for any remaining old imports
grep -r '"github.com/sambeau/basil/auth"' --include="*.go" .
grep -r '"github.com/sambeau/basil/config"' --include="*.go" .
grep -r '"github.com/sambeau/basil/pkg/search"' --include="*.go" .
# Should return NO results
```

**⚠️ CHECKPOINT:** If tests fail, FIX before continuing to Phase 7

**Commit message:**
```
refactor: move auth/, config/, and search/ to server/

Relocated server-specific packages to server/ directory:
- auth/ → server/auth/
- config/ → server/config/
- pkg/search/ → server/search/

These are Basil-server-specific implementations, not reusable
packages, so they belong under server/ rather than root/pkg/.

Updated all imports:
- github.com/sambeau/basil/auth → github.com/sambeau/basil/server/auth
- github.com/sambeau/basil/config → github.com/sambeau/basil/server/config
- github.com/sambeau/basil/pkg/search → github.com/sambeau/basil/server/search

Breaking change: External imports of these packages will need updating.

All tests pass.
```

---

### Phase 7: Final Validation & Documentation
**Estimated effort:** Small (1 hour)  
**Risk:** Low  
**Blocking:** No

**Steps:**
1. Create comprehensive work/README.md (see template in phased plan)

2. Update root README.md:
   - Add note about docs/ vs work/ split
   - Link to docs/ for users, work/ for contributors

3. Run final checks:
   ```bash
   # Verify directory structure
   tree -L 2 -d
   
   # Run all tests again
   make check
   
   # Check for any lingering issues
   grep -r "docs/specs" .
   grep -r "docs/plans" .
   grep -r "docs/bugs" .
   # Should only find historical references in old commits
   ```

4. Update work/BACKLOG.md if any cleanup items deferred

**Validation:**
```bash
# Test AI workflows one final time
# In GitHub Copilot Chat:
# 1. /new-feature Test feature after reorganization
# 2. Verify it creates work/specs/FEAT-XXX.md correctly
# 3. /fix-bug Test bug after reorganization
# 4. Verify it creates work/bugs/BUG-XXX.md correctly
```

**Commit message:**
```
docs: finalize work/ structure with README files

Added comprehensive README files documenting the work/ structure
and workflow documentation organization.

Updated root README.md to clarify docs/ (user docs) vs work/
(contributor docs) split.

All tests pass. Ready for merge to main.
```

---

### Phase 8: Merge to Main
**Estimated effort:** Small (30 minutes)  
**Risk:** Low (already tested)  
**Blocking:** Final human review

**Steps:**
1. Final review:
   ```bash
   git log --oneline feat/docs-reorganization
   git diff main --stat
   ```

2. Merge to main:
   ```bash
   git checkout main
   git merge --no-ff feat/docs-reorganization -m "refactor: reorganize documentation by audience (docs/ vs work/)

   Completed comprehensive documentation reorganization to separate:
   - docs/ = User-facing documentation (guides, reference, examples)
   - work/ = Workflow/contributor documentation (specs, plans, bugs, design)
   
   Major changes:
   - Moved 249 workflow files to work/
   - Updated all .github/ workflow files
   - Moved auth/, config/, and search/ to server/ (breaking change)
   - Updated 100+ internal cross-references
   - Cleaned up metadata and archive files
   
   All tests pass. Ready for public alpha."
   ```

3. Push to remote:
   ```bash
   git push origin main
   ```

4. Delete feature branch:
   ```bash
   git branch -d feat/docs-reorganization
   git push origin --delete feat/docs-reorganization
   ```

**Commit message:**
```
refactor: reorganize documentation by audience (docs/ vs work/)

Completed comprehensive documentation reorganization to separate:
- docs/ = User-facing documentation (guides, reference, examples)
- work/ = Workflow/contributor documentation (specs, plans, bugs, design)

Major changes:
- Moved 249 workflow files to work/
- Updated all .github/ workflow files
- Moved auth/, config/, and search/ to server/ (breaking change)
- Updated 100+ internal cross-references
- Cleaned up metadata and archive files

All tests pass. Ready for public alpha.
```

---

## Validation Checklist

### After Phase 3 (Critical Workflow Files)
- [ ] `/new-feature` prompt works and creates specs in work/specs/
- [ ] `/fix-bug` prompt works and creates bugs in work/bugs/
- [ ] .github/ files have no references to old work/specs/, work/plans/, etc.
- [ ] AGENTS.md reflects new structure

### After Phase 6 (Breaking Changes)
- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make dev`
- [ ] Full validation passes: `make check`
- [ ] examples/hello runs correctly
- [ ] examples/auth runs correctly
- [ ] No old import paths remain: `grep -r 'basil/auth"' --include="*.go" .`
- [ ] No old import paths remain: `grep -r 'basil/config"' --include="*.go" .`

### Final (Before Merge)
- [ ] Tree structure looks correct: `tree -L 2 -d`
- [ ] All tests pass: `make check`
- [ ] No broken internal links in work/ docs
- [ ] README files accurate
- [ ] work/BACKLOG.md updated if items deferred

## Progress Log

| Date | Phase | Status | Notes |
|------|-------|--------|-------|
| — | Phase 1: Branch & Structure | ⏳ Not Started | — |
| — | Phase 2: Move Files | ⏳ Not Started | — |
| — | Phase 3: Update Workflow Files | ⏳ Not Started | CRITICAL - Test before continuing |
| — | Phase 4: Cross-References | ⏳ Not Started | — |
| — | Phase 5: User Docs | ⏳ Not Started | — |
| — | Phase 6: Breaking Changes | ⏳ Not Started | CRITICAL - Test before continuing |
| — | Phase 7: Final Validation | ⏳ Not Started | — |
| — | Phase 8: Merge | ⏳ Not Started | Human review required |

## Rollback Plan

**If issues discovered after merge:**

```bash
# Option A: Revert the merge commit
git revert -m 1 <merge-commit-hash>
git push origin main

# Option B: Hard reset (if no one else has pulled)
git reset --hard <commit-before-merge>
git push --force origin main

# Option C: Keep feature branch until confident
# Don't delete feat/docs-reorganization until after alpha release
```

## Deferred Items

Items to handle separately (not part of this reorganization):

1. **Archive file cleanup** — Human will review before 1.0:
   - `docs/parsley/CHEATSHEET.old.md`
   - `docs/parsley/Parts.txt`
   - `docs/parsley/TODO.md`
   - `docs/parsley/archive/` folder

2. **Root directory cleanup** (from original audit):
   - Remove `tubbo/`, `setup/`, `test-dev-error/` directories
   - Delete `.DS_Store` files
   - Remove SQLite `.db` files from examples
   - Verify binaries are gitignored

3. **docs/guide/ duplication**:
   - Resolve quick-start.md vs basil-quick-start.md
   - Clarify faq.md audience (user vs contributor questions)

## Notes

- **AI Workflow Compatibility:** This reorganization MUST work for AI agents. All path changes in .github/ instructions are critical. Phase 3 testing validates AI agent workflows work correctly.
- **AGENTS.md Location:** Kept in root as best practice for AI tooling discoverability (like README.md for humans).
- **ID_COUNTER.md Location:** Moved to work/ID_COUNTER.md since it's workflow-only. All .github/ files updated to reference new location.
- **BACKLOG.md Location:** Moved to work/BACKLOG.md since it's workflow-only (tracks deferred items during feature implementation). All .github/ files updated to reference new location.
- **CHANGELOG.md Location:** Stays in root (user-facing release history, not workflow tracking).
- **Breaking Changes:** Phase 6 moves auth/, config/, and search/ packages. External projects importing these will need to update their imports (unlikely before 1.0). search/ was in pkg/ but is server-specific, not a reusable library.
- **Critical Testing Points:** Phase 3 (AI workflows) and Phase 6 (Go imports/tests)
- **Time Estimate:** 10-12 hours total, can be done over 2-3 sessions
- **Safety:** Feature branch approach allows easy rollback
- **Timing:** Should complete before public alpha to avoid confusion about documentation organization

## Related Documents

- Original audit report: docs/FEAT-085-AUDIT.md → work/reports/FEAT-085-AUDIT.md (after move)
- Performance analysis: docs/basil-performance-analysis.md → work/reports/basil-performance-analysis.md (after move)
- ADR-001: docs/decisions/ADR-001-notification-api-defer.md → work/design/ADR-001-notification-api-defer.md (merged into design)
- ID_COUNTER.md: root → work/ID_COUNTER.md (workflow tracking, all .github/ references updated)
- BACKLOG.md: root → work/BACKLOG.md (deferred items tracking, all .github/ references updated)
- AGENTS.md: stays in root (AI tooling best practice)
- CHANGELOG.md: stays in root (user-facing release history)
- Investigation report: (conversation context)
- Phased plan: (conversation context)
