# Release Workflow

Use this prompt when preparing a release: `/release`

---

## Phase 1: Gather Changes

1. Get the current version from `CHANGELOG.md` or git tags
2. List all commits since the last release
3. Categorize changes by type:
   - **Features** (feat commits)
   - **Bug Fixes** (fix commits)
   - **Breaking Changes** (commits with `!` or `BREAKING CHANGE`)
   - **Other** (docs, refactor, chore, etc.)

Present the summary to the user.

---

## Phase 2: Determine Version

Based on changes, recommend version bump:
- **MAJOR** (X.0.0): Breaking changes
- **MINOR** (0.X.0): New features, no breaking changes
- **PATCH** (0.0.X): Bug fixes only

**Checkpoint**: Confirm version number with user.

---

## Phase 3: Prepare Changelog

1. Draft the changelog entry for the new version
2. Format:
   ```markdown
   ## [X.Y.Z] - YYYY-MM-DD

   ### Added
   - Feature description (FEAT-XXX)

   ### Fixed
   - Bug fix description (BUG-XXX)

   ### Changed
   - Change description

   ### Breaking Changes
   - Breaking change description
   ```
3. Present draft to user for review

**Checkpoint**: Wait for user approval.

---

## Phase 4: Update Files

1. Update `CHANGELOG.md` with the new entry
2. Update version in code if applicable (e.g., `version.go`)
3. Run full validation: `go build -o basil . && go test ./...`
4. Commit: `chore(release): prepare vX.Y.Z`

---

## Phase 5: Release Instructions

Provide the user with git commands to execute:

```bash
# Review the changes
git log --oneline

# Create the tag (user executes this)
git tag -a vX.Y.Z -m "Release vX.Y.Z"

# Push the tag (user executes this)
git push origin vX.Y.Z
```

**Note**: AI prepares the release, human creates and pushes tags.

---

## Quick Reference

| Version Bump | When |
|--------------|------|
| MAJOR | Breaking changes |
| MINOR | New features |
| PATCH | Bug fixes only |

| File | Purpose |
|------|---------|
| `CHANGELOG.md` | Release history |
| Git tags | Version markers |
