# Walkthrough: Making a Release

Step-by-step guide to releasing a new version.

## Overview

| Step | Who | What |
|------|-----|------|
| 1 | Human | Request release |
| 2 | AI | Gather changes and propose version |
| 3 | Human | Review changelog |
| 4 | Human | Execute release commands |

## Step 1: Request Release

Open Copilot Chat and use the release prompt:

```
/release
```

Or with more context:

```
/release We've finished FEAT-001 and fixed BUG-002, ready to release
```

## Step 2: AI Prepares Release

The AI will:
1. Check that tests pass
2. List all commits since last tag
3. Categorize changes (features, fixes, etc.)
4. Determine version bump:
   - Breaking changes → MAJOR
   - New features → MINOR
   - Bug fixes only → PATCH
5. Update `CHANGELOG.md`
6. Present summary for approval

You'll see something like:

```
## Release Summary: v0.1.0

### Changes
- [FEAT-001] Development process framework
- Initial project setup

### Breaking Changes
- None

### Upgrade Notes
- None
```

## Step 3: Review Changelog

Check `CHANGELOG.md`:

- [ ] Are all changes listed?
- [ ] Are they categorized correctly?
- [ ] Is the version number appropriate?
- [ ] Are breaking changes clearly noted?

**If changes needed:**
```
Please also mention the new documentation in the changelog
```

**If approved:**
```
Changelog looks good, I'll execute the release
```

## Step 4: Execute Release (Human)

The AI will tell you the commands, but **you execute them**:

```bash
# Commit the changelog update
git add CHANGELOG.md
git commit -m "chore(release): v0.1.0"

# Create tag
git tag v0.1.0

# Push with tags
git push origin main --tags
```

### Optional: GitHub Release

If you use GitHub releases:
1. Go to your repo on GitHub
2. Click "Releases" → "Create a new release"
3. Select the tag you just created
4. Copy release notes from CHANGELOG.md
5. Publish

## Version Numbering

We use [Semantic Versioning](https://semver.org/):

```
MAJOR.MINOR.PATCH

1.0.0 → 2.0.0  Breaking change
1.0.0 → 1.1.0  New feature (backwards compatible)
1.0.0 → 1.0.1  Bug fix (backwards compatible)
```

### Pre-1.0.0

Before version 1.0.0:
- MINOR bumps can include breaking changes
- PATCH bumps for bug fixes
- Use 0.x.x until API is stable

## Tips

- **Release often** — smaller releases are easier to debug
- **Write good commit messages** — they become changelog entries
- **Tag format** — always use `v` prefix: `v1.0.0`, not `1.0.0`
- **Don't forget to push tags** — `git push` alone doesn't push tags
