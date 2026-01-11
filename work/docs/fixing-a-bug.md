# Walkthrough: Fixing a Bug

Step-by-step guide to fixing a bug from report to merge.

## Overview

| Step | Who | What |
|------|-----|------|
| 1 | Human | Report the bug |
| 2 | AI | Investigate and analyze |
| 3 | Human | Approve fix strategy |
| 4 | AI | Implement fix |
| 5 | Human | Review and merge |

## Step 1: Report the Bug

Open Copilot Chat and use the bug prompt:

```
/fix-bug The config parser crashes when the config file is empty
```

Provide as much detail as you can:
- What you were doing
- What happened
- What you expected

## Step 2: AI Investigates

The AI will:
1. Allocate an ID (e.g., `BUG-001`)
2. Create `work/bugs/BUG-001.md`
3. Try to reproduce the issue
4. Analyze the root cause

You'll see updates in the bug report:
- Root Cause explanation
- Affected Code locations
- Fix Strategy
- Regression Risk assessment

## Step 3: Approve Fix Strategy

Review the AI's analysis in `work/bugs/BUG-001.md`:

- [ ] Does the root cause make sense?
- [ ] Is the fix strategy appropriate?
- [ ] Is the regression risk acceptable?

**If changes needed:**
```
I think the fix should also handle the case where the file doesn't exist
```

**If approved:**
```
Fix strategy approved, please implement
```

## Step 4: AI Implements Fix

The AI will:
1. Create branch `fix/BUG-001-empty-config`
2. Write a failing test (proves bug exists)
3. Implement the fix
4. Verify test passes
5. Check for similar issues elsewhere
6. Commit with `fix(BUG-001): description`

## Step 5: Review and Merge

When AI presents completed work:

1. **Check tests pass:**
   ```bash
   go test ./...
   ```

2. **Verify the fix:**
   Try to reproduce the original bug

3. **Review the test:**
   Does it actually test the bug scenario?

4. **Merge:**
   ```bash
   git checkout main
   git merge fix/BUG-001-empty-config
   git push
   ```

5. **Clean up:**
   ```bash
   git branch -d fix/BUG-001-empty-config
   ```

## Tips

- **Include reproduction steps** — helps AI find the issue faster
- **Check "Similar Patterns"** — AI might find the same bug elsewhere
- **Test-first approach** — the failing test proves the bug exists before fix
- **Small fixes** are safer than large refactors
